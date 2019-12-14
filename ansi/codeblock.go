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
			chroma.Text:                chromaColor(rules.Chroma.Text),
			chroma.Error:               chromaColor(rules.Chroma.Error),
			chroma.Comment:             chromaColor(rules.Chroma.Comment),
			chroma.CommentPreproc:      chromaColor(rules.Chroma.CommentPreproc),
			chroma.Keyword:             chromaColor(rules.Chroma.Keyword),
			chroma.KeywordReserved:     chromaColor(rules.Chroma.KeywordReserved),
			chroma.KeywordNamespace:    chromaColor(rules.Chroma.KeywordNamespace),
			chroma.KeywordType:         chromaColor(rules.Chroma.KeywordType),
			chroma.Operator:            chromaColor(rules.Chroma.Operator),
			chroma.Punctuation:         chromaColor(rules.Chroma.Punctuation),
			chroma.Name:                chromaColor(rules.Chroma.Name),
			chroma.NameBuiltin:         chromaColor(rules.Chroma.NameBuiltin),
			chroma.NameTag:             chromaColor(rules.Chroma.NameTag),
			chroma.NameAttribute:       chromaColor(rules.Chroma.NameAttribute),
			chroma.NameClass:           chromaColor(rules.Chroma.NameClass),
			chroma.NameConstant:        chromaColor(rules.Chroma.NameConstant),
			chroma.NameDecorator:       chromaColor(rules.Chroma.NameDecorator),
			chroma.NameException:       chromaColor(rules.Chroma.NameException),
			chroma.NameFunction:        chromaColor(rules.Chroma.NameFunction),
			chroma.NameOther:           chromaColor(rules.Chroma.NameOther),
			chroma.Literal:             chromaColor(rules.Chroma.Literal),
			chroma.LiteralNumber:       chromaColor(rules.Chroma.LiteralNumber),
			chroma.LiteralDate:         chromaColor(rules.Chroma.LiteralDate),
			chroma.LiteralString:       chromaColor(rules.Chroma.LiteralString),
			chroma.LiteralStringEscape: chromaColor(rules.Chroma.LiteralStringEscape),
			chroma.GenericDeleted:      chromaColor(rules.Chroma.GenericDeleted),
			chroma.GenericEmph:         chromaColor(rules.Chroma.GenericEmph),
			chroma.GenericInserted:     chromaColor(rules.Chroma.GenericInserted),
			chroma.GenericStrong:       chromaColor(rules.Chroma.GenericStrong),
			chroma.GenericSubheading:   chromaColor(rules.Chroma.GenericSubheading),
			chroma.Background:          chromaColor(rules.Chroma.Background),
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
