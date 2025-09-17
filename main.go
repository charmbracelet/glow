// Package main provides the entry point for the Glow CLI application.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/glow/v2/flow"
	"github.com/charmbracelet/glow/v2/ui"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	// ExitCodeTimeout is the exit code for timeout (used by timeout command)
	ExitCodeTimeout = 124

	// ExitCodeSIGPIPE is the exit code for SIGPIPE (when downstream closes)
	ExitCodeSIGPIPE = 141

	// ExitCodeSIGINT is the signal offset for SIGINT (Ctrl+C)
	ExitCodeSIGINT = 128 + 2

	// ExitCodeSIGTERM is the signal offset for SIGTERM
	ExitCodeSIGTERM = 128 + 15
)

// ExitError represents an error that should cause the program to exit with a specific code
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit with code %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

var (
	// Version as provided by goreleaser.
	Version = ""
	// CommitSHA as provided by goreleaser.
	CommitSHA = ""

	readmeNames      = []string{"README.md", "README", "Readme.md", "Readme", "readme.md", "readme"}
	configFile       string
	pager            bool
	tui              bool
	window           int64
	style            string
	width            uint
	showAllFiles     bool
	showLineNumbers  bool
	preserveNewLines bool
	mouse            bool

	rootCmd = &cobra.Command{
		Use:   "glow [SOURCE|DIR]",
		Short: "Render markdown on the CLI, with pizzazz!",
		Long: paragraph(
			fmt.Sprintf("\nRender markdown on the CLI, %s!", keyword("with pizzazz")),
		),
		SilenceErrors:    false,
		SilenceUsage:     true,
		TraverseChildren: true,
		Args:             cobra.MaximumNArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return validateOptions(cmd)
		},
		RunE: execute,
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

	r, err := os.Open(arg)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	u, err := filepath.Abs(arg)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path: %w", err)
	}
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
	window = viper.GetInt64("flow")
	showAllFiles = viper.GetBool("all")
	preserveNewLines = viper.GetBool("preserveNewLines")
	showLineNumbers = viper.GetBool("showLineNumbers")

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
		// DO NOT close stdin - it's not ours to close and causes race conditions
		return executeCLI(cmd, src, os.Stdout)
	}

	switch len(args) {
	// TUI running on cwd
	case 0:
		if cmd.Flags().Changed("flow") {
			// --flow flag was explicitly provided, incompatible with directories
			return fmt.Errorf("--flow is not compatible with directory input")
		}
		return runTUI("", "")

	// TUI with possible dir argument
	case 1:
		// Validate that the argument is a directory. If it's not treat it as
		// an argument to the non-TUI version of Glow (via fallthrough).
		info, err := os.Stat(args[0])
		if err == nil && info.IsDir() {
			// Directory input detected
			if cmd.Flags().Changed("flow") {
				// --flow flag was explicitly provided, incompatible with directories
				return fmt.Errorf("--flow is not compatible with directory input")
			}
			// No --flow specified, launch TUI for directory
			p, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("could not resolve directory path: %w", err)
			}
			return runTUI(p, "")
		}
		// Not a directory, treat as regular file argument
		fallthrough

	// CLI
	default:
		for _, arg := range args {
			if err := executeArg(cmd, arg, os.Stdout); err != nil {
				return err
			}
		}
	}

	return nil
}

func executeArg(cmd *cobra.Command, arg string, w io.Writer) error {
	// create an io.Reader from the markdown source in cli-args
	src, err := sourceFromArg(arg)
	if err != nil {
		return err
	}
	// Only close if it's not stdin (stdin is not ours to close)
	if src.reader != os.Stdin {
		defer src.reader.Close() //nolint:errcheck
	}
	return executeCLI(cmd, src, w)
}

