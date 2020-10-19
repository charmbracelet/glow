package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/meowgorithm/babyenv"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/ui"
)

var (
	Version   = ""
	CommitSHA = ""

	readmeNames  = []string{"README.md", "README"}
	pager        bool
	style        string
	width        uint
	showAllFiles bool
	localOnly    bool

	rootCmd = &cobra.Command{
		Use:              "glow SOURCE",
		Short:            "Render markdown on the CLI, with pizzazz!",
		Long:             formatBlock(fmt.Sprintf("\nRender markdown on the CLI, %s!", common.Keyword("with pizzazz"))),
		SilenceErrors:    false,
		SilenceUsage:     false,
		TraverseChildren: true,
		RunE:             execute,
	}
)

// source provides a readable markdown source.
type source struct {
	reader io.ReadCloser
	URL    string
}

// sourceFromArg parses an argument and creates a readable source for it.
func sourceFromArg(arg string) (*source, error) {
	// from stdin
	if arg == "-" {
		return &source{reader: os.Stdin}, nil
	}

	// a GitHub or GitLab URL (even without the protocol):
	if u, ok := isGitHubURL(arg); ok {
		src, err := findGitHubREADME(u)
		if err != nil {
			return nil, err
		}
		return src, nil
	}
	if u, ok := isGitLabURL(arg); ok {
		src, err := findGitLabREADME(u)
		if err != nil {
			return nil, err
		}
		return src, nil
	}

	// HTTP(S) URLs:
	if u, err := url.ParseRequestURI(arg); err == nil && strings.Contains(arg, "://") {
		if u.Scheme != "" {
			if u.Scheme != "http" && u.Scheme != "https" {
				return nil, fmt.Errorf("%s is not a supported protocol", u.Scheme)
			}

			resp, err := http.Get(u.String())
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
			}
			return &source{resp.Body, u.String()}, nil
		}
	}

	// a directory:
	if len(arg) == 0 {
		// use the current working dir if no argument was supplied
		arg = "."
	}
	st, err := os.Stat(arg)
	if err == nil && st.IsDir() {
		var src *source
		_ = filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			for _, v := range readmeNames {
				if strings.EqualFold(filepath.Base(path), v) {
					r, err := os.Open(path)
					if err != nil {
						continue
					}

					u, _ := filepath.Abs(path)
					src = &source{r, u}

					// abort filepath.Walk
					return errors.New("source found")
				}
			}
			return nil
		})

		if src != nil {
			return src, nil
		}

		return nil, errors.New("missing markdown source")
	}

	// a file:
	r, err := os.Open(arg)
	u, _ := filepath.Abs(arg)
	return &source{r, u}, err
}

func validateOptions(cmd *cobra.Command) {
	isTerminal := terminal.IsTerminal(int(os.Stdout.Fd()))
	// We want to use a special no-TTY style, when stdout is not a terminal
	// and there was no specific style passed by arg
	if !isTerminal && !cmd.Flags().Changed("style") {
		style = "notty"
	}

	// Detect terminal width
	if isTerminal && !cmd.Flags().Changed("width") {
		w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			width = uint(w)
		}
	}
	if width == 0 {
		width = 80
	}
	if width > 120 {
		width = 120
	}
}

func execute(cmd *cobra.Command, args []string) error {
	validateOptions(cmd)

	if len(args) == 0 {
		return executeArg(cmd, "", os.Stdout)
	}

	for _, arg := range args {
		if err := executeArg(cmd, arg, os.Stdout); err != nil {
			return err
		}
	}
	return nil
}

func executeArg(cmd *cobra.Command, arg string, w io.Writer) error {

	// Only run TUI if there are no arguments (excluding flags)
	if arg == "" {
		return runTUI(false)
	}

	// create an io.Reader from the markdown source in cli-args
	src, err := sourceFromArg(arg)
	if err != nil {
		return err
	}
	defer src.reader.Close()
	b, err := ioutil.ReadAll(src.reader)
	if err != nil {
		return err
	}

	// render
	var baseURL string
	u, err := url.ParseRequestURI(src.URL)
	if err == nil {
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String() + "/"
	}

	// initialize glamour
	var gs glamour.TermRendererOption
	if style == "auto" {
		gs = glamour.WithEnvironmentConfig()
	} else {
		gs = glamour.WithStylePath(style)
	}

	r, err := glamour.NewTermRenderer(
		gs,
		glamour.WithWordWrap(int(width)),
		glamour.WithBaseURL(baseURL),
	)
	if err != nil {
		return err
	}

	out, err := r.RenderBytes(b)
	if err != nil {
		return err
	}

	// trim lines
	lines := strings.Split(string(out), "\n")
	var content string
	for i, s := range lines {
		content += strings.TrimSpace(s)

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content += "\n"
		}
	}

	// display
	if cmd.Flags().Changed("pager") {
		pager := os.Getenv("PAGER")
		if pager == "" {
			pager = "less -r"
		}

		pa := strings.Split(pager, " ")
		c := exec.Command(pa[0], pa[1:]...)
		c.Stdin = strings.NewReader(content)
		c.Stdout = os.Stdout
		return c.Run()
	}

	fmt.Fprint(w, content)
	return nil
}

func runTUI(stashedOnly bool) error {
	// Read environment to get debugging stuff
	var cfg ui.Config
	if err := babyenv.Parse(&cfg); err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	// Log to file, if set
	if cfg.Logfile != "" {
		f, err := tea.LogToFile(cfg.Logfile, "glow")
		if err != nil {
			return err
		}
		defer f.Close()
	}

	cfg.ShowAllFiles = showAllFiles

	if stashedOnly {
		cfg.DocumentTypes = ui.StashedDocument | ui.NewsDocument
	} else if localOnly {
		cfg.DocumentTypes = ui.LocalDocument
	}

	// Run Bubble Tea program
	p := ui.NewProgram(style, cfg)
	p.EnterAltScreen()
	if err := p.Start(); err != nil {
		return err
	}
	p.ExitAltScreen()
	p.DisableMouseCellMotion()

	// Exit message
	fmt.Printf("\n  Thanks for using Glow!\n\n")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version

	// "Glow Classic" cli arguments
	rootCmd.Flags().BoolVarP(&pager, "pager", "p", false, "display with pager")
	rootCmd.Flags().StringVarP(&style, "style", "s", "auto", "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 0, "word-wrap at width")
	rootCmd.Flags().BoolVarP(&showAllFiles, "all", "a", false, "show system files and directories (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&localOnly, "local", "l", false, "show local files only; no network (TUI-mode only)")

	// Stash
	stashCmd.PersistentFlags().StringVarP(&memo, "memo", "m", "", "memo/note for stashing")
	rootCmd.AddCommand(stashCmd)
}
