package main

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestSetupLogging_notDebug_stderrOnlyAtInfoLevel(t *testing.T) {
	var stderr bytes.Buffer

	logFile, err := setupLogging(&stderr, false, t.TempDir())
	if err != nil {
		t.Fatalf("setupLogging(): %v", err)
	}
	if logFile != "" {
		t.Errorf("logFile = %q, want empty outside debug mode", logFile)
	}

	slog.Debug("should not appear")
	slog.Info("should appear")

	if strings.Contains(stderr.String(), "should not appear") {
		t.Errorf("stderr contains a Debug record at Info level: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "should appear") {
		t.Errorf("stderr missing Info record: %s", stderr.String())
	}
}

func TestSetupLogging_debug_mirrorsToFileAtDebugLevel(t *testing.T) {
	var stderr bytes.Buffer

	logFile, err := setupLogging(&stderr, true, t.TempDir())
	if err != nil {
		t.Fatalf("setupLogging(): %v", err)
	}
	if logFile == "" {
		t.Fatal(`logFile = "", want a path in debug mode`)
	}

	slog.Debug("debug record")

	if !strings.Contains(stderr.String(), "debug record") {
		t.Errorf("stderr missing Debug record: %s", stderr.String())
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", logFile, err)
	}
	if !strings.Contains(string(content), "debug record") {
		t.Errorf("log file missing Debug record: %s", content)
	}
}
