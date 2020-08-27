package main

import (
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
)

const (
	wrapAt       = 78
	indentAmount = 2
)

func formatBlock(s string) string {
	return indent.String(wordwrap.String(s, wrapAt-indentAmount), indentAmount)
}
