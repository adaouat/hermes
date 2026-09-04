# 0008 — Golden-parity test contract with Dart 2.6.2

## Status

Accepted

## Context

Roadmap M7's objective is proving the Go binary is a drop-in for the 2.6.2 Dart CLI in
Alfred. That requires an actual reference: a real `alfred_jetbrains_cli` 2.6.2 build run
against a real fake macOS home, not just an assertion that the Go port "should" match. The
predecessor Dart repo (`bchatard/alfred-jetbrains-cli`) isn't part of this repo (ADR-0001
N2), but is available locally, and `dart compile exe` on its `2.6.2`-tagged source produces a
real reference binary. This ADR records how that binary was used and the resulting test
contract, since the reference binary itself can't be committed (it's not this repo's to
ship, and it's a machine-specific compiled artifact).

## Decisions

### Fixture: `test/fixtures/home/`

A committed fake macOS home covering three products, chosen to exercise the hard-won edge
cases `.claude/rules/testing.md` calls out rather than all 19 supported products (which
would mostly re-test what M1's per-product config table already unit-tests):

- **PhpStorm** — the standard case (single `ApplicationNames` entry, `year.quarter`
  settings-dir regex, `recentProjectDirectories.xml`).
- **Android Studio** — the settings-dir regex requiring the full `year.quarter.fix` form
  (`AndroidStudio2024.3.2`), not the `year.quarter` two-part fallback every other product
  accepts.
- **Fleet** — the unanchored substring settings-dir match (a bare `Fleet` directory, no
  version suffix) and the `backend/**/trusted-paths.xml` glob fallback (Fleet has no
  `recentProjectDirectories.xml`/`recentProjects.xml`/`recentSolutions.xml`).

Each product gets a fake `.app` bundle (`Contents/MacOS/<binary>`, matching the post-2023
"modern binary" layout `SettingsRegexp`'s sibling logic assumes) and a fake settings
directory with more than one child (`options/` + `codestyles/`, or `backend/` +
`settings.json` for Fleet) — `LocateSettingsDirectory`'s own "more than one child" heuristic
(testing.md) treats a single-child candidate as an empty/uninitialized install and skips it.

### Isolation: `jb_application`/`jb_settings`/`jb_binaries`, not bare `HOME`

Both the reference Dart binary and the Go test set `jb_application` and `jb_settings` to
paths under the fixture, rather than relying on `HOME`-relative defaults alone — the default
application search path list starts with the literal `/Applications`, a real,
machine-specific, uncontrolled directory (this development machine has real PhpStorm/
IntelliJ/Android Studio/WebStorm installs). `jb_binaries` is set to a path that doesn't
exist, so `LocateBin` can only ever resolve a fixture binary via the appended
`<app>/Contents/MacOS` entry — never a real binary shadowing it from `$PATH` (the exact
hazard M4's doctor smoke-test note already flagged once: "the smoke test's `binary:` line
resolved to this machine's real JetBrains Toolbox script rather than the fixture's fake
binary"). `HOME` is still set to the fixture root, since `$USER_HOME$` expansion inside the
recent-projects XML and the project paths themselves both need it.

### Golden fixtures: `{{HOME}}`-templated, not raw captures

`test/fixtures/golden/*.json` are the reference binary's real output for the given command +
fixture, with the machine-specific absolute fixture path replaced by a `{{HOME}}` token
(`test/golden_parity_test.go`'s `wantGolden` substitutes it back in with the *test-run's own*
resolved fixture path before comparing). A raw capture would only ever match on the exact
machine and directory layout it was captured from; every other repo checkout (CI, a
contributor's machine) has a different absolute path to `test/fixtures/home/`.

Capture procedure (not automated - a one-time step per fixture change, same as the roadmap's
own "one-time" wording): `dart compile exe bin/alfred_jetbrains_cli.dart` from a
`2.6.2`-tagged checkout of the predecessor repo, run against the fixture with the env
overrides above, `tail -1` to drop the CLI's own stdout-interleaved log lines (the 2.6.2
logger writes to stdout, not stderr - the JSON is always the final line), then template out
the fixture path.

### `open`'s golden fixture is the enveloped shape, not a raw capture

`open_phpstorm.json` (raw 2.6.2 capture, unenveloped - kept for reference/documentation) and
`open_phpstorm_enveloped.json` (what `TestGoldenParity_Open` actually asserts against) are
deliberately different files. ADR-0004 already recorded that `open`'s output gains the
`{"cache":...,"items":[...]}` envelope in the Go rewrite (fixing `[bug] renderItem vs
renderItems shape mismatch` structurally, since there's only one `Render` path) - `open`'s
golden test proves *that* shape, not byte-identity with the legacy unenveloped one.

### Scope: three products, not all nineteen

`search --filter` and the other sixteen products aren't covered by golden fixtures. Their
per-product location/matching rules are already unit-tested (with fakes) per M1; what this
milestone's fixtures add is proof that the real `iofs`/`env` implementations and the Alfred
renderer agree with the real reference binary end-to-end, on real disk - not a product
coverage sweep. Extending to more products is cheap (the pattern is established) if a future
regression specifically motivates it.

## Consequences

- `test/golden_parity_test.go` is the only test in the suite that touches the real
  filesystem outside `t.TempDir()` (`test/fixtures/home/` is a real, committed tree) - an
  explicit, narrow exception to testing.md's "no filesystem outside `t.TempDir()` or
  `iofs.FS` fakes" rule, justified because proving the *real* `iofs.New()`/`env.New()`
  implementations is the entire point of this test.
- The 2.6.2 reference binary itself is not committed anywhere in this repo (not this repo's
  artifact to ship); recapturing after a fixture change requires local access to the
  predecessor Dart repo and a Dart SDK.
- `search`/`all`'s byte-identity is proven; `open`'s deliberate divergence is proven instead
  of disproven parity.

## References

- `docs/tasks/roadmap.md` M7
- [`0001-go-rewrite-on-forge.md`](0001-go-rewrite-on-forge.md) N2 (predecessor repo stays
  separate)
- [`0004-alfred-script-filter-contract-frozen-at-2.6.2.md`](0004-alfred-script-filter-contract-frozen-at-2.6.2.md)
  — the `open` envelope divergence this ADR's golden fixture proves
- `.claude/rules/testing.md` — hard-won edge cases (settings-dir "more than one child"
  heuristic, Android Studio's full-version regex, Fleet's unanchored substring match)
