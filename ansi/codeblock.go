package ansi

import (
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/styles"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/indent"
)

type CodeBlockElement struct {
	Code     string
	Language string
}

func chromaStyle(style StylePrimitive) string {
	var s string

	if style.Color != nil {
		s = *style.Color
	}
	if style.BackgroundColor != nil {
		if s != "" {
			s += " "
		}
		s += "bg:" + *style.BackgroundColor
	}
	if style.Italic != nil && *style.Italic {
		if s != "" {
			s += " "
		}
		s += "italic"
	}
	if style.Bold != nil && *style.Bold {
		if s != "" {
			s += " "
		}
		s += "bold"
	}
	if style.Underline != nil && *style.Underline {
		if s != "" {
			s += " "
		}
		s += "underline"
	}

	return s
}

func (e *CodeBlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	var indentation uint
	var margin uint
	rules := ctx.options.Styles.CodeBlock
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	theme := rules.Theme

	styles.Register(chroma.MustNewStyle("charm",
		chroma.StyleEntries{
			chroma.Text:                chromaStyle(rules.Chroma.Text),
			chroma.Error:               chromaStyle(rules.Chroma.Error),
			chroma.Comment:             chromaStyle(rules.Chroma.Comment),
			chroma.CommentPreproc:      chromaStyle(rules.Chroma.CommentPreproc),
			chroma.Keyword:             chromaStyle(rules.Chroma.Keyword),
			chroma.KeywordReserved:     chromaStyle(rules.Chroma.KeywordReserved),
			chroma.KeywordNamespace:    chromaStyle(rules.Chroma.KeywordNamespace),
			chroma.KeywordType:         chromaStyle(rules.Chroma.KeywordType),
			chroma.Operator:            chromaStyle(rules.Chroma.Operator),
			chroma.Punctuation:         chromaStyle(rules.Chroma.Punctuation),
			chroma.Name:                chromaStyle(rules.Chroma.Name),
			chroma.NameBuiltin:         chromaStyle(rules.Chroma.NameBuiltin),
			chroma.NameTag:             chromaStyle(rules.Chroma.NameTag),
			chroma.NameAttribute:       chromaStyle(rules.Chroma.NameAttribute),
			chroma.NameClass:           chromaStyle(rules.Chroma.NameClass),
			chroma.NameConstant:        chromaStyle(rules.Chroma.NameConstant),
			chroma.NameDecorator:       chromaStyle(rules.Chroma.NameDecorator),
			chroma.NameException:       chromaStyle(rules.Chroma.NameException),
			chroma.NameFunction:        chromaStyle(rules.Chroma.NameFunction),
			chroma.NameOther:           chromaStyle(rules.Chroma.NameOther),
			chroma.Literal:             chromaStyle(rules.Chroma.Literal),
			chroma.LiteralNumber:       chromaStyle(rules.Chroma.LiteralNumber),
			chroma.LiteralDate:         chromaStyle(rules.Chroma.LiteralDate),
			chroma.LiteralString:       chromaStyle(rules.Chroma.LiteralString),
			chroma.LiteralStringEscape: chromaStyle(rules.Chroma.LiteralStringEscape),
			chroma.GenericDeleted:      chromaStyle(rules.Chroma.GenericDeleted),
			chroma.GenericEmph:         chromaStyle(rules.Chroma.GenericEmph),
			chroma.GenericInserted:     chromaStyle(rules.Chroma.GenericInserted),
			chroma.GenericStrong:       chromaStyle(rules.Chroma.GenericStrong),
			chroma.GenericSubheading:   chromaStyle(rules.Chroma.GenericSubheading),
			chroma.Background:          chromaStyle(rules.Chroma.Background),
		}))

	iw := &indent.Writer{
		Indent: indentation + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Current().Style.StylePrimitive, " ")
		},
		Forward: &ansi.Writer{
			Forward: w,
		},
	}

	if len(theme) > 0 {
		renderText(iw, bs.Current().Style.StylePrimitive, rules.BlockPrefix)
		err := quick.Highlight(iw, e.Code, e.Language, "terminal256", theme)
		renderText(iw, bs.Current().Style.StylePrimitive, rules.BlockSuffix)
		return err
	}

	// fallback rendering
	el := &BaseElement{
		Token: e.Code,
		Style: rules.StylePrimitive,
	}

	return el.Render(iw, ctx)
}
