package unit

import (
	z "Node-tion/backend/internal/testing"
	"Node-tion/backend/peer/tests"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/types"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ----- Helper functions -----
func CreateInsertsFromString(content string, addr string, blockID string, insertStart int) []types.CRDTOperation {
	ops := make([]types.CRDTOperation, len(content))
	for i, char := range content {
		if i == 0 {
			ops[i] = types.CRDTOperation{
				Type:        types.CRDTInsertCharType,
				Origin:      addr,
				OperationID: uint64(i + insertStart),
				DocumentID:  "doc1",
				BlockID:     blockID,
				Operation:   CreateInsertOp("", string(char)),
			}
		} else {
			ops[i] = types.CRDTOperation{
				Type:        types.CRDTInsertCharType,
				Origin:      addr,
				OperationID: uint64(i + insertStart),
				DocumentID:  "doc1",
				BlockID:     blockID,
				Operation:   CreateInsertOp(strconv.Itoa(i+insertStart-1)+"@"+addr, string(char)),
			}
		}
	}
	return ops
}

func CreateInsertOp(afterID string, content string) types.CRDTInsertChar {
	return types.CRDTInsertChar{
		AfterID:   afterID,
		Character: content,
	}
}

// ----- Tests -----

// Check that the CompileDocument can generate a JSON string from the editor of one peer
func Test_Document_Compilation_1Peer_MultipleBlocks(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	docID := "doc1"
	block1ID := "1" + "@temp"
	addBlock1 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	// Populate the editor with some operations
	//Add a block
	err := peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   addBlock1,
	}})
	require.NoError(t, err)
	// Add some text to the block
	inserts := tests.CreateInsertsFromString("Hello!", "temp", docID, block1ID, 2)
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Compile the document
	doc, err := peer.CompileDocument(docID)
	require.NoError(t, err)
	expected := "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\": \"text\",\"charIds\":[\"2@temp\",\"3@temp\",\"4@temp\",\"5@temp\",\"6@temp\",\"7@temp\"],\"text\":\"Hello!\",\"styles\":{}}],\"children\":[]}]"

	require.JSONEq(t, expected, doc)

	block2ID := "8" + "@temp"
	addBlock2 := types.CRDTAddBlock{
		AfterBlock:  block1ID,
		ParentBlock: "",
		BlockType:   types.HeadingBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
			Level:           types.H1,
		},
	}

	// Add a block
	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 8,
		DocumentID:  docID,
		BlockID:     block2ID,
		Operation:   addBlock2,
	},
	})
	require.NoError(t, err)

	// Add some text to the block
	inserts = tests.CreateInsertsFromString("World!", "temp", docID, block2ID, 9)
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Add a bold mark to the text
	boldMark := types.CRDTAddMark{
		Start: types.MarkStart{
			Type: "Before",
			OpID: "2@temp",
		},
		End: types.MarkEnd{
			Type: "After",
			OpID: "7@temp",
		},
		MarkType: types.Bold,
		Options:  types.MarkOptions{},
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddMarkType,
		Origin:      "temp",
		OperationID: 15,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   boldMark,
	},
	})

	require.NoError(t, err)

	// Compile the document
	doc, err = peer.CompileDocument(docID)
	require.NoError(t, err)

	expected = "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"3@temp\",\"4@temp\",\"5@temp\",\"6@temp\",\"7@temp\"],\"text\":\"Hello!\",\"styles\":{\"bold\":true}}],\"children\":[]},{\"id\":\"8@temp\",\"type\":\"heading\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\",\"level\":1},\"content\":[{\"type\":\"text\",\"charIds\":[\"9@temp\",\"10@temp\",\"11@temp\",\"12@temp\",\"13@temp\",\"14@temp\"],\"text\":\"World!\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)

	// Add a third block with the text "Hello World!"
	block3ID := "16" + "@temp"
	addBlock3 := types.CRDTAddBlock{
		AfterBlock:  block2ID,
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}
	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 16,
		DocumentID:  docID,
		BlockID:     block3ID,
		Operation:   addBlock3,
	}})
	require.NoError(t, err)

	// Add some text to the block
	inserts = tests.CreateInsertsFromString("Block3", "temp", docID, block3ID, 17) // last opId is 22
	_ = peer.UpdateEditor(inserts)

	// Compile the document
	doc, err = peer.CompileDocument(docID)
	require.NoError(t, err)

	expected = "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"3@temp\",\"4@temp\",\"5@temp\",\"6@temp\",\"7@temp\"],\"text\":\"Hello!\",\"styles\":{\"bold\":true}}],\"children\":[]},{\"id\":\"8@temp\",\"type\":\"heading\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\",\"level\":1},\"content\":[{\"type\":\"text\",\"charIds\":[\"9@temp\",\"10@temp\",\"11@temp\",\"12@temp\",\"13@temp\",\"14@temp\"],\"text\":\"World!\",\"styles\":{}}],\"children\":[]},{\"id\":\"16@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"17@temp\",\"18@temp\",\"19@temp\",\"20@temp\",\"21@temp\",\"22@temp\"],\"text\":\"Block3\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)
}

