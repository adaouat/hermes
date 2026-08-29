package main

import (
	"bytes"
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
