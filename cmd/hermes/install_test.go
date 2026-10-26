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
	rt := &runtime{env: envtest.New(nil), launcher: generic.NewAdapter()}

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

// brewLayout mimics a Homebrew cask install under root: the real binary lives in the
// version-specific root/Caskroom/hermes/<version>/hermes and root/bin/hermes symlinks to it.
func brewLayout(t *testing.T, root, version string) (bin, versioned string) {
	t.Helper()
	versioned = filepath.Join(root, "Caskroom", "hermes", version, "hermes")
	bin = filepath.Join(root, "bin", "hermes")
	if err := os.MkdirAll(filepath.Dir(versioned), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(versioned, []byte("hermes "+version), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(bin)
	if err := os.Symlink(versioned, bin); err != nil {
		t.Fatal(err)
	}
	return bin, versioned
}

func TestStableBinaryPath(t *testing.T) {
	root := t.TempDir()
	bin, versioned := brewLayout(t, root, "0.2.0")

	other := filepath.Join(root, "other", "hermes")
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("a different hermes build"), 0o755); err != nil {
		t.Fatal(err)
	}

	plain := filepath.Join(root, ".local", "bin", "hermes")
	if err := os.MkdirAll(filepath.Dir(plain), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte("plain hermes"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		exe  string
		path []string
		want string
	}{
		{"invoked via brew bin symlink on PATH", bin, []string{filepath.Dir(bin)}, bin},
		{"invoked via versioned Caskroom path, brew bin on PATH", versioned, []string{"/usr/bin", filepath.Dir(bin)}, bin},
		{"different hermes earlier on PATH is skipped", bin, []string{filepath.Dir(other), filepath.Dir(bin)}, bin},
		{"relative PATH entry is ignored", bin, []string{"bin"}, bin},
		{"not on PATH falls back to the invoked path, unresolved", bin, nil, bin},
		{"plain binary on PATH", plain, []string{filepath.Dir(plain)}, plain},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stableBinaryPath(tc.exe, tc.path)
			if err != nil {
				t.Fatalf("stableBinaryPath(): %v", err)
			}
			if got != tc.want {
				t.Errorf("stableBinaryPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStableBinaryPath_survivesBrewUpgrade(t *testing.T) {
	root := t.TempDir()
	bin, oldVersioned := brewLayout(t, root, "0.2.0")

	recorded, err := stableBinaryPath(oldVersioned, []string{filepath.Dir(bin)})
	if err != nil {
		t.Fatalf("stableBinaryPath(): %v", err)
	}

	_, newVersioned := brewLayout(t, root, "0.2.1")
	if err := os.RemoveAll(filepath.Dir(oldVersioned)); err != nil {
		t.Fatal(err)
	}

	resolved, err := filepath.EvalSymlinks(recorded)
	if err != nil {
		t.Fatalf("recorded path %q dangles after upgrade: %v", recorded, err)
	}
	wantResolved, err := filepath.EvalSymlinks(newVersioned)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != wantResolved {
		t.Errorf("recorded path %q resolves to %q, want the upgraded %q", recorded, resolved, wantResolved)
	}
}
