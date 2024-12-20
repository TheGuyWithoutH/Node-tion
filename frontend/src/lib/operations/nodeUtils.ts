import {
  ReplaceStep,
  ReplaceAroundStep,
  Transform,
} from "prosemirror-transform";
import { Node, ResolvedPos } from "prosemirror-model";
import { BlockTypeName } from "@/types/operations.type";

/**
 * Find the block node at the given position.
 * @param $pos - The resolved position in the document.
 * @returns The block node at the given position, if any.
 */
function findBlockNode($pos: ResolvedPos) {
  const { node } = findBlockNodeAndDepth($pos) || { node: null };
  return node;
}

/**
 * Find the block node and its depth at the given position.
 * @param $pos - The resolved position in the document.
 * @returns The block node and its depth at the given position, if any.
 */
function findBlockNodeAndDepth($pos: ResolvedPos) {
  for (let depth = $pos.depth; depth >= 0; depth--) {
    const node = $pos.node(depth);
    if (node && node.isBlock && node.attrs && node.attrs.id) {
      return { node, depth };
    }
  }
  return null;
}

/**
 * Get the block ID of the parent block at the given position.
 * @param tr - The transaction to get the document from.
 * @param pos - The position in the document.
 * @returns The block ID of the parent block at the given position, if any.
 */
function getParentBlockID(tr: Transform, pos: number) {
  const $pos = tr.doc.resolve(pos);

  // Start from one level up the tree
  for (let depth = $pos.depth; depth >= 0; depth--) {
    const node = $pos.node(depth);
    if (node && node.isBlock && node.attrs && node.attrs.id) {
      return node.attrs.id;
    }
  }

  // If no parent blockId found
  return null;
}

/**
 * Get the block ID of the block after the given position.
 * @param tr - The transaction to get the document from.
 * @param pos - The position in the document.
 * @returns The block ID of the block after the given position, if any.
 */
function getAfterBlockID(tr: Transform, pos: number) {
  const $pos = tr.doc.resolve(pos);
  const parent = $pos.parent;
  const index = $pos.index(); // The index where content is inserted in parent

  // If index > 0, there is a preceding sibling
  if (index > 0) {
    const previousNode = parent.child(index - 1);
    if (previousNode.isBlock && previousNode.attrs && previousNode.attrs.id) {
      return previousNode.attrs.id;
    }
  }

  // If no previous block or no valid blockId
  return null;
}

/**
 * Get the block ID of the block inserted by the given step.
 * @param step - The step to get the inserted block ID from.
 * @returns The block ID of the block inserted by the given step, if any.
 */
function getInsertedBlockIDFromStep(step: ReplaceStep | ReplaceAroundStep) {
  if (!step.slice || !step.slice.content) return null;
  let foundId = null;
  step.slice.content.forEach((node: Node) => {
    if (node.isBlock && node.attrs && node.attrs.id) {
      foundId = node.attrs.id;
    } else if (node.isBlock && node.content) {
      node.content.forEach((child) => {
        if (child.isBlock && child.attrs && child.attrs.id) {
          foundId = child.attrs.id;
        }
      });
    }
  });
  return foundId;
}

/**
 * Find the block ID of a nested block in the document.
 * @param tr - The transaction to get the document from.
 * @param pos - The position in the document.
 * @returns The block ID of the nested block in the document, if any.
 */
function findNestedBlockIDInDoc(tr: Transform, pos: number) {
  const $pos = tr.doc.resolve(pos);
  const containerNode = $pos.node($pos.depth);

  let foundId = null;
  containerNode.descendants((node) => {
    if (node.isBlock && node.attrs && node.attrs.id) {
      foundId = node.attrs.id;
      return false; // stop searching
    }
  });
  return foundId;
}

// Find a block node by its id anywhere in the doc
function findBlockNodeById(tr: Transform, blockId: string) {
  let foundNode = null;
  tr.doc.descendants((node) => {
    if (node.isBlock && node.attrs && node.attrs.id === blockId) {
      foundNode = node;
      return false;
    }
  });
  return foundNode;
}

/**
 * Get the props of a block node.
 * @param blockNode - The block node to get the props from.
 * @returns The props of the block node.
 */
function getBlockProps(blockNode: Node) {
  // Return node.attrs filtered or processed as needed.
  let { id, ...props } = blockNode.attrs;

  // Get the type of the block
  const type =
    blockNode.content.firstChild?.type.name || BlockTypeName.Paragraph;
  const blockProps = blockNode.content.firstChild?.attrs || {};

  props = {
    type,
    ...props,
    ...blockProps,
  };

  return props;
}

export {
  findBlockNode,
  findBlockNodeAndDepth,
  getParentBlockID,
  getAfterBlockID,
  getInsertedBlockIDFromStep,
  findNestedBlockIDInDoc,
  findBlockNodeById,
  getBlockProps,
};
