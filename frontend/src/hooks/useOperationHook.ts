import { ReactNode, useEffect, useState } from "react";
import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "prosemirror-state";
import { Node } from "prosemirror-model";
import { mapTransactionToOperations } from "../lib/operations/mapping";
import { CompileDocument, SaveTransactions } from "../../wailsjs/go/impl/node";
import { types } from "../../wailsjs/go/models";
import { PartialBlock } from "@blocknote/core";
import mockDocument from "@/tests/document.mock";
import { extractCharIds } from "@/lib/operations/charMapUtils";
import { EMPTY_DOC, EMPTY_DOC_HISTORY, MapCharID } from "@/types/docs.type";
import { fixAddBlockOperations } from "@/lib/operations/operationMerger";
import { CRDTOpType } from "@/types/operations.type";
import { flushSync } from "react-dom";

const useOperationsHook = (documentId: string) => {
  const [nextTempOpNumber, setNextTempOpNumber] = useState(100);
  const [operationsHistory, setOperationsHistory] = useState<
    types.CRDTOperation[]
  >([]);
  const [document, setDocument] = useState<PartialBlock[]>([]);
  const [charIds, setCharIds] = useState<MapCharID>({});
  const [editorView, setEditorView] = useState<ReactNode>();

  // Helper to find node position by id
  function findNodePosById(doc: Node, id: string) {
    let foundPos = -1;
    doc.descendants((node, pos) => {
      if (node.isBlock && node.attrs && node.attrs.id === id) {
        foundPos = pos;
        return false;
      }
    });
    return foundPos;
  }

  // Create a new extension for ProseMirror
  // This extension will be used to track the operations and save them
  const StepsTracker = Extension.create({
    name: "stepsTracker",

    addProseMirrorPlugins() {
      return [
        new Plugin({
          key: new PluginKey("stepsTracker"),

          state: {
            init() {
              return { addBlockOps: [] } as {
                addBlockOps: types.CRDTOperation[];
              };
            },

            apply(tr, value, oldState) {
              const oldDoc = oldState.doc;
              let addBlockOps: types.CRDTOperation[] = [];

              // Perform the operations in a flushSync to ensure that the state is updated
              flushSync(() => {
                setCharIds((prevCharIds) => {
                  let nextCharIds = { ...prevCharIds } as Record<
                    string,
                    string[]
                  >;

                  // We need to embed the operations mapping in the next op number logic
                  // to ensure that we get the right previous operation number
                  setNextTempOpNumber((prevNextTempOpNumber) => {
                    // Perform the mapping of the transaction to operations
                    const [operations, newNextTempOpId, _nextCharIds] =
                      mapTransactionToOperations(
                        tr,
                        oldDoc,
                        prevNextTempOpNumber,
                        prevCharIds,
                        documentId
                      );

                    console.log("Next char ids", _nextCharIds);

                    if (tr.steps.length && operations.length) {
                      // Add the operations to the history
                      setOperationsHistory((prev) => {
                        let newOperations = operations;
                        let newOperationHistory = [...prev, ...newOperations];

                        // Check last operation in history if it's addBlock
                        if (newOperationHistory.length > 1) {
                          const lastOp =
                            newOperationHistory[newOperationHistory.length - 2];
                          const currentOp =
                            newOperationHistory[newOperationHistory.length - 1];

                          if (
                            lastOp.Type === CRDTOpType.AddBlock &&
                            currentOp.Type === CRDTOpType.UpdateBlock &&
                            (!currentOp.BlockID.endsWith("@temp") ||
                              !lastOp.BlockID.endsWith("@temp"))
                          ) {
                            // Merge logic
                            const mergedOp = new types.CRDTOperation({
                              ...lastOp,
                              BlockID: currentOp.BlockID,
                              BlockType:
                                lastOp.Operation.Props.type ||
                                currentOp.Operation.Props.type,
                              Operation: {
                                ...lastOp.Operation,
                                Props: {
                                  ...lastOp.Operation.Props,
                                  ...currentOp.Operation.Props,
                                },
                                AfterBlock:
                                  currentOp.Operation.AfterBlock ||
                                  lastOp.Operation.AfterBlock,
                                ParentBlock:
                                  currentOp.Operation.ParentBlock ||
                                  lastOp.Operation.ParentBlock,
                              },
                            });

                            // Remove the last two and replace with merged
                            newOperationHistory = newOperationHistory.slice(
                              0,
                              -2
                            );
                            newOperationHistory.push(mergedOp);
                            newOperations = [mergedOp];
                          }
                        }

                        // Extract addBlock operations
                        addBlockOps = newOperations.filter(
                          (op) => op.Type === CRDTOpType.AddBlock
                        );

                        return newOperationHistory;
                      });

                      // Update the charIds
                      nextCharIds = _nextCharIds;
                    }

                    return newNextTempOpId;
                  });

                  return nextCharIds;
                });
              });

              return { addBlockOps };
            },
          },
          appendTransaction(transactions, oldState, newState) {
            const pluginState = this.getState(newState);
            const { addBlockOps } = pluginState;

            if (addBlockOps && addBlockOps.length > 0) {
              let tr = newState.tr;
              let changed = false;
              for (const op of addBlockOps) {
                const oldBlockID = op.BlockID;
                console.log("Add block operation", op);
                const newBlockID = op.OperationID + "@temp";

                const pos = findNodePosById(newState.doc, oldBlockID);
                if (pos >= 0) {
                  const node = newState.doc.nodeAt(pos);
                  if (node) {
                    tr = tr.setNodeMarkup(pos, null, {
                      ...node.attrs,
                      id: newBlockID,
                    });
                    changed = true;
                  }
                }
              }

              if (changed) {
                return tr;
              }
            }
          },
        }),
      ];
    },
  });

  const sendOperations = () => {
    setDocument([]);

    // If there are no operations, only update the document from the server
    if (!operationsHistory.length) {
      // Get the new document from the server and update the local document
      CompileDocument("doc1")
        .then((doc) => {
          console.log(doc);
          setDocument(JSON.parse(doc));
          setCharIds(extractCharIds(JSON.parse(doc)));
        })
        .catch((err) => {
          console.error("Error getting document", err);
          setDocument(mockDocument);
          setCharIds(extractCharIds(mockDocument));
        });
    } else {
      // Fix addBlock operations
      const fixedOps = fixAddBlockOperations(operationsHistory);

      console.log("Sending operations", fixedOps);

      SaveTransactions(
        new types.CRDTOperationsMessage({
          Operations: fixedOps,
        })
      )
        .then((res) => {
          console.log("Operations sent", res);
          setDocument([]);
          setOperationsHistory([]);
          setNextTempOpNumber(1);

          // Get the new document from the server and update the local document
          CompileDocument("doc1")
            .then((doc) => {
              console.log(doc);
              setDocument(JSON.parse(doc));
              setCharIds(extractCharIds(JSON.parse(doc)));
            })
            .catch((err) => {
              console.error("Error getting document", err);
              setDocument(mockDocument);
              setCharIds(extractCharIds(mockDocument));
            });
        })
        .catch((err) => {
          console.error("Error sending operations", err);
          setDocument(mockDocument);
          setCharIds(extractCharIds(mockDocument));
        });
    }
  };

  useEffect(() => {
    try {
      // Get the document from the server
      CompileDocument(documentId)
        .then((doc) => {
          const parsedDoc = JSON.parse(doc);

          // If the document is empty, add a default block and set the next temp op number
          if (parsedDoc.length === 0) {
            setNextTempOpNumber(3);
            setDocument(EMPTY_DOC);
            setOperationsHistory(EMPTY_DOC_HISTORY(documentId));
            return;
          }

          setDocument(parsedDoc);
          setCharIds(extractCharIds(parsedDoc));
        })
        .catch((err) => {
          console.error("Error getting document", err);
          setDocument(mockDocument);
          setCharIds(extractCharIds(mockDocument));
        });
    } catch (err) {
      console.error("Error getting document", err);
      setDocument(mockDocument);
      setCharIds(extractCharIds(mockDocument));
    }
  }, []);

  return [
    StepsTracker,
    sendOperations,
    document,
    setDocument,
    editorView,
    setEditorView,
  ] as const;
};

export default useOperationsHook;
