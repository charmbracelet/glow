package gold

import (
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
	Text
	HTMLBlock
	CodeBlock
	Softbreak
	Hardbreak
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
	Theme           string `json:"theme"`
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
