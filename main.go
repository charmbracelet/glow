// Package main provides the entry point for the Glow CLI application.
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/glow/v2/ui"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	// Version as provided by goreleaser.
	Version = ""
	// CommitSHA as provided by goreleaser.
	CommitSHA = ""

	readmeNames      = []string{"README.md", "README", "Readme.md", "Readme", "readme.md", "readme"}
	configFile       string
	pager            bool
	tui              bool
	style            string
	width            uint
	showAllFiles     bool
	showLineNumbers  bool
	preserveNewLines bool
	mouse            bool
	zenMode          bool
	zenWidth         uint
	zenMarginPercent uint

	rootCmd = &cobra.Command{
		Use:   "glow [SOURCE|DIR]",
		Short: "Render markdown on the CLI, with pizzazz!",
		Long: paragraph(
			fmt.Sprintf("\nRender markdown on the CLI, %s!", keyword("with pizzazz")),
		),
		SilenceErrors:    false,
		SilenceUsage:     true,
		TraverseChildren: true,
		RunE:             execute,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return validateOptions(cmd)
		},
	}
)

// source provides a readable source for markdown content.
type source struct {
	reader io.ReadCloser
	URL    string
}

func sourceFromArg(arg string) (*source, error) {
	// from stdin
	if arg == "-" {
		return &source{reader: os.Stdin}, nil
	}

	// a GitHub or GitLab URL (even without the protocol):
	src, err := readmeURL(arg)
	if src != nil && err == nil {
		// if there's an error, try next methods...
		return src, nil
	}

	// HTTP(S) URLs:
	if u, err := url.ParseRequestURI(arg); err == nil && strings.Contains(arg, "://") { //nolint:nestif
		if u.Scheme != "" {
			if u.Scheme != "http" && u.Scheme != "https" {
				return nil, fmt.Errorf("%s is not a supported protocol", u.Scheme)
			}
			// consumer of the source is responsible for closing the ReadCloser.
			resp, err := http.Get(u.String()) //nolint: noctx,bodyclose
			if err != nil {
				return nil, fmt.Errorf("unable to get url: %w", err)
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
	if err == nil && st.IsDir() { //nolint:nestif
		var src *source
		_ = filepath.Walk(arg, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
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

	// a regular file
	r, err := os.Open(arg)
	if err != nil {
		return nil, err
	}

	u, _ := filepath.Abs(arg)
	return &source{r, u}, nil
}

// validateStyle checks if the style is a default style, if not, checks that
// the custom style exists.
func validateStyle(style string) error {
	if style != "auto" && styles.DefaultStyles[style] == nil {
		style = utils.ExpandPath(style)
		if _, err := os.Stat(style); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("specified style does not exist: %s", style)
		} else if err != nil {
			return fmt.Errorf("unable to stat file: %w", err)
		}
	}
	return nil
}

func validateOptions(cmd *cobra.Command) error {
	// grab config values from Viper
	width = viper.GetUint("width")
	mouse = viper.GetBool("mouse")
	pager = viper.GetBool("pager")
	tui = viper.GetBool("tui")
	showAllFiles = viper.GetBool("all")
	preserveNewLines = viper.GetBool("preserveNewLines")
	zenMode = viper.GetBool("zenMode")
	zenWidth = viper.GetUint("zenWidth")
	zenMarginPercent = viper.GetUint("zenMarginPercent")

	// Apply zen-mode width settings (after all viper values are read)
	if zenMode {
		// If zen-width is explicitly set, use it (overrides everything)
		if cmd.Flags().Changed("zen-width") && zenWidth > 0 {
			width = zenWidth
		}
		// Otherwise, zen-mode just uses whatever width was detected/set
		// The magic is in the margin percentage applied to that width
	}

	if pager && tui {
		return errors.New("cannot use both pager and tui")
	}

	// validate the glamour style
	style = viper.GetString("style")
	if err := validateStyle(style); err != nil {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	// We want to use a special no-TTY style, when stdout is not a terminal
	// and there was no specific style passed by arg
	if !isTerminal && !cmd.Flags().Changed("style") {
		style = "notty"
	}

	// Detect terminal width
	if !cmd.Flags().Changed("width") { //nolint:nestif
		if isTerminal && width == 0 {
			w, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				width = uint(w) //nolint:gosec
			}

			if width > 120 {
				width = 120  // Standard cap for non-zen mode (zen-mode handled later)
			}
		}
		if width == 0 {
			width = 80
		}
	}
	return nil
}

func stdinIsPipe() (bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false, fmt.Errorf("unable to open file: %w", err)
	}
	if stat.Mode()&os.ModeCharDevice == 0 || stat.Size() > 0 {
		return true, nil
	}
	return false, nil
}

func execute(cmd *cobra.Command, args []string) error {
	// if stdin is a pipe then use stdin for input. note that you can also
	// explicitly use a - to read from stdin.
	if yes, err := stdinIsPipe(); err != nil {
		return err
	} else if yes {
		src := &source{reader: os.Stdin}
		defer src.reader.Close() //nolint:errcheck
		return executeCLI(cmd, src, os.Stdout)
	}

	switch len(args) {
	// TUI running on cwd
	case 0:
		return runTUI("", "")

	// TUI with possible dir argument
	case 1:
		// Validate that the argument is a directory. If it's not treat it as
		// markdown source.
		if isDir(args[0]) {
			return runTUI(args[0], "")
		}
		return executeArg(cmd, args[0], os.Stdout)

	// Execute file
	default:
		return executeArg(cmd, args[0], os.Stdout)
	}
}

func executeArg(cmd *cobra.Command, arg string, w io.Writer) error {
	src, err := sourceFromArg(arg)
	if err != nil {
		return err
	}
	defer src.reader.Close() //nolint:errcheck

	return executeCLI(cmd, src, w)
}

func executeCLI(cmd *cobra.Command, src *source, w io.Writer) error {
	b, err := io.ReadAll(src.reader)
	if err != nil {
		return fmt.Errorf("unable to read source: %w", err)
	}

	// render
	var baseURL string
	u, err := url.ParseRequestURI(src.URL)
	if err == nil && u.IsAbs() {
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String() + "/"
	} else {
		if err == nil {
			u.Path = filepath.Dir(u.Path)
			baseURL = u.String() + "/"
		}
	}

	isCode := !utils.IsMarkdownFile(src.URL)

	// initialize glamour
	glamourOptions := []glamour.TermRendererOption{
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(style, isCode),
		glamour.WithWordWrap(int(width)), //nolint:gosec
		glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	}
	
	// Handle zen-mode margins
	if zenMode {
		// Zen mode: centered column with clean margins (like VSCode/Neovim zen-mode)
		// Use configurable margin percentage for comfortable zen reading
		autoMargin := width * zenMarginPercent / 100
		if autoMargin < 10 {
			autoMargin = 10  // Minimum margin for readability
		}
		if autoMargin > 50 {
			autoMargin = 50  // Cap for very large margins
		}
		glamourOptions = append(glamourOptions, glamour.WithZenMode(autoMargin))
	}

	r, err := glamour.NewTermRenderer(glamourOptions...)
	if err != nil {
		return fmt.Errorf("unable to create renderer: %w", err)
	}

	content := string(b)
	ext := filepath.Ext(src.URL)
	if isCode {
		content = utils.WrapCodeBlock(string(b), ext)
	}

	out, err := r.Render(content)
	if err != nil {
		return fmt.Errorf("unable to render source: %w", err)
	}

	if pager {
		// Respect PAGER environment variable, fall back to less -r if not set
		pagerCmd := os.Getenv("PAGER")
		if pagerCmd == "" {
			pagerCmd = "less -r"
		}
		
		// Parse pager command to handle arguments
		parts := strings.Fields(pagerCmd)
		var cmd *exec.Cmd
		if len(parts) > 1 {
			cmd = exec.Command(parts[0], parts[1:]...)
		} else {
			cmd = exec.Command(parts[0])
		}
		
		cmd.Stdin = strings.NewReader(out)
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(w, "%s", out)
		}
		return nil
	}

	fmt.Fprintf(w, "%s", out)
	return nil
}

func isDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}
	return f.IsDir()
}

