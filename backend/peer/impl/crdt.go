package impl

import (
	"Node-tion/backend/types"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (n *node) ApplyOperation(op types.CRDTOperation) error {
	return nil
}

func (n *node) StoreDocument(docID, doc string) error {
	// Get the directory to store documents
	docDir := n.conf.DocumentDir

	// Generate the current timestamp
	currentTimestamp := time.Now().Unix()

	// Define the file name with the timestamp
	newFileName := fmt.Sprintf("%s_%d.txt", docID, currentTimestamp)
	newFilePath := filepath.Join(docDir, newFileName)

	// If a document has the corresponding newest timestamp, calculate if enough time has passed
	// Check if the new timestamp is greater than the existing one by the threshold
	if currentTimestamp <= n.docTimestampMap.GetNewestTimestamp(docID)+int64(n.conf.DocTimestampThreshold.Seconds()) {
		n.logCRDT.Info().Msg("not enough time has passed since the last document")
		return nil
	}

	oldestDoc := n.docTimestampMap.GetOldestDoc(docID)
	if oldestDoc != "" {
		if n.docTimestampMap.DocSavedLen(docID) >= n.conf.DocQueueSize {
			// Remove the oldest document if the threshold is reached
			if err := os.Remove(n.docTimestampMap.GetOldestDoc(docID)); err != nil {
				return fmt.Errorf("failed to remove oldest document: %w", err)
			}
		}
	}

	// Update the document timestamp map
	n.docTimestampMap.SetNewestTimestamp(docID, currentTimestamp)
	// Check if the document queue size is reached
	if n.docTimestampMap.DocSavedLen(docID) >= n.conf.DocQueueSize {
		n.docTimestampMap.DequeueDoc(docID)
	}
	n.docTimestampMap.EnqueueDoc(docID, newFilePath)

	n.logCRDT.Info().Msgf("storing document %s", newFilePath)
	// Save the new document
	if err := os.WriteFile(newFilePath, []byte(doc), 0600); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

func (n *node) SaveTransactions(transactions types.CRDTOperationsMessage) error {
	operations := transactions.Operations
	n.logCRDT.Debug().Msgf("SaveTransactions: %d operations", len(operations))

	// Step 1: Update CRDT states and initialize operations
	for i, operation := range operations {
		if err := n.updateCRDTState(&operation); err != nil {
			return err
		}
		operations[i] = operation
	}

	// Step 2: Update operation attributes
	for i, operation := range operations {
		if err := n.updateOperationAttributes(&operation); err != nil {
			return err
		}
		operations[i] = operation
	}

	transactions.Operations = operations

	// Reset temporary IDs
	n.crdtState.ResetTmp()

	// Step 3: Process and broadcast the operations
	return n.processAndBroadcast(transactions)
}

func (n *node) updateCRDTState(operation *types.CRDTOperation) error {
	opDocID := operation.DocumentID
	operation.Origin = n.conf.Socket.GetAddress()

	// Increment CRDT state for the document
	newState := n.crdtState.GetState(opDocID) + 1
	n.crdtState.SetState(opDocID, newState)

	tmp := operation.OperationID
	n.logCRDT.Debug().Msgf("updateCRDTState: %d -> %d", tmp, newState)
	n.crdtState.SetTmpID(tmp, newState)

	n.logCRDT.Debug().Msgf("tmp %d :", n.crdtState.GetTmpID(tmp))

	// Assign the new operation ID
	operation.OperationID = newState
	return nil
}

func (n *node) updateOperationAttributes(operation *types.CRDTOperation) error {
	switch op := operation.Operation.(type) {
	case types.CRDTAddBlock:
		return n.handleCRDTAddBlock(operation, op)
	case types.CRDTRemoveBlock:
		return n.handleCRDTRemoveBlock(operation, op)
	case types.CRDTUpdateBlock:
		return n.handleCRDTUpdateBlock(operation, op)
	case types.CRDTInsertChar:
		return n.handleCRDTInsertChar(operation, op)
	case types.CRDTAddMark:
		return n.handleCRDTAddMark(operation, op)
	case types.CRDTRemoveMark:
		return n.handleCRDTRemoveMark(operation, op)
	default:
		return fmt.Errorf("unknown CRDT operation type: %T", op)
	}
}

func (n *node) handleCRDTAddBlock(operation *types.CRDTOperation, op types.CRDTAddBlock) error {
	after, err1 := n.updateBlockReferences(&op.AfterBlock)
	parent, err2 := n.updateBlockReferences(&op.ParentBlock)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("failed to update block references: %w", err1)
	}
	op.AfterBlock = after
	op.ParentBlock = parent
	operation.Operation = op
	return nil
}

func (n *node) handleCRDTRemoveBlock(operation *types.CRDTOperation, op types.CRDTRemoveBlock) error {
	removed, err := n.updateBlockReferences(&op.RemovedBlock)
	if err != nil {
		return fmt.Errorf("failed to update block references: %w", err)
	}
	op.RemovedBlock = removed
	operation.Operation = op
	return nil
}

func (n *node) handleCRDTUpdateBlock(operation *types.CRDTOperation, op types.CRDTUpdateBlock) error {
	updated, err1 := n.updateBlockReferences(&op.UpdatedBlock)
	after, err2 := n.updateBlockReferences(&op.AfterBlock)
	parent, err3 := n.updateBlockReferences(&op.ParentBlock)
	if err1 != nil || err2 != nil || err3 != nil {
		return fmt.Errorf("failed to update block references: %w", err1)
	}
	op.UpdatedBlock = updated
	op.AfterBlock = after
	op.ParentBlock = parent
	operation.Operation = op
	return nil
}

func (n *node) handleCRDTInsertChar(operation *types.CRDTOperation, op types.CRDTInsertChar) error {
	block, err := n.updateBlockReferences(&op.AfterID)
	if err != nil {
		return fmt.Errorf("failed to update block references: %w", err)
	}
	op.AfterID = block
	operation.Operation = op
	return nil
}

func (n *node) handleCRDTAddMark(operation *types.CRDTOperation, op types.CRDTAddMark) error {
	start, err1 := n.updateBlockReferences(&op.Start.OpID)
	end, err2 := n.updateBlockReferences(&op.End.OpID)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("failed to update block references: %w", err1)
	}
	op.Start.OpID = start
	op.End.OpID = end
	operation.Operation = op
	return nil
}

