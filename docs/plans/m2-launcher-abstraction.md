# M2 — Launcher Abstraction + Alfred Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Define the `Launcher` interface + registry, implement Alfred as the first concrete adapter with a byte-identical-to-2.6.2 Script Filter JSON renderer, and add the `generic` JSON-passthrough adapter — locking the launcher contract before Raycast (M5).

**Architecture:** One frozen `Launcher` interface (`internal/launcher`) consumed by two adapters this milestone (`internal/launcher/alfred`, `internal/launcher/generic`). Alfred's renderer is a pure function over `[]domain.Item` → JSON bytes, verified against a committed golden fixture. The Adapter wraps the renderer with the debug/TTL/clock concerns the legacy Dart CLI hardcoded.

**Tech Stack:** Go 1.26, `encoding/json`, `github.com/adaouat/hermes/internal/{env,iofs}` ports, `github.com/adaouat/hermes/pkg/domain`.

## Global Constraints

- TDD: write the failing test before the implementation, every task (`.claude/rules/testing.md`).
- `internal/jetbrains` and `internal/launcher` never call `os.*` directly — only `env.Env`/`iofs.FS` (already satisfied; this milestone doesn't need iofs at all).
- Every `if err != nil` wraps with `fmt.Errorf("doing X: %w", err)`.
- No comments unless the *why* is non-obvious. No panics in `internal/launcher`.
- Conventional commit per task; scope `launcher`, `launcher/alfred`, or `launcher/generic` to match the changed package.
- Byte-identity target for Alfred's compact (non-debug) JSON output is verified by a checked-in fixture at `test/fixtures/alfred/`.
- **Deviation from the M2 roadmap checklist (user-approved):** the `installer.go` + `assets/info.plist.tmpl` skeleton bullet is deferred to M4 in full, not stubbed here. The legacy `info.plist` is a real 78KB Alfred workflow with ~30 script-filter objects invoking `./bin/alfred_jetbrains_cli <cmd>` via a relative path; ADR-0001 A1 already changed the install model to an absolute resolved binary path, and N4 defers the new bundle id to M4. Templating placeholder content now would be thrown away. Task 8 records this in the roadmap note.

## Reference: legacy Alfred contract (from `bchatard/alfred-jetbrains-cli` @ 2.6.2)

Read directly from `lib/alfred/result_item.dart` and `lib/alfred/response.dart` (cloned read-only to `/tmp/alfred-jetbrains-cli` for this research; not part of this repo). `ResultItemBuilder.build()` always computes (never trusts a passed-in value):

```
uid          = name
title        = name
match        = "$name $basePath"      // basePath = basename(path)
subtitle     = path
arg          = path
autocomplete = name
text.copy        = path
text.largetype   = name
icon.path    = iconPath
icon.type    = null if extension(iconPath) == ".icns" else "fileicon"
variables    = null if binPath == null else {
  jb_project_name:     name
  jb_bin:               binPath
  jb_search_basename:   basePath
  jb_is_new_bin:        binPath.contains("MacOS")
}
```

Envelope (`renderItems`): `{"cache":{"seconds":86400,"loosereload":true},"items":[...]}`, compact (no indent) unless `alfred_debug` is set (then 2-space indent). `renderItem` (single item, no envelope) is the `[bug]` this milestone fixes — in this Go rewrite there is only one `Render([]domain.Item, io.Writer)` method, so the bug structurally cannot recur (M3's `open` command just calls `Render` with a 1-item slice).

`_addDebug` (legacy `[bug]`: mutates the caller's list) appends two items when `alfred_debug` is set: a "Debug: CLI version X" item and a "Debug: Took Nms" timer item. (A third "Debug: Log <file>" item exists in the legacy CLI; it's deferred here until M3 sets up the `slog` file handler it would point at — there is no log file to reference yet.)

---

### Task 1: `Launcher` interface, `InstallOpts`, `Report`

**Files:**
- Create: `internal/launcher/launcher.go`
- Test: `internal/launcher/launcher_test.go`

**Interfaces:**
- Produces: `launcher.Launcher` (methods `Name() string`, `Detect(e env.Env) bool`, `Render(items []domain.Item, w io.Writer) error`, `Install(ctx context.Context, opts InstallOpts) error`, `Verify(ctx context.Context, opts InstallOpts) (Report, error)`), `launcher.InstallOpts{DryRun, Force bool; BinaryPath, Version string; FS iofs.FS; Env env.Env; Out io.Writer}`, `launcher.Report{Installed bool; Path string; Drift []string}`.

- [ ] **Step 1: Write the failing test**

```go
// internal/launcher/launcher_test.go
package launcher

import "testing"

func TestInstallOpts_zeroValue(t *testing.T) {
	var opts InstallOpts
	if opts.DryRun || opts.Force {
		t.Errorf("zero InstallOpts has a true bool field: %+v", opts)
	}
	if opts.BinaryPath != "" || opts.Version != "" {
		t.Errorf("zero InstallOpts has a non-empty string field: %+v", opts)
	}
	if opts.FS != nil || opts.Env != nil || opts.Out != nil {
		t.Errorf("zero InstallOpts has a non-nil interface field: %+v", opts)
	}
}

func TestReport_fields(t *testing.T) {
	r := Report{Installed: true, Path: "/path", Drift: []string{"a", "b"}}
	if !r.Installed {
		t.Errorf("Installed = false, want true")
	}
	if r.Path != "/path" {
		t.Errorf("Path = %q, want %q", r.Path, "/path")
	}
	if len(r.Drift) != 2 {
		t.Errorf("Drift = %v, want 2 entries", r.Drift)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/launcher/... -run TestInstallOpts_zeroValue -v`
Expected: FAIL — `package launcher: no Go files` or `undefined: InstallOpts` (no `launcher.go` yet).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/launcher/launcher.go

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/launcher/... -v`
Expected: `PASS` for `TestInstallOpts_zeroValue` and `TestReport_fields`.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/launcher.go internal/launcher/launcher_test.go
git commit -m "feat(launcher): add Launcher interface, InstallOpts, Report"
```

---

### Task 2: `Registry`

**Files:**
- Create: `internal/launcher/registry.go`
- Test: `internal/launcher/registry_test.go`

**Interfaces:**
- Consumes: `launcher.Launcher` (Task 1).
- Produces: `launcher.NewRegistry(def Launcher, others ...Launcher) *Registry`, `(*Registry).Register(l Launcher)`, `(*Registry).Get(name string) (Launcher, bool)`, `(*Registry).Detect(e env.Env) (Launcher, bool)`, `(*Registry).Default() Launcher`.

- [ ] **Step 1: Write the failing test**

```go
// internal/launcher/registry_test.go
package launcher

import (
	"context"
	"io"
	"testing"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/pkg/domain"
)

type fakeLauncher struct {
	name   string
	detect bool
}

func (f fakeLauncher) Name() string                  { return f.name }
func (f fakeLauncher) Detect(env.Env) bool            { return f.detect }
func (f fakeLauncher) Render([]domain.Item, io.Writer) error { return nil }
func (f fakeLauncher) Install(context.Context, InstallOpts) error { return nil }
func (f fakeLauncher) Verify(context.Context, InstallOpts) (Report, error) {
	return Report{}, nil
}

func TestRegistry_GetReturnsRegisteredLauncher(t *testing.T) {
	def := fakeLauncher{name: "generic"}
	alfred := fakeLauncher{name: "alfred"}
	r := NewRegistry(def, alfred)

	got, ok := r.Get("alfred")
	if !ok || got.Name() != "alfred" {
		t.Errorf("Get(%q) = (%v, %v), want (alfred, true)", "alfred", got, ok)
	}

	got, ok = r.Get("generic")
	if !ok || got.Name() != "generic" {
		t.Errorf("Get(%q) = (%v, %v), want (generic, true)", "generic", got, ok)
	}

	if _, ok := r.Get("missing"); ok {
		t.Errorf("Get(missing) ok = true, want false")
	}
}

func TestRegistry_Detect(t *testing.T) {
	tests := []struct {
		name     string
		others   []Launcher
		wantOK   bool
		wantName string
	}{
		{
			name:     "first matching launcher wins",
			others:   []Launcher{fakeLauncher{name: "alfred", detect: true}, fakeLauncher{name: "raycast", detect: true}},
			wantOK:   true,
			wantName: "alfred",
		},
		{
			name:   "no match returns false",
			others: []Launcher{fakeLauncher{name: "alfred", detect: false}},
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry(fakeLauncher{name: "generic"}, tc.others...)
			got, ok := r.Detect(envtest.New(nil))
			if ok != tc.wantOK {
				t.Fatalf("Detect() ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got.Name() != tc.wantName {
				t.Errorf("Detect() = %q, want %q", got.Name(), tc.wantName)
			}
		})
	}
}

func TestRegistry_Default(t *testing.T) {
	def := fakeLauncher{name: "generic"}
	r := NewRegistry(def, fakeLauncher{name: "alfred"})

	if got := r.Default(); got.Name() != "generic" {
		t.Errorf("Default() = %q, want %q", got.Name(), "generic")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/launcher/... -run TestRegistry -v`
Expected: FAIL — `undefined: NewRegistry`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/launcher/registry.go
package launcher

import "github.com/adaouat/hermes/internal/env"

// Registry holds every available Launcher and selects one at runtime. def is
// always returned by Default and is never included in Detect's
// auto-detection loop (the generic adapter has no reliable env signal — it's
// the deliberate fallback, not something to race against real signals).
type Registry struct {
	def    Launcher
	others []Launcher
	byName map[string]Launcher
}

// NewRegistry builds a Registry. def is the launcher Default returns (the
// generic adapter in practice); others are probed in order by Detect.
func NewRegistry(def Launcher, others ...Launcher) *Registry {
	r := &Registry{def: def, byName: map[string]Launcher{def.Name(): def}}
	for _, l := range others {
		r.Register(l)
	}
	return r
}

// Register adds l to the set Get and Detect can return.
func (r *Registry) Register(l Launcher) {
	r.others = append(r.others, l)
	r.byName[l.Name()] = l
}

// Get returns the launcher registered under name.
func (r *Registry) Get(name string) (Launcher, bool) {
	l, ok := r.byName[name]
	return l, ok
}

// Detect returns the first registered (non-default) launcher whose
// Detect(e) is true, in registration order. Returns false if none match —
// callers fall back to Default() (ADR-0002 O1: "no launcher resolved is not
// an error").
func (r *Registry) Detect(e env.Env) (Launcher, bool) {
	for _, l := range r.others {
		if l.Detect(e) {
			return l, true
		}
	}
	return nil, false
}

// Default returns the launcher used when no flag, env var, or Detect signal
// resolves one (the generic adapter, per ADR-0002 O3).
func (r *Registry) Default() Launcher {
	return r.def
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/launcher/... -v`
Expected: `PASS` for all `TestRegistry_*` cases plus Task 1's tests.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/registry.go internal/launcher/registry_test.go
git commit -m "feat(launcher): add Registry with Get/Detect/Default"
```

---

### Task 3: Alfred render — result item shape + envelope

**Files:**
- Create: `internal/launcher/alfred/render.go`
- Create: `internal/launcher/alfred/render_test.go`
- Create: `test/fixtures/alfred/search_basic.json`

**Interfaces:**
- Consumes: `domain.Item{Name, Path, IconPath, BinaryPath string; IsModernBinary bool; Match string}` (existing, `pkg/domain/item.go`).
- Produces (package-private, consumed by Task 4): `buildResultItem(item domain.Item) resultItem`, `writeEnvelope(w io.Writer, env envelope, pretty bool) error`, types `resultItem`, `resultItemText`, `resultItemIcon`, `resultItemVariables`, `envelope{Cache cacheInfo; Items []resultItem}`, `cacheInfo{Seconds int; LooseReload bool}`, constants `iconNote`, `iconClock`.

- [ ] **Step 1: Write the failing test**

```go
// internal/launcher/alfred/render_test.go
package alfred

import (
	"bytes"
	"os"
	"testing"

	"github.com/adaouat/hermes/pkg/domain"
)

func TestBuildResultItem(t *testing.T) {
	tests := []struct {
		name string
		item domain.Item
		want resultItem
	}{
		{
			name: "modern binary gets fileicon and variables",
			item: domain.Item{
				Name:           "AuroraProject",
				Path:           "/Users/x/projects/aurora",
				IconPath:       "/Applications/PhpStorm.app",
				BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
				IsModernBinary: true,
				Match:          "AuroraProject aurora",
			},
			want: resultItem{
				UID: "AuroraProject", Title: "AuroraProject", Match: "AuroraProject aurora",
				Subtitle: "/Users/x/projects/aurora", Arg: "/Users/x/projects/aurora", Autocomplete: "AuroraProject",
				Text: resultItemText{Copy: "/Users/x/projects/aurora", LargeType: "AuroraProject"},
				Icon: resultItemIcon{Path: "/Applications/PhpStorm.app", Type: "fileicon"},
				Variables: &resultItemVariables{
					ProjectName: "AuroraProject", Bin: "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
					SearchBasename: "aurora", IsNewBin: true,
				},
			},
		},
		{
			name: "icns icon path has no type",
			item: domain.Item{Name: "n", Path: "/p", IconPath: "/x/icon.icns", Match: "n p"},
			want: resultItem{
				UID: "n", Title: "n", Match: "n p", Subtitle: "/p", Arg: "/p", Autocomplete: "n",
				Text: resultItemText{Copy: "/p", LargeType: "n"},
				Icon: resultItemIcon{Path: "/x/icon.icns"},
			},
		},
		{
			name: "empty binary path omits variables",
			item: domain.Item{Name: "n", Path: "/p", IconPath: "/a.app", Match: "n p"},
			want: resultItem{
				UID: "n", Title: "n", Match: "n p", Subtitle: "/p", Arg: "/p", Autocomplete: "n",
				Text: resultItemText{Copy: "/p", LargeType: "n"},
				Icon: resultItemIcon{Path: "/a.app", Type: "fileicon"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildResultItem(tc.item)
			if got.UID != tc.want.UID || got.Title != tc.want.Title || got.Match != tc.want.Match ||
				got.Subtitle != tc.want.Subtitle || got.Arg != tc.want.Arg || got.Autocomplete != tc.want.Autocomplete ||
				got.Text != tc.want.Text || got.Icon != tc.want.Icon {
				t.Errorf("buildResultItem() = %+v, want %+v", got, tc.want)
			}
			switch {
			case tc.want.Variables == nil && got.Variables != nil:
				t.Errorf("Variables = %+v, want nil", got.Variables)
			case tc.want.Variables != nil && (got.Variables == nil || *got.Variables != *tc.want.Variables):
				t.Errorf("Variables = %+v, want %+v", got.Variables, tc.want.Variables)
			}
		})
	}
}

func TestWriteEnvelope_goldenSearchBasic(t *testing.T) {
	item := domain.Item{
		Name:           "AuroraProject",
		Path:           "/Users/x/projects/aurora",
		IconPath:       "/Applications/PhpStorm.app",
		BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
		IsModernBinary: true,
		Match:          "AuroraProject aurora",
	}

	want, err := os.ReadFile("../../../test/fixtures/alfred/search_basic.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var buf bytes.Buffer
	env := envelope{Cache: cacheInfo{Seconds: 86400, LooseReload: true}, Items: []resultItem{buildResultItem(item)}}
	if err := writeEnvelope(&buf, env, false); err != nil {
		t.Fatalf("writeEnvelope(): %v", err)
	}

	if buf.String() != string(want) {
		t.Errorf("writeEnvelope() output mismatch:\ngot:  %s\nwant: %s", buf.String(), want)
	}
}
```

- [ ] **Step 2: Create the golden fixture**

```json
{"cache":{"seconds":86400,"loosereload":true},"items":[{"uid":"AuroraProject","title":"AuroraProject","match":"AuroraProject aurora","subtitle":"/Users/x/projects/aurora","arg":"/Users/x/projects/aurora","autocomplete":"AuroraProject","text":{"copy":"/Users/x/projects/aurora","largetype":"AuroraProject"},"icon":{"path":"/Applications/PhpStorm.app","type":"fileicon"},"variables":{"jb_project_name":"AuroraProject","jb_bin":"/Applications/PhpStorm.app/Contents/MacOS/phpstorm","jb_search_basename":"aurora","jb_is_new_bin":true}}]}
```

Write this as `test/fixtures/alfred/search_basic.json` with a trailing newline (matches Dart's `print()`, which always appends one).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/launcher/alfred/... -v`
Expected: FAIL — `package alfred: no Go files`.

- [ ] **Step 4: Write minimal implementation**

```go
// internal/launcher/alfred/render.go
package alfred

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/adaouat/hermes/pkg/domain"
)

const (
	iconBasePath = "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources"
	iconNote     = iconBasePath + "/AlertNoteIcon.icns"
	iconClock    = iconBasePath + "/Clock.icns"
)

// resultItem mirrors the legacy CLI's ResultItem (alfred/result_item.dart)
// field-for-field, including declaration order, so json.Marshal's key order
// stays byte-identical to the Dart json_serializable output (docs/adr/0004).
type resultItem struct {
	UID          string               `json:"uid"`
	Title        string               `json:"title"`
	Match        string               `json:"match"`
	Subtitle     string               `json:"subtitle"`
	Arg          string               `json:"arg"`
	Autocomplete string               `json:"autocomplete"`
	Text         resultItemText       `json:"text"`
	Icon         resultItemIcon       `json:"icon"`
	Variables    *resultItemVariables `json:"variables,omitempty"`
}

type resultItemText struct {
	Copy      string `json:"copy"`
	LargeType string `json:"largetype"`
}

type resultItemIcon struct {
	Path string `json:"path"`
	Type string `json:"type,omitempty"`
}

type resultItemVariables struct {
	ProjectName    string `json:"jb_project_name"`
	Bin            string `json:"jb_bin"`
	SearchBasename string `json:"jb_search_basename"`
	IsNewBin       bool   `json:"jb_is_new_bin"`
}

// envelope is the {"cache": ..., "items": [...]} shape every render uses.
// Unlike the legacy CLI (whose renderItem skipped this envelope — [bug]
// renderItem vs renderItems shape mismatch), there is only one render path
// here, so every caller gets the same shape.
type envelope struct {
	Cache cacheInfo    `json:"cache"`
	Items []resultItem `json:"items"`
}

type cacheInfo struct {
	Seconds     int  `json:"seconds"`
	LooseReload bool `json:"loosereload"`
}

func buildResultItem(item domain.Item) resultItem {
	iconType := "fileicon"
	if filepath.Ext(item.IconPath) == ".icns" {
		iconType = ""
	}

	var variables *resultItemVariables
	if item.BinaryPath != "" {
		variables = &resultItemVariables{
			ProjectName:    item.Name,
			Bin:            item.BinaryPath,
			SearchBasename: filepath.Base(item.Path),
			IsNewBin:       item.IsModernBinary,
		}
	}

	return resultItem{
		UID:          item.Name,
		Title:        item.Name,
		Match:        item.Match,
		Subtitle:     item.Path,
		Arg:          item.Path,
		Autocomplete: item.Name,
		Text:         resultItemText{Copy: item.Path, LargeType: item.Name},
		Icon:         resultItemIcon{Path: item.IconPath, Type: iconType},
		Variables:    variables,
	}
}

func writeEnvelope(w io.Writer, env envelope, pretty bool) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("encoding alfred envelope: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/launcher/alfred/... -v`
Expected: `PASS` for `TestBuildResultItem` (all 3 subtests) and `TestWriteEnvelope_goldenSearchBasic`.

- [ ] **Step 6: Commit**

```bash
git add internal/launcher/alfred/render.go internal/launcher/alfred/render_test.go test/fixtures/alfred/search_basic.json
git commit -m "feat(launcher/alfred): add Script Filter JSON renderer

Byte-identical to 2.6.2 for the same inputs, verified against a golden
fixture. Fixes [bug] renderItem/renderItems shape mismatch by having a
single render path instead of two."
```

---

### Task 4: Alfred adapter — `Name`/`Detect`/`Render`/TTL option/debug items/Install+Verify stub

**Files:**
- Create: `internal/launcher/alfred/adapter.go`
- Create: `internal/launcher/alfred/adapter_test.go`

**Interfaces:**
- Consumes: `resultItem`, `envelope`, `cacheInfo`, `buildResultItem`, `writeEnvelope`, `iconNote`, `iconClock` (Task 3); `launcher.Launcher`, `launcher.InstallOpts`, `launcher.Report` (Task 1).
- Produces: `alfred.NewAdapter(e env.Env, opts ...Option) *Adapter`, `alfred.WithTTLSeconds(int) Option`, `alfred.WithVersion(string) Option`, `alfred.WithClock(func() time.Time) Option`, `alfred.ErrInstallNotImplemented error`, and `*Adapter` satisfying `launcher.Launcher`.

- [ ] **Step 1: Write the failing test**

```go
// internal/launcher/alfred/adapter_test.go
package alfred

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestAdapter_Name(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	if got := a.Name(); got != "alfred" {
		t.Errorf("Name() = %q, want %q", got, "alfred")
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want bool
	}{
		{name: "alfred_version set", vars: map[string]string{"alfred_version": "5.5"}, want: true},
		{name: "alfred_version unset", vars: map[string]string{}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAdapter(envtest.New(nil))
			if got := a.Detect(envtest.New(tc.vars)); got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAdapter_Render_defaultTTL(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	var buf bytes.Buffer
	if err := a.Render(nil, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if got.Cache.Seconds != 86400 || !got.Cache.LooseReload {
		t.Errorf("Cache = %+v, want {86400 true}", got.Cache)
	}
}

func TestAdapter_Render_customTTL(t *testing.T) {
	a := NewAdapter(envtest.New(nil), WithTTLSeconds(60))
	var buf bytes.Buffer
	if err := a.Render(nil, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if got.Cache.Seconds != 60 {
		t.Errorf("Cache.Seconds = %d, want 60", got.Cache.Seconds)
	}
}

func TestAdapter_Render_appendsDebugItemsWithoutMutatingCaller(t *testing.T) {
	fixed := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	a := NewAdapter(
		envtest.New(map[string]string{"alfred_debug": "1"}),
		WithVersion("1.2.3"),
		WithClock(func() time.Time { return fixed }),
	)

	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}
	originalLen := len(items)

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}
	if len(items) != originalLen {
		t.Errorf("caller's items mutated: len = %d, want %d", len(items), originalLen)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("Items = %d, want 3 (1 project + 2 debug)", len(got.Items))
	}
	if got.Items[1].Title != "Debug: CLI version 1.2.3" {
		t.Errorf("Items[1].Title = %q, want %q", got.Items[1].Title, "Debug: CLI version 1.2.3")
	}
	if got.Items[2].Title != "Debug: Took 0ms" {
		t.Errorf("Items[2].Title = %q, want %q", got.Items[2].Title, "Debug: Took 0ms")
	}
}

func TestAdapter_Render_noDebugItemsWithoutDebugEnv(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 1 {
		t.Errorf("Items = %d, want 1 (no debug items)", len(got.Items))
	}
}

func TestAdapter_InstallAndVerifyNotImplemented(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	if err := a.Install(context.Background(), launcher.InstallOpts{}); !errors.Is(err, ErrInstallNotImplemented) {
		t.Errorf("Install() error = %v, want ErrInstallNotImplemented", err)
	}
	if _, err := a.Verify(context.Background(), launcher.InstallOpts{}); !errors.Is(err, ErrInstallNotImplemented) {
		t.Errorf("Verify() error = %v, want ErrInstallNotImplemented", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/launcher/alfred/... -v`
Expected: FAIL — `undefined: NewAdapter`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/launcher/alfred/adapter.go
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

const defaultTTLSeconds = 86400

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

	env := envelope{
		Cache: cacheInfo{Seconds: a.ttlSeconds, LooseReload: true},
		Items: results,
	}
	if err := writeEnvelope(w, env, debug); err != nil {
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
	version := buildResultItem(debugItem(fmt.Sprintf("Debug: CLI version %s", a.version), a.version, iconNote))

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/launcher/alfred/... -v`
Expected: `PASS` for every `TestAdapter_*` case plus Task 3's tests.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/alfred/adapter.go internal/launcher/alfred/adapter_test.go
git commit -m "feat(launcher/alfred): add Adapter implementing launcher.Launcher

Cache TTL is a constructor option (fixes [smell] hardcoded TTL); debug
items are appended to a fresh slice, never the caller's (fixes [bug]
_addDebug mutates caller list)."
```

---

### Task 5: Generic adapter (launcher-neutral JSON passthrough)

**Files:**
- Create: `internal/launcher/generic/adapter.go`
- Create: `internal/launcher/generic/adapter_test.go`

**Interfaces:**
- Consumes: `domain.Item`, `launcher.Launcher`, `launcher.InstallOpts`, `launcher.Report`.
- Produces: `generic.NewAdapter() *Adapter`, `*Adapter` satisfying `launcher.Launcher`.

- [ ] **Step 1: Write the failing test**

```go
// internal/launcher/generic/adapter_test.go
package generic

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestAdapter_Name(t *testing.T) {
	if got := NewAdapter().Name(); got != "generic" {
		t.Errorf("Name() = %q, want %q", got, "generic")
	}
}

func TestAdapter_Detect_alwaysFalse(t *testing.T) {
	if got := NewAdapter().Detect(envtest.New(map[string]string{"alfred_version": "5"})); got {
		t.Errorf("Detect() = true, want false")
	}
}

func TestAdapter_Render(t *testing.T) {
	items := []domain.Item{
		{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", IsModernBinary: true, Match: "p p"},
	}

	var buf bytes.Buffer
	if err := NewAdapter().Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got) != 1 || got[0]["name"] != "p" || got[0]["binaryPath"] != "/a.app/Contents/MacOS/p" {
		t.Errorf("Render() = %v, want one item named %q", got, "p")
	}
}

func TestAdapter_InstallAndVerify(t *testing.T) {
	a := NewAdapter()
	if err := a.Install(context.Background(), launcher.InstallOpts{}); err != nil {
		t.Errorf("Install() error = %v, want nil", err)
	}
	report, err := a.Verify(context.Background(), launcher.InstallOpts{})
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
	if !report.Installed {
		t.Errorf("Verify().Installed = false, want true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/launcher/generic/... -v`
Expected: FAIL — `package generic: no Go files`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/launcher/generic/adapter.go

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/launcher/generic/... -v`
Expected: `PASS` for every `TestAdapter_*` case.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/generic/adapter.go internal/launcher/generic/adapter_test.go
git commit -m "feat(launcher/generic): add launcher-neutral JSON adapter"
```

---

### Task 6: ADR-0004 — Alfred Script Filter contract frozen at 2.6.2

**Files:**
- Create: `docs/adr/0004-alfred-script-filter-contract-frozen-at-2.6.2.md`

- [ ] **Step 1: Write the ADR**

```markdown
# 0004 — Alfred Script Filter contract frozen at 2.6.2

## Status

Accepted

## Context

`internal/launcher/alfred` (roadmap M2) is the first concrete `Launcher`
adapter. Its `Render` output is parsed directly by Alfred from stdout, so its
shape is an external contract, not an internal implementation detail. This
ADR records what's frozen and the one deliberate behavioral change from the
2.6.2 Dart CLI.

## Decisions

### Frozen: the `ResultItem` JSON shape

Ported field-for-field from the legacy CLI's `lib/alfred/result_item.dart`
(`ResultItemBuilder.build()`), including field declaration order (so
`encoding/json`'s key order matches Dart's `json_serializable` output):

- `uid`, `title`, `autocomplete` ← project name.
- `match` ← `"<name> <basename(path)>"`.
- `subtitle`, `arg` ← project path.
- `text.copy` ← project path; `text.largetype` ← project name.
- `icon.path` ← the located `.app` bundle path; `icon.type` ← `"fileicon"`
  unless the path's extension is `.icns`, in which case the key is omitted.
- `variables` ← omitted entirely when there's no binary path; otherwise
  `jb_project_name`, `jb_bin`, `jb_search_basename`,
  `jb_is_new_bin` (`true` when the binary path contains `MacOS`).

Frozen envelope: `{"cache":{"seconds":<ttl>,"loosereload":true},"items":[...]}`,
compact unless `alfred_debug` is set (then 2-space indented). Verified
against a committed golden fixture (`test/fixtures/alfred/search_basic.json`)
per `.claude/rules/testing.md`'s golden-test rule.

### Changed: `Render` always uses the envelope

The legacy CLI's `renderItem` (used by `open`) skipped the
`{"cache":...,"items":[...]}` envelope that `renderItems` (used by `search`/
`all`) used — `[bug] renderItem vs renderItems shape mismatch`. The
`launcher.Launcher` interface has a single `Render([]domain.Item, io.Writer)
error` method, not separate `RenderItem`/`RenderItems` methods, so this
mismatch can't exist in the Go rewrite: M3's `open` command calls the same
`Render` with a one-item slice and gets the same envelope `search`/`all` get.

This is a deliberate, intentional break from 2.6.2's `open` output shape,
made possible by the interface design rather than a special case in the
Alfred adapter.

### Changed: cache TTL is configurable, debug items don't mutate the caller

- `cache.seconds` defaults to 86400 (matching 2.6.2) but is a
  `alfred.WithTTLSeconds` constructor option, not a hardcoded literal
  (`[smell] hardcoded cache TTL`).
- Debug items (`alfred_debug` set) are appended to a slice
  `Adapter.Render` allocates itself, never the caller's `[]domain.Item`
  (`[bug] _addDebug mutates caller's list`).
- The legacy debug output included a third "Debug: Log `<file>`" item
  pointing at the file logger's output. That item is deferred until roadmap
  M3 sets up the `slog` file handler it would reference — there's no log
  file to point at yet at M2.

## Consequences

- Any future change to the `ResultItem` shape, the envelope, or the
  debug-item set needs a new ADR before landing, per
  `.claude/rules/claude.md`'s "frozen external contracts" rule.
- `open`'s output shape is *not* byte-identical to 2.6.2 (it gains the
  envelope). This is accepted as a deliberate fix, not an oversight — M7's
  golden-parity work should test `search`/`all` for byte-identity and `open`
  for the new (enveloped) shape specifically.

## References

- `docs/tasks/roadmap.md` M2, M7
- `docs/specs/original-spec.md` §5.2, §10 (`[bug] renderItem vs renderItems`,
  `[bug] _addDebug mutates caller's list`, `[smell] hardcoded cache TTL`)
- [`0002-launcher-abstraction.md`](0002-launcher-abstraction.md) — the
  `Launcher` interface this adapter implements
```

- [ ] **Step 2: Commit**

```bash
git add docs/adr/0004-alfred-script-filter-contract-frozen-at-2.6.2.md
git commit -m "docs(adr): add 0004 Alfred Script Filter contract frozen at 2.6.2"
```

---

### Task 7: Wire the registry in `main`

**Files:**
- Create: `cmd/hermes/launchers.go`
- Create: `cmd/hermes/launchers_test.go`

**Interfaces:**
- Consumes: `launcher.NewRegistry`, `alfred.NewAdapter`, `alfred.WithVersion`, `generic.NewAdapter`, `env.New`.
- Produces: `newLauncherRegistry(version string, e env.Env) *launcher.Registry` — not yet called by any command (M3 wires `internal/cmd` to it); this task only satisfies the roadmap's "wired in main" bullet for the registry itself.

- [ ] **Step 1: Write the failing test**

```go
// cmd/hermes/launchers_test.go
package main

import (
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
)

func TestNewLauncherRegistry_detectsAlfred(t *testing.T) {
	r := newLauncherRegistry("1.2.3", envtest.New(nil))

	got, ok := r.Detect(envtest.New(map[string]string{"alfred_version": "5.5"}))
	if !ok || got.Name() != "alfred" {
		t.Errorf("Detect(alfred_version set) = (%v, %v), want (alfred, true)", got, ok)
	}

	_, ok = r.Detect(envtest.New(nil))
	if ok {
		t.Errorf("Detect(no signal) ok = true, want false")
	}
}

func TestNewLauncherRegistry_defaultIsGeneric(t *testing.T) {
	r := newLauncherRegistry("1.2.3", envtest.New(nil))
	if got := r.Default().Name(); got != "generic" {
		t.Errorf("Default() = %q, want %q", got, "generic")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/hermes/... -run TestNewLauncherRegistry -v`
Expected: FAIL — `undefined: newLauncherRegistry`.

- [ ] **Step 3: Write minimal implementation**

```go
// cmd/hermes/launchers.go
package main

import (
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/internal/launcher/alfred"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

// newLauncherRegistry builds the Registry every launcher-aware command (M3)
// will read from: Get/Detect/Default per ADR-0002 O1's precedence chain.
func newLauncherRegistry(version string, e env.Env) *launcher.Registry {
	return launcher.NewRegistry(
		generic.NewAdapter(),
		alfred.NewAdapter(e, alfred.WithVersion(version)),
	)
}
```

Note: `cmd/hermes/main.go` doesn't call `newLauncherRegistry` yet — `internal/cmd`'s `search`/`all`/`open` commands (M3) are what actually consume it. Go doesn't flag an unused top-level function, so this compiles and is tested standalone.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/hermes/... -v`
Expected: `PASS` for both `TestNewLauncherRegistry_*` cases.

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/launchers.go cmd/hermes/launchers_test.go
git commit -m "feat(cmd): wire launcher.Registry with alfred and generic adapters"
```

---

### Task 8: Roadmap closeout

**Files:**
- Modify: `docs/tasks/roadmap.md` (M2 section)

- [ ] **Step 1: Flip M2's checkboxes**

Mark `[ ]` → `[x]` for every M2 bullet **except** the installer/template one (deferred — see below). Add a completion note matching the M1 note's style, e.g.:

```markdown
### M2 — Launcher abstraction + Alfred adapter  (≈ 1.5 days)

**Objective:** define the `Launcher` interface and implement Alfred as the first concrete adapter. Lock the contract before adding Raycast.

- [x] `internal/launcher/launcher.go` — `Launcher` interface, `InstallOpts`, `Report` (signatures per §1).
- [x] `internal/launcher/registry.go` — `Registry` with `Register`, `Get(name)`, `Detect(env)`, `Default()`. Wired in `main`.
- [x] `internal/launcher/alfred/adapter.go` — implements `Launcher`. `Detect()` checks `alfred_version`. `Name()` returns `"alfred"`.
- [x] `internal/launcher/alfred/render.go` — Script Filter JSON renderer. **Byte-identical to 2.6.2** for the same inputs. `RenderItem` wraps in the same envelope as `RenderItems`. **Fixes [bug] renderItem/renderItems shape mismatch.**
- [x] Cache TTL exposed via constructor option, default 86400. **Fixes [smell] hardcoded TTL.**
- [x] Debug items (`alfred_debug=1`) appended via copy-on-write, no caller mutation. **Fixes [bug] `_addDebug` mutates caller list.**
- [ ] Skeleton `internal/launcher/alfred/installer.go` + `assets/info.plist.tmpl` (template ready, no `Install` impl yet — that's M4).
- [x] `internal/launcher/generic/adapter.go` — emits launcher-neutral JSON (`[]domain.Item`). Used by tests, the future Raycast TS extension, and curious users (`hermes search --product phpStorm --launcher generic`).
- [x] Golden test: Alfred JSON output for a fixture domain.Item, asserted against a 2.6.2-captured baseline.

**Note (2026-06-22):** Implemented bottom-up: `internal/launcher` (interface +
registry), then `internal/launcher/alfred` (render → adapter, in that order
so the pure JSON-shape logic could be golden-tested independently of the
debug/TTL/clock concerns the Adapter wraps it in), then
`internal/launcher/generic`, then wiring a `launcher.Registry` constructor
into `cmd/hermes` (unused until M3's commands exist, but satisfies "wired in
main"). The exact `ResultItem` field semantics (which fields are computed vs.
omitted, the `_addDebug` item set, the cache envelope) were ported from the
legacy CLI's actual `lib/alfred/result_item.dart` and `lib/alfred/response.dart`
(`bchatard/alfred-jetbrains-cli` @ 2.6.2, read via a temporary local clone),
not just the spec's summary — this caught details the spec didn't spell out
(e.g. `match` and `jb_search_basename` are always *computed* from name+path,
never passed through; `icon.type` defaults to `"fileicon"` and is omitted
only for `.icns`). One checklist item — the installer skeleton +
`assets/info.plist.tmpl` — is deferred to M4 in full rather than stubbed: the
legacy `info.plist` is a real 78KB Alfred workflow with ~30 script-filter
objects each invoking `./bin/alfred_jetbrains_cli <cmd>` via a path relative
to the workflow directory, and ADR-0001 already changed the install model to
an absolute resolved binary path (A1) while explicitly deferring the new
bundle id to M4 (N4) — templating placeholder content now would just be
thrown away when M4 designs the real workflow object graph. The third legacy
debug item ("Debug: Log `<file>`") is also deferred — M3 is what sets up the
`slog` file handler it would point at. ADR-0004 documents the frozen
`ResultItem`/envelope contract and the one deliberate behavior change
(`Render` always uses the envelope, fixing `[bug] renderItem/renderItems
shape mismatch` structurally rather than as a special case).
```

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/roadmap.md
git commit -m "docs(roadmap): close out M2 launcher abstraction (partial)"
```

---

## Verification (run after Task 8)

```bash
go test ./... -v
golangci-lint run
go build -o ./hermes ./cmd/hermes/
```

Expected: all tests pass, lint is silent, build succeeds. The installer/template deferral means M2 is not fully closed (one checkbox remains `[ ]`) — that's expected and documented in Task 8's note; it becomes part of M4's scope.
