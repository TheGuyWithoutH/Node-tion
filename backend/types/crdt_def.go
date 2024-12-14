package types

import (
	"strconv"
	"strings"
)

// CRDTOperationsMessage describes a message that contains a list of CRDT operations.
//
// - implements types.Message
type CRDTOperationsMessage struct {
	Operations []CRDTOperation
}

// CRDTOp is an interface that defines the methods that a CRDT operation must implement.
type CRDTOp interface {
	// NewEmpty() CRDTOp
	// Name() string
}

type TextAlignment string
type HeadingLevel int
type BlockTypeName string

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

func (t *TextStyle) ToJson() string {
	// If the style is present, we need to add it to the JSON, if false, we do not need to add it
	json := "{ "
	if t.Bold {
		json += "\"bold\": " + strconv.FormatBool(t.Bold) + ","
	}
	if t.Italic {
		json += "\"italic\": " + strconv.FormatBool(t.Italic) + ","
	}
	if t.Underline {
		json += "\"underline\": " + strconv.FormatBool(t.Underline) + ","
	}
	if t.Strikethrough {
		json += "\"strikethrough\": " + strconv.FormatBool(t.Strikethrough) + ","
	}
	if t.TextColor != "" {
		json += "\"textColor\": \"" + t.TextColor + "\","
	}
	if t.BackgroundColor != "" {
		json += "\"backgroundColor\": \"" + t.BackgroundColor + "\","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "}"
	return json
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

const ( // CRDT Operation Types
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

var defaultBlockProps = DefaultBlockProps{
	BackgroundColor: "white",
	TextColor:       "black",
	TextAlignment:   Left,
}

// BlockType is an interface that defines operations on blocks.
type BlockType interface {
	NewEmpty() BlockType
	Name() string
	AddChildren(children []BlockType)
	AddContent(content []CRDTInsertChar, style map[string]TextStyle)
	ToJson() string
}

// InlineContent is an interface that defines operations on inline content.
type InlineContent interface {
	NewEmpty() InlineContent
	Name() string
	ToJson() string
}

// TableContent is a struct that defines the content of a table.
type TableContent struct{}

// -------------------------------------------------------------------
// Data Structures

// ----------------------InlineContent------------------------

// StyledText implements InlineContent.
type StyledText struct {
	InlineContent
	CharIds []string
	Text    string
	Styles  TextStyle
}

func (s *StyledText) ToJson() string {

	json := "{"
	json += "\"type\": \"" + "text" + "\","
	json += "\"charIds\": ["
	for _, charId := range s.CharIds {
		json += "\"" + charId + "\","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "],"
	json += "\"text\": \"" + s.Text + "\","
	json += "\"styles\": " + s.Styles.ToJson()
	json += "}"
	return json
}

func (s *StyledText) NewEmpty() InlineContent {
	return &StyledText{}
}

func (s *StyledText) Name() string {
	return "StyledText"
}

// Link implements InlineContent.
type Link struct {
	InlineContent
	Content []StyledText
	Href    string
}

func (l *Link) NewEmpty() InlineContent {
	return &Link{}
}

func (l *Link) Name() string {
	return "Link"
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

func (b *ParagraphBlock) NewEmpty() BlockType {
	return &ParagraphBlock{}
}

func (b *ParagraphBlock) Name() string {
	return "ParagraphBlock"
}

func (b *ParagraphBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = AddContentToBlock(content, style)
}

func (b *ParagraphBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

func (b *ParagraphBlock) ToJson() string {
	json := "{"
	json += "\"id\": \"" + b.ID + "\","
	json += "\"type\": \"" + "paragraph" + "\","
	// Props
	json += "\"props\" : {"
	json += "\"textColor\": \"" + b.Default.TextColor + "\","
	json += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	json += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	json += "},"
	// Content
	json += "\"content\": [ "
	for _, content := range b.Content {
		if content != nil {
			json += content.ToJson() + ","
		}
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "],"
	// Children
	json += "\"children\": [ "
	for _, child := range b.Children {
		json += child.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "]}"

	return json
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

func (b *HeadingBlock) NewEmpty() BlockType {
	return &HeadingBlock{}
}

func (b *HeadingBlock) Name() string {
	return "HeadingBlock"
}

func (b *HeadingBlock) ToJson() string {
	json := "{"
	json += "\"id\": \"" + b.ID + "\","
	json += "\"type\": \"" + "heading" + "\","
	// Props
	json += "\"props\" : {"
	json += "\"level\": " + strconv.Itoa(int(b.Level)) + ","
	json += "\"textColor\": \"" + b.Default.TextColor + "\","
	json += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	json += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	json += "},"
	// Content
	json += "\"content\": [ "
	for _, content := range b.Content {
		json += content.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "],"
	// Children
	json += "\"children\": [ "
	for _, child := range b.Children {
		json += child.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "]}"

	return json
}

func (b *HeadingBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = AddContentToBlock(content, style)
}

func (b *HeadingBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

// BulletedListBlock implements BlockType.
type BulletedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  []InlineContent
	Children []BlockType
}

func (b *BulletedListBlock) NewEmpty() BlockType {
	return &BulletedListBlock{}
}

func (b *BulletedListBlock) Name() string {
	return "BulletedListBlock"
}

// NumberedListBlock implements BlockType.

func (b *BulletedListBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = AddContentToBlock(content, style)

}
func (b *BulletedListBlock) ToJson() string {
	json := "{"
	json += "\"id\": \"" + b.ID + "\","
	json += "\"type\": \"" + "bulletListItem" + "\","
	// Props
	json += "\"props\" : {"
	json += "\"textColor\": \"" + b.Default.TextColor + "\","
	json += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	json += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	json += "},"
	// Content
	json += "\"content\": [ "
	for _, content := range b.Content {
		json += content.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "],"
	// Children
	json += "\"children\": [ "
	for _, child := range b.Children {
		json += child.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "]}"

	return json
}

func (b *BulletedListBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

type NumberedListBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  []InlineContent
	Children []BlockType
}

func (b *NumberedListBlock) NewEmpty() BlockType {
	return &NumberedListBlock{}
}

func (b *NumberedListBlock) Name() string {
	return "NumberedListBlock"
}

func (b *NumberedListBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = AddContentToBlock(content, style)
}

func (b *NumberedListBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

func (b *NumberedListBlock) ToJson() string {
	json := "{"
	json += "\"id\": \"" + b.ID + "\","
	json += "\"type\": \"" + "numberedListItem" + "\","
	// Props
	json += "\"props\" : {"
	json += "\"textColor\": \"" + b.Default.TextColor + "\","
	json += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	json += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	json += "},"
	// Content
	json += "\"content\": [ "
	for _, content := range b.Content {
		json += content.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "],"
	// Children
	json += "\"children\": [ "
	for _, child := range b.Children {
		json += child.ToJson() + ","
	}
	json = json[:len(json)-1] // Remove the additional ","
	json += "]}"

	return json
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

func (b *ImageBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	// Do nothing for now, as images do not have content characters
}

func (b *ImageBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

func (b *ImageBlock) ToJson() string {
	return ""
}

type TableBlock struct {
	BlockType
	Default  DefaultBlockProps
	ID       string
	Content  TableContent
	Children []BlockType
}

func (b *TableBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	// Do nothing for now
}

func (b *TableBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)

}

func (b *TableBlock) ToJson() string {
	return ""
}

type DefaultBlockProps struct {
	BackgroundColor string
	TextColor       string
	TextAlignment   TextAlignment
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
	BlockType   string
	Props       BlockType
}

func (op *CRDTAddBlock) NewEmpty() CRDTOp {
	return &CRDTAddBlock{}
}

func (op *CRDTAddBlock) Name() string {
	return "CRDTAddBlock"
}

// CRDTRemoveBlock implements CRDTOp.
type CRDTRemoveBlock struct {
	CRDTOp
	OpID         string
	RemovedBlock string
}

func (op *CRDTRemoveBlock) NewEmpty() CRDTOp {
	return &CRDTRemoveBlock{}
}

func (op *CRDTRemoveBlock) Name() string {
	return "CRDTRemoveBlock"
}

// CRDTUpdateBlock implements CRDTOp.
type CRDTUpdateBlock struct {
	CRDTOp
	OpID         string
	UpdatedBlock string
	AfterBlock   string
	ParentBlock  string
	BlockType    string
	Props        BlockType
}

func (op *CRDTUpdateBlock) NewEmpty() CRDTOp {
	return &CRDTUpdateBlock{}
}

func (op CRDTUpdateBlock) Name() string {
	return "CRDTUpdateBlock"
}

// CRDTInsertChar implements CRDTOp.
type CRDTInsertChar struct {
	CRDTOp
	OpID      string
	AfterID   string
	Character string
}

func (op *CRDTInsertChar) NewEmpty() CRDTOp {
	return &CRDTInsertChar{}
}

func (op *CRDTInsertChar) Name() string {
	return "CRDTInsertChar"
}

// CRDTDeleteChar implements CRDTOp.
type CRDTDeleteChar struct {
	CRDTOp
	OpID      string
	RemovedID string
}

func (op CRDTDeleteChar) NewEmpty() CRDTOp {
	return CRDTDeleteChar{}
}

func (op CRDTDeleteChar) Name() string {
	return "CRDTDeleteChar"
}

type MarkOptions struct {
	Color string
	Href  string
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

func (op *CRDTAddMark) NewEmpty() CRDTOp {
	return &CRDTAddMark{}
}

func (op *CRDTAddMark) Name() string {
	return "CRDTAddMark"
}

// CRDTRemoveMark implements CRDTOp.
type CRDTRemoveMark struct {
	CRDTOp
	OpID     string
	Start    MarkStart
	End      MarkEnd
	MarkType string
}

func (op *CRDTRemoveMark) NewEmpty() CRDTOp {
	return &CRDTRemoveMark{}
}

func (op *CRDTRemoveMark) Name() string {
	return "CRDTRemoveMark"
}

func CompareTextStyle(a TextStyle, b TextStyle) bool {
	if a.Bold != b.Bold || a.Italic != b.Italic || a.Underline != b.Underline ||
		a.Strikethrough != b.Strikethrough || a.TextColor != b.TextColor ||
		a.BackgroundColor != b.BackgroundColor {
		return false
	}

	return true
}

func AddContentToBlock(content []CRDTInsertChar, style map[string]TextStyle) []InlineContent {
	// Create one InlineContent for characters with the same style
	var styledTexts []StyledText
	// If the style is the same, we can group the characters together
	var previousStyles TextStyle
	var stringContent string
	var charIds []string

	for _, char := range content {
		if !CompareTextStyle(style[char.OpID], previousStyles) {
			// If the style is different, we need to create a new InlineContent
			if len(stringContent) > 0 {
				styledTexts = append(styledTexts, StyledText{
					CharIds: charIds,
					Text:    strings.Clone(stringContent),
					Styles:  previousStyles,
				})
				// Reset the stringContent
				stringContent = ""
			}
		}
		stringContent += string(char.Character)
		charIds = append(charIds, char.OpID)
		previousStyles = style[char.OpID]
	}

	// We need to add the last block of text
	if len(stringContent) > 0 {
		styledTexts = append(styledTexts, StyledText{
			CharIds: charIds,
			Text:    strings.Clone(stringContent),
			Styles:  previousStyles,
		})
	}

	var inlineContents []InlineContent = make([]InlineContent, len(styledTexts))
	for i, styledText := range styledTexts {
		inlineContents[i] = &styledText
	}

	return inlineContents
}
