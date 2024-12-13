package types

// CRDTOperationsMessage describes a message that contains a list of CRDT operations.
//
// - implements types.Message
type CRDTOperationsMessage struct {
	Operations []CRDTOperation
}

// CRDTOp is an interface that defines the methods that a CRDT operation must implement.
type CRDTOp interface {
	NewEmpty() CRDTOp
	Name() string
}

type TextAlignment string
type HeadingLevel int
type BlockTypeName string

type TextStyle struct {
	Bold            bool
	Italic          bool
	Underline       bool
	Strikethrough   bool
	TextColor       string
	BackgroundColor string
}

const (
	ParagraphBlockType    BlockTypeName = "paragraph"
	HeadingBlockType      BlockTypeName = "heading"
	BulletedListBlockType BlockTypeName = "bulleted_list"
	NumberedListBlockType BlockTypeName = "numbered_list"
	ImageBlockType        BlockTypeName = "image"
	TableBlockType        BlockTypeName = "table"
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

// BlockType is an interface that defines operations on blocks.
type BlockType interface {
	NewEmpty() BlockType
	Name() string
}

// InlineContent is an interface that defines operations on inline content.
type InlineContent interface {
	NewEmpty() InlineContent
	Name() string
}

// TableContent is a struct that defines the content of a table.
type TableContent struct{}

// -------------------------------------------------------------------
// Data Structures

// StyledText implements InlineContent.
type StyledText struct {
	InlineContent
	Text            string
	Styles          TextStyle
	Color           string
	BackgroundColor string
}

func (s StyledText) NewEmpty() InlineContent {
	return StyledText{}
}

func (s StyledText) Name() string {
	return "StyledText"
}

// Link implements InlineContent.
type Link struct {
	InlineContent
	Content []StyledText
	Href    string
}

func (l Link) NewEmpty() InlineContent {
	return Link{}
}

func (l Link) Name() string {
	return "Link"
}

// ParagraphBlock implements BlockType.
type ParagraphBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Content  []InlineContent
	Children []BlockType
}

func (b ParagraphBlock) NewEmpty() BlockType {
	return ParagraphBlock{}
}

func (b ParagraphBlock) Name() string {
	return "ParagraphBlock"
}

// HeadingBlock implements BlockType.
type HeadingBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Level    HeadingLevel
	Content  []InlineContent
	Children []BlockType
}

func (b HeadingBlock) NewEmpty() BlockType {
	return HeadingBlock{}
}

func (b HeadingBlock) Name() string {
	return "HeadingBlock"
}

// BulletedListBlock implements BlockType.
type BulletedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Content  []InlineContent
	Children []BlockType
}

func (b BulletedListBlock) NewEmpty() BlockType {
	return BulletedListBlock{}
}

func (b BulletedListBlock) Name() string {
	return "BulletedListBlock"
}

// NumberedListBlock implements BlockType.
type NumberedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	Id       string
	Content  []InlineContent
	Children []BlockType
}

func (b NumberedListBlock) NewEmpty() BlockType {
	return NumberedListBlock{}
}

func (b NumberedListBlock) Name() string {
	return "NumberedListBlock"
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
	Type      string
	BlockType string

	Origin      string
	OperationId uint64 // Starts from 1
	DocumentId  string // OperationId@Origin that creates the document
	BlockId     string // OperationId@Origin that creates the block
	Operation   CRDTOp
}

type CRDTAddBlock struct {
	CRDTOp
	AfterBlock  string
	ParentBlock string
	BlockType   string
	Props       BlockType
}

func (op CRDTAddBlock) NewEmpty() CRDTOp {
	return CRDTAddBlock{}
}

func (op CRDTAddBlock) Name() string {
	return "CRDTAddBlock"
}

// CRDTRemoveBlock implements CRDTOp.
type CRDTRemoveBlock struct {
	CRDTOp
	RemovedBlock string
}

func (op CRDTRemoveBlock) NewEmpty() CRDTOp {
	return CRDTRemoveBlock{}
}

func (op CRDTRemoveBlock) Name() string {
	return "CRDTRemoveBlock"
}

// CRDTUpdateBlock implements CRDTOp.
type CRDTUpdateBlock struct {
	CRDTOp
	UpdatedBlock string
	AfterBlock   string
	ParentBlock  string
	BlockType    string
	Props        BlockType
}

func (op CRDTUpdateBlock) NewEmpty() CRDTOp {
	return CRDTUpdateBlock{}
}

func (op CRDTUpdateBlock) Name() string {
	return "CRDTUpdateBlock"
}

// CRDTInsertChar implements CRDTOp.
type CRDTInsertChar struct {
	CRDTOp
	AfterID   string
	Character string
}

func (op CRDTInsertChar) NewEmpty() CRDTOp {
	return CRDTInsertChar{}
}

func (op CRDTInsertChar) Name() string {
	return "CRDTInsertChar"
}

// CRDTDeleteChar implements CRDTOp.
type CRDTDeleteChar struct {
	CRDTOp
	RemovedID string
}

func (op CRDTDeleteChar) NewEmpty() CRDTOp {
	return CRDTDeleteChar{}
}

func (op CRDTDeleteChar) Name() string {
	return "CRDTDeleteChar"
}

// CRDTAddMark implements CRDTOp.
type CRDTAddMark struct {
	CRDTOp
	Start    struct{}
	End      struct{}
	MarkType TextStyle
	Options  struct{}
}

func (op CRDTAddMark) NewEmpty() CRDTOp {
	return CRDTAddMark{}
}

func (op CRDTAddMark) Name() string {
	return "CRDTAddMark"
}

// CRDTRemoveMark implements CRDTOp.
type CRDTRemoveMark struct {
	CRDTOp
	Start    struct{}
	End      struct{}
	MarkType TextStyle
}

func (op CRDTRemoveMark) NewEmpty() CRDTOp {
	return CRDTRemoveMark{}
}

func (op CRDTRemoveMark) Name() string {
	return "CRDTRemoveMark"
}
