package unit

import (
	z "Node-tion/backend/internal/testing"
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

	crdtOp := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      sender.GetAddress(),
		OperationID: 1,
		DocumentID:  "doc1",
		BlockID:     "block1",
		Operation: types.CRDTAddBlock{
			BlockType: types.HeadingBlockType,
			Props: types.DefaultBlockProps{
				BackgroundColor: "white",
				TextColor:       "black",
				TextAlignment:   "left",
			},
		},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: []types.CRDTOperation{crdtOp},
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

	require.Len(t, receiver.GetDocumentOps("doc1"), 1)
	require.Len(t, receiver.GetBlockOps("doc1", "block1"), 1)

	require.Equal(t, crdtOp.OperationID, receiver.GetBlockOps("doc1", "block1")[0].OperationID)
	require.Equal(t, crdtOp.Origin, receiver.GetBlockOps("doc1", "block1")[0].Origin)

	require.Equal(t, crdtOp, receiver.GetBlockOps("doc1", "block1")[0])
}

// Check that the Editor can handle an update with multiple operations.
func Test_Editor_Multiple_Operations(t *testing.T) {
	transp := channel.NewTransport()

	receiver := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer receiver.Stop()

	sender, err := z.NewSenderSocket(transp, "127.0.0.1:0")
	require.NoError(t, err)

	receiver.AddPeer(sender.GetAddress())

	// sending multiple CRDT messages

	crdt1 := types.CRDTAddBlock{
		BlockType: types.ParagraphBlockType,
		Props:     types.DefaultBlockProps{},
	}

	crdtOp1 := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      sender.GetAddress(),
		OperationID: 1,
		DocumentID:  "doc1",
		BlockID:     "block1",
		Operation:   crdt1,
	}

	crdt2 := types.CRDTInsertChar{}

	crdtOp2 := types.CRDTOperation{
		Type:        types.CRDTInsertCharType,
		Origin:      sender.GetAddress(),
		OperationID: 2,
		DocumentID:  "doc1",
		BlockID:     "block1",
		Operation:   crdt2,
	}

	crdt3 := types.CRDTAddBlock{
		BlockType: types.ParagraphBlockType,
		Props:     types.DefaultBlockProps{},
	}

	crdtOp3 := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      sender.GetAddress(),
		OperationID: 3,
		DocumentID:  "doc1",
		BlockID:     "block2",
		Operation:   crdt3,
	}

	ops := []types.CRDTOperation{crdtOp1, crdtOp2, crdtOp3}

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

	require.Len(t, receiver.GetDocumentOps("doc1"), 2)
	require.Len(t, receiver.GetBlockOps("doc1", "block1"), 2)
	require.Len(t, receiver.GetBlockOps("doc1", "block2"), 1)

	require.Len(t, receiver.GetBlockOps("doc1", "block0"), 0) // No operations on block0
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

			crdtOp := types.CRDTOperation{
				Type:        types.CRDTAddBlockType,
				Origin:      node1.GetAddr(),
				OperationID: 1,
				DocumentID:  "doc1",
				BlockID:     "block1",
				Operation: types.CRDTAddBlock{
					BlockType: types.NumberedListBlockType,
					Props:     types.DefaultBlockProps{},
				},
			}

			crdtMsg := types.CRDTOperationsMessage{
				Operations: []types.CRDTOperation{crdtOp},
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

			require.Len(t, node1.GetDocumentOps("doc1"), 1)
			require.Len(t, node1.GetBlockOps("doc1", "block1"), 1)

			require.Len(t, node2.GetDocumentOps("doc1"), 1)
			require.Len(t, node2.GetBlockOps("doc1", "block1"), 1)

			require.Equal(t, crdtOp, node1.GetBlockOps("doc1", "block1")[0])

			require.Equal(t, crdtOp, node2.GetBlockOps("doc1", "block1")[0])
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

	crdtOp := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      node1.GetAddr(),
		OperationID: 1,
		DocumentID:  "doc1",
		BlockID:     "block1",
		Operation: types.CRDTAddBlock{
			BlockType: types.BulletedListBlockType,
			Props:     types.DefaultBlockProps{},
		},
	}

	crdtMsg := types.CRDTOperationsMessage{
		Operations: []types.CRDTOperation{crdtOp},
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

	require.Len(t, node1.GetDocumentOps("doc1"), 1)
	require.Len(t, node1.GetBlockOps("doc1", "block1"), 1)

	require.Len(t, node2.GetDocumentOps("doc1"), 1)
	require.Len(t, node2.GetBlockOps("doc1", "block1"), 1)

	require.Len(t, node3.GetDocumentOps("doc1"), 1)
	require.Len(t, node3.GetBlockOps("doc1", "block1"), 1)

	require.Equal(t, crdtOp, node1.GetBlockOps("doc1", "block1")[0])
	require.Equal(t, crdtOp, node2.GetBlockOps("doc1", "block1")[0])
	require.Equal(t, crdtOp, node3.GetBlockOps("doc1", "block1")[0])
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

	crdtOp1 := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      node1.GetAddr(),
		OperationID: 1,
		DocumentID:  "doc1",
		BlockID:     "block1",
		Operation: types.CRDTAddBlock{
			BlockType: types.ParagraphBlockType,
			Props:     types.DefaultBlockProps{},
		},
	}

	crdtMsg1 := types.CRDTOperationsMessage{
		Operations: []types.CRDTOperation{crdtOp1},
	}

	transpMsg1, err := node1.GetRegistry().MarshalMessage(&crdtMsg1)
	require.NoError(t, err)

	crdtOp2 := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      node3.GetAddr(),
		OperationID: 2,
		DocumentID:  "doc1",
		BlockID:     "block1",
		Operation: types.CRDTAddBlock{
			BlockType: types.ParagraphBlockType,
			Props:     types.DefaultBlockProps{},
		},
	}

	crdtMsg2 := types.CRDTOperationsMessage{
		Operations: []types.CRDTOperation{crdtOp2},
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

	require.Len(t, node1.GetDocumentOps("doc1"), 1)
	require.Len(t, node1.GetBlockOps("doc1", "block1"), 2)

	require.Len(t, node2.GetDocumentOps("doc1"), 1)
	require.Len(t, node2.GetBlockOps("doc1", "block1"), 2)

	require.Len(t, node3.GetDocumentOps("doc1"), 1)
	require.Len(t, node3.GetBlockOps("doc1", "block1"), 2)

	// > sort the operations by Origin (since OperationID is the same) and check that they are the same
	ops := node1.GetBlockOps("doc1", "block1")
	sorted1 := make([]types.CRDTOperation, len(ops))
	copy(sorted1, ops)
	sort.Slice(sorted1, func(i, j int) bool {
		return sorted1[i].Origin < sorted1[j].Origin
	})

	ops = node2.GetBlockOps("doc1", "block1")
	sorted2 := make([]types.CRDTOperation, len(ops))
	copy(sorted2, ops)
	sort.Slice(sorted2, func(i, j int) bool {
		return sorted2[i].Origin < sorted2[j].Origin
	})

	ops = node3.GetBlockOps("doc1", "block1")
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
