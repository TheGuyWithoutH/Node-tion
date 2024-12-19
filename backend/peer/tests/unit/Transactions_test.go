package unit

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/types"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test_SaveTransactions_SingleOperation verifies SaveTransactions with a single operation.
func Test_SaveTransactions_SingleOperation(t *testing.T) {
	transp := channel.NewTransport()

	node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer node.Stop()

	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer peer.Stop()

	node.AddPeer(peer.GetAddr())

	crdtOp := types.CRDTOperation{
		Origin:      node.GetAddr(),
		OperationID: 1, // Will be updated in SaveTransactions
		DocumentID:  "doc1",
		BlockID:     "1@temp",
		Operation:   types.CRDTAddBlock{},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: []types.CRDTOperation{crdtOp},
	}

	err := node.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// ValIDate the CRDT state is updated
	require.Equal(t, uint64(1), node.GetCRDTState("doc1"))

	// ValIDate the operation ID is updated and saved
	ops := node.GetBlockOps("doc1", "1@"+node.GetAddr())
	require.Len(t, ops, 1)
	require.Equal(t, uint64(1), ops[0].OperationID)

	// ValIDate the operation is broadcasted
	require.Len(t, node.GetOuts(), 1)
}

// Test_SaveTransactions_MultipleOperations verifies SaveTransactions with multiple operations.
func Test_SaveTransactions_MultipleOperations(t *testing.T) {
	transp := channel.NewTransport()

	node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer node.Stop()

	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer peer.Stop()

	node.AddPeer(peer.GetAddr())

	ops := []types.CRDTOperation{
		{
			Origin:      node.GetAddr(),
			OperationID: 1,
			DocumentID:  "doc1",
			BlockID:     "1@temp",
			Operation:   types.CRDTAddBlock{},
		},
		{
			Origin:      node.GetAddr(),
			OperationID: 2,
			DocumentID:  "doc1",
			BlockID:     "2@temp",
			Operation:   types.CRDTInsertChar{},
		},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	err := node.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// ValIDate the CRDT state is updated
	require.Equal(t, uint64(2), node.GetCRDTState("doc1"))

	// ValIDate operations are updated and saved
	require.Len(t, node.GetBlockOps("doc1", "1@"+node.GetAddr()), 1)
	require.Len(t, node.GetBlockOps("doc1", "2@"+node.GetAddr()), 1)
	require.Equal(t, uint64(1), node.GetBlockOps("doc1", "1@"+node.GetAddr())[0].OperationID)
	require.Equal(t, uint64(2), node.GetBlockOps("doc1", "2@"+node.GetAddr())[0].OperationID)

	// ValIDate the operations are broadcasted
	require.Len(t, node.GetOuts(), 1)
}

// Test_SaveTransactions_TempIDMapping verifies the temporary ID mapping functionality during SaveTransactions.
// Test_SaveTransactions_TempIDMappingWithEditor verifies the temporary ID mapping functionality and uses Editor for validation.
func Test_SaveTransactions_TempIDMappingWithEditor(t *testing.T) {
	transp := channel.NewTransport()

	node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer node.Stop()

	pe := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
	defer pe.Stop()

	node.AddPeer(pe.GetAddr())

	// Create operations with temporary IDs
	ops := []types.CRDTOperation{
		{
			Origin:      node.GetAddr(),
			OperationID: 42, // Temporary ID
			DocumentID:  "doc1",
			BlockID:     "42@temp",
			Operation: types.CRDTAddBlock{
				AfterBlock: "43@temp",
			},
		},
		{
			Origin:      node.GetAddr(),
			OperationID: 43, // Another Temporary ID
			DocumentID:  "doc1",
			BlockID:     "43@temp",
			Operation: types.CRDTInsertChar{
				AfterID: "42@temp",
			},
		},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	err := node.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	editor := node.GetEditor()

	require.Equal(t, editor["doc1"]["1@"+node.GetAddr()][0].OperationID, uint64(1))
	require.Equal(t, editor["doc1"]["2@"+node.GetAddr()][0].OperationID, uint64(2))

	println("checkpoint")

	bs, _ := json.Marshal(editor["doc1"]["1@"+node.GetAddr()][0].Operation)

	addOp := types.CRDTAddBlock{}

	err = json.Unmarshal(bs, &addOp)
	require.NoError(t, err)
	require.Equal(t, addOp.AfterBlock, "2@"+node.GetAddr())

}
