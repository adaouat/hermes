package main

import "github.com/adaouat/hermes/internal/env"

// resolveDebug implements ADR-0002 O2: an explicit --debug flag wins, otherwise the
// HERMES_DEBUG env var's mere presence (any value, matching alfred_debug's own
// presence-based semantics) turns debug mode on.
func resolveDebug(flagValue bool, e env.Env) bool {
	if flagValue {
		return true
	}
	_, ok := e.Lookup("HERMES_DEBUG")
	return ok
}