func Test_Document_Compilation_1Peer_MultipleStyles(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	docID := "doc1"
	block1ID := "1" + "@temp"
	addBlock1 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	// Populate the editor with some operations
	//Add a block
	err := peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   addBlock1,
	}})
	require.NoError(t, err)
	// Add some text to the block
	inserts := tests.CreateInsertsFromString("Hello World!", "temp", docID, block1ID, 2) // last opId is 13
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Add MarkStyles to the text

	// Add a bold mark to the text
	boldMark := types.CRDTAddMark{
		Start: types.MarkStart{
			Type: "Before",
			OpID: "2@temp",
		},
		End: types.MarkEnd{
			Type: "After",
			OpID: "7@temp",
		},
		MarkType: types.Bold,
		Options:  types.MarkOptions{},
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddMarkType,
		Origin:      "temp",
		OperationID: 14,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   boldMark,
	}})
	require.NoError(t, err)

	// Add an italic mark to the text
	italicMark := types.CRDTAddMark{
		Start: types.MarkStart{
			Type: "Before",
			OpID: "5@temp",
		},
		End: types.MarkEnd{
			Type: "After",
			OpID: "13@temp",
		},
		MarkType: types.Italic,
		Options:  types.MarkOptions{},
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddMarkType,
		Origin:      "temp",
		OperationID: 15,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   italicMark,
	}})
	require.NoError(t, err)

	// Generate the document
	doc, err := peer.CompileDocument(docID)
	require.NoError(t, err)

	expected := "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"3@temp\",\"4@temp\"],\"text\":\"Hel\",\"styles\":{\"bold\":true}},{\"type\":\"text\",\"charIds\":[\"5@temp\",\"6@temp\",\"7@temp\"],\"text\":\"lo \",\"styles\":{\"bold\":true,\"italic\":true}},{\"type\":\"text\",\"charIds\":[\"8@temp\",\"9@temp\",\"10@temp\",\"11@temp\",\"12@temp\",\"13@temp\"],\"text\":\"World!\",\"styles\":{\"italic\":true}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)
}

func Test_Document_Compilation_1Peer_BlockWithChildren(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	docID := "doc1"
	block1ID := "1@temp"
	addBlock1 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.HeadingBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
			Level:           types.H1,
		},
	}

	// Populate the editor with some operations
	//Add a block
	err := peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   addBlock1,
	}})
	require.NoError(t, err)
	// Add some text to the block
	inserts := tests.CreateInsertsFromString("H1", "temp", docID, block1ID, 2) // last opId is 13
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Add a child block
	block2ID := "4@temp"
	addBlock2 := types.CRDTAddBlock{
		ParentBlock: block1ID,
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 4,
		DocumentID:  docID,
		BlockID:     block2ID,
		Operation:   addBlock2,
	}})
	require.NoError(t, err)

	// Add some text to the child block
	inserts = tests.CreateInsertsFromString("Child", "temp", docID, block2ID, 5) // last opId is 9
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Generate the document
	doc, err := peer.CompileDocument(docID)
	require.NoError(t, err)

	expected := "[{\"id\":\"1@temp\",\"type\":\"heading\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\",\"level\":1},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"3@temp\"],\"text\":\"H1\",\"styles\":{}}],\"children\":[{\"id\":\"4@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"5@temp\",\"6@temp\",\"7@temp\",\"8@temp\",\"9@temp\"],\"text\":\"Child\",\"styles\":{}}],\"children\":[]}]}]"
	require.JSONEq(t, expected, doc)

	// Add a second child to the parent block
	block3ID := "10@temp"
	addBlock3 := types.CRDTAddBlock{
		AfterBlock:  block2ID,
		ParentBlock: block1ID,
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 10,
		DocumentID:  docID,
		BlockID:     block3ID,
		Operation:   addBlock3,
	}})
	require.NoError(t, err)

	// Add some text to the child block
	inserts = tests.CreateInsertsFromString("Child2", "temp", docID, block3ID, 11) // last opId is 16
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Generate the document
	doc, err = peer.CompileDocument(docID)
	require.NoError(t, err)

	expected = "[{\"id\":\"1@temp\",\"type\":\"heading\",\"props\":{\"level\":1,\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"3@temp\"],\"text\":\"H1\",\"styles\":{}}],\"children\":[{\"id\":\"4@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"5@temp\",\"6@temp\",\"7@temp\",\"8@temp\",\"9@temp\"],\"text\":\"Child\",\"styles\":{}}],\"children\":[]},{\"id\":\"10@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"11@temp\",\"12@temp\",\"13@temp\",\"14@temp\",\"15@temp\",\"16@temp\"],\"text\":\"Child2\",\"styles\":{}}],\"children\":[]}]}]"
	require.JSONEq(t, expected, doc)
}

