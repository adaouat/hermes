// Package raycast implements launcher.Launcher for the Raycast JSON contract (docs/adr/0005).
// The TypeScript extension that consumes this output lives out-of-tree for 3.0
// (roadmap §7); this package ships the Go-side of the contract only.
package raycast

import (
	"context"
	"io"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

// Adapter implements launcher.Launcher for Raycast.
type Adapter struct{}

// NewAdapter returns a raycast Adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Name() string { return "raycast" }

// Detect reports whether RAYCAST_VERSION is set - the signal the future extensions/raycast
// TS code sets explicitly before shelling out to hermes. Unlike Alfred, Raycast doesn't
// auto-inject anything into a spawned child process, so our own extension owns setting this
// on both ends of the contract (docs/adr/0005).
func (a *Adapter) Detect(e env.Env) bool {
	_, ok := e.Lookup("RAYCAST_VERSION")
	return ok
}

// Render writes items as the {"items": [...]} Raycast JSON contract (render.go).
func (a *Adapter) Render(items []domain.Item, w io.Writer) error {
	return writeEnvelope(w, items)
}

// Install is a no-op: Raycast Store extensions install themselves (ADR-0002 R1). It reports
// whether hermes is reachable on $PATH so the caller knows whether to point the user at
// brew/mise (installer.go).
func (a *Adapter) Install(_ context.Context, opts launcher.InstallOpts) error {
	return install(opts)
}

// Verify checks whether hermes is reachable on $PATH, reporting drift if not (installer.go).
func (a *Adapter) Verify(_ context.Context, opts launcher.InstallOpts) (launcher.Report, error) {
	return verify(opts)
}
