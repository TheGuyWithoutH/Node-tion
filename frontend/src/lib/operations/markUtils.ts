import { AddMarkStep, RemoveMarkStep, Transform } from "prosemirror-transform";
import { findBlockNodeAndDepth } from "./nodeUtils";
import { types } from "../../../wailsjs/go/models";
import { LinkType, MarkType } from "@/types/operations.type";
import { MapCharID } from "@/types/docs.type";

/**
 * Get the mark boundaries for a given transaction. The mark boundaries are the start
 * and end boundaries of the mark. The start boundary is the character ID just before
 * the mark starts, and the end boundary is the character ID just after the mark ends.
 *
 * @param tr - The transaction to get the mark boundaries from
 * @param from - The start position of the mark
 * @param to - The end position of the mark
 * @param charMap - The character map to get the mark boundaries from
 *
 * @returns The start and end boundaries of the mark
 */
function getMarkBoundaries(
  tr: Transform,
  from: number,
  to: number,
  charMap: MapCharID
) {
  const $fromOld = tr.doc.resolve(from);
  const fromInfo = findBlockNodeAndDepth($fromOld);
  if (!fromInfo) return { start: null, end: null };

  const { node: fromBlockNode, depth: fromDepth } = fromInfo;
  const fromBlockID = fromBlockNode.attrs.id;
  const fromBlockStart = $fromOld.start(fromDepth);
  const fromOffset = from - fromBlockStart;

  const $toOld = tr.doc.resolve(to);
  const toInfo = findBlockNodeAndDepth($toOld);
  if (!toInfo) return { start: null, end: null };

  const { node: toBlockNode, depth: toDepth } = toInfo;
  const toBlockID = toBlockNode.attrs.id;
  const toBlockStart = $toOld.start(toDepth);
  const toOffset = to - toBlockStart - 1; // -1 to get the character before pos of first char after mark

  // Check if mark spans multiple blocks
  if (fromBlockID !== toBlockID) {
    // If you need multi-block support, handle it here.
    // For now, just return null as a placeholder.
    return { start: null, end: null };
  }

  const charIds = charMap[fromBlockID];
  if (!charIds) return { start: null, end: null };

  // Mark covers chars from fromOffset to (toOffset - 1)
  // The start boundary: just before fromOffset
  // If fromOffset is 0, the mark starts at the very beginning, so start boundary might have no preceding char.
  const startCharId = fromOffset > 0 ? charIds[fromOffset - 1] : null;
  const start = new types.MarkStart({ Type: "before", OpID: startCharId });

  // The end boundary: just after the last affected character at toOffset - 1
  // If toOffset <= fromOffset, no chars affected, but typically to > from.
  const endCharId =
    toOffset > 0 && toOffset <= charIds.length ? charIds[toOffset - 1] : null;
  const end = new types.MarkEnd({ Type: "after", OpID: endCharId });

  return { start, end };
}

/**
 * Get the options for a mark based on the step that added or removed the mark.
 *
 * @param markType - The type of mark to get options for
 * @param step - The step that added or removed the mark
 *
 * @returns The options for the mark (namely color or link)
 */
function getMarkOptions(markType: string, step: AddMarkStep | RemoveMarkStep) {
  if (markType === MarkType.TextColor) {
    return { color: step.mark.attrs.stringValue };
  } else if (markType === MarkType.BackgroundColor) {
    return { color: step.mark.attrs.stringValue };
  } else if (markType === LinkType) {
    return { link: step.mark.attrs.href };
  } else {
    return {};
  }
}

export { getMarkBoundaries, getMarkOptions };
