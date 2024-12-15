package impl

import (
	"Node-tion/backend/storage"
	"Node-tion/backend/transport"
	"Node-tion/backend/types"
	"encoding/hex"
	"golang.org/x/xerrors"
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

// ChatMessageCallback logs the chat message
func (n *node) ChatMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is a chat message
	chatMsg, ok := msg.(*types.ChatMessage)
	if !ok {
		return xerrors.Errorf("message is not a ChatMessage")
	}
	// log the message
	n.log.Info().Msgf("Received message from %s: %s", pkt.Header.Source, chatMsg.Message)
	return nil
}

// RumorsMessageCallback handles the RumorsMessage
func (n *node) RumorsMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is a rumors message
	rumorsMsg, ok := msg.(*types.RumorsMessage)
	if !ok {
		return xerrors.Errorf("message is not a RumorsMessage")
	}

	// flag for expected rumor
	expectedRumor := false
	for _, rumor := range rumorsMsg.Rumors {
		if n.view.AddRumorView(rumor, rumor.Origin) {
			expectedRumor = true // set to true if (at least one) rumor is expected

			// process the rumor
			err := n.ProcessRumor(rumor, pkt)
			if err != nil {
				return xerrors.Errorf("Failed to process rumor: %v", err)
			}

			// if the message is from an indirect neighbor, update the routing table
			if rumor.Origin != pkt.Header.RelayedBy || pkt.Header.Source == pkt.Header.RelayedBy {
				n.log.Info().Msgf("Updating routing table with %s as origin and %s as relay", rumor.Origin, pkt.Header.RelayedBy)
				n.SetRoutingEntry(rumor.Origin, pkt.Header.RelayedBy)
			}
		}
	}

	// send an AckMessage to the source
	err := n.SendAckMessage(pkt)
	if err != nil {
		return xerrors.Errorf("Failed to send AckMessage to Source: %v", err)
	}

	// if one of the rumors was expected, send a RumorsMessage to a random neighbor
	if expectedRumor {
		// marshal the RumorsMessage
		payload, err := n.conf.MessageRegistry.MarshalMessage(rumorsMsg)
		if err != nil {
			return xerrors.Errorf("Failed to marshal RumorsMessage: %v", err)
		}
		n.log.Info().Msgf("Sending RumorsMessage to rand neighbor after expected rumor, I am %s", n.conf.Socket.GetAddress())
		// send the RumorsMessage to a random neighbor
		_, err = n.SendRumorsMessageRand(payload, pkt.Header.Source)
		if err != nil {
			return xerrors.Errorf("Failed to send RumorsMessage: %v", err)
		}
	}
	return nil
}

// ProcessRumor processes the rumor
func (n *node) ProcessRumor(rumor types.Rumor, pkt transport.Packet) error {
	// process the message
	newPkt := transport.Packet{
		Header: pkt.Header,
		Msg:    rumor.Msg,
	}
	err := n.ProcessMsg(newPkt)
	if err != nil {
		return xerrors.Errorf("Failed to process message: %v", err)
	}
	return nil
}

