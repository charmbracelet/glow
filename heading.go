package gold

import (
	"fmt"
	"io"
	"strings"

	bf "gopkg.in/russross/blackfriday.v2"
)

type HeadingElement struct {
}

func (e *HeadingElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	var pre string
	if node.Prev != nil {
		pre = "\n"
	}

	el := &BaseElement{
		Pre:   pre,
		Token: fmt.Sprintf("%s %s", strings.Repeat("#", node.HeadingData.Level), node.FirstChild.Literal),
		Style: Heading,
	}
	return el.Render(w, node, tr)
}
