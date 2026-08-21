# Testing rules

## TDD is required

Write the failing test before writing implementation code. The cycle is **red → green →
refactor**. If you are tempted to skip the test "because the change is small", stop — that
is exactly when the test is most valuable.

If the user asks you to skip tests, push back and explain why. Do not silently agree.

## Coverage discipline

Every exported function in `internal/jetbrains`, `internal/launcher`, and `pkg/domain` has
tests — these are the domain core and the frozen launcher contracts; an untested edge case
here either breaks discovery for a real IDE or corrupts a launcher's JSON envelope. Port
real-app coverage from the 2.x Dart suite (`docs/specs/original-spec.md` §7, §10) plus any
gap it names.

## Table-driven tests preferred

Group related cases into one test with a `[]struct` of inputs and expected outputs. Each row
gets a descriptive `name`:

```go
tests := []struct {
    name string
    in   string
    want string
}{
    {"flag wins over env", ...},
    {"env wins over .config", ...},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) { /* … */ })
}
```

## `iofs.FS` / `env.Env` — the contract-test workhorses

hermes never calls `os.Stat`, `os.ReadFile`, `os.Getenv`, or `Platform.environment`-style
globals directly inside `internal/jetbrains` or `internal/launcher` — every filesystem and
environment read flows through the `iofs.FS` and `env.Env` ports (`internal/iofs`,
`internal/env`). Tests inject a fake implementation instead of touching the real disk or
process environment:

```go
fs := iofstest.New(map[string]string{
    "/Users/x/Library/Preferences/PhpStorm2024.1/options/recentProjectDirectories.xml": fixtureXML,
})
env := envtest.New(map[string]string{"HOME": "/Users/x"})

loc := jetbrains.NewLocator(fs, env, jetbrains.PhpStorm)
```

When the assertion is about *which paths were searched* or *what was found*, use the fakes.
Never reach for a real `os.*` call in `internal/jetbrains` or `internal/launcher` tests.

## Golden tests for frozen launcher contracts

Alfred's Script Filter JSON and Raycast's JSON shape are external contracts other systems
parse — Alfred reads it from stdout, the Raycast extension expects a specific schema. Every
change to `internal/launcher/alfred/render.go` or `internal/launcher/raycast/render.go` runs
against a committed golden fixture (`<package>/testdata/`) and asserts byte-identity (Alfred) or
schema validity (Raycast). A diff is either a bug or a deliberate breaking change — the
latter needs an ADR before the golden file moves.

## Determinism — never break these

- **No time-of-day dependencies.** Any time-sensitive code (cache TTL, update-check) takes a
  `now func() time.Time` so tests can fix the clock.
- **No network calls.** `updatecheck` integration is tested against an `httptest.Server`,
  never the real GitHub API.
- **No filesystem outside `t.TempDir()` or `iofs.FS` fakes.** Tests never touch the source
  tree or the real home directory.
- **No environment leakage.** Set env with `t.Setenv(...)` so it is restored on test exit —
  never `os.Setenv` directly, and never read `env.Env` fakes that leak between test cases.

## Preserve hard-won edge cases

The suite encodes hard-won edge cases ported from the 2.x Dart implementation — settings-dir
discovery continuing past a missing candidate, duplicate-match first-wins-with-a-warning,
`Display()` not mangling already-capitalised product names, the `.idea/name` →
`.idea/.name` → `*.iml` → XPath → basename fallback chain for project naming, and more.
**Never delete a test row to make a change easier.** An assertion is load-bearing until
proven otherwise. Drop a row only when the behaviour it tested is deliberately changed — and
only with an ADR documenting it.

## Shared helpers live in one place

Reusable test helpers go in the package that owns the contract (`internal/iofs/iofstest`,
`internal/env/envtest`, `test/fixtures/`), never duplicated across test files. If a second
test needs the same setup, move it to the shared helper first, then both call it.

## When a hook or test fails

Fix the root cause. Do not comment out the assertion, add an unexplained `t.Skip()`, loosen
the assertion, or suppress the linter. Each defeats the test's purpose. If the test itself is
wrong, fix it in a separate commit with an explanation.
