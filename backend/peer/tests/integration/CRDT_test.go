package integration

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/transport"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/transport/disrupted"
	"Node-tion/backend/types"
	"fmt"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
	"sync"
	"testing"
	"time"
)

// Test_CRDT_Integration_Pipeline runs the CRDT pipeline with a single node.
// SaveTransactions -> CRDTOperationsMessageCallback -> CompileDocument
//
// The document should contain 1 block with "Hello, World!".
func Test_CRDT_Integration_Pipeline(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node1.Stop()

	docID := "0@" + node1.GetAddr() // TODO
	helloWorld := "Hello World!"
	ops := generateStringOps(node1.GetAddr(), docID, helloWorld)

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	err := node1.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// Validate the document is compiled correctly
	_, err = node1.CompileDocument("0@" + node1.GetAddr())
	require.NoError(t, err)

	// TODO Yasmin's functions for checking the document json content
}

// Test_CRDT_Integration_Strong_Eventual_Consistency runs the CRDT pipeline with two nodes.
//
// A: "See you later, alligator!"
//
//	"See y" "ou later" ", alligator!"
//
// B: "In a while, crocodile!"
//
//	"In a " "while" ", croco" "dile!"
func Test_CRDT_Integration_Strong_Eventual_Consistency(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node1.Stop()

	node2 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())

	docID := "0" // TODO
	ops1 := generateStringOps(node1.GetAddr(), docID, "See you later, alligator!")
	ops2 := generateStringOps(node2.GetAddr(), docID, "In a while, crocodile!")

	// Break ops1 and ops2 into random chunks and send them to the nodes
	chunks1 := breakIntoChunks(ops1)
	chunks2 := breakIntoChunks(ops2)

	// Save the chunks to the nodes concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for _, chunk := range chunks1 {
			err := node1.SaveTransactions(types.CRDTOperationsMessage{Operations: chunk})
			require.NoError(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for _, chunk := range chunks2 {
			err := node2.SaveTransactions(types.CRDTOperationsMessage{Operations: chunk})
			require.NoError(t, err)
		}
	}()

	wg.Wait()

	time.Sleep(time.Millisecond * 200)

	// Validate the document is compiled correctly
	doc1, err := node1.CompileDocument(docID)
	require.NoError(t, err)

	doc2, err := node2.CompileDocument(docID)
	require.NoError(t, err)

	require.Equal(t, doc1, doc2)
}

// Test_CRDT_Integration_Scenario runs the CRDT pipeline with four nodes.
// Jammed, delayed, and disrupted nodes.
func Test_CRDT_Integration_Scenario(t *testing.T) {

	scenarios := func(transportA transport.Transport, transportB transport.Transport,
		transportC transport.Transport, transportD transport.Transport) func(*testing.T) {
		return func(t *testing.T) {
			setupFunc := setupNetwork(transportA, transportD)
			stages := []stage{
				setupFunc,
				writeContent,
				checkDocConsistency,
			}

			for i := 1; i < len(stages); i++ {
				maxStage := i
				t.Run(fmt.Sprintf("stage %d", i), func(t *testing.T) {
					t.Parallel()

					s := &state{t: t}
					defer stop(s)

					// iterating over all the stages, from 0 (setup) to maxStage (included)
					for k := 0; k < maxStage+1; k++ {
						stages[k](s)
					}
				})
			}
		}
	}

	t.Run("non-disrupted topology", scenarios(udpFac(), udpFac(), udpFac(), udpFac()))
	t.Run("Jammed nodes", scenarios(
		disrupted.NewDisrupted(udpFac(), disrupted.WithJam(time.Second, 8)),
		disrupted.NewDisrupted(udpFac(), disrupted.WithJam(time.Second, 8)),
		disrupted.NewDisrupted(udpFac(), disrupted.WithJam(time.Second, 8)),
		disrupted.NewDisrupted(udpFac(), disrupted.WithJam(time.Second, 8)),
	))
	t.Run("delayed nodes", scenarios(
		disrupted.NewDisrupted(udpFac(), disrupted.WithFixedDelay(500*time.Millisecond)),
		disrupted.NewDisrupted(udpFac(), disrupted.WithFixedDelay(500*time.Millisecond)),
		disrupted.NewDisrupted(udpFac(), disrupted.WithFixedDelay(500*time.Millisecond)),
		disrupted.NewDisrupted(udpFac(), disrupted.WithFixedDelay(500*time.Millisecond)),
	))
}

