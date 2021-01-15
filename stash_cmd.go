package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

var (
	memo string

	stashCmd = &cobra.Command{
		Use:     "stash [SOURCE]",
		Hidden:  false,
		Short:   "Stash a markdown",
		Long:    formatBlock(fmt.Sprintf("\nDo %s stuff. Run with no arguments to browse your stash or pass a path to a markdown file to stash it.", common.Keyword("stash"))),
		Example: formatBlock("glow stash\nglow stash README.md\nglow stash -m \"secret notes\" path/to/notes.md"),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initConfig()
			if len(args) == 0 {
				return runTUI("", true)
			}

			filePath := args[0]

			if memo == "" {
				memo = strings.Replace(path.Base(filePath), path.Ext(filePath), "", 1)
			}

			cc := initCharmClient()
			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("bad filename")
			}

			defer f.Close()
			b, err := ioutil.ReadAll(f)
			if err != nil {
				return fmt.Errorf("error reading file")
			}

			_, err = cc.StashMarkdown(memo, string(b))
			if err != nil {
				return fmt.Errorf("error stashing markdown")
			}

			dot := termenv.String("•").Foreground(common.Green.Color()).String()
			fmt.Println(dot + " Stashed!")
			return nil
		},
	}
)

func getCharmConfig() *charm.Config {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

func initCharmClient() *charm.Client {
	cfg := getCharmConfig()
	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		fmt.Println(formatBlock("We had some trouble authenticating via SSH. If this continues to happen the Charm tool may be able to help you. More info at https://github.com/charmbracelet/charm."))
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}
