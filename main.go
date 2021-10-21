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
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/ui"
	"github.com/charmbracelet/glow/utils"
)

var (
	Version   = ""
	CommitSHA = ""

	readmeNames  = []string{"README.md", "README"}
	configFile   string
	pager        bool
	style        string
	width        uint
	showAllFiles bool
	localOnly    bool
	mouse        bool

	rootCmd = &cobra.Command{
		Use:              "glow [SOURCE|DIR]",
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

func validateOptions(cmd *cobra.Command) error {
	// grab config values from Viper
	width = viper.GetUint("width")
	localOnly = viper.GetBool("local")
	mouse = viper.GetBool("mouse")
	pager = viper.GetBool("pager")

	// validate the glamour style
	style = viper.GetString("style")
	if style != "auto" && glamour.DefaultStyles[style] == nil {
		style = utils.ExpandPath(style)
		if _, err := os.Stat(style); os.IsNotExist(err) {
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
	initConfig()
	if err := validateOptions(cmd); err != nil {
		return err
	}

	// if stdin is a pipe then use stdin for input. note that you can also
	// explicitly use a - to read from stdin.
	if yes, err := stdinIsPipe(); err != nil {
		return err
	} else if yes {
		src := &source{reader: os.Stdin}
		defer src.reader.Close()
		return executeCLI(cmd, src, os.Stdout)
	}

	switch len(args) {

	// TUI running on cwd
	case 0:
		return runTUI("", false)

	// TUI with possible dir argument
	case 1:
		// Validate that the argument is a directory. If it's not treat it as
		// an argument to the non-TUI version of Glow (via fallthrough).
		info, err := os.Stat(args[0])
		if err == nil && info.IsDir() {
			p, err := filepath.Abs(args[0])
			if err == nil {
				return runTUI(p, false)
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
	defer src.reader.Close()
	return executeCLI(cmd, src, w)
}

func executeCLI(cmd *cobra.Command, src *source, w io.Writer) error {
	b, err := ioutil.ReadAll(src.reader)
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
	if pager || cmd.Flags().Changed("pager") {
		pagerCmd := os.Getenv("PAGER")
		if pagerCmd == "" {
			pagerCmd = "less -r"
		}

		pa := strings.Split(pagerCmd, " ")
		c := exec.Command(pa[0], pa[1:]...)
		c.Stdin = strings.NewReader(content)
		c.Stdout = os.Stdout
		return c.Run()
	}

	fmt.Fprint(w, content)
	return nil
}

func runTUI(workingDirectory string, stashedOnly bool) error {
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

	cfg.WorkingDirectory = workingDirectory
	cfg.DocumentTypes = ui.NewDocTypeSet()
	cfg.ShowAllFiles = showAllFiles
	cfg.GlamourMaxWidth = width
	cfg.GlamourStyle = style

	if stashedOnly {
		cfg.DocumentTypes.Add(ui.StashedDoc, ui.NewsDoc)
	} else if localOnly {
		cfg.DocumentTypes.Add(ui.LocalDoc)
	}

	// Run Bubble Tea program
	p := ui.NewProgram(cfg)
	p.EnterAltScreen()
	defer p.ExitAltScreen()
	if mouse {
		p.EnableMouseCellMotion()
		defer p.DisableMouseCellMotion()
	}
	if err := p.Start(); err != nil {
		return err
	}

	// Exit message
	fmt.Printf("\n  Thanks for using Glow!\n\n")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
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

	scope := gap.NewScope(gap.User, "glow")
	defaultConfigFile, _ := scope.ConfigPath("glow.yml")

	// "Glow Classic" cli arguments
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (default %s)", defaultConfigFile))
	rootCmd.Flags().BoolVarP(&pager, "pager", "p", false, "display with pager")
	rootCmd.Flags().StringVarP(&style, "style", "s", "auto", "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 0, "word-wrap at width")
	rootCmd.Flags().BoolVarP(&showAllFiles, "all", "a", false, "show system files and directories (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&localOnly, "local", "l", false, "show local files only; no network (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&mouse, "mouse", "m", false, "enable mouse wheel (TUI-mode only)")
	rootCmd.Flags().MarkHidden("mouse")

	// Config bindings
	_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
	_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
	_ = viper.BindPFlag("local", rootCmd.Flags().Lookup("local"))
	_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
	viper.SetDefault("style", "auto")
	viper.SetDefault("width", 0)
	viper.SetDefault("local", "false")

	// Stash
	stashCmd.PersistentFlags().StringVarP(&memo, "memo", "m", "", "memo/note for stashing")
	rootCmd.AddCommand(stashCmd)

	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		scope := gap.NewScope(gap.User, "glow")
		dirs, err := scope.ConfigDirs()
		if err != nil {
			fmt.Println("Can't retrieve default config. Please manually pass a config file with '--config'")
			os.Exit(1)
		}

		for _, v := range dirs {
			viper.AddConfigPath(v)
		}
		viper.SetConfigName("glow")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("glow")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Println("Error parsing config:", err)
			os.Exit(1)
		}
	}

	// fmt.Println("Using config file:", viper.ConfigFileUsed())
}
