package impl

import (
	"Node-tion/backend/peer"
	"Node-tion/backend/transport"
	"Node-tion/backend/types"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"
)

var logIO = zerolog.ConsoleWriter{
	Out:        os.Stdout,
	TimeFormat: time.RFC3339,
}

// NewPeer creates a new peer. You can change the content and location of this
// function, but you MUST NOT change its signature and package location.
func NewPeer(conf peer.Configuration) peer.Peer {
	logger := newLogger(logIO, zerolog.DebugLevel)
	loggerPAXOS := newLogger(logIO, zerolog.DebugLevel)
	loggerCRDT := newLogger(logIO, zerolog.DebugLevel)

	routingTable := newRoutingTable()
	view := newView()
	ackTickers := newAckMap()
	catalog := newCatalog()
	dataReplyChanMap := newDataReplyChanMap()
	searchReplyChanMap := newSearchReplyChanMap()
	requests := newRequests()
	logicalClock := newLogicalClock()
	acceptor := newAcceptor()
	proposer := newProposer()
	tlcMessages := newTLC()
	editor := newEditor()
	docTimestampMap := newDocTimestampMap()
	crdtState := newCRDTState()

	maxUploadSize := 2 * 1024 * 1024 // 2 MiB

	node := node{
		conf:               conf,
		mu:                 sync.Mutex{},
		log:                logger,
		logPAXOS:           loggerPAXOS,
		logCRDT:            loggerCRDT,
		routingTable:       routingTable,
		view:               view,
		ackTickers:         ackTickers,
		catalog:            catalog,
		dataReplyChanMap:   dataReplyChanMap,
		searchReplyChanMap: searchReplyChanMap,
		requests:           requests,
		maxUploadSize:      int64(maxUploadSize),
		logicalClock:       logicalClock,
		acceptor:           acceptor,
		proposer:           proposer,
		tlcMessages:        tlcMessages,
		editor:             editor,
		docTimestampMap:    docTimestampMap,
		crdtState:          crdtState,
	}

	return &node
}

// Helper functions

func newLogger(io io.Writer, level zerolog.Level) zerolog.Logger {
	logger := zerolog.New(io).With().Timestamp().Logger()
	return logger.Level(level)
}

func newRoutingTable() *RoutingTable {
	return &RoutingTable{
		mu: sync.Mutex{},
		rt: make(peer.RoutingTable),
	}
}

func newView() *View {
	return &View{
		mu:       sync.Mutex{},
		rumorSeq: 0,
		peerSeq:  make(map[string]uint),
		rumors:   make(map[string][]types.Rumor),
	}
}

func newAckMap() *AckMap {
	return &AckMap{
		mu:  sync.Mutex{},
		ack: make(map[string]chan bool),
	}
}

func newCatalog() *Catalog {
	return &Catalog{
		mu:  sync.Mutex{},
		cat: make(peer.Catalog),
	}
}

func newDataReplyChanMap() *DataReplyChanMap {
	return &DataReplyChanMap{
		mu:   sync.Mutex{},
		repl: make(map[string]chan []byte),
	}
}

func newSearchReplyChanMap() *SearchReplyChanMap {
	return &SearchReplyChanMap{
		mu:   sync.Mutex{},
		repl: make(map[string]chan string),
	}
}

func newRequests() *Requests {
	return &Requests{
		mu:  sync.Mutex{},
		req: make(map[string]time.Time),
	}
}

func newLogicalClock() *LogicalClock {
	return &LogicalClock{
		currentStep: 0,
		maxID:       0,
	}
}

func newAcceptor() *Acceptor {
	return &Acceptor{
		mu:          sync.Mutex{},
		acceptedVal: nil,
		acceptedID:  0,
	}
}

func newProposer() *Proposer {
	return &Proposer{
		mu:                 sync.Mutex{},
		phase:              0,
		proposalID:         0,
		promisesCollected:  0,
		collectingPromises: make(chan bool),
		highestAccepted:    newAcceptor(),
		acceptedProposals:  make(map[uint]uint),
		collectingAccepts:  make(chan bool, 1),
		consensus:          types.PaxosValue{},
		tlcBroadcasted:     false,
	}
}

func newTLC() *TLC {
	return &TLC{
		mu:          sync.Mutex{},
		tlcMessages: make(map[uint][]*types.TLCMessage),
	}
}

func newEditor() *Editor {
	return &Editor{
		mu: sync.Mutex{},
		ed: make(peer.Editor),
	}
}

func newDocTimestampMap() *DocTimestampMap {
	return &DocTimestampMap{
		mu:              sync.Mutex{},
		newestTimestamp: make(map[string]int64),
		docSaved:        make(map[string][]string),
	}
}

func newCRDTState() *CRDTState {
	return &CRDTState{
		state: make(map[string]uint64),
	}
}

