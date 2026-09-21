package main

import (
	"testing"

	"charm.land/glow/v3/ui"
	"github.com/caarlos0/env/v11"
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

// TestLoadRemoteImagesEnv checks the environment variable that enables
// loading remote images.
func TestLoadRemoteImagesEnv(t *testing.T) {
	t.Setenv("GLOW_LOAD_REMOTE_IMAGES", "false")
	cfg, err := env.ParseAs[ui.Config]()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LoadRemoteImages {
		t.Error("expected remote image loading to be disabled by default")
	}

	t.Setenv("GLOW_LOAD_REMOTE_IMAGES", "true")
	cfg, err = env.ParseAs[ui.Config]()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LoadRemoteImages {
		t.Error("expected GLOW_LOAD_REMOTE_IMAGES to enable remote image loading")
	}
}
