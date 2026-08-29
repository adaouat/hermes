package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/adaouat/hermes/internal/jetbrains"
)

func TestRunConfiguration_printsMergedConfig(t *testing.T) {
	rt := &runtime{config: jetbrains.Defaults()}
	var buf bytes.Buffer

	if err := runConfiguration(&buf, rt); err != nil {
		t.Fatalf("runConfiguration(): %v", err)
	}

	var got map[string]jetbrains.ProductDetails
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got) != 19 {
		t.Errorf("configuration has %d products, want 19", len(got))
	}
	if got["phpStorm"].PreferencePrefix != "PhpStorm" {
		t.Errorf(`configuration["phpStorm"].PreferencePrefix = %q, want %q`, got["phpStorm"].PreferencePrefix, "PhpStorm")
	}
}
