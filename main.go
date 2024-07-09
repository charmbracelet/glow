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
	"github.com/charmbracelet/glow/ui"
	"github.com/charmbracelet/glow/utils"
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
	readmeBranches   = []string{"main", "master"}
	configFile       string
	pager            bool
	style            string
	width            uint
	showAllFiles     bool
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
	if src != nil || err != nil {
		return src, err
	}

	// HTTP(S) URLs:
	if u, err := url.ParseRequestURI(arg); err == nil && strings.Contains(arg, "://") {
		if u.Scheme != "" {
			if u.Scheme != "http" && u.Scheme != "https" {
				return nil, fmt.Errorf("%s is not a supported protocol", u.Scheme)
			}
			// consumer of the source is responsible for closing the ReadCloser.
			resp, err := http.Get(u.String()) // nolint:bodyclose
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

	// a file:
	r, err := os.Open(arg)
	u, _ := filepath.Abs(arg)
	return &source{r, u}, err
}

func validateOptions(cmd *cobra.Command) error {
	// grab config values from Viper
	width = viper.GetUint("width")
	mouse = viper.GetBool("mouse")
	pager = viper.GetBool("pager")
	preserveNewLines = viper.GetBool("preserveNewLines")

	// validate the glamour style
	style = viper.GetString("style")
	if style != glamour.AutoStyle && glamour.DefaultStyles[style] == nil {
		style = utils.ExpandPath(style)
		if _, err := os.Stat(style); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("Specified style does not exist: %s", style)
		} else if err != nil {
			return err
		}
	}

	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	// We want to use a special no-TTY style, when stdout is not a terminal
	// and there was no specific style passed by arg
	if !isTerminal && !cmd.Flags().Changed("style") {
		style = "notty"
	}

	// Detect terminal width
	if isTerminal && width == 0 && !cmd.Flags().Changed("width") {
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			width = uint(w)
		}

		if width > 120 {
			width = 120
		}
	}
	if width == 0 {
		width = 80
	}
	return nil
}

func stdinIsPipe() (bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false, err
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
		return runTUI("")

	// TUI with possible dir argument
	case 1:
		// Validate that the argument is a directory. If it's not treat it as
		// an argument to the non-TUI version of Glow (via fallthrough).
		info, err := os.Stat(args[0])
		if err == nil && info.IsDir() {
			p, err := filepath.Abs(args[0])
			if err == nil {
				return runTUI(p)
			}
		}
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
	defer src.reader.Close() //nolint:errcheck
	return executeCLI(cmd, src, w)
}

func executeCLI(cmd *cobra.Command, src *source, w io.Writer) error {
	b, err := io.ReadAll(src.reader)
	if err != nil {
		return err
	}

	b = utils.RemoveFrontmatter(b)

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
		glamour.WithWordWrap(int(width)),
		glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return err
	}

	s := string(b)
	ext := filepath.Ext(src.URL)
	if isCode {
		s = utils.WrapCodeBlock(string(b), ext)
	}

	out, err := r.Render(s)
	if err != nil {
		return err
	}

	// trim lines
	lines := strings.Split(out, "\n")
	var content strings.Builder
	for i, s := range lines {
		content.WriteString(strings.TrimSpace(s))

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content.WriteByte('\n')
		}
	}

	// display
	if pager || cmd.Flags().Changed("pager") {
		pagerCmd := os.Getenv("PAGER")
		if pagerCmd == "" {
			pagerCmd = "less -r"
		}

		pa := strings.Split(pagerCmd, " ")
		c := exec.Command(pa[0], pa[1:]...) // nolint:gosec
		c.Stdin = strings.NewReader(content.String())
		c.Stdout = os.Stdout
		return c.Run()
	}

	fmt.Fprint(w, content.String()) //nolint: errcheck
	return nil
}

func runTUI(workingDirectory string) error {
	// Read environment to get debugging stuff
	cfg, err := env.ParseAs[ui.Config]()
	if err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	cfg.WorkingDirectory = workingDirectory

	cfg.ShowAllFiles = showAllFiles
	cfg.GlamourMaxWidth = width
	cfg.GlamourStyle = style
	cfg.EnableMouse = mouse
	cfg.PreserveNewLines = preserveNewLines

	// Run Bubble Tea program
	if _, err := ui.NewProgram(cfg).Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	closer, err := setupLog()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		_ = closer()
		os.Exit(1)
	}
	_ = closer()
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
	rootCmd.Flags().StringVarP(&style, "style", "s", glamour.AutoStyle, "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 0, "word-wrap at width")
	rootCmd.Flags().BoolVarP(&showAllFiles, "all", "a", false, "show system files and directories (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&preserveNewLines, "preserve-new-lines", "n", false, "preserve newlines in the output")
	rootCmd.Flags().BoolVarP(&mouse, "mouse", "m", false, "enable mouse wheel (TUI-mode only)")
	_ = rootCmd.Flags().MarkHidden("mouse")

	// Config bindings
	_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
	_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
	_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
	_ = viper.BindPFlag("preserveNewLines", rootCmd.Flags().Lookup("preserve-new-lines"))

	viper.SetDefault("style", glamour.AutoStyle)
	viper.SetDefault("width", 0)

	rootCmd.AddCommand(configCmd, manCmd)
}

func tryLoadConfigFromDefaultPlaces() {
	scope := gap.NewScope(gap.User, "glow")
	dirs, err := scope.ConfigDirs()
	if err != nil {
		fmt.Println("Could not load find configuration directory.")
		os.Exit(1)
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
