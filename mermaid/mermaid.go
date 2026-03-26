package mermaid

import (
	"fmt"
	"regexp"
	"strings"

	mermaidcmd "github.com/AlexanderGrooff/mermaid-ascii/cmd"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/log"
)

var mermaidBlockRegex = regexp.MustCompile("(?s)```mermaid\\s*\n(.*?)```")

var unsupportedKeywords = []string{
	"loop",
	"alt",
	"opt",
	"par",
	"rect",
	"Note",
	"note",
	"activate",
	"deactivate",
}

var unsupportedDiagramTypes = []string{
	"classDiagram",
	"stateDiagram",
	"erDiagram",
	"pie",
	"gantt",
	"journey",
	"gitGraph",
	"mindmap",
	"timeline",
	"quadrantChart",
	"xychart",
	"sankey",
}

func Process(markdown string, enabled bool) string {
	if !enabled {
		return markdown
	}

	if !strings.Contains(markdown, "```mermaid") {
		return markdown
	}

	return mermaidBlockRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		submatches := mermaidBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		code := strings.TrimSpace(submatches[1])
		if code == "" {
			return match
		}

		if err := checkSupported(code); err != nil {
			log.Debug("mermaid diagram has unsupported features", "error", err)
			return match
		}

		ascii, err := renderDiagram(code)
		if err != nil {
			log.Debug("mermaid render failed", "error", err)
			return match
		}

		return utils.WrapCodeBlock(ascii, "")
	})
}

func checkSupported(code string) error {
	lines := strings.Split(code, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, diagType := range unsupportedDiagramTypes {
			if strings.HasPrefix(trimmed, diagType) {
				return fmt.Errorf("unsupported diagram type: %s", diagType)
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, keyword := range unsupportedKeywords {
			if strings.HasPrefix(trimmed, keyword) {
				rest := trimmed[len(keyword):]
				if rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '[' || rest[0] == '(' {
					return fmt.Errorf("unsupported feature: %s", keyword)
				}
			}
		}
	}

	return nil
}

func renderDiagram(code string) (string, error) {
	d, err := mermaidcmd.DiagramFactory(code)
	if err != nil {
		return "", err
	}

	if err := d.Parse(code); err != nil {
		return "", err
	}

	config := diagram.DefaultConfig()
	config.StyleType = "cli"

	return d.Render(config)
}
