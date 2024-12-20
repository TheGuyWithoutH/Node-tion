package unit

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/peer/tests"
	"Node-tion/backend/transport"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/types"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Check that the Editor can handle a simple update.
func Test_Editor_Simple_Update(t *testing.T) {
	transp := channel.NewTransport()

	receiver := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer receiver.Stop()

	sender, err := z.NewSenderSocket(transp, "127.0.0.1:0")
	require.NoError(t, err)

	receiver.AddPeer(sender.GetAddress())

	// sending a CRDT message

	docID := "doc1"
	blockID := "block1"
	ops := tests.CreateNewBlockOp(sender.GetAddress(), docID, blockID)

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	transpMsg, err := receiver.GetRegistry().MarshalMessage(&crdtMsg)
	require.NoError(t, err)

	header := transport.NewHeader(sender.GetAddress(), sender.GetAddress(), receiver.GetAddr())

	packet := transport.Packet{
		Header: &header,
		Msg:    &transpMsg,
	}

	err = sender.Send(receiver.GetAddr(), packet, 0)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > editor must be updated

	require.Len(t, receiver.GetIns(), 1)
	require.Len(t, receiver.GetOuts(), 0)

	require.Len(t, receiver.GetDocumentOps(docID), 2) // 1 for the ops of BlockType and 1 for the block
	require.Len(t, receiver.GetBlockOps(docID, docID), 1)
	require.Len(t, receiver.GetBlockOps(docID, blockID), 0)

	require.Equal(t, ops[0].OperationID, receiver.GetBlockOps(docID, docID)[0].OperationID)
	require.Equal(t, ops[0].Origin, receiver.GetBlockOps(docID, docID)[0].Origin)

	require.Equal(t, ops[0], receiver.GetBlockOps(docID, docID)[0])
}

// Check that the Editor can handle an update with multiple operations.
//
// add block, insert char, add block
func Test_Editor_Multiple_Operations(t *testing.T) {
	transp := channel.NewTransport()

	receiver := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer receiver.Stop()

	sender, err := z.NewSenderSocket(transp, "127.0.0.1:0")
	require.NoError(t, err)

	receiver.AddPeer(sender.GetAddress())

	// sending multiple CRDT messages

	docID := "doc1"
	blockID1 := "block1"
	ops1 := tests.CreateNewBlockOp(sender.GetAddress(), docID, blockID1)

	ops2 := tests.CreateInsertsFromString("a", sender.GetAddress(), docID, blockID1, 1)

	blockID2 := "block2"
	ops3 := tests.CreateNewBlockOp(sender.GetAddress(), docID, blockID2)

	ops := append(ops1, ops2...)
	ops = append(ops, ops3...)

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	transpMsg, err := receiver.GetRegistry().MarshalMessage(&crdtMsg)
	require.NoError(t, err)

	header := transport.NewHeader(sender.GetAddress(), sender.GetAddress(), receiver.GetAddr())

	packet := transport.Packet{
		Header: &header,
		Msg:    &transpMsg,
	}

	err = sender.Send(receiver.GetAddr(), packet, 0)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > editor must be updated

	require.Len(t, receiver.GetIns(), 1)
	require.Len(t, receiver.GetOuts(), 0)

	require.Len(t, receiver.GetDocumentOps(docID), 3)        // 1 for the ops of BlockType and 2 for the blocks
	require.Len(t, receiver.GetBlockOps(docID, docID), 2)    // 2 add blocks
	require.Len(t, receiver.GetBlockOps(docID, blockID1), 1) // 1 insert char
	require.Len(t, receiver.GetBlockOps(docID, blockID2), 0) // No operations on block2
}

