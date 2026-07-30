package ui

import "testing"

// newPagerModel wires viewport.HighPerformanceRendering from the
// package-level config var (set via NewProgram in normal operation).
// These tests set it directly to exercise that wiring in isolation.

func TestNewPagerModel_HighPerformanceRenderingOff(t *testing.T) {
	old := config
	defer func() { config = old }()

	config = Config{HighPerformancePager: false}
	common := &commonModel{cfg: config}

	m := newPagerModel(common)

	if m.viewport.HighPerformanceRendering {
		t.Error("viewport.HighPerformanceRendering = true, want false when config.HighPerformancePager is false (see #554)")
	}
}

func TestNewPagerModel_HighPerformanceRenderingOptIn(t *testing.T) {
	old := config
	defer func() { config = old }()

	config = Config{HighPerformancePager: true}
	common := &commonModel{cfg: config}

	m := newPagerModel(common)

	if !m.viewport.HighPerformanceRendering {
		t.Error("viewport.HighPerformanceRendering = false, want true when config.HighPerformancePager is explicitly true (opt-in must still work)")
	}
}
