package main

import (
	"fmt"
	"io"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/internal/launcher"
)

// runtime is the composition root's output (workflow.md's "composition root pattern"):
// the real dependencies every command's RunE needs, built once by root.go's
// PersistentPreRunE and read by every subcommand.
type runtime struct {
	fs       iofs.FS
	env      env.Env
	config   jetbrains.Config
	launcher launcher.Launcher
}

// init wires fs/e (injected so tests use fakes; production passes iofs.New()/env.New()),
// the merged product configuration (loadConfig), logging (setupLogging), and the launcher
// resolved per ADR-0002 O1 (resolveLauncher).
func (rt *runtime) init(fs iofs.FS, e env.Env, version string, debug bool, launcherFlag, configFlag string, stderr io.Writer) error {
	rt.fs = fs
	rt.env = e

	logFile, err := setupLogging(stderr, debug, "")
	if err != nil {
		return fmt.Errorf("setting up logging: %w", err)
	}

	cfg, err := loadConfig(e, configFlag)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}
	rt.config = cfg

	reg := newLauncherRegistry(version, e, logFile)
	l, err := resolveLauncher(launcherFlag, e, reg)
	if err != nil {
		return err
	}
	rt.launcher = l

	return nil
}
