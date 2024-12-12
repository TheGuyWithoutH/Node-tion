package impl

import (
	"Node-tion/backend/peer"
	"Node-tion/backend/types"
	"crypto"
	"encoding/hex"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Set represents a collection of unique elements.
type Set[T comparable] struct {
	data map[T]struct{}
}

// NewSet creates and returns a new empty set.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{data: make(map[T]struct{})}
}

// Add adds an element to the set.
func (s *Set[T]) Add(value T) {
	s.data[value] = struct{}{}
}

// Remove removes an element from the set.
func (s *Set[T]) Remove(value T) {
	delete(s.data, value)
}

// Contains checks if an element is in the set.
func (s *Set[T]) Contains(value T) bool {
	_, exists := s.data[value]
	return exists
}

// Size returns the number of elements in the set.
func (s *Set[T]) Size() int {
	return len(s.data)
}

// Values returns all elements in the set as a slice.
func (s *Set[T]) Values() []T {
	keys := make([]T, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

// RoutingTable is a struct that holds the routing table of the node
type RoutingTable struct {
	mu sync.Mutex
	rt peer.RoutingTable
}

// GetRoutingTable implements peer.Messaging
func (n *node) GetRoutingTable() peer.RoutingTable {
	// concurrent read routing table
	n.routingTable.mu.Lock()
	defer n.routingTable.mu.Unlock()
	// create a copy of the routing table
	rt := make(peer.RoutingTable, len(n.routingTable.rt))
	for k, v := range n.routingTable.rt {
		rt[k] = v
	}
	return rt
}

// SetRoutingEntry implements peer.Messaging
func (n *node) SetRoutingEntry(origin, relayAddr string) {
	// concurrent set routing entry
	n.routingTable.mu.Lock()
	defer n.routingTable.mu.Unlock()

	if relayAddr == "" {
		delete(n.routingTable.rt, origin)
	} else {
		n.routingTable.rt[origin] = relayAddr
	}
}

// AddPeer implements peer.Messaging
func (n *node) AddPeer(addr ...string) {
	for _, a := range addr {
		if a == n.conf.Socket.GetAddress() {
			n.log.Info().Msg("Ignoring adding self to routing table")
			continue
		}
		rt := n.GetRoutingTable()
		if _, exists := rt[a]; !exists {
			n.SetRoutingEntry(a, a)
			n.log.Info().Msgf("Added peer %s to routing table for node %s", a, n.conf.Socket.GetAddress())
		} else {
			n.log.Info().Msgf("Peer %s already in routing table", a)
		}
	}
}

// GetNeighbors returns the neighbors of the node
func (n *node) GetNeighbors(excluding ...string) []string {
	rt := n.GetRoutingTable()

	// convert list to a map for O(1) lookups
	exclSet := make(map[string]struct{}, len(excluding))
	for _, e := range excluding {
		exclSet[e] = struct{}{}
	}

	var neighbors []string
	for k, v := range rt {
		if k != n.conf.Socket.GetAddress() && k == v {
			if _, excluded := exclSet[k]; !excluded {
				neighbors = append(neighbors, v)
			}
		}
	}
	return neighbors
}

// GetRandNeighsFromBudget returns a random subset of neighbors.
func (n *node) GetRandNeighsFromBudget(budget uint, excluding ...string) []string {
	neighbors := n.GetNeighbors(excluding...)
	if uint(len(neighbors)) > budget {
		return neighbors[:budget]
	}
	return neighbors
}

// GetRandomNeighborFromRoutingTable Function to get a random neighbor
func (n *node) GetRandomNeighborFromRoutingTable(source string) string {
	rt := n.GetRoutingTable()
	// Get the DIRECT neighbors
	var neighbors []string
	for k, v := range rt {
		// exclude:
		// - self
		// - source
		// - neighbors that are not direct neighbors
		if k != n.conf.Socket.GetAddress() && k != source && k == v { // exclude self and source
			neighbors = append(neighbors, v)
		}
	}

	// Choose a random neighbor using the neighbors slice
	// If there are no neighbors, return an error
	if len(neighbors) == 0 {
		return ""
	}

	// select a random entry (socket address) from the neighbors slice
	randomKey := rand.Intn(len(neighbors))
	n.log.Info().Msgf("Random neighbor: %s", neighbors[randomKey])
	return neighbors[randomKey]
}

// View is a struct that holds the view of the node
type View struct {
	mu       sync.Mutex
	rumorSeq uint                     // sequence number of the last rumor sent by this node
	peerSeq  map[string]uint          // map of peers to their last sequence number (includes self)
	rumors   map[string][]types.Rumor // map of peers to their rumors in ordered sequence
}

func (v *View) GetPeerSeq() map[string]uint {
	v.mu.Lock()
	defer v.mu.Unlock()

	peerSeq := make(map[string]uint)
	for k, v := range v.peerSeq {
		peerSeq[k] = v
	}
	return peerSeq
}

// IncrementRumorSeq increments the sequence number of the last rumor sent by this node
func (v *View) IncrementRumorSeq() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.rumorSeq++
}

// GetRumorSeq returns the sequence number of the last rumor sent by this node
func (v *View) GetRumorSeq() uint {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.rumorSeq
}

func (v *View) GetRumors() map[string][]types.Rumor {
	v.mu.Lock()
	defer v.mu.Unlock()

	rumors := make(map[string][]types.Rumor)
	for k, v := range v.rumors {
		rumors[k] = make([]types.Rumor, len(v))
		copy(rumors[k], v)
	}
	return rumors
}

func (v *View) AddRumorView(rumor types.Rumor, origin string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	// if expected sequence number, add to view
	if v.peerSeq[origin]+1 == rumor.Sequence {
		v.peerSeq[origin] = rumor.Sequence
		v.rumors[origin] = append(v.rumors[origin], rumor)

		return true
	}
	return false
}

// GetRumorView returns the rumors' slice for a peer
func (v *View) GetRumorView(origin string, init uint, end uint) []types.Rumor {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.rumors[origin][init:end]
}

// AckMap is a map of PacketID to bool
type AckMap struct {
	mu  sync.Mutex
	ack map[string]chan bool // map of PacketID to ticker
}

// GetAck returns the channel for a packet
func (n *node) GetAck(packetID string) (chan bool, bool) {
	n.ackTickers.mu.Lock()
	defer n.ackTickers.mu.Unlock()

	ticker, exists := n.ackTickers.ack[packetID]
	return ticker, exists
}

// SetAck sets the channel for a packet
func (n *node) SetAck(packetID string, ticker chan bool) {
	n.ackTickers.mu.Lock()
	defer n.ackTickers.mu.Unlock()

	n.ackTickers.ack[packetID] = ticker
}

// DeleteAck deletes the channel for a packet
func (n *node) DeleteAck(packetID string) {
	n.ackTickers.mu.Lock()
	defer n.ackTickers.mu.Unlock()

	delete(n.ackTickers.ack, packetID)
}

// DataReplyChanMap is a map of RequestID to reply channel
type DataReplyChanMap struct {
	mu   sync.Mutex
	repl map[string]chan []byte // map of RequestID to reply channel
}

// SetDataReplyChan sets the reply channel for a request
func (n *node) SetDataReplyChan(requestID string, replyChan chan []byte) {
	n.dataReplyChanMap.mu.Lock()
	defer n.dataReplyChanMap.mu.Unlock()

	n.dataReplyChanMap.repl[requestID] = replyChan
}

// DeleteDataReplyChan deletes the reply channel for a request
func (n *node) DeleteDataReplyChan(requestID string) {
	n.dataReplyChanMap.mu.Lock()
	defer n.dataReplyChanMap.mu.Unlock()

	delete(n.dataReplyChanMap.repl, requestID)
}

// SearchReplyChanMap is a map of RequestID to reply channel
type SearchReplyChanMap struct {
	mu   sync.Mutex
	repl map[string]chan string // map of RequestID to reply channel
}

// SetSearchReplyChan sets the reply channel for a search request
func (n *node) SetSearchReplyChan(requestID string, replyChan chan string) {
	n.searchReplyChanMap.mu.Lock()
	defer n.searchReplyChanMap.mu.Unlock()

	n.searchReplyChanMap.repl[requestID] = replyChan
}

// DeleteSearchReplyChan deletes the reply channel for a search request
func (n *node) DeleteSearchReplyChan(requestID string) {
	n.searchReplyChanMap.mu.Lock()
	defer n.searchReplyChanMap.mu.Unlock()

	delete(n.searchReplyChanMap.repl, requestID)
}

// Catalog is a struct that holds the catalog of the node
type Catalog struct {
	mu  sync.Mutex
	cat peer.Catalog
}

// GetCatalog implements peer.DataSharing
func (n *node) GetCatalog() peer.Catalog {
	n.catalog.mu.Lock()
	defer n.catalog.mu.Unlock()

	cat := make(peer.Catalog, len(n.catalog.cat))
	for k, v := range n.catalog.cat {
		cat[k] = v
	}
	return cat
}

// UpdateCatalog implements peer.DataSharing
func (n *node) UpdateCatalog(key string, peer string) {
	n.catalog.mu.Lock()
	defer n.catalog.mu.Unlock()

	//	{
	//	  "aef123": {
	//	    "127.0.0.1:3": {}, "127.0.0.1:2": {}
	//	  },
	//	  ...
	//	}

	// if the key does not exist in the catalog, create a new entry
	if _, exists := n.catalog.cat[key]; !exists {
		n.catalog.cat[key] = make(map[string]struct{})
		// add the peer to the catalog
		n.catalog.cat[key][peer] = struct{}{}
	}

	// if the key exists in the catalog, add the peer to the existing entry
	n.catalog.cat[key][peer] = struct{}{}
}

// RemovePeerFromCatalog removes a peer from the catalog
func (n *node) RemovePeerFromCatalog(key string, peer string) {
	n.catalog.mu.Lock()
	defer n.catalog.mu.Unlock()

	// if the key exists in the catalog, remove the peer from the entry
	if _, exists := n.catalog.cat[key]; exists {
		delete(n.catalog.cat[key], peer)
	}
}

func (n *node) GetRandomPeerFromCatalog(key string) string {
	n.catalog.mu.Lock()
	defer n.catalog.mu.Unlock()

	// get a random neighbor
	peers, exists := n.catalog.cat[key]
	if !exists || len(peers) == 0 {
		return ""
	}

	rndIdx := rand.Intn(len(peers))
	i := 0
	for k := range peers {
		if i == rndIdx {
			return k
		}
		i++
	}
	return ""
}

// HexEncode returns the hex-encoded hash of the data.
func (n *node) HexEncode(data []byte) string {
	h := crypto.SHA256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Requests is a struct that holds the requests of the node
type Requests struct {
	mu  sync.Mutex
	req map[string]time.Time
}

// AddRequest adds a request to the requests map
func (n *node) AddRequest(requestID string) bool {
	n.requests.mu.Lock()
	defer n.requests.mu.Unlock()

	// return if the request already exists
	if _, exists := n.requests.req[requestID]; exists {
		return true
	}
	n.requests.req[requestID] = time.Now()
	return false
}

// CreateBudgetMap creates a map of neighbors to their budget.
func (n *node) CreateBudgetMap(budget uint, numNeighbors int) map[int]uint {
	budgetMap := make(map[int]uint)

	base := budget / uint(numNeighbors)
	remainder := budget % uint(numNeighbors)

	for i := 0; i < numNeighbors; i++ {
		budgetMap[i] = base
	}

	for remainder > 0 {
		randomIndex := rand.Intn(numNeighbors)
		budgetMap[randomIndex]++
		remainder--
	}
	return budgetMap
}

// LogicalClock is a struct that holds the logical clock of the node
type LogicalClock struct {
	currentStep uint64
	maxID       uint64
}

// Increment increments the logical clock atomically
func (lc *LogicalClock) Increment() uint64 {
	return atomic.AddUint64(&lc.currentStep, 1)
}

// GetStep gets the current step of the logical clock
func (lc *LogicalClock) GetStep() uint64 {
	return atomic.LoadUint64(&lc.currentStep)
}

// ResetMaxID resets the max ID of the logical clock
func (lc *LogicalClock) ResetMaxID() {
	atomic.StoreUint64(&lc.maxID, 0)
}

// SetMaxID sets the max ID of the logical clock
func (lc *LogicalClock) SetMaxID(maxID uint64) {
	atomic.StoreUint64(&lc.maxID, maxID)
}

// GetMaxID gets the max ID of the logical clock
func (lc *LogicalClock) GetMaxID() uint64 {
	return atomic.LoadUint64(&lc.maxID)
}

// Acceptor is a struct that holds the accepted proposal of the node
type Acceptor struct {
	mu          sync.Mutex
	acceptedVal *types.PaxosValue
	acceptedID  uint
}

// Reset resets the acceptor
func (a *Acceptor) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.acceptedVal = nil
	a.acceptedID = 0
}

// SetAcceptedProposal sets the accepted proposal of the node
func (a *Acceptor) SetAcceptedProposal(val *types.PaxosValue, id uint) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.acceptedVal = val
	a.acceptedID = id
}

