package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/charmbracelet/gold"
)

var (
	rootCmd = &cobra.Command{
		Use:           "gold SOURCE",
		Short:         "Render markdown on the CLI, with pizzazz!",
		SilenceErrors: false,
		SilenceUsage:  false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return execute(args)
		},
	}

	style string
)

func readerFromArgument(s string) (io.ReadCloser, error) {
	if s == "-" {
		return os.Stdin, nil
	}

	if isGitHubURL(s) {
		resp, err := findGitHubREADME(s)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}

	return os.Open(s)
}

func execute(args []string) error {
	if len(args) != 1 {
		return errors.New("missing markdown source")
	}

	in, err := readerFromArgument(args[0])
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
