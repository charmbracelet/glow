package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

type theme struct {
	Name      string
	Doc       string
	H1        string
	H1Bg      string
	H1Bold    bool
	H2        string
	H3        string
	H6        string
	Code      string
	CodeBg    string
	CodeBlock string
	Link      string
	LinkUnder bool
	HR        string
	BqToken   string
}

var themes = []theme{
	{Name: "Catppuccin Mocha", Doc: "252", H1: "228", H1Bg: "63", H1Bold: true, H2: "39", H3: "39", H6: "35", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "30", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Catppuccin Latte", Doc: "234", H1: "228", H1Bg: "63", H1Bold: true, H2: "27", H3: "27", H6: "35", Code: "203", CodeBg: "254", CodeBlock: "242", Link: "36", LinkUnder: true, HR: "249", BqToken: "│ "},
	{Name: "Nord", Doc: "252", H1: "228", H1Bg: "67", H1Bold: true, H2: "110", H3: "110", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "110", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Gruvbox Dark", Doc: "223", H1: "228", H1Bg: "88", H1Bold: true, H2: "214", H3: "214", H6: "108", Code: "203", CodeBg: "237", CodeBlock: "244", Link: "109", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Gruvbox Light", Doc: "237", H1: "228", H1Bg: "88", H1Bold: true, H2: "214", H3: "214", H6: "108", Code: "203", CodeBg: "254", CodeBlock: "242", Link: "109", LinkUnder: true, HR: "249", BqToken: "│ "},
	{Name: "Solarized Dark", Doc: "252", H1: "228", H1Bg: "33", H1Bold: true, H2: "37", H3: "37", H6: "64", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "33", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Solarized Light", Doc: "234", H1: "228", H1Bg: "33", H1Bold: true, H2: "37", H3: "37", H6: "64", Code: "203", CodeBg: "254", CodeBlock: "242", Link: "33", LinkUnder: true, HR: "249", BqToken: "│ "},
	{Name: "Tokyo Night", Doc: "252", H1: "228", H1Bg: "99", H1Bold: true, H2: "147", H3: "147", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "147", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Dracula", Doc: "252", H1: "228", H1Bg: "99", H1Bold: true, H2: "212", H3: "212", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "212", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "One Dark", Doc: "252", H1: "228", H1Bg: "67", H1Bold: true, H2: "75", H3: "75", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "75", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Surprise Me", Doc: "252", H1: "228", H1Bg: "63", H1Bold: true, H2: "39", H3: "39", H6: "35", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "30", LinkUnder: true, HR: "240", BqToken: "│ "},
}

var outputPath string

var styleCmd = &cobra.Command{
	Use:   "style",
	Short: "Manage glow color styles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

var styleInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a color style from a theme palette",
	Long: paragraph(fmt.Sprintf(
		"\nPick from preset color themes using arrow keys, preview the colors, and save.\nThe generated file can be used with %s.",
		keyword("glow -s path/to/style.json"),
	)),
	Example: paragraph("  glow style init\n  glow style init -o ~/mytheme.json"),
	Args:    cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		return runStyleInitTUI()
	},
}

func init() {
	styleInitCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for the style JSON (default ~/.config/glow/style.json)")
	styleCmd.AddCommand(styleInitCmd)
}

type keyMap struct {
	Up, Down, Enter, Random, Quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Random, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Enter, k.Random, k.Quit}}
}

var keys = keyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
	Random: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "random")),
	Quit:   key.NewBinding(key.WithKeys("esc", "q", "ctrl+c"), key.WithHelp("esc/q", "cancel")),
}

type model struct {
	themes   []theme
	cursor   int
	result   *result
	quitting bool
	help     help.Model
}

