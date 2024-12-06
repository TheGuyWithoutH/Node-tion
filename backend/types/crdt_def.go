package types

// CRDTOperationsMessage describes a message that contains a list of CRDT operations.
//
// - implements types.Message
type CRDTOperationsMessage struct {
	Operations []CRDTOperation
}

type CRDTOp interface{}

type TextAlignment string
type HeadingLevel int
type TextStyle string

const (
	Bold          TextStyle = "bold"
	Italic        TextStyle = "italic"
	Underline     TextStyle = "underline"
	Strikethrough TextStyle = "strikethrough"
)

const (
	H1 HeadingLevel = 1
	H2 HeadingLevel = 2
	H3 HeadingLevel = 3
	H4 HeadingLevel = 4
)

const (
	Left    TextAlignment = "left"
	Center  TextAlignment = "center"
	Right   TextAlignment = "right"
	Justify TextAlignment = "justify"
)

type BlockType interface{}
type InlineContent interface{}
type TableContent struct{}

// -------------------------------------------------------------------
// Data Structures

type StyledText struct {
	InlineContent
	Text            string
	Styles          []TextStyle
	Color           string
	BackgroundColor string
}

type Link struct {
	InlineContent
	Content []StyledText
	Href    string
}

type ParagraphBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Content  []InlineContent
	Children []BlockType
}
type HeadingBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Level    HeadingLevel
	Content  []InlineContent
	Children []BlockType
}
type BulletedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Content  []InlineContent
	Children []BlockType
}
type NumberedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Content  []InlineContent
	Children []BlockType
}
type ImageBlock struct {
	Default      DefaultBlockProps
	Id           string
	URL          string
	Caption      string
	PreviewWidth uint
	Children     []BlockType
}
type TableBlock struct {
	Default  DefaultBlockProps
	Id       string
	Content  TableContent
	Children []BlockType
}

type DefaultBlockProps struct {
	BackgroundColor string
	TextColor       string
	TextAlignment   TextAlignment
}

// -------------------------------------------------------------------
// CRDT Operation Types

type CRDTOperation struct {
	Origin      string
	OperationId uint64 // TODO ask what it is
	DocumentId  string
	BlockId     string
	Operation   CRDTOp
}

type CRDTAddBlock[T BlockType] struct {
	CRDTOp
	OpID        string
	AfterBlock  string
	ParentBlock string
	Props       T
}

type CRDTRemoveBlock struct {
	CRDTOp
	OpID         string
	RemovedBlock string
}

type CRDTUpdateBlock[T BlockType] struct {
	CRDTOp
	OpID         string
	UpdatedBlock string
	AfterBlock   string
	ParentBlock  string
	Props        T
}

type CRDTInsertChar struct {
	CRDTOp
	OpID      string
	AfterID   string
	Character string
}

type CRDTDeleteChar struct {
	CRDTOp
	OpID      string
	RemovedID string
}

type CRDTAddMark struct {
	CRDTOp
	OpID     string
	Start    struct{}
	End      struct{}
	MarkType TextStyle
	Options  struct{}
}

type CRDTRemoveMark struct {
	CRDTOp
	OpID     string
	Start    struct{}
	End      struct{}
	MarkType TextStyle
}