// GetAcceptedProposal gets the accepted proposal of the node
func (a *Acceptor) GetAcceptedProposal() (*types.PaxosValue, uint) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.acceptedVal, a.acceptedID
}

// Phase is an enum for the phase of the proposer
type Phase uint

// Proposer is a struct that holds the proposed proposal of the node
type Proposer struct {
	mu                 sync.Mutex
	phase              Phase
	proposalID         uint
	promisesCollected  uint
	collectingPromises chan bool
	highestAccepted    *Acceptor
	acceptedProposals  map[uint]uint
	collectingAccepts  chan bool
	consensus          types.PaxosValue
	tlcBroadcasted     bool
}

// Reset resets the proposer
func (p *Proposer) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.phase = 0
	p.highestAccepted = &Acceptor{}
	p.collectingAccepts = make(chan bool, 1)
	p.consensus = types.PaxosValue{}
	p.tlcBroadcasted = false
}

// GetPhase returns the phase ot the proposer
func (p *Proposer) GetPhase() Phase {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.phase
}

// SetPhase sets the phase of the proposer
func (p *Proposer) SetPhase(phase Phase) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.phase = phase
}

// GetProposalID returns the proposal ID
func (p *Proposer) GetProposalID() uint {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.proposalID
}

// SetProposalID sets the proposal ID
func (p *Proposer) SetProposalID(id uint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proposalID = id
}

