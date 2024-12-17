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

const EMPTY_DOC_HISTORY: (documentId: string) => types.CRDTOperation[] = (
  documentId: string
) => [
  new types.CRDTOperation({
    Type: CRDTOpType.AddBlock,
    OperationID: 1,
    Origin: TEMP_NODE_ID,
    BlockID: "1@temp",
    DocumentID: documentId,
    Operation: new types.CRDTAddBlock({
      BlockType: BlockTypeName.Paragraph,
    }),
  }),
  new types.CRDTOperation({
    Type: CRDTOpType.AddBlock,
    OperationID: 2,
    Origin: TEMP_NODE_ID,
    BlockID: "2@temp",
    DocumentID: documentId,
    Operation: new types.CRDTAddBlock({
      BlockType: BlockTypeName.Paragraph,
      AfterBlock: "1@temp",
    }),
  }),
];

export type MapCharID = Record<string, string[]>;

export { EMPTY_DOC, EMPTY_DOC_HISTORY };
