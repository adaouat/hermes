# 0002 — Launcher abstraction, Raycast contract, output format

## Status

Accepted

## Context

hermes 3.0's defining architectural move is splitting the launcher-neutral JetBrains
discovery domain (`internal/jetbrains`) from launcher-specific output adapters
(`internal/launcher/*`) behind a `Launcher` interface (`docs/tasks/roadmap.md` §1). This ADR
records the answers to roadmap §2.3 (Raycast) and §2.4 (output format) needed before M0
could close, and ratifies the `Launcher` interface shape proposed in §1.

## Decisions

### `Launcher` interface

Adopted as proposed in `docs/tasks/roadmap.md` §1:

```go
type Launcher interface {
    Name() string
    Detect(env env.Env) bool
    Render(items []domain.Item, w io.Writer) error
    Install(ctx context.Context, opts InstallOpts) error
    Verify(ctx context.Context, opts InstallOpts) (Report, error)
}
```

Concrete adapters: `internal/launcher/alfred`, `internal/launcher/raycast`,
`internal/launcher/generic`. Adding a launcher means adding an adapter package; it never
means changing this interface or an existing adapter's `Render`/`Install` signature without a
new ADR.

### Raycast (§2.3)

- **R1 — `install --launcher raycast`.** Verifies the binary is reachable on `$PATH` and
  prints the Raycast Store URL. No bundle to write — Raycast installs the TS extension
  itself.
- **R2 — Extension location.** `extensions/raycast/` in this repo. Single source of truth
  for keeping the Go-side JSON contract and the TS extension in sync. The TS code itself is
  out of scope for 3.0 (per roadmap §7); only the directory + `README.md` describing the
  contract land now.
- **R3 — Distribution.** Published to the Raycast Store. No developer-mode install required
  for end users.
- **R4 — Binary discovery.** Via `$PATH`, inherited from the user's shell environment. If not
  found, the extension shows a friendly "install via brew/mise" preference page.
- **R5 — Per-product configuration.** Exposed through Raycast's native preferences UI;
  preferences are passed to the binary via the **existing, frozen `jb_*` env-var contract**
  (`jb_custom_config`, `jb_binaries`, `jb_application`, `jb_settings` — see
  `docs/specs/original-spec.md` §5). No new env-var surface for configuration.

### Output format / launcher selection (§2.4)

- **O1 — Launcher selection precedence.** `--launcher <name>` flag → `HERMES_LAUNCHER` env
  var → auto-detect (`alfred_version` set → alfred; `RAYCAST_*` set → raycast) → **no
  launcher resolved is not an error**: falls through to the `generic` adapter (O3). This
  replaces the roadmap's original proposal of `JB_LAUNCHER` + a default-to-alfred fallback —
  both superseded by the clean-break naming decided in
  [`0001-go-rewrite-on-forge.md`](0001-go-rewrite-on-forge.md).
- **O2 — Canonical config var prefix.** New cross-launcher config vars use the `hermes_`
  prefix (e.g. `hermes_debug`), introduced immediately with **no `jb_*` or `alfred_*` alias
  period** — superseding the roadmap's original `jb_debug`/`jb_launcher_version` proposal.
  This is scoped narrowly:
  - `hermes_*` is for new, launcher-neutral *behavioral* signals we invent (debug mode,
    launcher selection).
  - The existing `jb_*` *configuration* surface (`jb_custom_config`, `jb_binaries`,
    `jb_application`, `jb_settings`) is unrelated and stays frozen (R5, unchanged from
    2.x).
  - `alfred_version` and `alfred_debug` are signals **Alfred itself** sets automatically
    when invoking a Script Filter (or opening its debugger panel) — not ours to rename or
    alias. The Alfred adapter's `Detect()` reads `alfred_version` as-is; presence of
    `alfred_debug` is an Alfred-native detail orthogonal to the new `hermes_debug` mechanism,
    not deprecated, just not part of this rename.
- **O3 — Generic output.** Added. `--launcher generic` (and the implicit fallback in O1)
  emits launcher-neutral JSON (`[]domain.Item`) — used by scripts, the Raycast extension
  during development, and as the safe default when no launcher context is detected.
- **O4 — Alfred byte-identity.** The Alfred adapter's JSON envelope stays byte-identical to
  the 2.6.2 Dart CLI's output for the same inputs. M7 golden tests prove it. This is
  independent of the bundle-id/repo/binary renaming in ADR-0001 — the *shape* parsed by the
  Script Filter stays stable even though the *workflow identity* around it changes.

## Consequences

- No `jb_*`/`alfred_*` alias period exists for the new behavioral vars — any external
  tooling relying on a transitional alias must be updated directly to `hermes_*`.
- The existing `jb_*` configuration contract (binaries/application/settings/custom config)
  is untouched, so 2.x users migrating their env-var setup only need to change debug/version
  signals, not their product configuration.
- `generic` becomes a real fallback path, not just a debugging convenience — it must be kept
  correct and stable from M2 onward, not deferred.

## References

- `docs/tasks/roadmap.md` §1, §2.3, §2.4, M2, M5
- `docs/specs/original-spec.md` §5 (existing `jb_*`/`alfred_*` env-var contract)
- [`0001-go-rewrite-on-forge.md`](0001-go-rewrite-on-forge.md) — naming and Alfred install
  decisions
