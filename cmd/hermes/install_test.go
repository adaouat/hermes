package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/internal/launcher/alfred"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

// newAlfredTestRuntime returns a runtime whose launcher is a real alfred.Adapter and whose
// fs/env are real (iofs.New()/envtest with HOME under tmp) so installer.go's real-disk
// writes and opts.FS-backed reads observe the same files - mirroring
// internal/launcher/alfred's own TestAdapter_InstallAndVerifyDelegateToInstaller.
func newAlfredTestRuntime(t *testing.T, prefsRoot string) *runtime {
	t.Helper()
	tmp := t.TempDir()
	prefsPath := filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json")
	if err := os.MkdirAll(filepath.Dir(prefsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefsPath, []byte(`{"current": "`+prefsRoot+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	e := envtest.New(map[string]string{"HOME": tmp})
	return &runtime{
		fs:       iofs.New(),
		env:      e,
		launcher: alfred.NewAdapter(e, alfred.WithVersion("3.0.0")),
	}
}

func TestRunInstall_installsAlfredWorkflow(t *testing.T) {
	prefsRoot := filepath.Join(t.TempDir(), "Alfred.alfredpreferences")
	rt := newAlfredTestRuntime(t, prefsRoot)
	var out bytes.Buffer

	if err := runInstall(context.Background(), &out, rt, "3.0.0", false, false, ""); err != nil {
		t.Fatalf("runInstall(): %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("installed alfred workflow to")) {
		t.Errorf("output = %s, want an install confirmation", out.String())
	}
	if _, err := os.Stat(filepath.Join(prefsRoot, "workflows", "dev.adaouat.hermes.alfred", "info.plist")); err != nil {
		t.Errorf("info.plist not written: %v", err)
	}
}

func TestRunInstall_checkReportsWithoutInstalling(t *testing.T) {
	prefsRoot := filepath.Join(t.TempDir(), "Alfred.alfredpreferences")
	rt := newAlfredTestRuntime(t, prefsRoot)
	var out bytes.Buffer

	if err := runInstall(context.Background(), &out, rt, "3.0.0", true, false, ""); err != nil {
		t.Fatalf("runInstall(--check): %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("not installed")) {
		t.Errorf("output = %s, want a not-installed report", out.String())
	}
	if _, err := os.Stat(filepath.Join(prefsRoot, "workflows")); !os.IsNotExist(err) {
		t.Errorf("--check wrote to disk, want no writes")
	}
}

func TestRunInstall_checkReportsDriftWithoutOverwriting(t *testing.T) {
	prefsRoot := filepath.Join(t.TempDir(), "Alfred.alfredpreferences")
	rt := newAlfredTestRuntime(t, prefsRoot)

	if err := runInstall(context.Background(), &bytes.Buffer{}, rt, "2.0.0", false, false, ""); err != nil {
		t.Fatalf("seeding stale install: %v", err)
	}

	var out bytes.Buffer
	if err := runInstall(context.Background(), &out, rt, "3.0.0", true, false, ""); err != nil {
		t.Fatalf("runInstall(--check): %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("drift")) {
		t.Errorf("output = %s, want drift reported (installed 2.0.0, current 3.0.0)", out.String())
	}

	plist, err := os.ReadFile(filepath.Join(prefsRoot, "workflows", "dev.adaouat.hermes.alfred", "info.plist"))
	if err != nil {
		t.Fatalf("ReadFile(info.plist): %v", err)
	}
	if !bytes.Contains(plist, []byte("<string>2.0.0</string>")) {
		t.Errorf("--check overwrote the stale install; info.plist = %s", plist)
	}
}

func TestRunInstall_verifyAfterInstallReportsNoDrift(t *testing.T) {
	prefsRoot := filepath.Join(t.TempDir(), "Alfred.alfredpreferences")
	rt := newAlfredTestRuntime(t, prefsRoot)
	var out bytes.Buffer

	if err := runInstall(context.Background(), &out, rt, "3.0.0", false, true, ""); err != nil {
		t.Fatalf("runInstall(--verify): %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("installed alfred workflow to")) {
		t.Errorf("output = %s, want the install confirmation", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("no drift")) {
		t.Errorf("output = %s, want a no-drift verification report", out.String())
	}
}

func TestRunInstall_alfredNotInstalledMapsToConfigExitCode(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": t.TempDir()})
	rt := &runtime{fs: iofs.New(), env: e, launcher: alfred.NewAdapter(e, alfred.WithVersion("3.0.0"))}

	err := runInstall(context.Background(), &bytes.Buffer{}, rt, "3.0.0", false, false, "")
	if !errors.Is(err, alfred.ErrAlfredNotInstalled) {
		t.Fatalf("runInstall() error = %v, want it to wrap alfred.ErrAlfredNotInstalled", err)
	}
	if forgeexit.Resolve(err) != forgeexit.Config {
		t.Errorf("exitcode.Resolve(err) = %d, want exitcode.Config (%d)", forgeexit.Resolve(err), forgeexit.Config)
	}
}

func TestRunInstall_prefsOverrideTargetsCustomPath(t *testing.T) {
	tmp := t.TempDir()
	customPrefsRoot := filepath.Join(tmp, "custom-alfred-root")
	customPrefsPath := filepath.Join(tmp, "custom-prefs.json")
	if err := os.WriteFile(customPrefsPath, []byte(`{"current": "`+customPrefsRoot+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// HOME points somewhere with no prefs.json at all - if --prefs is ignored, install
	// fails with ErrAlfredNotInstalled instead of targeting customPrefsRoot.
	e := envtest.New(map[string]string{"HOME": filepath.Join(tmp, "unrelated-home")})
	rt := &runtime{fs: iofs.New(), env: e, launcher: alfred.NewAdapter(e, alfred.WithVersion("3.0.0"))}

	if err := runInstall(context.Background(), &bytes.Buffer{}, rt, "3.0.0", false, false, customPrefsPath); err != nil {
		t.Fatalf("runInstall(--prefs): %v", err)
	}
	if _, err := os.Stat(filepath.Join(customPrefsRoot, "workflows", "dev.adaouat.hermes.alfred", "info.plist")); err != nil {
		t.Errorf("info.plist not written under --prefs target: %v", err)
	}
}

func TestRunInstall_prefsOverrideIgnoredForNonAlfredLauncher(t *testing.T) {
	rt := &runtime{launcher: generic.NewAdapter()}

	if err := runInstall(context.Background(), &bytes.Buffer{}, rt, "3.0.0", false, false, "/some/path.json"); err != nil {
		t.Fatalf("runInstall() with generic launcher: %v", err)
	}
}

func TestNewInstallCmd_checkAndVerifyMutuallyExclusive(t *testing.T) {
	rt := &runtime{launcher: generic.NewAdapter()}
	cmd := newInstallCmd(rt, "3.0.0")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--check", "--verify"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute(--check --verify): want error, got nil")
	}
}

func TestNewInstallCmd_executesEndToEnd(t *testing.T) {
	prefsRoot := filepath.Join(t.TempDir(), "Alfred.alfredpreferences")
	rt := newAlfredTestRuntime(t, prefsRoot)
	cmd := newInstallCmd(rt, "3.0.0")
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("installed alfred workflow to")) {
		t.Errorf("output = %s, want an install confirmation", out.String())
	}
}
