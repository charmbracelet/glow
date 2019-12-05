package gold

import (
	"bytes"
)

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
