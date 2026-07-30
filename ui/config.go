package ui

// Config contains TUI-specific configuration.
type Config struct {
	ShowAllFiles     bool
	ShowLineNumbers  bool
	Gopath           string `env:"GOPATH"`
	HomeDir          string `env:"HOME"`
	GlamourMaxWidth  uint
	GlamourStyle     string `env:"GLAMOUR_STYLE"`
	EnableMouse      bool
	PreserveNewLines bool

	// Working directory or file path
	Path string

	// For debugging the UI
	//
	// Defaults to false: bubbletea's high-performance/scroll-region
	// rendering is deprecated upstream and known to corrupt output on
	// several terminals (e.g. kitty) during scroll. See #554. Users can
	// still opt in via GLOW_HIGH_PERFORMANCE_PAGER=true.
	HighPerformancePager bool `env:"GLOW_HIGH_PERFORMANCE_PAGER" envDefault:"false"`
	GlamourEnabled       bool `env:"GLOW_ENABLE_GLAMOUR"         envDefault:"true"`
}
