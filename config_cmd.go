package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/charmbracelet/x/editor"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
)

const defaultConfig = `# style name or JSON path (default "auto")
style: "auto"
# mouse support (TUI-mode only)
mouse: false
# use pager to display markdown
pager: false
# word-wrap at width
width: 80
`

func defaultConfigFile() string {
	scope := gap.NewScope(gap.User, "glow")
	path, _ := scope.ConfigPath("glow.yml")
	return path
}

var configCmd = &cobra.Command{
	Use:     "config",
	Hidden:  false,
	Short:   "Edit the glow config file",
	Long:    paragraph(fmt.Sprintf("\n%s the glow config file. We’ll use EDITOR to determine which editor to use. If the config file doesn't exist, it will be created.", keyword("Edit"))),
	Example: paragraph("glow config\nglow config --config path/to/config.yml"),
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureConfigFile(); err != nil {
			return err
		}

		c, err := editor.Cmd("glow", configFile)
		if err != nil {
			return err
		}
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

func ensureConfigFile() error {
	if configFile == "" {
		configFile = defaultConfigFile()
		if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
			return fmt.Errorf("Could not write config file: %w", err)
		}
	}

	if ext := path.Ext(configFile); ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("'%s' is not a supported config type: use '%s' or '%s'", ext, ".yaml", ".yml")
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// File doesn't exist yet, create all necessary directories and
		// write the default config file
		if err := os.MkdirAll(filepath.Dir(configFile), 0o700); err != nil {
			return err
		}

		f, err := os.Create(configFile)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		if _, err := f.WriteString(defaultConfig); err != nil {
			return err
		}
	} else if err != nil { // some other error occurred
		return err
	}
	return nil
}