// Check that a Broadcast of CRDTOperationsMessage between two nodes works.
// Editors must be updated on both nodes.
//
// A -> B, A broadcasts a CRDTOperationsMessage to B.
func Test_Editor_Broadcast(t *testing.T) {

	getTest := func(transp transport.Transport) func(*testing.T) {
		return func(t *testing.T) {

			node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
			defer node1.Stop()

			node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0")
			defer node2.Stop()

			node1.AddPeer(node2.GetAddr())

			docID := "doc1"
			blockID := "block1"
			ops := tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)

			crdtMsg := types.CRDTOperationsMessage{
				Operations: ops,
			}

			transpMsg, err := node1.GetRegistry().MarshalMessage(&crdtMsg)
			require.NoError(t, err)

			err = node1.Broadcast(transpMsg)
			require.NoError(t, err)

			time.Sleep(time.Second * 2)

			n1Ins := node1.GetIns()
			n2Ins := node2.GetIns()

			n1Outs := node1.GetOuts()
			n2Outs := node2.GetOuts()

			// > n1 should have received an ack from n2

			require.Len(t, n1Ins, 1)
			pkt := n1Ins[0]
			require.Equal(t, "ack", pkt.Msg.Type)

			// > n2 should have received 1 rumor packet from n1

			require.Len(t, n2Ins, 1)

			pkt = n2Ins[0]
			require.Equal(t, node2.GetAddr(), pkt.Header.Destination)
			require.Equal(t, node1.GetAddr(), pkt.Header.RelayedBy)
			require.Equal(t, node1.GetAddr(), pkt.Header.Source)

			rumor := z.GetRumor(t, pkt.Msg)
			require.Len(t, rumor.Rumors, 1)
			r := rumor.Rumors[0]
			require.Equal(t, node1.GetAddr(), r.Origin)
			require.Equal(t, uint(1), r.Sequence) // must start with 1

			// > n1 should have sent 1 packet to n2

			require.Len(t, n1Outs, 1)
			require.Equal(t, node2.GetAddr(), pkt.Header.Destination)
			require.Equal(t, node1.GetAddr(), pkt.Header.RelayedBy)
			require.Equal(t, node1.GetAddr(), pkt.Header.Source)

			rumor = z.GetRumor(t, pkt.Msg)
			require.Len(t, rumor.Rumors, 1)
			r = rumor.Rumors[0]
			require.Equal(t, node1.GetAddr(), r.Origin)
			require.Equal(t, uint(1), r.Sequence)

			// > n2 should have sent an ack packet to n1

			require.Len(t, n2Outs, 1)

			pkt = n2Outs[0]
			ack := z.GetAck(t, pkt.Msg)
			require.Equal(t, n1Outs[0].Header.PacketID, ack.AckedPacketID)

			// >> node2 should have sent the following status to n1 {node1 => 1}

			require.Len(t, ack.Status, 1)
			require.Equal(t, uint(1), ack.Status[node1.GetAddr()])

			// > routing table of node1 should be updated

			routing := node1.GetRoutingTable()
			require.Len(t, routing, 2)

			entry, found := routing[node1.GetAddr()]
			require.True(t, found)

			require.Equal(t, node1.GetAddr(), entry)

			entry, found = routing[node2.GetAddr()]
			require.True(t, found)

			require.Equal(t, node2.GetAddr(), entry)

			// > routing table of node2 should be updated with node1

			routing = node2.GetRoutingTable()
			require.Len(t, routing, 2)

			entry, found = routing[node2.GetAddr()]
			require.True(t, found)

			require.Equal(t, node2.GetAddr(), entry)

			entry, found = routing[node1.GetAddr()]
			require.True(t, found)

			require.Equal(t, node1.GetAddr(), entry)

			// > Editor of node1 and node2 should be updated

			require.Len(t, node1.GetDocumentOps(docID), 2)
			require.Len(t, node1.GetBlockOps(docID, docID), 1)
			require.Len(t, node1.GetBlockOps(docID, blockID), 0)

			require.Len(t, node2.GetDocumentOps(docID), 2)
			require.Len(t, node2.GetBlockOps(docID, docID), 1)
			require.Len(t, node2.GetBlockOps(docID, blockID), 0)

			require.Equal(t, ops[0], node1.GetBlockOps(docID, docID)[0])

			require.Equal(t, ops[0], node2.GetBlockOps(docID, docID)[0])
		}
	}

	t.Run("channel transport", getTest(channelFac()))
	t.Run("UDP transport", getTest(udpFac()))
}

