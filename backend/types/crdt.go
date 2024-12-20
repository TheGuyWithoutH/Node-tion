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

// ---------------------Data Strutures Functions------------------------
// TextStyle

func (t *TextStyle) ToJSON() string {
	// If the style is present, we need to add it to the JSON, if false, we do not need to add it
	JSON := "{ "
	if t.Bold {
		JSON += "\"bold\": " + strconv.FormatBool(t.Bold) + ","
	}
	if t.Italic {
		JSON += "\"italic\": " + strconv.FormatBool(t.Italic) + ","
	}
	if t.Underline {
		JSON += "\"underline\": " + strconv.FormatBool(t.Underline) + ","
	}
	if t.Strikethrough {
		JSON += "\"strikethrough\": " + strconv.FormatBool(t.Strikethrough) + ","
	}
	if t.TextColor != "" {
		JSON += "\"textColor\": \"" + t.TextColor + "\","
	}
	if t.BackgroundColor != "" {
		JSON += "\"backgroundColor\": \"" + t.BackgroundColor + "\","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "}"
	return JSON
}

// StyledText

func (s *StyledText) ToJSON() string {

	JSON := "{"
	JSON += "\"type\": \"" + "text" + "\","
	JSON += "\"charIds\": ["
	for _, charID := range s.CharIDs {
		JSON += "\"" + charID + "\","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "],"
	JSON += "\"text\": \"" + s.Text + "\","
	JSON += "\"styles\": " + s.Styles.ToJSON()
	JSON += "}"
	return JSON
}

// Link

func (l *Link) ToJSON() string {
	return ""
}

// ParagraphBlock

func (b *ParagraphBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)
}

func (b *ParagraphBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

const (
	CHILDREN = "\"children\": [ "
	CONTENT  = "\"content\": [ "
	PROPS    = "\"props\" : {"
)

func (b *ParagraphBlock) ToJSON() string {

	JSON := "{"
	JSON += "\"id\": \"" + b.ID + "\","
	JSON += "\"type\": \"" + "paragraph" + "\","
	// Props
	JSON += PROPS
	JSON += "\"textColor\": \"" + b.Default.TextColor + "\","
	JSON += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	JSON += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	JSON += "},"
	// Content
	JSON += CONTENT
	for _, content := range b.Content {
		if content != nil {
			JSON += SerializeInlineContent(content) + ","
		}
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "],"
	// Children
	JSON += CHILDREN
	for _, child := range b.Children {
		JSON += SerializeBlock(child) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "]}"

	return JSON
}

// HeadingBlock

func (b *HeadingBlock) ToJSON() string {
	JSON := "{"
	JSON += "\"id\": \"" + b.ID + "\","
	JSON += "\"type\": \"" + "heading" + "\","
	// Props
	JSON += PROPS
	JSON += "\"level\": " + strconv.Itoa(int(b.Level)) + ","
	JSON += "\"textColor\": \"" + b.Default.TextColor + "\","
	JSON += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	JSON += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	JSON += "},"
	// Content
	JSON += CONTENT
	for _, content := range b.Content {
		JSON += SerializeInlineContent(content) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "],"
	// Children
	JSON += CHILDREN
	for _, child := range b.Children {
		JSON += SerializeBlock(child) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "]}"

	return JSON
}

func (b *HeadingBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)
}

func (b *HeadingBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

// BulletedListBlock

func (b *BulletedListBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)

}
func (b *BulletedListBlock) ToJSON() string {
	JSON := "{"
	JSON += "\"id\": \"" + b.ID + "\","
	JSON += "\"type\": \"" + "bulletListItem" + "\","
	// Props
	JSON += "\"props\" : {"
	JSON += "\"textColor\": \"" + b.Default.TextColor + "\","
	JSON += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	JSON += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	JSON += "},"
	// Content
	JSON += "\"content\": [ "
	for _, content := range b.Content {
		JSON += SerializeInlineContent(content) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "],"
	// Children
	JSON += "\"children\": [ "
	for _, child := range b.Children {
		JSON += SerializeBlock(child) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "]}"

	return JSON
}

func (b *BulletedListBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

// NumberedListBlock

func (b *NumberedListBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	b.Content = addContentToBlock(content, style)
}

func (b *NumberedListBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

func (b *NumberedListBlock) ToJSON() string {
	JSON := "{"
	JSON += "\"id\": \"" + b.ID + "\","
	JSON += "\"type\": \"" + "numberedListItem" + "\","
	// Props
	JSON += "\"props\" : {"
	JSON += "\"textColor\": \"" + b.Default.TextColor + "\","
	JSON += "\"backgroundColor\": \"" + b.Default.BackgroundColor + "\","
	JSON += "\"textAlignment\": \"" + string(b.Default.TextAlignment) + "\""
	JSON += "},"
	// Content
	JSON += "\"content\": [ "
	for _, content := range b.Content {
		JSON += SerializeInlineContent(content) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "],"
	// Children
	JSON += "\"children\": [ "
	for _, child := range b.Children {
		JSON += SerializeBlock(child) + ","
	}
	JSON = JSON[:len(JSON)-1] // Remove the additional ","
	JSON += "]}"

	return JSON
}

// ImageBlock

func (b *ImageBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	// Do nothing for now, as images do not have content characters
}

func (b *ImageBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)
}

func (b *ImageBlock) ToJSON() string {
	return ""
}

// TableBock

func (b *TableBlock) AddContent(content []CRDTInsertChar, style map[string]TextStyle) {
	// Do nothing for now
}

func (b *TableBlock) AddChildren(children []BlockType) {
	b.Children = append(b.Children, children...)

}

func (b *TableBlock) ToJSON() string {
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
	var charIDs []string

	for _, char := range content {
		if !compareTextStyle(style[char.OpID], previousStyles) {
			// If the style is different, we need to create a new InlineContent
			if len(stringContent) > 0 {
				styledTexts = append(styledTexts, StyledText{
					CharIDs: charIDs,
					Text:    strings.Clone(stringContent),
					Styles:  previousStyles,
				})
				// Reset the stringContent
				stringContent = ""
				charIDs = nil
			}
		}
		stringContent += string(char.Character)
		charIDs = append(charIDs, char.OpID)
		previousStyles = style[char.OpID]
	}

	// We need to add the last block of text
	if len(stringContent) > 0 {
		styledTexts = append(styledTexts, StyledText{
			CharIDs: charIDs,
			Text:    strings.Clone(stringContent),
			Styles:  previousStyles,
		})
	}

	var inlineContents = make([]InlineContent, len(styledTexts))
	for i, styledText := range styledTexts {
		styledTextCopy := styledText
		inlineContents[i] = &styledTextCopy
	}

	return inlineContents
}

func SerializeBlock(block BlockType) string {
	switch b := block.(type) {
	case *ParagraphBlock:
		return b.ToJSON()
	case *HeadingBlock:
		return b.ToJSON()
	case *BulletedListBlock:
		return b.ToJSON()
	case *NumberedListBlock:
		return b.ToJSON()
	case *ImageBlock:
		return b.ToJSON()
	case *TableBlock:
		return b.ToJSON()
	default:
		return "{}" // Fallback for unknown types
	}
}

func AddContent(block BlockType, content []CRDTInsertChar, style map[string]TextStyle) {
	switch b := block.(type) {
	case *ParagraphBlock:
		b.AddContent(content, style)
	case *HeadingBlock:
		b.AddContent(content, style)
	case *BulletedListBlock:
		b.AddContent(content, style)
	case *NumberedListBlock:
		b.AddContent(content, style)
	case *ImageBlock:
		b.AddContent(content, style)
	case *TableBlock:
		b.AddContent(content, style)
	}
}

func AddChildren(block BlockType, children []BlockType) {
	switch b := block.(type) {
	case *ParagraphBlock:
		b.AddChildren(children)
	case *HeadingBlock:
		b.AddChildren(children)
	case *BulletedListBlock:
		b.AddChildren(children)
	case *NumberedListBlock:
		b.AddChildren(children)
	case *ImageBlock:
		b.AddChildren(children)
	case *TableBlock:
		b.AddChildren(children)
	}
}

func SerializeInlineContent(content InlineContent) string {
	switch c := content.(type) {
	case *StyledText:
		return c.ToJSON()
	case *Link:
		return c.ToJSON()
	default:
		return "{}" // Fallback for unknown types
	}
}
