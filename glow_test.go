package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGlowFlags(t *testing.T) {
	tt := []struct {
		args  []string
		check func() bool
	}{
		{
			args: []string{"-p"},
			check: func() bool {
				return pager
			},
		},
		{
			args: []string{"-s", "light"},
			check: func() bool {
				return style == "light"
			},
		},
		{
			args: []string{"-w", "40"},
			check: func() bool {
				return width == 40
			},
		},
	}

	for _, v := range tt {
		err := rootCmd.ParseFlags(v.args)
		if err != nil {
			t.Fatal(err)
		}
		if !v.check() {
			t.Errorf("Parsing flag failed: %s", v.args)
		}
	}
}

func TestValidateOptions_StyleWhenPiped(t *testing.T) {
	origIsTerminal := isTerminal
	defer func() {
		isTerminal = origIsTerminal
		viper.Reset()
		_ = viper.BindPFlag("pager", rootCmd.Flags().Lookup("pager"))
		_ = viper.BindPFlag("tui", rootCmd.Flags().Lookup("tui"))
		_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
		_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
		_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
		_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
		_ = viper.BindPFlag("preserveNewLines", rootCmd.Flags().Lookup("preserve-new-lines"))
		_ = viper.BindPFlag("showLineNumbers", rootCmd.Flags().Lookup("line-numbers"))
		_ = viper.BindPFlag("all", rootCmd.Flags().Lookup("all"))
		viper.SetDefault("style", "auto")
		viper.SetDefault("width", 0)
		viper.SetDefault("all", true)
	}()

	tests := []struct {
		name          string
		configStyle   string
		flagArgs      []string
		isTerminalVal bool
		expectedStyle string
	}{
		{
			name:          "piped stdout with default auto style becomes notty",
			configStyle:   "auto",
			isTerminalVal: false,
			expectedStyle: "notty",
		},
		{
			name:          "piped stdout with configured dark style preserves dark",
			configStyle:   "dark",
			isTerminalVal: false,
			expectedStyle: "dark",
		},
		{
			name:          "piped stdout with configured light style preserves light",
			configStyle:   "light",
			isTerminalVal: false,
			expectedStyle: "light",
		},
		{
			name:          "piped stdout with configured pink style preserves pink",
			configStyle:   "pink",
			isTerminalVal: false,
			expectedStyle: "pink",
		},
		{
			name:          "piped stdout with configured notty style preserves notty",
			configStyle:   "notty",
			isTerminalVal: false,
			expectedStyle: "notty",
		},
		{
			name:          "piped stdout with CLI flag --style overrides config style",
			configStyle:   "dark",
			flagArgs:      []string{"--style", "light"},
			isTerminalVal: false,
			expectedStyle: "light",
		},
		{
			name:          "piped stdout with CLI flag --style notty overrides config style",
			configStyle:   "dark",
			flagArgs:      []string{"--style", "notty"},
			isTerminalVal: false,
			expectedStyle: "notty",
		},
		{
			name:          "piped stdout with CLI flag --style auto preserves auto",
			configStyle:   "dark",
			flagArgs:      []string{"--style", "auto"},
			isTerminalVal: false,
			expectedStyle: "auto",
		},
		{
			name:          "terminal stdout with auto style preserves auto",
			configStyle:   "auto",
			isTerminalVal: true,
			expectedStyle: "auto",
		},
		{
			name:          "terminal stdout with configured dark style preserves dark",
			configStyle:   "dark",
			isTerminalVal: true,
			expectedStyle: "dark",
		},
		{
			name:          "terminal stdout with configured light style preserves light",
			configStyle:   "light",
			isTerminalVal: true,
			expectedStyle: "light",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isTerminal = func(fd int) bool {
				return tc.isTerminalVal
			}

			viper.Reset()
			viper.SetConfigType("yaml")
			if err := viper.ReadConfig(strings.NewReader("style: " + tc.configStyle)); err != nil {
				t.Fatalf("unexpected read config error: %v", err)
			}

			cmd := &cobra.Command{}
			var cmdStyle string
			var cmdWidth uint
			cmd.Flags().StringVarP(&cmdStyle, "style", "s", "auto", "style")
			cmd.Flags().UintVarP(&cmdWidth, "width", "w", 0, "width")
			_ = viper.BindPFlag("style", cmd.Flags().Lookup("style"))
			_ = viper.BindPFlag("width", cmd.Flags().Lookup("width"))

			if len(tc.flagArgs) > 0 {
				if err := cmd.ParseFlags(tc.flagArgs); err != nil {
					t.Fatalf("unexpected flag parse error: %v", err)
				}
			}

			if err := validateOptions(cmd); err != nil {
				t.Fatalf("unexpected validateOptions error: %v", err)
			}

			if style != tc.expectedStyle {
				t.Errorf("expected style %q, got %q", tc.expectedStyle, style)
			}
		})
	}
}

