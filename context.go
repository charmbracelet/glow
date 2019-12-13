package gold

import (
	"html"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

type RenderContext struct {
	options Options
	styles  StyleConfig

	blockStack *BlockStack
	table      *TableElement

	stripper *bluemonday.Policy
}

func NewRenderContext(options Options) RenderContext {
	return RenderContext{
		options:    options,
		blockStack: &BlockStack{},
		table:      &TableElement{},
		stripper:   bluemonday.StrictPolicy(),
	}
}

func (ctx RenderContext) SanitizeHTML(s string, trimSpaces bool) string {
	s = ctx.stripper.Sanitize(s)
	if trimSpaces {
		s = strings.TrimSpace(s)
	}

	return html.UnescapeString(s)
}
