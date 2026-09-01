package alfred

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/internal/launcher"
)

// bundleID is this workflow's folder name under <Alfred root>/workflows/. Chosen fresh for
// hermes per ADR-0001 N4 - it replaces (not aliases) the 2.x CLI's fr.chatard.jetbrains.workflow,
// so every existing install is orphaned and must be recreated via `hermes install`.
const bundleID = "dev.adaouat.hermes.alfred"

//go:embed assets/info.plist.tmpl
var infoPlistTmplSource string

//go:embed assets/icon.png
var iconPNG []byte

// ErrAlfredNotInstalled means prefs.json is missing - Alfred itself isn't installed on this
// machine (ADR-0001 A5).
var ErrAlfredNotInstalled = errors.New("alfred: prefs.json not found - is Alfred installed?")

type alfredPrefs struct {
	Current string `json:"current"`
}

// defaultPrefsPath is where Alfred keeps its own preferences root pointer (docs/specs/original-spec.md
// §5.6.1).
func defaultPrefsPath(e env.Env) string {
	return filepath.Join(e.Home(), "Library", "Application Support", "Alfred", "prefs.json")
}

func workflowTarget(prefsRoot string) string {
	return filepath.Join(prefsRoot, "workflows", bundleID)
}

// resolvePrefsPath honors the WithPrefsPath escape hatch when set, else falls back to
// defaultPrefsPath.
func resolvePrefsPath(override string, e env.Env) string {
	if override != "" {
		return override
	}
	return defaultPrefsPath(e)
}

// readPrefsRoot reads prefs.json's "current" key: Alfred's active preferences root, under
// which every workflow lives (docs/specs/original-spec.md §5.6.2).
func readPrefsRoot(fs iofs.FS, prefsPath string) (string, error) {
	if !fs.Exists(prefsPath) {
		return "", ErrAlfredNotInstalled
	}
	raw, err := fs.ReadFile(prefsPath)
	if err != nil {
		return "", fmt.Errorf("reading alfred prefs %s: %w", prefsPath, err)
	}
	var prefs alfredPrefs
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return "", fmt.Errorf("parsing alfred prefs %s: %w", prefsPath, err)
	}
	if prefs.Current == "" {
		return "", fmt.Errorf("alfred prefs %s: missing \"current\" key", prefsPath)
	}
	return prefs.Current, nil
}

// renderInfoPlist fills the embedded info.plist template with the resolved binary path and
// CLI version. ADR-0001 A1: the binary path is always the caller's already-resolved absolute
// path (os.Executable() + EvalSymlinks, done by the cmd/hermes composition root) - this
// function never touches the binary itself, only writes its path as text.
func renderInfoPlist(version, binary string) ([]byte, error) {
	tmpl, err := template.New("info.plist").Parse(infoPlistTmplSource)
	if err != nil {
		return nil, fmt.Errorf("parsing embedded info.plist template: %w", err)
	}
	var buf bytes.Buffer
	data := struct{ Version, Binary string }{Version: version, Binary: binary}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("rendering info.plist: %w", err)
	}
	return buf.Bytes(), nil
}

// writeFileAtomically writes data to path via a tmp file in the same directory followed by
// os.Rename, so a reader never observes a partially-written info.plist/icon.png.
func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".hermes-install-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, path, err)
	}
	return nil
}

// install implements ADR-0001's install model: never move or copy the running binary (A1),
// write info.plist + icon.png atomically (roadmap M4). opts.FS/opts.Env cover every read
// (prefs.json); the writes below use the real os package directly because iofs.FS is a
// read-only port (see internal/iofs's doc comment) - there is no write-side abstraction to
// go through, and testing.md's determinism rule allows real disk access scoped to
// t.TempDir(), which is how installer_test.go exercises this path.
func install(prefsPathOverride string, opts launcher.InstallOpts) error {
	prefsRoot, err := readPrefsRoot(opts.FS, resolvePrefsPath(prefsPathOverride, opts.Env))
	if err != nil {
		return err
	}
	target := workflowTarget(prefsRoot)

	plist, err := renderInfoPlist(opts.Version, opts.BinaryPath)
	if err != nil {
		return err
	}

	if opts.DryRun {
		if _, err := fmt.Fprintf(opts.Out, "would install alfred workflow to %s\n", target); err != nil {
			return fmt.Errorf("writing dry-run report: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("creating workflow dir %s: %w", target, err)
	}
	if err := writeFileAtomically(filepath.Join(target, "info.plist"), plist, 0o644); err != nil {
		return err
	}
	if err := writeFileAtomically(filepath.Join(target, "icon.png"), iconPNG, 0o644); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(opts.Out, "installed alfred workflow to %s\n", target); err != nil {
		return fmt.Errorf("writing install report: %w", err)
	}
	return nil
}

// verify reports whether the workflow is installed and, if so, whether its info.plist
// matches what the current version/binary would render (ADR-0001 A3/A4 drift detection).
func verify(prefsPathOverride string, opts launcher.InstallOpts) (launcher.Report, error) {
	prefsRoot, err := readPrefsRoot(opts.FS, resolvePrefsPath(prefsPathOverride, opts.Env))
	if err != nil {
		return launcher.Report{}, err
	}
	target := workflowTarget(prefsRoot)
	infoPlistPath := filepath.Join(target, "info.plist")

	if !opts.FS.Exists(infoPlistPath) {
		return launcher.Report{Installed: false, Path: target}, nil
	}

	want, err := renderInfoPlist(opts.Version, opts.BinaryPath)
	if err != nil {
		return launcher.Report{}, err
	}
	got, err := opts.FS.ReadFile(infoPlistPath)
	if err != nil {
		return launcher.Report{}, fmt.Errorf("reading installed info.plist %s: %w", infoPlistPath, err)
	}

	report := launcher.Report{Installed: true, Path: target}
	if !bytes.Equal(want, got) {
		report.Drift = append(report.Drift, "info.plist differs from the current version/binary - re-run `hermes install`")
	}
	return report, nil
}
