import { Transform } from "prosemirror-transform";
import { findBlockNodeAndDepth } from "./nodeUtils";
import { CRDTOpType } from "@/types/operations.type";
import { MapCharID } from "@/types/docs.type";

/**
 * Get the character id before the given position
 * @param tr - The transaction to get the character id from
 * @param pos - The position to get the character id before
 * @param charMap - The character map to get the character id from
 * @returns The character id before the given position
 */
function getCharacterIdBefore(tr: Transform, pos: number, charMap: MapCharID) {
  const $pos = tr.doc.resolve(pos);
  const blockInfo = findBlockNodeAndDepth($pos);
  if (!blockInfo) return null;

  const { node: blockNode, depth } = blockInfo;
  const blockId = blockNode.attrs.id;
  const blockStart = $pos.start(depth);
  const offsetInBlock = pos - blockStart - 1; // -1 to get the character before new pos

  const charIds = charMap[blockId];
  if (!charIds) return null;

  // If offsetInBlock is 0, no character before pos
  if (offsetInBlock > 0) {
    return charIds[offsetInBlock - 1];
  }

  return null;
}

/**
 * Get the character id removed at the given position
 * @param tr - The transaction to get the character id from
 * @param pos - The position to get the character id removed
 * @param charMap - The character map to get the character id from
 * @returns The character id removed at the given position
 */
function getCharacterIdRemoved(tr: Transform, pos: number, charMap: MapCharID) {
  const $pos = tr.doc.resolve(pos);
  const blockInfo = findBlockNodeAndDepth($pos);
  if (!blockInfo) return null;

  const { node: blockNode, depth } = blockInfo;
  const blockId = blockNode.attrs.id;
  const blockStart = $pos.start(depth);
  const offsetInBlock = pos - blockStart;

  const charIds = charMap[blockId];
  if (!charIds) return null;

  // If offsetInBlock is 0, no character before pos
  if (offsetInBlock > 0) {
    return charIds[offsetInBlock - 1];
  }

  return null;
}

/**
 * Update the character map with the new character id
 * @param charMap - The character map to update
 * @param blockId - The block id to update
 * @param refCharId - The reference character id
 * @param newCharId - The new character id
 * @param action - The action to perform (insert or delete)
 */
const updateMapCharIds = (
  charMap: MapCharID,
  blockId: string,
  refCharId: string,
  newCharId: string,
  action: CRDTOpType.InsertChar | CRDTOpType.DeleteChar = CRDTOpType.InsertChar
) => {
  if (!charMap[blockId]) charMap[blockId] = [];
  const index = charMap[blockId].indexOf(refCharId);
  if (action === CRDTOpType.InsertChar) {
    if (index >= 0) {
      charMap[blockId].splice(index + 1, 0, newCharId);
    } else {
      charMap[blockId] = [newCharId, ...charMap[blockId]];
    }
  } else {
    if (index >= 0) {
      charMap[blockId].splice(index, 1);
    }
  }
};

export { getCharacterIdBefore, getCharacterIdRemoved, updateMapCharIds };
