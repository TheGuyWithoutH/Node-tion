package unit

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/types"
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
		OperationId: 0, // Will be updated in SaveTransactions
		DocumentId:  "doc1",
		BlockId:     "block1",
		Operation:   types.CRDTAddBlock[types.BlockType]{},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: []types.CRDTOperation{crdtOp},
	}

	err := node.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// Validate the CRDT state is updated
	require.Equal(t, uint64(1), node.GetCRDTState("doc1"))

	// Validate the operation ID is updated and saved
	ops := node.GetBlockOps("doc1", "block1")
	require.Len(t, ops, 1)
	require.Equal(t, uint64(1), ops[0].OperationId)

	// Validate the operation is broadcasted
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
			OperationId: 0,
			DocumentId:  "doc1",
			BlockId:     "block1",
			Operation:   types.CRDTAddBlock[types.BlockType]{},
		},
		{
			Origin:      node.GetAddr(),
			OperationId: 0,
			DocumentId:  "doc1",
			BlockId:     "block2",
			Operation:   types.CRDTInsertChar{},
		},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	err := node.SaveTransactions(crdtMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// Validate the CRDT state is updated
	require.Equal(t, uint64(2), node.GetCRDTState("doc1"))

	// Validate operations are updated and saved
	require.Len(t, node.GetBlockOps("doc1", "block1"), 1)
	require.Len(t, node.GetBlockOps("doc1", "block2"), 1)
	require.Equal(t, uint64(1), node.GetBlockOps("doc1", "block1")[0].OperationId)
	require.Equal(t, uint64(2), node.GetBlockOps("doc1", "block2")[0].OperationId)

	// Validate the operations are broadcasted
	require.Len(t, node.GetOuts(), 1)
}
