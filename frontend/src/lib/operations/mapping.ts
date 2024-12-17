import {
  AddMarkStep,
  RemoveMarkStep,
  ReplaceStep,
  ReplaceAroundStep,
  Transform,
} from "prosemirror-transform";

import { types } from "../../../wailsjs/go/models";
import { Node } from "prosemirror-model";
import { findBlockNode } from "./nodeUtils";
import { combineUpdateBlockFinalOps } from "./operationMerger";
import { buildFinalOperationObject } from "./operationBuilder";
import { CRDTOpType } from "@/types/operations.type";
import { MapCharID } from "@/types/docs.type";

/**
 * This function takes a ProseMirror transaction and maps it to a list of CRDTOperations.
 * It also returns the next operation ID and the updated character ID map.
 *
 * @param tr ProseMirror transaction
 * @param oldDoc Old ProseMirror document
 * @param currentNextOpId Current operation ID
 * @param currentCharIds Current character ID map
 * @returns [CRDTOperation[], number, MapCharID]
 */
function mapTransactionToOperations(
  tr: Transform,
  oldDoc: Node,
  currentNextOpId: number,
  currentCharIds: MapCharID
): [types.CRDTOperation[], number, MapCharID] {
  const intermediateOps = [];
  let nextOpId = currentNextOpId;
  const nextCharIds = { ...currentCharIds } as MapCharID;

  // If no steps, return empty array
  if (tr.steps.length === 0) {
    return [[], nextOpId, nextCharIds];
  }

  // Determine intermediate operations
  for (const step of tr.steps) {
    let operationType = null;
    let blockId = null;
    let from, to;

    // Get from and to positions
    if (step instanceof AddMarkStep || step instanceof RemoveMarkStep) {
      from = step.from;
      to = step.to;
    } else if (step instanceof ReplaceStep) {
      from = step.from;
      to = step.to;
    } else if (step instanceof ReplaceAroundStep) {
      from = step.from;
      to = step.to;
    } else {
      continue;
    }

    // Find the block corresponding to the positions
    const $from = tr.doc.resolve(from);
    const blockNode = findBlockNode($from);
    if (blockNode) {
      blockId = blockNode.attrs.id;
    }

    // Process each type of step
    if (step instanceof AddMarkStep) {
      operationType = CRDTOpType.AddMark;
    } else if (step instanceof RemoveMarkStep) {
      operationType = CRDTOpType.RemoveMark;
    } else if (step instanceof ReplaceStep) {
      const insertedContent = step.slice.content;
      const insertedCount = insertedContent.childCount;
      const replacedLength = to - from;

      if (insertedCount > 0 && replacedLength === 0) {
        let insertedBlock = false;
        insertedContent.forEach((node) => {
          if (node.isBlock) {
            insertedBlock = true;
            blockId = node.attrs.id;
          }
        });
        if (insertedBlock) {
          operationType = CRDTOpType.AddBlock;
        } else {
          if (
            insertedCount === 1 &&
            insertedContent.firstChild &&
            insertedContent.firstChild.isText
          ) {
            const text = insertedContent.firstChild?.text || "";
            operationType =
              text.length === 1 ? CRDTOpType.InsertChar : CRDTOpType.InsertText;
          } else {
            operationType = CRDTOpType.InsertText;
          }
        }
      } else if (insertedCount === 0 && replacedLength > 0) {
        // Check what was removed
        const removedSlice = oldDoc?.slice(from, to) || null;
        let removedBlock = false;
        removedSlice?.content.forEach((node: Node) => {
          if (node.isBlock) {
            blockId = node.attrs.id;
            removedBlock = true;
          }
        });
        operationType = removedBlock
          ? CRDTOpType.RemoveBlock
          : CRDTOpType.DeleteChar;
      } else if (insertedCount > 0 && replacedLength > 0) {
        // Mixed insertion and deletion
        const removedSlice = oldDoc?.slice(from, to);
        let removedBlock = false;
        removedSlice?.content.forEach((node: Node, i: number) => {
          if (node.isBlock) {
            if (removedSlice.content.childCount === 3 && i === 2)
              blockId = node.attrs.id;
            else if (removedSlice.content.childCount === 2)
              blockId = node.attrs.id;
            else if (
              removedSlice.content.childCount === 1 &&
              node.childCount > 0
            )
              blockId = node.child(0).attrs.id;
            removedBlock = true;
          }
        });

        let insertedBlock = false;
        step.slice.content.forEach((node) => {
          if (node.isBlock) insertedBlock = true;
        });

        if (removedBlock) {
          operationType = CRDTOpType.RemoveBlock;
        } else {
          // Removed text and inserted something else
          // Possibly insertText or updateBlock depending on logic
          operationType = insertedBlock
            ? CRDTOpType.AddBlock
            : CRDTOpType.InsertText;
        }
      } else {
        operationType = CRDTOpType.UpdateBlock;
      }
    } else if (step instanceof ReplaceAroundStep) {
      const insertedContent = step.slice.content;
      let insertedBlock = false;
      insertedContent.forEach((node) => {
        if (node.isBlock) {
          insertedBlock = true;
          blockId = node.attrs.id;
        }
      });

      // Consider this restructuring as updateBlock
      operationType = insertedBlock ? CRDTOpType.UpdateBlock : null;
      if (!operationType) continue;
    }

    intermediateOps.push({ operationType, blockId, from, to, step });
  }

  // Convert intermediate ops to final ops
  const finalOps = intermediateOps.map((op) =>
    buildFinalOperationObject(tr, op, nextCharIds, nextOpId++)
  );

  // Now that we have final operations, combine updateBlock operations if partial
  const combinedFinalOps = combineUpdateBlockFinalOps(
    finalOps.filter((op) => op !== null)
  );

  // Return a single-element array with the chosen operation
  return [combinedFinalOps, nextOpId, nextCharIds];
}

export { mapTransactionToOperations };