func Test_Document_Compilation_1Peer_UnorderedInserts(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	docID := "doc1"
	block1ID := "1" + "@temp"
	addBlock1 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	// Populate the editor with some operations
	//Add a block
	err := peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   addBlock1,
	}})
	require.NoError(t, err)

	// Add some text to the block
	inserts := tests.CreateInsertsFromString("ac", "temp", docID, block1ID, 2) // last opId is 3
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Add some text to the block in the 3rd position
	inserts = []types.CRDTOperation{{
		Type:        types.CRDTInsertCharType,
		Origin:      "temp",
		OperationID: 4,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   tests.CreateInsertOp("2@temp", "b"),
	}}
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Generate the document
	doc, err := peer.CompileDocument(docID)
	require.NoError(t, err)

	expected := "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"4@temp\",\"3@temp\"],\"text\":\"abc\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)
}

func Test_Document_Compilation_1Peer_RemoveUpdateBlock(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	docID := "doc1"
	block1ID := "1" + "@temp"
	addBlock1 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	// Populate the editor with some operations
	//Add a block
	err := peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   addBlock1,
	}})
	require.NoError(t, err)
	// Add some text to the block
	inserts := tests.CreateInsertsFromString("Hello!", "temp", docID, block1ID, 2)
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Remove the block
	removeBlock := types.CRDTRemoveBlock{
		RemovedBlock: block1ID,
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTRemoveBlockType,
		Origin:      "temp",
		OperationID: 8,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   removeBlock,
	}})
	require.NoError(t, err)

	// Compile the document
	doc, err := peer.CompileDocument(docID)
	require.NoError(t, err)
	expected := "[]"
	require.JSONEq(t, expected, doc)

	// Add another block with the text "World!"
	block2ID := "9" + "@temp"
	addBlock2 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.HeadingBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
			Level:           types.H1,
		},
	}

	// Add a block
	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "temp",
		OperationID: 9,
		DocumentID:  docID,
		BlockID:     block2ID,
		Operation:   addBlock2,
	},
	})
	require.NoError(t, err)

	// Add some text to the block
	inserts = tests.CreateInsertsFromString("World!", "temp", docID, block2ID, 10)
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Update the block
	updateBlock := types.CRDTUpdateBlock{
		UpdatedBlock: block2ID,
		BlockType:    types.ParagraphBlockType,
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTUpdateBlockType,
		Origin:      "temp",
		OperationID: 16,
		DocumentID:  docID,
		BlockID:     block2ID,
		Operation:   updateBlock,
	},
	})
	require.NoError(t, err)

	// Compile the document
	doc, err = peer.CompileDocument(docID)
	require.NoError(t, err)

	expected = "[{\"id\":\"9@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"10@temp\",\"11@temp\",\"12@temp\",\"13@temp\",\"14@temp\",\"15@temp\"],\"text\":\"World!\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)

	// Update the block again to a heading block and change the blockProps
	updateBlock = types.CRDTUpdateBlock{
		UpdatedBlock: block2ID,
		BlockType:    types.HeadingBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "white",
			TextColor:       "blue",
			TextAlignment:   "center",
			Level:           types.H2,
		},
	}

	err = peer.UpdateEditor([]types.CRDTOperation{{
		Type:        types.CRDTUpdateBlockType,
		Origin:      "temp",
		OperationID: 17,
		DocumentID:  docID,
		BlockID:     block2ID,
		Operation:   updateBlock,
	}})
	require.NoError(t, err)

	// Compile the document
	_, err = peer.CompileDocument(docID)
	require.NoError(t, err)

	expected = "[{\"id\":\"9@temp\",\"type\":\"heading\",\"props\":{\"textColor\":\"blue\",\"backgroundColor\":\"white\",\"textAlignment\":\"center\",\"level\":2},\"content\":[{\"type\":\"text\",\"charIds\":[\"10@temp\",\"11@temp\",\"12@temp\",\"13@temp\",\"14@temp\",\"15@temp\"],\"text\":\"World!\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)
}

