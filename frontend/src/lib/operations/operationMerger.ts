import { CRDTOpType } from "@/types/operations.type";
import { types } from "../../../wailsjs/go/models";

/**
 * This function takes the finalOps array and, if multiple updateBlock operations are found,
 * merges their partial info into a single updateBlock operation.
 *
 * @param finalOps Array of CRDTOperation objects
 * @returns Array of CRDTOperation objects
 *
 * @example # Example scenarios:
 *
 * ## 1. Two updateBlock operations with different updatedBlock and afterBlock:
 *
 * finalOps:
 * ```javascript
 * [
 *   {
 *     action: "updateBlock",
 *     updatedBlock: "5c35eee7...",
 *     afterBlock: null,
 *     parentBlock: null,
 *     props: { type: "heading", ... }
 *   },
 *   {
 *     action: "updateBlock",
 *     updatedBlock: null,
 *     afterBlock: "659da16f...",
 *     parentBlock: null,
 *     props: {}
 *   }
 * ]
 * ```
 *
 * We want to merge these two:
 * Final result:
 * ```javascript
 * {
 *   action: "updateBlock",
 *   updatedBlock: "5c35eee7...",  // from first op
 *   afterBlock: "659da16f...",    // from second op
 *   parentBlock: null,
 *   props: { type: "heading", ... } // Prefer non-empty props from earlier if second is empty, or merge
 * }
 * ```
 *
 * ## 2. An addBlock operation followed by an updateBlock operation:
 *
 * finalOps:
 * ```javascript
 * [
 *   {
 *       "Type": "addBlock",
 *       "Origin": TEMP_NODE_ID,
 *       "OperationID": 100,
 *       "DocumentID": "doc1",
 *       "BlockID": null,
 *       "Operation": {
 *           "AfterBlock": "1@temp",
 *           "ParentBlock": null,
 *           "Props": {}
 *       }
 *   },
 *   {
 *       "Type": "updateBlock",
 *       "Origin": TEMP_NODE_ID,
 *       "OperationID": 101,
 *       "DocumentID": "doc1",
 *       "BlockID": "8860708c-50a6-4d09-bff7-86e24dde5b1a",
 *       "Operation": {
 *           "AfterBlock": "1@temp",
 *           "ParentBlock": null,
 *           "Props": {
 *               "type": "paragraph",
 *               "textColor": "default",
 *               "backgroundColor": "default"
 *           }
 *       }
 *   }
 * ]
 * ```
 *
 * We want to merge these two:
 *
 * Final result:
 * ```javascript
 * [
 *  {
 *   "Type": "addBlock",
 *   "Origin": TEMP_NODE_ID,
 *   "OperationID": 100,
 *   "DocumentID": "doc1",
 *   "BlockID": "8860708c-50a6-4d09-bff7-86e24dde5b1a",
 *   "Operation": {
 *     "AfterBlock": "1@temp",
 *     "ParentBlock": null,
 *     "Props": {
 *       "type": "paragraph",
 *       "textColor": "default",
 *       "backgroundColor": "default"
 *     }
 *   }
 *  }
 * ]
 * ```
 *
 * ## 3. A removeBlock operation followed by an addBlock operation (displacement of block):
 *
 * finalOps:
 * ```javascript
 * [
 *  {
 *      "Type": "removeBlock",
 *      "Origin": TEMP_NODE_ID,
 *      "OperationID": 100,
 *      "DocumentID": "doc1",
 *      "BlockID": "1@temp",
 *      "Operation": {
 *          "RemovedBlock": "1@temp"
 *      }
 *  },
 *  {
 *      "Type": "addBlock",
 *      "Origin": TEMP_NODE_ID,
 *      "OperationID": 101,
 *      "DocumentID": "doc1",
 *      "BlockID": "1@temp",
 *      "Operation": {
 *          "AfterBlock": "2@temp",
 *          "ParentBlock": null,
 *          "Props": {}
 *      }
 *  }
 * ]
 * ```
 *
 * We want to merge these two:
 *
 * Final result:
 * ```javascript
 * [
 *   {
 *     "Type": "updateBlock",
 *     "Origin": TEMP_NODE_ID,
 *     "OperationID": 100,
 *     "DocumentID": "doc1",
 *     "BlockID": "1@temp",
 *     "Operation": {
 *       "AfterBlock": "2@temp",
 *       "ParentBlock": null,
 *       "Props": {}
 *     }
 *   }
 * ]
 * ```
 */
