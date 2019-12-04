package gold

import (
	"bytes"
	"fmt"
)

type StyleType int

const (
	Document StyleType = iota
	BlockQuote
	List
	Item
	Paragraph
	Heading
	HorizontalRule
	Emph
	Strong
	Del
	Link
	LinkText
	Image
	ImageText
	Text
	HTMLBlock
	CodeBlock
	Softbreak
	Hardbreak
	Indent
	Code
	HTMLSpan
	Table
	TableCell
	TableHead
	TableBody
	TableRow
)

type ElementStyle struct {
	Color           string `json:"color"`
	BackgroundColor string `json:"background_color"`
	Underline       bool   `json:"underline"`
	Bold            bool   `json:"bold"`
	Italic          bool   `json:"italic"`
	CrossedOut      bool   `json:"crossed_out"`
	Faint           bool   `json:"faint"`
	Conceal         bool   `json:"conceal"`
	Overlined       bool   `json:"overlined"`
	Inverse         bool   `json:"inverse"`
	Blink           bool   `json:"blink"`
	Indent          uint   `json:"indent"`
	Margin          uint   `json:"margin"`
	Theme           string `json:"theme"`
	Prefix          string `json:"prefix"`
	Suffix          string `json:"suffix"`
}

type BlockElement struct {
	Block *bytes.Buffer
	Style *ElementStyle
}

type BlockStack []BlockElement

func (s *BlockStack) Push(e BlockElement) {
	*s = append(*s, e)
}

func (s *BlockStack) Pop() {
	stack := *s
	if len(stack) == 0 {
		return
	}

	stack = stack[0 : len(stack)-1]
	*s = stack
}

func (s BlockStack) Indent() uint {
	var i uint

	for _, v := range s {
		if v.Style == nil {
			continue
		}
		i += v.Style.Indent
	}

	return i
}

func (s BlockStack) Margin() uint {
	var i uint

	for _, v := range s {
		if v.Style == nil {
			continue
		}
		i += v.Style.Margin
	}

	return i
}

func (s BlockStack) Parent() BlockElement {
	if len(s) < 2 {
		return s.Current()
	}

	return s[len(s)-2]
}

func (s BlockStack) Current() BlockElement {
	if len(s) == 0 {
		return BlockElement{
			Block: &bytes.Buffer{},
		}
	}

	return s[len(s)-1]
}

func (s BlockStack) With(child *ElementStyle) *ElementStyle {
	return cascadeStyle(s.Current().Style, child)
}

func cascadeStyle(parent *ElementStyle, child *ElementStyle) *ElementStyle {
	if parent == nil {
		return child
	}

	s := ElementStyle{}
	if child != nil {
		s = *child
	}

	s.Color = parent.Color
	s.BackgroundColor = parent.BackgroundColor

	if child != nil {
		if child.Color != "" {
			s.Color = child.Color
		}
		if child.BackgroundColor != "" {
			s.BackgroundColor = child.BackgroundColor
		}
	}

	return &s
}

func keyToType(key string) (StyleType, error) {
	switch key {
	case "document":
		return Document, nil
	case "block_quote":
		return BlockQuote, nil
	case "list":
		return List, nil
	case "item":
		return Item, nil
	case "paragraph":
		return Paragraph, nil
	case "heading":
		return Heading, nil
	case "hr":
		return HorizontalRule, nil
	case "emph":
		return Emph, nil
	case "strong":
		return Strong, nil
	case "del":
		return Del, nil
	case "link":
		return Link, nil
	case "link_text":
		return LinkText, nil
	case "image":
		return Image, nil
	case "image_text":
		return ImageText, nil
	case "text":
		return Text, nil
	case "html_block":
		return HTMLBlock, nil
	case "code_block":
		return CodeBlock, nil
	case "softbreak":
		return Softbreak, nil
	case "hardbreak":
		return Hardbreak, nil
	case "indent":
		return Indent, nil
	case "code":
		return Code, nil
	case "html_span":
		return HTMLSpan, nil
	case "table":
		return Table, nil
	case "table_cel":
		return TableCell, nil
	case "table_head":
		return TableHead, nil
	case "table_body":
		return TableBody, nil
	case "table_row":
		return TableRow, nil

	default:
		return 0, fmt.Errorf("Invalid style element type: %s", key)
	}
}
