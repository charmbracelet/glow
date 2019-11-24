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

	return os.Open(s)
}

func execute(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("missing markdown source")
	}

	in, err := readerFromArg(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	b, _ := ioutil.ReadAll(in)
	out, err := gold.RenderBytes(b, style)
	if err != nil {
		return err
	}
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
