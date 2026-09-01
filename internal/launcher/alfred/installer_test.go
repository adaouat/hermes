package alfred

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/internal/launcher"
)

func TestInstall_writesInfoPlistAndIconAtomicallyUnderRealTempDir(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")

	fs := iofstest.New(map[string]string{
		filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json"): fakePrefsJSONFor(prefsRoot),
	})
	e := envtest.New(map[string]string{"HOME": tmp})
	var out bytes.Buffer

	opts := launcher.InstallOpts{
		Version:    "3.0.0",
		BinaryPath: "/opt/homebrew/bin/hermes",
		FS:         fs,
		Env:        e,
		Out:        &out,
	}

	if err := install("", opts); err != nil {
		t.Fatalf("install(): %v", err)
	}

	target := filepath.Join(prefsRoot, "workflows", bundleID)

	plistPath := filepath.Join(target, "info.plist")
	plist, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("ReadFile(info.plist): %v", err)
	}
	if !bytes.Contains(plist, []byte("/opt/homebrew/bin/hermes open --product appCode")) {
		t.Errorf("info.plist missing rendered binary path in a script command: %s", plist)
	}
	if !bytes.Contains(plist, []byte("<string>3.0.0</string>")) {
		t.Errorf("info.plist missing rendered version")
	}
	if bytes.Contains(plist, []byte("{{.Binary}}")) || bytes.Contains(plist, []byte("{{.Version}}")) {
		t.Errorf("info.plist still contains unrendered template placeholders")
	}

	iconPath := filepath.Join(target, "icon.png")
	icon, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatalf("ReadFile(icon.png): %v", err)
	}
	if len(icon) == 0 {
		t.Errorf("icon.png is empty")
	}
}

func TestInstall_neverTouchesTheRunningBinary(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")
	binary := filepath.Join(tmp, "bin", "hermes")

	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	fs := iofstest.New(map[string]string{
		filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json"): fakePrefsJSONFor(prefsRoot),
	})
	e := envtest.New(map[string]string{"HOME": tmp})
	var out bytes.Buffer

	opts := launcher.InstallOpts{Version: "3.0.0", BinaryPath: binary, FS: fs, Env: e, Out: &out}
	if err := install("", opts); err != nil {
		t.Fatalf("install(): %v", err)
	}

	target := filepath.Join(prefsRoot, "workflows", bundleID)
	if _, err := os.Stat(filepath.Join(target, "bin")); !os.IsNotExist(err) {
		t.Errorf("install() created a bin/ dir in the workflow - the binary must never be copied")
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("ReadFile(binary): %v", err)
	}
	if string(content) != "fake binary" {
		t.Errorf("running binary was modified by install()")
	}
}

func TestInstall_dryRunWritesNothing(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")

	fs := iofstest.New(map[string]string{
		filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json"): fakePrefsJSONFor(prefsRoot),
	})
	e := envtest.New(map[string]string{"HOME": tmp})
	var out bytes.Buffer

	opts := launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/opt/homebrew/bin/hermes", DryRun: true, FS: fs, Env: e, Out: &out}
	if err := install("", opts); err != nil {
		t.Fatalf("install(): %v", err)
	}

	target := filepath.Join(prefsRoot, "workflows", bundleID)
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("install() with DryRun wrote to %s, want no filesystem writes", target)
	}
	if out.Len() == 0 {
		t.Errorf("install() with DryRun wrote nothing to Out, want a description of what would happen")
	}
}

