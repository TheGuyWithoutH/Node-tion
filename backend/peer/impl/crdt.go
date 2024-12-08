package impl

import (
	"Node-tion/backend/transport"
	"Node-tion/backend/types"
)

// ApplyOperation applies a CRDT operation to the document.
func (n *node) ApplyOperation(op types.CRDTOperation) error {
	return nil
}

func (n *node) SaveTransactions(transactions types.CRDTOperationsMessage) error {

	operations := transactions.Operations
	for i, operation := range operations {
		opDocId := operation.DocumentId

		// Update the CRDT state by incrementing document wide operation ids.
		n.crdtState.SetState(opDocId, n.crdtState.GetState(opDocId)+1)

		operation.OperationId = n.crdtState.GetState(opDocId)
		operations[i] = operation

	}

	transactions.Operations = operations

	// process the operations locally
	msg, err := n.conf.MessageRegistry.MarshalMessage(transactions)
	if err != nil {
		return err
	}
	header := transport.NewHeader(n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress(), n.conf.Socket.GetAddress())
	pkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}
	err = n.conf.MessageRegistry.ProcessPacket(pkt)
	if err != nil {
		return err
	}

	// broadcast the operations to other nodes
	return n.Broadcast(msg)
}
