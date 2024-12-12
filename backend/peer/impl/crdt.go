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
	if err := os.WriteFile(newFilePath, []byte(doc), 0644); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

func (n *node) SaveTransactions(transactions types.CRDTOperationsMessage) error {

	operations := transactions.Operations
	n.logCRDT.Debug().Msgf("SaveTransactions: %d operations", len(operations))
	for i, operation := range operations {
		opDocId := operation.DocumentId

		// Update the CRDT state by incrementing document wide operation ids.
		n.crdtState.SetState(opDocId, n.crdtState.GetState(opDocId)+1)

		operation.OperationId = n.crdtState.GetState(opDocId)
		operations[i] = operation

	}

	transactions.Operations = operations

	// Process the operations locally
	msg, err := n.conf.MessageRegistry.MarshalMessage(transactions)
	if err != nil {
		return err
	}

	// Broadcast the operations to other nodes
	return n.Broadcast(msg)
}
