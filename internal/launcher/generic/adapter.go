// Package generic implements the launcher-neutral JSON fallback: a direct
// serialization of []domain.Item with no launcher-specific envelope or
// fields. Selected via Registry.Default() when no real launcher signal is
// detected (ADR-0002 O3), and used by tests and the future Raycast TS
// extension during development.
package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

// Adapter implements launcher.Launcher for the launcher-neutral JSON output.
type Adapter struct{}

// NewAdapter returns a generic Adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Name() string { return "generic" }

// Detect always returns false — generic has no env signal of its own. It's
// selected via Registry.Default(), never via Registry.Detect.
func (a *Adapter) Detect(env.Env) bool { return false }

// Render writes items as a JSON array of launcher-neutral objects.
func (a *Adapter) Render(items []domain.Item, w io.Writer) error {
	out := make([]item, 0, len(items))
	for _, it := range items {
		out = append(out, item{
			Name:           it.Name,
			Path:           it.Path,
			IconPath:       it.IconPath,
			BinaryPath:     it.BinaryPath,
			IsModernBinary: it.IsModernBinary,
			Match:          it.Match,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding generic output: %w", err)
	}
	return nil
}

// Install is a no-op: generic has nothing to install.
func (a *Adapter) Install(context.Context, launcher.InstallOpts) error {
	return nil
}

// Verify reports an always-healthy state: there's nothing generic installs,
// so there's nothing to drift.
func (a *Adapter) Verify(context.Context, launcher.InstallOpts) (launcher.Report, error) {
	return launcher.Report{Installed: true}, nil
}

type item struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	IconPath       string `json:"iconPath"`
	BinaryPath     string `json:"binaryPath"`
	IsModernBinary bool   `json:"isModernBinary"`
	Match          string `json:"match"`
}
