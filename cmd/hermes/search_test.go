package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/internal/launcher/generic"
	"github.com/adaouat/hermes/pkg/domain"
)

// errRenderFailed is failingLauncher's sentinel Render error, asserted with errors.Is in
// tests that need to prove a command wraps the launcher's error rather than swallowing it.
var errRenderFailed = errors.New("render boom")

// failingLauncher is a launcher.Launcher whose Render always fails, for exercising the
// run*'s "rendering ...: %w" wrap branch without needing a real launcher to break.
type failingLauncher struct{}

func (failingLauncher) Name() string        { return "failing" }
func (failingLauncher) Detect(env.Env) bool { return false }
func (failingLauncher) Render([]domain.Item, io.Writer) error {
	return errRenderFailed
}
func (failingLauncher) Install(context.Context, launcher.InstallOpts) error { return nil }
func (failingLauncher) Verify(context.Context, launcher.InstallOpts) (launcher.Report, error) {
	return launcher.Report{}, nil
}

func phpStormFixture() map[string]string {
	return map[string]string{
		"/Applications/PhpStorm.app/Contents/MacOS/phpstorm":   "",
		"/home/x/Library/Preferences/PhpStorm2024.1/other.txt": "",
		"/home/x/Library/Preferences/PhpStorm2024.1/options/recentProjectDirectories.xml": `<application>
  <component name="RecentDirectoryProjectsManager">
    <option name="recentPaths">
      <list>
        <option value="$USER_HOME$/projects/aurora" />
      </list>
    </option>
  </component>
</application>`,
		"/home/x/projects/aurora/.idea/name": "AuroraProject",
	}
}

func newTestRuntime(files map[string]string) *runtime {
	return &runtime{
		fs:       iofstest.New(files),
		env:      envtest.New(map[string]string{"HOME": "/home/x"}),
		config:   jetbrains.Defaults(),
		launcher: generic.NewAdapter(),
	}
}

func TestRunSearch_rendersItems(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runSearch(&buf, rt, "phpStorm", ""); err != nil {
		t.Fatalf("runSearch(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestRunSearch_unknownProductReturnsError(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runSearch(&buf, rt, "notAProduct", ""); err == nil {
		t.Fatal("runSearch(unknown product): want error, got nil")
	}
}

func TestRunSearch_notFoundWrapsExitCode(t *testing.T) {
	rt := newTestRuntime(nil) // no PhpStorm.app fixture -> NotFoundError
	var buf bytes.Buffer

	err := runSearch(&buf, rt, "phpStorm", "")
	if err == nil {
		t.Fatal("runSearch(): want error, got nil")
	}
	if forgeexit.Resolve(err) != exitNotFound {
		t.Errorf("exitcode.Resolve(err) = %d, want %d", forgeexit.Resolve(err), exitNotFound)
	}
	var notFound *jetbrains.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error chain does not contain *jetbrains.NotFoundError")
	}
}

func TestRunSearch_nonNotFoundErrorPassesThrough(t *testing.T) {
	files := phpStormFixture()
	files["/home/x/Library/Preferences/PhpStorm2024.1/options/recentProjectDirectories.xml"] = "not valid xml <<<"
	rt := newTestRuntime(files)
	var buf bytes.Buffer

	err := runSearch(&buf, rt, "phpStorm", "")
	if err == nil {
		t.Fatal("runSearch() with malformed XML: want error, got nil")
	}
	var notFound *jetbrains.NotFoundError
	if errors.As(err, &notFound) {
		t.Errorf("error = %v, want a non-NotFoundError (XML parse failure)", err)
	}
	if forgeexit.Resolve(err) == exitNotFound {
		t.Errorf("exitcode.Resolve(err) = %d, want anything but %d", forgeexit.Resolve(err), exitNotFound)
	}
}

func TestRunSearch_renderErrorWrapped(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	rt.launcher = failingLauncher{}
	var buf bytes.Buffer

	if err := runSearch(&buf, rt, "phpStorm", ""); !errors.Is(err, errRenderFailed) {
		t.Errorf("runSearch() error = %v, want it to wrap errRenderFailed", err)
	}
}

func TestNewSearchCmd_executesEndToEnd(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	cmd := newSearchCmd(rt)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--product", "phpStorm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestNewSearchCmd_missingProductFlagErrors(t *testing.T) {
	rt := newTestRuntime(nil)
	cmd := newSearchCmd(rt)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute() without --product: want error, got nil")
	}
}
