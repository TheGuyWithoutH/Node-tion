package integration

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/transport"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/transport/disrupted"
	"Node-tion/backend/types"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

// Test_CRDT_Integration_Pipeline runs the CRDT pipeline with a single node.
// SaveTransactions -> CRDTOperationsMessageCallback -> CompileDocument
//
// The document should contain 1 block with "Hello, World!".
func Test_CRDT_Integration_Pipeline(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:8000")
	defer node1.Stop()

	docID := "0@" + node1.GetAddr()
	helloWorld := "Hello World!"
	ops := generateStringOps(node1.GetAddr(), docID, helloWorld)

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	err := node1.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// ValIDate the document is compiled correctly
	doc, err := node1.CompileDocument("0@" + node1.GetAddr())
	require.NoError(t, err)

	expectedDoc := "[{\"id\":\"1@127.0.0.1:8000\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"\",\"backgroundColor\":\"\",\"textAlignment\":\"\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@127.0.0.1:8000\",\"3@127.0.0.1:8000\",\"4@127.0.0.1:8000\",\"5@127.0.0.1:8000\",\"6@127.0.0.1:8000\",\"7@127.0.0.1:8000\",\"8@127.0.0.1:8000\",\"9@127.0.0.1:8000\",\"10@127.0.0.1:8000\",\"11@127.0.0.1:8000\",\"12@127.0.0.1:8000\",\"13@127.0.0.1:8000\"],\"text\":\"Hello World!\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expectedDoc, doc)
}

// Test_CRDT_Integration_Strong_Eventual_Consistency_Same_Block runs the CRDT pipeline with two nodes.
//
// A: create block
//
//	Sync
//
// B: "Hello, World!"
func Test_CRDT_Integration_Strong_Eventual_Consistency_Same_Block(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node1.Stop()

	node2 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())

	// > Create block

	docID := "doc1"
	blockID := "1@" + node1.GetAddr()
	ops := createNewBlockOp(node1.GetAddr(), docID, blockID)

	err := node1.SaveTransactions(types.CRDTOperationsMessage{Operations: ops})
	require.NoError(t, err)

	// > Sync

	time.Sleep(time.Millisecond * 200) // Wait for the block to be created and synced

	// > B: "Hello, World!"

	// assert that the block is created => editor of node2 has the block
	require.Condition(t, func() (success bool) {
		_, ok := node2.GetDocumentOps(docID)[blockID]
		return ok
	})

	ops = createInsertsFromString("Hello, World!", node2.GetAddr(), docID, blockID, 1)
	err = node2.SaveTransactions(types.CRDTOperationsMessage{Operations: ops})
	require.NoError(t, err)

	// > Sync

	time.Sleep(time.Millisecond * 200)

	doc1, err := node1.CompileDocument(docID)
	require.NoError(t, err)

	doc2, err := node2.CompileDocument(docID)
	require.NoError(t, err)

	require.Equal(t, doc1, doc2)
}

// Test_CRDT_Integration_Strong_Eventual_Consistency_Different_Blocks runs the CRDT pipeline with two nodes.
//
// A: "See you later, alligator!"
//
//	"See y" "ou later" ", alligator!"
//
// B: "In a while, crocodile!"
//
//	"In a " "while" ", croco" "dile!"
func Test_CRDT_Integration_Strong_Eventual_Consistency_Different_Blocks(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node1.Stop()

	node2 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())

	docID := "0"
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

	doc1, err := node1.CompileDocument(docID)
	require.NoError(t, err)

	doc2, err := node2.CompileDocument(docID)
	require.NoError(t, err)

	require.Equal(t, doc1, doc2)
}

// Test_CRDT_Integration_Strong_Eventual_Consistency_Same_Block_Concurrent_Edit
// runs the CRDT pipeline with two nodes concurrently editing the same block.
//
// A: create block
//
//	Sync
//
// B: "I am B." & A: "I am A."
func Test_CRDT_Integration_Strong_Eventual_Consistency_Same_Block_Concurrent_Edit(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node1.Stop()

	node2 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0")
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())

	// > Create block

	docID := "doc1"
	blockID := "1@" + node1.GetAddr()
	ops := createNewBlockOp(node1.GetAddr(), docID, blockID)

	err := node1.SaveTransactions(types.CRDTOperationsMessage{Operations: ops})
	require.NoError(t, err)

	// > Sync

	time.Sleep(time.Millisecond * 200) // Wait for the block to be created and synced

	// > B: "I am B." & A: "I am A."

	ops1 := createInsertsFromString("I am B.", node2.GetAddr(), docID, blockID, 1)
	ops2 := createInsertsFromString("I am A.", node1.GetAddr(), docID, blockID, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err = node2.SaveTransactions(types.CRDTOperationsMessage{Operations: ops1})
		require.NoError(t, err)
	}()

	go func() {
		defer wg.Done()
		err = node1.SaveTransactions(types.CRDTOperationsMessage{Operations: ops2})
		require.NoError(t, err)
	}()

	wg.Wait()

	// > Compare the documents

	doc1, err := node1.CompileDocument(docID)
	require.NoError(t, err)

	doc2, err := node2.CompileDocument(docID)
	require.NoError(t, err)

	require.Equal(t, doc1, doc2)
}

