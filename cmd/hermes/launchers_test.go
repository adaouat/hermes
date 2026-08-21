package main

import (
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
)

func TestNewLauncherRegistry_detectsAlfred(t *testing.T) {
	r := newLauncherRegistry("1.2.3", envtest.New(nil))

	got, ok := r.Detect(envtest.New(map[string]string{"alfred_version": "5.5"}))
	if !ok || got.Name() != "alfred" {
		t.Errorf("Detect(alfred_version set) = (%v, %v), want (alfred, true)", got, ok)
	}

	_, ok = r.Detect(envtest.New(nil))
	if ok {
		t.Errorf("Detect(no signal) ok = true, want false")
	}
}

func TestNewLauncherRegistry_defaultIsGeneric(t *testing.T) {
	r := newLauncherRegistry("1.2.3", envtest.New(nil))
	if got := r.Default().Name(); got != "generic" {
		t.Errorf("Default() = %q, want %q", got, "generic")
	}
}
