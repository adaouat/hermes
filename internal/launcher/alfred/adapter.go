package alfred

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

const (
	defaultTTLSeconds = 86400

	iconBasePath = "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources"
	iconNote     = iconBasePath + "/AlertNoteIcon.icns"
	iconClock    = iconBasePath + "/Clock.icns"
)

// ErrInstallNotImplemented is returned by Adapter.Install and Verify until
// the real implementation lands in roadmap M4.
var ErrInstallNotImplemented = errors.New("alfred: install not implemented yet (see roadmap M4)")

// Adapter implements launcher.Launcher for Alfred's Script Filter contract.
type Adapter struct {
	env        env.Env
	ttlSeconds int
	version    string
	now        func() time.Time
}

// Option configures an Adapter built by NewAdapter.
type Option func(*Adapter)

// WithTTLSeconds overrides the cache.seconds field Render emits. Fixes
// [smell] hardcoded cache TTL (60*60*24 baked into the legacy response.dart).
func WithTTLSeconds(seconds int) Option {
	return func(a *Adapter) { a.ttlSeconds = seconds }
}

// WithVersion sets the version string the debug "CLI version" item reports.
func WithVersion(version string) Option {
	return func(a *Adapter) { a.version = version }
}

// WithClock overrides the clock Render uses for the debug timer item. Tests
// inject a fixed clock; production uses the default, time.Now.
func WithClock(now func() time.Time) Option {
	return func(a *Adapter) { a.now = now }
}

// NewAdapter returns an Adapter reading debug/detect signals from e.
func NewAdapter(e env.Env, opts ...Option) *Adapter {
	a := &Adapter{env: e, ttlSeconds: defaultTTLSeconds, now: time.Now}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Adapter) Name() string { return "alfred" }

// Detect reports whether alfred_version is set — the signal Alfred itself
// sets when invoking a Script Filter (docs/adr/0002 O2).
func (a *Adapter) Detect(e env.Env) bool {
	_, ok := e.Lookup("alfred_version")
	return ok
}

// Render writes items as Alfred Script Filter JSON:
// {"cache":...,"items":[...]}. Every call uses this envelope, including the
// single-item "open" case — the legacy CLI's renderItem skipped it ([bug]
// renderItem vs renderItems shape mismatch); here there is only one Render
// path, so the bug can't recur.
func (a *Adapter) Render(items []domain.Item, w io.Writer) error {
	start := a.now()

	results := make([]resultItem, 0, len(items))
	for _, item := range items {
		results = append(results, buildResultItem(item))
	}

	debug := a.debugMode()
	if debug {
		results = append(results, a.debugItems(start)...)
	}

	ef := envelope{
		Cache: cacheInfo{Seconds: a.ttlSeconds, LooseReload: true},
		Items: results,
	}
	if err := writeEnvelope(w, ef, debug); err != nil {
		return fmt.Errorf("rendering alfred output: %w", err)
	}
	return nil
}

func (a *Adapter) debugMode() bool {
	_, ok := a.env.Lookup("alfred_debug")
	return ok
}

// debugItems mirrors the legacy _addDebug's version and timer items. Unlike
// _addDebug ([bug] mutates caller's list), Render above appends these to a
// fresh []resultItem it owns, never the caller's []domain.Item. The third
// legacy item ("Debug: Log <file>") is deferred to M3, which is what sets up
// the slog file handler this item would point at.
func (a *Adapter) debugItems(start time.Time) []resultItem {
	v := a.version
	if v == "" {
		v = "unknown"
	}
	version := buildResultItem(debugItem(fmt.Sprintf("Debug: CLI version %s", v), v, iconNote))

	stop := a.now()
	took := fmt.Sprintf("Debug: Took %dms", stop.Sub(start).Milliseconds())
	timerPath := fmt.Sprintf("Started at: %s || Ended at: %s",
		start.Format("2006-01-02T15:04:05.000"), stop.Format("2006-01-02T15:04:05.000"))
	timer := buildResultItem(debugItem(took, timerPath, iconClock))

	return []resultItem{version, timer}
}

func debugItem(name, path, iconPath string) domain.Item {
	return domain.Item{
		Name:     name,
		Path:     path,
		IconPath: iconPath,
		Match:    name + " " + filepath.Base(path),
	}
}

// Install is not implemented until roadmap M4 (the Alfred prefs.json install).
func (a *Adapter) Install(_ context.Context, _ launcher.InstallOpts) error {
	return ErrInstallNotImplemented
}

// Verify is not implemented until roadmap M4.
func (a *Adapter) Verify(_ context.Context, _ launcher.InstallOpts) (launcher.Report, error) {
	return launcher.Report{}, ErrInstallNotImplemented
}
