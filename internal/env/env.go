// Package env is the only door to the process environment from internal/jetbrains and
// internal/launcher.
package env

import (
	"os"
	"strings"
)

// Env abstracts process environment reads so callers are testable without touching the
// real environment.
type Env interface {
	Lookup(key string) (string, bool)
	Home() string
	Path() []string
}

// New returns an Env backed by the real process environment.
func New() Env {
	return osEnv{}
}

type osEnv struct{}

func (osEnv) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (e osEnv) Home() string {
	v, _ := e.Lookup("HOME")
	return v
}

func (e osEnv) Path() []string {
	v, ok := e.Lookup("PATH")
	if !ok || v == "" {
		return nil
	}
	return strings.Split(v, ":")
}
