package ansi

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
)

// A LinkElement is used to render hyperlinks.
type LinkElement struct {
	BaseURL  string
	URL      string
	Children []ElementRenderer
	SkipText bool
	SkipHref bool
}

// Render renders a LinkElement.
func (e *LinkElement) Render(w io.Writer, ctx RenderContext) error {
	if !e.SkipText {
		if err := e.renderTextPart(w, ctx); err != nil {
			return err
		}
	}
	if !e.SkipHref {
		if err := e.renderHrefPart(w, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (e *LinkElement) renderTextPart(w io.Writer, ctx RenderContext) error {
	for _, child := range e.Children {
		if r, ok := child.(StyleOverriderElementRenderer); ok {
			st := ctx.options.Styles.LinkText
			if err := r.StyleOverrideRender(w, ctx, st); err != nil {
				return fmt.Errorf("glamour: error rendering with style: %w", err)
			}
		} else {
			var b bytes.Buffer
			if err := child.Render(&b, ctx); err != nil {
				return fmt.Errorf("glamour: error rendering: %w", err)
			}
			el := &BaseElement{
				Token: b.String(),
				Style: ctx.options.Styles.LinkText,
			}
			if err := el.Render(w, ctx); err != nil {
				return fmt.Errorf("glamour: error rendering: %w", err)
			}
		}
	}
	return nil
}

func (e *LinkElement) renderHrefPart(w io.Writer, ctx RenderContext) error {
	prefix := ""
	if !e.SkipText {
		prefix = " "
	}

	u, err := url.Parse(e.URL)
	if err == nil && "#"+u.Fragment != e.URL { // if the URL only consists of an anchor, ignore it
		el := &BaseElement{
			Token:  resolveRelativeURL(e.BaseURL, e.URL),
			Prefix: prefix,
			Style:  ctx.options.Styles.Link,
		}
		if err := el.Render(w, ctx); err != nil {
			return err
		}
	}
	return nil
}
