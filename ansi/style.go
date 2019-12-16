package ansi

import (
	"github.com/lucasb-eyer/go-colorful"
)

type StylePrimitive struct {
	BlockPrefix     string  `json:"block_prefix"`
	BlockSuffix     string  `json:"block_suffix"`
	Prefix          string  `json:"prefix"`
	Suffix          string  `json:"suffix"`
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
	Format          string  `json:"format"`
}

type StyleTask struct {
	StylePrimitive
	Ticked   string `json:"ticked"`
	Unticked string `json:"unticked"`
}

type StyleBlock struct {
	StylePrimitive
	Indent *uint `json:"indent"`
	Margin *uint `json:"margin"`
}

type StyleCodeBlock struct {
	StyleBlock
	Theme  string `json:"theme"`
	Chroma *struct {
		Text                StylePrimitive `json:"text"`
		Error               StylePrimitive `json:"error"`
		Comment             StylePrimitive `json:"comment"`
		CommentPreproc      StylePrimitive `json:"comment_preproc"`
		Keyword             StylePrimitive `json:"keyword"`
		KeywordReserved     StylePrimitive `json:"keyword_reserved"`
		KeywordNamespace    StylePrimitive `json:"keyword_namespace"`
		KeywordType         StylePrimitive `json:"keyword_type"`
		Operator            StylePrimitive `json:"operator"`
		Punctuation         StylePrimitive `json:"punctuation"`
		Name                StylePrimitive `json:"name"`
		NameBuiltin         StylePrimitive `json:"name_builtin"`
		NameTag             StylePrimitive `json:"name_tag"`
		NameAttribute       StylePrimitive `json:"name_attribute"`
		NameClass           StylePrimitive `json:"name_class"`
		NameConstant        StylePrimitive `json:"name_constant"`
		NameDecorator       StylePrimitive `json:"name_decorator"`
		NameException       StylePrimitive `json:"name_exception"`
		NameFunction        StylePrimitive `json:"name_function"`
		NameOther           StylePrimitive `json:"name_other"`
		Literal             StylePrimitive `json:"literal"`
		LiteralNumber       StylePrimitive `json:"literal_number"`
		LiteralDate         StylePrimitive `json:"literal_date"`
		LiteralString       StylePrimitive `json:"literal_string"`
		LiteralStringEscape StylePrimitive `json:"literal_string_escape"`
		GenericDeleted      StylePrimitive `json:"generic_deleted"`
		GenericEmph         StylePrimitive `json:"generic_emph"`
		GenericInserted     StylePrimitive `json:"generic_inserted"`
		GenericStrong       StylePrimitive `json:"generic_strong"`
		GenericSubheading   StylePrimitive `json:"generic_subheading"`
		Background          StylePrimitive `json:"background"`
	} `json:"chroma"`
}

type StyleList struct {
	StyleBlock
	LevelIndent uint `json:"level_indent"`
}

type StyleConfig struct {
	Document   StyleBlock `json:"document"`
	BlockQuote StyleBlock `json:"block_quote"`
	Paragraph  StyleBlock `json:"paragraph"`
	List       StyleList  `json:"list"`

	Heading StyleBlock `json:"heading"`
	H1      StyleBlock `json:"h1"`
	H2      StyleBlock `json:"h2"`
	H3      StyleBlock `json:"h3"`
	H4      StyleBlock `json:"h4"`
	H5      StyleBlock `json:"h5"`
	H6      StyleBlock `json:"h6"`

	Text           StylePrimitive `json:"text"`
	Strikethrough  StylePrimitive `json:"strike_through"`
	Emph           StylePrimitive `json:"emph"`
	Strong         StylePrimitive `json:"strong"`
	HorizontalRule StylePrimitive `json:"hr"`

	Item        StylePrimitive `json:"item"`
	Enumeration StylePrimitive `json:"enumeration"`
	Task        StyleTask      `json:"task"`

	Link     StylePrimitive `json:"link"`
	LinkText StylePrimitive `json:"link_text"`

	Image     StylePrimitive `json:"image"`
	ImageText StylePrimitive `json:"image_text"`

	Code      StyleBlock     `json:"code"`
	CodeBlock StyleCodeBlock `json:"code_block"`

	Table StyleBlock `json:"table"`

	DefinitionList        StyleBlock     `json:"definition_list"`
	DefinitionTerm        StylePrimitive `json:"definition_term"`
	DefinitionDescription StylePrimitive `json:"definition_description"`

	HTMLBlock StyleBlock `json:"html_block"`
	HTMLSpan  StyleBlock `json:"html_span"`
}

func cascadeStyles(onlyColors bool, s ...StyleBlock) StyleBlock {
	var r StyleBlock

	for _, v := range s {
		r = cascadeStyle(r, v, onlyColors)
	}
	return r
}

func cascadeStyle(parent StyleBlock, child StyleBlock, onlyColors bool) StyleBlock {
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
		s.BlockPrefix = parent.BlockPrefix
		s.BlockSuffix = parent.BlockSuffix
		s.Prefix = parent.Prefix
		s.Suffix = parent.Suffix
		s.Format = parent.Format
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
	if child.BlockPrefix != "" {
		s.BlockPrefix = child.BlockPrefix
	}
	if child.BlockSuffix != "" {
		s.BlockSuffix = child.BlockSuffix
	}
	if child.Prefix != "" {
		s.Prefix = child.Prefix
	}
	if child.Suffix != "" {
		s.Suffix = child.Suffix
	}
	if child.Format != "" {
		s.Format = child.Format
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
	c2 := colorful.Color{R: float64(cr) / 255.0, G: float64(cg) / 255.0, B: float64(cb) / 255.0}
	g2 := colorful.Color{R: float64(gv) / 255.0, G: float64(gv) / 255.0, B: float64(gv) / 255.0}
	colorDist := c.DistanceLab(c2)
	grayDist := c.DistanceLab(g2)

	if colorDist <= grayDist {
		return 16 + ci, nil
	}
	return 232 + grayIdx, nil
}