func TestInstall_missingPrefsJSONReturnsErrAlfredNotInstalled(t *testing.T) {
	tmp := t.TempDir()
	fs := iofstest.New(nil)
	e := envtest.New(map[string]string{"HOME": tmp})
	var out bytes.Buffer

	err := install("", launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/bin/hermes", FS: fs, Env: e, Out: &out})
	if !errors.Is(err, ErrAlfredNotInstalled) {
		t.Errorf("install() error = %v, want ErrAlfredNotInstalled", err)
	}
}

func TestInstall_prefsPathOverrideWinsOverDefault(t *testing.T) {
	tmp := t.TempDir()
	overrideRoot := filepath.Join(tmp, "override-root")
	overridePath := filepath.Join(tmp, "override-prefs.json")
	if err := os.WriteFile(overridePath, []byte(fakePrefsJSONFor(overrideRoot)), 0o644); err != nil {
		t.Fatal(err)
	}

	// No prefs.json at the default HOME-based location - proves the override, not the
	// default, is what install() actually read.
	fs := iofstest.New(map[string]string{overridePath: fakePrefsJSONFor(overrideRoot)})
	e := envtest.New(map[string]string{"HOME": filepath.Join(tmp, "unrelated-home")})
	var out bytes.Buffer

	if err := install(overridePath, launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/bin/hermes", FS: fs, Env: e, Out: &out}); err != nil {
		t.Fatalf("install(): %v", err)
	}

	target := filepath.Join(overrideRoot, "workflows", bundleID)
	if _, err := os.Stat(filepath.Join(target, "info.plist")); err != nil {
		t.Errorf("info.plist not written under the override root: %v", err)
	}
}

func TestVerify_notInstalledReportsFalse(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")

	fs := iofstest.New(map[string]string{
		filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json"): fakePrefsJSONFor(prefsRoot),
	})
	e := envtest.New(map[string]string{"HOME": tmp})

	report, err := verify("", launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/bin/hermes", FS: fs, Env: e})
	if err != nil {
		t.Fatalf("verify(): %v", err)
	}
	if report.Installed {
		t.Errorf("Installed = true, want false (nothing installed yet)")
	}
}

func TestVerify_installedNoDriftMatchesFreshInstall(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")
	target := filepath.Join(prefsRoot, "workflows", bundleID)

	plist, err := renderInfoPlist("3.0.0", "/opt/homebrew/bin/hermes")
	if err != nil {
		t.Fatalf("renderInfoPlist(): %v", err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "info.plist"), plist, 0o644); err != nil {
		t.Fatal(err)
	}

	fs := iofstest.New(map[string]string{
		filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json"): fakePrefsJSONFor(prefsRoot),
		filepath.Join(target, "info.plist"):                                 string(plist),
	})
	e := envtest.New(map[string]string{"HOME": tmp})

	report, err := verify("", launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/opt/homebrew/bin/hermes", FS: fs, Env: e})
	if err != nil {
		t.Fatalf("verify(): %v", err)
	}
	if !report.Installed {
		t.Errorf("Installed = false, want true")
	}
	if len(report.Drift) != 0 {
		t.Errorf("Drift = %v, want none (freshly installed content matches)", report.Drift)
	}
}

func TestVerify_driftWhenInstalledVersionDiffers(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")
	target := filepath.Join(prefsRoot, "workflows", bundleID)

	stale, err := renderInfoPlist("2.0.0", "/opt/homebrew/bin/hermes")
	if err != nil {
		t.Fatalf("renderInfoPlist(): %v", err)
	}

	fs := iofstest.New(map[string]string{
		filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json"): fakePrefsJSONFor(prefsRoot),
		filepath.Join(target, "info.plist"):                                 string(stale),
	})
	e := envtest.New(map[string]string{"HOME": tmp})

	report, err := verify("", launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/opt/homebrew/bin/hermes", FS: fs, Env: e})
	if err != nil {
		t.Fatalf("verify(): %v", err)
	}
	if !report.Installed {
		t.Errorf("Installed = false, want true")
	}
	if len(report.Drift) == 0 {
		t.Errorf("Drift = none, want drift reported (installed version 2.0.0 != current 3.0.0)")
	}
}

func TestAdapter_InstallAndVerifyDelegateToInstaller(t *testing.T) {
	tmp := t.TempDir()
	prefsRoot := filepath.Join(tmp, "Alfred.alfredpreferences")
	prefsPath := filepath.Join(tmp, "Library/Application Support/Alfred/prefs.json")

	if err := os.MkdirAll(filepath.Dir(prefsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefsPath, []byte(fakePrefsJSONFor(prefsRoot)), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify reads through opts.FS, and Install's writes go straight to the real disk
	// (installer.go's doc comment on install explains why) - only the real iofs.FS observes
	// both, so this round-trip test needs it instead of the in-memory fake the other tests
	// in this file use.
	fs := iofs.New()
	e := envtest.New(map[string]string{"HOME": tmp})
	var out bytes.Buffer

	a := NewAdapter(e)
	opts := launcher.InstallOpts{Version: "3.0.0", BinaryPath: "/opt/homebrew/bin/hermes", FS: fs, Env: e, Out: &out}

	if err := a.Install(context.Background(), opts); err != nil {
		t.Fatalf("Adapter.Install(): %v", err)
	}

	report, err := a.Verify(context.Background(), opts)
	if err != nil {
		t.Fatalf("Adapter.Verify(): %v", err)
	}
	if !report.Installed {
		t.Errorf("Installed = false, want true after Install()")
	}
}

func fakePrefsJSONFor(prefsRoot string) string {
	return `{"current": "` + prefsRoot + `"}`
}
