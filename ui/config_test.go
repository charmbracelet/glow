package ui

import (
	"testing"

	"github.com/caarlos0/env/v11"
)

func TestConfig_HighPerformancePagerDefault(t *testing.T) {
	t.Setenv("GLOW_HIGH_PERFORMANCE_PAGER", "")

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		t.Fatalf("env.ParseAs[Config]() error = %v", err)
	}

	if cfg.HighPerformancePager {
		t.Errorf("HighPerformancePager default = true, want false (see #554: high-performance rendering is deprecated upstream and corrupts scroll on some terminals)")
	}
}

func TestConfig_HighPerformancePagerOptIn(t *testing.T) {
	t.Setenv("GLOW_HIGH_PERFORMANCE_PAGER", "true")

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		t.Fatalf("env.ParseAs[Config]() error = %v", err)
	}

	if !cfg.HighPerformancePager {
		t.Error("HighPerformancePager with GLOW_HIGH_PERFORMANCE_PAGER=true = false, want true (opt-in must still work)")
	}
}

func TestConfig_HighPerformancePagerExplicitOff(t *testing.T) {
	t.Setenv("GLOW_HIGH_PERFORMANCE_PAGER", "false")

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		t.Fatalf("env.ParseAs[Config]() error = %v", err)
	}

	if cfg.HighPerformancePager {
		t.Error("HighPerformancePager with GLOW_HIGH_PERFORMANCE_PAGER=false = true, want false")
	}
}
