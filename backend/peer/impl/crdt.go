package impl

import (
	"Node-tion/backend/types"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func (n *node) createBlockContent(ops []types.CRDTOperation) []types.InlineContent {
	ops = n.sortOps(ops)

	var text string
	textStyles := make(map[string]types.TextStyle) // opID -> textStyle
	var charIDs []string
	var removedIDs []string

	n.logCRDT.Debug().Msgf("ops %v", ops)

	for _, op := range ops {
		if err := n.processOperation(op, &text, &textStyles, &charIDs, &removedIDs); err != nil {
			n.logCRDT.Error().Msgf("Error processing operation: %v", err)
		}
	}

	n.removeDeletedCharacters(&text, &charIDs, removedIDs)

	return n.generateInlineContent(text, textStyles, charIDs)
}

func (n *node) processOperation(
	op types.CRDTOperation,
	text *string,
	textStyles *map[string]types.TextStyle,
	charIDs *[]string,
	removedIDs *[]string,
) error {
	opID, err := ReconstructOpID(op.OperationID, op.Origin)
	if err != nil {
		return fmt.Errorf("failed to convert operationID to string: %w", err)
	}

	switch op.Type {
	case types.CRDTInsertCharType:
		return n.processInsertChar(op, opID, text, charIDs)
	case types.CRDTDeleteCharType:
		return n.processDeleteChar(op, removedIDs)
	case types.CRDTAddMarkType:
		return n.processAddMark(op, textStyles, *charIDs)
	case types.CRDTRemoveMarkType:
		return n.processRemoveMark(op, textStyles, *charIDs)
	default:
		return fmt.Errorf("unknown operation type: %v", op.Type)
	}
}

func (n *node) processInsertChar(
	op types.CRDTOperation,
	opID string,
	text *string,
	charIDs *[]string,
) error {
	insertOp, ok := op.Operation.(types.CRDTInsertChar)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTInsertChar")
	}

	afterID := insertOp.AfterID
	lastAfterID := ""
	if len(*charIDs) > 0 {
		lastAfterID = (*charIDs)[len(*charIDs)-1]
	}

	if afterID != lastAfterID {
		pos := n.getIDIndex(afterID, *charIDs)
		if pos == -1 {
			return fmt.Errorf("failed to find afterID in charIDs")
		}
		*charIDs = append((*charIDs)[:pos+1], append([]string{opID}, (*charIDs)[pos+1:]...)...)
		*text = (*text)[:pos+1] + insertOp.Character + (*text)[pos+1:]
	} else {
		*text += insertOp.Character
		*charIDs = append(*charIDs, opID)
	}

	return nil
}

func (n *node) processDeleteChar(op types.CRDTOperation, removedIDs *[]string) error {
	deleteOp, ok := op.Operation.(types.CRDTDeleteChar)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTDeleteChar")
	}
	*removedIDs = append(*removedIDs, deleteOp.RemovedID)
	return nil
}

func (n *node) processAddMark(
	op types.CRDTOperation,
	textStyles *map[string]types.TextStyle,
	charIDs []string,
) error {
	addMarkOp, ok := op.Operation.(types.CRDTAddMark)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTAddMark")
	}
	n.applyAddMark(*textStyles, charIDs, addMarkOp.Start.OpID, addMarkOp.End.OpID, addMarkOp)
	return nil
}

func (n *node) processRemoveMark(
	op types.CRDTOperation,
	textStyles *map[string]types.TextStyle,
	charIDs []string,
) error {
	removeMarkOp, ok := op.Operation.(types.CRDTRemoveMark)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTRemoveMark")
	}
	n.applyRemoveMark(*textStyles, charIDs, removeMarkOp.Start.OpID, removeMarkOp.End.OpID, removeMarkOp)
	return nil
}

func (n *node) removeDeletedCharacters(
	text *string,
	charIDs *[]string,
	removedIDs []string,
) {
	for _, removedID := range removedIDs {
		pos := n.getIDIndex(removedID, *charIDs)
		if pos == -1 {
			n.logCRDT.Error().Msgf("failed to find removedID in charIDs")
			continue
		}
		*charIDs = append((*charIDs)[:pos], (*charIDs)[pos+1:]...)
		*text = (*text)[:pos] + (*text)[pos+1:]
	}
}