function combineUpdateBlockFinalOps(
  finalOps: types.CRDTOperation[]
): types.CRDTOperation[] {
  let ops = [...finalOps];

  // 1. Handle removeBlock + addBlock => updateBlock
  // If we find a removeBlock immediately followed by an addBlock,
  // we convert them into a single updateBlock op.
  // Example:
  // removeBlock("blockA"), addBlock(... with same position)
  // => updateBlock("blockA", final afterBlock and props from the addBlock)
  ops = mergeRemoveBlockAddBlock(ops);

  // 2. Handle addBlock + updateBlock => single addBlock
  // If we find an addBlock followed by an updateBlock for the same block (or where updateBlock clarifies the block ID),
  // merge them so that the addBlock ends up with final BlockID and Props.
  ops = mergeAddBlockUpdateBlock(ops);

  // 3. Merge multiple updateBlock ops if there are more than one
  ops = mergeMultipleUpdateBlockOps(ops);

  return ops;
}

/**
 * This function merges a removeBlock operation followed by an addBlock operation into a single updateBlock operation.
 * The final updateBlock operation will have the BlockID from the addBlock operation and the AfterBlock from the removeBlock operation.
 * The Origin and DocumentID will be kept from the removeBlock operation.
 * The OperationType will be set to "updateBlock".
 *
 * @param ops Array of CRDTOperation objects
 * @returns Array of CRDTOperation objects
 */
function mergeRemoveBlockAddBlock(
  ops: types.CRDTOperation[]
): types.CRDTOperation[] {
  const merged = [];
  for (let i = 0; i < ops.length; i++) {
    const current = ops[i];
    const next = ops[i + 1];
    if (
      current.Type === CRDTOpType.RemoveBlock &&
      next &&
      next.Type === CRDTOpType.AddBlock
    ) {
      // Convert removeBlock+addBlock into updateBlock
      // We'll use the OperationID of the removeBlock or choose how to handle it:
      // Usually, we want the final result to appear as an updateBlock with the removeBlock's OperationID.
      // The final BlockID should be the one from addBlock if it sets one, else keep the old.
      const blockId = next.BlockID || current.BlockID;
      const afterBlock =
        next.Operation.AfterBlock !== null ? next.Operation.AfterBlock : null;
      const parentBlock =
        next.Operation.ParentBlock !== null ? next.Operation.ParentBlock : null;
      const props = next.Operation.Props || {};

      merged.push(
        new types.CRDTOperation({
          Type: CRDTOpType.UpdateBlock,
          Origin: current.Origin,
          OperationID: current.OperationID, // keep the current op's ID
          DocumentID: current.DocumentID,
          BlockID: blockId,
          Operation: new types.CRDTUpdateBlock({
            AfterBlock: afterBlock,
            ParentBlock: parentBlock,
            Props: props,
          }),
        })
      );
      i++; // skip the next since we merged it
    } else {
      merged.push(current);
    }
  }
  return merged;
}

/**
 * This function merges an addBlock operation followed by an updateBlock operation into a single addBlock operation.
 * The final addBlock operation will have the BlockID from the updateBlock operation and the Props from both operations.
 * The Origin and DocumentID will be kept from the addBlock operation.
 * The OperationType will be set to "addBlock".
 *
 * @param ops Array of CRDTOperation objects
 * @returns Array of CRDTOperation objects
 */
function mergeAddBlockUpdateBlock(
  ops: types.CRDTOperation[]
): types.CRDTOperation[] {
  const merged = [];
  for (let i = 0; i < ops.length; i++) {
    const current = ops[i];
    const next = ops[i + 1];
    if (
      current.Type === CRDTOpType.AddBlock &&
      next &&
      next.Type === CRDTOpType.UpdateBlock
    ) {
      // Merge into a single addBlock with final props and BlockID from updateBlock
      const finalBlockID = next.BlockID || current.BlockID;
      const afterBlock =
        next.Operation.AfterBlock !== null
          ? next.Operation.AfterBlock
          : current.Operation.AfterBlock;
      const parentBlock =
        next.Operation.ParentBlock !== null
          ? next.Operation.ParentBlock
          : current.Operation.ParentBlock;
      // Merge props
      const finalProps = {
        ...(current.Operation.Props || {}),
        ...(next.Operation.Props || {}),
      };

      merged.push(
        new types.CRDTOperation({
          Type: CRDTOpType.AddBlock,
          Origin: current.Origin,
          OperationID: current.OperationID, // Keep the current operation's ID
          DocumentID: current.DocumentID,
          BlockID: finalBlockID,
          Operation: new types.CRDTAddBlock({
            BlockType: next.Operation.BlockType || current.Operation.BlockType,
            AfterBlock: afterBlock,
            ParentBlock: parentBlock,
            Props: finalProps,
          }),
        })
      );
      i++; // Skip the next op
    } else {
      merged.push(current);
    }
  }
  return merged;
}

