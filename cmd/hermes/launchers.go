package main

import (
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/internal/launcher/alfred"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

// newLauncherRegistry builds the Registry every launcher-aware command reads from:
// Get/Detect/Default per ADR-0002 O1's precedence chain. logFile, when non-empty, threads
// the active debug log file into the Alfred adapter's third debug item.
func newLauncherRegistry(version string, e env.Env, logFile string) *launcher.Registry {
	return launcher.NewRegistry(
		generic.NewAdapter(),
		alfred.NewAdapter(e, alfred.WithVersion(version), alfred.WithLogFile(logFile)),
	)
}
