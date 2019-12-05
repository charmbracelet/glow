package gold

import (
	"fmt"

	"github.com/lucasb-eyer/go-colorful"
)

type StyleType int

const (
	Document StyleType = iota
	BlockQuote
	List
	Item
	Enumeration
	Paragraph
	Heading
	H1
	H2
	H3
	H4
	H5
	H6
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
	Code
	HTMLSpan
	Table
	TableCell
	TableHead
	TableBody
	TableRow
)

type ElementStyle struct {
	Color           *string `json:"color"`
	BackgroundColor *string `json:"background_color"`
	Underline       *bool   `json:"underline"`
	Bold            *bool   `json:"bold"`
	Italic          *bool   `json:"italic"`
	CrossedOut      *bool   `json:"crossed_out"`
	Faint           *bool   `json:"faint"`
	Conceal         *bool   `json:"conceal"`
	Overlined       *bool   `json:"overlined"`
	Inverse         *bool   `json:"inverse"`
	Blink           *bool   `json:"blink"`
	Indent          *uint   `json:"indent"`
	Margin          *uint   `json:"margin"`
	Theme           string  `json:"theme"`
	Prefix          string  `json:"prefix"`
	Suffix          string  `json:"suffix"`
}

func cascadeStyles(onlyColors bool, s ...ElementStyle) ElementStyle {
	var r ElementStyle

	for _, v := range s {
		r = cascadeStyle(r, v, onlyColors)
	}
	return r
}

func cascadeStyle(parent ElementStyle, child ElementStyle, onlyColors bool) ElementStyle {
	s := child

	s.Color = parent.Color
	s.BackgroundColor = parent.BackgroundColor

	if !onlyColors {
		s.Indent = parent.Indent
		s.Margin = parent.Margin
		s.Underline = parent.Underline
		s.Bold = parent.Bold
		s.Italic = parent.Italic
		s.CrossedOut = parent.CrossedOut
		s.Faint = parent.Faint
		s.Conceal = parent.Conceal
		s.Overlined = parent.Overlined
		s.Inverse = parent.Inverse
		s.Blink = parent.Blink
		s.Prefix = parent.Prefix
		s.Suffix = parent.Suffix
	}

	if child.Color != nil {
		s.Color = child.Color
	}
	if child.BackgroundColor != nil {
		s.BackgroundColor = child.BackgroundColor
	}
	if child.Indent != nil {
		s.Indent = child.Indent
	}
	if child.Margin != nil {
		s.Margin = child.Margin
	}
	if child.Underline != nil {
		s.Underline = child.Underline
	}
	if child.Bold != nil {
		s.Bold = child.Bold
	}
	if child.Italic != nil {
		s.Italic = child.Italic
	}
	if child.CrossedOut != nil {
		s.CrossedOut = child.CrossedOut
	}
	if child.Faint != nil {
		s.Faint = child.Faint
	}
	if child.Conceal != nil {
		s.Conceal = child.Conceal
	}
	if child.Overlined != nil {
		s.Overlined = child.Overlined
	}
	if child.Inverse != nil {
		s.Inverse = child.Inverse
	}
	if child.Blink != nil {
		s.Blink = child.Blink
	}
	if child.Prefix != "" {
		s.Prefix = child.Prefix
	}
	if child.Suffix != "" {
		s.Suffix = child.Suffix
	}

	return s
}

func hexToANSIColor(h string) (int, error) {
	c, err := colorful.Hex(h)
	if err != nil {
		return 0, err
	}

	v2ci := func(v float64) int {
		if v < 48 {
			return 0
		}
		if v < 115 {
			return 1
		}
		return int((v - 35) / 40)
	}

	// Calculate the nearest 0-based color index at 16..231
	r := v2ci(c.R * 255.0) // 0..5 each
	g := v2ci(c.G * 255.0)
	b := v2ci(c.B * 255.0)
	ci := 36*r + 6*g + b /* 0..215 */

	// Calculate the represented colors back from the index
	i2cv := [6]int{0, 0x5f, 0x87, 0xaf, 0xd7, 0xff}
	cr := i2cv[r] // r/g/b, 0..255 each
	cg := i2cv[g]
	cb := i2cv[b]

	// Calculate the nearest 0-based gray index at 232..255
	var grayIdx int
	average := (r + g + b) / 3
	if average > 238 {
		grayIdx = 23
	} else {
		grayIdx = (average - 3) / 10 // 0..23
	}
	gv := 8 + 10*grayIdx // same value for r/g/b, 0..255

	// Return the one which is nearer to the original input rgb value
	c2 := colorful.Color{float64(cr) / 255.0, float64(cg) / 255.0, float64(cb) / 255.0}
	g2 := colorful.Color{float64(gv) / 255.0, float64(gv) / 255.0, float64(gv) / 255.0}
	colorDist := c.DistanceLab(c2)
	grayDist := c.DistanceLab(g2)

	if colorDist <= grayDist {
		return 16 + ci, nil
	}
	return 232 + grayIdx, nil
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
	case "enumeration":
		return Enumeration, nil
	case "paragraph":
		return Paragraph, nil
	case "heading":
		return Heading, nil
	case "h1":
		return H1, nil
	case "h2":
		return H2, nil
	case "h3":
		return H3, nil
	case "h4":
		return H4, nil
	case "h5":
		return H5, nil
	case "h6":
		return H6, nil
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
