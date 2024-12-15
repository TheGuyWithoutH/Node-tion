package types

import (
	"fmt"
	"strconv"
	"strings"
)

// -----------------------------------------------------------------------------
// CRDTOperationsMessage

// NewEmpty implements types.Message.
func (c CRDTOperationsMessage) NewEmpty() Message {
	return &CRDTOperationsMessage{}
}

// Name implements types.Message.
func (c CRDTOperationsMessage) Name() string {
	return "crdtoperations"
}

// String implements types.Message.
func (c CRDTOperationsMessage) String() string {
	return fmt.Sprintf("crdtoperations{%d operations}", len(c.Operations))
}

// HTML implements types.Message.
func (c CRDTOperationsMessage) HTML() string { return c.String() }

//AddBlock

func (op *CRDTAddBlock) NewEmpty() CRDTOp {
	return &CRDTAddBlock{}
}

func (op *CRDTAddBlock) Name() string {
	return "CRDTAddBlock"
}

//RemoveBlock

func (op *CRDTRemoveBlock) NewEmpty() CRDTOp {
	return &CRDTRemoveBlock{}
}

func (op *CRDTRemoveBlock) Name() string {
	return "CRDTRemoveBlock"
}

//UpdateBlock

func (op *CRDTUpdateBlock) NewEmpty() CRDTOp {
	return &CRDTUpdateBlock{}
}

func (op *CRDTUpdateBlock) Name() string {
	return "CRDTUpdateBlock"
}

//InsertChar

func (op *CRDTInsertChar) NewEmpty() CRDTOp {
	return &CRDTInsertChar{}
}

func (op *CRDTInsertChar) Name() string {
	return "CRDTInsertChar"
}

//DeleteChar

func (op *CRDTDeleteChar) NewEmpty() CRDTOp {
	return CRDTDeleteChar{}
}

func (op *CRDTDeleteChar) Name() string {
	return "CRDTDeleteChar"
}

//AddMark

func (op *CRDTAddMark) NewEmpty() CRDTOp {
	return &CRDTAddMark{}
}

func (op *CRDTAddMark) Name() string {
	return "CRDTAddMark"
}

//RemoveMark

func (op *CRDTRemoveMark) NewEmpty() CRDTOp {
	return &CRDTRemoveMark{}
}

func (op *CRDTRemoveMark) Name() string {
	return "CRDTRemoveMark"
}

// ---------------------Data Strutures Functions------------------------
// TextStyle

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

// StyledText

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

// Link

func (l *Link) NewEmpty() InlineContent {
	return &Link{}
}

func (l *Link) Name() string {
	return "Link"
}

func (l *Link) ToJson() string {
	return ""
}

// ParagraphBlock

func (b *ParagraphBlock) NewEmpty() BlockType {
	return &ParagraphBlock{}
}

func (b *ParagraphBlock) Name() string {
	return "ParagraphBlock"
}

func (b *ParagraphBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)
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

// HeadingBlock

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
	b.Content = addContentToBlock(content, style)
}

func (b *HeadingBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

// BulletedListBlock

func (b *BulletedListBlock) NewEmpty() BlockType {
	return &BulletedListBlock{}
}

func (b *BulletedListBlock) Name() string {
	return "BulletedListBlock"
}

// NumberedListBlock implements BlockType.

func (b *BulletedListBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)

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

// NumberedListBlock

func (b *NumberedListBlock) NewEmpty() BlockType {
	return &NumberedListBlock{}
}

func (b *NumberedListBlock) Name() string {
	return "NumberedListBlock"
}

func (b *NumberedListBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)
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

// ImageBlock

func (b *ImageBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	// Do nothing for now, as images do not have content characters
}

func (b *ImageBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

func (b *ImageBlock) ToJson() string {
	return ""
}

// TableBock

func (b *TableBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	// Do nothing for now
}

func (b *TableBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)

}

func (b *TableBlock) ToJson() string {
	return ""
}

// Utils

func compareTextStyle(a TextStyle, b TextStyle) bool {
	if a.Bold != b.Bold || a.Italic != b.Italic || a.Underline != b.Underline ||
		a.Strikethrough != b.Strikethrough || a.TextColor != b.TextColor ||
		a.BackgroundColor != b.BackgroundColor {
		return false
	}

	return true
}

func addContentToBlock(content []CRDTInsertChar, style map[string]TextStyle) []InlineContent {
	// Create one InlineContent for characters with the same style
	var styledTexts []StyledText
	// If the style is the same, we can group the characters together
	var previousStyles TextStyle
	var stringContent string
	var charIds []string

	for _, char := range content {
		if !compareTextStyle(style[char.OpID], previousStyles) {
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
