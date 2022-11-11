package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	memo string
	dot  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render("•")

	stashCmd = &cobra.Command{
		Use:     "stash [SOURCE]",
		Hidden:  false,
		Short:   "Stash a markdown",
		Long:    paragraph(fmt.Sprintf("\nDo %s stuff. Run with no arguments to browse your stash or pass a path to a markdown file to stash it.", keyword("stash"))),
		Example: paragraph("glow stash\nglow stash README.md\nglow stash -m \"secret notes\" path/to/notes.md"),
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

			defer f.Close() //nolint:errcheck
			b, err := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("error reading file")
			}

			_, err = cc.StashMarkdown(memo, string(b))
			if err != nil {
				return fmt.Errorf("error stashing markdown")
			}

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
		fmt.Println(paragraph("We had some trouble authenticating via SSH. If this continues to happen the Charm tool may be able to help you. More info at https://github.com/charmbracelet/charm."))
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}
