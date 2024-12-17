import { PartialBlock } from "@blocknote/core";
import { types } from "../../wailsjs/go/models";
import { TEMP_NODE_ID } from "./node.type";
import { BlockTypeName, CRDTOpType } from "./operations.type";

const EMPTY_DOC: PartialBlock[] = [
  {
    id: "1@temp",
    type: "paragraph",
    content: [],
  },
  {
    id: "2@temp",
    type: "paragraph",
    content: [],
  },
];

const EMPTY_DOC_HISTORY: types.CRDTOperation[] = [
  new types.CRDTOperation({
    Type: CRDTOpType.AddBlock,
    OperationID: 1,
    Origin: TEMP_NODE_ID,
    BlockID: "1@temp",
    DocumentID: "doc1",
    Operation: new types.CRDTAddBlock({
      BlockType: BlockTypeName.Paragraph,
    }),
  }),
  new types.CRDTOperation({
    Type: CRDTOpType.AddBlock,
    OperationID: 2,
    Origin: TEMP_NODE_ID,
    BlockID: "2@temp",
    DocumentID: "doc1",
    Operation: new types.CRDTAddBlock({
      BlockType: BlockTypeName.Paragraph,
      AfterBlock: "1@temp",
    }),
  }),
];

export type MapCharID = Record<string, string[]>;

export { EMPTY_DOC, EMPTY_DOC_HISTORY };