// NewPromise increments the number of promises collected
func (p *Proposer) NewPromise() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.promisesCollected++
}

// ResetPromises resets the number of promises collected
func (p *Proposer) ResetPromises() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.promisesCollected = 0
}

// GetPromisesCollected returns the number of promises collected
func (p *Proposer) GetPromisesCollected() uint {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.promisesCollected
}

// GetHighestAccepted returns the highest accepted proposal
func (p *Proposer) GetHighestAccepted() (*types.PaxosValue, uint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.highestAccepted.GetAcceptedProposal()
}

// SetHighestAcceptedProposal sets the highest accepted proposal
func (p *Proposer) SetHighestAcceptedProposal(val *types.PaxosValue, id uint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.highestAccepted.SetAcceptedProposal(val, id)
}

// AddAcceptedProposal adds an accepted proposal
func (p *Proposer) AddAcceptedProposal(id uint) { //, val string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.acceptedProposals[id]; !exists {
		p.acceptedProposals[id] = 0
	}
	p.acceptedProposals[id]++
}

// LenAcceptedProposals returns the number of accepted proposals
func (p *Proposer) LenAcceptedProposals(id uint) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.acceptedProposals[id]; !exists {
		return 0
	}
	return int(p.acceptedProposals[id])
}

// GetConsensus returns the consensus value
func (p *Proposer) GetConsensus() types.PaxosValue {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.consensus
}

