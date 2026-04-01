package ui

import "github.com/spf13/viper"

// KeyMap holds the configurable key bindings for glow.
// Each field is a slice of key names (e.g. "j", "down", "ctrl+j").
type KeyMap struct {
	// Pager keys
	Up       []string
	Down     []string
	PageUp   []string
	PageDown []string
	HalfUp   []string
	HalfDown []string
	GoToTop  []string
	GoToEnd  []string
	Quit     []string
	Back     []string
	Help     []string
	Edit     []string
	Copy     []string
	Reload   []string

	// Stash/file listing keys
	Open   []string
	Filter []string
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       []string{"k", "up"},
		Down:     []string{"j", "down"},
		PageUp:   []string{"b", "pgup"},
		PageDown: []string{"f", "pgdn"},
		HalfUp:   []string{"u"},
		HalfDown: []string{"d"},
		GoToTop:  []string{"home", "g"},
		GoToEnd:  []string{"end", "G"},
		Quit:     []string{"q"},
		Back:     []string{"esc", "left", "h", "delete"},
		Help:     []string{"?"},
		Edit:     []string{"e"},
		Copy:     []string{"c"},
		Reload:   []string{"r"},
		Open:     []string{"enter"},
		Filter:   []string{"/"},
	}
}

// LoadKeyMap loads the key map from viper config, falling back to defaults.
func LoadKeyMap() KeyMap {
	km := DefaultKeyMap()

	if v := viper.GetStringSlice("keymap.up"); len(v) > 0 {
		km.Up = v
	}
	if v := viper.GetStringSlice("keymap.down"); len(v) > 0 {
		km.Down = v
	}
	if v := viper.GetStringSlice("keymap.pageUp"); len(v) > 0 {
		km.PageUp = v
	}
	if v := viper.GetStringSlice("keymap.pageDown"); len(v) > 0 {
		km.PageDown = v
	}
	if v := viper.GetStringSlice("keymap.halfUp"); len(v) > 0 {
		km.HalfUp = v
	}
	if v := viper.GetStringSlice("keymap.halfDown"); len(v) > 0 {
		km.HalfDown = v
	}
	if v := viper.GetStringSlice("keymap.goToTop"); len(v) > 0 {
		km.GoToTop = v
	}
	if v := viper.GetStringSlice("keymap.goToEnd"); len(v) > 0 {
		km.GoToEnd = v
	}
	if v := viper.GetStringSlice("keymap.quit"); len(v) > 0 {
		km.Quit = v
	}
	if v := viper.GetStringSlice("keymap.back"); len(v) > 0 {
		km.Back = v
	}
	if v := viper.GetStringSlice("keymap.help"); len(v) > 0 {
		km.Help = v
	}
	if v := viper.GetStringSlice("keymap.edit"); len(v) > 0 {
		km.Edit = v
	}
	if v := viper.GetStringSlice("keymap.copy"); len(v) > 0 {
		km.Copy = v
	}
	if v := viper.GetStringSlice("keymap.reload"); len(v) > 0 {
		km.Reload = v
	}
	if v := viper.GetStringSlice("keymap.open"); len(v) > 0 {
		km.Open = v
	}
	if v := viper.GetStringSlice("keymap.filter"); len(v) > 0 {
		km.Filter = v
	}

	return km
}

// matchesKey checks if the given key string matches any of the configured keys.
func matchesKey(key string, bindings []string) bool {
	for _, b := range bindings {
		if key == b {
			return true
		}
	}
	return false
}
