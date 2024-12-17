import { MapCharID } from "@/types/docs.type";
import { PartialBlock } from "@blocknote/core";

/**
 * Extracts the character IDs from a document
 * @param document - The document to extract character IDs from
 * @returns A map of block IDs to character IDs
 */
const extractCharIds = (document: PartialBlock[]) => {
  const charMap: MapCharID = {};

  for (const block of document) {
    const flattenedCharIds = [];
    if (!block.content) {
      continue;
    }

    for (const contentItem of Array.isArray(block.content)
      ? block.content
      : []) {
      // @ts-ignore
      if (contentItem.charIds && Array.isArray(contentItem.charIds)) {
        // @ts-ignore
        flattenedCharIds.push(...contentItem.charIds);
      }
    }
    charMap[block.id || "id"] = flattenedCharIds;
  }

  return charMap;
};

export { extractCharIds };
