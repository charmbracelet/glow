package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/ansi"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

var manCmd = &cobra.Command{
	Use:                   "man",
	Short:                 "Generates manpages",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Hidden:                true,
	Args:                  cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		// Styled help text must not leak terminal escape codes into roff.
		rootCmd.Long = ansi.Strip(rootCmd.Long)

		manPage, err := mcobra.NewManPage(1, rootCmd)
		if err != nil {
			return fmt.Errorf("unable to instantiate man page: %w", err)
		}
		if _, err := fmt.Fprint(os.Stdout, manPage.Build(roff.NewDocument())); err != nil {
			return fmt.Errorf("unable to build man page: %w", err)
		}
		return nil
	},
}
