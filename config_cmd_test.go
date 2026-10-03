package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestDefaultConfigMatchesDefaults guards against the config file written on
// first run disagreeing with the defaults glow runs with when there is no
// config file, which would make the first run behave differently from the ones
// after it.
func TestDefaultConfigMatchesDefaults(t *testing.T) {
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	if err := cfg.ReadConfig(strings.NewReader(defaultConfig)); err != nil {
		t.Fatalf("unable to parse the default config: %v", err)
	}

	want := viper.New()
	setDefaults(want)
	// These settings are defaulted by their flags rather than by viper. Their
	// config keys don't always match the flag names, see the bindings in init.
	for key, name := range map[string]string{
		"mouse":            "mouse",
		"pager":            "pager",
		"showLineNumbers":  "line-numbers",
		"preserveNewLines": "preserve-new-lines",
	} {
		f := rootCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("no --%s flag", name)
		}
		b, err := strconv.ParseBool(f.DefValue)
		if err != nil {
			t.Fatalf("--%s has a non-boolean default %q", name, f.DefValue)
		}
		want.SetDefault(key, b)
	}

	for _, k := range cfg.AllKeys() {
		if !want.IsSet(k) {
			t.Errorf("the default config sets %q, but glow has no default for it", k)
			continue
		}
		got := fmt.Sprintf("%T(%v)", cfg.Get(k), cfg.Get(k))
		def := fmt.Sprintf("%T(%v)", want.Get(k), want.Get(k))
		if got != def {
			t.Errorf("the default config sets %q to %s, glow's default is %s", k, got, def)
		}
	}
}

// TestSetupConfigCreatesAndLoads makes sure a config file created on a given run
// is also used by that very run, and that later runs keep using it.
func TestSetupConfigCreatesAndLoads(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "glow.yml")

	origConfigFile := configFile
	t.Cleanup(func() { configFile = origConfigFile })
	// Keep the environment out of the way of the settings under test.
	t.Setenv("GLOW_WIDTH", "0")

	v := viper.New()
	if err := setupConfig(v, []string{dir}); err != nil {
		t.Fatalf("unable to set up config: %v", err)
	}

	if got := v.ConfigFileUsed(); got != want {
		t.Errorf("config file used = %q, want %q", got, want)
	}
	if got := configFile; got != want {
		t.Errorf("configFile = %q, want %q", got, want)
	}
	if got := v.GetUint("width"); got != 0 {
		t.Errorf("width = %d, want 0 (fit the terminal)", got)
	}

	b, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("unable to read the generated config: %v", err)
	}
	if string(b) != defaultConfig {
		t.Errorf("generated config is\n%s\nwant\n%s", b, defaultConfig)
	}

	// An existing config must be picked up as is.
	if err := setupConfig(v, []string{dir}); err != nil {
		t.Fatalf("unable to set up config on the second run: %v", err)
	}
	if got := v.ConfigFileUsed(); got != want {
		t.Errorf("config file used on the second run = %q, want %q", got, want)
	}
}

// TestSetupConfigWarnsOnInvalidConfig makes sure an unparseable config file
// doesn't keep glow from running, it just gets ignored.
func TestSetupConfigWarnsOnInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "glow.yml")
	if err := os.WriteFile(path, []byte("style: [1, 2\n"), 0o600); err != nil {
		t.Fatalf("unable to write config: %v", err)
	}

	origConfigFile := configFile
	t.Cleanup(func() { configFile = origConfigFile })

	v := viper.New()
	setDefaults(v)
	if err := setupConfig(v, []string{dir}); err != nil {
		t.Errorf("setupConfig with an invalid config returned %v, want nil", err)
	}
	if got := configFile; got != path {
		t.Errorf("configFile = %q, want %q", got, path)
	}
	if got := v.GetString("style"); got != "auto" {
		t.Errorf("style = %q, want %q", got, "auto")
	}
}
