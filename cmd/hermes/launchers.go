package main

import (
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/internal/launcher/alfred"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

// newLauncherRegistry builds the Registry every launcher-aware command (M3)
// will read from: Get/Detect/Default per ADR-0002 O1's precedence chain.
func newLauncherRegistry(version string, e env.Env) *launcher.Registry {
	return launcher.NewRegistry(
		generic.NewAdapter(),
		alfred.NewAdapter(e, alfred.WithVersion(version)),
	)
}
