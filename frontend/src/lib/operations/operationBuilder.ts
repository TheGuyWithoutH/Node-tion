import {
  AddMarkStep,
  RemoveMarkStep,
  ReplaceStep,
  ReplaceAroundStep,
  Transform,
} from "prosemirror-transform";

import { types } from "../../../wailsjs/go/models";
import {
  findBlockNode,
  findBlockNodeById,
  findNestedBlockIDInDoc,
  getAfterBlockID,
  getBlockProps,
  getInsertedBlockIDFromStep,
  getParentBlockID,
} from "./nodeUtils";
import {
  getCharacterIdBefore,
  getCharacterIdRemoved,
  updateMapCharIds,
} from "./charUtils";
import { getMarkBoundaries, getMarkOptions } from "./markUtils";
import { CRDTOpType } from "@/types/operations.type";
import { MapCharID } from "@/types/docs.type";
import { TEMP_NODE_ID } from "@/types/node.type";

/**
 * This function takes a ProseMirror transaction and a detected operation object,
 * and maps it to the final CRDTOperation object.
 *
 * @param tr ProseMirror transaction
 * @param op Operation object
 * @param charMap Character ID map
 * @param opId Operation ID
 * @returns CRDTOperation | null
 */
function buildFinalOperationObject(
  tr: Transform,
  op: {
    operationType: string | null;
    blockId: any;
    from: number;
    to: number;
    step: AddMarkStep | RemoveMarkStep | ReplaceStep | ReplaceAroundStep;
  },
  charMap: MapCharID,
  opId: number,
  documentId: string
): types.CRDTOperation | null {
  const { operationType, blockId, from, to, step } = op;

  switch (operationType) {
    case CRDTOpType.AddBlock: {
      const afterBlock =
        getAfterBlockID(tr, from) ?? getParentBlockID(tr, from);
      const parentBlock =
        afterBlock === getParentBlockID(tr, from)
          ? null
          : getParentBlockID(tr, from);
      const blockNode = blockId ? findBlockNode(tr.doc.resolve(from)) : null;
      const props = blockNode ? getBlockProps(blockNode) : {};

      charMap[blockId] = [];

      return new types.CRDTOperation({
        Type: CRDTOpType.AddBlock,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: blockId,
        Operation: new types.CRDTAddBlock({
          BlockType: props.type || "paragraph",
          AfterBlock: afterBlock,
          ParentBlock: parentBlock,
          Props: new types.DefaultBlockProps({
            BackgroundColor: props.backgroundColor || "",
            TextColor: props.textColor || "",
            TextAlign: props.textAlign || "",
            Level: props.level || 1,
          }),
        }),
      });
    }

    case CRDTOpType.RemoveBlock: {
      let finalBlockID = blockId;

      return new types.CRDTOperation({
        Type: CRDTOpType.RemoveBlock,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: finalBlockID,
        Operation: new types.CRDTRemoveBlock({
          RemovedBlock: finalBlockID,
        }),
      });
    }

    case CRDTOpType.UpdateBlock: {
      let finalBlockID = null;
      if (step instanceof ReplaceStep || step instanceof ReplaceAroundStep) {
        finalBlockID = getInsertedBlockIDFromStep(step);
      }

      if (!finalBlockID) {
        // If we didn't find a blockId in the inserted slice,
        // we try to find the updated block by inspecting the final doc.
        finalBlockID = findNestedBlockIDInDoc(tr, from);
      }

      // If we still don't have a finalBlockID, fallback to the originally found blockId
      finalBlockID = finalBlockID || blockId;

      const afterBlock = getAfterBlockID(tr, from);
      const parentBlock = getParentBlockID(tr, from);
      const blockNode = finalBlockID
        ? findBlockNodeById(tr, finalBlockID)
        : null;
      const props = blockNode ? getBlockProps(blockNode) : {};

      return new types.CRDTOperation({
        Type: CRDTOpType.UpdateBlock,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: finalBlockID,
        Operation: new types.CRDTUpdateBlock({
          AfterBlock: afterBlock,
          ParentBlock: parentBlock,
          BlockType: props.type || "paragraph",
          Props: new types.DefaultBlockProps({
            BackgroundColor: props.backgroundColor || "",
            TextColor: props.textColor || "",
            TextAlignment: props.textAlignment || "",
            Level: props.level || 1,
          }),
        }),
      });
    }

    case CRDTOpType.InsertChar: {
      if (step instanceof AddMarkStep || step instanceof RemoveMarkStep)
        return null;

      const character = step.slice.content.firstChild?.text || "";
      const afterId = getCharacterIdBefore(tr, from, charMap) || "";

      // Update charMap
      updateMapCharIds(
        charMap,
        blockId,
        afterId,
        opId + "@temp",
        CRDTOpType.InsertChar
      );

      return new types.CRDTOperation({
        Type: CRDTOpType.InsertChar,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: blockId,
        Operation: new types.CRDTInsertChar({
          AfterID: afterId,
          Character: character,
        }),
      });
    }

    case CRDTOpType.DeleteChar: {
      const removedId = getCharacterIdRemoved(tr, from, charMap) || "";

      // Update charMap
      updateMapCharIds(
        charMap,
        blockId,
        removedId,
        opId + "@temp",
        CRDTOpType.DeleteChar
      );

      return new types.CRDTOperation({
        Type: CRDTOpType.DeleteChar,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: blockId,
        Operation: new types.CRDTDeleteChar({
          RemovedID: removedId,
        }),
      });
    }

    case CRDTOpType.AddMark: {
      if (!(step instanceof AddMarkStep)) return null;

      const { start, end } = getMarkBoundaries(tr, from, to, charMap);
      const markType = step.mark.type.name;
      const options = getMarkOptions(markType, step);

      return new types.CRDTOperation({
        Type: CRDTOpType.AddMark,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: blockId,
        Operation: new types.CRDTAddMark({
          Start: start,
          End: end,
          MarkType: markType,
          Options: options,
        }),
      });
    }

    case CRDTOpType.RemoveMark: {
      if (!(step instanceof RemoveMarkStep)) return null;

      const { start, end } = getMarkBoundaries(tr, from, to, charMap);
      const markType = step.mark.type.name;
      return new types.CRDTOperation({
        Type: CRDTOpType.RemoveMark,
        Origin: TEMP_NODE_ID,
        OperationID: opId,
        DocumentID: documentId,
        BlockID: blockId,
        Operation: new types.CRDTRemoveMark({
          Start: start,
          End: end,
          MarkType: markType,
        }),
      });
    }

    case CRDTOpType.InsertText: {
      const text = "example"; // Placeholder
      const afterId = getCharacterIdBefore(tr, from, charMap);

      console.log("Inserting text:", text);
      return null;
      // TODO: Implement InsertText operation by adding characters one by one
    }

    default:
      return null;
  }
}

export { buildFinalOperationObject };
