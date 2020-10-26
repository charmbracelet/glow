package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:    "config",
	Hidden: false,
	Short:  "Edit the glow config file",
	Long:   formatBlock(fmt.Sprintf("\n%s the glow config file. We’ll use EDITOR to determine which editor to use. If the config file doesn't exist, it will be created.", common.Keyword("Edit"))),
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("EDITORS") == "" {
			return errors.New("no EDITOR environment variable set")
		}

		return nil
	},
}
