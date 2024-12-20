// tests/nodeUtils.test.ts

import { Node, ResolvedPos } from "prosemirror-model";
import { Transform } from "prosemirror-transform";
import {
  findBlockNode,
  findBlockNodeAndDepth,
  getParentBlockID,
  getAfterBlockID,
  getInsertedBlockIDFromStep,
  findNestedBlockIDInDoc,
  findBlockNodeById,
  getBlockProps,
} from "../../lib/operations/nodeUtils";

describe("findBlockNode", () => {
  it("should return null if no block node is found", () => {
    const mockResolvedPos = {
      depth: 0,
      node: () => null,
    } as unknown as ResolvedPos;
    const result = findBlockNode(mockResolvedPos);
    expect(result).toBeNull();
  });
});

describe("findBlockNodeAndDepth", () => {
  it("should return block node and depth if block node is found", () => {
    const mockResolvedPos = {
      depth: 1,
      node: (depth: number) =>
        depth === 1 ? { isBlock: true, attrs: { id: "block1" } } : null,
    } as unknown as ResolvedPos;
    const result = findBlockNodeAndDepth(mockResolvedPos);
    expect(result).toEqual({
      node: { isBlock: true, attrs: { id: "block1" } },
      depth: 1,
    });
  });

  it("should return null if no block node is found", () => {
    const mockResolvedPos = {
      depth: 0,
      node: () => null,
    } as unknown as ResolvedPos;
    const result = findBlockNodeAndDepth(mockResolvedPos);
    expect(result).toBeNull();
  });
});

describe("getParentBlockID", () => {
  it("should return the parent block ID", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({
          depth: 1,
          node: (depth: number) =>
            depth === 1 ? { isBlock: true, attrs: { id: "block1" } } : null,
        }),
      },
    } as unknown as Transform;
    const result = getParentBlockID(mockTransform, 1);
    expect(result).toBe("block1");
  });

  it("should return null if no parent block ID is found", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({
          depth: 0,
          node: () => null,
        }),
      },
    } as unknown as Transform;
    const result = getParentBlockID(mockTransform, 1);
    expect(result).toBeNull();
  });
});

describe("getAfterBlockID", () => {
  it("should return the block ID after the given position", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({
          parent: {
            child: (index: number) =>
              index === 0 ? { isBlock: true, attrs: { id: "block1" } } : null,
          },
          index: () => 1,
        }),
      },
    } as unknown as Transform;
    const result = getAfterBlockID(mockTransform, 1);
    expect(result).toBe("block1");
  });

  it("should return null if no block ID is found after the given position", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({
          parent: {
            child: () => null,
          },
          index: () => 0,
        }),
      },
    } as unknown as Transform;
    const result = getAfterBlockID(mockTransform, 1);
    expect(result).toBeNull();
  });
});

describe("getInsertedBlockIDFromStep", () => {
  it("should return the block ID from the step", () => {
    const mockStep = {
      slice: {
        content: {
          forEach: (
            callback: (arg0: {
              isBlock: boolean;
              attrs: { id: string };
            }) => void
          ) => {
            callback({ isBlock: true, attrs: { id: "block1" } });
          },
        },
      },
    } as unknown as any;
    const result = getInsertedBlockIDFromStep(mockStep);
    expect(result).toBe("block1");
  });

  it("should return null if no block ID is found", () => {
    const mockStep = {
      slice: {
        content: {
          forEach: () => {},
        },
      },
    } as unknown as any;
    const result = getInsertedBlockIDFromStep(mockStep);
    expect(result).toBeNull();
  });
});

describe("findNestedBlockIDInDoc", () => {
  it("should return the nested block ID in the document", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({
          node: () => ({
            descendants: (
              callback: (arg0: {
                isBlock: boolean;
                attrs: { id: string };
              }) => void
            ) => {
              callback({ isBlock: true, attrs: { id: "block1" } });
            },
          }),
        }),
      },
    } as unknown as any;
    const result = findNestedBlockIDInDoc(mockTransform, 1);
    expect(result).toBe("block1");
  });

  it("should return null if no nested block ID is found", () => {
    const mockTransform = {
      doc: {
        resolve: () => ({
          node: () => ({
            descendants: () => {},
          }),
        }),
      },
    } as unknown as any;
    const result = findNestedBlockIDInDoc(mockTransform, 1);
    expect(result).toBeNull();
  });
});

describe("getBlockProps", () => {
  it("should return the block properties", () => {
    const mockBlockNode = {
      attrs: { id: "block1", type: "paragraph", extraProp: "value" },
      content: {
        firstChild: {
          type: { name: "text" },
          attrs: { specificProp: "specificValue" },
        },
      },
    } as unknown as Node;
    const result = getBlockProps(mockBlockNode);
    expect(result).toEqual({
      type: "paragraph",
      extraProp: "value",
      specificProp: "specificValue",
    });
  });
});