func (n *node) generateInlineContent(text string,
	textStyles map[string]types.TextStyle,
	charIDs []string,
) []types.InlineContent {
	var styledTexts []types.StyledText
	// If the style is the same, we can group the characters together
	var previousStyles types.TextStyle
	var tmpStringContent string
	var tmpCharIDs []string

	n.logCRDT.Debug().Msgf("charIDs %v", charIDs)
	n.logCRDT.Debug().Msgf("textStyles %v", textStyles)
	n.logCRDT.Debug().Msgf("text %v", text)

	for _, charID := range charIDs {
		if !compareTextStyle(textStyles[charID], previousStyles) {
			// If the style is different, we need to create a new InlineContent
			if len(tmpStringContent) > 0 {
				styledTexts = append(styledTexts, types.StyledText{
					CharIDs: tmpCharIDs,
					Text:    tmpStringContent,
					Styles:  previousStyles,
				})
				// Reset the stringContent
				tmpStringContent = ""
				tmpCharIDs = nil
			}
		}
		tmpStringContent += string(text[n.getIDIndex(charID, charIDs)])
		tmpCharIDs = append(tmpCharIDs, charID)
		previousStyles = textStyles[charID]
	}

	// We need to add the last block of text
	if len(tmpStringContent) > 0 {
		styledTexts = append(styledTexts, types.StyledText{
			CharIDs: tmpCharIDs,
			Text:    tmpStringContent,
			Styles:  previousStyles,
		})
	}

	var inlineContents = make([]types.InlineContent, len(styledTexts))
	n.logCRDT.Debug().Msgf("Length styledTexts %v", styledTexts)
	for i, styledText := range styledTexts {
		n.logCRDT.Debug().Msgf("styledText %v", styledText)
		cp := styledText
		inlineContents[i] = &cp
	}

	return inlineContents
}

