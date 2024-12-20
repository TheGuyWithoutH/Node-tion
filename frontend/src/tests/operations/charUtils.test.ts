// tests/charUtils.test.ts
import { Transform } from "prosemirror-transform";
import {
  getCharacterIdBefore,
  getCharacterIdRemoved,
  updateMapCharIds,
} from "../../lib/operations/charUtils";
import { CRDTOpType } from "@/types/operations.type";

describe("getCharacterIdBefore", () => {
  it("should return null if there is no block info", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({ depth: 0, node: () => null }),
      },
    } as unknown as Transform;
    const result = getCharacterIdBefore(mockTransform, 1, {});
    expect(result).toBeNull();
  });
});

describe("getCharacterIdRemoved", () => {
  it("should return null if there is no block info", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({ depth: 0, node: () => null }),
      },
    } as unknown as Transform;
    const result = getCharacterIdRemoved(mockTransform, 1, {});
    expect(result).toBeNull();
  });
});

describe("updateMapCharIds", () => {
  it("should insert a new character ID after the reference ID", () => {
    const charMap = {
      block1: ["char1", "char2"],
    };
    updateMapCharIds(
      charMap,
      "block1",
      "char1",
      "charNew",
      CRDTOpType.InsertChar
    );
    expect(charMap.block1).toEqual(["char1", "charNew", "char2"]);
  });

  it("should remove the specified character ID", () => {
    const charMap = {
      block1: ["char1", "char2", "char3"],
    };
    updateMapCharIds(charMap, "block1", "char2", "", CRDTOpType.DeleteChar);
    expect(charMap.block1).toEqual(["char1", "char3"]);
  });

  it("should initialize a block if it does not exist", () => {
    const charMap = {};
    updateMapCharIds(charMap, "block1", "", "charNew", CRDTOpType.InsertChar);
    // @ts-ignore
    expect(charMap.block1).toEqual(["charNew"]);
  });
});
