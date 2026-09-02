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

	// Images controls whether to render images in the pager using the
	// terminal's graphics protocol. ImageProtocol can be one of "auto",
	// "kitty", or "none" (see utils.ImageProtocol* constants). ImageMaxRows
	// limits the number of terminal rows an image may occupy; zero means no
	// limit.
	Images        bool   `env:"GLOW_IMAGES" envDefault:"true"`
	ImageProtocol string `env:"GLOW_IMAGE_PROTOCOL" envDefault:"auto"`
	ImageMaxRows  int    `env:"GLOW_IMAGE_MAX_ROWS" envDefault:"20"`

	// Working directory or file path
	Path string

	// For debugging the UI
	GlamourEnabled bool `env:"GLOW_ENABLE_GLAMOUR" envDefault:"true"`
}