// Check that the CompileDocument can generate a JSON string from the editor of two peers
func Test_Document_Compilation_2Peers_Separate_Blocks(t *testing.T) {
	transp := channel.NewTransport()
	peerYas := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peerYas.Stop()
	peerUgo := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peerUgo.Stop()

	docID := "doc1"
	block1ID := "1" + "@yas"
	addBlock1 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		},
	}

	// Populate the editor with some operations
	//Add a block
	opYas := []types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "yas",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block1ID,
		Operation:   addBlock1,
	}}
	// Add some text to the block
	inserts1 := tests.CreateInsertsFromString("1Y!", "yas", docID, block1ID, 2) // last opId is 4
	opYas = append(opYas, inserts1...)

	// Add another block with the text "Block2Y!"
	block2ID := "5" + "@yas"
	addBlock2 := types.CRDTAddBlock{
		AfterBlock:  block1ID,
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		}}

	opYas = append(opYas, types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      "yas",
		OperationID: 5,
		DocumentID:  docID,
		BlockID:     block2ID,
		Operation:   addBlock2,
	})
	inserts2 := tests.CreateInsertsFromString("2Y!", "yas", docID, block2ID, 6) // last opId is 8
	opYas = append(opYas, inserts2...)

	// Now we create the operations for Ugo
	block3ID := "1" + "@ugo"
	addBlock3 := types.CRDTAddBlock{
		AfterBlock:  "",
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		}}

	opUgo := []types.CRDTOperation{{
		Type:        types.CRDTAddBlockType,
		Origin:      "ugo",
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     block3ID,
		Operation:   addBlock3,
	}}

	inserts3 := tests.CreateInsertsFromString("1U!", "ugo", docID, block3ID, 2) // last opId is 4
	opUgo = append(opUgo, inserts3...)

	// Add another block with the text "2U!"
	block4ID := "5" + "@ugo"
	addBlock4 := types.CRDTAddBlock{
		AfterBlock:  block3ID,
		ParentBlock: "",
		BlockType:   types.ParagraphBlockType,
		Props: types.DefaultBlockProps{
			BackgroundColor: "default",
			TextColor:       "default",
			TextAlignment:   "left",
		}}

	opUgo = append(opUgo, types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      "ugo",
		OperationID: 5,
		DocumentID:  docID,
		BlockID:     block4ID,
		Operation:   addBlock4,
	})
	inserts4 := tests.CreateInsertsFromString("2U!", "ugo", docID, block4ID, 6) // last opId is 8
	opUgo = append(opUgo, inserts4...)

	// Now we apply the operations
	err := peerYas.UpdateEditor(opYas)
	require.NoError(t, err)
	err = peerYas.UpdateEditor(opUgo)
	require.NoError(t, err)
	err = peerUgo.UpdateEditor(opUgo)
	require.NoError(t, err)
	err = peerUgo.UpdateEditor(opYas)
	require.NoError(t, err)

	// Compile the document
	docYas, err := peerYas.CompileDocument(docID)
	require.NoError(t, err)
	docUgo, err := peerUgo.CompileDocument(docID)
	require.NoError(t, err)

	require.JSONEq(t, docYas, docUgo)
}

