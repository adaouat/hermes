package main

import (
	"bytes"
	"errors"
	"io"
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

func TestRunOpen_nonNotFoundErrorPassesThrough(t *testing.T) {
	files := phpStormFixture()
	files["/home/x/projects/broken/.idea/workspace.xml"] = "not valid xml <<<"
	rt := newTestRuntime(files)
	var buf bytes.Buffer

	err := runOpen(&buf, rt, "phpStorm", "/home/x/projects/broken")
	if err == nil {
		t.Fatal("runOpen() with malformed workspace.xml: want error, got nil")
	}
	var notFound *jetbrains.NotFoundError
	if errors.As(err, &notFound) {
		t.Errorf("error = %v, want a non-NotFoundError (XML parse failure)", err)
	}
	if forgeexit.Resolve(err) == exitNotFound {
		t.Errorf("exitcode.Resolve(err) = %d, want anything but %d", forgeexit.Resolve(err), exitNotFound)
	}
}

func TestRunOpen_renderErrorWrapped(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	rt.launcher = failingLauncher{}
	var buf bytes.Buffer

	if err := runOpen(&buf, rt, "phpStorm", "/home/x/projects/aurora"); !errors.Is(err, errRenderFailed) {
		t.Errorf("runOpen() error = %v, want it to wrap errRenderFailed", err)
	}
}

func TestNewOpenCmd_executesEndToEnd(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	cmd := newOpenCmd(rt)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--product", "phpStorm", "--path", "/home/x/projects/aurora"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestNewOpenCmd_missingRequiredFlagsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing product", args: []string{"--path", "/home/x/projects/aurora"}},
		{name: "missing path", args: []string{"--product", "phpStorm"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := newTestRuntime(phpStormFixture())
			cmd := newOpenCmd(rt)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs(tc.args)

			if err := cmd.Execute(); err == nil {
				t.Fatalf("Execute(%v): want error, got nil", tc.args)
			}
		})
	}
}
