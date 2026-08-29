package main

import (
	"bytes"
	"errors"
	"testing"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/jetbrains"
)

func TestRunOpen_rendersOneItem(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runOpen(&buf, rt, "phpStorm", "/home/x/projects/aurora"); err != nil {
		t.Fatalf("runOpen(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestRunOpen_unknownProductReturnsError(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runOpen(&buf, rt, "notAProduct", "/home/x/projects/aurora"); err == nil {
		t.Fatal("runOpen(unknown product): want error, got nil")
	}
}

func TestRunOpen_notFoundWrapsExitCode(t *testing.T) {
	rt := newTestRuntime(nil) // no PhpStorm.app fixture -> NotFoundError
	var buf bytes.Buffer

	err := runOpen(&buf, rt, "phpStorm", "/home/x/projects/aurora")
	if err == nil {
		t.Fatal("runOpen(): want error, got nil")
	}
	if forgeexit.Resolve(err) != exitNotFound {
		t.Errorf("exitcode.Resolve(err) = %d, want %d", forgeexit.Resolve(err), exitNotFound)
	}
	var notFound *jetbrains.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error chain does not contain *jetbrains.NotFoundError")
	}
}