// StatusMessageCallback handles the StatusMessage
func (n *node) StatusMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is a status message
	statusMsg, ok := msg.(*types.StatusMessage)
	if !ok {
		return xerrors.Errorf("message is not a StatusMessage")
	}

	// compare the status message with the view

	// case 1: remote peer has rumors that this node doesn't have -> send a StatusMessage
	case1 := false
	// case 2: this node has rumors that remote peer doesn't have -> send a RumorsMessage
	case2 := false
	// case 3: both have new rumors -> send StatusMessage and RumorsMessage
	// case 4: both have the same rumors -> send StatusMessage to a random neighbor

	// missing rumors
	var missingRumors []types.Rumor

	peerSeq := n.view.GetPeerSeq()
	// compare the status message with the view
	for origin, seq := range peerSeq {
		if remoteSeq, exists := (*statusMsg)[origin]; exists { // if the origin is in the status message
			if remoteSeq > seq {
				// remote peer has rumors that this node doesn't have
				case1 = true
			} else if remoteSeq < seq {
				// this node has rumors that the remote peer doesn't have
				case2 = true
				missingRumors = append(missingRumors, n.view.GetRumorView(origin, remoteSeq, seq)...)
			}
		} else { // if the origin (in local peerSeq) is not in the status message
			case2 = true
			missingRumors = append(missingRumors, n.view.GetRumorView(origin, 0, seq)...)
		}
	}

	// case 1: remote peer has rumors that this node doesn't have -> send a StatusMessage
	// ex. local view does not contain a peer that the remote peer has
	if len(*statusMsg) > len(peerSeq) {
		case1 = true // remote peer has rumors that this node doesn't have
	}

	if case1 {
		n.log.Info().Msgf("(case 1) Sending StatusMessage to %s because this node has missing rumors", pkt.Header.Source)
		// send a StatusMessage to the origin
		err := n.SendStatusMessage(pkt.Header.Source, peerSeq)
		if err != nil {
			return xerrors.Errorf("Failed to send StatusMessage: %v", err)
		}
	}

	if case2 {
		n.log.Info().Msgf("(case 2) Sending RumorsMessage to the source neighbor because he has missing rumors")
		// send a RumorsMessage to the source neighbor
		err := n.SendRumorsMessage(pkt, missingRumors)
		if err != nil {
			return xerrors.Errorf("Failed to send RumorsMessage: %v", err)
		}
	}

	if !case1 && !case2 {
		n.log.Info().Msgf("(case 4) Sending StatusMessage to a random neighbor because both have the same rumors")
		// send a StatusMessage to a random neighbor with a probability of ContinueMongering
		err := n.SendWithProbability(pkt, peerSeq)
		if err != nil {
			return xerrors.Errorf("Failed to send StatusMessage with probability of ContinueMongering: %v", err)
		}
	}

	return nil
}

// AckMessageCallback handles the AckMessage
func (n *node) AckMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is an ack message
	ackMsg, ok := msg.(*types.AckMessage)
	if !ok {
		return xerrors.Errorf("message is not an AckMessage")
	}

	// set the acked packet ID
	channel, ok := n.GetAck(ackMsg.AckedPacketID)
	if ok {
		channel <- true
	}

	// process the status message
	payload, err := n.conf.MessageRegistry.MarshalMessage(ackMsg.Status)
	if err != nil {
		return xerrors.Errorf("Failed to marshal StatusMessage: %v", err)
	}
	newPkt := transport.Packet{
		Header: pkt.Header,
		Msg:    &payload,
	}
	err = n.ProcessMsg(newPkt)
	if err != nil {
		return xerrors.Errorf("Failed to process message: %v", err)
	}

	n.log.Info().Msgf("Received AckMessage from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())
	return nil
}

// PrivateMessageCallback handles the PrivateMessage
func (n *node) PrivateMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check the peer’s socket address is in the list of recipients
	privateMsg, ok := msg.(*types.PrivateMessage)
	if !ok {
		return xerrors.Errorf("message is not a PrivateMessage")
	}
	// check if the message is for this node
	_, exists := privateMsg.Recipients[n.conf.Socket.GetAddress()]
	if exists {
		// process the message
		newPkt := transport.Packet{
			Header: pkt.Header,
			Msg:    privateMsg.Msg,
		}
		err := n.ProcessMsg(newPkt)
		if err != nil {
			return xerrors.Errorf("Failed to process message: %v", err)
		}
	}
	return nil
}

// EmptyMessageCallback handles the EmptyMessage
func (n *node) EmptyMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is an empty message
	_, ok := msg.(*types.EmptyMessage)
	if !ok {
		return xerrors.Errorf("message is not an EmptyMessage")
	}
	return nil
}

