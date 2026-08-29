package main

import (
	"fmt"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
)

// resolveLauncher implements ADR-0002 O1's precedence: an explicit --launcher flag wins,
// then the HERMES_LAUNCHER env var, then auto-detection, and finally reg.Default() (the
// generic adapter) when nothing else resolves - "no launcher resolved is not an error".
func resolveLauncher(flagValue string, e env.Env, reg *launcher.Registry) (launcher.Launcher, error) {
	if flagValue != "" {
		l, ok := reg.Get(flagValue)
		if !ok {
			return nil, fmt.Errorf("unknown launcher %q", flagValue)
		}
		return l, nil
	}

	if name, ok := e.Lookup("HERMES_LAUNCHER"); ok && name != "" {
		l, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown launcher %q (from HERMES_LAUNCHER)", name)
		}
		return l, nil
	}

	if l, ok := reg.Detect(e); ok {
		return l, nil
	}
	return reg.Default(), nil
}