// Test_CRDT_Integration_Scenario_5_Nodes_With_Late_Joiners runs the CRDT pipeline with five nodes.
// Two nodes are late joiners.
func Test_CRDT_Integration_Scenario_5_Nodes_With_Late_Joiners(t *testing.T) {
	transp := channel.NewTransport()

	antiEntropy := time.Second * 10

	node1 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0", z.WithAntiEntropy(antiEntropy))
	defer node1.Stop()

	node2 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0", z.WithAntiEntropy(antiEntropy))
	defer node2.Stop()

	node3 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0", z.WithAntiEntropy(antiEntropy))
	defer node3.Stop()

	node1.AddPeer(node2.GetAddr())
	node1.AddPeer(node3.GetAddr())

	// > node1 creates a block & node2 creates a block

	docID := "doc1"
	blockID1 := "1@" + node1.GetAddr()
	blockID2 := "1@" + node2.GetAddr()
	ops1 := createNewBlockOp(node1.GetAddr(), docID, blockID1)
	ops2 := createNewBlockOp(node2.GetAddr(), docID, blockID2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := node1.SaveTransactions(types.CRDTOperationsMessage{Operations: ops1})
		require.NoError(t, err)
	}()

	go func() {
		defer wg.Done()
		err := node2.SaveTransactions(types.CRDTOperationsMessage{Operations: ops2})
		require.NoError(t, err)
	}()

	wg.Wait()

	// > Sync

	time.Sleep(time.Millisecond * 200)

	// > node1 writes in block1
	// > node2 & node3 write in block2

	ops1 = createInsertsFromString("Hello, World!", node1.GetAddr(), docID, blockID1, 1)
	ops2 = createInsertsFromString("In a while, crocodile!", node2.GetAddr(), docID, blockID2, 1)
	ops3 := createInsertsFromString("See you later, alligator!", node3.GetAddr(), docID, blockID2, 1)

	var wg2 sync.WaitGroup
	wg2.Add(3)

	go func() {
		defer wg2.Done()
		err := node1.SaveTransactions(types.CRDTOperationsMessage{Operations: ops1})
		require.NoError(t, err)
	}()

	go func() {
		defer wg2.Done()
		err := node2.SaveTransactions(types.CRDTOperationsMessage{Operations: ops2})
		require.NoError(t, err)
	}()

	go func() {
		defer wg2.Done()
		err := node3.SaveTransactions(types.CRDTOperationsMessage{Operations: ops3})
		require.NoError(t, err)
	}()

	wg2.Wait()

	// > node4 & node5 join the network

	node4 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0", z.WithAntiEntropy(antiEntropy))
	defer node4.Stop()

	node5 := z.NewTestNode(t, studentFac, transp, "127.0.0.1:0", z.WithAntiEntropy(antiEntropy))
	defer node5.Stop()

	node4.AddPeer(node1.GetAddr())
	node5.AddPeer(node1.GetAddr())

	// > Sync

	time.Sleep(time.Millisecond * 200)

	// > Compare the documents

	doc1, err := node1.CompileDocument(docID)
	require.NoError(t, err)

	doc2, err := node2.CompileDocument(docID)
	require.NoError(t, err)

	doc3, err := node3.CompileDocument(docID)
	require.NoError(t, err)

	doc4, err := node4.CompileDocument(docID)
	require.NoError(t, err)

	doc5, err := node5.CompileDocument(docID)
	require.NoError(t, err)

	require.Equal(t, doc1, doc2)
	require.Equal(t, doc2, doc3)
	require.Equal(t, doc3, doc4)
	require.Equal(t, doc4, doc5)
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
	t.Run("jammed nodes", scenarios(
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

// ----- Helper functions -----
func createInsertsFromString(content string, addr, docID, blockID string, insertStart int) []types.CRDTOperation {
	ops := make([]types.CRDTOperation, len(content))
	for i, char := range content {
		if i == 0 {
			ops[i] = types.CRDTOperation{
				Type:        types.CRDTInsertCharType,
				Origin:      addr,
				OperationID: uint64(i + insertStart),
				DocumentID:  docID,
				BlockID:     blockID,
				Operation:   createInsertOp("", string(char)),
			}
		} else {
			ops[i] = types.CRDTOperation{
				Type:        types.CRDTInsertCharType,
				Origin:      addr,
				OperationID: uint64(i + insertStart),
				DocumentID:  "doc1",
				BlockID:     blockID,
				Operation:   createInsertOp(strconv.Itoa(i+insertStart-1)+"@"+addr, string(char)),
			}
		}
	}
	return ops
}

func createInsertOp(afterID string, content string) types.CRDTInsertChar {
	return types.CRDTInsertChar{
		AfterID:   afterID,
		Character: content,
	}
}

func createNewBlockOp(addr, docID, blockID string) []types.CRDTOperation {
	crdtOp := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      addr,
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     blockID,
		Operation: types.CRDTAddBlock{
			BlockType: types.ParagraphBlockType,
			Props:     types.DefaultBlockProps{},
		},
	}

	ops := []types.CRDTOperation{crdtOp}
	return ops
}

func generateStringOps(addr, docID, str string) []types.CRDTOperation {
	blockID := "1@" + addr

	// Generate CRDTOperationsMessage
	crdtOp := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      addr,
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     blockID,
		Operation: types.CRDTAddBlock{
			BlockType: types.ParagraphBlockType,
			Props:     types.DefaultBlockProps{},
		},
	}

	ops := []types.CRDTOperation{crdtOp}

	prevOpID := fmt.Sprintf("%d@%s", crdtOp.OperationID, crdtOp.Origin)

	for i, char := range str {
		crdtOp := types.CRDTOperation{
			Type:        types.CRDTInsertCharType,
			Origin:      addr,
			OperationID: uint64(i + 2),
			DocumentID:  docID,
			BlockID:     blockID,
			Operation: types.CRDTInsertChar{
				AfterID:   prevOpID,
				Character: string(char),
			},
		}
		prevOpID = fmt.Sprintf("%d@%s", crdtOp.OperationID, crdtOp.Origin)

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
