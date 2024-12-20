package types

// CRDTOp is an interface that defines the methods that a CRDT operation must implement.
type CRDTOp interface{}

type TextAlignment string
type HeadingLevel int

type MarkStart struct {
	Type string
	OpID string
}

type MarkEnd struct {
	Type string
	OpID string
}

type TextStyle struct {
	Bold            bool
	Italic          bool
	Underline       bool
	Strikethrough   bool
	TextColor       string
	BackgroundColor string
}

type BlockTypeName string

const (
	ParagraphBlockType    BlockTypeName = "paragraph"
	HeadingBlockType      BlockTypeName = "heading"
	BulletedListBlockType BlockTypeName = "bulletListItem"
	NumberedListBlockType BlockTypeName = "numberedListItem"
	ImageBlockType        BlockTypeName = "image"
	TableBlockType        BlockTypeName = "table"
)

const ( // CRDTOp Operation Types
	CRDTAddBlockType    = "addBlock"
	CRDTRemoveBlockType = "removeBlock"
	CRDTUpdateBlockType = "updateBlock"
	CRDTInsertCharType  = "insert"
	CRDTDeleteCharType  = "delete"
	CRDTAddMarkType     = "addMark"
	CRDTRemoveMarkType  = "removeMark"
)

const ( // Mark Types
	Bold            = "bold"
	Italic          = "italic"
	Underline       = "underline"
	Strikethrough   = "strikethrough"
	TextColor       = "textColor"
	BackgroundColor = "backgroundColor"
)

const StyledTextType = "styledText"

const LinkType = "link"

const ( // Heading Levels
	H1 HeadingLevel = 1
	H2 HeadingLevel = 2
	H3 HeadingLevel = 3
	H4 HeadingLevel = 4
)

const ( // Text Alignments
	Left    TextAlignment = "left"
	Center  TextAlignment = "center"
	Right   TextAlignment = "right"
	Justify TextAlignment = "justify"
)

// BlockType is an interface that defines operations on blocks.
type BlockType interface{}

type BlockFactory struct {
	BlockType BlockTypeName
	Props   DefaultBlockProps
	ID        string
	Deleted	  bool
	Children  []BlockFactory
}


// InlineContent is an interface that defines operations on inline content.
type InlineContent interface{}

// TableContent is a struct that defines the content of a table.
type TableContent struct{}

// -------------------------------------------------------------------
// Data Structures

// ----------------------InlineContent------------------------

// StyledText implements InlineContent.
type StyledText struct {
	InlineContent
	CharIDs []string
	Text    string
	Styles  TextStyle
}

// Link implements InlineContent.
type Link struct {
	InlineContent
	Content []StyledText
	Href    string
}

// ----------------------Blocks------------------------

// ParagraphBlock implements BlockType.
type ParagraphBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  []InlineContent
	Children []BlockType
}

// HeadingBlock implements BlockType.
type HeadingBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Level    HeadingLevel
	Content  []InlineContent
	Children []BlockType
}

// BulletedListBlock implements BlockType.
type BulletedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  []InlineContent
	Children []BlockType
}

type NumberedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  []InlineContent
	Children []BlockType
}

type ImageBlock struct {
	BlockType
	Default      DefaultBlockProps
	ID           string
	URL          string
	Caption      string
	PreviewWidth uint
	Children     []BlockType
}

type TableBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  TableContent
	Children []BlockType
}

type DefaultBlockProps struct {
	BackgroundColor string
	TextColor       string
	TextAlignment   TextAlignment
	Level           HeadingLevel
}

// -------------------------------------------------------------------
// CRDT Operation Types

type CRDTOperation struct {
	Type        string
	Origin      string
	OperationID uint64 // Starts from 1
	DocumentID  string // OperationID@Origin that creates the document
	BlockID     string // OperationID@Origin that creates the block
	Operation   CRDTOp
}

type CRDTAddBlock struct {
	CRDTOp
	OpID        string
	AfterBlock  string
	ParentBlock string
	BlockType   BlockTypeName
	Props       DefaultBlockProps
}

// CRDTRemoveBlock implements CRDTOp.
type CRDTRemoveBlock struct {
	CRDTOp
	OpID         string
	RemovedBlock string
}

// CRDTUpdateBlock implements CRDTOp.
type CRDTUpdateBlock struct {
	CRDTOp
	//OpID         string
	UpdatedBlock string
	AfterBlock   string
	ParentBlock  string
	BlockType    BlockTypeName
	Props        DefaultBlockProps
}

// CRDTInsertChar implements CRDTOp.
type CRDTInsertChar struct {
	CRDTOp
	OpID      string
	AfterID   string
	Character string
}

// CRDTDeleteChar implements CRDTOp.
type CRDTDeleteChar struct {
	CRDTOp
	OpID      string
	RemovedID string
}

// CRDTAddMark implements CRDTOp.
type CRDTAddMark struct {
	CRDTOp
	OpID     string
	Start    MarkStart
	End      MarkEnd
	MarkType string
	Options  MarkOptions
}

// CRDTRemoveMark implements CRDTOp.
type CRDTRemoveMark struct {
	CRDTOp
	OpID     string
	Start    MarkStart
	End      MarkEnd
	MarkType string
}

type MarkOptions struct {
	Color string
	Href  string
}
