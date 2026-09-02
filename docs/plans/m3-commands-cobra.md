# M3 — Commands (cobra) + forge wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire `search`, `all`, `open`, and `configuration` end-to-end through the domain core (`internal/jetbrains`, M1) and the launcher abstraction (`internal/launcher`, M2), with structured exit codes, launcher selection precedence, and slog-based logging.

**Architecture:** Every cobra command lives in `cmd/hermes` (package `main`) alongside the existing `root.go`/`launchers.go` scaffolding — this repo already deviated from the roadmap's original `internal/cmd/` sketch during M0/M2 (see `docs/tasks/roadmap.md`'s M0 completion note); this plan follows that established precedent rather than reopening it. A small `runtime` struct is the composition root's output: `root.go`'s `PersistentPreRunE` builds the real `iofs.FS`/`env.Env`, merges configuration, sets up logging, and resolves the active launcher once per invocation; each command's `RunE` is a thin cobra wrapper around a plain `run*(w io.Writer, rt *runtime, ...) error` function that contains the actual logic and needs no cobra machinery to test.

**Tech Stack:** Go 1.27, `github.com/spf13/cobra`, `github.com/adaouat/forge/exitcode`, `github.com/adaouat/hermes/internal/{env,iofs,jetbrains,launcher}`, `github.com/adaouat/hermes/pkg/domain`.

**Spec:** `docs/tasks/roadmap.md` M3; `docs/specs/original-spec.md` §3 (commands), §5.1–§5.3 (env vars, output formats, discovery), §5.5 (legacy failure semantics — see the Global Constraints note on the deliberate break from it); `docs/adr/0001-go-rewrite-on-forge.md`; `docs/adr/0002-launcher-abstraction.md` (O1 launcher precedence, O2 `hermes_*`/`jb_*` env-var split); `docs/adr/0004-alfred-script-filter-contract-frozen-at-2.6.2.md`.

## Global Constraints

- TDD: write the failing test before the implementation, every task (`.claude/rules/testing.md`).
- Every `if err != nil` wraps with `fmt.Errorf("doing X: %w", err)` (`.claude/rules/coding.md`).
- `internal/jetbrains` and `internal/launcher` never call `os.*`/`Platform.environment` directly — only `env.Env`/`iofs.FS`. `cmd/hermes` is the composition root and is where the real `env.New()`/`iofs.New()` get instantiated (already true of the existing `root.go`).
- No comments unless the *why* is non-obvious. No panics below `main`; every error returns up to `main.go`'s existing `exitcode.Resolve(err)` call.
- Conventional commit per task; scope `cmd`, `jetbrains`, or `launcher/alfred` to match the changed package.
- **Deliberate break from `docs/specs/original-spec.md` §5.5 (user-approved during planning):** the legacy Dart CLI caught a `NotFoundException` in `search`/`open` and rendered it as a normal Alfred result item (error icon + troubleshoot text), so a failure still produced valid Script Filter JSON. hermes 3.0 instead returns a Go `error` that `exitcode.Wrap`s to a structured exit code (`exitNotFound = 10`); nothing is written to stdout on failure. This trades the old "always show something in Alfred" UX for the roadmap's exit-code fix ([risk] no structured exit codes) and keeps the `Launcher` interface untouched (no error-rendering method). Document this break in a new ADR as part of Task 14's roadmap closeout — it changes user-visible Alfred behavior on a not-found product.
- **`--config <path>` is a new, CLI-only config source** (not in the legacy CLI or any ADR): it reads a JSON file shaped exactly like the existing `jb_custom_config` env var's value and merges it onto `jetbrains.Defaults()` via the already-frozen `jetbrains.Merge`. When given, it takes precedence over `jb_custom_config` (flag beats env, mirroring `--launcher`'s precedence). This is a reasonable filling-in of an underspecified roadmap bullet, not a contract change — flag it during review if a different shape was intended.
- `docs/specs/original-spec.md` §5.3's discovery rules and the `jb_*` env vars are already fully implemented in `internal/jetbrains` (M1) — this plan only *calls* that layer, it does not re-implement discovery.

---

### Task 1: `jetbrains.Service.Open`

**Files:**
- Modify: `internal/jetbrains/service.go`
- Modify: `internal/jetbrains/service_test.go`

