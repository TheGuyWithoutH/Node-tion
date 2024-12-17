package impl

import (
	"Node-tion/backend/types"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

/*CompileDocument compiles the document requested from the editor into a JSON string.
 * Algorithm:
 * 1. Get the document editor.
 * 2. For each block in the editor, open a new block in the JSON string.
 * 3. For each op in the block, sort the ops by the afterID and then by the operation id.
 * 4. Apply the non mark operations.
 * 5. Apply the mark operations.
 */
func (n *node) CompileDocumentOld(docID string) (string, error) {
	editor := n.GetDocumentOps(docID)
	if editor == nil {
		return "", xerrors.Errorf("document not found")
	}

	finalDoc := make(map[string]types.BlockType, len(editor))
	var CRDTAddBlockOps []types.CRDTOperation
	childrenAddBlockOps := make(map[string][]types.CRDTOperation) //parentBlock -> addBlockOpChildren

	// Loop through the blocks of the document
	// Subsequent blocks may be children and should therefore be added to the parent block
	for _, blockOps := range editor {
		// Filter the insert operations
		insertOps, updatedBlock, removed := n.filterOps(blockOps, types.CRDTInsertCharType)
		if removed {
			n.logCRDT.Debug().Msgf("block removed")
			continue
		}

		// Sort the blockOps and remove the chars that are marked for deletion
		removeOps, _, _ := n.filterOps(blockOps, types.CRDTDeleteCharType)
		sortedChars, err := n.sortInsertOps(insertOps, removeOps)
		if err != nil {
			return "", xerrors.Errorf("failed to sort insert operations: %v", err)
		}

		// ---------- Block Ops
		// Create a new block, this assumes that the first op is an addBlock op
		Op1 := blockOps[0]
		if Op1.Type != types.CRDTAddBlockType {
			return "", xerrors.Errorf("first operation must be a create block operation")
		}
		blockOp, ok := Op1.Operation.(types.CRDTAddBlock)
		if !ok {
			return "", xerrors.Errorf("failed to cast operation to CRDTAddBlock")
		}
		opID, err := ReconstructOpID(Op1.OperationID, Op1.Origin)
		if err != nil {
			return "", xerrors.Errorf("failed to convert operationID to string: %v", err)
		}
		blockOp.OpID = opID
		blockOp = n.updateBlock(blockOp, updatedBlock) // Updates the block with the updated block props if applicable

		block := n.createBlock(blockOp.BlockType, blockOp.Props, blockOp.OpID)

		// ---------- Mark Ops
		// Create a map opID -> textStyle
		textStyles := make(map[string]types.TextStyle, len(sortedChars))
		// Apply the addMark operations
		addMarkOps, _, _ := n.filterOps(blockOps, types.CRDTAddMarkType)
		for _, op := range addMarkOps {
			addMark, ok := op.Operation.(types.CRDTAddMark)
			if !ok {
				return "", xerrors.Errorf("failed to cast operation to CRDTAddMark")
			}
			startFound := false
			for _, char := range sortedChars {
				if char.OpID == addMark.Start.OpID {
					startFound = true
				}
				if startFound {
					textStyles[char.OpID] = n.addMark2TextStyle(textStyles[char.OpID], addMark)
				}
				if char.OpID == addMark.End.OpID {
					break
				}
			}
		}
		// Remove the marks
		deleteMarkOps, _, _ := n.filterOps(blockOps, types.CRDTRemoveMarkType)
		for _, op := range deleteMarkOps {
			deleteMark, ok := op.Operation.(types.CRDTRemoveMark)
			if !ok {
				return "", xerrors.Errorf("failed to cast operation to CRDTRemoveMark")
			}
			startFound := false
			for _, char := range sortedChars {
				if char.OpID == deleteMark.Start.OpID {
					startFound = true
				}
				if startFound {
					textStyles[char.OpID] = n.removeMark2TextStyle(textStyles[char.OpID], deleteMark.MarkType)
				}
				if char.OpID == deleteMark.End.OpID {
					break
				}
			}
		}

		// ----- Adding the content to the block
		types.AddContent(block, sortedChars, textStyles)
		finalDoc[blockOp.OpID] = block
		n.logCRDT.Debug().Msgf("block %s added to finalDoc", blockOp.OpID)

		// Check if the block has parents
		if blockOp.ParentBlock != "" {
			// Add the block to the parent-children map to be sorted and added later
			childrenAddBlockOps[blockOp.ParentBlock] = append(childrenAddBlockOps[blockOp.ParentBlock], Op1)
		} else {
			CRDTAddBlockOps = append(CRDTAddBlockOps, Op1)
		}
	}

	// Add the children blocks to the parent blocks
	// For each parent block (Iterating over the keys)
	for parentID, addBlocks := range childrenAddBlockOps {
		// Sort the add blocks in the correct generation order
		sortedChildrenBlockIDs := n.sortAddBlockOpIDs(addBlocks)
		n.logCRDT.Debug().Msgf("Parent Block %s : Sorted children blockIDs %s", parentID, sortedChildrenBlockIDs)
		// For each child block, add it to the parent block
		parentBlock := finalDoc[parentID]
		for _, childID := range sortedChildrenBlockIDs {
			childBlock := finalDoc[childID]
			// Add the child block to the parent block
			types.AddChildren(parentBlock, []types.BlockType{childBlock})
			n.logCRDT.Debug().Msgf("block %s added to parent block %s", childID, parentID)
			// Delete the child block from the final document
			delete(finalDoc, childID)
			n.logCRDT.Debug().Msgf("block %s removed from finalDoc", childID)
		}

	}

	// Now that we have the final document, we can convert it to a JSON string
	finalJSON := "[ "

	// We need to iterate over the blocks in the correct order:
	// Get the indices of the blocks and sort them by the block id
	docBlockOps := n.sortAddBlockOpIDs(CRDTAddBlockOps)
	n.logCRDT.Info().Msgf("Sorted blockIDs %s", docBlockOps)

	for _, blockID := range docBlockOps {
		n.logCRDT.Debug().Msgf("block %s being compiled", blockID)
		block := finalDoc[blockID]
		finalJSON += types.SerializeBlock(block) + ","
	}
	finalJSON = finalJSON[:len(finalJSON)-1] // Remove the additional ","
	finalJSON += "]"

	return finalJSON, nil
}

func (n *node) getIDIndex(ID string, charIDs []string) int {
	pos := -1
	for i, charID := range charIDs {
		if charID == ID {
			pos = i
			break
		}
	}
	return pos
}

// Map is by reference TODO: Check if this is correct
func (n *node) applyAddMark(textStyles map[string]types.TextStyle, charIDs []string, startID, endID string, op types.CRDTAddMark) {

	startFound := false
	for _, charID := range charIDs {
		if charID == startID {
			startFound = true
		}
		if startFound {
			textStyles[charID] = n.addMark2TextStyle(textStyles[charID], op)
		}
		if charID == endID {
			break
		}
	}
}

func (n *node) applyRemoveMark(textStyles map[string]types.TextStyle, charIDs []string, startID, endID string, op types.CRDTRemoveMark) {

	startFound := false
	for _, charID := range charIDs {
		if charID == startID {
			startFound = true
		}
		if startFound {
			textStyles[charID] = n.removeMark2TextStyle(textStyles[charID], op.MarkType)
		}
		if charID == endID {
			break
		}
	}
}

func (n *node) createBlockContent(ops []types.CRDTOperation) []types.InlineContent {

	var text string
	textStyles := make(map[string]types.TextStyle) // opID -> textStyle
	var charIDs []string
	var removedIDs []string

	n.logCRDT.Debug().Msgf("ops %v", ops)

	for _, op := range ops {
		opID, err := ReconstructOpID(op.OperationID, op.Origin)
		if err != nil {
			n.logCRDT.Error().Msgf("failed to convert operationID to string: %v", err)
		}
		n.logCRDT.Debug().Msgf("Operation %v", op.Type)
		switch op.Type {
		case types.CRDTInsertCharType:
			insertOp, ok := op.Operation.(types.CRDTInsertChar)
			if !ok {
				n.logCRDT.Error().Msgf("failed to cast operation to CRDTInsertChar")
			}
			// Insert the character into the text after the AfterID
			afterID := insertOp.AfterID

			var lastAfterID string
			if len(charIDs) > 0 {
				lastAfterID = charIDs[len(charIDs)-1]
			}
			if afterID != lastAfterID {
				// Get the position index of afterID in charIDs
				pos := n.getIDIndex(afterID, charIDs)
				if pos == -1 {
					n.logCRDT.Error().Msgf("failed to find afterID in charIDs")
				}
				// Insert the character at the position
				charIDs = append(charIDs[:pos+1], append([]string{opID}, charIDs[pos+1:]...)...)
				text = text[:pos+1] + insertOp.Character + text[pos+1:]
			} else {
				// Add the character to the end of the text
				n.logCRDT.Debug().Msgf("Inserting character %s", insertOp.Character)
				text += insertOp.Character
				charIDs = append(charIDs, opID)
			}
			n.logCRDT.Debug().Msgf("charIDs %v", charIDs)
			n.logCRDT.Debug().Msgf("text %s", text)

		case types.CRDTDeleteCharType:
			deleteOp, ok := op.Operation.(types.CRDTDeleteChar)
			if !ok {
				n.logCRDT.Error().Msgf("failed to cast operation to CRDTDeleteChar")
			}
			removedIDs = append(removedIDs, deleteOp.RemovedID)

		case types.CRDTAddMarkType:
			op, ok := op.Operation.(types.CRDTAddMark)
			if !ok {
				n.logCRDT.Error().Msgf("failed to cast operation to CRDTAddMark")
			}
			startID := op.Start.OpID
			endID := op.End.OpID
			n.applyAddMark(textStyles, charIDs, startID, endID, op)

		case types.CRDTRemoveMarkType:
			op, ok := op.Operation.(types.CRDTRemoveMark)
			if !ok {
				n.logCRDT.Error().Msgf("failed to cast operation to CRDTRemoveMark")
			}
			startID := op.Start.OpID
			endID := op.End.OpID
			n.applyRemoveMark(textStyles, charIDs, startID, endID, op)
		}
	}

	// Removes the characters that are marked for deletion
	for _, removedID := range removedIDs {
		pos := n.getIDIndex(removedID, charIDs)
		if pos == -1 {
			n.logCRDT.Error().Msgf("failed to find removedID in charIDs")
		}
		charIDs = append(charIDs[:pos], charIDs[pos+1:]...)
		text = text[:pos] + text[pos+1:]
	}

	return n.generateInlineContent(text, textStyles, charIDs)
}

func (n *node) generateInlineContent(text string, textStyles map[string]types.TextStyle, charIDs []string) []types.InlineContent {
	var styledTexts []types.StyledText
	// If the style is the same, we can group the characters together
	var previousStyles types.TextStyle
	var tmpStringContent string
	var tmpCharIds []string

	n.logCRDT.Debug().Msgf("charIDs %v", charIDs)
	n.logCRDT.Debug().Msgf("textStyles %v", textStyles)
	n.logCRDT.Debug().Msgf("text %v", text)

	for _, charID := range charIDs {
		if !compareTextStyle(textStyles[charID], previousStyles) {
			// If the style is different, we need to create a new InlineContent
			if len(tmpStringContent) > 0 {
				styledTexts = append(styledTexts, types.StyledText{
					CharIDs: tmpCharIds,
					Text:    tmpStringContent,
					Styles:  previousStyles,
				})
				// Reset the stringContent
				tmpStringContent = ""
				tmpCharIds = nil
			}
		}
		tmpStringContent += string(text[n.getIDIndex(charID, charIDs)])
		tmpCharIds = append(tmpCharIds, charID)
		previousStyles = textStyles[charID]
	}

	// We need to add the last block of text
	if len(tmpStringContent) > 0 {
		styledTexts = append(styledTexts, types.StyledText{
			CharIDs: tmpCharIds,
			Text:    tmpStringContent,
			Styles:  previousStyles,
		})
	}

	var inlineContents = make([]types.InlineContent, len(styledTexts))
	n.logCRDT.Debug().Msgf("Length styledTexts %v", styledTexts)
	for i, styledText := range styledTexts {
		n.logCRDT.Debug().Msgf("styledText %v", styledText)
		inlineContents[i] = &styledText
	}

	return inlineContents
}

// TODO : Add it as a function of the TextStyle struct
func compareTextStyle(a types.TextStyle, b types.TextStyle) bool {
	if a.Bold != b.Bold || a.Italic != b.Italic || a.Underline != b.Underline ||
		a.Strikethrough != b.Strikethrough || a.TextColor != b.TextColor ||
		a.BackgroundColor != b.BackgroundColor {
		return false
	}

	return true
}

func (n *node) CompileDocument(docID string) (string, error) {
	document := make([]types.BlockFactory, 0)

	// Step 1: Populate document blocks in order
	blockChangeOperations := n.GetDocumentOps(docID)[docID]

	for _, blockChangeOp := range blockChangeOperations {
		// Determine the type of operation
		switch blockChangeOp.Type {
		case types.CRDTAddBlockType:
			addBlockOp, ok := blockChangeOp.Operation.(types.CRDTAddBlock)
			if !ok {
				return "", xerrors.Errorf("failed to cast operation to CRDTAddBlock")
			}
			addBlockOp.OpID = fmt.Sprintf("%d@%s", blockChangeOp.OperationID, blockChangeOp.Origin)

			// Add the block to the document in the correct spot
			added := false
			if len(document) == 0 {
				// If the document is empty, add the block to the beginning
				newBlock := types.BlockFactory{
					ID:        addBlockOp.OpID,
					BlockType: addBlockOp.BlockType,
					Props:     addBlockOp.Props,
					Children:  nil,
				}
				document = append(document, newBlock)
			} else {
				for i := range document {
					added, document = n.checkAddBlockAtPosition(document, i, addBlockOp)
					if added {
						break
					}
				}
			}
		case types.CRDTRemoveBlockType:
			removeBlockOp, ok := blockChangeOp.Operation.(types.CRDTRemoveBlock)
			if !ok {
				return "", xerrors.Errorf("failed to cast operation to CRDTRemoveBlock")
			}
			removeBlockOp.OpID = fmt.Sprintf("%d@%s", blockChangeOp.OperationID, blockChangeOp.Origin)

			// Remove the block from the document
			removed := false
			for i := range document {
				removed, document = checkRemoveBlockAtPosition(document, i, removeBlockOp)
				if removed {
					break
				}
			}
		case types.CRDTUpdateBlockType:
			updateBlockOp, ok := blockChangeOp.Operation.(types.CRDTUpdateBlock)
			if !ok {
				return "", xerrors.Errorf("failed to cast operation to CRDTUpdateBlock")
			}
			updateBlockOp.UpdatedBlock = blockChangeOp.BlockID

			// Find the block to update and remove it for now (as it can change position)
			oldBlock := &types.BlockFactory{}
			for i := range document {
				oldBlock, document = n.findBlockToUpdateAndRemove(document, i, updateBlockOp)
				if oldBlock != nil {
					break
				}
			}

			// Update the block properties
			if oldBlock != nil {
				updatedBlock := &types.BlockFactory{
					ID:        oldBlock.ID,
					BlockType: updateBlockOp.BlockType, // We assume that the block type is always updated
					Props:     n.updateBlockProps(oldBlock.Props, updateBlockOp.Props),
					Children:  oldBlock.Children,
				}

				added := false

				// Add the block back to the document
				for i := range document {
					added, document = n.checkAddBackBlockAtPosition(document, i, updateBlockOp, *updatedBlock)
					if added {
						break
					}
				}
			}
		}
	}

	// Step 2: Populate block content for each block in the document
	finalDocument := make([]types.BlockType, 0)
	n.logCRDT.Debug().Msgf("document %s being compiled, factory: %v", docID, document)

	for _, block := range document {
		// Skip deleted blocks
		if block.Deleted {
			continue
		}
		n.logCRDT.Debug().Msgf("block %s being compiled, factory: %v", block.ID, block)
		// Create the block
		blockOperations := n.GetBlockOps(docID, block.ID)
		n.logCRDT.Debug().Msgf("block %s being compiled, ops: %v", block.ID, blockOperations)
		newBlock := n.createBlock(docID, block, blockOperations)
		n.logCRDT.Debug().Msgf("block %s added to finalDoc", block.ID)
		finalDocument = append(finalDocument, newBlock)
	}

	// Step 3: Serialize the document
	// Now that we have the final document, we can convert it to a JSON string
	finalJSON := "[ "

	// Serialize the document
	for _, block := range finalDocument {
		finalJSON += types.SerializeBlock(block) + ","
	}

	finalJSON = finalJSON[:len(finalJSON)-1] // Remove the additional ","
	finalJSON += "]"

	return finalJSON, nil
}

func (n *node) createBlock(docID string, block types.BlockFactory, blockOperations []types.CRDTOperation) types.BlockType {
	// Create the children blocks if applicable
	var childrenBlocks []types.BlockType

	if block.Children != nil {
		for _, childBlock := range block.Children {
			childBlockOperations := n.GetBlockOps(docID, childBlock.ID)
			childrenBlocks = append(childrenBlocks, n.createBlock(docID, childBlock, childBlockOperations))
		}
	}

	// Create the block based on the block type and props, populate the content and children
	switch block.BlockType {
	case types.ParagraphBlockType:
		newBlock := &types.ParagraphBlock{
			BlockType: nil,
			Default:   block.Props,
			ID:        block.ID,
			Content:   n.createBlockContent(blockOperations),
			Children:  childrenBlocks,
		}
		return newBlock
	case types.HeadingBlockType:
		newBlock := &types.HeadingBlock{
			BlockType: nil,
			Default:   block.Props,
			ID:        block.ID,
			Level:     block.Props.Level,
			Content:   n.createBlockContent(blockOperations),
			Children:  childrenBlocks,
		}
		return newBlock
	case types.BulletedListBlockType:
		newBlock := &types.BulletedListBlock{
			BlockType: nil,
			Default:   block.Props,
			ID:        block.ID,
			Content:   n.createBlockContent(blockOperations),
			Children:  childrenBlocks,
		}
		return newBlock
	case types.NumberedListBlockType:
		newBlock := &types.NumberedListBlock{
			BlockType: nil,
			Default:   block.Props,
			ID:        block.ID,
			Content:   n.createBlockContent(blockOperations),
			Children:  childrenBlocks,
		}
		return newBlock
	case types.ImageBlockType:
		newBlock := &types.ImageBlock{
			BlockType:    nil,
			Default:      block.Props,
			ID:           block.ID,
			URL:          "",
			Caption:      "",
			PreviewWidth: 0,
			Children:     nil,
		}
		return newBlock
	case types.TableBlockType:
		newBlock := &types.TableBlock{
			BlockType: nil,
			Default:   block.Props,
			ID:        block.ID,
			Content:   types.TableContent{},
			Children:  nil,
		}
		return newBlock
	default:
		return nil
	}
}

// checkAddBlockAtPosition checks if the addBlockOp should be added to the document at the current index
// Returns true if the block was added, false otherwise
func (n *node) checkAddBlockAtPosition(document []types.BlockFactory, index int, addBlockOp types.CRDTAddBlock) (bool, []types.BlockFactory) {
	added := false

	// Check if the block is a child block
	if addBlockOp.ParentBlock != "" && addBlockOp.ParentBlock == document[index].ID {
		// Check if the block has no children yet
		if document[index].Children == nil {
			document[index].Children = make([]types.BlockFactory, 0)

			newBlock := types.BlockFactory{
				ID:        addBlockOp.OpID,
				BlockType: addBlockOp.BlockType,
				Props:     addBlockOp.Props,
				Children:  nil,
			}
			document[index].Children = append(document[index].Children, newBlock)
			added = true
		} else if addBlockOp.AfterBlock == "" {
			// Add the block to the start of the children
			newBlock := types.BlockFactory{
				ID:        addBlockOp.OpID,
				BlockType: addBlockOp.BlockType,
				Props:     addBlockOp.Props,
				Children:  nil,
			}
			document[index].Children = append([]types.BlockFactory{newBlock}, document[index].Children...)
			added = true
		} else {
			// Check where to add the block in the children
			for i := range document[index].Children {
				added, document[index].Children = n.checkAddBlockAtPosition(document[index].Children, i, addBlockOp)
				if added {
					break
				}
			}
		}
	}

	// Check if the block is going at the start of the document
	if !added && (addBlockOp.AfterBlock == "" && addBlockOp.ParentBlock == "") {
		newBlock := types.BlockFactory{
			ID:        addBlockOp.OpID,
			BlockType: addBlockOp.BlockType,
			Props:     addBlockOp.Props,
			Children:  nil,
		}
		document = append([]types.BlockFactory{newBlock}, document...)
		added = true
	} else if !added && (document[index].ID == addBlockOp.AfterBlock) {
		// Check if the block is going after the current block
		newBlock := types.BlockFactory{
			ID:        addBlockOp.OpID,
			BlockType: addBlockOp.BlockType,
			Props:     addBlockOp.Props,
			Children:  nil,
		}
		document = append(document[:index+1], append([]types.BlockFactory{newBlock}, document[index+1:]...)...)
		added = true
	} else if !added {
		if document[index].Children != nil {
			for i := range document[index].Children {
				// Recursively check the children blocks
				added, document[index].Children = n.checkAddBlockAtPosition(document[index].Children, i, addBlockOp)
				if added {
					break
				}
			}
		}
	}

	n.logCRDT.Debug().Msgf("block %s added to finalDoc in CheckAddBlockPos", addBlockOp.OpID)
	n.logCRDT.Debug().Msgf("document %v", document)
	return added, document
}

// checkRemoveBlockAtPosition checks if the removeBlockOp should be removed from the document at the current index
// Returns true if the block was removed, false otherwise
func checkRemoveBlockAtPosition(document []types.BlockFactory, index int, removeBlockOp types.CRDTRemoveBlock) (bool, []types.BlockFactory) {
	removed := false

	// Check if the block is going after the current block
	if document[index].ID == removeBlockOp.RemovedBlock {
		document[index].Deleted = true
		removed = true
	}

	// Check if the block is a child block
	if document[index].Children != nil {
		for i := range document[index].Children {
			return checkRemoveBlockAtPosition(document[index].Children, i, removeBlockOp)
		}
	}

	return removed, document
}

func (n *node) findBlockToUpdateAndRemove(document []types.BlockFactory, index int, updateBlockOp types.CRDTUpdateBlock) (*types.BlockFactory, []types.BlockFactory) {
	var updated *types.BlockFactory = nil

	// Check if the block is going after the current block
	if document[index].ID == updateBlockOp.UpdatedBlock {
		// Copy the block to be updated to avoid the reference being updated
		oldBlock := document[index]
		updated = &oldBlock
		document = append(document[:index], document[index+1:]...)
	}

	// Check if the block is a child block
	if document[index].Children != nil {
		for i := range document[index].Children {
			updated, document[index].Children = n.findBlockToUpdateAndRemove(document[index].Children, i, updateBlockOp)
			if updated != nil {
				break
			}
		}
	}

	return updated, document
}

// checkAddBlockAtPosition checks if the addBlockOp should be added to the document at the current index
// Returns true if the block was added, false otherwise
func (n *node) checkAddBackBlockAtPosition(document []types.BlockFactory, index int, updateBlockOp types.CRDTUpdateBlock, updatedBlock types.BlockFactory) (bool, []types.BlockFactory) {
	added := false

	// Check if the block is a child block
	if updateBlockOp.ParentBlock != "" && updateBlockOp.ParentBlock == document[index].ID {
		// Check if the block has no children yet
		if document[index].Children == nil {
			document[index].Children = make([]types.BlockFactory, 0)
			document[index].Children = append(document[index].Children, updatedBlock)
			added = true
		} else if updateBlockOp.AfterBlock == "" {
			// Add the block to the start of the children
			document[index].Children = append([]types.BlockFactory{updatedBlock}, document[index].Children...)
			added = true
		} else {
			// Check where to add the block in the children
			for i := range document[index].Children {
				added, document[index].Children = n.checkAddBackBlockAtPosition(document[index].Children, i, updateBlockOp, updatedBlock)
				if added {
					break
				}
			}
		}
	}

	// Check if the block is going at the start of the document
	if !added && (updateBlockOp.AfterBlock == "" && updateBlockOp.ParentBlock == "") {
		document = append([]types.BlockFactory{updatedBlock}, document...)
		added = true
	} else if !added && (document[index].ID == updateBlockOp.AfterBlock) {
		// Check if the block is going after the current block
		document = append(document[:index+1], append([]types.BlockFactory{updatedBlock}, document[index+1:]...)...)
		added = true
	} else if !added {
		if document[index].Children != nil {
			for i := range document[index].Children {
				// Recursively check the children blocks
				added, document[index].Children = n.checkAddBackBlockAtPosition(document[index].Children, i, updateBlockOp, updatedBlock)
				if added {
					break
				}
			}
		}
	}

	return added, document
}

func (n *node) updateBlockProps(blockProps types.DefaultBlockProps, updatedProps types.DefaultBlockProps) types.DefaultBlockProps {

	if updatedProps.Level != 0 {
		blockProps.Level = updatedProps.Level
	}
	if updatedProps.BackgroundColor != "" {
		blockProps.BackgroundColor = updatedProps.BackgroundColor
	}
	if updatedProps.TextColor != "" {
		blockProps.TextColor = updatedProps.TextColor
	}
	if updatedProps.TextAlignment != "" {
		blockProps.TextAlignment = updatedProps.TextAlignment
	}

	return blockProps
}

func (n *node) updateBlock(blockOp types.CRDTAddBlock, updatedBlock *types.CRDTUpdateBlock) types.CRDTAddBlock {
	if updatedBlock != nil {
		blockOp.Props = n.updateBlockProps(blockOp.Props, updatedBlock.Props)
		if updatedBlock.BlockType != "" {
			blockOp.BlockType = updatedBlock.BlockType
		}
		if updatedBlock.AfterBlock != "" {
			blockOp.AfterBlock = updatedBlock.AfterBlock
		}
		if updatedBlock.ParentBlock != "" {
			blockOp.ParentBlock = updatedBlock.ParentBlock
		}
	}
	return blockOp
}

func (n *node) addMark2TextStyle(textStyle types.TextStyle, toAdd types.CRDTAddMark) types.TextStyle {

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
	case types.BackgroundColor:
		textStyle.BackgroundColor = toAdd.Options.Color
	}

	return textStyle
}

