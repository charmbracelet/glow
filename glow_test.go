package main

import (
	"os"
	"path/filepath"
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

func TestDisplayModePrecedence(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		envPager  string
		args      []string
		wantPager bool
		wantTUI   bool
		wantError bool
	}{
		{name: "configured pager", config: "pager: true", wantPager: true},
		{name: "short tui overrides configured pager", config: "pager: true", args: []string{"-t"}, wantTUI: true},
		{name: "long tui overrides configured pager", config: "pager: true", args: []string{"--tui=true"}, wantTUI: true},
		{name: "tui overrides environment pager", envPager: "true", args: []string{"-t"}, wantTUI: true},
		{name: "explicit modes conflict", args: []string{"-p", "-t"}, wantPager: true, wantTUI: true, wantError: true},
		{name: "explicit long modes conflict", args: []string{"--pager=true", "--tui=true"}, wantPager: true, wantTUI: true, wantError: true},
		{name: "configured modes conflict", config: "pager: true\ntui: true", wantPager: true, wantTUI: true, wantError: true},
		{name: "disabled tui preserves configured pager", config: "pager: true", args: []string{"--tui=false"}, wantPager: true},
		{name: "explicit pager preserves configured conflict", config: "tui: true", args: []string{"-p"}, wantPager: true, wantTUI: true, wantError: true},
		{name: "explicitly disabled pager", config: "pager: true", args: []string{"--pager=false", "-t"}, wantTUI: true},
		{name: "default modes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GLOW_PAGER", tt.envPager)
			t.Setenv("GLOW_TUI", "")
			t.Setenv("GLOW_STYLE", "dark")
			t.Setenv("GLOW_WIDTH", "80")
			viper.Reset()
			t.Cleanup(viper.Reset)
			configPath := filepath.Join(t.TempDir(), "glow.yml")
			if err := os.WriteFile(configPath, []byte("style: dark\n"+tt.config+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			viper.SetConfigFile(configPath)
			if err := viper.ReadInConfig(); err != nil {
				t.Fatal(err)
			}
			viper.SetEnvPrefix("GLOW")
			viper.AutomaticEnv()
			cmd := &cobra.Command{Use: "glow"}
			cmd.Flags().BoolP("pager", "p", false, "")
			cmd.Flags().BoolP("tui", "t", false, "")
			for _, name := range []string{"pager", "tui"} {
				if err := viper.BindPFlag(name, cmd.Flags().Lookup(name)); err != nil {
					t.Fatal(err)
				}
			}
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatal(err)
			}
			err := validateOptions(cmd)
			if tt.wantError {
				if err == nil || err.Error() != "cannot use both pager and tui" {
					t.Fatalf("expected display mode conflict, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if pager != tt.wantPager || tui != tt.wantTUI {
				t.Errorf("pager=%v, tui=%v; want pager=%v, tui=%v", pager, tui, tt.wantPager, tt.wantTUI)
			}
		})
	}
}