type result struct {
	styleJSON []byte
	themeName string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.themes) - 1
			}
		case key.Matches(msg, keys.Down):
			m.cursor++
			if m.cursor >= len(m.themes) {
				m.cursor = 0
			}
		case key.Matches(msg, keys.Random):
			rt := randomTheme()
			m.themes = append(m.themes[:len(m.themes)-1], rt)
			m.cursor = len(m.themes) - 1
		case key.Matches(msg, keys.Enter):
			t := m.themes[m.cursor]
			data, _ := json.MarshalIndent(buildStyle(t.Doc, t.H1, t.H1Bg, t.H1Bold, t.H2, t.H3, t.H6,
				t.Code, t.CodeBg, t.CodeBlock, t.Link, t.LinkUnder, t.HR, t.BqToken), "", "  ")
			m.result = &result{styleJSON: data, themeName: t.Name}
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	t := m.themes[m.cursor]
	div := strings.Repeat("─", 70)

	listRows := make([]string, len(m.themes))
	for i, th := range m.themes {
		prefix := "  "
		var row string
		if i == m.cursor {
			prefix = cBg("#04B575").Render("▸")
			sel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFDF5"))
			row = sel.Render(th.Name)
		} else {
			row = cFg("#A49FA5").Render(th.Name)
		}
		listRows[i] = fmt.Sprintf("%s %s", prefix, row)
	}

	square := func(color string) string {
		return cBg(color).Render("  ")
	}

	palette := fmt.Sprintf(
		"  %s %-6s %-4s    %s %-6s %-4s    %s %-6s %-4s\n  %s %-6s %-4s    %s %-6s %-4s    %s %-6s %-4s",
		square(t.Doc), "Text", t.Doc,
		square(t.H1), "H1", t.H1,
		square(t.Code), "Code", t.Code,
		square(t.Link), "Link", t.Link,
		square(t.HR), "HR", t.HR,
		square(t.H1Bg), "H1Bg", t.H1Bg,
	)

	sample := fmt.Sprintf(
		"  %s\n  %s\n  %s\n  %s %s",
		cBoth(t.H1, t.H1Bg).Render(" Heading 1 (H1) "),
		cFg(t.H2).Render("  ## Heading 2 (H2)"),
		cFg(t.Doc).Render("  Normal text with ")+cBoth(t.Code, t.CodeBg).Render(" inline code ")+cFg(t.Doc).Render(" here."),
		cFg(t.Link).Render("  > Link hover preview"),
		cFg(t.HR).Render("  ─────────────"),
	)

	content := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n  %s\n%s\n\n  %s\n%s\n\n%s\n",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Render("Pick a theme"),
		div,
		strings.Join(listRows, "\n"),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#777777")).Render("Palette"),
		palette,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#777777")).Render("Sample"),
		sample,
		m.help.View(keys),
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func runStyleInitTUI() error {
	if outputPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			return fmt.Errorf("unable to determine home directory: %w", err)
		}
		outputPath = filepath.Join(home, ".config", "glow", "style.json")
	}

	m := model{themes: themes, help: help.New()}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}

	m = final.(model)
	if m.result == nil {
		fmt.Println("  Cancelled.")
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o700); err != nil {
		return fmt.Errorf("unable to create directory: %w", err)
	}
	if err := os.WriteFile(outputPath, m.result.styleJSON, 0o600); err != nil {
		return fmt.Errorf("unable to write style file: %w", err)
	}

	fmt.Printf("  ✓ %s saved to: %s\n", m.result.themeName, outputPath)
	fmt.Printf("  Use it with: %s\n", keyword("glow -s "+outputPath))
	return nil
}

func cFg(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func cBg(color string) lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(color))
}

func cBoth(fg, bg string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bg))
}

func randomTheme() theme {
	colors := make([]string, 10)
	for i := range colors {
		colors[i] = fmt.Sprintf("%d", rand.Intn(256))
	}
	return theme{
		Name: "Surprise Me", Doc: colors[0], H1: colors[1], H1Bg: colors[2], H1Bold: true,
		H2: colors[3], H3: colors[4], H6: colors[5], Code: colors[6], CodeBg: colors[7],
		CodeBlock: colors[8], Link: colors[9], LinkUnder: true, HR: colors[0], BqToken: "│ ",
	}
}

