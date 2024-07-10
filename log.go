package main

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	gap "github.com/muesli/go-app-paths"
)

func getLogFilePath() (string, error) {
	dir, err := gap.NewScope(gap.User, "glow").CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "glow.log"), nil
}

func setupLog() (func() error, error) {
	// Log to file, if set
	logFile, err := getLogFilePath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(logFile), 0o644); err != nil {
		// log disabled
		return nil, nil
	}
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		// log disabled
		return nil, nil
	}
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
	return f.Close, nil
}