func (n *node) handleCRDTRemoveMark(operation *types.CRDTOperation, op types.CRDTRemoveMark) error {
	start, err1 := n.updateBlockReferences(&op.Start.OpID)
	end, err2 := n.updateBlockReferences(&op.End.OpID)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("failed to update block references: %w", err1)
	}
	op.Start.OpID = start
	op.End.OpID = end
	operation.Operation = op
	return nil
}

func (n *node) updateBlockReferences(ref *string) (string, error) {
	if *ref == "" {
		n.logCRDT.Warn().Msg("updateBlockReferences: empty reference")
		return "", nil
	}
	id, username, err := ParseID(*ref)
	if err != nil {
		n.logCRDT.Error().Msgf("updateBlockReferences: %s", err)
		return "", err
	}
	id = n.crdtState.GetTmpID(id)
	res, err := ReconstructString(id, username)
	if err != nil {
		n.logCRDT.Error().Msgf("updateBlockReferences: %s", err)
		return "", err
	}
	n.logCRDT.Debug().Msgf("updateBlockReferences: %s -> %s", *ref, res)

	return res, nil
}

func (n *node) processAndBroadcast(transactions types.CRDTOperationsMessage) error {
	msg, err := n.conf.MessageRegistry.MarshalMessage(transactions)
	if err != nil {
		return err
	}
	return n.Broadcast(msg)
}

func (n *node) CompileDocument(docID string) (string, error) {
	return "", nil
}
