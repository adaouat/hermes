# 0005 — Raycast extension architecture (TS shells out to Go)

## Status

Accepted

## Context

`internal/launcher/raycast` (roadmap M5) is the second concrete `Launcher` adapter. Unlike
Alfred, there is no existing 2.x contract to port — this ADR *defines* the JSON contract
`hermes --launcher raycast` emits, plus the launcher-selection and install semantics ADR-0002
§2.3/§2.4 already answered at the R1–R5/O1 level. The TypeScript extension that consumes this
output is out of scope for 3.0 (roadmap §7) and doesn't exist yet; this ADR is written so that
follow-up work has a stable, already-tested target.

## Decisions

### Frozen: the item JSON shape

```json
{
  "items": [
    {
      "id": "MyProject",
      "title": "MyProject",
      "subtitle": "/Users/.../my-project",
      "icon": "/Applications/PhpStorm.app",
      "actions": [
        { "type": "open", "binary": "/opt/homebrew/bin/phpstorm", "path": "/Users/.../my-project", "isModernBinary": true }
      ]
    }
  ]
}
```

- `id` / `title` ← project name (`domain.Item.Name`).
- `subtitle` ← project path.
- `icon` ← the located `.app`/`.icns` path, unchanged from `domain.Item.IconPath` — no
  Alfred-style `fileicon`/`.icns` type discrimination needed since Raycast's `Image.ImageLike`
  accepts a bare path.
- `actions` ← empty (`[]`, never omitted or `null`) when `domain.Item.BinaryPath` is empty —
  there's nothing to open. Otherwise exactly one `{"type":"open",...}` action today. Kept as
  an array, not a single object, so a future action type (e.g. "reveal in Finder") doesn't
  need a breaking shape change.
- `actions[].isModernBinary` ← `domain.Item.IsModernBinary`, **added beyond the roadmap's
  illustrative example shape**. It's required for correctness, not a nice-to-have: the Alfred
  workflow's own open script (`internal/launcher/alfred/assets/info.plist.tmpl`) already
  branches on this same field to choose between `open -na "<bin>" --args "<path>"` (modern,
  post-2023 DMG layout) and a direct `<bin> <path>` exec (legacy `bin/` script layout) — the
  Raycast extension needs the same branch to invoke the binary correctly, and has no other
  way to derive it.

Verified against a committed golden fixture
(`internal/launcher/raycast/testdata/search_basic.json`) per `.claude/rules/testing.md`'s
golden-test rule. Any shape change here needs a new ADR, per `.claude/rules/claude.md`'s
"frozen external contracts" rule.

### Detect signal: a single `RAYCAST_VERSION` variable, not a `RAYCAST_*` prefix scan

ADR-0002 O1 described the auto-detect signal loosely as "`RAYCAST_*` set". Implementing that
literally would require `env.Env` to enumerate or prefix-match arbitrary keys — a capability
it doesn't have today (`Lookup(key)`, `Home()`, `Path()` only), and would be a shared-port
change serving exactly one caller.

More importantly: unlike Alfred, which reliably sets `alfred_version` on every Script Filter
invocation itself, Raycast does not auto-inject environment variables into a spawned child
process — the extension author controls exactly what env the child gets. Since we own both
ends of this contract (our own not-yet-built TS extension, and this adapter), "detecting
Raycast via an env var" really means "the future extension deliberately sets one before
invoking hermes." A wildcard prefix scan buys nothing over a single named variable in that
world.

**Decision:** `Adapter.Detect` checks `RAYCAST_VERSION` via the existing `Lookup(key)`
method — the same single-`Lookup` pattern the Alfred adapter already uses for
`alfred_version`. The future extension is responsible for setting it explicitly (documented
in `extensions/raycast/README.md`). `--launcher raycast` or `HERMES_LAUNCHER=raycast` (ADR-0002
O1's higher-precedence options) don't depend on this at all and are the extension's simpler,
recommended path; auto-detect exists as the fallback for callers that can't pass a flag.

This supersedes ADR-0002 O1's literal "`RAYCAST_*`" wording with a concrete single-variable
signal — the same kind of concretization M4 did for the Alfred bundle ID (chosen during M4,
ADR-0001 N4 updated to record it) once an underspecified roadmap detail needed an actual
implementation.

### Install/Verify: `$PATH` reachability, no bundle to write

Per ADR-0002 R1/R4: `Install` never writes anything (Raycast Store extensions install
themselves) and reports whether `opts.BinaryPath`'s directory is one of `opts.Env.Path()`'s
entries — "reachable via `$PATH`" is the only thing meaningful to check, since that's how the
(future) extension's `child_process.execFile("hermes", ...)` call would find the binary.
`Verify` reports the same check as a `launcher.Report`, with `Installed` mirroring
reachability and `Drift` populated with a human-readable message when the binary isn't
reachable. Both point the user at `extensions/raycast/README.md` for the JSON contract, since
the Raycast Store listing itself doesn't exist yet (`storeNote` in `installer.go`) — printing
a real Store URL before the extension is published would be misleading.

## Consequences

- The Raycast contract is genuinely new (not ported from 2.x), so there's more freedom here
  than ADR-0004 had with Alfred — but that freedom ends the moment this ADR is accepted; the
  shape above is frozen the same way.
- `env.Env` stays unchanged (no `Environ()`/prefix-match addition). If a future launcher
  genuinely needs to enumerate env vars, that's the point to revisit this, not before.
- The extension itself (whenever built) must explicitly set `RAYCAST_VERSION` in the child
  environment for auto-detect to work at all; this is documented, not enforced by hermes.

## References

- `docs/tasks/roadmap.md` §2.3, §2.4, M5
- [`0002-launcher-abstraction.md`](0002-launcher-abstraction.md) — R1–R5 (Raycast open
  questions), O1 (launcher selection precedence, the original "`RAYCAST_*`" wording this ADR
  concretizes)
- [`0004-alfred-script-filter-contract-frozen-at-2.6.2.md`](0004-alfred-script-filter-contract-frozen-at-2.6.2.md)
  — the `isModernBinary`/`jb_is_new_bin` open-script branch this contract mirrors
- `extensions/raycast/README.md` — the contract from the (future) extension author's side
