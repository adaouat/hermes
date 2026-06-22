// Package launcher defines the Launcher contract every launcher adapter
// (Alfred, Raycast, generic) implements, plus the registry that selects one
// at runtime.
package launcher

import (
	"context"
	"io"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/pkg/domain"
)

// Launcher renders domain.Items in one launcher's output format and manages
// that launcher's install lifecycle. Adding a launcher means adding an
// adapter package; it never means changing this interface or an existing
// adapter's Render/Install signature without an ADR (docs/adr/0002).
type Launcher interface {
	// Name returns the canonical launcher identifier (e.g. "alfred", "generic").
	Name() string

	// Detect reports whether e signals this launcher is the running context
	// (e.g. alfred_version for Alfred). Returns false for launchers with no
	// reliable signal.
	Detect(e env.Env) bool

	// Render writes items to w in the launcher's expected output format.
	Render(items []domain.Item, w io.Writer) error

	// Install performs launcher-specific setup. May be a no-op for launchers
	// installed out-of-band.
	Install(ctx context.Context, opts InstallOpts) error

	// Verify checks an existing install is healthy and reports findings.
	Verify(ctx context.Context, opts InstallOpts) (Report, error)
}

// InstallOpts carries the inputs every Launcher.Install/Verify needs.
type InstallOpts struct {
	DryRun     bool
	Force      bool
	BinaryPath string
	Version    string
	FS         iofs.FS
	Env        env.Env
	Out        io.Writer
}

// Report is the result of Launcher.Verify.
type Report struct {
	Installed bool
	Path      string
	Drift     []string
}
