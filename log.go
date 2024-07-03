package main

import (
	"os"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
)

func getLogFilePath() (string, error) {
	return xdg.CacheFile("glow/glow.log")
}

func setupLog() (func() error, error) {
	// Log to file, if set
	logFile, err := getLogFilePath()
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)
	return f.Close, nil
}
