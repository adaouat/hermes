# Coding rules

## Architecture: domain core + launcher adapters

One domain core, many launchers. JetBrains discovery is launcher-neutral; launcher-specific
code (install rituals, output formats) is isolated behind a `launcher.Launcher` interface.
Full diagram and folder layout: `docs/tasks/roadmap.md` §1.

| Layer                          | Knows about                              | Doesn't know about                  |
|---------------------------------|-------------------------------------------|--------------------------------------|
| `pkg/domain`                   | `Item`, `Variables`, `Icon` — neutral      | Alfred, Raycast, JSON                |
| `internal/jetbrains`           | IDEs, prefs files, XML                     | launchers, output format             |
| `internal/launcher`            | `Launcher` interface + registry            | JetBrains, file paths                |
| `internal/launcher/alfred`     | `info.plist`, prefs.json, Script Filter    | Raycast                              |
| `internal/launcher/raycast`    | Raycast extension contract                 | Alfred                               |
| `internal/cmd`                 | cobra wiring + flags                       | how Alfred or Raycast actually work  |
| `forge/*`                      | CLI framework, theme, exec, exit codes     | hermes's domain                      |

If you find yourself importing "up" the stack (e.g. `internal/jetbrains` importing
`internal/launcher`), the design is wrong — fix the dependency direction, do not add the
import.

## The `Launcher` interface is a frozen contract

Once the Alfred adapter lands (roadmap M2), `Launcher.Render`'s output for Alfred is
byte-identical to the 2.x Dart CLI for the same inputs — Alfred parses it from stdout, so a
shape change breaks every existing user's workflow silently. Adding a launcher means adding
an adapter package; it never means changing an existing adapter's `Render`/`Install`
signature without an ADR.

`internal/iofs` (`FS`) and `internal/env` (`Env`) are the only doors to the filesystem and
the process environment from `internal/jetbrains` and `internal/launcher`. No direct
`os.Stat`/`os.Getenv` inside those packages — that's what makes them testable without
touching the real disk (see `testing.md`).

## Error handling

- **Always wrap.** Every `if err != nil` returns `fmt.Errorf("doing X: %w", err)`. The `%w`
  is mandatory; without it, callers cannot `errors.Is` / `errors.As`.
- **Never string-match errors.** Use `errors.Is(err, target)` for sentinels and
  `errors.As(err, &typed)` for typed errors.
- **Expose sentinels/typed errors at package boundaries** so `internal/cmd` can map them to
  exit codes via forge's `exitcode` package.
- **Never panic** in `internal/jetbrains`, `internal/launcher`, or `pkg/domain` — return an
  error. A panic here would crash mid-render and corrupt the launcher's stdout contract.
- **Never call `os.Exit` below `cmd/hermes/main.go`.** Every other layer returns an error;
  only `main` decides the process exit code via `exitcode.Resolve`.

## Code quality

- No comments unless the *why* is non-obvious — a hidden constraint, a subtle invariant, a
  workaround for a specific bug (reference the SPEC/ADR issue tag, e.g. `[bug]` from
  `docs/specs/original-spec.md` §10, when porting a fix). Never describe *what* the code does; well-named
  identifiers do that.
- No multi-paragraph docstrings. Exported identifiers still get a one-line doc comment.
- No features, abstractions, or refactoring beyond what the current roadmap task requires.
  YAGNI.
- Three similar lines are better than a premature abstraction.
- Only validate at boundaries: CLI flags, env vars, external files (`prefs.json`, IDE XML).
  Trust internal guarantees — no error handling for scenarios that cannot happen.
- No backwards-compatibility shims for code that does not exist yet.

## Charmbracelet dependencies

All charmbracelet packages use the `charm.land` module registry, not
`github.com/charmbracelet`.

```
go get charm.land/<module>/v2   # e.g. charm.land/huh/v2, charm.land/bubbles/v2
```

Never add `github.com/charmbracelet/<module>` as a direct dependency.

**Documented exceptions.** A few lower-level charm packages were never republished under
`charm.land` — their `go.mod` still declares `github.com/charmbracelet/<module>`, so the
vanity path cannot be `require`d. For these, the `github.com/charmbracelet/<module>` import
is the only option and is allowed:

- `github.com/charmbracelet/colorprofile` — color/TTY capability detection.
- `github.com/charmbracelet/x/term` — terminal detection.

Re-confirm before adding any other `github.com/charmbracelet` dependency: if a `charm.land`
path resolves *and* the module's `go.mod` declares it, use that; otherwise document it here.