**Interfaces:**
- Consumes: `Locator.LocateApplication/LocateBin` (`internal/jetbrains/locator.go`), `ProjectName.Resolve` (`internal/jetbrains/name.go`), `NotFoundError` (`internal/jetbrains/errors.go`), `domain.Item`/`domain.Product` (`pkg/domain`).
- Produces: `Service.Open(product domain.Product, path string) (domain.Item, error)` — used by Task 12 (`open` command). Mirrors the legacy `open` command's semantics exactly: it does **not** check `path` against the product's recent-projects list (the 2.x `open_project.dart` never did either — it just builds an item from whatever path it's given).

- [ ] **Step 1: Write the failing tests**

```go
// internal/jetbrains/service_test.go — add alongside the existing Search tests

func TestService_Open_buildsItem(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(phpStormFixture()), e, Defaults())

	item, err := svc.Open(domain.PhpStorm, "/home/x/projects/aurora")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}

	want := domain.Item{
		Name:           "AuroraProject",
		Path:           "/home/x/projects/aurora",
		IconPath:       "/Applications/PhpStorm.app",
		BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
		IsModernBinary: true,
		Match:          "AuroraProject aurora",
	}
	if item != want {
		t.Errorf("Open() = %+v, want %+v", item, want)
	}
}

func TestService_Open_pathNotAmongRecentProjectsStillResolves(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(phpStormFixture()), e, Defaults())

	item, err := svc.Open(domain.PhpStorm, "/home/x/projects/never-seen")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	if item.Name != "never-seen" {
		t.Errorf("Open() on a path with no .idea dir: Name = %q, want basename fallback %q", item.Name, "never-seen")
	}
}

func TestService_Open_appNotFound(t *testing.T) {
	files := phpStormFixture()
	delete(files, "/Applications/PhpStorm.app/Contents/MacOS/phpstorm")
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(files), e, Defaults())

	_, err := svc.Open(domain.PhpStorm, "/home/x/projects/aurora")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Open() error = %v, want *NotFoundError", err)
	}
}

func TestService_Open_unknownProduct(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(phpStormFixture()), e, Defaults())

	_, err := svc.Open(domain.Product("notAProduct"), "/home/x/projects/aurora")
	if err == nil {
		t.Fatal("Open(unknown product): want error, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/jetbrains/... -run TestService_Open -v`
Expected: FAIL — `svc.Open undefined (type Service has no field or method Open)`

- [ ] **Step 3: Implement `Service.Open`**

```go
// internal/jetbrains/service.go — add after Search, before SearchAll

// Open resolves path into a single launcher-neutral domain.Item for product, without
// listing (or checking against) the product's recent projects - the legacy open command
// never validated the given path either, it just built an item from it directly.
func (s Service) Open(product domain.Product, path string) (domain.Item, error) {
	details, ok := s.Config[product]
	if !ok {
		return domain.Item{}, fmt.Errorf("jetbrains: unknown product %q", product)
	}

	locator := NewLocator(s.FS, s.Env, product, details)
	appPath, err := locator.LocateApplication()
	if err != nil {
		return domain.Item{}, err
	}
	binPath, err := locator.LocateBin()
	if err != nil {
		return domain.Item{}, err
	}

	name, err := NewProjectName(s.FS).Resolve(path)
	if err != nil {
		return domain.Item{}, err
	}

	return domain.Item{
		Name:           name,
		Path:           path,
		IconPath:       appPath,
		BinaryPath:     binPath,
		IsModernBinary: strings.Contains(binPath, "MacOS"),
		Match:          name + " " + filepath.Base(path),
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/jetbrains/... -run TestService_Open -v`
Expected: PASS (all four)

- [ ] **Step 5: Commit**

```bash
git add internal/jetbrains/service.go internal/jetbrains/service_test.go
git commit -m "feat(jetbrains): add Service.Open for the open command"
```

---

### Task 2: `ProductDetails` JSON tags for `configuration`

**Files:**
- Modify: `internal/jetbrains/config.go`
- Modify: `internal/jetbrains/config_test.go`

**Interfaces:**
- Produces: `ProductDetails` gains `json` struct tags (`applicationNames`, `preferencePrefix`, `binaries`) matching the keys `Merge`'s `overrideStrings`/`overrideString` already read — so `jetbrains.Config` round-trips through `encoding/json`. Used by Task 13 (`configuration` command).

- [ ] **Step 1: Write the failing tests**

```go
// internal/jetbrains/config_test.go — add; needs "encoding/json" and "reflect" imports

func TestProductDetails_marshalKeys(t *testing.T) {
	raw, err := json.Marshal(Defaults()[domain.PhpStorm])
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	for _, key := range []string{"applicationNames", "preferencePrefix", "binaries"} {
		if _, ok := got[key]; !ok {
			t.Errorf("marshaled ProductDetails missing key %q, got %v", key, got)
		}
	}
}

func TestConfig_marshalRoundTrips(t *testing.T) {
	raw, err := json.Marshal(Defaults())
	if err != nil {
		t.Fatalf("Marshal(Defaults()): %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}

	merged, err := Merge(Defaults(), decoded)
	if err != nil {
		t.Fatalf("Merge(Defaults(), decoded): %v", err)
	}
	if !reflect.DeepEqual(merged, Defaults()) {
		t.Errorf("round-tripped config != Defaults()")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/jetbrains/... -run 'TestProductDetails_marshalKeys|TestConfig_marshalRoundTrips' -v`
Expected: FAIL — the unmarshaled map has empty-string keys (`ApplicationNames`, `PreferencePrefix`, `Binaries`, Go's default exported-field-name marshaling) instead of `applicationNames`/`preferencePrefix`/`binaries`.

- [ ] **Step 3: Add JSON tags**

```go
// internal/jetbrains/config.go

type ProductDetails struct {
	ApplicationNames []string `json:"applicationNames"`
	PreferencePrefix string   `json:"preferencePrefix"`
	Binaries         []string `json:"binaries"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/jetbrains/... -v`
Expected: PASS — every existing test in the package still passes (the tags don't change Go-side field access, only JSON keys) plus the two new ones.

- [ ] **Step 5: Commit**

```bash
git add internal/jetbrains/config.go internal/jetbrains/config_test.go
git commit -m "feat(jetbrains): tag ProductDetails for JSON round-tripping"
```

---

### Task 3: Alfred adapter — `WithLogFile` + third debug item

**Files:**
- Modify: `internal/launcher/alfred/adapter.go`
- Modify: `internal/launcher/alfred/adapter_test.go`

**Interfaces:**
- Produces: `alfred.WithLogFile(path string) Option` — used by Task 4 (`newLauncherRegistry`). Empty path (the zero value) omits the item, so every existing M2 test (none of which call `WithLogFile`) keeps passing unchanged.

- [ ] **Step 1: Write the failing test**

```go
// internal/launcher/alfred/adapter_test.go — add

func TestAdapter_Render_debugItemsIncludeLogFileWhenSet(t *testing.T) {
	fixed := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	a := NewAdapter(
		envtest.New(map[string]string{"alfred_debug": "1"}),
		WithClock(func() time.Time { return fixed }),
		WithLogFile("/tmp/hermes-debug-123.log"),
	)

	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 4 {
		t.Fatalf("Items = %d, want 4 (1 project + 3 debug)", len(got.Items))
	}
	if got.Items[3].Title != "Debug: Log /tmp/hermes-debug-123.log" {
		t.Errorf("Items[3].Title = %q, want %q", got.Items[3].Title, "Debug: Log /tmp/hermes-debug-123.log")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/launcher/alfred/... -run TestAdapter_Render_debugItemsIncludeLogFileWhenSet -v`
Expected: FAIL — `WithLogFile undefined`

- [ ] **Step 3: Implement**

```go
// internal/launcher/alfred/adapter.go

type Adapter struct {
	env        env.Env
	ttlSeconds int
	version    string
	logFile    string
	now        func() time.Time
}

// WithLogFile sets the path Render's third debug item ("Debug: Log <file>") points
// readers at when alfred_debug is set. Empty (the default) omits that item - there's
// nothing to point at until a caller (cmd/hermes's setupLogging, roadmap M3) has actually
// opened a debug log file.
func WithLogFile(path string) Option {
	return func(a *Adapter) { a.logFile = path }
}
```

```go
// internal/launcher/alfred/adapter.go — replace debugItems

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

	items := []resultItem{version, timer}
	if a.logFile != "" {
		items = append(items, buildResultItem(debugItem(fmt.Sprintf("Debug: Log %s", a.logFile), a.logFile, iconNote)))
	}
	return items
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/launcher/alfred/... -v`
Expected: PASS — the new test plus every existing M2 test (unaffected, since `logFile` defaults to `""`).

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/alfred/adapter.go internal/launcher/alfred/adapter_test.go
git commit -m "feat(launcher/alfred): add WithLogFile for the deferred third debug item"
```

---

### Task 4: `newLauncherRegistry` threads the debug log file

**Files:**
- Modify: `cmd/hermes/launchers.go`
- Modify: `cmd/hermes/launchers_test.go`

**Interfaces:**
- Consumes: `alfred.WithLogFile` (Task 3).
- Produces: `newLauncherRegistry(version string, e env.Env, logFile string) *launcher.Registry` (signature change — was `(version string, e env.Env)`). Used by Task 6 (`resolveLauncher` tests) and Task 9 (`runtime.init`).

- [ ] **Step 1: Update the failing tests first (signature change)**

```go
// cmd/hermes/launchers_test.go — change both call sites

func TestNewLauncherRegistry_detectsAlfred(t *testing.T) {
	r := newLauncherRegistry("1.2.3", envtest.New(nil), "")

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
	r := newLauncherRegistry("1.2.3", envtest.New(nil), "")
	if got := r.Default().Name(); got != "generic" {
		t.Errorf("Default() = %q, want %q", got, "generic")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestNewLauncherRegistry -v`
Expected: FAIL — too many arguments in call to `newLauncherRegistry`

- [ ] **Step 3: Update the implementation**

```go
// cmd/hermes/launchers.go

package main

import (
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/internal/launcher/alfred"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

// newLauncherRegistry builds the Registry every launcher-aware command reads from:
// Get/Detect/Default per ADR-0002 O1's precedence chain. logFile, when non-empty, threads
// the active debug log file into the Alfred adapter's third debug item.
func newLauncherRegistry(version string, e env.Env, logFile string) *launcher.Registry {
	return launcher.NewRegistry(
		generic.NewAdapter(),
		alfred.NewAdapter(e, alfred.WithVersion(version), alfred.WithLogFile(logFile)),
	)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/launchers.go cmd/hermes/launchers_test.go
git commit -m "feat(cmd): thread the debug log file into the launcher registry"
```

---

### Task 5: `parseProduct`

**Files:**
- Create: `cmd/hermes/product.go`
- Create: `cmd/hermes/product_test.go`

**Interfaces:**
- Consumes: `domain.Products()`, `domain.Product` (`pkg/domain`).
- Produces: `parseProduct(raw string) (domain.Product, error)` — used by Task 10 (`search`) and Task 12 (`open`).

- [ ] **Step 1: Write the failing test**

```go
// cmd/hermes/product_test.go
package main

import "testing"

func TestParseProduct(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "known product", raw: "phpStorm", wantErr: false},
		{name: "unknown product", raw: "notAProduct", wantErr: true},
		{name: "empty string", raw: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseProduct(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseProduct(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if err == nil && string(got) != tc.raw {
				t.Errorf("parseProduct(%q) = %q, want %q", tc.raw, got, tc.raw)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/hermes/... -run TestParseProduct -v`
Expected: FAIL — `parseProduct undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/product.go
package main

import (
	"fmt"
	"slices"

	"github.com/adaouat/hermes/pkg/domain"
)

// parseProduct validates raw against every domain.Product hermes supports. An unclassified
// error like this one defaults to exitcode.Usage via forge's exitcode.Resolve.
func parseProduct(raw string) (domain.Product, error) {
	p := domain.Product(raw)
	if !slices.Contains(domain.Products(), p) {
		return "", fmt.Errorf(`unknown product %q (run "hermes configuration" for the full list)`, raw)
	}
	return p, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/hermes/... -run TestParseProduct -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/product.go cmd/hermes/product_test.go
git commit -m "feat(cmd): add parseProduct flag validation"
```

---

### Task 6: `loadConfig`

**Files:**
- Create: `cmd/hermes/config.go`
- Create: `cmd/hermes/config_test.go`

**Interfaces:**
- Consumes: `jetbrains.Load`, `jetbrains.Defaults`, `jetbrains.Merge` (`internal/jetbrains`).
- Produces: `loadConfig(e env.Env, path string) (jetbrains.Config, error)` — used by Task 9 (`runtime.init`).

- [ ] **Step 1: Write the failing tests**

```go
// cmd/hermes/config_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestLoadConfig_emptyPathFallsBackToEnv(t *testing.T) {
	e := envtest.New(map[string]string{"jb_custom_config": `{"phpStorm":{"preferencePrefix":"Custom"}}`})

	cfg, err := loadConfig(e, "")
	if err != nil {
		t.Fatalf("loadConfig(): %v", err)
	}
	if cfg[domain.PhpStorm].PreferencePrefix != "Custom" {
		t.Errorf("PreferencePrefix = %q, want %q", cfg[domain.PhpStorm].PreferencePrefix, "Custom")
	}
}

func TestLoadConfig_fileOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hermes-config.json")
	if err := os.WriteFile(path, []byte(`{"phpStorm":{"preferencePrefix":"FromFile"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	e := envtest.New(map[string]string{"jb_custom_config": `{"phpStorm":{"preferencePrefix":"FromEnv"}}`})

	cfg, err := loadConfig(e, path)
	if err != nil {
		t.Fatalf("loadConfig(): %v", err)
	}
	if cfg[domain.PhpStorm].PreferencePrefix != "FromFile" {
		t.Errorf("PreferencePrefix = %q, want %q", cfg[domain.PhpStorm].PreferencePrefix, "FromFile")
	}
}

func TestLoadConfig_missingFileReturnsError(t *testing.T) {
	if _, err := loadConfig(envtest.New(nil), "/nonexistent/hermes-config.json"); err == nil {
		t.Fatal("loadConfig(missing file): want error, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestLoadConfig -v`
Expected: FAIL — `loadConfig undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/config.go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/jetbrains"
)

// loadConfig returns the merged per-product configuration. An explicit --config path takes
// precedence over the jb_custom_config env var (jetbrains.Load); an empty path falls back
// to jetbrains.Load(e) so the frozen 2.x env-var contract keeps working untouched.
func loadConfig(e env.Env, path string) (jetbrains.Config, error) {
	if path == "" {
		return jetbrains.Load(e)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var custom map[string]any
	if err := json.Unmarshal(raw, &custom); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return jetbrains.Merge(jetbrains.Defaults(), custom)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -run TestLoadConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/config.go cmd/hermes/config_test.go
git commit -m "feat(cmd): add --config file loading via loadConfig"
```

---

### Task 7: `resolveLauncher`

**Files:**
- Create: `cmd/hermes/launcherselect.go`
- Create: `cmd/hermes/launcherselect_test.go`

**Interfaces:**
- Consumes: `newLauncherRegistry` (Task 4), `launcher.Registry.Get/Detect/Default` (`internal/launcher`).
- Produces: `resolveLauncher(flagValue string, e env.Env, reg *launcher.Registry) (launcher.Launcher, error)` — used by Task 9 (`runtime.init`).

- [ ] **Step 1: Write the failing test**

```go
// cmd/hermes/launcherselect_test.go
package main

import (
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
)

func TestResolveLauncher(t *testing.T) {
	reg := newLauncherRegistry("1.0.0", envtest.New(nil), "")

	tests := []struct {
		name      string
		flagValue string
		envVars   map[string]string
		wantName  string
		wantErr   bool
	}{
		{name: "flag wins", flagValue: "alfred", wantName: "alfred"},
		{name: "env var used when no flag", envVars: map[string]string{"HERMES_LAUNCHER": "alfred"}, wantName: "alfred"},
		{name: "auto-detect when no flag or env", envVars: map[string]string{"alfred_version": "5.5"}, wantName: "alfred"},
		{name: "falls back to default", wantName: "generic"},
		{name: "unknown flag value errors", flagValue: "bogus", wantErr: true},
		{name: "unknown env value errors", envVars: map[string]string{"HERMES_LAUNCHER": "bogus"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := envtest.New(tc.envVars)
			got, err := resolveLauncher(tc.flagValue, e, reg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveLauncher() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && got.Name() != tc.wantName {
				t.Errorf("resolveLauncher() = %q, want %q", got.Name(), tc.wantName)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/hermes/... -run TestResolveLauncher -v`
Expected: FAIL — `resolveLauncher undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/launcherselect.go
package main

import (
	"fmt"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/launcher"
)

// resolveLauncher implements ADR-0002 O1's precedence: an explicit --launcher flag wins,
// then the HERMES_LAUNCHER env var, then auto-detection, and finally reg.Default() (the
// generic adapter) when nothing else resolves - "no launcher resolved is not an error".
func resolveLauncher(flagValue string, e env.Env, reg *launcher.Registry) (launcher.Launcher, error) {
	if flagValue != "" {
		l, ok := reg.Get(flagValue)
		if !ok {
			return nil, fmt.Errorf("unknown launcher %q", flagValue)
		}
		return l, nil
	}

	if name, ok := e.Lookup("HERMES_LAUNCHER"); ok && name != "" {
		l, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown launcher %q (from HERMES_LAUNCHER)", name)
		}
		return l, nil
	}

	if l, ok := reg.Detect(e); ok {
		return l, nil
	}
	return reg.Default(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/hermes/... -run TestResolveLauncher -v`
Expected: PASS (all 6 subtests)

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/launcherselect.go cmd/hermes/launcherselect_test.go
git commit -m "feat(cmd): add resolveLauncher (ADR-0002 O1 precedence)"
```

---

### Task 8: `setupLogging`

**Files:**
- Create: `cmd/hermes/logging.go`
- Create: `cmd/hermes/logging_test.go`

**Interfaces:**
- Produces: `setupLogging(stderr io.Writer, debug bool, dir string) (logFile string, err error)` — used by Task 9 (`runtime.init`). `dir` is dependency-injected so tests stay inside `t.TempDir()`; production passes `""` (⇒ `os.CreateTemp`'s default OS temp dir).

- [ ] **Step 1: Write the failing tests**

```go
// cmd/hermes/logging_test.go
package main

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestSetupLogging_notDebug_stderrOnlyAtInfoLevel(t *testing.T) {
	var stderr bytes.Buffer

	logFile, err := setupLogging(&stderr, false, t.TempDir())
	if err != nil {
		t.Fatalf("setupLogging(): %v", err)
	}
	if logFile != "" {
		t.Errorf("logFile = %q, want empty outside debug mode", logFile)
	}

	slog.Debug("should not appear")
	slog.Info("should appear")

	if strings.Contains(stderr.String(), "should not appear") {
		t.Errorf("stderr contains a Debug record at Info level: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "should appear") {
		t.Errorf("stderr missing Info record: %s", stderr.String())
	}
}

func TestSetupLogging_debug_mirrorsToFileAtDebugLevel(t *testing.T) {
	var stderr bytes.Buffer

	logFile, err := setupLogging(&stderr, true, t.TempDir())
	if err != nil {
		t.Fatalf("setupLogging(): %v", err)
	}
	if logFile == "" {
		t.Fatal(`logFile = "", want a path in debug mode`)
	}

	slog.Debug("debug record")

	if !strings.Contains(stderr.String(), "debug record") {
		t.Errorf("stderr missing Debug record: %s", stderr.String())
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", logFile, err)
	}
	if !strings.Contains(string(content), "debug record") {
		t.Errorf("log file missing Debug record: %s", content)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestSetupLogging -v`
Expected: FAIL — `setupLogging undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/logging.go
package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// setupLogging installs the process-wide slog default: a stderr-only Info handler
// normally, or - when debug is true - a Debug-level handler mirroring every record to
// both stderr and a temp log file (dir; empty means the OS default temp directory),
// returning that file's path so callers (the Alfred adapter's third debug item, via
// alfred.WithLogFile) can point users at it. hermes is a one-shot CLI, not a daemon, so
// the file is left open for the process lifetime and reclaimed on exit - no explicit
// Close.
func setupLogging(stderr io.Writer, debug bool, dir string) (string, error) {
	if !debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
		return "", nil
	}

	f, err := os.CreateTemp(dir, "hermes-debug-*.log")
	if err != nil {
		return "", fmt.Errorf("creating debug log file: %w", err)
	}

	handler := slog.NewTextHandler(io.MultiWriter(stderr, f), &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	return f.Name(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -run TestSetupLogging -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/logging.go cmd/hermes/logging_test.go
git commit -m "feat(cmd): add setupLogging (stderr + debug temp file)"
```

---

### Task 9: `runtime` + `root.go` wiring

**Files:**
- Create: `cmd/hermes/runtime.go`
- Create: `cmd/hermes/runtime_test.go`
- Create: `cmd/hermes/exitcode.go`
- Modify: `cmd/hermes/root.go`

**Interfaces:**
- Consumes: `setupLogging` (Task 8), `loadConfig` (Task 6), `newLauncherRegistry` (Task 4), `resolveLauncher` (Task 7).
- Produces: `type runtime struct { fs iofs.FS; env env.Env; config jetbrains.Config; launcher launcher.Launcher }` with `(rt *runtime) init(fs iofs.FS, e env.Env, version string, debug bool, launcherFlag, configFlag string, stderr io.Writer) error`, and the constant `exitNotFound = 10`. Used by Tasks 10–13 (all four commands read `rt.fs`/`rt.env`/`rt.config`/`rt.launcher`; `search`/`open` use `exitNotFound`).
- `root.go`'s `PersistentPreRunE` is the only production caller of `rt.init`; no dedicated test for `rootCmd`/`PersistentPreRunE` itself (matches this file's existing untested `updateHint`/`rootCmd` — a thin wiring function calling only already-tested pieces).

- [ ] **Step 1: Write the failing tests**

```go
// cmd/hermes/runtime_test.go
package main

import (
	"bytes"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
)

func TestRuntimeInit_wiresConfigAndLauncher(t *testing.T) {
	rt := &runtime{}
	e := envtest.New(map[string]string{"alfred_version": "5.5"})
	var stderr bytes.Buffer

	if err := rt.init(iofstest.New(nil), e, "1.0.0", false, "", "", &stderr); err != nil {
		t.Fatalf("init(): %v", err)
	}
	if rt.launcher.Name() != "alfred" {
		t.Errorf("launcher = %q, want %q (auto-detected via alfred_version)", rt.launcher.Name(), "alfred")
	}
	if len(rt.config) != 19 {
		t.Errorf("config has %d products, want 19 (defaults, no overrides)", len(rt.config))
	}
}

func TestRuntimeInit_unknownLauncherFlagFails(t *testing.T) {
	rt := &runtime{}
	var stderr bytes.Buffer

	if err := rt.init(iofstest.New(nil), envtest.New(nil), "1.0.0", false, "bogus", "", &stderr); err == nil {
		t.Fatal("init() with unknown --launcher: want error, got nil")
	}
}

func TestRuntimeInit_badConfigPathFails(t *testing.T) {
	rt := &runtime{}
	var stderr bytes.Buffer

	if err := rt.init(iofstest.New(nil), envtest.New(nil), "1.0.0", false, "", "/nonexistent/hermes-config.json", &stderr); err == nil {
		t.Fatal("init() with unreadable --config path: want error, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestRuntimeInit -v`
Expected: FAIL — `runtime` undefined

- [ ] **Step 3: Implement `runtime` and `exitcode.go`**

```go
// cmd/hermes/runtime.go
package main

import (
	"fmt"
	"io"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/internal/launcher"
)

// runtime is the composition root's output (workflow.md's "composition root pattern"):
// the real dependencies every command's RunE needs, built once by root.go's
// PersistentPreRunE and read by every subcommand.
type runtime struct {
	fs       iofs.FS
	env      env.Env
	config   jetbrains.Config
	launcher launcher.Launcher
}

// init wires fs/e (injected so tests use fakes; production passes iofs.New()/env.New()),
// the merged product configuration (loadConfig), logging (setupLogging), and the launcher
// resolved per ADR-0002 O1 (resolveLauncher).
func (rt *runtime) init(fs iofs.FS, e env.Env, version string, debug bool, launcherFlag, configFlag string, stderr io.Writer) error {
	rt.fs = fs
	rt.env = e

	logFile, err := setupLogging(stderr, debug, "")
	if err != nil {
		return fmt.Errorf("setting up logging: %w", err)
	}

	cfg, err := loadConfig(e, configFlag)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}
	rt.config = cfg

	reg := newLauncherRegistry(version, e, logFile)
	l, err := resolveLauncher(launcherFlag, e, reg)
	if err != nil {
		return err
	}
	rt.launcher = l

	return nil
}
```

```go
// cmd/hermes/exitcode.go
package main

// exitNotFound is hermes's domain-specific exit code for jetbrains.NotFoundError, in
// forge exitcode's reserved app range (4-69).
const exitNotFound = 10
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -run TestRuntimeInit -v`
Expected: PASS

- [ ] **Step 5: Wire `root.go`**

```go
// cmd/hermes/root.go
package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/adaouat/forge/updatecheck"
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
)

// jsonOutputCommands never get the update-check banner appended by PersistentPostRunE -
// it would corrupt their stdout, which Alfred/Raycast parse as JSON. doctor/install (M4)
// don't emit JSON and are deliberately left off this set so they keep the hint once they
// land (full finalization is roadmap M6's job).
var jsonOutputCommands = map[string]bool{
	"search": true,
	"all":    true,
	"open":   true,
}

func rootCmd(version string) *cobra.Command {
	rt := &runtime{}

	var launcherFlag, configFlag string
	var debug bool

	cmd := &cobra.Command{
		Use:           "hermes",
		Short:         "Locate JetBrains IDEs and recent projects for Alfred, Raycast, and other launchers",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			return rt.init(iofs.New(), env.New(), version, debug, launcherFlag, configFlag, c.ErrOrStderr())
		},
	}
	cmd.PersistentFlags().StringVar(&launcherFlag, "launcher", "", "Launcher output format: alfred, generic (default: auto-detect)")
	cmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to a jb_custom_config-shaped JSON file (overrides jb_custom_config)")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Verbose logging, mirrored to a temp file")

	cmd.PersistentPostRunE = func(c *cobra.Command, _ []string) error {
		if jsonOutputCommands[c.Name()] {
			return nil
		}
		updateHint(c, version)
		return nil
	}

	cmd.AddCommand(newSearchCmd(rt), newAllCmd(rt), newOpenCmd(rt), newConfigurationCmd(rt))
	return cmd
}

func updateHint(c *cobra.Command, version string) {
	if version == "dev" || os.Getenv("HERMES_CHECK_UPDATE") == "false" {
		return
	}
	cache, _ := os.UserCacheDir()
	updatecheck.Hinter{
		Repo:      "adaouat/hermes",
		Bin:       "hermes",
		Module:    "github.com/adaouat/hermes/cmd/hermes",
		Current:   version,
		CacheFile: filepath.Join(cache, "hermes", "update-check.json"),
	}.Print(c.Context(), c.ErrOrStderr())
}
```

Note: this won't compile until Tasks 10–13 add `newSearchCmd`/`newAllCmd`/`newOpenCmd`/`newConfigurationCmd` — that's expected; `go build ./...` succeeds again at the end of Task 13.

- [ ] **Step 6: Commit**

```bash
git add cmd/hermes/runtime.go cmd/hermes/runtime_test.go cmd/hermes/exitcode.go cmd/hermes/root.go
git commit -m "feat(cmd): add the runtime composition root and wire root.go's flags"
```

---

### Task 10: `search` command

**Files:**
- Create: `cmd/hermes/search.go`
- Create: `cmd/hermes/search_test.go`

**Interfaces:**
- Consumes: `parseProduct` (Task 5), `runtime` (Task 9), `exitNotFound` (Task 9), `jetbrains.NewService`/`Service.Search`/`NotFoundError` (`internal/jetbrains`).
- Produces: `newSearchCmd(rt *runtime) *cobra.Command`, `runSearch(w io.Writer, rt *runtime, productFlag, filter string) error`. Also introduces `phpStormFixture()` and `newTestRuntime(files map[string]string) *runtime` in `search_test.go`, reused (same package, no import needed) by Tasks 11–12's test files.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/hermes/search_test.go
package main

import (
	"bytes"
	"errors"
	"testing"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/internal/launcher/generic"
)

func phpStormFixture() map[string]string {
	return map[string]string{
		"/Applications/PhpStorm.app/Contents/MacOS/phpstorm": "",
		"/home/x/Library/Preferences/PhpStorm2024.1/options/recentProjectDirectories.xml": `<application>
  <component name="RecentDirectoryProjectsManager">
    <option name="recentPaths">
      <list>
        <option value="$USER_HOME$/projects/aurora" />
      </list>
    </option>
  </component>
</application>`,
		"/home/x/projects/aurora/.idea/name": "AuroraProject",
	}
}

func newTestRuntime(files map[string]string) *runtime {
	return &runtime{
		fs:       iofstest.New(files),
		env:      envtest.New(map[string]string{"HOME": "/home/x"}),
		config:   jetbrains.Defaults(),
		launcher: generic.NewAdapter(),
	}
}

func TestRunSearch_rendersItems(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runSearch(&buf, rt, "phpStorm", ""); err != nil {
		t.Fatalf("runSearch(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestRunSearch_unknownProductReturnsError(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runSearch(&buf, rt, "notAProduct", ""); err == nil {
		t.Fatal("runSearch(unknown product): want error, got nil")
	}
}

func TestRunSearch_notFoundWrapsExitCode(t *testing.T) {
	rt := newTestRuntime(nil) // no PhpStorm.app fixture -> NotFoundError
	var buf bytes.Buffer

	err := runSearch(&buf, rt, "phpStorm", "")
	if err == nil {
		t.Fatal("runSearch(): want error, got nil")
	}
	if forgeexit.Resolve(err) != exitNotFound {
		t.Errorf("exitcode.Resolve(err) = %d, want %d", forgeexit.Resolve(err), exitNotFound)
	}
	var notFound *jetbrains.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error chain does not contain *jetbrains.NotFoundError")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestRunSearch -v`
Expected: FAIL — `runSearch undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/search.go
package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/jetbrains"
)

func newSearchCmd(rt *runtime) *cobra.Command {
	var product, filter string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search recent projects for one JetBrains product",
		RunE: func(c *cobra.Command, _ []string) error {
			return runSearch(c.OutOrStdout(), rt, product, filter)
		},
	}
	cmd.Flags().StringVar(&product, "product", "", `JetBrains product (see "hermes configuration" for the full list)`)
	cmd.Flags().StringVar(&filter, "filter", "", "Filter projects by name or path")
	_ = cmd.MarkFlagRequired("product")
	return cmd
}

func runSearch(w io.Writer, rt *runtime, productFlag, filter string) error {
	product, err := parseProduct(productFlag)
	if err != nil {
		return err
	}

	svc := jetbrains.NewService(rt.fs, rt.env, rt.config)
	items, err := svc.Search(product, filter)
	if err != nil {
		var notFound *jetbrains.NotFoundError
		if errors.As(err, &notFound) {
			return forgeexit.Wrap(exitNotFound, err)
		}
		return err
	}

	if err := rt.launcher.Render(items, w); err != nil {
		return fmt.Errorf("rendering search results: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -run TestRunSearch -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/search.go cmd/hermes/search_test.go
git commit -m "feat(cmd): add the search command"
```

---

### Task 11: `all` command

**Files:**
- Create: `cmd/hermes/searchall.go`
- Create: `cmd/hermes/searchall_test.go`

**Interfaces:**
- Consumes: `runtime` (Task 9), `phpStormFixture`/`newTestRuntime` (Task 10, same package), `jetbrains.NewService`/`Service.SearchAll`.
- Produces: `newAllCmd(rt *runtime) *cobra.Command`, `runAll(w io.Writer, rt *runtime, filter string) error`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/hermes/searchall_test.go
package main

import (
	"bytes"
	"testing"
)

func TestRunAll_rendersAcrossProducts(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runAll(&buf, rt, ""); err != nil {
		t.Fatalf("runAll(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestRunAll_missingProductsAreSkipped(t *testing.T) {
	rt := newTestRuntime(nil)
	var buf bytes.Buffer

	if err := runAll(&buf, rt, ""); err != nil {
		t.Fatalf("runAll() with no products installed: %v", err)
	}
	if buf.String() != "[]\n" {
		t.Errorf("output = %q, want %q (empty generic array)", buf.String(), "[]\n")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestRunAll -v`
Expected: FAIL — `runAll undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/searchall.go
package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/adaouat/hermes/internal/jetbrains"
)

func newAllCmd(rt *runtime) *cobra.Command {
	var filter string

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Search recent projects across every installed JetBrains product",
		RunE: func(c *cobra.Command, _ []string) error {
			return runAll(c.OutOrStdout(), rt, filter)
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "", "Filter projects by name or path")
	return cmd
}

func runAll(w io.Writer, rt *runtime, filter string) error {
	svc := jetbrains.NewService(rt.fs, rt.env, rt.config)
	items := svc.SearchAll(filter)

	if err := rt.launcher.Render(items, w); err != nil {
		return fmt.Errorf("rendering search results: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -run TestRunAll -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/searchall.go cmd/hermes/searchall_test.go
git commit -m "feat(cmd): add the all command"
```

---

### Task 12: `open` command

**Files:**
- Create: `cmd/hermes/open.go`
- Create: `cmd/hermes/open_test.go`

**Interfaces:**
- Consumes: `parseProduct` (Task 5), `runtime`/`exitNotFound` (Task 9), `newTestRuntime`/`phpStormFixture` (Task 10), `jetbrains.NewService`/`Service.Open` (Task 1), `domain.Item` (`pkg/domain`).
- Produces: `newOpenCmd(rt *runtime) *cobra.Command`, `runOpen(w io.Writer, rt *runtime, productFlag, path string) error`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/hermes/open_test.go
package main

import (
	"bytes"
	"errors"
	"testing"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/jetbrains"
)

func TestRunOpen_rendersOneItem(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runOpen(&buf, rt, "phpStorm", "/home/x/projects/aurora"); err != nil {
		t.Fatalf("runOpen(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("AuroraProject")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "AuroraProject")
	}
}

func TestRunOpen_unknownProductReturnsError(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runOpen(&buf, rt, "notAProduct", "/home/x/projects/aurora"); err == nil {
		t.Fatal("runOpen(unknown product): want error, got nil")
	}
}

func TestRunOpen_notFoundWrapsExitCode(t *testing.T) {
	rt := newTestRuntime(nil) // no PhpStorm.app fixture -> NotFoundError
	var buf bytes.Buffer

	err := runOpen(&buf, rt, "phpStorm", "/home/x/projects/aurora")
	if err == nil {
		t.Fatal("runOpen(): want error, got nil")
	}
	if forgeexit.Resolve(err) != exitNotFound {
		t.Errorf("exitcode.Resolve(err) = %d, want %d", forgeexit.Resolve(err), exitNotFound)
	}
	var notFound *jetbrains.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error chain does not contain *jetbrains.NotFoundError")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/hermes/... -run TestRunOpen -v`
Expected: FAIL — `runOpen undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/open.go
package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/pkg/domain"
)

func newOpenCmd(rt *runtime) *cobra.Command {
	var product, path string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Resolve one project path into a single launcher item",
		RunE: func(c *cobra.Command, _ []string) error {
			return runOpen(c.OutOrStdout(), rt, product, path)
		},
	}
	cmd.Flags().StringVar(&product, "product", "", `JetBrains product (see "hermes configuration" for the full list)`)
	cmd.Flags().StringVar(&path, "path", "", "Path to the project")
	_ = cmd.MarkFlagRequired("product")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

func runOpen(w io.Writer, rt *runtime, productFlag, path string) error {
	product, err := parseProduct(productFlag)
	if err != nil {
		return err
	}

	svc := jetbrains.NewService(rt.fs, rt.env, rt.config)
	item, err := svc.Open(product, path)
	if err != nil {
		var notFound *jetbrains.NotFoundError
		if errors.As(err, &notFound) {
			return forgeexit.Wrap(exitNotFound, err)
		}
		return err
	}

	if err := rt.launcher.Render([]domain.Item{item}, w); err != nil {
		return fmt.Errorf("rendering opened project: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/hermes/... -run TestRunOpen -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/open.go cmd/hermes/open_test.go
git commit -m "feat(cmd): add the open command"
```

---

### Task 13: `configuration` command

**Files:**
- Create: `cmd/hermes/configuration.go`
- Create: `cmd/hermes/configuration_test.go`

**Interfaces:**
- Consumes: `runtime.config` (Task 9), `jetbrains.Defaults`/`jetbrains.ProductDetails` (Task 2).
- Produces: `newConfigurationCmd(rt *runtime) *cobra.Command`, `runConfiguration(w io.Writer, rt *runtime) error`.

- [ ] **Step 1: Write the failing test**

```go
// cmd/hermes/configuration_test.go
package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/adaouat/hermes/internal/jetbrains"
)

func TestRunConfiguration_printsMergedConfig(t *testing.T) {
	rt := &runtime{config: jetbrains.Defaults()}
	var buf bytes.Buffer

	if err := runConfiguration(&buf, rt); err != nil {
		t.Fatalf("runConfiguration(): %v", err)
	}

	var got map[string]jetbrains.ProductDetails
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got) != 19 {
		t.Errorf("configuration has %d products, want 19", len(got))
	}
	if got["phpStorm"].PreferencePrefix != "PhpStorm" {
		t.Errorf(`configuration["phpStorm"].PreferencePrefix = %q, want %q`, got["phpStorm"].PreferencePrefix, "PhpStorm")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/hermes/... -run TestRunConfiguration -v`
Expected: FAIL — `runConfiguration undefined`

- [ ] **Step 3: Implement**

```go
// cmd/hermes/configuration.go
package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newConfigurationCmd(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "configuration",
		Short: "Print the merged per-product configuration as JSON",
		RunE: func(c *cobra.Command, _ []string) error {
			return runConfiguration(c.OutOrStdout(), rt)
		},
	}
}

func runConfiguration(w io.Writer, rt *runtime) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rt.config); err != nil {
		return fmt.Errorf("encoding configuration: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/hermes/... -v`
Expected: PASS — every `cmd/hermes` test, and `go build ./...` now succeeds (root.go's forward references to all four `new*Cmd` constructors resolve).

- [ ] **Step 5: Commit**

```bash
git add cmd/hermes/configuration.go cmd/hermes/configuration_test.go
git commit -m "feat(cmd): add the configuration command"
```

---

### Task 14: ADR for the failure-semantics break + roadmap closeout

**Files:**
- Create: `docs/adr/0006-command-error-handling-breaks-from-2x-error-items.md`
- Modify: `docs/tasks/roadmap.md` (M3 section)

- [ ] **Step 1: Write the ADR**

```markdown
# 0006 — Command error handling: structured exit codes, not rendered error items

## Status

Accepted

## Context

`docs/specs/original-spec.md` §5.5 documents the legacy 2.x Dart CLI's failure semantics:
`search` and `open` catch a `NotFoundException` and render it as a normal Alfred result item
(error icon, troubleshoot text in the subtitle), so a failure still produces valid Script
Filter JSON and the user sees *something* in Alfred rather than nothing. `search_all`
silently skips failing products.

`docs/tasks/roadmap.md`'s M3 checklist calls for every command to return a Go `error` that
maps to a structured process exit code (`NotFound` in forge exitcode's reserved 4-69 range),
fixing `[risk] no structured exit codes`. The `Launcher` interface
(`docs/adr/0002-launcher-abstraction.md`) has no channel for "render this error" - only
`Render([]domain.Item, io.Writer) error` for real items - so honoring both the legacy
render-an-item UX and the roadmap's exit-code fix would require extending the frozen
interface across all three adapters, a bigger change than M3's scope.

## Decision

`search` and `open` (`cmd/hermes/search.go`, `cmd/hermes/open.go`) return the wrapped Go
error on a `*jetbrains.NotFoundError` (`forgeexit.Wrap(exitNotFound, err)`, `exitNotFound =
10`) instead of rendering an error item. Nothing is written to stdout on failure; `main`'s
`exitcode.Resolve(err)` sets the process exit code, and fang's error panel (via `cli.Run`)
prints the message to stderr. `all` keeps the legacy CLI's per-product silent-skip
(`jetbrains.Service.SearchAll` already implements this, unchanged from M1).

The `Launcher` interface is not extended. Adding an error-rendering path is left as a future
option if Alfred UX parity is later required — it would need its own ADR and a design pass
across `alfred`/`generic`/the future `raycast` adapter.

## Consequences

- **Alfred UX regression from 2.x:** when a product can't be located, Alfred's Script Filter
  now shows nothing (or its own generic "no results" state) instead of a helpful row
  explaining why. Existing 2.x users upgrading to hermes 3.0 lose this feedback.
- Direct CLI/script usage (and the future Raycast extension) gets a real, scriptable exit
  code instead of having to parse a rendered item to detect failure — an improvement over
  2.x for those callers.
- If Alfred UX parity is prioritized later, revisit this ADR alongside a `Launcher`
  interface change (new ADR required per `docs/adr/0002`'s "never means changing... without
  an ADR" rule).

## References

- `docs/tasks/roadmap.md` M3
- `docs/specs/original-spec.md` §5.5
- [`0002-launcher-abstraction.md`](0002-launcher-abstraction.md)
- [`0004-alfred-script-filter-contract-frozen-at-2.6.2.md`](0004-alfred-script-filter-contract-frozen-at-2.6.2.md)
```

- [ ] **Step 2: Commit the ADR**

```bash
git add docs/adr/0006-command-error-handling-breaks-from-2x-error-items.md
git commit -m "docs(adr): record the command error-handling break from 2.x"
```

- [ ] **Step 3: Flip M3's checkboxes in `docs/tasks/roadmap.md`**

Mark every M3 bullet `[ ]` → `[x]` and add a completion note in the same style as M1/M2's:

```markdown
**Note (<today's date>):** Implemented bottom-up: `jetbrains.Service.Open` and
`ProductDetails` JSON tags (M1 extensions the `open`/`configuration` commands needed) and
the Alfred adapter's deferred third debug item (`alfred.WithLogFile`, M2's ADR-0004 note)
landed first, then `cmd/hermes`'s composition-root pieces (`parseProduct`, `loadConfig`,
`resolveLauncher`, `setupLogging`, the `runtime` struct wiring them together and `root.go`'s
persistent flags), then the four commands themselves. Cobra command files live directly in
`cmd/hermes` (package `main`), not a separate `internal/cmd` - continuing the precedent M0/M2
already set (see M0's completion note); `coding.md`'s layer table still names `internal/cmd`
and should be corrected in a follow-up doc pass. `--config <path>` is a new, roadmap-implied
but previously unspecified flag: it reads a `jb_custom_config`-shaped JSON file and takes
precedence over the env var. The legacy CLI's §5.5 failure semantics (render an error item
in Alfred on a not-found product) were deliberately **not** ported - `search`/`open` now
return a structured exit code (`exitNotFound = 10`) instead, a user-approved break recorded
in ADR-0006. `root.go`'s pre-existing (M0) unconditional `updateHint` call would have
corrupted `search`/`all`/`open`'s JSON stdout the moment they existed, so it's now gated by a
`jsonOutputCommands` set - a necessary fix exposed by M3's own commands, not a pull-forward
of M6's full `updatecheck.Hinter` wiring (which still needs to add `doctor`/`install` to that
set once M4 lands). `rootCmd`/`PersistentPreRunE`/`updateHint` remain untested, consistent
with this file's existing precedent for thin wiring functions; every piece they call
(`setupLogging`, `loadConfig`, `resolveLauncher`, `newLauncherRegistry`, `runtime.init`) is
unit-tested with fakes.
```

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/roadmap.md
git commit -m "docs(roadmap): close out M3 commands + forge wiring"
```

---

## Verification (run after Task 14)

```bash
go test ./... -v
golangci-lint run
go build -o ./hermes ./cmd/hermes/
./hermes configuration | head -20
HERMES_CHECK_UPDATE=false ./hermes search --product phpStorm --launcher generic
HERMES_CHECK_UPDATE=false ./hermes all --launcher generic
./hermes search --product notAProduct; echo "exit: $?"   # want a non-zero, non-10 exit (Usage)
```

Expected: all tests pass, lint is silent, the build succeeds, `configuration` prints 19
products as JSON, `search`/`all` against a real machine with at least one installed
JetBrains product print generic JSON items, and an unknown `--product` exits non-zero via
forge's default `exitcode.Usage` classification (not `exitNotFound`, since that error never
reaches `jetbrains`).
