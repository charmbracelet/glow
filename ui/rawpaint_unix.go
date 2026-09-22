//go:build !windows

package ui

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func openRawTerminal() (*os.File, *term.State, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	state, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		_ = tty.Close()
		return nil, nil, err
	}
	return tty, state, nil
}

func winchNotify(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}

func winchStop(ch chan os.Signal) {
	signal.Stop(ch)
}
