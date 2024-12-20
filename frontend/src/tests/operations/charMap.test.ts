// tests/charMapUtils.test.ts

import { PartialBlock } from "@blocknote/core";
import { extractCharIds } from "../../lib/operations/charMapUtils";

describe("extractCharIds", () => {
  it("should return an empty map when the document is empty", () => {
    const result = extractCharIds([]);
    expect(result).toEqual({});
  });

  it("should map block IDs to character IDs correctly", () => {
    const mockDocument = [
      {
        id: "block1",
        // @ts-ignore
        content: [{ charIds: ["char1", "char2"] }, { charIds: ["char3"] }],
      },
      {
        id: "block2",
        // @ts-ignore
        content: [{ charIds: ["char4"] }],
      },
    ] as PartialBlock[];

    const result = extractCharIds(mockDocument);

    expect(result).toEqual({
      block1: ["char1", "char2", "char3"],
      block2: ["char4"],
    });
  });

  it("should handle blocks with no content gracefully", () => {
    const mockDocument = [
      { id: "block1", content: null },
      {
        id: "block2",
        // @ts-ignore
        content: [{ charIds: ["char1"] }],
      },
    ] as PartialBlock[];

    const result = extractCharIds(mockDocument);

    expect(result).toEqual({
      block1: [],
      block2: ["char1"],
    });
  });

  it("should use a default block ID if none is provided", () => {
    const mockDocument = [
      {
        // @ts-ignore
        content: [{ charIds: ["char1"] }],
      },
    ] as PartialBlock[];

    const result = extractCharIds(mockDocument);

    expect(result).toHaveProperty("id", ["char1"]);
  });
});