// SetConsensus sets the consensus value
func (p *Proposer) SetConsensus(consensus types.PaxosValue) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.consensus = consensus
}

// SetTLCBroadcasted sets the TLC broadcasted value
func (p *Proposer) SetTLCBroadcasted(broadcasted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tlcBroadcasted = broadcasted
}

// GetTLCBroadcasted returns the TLC broadcasted value
func (p *Proposer) GetTLCBroadcasted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.tlcBroadcasted
}

// TLC is a struct that holds the TLC of the node
type TLC struct {
	mu          sync.Mutex
	tlcMessages map[uint][]*types.TLCMessage // step to TLC messages
}

// AddTLCMessage adds a TLC message to the TLC
func (t *TLC) AddTLCMessage(msg *types.TLCMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	step := msg.Step

	if _, exists := t.tlcMessages[step]; !exists {
		t.tlcMessages[step] = make([]*types.TLCMessage, 0)
	}
	t.tlcMessages[step] = append(t.tlcMessages[step], msg)
}

// GetTLCMessages returns the TLC messages
func (t *TLC) GetTLCMessages(step uint) []*types.TLCMessage {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.tlcMessages[step]; !exists {
		return nil
	}
	return t.tlcMessages[step]
}

// LenTLCMessages returns the number of TLC messages
func (t *TLC) LenTLCMessages(step uint) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.tlcMessages[step]; !exists {
		return 0
	}
	return len(t.tlcMessages[step])
}

// Editor is a map of documents to blocks
type Editor struct {
	mu sync.Mutex
	ed peer.Editor
}

// GetEditor returns the editor of the CRDT
func (n *node) GetEditor() peer.Editor {
	n.editor.mu.Lock()
	defer n.editor.mu.Unlock()

	editor := make(peer.Editor, len(n.editor.ed))
	for k, v := range n.editor.ed {
		editor[k] = make(map[string][]types.CRDTOperation)
		for k1, v1 := range v {
			editor[k][k1] = make([]types.CRDTOperation, len(v1))
			copy(editor[k][k1], v1)
		}
	}
	return editor
}

