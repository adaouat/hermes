package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/internal/launcher/alfred"
)

func newInstallCmd(rt *runtime, version string) *cobra.Command {
	var check, verifyAfter bool
	var prefsPath string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the resolved launcher's integration (e.g. the Alfred workflow)",
		RunE: func(c *cobra.Command, _ []string) error {
			return runInstall(c.Context(), c.OutOrStdout(), rt, version, check, verifyAfter, prefsPath)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Report drift without installing anything (dry run)")
	cmd.Flags().BoolVar(&verifyAfter, "verify", false, "Validate the install after writing it")
	cmd.Flags().StringVar(&prefsPath, "prefs", "", "Override the Alfred prefs.json path (escape hatch)")
	cmd.MarkFlagsMutuallyExclusive("check", "verify")
	return cmd
}

func runInstall(ctx context.Context, w io.Writer, rt *runtime, version string, check, verifyAfter bool, prefsPath string) error {
	l := targetLauncher(rt.launcher, rt.env, version, prefsPath)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving hermes binary path: os.Executable: %w", err)
	}
	binary, err := stableBinaryPath(exe, rt.env.Path())
	if err != nil {
		return fmt.Errorf("resolving hermes binary path: %w", err)
	}
	opts := launcher.InstallOpts{Version: version, BinaryPath: binary, FS: rt.fs, Env: rt.env, Out: w}

	if check {
		return reportVerify(ctx, w, l, opts)
	}

	if err := l.Install(ctx, opts); err != nil {
		return mapInstallErr(err)
	}

	if verifyAfter {
		return reportVerify(ctx, w, l, opts)
	}
	return nil
}

// targetLauncher applies --prefs (alfred-specific, so it's only meaningful when the
// resolved launcher is alfred) by constructing a fresh alfred.Adapter rather than mutating
// l - the shared registry's adapter has no setter, by design (adapter.go's Options are
// construction-time only).
func targetLauncher(l launcher.Launcher, e env.Env, version, prefsPath string) launcher.Launcher {
	if prefsPath == "" || l.Name() != "alfred" {
		return l
	}
	return alfred.NewAdapter(e, alfred.WithVersion(version), alfred.WithPrefsPath(prefsPath))
}

func reportVerify(ctx context.Context, w io.Writer, l launcher.Launcher, opts launcher.InstallOpts) error {
	report, err := l.Verify(ctx, opts)
	if err != nil {
		return mapInstallErr(err)
	}

	if !report.Installed {
		_, err := fmt.Fprintf(w, "not installed (target: %s)\n", report.Path)
		return err
	}
	if len(report.Drift) == 0 {
		_, err := fmt.Fprintf(w, "installed, no drift (%s)\n", report.Path)
		return err
	}
	if _, err := fmt.Fprintf(w, "installed, drift detected (%s):\n", report.Path); err != nil {
		return err
	}
	for _, d := range report.Drift {
		if _, err := fmt.Fprintf(w, "  - %s\n", d); err != nil {
			return err
		}
	}
	return nil
}

// mapInstallErr maps launcher-specific install/verify errors to forge exit codes at the
// cmd boundary (coding.md: sentinels/typed errors are exposed at package boundaries for
// exactly this). ErrAlfredNotInstalled -> exitcode.Config per ADR-0001 A5.
func mapInstallErr(err error) error {
	if errors.Is(err, alfred.ErrAlfredNotInstalled) {
		return forgeexit.WrapSummary(forgeexit.Config, err, "Alfred isn't installed - get it from https://www.alfredapp.com")
	}
	return err
}

// stableBinaryPath returns the path install records for the running binary exe: the first
// hermes on path that is the same file as exe, else exe as invoked. Never the
// symlink-resolved target - under Homebrew that is Caskroom/hermes/<version>/hermes, which
// `brew upgrade` deletes, whereas bin/hermes survives it (ADR-0009).
func stableBinaryPath(exe string, path []string) (string, error) {
	exeInfo, err := os.Stat(exe)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", exe, err)
	}
	for _, dir := range path {
		if !filepath.IsAbs(dir) {
			continue
		}
		candidate := filepath.Join(dir, filepath.Base(exe))
		info, err := os.Stat(candidate)
		if err == nil && os.SameFile(exeInfo, info) {
			return candidate, nil
		}
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return "", fmt.Errorf("absolute path of %s: %w", exe, err)
	}
	return abs, nil
}
