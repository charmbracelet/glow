package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/charmbracelet/charm/ui/common"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
)

const defaultConfig = `# style name or JSON path (default "auto")
style: "auto"
# show local files only; no network (TUI-mode only)
local: false
# mouse support (TUI-mode only)
mouse: false
# use pager to display markdown
pager: false
# word-wrap at width
width: 80`

var configCmd = &cobra.Command{
	Use:     "config",
	Hidden:  false,
	Short:   "Edit the glow config file",
	Long:    formatBlock(fmt.Sprintf("\n%s the glow config file. We’ll use EDITOR to determine which editor to use. If the config file doesn't exist, it will be created.", common.Keyword("Edit"))),
	Example: formatBlock("glow config\nglow config --config path/to/config.yml"),
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			return errors.New("no EDITOR environment variable set")
		}

		if configFile == "" {
			scope := gap.NewScope(gap.User, "glow")

			var err error
			configFile, err = scope.ConfigPath("glow.yml")
			if err != nil {
				return err
			}
		}

		if ext := path.Ext(configFile); ext != ".yaml" && ext != ".yml" {
			return fmt.Errorf("'%s' is not a supported config type: use '%s' or '%s'\n", ext, ".yaml", ".yml")
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			// File doesn't exist yet, create all necessary directories and
			// write the default config file
			if err := os.MkdirAll(path.Dir(configFile), 0700); err != nil {
				return err
			}

			f, err := os.Create(configFile)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := f.WriteString(defaultConfig); err != nil {
				return err
			}
		} else if err != nil { // some other error occurred
			return err
		}

		c := exec.Command(editor, configFile)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return err
		}

		fmt.Println("Wrote config file to:", configFile)
		return nil
	},
}
