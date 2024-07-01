package ui

// Config contains TUI-specific configuration.
type Config struct {
	ShowAllFiles    bool
	Gopath          string `env:"GOPATH"`
	HomeDir         string `env:"HOME"`
	GlamourMaxWidth uint
	GlamourStyle    string
	EnableMouse     bool

	// Which directory should we start from?
	WorkingDirectory string

	// For debugging the UI
	HighPerformancePager bool `env:"GLOW_HIGH_PERFORMANCE_PAGER" envDefault:"true"`
	GlamourEnabled       bool `env:"GLOW_ENABLE_GLAMOUR"         envDefault:"true"`
}