func executeCLI(cmd *cobra.Command, src *source, w io.Writer) error {

	// render
	var baseURL string
	u, err := url.ParseRequestURI(src.URL)
	if err == nil {
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String() + "/"
	}

	isCode := !utils.IsMarkdownFile(src.URL)

	// initialize glamour
	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(style, isCode),
		glamour.WithWordWrap(int(width)), //nolint:gosec
		glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return fmt.Errorf("unable to create renderer: %w", err)
	}

	// display
	switch {
	case pager || cmd.Flags().Changed("pager"):
		pagerCmd := os.Getenv("PAGER")
		if pagerCmd == "" {
			pagerCmd = "less -r"
		}

		pa := strings.Split(pagerCmd, " ")
		c := exec.CommandContext(cmd.Context(), pa[0], pa[1:]...) //nolint:gosec
		pw, err := c.StdinPipe()
		if err != nil {
			return fmt.Errorf("unable to create stdin pipe: %w", err)
		}
		c.Stdout = w
		c.Stderr = os.Stderr

		// Start the pager process
		if err := c.Start(); err != nil {
			pw.Close() //nolint:errcheck
			return fmt.Errorf("unable to start pager: %w", err)
		}

		// Create cancellable context for pager operations
		pagerCtx, pagerCancel := context.WithCancel(cmd.Context())
		defer pagerCancel()

		// Launch Flow goroutine with cancellable context
		flowDone := make(chan error, 1)
		go func() {
			defer pw.Close() // CRITICAL: Always close pipe when Flow exits
			flowDone <- flow.Flow(pagerCtx, src.reader, pw, window, r.RenderBytes)
		}()

		// Launch pager monitor goroutine
		pagerDone := make(chan error, 1)
		go func() {
			err := c.Wait()
			pagerCancel() // KEY: Cancel Flow when pager exits
			pagerDone <- err
		}()

		// Select on first completion - pager exit wins, cancels context
		select {
		case flowErr := <-flowDone:
			// Flow completed first - pipe already closed via defer, wait for pager
			pagerErr := <-pagerDone
			if pagerErr != nil && flowErr != nil {
				return fmt.Errorf("flow error: %w, pager error: %w", flowErr, pagerErr)
			}
			if pagerErr != nil {
				return fmt.Errorf("pager error: %w", pagerErr)
			}
			// Handle EPIPE in non-pager mode too (e.g., when piped to head)
			if flowErr != nil && errors.Is(flowErr, syscall.EPIPE) {
				// EPIPE is expected when downstream closes pipe
				flowErr = nil
			}
			return flowErr

		case pagerErr := <-pagerDone:
			// Pager exited first - context already cancelled in goroutine
			// Wait for Flow to finish (will exit quickly due to context cancellation)
			flowErr := <-flowDone

			// Handle expected flow errors
			if flowErr != nil {
				if errors.Is(flowErr, context.Canceled) {
					// Context cancellation is expected when pager exits
					flowErr = nil
				} else if errors.Is(flowErr, syscall.EPIPE) {
					// EPIPE is expected when pager closes pipe
					flowErr = nil
				}
			}

			// Check if pager error is actually a normal termination
			if pagerErr != nil {
				// Check for expected exit codes that aren't really errors
				if exitErr, ok := pagerErr.(*exec.ExitError); ok {
					exitCode := exitErr.ExitCode()
					if exitCode == ExitCodeTimeout {
						// Timeout is normal for streaming when pager exits early
						// Propagate the exit code by returning ExitError
						return &ExitError{Code: ExitCodeTimeout, Err: flowErr}
					}
					if exitCode == ExitCodeSIGPIPE {
						// SIGPIPE is normal when pager closes early
						return flowErr
					}
					// For other exit codes, propagate them directly
					// This includes SIGTERM (143), SIGINT (130), etc.
					return &ExitError{Code: exitCode, Err: flowErr}
				}
				return fmt.Errorf("pager error: %w", pagerErr)
			}
			return flowErr
		}
	case tui || cmd.Flags().Changed("tui"):
		// Read all content for TUI
		b, err := io.ReadAll(src.reader)
		if err != nil {
			return fmt.Errorf("unable to read markdown: %w", err)
		}

		b = utils.RemoveFrontmatter(b)
		content := string(b)
		ext := filepath.Ext(src.URL)
		if isCode {
			content = utils.WrapCodeBlock(string(b), ext)
		}

		path := ""
		if !isURL(src.URL) {
			path = src.URL
		}
		return runTUI(path, content)
	default:
		// Stream directly to writer
		if err := flow.Flow(cmd.Context(), src.reader, w, window, r.RenderBytes); err != nil {
			// Handle EPIPE gracefully (e.g., when piped to head)
			if errors.Is(err, syscall.EPIPE) {
				return nil
			}
			return fmt.Errorf("unable to flow markdown: %w", err)
		}
		return nil
	}
}

