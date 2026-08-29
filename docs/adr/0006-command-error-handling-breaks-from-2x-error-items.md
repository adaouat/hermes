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
