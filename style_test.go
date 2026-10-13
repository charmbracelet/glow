package main

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestValidateOptionsPipedStyle(t *testing.T) {
	// Start from an unset --style flag (other tests parse flags on rootCmd).
	flag := rootCmd.Flags().Lookup("style")
	value, changed := flag.Value.String(), flag.Changed
	_ = flag.Value.Set(flag.DefValue)
	flag.Changed = false
	t.Cleanup(func() {
		_ = flag.Value.Set(value)
		flag.Changed = changed
	})

	// go test's stdout is not a terminal, like `glow README.md | cat`.
	for _, tc := range []struct {
		name   string
		config string
		env    string
		want   string
	}{
		{"configured style is kept", "style: dark\n", "", "dark"},
		{"style from the environment is kept", "", "light", "light"},
		{"no style falls back to notty", "", "", "notty"},
		{"generated config with auto falls back to notty", "style: \"auto\"\n", "", "notty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("GLOW_STYLE", tc.env)
			}
			viper.SetConfigType("yaml")
			if err := viper.ReadConfig(strings.NewReader(tc.config)); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = viper.ReadConfig(strings.NewReader("")) })

			if err := validateOptions(rootCmd); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if style != tc.want {
				t.Errorf("expected style %q, got %q", tc.want, style)
			}
		})
	}
}
