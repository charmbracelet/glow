package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cobra"

	"github.com/charmbracelet/gold"
	_ "github.com/charmbracelet/gold/cmd/gold/statik"
)

var (
	readmeNames = []string{"README.md", "README"}

	rootCmd = &cobra.Command{
		Use:           "gold SOURCE",
		Short:         "Render markdown on the CLI, with pizzazz!",
		SilenceErrors: false,
		SilenceUsage:  false,
		RunE:          execute,
	}

	style string
	width uint
)

type Source struct {
	reader io.ReadCloser
	URL    string
}

func readerFromArg(s string) (*Source, error) {
	if s == "-" {
		return &Source{reader: os.Stdin}, nil
	}

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

	if len(s) == 0 {
		for _, v := range readmeNames {
			r, err := os.Open(v)
			if err == nil {
				u, _ := filepath.Abs(v)
				return &Source{r, u}, nil
			}
		}

		return nil, errors.New("missing markdown source")
	}

	r, err := os.Open(s)
	u, _ := filepath.Abs(s)
	return &Source{r, u}, err
}

func loadStyle(f string) ([]byte, error) {
	var r io.ReadCloser
	var err error

	r, err = os.Open(f)
	if err != nil {
		statikFS, err := fs.New()
		if err != nil {
			return nil, err
		}

		r, err = statikFS.Open("/" + f + ".json")
		if err != nil {
			return nil, err
		}
	}

	defer r.Close()
	return ioutil.ReadAll(r)
}

func execute(cmd *cobra.Command, args []string) error {
	var arg string
	if len(args) > 0 {
		arg = args[0]
	}
	src, err := readerFromArg(arg)
	if err != nil {
		return err
	}
	defer src.reader.Close()
	b, _ := ioutil.ReadAll(src.reader)

	r := gold.NewPlainTermRenderer()
	if isatty.IsTerminal(os.Stdout.Fd()) {
		json, err := loadStyle(style)
		if err != nil {
			return err
		}

		r, err = gold.NewTermRendererFromBytes(json)
		if err != nil {
			return err
		}
	}
	r.WordWrap = int(width)

	u, err := url.ParseRequestURI(src.URL)
	if err == nil {
		u.Path = filepath.Dir(u.Path)
		r.BaseURL = u.String() + "/"
	}

	out := r.RenderBytes(b)
	fmt.Printf("%s", string(out))
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&style, "style", "s", "dark", "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 100, "word-wrap at width")
}
