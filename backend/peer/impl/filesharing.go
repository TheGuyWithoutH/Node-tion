package impl

import (
	"Node-tion/backend/peer"
	"Node-tion/backend/types"
	"bytes"
	"errors"
	"github.com/rs/xid"
	"golang.org/x/xerrors"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Upload implements DataSharing
func (n *node) Upload(data io.Reader) (metahash string, err error) {
	// max upload size
	read := io.LimitReader(data, n.maxUploadSize+1)

	buffer := new(bytes.Buffer)
	_, err = io.Copy(buffer, read)
	if err != nil {
		return "", err
	}

	// Check if the data size exceeds the limit
	if buffer.Len() > int(n.maxUploadSize) {
		return "", errors.New("file size exceeds 2 MiB limit")
	}

	// split the file into chunks and store each chunk using the hex-encoded chunk’s hash as the key
	chunkSize := int(n.conf.ChunkSize)

	// split the data into chunks
	var chunks [][]byte
	for i := 0; i < len(buffer.Bytes()); i += chunkSize {
		end := i + chunkSize
		if end > len(buffer.Bytes()) {
			end = len(buffer.Bytes())
		}
		chunks = append(chunks, buffer.Bytes()[i:end])
	}

	dataBlobStore := n.conf.Storage.GetDataBlobStore()
	metafileValue := ""

	// hex-encoded chunk’s hash as the key
	for _, chunk := range chunks {
		chunkHash := n.HexEncode(chunk)
		dataBlobStore.Set(chunkHash, chunk)
		metafileValue += chunkHash + peer.MetafileSep
	}

	// remove the last separator
	if len(metafileValue) > 0 {
		metafileValue = metafileValue[:len(metafileValue)-len(peer.MetafileSep)]
	} else {
		return "", errors.New("no data to upload")
	}

	// hash the metafileValue
	metahash = n.HexEncode([]byte(metafileValue))
	dataBlobStore.Set(metahash, []byte(metafileValue))

	return metahash, nil
}

// Download implements DataSharing
func (n *node) Download(metahash string) ([]byte, error) {
	// get the metafile
	metafile, err := n.DownloadElement(metahash)
	if err != nil {
		return nil, err
	}
	// split the metafile into chunks
	chunkHashes := n.SplitMetafile(metafile)

	// get the chunks from the data store
	var data []byte
	for _, chunkHash := range chunkHashes {
		chunk, err := n.DownloadElement(chunkHash)
		if err != nil {
			return nil, err
		}
		data = append(data, chunk...) // reconstruct the file
	}
	return data, nil
}

// DownloadElement downloads a single element from the data store.
func (n *node) DownloadElement(key string) ([]byte, error) {
	// get the metafile from the naming store
	metafile := n.conf.Storage.GetDataBlobStore().Get(key)
	if metafile != nil {
		return metafile, nil
	}

	metafile, err := n.RemoteDownload(key)
	if err != nil {
		return nil, err
	}

	if key != n.HexEncode(metafile) {
		// update catalog to exclude the peer
		//n.RemovePeerFromCatalog(key, randPeer) ISSUE HERE
		return nil, xerrors.Errorf("Data is tampered with")
	}
	if metafile == nil {
		return nil, xerrors.Errorf("Reply data is nil")
	}

	// store the data in the data store
	n.conf.Storage.GetDataBlobStore().Set(key, metafile)
	return metafile, nil
}

// RemoteDownload downloads a single element from a remote peer.
func (n *node) RemoteDownload(key string) ([]byte, error) {
	// select a random neighbor from the catalog and ask for the metafile
	// through a DataRequestMessage

	// get a random peer from the catalog
	randPeer := n.GetRandomPeerFromCatalog(key)
	if randPeer == "" {
		return nil, xerrors.Errorf("No way to get the metafile")
	}

	requestID := xid.New().String()
	// set up a channel to receive the DataReplyMessage, with the requestID
	// as the key BEFORE sending the DataRequestMessage
	replyChan := make(chan []byte, 1)
	n.SetDataReplyChan(requestID, replyChan)
	defer close(replyChan)
	defer n.DeleteDataReplyChan(requestID)

	// send a DataRequestMessage to the neighbor
	err := n.SendDataRequestMessage(randPeer, key, requestID)
	if err != nil {
		return nil, err
	}

	// wait for the DataReplyMessage
	// with back-off

	// Ticker to check if the replyChan has been closed
	replyRetry := int(n.conf.BackoffDataRequest.Retry)
	timeTicker := n.conf.BackoffDataRequest.Initial

	for replyRetry > 0 {
		replyTicker := time.NewTimer(timeTicker)

		select {
		case metafile := <-replyChan:
			replyTicker.Stop()
			return metafile, nil

		case <-replyTicker.C:
			replyTicker.Stop()
			err := n.SendDataRequestMessage(randPeer, key, requestID)
			if err != nil {
				return nil, err
			}
		}
		replyRetry--
		timeTicker *= time.Duration(n.conf.BackoffDataRequest.Factor)
	}
	return nil, xerrors.Errorf("Reply retry is exhausted")
}

// SplitMetafile splits the metafile into chunks.
func (n *node) SplitMetafile(metafile []byte) []string {
	return strings.Split(string(metafile), peer.MetafileSep)
}

// Tag implements DataSharing
func (n *node) Tag(name string, mh string) error {
	if n.conf.Storage.GetNamingStore().Get(name) != nil {
		return xerrors.Errorf("Name %s already tagged", name)
	}

	// if there is only one peer in the network, store the mapping between the name and the metahash
	// no need for PAXOS because there is only one peer
	if n.conf.TotalPeers <= 1 {
		n.conf.Storage.GetNamingStore().Set(name, []byte(mh))
		return nil
	}

	n.logPAXOS.Info().Msgf("Starting PAXOS for %s, I am %s", name, n.conf.Socket.GetAddress())

PAXOS_LOOP:
	// start the PAXOS loop
	err := n.PaxosLoop(name, mh)
	if err != nil {
		return err
	}

	// check that the tag used is unique
	if name == n.proposer.GetConsensus().Filename && mh != n.proposer.GetConsensus().Metahash {
		return xerrors.Errorf("Tag %s already used", name)
	}

	// check that the value accepted is the same as the one proposed
	if name != n.proposer.GetConsensus().Filename || mh != n.proposer.GetConsensus().Metahash {
		goto PAXOS_LOOP
	}

	return nil
}

// PaxosLoop executes the Paxos loop.
func (n *node) PaxosLoop(name string, mh string) error {
	// PAXOS
	currID := n.conf.PaxosID

PAXOS_RETRY:
	// retry for a certain amount of time if not enough promises collected
	// wait for the promises to be collected
	for {
		// Phase 1

		// set phase 1 and the proposal ID
		n.proposer.SetProposalID(currID)
		n.proposer.SetPhase(1)

		// broadcast a PaxosPrepareMessage
		prepareMsg := types.PaxosPrepareMessage{
			Step:   uint(n.logicalClock.GetStep()),
			ID:     currID,
			Source: n.conf.Socket.GetAddress(),
		}

		// marshal the prepare message
		payload, err := n.conf.MessageRegistry.MarshalMessage(prepareMsg)
		if err != nil {
			return xerrors.Errorf("Error marshalling PaxosPrepareMessage: %v", err)
		}

		err = n.Broadcast(payload)
		if err != nil {
			return xerrors.Errorf("Error broadcasting PaxosPrepareMessage: %v", err)
		}

		// instantiate timer
		timer := time.NewTimer(n.conf.PaxosProposerRetry)

		select {
		case <-timer.C:
			timer.Stop()
			n.logPAXOS.Info().Msgf("Timeout, I am %s", n.conf.Socket.GetAddress())
			currID += n.conf.TotalPeers // next ID = current ID + total number of peers
			n.proposer.ResetPromises()
			continue // retry
		case <-n.proposer.collectingPromises:
			timer.Stop()
			n.proposer.SetPhase(2)
		}

		// Phase 2
		paxosValue := types.PaxosValue{
			Filename: name,
			Metahash: mh,
		}

		// get the highest accepted value
		val, _ := n.proposer.GetHighestAccepted()
		if val != nil {
			paxosValue = *val
		}

		// broadcast a PaxosProposeMessage
		proposeMsg := types.PaxosProposeMessage{
			Step:  uint(n.logicalClock.GetStep()),
			ID:    currID,
			Value: paxosValue,
		}

		// marshal the propose message
		payload, err = n.conf.MessageRegistry.MarshalMessage(proposeMsg)
		if err != nil {
			return xerrors.Errorf("Error marshalling PaxosProposeMessage: %v", err)
		}

		err = n.Broadcast(payload)
		if err != nil {
			return xerrors.Errorf("Error broadcasting PaxosProposeMessage: %v", err)
		}

		// restart timer
		timer.Reset(n.conf.PaxosProposerRetry)

		select {
		case <-timer.C:
			timer.Stop()
			currID += n.conf.TotalPeers // next ID = current ID + total number of peers
			continue                    // retry
		case <-n.proposer.collectingAccepts:
			timer.Stop()
			break PAXOS_RETRY
		}
	}
	return nil
}

// Resolve implements DataSharing
func (n *node) Resolve(name string) (metahash string) {
	return string(n.conf.Storage.GetNamingStore().Get(name))
}

// SearchAll implements DataSharing
func (n *node) SearchAll(reg regexp.Regexp, budget uint, timeout time.Duration) (names []string, err error) {
	// distribute budget to neighbors

	// if the budget is greater or equal to the number of neighbors, distribute the budget evenly
	// among all neighbors
	// if the budget is less than the number of neighbors, distribute the budget evenly among a
	// random subset of neighbors, with budget 1 for each neighbor
	neighbors := n.GetRandNeighsFromBudget(budget)
	if len(neighbors) != 0 {

		var wait sync.WaitGroup // wait for all the SearchReplyMessages to be received

		neighsLeft := len(neighbors)
		budgetLeft := budget

		for _, neigh := range neighbors {
			b := budgetLeft / uint(neighsLeft)
			// if budget < neighbors, only b = 1 for budget neighbors
			// if budget >= neighbors, b = budget / neighbors
			// case 1: perfect distribution
			// case 2: modulo budget % neighbors and distribute the remainder equally
			if b == 0 {
				continue
			}

			wait.Add(1)
			go func(neigh string) {
				defer wait.Done()
				// send a SearchRequestMessage to the neighbor
				requestID := xid.New().String()
				err := n.SendSearchRequestMessage(neigh, reg, b, requestID, n.conf.Socket.GetAddress())
				if err != nil {
					n.log.Error().Err(err).Msg("Error sending SearchRequestMessage")
					return
				}

				timer := time.NewTimer(timeout)
				<-timer.C // wait for the timeout
				timer.Stop()
			}(neigh)

			neighsLeft--
			budgetLeft -= b
		}
		wait.Wait()
	}

	// once the SearchReplyMessages are received, the name storage has been updated
	// names from the local storage containing reg
	names = n.SearchMatch(reg)

	return names, nil
}

// SearchMatch looks for filenames that match the pattern.
func (n *node) SearchMatch(reg regexp.Regexp) []string {
	var names []string

	n.conf.Storage.GetNamingStore().ForEach(func(k string, v []byte) bool {
		if reg.Match([]byte(k)) {
			names = append(names, k)
		}
		return true
	})
	return names
}

// SearchFirstLocal searches for the first file that matches the pattern.
func (n *node) SearchFirstLocal(pattern regexp.Regexp) (name string) {
	names := n.SearchMatch(pattern)
	for _, name := range names {
		metahash := n.Resolve(name)
		metafile := n.conf.Storage.GetDataBlobStore().Get(metahash)
		chunkHashes := n.SplitMetafile(metafile)
		full := true
		for _, chunkHash := range chunkHashes {
			if n.conf.Storage.GetDataBlobStore().Get(chunkHash) == nil {
				full = false
				break
			}
		}
		if full {
			return name
		}
	}
	return ""
}

// ExpandRing expands the ring.
func (n *node) ExpandRing(conf peer.ExpandingRing,
	neigh string,
	reg regexp.Regexp,
	budget uint,
	wait *sync.WaitGroup,
	match chan<- string) {

	defer wait.Done()
	// send a SearchRequestMessage to the neighbor
	requestID := xid.New().String()

	// set up a channel to receive the SearchReplyMessage, with the requestID
	// as the key
	replyChan := make(chan string, 1) // only one reply is expected
	n.SetSearchReplyChan(requestID, replyChan)
	defer close(replyChan)
	defer n.DeleteSearchReplyChan(requestID)

	err := n.SendSearchRequestMessage(neigh, reg, budget, requestID, n.conf.Socket.GetAddress())
	if err != nil {
		n.log.Error().Err(err).Msg("Error sending SearchRequestMessage")
	}

	n.log.Info().Msg("Sent SearchRequestMessage in SearchFirst")

	timer := time.NewTimer(conf.Timeout)
	defer timer.Stop()

	n.log.Info().Msg("Waiting for SearchReplyMessage")
	select {
	case filename := <-replyChan:
		n.log.Info().Msgf("Received filename %s from %s", filename, neigh)
		if filename != "" {
			match <- filename
		}
	case <-timer.C:
		n.log.Info().Msgf("Stopped timer for %s", neigh)
	}
}

// SearchFirst implements DataSharing
func (n *node) SearchFirst(pattern regexp.Regexp, conf peer.ExpandingRing) (name string, err error) {
	// check locally if node has all chunks of a file
	name = n.SearchFirstLocal(pattern)
	if name != "" {
		return name, nil
	}

	// expanding-ring search
	budget := conf.Initial

	for i := 0; i < int(conf.Retry); i++ {
		neighbors := n.GetRandNeighsFromBudget(budget)
		if len(neighbors) == 0 {
			continue
		}

		match := make(chan string, 1)
		budgetMap := n.CreateBudgetMap(budget, len(neighbors))

		var wait sync.WaitGroup // wait for all the SearchReplyMessages to be received

		for i, neigh := range neighbors {
			b := budgetMap[i]
			if b == 0 {
				continue
			}

			wait.Add(1)
			go n.ExpandRing(conf, neigh, pattern, b, &wait, match)
		}

		timeout := make(chan bool)
		go func() {
			wait.Wait()
			n.log.Info().Msg("Waiting for waitgroup")
			timeout <- true
			close(timeout)
		}()

		select {
		case name := <-match:
			return name, nil
		case <-timeout: // expand ring
			budget *= conf.Factor
		}
	}
	return name, nil
}