// node implements a peer to build a Peerster system
//
// - implements peer.Peer
type node struct {
	peer.Peer
	conf               peer.Configuration
	mu                 sync.Mutex
	ctx                context.Context    // for managing the start/stop
	cancel             context.CancelFunc // to cancel the listening goroutine
	log                zerolog.Logger
	logPAXOS           zerolog.Logger
	logCRDT            zerolog.Logger
	routingTable       *RoutingTable
	view               *View
	ackTickers         *AckMap
	catalog            *Catalog
	dataReplyChanMap   *DataReplyChanMap
	searchReplyChanMap *SearchReplyChanMap
	requests           *Requests
	maxUploadSize      int64
	logicalClock       *LogicalClock
	acceptor           *Acceptor
	proposer           *Proposer
	tlcMessages        *TLC
	editor             *Editor
	docTimestampMap    *DocTimestampMap
	crdtState          *CRDTState
}

// Start implements peer.Service
func (n *node) Start() error {
	n.ctx, n.cancel = context.WithCancel(context.Background())

	// register the callback functions
	n.conf.MessageRegistry.RegisterMessageCallback(&types.ChatMessage{}, n.ChatMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.RumorsMessage{}, n.RumorsMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.StatusMessage{}, n.StatusMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.AckMessage{}, n.AckMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.PrivateMessage{}, n.PrivateMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.EmptyMessage{}, n.EmptyMessageCallback)

	n.conf.MessageRegistry.RegisterMessageCallback(&types.DataRequestMessage{}, n.DataRequestMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.DataReplyMessage{}, n.DataReplyMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.SearchRequestMessage{}, n.SearchRequestMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.SearchReplyMessage{}, n.SearchReplyMessageCallback)

	n.conf.MessageRegistry.RegisterMessageCallback(&types.PaxosPrepareMessage{}, n.PaxosPrepareMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.PaxosProposeMessage{}, n.PaxosProposeMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.PaxosPromiseMessage{}, n.PaxosPromiseMessageCallback)
	n.conf.MessageRegistry.RegisterMessageCallback(&types.PaxosAcceptMessage{}, n.PaxosAcceptMessageCallback)

	n.conf.MessageRegistry.RegisterMessageCallback(&types.TLCMessage{}, n.TLCMessageCallback)

	n.conf.MessageRegistry.RegisterMessageCallback(&types.CRDTOperationsMessage{}, n.CRDTOperationsMessageCallback)

	n.SetRoutingEntry(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress())

	// use non-blocking GoRoutine to listen on incoming messages
	go n.Listen()

	// check if HeartbeatInterval is > 0
	if n.conf.HeartbeatInterval > 0 {
		// first heartbeat
		err := n.SendHeartbeat()
		if err != nil {
			return xerrors.Errorf("failed to send first heartbeat: %v", err)
		}
		// heart beat go routine
		go n.HeartbeatTicker()
	}

	// anti-entropy go routine
	if n.conf.AntiEntropyInterval > 0 {
		go n.AntiEntropyTicker()
	}

	return nil
}

// Stop implements peer.Service
func (n *node) Stop() error {
	if n.cancel != nil {
		n.cancel() // cancel the context to stop the listening loop
	}
	return nil
}

func (n *node) Listen() {
	for {
		select {
		case <-n.ctx.Done():
			// exit the loop when the context is canceled (i.e., when Stop is called)
			n.log.Info().Msg("Stopping listening to incoming messages")
			return
		default:
			// listen for incoming messages
			pkt, err := n.conf.Socket.Recv(time.Second * 1)
			if errors.Is(err, transport.TimeoutError(0)) {
				// No message, just continue
				continue
			}
			if err != nil {
				n.log.Error().Err(err).Msg("Failed to receive message")
				continue
			}

			// determine if the message is for this node
			if pkt.Header.Destination == n.conf.Socket.GetAddress() {
				// if for this node, process it using the message registry
				err = n.ProcessMsg(pkt)
				if err != nil {
					n.log.Error().Err(err).Msg("Failed to process message")
				}
			} else {
				// if for another node, relay it
				n.RelayMsg(pkt)
			}
		}
	}
}

// Unicast implements peer.Messaging
func (n *node) Unicast(dest string, msg transport.Message) error {
	// If I send unicast to an unknown node, then I should get an error and the
	// other node should not receive the packet.
	rt := n.GetRoutingTable()
	// check if the destination is in the routing table
	relay, exists := rt[dest]
	if !exists {
		return xerrors.Errorf("destination %s is not in the routing table", dest)
	}

	// create packet
	header := transport.NewHeader(n.conf.Socket.GetAddress(), rt[n.conf.Socket.GetAddress()], dest)
	pkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}

	// send packet
	return n.conf.Socket.Send(relay, pkt, time.Second*1)
}

