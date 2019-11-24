package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/charmbracelet/gold"
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
)

func readerFromArg(s string) (io.ReadCloser, error) {
	if s == "-" {
		return os.Stdin, nil
	}

	if u, err := url.ParseRequestURI(s); err == nil {
		if !strings.HasPrefix(u.Scheme, "http") {
			return nil, fmt.Errorf("%s is not a supported protocol", u.Scheme)
		}

		if isGitHubURL(s) {
			resp, err := findGitHubREADME(s)
			if err != nil {
				return nil, err
			}
			return resp.Body, nil
		}

		resp, err := http.Get(u.String())
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
		}
		return resp.Body, nil
	}

	if len(s) == 0 {
		for _, v := range readmeNames {
			r, err := os.Open(v)
			if err == nil {
				return r, err
			}
		}

		return nil, errors.New("missing markdown source")
	}

	return os.Open(s)
}

func execute(cmd *cobra.Command, args []string) error {
	var arg string
	if len(args) > 0 {
		arg = args[0]
	}
	in, err := readerFromArg(arg)
	if err != nil {
		return err
	}
	defer in.Close()

	b, _ := ioutil.ReadAll(in)
	r, err := gold.NewTermRenderer(style)
	if err != nil {
		return err
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
	rootCmd.Flags().StringVarP(&style, "style", "s", "dark.json", "style JSON path")
}
