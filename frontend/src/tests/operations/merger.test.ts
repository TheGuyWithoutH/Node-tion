import { describe, test, expect } from "@jest/globals";
import { BlockTypeName, CRDTOpType } from "@/types/operations.type";
import {
  mergeAddBlockUpdateBlock,
  mergeMultipleUpdateBlockOps,
  fixAddBlockOperations,
} from "../../lib/operations/operationMerger";
import { types } from "../../../wailsjs/go/models";

describe("Operation Merging", () => {
  test("Merge AddBlock and UpdateBlock", () => {
    const addBlock = new types.CRDTOperation({
      Type: CRDTOpType.AddBlock,
      BlockID: "op1@temp",
      OperationID: "op1",
      DocumentID: "doc1",
      Operation: new types.CRDTAddBlock({
        BlockID: "op1@temp",
        BlockType: BlockTypeName.Paragraph,
        AfterBlock: "",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "red",
          TextColor: "black",
          TextAlignment: "left",
          Level: 1,
        }),
      }),
    });

    const updateBlock = new types.CRDTOperation({
      Type: CRDTOpType.UpdateBlock,
      BlockID: "op1@temp",
      OperationID: "op2",
      DocumentID: "doc1",
      Operation: new types.CRDTUpdateBlock({
        BlockID: "op1@temp",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "blue",
          TextColor: "white",
          TextAlignment: "center",
          Level: 2,
        }),
      }),
    });

    const mergedOps = mergeAddBlockUpdateBlock([addBlock, updateBlock]);

    expect(mergedOps.length).toBe(1);
    const merged = mergedOps[0];
    expect(merged.Type).toBe(CRDTOpType.AddBlock);
    expect(merged.BlockID).toBe("op1@temp");
    expect(merged.Operation.Props).toEqual(
      new types.DefaultBlockProps({
        BackgroundColor: "blue",
        TextColor: "white",
        TextAlignment: "center",
        Level: 2,
      })
    );
  });

  test("Combine Multiple UpdateBlock Operations", () => {
    const updateBlock1 = new types.CRDTOperation({
      Type: CRDTOpType.UpdateBlock,
      BlockID: "op1@temp",
      OperationID: "op1",
      DocumentID: "doc1",
      Operation: new types.CRDTUpdateBlock({
        BlockID: "op1@temp",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "green",
          TextColor: "black",
          TextAlignment: "left",
          Level: 1,
        }),
      }),
    });

    const updateBlock2 = new types.CRDTOperation({
      Type: CRDTOpType.UpdateBlock,
      BlockID: "op1@temp",
      OperationID: "op2",
      DocumentID: "doc1",
      Operation: new types.CRDTUpdateBlock({
        BlockID: "op1@temp",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "yellow",
          TextColor: "blue",
          TextAlignment: "right",
          Level: 3,
        }),
      }),
    });

    const combinedOps = mergeMultipleUpdateBlockOps([
      updateBlock1,
      updateBlock2,
    ]);

    expect(combinedOps.length).toBe(1);
    const combined = combinedOps[0];
    expect(combined.Type).toBe(CRDTOpType.UpdateBlock);
    expect(combined.BlockID).toBe("op1@temp");
    expect(combined.Operation.Props).toEqual(
      new types.DefaultBlockProps({
        BackgroundColor: "yellow",
        TextColor: "blue",
        TextAlignment: "right",
        Level: 3,
      })
    );
  });

  test("Fix AddBlock IDs and Update References", () => {
    const addBlock1 = new types.CRDTOperation({
      Type: CRDTOpType.AddBlock,
      BlockID: "op1@temp",
      OperationID: "op1",
      DocumentID: "doc1",
      Operation: new types.CRDTAddBlock({
        BlockID: "op1@temp",
        BlockType: BlockTypeName.Paragraph,
        AfterBlock: "",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "red",
          TextColor: "black",
          TextAlignment: "left",
          Level: 1,
        }),
      }),
    });

    const addBlock2 = new types.CRDTOperation({
      Type: CRDTOpType.AddBlock,
      BlockID: "op2@temp",
      OperationID: "op2",
      DocumentID: "doc1",
      Operation: new types.CRDTAddBlock({
        BlockID: "op2@temp",
        BlockType: BlockTypeName.Paragraph,
        AfterBlock: "op1@temp",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "blue",
          TextColor: "white",
          TextAlignment: "center",
          Level: 2,
        }),
      }),
    });

    const fixedOps = fixAddBlockOperations([addBlock1, addBlock2]);

    expect(fixedOps.length).toBe(2);
    expect(fixedOps[0].BlockID).toBe("op1@temp");
    expect(fixedOps[1].BlockID).toBe("op2@temp");
  });

  test("Ensure valid operations after transformation", () => {
    const addBlock = new types.CRDTOperation({
      Type: CRDTOpType.AddBlock,
      BlockID: "op1@temp",
      OperationID: "op1",
      DocumentID: "doc1",
      Operation: new types.CRDTAddBlock({
        BlockID: "op1@temp",
        BlockType: BlockTypeName.Paragraph,
        AfterBlock: "",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "red",
          TextColor: "black",
          TextAlignment: "left",
          Level: 1,
        }),
      }),
    });

    const updateBlock = new types.CRDTOperation({
      Type: CRDTOpType.UpdateBlock,
      BlockID: "op1@temp",
      OperationID: "op2",
      DocumentID: "doc1",
      Operation: new types.CRDTUpdateBlock({
        BlockID: "op1@temp",
        Props: new types.DefaultBlockProps({
          BackgroundColor: "blue",
          TextColor: "white",
          TextAlignment: "center",
          Level: 2,
        }),
      }),
    });

    const mergedOps = mergeAddBlockUpdateBlock([addBlock, updateBlock]);
    const fixedOps = fixAddBlockOperations(mergedOps);

    expect(fixedOps.length).toBe(1);
    expect(fixedOps[0].Type).toBe(CRDTOpType.AddBlock);
    expect(fixedOps[0].BlockID).toBe("op1@temp");
    expect(fixedOps[0].Operation.Props).toEqual(
      new types.DefaultBlockProps({
        BackgroundColor: "blue",
        TextColor: "white",
        TextAlignment: "center",
        Level: 2,
      })
    );
  });
});
