package raycast

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/adaouat/hermes/internal/launcher"
)

// storeNote points users at the JSON contract docs until the TS extension is published to
// the Raycast Store - out of scope for 3.0 (roadmap §7, ADR-0002 R2/R3).
const storeNote = "Raycast extension not yet published to the Raycast Store - see " +
	"https://github.com/adaouat/hermes/tree/main/extensions/raycast for the JSON contract."

// onPath reports whether dir (the resolved binary's directory) is one of path's entries -
// what "reachable via $PATH" means for a Raycast Store extension that shells out by name
// (ADR-0002 R1/R4).
func onPath(dir string, path []string) bool {
	return slices.Contains(path, dir)
}

// install never writes anything - Raycast has no bundle to install (ADR-0002 R1). It reports
// $PATH reachability to opts.Out so `hermes install --launcher raycast` gives the user
// actionable feedback either way.
func install(opts launcher.InstallOpts) error {
	dir := filepath.Dir(opts.BinaryPath)
	reachable := onPath(dir, opts.Env.Path())

	if opts.DryRun {
		_, err := fmt.Fprintf(opts.Out, "would verify hermes is on $PATH (checked %s)\n", dir)
		return err
	}
	if !reachable {
		_, err := fmt.Fprintf(opts.Out, "hermes is not on $PATH (checked %s) - install it via brew/mise first.\n%s\n", dir, storeNote)
		return err
	}
	_, err := fmt.Fprintf(opts.Out, "hermes is reachable on $PATH (%s).\n%s\n", dir, storeNote)
	return err
}

// verify reports $PATH reachability as the Report (ADR-0002 R1/R4): Installed mirrors
// reachability since Raycast has no other install state to check, and a missing PATH entry
// is reported as drift.
func verify(opts launcher.InstallOpts) (launcher.Report, error) {
	dir := filepath.Dir(opts.BinaryPath)
	reachable := onPath(dir, opts.Env.Path())

	report := launcher.Report{Installed: reachable, Path: dir}
	if !reachable {
		report.Drift = append(report.Drift, fmt.Sprintf("hermes is not on $PATH (checked %s)", dir))
	}
	return report, nil
}