func (n *node) removeMark2TextStyle(textStyle types.TextStyle, toRemove string) types.TextStyle {
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

// filterOps filters the opType operations from the block's operations op and checks if the block is removed or updated
// Returns the filtered operations, the UpdateBlockOp if applicable and  a boolean indicating if the block is removed
func (n *node) filterOps(ops []types.CRDTOperation, opType string) ([]types.CRDTOperation, *types.CRDTUpdateBlock, bool) {
	var filteredOps []types.CRDTOperation
	var updateBlockOp types.CRDTUpdateBlock
	ok := false
	for _, op := range ops {
		if op.Type == types.CRDTRemoveBlockType {
			return nil, &updateBlockOp, true
		}
		if op.Type == opType {
			filteredOps = append(filteredOps, op)
		}
		if op.Type == types.CRDTUpdateBlockType {
			updateBlockOp, ok = op.Operation.(types.CRDTUpdateBlock)
			if !ok {
				n.logCRDT.Error().Msgf("failed to cast operation to CRDTUpdateBlock")
			}
		}
	}
	return filteredOps, &updateBlockOp, false
}

// SortAddBlockOpIDs sorts the operations in the block by their afterBlockID and then by their Operation id.
// Returns the blockIDs in the correct order of generation
func (n *node) sortAddBlockOpIDs(ops []types.CRDTOperation) []string {

	sort.Slice(ops, func(i, j int) bool {
		// Cast the operations to the correct type
		addBlockOp1, ok := ops[i].Operation.(types.CRDTAddBlock)
		if !ok {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTAddBlock")
		}
		addBlockOp2, ok := ops[j].Operation.(types.CRDTAddBlock)
		if !ok {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTAddBlock")
		}

		split1 := strings.Split(addBlockOp1.AfterBlock, "@")
		afterOp1, err := strconv.Atoi(split1[0])
		afterAddr1 := split1[1]
		if err != nil {
			n.logCRDT.Error().Msgf("failed to convert afterID to int: %s", err)
			afterOp1 = 0
			afterAddr1 = ""
		}

		split2 := strings.Split(addBlockOp2.AfterBlock, "@")
		afterOp2, err := strconv.Atoi(split2[0])
		afterAddr2 := split2[1]
		if err != nil {
			n.logCRDT.Error().Msgf("failed to convert afterID to int: %s", err)
			afterOp2 = 0
			afterAddr2 = ""
		}

		if afterOp1 == afterOp2 { // AftersOpIDs are the same
			if afterAddr1 == afterAddr2 { // Addresses of the afterID are also the same
				// Compare the operation ids of the insert
				if ops[i].OperationID == ops[j].OperationID {
					return ops[i].Origin < ops[j].Origin
				}
				return ops[i].OperationID > ops[j].OperationID
			}
		}

		if addBlockOp1.AfterBlock == "" {
			return true
		}
		if addBlockOp2.AfterBlock == "" {
			return false
		}

		return afterOp1 < afterOp2
	})

	// Turn the operations into a slice of blockIDs and CRDTADDBlock
	var blockIDs []string
	for _, op := range ops {
		blockID, err := ReconstructOpID(op.OperationID, op.Origin)
		if err != nil {
			n.logCRDT.Error().Msgf("failed to reconstruct opID: %s", err)
		}
		blockIDs = append(blockIDs, blockID)

		addBlockOp, ok := op.Operation.(types.CRDTAddBlock)
		if !ok {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTAddBlock")
		}
		n.logCRDT.Debug().Msgf("blockID %s : AfterBlock %s", blockID, addBlockOp.AfterBlock)
	}

	return blockIDs
}

// sortInsertOps sorts the operations in the block by their afterID and then by their Operation id.
// It also removes the characters that are marked for deletion.
// Fills in the opID field of the insert operations
func (n *node) sortInsertOps(ops []types.CRDTOperation, toRemove []types.CRDTOperation) ([]types.CRDTInsertChar, error) {
	sort.Slice(ops, func(i, j int) bool {
		// Cast the operations to the correct type
		insertOp1, ok := ops[i].Operation.(types.CRDTInsertChar)
		if !ok {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTInsertChar: %v", insertOp1)
		}
		insertOp2, ok := ops[j].Operation.(types.CRDTInsertChar)
		if !ok {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTInsertChar: %v", insertOp2)
		}

		//TODO: Add these lines at the end
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
		afterOp2, err2 := strconv.Atoi(split2[0])
		if err2 != nil {
			n.logCRDT.Error().Msgf("failed to convert afterID to int: %s", err)
		}
		afterAddr2 := split2[1]

		if afterOp1 == afterOp2 { // AftersOpIDs are the same
			if afterAddr1 == afterAddr2 { // Addresses of the afterID are also the same
				// Compare the operation ids of the insert
				if ops[i].OperationID == ops[j].OperationID {
					return ops[i].Origin < ops[j].Origin
				}
				return ops[i].OperationID > ops[j].OperationID
			}

			return afterAddr1 < afterAddr2
		}

		return afterOp1 < afterOp2
	})

	// Turn the operations into a slice of CRDTInsertChar
	var insertOps []types.CRDTInsertChar
	for _, op := range ops {
		insertOp, ok := op.Operation.(types.CRDTInsertChar)
		if !ok {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTInsertChar")
			return nil, xerrors.Errorf("failed to cast operation to CRDTInsertChar")
		}
		opID, err := ReconstructOpID(op.OperationID, op.Origin)
		if err != nil {
			n.logCRDT.Error().Msgf("failed to reconstruct opID: %s", err)
			return nil, xerrors.Errorf("failed to reconstruct opID: %w", err)
		}
		insertOp.OpID = opID
		insertOps = append(insertOps, insertOp)
	}

	// Remove the characters that are marked for deletion
	for _, op := range toRemove {
		// Cast the operation to the correct type
		removeOp, err := op.Operation.(types.CRDTDeleteChar)
		if err {
			n.logCRDT.Error().Msgf("failed to cast operation to CRDTDeleteChar")
		}
		for i, insertOp := range insertOps {
			// Cast the operation to the correct type
			if insertOp.OpID == removeOp.RemovedID {
				insertOps = append(insertOps[:i], insertOps[i+1:]...)
				break
			}
		}
	}
	return insertOps, nil
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

// GetDocumentList returns a list of documents that are stored in the peer.
// The documents are either in the document directory or in the Editor.
func (n *node) GetDocumentList() ([]string, error) {
	// Get the directory to store documents
	// docDir := n.conf.DocumentDir

	// // Get the list of documents in the directory
	// files, err := os.ReadDir(docDir)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to read directory: %w", err)
	// }

	files := make([]os.DirEntry, 0)

	// Get the editor
	editor := n.GetEditor()

	// Get the list of documents in the editor
	editorDocs := make([]string, 0, len(editor))

	for docID := range editor {
		editorDocs = append(editorDocs, docID)
	}

	// Combine the list of documents in the directory and the editor
	docList := make([]string, 0, len(files)+len(editorDocs))

	for _, file := range files {
		docList = append(docList, file.Name())
	}

	docList = append(docList, editorDocs...)

	return docList, nil
}

// AddNewDocument adds a new document to the editor.
func (n *node) AddNewDocument(docID string) error {
	n.editor.mu.Lock()
	defer n.editor.mu.Unlock()

	editor := n.editor.ed
	if _, ok := editor[docID]; ok {
		return fmt.Errorf("document already exists")
	}

	// Add the document to the editor
	n.editor.ed[docID] = make(map[string][]types.CRDTOperation)

	return nil
}

func (n *node) SaveTransactions(transactions types.CRDTOperationsMessage) error {
	operations := transactions.Operations
	n.logCRDT.Debug().Msgf("SaveTransactions: %d operations", len(operations))

	// Step 0: Cast all interfaces to the respective types
	// Use an indexed loop so we can get a pointer to the actual slice element
	for i := range operations {
		n.CastOperation(&operations[i])
	}

	// Step 1: Update CRDT states and initialize operations
	for i := range operations {
		if err := n.updateCRDTState(&operations[i]); err != nil {
			return err
		}
	}

	// Step 2: Update operation attributes
	for i := range operations {
		if err := n.updateOperationAttributes(&operations[i]); err != nil {
			return err
		}
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
	// Update blockID reference
	blockID, err := n.updateBlockReferences(&operation.BlockID)
	if err != nil {
		return fmt.Errorf("failed to update block references: %w", err)
	}
	operation.BlockID = blockID

	// Update other block references
	switch op := operation.Operation.(type) {
	case types.CRDTAddBlock:
		return n.handleCRDTAddBlock(operation, op)
	case types.CRDTRemoveBlock:
		return n.handleCRDTRemoveBlock(operation, op)
	case types.CRDTUpdateBlock:
		return n.handleCRDTUpdateBlock(operation, op)
	case types.CRDTInsertChar:
		return n.handleCRDTInsertChar(operation, op)
	case types.CRDTDeleteChar:
		return n.handleCRDTDeleteChar(operation, op)
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

func (n *node) handleCRDTDeleteChar(operation *types.CRDTOperation, op types.CRDTDeleteChar) error {
	block, err := n.updateBlockReferences(&op.RemovedID)
	if err != nil {
		return fmt.Errorf("failed to update block references: %w", err)
	}
	op.RemovedID = block
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

	// Parse the ID and username
	id, username, err := ParseID(*ref)
	if err != nil {
		n.logCRDT.Error().Msgf("updateBlockReferences: %s", err)
		return "", err
	}

	// Check if the ID is a temporary ID
	if username == "temp" {
		id = n.crdtState.GetTmpID(id)
		username = n.conf.Socket.GetAddress()
		res, err := ReconstructOpID(id, username)
		if err != nil {
			n.logCRDT.Error().Msgf("updateBlockReferences: %s", err)
			return "", err
		}
		n.logCRDT.Debug().Msgf("updateBlockReferences: %s -> %s", *ref, res)

		return res, nil
	}

	return *ref, nil
}

func (n *node) processAndBroadcast(transactions types.CRDTOperationsMessage) error {
	msg, err := n.conf.MessageRegistry.MarshalMessage(transactions)
	if err != nil {
		return err
	}
	return n.Broadcast(msg)
}

func (n *node) createBlock(blockType types.BlockTypeName, props types.DefaultBlockProps, blockID string) types.BlockType {
	switch blockType {
	case types.ParagraphBlockType:
		return &types.ParagraphBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockID,
			Content:   nil,
			Children:  nil,
		}
	case types.HeadingBlockType:
		return &types.HeadingBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockID,
			Level:     props.Level,
			Content:   nil,
			Children:  nil,
		}
	case types.BulletedListBlockType:
		return &types.BulletedListBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockID,
			Content:   nil,
			Children:  nil,
		}
	case types.NumberedListBlockType:
		return &types.NumberedListBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockID,
			Content:   nil,
			Children:  nil,
		}
	case types.ImageBlockType:
		return &types.ImageBlock{
			BlockType:    nil,
			Default:      props,
			ID:           blockID,
			URL:          "",
			Caption:      "",
			PreviewWidth: 0,
			Children:     nil,
		}
	case types.TableBlockType:
		return &types.TableBlock{
			BlockType: nil,
			Default:   props,
			ID:        blockID,
			Content:   types.TableContent{},
			Children:  nil,
		}

	default:
		return nil
	}
}

// -------------------------------------------------------------------
// Exported CRDT Operations
//
// These functions are only defined in the node to enable wails to generate the
// necessary bindings for the frontend for these operations.

func (n *node) ExportCRDTAddBlock(addBlockOp types.CRDTAddBlock) error {
	return nil
}

func (n *node) ExportCRDTRemoveBlock(removeBlockOp types.CRDTRemoveBlock) error {
	return nil
}

func (n *node) ExportCRDTUpdateBlock(updateBlockOp types.CRDTUpdateBlock) error {
	return nil
}

func (n *node) ExportCRDTInsertChar(insertCharOp types.CRDTInsertChar) error {
	return nil
}

func (n *node) ExportCRDTDeleteChar(deleteCharOp types.CRDTDeleteChar) error {
	return nil
}

func (n *node) ExportCRDTAddMark(addMarkOp types.CRDTAddMark) error {
	return nil
}

func (n *node) ExportCRDTRemoveMark(removeMarkOp types.CRDTRemoveMark) error {
	return nil
}
