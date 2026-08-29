package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/adaouat/hermes/internal/jetbrains"
)

// errWriteFailed is erroringWriter's sentinel error, asserted with errors.Is to prove
// runConfiguration wraps an encoding failure rather than swallowing it.
var errWriteFailed = errors.New("write boom")

// erroringWriter is an io.Writer that always fails, for exercising runConfiguration's
// "encoding configuration: %w" wrap branch without needing a real broken destination.
type erroringWriter struct{}

func (erroringWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

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

func TestRunConfiguration_encodeErrorWrapped(t *testing.T) {
	rt := &runtime{config: jetbrains.Defaults()}

	if err := runConfiguration(erroringWriter{}, rt); !errors.Is(err, errWriteFailed) {
		t.Errorf("runConfiguration() error = %v, want it to wrap errWriteFailed", err)
	}
}

func TestNewConfigurationCmd_executesEndToEnd(t *testing.T) {
	rt := &runtime{config: jetbrains.Defaults()}
	cmd := newConfigurationCmd(rt)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("phpStorm")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "phpStorm")
	}
}
