package tests

import (
	"Node-tion/backend/types"
	"strconv"
)

func CreateInsertsFromString(content string, addr, docID, blockID string, insertStart int) []types.CRDTOperation {
	ops := make([]types.CRDTOperation, len(content))
	for i, char := range content {
		if i == 0 {
			ops[i] = types.CRDTOperation{
				Type:        types.CRDTInsertCharType,
				Origin:      addr,
				OperationID: uint64(i + insertStart),
				DocumentID:  docID,
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

func CreateNewBlockOp(addr, docID, blockID string) []types.CRDTOperation {
	crdtOp := types.CRDTOperation{
		Type:        types.CRDTAddBlockType,
		Origin:      addr,
		OperationID: 1,
		DocumentID:  docID,
		BlockID:     blockID,
		Operation: types.CRDTAddBlock{
			BlockType: types.ParagraphBlockType,
			Props:     types.DefaultBlockProps{},
		},
	}

	ops := []types.CRDTOperation{crdtOp}
	return ops
}
