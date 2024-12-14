package peer

import "Node-tion/backend/types"

// CRDT defines the interface for a Conflict-free CRDT update.
type CRDT interface {
	// GetEditor returns the editor of the CRDT.
	GetEditor() Editor

	// UpdateEditor updates the editor of the CRDT.
	UpdateEditor([]types.CRDTOperation) error

	// GetDocumentOps returns the document of the CRDT.
	GetDocumentOps(docID string) map[string][]types.CRDTOperation

	// GetBlockOps returns the block of the CRDT.
	GetBlockOps(docID, blockID string) []types.CRDTOperation

	// ApplyOperation applies a CRDT operation to the document.
	ApplyOperation(op types.CRDTOperation) error

	// StoreDocument stores the document as a text file in a directory.
	StoreDocument(docID, doc string) error

	// SaveTransactions saves a list of CRDT operations.
	SaveTransactions(transactions types.CRDTOperationsMessage) error

	// GetCRDTState returns the CRDT state of the document.
	GetCRDTState(docID string) uint64

	// CompileDocument compiles the document from the CRDT operations.
	CompileDocument(docID string) (string, error)

	GetTmpID(id uint64) uint64
}

// Editor tells, for a given document referenced by a key, a bag of blocks
// that are contained in the document; for a given block referenced by a key,
// a bag of CRDT operations that are contained in the block.
// For example:
//
//	{
//	  "doc1": {
//	    "block1": {op1, op2}, "block2": {op1, op2, op3}
//	  },
//	  ...
//	}
type Editor map[string]map[string][]types.CRDTOperation
