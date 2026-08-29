package main

import (
	"bytes"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
)

func TestRuntimeInit_wiresConfigAndLauncher(t *testing.T) {
	rt := &runtime{}
	e := envtest.New(map[string]string{"alfred_version": "5.5"})
	var stderr bytes.Buffer

	if err := rt.init(iofstest.New(nil), e, "1.0.0", false, "", "", &stderr); err != nil {
		t.Fatalf("init(): %v", err)
	}
	if rt.launcher.Name() != "alfred" {
		t.Errorf("launcher = %q, want %q (auto-detected via alfred_version)", rt.launcher.Name(), "alfred")
	}
	if len(rt.config) != 19 {
		t.Errorf("config has %d products, want 19 (defaults, no overrides)", len(rt.config))
	}
}

func TestRuntimeInit_unknownLauncherFlagFails(t *testing.T) {
	rt := &runtime{}
	var stderr bytes.Buffer

	if err := rt.init(iofstest.New(nil), envtest.New(nil), "1.0.0", false, "bogus", "", &stderr); err == nil {
		t.Fatal("init() with unknown --launcher: want error, got nil")
	}
}

func TestRuntimeInit_badConfigPathFails(t *testing.T) {
	rt := &runtime{}
	var stderr bytes.Buffer

	if err := rt.init(iofstest.New(nil), envtest.New(nil), "1.0.0", false, "", "/nonexistent/hermes-config.json", &stderr); err == nil {
		t.Fatal("init() with unreadable --config path: want error, got nil")
	}
}

// exitNotFound has no production caller until Task 10's search command lands; this
// keeps golangci-lint's unused check green in the interim by asserting the value stays in
// forge exitcode's reserved app range (4-69), same as exitcode.go's doc comment promises.
func TestExitNotFound_inForgeAppRange(t *testing.T) {
	if exitNotFound < 4 || exitNotFound > 69 {
		t.Errorf("exitNotFound = %d, want in forge exitcode's reserved app range [4, 69]", exitNotFound)
	}
}