// Check that nodes have the same Editor after a Broadcast of CRDTOperationsMessage.
// B joins the network later.
//
// A -> B, A broadcasts a CRDTOperationsMessage to B.
// B joins the network later.
// A -> B -> C
func Test_Editor_Broadcast_CatchUp(t *testing.T) {

	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAntiEntropy(time.Millisecond*50))
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAntiEntropy(time.Millisecond*50), z.WithAutostart(false))

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAntiEntropy(time.Millisecond*50))
	defer node3.Stop()

	docID := "doc1"
	blockID := "block1"
	ops := tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)

	crdtMsg := types.CRDTOperationsMessage{
		Operations: ops,
	}

	transpMsg, err := node1.GetRegistry().MarshalMessage(&crdtMsg)
	require.NoError(t, err)

	err = node1.Broadcast(transpMsg)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	err = node2.Start()
	require.NoError(t, err)
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node3.GetAddr())

	time.Sleep(time.Millisecond * 500)

	// > check that every node has the same Editors

	require.Len(t, node1.GetDocumentOps(docID), 2)
	require.Len(t, node1.GetBlockOps(docID, docID), 1)
	require.Len(t, node1.GetBlockOps(docID, blockID), 0)

	require.Len(t, node2.GetDocumentOps(docID), 2)
	require.Len(t, node2.GetBlockOps(docID, docID), 1)
	require.Len(t, node2.GetBlockOps(docID, blockID), 0)

	require.Len(t, node3.GetDocumentOps(docID), 2)
	require.Len(t, node3.GetBlockOps(docID, docID), 1)
	require.Len(t, node3.GetBlockOps(docID, blockID), 0)

	require.Equal(t, ops[0], node1.GetBlockOps(docID, docID)[0])
	require.Equal(t, ops[0], node2.GetBlockOps(docID, docID)[0])
	require.Equal(t, ops[0], node3.GetBlockOps(docID, docID)[0])
}

// Check that Editors have the same sizes after a Broadcast of CRDTOperationsMessage
// from multiple nodes.
//
// A -> B
// C -> B
func Test_Editor_Broadcast_Two_Sides(t *testing.T) {

	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAntiEntropy(time.Millisecond*50))
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAntiEntropy(time.Millisecond*50))
	defer node2.Stop()

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAntiEntropy(time.Millisecond*50))
	defer node3.Stop()

	docID := "doc1"
	blockID := "block1"
	ops1 := tests.CreateNewBlockOp(node1.GetAddr(), docID, blockID)

	crdtMsg1 := types.CRDTOperationsMessage{
		Operations: ops1,
	}

	transpMsg1, err := node1.GetRegistry().MarshalMessage(&crdtMsg1)
	require.NoError(t, err)

	ops2 := tests.CreateNewBlockOp(node3.GetAddr(), docID, blockID)

	crdtMsg2 := types.CRDTOperationsMessage{
		Operations: ops2,
	}

	transpMsg2, err := node1.GetRegistry().MarshalMessage(&crdtMsg2)
	require.NoError(t, err)

	err = node1.Broadcast(transpMsg1)
	require.NoError(t, err)

	err = node3.Broadcast(transpMsg2)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 200)

	// A -> B, C -> B
	node1.AddPeer(node2.GetAddr())
	node3.AddPeer(node2.GetAddr())

	time.Sleep(time.Millisecond * 500)

	// > check that every node has the same Editors

	require.Len(t, node1.GetDocumentOps(docID), 2)
	require.Len(t, node1.GetBlockOps(docID, docID), 2)
	require.Len(t, node1.GetBlockOps(docID, blockID), 0)

	require.Len(t, node2.GetDocumentOps(docID), 2)
	require.Len(t, node2.GetBlockOps(docID, docID), 2)
	require.Len(t, node2.GetBlockOps(docID, blockID), 0)

	require.Len(t, node3.GetDocumentOps(docID), 2)
	require.Len(t, node3.GetBlockOps(docID, docID), 2)
	require.Len(t, node3.GetBlockOps(docID, blockID), 0)

	// > sort the operations by Origin (since OperationID is the same) and check that they are the same
	ops := node1.GetBlockOps(docID, docID)
	sorted1 := make([]types.CRDTOperation, len(ops))
	copy(sorted1, ops)
	sort.Slice(sorted1, func(i, j int) bool {
		return sorted1[i].Origin < sorted1[j].Origin
	})

	ops = node2.GetBlockOps(docID, docID)
	sorted2 := make([]types.CRDTOperation, len(ops))
	copy(sorted2, ops)
	sort.Slice(sorted2, func(i, j int) bool {
		return sorted2[i].Origin < sorted2[j].Origin
	})

	ops = node3.GetBlockOps(docID, docID)
	sorted3 := make([]types.CRDTOperation, len(ops))
	copy(sorted3, ops)
	sort.Slice(sorted3, func(i, j int) bool {
		return sorted3[i].Origin < sorted3[j].Origin
	})

	require.Equal(t, sorted1[0].Origin, sorted2[0].Origin)
	require.Equal(t, sorted1[0].Origin, sorted3[0].Origin)

	require.Equal(t, sorted1[1].Origin, sorted2[1].Origin)
	require.Equal(t, sorted1[1].Origin, sorted3[1].Origin)
}