// DataRequestMessageCallback handles the DataRequestMessage
func (n *node) DataRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is a data request message
	dataReqMsg, ok := msg.(*types.DataRequestMessage)
	if !ok {
		return xerrors.Errorf("message is not a DataRequestMessage")
	}

	// respond with DataReplyMessage
	dataRepMsg := types.DataReplyMessage{
		RequestID: dataReqMsg.RequestID,
		Key:       dataReqMsg.Key,
		Value:     n.conf.Storage.GetDataBlobStore().Get(dataReqMsg.Key),
	}

	// add request ID and send the DataReplyMessage if it doesn't exist
	exists := n.AddRequest(dataReqMsg.RequestID)
	if !exists {
		payload, err := n.conf.MessageRegistry.MarshalMessage(dataRepMsg)
		if err != nil {
			return xerrors.Errorf("Failed to marshal DataReplyMessage: %v", err)
		}

		n.log.Info().Msgf("Sending DataReplyMessage to %s from %s", pkt.Header.Source, n.conf.Socket.GetAddress())
		err = n.Unicast(pkt.Header.Source, payload)
		if err != nil {
			return xerrors.Errorf("Failed to send DataReplyMessage: %v", err)
		}
	}
	return nil
}

// DataReplyMessageCallback handles the DataReplyMessage
func (n *node) DataReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	// check if the message is a data reply message
	dataRepMsg, ok := msg.(*types.DataReplyMessage)
	if !ok {
		return xerrors.Errorf("message is not a DataReplyMessage")
	}

	// send the data to the channel
	n.dataReplyChanMap.mu.Lock()
	replyChan, exists := n.dataReplyChanMap.repl[dataRepMsg.RequestID]
	n.dataReplyChanMap.mu.Unlock()
	if !exists {
		return xerrors.Errorf("No reply channel found for request ID %s", dataRepMsg.RequestID)
	}

	replyChan <- dataRepMsg.Value
	return nil
}

// ForwardSearchRequest handles the ForwardSearchRequest
func (n *node) ForwardSearchRequest(budget uint, src string, reg *regexp.Regexp, req *types.SearchRequestMessage) {
	neighbors := n.GetRandNeighsFromBudget(budget, src)

	neighsLeft := len(neighbors)
	for _, neigh := range neighbors {
		b := budget / uint(neighsLeft)
		if b == 0 {
			continue
		}

		// send a SearchRequestMessage to the neighbor, only budget changes
		err := n.SendSearchRequestMessage(neigh, *reg, b, req.RequestID, req.Origin)
		if err != nil {
			n.log.Error().Err(err).Msg("Error sending SearchRequestMessage")
			return
		}
		neighsLeft--
		budget -= b
	}
}

// SearchRequestMessageCallback handles the SearchRequestMessage
func (n *node) SearchRequestMessageCallback(msg types.Message, pkt transport.Packet) error {
	searchReqMsg, ok := msg.(*types.SearchRequestMessage)
	if !ok {
		return xerrors.Errorf("Message is not a SearchRequestMessage")
	}

	// forward the search request
	budgetLeft := searchReqMsg.Budget - 1
	reg := regexp.MustCompile(searchReqMsg.Pattern)
	if budgetLeft > 0 {
		go n.ForwardSearchRequest(budgetLeft, pkt.Header.Source, reg, searchReqMsg)
	}

	// construct the FileInfo
	// initial size of the fileInfos slice
	var fileInfos []types.FileInfo
	n.conf.Storage.GetNamingStore().ForEach(func(key string, val []byte) bool {
		if reg.Match([]byte(key)) {
			fileInfo, err := n.GetFileInfo(key, string(val))
			if err == nil {
				fileInfos = append(fileInfos, fileInfo)
			}
		}
		return true
	})

	// send the search reply message
	searchReplyMsg := types.SearchReplyMessage{
		RequestID: searchReqMsg.RequestID,
		Responses: fileInfos,
	}
	// marshal
	searchReplyPayload, err := n.conf.MessageRegistry.MarshalMessage(searchReplyMsg)
	if err != nil {
		return xerrors.Errorf("Failed to marshal SearchReplyMessage: %v", err)
	}
	// create header
	searchReplyHeader := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), searchReqMsg.Origin)
	searchReplyPkt := transport.Packet{
		Header: &searchReplyHeader,
		Msg:    &searchReplyPayload,
	}
	// send to source
	err = n.conf.Socket.Send(pkt.Header.Source, searchReplyPkt, time.Second*1)
	if err != nil {
		return xerrors.Errorf("Failed to send SearchReplyMessage: %v", err)
	}
	return nil
}