func runTUI(path string, content string) error {
	// Read environment to get debugging stuff
	cfg, err := env.ParseAs[ui.Config]()
	if err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	// use style set in env, or auto if unset
	if err := validateStyle(cfg.GlamourStyle); err != nil {
		cfg.GlamourStyle = style
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
	// CRITICAL: Signal handling MUST be set up FIRST before any defers
	// This ensures all cleanup happens properly when signals are received
	var sig os.Signal
	var err error

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Ignore SIGPIPE to prevent process termination with exit 141
	// Instead we'll detect EPIPE errors in write operations
	signal.Ignore(syscall.SIGPIPE)

	// Set up signal notification channel for other signals
	notify := make(chan os.Signal, 1)
	signal.Notify(notify, syscall.SIGTERM, syscall.SIGINT)
	// Note: SIGPIPE is ignored, not handled as a signal

	// Start signal handler goroutine BEFORE any defers
	go func() {
		for {
			select {
			case s := <-notify:
				sig = s
				cancel()
				// Let the defer handle the exit code
				// NO os.Exit() here - avoid double exit!
				return
			case <-ctx.Done():
				// Context cancelled, exit goroutine
				return
			}
		}
	}()

	// NOW set up defers - they will run properly even if signals are received
	defer func() {
		// Clean up signal handling
		signal.Stop(notify)
		cancel()

		// Convert signals to exit codes through error bubbling
		if sig != nil {
			switch sig {
			case syscall.SIGINT:
				if err == nil {
					err = &ExitError{Code: ExitCodeSIGINT, Err: errors.New("interrupted")}
				}
			case syscall.SIGTERM:
				if err == nil {
					err = &ExitError{Code: ExitCodeSIGTERM, Err: errors.New("terminated")}
				}
			}
		}

		// Unified exit point with proper error code
		if err != nil {
			var exitErr *ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.Code)
			} else if errors.Is(err, syscall.EPIPE) {
				// EPIPE is normal when downstream closes pipe - exit cleanly
				os.Exit(0)
			} else if errors.Is(err, context.Canceled) {
				// Context cancellation should exit cleanly
				if sig == nil {
					os.Exit(0) // Clean cancellation without signal
				}
			} else {
				os.Exit(1)
			}
		}
	}()

	// Set up logging
	closer, err := setupLog()
	if err != nil {
		// Deferred func handles exit code
		fmt.Println(err)
		return
	}
	defer closer()

	// Execute root command with context
	err = rootCmd.ExecuteContext(ctx)
}

func init() {
	tryLoadConfigFromDefaultPlaces()
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version
	rootCmd.InitDefaultCompletionCmd()

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
	rootCmd.Flags().Int64VarP(&window, "flow", "f", flow.Buffered, fmt.Sprintf("flow window size (-1: block, 0: stream, 1-%d: bytes)", flow.Windowed))
	rootCmd.Flags().Lookup("flow").NoOptDefVal = fmt.Sprintf("%d", flow.Unbuffered)
	_ = rootCmd.Flags().MarkHidden("mouse")

	// Config bindings
	_ = viper.BindPFlag("pager", rootCmd.Flags().Lookup("pager"))
	_ = viper.BindPFlag("tui", rootCmd.Flags().Lookup("tui"))
	_ = viper.BindPFlag("flow", rootCmd.Flags().Lookup("flow"))
	_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
	_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
	_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
	_ = viper.BindPFlag("preserveNewLines", rootCmd.Flags().Lookup("preserve-new-lines"))
	_ = viper.BindPFlag("showLineNumbers", rootCmd.Flags().Lookup("line-numbers"))
	_ = viper.BindPFlag("all", rootCmd.Flags().Lookup("all"))

	viper.SetDefault("style", styles.AutoStyle)
	viper.SetDefault("flow", flow.Buffered)
	viper.SetDefault("width", 0)
	viper.SetDefault("all", true)

	rootCmd.AddCommand(configCmd, manCmd)
}

func tryLoadConfigFromDefaultPlaces() {
	scope := gap.NewScope(gap.User, "glow")
	dirs, err := scope.ConfigDirs()
	if err != nil {
		// Log error but don't exit - config is optional
		fmt.Fprintf(os.Stderr, "Warning: Could not load configuration directory: %v\n", err)
		return
	}

	if c := os.Getenv("XDG_CONFIG_HOME"); c != "" {
		dirs = append([]string{filepath.Join(c, "glow")}, dirs...)
	}

	if c := os.Getenv("GLOW_CONFIG_HOME"); c != "" {
		dirs = append([]string{c}, dirs...)
	}

	for _, v := range dirs {
		viper.AddConfigPath(v)
	}

	viper.SetConfigName("glow")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("glow")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Warn("Could not parse configuration file", "err", err)
		}
	}

	if used := viper.ConfigFileUsed(); used != "" {
		log.Debug("Using configuration file", "path", viper.ConfigFileUsed())
		return
	}

	if viper.ConfigFileUsed() == "" {
		configFile = filepath.Join(dirs[0], "glow.yml")
	}
	if err := ensureConfigFile(); err != nil {
		log.Error("Could not create default configuration", "error", err)
	}
}
