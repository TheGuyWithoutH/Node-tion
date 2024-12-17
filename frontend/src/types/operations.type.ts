// This file contains all the types of operations that can be performed on the CRDT.
// It is a translation of the enums defined in the backend `/backend/types/crdt_def.go` file.
//
// The other types are already defined in the wails bindings generated from the go code.
// You can find them in the `frontend/wailsjs/go/models.ts` file.

// Definition of types of blocks
enum BlockTypeName {
  Paragraph = "paragraph",
  Heading = "heading",
  BulletedList = "bulleted_list",
  NumberedList = "numbered_list",
  Image = "image",
  Table = "table",
}

// Operations
enum CRDTOpType {
  AddBlock = "addBlock",
  RemoveBlock = "removeBlock",
  UpdateBlock = "updateBlock",
  InsertChar = "insert",
  DeleteChar = "delete",
  AddMark = "addMark",
  RemoveMark = "removeMark",

  // Custom operations for Frontend
  InsertText = "insertText",
}

// Mark types
enum MarkType {
  Bold = "bold",
  Italic = "italic",
  Underline = "underline",
  Strikethrough = "strikethrough",
  TextColor = "textColor",
  BackgroundColor = "backgroundColor",
}

// Styled text type
const StyledTextType = "styledText";

// Link type
const LinkType = "link";

// Heading levels
enum HeadingLevel {
  H1 = 1,
  H2,
  H3,
  H4,
}

// Text Alignment
enum TextAlignment {
  Left = "left",
  Center = "center",
  Right = "right",
  Justify = "justify",
}

// Block properties
interface BlockProps {
  BackgroundColor: string;
  TextColor: string;
  TextAlignment: TextAlignment;
}

// Default block properties
const defaultBlockProps: BlockProps = {
  BackgroundColor: "white",
  TextColor: "black",
  TextAlignment: TextAlignment.Left,
};

export {
  BlockTypeName,
  CRDTOpType,
  MarkType,
  StyledTextType,
  LinkType,
  HeadingLevel,
  TextAlignment,
  defaultBlockProps,
};