func buildStyle(docColor, h1Color, h1Bg string, h1Bold bool, h2Color, h3Color, h6Color string,
	codeColor, codeBg, codeBlockColor, linkColor string, linkUnderline bool, hrColor, blockQuoteToken string) map[string]interface{} {

	return map[string]interface{}{
		"document":      map[string]interface{}{"block_prefix": "\n", "block_suffix": "\n", "color": docColor, "margin": 2},
		"block_quote":   map[string]interface{}{"indent": 1, "indent_token": blockQuoteToken},
		"paragraph":     map[string]interface{}{},
		"list":          map[string]interface{}{"level_indent": 2},
		"heading":       map[string]interface{}{"block_suffix": "\n", "color": h2Color, "bold": true},
		"h1":            map[string]interface{}{"prefix": " ", "suffix": " ", "color": h1Color, "background_color": h1Bg, "bold": h1Bold},
		"h2":            map[string]interface{}{"prefix": "## "},
		"h3":            map[string]interface{}{"prefix": "### ", "color": h3Color},
		"h4":            map[string]interface{}{"prefix": "#### "},
		"h5":            map[string]interface{}{"prefix": "##### "},
		"h6":            map[string]interface{}{"prefix": "###### ", "color": h6Color, "bold": false},
		"text":          map[string]interface{}{},
		"strikethrough": map[string]interface{}{"crossed_out": true},
		"emph":          map[string]interface{}{"italic": true},
		"strong":        map[string]interface{}{"bold": true},
		"hr":            map[string]interface{}{"color": hrColor, "format": "\n--------\n"},
		"item":          map[string]interface{}{"block_prefix": "• "},
		"enumeration":   map[string]interface{}{"block_prefix": ". "},
		"task":          map[string]interface{}{"ticked": "[✓] ", "unticked": "[ ] "},
		"link":          map[string]interface{}{"color": linkColor, "underline": linkUnderline},
		"link_text":     map[string]interface{}{"color": "35", "bold": true},
		"image":         map[string]interface{}{"color": "212", "underline": true},
		"image_text":    map[string]interface{}{"color": "243", "format": "Image: {{.text}} →"},
		"code":          map[string]interface{}{"prefix": "\u00a0", "suffix": "\u00a0", "color": codeColor, "background_color": codeBg},
		"code_block": map[string]interface{}{
			"color": codeBlockColor, "margin": 2,
			"chroma": map[string]interface{}{
				"text":                  map[string]interface{}{"color": "#C4C4C4"},
				"error":                 map[string]interface{}{"color": "#F1F1F1", "background_color": "#F05B5B"},
				"comment":               map[string]interface{}{"color": "#676767"},
				"comment_preproc":       map[string]interface{}{"color": "#FF875F"},
				"keyword":               map[string]interface{}{"color": "#00AAFF"},
				"keyword_reserved":      map[string]interface{}{"color": "#FF5FD2"},
				"keyword_namespace":     map[string]interface{}{"color": "#FF5F87"},
				"keyword_type":          map[string]interface{}{"color": "#6E6ED8"},
				"operator":              map[string]interface{}{"color": "#EF8080"},
				"punctuation":           map[string]interface{}{"color": "#E8E8A8"},
				"name":                  map[string]interface{}{"color": "#C4C4C4"},
				"name_builtin":          map[string]interface{}{"color": "#FF8EC7"},
				"name_tag":              map[string]interface{}{"color": "#B083EA"},
				"name_attribute":        map[string]interface{}{"color": "#7A7AE6"},
				"name_class":            map[string]interface{}{"color": "#F1F1F1", "underline": true, "bold": true},
				"name_decorator":        map[string]interface{}{"color": "#FFFF87"},
				"name_function":         map[string]interface{}{"color": "#00D787"},
				"literal_number":        map[string]interface{}{"color": "#6EEFC0"},
				"literal_string":        map[string]interface{}{"color": "#C69669"},
				"literal_string_escape": map[string]interface{}{"color": "#AFFFD7"},
				"generic_deleted":       map[string]interface{}{"color": "#FD5B5B"},
				"generic_emph":          map[string]interface{}{"italic": true},
				"generic_inserted":      map[string]interface{}{"color": "#00D787"},
				"generic_strong":        map[string]interface{}{"bold": true},
				"generic_subheading":    map[string]interface{}{"color": "#777777"},
				"background":            map[string]interface{}{"background_color": "#373737"},
			},
		},
		"table":                  map[string]interface{}{},
		"definition_list":        map[string]interface{}{},
		"definition_term":        map[string]interface{}{},
		"definition_description": map[string]interface{}{"block_prefix": "\n🠶 "},
		"html_block":             map[string]interface{}{},
		"html_span":              map[string]interface{}{},
	}
}
