package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// setupLogging installs the process-wide slog default: a stderr-only Info handler
// normally, or - when debug is true - a Debug-level handler mirroring every record to
// both stderr and a temp log file (dir; empty means the OS default temp directory),
// returning that file's path so callers (the Alfred adapter's third debug item, via
// alfred.WithLogFile) can point users at it. hermes is a one-shot CLI, not a daemon, so
// the file is left open for the process lifetime and reclaimed on exit - no explicit
// Close.
func setupLogging(stderr io.Writer, debug bool, dir string) (string, error) {
	if !debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
		return "", nil
	}

	f, err := os.CreateTemp(dir, "hermes-debug-*.log")
	if err != nil {
		return "", fmt.Errorf("creating debug log file: %w", err)
	}

	handler := slog.NewTextHandler(io.MultiWriter(stderr, f), &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	return f.Name(), nil
}