// GetFileInfo returns the FileInfo for the given name
func (n *node) GetFileInfo(name string, metahash string) (types.FileInfo, error) {
	dataBlobStore := n.conf.Storage.GetDataBlobStore()

	metafile := dataBlobStore.Get(metahash)
	if metafile == nil {
		// include only if the metafile is in the data store
		return types.FileInfo{}, xerrors.Errorf("Failed to get metafile for file %s", name)
	}
	// split the metafile into chunks
	chunkHashes := n.SplitMetafile(metafile)
	data := make([][]byte, len(chunkHashes))

	for i, chunkHash := range chunkHashes {
		if dataBlobStore.Get(chunkHash) == nil {
			data[i] = nil
		} else {
			data[i] = []byte(chunkHash) // append the keys (chunk hashes)
		}
	}
	return types.FileInfo{
		Name:     name,
		Metahash: metahash,
		Chunks:   data,
	}, nil
}

// SearchReplyMessageCallback handles the SearchReplyMessage
func (n *node) SearchReplyMessageCallback(msg types.Message, pkt transport.Packet) error {
	searchReplyMsg, ok := msg.(*types.SearchReplyMessage)
	if !ok {
		return xerrors.Errorf("Message is not a SearchReplyMessage")
	}

	allChunks := true
	// update the catalog and naming store
	for _, fileInfo := range searchReplyMsg.Responses {
		// file metahash and chunks
		n.UpdateCatalog(fileInfo.Metahash, pkt.Header.Source)
		for _, chunkKey := range fileInfo.Chunks {
			if chunkKey != nil {
				n.UpdateCatalog(string(chunkKey), pkt.Header.Source)
			} else {
				allChunks = false
			}
		}

		if n.Resolve(fileInfo.Name) == "" {
			err := n.Tag(fileInfo.Name, fileInfo.Metahash)
			if err != nil {
				return xerrors.Errorf("Failed to tag file: %v", err)
			}
		}

		if allChunks {
			n.log.Info().Msgf("All chunks are available for file %s", fileInfo.Name)
			// notify: send the data to the channel
			n.searchReplyChanMap.mu.Lock()
			replyChan, exists := n.searchReplyChanMap.repl[searchReplyMsg.RequestID]
			n.searchReplyChanMap.mu.Unlock()
			if exists {
				replyChan <- fileInfo.Name
				//close(replyChan)
				n.log.Info().Msgf("Sent file %s to the channel", fileInfo.Name)
			}
		}
	}
	return nil
}

// Acceptor

