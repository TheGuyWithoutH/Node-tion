package impl

import (
	"Node-tion/backend/types"
	"fmt"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

/*CompileDocument compiles the document requested from the editor into a json string.
 * Algorithm:
 * 1. Get the document editor.
 * 2. For each block in the editor, open a new block in the json string.
 * 3. For each op in the block, sort the ops by the afterID and then by the operation id.
 * 4. Apply the non mark operations.
 * 5. Apply the mark operations.
 */
func (n *node) CompileDocument(docID string) (string, error) {
	editor := n.GetDocumentOps(docID)
	if editor == nil {
		return "", xerrors.Errorf("document not found")
	}

	finalDoc := make(map[string]types.BlockType, len(editor))

	// Loop through the blocks of the document -> By order of BlockID
	// Subsequent blocks may be children and should therefore be added to the parent block

	blockIDs := make([]string, 0, len(editor))
	for blockID := range editor {
		blockIDs = append(blockIDs, blockID)
	}
	// Sort the blockIDs
	blockIDs = sortOpIDs(blockIDs)

	for id := range blockIDs {
		ops := editor[blockIDs[id]]
		// Filter the insert operations
		insertOps := n.FilterOps(ops, types.CRDTInsertCharType)
		// Sort the ops and remove the chars that are marked for deletion
		removeOps := n.FilterOps(ops, types.CRDTDeleteCharType)
		sortedChars := n.SortInsertOps(insertOps, removeOps)
		// Create a new block, this assumes that the first op is an addBlock op
		Op1 := ops[0]
		if Op1.Type != types.CRDTAddBlockType {
			return "", xerrors.Errorf("first operation must be a create block operation")
		}
		blockOp := Op1.Operation.(types.CRDTAddBlock)
		blockOp.OpID = strconv.FormatUint(Op1.OperationID, 10) + "@" + Op1.Origin

		block := n.CreateBlock(blockOp.BlockType, blockOp.Props, blockOp.OpID)

		// Mark Ops
		// Create a map opID -> textStyle
		textStyles := make(map[string]types.TextStyle, len(sortedChars))
		// Apply the addMark operations
		addMarkOps := n.FilterOps(ops, types.CRDTAddMarkType)
		for _, op := range addMarkOps {
			addMark := op.Operation.(types.CRDTAddMark)
			startFound := false
			for _, char := range sortedChars {
				if char.OpID == addMark.Start.OpID {
					startFound = true
				}
				if startFound {
					textStyles[char.OpID] = n.AddMark(textStyles[char.OpID], addMark)
				}
				if char.OpID == addMark.End.OpID {
					break
				}
			}
		}
		// Remove the marks
		deleteMarkOps := n.FilterOps(ops, types.CRDTRemoveMarkType)
		for _, op := range deleteMarkOps {
			deleteMark := op.Operation.(types.CRDTRemoveMark)
			startFound := false
			for _, char := range sortedChars {
				if char.OpID == deleteMark.Start.OpID {
					startFound = true
				}
				if startFound {
					textStyles[char.OpID] = n.RemoveMark(textStyles[char.OpID], deleteMark.MarkType)
				}
				if char.OpID == deleteMark.End.OpID {
					break
				}
			}
		}

		types.AddContent(block, sortedChars, textStyles)

		// Check if the block has parents
		if blockOp.ParentBlock != "" {
			// Find the parent block
			parentBlock := finalDoc[blockOp.ParentBlock]
			if parentBlock == nil {
				return "", xerrors.Errorf("parent block not found")
			}
			types.AddChildren(parentBlock, []types.BlockType{block})
		} else {
			finalDoc[blockOp.OpID] = block
		}
	}

	// Now that we have the final document, we can convert it to a json string
	finalJson := "[ "

	// We need to iterate over the blocks in the correct order:
	// Get the indices of the blocks and sort them by the block id
	docBlockOps := make([]string, 0, len(finalDoc))
	for opID := range finalDoc {
		docBlockOps = append(docBlockOps, opID)
	}
	// Sort the docBlockOps
	docBlockOps = sortOpIDs(docBlockOps)

	for i := range docBlockOps {
		n.logCRDT.Debug().Msgf("block %s being compiled", docBlockOps[i])
		block := finalDoc[docBlockOps[i]]
		finalJson += types.SerializeBlock(block) + ","
	}
	finalJson = finalJson[:len(finalJson)-1] // Remove the additional ","
	finalJson += "]"

	return finalJson, nil
}

func sortOpIDs(ops []string) []string {

	sort.Slice(ops, func(i, j int) bool {
		split1 := strings.Split(ops[i], "@")
		split2 := strings.Split(ops[j], "@")
		opID1, err := strconv.Atoi(split1[0])
		if err != nil {
			return false
		}
		opID2, err := strconv.Atoi(split2[0])
		if err != nil {
			return false
		}
		if opID1 == opID2 {
			return split1[1] < split2[1]
		}
		return opID1 < opID2
	})
	return ops

}

func (n *node) AddMark(textStyle types.TextStyle, toAdd types.CRDTAddMark) types.TextStyle {

	switch toAdd.MarkType {
	case types.Bold:
		textStyle.Bold = true
	case types.Italic:
		textStyle.Italic = true
	case types.Underline:
		textStyle.Underline = true
	case types.Strikethrough:
		textStyle.Strikethrough = true
	case types.TextColor:
		textStyle.TextColor = toAdd.Options.Color
	}

	return textStyle
}

func (n *node) RemoveMark(textStyle types.TextStyle, toRemove string) types.TextStyle {
	switch toRemove {
	case types.Bold:
		textStyle.Bold = false
	case types.Italic:
		textStyle.Italic = false
	case types.Underline:
		textStyle.Underline = false
	case types.Strikethrough:
		textStyle.Strikethrough = false
	}

	return textStyle
}

// FilterOps filters the insert operations from the operations.
func (n *node) FilterOps(ops []types.CRDTOperation, opType string) []types.CRDTOperation {
	var insertOps []types.CRDTOperation
	for _, op := range ops {
		if op.Type == opType {
			insertOps = append(insertOps, op)
		}
	}
	return insertOps
}

// SortInsertOps sorts the operations in the block by their afterID and then by their Operation id.
// It also removes the characters that are marked for deletion.
// Fills in the opID field of the insert operations
func (n *node) SortInsertOps(ops []types.CRDTOperation, toRemove []types.CRDTOperation) []types.CRDTInsertChar {
	sort.Slice(ops, func(i, j int) bool {
		// Cast the operations to the correct type
		insertOp1 := ops[i].Operation.(types.CRDTInsertChar)
		insertOp2 := ops[j].Operation.(types.CRDTInsertChar)

		if insertOp1.AfterID == "" {
			return true
		}

		if insertOp2.AfterID == "" {
			return false
		}

		split1 := strings.Split(insertOp1.AfterID, "@")
		afterOp1, err := strconv.Atoi(split1[0])
		if err != nil {
			n.logCRDT.Error().Msgf("failed to convert afterID to int: %s", err)
		}
		afterAddr1 := split1[1]

		split2 := strings.Split(insertOp2.AfterID, "@")
		afterOp2, err := strconv.Atoi(split2[0])
		if err != nil {
			n.logCRDT.Error().Msgf("failed to convert afterID to int: %s", err)
		}
		afterAddr2 := split2[1]

		if afterOp1 == afterOp2 { // AftersOpIDs are the same
			if afterAddr1 == afterAddr2 { // Addresses of the afterID are also the same
				// Compare the operation ids of the insert
				if ops[i].OperationID == ops[j].OperationID {
					return ops[i].Origin < ops[j].Origin
				}
				return ops[i].OperationID < ops[j].OperationID
			}

			return afterAddr1 < afterAddr2
		}

		return afterOp1 < afterOp2
	})

	// Turn the operations into a slice of CRDTInsertChar
	var insertOps []types.CRDTInsertChar
	for _, op := range ops {
		insertOp := op.Operation.(types.CRDTInsertChar)
		insertOp.OpID = strconv.FormatUint(op.OperationID, 10) + "@" + op.Origin
		insertOps = append(insertOps, insertOp)
	}

	// Remove the characters that are marked for deletion
	for _, op := range toRemove {
		// Cast the operation to the correct type
		removeOp := op.Operation.(types.CRDTDeleteChar)
		for i, insertOp := range insertOps {
			// Cast the operation to the correct type
			if insertOp.OpID == removeOp.RemovedID {
				insertOps = append(insertOps[:i], insertOps[i+1:]...)
				break
			}
		}
	}
	return insertOps
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

	// Step 0: Cast all interfaces to the respective types
	for _, operation := range operations {
		n.CastOperation(&operation)
	}

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

func (n *node) CreateBlock(blockType types.BlockTypeName, props types.DefaultBlockProps, blockId string) types.BlockType {
	switch blockType {
	case types.ParagraphBlockType:
		return &types.ParagraphBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockId,
			Content:   nil,
			Children:  nil,
		}
	case types.HeadingBlockType:
		return &types.HeadingBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockId,
			Level:     props.Level,
			Content:   nil,
			Children:  nil,
		}
	case types.BulletedListBlockType:
		return &types.BulletedListBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockId,
			Content:   nil,
			Children:  nil,
		}
	case types.NumberedListBlockType:
		return &types.NumberedListBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockId,
			Content:   nil,
			Children:  nil,
		}
	case types.ImageBlockType:
		return &types.ImageBlock{
			BlockType:    nil,
			Default:      props,
			ID:           blockId,
			URL:          "",
			Caption:      "",
			PreviewWidth: 0,
			Children:     nil,
		}
	case types.TableBlockType:
		return &types.TableBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockId,
			Content:   types.TableContent{},
			Children:  nil,
		}

	default:
		return nil
	}
}
