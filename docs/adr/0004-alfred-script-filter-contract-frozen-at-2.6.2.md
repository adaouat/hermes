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
