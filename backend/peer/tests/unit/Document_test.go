package unit

import (
	z "Node-tion/backend/internal/testing"
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
				Type:       types.CRDTInsertCharType,
				Origin:     addr,
				DocumentID: "doc1",
				BlockID:    blockID,
				Operation:  CreateInsertOp(strconv.Itoa(i+insertStart)+"@"+addr, "", string(char)),
			}
		} else {
			ops[i] = types.CRDTOperation{
				Type:       types.CRDTInsertCharType,
				Origin:     addr,
				DocumentID: "doc1",
				BlockID:    blockID,
				Operation:  CreateInsertOp(strconv.Itoa(i+insertStart)+"@"+addr, strconv.Itoa(i+insertStart-1)+"@"+addr, string(char)),
			}
		}
	}
	return ops
}

func CreateInsertOp(opID string, afterID string, content string) types.CRDTInsertChar {
	return types.CRDTInsertChar{
		OpID:      opID,
		AfterID:   afterID,
		Character: content,
	}
}

// ----- Tests -----

// Check that the CompileDocument can generate a json string from the editor of one peer
func Test_Document_Compilation_1Peer(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	block1ID := "1" + "@temp"
	addBlock1 := types.CRDTAddBlock{
		OpID:        block1ID,
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
		Type:       types.CRDTAddBlockType,
		Origin:     "temp",
		DocumentID: "doc1",
		BlockID:    block1ID,
		Operation:  addBlock1,
	}})
	require.NoError(t, err)
	// Add some text to the block
	inserts := CreateInsertsFromString("Hello!", "temp", block1ID, 2)
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Compile the document
	doc, err := peer.CompileDocument("doc1")
	require.NoError(t, err)
	expected := "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\": \"text\",\"charIds\":[\"2@temp\",\"3@temp\",\"4@temp\",\"5@temp\",\"6@temp\",\"7@temp\"],\"text\":\"Hello!\",\"styles\":{}}],\"children\":[]}]"

	require.JSONEq(t, expected, doc)

	// Add another block with the text "World!" and in bold
	block2ID := "8" + "@temp"
	addBlock2 := types.CRDTAddBlock{
		OpID:        block2ID,
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
		Type:       types.CRDTAddBlockType,
		Origin:     "temp",
		DocumentID: "doc1",
		BlockID:    block2ID,
		Operation:  addBlock2,
	},
	})
	require.NoError(t, err)

	// Add some text to the block
	inserts = CreateInsertsFromString("World!", "temp", block2ID, 9)
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Add a bold mark to the text
	boldMark := types.CRDTAddMark{
		OpID: "15@temp",
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
		Type:       types.CRDTAddMarkType,
		Origin:     "temp",
		DocumentID: "doc1",
		BlockID:    block1ID,
		Operation:  boldMark,
	},
	})

	require.NoError(t, err)

	// Compile the document
	doc, err = peer.CompileDocument("doc1")
	require.NoError(t, err)

	expected = "[{\"id\":\"1@temp\",\"type\":\"paragraph\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\"},\"content\":[{\"type\":\"text\",\"charIds\":[\"2@temp\",\"3@temp\",\"4@temp\",\"5@temp\",\"6@temp\",\"7@temp\"],\"text\":\"Hello!\",\"styles\":{\"bold\":true}}],\"children\":[]},{\"id\":\"8@temp\",\"type\":\"heading\",\"props\":{\"textColor\":\"default\",\"backgroundColor\":\"default\",\"textAlignment\":\"left\",\"level\":1},\"content\":[{\"type\":\"text\",\"charIds\":[\"9@temp\",\"10@temp\",\"11@temp\",\"12@temp\",\"13@temp\",\"14@temp\"],\"text\":\"World!\",\"styles\":{}}],\"children\":[]}]"
	require.JSONEq(t, expected, doc)
}

func Test_Document_Compilation_1Peer_MultipleStyles(t *testing.T) {
	transp := channel.NewTransport()
	peer := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(1))
	defer peer.Stop()

	block1ID := "1" + "@temp"
	addBlock1 := types.CRDTAddBlock{
		OpID:        block1ID,
		AfterBlock:  "", // TODO
		ParentBlock: "", // TODO
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
		Type:       types.CRDTAddBlockType,
		Origin:     "temp",
		DocumentID: "doc1",
		BlockID:    block1ID,
		Operation:  addBlock1,
	}})
	require.NoError(t, err)
	// Add some text to the block
	inserts := CreateInsertsFromString("Hello World!", "temp", block1ID, 2) // last opId is 13
	err = peer.UpdateEditor(inserts)
	require.NoError(t, err)

	// Add MarkStyles to the text

	// Add a bold mark to the text
	boldMark := types.CRDTAddMark{
		OpID: "15@temp",
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
		Type:       types.CRDTAddMarkType,
		Origin:     "temp",
		DocumentID: "doc1",
		BlockID:    block1ID,
		Operation:  boldMark,
	},
	})

}

func Test_Document_Compilation_1Peer_MultipleBlocks(t *testing.T) {

}

// Check that the CompileDocument can generate a json string from the editor of two peers
func Test_Document_Compilation_2Peers(t *testing.T) {

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

	docTimestampThreshold := time.Second * 2
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