/**
 * This function merges multiple updateBlock operations into a single updateBlock operation.
 * The final updateBlock operation will have the BlockID from the first operation, and the AfterBlock, ParentBlock, and Props from all operations.
 * The Origin and DocumentID will be kept from the first operation.
 *
 * @param ops Array of CRDTOperation objects
 * @returns Array of CRDTOperation objects
 */
function mergeMultipleUpdateBlockOps(
  ops: types.CRDTOperation[]
): types.CRDTOperation[] {
  const updateOps = ops.filter((op) => op.Type === CRDTOpType.UpdateBlock);
  if (updateOps.length <= 1) return ops;

  // Merge all updateBlock ops into the first one
  const others = ops.filter((op) => op.Type !== CRDTOpType.UpdateBlock);
  // Start with the first updateBlock op as baseline
  let combined = { ...updateOps[0] };

  for (let i = 1; i < updateOps.length; i++) {
    const op = updateOps[i];
    // If BlockID is missing in combined and present in op, update it
    if (!combined.BlockID && op.BlockID) {
      combined.BlockID = op.BlockID;
    }

    // Merge afterBlock
    if (op.Operation.AfterBlock !== null) {
      combined.Operation.AfterBlock = op.Operation.AfterBlock;
    }

    // Merge parentBlock
    if (op.Operation.ParentBlock !== null) {
      combined.Operation.ParentBlock = op.Operation.ParentBlock;
    }

    // Merge BlockType
    if (op.Operation.BlockType) {
      combined.Operation.BlockType = op.Operation.BlockType;
    }

    // Merge props
    if (op.Operation.Props && Object.keys(op.Operation.Props).length > 0) {
      combined.Operation.Props = {
        ...(combined.Operation.Props || {}),
        ...op.Operation.Props,
      };
    }
  }

  others.push(combined);
  return others;
}

/**
 * Fix addBlock operations by assigning proper block IDs if missing
 * @param finalOps - The final operations to fix
 * @returns The fixed operations
 */
const fixAddBlockOperations = (finalOps: types.CRDTOperation[]) => {
  // Map from oldBlockID -> newBlockID
  const blockIdMap = {} as Record<string, string>;

  // 1. First pass: find addBlock ops and assign proper block IDs if missing
  for (const op of finalOps) {
    if (op.Type === CRDTOpType.AddBlock) {
      // Check if blockId is missing or not proper
      // Define your condition here. For example, if blockId is null or empty:
      if (!op.BlockID || op.BlockID === null || !op.BlockID.endsWith("@temp")) {
        const newBlockID = op.OperationID + "@temp";
        if (op.BlockID && op.BlockID !== newBlockID) {
          blockIdMap[op.BlockID] = newBlockID;
        } else if (!op.BlockID) {
          // If there was no oldBlockID at all
          blockIdMap["__no_id__" + op.OperationID] = newBlockID;
        }
        op.BlockID = newBlockID;
      } else {
        // If the blockId is "improper" by some definition, set a rule here.
        // For now, assume if it's valid we leave it.
        // If you want to always use OperationId for addBlock:
        const oldId = op.BlockID;
        const newBlockID = op.OperationID.toString() + "@temp";
        if (oldId !== newBlockID) {
          blockIdMap[oldId] = newBlockID;
          op.BlockID = newBlockID;
        }
      }
    }
  }

  // If no mapping is needed, return early
  if (Object.keys(blockIdMap).length === 0) {
    return finalOps;
  }

  // 2. Second pass: Update references in all operations using blockIdMap
  for (const op of finalOps) {
    // Update op.BlockID if it exists in the map
    if (op.BlockID && blockIdMap[op.BlockID]) {
      op.BlockID = blockIdMap[op.BlockID];
    }

    // The operation field may have AfterBlock, ParentBlock, RemovedBlock, UpdatedBlock, etc.
    // Update these if they match old IDs
    if (op.Operation) {
      if (op.Operation.AfterBlock && blockIdMap[op.Operation.AfterBlock]) {
        op.Operation.AfterBlock = blockIdMap[op.Operation.AfterBlock];
      }
      if (op.Operation.ParentBlock && blockIdMap[op.Operation.ParentBlock]) {
        op.Operation.ParentBlock = blockIdMap[op.Operation.ParentBlock];
      }
      if (op.Operation.RemovedBlock && blockIdMap[op.Operation.RemovedBlock]) {
        op.Operation.RemovedBlock = blockIdMap[op.Operation.RemovedBlock];
      }
      if (op.Operation.updatedBlock && blockIdMap[op.Operation.updatedBlock]) {
        op.Operation.updatedBlock = blockIdMap[op.Operation.updatedBlock];
      }
    }
  }

  return finalOps;
};

export {
  combineUpdateBlockFinalOps,
  mergeRemoveBlockAddBlock,
  mergeAddBlockUpdateBlock,
  mergeMultipleUpdateBlockOps,
  fixAddBlockOperations,
};