// Check that a document is stored in the correct directory.
func Test_Document_Directory_Store(t *testing.T) {
	transp := channel.NewTransport()

	docTimestampThreshold := time.Second * 2
	docDir := "../../../../documents"

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithDocTimestampThreshold(docTimestampThreshold), z.WithDocumentDir(docDir))
	defer node1.Stop()

	doc := "This is a test document."
	docID := "test"

	// Store the document
	err := node1.StoreDocument(docID, doc)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		return
	}

	// > check that the directory contains 1 file

	files, err := os.ReadDir(docDir)
	if err != nil {
		t.Errorf("failed to read directory: %v", err)
		return
	}
	require.Len(t, files, 1)

	// > check the format of the file name

	fileName := files[0].Name()
	require.Regexp(t, fmt.Sprintf("^%s_\\d+\\.txt$", docID), fileName)

	// > check the file name contains a timestamp

	timestampStr := fileName[len(docID)+1 : len(fileName)-4]
	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		t.Errorf("failed to parse timestamp: %v", err)
		// Remove the invalid file
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		return
	}

	// convert Unix timestamp to time.Time
	timestamp := time.Unix(timestampInt, 0)
	require.WithinDuration(t, time.Now(), timestamp, time.Second)

	// > check that the file contains the document

	file, err := os.ReadFile(fmt.Sprintf("%s/%s", docDir, fileName))
	if err != nil {
		t.Errorf("failed to read file: %v", err)
	}
	require.Equal(t, doc, string(file))

	// remove the file
	err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName))
	if err != nil {
		t.Errorf("failed to remove file: %v", err)
	}
}

// Check that second documents with the same docID as first is not stored
// if the timestamp threshold has not been reached.
func Test_Document_Directory_Store_Threshold_No(t *testing.T) {
	transp := channel.NewTransport()

	docTimestampThreshold := time.Second * 2
	docDir := "../../../../documents"

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithDocTimestampThreshold(docTimestampThreshold), z.WithDocumentDir(docDir))
	defer node1.Stop()

	doc1 := "This is a test document."
	doc2 := "This is another test document."
	docID := "test"

	// Store the document
	err := node1.StoreDocument(docID, doc1)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		return
	}
	err = node1.StoreDocument(docID, doc2)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		// remove the first file
		files, err := os.ReadDir(docDir)
		if err != nil {
			t.Errorf("failed to read directory: %v", err)
			return
		}
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, files[0].Name()))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		return
	}

	// > check that the directory contains 1 file

	files, err := os.ReadDir(docDir)
	if err != nil {
		t.Errorf("failed to read directory: %v", err)
		return
	}
	require.Len(t, files, 1)

	// > check that file name contains the docID

	fileName := files[0].Name()
	require.Regexp(t, fmt.Sprintf("^%s_\\d+\\.txt$", docID), fileName)

	// remove the file
	err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName))
	if err != nil {
		t.Errorf("failed to remove file: %v", err)
	}
}

// Check that second documents with the same docID as first is stored
// if the timestamp threshold has been reached.
func Test_Document_Directory_Store_Threshold_Yes(t *testing.T) {
	transp := channel.NewTransport()

	docTimestampThreshold := time.Second * 2
	docDir := "../../../../documents"

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithDocTimestampThreshold(docTimestampThreshold), z.WithDocumentDir(docDir))
	defer node1.Stop()

	doc1 := "This is a test document."
	doc2 := "This is another test document."
	docID := "test"

	// Store the document
	err := node1.StoreDocument(docID, doc1)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		return
	}

	// Wait for the timestamp threshold to be reached
	time.Sleep(docTimestampThreshold + time.Second)

	err = node1.StoreDocument(docID, doc2)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		// remove the first file
		files, err := os.ReadDir(docDir)
		if err != nil {
			t.Errorf("failed to read directory: %v", err)
			return
		}
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, files[0].Name()))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		return
	}

	// > check that the directory contains 2 files

	files, err := os.ReadDir(docDir)
	if err != nil {
		t.Errorf("failed to read directory: %v", err)
		return
	}
	require.Len(t, files, 2)

	// > check that both file names contain the docID

	fileName1 := files[0].Name()
	require.Regexp(t, fmt.Sprintf("^%s_\\d+\\.txt$", docID), fileName1)

	fileName2 := files[1].Name()
	require.Regexp(t, fmt.Sprintf("^%s_\\d+\\.txt$", docID), fileName2)

	// > check that the timestamp of the second file is greater than the first
	// by at least the threshold

	timestampStr := fileName1[len(docID)+1 : len(fileName1)-4]
	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		t.Errorf("failed to parse timestamp: %v", err)

		err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName1))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName2))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		return
	}
	timestamp1 := time.Unix(timestampInt, 0)

	timestampStr = fileName2[len(docID)+1 : len(fileName2)-4]
	timestampInt, err = strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		t.Errorf("failed to parse timestamp: %v", err)

		err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName1))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName2))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
		return
	}
	timestamp2 := time.Unix(timestampInt, 0)

	require.Condition(t, func() bool {
		return timestamp2.Sub(timestamp1) >= docTimestampThreshold
	})

	// remove the files
	err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName1))
	if err != nil {
		t.Errorf("failed to remove file: %v", err)
	}
	err = os.Remove(fmt.Sprintf("%s/%s", docDir, fileName2))
	if err != nil {
		t.Errorf("failed to remove file: %v", err)
	}
}