// Broadcast implements peer.Messaging
func (n *node) Broadcast(msg transport.Message) error {
	// Create a RumorsMessage containing one Rumor
	n.view.IncrementRumorSeq()
	rumor := types.Rumor{
		Origin:   n.conf.Socket.GetAddress(),
		Sequence: n.view.GetRumorSeq(),
		Msg:      &msg,
	}

	rumorsMsg := types.RumorsMessage{
		Rumors: []types.Rumor{rumor},
	}

	// save the rumor in the view
	n.view.AddRumorView(rumor, n.conf.Socket.GetAddress())

	// cast to transport.Message
	// marshal the RumorsMessage
	payload, err := n.conf.MessageRegistry.MarshalMessage(rumorsMsg)
	if err != nil {
		return xerrors.Errorf("failed to marshal RumorsMessage: %v", err)
	}

	packetHeader, err := n.SendRumorsMessageRand(payload, "")
	if err != nil {
		n.log.Error().Err(err).Msg("Failed to send RumorsMessage")
	}

	if n.conf.AckTimeout > 0 {
		// wait for ack
		// non-blocking GoRoutine
		go n.AckTicker(&packetHeader, payload)
	}

	go func() {
		// Process the message locally
		header := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress())
		pkt := transport.Packet{
			Header: &header,
			Msg:    &msg,
		}
		err = n.ProcessMsg(pkt)
		if err != nil {
			n.log.Error().Err(err).Msg("Failed to process message")
		}
		n.log.Info().Msg("Processed message locally")
	}()

	return nil
}

func (n *node) HeartbeatTicker() {
	// create a ticker to trigger heartbeats at the HeartbeatInterval
	heartbeatTicker := time.NewTicker(n.conf.HeartbeatInterval)
	defer heartbeatTicker.Stop() // Ensure the ticker is stopped when the goroutine exits

	for {
		select {
		case <-n.ctx.Done():
			// exit the loop when the context is canceled (i.e., when Stop is called)
			n.log.Info().Msg("Stopping heartbeat")
			return
		case <-heartbeatTicker.C:
			// send a heartbeat
			err := n.SendHeartbeat()
			if err != nil {
				n.log.Error().Err(err).Msg("Failed to send heartbeat")
			}
		}
	}
}

func (n *node) AntiEntropyTicker() {
	// create a ticker for anti-entropy at the AntiEntropyInterval
	antiEntropyTicker := time.NewTicker(n.conf.AntiEntropyInterval)
	defer antiEntropyTicker.Stop() // Ensure the ticker is stopped when the goroutine exits

	for {
		select {
		case <-n.ctx.Done():
			// exit the loop when the context is canceled (i.e., when Stop is called)
			n.log.Info().Msg("Stopping anti-entropy")
			return
		case <-antiEntropyTicker.C:
			// send a status message to a random neighbor
			neighbor := n.GetRandomNeighborFromRoutingTable("")
			n.log.Info().Msgf("Sending anti-entropy StatusMessage to %s", neighbor)

			peerSeq := n.view.GetPeerSeq()
			err := n.SendStatusMessage(neighbor, peerSeq)
			if err != nil {
				n.log.Error().Err(err).Msg("Failed to send anti-entropy StatusMessage")
			}

			// send CRDTMessage
		}
	}
}

func (n *node) AckTicker(packetHeader *transport.Header, payload transport.Message) {
	// create a new timer associated with a PacketID
	ticker := time.NewTimer(n.conf.AckTimeout)
	defer ticker.Stop()

	channel := make(chan bool, 1)
	n.SetAck(packetHeader.PacketID, channel)
	defer close(channel)
	defer n.DeleteAck(packetHeader.PacketID)

	n.log.Info().Msg("Created ticker")

	select {
	case <-channel:
		n.log.Info().Msg("Received Ack")
		return
	case <-ticker.C:
		n.log.Info().Msg("Ticker timed out")
		// if the ticker times out, resend the RumorsMessage to a random neighbor
		packetHeader, err := n.SendRumorsMessageRand(payload, packetHeader.Destination)
		if err != nil {
			n.log.Error().Err(err).Msg("Failed to send RumorsMessage")
		}

		n.log.Info().Msgf("Resent RumorsMessage with PacketID: %s", packetHeader.PacketID)
	}
}

// ProcessMsg handles the message if it's for this node.
func (n *node) ProcessMsg(pkt transport.Packet) error {
	err := n.conf.MessageRegistry.ProcessPacket(pkt)
	if err != nil {
		return xerrors.Errorf("failed to process message: %v", err)
	}
	return nil
}

// RelayMsg relays the message to its next hop.
func (n *node) RelayMsg(pkt transport.Packet) {
	// update the relayed by field
	pkt.Header.RelayedBy = n.conf.Socket.GetAddress()

	// find the next hop from the routing table
	relay := n.GetRoutingTable()[pkt.Header.Destination]
	err := n.conf.Socket.Send(relay, pkt, time.Second*1)
	if err != nil {
		n.log.Error().Err(err).Msg("Failed to relay message")
	}
}

// SendMsg sends a message to the destination.
func (n *node) SendMsg(dest string, msg types.Message) error {
	payload, err := n.conf.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return xerrors.Errorf("failed to marshal message: %v", err)
	}

	// create packet
	header := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), dest)
	pkt := transport.Packet{
		Header: &header,
		Msg:    &payload,
	}

	// send packet
	return n.conf.Socket.Send(dest, pkt, time.Second*1)
}
