import { beforeAll, beforeEach, describe, test, expect } from "@jest/globals";
import { Node } from "prosemirror-model";
import { types } from "../../../wailsjs/go/models";
import { Editor } from "@tiptap/core";
import { BlockNoteEditor } from "@blocknote/core";
import mockDocument from "../document.mock";
import { mapTransactionToOperations } from "@/lib/operations/mapping";
import { extractCharIds } from "@/lib/operations/charMapUtils";
import { MapCharID } from "@/types/docs.type";
import { CRDTOpType } from "@/types/operations.type";

let editor: Editor;
let initialDoc: Node;

// Test state variables, reset before each test:
let oldDoc: Node;
let currentNextOpId = 5;
let currentCharIds: MapCharID = {};

beforeAll(() => {
  const blockEditor = BlockNoteEditor.create({
    initialContent: mockDocument,
  });
  editor = blockEditor._tiptapEditor;
  initialDoc = editor.state.doc;
});

beforeEach(() => {
  // Reset the state before each test to ensure no conflicts between tests
  oldDoc = initialDoc;
  currentNextOpId = 5;
  currentCharIds = extractCharIds(mockDocument);
});

describe("mapTransactionToOperations with Tiptap", () => {
  test("inserting a single character maps to insertChar operation", () => {
    // Insert 'A' at position 34
    const tr = editor.state.tr.insertText("A", 34, 34);

    // Check what the transaction doc looks like (not dispatched)
    expect(tr.doc.content.firstChild?.firstChild?.textContent).toBe(
      "Welcome to the BlockNote EditorA"
    );

    const [operations, nextOpId, updatedCharIds] = mapTransactionToOperations(
      tr,
      oldDoc,
      currentNextOpId,
      currentCharIds
    );

    expect(Array.isArray(operations)).toBe(true);
    expect(operations.length).toBeGreaterThan(0);

    const insertOp = operations.find(
      (op: types.CRDTOperation) => op.Type === CRDTOpType.InsertChar
    );
    expect(insertOp).toBeDefined();
    expect(insertOp?.Operation.Character).toBe("A");

    // No need to update globals since we want isolation per test
  });

  test("removing a character maps to deleteChar operation", () => {
    // Remove a character (for example, remove the 'W' at position 1)
    // We'll delete from 1 to 2 to remove the first character "W"
    const tr = editor.state.tr.delete(3, 4);

    // Check the transaction's doc content
    expect(tr.doc.content.firstChild?.firstChild?.textContent).toBe(
      "elcome to the BlockNote Editor"
    );

    const [operations] = mapTransactionToOperations(
      tr,
      oldDoc,
      currentNextOpId,
      currentCharIds
    );

    const deleteOp = operations.find(
      (op: types.CRDTOperation) => op.Type === CRDTOpType.DeleteChar
    );
    expect(deleteOp).toBeDefined();
    expect(deleteOp?.Operation.RemovedID).toBe("2@temp");
  });

  test("adding a new block maps to addBlock operation", () => {
    const endPos = editor.state.doc.content.size - 1;
    const newBlock = editor.state.schema.nodes.paragraph.create(
      {},
      editor.state.schema.text("New block")
    );

    const tr = editor.state.tr.replace(
      endPos,
      endPos,
      newBlock.slice(0, newBlock.content.size)
    );

    const [operations] = mapTransactionToOperations(
      tr,
      oldDoc,
      currentNextOpId,
      currentCharIds
    );

    // Check if the new block was added
    expect(tr.doc.content.firstChild?.childCount).toBeGreaterThan(
      oldDoc.content.firstChild?.childCount ?? 0
    );

    const addBlockOp = operations.find(
      (op: types.CRDTOperation) => op.Type === CRDTOpType.AddBlock
    );
    expect(addBlockOp).toBeDefined();
  });

  test("removing a block maps to removeBlock operation", () => {
    // Remove the last block of the initial doc
    // The initial doc has 4 paragraphs. Let's remove the fourth block by calculating its position.
    const doc = editor.state.doc;
    const lastNodePos =
      doc.content.size -
      (doc.firstChild?.child(1)?.nodeSize ?? doc.content.size);
    const lastNodeEnd =
      lastNodePos +
      (doc.firstChild?.child(1)?.nodeSize ?? doc.content.size) -
      1;

    const tr = editor.state.tr.delete(lastNodePos, lastNodeEnd);

    // Check if the block was removed
    expect(tr.doc.content.size).toBe(
      oldDoc.content.size - (oldDoc.firstChild?.child(1)?.nodeSize ?? 0) + 1
    );

    const [operations] = mapTransactionToOperations(
      tr,
      oldDoc,
      currentNextOpId,
      currentCharIds
    );

    const removeBlockOp = operations.find(
      (op: types.CRDTOperation) => op.Type === CRDTOpType.RemoveBlock
    );
    expect(removeBlockOp).toBeDefined();
  });

  test("adding a mark maps to addMark operation", () => {
    // Add a bold mark to "Welcome" (chars 1 to 7)
    const from = 1;
    const to = from + "Welcome".length;
    const boldMark = editor.state.schema.marks.bold.create();
    const tr = editor.state.tr.addMark(from, to, boldMark);

    const [operations] = mapTransactionToOperations(
      tr,
      oldDoc,
      currentNextOpId,
      currentCharIds
    );

    const addMarkOp = operations.find(
      (op: types.CRDTOperation) => op.Type === CRDTOpType.AddMark
    );
    expect(addMarkOp).toBeDefined();
    expect(addMarkOp?.Operation.MarkType).toBe("bold");
  });

  test("removing a mark maps to removeMark operation", () => {
    const from = 1;
    const to = from + "Welcome".length;
    const boldMark = editor.state.schema.marks.bold.create();
    // Add and remove the bold mark to "Welcome" (chars 1 to 7)
    const tr = editor.state.tr
      .addMark(from, to, boldMark)
      .removeMark(from, to, boldMark);

    const [operations] = mapTransactionToOperations(
      tr,
      oldDoc,
      currentNextOpId,
      currentCharIds
    );

    const removeMarkOp = operations.find(
      (op: types.CRDTOperation) => op.Type === CRDTOpType.RemoveMark
    );
    expect(removeMarkOp).toBeDefined();
    expect(removeMarkOp?.Operation.MarkType).toBe("bold");
  });
});
