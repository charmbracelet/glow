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

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/charmbracelet/glamour"
)

var (
	Version   = ""
	CommitSHA = ""

	readmeNames = []string{"README.md", "README"}
	pager       bool
	style       string
	width       uint
	walk		bool

	rootCmd = &cobra.Command{
		Use:           "glow SOURCE",
		Short:         "Render markdown on the CLI, with pizzazz!",
		SilenceErrors: false,
		SilenceUsage:  false,
		RunE:          execute,
	}
)

type Source struct {
	reader io.ReadCloser
	URL    string
}

func readerFromArg(cmd *cobra.Command, s string) (*Source, error) {
	if s == "-" {
		return &Source{reader: os.Stdin}, nil
	}

	// a GitHub or GitLab URL (even without the protocol):
	if u, ok := isGitHubURL(s); ok {
		src, err := findGitHubREADME(u)
		if err != nil {
			return nil, err
		}
		return src, nil
	}
	if u, ok := isGitLabURL(s); ok {
		src, err := findGitLabREADME(u)
		if err != nil {
			return nil, err
		}
		return src, nil
	}

	// HTTP(S) URLs:
	if u, err := url.ParseRequestURI(s); err == nil {
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
			return &Source{resp.Body, u.String()}, nil
		}
	}

	// a valid file or directory:
	st, err := os.Stat(s)
	if len(s) == 0 || (err == nil && st.IsDir()) {
		for _, v := range readmeNames {
			n := filepath.Join(s, v)
			r, err := os.Open(n)
			if err == nil {
				u, _ := filepath.Abs(n)
				return &Source{r, u}, nil
			}
		}

		// walk directory
		if cmd.Flags().Changed("walk") {
			filepath.Walk(s,
				func(path string, info os.FileInfo, err error) error {
				if strings.Contains(path, "README") {
					fmt.Println(path)
				}

				return nil
			})

			fmt.Println("Info: possible markdown sources")
		}

		return nil, errors.New("missing markdown source in current directory")
	}

	r, err := os.Open(s)
	u, _ := filepath.Abs(s)
	return &Source{r, u}, err
}

func execute(cmd *cobra.Command, args []string) error {
	var arg string
	if len(args) > 0 {
		arg = args[0]
	}

	// create an io.Reader from the markdown source in cli-args
	src, err := readerFromArg(cmd, arg)
	if err != nil {
		return err
	}
	defer src.reader.Close()
	b, err := ioutil.ReadAll(src.reader)
	if err != nil {
		return err
	}

	// We want to use a special no-TTY style, when stdout is not a terminal
	// and there was no specific style passed by arg
	if !isatty.IsTerminal(os.Stdout.Fd()) &&
		!cmd.Flags().Changed("style") {
		style = "notty"
	}

	// render
	var baseURL string
	u, err := url.ParseRequestURI(src.URL)
	if err == nil {
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String() + "/"
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
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

	fmt.Print(content)
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

	rootCmd.Flags().BoolVarP(&pager, "pager", "p", false, "display with pager")
	rootCmd.Flags().StringVarP(&style, "style", "s", "dark", "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 100, "word-wrap at width")
	rootCmd.Flags().BoolVarP(&walk, "walk", "k", false, "find README file in directory")
}