// => Stage 2
//
// Write to the same documents from all nodes.
func writeContent(s *state) *state {
	s.t.Log("~~ stage 2 <> write content ~~")

	docID1 := "doc1"
	docID2 := "doc2"
	content1 := "Content for document 1"
	extra1 := "Extra1"
	content2 := "Content for document 2"
	extra2 := "Extra2"

	ops1 := generateStringOps(s.nodes["nodeA"].GetAddr(), docID1, content1)
	ops2 := generateStringOps(s.nodes["nodeB"].GetAddr(), docID1, extra1)
	ops3 := generateStringOps(s.nodes["nodeC"].GetAddr(), docID2, content2)
	ops4 := generateStringOps(s.nodes["nodeD"].GetAddr(), docID2, extra2)

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for _, op := range ops1 {
			err := s.nodes["nodeA"].SaveTransactions(types.CRDTOperationsMessage{Operations: []types.CRDTOperation{op}})
			require.NoError(s.t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for _, op := range ops2 {
			err := s.nodes["nodeB"].SaveTransactions(types.CRDTOperationsMessage{Operations: []types.CRDTOperation{op}})
			require.NoError(s.t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for _, op := range ops3 {
			err := s.nodes["nodeC"].SaveTransactions(types.CRDTOperationsMessage{Operations: []types.CRDTOperation{op}})
			require.NoError(s.t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for _, op := range ops4 {
			err := s.nodes["nodeD"].SaveTransactions(types.CRDTOperationsMessage{Operations: []types.CRDTOperation{op}})
			require.NoError(s.t, err)
		}
	}()

	wg.Wait()

	time.Sleep(time.Millisecond * 200)

	return s
}

// => Stage 3
//
// Check the document consistency across all nodes.
func checkDocConsistency(s *state) *state {
	s.t.Log("~~ stage 3 <> check document consistency ~~")

	docID1 := "doc1"
	docID2 := "doc2"

	doc1A, err := s.nodes["nodeA"].CompileDocument(docID1)
	require.NoError(s.t, err)
	doc1B, err := s.nodes["nodeB"].CompileDocument(docID1)
	require.NoError(s.t, err)
	doc1C, err := s.nodes["nodeC"].CompileDocument(docID1)
	require.NoError(s.t, err)
	doc1D, err := s.nodes["nodeD"].CompileDocument(docID1)
	require.NoError(s.t, err)

	require.Equal(s.t, doc1A, doc1B)
	require.Equal(s.t, doc1B, doc1C)
	require.Equal(s.t, doc1C, doc1D)

	doc2A, err := s.nodes["nodeA"].CompileDocument(docID2)
	require.NoError(s.t, err)
	doc2B, err := s.nodes["nodeB"].CompileDocument(docID2)
	require.NoError(s.t, err)
	doc2C, err := s.nodes["nodeC"].CompileDocument(docID2)
	require.NoError(s.t, err)
	doc2D, err := s.nodes["nodeD"].CompileDocument(docID2)
	require.NoError(s.t, err)

	require.Equal(s.t, doc2A, doc2B)
	require.Equal(s.t, doc2B, doc2C)
	require.Equal(s.t, doc2C, doc2D)

	return s
}

func generateStringOps(addr, docID, str string) []types.CRDTOperation {
	blockID := "1@" + addr

	// Generate CRDTOperationsMessage
	crdtOp := types.CRDTOperation{
		Type:        types.CRDTAddBlock[types.ParagraphBlock]{}.Name(),
		BlockType:   types.ParagraphBlock{}.Name(),
		Origin:      addr,
		OperationId: 1,
		DocumentId:  docID,
		BlockId:     blockID,
		Operation:   types.CRDTAddBlock[types.ParagraphBlock]{},
	}

	ops := []types.CRDTOperation{crdtOp}

	for i, char := range str {
		blockID1 := fmt.Sprintf("%d@%s", i+2, addr)

		crdtOp := types.CRDTOperation{
			Type:        types.CRDTInsertChar{}.Name(),
			BlockType:   types.ParagraphBlock{}.Name(),
			Origin:      addr,
			OperationId: uint64(i + 2),
			DocumentId:  docID,
			BlockId:     blockID1,
			Operation: types.CRDTInsertChar{
				AfterID:   blockID,
				Character: string(char),
			},
		}

		blockID = blockID1

		ops = append(ops, crdtOp)
	}
	return ops
}

func breakIntoChunks(ops []types.CRDTOperation) [][]types.CRDTOperation {
	chunkSize := rand.Intn(len(ops)/2) + 1
	var chunks [][]types.CRDTOperation

	for i := 0; i < len(ops); i += chunkSize {
		end := i + chunkSize
		if end > len(ops) {
			end = len(ops)
		}
		chunks = append(chunks, ops[i:end])
	}

	return chunks
}
