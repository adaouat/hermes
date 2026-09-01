# Roadmap: hermes 3.0 — Multi-Launcher Rewrite on Forge (Go)

> Companion to `docs/specs/original-spec.md` (which captures *what 2.6.2 does today* in Dart, Alfred-only).
> This roadmap describes the **Go rewrite on top of [`github.com/adaouat/forge`](https://github.com/adaouat/forge)**, with first-class support for **multiple launchers** (Alfred today, Raycast next, others later).
>
> **Foundation pin:** forge `v0.17.x` (latest published tag). See forge [ADR-0001](https://github.com/adaouat/forge/blob/main/docs/adr/0001-shared-core-module.md) (extraction bar) and [ADR-0010](https://github.com/adaouat/forge/blob/main/docs/adr/0010-cli-framework-foundation.md) (forge owns fang + theme).
>
> **Phase convention:** mirrors forge's `M0–MN` milestone style.

---

## 0. Guiding Principles

1. **One domain core, many launchers.** JetBrains discovery (locating apps, binaries, settings dirs, recent projects, project names) is *launcher-neutral* and lives in `internal/jetbrains/`. Launcher-specific code — install rituals, output formats — is isolated behind a `launcher.Launcher` interface.
2. **External contracts of every supported launcher are frozen.** Alfred's Script Filter JSON shape and the existing 2.x env-var inputs stay byte-compatible. A new launcher = a new adapter, not a change to existing adapters.
3. **Forge owns CLI framework + utilities.** This repo never re-implements what forge ships. It also never pushes JetBrains/launcher logic *into* forge — that's domain code (forge ADR-0001 bar: identical + stable + ≥2 consumers; we satisfy none).
4. **Distribution via package managers.** No binary self-replacement (forge [ADR-0005](https://github.com/adaouat/forge/blob/main/docs/adr/0005-updates-via-package-managers.md)). brew + mise + curl install + goreleaser.
5. **Composition root pattern.** All side effects (filesystem, environment, clock, stdout) flow through interfaces wired in `main`. No global state.
6. **Launcher install ≠ binary install.** Each launcher decides what "install" means (Alfred: write workflow bundle; Raycast: verify PATH + point at Raycast Store). The binary itself is always installed by the user's package manager.

---

## 1. Architectural Target

```
                          bin/hermes (entry, ~30 LoC)
                                       │
                                       ▼
                            cli.Run(ctx, root, version, accent)   ◀── forge/cli
                                       │
                                       ▼
                            internal/cmd/root.go  (cobra)
                                       │
            ┌──────────┬──────────┬────┴────┬──────────┬──────────┬──────────┐
            ▼          ▼          ▼         ▼          ▼          ▼          ▼
          search       all       open    install  configuration  doctor   version
            │          │          │         │
            │          │          │         └──▶ launcher.Registry
            │          │          │               .Get("alfred"|"raycast")
            │          │          │               .Install(ctx, opts)
            │          │          │
            └──────────┴──────────┴───────▶ jetbrains.Service (domain core)
                                              │
                                              ▼
                                      []domain.Item   (launcher-neutral)
                                              │
                                              ▼
                                   launcher.Registry
                                   .Get(--launcher).Render(items, w)
                                              │
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                       AlfredAdapter    RaycastAdapter   GenericJSON
                       (script filter)  (typed list)      (passthrough)
```

**Layer responsibilities:**

| Layer                         | Knows about                              | Doesn't know about                    |
|-------------------------------|------------------------------------------|---------------------------------------|
| `pkg/domain`                  | `Item`, `Variables`, `Icon` — neutral    | Alfred, Raycast, JSON                 |
| `internal/jetbrains`          | IDEs, prefs files, XML                   | launchers, output format              |
| `internal/launcher`           | `Launcher` interface + registry          | JetBrains, file paths                 |
| `internal/launcher/alfred`    | `info.plist`, prefs.json, Script Filter  | Raycast                               |
| `internal/launcher/raycast`   | Raycast extension contract               | Alfred                                |
| `internal/cmd`                | cobra wiring + flags                     | how Alfred or Raycast actually work   |
| `forge/*`                     | CLI framework, theme, exec, exit codes   | this tool's domain                    |

**Folder layout:**

```
.
├── bin/hermes/                # entrypoint (main package)
│   └── main.go
├── pkg/
│   └── domain/                     # launcher-neutral types
│       ├── item.go                 # Item, Variables, Icon
│       └── product.go              # Product enum + Display()
├── internal/
│   ├── cmd/                        # cobra commands
│   │   ├── root.go
│   │   ├── search.go
│   │   ├── searchall.go
│   │   ├── open.go
│   │   ├── install.go              # dispatches to launcher.Installer
│   │   ├── doctor.go
│   │   └── configuration.go
│   ├── jetbrains/                  # domain core — pure, launcher-agnostic
│   │   ├── config.go
│   │   ├── locator.go
│   │   ├── repository.go
│   │   ├── extractor.go
│   │   ├── name.go
│   │   └── service.go              # orchestrates the above into []domain.Item
│   ├── launcher/
│   │   ├── launcher.go             # Launcher interface + InstallOpts
│   │   ├── registry.go             # NewRegistry(...), Get(name), Detect(env)
│   │   ├── alfred/
│   │   │   ├── adapter.go          # implements launcher.Launcher
│   │   │   ├── render.go           # Script Filter JSON
│   │   │   ├── installer.go        # parse prefs.json, write workflow bundle
│   │   │   └── assets/             # //go:embed info.plist.tmpl, icon.png
│   │   ├── raycast/
│   │   │   ├── adapter.go
│   │   │   ├── render.go           # typed JSON for the TS extension
│   │   │   └── installer.go        # no-op + helpful pointer to the store
│   │   └── generic/
│   │       └── adapter.go          # raw JSON passthrough (for debugging)
│   ├── iofs/                       # FS port
│   └── env/                        # Env port
├── extensions/
│   └── raycast/                    # ⚠ separate TS package, out of scope for 3.0 Go work
│       ├── package.json
│       ├── src/index.tsx
│       └── README.md
├── test/
│   └── fixtures/                   # golden JSON, fake home dirs
├── docs/
│   ├── specs/
│   │   ├── README.md               # spec index (forge convention)
│   │   └── original-spec.md        # 2.x snapshot (existing, mostly still valid)
│   ├── adr/
│   │   ├── 0001-go-rewrite-on-forge.md
│   │   ├── 0002-launcher-abstraction.md
│   │   ├── 0003-no-binary-self-install.md
│   │   ├── 0004-alfred-script-filter-frozen-contract.md
│   │   ├── 0005-raycast-extension-architecture.md
│   │   └── ...
│   └── tasks/
│       └── roadmap.md              # per-task checklist (forge convention)
├── .config/                        # mise, hk, cocogitto, golangci, goreleaser
├── .goreleaser.yaml
├── go.mod
└── README.md
```

**The `Launcher` interface (full proposal):**

```go
// internal/launcher/launcher.go
package launcher

import (
    "context"
    "io"

    "github.com/adaouat/hermes/pkg/domain"
    "github.com/adaouat/hermes/internal/env"
    "github.com/adaouat/hermes/internal/iofs"
)

type Launcher interface {
    // Name returns the canonical launcher identifier (e.g. "alfred", "raycast").
    Name() string

    // Detect reports whether this launcher is the running context, based on env
    // vars the launcher exposes (e.g. alfred_version for Alfred). Returns false
    // for launchers without a reliable signal.
    Detect(env env.Env) bool

    // Render writes items to w in the launcher's expected output format.
    Render(items []domain.Item, w io.Writer) error

    // Install performs launcher-specific setup. May be a no-op for launchers
    // installed out-of-band (e.g. Raycast Store extensions).
    Install(ctx context.Context, opts InstallOpts) error

    // Verify checks an existing install is healthy and reports findings.
    Verify(ctx context.Context, opts InstallOpts) (Report, error)
}

type InstallOpts struct {
    DryRun     bool          // print actions without writing
    Force      bool          // overwrite even if drift detected
    BinaryPath string        // resolved os.Executable() target
    Version    string        // current CLI version (for info.plist, prompts)
    FS         iofs.FS       // injected filesystem (real or fake)
    Env        env.Env       // injected environment
    Out        io.Writer     // human-readable output (uses forge/ui)
}

type Report struct {
    Installed bool
    Path      string   // where the install lives (e.g. workflow dir)
    Drift     []string // human-readable drift findings (empty when clean)
}
```

---

## 2. Open Questions (must be answered before kickoff)

These shape M0–M5. My recommendations are marked **R:**.

### 2.1 Naming

| Q | Question | R: |
|---|----------|----|
| N1 | Rename the binary from `alfred_jetbrains_cli` to something launcher-neutral? | **Yes — `hermes`.** The current name is a lie once Raycast lands. Ship a one-version `alfred-jetbrains-cli` shim binary that execs `hermes` for back-compat. |
| N2 | Rename the GitHub repo? | **No** for 3.0 (breaks every existing brew/mise/curl install). Add a redirect note in README. Re-evaluate for 4.0. |
| N3 | Rename the Go module path? | **Yes — `github.com/adaouat/hermes`** (matches the binary). The repo URL stays; Go's module path doesn't need to mirror the repo. |
| N4 | Keep the workflow bundle id `@bchatard-alfred-jetbrains-next`? | **Yes.** Changing it orphans every existing Alfred install. |

### 2.2 Install — Alfred

| Q | Question | R: |
|---|----------|----|
| A1 | Does `install --launcher alfred` still copy/rename the running binary into the workflow dir? | **No.** `info.plist` is generated with the absolute resolved path (`os.Executable()` + `EvalSymlinks`). brew/mise own the binary; Alfred shells out to it directly. |
| A2 | What happens to `--retain`? | **Removed.** Reason for existing (npm flow that mustn't move the binary) goes away once A1 lands. Accepted with a deprecation warning for one minor version. |
| A3 | How does the user know when the workflow itself is stale (e.g. `info.plist` schema changed in a new CLI version)? | **`updatecheck.Hinter`** on non-JSON commands; `install --verify` reports drift between embedded `info.plist` and the installed copy. |
| A4 | How do we handle brew prefix variability (Apple Silicon `/opt/homebrew/bin` vs Intel `/usr/local/bin` vs mise `~/.local/share/mise/...`)? | Resolve at `install` time. Document that the user must re-run `install` after migrating prefixes; `--verify` flags the drift. |
| A5 | Failure mode when Alfred isn't installed (`prefs.json` missing)? | Exit `exitcode.Config` with a clear message + URL to Alfred install page. |
| A6 | Codesigning / notarization for the released binary (so Gatekeeper doesn't block)? | **Required.** Configure in goreleaser; assumes Apple Developer ID secret in CI. Out-of-band setup. |

### 2.3 Install — Raycast

| Q | Question | R: |
|---|----------|----|
| R1 | What does `install --launcher raycast` actually do? | **Verifies the binary is on `$PATH` and prints the Raycast Store URL.** No bundle to write — Raycast installs the TypeScript extension itself. |
| R2 | Should the Raycast TS extension live in this repo or a separate one? | **This repo, under `extensions/raycast/`.** Single source of truth, easier version sync between the CLI's JSON contract and the extension that consumes it. Out of scope for the Go 3.0 work, but the directory is reserved. |
| R3 | Will the Raycast extension be published to the Raycast Store by us, or do users install it manually? | **Publish to the Raycast Store.** No "developer mode" install required for end users. |
| R4 | How does the Raycast extension find the binary? | Via `$PATH` (Raycast inherits the user's shell PATH). If not found, the extension shows a friendly "install via brew/mise" preference page. |
| R5 | Configuration per-product (which IDEs to show, custom binaries) in Raycast? | Raycast extension exposes preferences UI; preferences are passed to the binary via env vars matching the existing `jb_*` contract. **No new env-var surface.** |

### 2.4 Output format

| Q | Question | R: |
|---|----------|----|
| O1 | How is the active launcher chosen at runtime? | Precedence: `--launcher <name>` flag → `JB_LAUNCHER` env var → auto-detect (`alfred_version` set → alfred; `RAYCAST_*` set → raycast) → default = `alfred` (back-compat). |
| O2 | Should we keep `alfred_*` env vars (`alfred_debug`, `alfred_version`) or rename to `jb_*`? | **Keep the originals as aliases**; add `jb_debug`, `jb_launcher_version` as canonical names. Aliases removed at 4.0. |
| O3 | Add a `--launcher generic` output that emits launcher-neutral JSON? | **Yes** — useful for scripts, the Raycast extension, and future launchers. |
| O4 | Should the Alfred JSON envelope stay byte-identical to 2.6.2's? | **Yes.** Golden tests in M7 prove it. |

---

## 3. Phase Plan

Each milestone ends with: `go test ./...` green, `golangci-lint run` silent, `goreleaser --snapshot --clean` succeeds, manual smoke against a real macOS + JetBrains install.

### M0 — Repo bootstrap & decisions  (≈ 0.5 day)

- [x] Resolve §2 questions. Commit answers as `docs/adr/0001-go-rewrite-on-forge.md` + `docs/adr/0002-launcher-abstraction.md`.
- [x] `go mod init github.com/adaouat/hermes`; require `github.com/adaouat/forge@v0.17.x`, `github.com/spf13/cobra@v1.10.2` (forge sibling baseline).
- [x] Copy forge's `.config/` (mise, hk, cocogitto, golangci, typos, yamlfmt) — Tier-2 canonical scaffolding per forge ADR-0001.
- [x] `cmd/hermes/main.go` — minimal `cli.Run` skeleton.
- [x] `.github/workflows/ci.yml` — Go lint/test/build via forge's reusable workflow + `hk check`.
- [x] `.goreleaser.yml` — darwin arm64+amd64, Homebrew cask, checksums. No codesign block (ADR-0001 A6 defers codesigning for 3.0).
- [x] No side branch — work lands directly on `main` per `.claude/rules/workflow.md`'s pre-v1.0 rule. There is no 2.x Dart code in this repo to protect; the Dart predecessor (`bchatard/alfred-jetbrains-cli`) lives in a separate repo (ADR-0001 N2).

**Note (2026-06-21):** M0's scaffolding (go.mod, `cmd/hermes/`, `.config/`, CI, goreleaser)
was already in place from the initial bootstrap commit. The remaining work this session was
resolving §2's open questions and writing ADR-0001/0002. Several decisions deviate from the
roadmap's own `R:` recommendations — most notably a clean-break naming strategy (no binary
shim, new Alfred bundle id, `hermes_*` env-var prefix with no `jb_*`/`alfred_*` alias period)
and deferring codesigning. The "side branch" item never applied: this checkout has no GitHub
remote yet and no 2.x Dart code, so there's nothing to protect `main` from. See ADR-0001 and
ADR-0002 for full rationale. `bin/hermes/` (as originally sketched in §1's folder layout) was
never used — the entrypoint lives at `cmd/hermes/`, matching `.goreleaser.yml`'s `main: ./cmd/hermes/` and standard Go convention; the folder layout in §1 is stale on this point.

---

### M1 — Domain core (launcher-neutral)  (≈ 2 days)

**Objective:** translate the JetBrains discovery logic to Go behind ports, with no launcher awareness.

- [x] `internal/iofs/fs.go` — `type FS interface { Stat / ReadDir / ReadFile / Exists / Glob }`. Real impl wraps `os` + `path/filepath`. Test impl is an in-memory tree (consider `testing/fstest.MapFS` + a small `Glob` shim).
- [x] `internal/env/env.go` — `type Env interface { Lookup(key string) (string, bool); Home() string; Path() []string }`.
- [x] `pkg/domain/product.go` — `Product` typed-string enum with `Display()` method returning the JB human name. **Fixes [bug] toJbName mishandles caps.**
- [x] `pkg/domain/item.go` — `Item` struct (launcher-neutral; carries `Name`, `Path`, `IconPath`, `BinaryPath`, `IsModernBinary`, `Match`). No JSON tags here — adapters handle serialization.
- [x] `internal/jetbrains/config.go` — `Defaults()` + `Merge(custom map[string]any)`. No static cache. **Fixes [smell] static `_config` cache.**
- [x] `internal/jetbrains/locator.go` — `Locator{FS, Env, Product}`. First-match-wins (with `slog.Warn` on duplicates). **Fixes [bug] `singleWhereOrNull` throws on duplicates.**
- [x] `internal/jetbrains/repository.go` — `LocateSettingsDir` continues on missing paths, errors only when all exhausted. **Fixes [bug] throws on first miss.**
- [x] `internal/jetbrains/extractor.go` — XML extractors via `encoding/xml` if expressive enough, else `github.com/antchfx/xmlquery`. Home injected, no `os.Getenv` inside. **Fixes [bug] HOME force-unwrap.**
- [x] `internal/jetbrains/name.go` — drop the dead `.sln` branch. **Fixes [smell] @todo .sln.**
- [x] `internal/jetbrains/service.go` — orchestrates Locator + Repository + Name into `Service.Search(product, filter) []domain.Item` and `SearchAll(filter)`. **Single source of truth for what `search` and `all` produce — fixes [smell] duplicated logic.**
- [x] Delete `gateway` (no commented-out code in Go). **Fixes [bug] gateway dead code.**
- [x] Per-package unit tests; `testdata/` for XML fixtures.

**Note (2026-06-21):** Implemented bottom-up: `internal/env` and `internal/iofs` ports
(each with a real + in-memory fake, `envtest`/`iofstest`) first, then `pkg/domain`
(`Product`/`Display()`/`Item`), then `internal/jetbrains/{config,locator,repository,
extractor,name,service}` in that order, each with a failing test written first (TDD) and
committed as its own `feat` commit. `gateway` had nothing to delete — this repo never had
the Dart source, so there was no commented-out code to port or remove. XML fixtures were
inlined as literal strings in table-driven tests rather than separate `testdata/` files —
small enough to stay readable next to their assertions, and exact byte-for-byte porting of
the legacy `product_config.dart`/`product_locator.dart`/`projects.dart`/
`projects_extractor.dart`/`project_name.dart` defaults and quirks was verified against the
predecessor repo (`bchatard/alfred-jetbrains-cli`) directly, including faithfully
preserving non-obvious existing behavior that isn't on the [bug]/[smell] fix list (e.g.
Android Studio's settings-dir regex requiring the full `year.quarter.fix` form with no
2-part fallback, Fleet's unanchored substring-match settings-dir pattern, and the
"more than one child" empty-directory heuristic in `LocateSettingsDirectory`). One
deliberate API deviation from the legacy CLI: `Service.Search`'s per-project filter
(name/basename substring match) and the application/binary/settings-dir lookups all live
in the domain layer here, but unlike the legacy `SearchCommand`, this layer never builds
an Alfred-shaped "error result item" on failure — it just returns a Go `error`
(`*NotFoundError` at the boundary), leaving error *rendering* to the launcher adapter
(M2) and command layer (M3), per `coding.md`'s layering rule that `internal/jetbrains`
doesn't know about output format. Aggregate test coverage across the new packages is
85.7% (CI gate is 85%); `internal/ui` (pre-existing, out of scope) is the only 0%-covered
package pulling the average down.

---

### M2 — Launcher abstraction + Alfred adapter  (≈ 1.5 days)

**Objective:** define the `Launcher` interface and implement Alfred as the first concrete adapter. Lock the contract before adding Raycast.

- [x] `internal/launcher/launcher.go` — `Launcher` interface, `InstallOpts`, `Report` (signatures per §1).
- [x] `internal/launcher/registry.go` — `Registry` with `Register`, `Get(name)`, `Detect(env)`, `Default()`. (Wiring in `main` deferred to M3.)
- [x] `internal/launcher/alfred/adapter.go` — implements `Launcher`. `Detect()` checks `alfred_version`. `Name()` returns `"alfred"`.
- [x] `internal/launcher/alfred/render.go` — Script Filter JSON renderer. **Byte-identical to 2.6.2** for the same inputs. `RenderItem` wraps in the same envelope as `RenderItems`. **Fixes [bug] renderItem/renderItems shape mismatch.**
- [x] Cache TTL exposed via constructor option, default 86400. **Fixes [smell] hardcoded TTL.**
- [x] Debug items (`alfred_debug=1`) appended via copy-on-write, no caller mutation. **Fixes [bug] `_addDebug` mutates caller list.**
- [ ] Skeleton `internal/launcher/alfred/installer.go` + `assets/info.plist.tmpl` (template ready, no `Install` impl yet — deferred to M4).
- [x] `internal/launcher/generic/adapter.go` — emits launcher-neutral JSON (`[]domain.Item`). Used by tests, the future Raycast TS extension, and curious users (`hermes search --product phpStorm --launcher generic`).
- [x] Golden test: Alfred JSON output for a fixture domain.Item, asserted against a 2.6.2-captured baseline.

**Note (2026-06-22):** M2 complete except for the installer skeleton. Bottom-up implementation: `internal/launcher` (interface + registry), then `internal/launcher/alfred` (render → adapter, golden-tested independently), then `internal/launcher/generic`, then registry wiring into `cmd/hermes` (unused until M3). Exact `ResultItem` field semantics (computed vs. omitted, `_addDebug` item set, cache envelope) ported from legacy 2.6.2 `result_item.dart` and `response.dart`. **Key deviation:** golden fixture moved from `test/fixtures/alfred/` to `internal/launcher/alfred/testdata/` because `//go:embed` cannot traverse `..` paths — `testdata/` is Go-idiomatic. **Deferred to M4:** installer skeleton + `assets/info.plist.tmpl` (templating placeholder content would be thrown away when M4 designs the real workflow object graph; ADR-0001 already changed the install model and deferred the bundle id). **Deferred to M3:** third debug item ("Debug: Log `<file>`") requires the `slog` file handler M3 sets up. ADR-0004 documents the frozen `ResultItem`/envelope contract and the one deliberate behavior change (Render always uses the envelope, fixing [bug] renderItem/renderItems shape mismatch structurally).

---

### M3 — Commands (cobra) + forge wiring  (≈ 1.5 days)

**Objective:** all read-only commands work end-to-end through the launcher abstraction.

- [x] `internal/cmd/root.go` — root cobra command. Persistent flags: `--launcher <name>` (precedence per O1), `--config <path>`, `--debug`.
- [x] `internal/cmd/search.go`, `searchall.go`, `open.go`, `configuration.go` — each:
  - parses flags
  - calls `jetbrains.Service`
  - hands `[]domain.Item` to `launcher.Get(flag).Render(items, os.Stdout)`
- [x] Exit codes: every command returns `error` → `exitcode.Wrap(...)` at command boundary → `main` calls `exitcode.Resolve`. Custom domain code: `NotFound=10` (in forge's reserved 4–69 range). **Fixes [risk] no structured exit codes.**
- [x] No bare `catch` (Go has none — every error path is explicit). **Fixes [bug] bare catches.**
- [x] Logging: `slog` with stderr handler (never stdout — Alfred reads JSON from stdout). Debug mode mirrors to a temp file. **Fixes [smell] dual logger, [bug] `Logger.level` global mutation.**
- [x] `--version` / `version` subcommand handled natively by fang via `cli.Run`. **Fixes [smell] dead `'completion'` fast-track.**

**Note (2026-08-29):** Implemented bottom-up: `jetbrains.Service.Open` and
`ProductDetails` JSON tags (M1 extensions the `open`/`configuration` commands needed) and
the Alfred adapter's deferred third debug item (`alfred.WithLogFile`, M2's ADR-0004 note)
landed first, then `cmd/hermes`'s composition-root pieces (`parseProduct`, `loadConfig`,
`resolveLauncher`, `setupLogging`, the `runtime` struct wiring them together and `root.go`'s
persistent flags), then the four commands themselves. Cobra command files live directly in
`cmd/hermes` (package `main`), not a separate `internal/cmd` - continuing the precedent M0/M2
already set (see M0's completion note); `coding.md`'s layer table still names `internal/cmd`
and should be corrected in a follow-up doc pass. `--config <path>` is a flag the roadmap
already named but left behaviorally unspecified: it reads a `jb_custom_config`-shaped JSON
file and takes precedence over the env var. The legacy CLI's §5.5 failure semantics (render
an error item in Alfred on a not-found product) were deliberately **not** ported - `search`/`open`
now return a structured exit code (`exitNotFound = 10`) instead, a user-approved break recorded
in ADR-0006. `root.go`'s pre-existing (M0) unconditional `updateHint` call is now gated by a
`jsonOutputCommands` set to avoid a noisy version-upgrade banner appearing on stderr during
Alfred/Raycast-driven invocations (stderr output can surface in Alfred's debugger and similar
tools, which would be confusing even though it doesn't break the JSON stdout contract) - a
necessary gate exposed by M3's own commands, not a pull-forward of M6's full `updatecheck.Hinter`
wiring (which still needs to wire the Hinter itself into the non-JSON commands; `doctor`/`install`
are deliberately left off `jsonOutputCommands` because they don't emit JSON, so they already keep
the hint once M4 lands - see `root.go`'s own comment on that set). `rootCmd`/`PersistentPreRunE`/`updateHint` remain untested, consistent
with this file's existing precedent for thin wiring functions; the four `new*Cmd` constructors
(`newSearchCmd`, `newAllCmd`, `newOpenCmd`, `newConfigurationCmd`) are untested too, and unlike the
wiring functions above they aren't purely thin plumbing - they declare the CLI's actual flag names
(`--product`, `--filter`, `--path`) and `MarkFlagRequired` calls, so that user-facing surface is
currently unverified by tests as well. Accepted as a known gap for this milestone rather than an
oversight. Every piece the wiring functions call
(`setupLogging`, `loadConfig`, `resolveLauncher`, `newLauncherRegistry`, `runtime.init`) is
unit-tested with fakes.

---

### M4 — Alfred installer + `doctor`  (≈ 1.5 days)

**⚠ Blocked on §2.2 (A1–A6).**

- [x] Implement the `HERMES_DEBUG`/`hermes_debug` env var per ADR-0002 O2 (deferred from M3 -
  `--debug` flag was implemented, the env-var trigger was not). Resolve ADR-0002's own casing
  inconsistency between `HERMES_LAUNCHER` (uppercase, O1) and `hermes_debug` (lowercase, O2) as
  part of this task - pick one and use it consistently.
- [x] `internal/launcher/alfred/installer.go`:
  - Parse `~/Library/Application Support/Alfred/prefs.json`.
  - Resolve workflow target = `<prefs.current>/workflows/dev.adaouat.hermes.alfred/` (ADR-0001 N4's
    new bundle id, not the 2.x `@bchatard-alfred-jetbrains-next`).
  - Render `info.plist` from template with `{{.Version}}` and `{{.Binary}}` (`opts.BinaryPath`,
    already resolved by the caller via `os.Executable()` + `EvalSymlinks` - that resolution is
    cmd/hermes's job, not installer.go's, per M4's note below).
  - Write `info.plist` + `icon.png` atomically (tmp + `os.Rename`).
  - **Never move/copy the running binary.** **Fixes [risk] destructive install.**
- [ ] `internal/cmd/install.go` flags: `--launcher`, `--check` (dry-run drift report), `--verify` (post-install validation), `--prefs <path>` (escape hatch). No `--retain` flag: ADR-0001 A2 removes it immediately with no deprecation period (this line originally said otherwise; corrected to match the ADR - see M4's note below).
- [ ] `internal/cmd/doctor.go` — for each (or specified) product, prints paths searched / matched / regex applied / which recents file was used. Uses `forge/ui` status helpers + `Spinner.Step` for the structured output. Stdout, not Alfred JSON.

**Note (2026-08-30):** M4 in progress. `HERMES_DEBUG` landed first: `resolveDebug` (new
`cmd/hermes/debug.go`) implements ADR-0002 O2's flag-wins-else-env-var-presence precedence,
matching `alfred_debug`'s own presence-based semantics (any value counts, not just truthy
strings) rather than a `strconv.ParseBool` scheme. Wired into `runtime.init` ahead of
`setupLogging` so both the `--debug` flag and the env var reach the same debug-logging path.
Resolved ADR-0002's casing inconsistency by picking uppercase throughout (`HERMES_DEBUG`),
matching `HERMES_LAUNCHER` (O1) rather than the ADR's original lowercase `hermes_debug`
example — ADR-0002 O2 edited to match. `--debug`'s flag help text now names the env var,
mirroring `--config`'s existing precedent of naming `jb_custom_config`.

**Note (2026-09-01):** `internal/launcher/alfred/installer.go` landed next, full-ported from
the real, currently-installed `@bchatard-alfred-jetbrains-next` workflow (`info.plist` +
`icon.png`, sourced from this machine's live Alfred install since neither asset existed in
this repo or the old Dart repo's checkout) rather than written from a placeholder. Two docs
conflicts surfaced and were resolved toward the ADR before implementing, per user decision:
(1) the bundle id - roadmap §2.1 N4 said keep `@bchatard-alfred-jetbrains-next`, but ADR-0001
N4 already overrode that to "a new id, chosen during M4"; **`dev.adaouat.hermes.alfred`** was
chosen now and ADR-0001 N4 updated to record it (reverse-DNS under the `adaouat` org, replacing
the personal `fr.chatard.jetbrains.workflow`/`@bchatard-...` naming - every 2.x install is
orphaned, matching N1/N3's clean break). (2) `--retain` - this file's own task line said
"deprecate with a one-version warning," contradicting ADR-0001 A2's "removed immediately, no
deprecation." The ADR wins; the task line above is corrected and `install.go` (not yet built)
will carry no `--retain` flag at all.

Porting the real 51-object/34-script workflow (not a synthetic template) meant adapting live
production content, not just plumbing: every `./bin/alfred_jetbrains_cli <cmd> ...` script
body (32 of them) became `{{.Binary}} <cmd> ...` (ADR-0001 A1 - the binary is never copied
into the workflow anymore, so the old relative `./bin/` path is gone); the root `bundleid` and
`version` (`2.6.2` → `{{.Version}}`) and `webaddress` (→ `github.com/adaouat/hermes`) were
retargeted; the embedded `readme` was rewritten to drop the npm/Node.js install path (roadmap
M6 already decided to drop npm) in favor of `brew install adaouat/tap/hermes` +
`hermes install --launcher alfred`. One real **[bug]** surfaced and was fixed while porting:
the Android Studio "open" script invoked `--product studio`, inconsistent with its own
"search" script's (correct) `--product androidStudio` and with `pkg/domain.Product`'s actual
enum value - the old workflow would have failed on "open" for that product. A pre-existing
missing-letter misspelling of "trigger" across 15 product config labels ("Keyword to ___ X")
was corrected, since the typos linter flagged it as ambiguous between two corrections and
refused to auto-fix it. Everything else - all 51 objects'
connections/UIDs, the `uidata` canvas layout, the `userconfigurationconfig` keyword/edition
fields, `icon.png` - carried over byte-identical; a `diff` against the source file after
porting has exactly 43 hunks, matching this list. Go's `go:embed` cannot traverse `..`
(same constraint M2's note already hit for golden fixtures), so the two assets live at
`internal/launcher/alfred/assets/` rather than a repo-root `assets/`, contra this file's
original phrasing.

`install`/`verify` (installer.go) read exclusively through `opts.FS`/`opts.Env` (prefs.json,
drift comparison), but *write* (`os.MkdirAll`, the tmp+rename in `writeFileAtomically`) via
the real `os` package directly - `iofs.FS` is a read-only port (its own doc comment says so)
with no write method to abstract through, and `coding.md`'s "no direct os calls" rule names
only `os.Stat`/`os.Getenv`-style reads. `testing.md`'s determinism rule already permits real
disk access scoped to `t.TempDir()`, which is how `installer_test.go` exercises every write
path; one test (`TestAdapter_InstallAndVerifyDelegateToInstaller`) needed the real `iofs.New()`
rather than the usual `iofstest` fake, since `Verify`'s fake-backed reads can't observe what
`Install`'s real-disk writes produced. `opts.BinaryPath` resolution (`os.Executable()` +
`EvalSymlinks`) stays the composition root's job (`cmd/hermes`, not yet built) - consistent
with `iofs.New()`/`env.New()` only ever being constructed there. `InstallOpts.Force` is
accepted but unused for now (no task text specifies its semantics yet; left for `install.go`).
`Adapter.Install`/`Verify` now delegate to `install`/`verify`; the `ErrInstallNotImplemented`
placeholder sentinel is gone, replaced by `ErrAlfredNotInstalled` (ADR-0001 A5).

---

### M5 — Raycast adapter (JSON contract only)  (≈ 1 day)

**Objective:** ship the Go-side contract for Raycast. The TypeScript extension itself stays out of scope.

- [ ] `internal/launcher/raycast/adapter.go` — implements `Launcher`. `Detect()` checks for any `RAYCAST_*` env vars.
- [ ] `internal/launcher/raycast/render.go` — emits typed JSON the TS extension consumes. Schema documented in `docs/adr/0005-raycast-extension-architecture.md` and frozen. Example shape:
  ```json
  {
    "items": [
      {
        "id": "MyProject",
        "title": "MyProject",
        "subtitle": "/Users/.../my-project",
        "icon": "/Applications/PhpStorm.app",
        "actions": [
          { "type": "open", "binary": "/opt/homebrew/bin/phpstorm", "path": "/Users/.../my-project" }
        ]
      }
    ]
  }
  ```
- [ ] `internal/launcher/raycast/installer.go` — `Install` is a no-op that prints the Raycast Store URL + verifies the binary is on `$PATH`. `Verify` checks `$PATH`, reports drift if the binary isn't reachable.
- [ ] Reserve `extensions/raycast/` directory with a `README.md` describing the JSON contract (no TS code yet).
- [ ] Golden test: Raycast JSON output for a fixture domain.Item.

---

### M6 — Distribution & update hint  (≈ 1 day)

- [ ] `.goreleaser.yaml` finalized:
  - `archives` — darwin arm64 + amd64.
  - `brews` — `bchatard/tap` formula. **Binary name: `hermes`** (with `alfred-jetbrains-cli` as a Brew formula alias for one minor version).
  - `signs` — Apple Developer ID codesign + notarize.
  - `checksum` — SHA256SUMS.
- [ ] `install.sh` — curl install script (resolves OS/arch, downloads from latest release, verifies SHA, drops in `$HOME/.local/bin`).
- [ ] mise: works against tagged releases automatically via the `github` backend. Document `mise use github:bchatard/alfred-jetbrains-cli` in README (repo name unchanged per N2).
- [ ] Drop npm distribution from CHANGELOG / README.
- [ ] Wire `updatecheck.Hinter` into the cobra root `PersistentPreRunE` — **only when stdout is not the Alfred or Raycast JSON path** (would corrupt the JSON envelope). Allowed on: `install`, `doctor`, `configuration`, `version`. Cache: `~/.cache/hermes/updatecheck.json`, 24h TTL.

---

### M7 — Migration, golden parity, cutover  (≈ 1 day)

**Objective:** prove the Go binary is a drop-in for the Dart binary in Alfred today, and a forward-compat path for Raycast tomorrow.

- [ ] `test/fixtures/home/` — committed fake macOS home with one project per common product (PhpStorm, WebStorm, IntelliJ, GoLand, Rider, Fleet, Android Studio).
- [ ] **Alfred golden parity**: for each command + fixture, capture the Dart 2.6.2 output (one-time) and assert byte-identity from the Go binary. Diff failures investigated and either fixed or documented.
- [ ] **Raycast golden contract**: schema-validate the Raycast output against the documented JSON Schema in `extensions/raycast/README.md`.
- [ ] CI matrix: macOS 14 + 15, arm64 + amd64.
- [ ] Manual QA: run against a real machine with at least 3 installed JetBrains products.
- [ ] Tag `3.0.0`, publish release with Homebrew formula + signed binary.
- [ ] Archive `lib/` + `bin/alfred_jetbrains_cli.dart` under `legacy/dart/`. Keep the Dart code on `release/2.x` for emergency backports for one quarter.

---

## 4. Effort Summary

| Phase | Title                                | Effort | Risk |
|------:|--------------------------------------|--------|------|
| M0    | Repo bootstrap & decisions           | 0.5 d  | Low  |
| M1    | Domain core (launcher-neutral)       | 2 d    | Med  |
| M2    | Launcher abstraction + Alfred adapter | 1.5 d  | Med (locks the contract — get it right) |
| M3    | Commands (cobra) + forge wiring      | 1.5 d  | Low  |
| M4    | Alfred installer + `doctor`          | 1.5 d  | **High** (depends on §2.2 answers) |
| M5    | Raycast adapter (JSON contract)      | 1 d    | Med (depends on §2.3 answers) |
| M6    | Distribution & update hint           | 1 d    | Med (codesigning setup) |
| M7    | Migration, golden parity, cutover    | 1 d    | Low  |
|       | **Total**                            | **~10 d** |    |

Post-3.0 (separate effort, not in this roadmap): publishing the Raycast TS extension at `extensions/raycast/`. Estimate ~3 days of TypeScript work + Raycast Store submission/review.

---

## 5. Issue → Phase Cross-Reference (from SPEC §10)

| Tag      | Issue                                          | Phase |
|----------|------------------------------------------------|------:|
| [bug]    | `locateSettingsDirectory` throws on first miss | M1    |
| [bug]    | `singleWhereOrNull` throws on duplicates       | M1    |
| [bug]    | `toJbName()` mishandles capitalised input      | M1    |
| [bug]    | `env['HOME']!` / `env['PATH']!` force-unwraps  | M1    |
| [bug]    | `_addDebug` mutates caller list                | M2    |
| [bug]    | `renderItem` vs `renderItems` shape mismatch   | M2    |
| [bug]    | `gateway` dead code                            | M1    |
| [bug]    | Bare `catch (e)` blocks                        | M3    |
| [bug]    | Global `Logger.level` mutation                 | M3    |
| [bug]    | `File('/dev/null')` POSIX-specific             | M3 (irrelevant in Go) |
| [smell]  | Static `_config` cache                         | M1    |
| [smell]  | Mutable JSON model fields                      | M1    |
| [smell]  | Duplicated SearchCommand / SearchAllCommand    | M1 (`jetbrains.Service`) |
| [smell]  | Dual `logger`/`consoleLogger`                  | M3 (`slog` + `forge/ui`) |
| [smell]  | Hardcoded cache TTL                            | M2    |
| [smell]  | Dead `'completion'` fast-track                 | M3 (cobra has it native) |
| [smell]  | `// @todo .sln` branch                         | M1    |
| [risk]   | Direct `Platform.environment` everywhere       | M1 (env port) |
| [risk]   | Destructive `install`                          | M4    |
| [risk]   | No structured exit codes                       | M3 (forge `exitcode`) |
| [dx]     | No integration test for JSON envelope          | M7 (golden tests) |
| [dx]     | README missing command reference               | M6    |

---

## 6. ADRs to Write Alongside the Rewrite

| ADR  | Title                                              | Phase |
|------|----------------------------------------------------|------:|
| 0001 | Go rewrite on forge                                | M0    |
| 0002 | Launcher abstraction (interface + registry)        | M0    |
| 0003 | No binary self-install (workflow points at host)   | M4    |
| 0004 | Alfred Script Filter contract frozen at 2.6.2      | M2    |
| 0005 | Raycast extension architecture (TS shells out to Go) | M5  |
| 0006 | Asset embedding (`embed.FS` + template)            | M4    |
| 0007 | Distribution channels (brew tap + mise + curl)     | M6    |
| 0008 | Golden-parity test contract with Dart 2.6.2        | M7    |

---

## 7. Out-of-Scope for 3.0 (parking lot)

- **Raycast TypeScript extension** — built in `extensions/raycast/` as a follow-up project, not part of 3.0 Go work.
- Other launchers (LaunchBar, Spotlight via shortcuts, Quicksilver). The abstraction exists; concrete adapters wait for demand.
- Linux/Windows support (Alfred + Raycast are both macOS-only as of writing).
- A daemon / long-lived process speaking to launchers over a Unix socket.
- Auto-import of legacy Dart-installed Alfred workflows (one-shot manual re-install is acceptable for a major bump).
- Renaming the GitHub repo (would break every existing brew/mise/curl install).

---

## 8. Risks & Mitigations

| Risk                                                                                       | Mitigation                                                                                      |
|--------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| Gatekeeper blocks the unsigned binary on first launch.                                     | Codesign + notarize in goreleaser (Q-A6). Failing that, document `xattr -d com.apple.quarantine`. |
| Brew/mise install path moves between machines, breaking `info.plist`'s embedded path.      | `install` records the resolved path; `install --verify` reports drift; `Hinter` nudges re-run. |
| Forge breaks its public API mid-rewrite.                                                   | Pin `forge v0.17.x` in `go.mod`; only consume the surface frozen in forge ADR-0007.              |
| Alfred 6.x changes `prefs.json` shape.                                                     | Wrap parse in a typed struct; surface `exitcode.Config` with a hint to file an issue.          |
| Raycast's extension API or JSON expectations shift before we ship the TS extension.        | The Go-side JSON is an internal contract between *our* binary and *our* extension. We control both ends; we can re-emit a different shape without breaking Raycast itself. |
| Golden tests diverge from Dart in trivial ways (JSON key order, whitespace).               | Custom `json.Marshaler` for stable key order; whitespace normalized in the differ.              |
| Launcher abstraction is over-engineered for 1.5 launchers (Alfred + planned Raycast).      | Interface kept *small* (5 methods). If a 3rd launcher never lands, the cost is one extra package layer — cheap. Reviewed at 4.0. |
