// Package envtest provides an in-memory env.Env fake for tests.
package envtest

import (
	"strings"

	"github.com/adaouat/hermes/internal/env"
)

// New returns an env.Env backed by the given map, never the real process environment.
func New(vars map[string]string) env.Env {
	return fakeEnv{vars: vars}
}

type fakeEnv struct {
	vars map[string]string
}

func (e fakeEnv) Lookup(key string) (string, bool) {
	v, ok := e.vars[key]
	return v, ok
}

func (e fakeEnv) Home() string {
	v, _ := e.Lookup("HOME")
	return v
}

func (e fakeEnv) Path() []string {
	v, ok := e.Lookup("PATH")
	if !ok || v == "" {
		return nil
	}
	return strings.Split(v, ":")
}