// UpdateEditor updates the editor of the CRDT
func (n *node) UpdateEditor(ops []types.CRDTOperation) error {
	n.logCRDT.Debug().Msgf("UpdateEditor: %d operations", len(ops))
	n.editor.mu.Lock()
	defer n.editor.mu.Unlock()

	// apply the operation to the editor
	for _, op := range ops {
		if _, exists := n.editor.ed[op.DocumentId]; !exists {
			n.editor.ed[op.DocumentId] = make(map[string][]types.CRDTOperation)
		}

		if _, exists := n.editor.ed[op.DocumentId][op.BlockId]; !exists {
			n.editor.ed[op.DocumentId][op.BlockId] = make([]types.CRDTOperation, 0)
		}
		n.editor.ed[op.DocumentId][op.BlockId] = append(n.editor.ed[op.DocumentId][op.BlockId], op)
	}
	return nil
}

// GetDocumentOps returns the document of the CRDT
func (n *node) GetDocumentOps(docID string) map[string][]types.CRDTOperation {
	n.editor.mu.Lock()
	defer n.editor.mu.Unlock()

	doc := make(map[string][]types.CRDTOperation)
	for k, v := range n.editor.ed[docID] {
		doc[k] = make([]types.CRDTOperation, len(v))
		copy(doc[k], v)
	}
	return doc
}

// GetBlockOps returns the block of the CRDT
func (n *node) GetBlockOps(docID, blockID string) []types.CRDTOperation {
	n.editor.mu.Lock()
	defer n.editor.mu.Unlock()

	block := make([]types.CRDTOperation, len(n.editor.ed[docID][blockID]))
	copy(block, n.editor.ed[docID][blockID])
	return block
}

// DocTimestampMap is the latest timestamp of the document saved in the directory
// and the documents saved in the directory.
type DocTimestampMap struct {
	mu              sync.Mutex
	newestTimestamp map[string]int64
	docSaved        map[string][]string // docID -> [docT1, docT2, ...]
}

// GetNewestTimestamp returns the newest timestamp of the document.
func (dtm *DocTimestampMap) GetNewestTimestamp(docID string) int64 {
	dtm.mu.Lock()
	defer dtm.mu.Unlock()

	if _, exists := dtm.newestTimestamp[docID]; !exists {
		return 0
	}
	return dtm.newestTimestamp[docID]
}

// SetNewestTimestamp sets the newest timestamp of the document.
func (dtm *DocTimestampMap) SetNewestTimestamp(docID string, timestamp int64) {
	dtm.mu.Lock()
	defer dtm.mu.Unlock()
	dtm.newestTimestamp[docID] = timestamp
}

// GetOldestDoc returns the oldest document of the document.
func (dtm *DocTimestampMap) GetOldestDoc(docID string) string {
	dtm.mu.Lock()
	defer dtm.mu.Unlock()

	if _, exists := dtm.docSaved[docID]; !exists {
		return ""
	}
	return dtm.docSaved[docID][0]
}

// EnqueueDoc adds the newest document of the document.
func (dtm *DocTimestampMap) EnqueueDoc(docID, doc string) {
	dtm.mu.Lock()
	defer dtm.mu.Unlock()
	dtm.docSaved[docID] = append(dtm.docSaved[docID], doc)
}

// DequeueDoc removes the oldest document of the document.
func (dtm *DocTimestampMap) DequeueDoc(docID string) {
	dtm.mu.Lock()
	defer dtm.mu.Unlock()
	dtm.docSaved[docID] = dtm.docSaved[docID][1:]
}

// DocSavedLen returns the number of documents saved in the directory
// with the corresponding document ID.
func (dtm *DocTimestampMap) DocSavedLen(docID string) int {
	dtm.mu.Lock()
	defer dtm.mu.Unlock()
	return len(dtm.docSaved[docID])
}

type CRDTState struct {
	sync.Mutex
	state map[string]uint64
}

func (c *CRDTState) GetState(docID string) uint64 {
	c.Lock()
	defer c.Unlock()

	opId, exists := c.state[docID]
	if !exists {
		return 0
	}
	return opId

}

func (c *CRDTState) SetState(docID string, state uint64) {
	c.Lock()
	defer c.Unlock()

	c.state[docID] = state
}

func (n *node) GetCRDTState(docID string) uint64 {
	return n.crdtState.GetState(docID)
}