// PaxosPrepareMessageCallback handles the PaxosPrepareMessage
func (n *node) PaxosPrepareMessageCallback(msg types.Message, pkt transport.Packet) error {
	prepareMsg, ok := msg.(*types.PaxosPrepareMessage)
	if !ok {
		return xerrors.Errorf("Message is not a PaxosPrepareMessage")
	}

	// ignore the message if the proposal number is less than the highest seen
	// ignore the message if the current step is not the same as the step in the message
	if prepareMsg.Step != uint(n.logicalClock.GetStep()) || prepareMsg.ID <= uint(n.logicalClock.GetMaxID()) {
		n.logPAXOS.Info().Msgf("Ignoring PaxosPrepareMessage from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())
		return nil
	}

	n.logPAXOS.Info().Msgf("PREPARE: Received from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())

	// update the highest seen proposal number
	n.logicalClock.SetMaxID(uint64(prepareMsg.ID))

	acceptedValue, acceptedID := n.acceptor.GetAcceptedProposal()

	// respond with a PaxosPromiseMessage
	promiseMsg := types.PaxosPromiseMessage{
		Step:          prepareMsg.Step,
		ID:            prepareMsg.ID,
		AcceptedValue: acceptedValue,
		AcceptedID:    acceptedID,
	}

	promisePayload, err := n.conf.MessageRegistry.MarshalMessage(promiseMsg)
	if err != nil {
		return xerrors.Errorf("Failed to marshal PaxosPromiseMessage: %v", err)
	}

	recipient := make(map[string]struct{})
	recipient[prepareMsg.Source] = struct{}{}

	// create a PrivateMessage
	privateMsg := types.PrivateMessage{
		Msg:        &promisePayload,
		Recipients: recipient,
	}

	privatePayload, err := n.conf.MessageRegistry.MarshalMessage(privateMsg)
	if err != nil {
		return xerrors.Errorf("Failed to marshal PrivateMessage: %v", err)
	}
	return n.Broadcast(privatePayload)
}

// PaxosProposeMessageCallback handles the PaxosProposeMessage
func (n *node) PaxosProposeMessageCallback(msg types.Message, pkt transport.Packet) error {
	proposeMsg, ok := msg.(*types.PaxosProposeMessage)
	if !ok {
		return xerrors.Errorf("Message is not a PaxosProposeMessage")
	}

	if proposeMsg.Step != uint(n.logicalClock.GetStep()) || proposeMsg.ID != uint(n.logicalClock.GetMaxID()) {
		n.logPAXOS.Info().Msgf("Ignoring PaxosProposeMessage from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())
		return nil
	}

	n.logPAXOS.Info().Msgf("PROPOSE: Received from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())

	// accept the proposal
	n.acceptor.SetAcceptedProposal(&proposeMsg.Value, proposeMsg.ID)

	// respond with a PaxosAcceptMessage
	acceptMsg := types.PaxosAcceptMessage{
		Step:  proposeMsg.Step,
		ID:    proposeMsg.ID,
		Value: proposeMsg.Value,
	}

	acceptPayload, err := n.conf.MessageRegistry.MarshalMessage(acceptMsg)
	if err != nil {
		return xerrors.Errorf("Failed to marshal PaxosAcceptMessage: %v", err)
	}
	return n.Broadcast(acceptPayload)
}

// Proposer

// PaxosPromiseMessageCallback handles the PaxosPromiseMessage
func (n *node) PaxosPromiseMessageCallback(msg types.Message, pkt transport.Packet) error {
	promiseMsg, ok := msg.(*types.PaxosPromiseMessage)
	if !ok {
		return xerrors.Errorf("Message is not a PaxosPromiseMessage")
	}

	if promiseMsg.Step != uint(n.logicalClock.GetStep()) || n.proposer.GetPhase() != 1 ||
		promiseMsg.ID != n.proposer.GetProposalID() {
		n.logPAXOS.Info().Msgf("Ignoring PaxosPromiseMessage from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())
		return nil
	}

	n.logPAXOS.Info().Msgf("PROMISE: Received from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())

	// update the number of promises collected
	n.proposer.NewPromise()

	// update the highest accepted proposal
	highestAcceptedValue, highestAcceptedID := n.proposer.GetHighestAccepted()
	if (highestAcceptedValue == nil || promiseMsg.AcceptedID > highestAcceptedID) && promiseMsg.AcceptedValue != nil {
		n.proposer.SetHighestAcceptedProposal(promiseMsg.AcceptedValue, promiseMsg.AcceptedID)
	}

	// check if threshold is reached
	if n.proposer.GetPromisesCollected() >= uint(n.conf.PaxosThreshold(n.conf.TotalPeers)) {
		n.logPAXOS.Info().Msgf("Promise threshold for step %d, I am %s", n.logicalClock.GetStep(), n.conf.Socket.GetAddress())
		n.proposer.collectingPromises <- false
	}

	return nil
}

// PaxosAcceptMessageCallback handles the PaxosAcceptMessage
func (n *node) PaxosAcceptMessageCallback(msg types.Message, pkt transport.Packet) error {
	acceptMsg, ok := msg.(*types.PaxosAcceptMessage)
	if !ok {
		return xerrors.Errorf("Message is not a PaxosAcceptMessage")
	}

	if n.proposer.GetConsensus().Filename != "" {
		return nil
	}

	if acceptMsg.Step != uint(n.logicalClock.GetStep()) || n.proposer.GetPhase() == 1 {
		n.logPAXOS.Info().Msgf("Ignoring PaxosAcceptMessage from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())
		return nil
	}

	n.logPAXOS.Info().Msgf("ACCEPT: Received from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())

	// update the number of acceptances collected
	n.proposer.AddAcceptedProposal(acceptMsg.ID)

	consensusReached := false
	// check if threshold is reached
	if n.proposer.LenAcceptedProposals(acceptMsg.ID) >= n.conf.PaxosThreshold(n.conf.TotalPeers) &&
		n.proposer.GetConsensus().Filename == "" {
		n.proposer.SetConsensus(acceptMsg.Value)
		consensusReached = true
		n.proposer.collectingAccepts <- false
		n.logPAXOS.Info().Msgf("Consensus reached for step %d, I am %s", n.logicalClock.GetStep(), n.conf.Socket.GetAddress())
	}

	if consensusReached { // will only be executed once

		prevHash := n.conf.Storage.GetBlockchainStore().Get(storage.LastBlockKey)
		if prevHash == nil {
			prevHash = make([]byte, 32)
		}

		// construct blockchain block
		block := types.BlockchainBlock{
			Index:    uint(n.logicalClock.GetStep()),
			Hash:     nil,
			Value:    acceptMsg.Value,
			PrevHash: prevHash,
		}

		hash := append(block.Hash, strconv.Itoa(int(block.Index))...)
		hash = append(hash, block.Value.Filename...)
		hash = append(hash, block.Value.Metahash...)
		hash = append(hash, block.PrevHash...)

		block.Hash, _ = hex.DecodeString(n.HexEncode(hash))

		// broadcast TCLMessage
		tclMsg := types.TLCMessage{
			Step:  uint(n.logicalClock.GetStep()),
			Block: block,
		}

		tclPayload, err := n.conf.MessageRegistry.MarshalMessage(tclMsg)
		if err != nil {
			return xerrors.Errorf("Failed to marshal TLCMessage: %v", err)
		}
		n.proposer.SetTLCBroadcasted(true)

		n.logPAXOS.Info().Msgf("Broadcast TLC for step %d, I am %s", n.logicalClock.GetStep(), n.conf.Socket.GetAddress())

		return n.Broadcast(tclPayload)
	}
	return nil
}

// TLCMessageCallback handles the TLCMessage
func (n *node) TLCMessageCallback(msg types.Message, pkt transport.Packet) error {
	tclMsg, ok := msg.(*types.TLCMessage)
	if !ok {
		return xerrors.Errorf("Message is not a TLCMessage")
	}

	// save TLC messages from step >= current step, ignore otherwise
	if tclMsg.Step < uint(n.logicalClock.GetStep()) {
		n.logPAXOS.Info().Msg("Ignoring TLCMessage from the past")
		return nil
	}

	n.logPAXOS.Info().Msgf("TLC: Received from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())

	n.tlcMessages.AddTLCMessage(tclMsg)
	n.logPAXOS.Info().Msgf("Length TLCs: %d", n.tlcMessages.LenTLCMessages(uint(n.logicalClock.GetStep())))

	if tclMsg.Step > uint(n.logicalClock.GetStep()) {
		return nil
	}

	for { // catchup loop
		tclMsg := n.tlcMessages.GetTLCMessages(uint(n.logicalClock.GetStep()))[0]
		// check tclMsg.Step == logicalClock.GetStep() to avoid processing messages from the future
		if n.tlcMessages.LenTLCMessages(uint(n.logicalClock.GetStep())) >= n.conf.PaxosThreshold(n.conf.TotalPeers) &&
			tclMsg.Step == uint(n.logicalClock.GetStep()) {
			// increase TLC step as first instruction to avoid adding the same block multiple times when 2 TLCs are too close
			n.logicalClock.Increment()
			n.logPAXOS.Info().Msgf("Next TLC step %d, I am %s", n.logicalClock.GetStep(), n.conf.Socket.GetAddress())

			err := n.AddBlockAndName(tclMsg.Block)
			if err != nil {
				return xerrors.Errorf("Failed to add block and name: %v", err)
			}

			err = n.TryBroadcast(tclMsg)
			if err != nil {
				return xerrors.Errorf("Failed to broadcast TLCMessage: %v", err)
			}

			// reset proposer, acceptor, logical clock maxID
			n.TLCStepReset()
		} else {
			break
		}
	}

	return nil
}

// AddBlockAndName adds the block and name to the blockchain and naming store
func (n *node) AddBlockAndName(block types.BlockchainBlock) error {
	// add block to blockchain
	marshaledBlock, err := block.Marshal()
	if err != nil {
		return xerrors.Errorf("Failed to marshal block: %v", err)
	}

	n.conf.Storage.GetBlockchainStore().Set(hex.EncodeToString(block.Hash), marshaledBlock)
	n.conf.Storage.GetBlockchainStore().Set(storage.LastBlockKey, block.Hash)

	n.logPAXOS.Info().Msgf("Block %s added, I am %s", block.String(), n.conf.Socket.GetAddress())

	// set name to metahash mapping
	n.conf.Storage.GetNamingStore().Set(block.Value.Filename, []byte(block.Value.Metahash))

	return nil
}

// TryBroadcast broadcasts the message if not already broadcasted
func (n *node) TryBroadcast(msg types.Message) error {
	if !n.proposer.GetTLCBroadcasted() {
		payload, err := n.conf.MessageRegistry.MarshalMessage(msg)
		if err != nil {
			return xerrors.Errorf("Failed to marshal message: %v", err)
		}
		err = n.Broadcast(payload)
		if err != nil {
			return xerrors.Errorf("Failed to broadcast message: %v", err)
		}
		n.proposer.SetTLCBroadcasted(true)
	}
	return nil
}

// TLCStepReset resets the proposer, acceptor, and logical clock maxID
func (n *node) TLCStepReset() {
	n.proposer.Reset()
	n.acceptor.Reset()
	n.logicalClock.ResetMaxID()
}

// CRDTOperationsMessageCallback handles the CRDTOperationsMessage
func (n *node) CRDTOperationsMessageCallback(msg types.Message, pkt transport.Packet) error {
	crdtMsg, ok := msg.(*types.CRDTOperationsMessage)
	if !ok {
		return xerrors.Errorf("Message is not a CRDTOperationsMessage")
	}

	n.logCRDT.Info().Msgf("Received CRDTOperationsMessage from %s, I am %s", pkt.Header.Source, n.conf.Socket.GetAddress())

	err := n.UpdateEditor(crdtMsg.Operations)
	if err != nil {
		return xerrors.Errorf("Failed to update editor: %v", err)
	}
	return nil
}

// SendRumorsMessage sends a RumorsMessage to the source neighbor
func (n *node) SendRumorsMessage(pkt transport.Packet, missingRumors []types.Rumor) error {
	rumorsMsg := types.RumorsMessage{
		Rumors: missingRumors,
	}
	err := n.SendMsg(pkt.Header.Source, rumorsMsg)
	if err != nil {
		return xerrors.Errorf("Failed to send RumorsMessage: %v", err)
	}
	return nil
}

// SendRumorsMessageRand sends a RumorsMessage to a random neighbor
func (n *node) SendRumorsMessageRand(msg transport.Message, source string) (packetHeader transport.Header, err error) {
	// if the ticker times out, resend the RumorsMessage to a random neighbor
	randomNeighbor := n.GetRandomNeighborFromRoutingTable(source)
	// if randomNeighbor is empty, do not send
	if randomNeighbor == "" {
		n.log.Info().Msg("No random neighbor to send RumorsMessage to")
		return transport.Header{}, nil
	}
	// Send the RumorsMessage to the chosen neighbor
	header := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), randomNeighbor)
	newPkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}

	n.log.Info().Msgf("Sending RumorsMessage to %s from %s", randomNeighbor, n.conf.Socket.GetAddress())
	err = n.conf.Socket.Send(randomNeighbor, newPkt, time.Second*1)
	if err != nil {
		n.log.Error().Err(err).Msg("Failed to send RumorsMessage")
	}

	return header, err
}

// SendStatusMessage sends a StatusMessage to the neighbor
func (n *node) SendStatusMessage(neighbor string, peerSeq map[string]uint) error {
	// if the neighbor is null, do not send
	if neighbor == "" {
		n.log.Info().Msg("No neighbor to send StatusMessage because no neighbor")
		return nil
	}
	statusMsg := types.StatusMessage(peerSeq)

	n.log.Info().Msgf("Sending StatusMessage to %s from %s", neighbor, n.conf.Socket.GetAddress())
	err := n.SendMsg(neighbor, statusMsg)
	if err != nil {
		return xerrors.Errorf("Failed to send StatusMessage: %v", err)
	}
	return nil
}

// SendWithProbability sends a StatusMessage to a random neighbor with a probability of ContinueMongering
func (n *node) SendWithProbability(pkt transport.Packet, peerSeq map[string]uint) error {
	p := n.conf.ContinueMongering // probability of sending a StatusMessage
	// send a StatusMessage to a random neighbor with a probability of ContinueMongering
	if p > 0 && p <= 1 {
		if rand.Float64() <= p { // send if pseudo-random number in the half-open interval [0.0,1.0) is less than p
			// send a StatusMessage to a random neighbor
			neighbor := n.GetRandomNeighborFromRoutingTable(pkt.Header.Source)

			n.log.Info().Msgf("Sending StatusMessage with probability of ContinueMongering, I am %s", n.conf.Socket.GetAddress())

			err := n.SendStatusMessage(neighbor, peerSeq)
			if err != nil {
				return xerrors.Errorf("Failed to send StatusMessage: %v", err)
			}
		}
	}
	return nil
}

// SendAckMessage sends an AckMessage to the origin
func (n *node) SendAckMessage(pkt transport.Packet) error {
	statusMsg := types.StatusMessage(n.view.GetPeerSeq())
	// create an AckMessage
	ackMsg := types.AckMessage{
		AckedPacketID: pkt.Header.PacketID,
		Status:        statusMsg,
	}

	n.log.Info().Msgf("Sending AckMessage to %s from %s", pkt.Header.Source, n.conf.Socket.GetAddress())
	err := n.SendMsg(pkt.Header.Source, ackMsg)
	if err != nil {
		return xerrors.Errorf("Failed to send AckMessage: %v", err)
	}
	return nil
}

// SendHeartbeat sends a heartbeat
func (n *node) SendHeartbeat() error {
	// send a heartbeat
	payload, err := n.conf.MessageRegistry.MarshalMessage(types.EmptyMessage{})
	if err != nil {
		return xerrors.Errorf("Failed to marshal EmptyMessage: %v", err)
	}
	err = n.Broadcast(payload)
	if err != nil {
		return xerrors.Errorf("Failed to send first heartbeat: %v", err)
	}
	return nil
}

// SendDataRequestMessage sends a DataRequestMessage to the neighbor.
func (n *node) SendDataRequestMessage(randPeer, metahash, requestID string) error {
	msg := types.DataRequestMessage{
		RequestID: requestID,
		Key:       metahash,
	}

	payload, err := n.conf.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return xerrors.Errorf("Failed to marshal DataRequestMessage: %v", err)
	}

	n.log.Info().Msgf("Sending DataRequestMessage to %s from %s", randPeer, n.conf.Socket.GetAddress())
	err = n.Unicast(randPeer, payload)
	if err != nil {
		return xerrors.Errorf("Failed to send DataRequestMessage: %v", err)
	}
	return nil
}

// SendSearchRequestMessage sends a SearchRequestMessage to the neighbor.
func (n *node) SendSearchRequestMessage(neigh string, r regexp.Regexp, b uint, requestID string, origin string) error {
	msg := types.SearchRequestMessage{
		RequestID: requestID,
		Origin:    origin,
		Pattern:   r.String(),
		Budget:    b,
	}

	n.log.Info().Msgf("Sending SearchRequestMessage to %s from %s", neigh, n.conf.Socket.GetAddress())
	err := n.SendMsg(neigh, msg)
	if err != nil {
		return xerrors.Errorf("Failed to send SearchRequestMessage: %v", err)
	}
	return nil
}
