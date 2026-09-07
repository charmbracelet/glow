package ui

import (
	"errors"
	"os"

	"golang.org/x/term"
)

func openRawTerminal() (*os.File, *term.State, error) {
	return nil, nil, errors.New("raw terminal is not supported on windows")
}

func winchNotify(chan os.Signal) {}

func winchStop(chan os.Signal) {}