func runTUI(path string, content string) error {
	cfg := ui.Config{}
	if err := env.ParseWithOptions(&cfg, env.Options{
		Prefix: "GLOW_",
	}); err != nil {
		return fmt.Errorf("unable to parse environment: %w", err)
	}

	cfg.Path = path
	cfg.ShowAllFiles = showAllFiles
	cfg.ShowLineNumbers = showLineNumbers
	cfg.GlamourMaxWidth = width
	cfg.EnableMouse = mouse
	cfg.PreserveNewLines = preserveNewLines

	// Run Bubble Tea program
	if _, err := ui.NewProgram(cfg, content).Run(); err != nil {
		return fmt.Errorf("unable to run tui program: %w", err)
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func init() {
	// "Glow Classic" cli arguments
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (default %s)", viper.GetViper().ConfigFileUsed()))
	rootCmd.Flags().BoolVarP(&pager, "pager", "p", false, "display with pager")
	rootCmd.Flags().BoolVarP(&tui, "tui", "t", false, "display with tui")
	rootCmd.Flags().StringVarP(&style, "style", "s", styles.AutoStyle, "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 0, "word-wrap at width (set to 0 to disable)")
	rootCmd.Flags().BoolVarP(&showAllFiles, "all", "a", false, "show system files and directories (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&showLineNumbers, "line-numbers", "l", false, "show line numbers (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&preserveNewLines, "preserve-new-lines", "n", false, "preserve newlines in the output")
	rootCmd.Flags().BoolVarP(&mouse, "mouse", "m", false, "enable mouse wheel (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&zenMode, "zen", "z", false, "zen-mode reading with justified text and auto margins (overrides other alignment settings)")
	rootCmd.Flags().UintVar(&zenWidth, "zen-width", 0, "line width for zen-mode (0 = auto based on terminal)")
	rootCmd.Flags().UintVar(&zenMarginPercent, "zen-margin", 20, "margin percentage for zen-mode (e.g., 20 = 20% margins on each side)")
	_ = rootCmd.Flags().MarkHidden("mouse")

	// Config bindings
	_ = viper.BindPFlag("pager", rootCmd.Flags().Lookup("pager"))
	_ = viper.BindPFlag("tui", rootCmd.Flags().Lookup("tui"))
	_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
	_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
	_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
	_ = viper.BindPFlag("preserveNewLines", rootCmd.Flags().Lookup("preserve-new-lines"))
	_ = viper.BindPFlag("showLineNumbers", rootCmd.Flags().Lookup("line-numbers"))
	_ = viper.BindPFlag("all", rootCmd.Flags().Lookup("all"))
	_ = viper.BindPFlag("zenMode", rootCmd.Flags().Lookup("zen"))
	_ = viper.BindPFlag("zenWidth", rootCmd.Flags().Lookup("zen-width"))
	_ = viper.BindPFlag("zenMarginPercent", rootCmd.Flags().Lookup("zen-margin"))

	viper.SetDefault("style", styles.AutoStyle)
	viper.SetDefault("width", 0)
	viper.SetDefault("all", true)

	rootCmd.AddCommand(configCmd, manCmd)
}

func tryLoadConfigFromDefaultPlaces() {
	for _, v := range []func(){
		func() { viper.SetConfigName(".glow") },
		func() { viper.SetConfigName("glow") },
	} {
		v()
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		if appPath := getConfigPath(); appPath != "" {
			viper.AddConfigPath(appPath)
		}
		if err := viper.ReadInConfig(); err == nil {
			break
		}
	}
}

func getConfigPath() string {
	scope := gap.NewScope(gap.User, "glow")
	dirs, err := scope.ConfigDirs()
	if err != nil {
		log.Debug("unable to get config directory", "error", err)
		return ""
	}
	if len(dirs) > 0 {
		return dirs[0]
	}
	return ""
}

func initConfig() error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("unable to use config file: %w", err)
		}
	} else {
		tryLoadConfigFromDefaultPlaces()
	}
	return nil
}