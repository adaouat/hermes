package main

import (
	"bytes"
	"errors"
	"testing"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

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