// Check that with more than DocQueueSize documents with same docID,
// the oldest document is removed.
func Test_Document_Directory_Store_Queue_Limit_Same_DocumentID(t *testing.T) {
	transp := channel.NewTransport()

	docTimestampThreshold := time.Second * 2
	docQueueSize := 10
	docDir := "../../../../documents"

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithDocTimestampThreshold(docTimestampThreshold), z.WithDocumentDir(docDir))
	defer node1.Stop()

	docID := "test"

	doc := "This is the oldest test document that should not be in the directory at the end."
	err := node1.StoreDocument(docID, doc)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		return
	}
	timestamp := time.Now().Unix()
	time.Sleep(docTimestampThreshold + time.Second)

	for i := 1; i < docQueueSize+1; i++ {
		// generate a document and store it
		doc := fmt.Sprintf("This is test document %d.", i)
		err := node1.StoreDocument(docID, doc)
		if err != nil {
			t.Errorf("failed to store document: %v", err)
			return
		}
		time.Sleep(docTimestampThreshold + time.Second)
	}

	// > check that the directory contains 10 files

	files, err := os.ReadDir(docDir)
	if err != nil {
		t.Errorf("failed to read directory: %v", err)
		return
	}
	require.Len(t, files, docQueueSize)

	// > check that the oldest file is not in the directory

	// timestamp of the oldest document not to be in the directory
	for _, file := range files {
		timestampStr := file.Name()[len(docID)+1 : len(file.Name())-4]
		timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			t.Errorf("failed to parse timestamp: %v", err)
			return
		}
		// if the difference is < node1.conf.DocTimestampThreshold, the oldest document is still in the directory
		require.Condition(t, func() bool {
			return timestampInt-timestamp > 2
		})
	}

	// remove the files
	for _, file := range files {
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, file.Name()))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}

}

// Check that DocQueueSize documents of the same docID are stored
// and 1 with a different docID is stored.
func Test_Document_Directory_Store_Queue_Limit_Different_DocumentIDs(t *testing.T) {
	transp := channel.NewTransport()

	docTimestampThreshold := time.Second
	docQueueSize := 10
	docDir := "../../../../documents"

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithDocTimestampThreshold(docTimestampThreshold), z.WithDocumentDir(docDir))
	defer node1.Stop()

	docID1 := "test1"

	for i := 0; i < docQueueSize; i++ {
		// generate a document and store it
		doc := fmt.Sprintf("This is test document %d.", i)
		err := node1.StoreDocument(docID1, doc)
		if err != nil {
			t.Errorf("failed to store document: %v", err)
			return
		}

		// Wait for the timestamp threshold to be reached
		time.Sleep(docTimestampThreshold + time.Second)
	}

	docID2 := "test2"
	doc := "This is a test document with a different docID."
	err := node1.StoreDocument(docID2, doc)
	if err != nil {
		t.Errorf("failed to store document: %v", err)
		return
	}

	// > check that the directory contains 10 files with docID1 and 1 file with docID2

	files, err := os.ReadDir(docDir)
	if err != nil {
		t.Errorf("failed to read directory: %v", err)
		return
	}

	docID1Count := 0
	docID2Count := 0
	for _, file := range files {
		if file.Name()[:5] == docID1 {
			docID1Count++
		} else if file.Name()[:5] == docID2 {
			docID2Count++
		}
	}

	require.Equal(t, docQueueSize, docID1Count)
	require.Equal(t, 1, docID2Count)

	// remove the files
	for _, file := range files {
		err = os.Remove(fmt.Sprintf("%s/%s", docDir, file.Name()))
		if err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}
}
