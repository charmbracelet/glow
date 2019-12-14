package ansi

import (
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/styles"
)

type CodeBlockElement struct {
	Code     string
	Language string
}

func chromaColor(style StylePrimitive) string {
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

	return s
}

func (e *CodeBlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	var indent uint
	var margin uint
	rules := ctx.options.Styles.CodeBlock
	if rules.Indent != nil {
		indent = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	theme := rules.Theme

	styles.Register(chroma.MustNewStyle("charm",
		chroma.StyleEntries{
			chroma.Text:                chromaColor(rules.Text),
			chroma.Error:               chromaColor(rules.Error),
			chroma.Comment:             chromaColor(rules.Comment),
			chroma.CommentPreproc:      chromaColor(rules.CommentPreproc),
			chroma.Keyword:             chromaColor(rules.Keyword),
			chroma.KeywordReserved:     chromaColor(rules.KeywordReserved),
			chroma.KeywordNamespace:    chromaColor(rules.KeywordNamespace),
			chroma.KeywordType:         chromaColor(rules.KeywordType),
			chroma.Operator:            chromaColor(rules.Operator),
			chroma.Punctuation:         chromaColor(rules.Punctuation),
			chroma.Name:                chromaColor(rules.Name),
			chroma.NameBuiltin:         chromaColor(rules.NameBuiltin),
			chroma.NameTag:             chromaColor(rules.NameTag),
			chroma.NameAttribute:       chromaColor(rules.NameAttribute),
			chroma.NameClass:           chromaColor(rules.NameClass),
			chroma.NameConstant:        chromaColor(rules.NameConstant),
			chroma.NameDecorator:       chromaColor(rules.NameDecorator),
			chroma.NameException:       chromaColor(rules.NameException),
			chroma.NameFunction:        chromaColor(rules.NameFunction),
			chroma.NameOther:           chromaColor(rules.NameOther),
			chroma.Literal:             chromaColor(rules.Literal),
			chroma.LiteralNumber:       chromaColor(rules.LiteralNumber),
			chroma.LiteralDate:         chromaColor(rules.LiteralDate),
			chroma.LiteralString:       chromaColor(rules.LiteralString),
			chroma.LiteralStringEscape: chromaColor(rules.LiteralStringEscape),
			chroma.GenericDeleted:      chromaColor(rules.GenericDeleted),
			chroma.GenericEmph:         chromaColor(rules.GenericEmph),
			chroma.GenericInserted:     chromaColor(rules.GenericInserted),
			chroma.GenericStrong:       chromaColor(rules.GenericStrong),
			chroma.GenericSubheading:   chromaColor(rules.GenericSubheading),
			chroma.Background:          chromaColor(rules.Background),
		}))

	iw := &IndentWriter{
		Indent: indent + margin,
		IndentFunc: func(wr io.Writer) {
			renderText(w, bs.Current().Style.StylePrimitive, " ")
		},
		Forward: &AnsiWriter{
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