func compareTextStyle(a types.TextStyle, b types.TextStyle) bool {
	if a.Bold != b.Bold || a.Italic != b.Italic || a.Underline != b.Underline ||
		a.Strikethrough != b.Strikethrough || a.TextColor != b.TextColor ||
		a.BackgroundColor != b.BackgroundColor {
		return false
	}

	return true
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

func (n *node) applyAddMark(textStyles map[string]types.TextStyle,
	charIDs []string,
	startID,
	endID string,
	op types.CRDTAddMark,
) {

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

func (n *node) applyRemoveMark(textStyles map[string]types.TextStyle,
	charIDs []string,
	startID,
	endID string,
	op types.CRDTRemoveMark,
) {

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

func (n *node) sortOps(ops []types.CRDTOperation) []types.CRDTOperation {
	// Sort the operations by the afterID and then by the operation id
	sort.Slice(ops, func(i, j int) bool {
		// If the OperationIDs are the same, sort by the origin
		if ops[i].OperationID == ops[j].OperationID {
			return ops[i].Origin < ops[j].Origin
		}
		return ops[i].OperationID < ops[j].OperationID
	})

	return ops

}
func (n *node) CompileDocument(docID string) (string, error) {
	// Step 1: Populate document blocks in order
	document, err := n.populateDocumentBlocks(docID)
	if err != nil {
		return "", fmt.Errorf("failed to populate document blocks: %w", err)
	}

	// Step 2: Populate block content for each block
	finalDocument := n.populateBlockContent(docID, document)

	// Step 3: Serialize the document
	return n.serializeDocument(finalDocument), nil
}

func (n *node) populateDocumentBlocks(docID string) ([]types.BlockFactory, error) {
	document := make([]types.BlockFactory, 0)
	blockChangeOperations := n.GetDocumentOps(docID)[docID]
	blockChangeOperations = n.sortOps(blockChangeOperations)

	for _, blockChangeOp := range blockChangeOperations {
		var err error
		switch blockChangeOp.Type {
		case types.CRDTAddBlockType:
			err = n.handleAddBlock(&document, blockChangeOp)
		case types.CRDTRemoveBlockType:
			err = n.handleRemoveBlock(&document, blockChangeOp)
		case types.CRDTUpdateBlockType:
			err = n.handleUpdateBlock(&document, blockChangeOp)
		default:
			return nil, fmt.Errorf("unknown operation type: %v", blockChangeOp.Type)
		}
		if err != nil {
			return nil, err
		}
	}

	return document, nil
}

func (n *node) handleAddBlock(document *[]types.BlockFactory, blockChangeOp types.CRDTOperation) error {
	addBlockOp, ok := blockChangeOp.Operation.(types.CRDTAddBlock)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTAddBlock")
	}
	addBlockOp.OpID = fmt.Sprintf("%d@%s", blockChangeOp.OperationID, blockChangeOp.Origin)

	if len(*document) == 0 {
		*document = append(*document, types.BlockFactory{
			ID:        addBlockOp.OpID,
			BlockType: addBlockOp.BlockType,
			Props:     addBlockOp.Props,
		})
		return nil
	}

	for i := range *document {
		added, newDocument := n.checkAddBlockAtPosition(*document, i, addBlockOp)
		if added {
			*document = newDocument
			break
		}
	}
	return nil
}

func (n *node) handleRemoveBlock(document *[]types.BlockFactory, blockChangeOp types.CRDTOperation) error {
	removeBlockOp, ok := blockChangeOp.Operation.(types.CRDTRemoveBlock)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTRemoveBlock")
	}
	removeBlockOp.OpID = fmt.Sprintf("%d@%s", blockChangeOp.OperationID, blockChangeOp.Origin)

	for i := range *document {
		removed, newDocument := checkRemoveBlockAtPosition(*document, i, removeBlockOp)
		if removed {
			*document = newDocument
			break
		}
	}
	return nil
}

func (n *node) handleUpdateBlock(document *[]types.BlockFactory, blockChangeOp types.CRDTOperation) error {
	updateBlockOp, ok := blockChangeOp.Operation.(types.CRDTUpdateBlock)
	if !ok {
		return fmt.Errorf("failed to cast operation to CRDTUpdateBlock")
	}
	updateBlockOp.UpdatedBlock = blockChangeOp.BlockID

	var oldBlock *types.BlockFactory
	for i := range *document {
		oldBlock, *document = n.findBlockToUpdateAndRemove(*document, i, updateBlockOp)
		if oldBlock != nil {
			break
		}
	}

	if oldBlock != nil {
		updatedBlock := &types.BlockFactory{
			ID:        oldBlock.ID,
			BlockType: updateBlockOp.BlockType,
			Props:     n.updateBlockProps(oldBlock.Props, updateBlockOp.Props),
			Children:  oldBlock.Children,
		}

		for i := range *document {
			added, newDocument := n.checkAddBackBlockAtPosition(*document, i, updateBlockOp, *updatedBlock)
			if added {
				*document = newDocument
				break
			}
		}
	}
	return nil
}

func (n *node) populateBlockContent(docID string, document []types.BlockFactory) []types.BlockType {
	finalDocument := make([]types.BlockType, 0)
	for _, block := range document {
		if block.Deleted {
			continue
		}
		blockOperations := n.GetBlockOps(docID, block.ID)
		newBlock := n.createBlock(docID, block, blockOperations)
		finalDocument = append(finalDocument, newBlock)
	}
	return finalDocument
}

func (n *node) serializeDocument(finalDocument []types.BlockType) string {
	finalJSON := "[ "
	for _, block := range finalDocument {
		finalJSON += types.SerializeBlock(block) + ","
	}
	if len(finalJSON) > 2 {
		finalJSON = finalJSON[:len(finalJSON)-1] // Remove the additional ","
	}
	finalJSON += "]"
	return finalJSON
}

func (n *node) createBlock(docID string,
	block types.BlockFactory,
	blockOperations []types.CRDTOperation,
) types.BlockType {
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
		n.logCRDT.Debug().Msgf("Creating Numbered List Block")
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
func (n *node) checkAddBlockAtPosition(document []types.BlockFactory,
	index int,
	addBlockOp types.CRDTAddBlock,
) (bool, []types.BlockFactory) {
	added := false

	// If the current block is the parent
	if isParentBlock(document, index, addBlockOp.ParentBlock) {
		added, document = n.handleParentBlockInsertion(document, index, addBlockOp)
		if added {
			return added, document
		}
	}

	// If we need to add the block at the start of the document
	if !added && addBlockOp.ParentBlock == "" && addBlockOp.AfterBlock == "" {
		newDoc := insertBlockAtStart(document, addBlockOp)
		n.logBlockAdded(addBlockOp.OpID, newDoc)
		return true, newDoc
	}

	// If we need to add the block after the current block
	if !added && document[index].ID == addBlockOp.AfterBlock {
		newDoc := insertBlockAfterBlock(document, index, addBlockOp)
		n.logBlockAdded(addBlockOp.OpID, newDoc)
		return true, newDoc
	}

	// If not added yet, try adding within children (recursively)
	if !added && document[index].Children != nil {
		added, newChildren := n.insertBlockInChildren(document[index].Children, addBlockOp)
		document[index].Children = newChildren
		if added {
			n.logBlockAdded(addBlockOp.OpID, document)
			return true, document
		}
	}

	n.logBlockAdded(addBlockOp.OpID, document)
	return added, document
}

// -------------------- Helper Functions --------------------

func isParentBlock(document []types.BlockFactory, index int, parentBlockID string) bool {
	return parentBlockID != "" && document[index].ID == parentBlockID
}

func (n *node) handleParentBlockInsertion(document []types.BlockFactory,
	index int,
	addBlockOp types.CRDTAddBlock,
) (bool, []types.BlockFactory) {
	// If no children, initialize and append
	if document[index].Children == nil {
		document[index].Children = []types.BlockFactory{}
		newBlock := buildNewBlock(addBlockOp)
		document[index].Children = append(document[index].Children, newBlock)
		return true, document
	}

	// If we need to add at the start of children
	if addBlockOp.AfterBlock == "" {
		newBlock := buildNewBlock(addBlockOp)
		document[index].Children = append([]types.BlockFactory{newBlock}, document[index].Children...)
		return true, document
	}

	// Otherwise, try inserting at the appropriate position in children
	added, newChildren := n.insertBlockInChildren(document[index].Children, addBlockOp)
	document[index].Children = newChildren
	return added, document
}

func (n *node) insertBlockInChildren(children []types.BlockFactory,
	addBlockOp types.CRDTAddBlock,
) (bool, []types.BlockFactory) {
	for i := range children {
		added, newChildren := n.checkAddBlockAtPosition(children, i, addBlockOp)
		if added {
			return true, newChildren
		}
	}
	return false, children
}

func insertBlockAtStart(document []types.BlockFactory,
	addBlockOp types.CRDTAddBlock,
) []types.BlockFactory {
	newBlock := buildNewBlock(addBlockOp)
	return append([]types.BlockFactory{newBlock}, document...)
}

func insertBlockAfterBlock(document []types.BlockFactory,
	index int,
	addBlockOp types.CRDTAddBlock,
) []types.BlockFactory {
	newBlock := buildNewBlock(addBlockOp)
	return append(document[:index+1], append([]types.BlockFactory{newBlock}, document[index+1:]...)...)
}

func buildNewBlock(addBlockOp types.CRDTAddBlock) types.BlockFactory {
	return types.BlockFactory{
		ID:        addBlockOp.OpID,
		BlockType: addBlockOp.BlockType,
		Props:     addBlockOp.Props,
		Children:  nil,
	}
}

func (n *node) logBlockAdded(opID string, document []types.BlockFactory) {
	n.logCRDT.Debug().Msgf("block %s added to finalDoc in CheckAddBlockPos", opID)
	n.logCRDT.Debug().Msgf("document %v", document)
}

// checkRemoveBlockAtPosition checks if the removeBlockOp should be removed from the document at the current index
// Returns true if the block was removed, false otherwise
func checkRemoveBlockAtPosition(document []types.BlockFactory,
	index int,
	removeBlockOp types.CRDTRemoveBlock,
) (bool, []types.BlockFactory) {
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

func (n *node) findBlockToUpdateAndRemove(document []types.BlockFactory,
	index int,
	updateBlockOp types.CRDTUpdateBlock,
) (*types.BlockFactory, []types.BlockFactory) {
	var updated *types.BlockFactory

	// Check if the block is going after the current block
	if document[index].ID == updateBlockOp.UpdatedBlock {
		// Copy the block to be updated to avoid the reference being updated
		oldBlock := document[index]
		updated = &oldBlock
		document = append(document[:index], document[index+1:]...)

		return updated, document
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
func (n *node) checkAddBackBlockAtPosition(
	document []types.BlockFactory,
	index int,
	updateBlockOp types.CRDTUpdateBlock,
	updatedBlock types.BlockFactory,
) (bool, []types.BlockFactory) {
	added := false

	// If current block is the parent
	if isParentBlock(document, index, updateBlockOp.ParentBlock) {
		added, document = n.handleParentBackBlockInsertion(document, index, updateBlockOp, updatedBlock)
		if added {
			return added, document
		}
	}

	// Try inserting at the start of the document
	if !added && updateBlockOp.ParentBlock == "" && updateBlockOp.AfterBlock == "" {
		newDoc := insertBlockAtDocumentStart(document, updatedBlock)
		return true, newDoc
	}

	// Try inserting after an existing block
	if !added && document[index].ID == updateBlockOp.AfterBlock {
		newDoc := insertBlockAfterExistingBlock(document, index, updatedBlock)
		return true, newDoc
	}

	// If not added yet, attempt insertion in children
	if !added && document[index].Children != nil {
		added, newChildren := n.tryInsertInChildren(document[index].Children, updateBlockOp, updatedBlock)
		document[index].Children = newChildren
		if added {
			return true, document
		}
	}

	return added, document
}

// -------------------- Helper Functions --------------------

func (n *node) handleParentBackBlockInsertion(
	document []types.BlockFactory,
	index int,
	updateBlockOp types.CRDTUpdateBlock,
	updatedBlock types.BlockFactory,
) (bool, []types.BlockFactory) {
	if document[index].Children == nil {
		// Initialize children and add the new block
		document[index].Children = []types.BlockFactory{updatedBlock}
		return true, document
	}

	// Insert at the start of the children if AfterBlock is not specified
	if updateBlockOp.AfterBlock == "" {
		document[index].Children = append([]types.BlockFactory{updatedBlock}, document[index].Children...)
		return true, document
	}

	// Otherwise, try to insert in the appropriate child position
	added, newChildren := n.tryInsertInChildren(document[index].Children, updateBlockOp, updatedBlock)
	document[index].Children = newChildren
	return added, document
}

func insertBlockAtDocumentStart(document []types.BlockFactory, updatedBlock types.BlockFactory) []types.BlockFactory {
	return append([]types.BlockFactory{updatedBlock}, document...)
}

func insertBlockAfterExistingBlock(document []types.BlockFactory,
	index int,
	updatedBlock types.BlockFactory,
) []types.BlockFactory {
	return append(document[:index+1], append([]types.BlockFactory{updatedBlock}, document[index+1:]...)...)
}

func (n *node) tryInsertInChildren(
	children []types.BlockFactory,
	updateBlockOp types.CRDTUpdateBlock,
	updatedBlock types.BlockFactory,
) (bool, []types.BlockFactory) {
	for i := range children {
		added, newChildren := n.checkAddBackBlockAtPosition(children, i, updateBlockOp, updatedBlock)
		if added {
			return true, newChildren
		}
	}
	return false, children
}

func (n *node) updateBlockProps(blockProps types.DefaultBlockProps,
	updatedProps types.DefaultBlockProps,
) types.DefaultBlockProps {

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
