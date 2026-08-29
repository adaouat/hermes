package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunAll_rendersAcrossProducts(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runAll(&buf, rt, ""); err != nil {
		t.Fatalf("runAll(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestRunAll_missingProductsAreSkipped(t *testing.T) {
	rt := newTestRuntime(nil)
	var buf bytes.Buffer

	if err := runAll(&buf, rt, ""); err != nil {
		t.Fatalf("runAll() with no products installed: %v", err)
	}
	if buf.String() != "[]\n" {
		t.Errorf("output = %q, want %q (empty generic array)", buf.String(), "[]\n")
	}
}

func TestRunAll_renderErrorWrapped(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	rt.launcher = failingLauncher{}
	var buf bytes.Buffer

	if err := runAll(&buf, rt, ""); !errors.Is(err, errRenderFailed) {
		t.Errorf("runAll() error = %v, want it to wrap errRenderFailed", err)
	}
}

func TestNewAllCmd_executesEndToEnd(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	cmd := newAllCmd(rt)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}
