# 0001 — Go rewrite on forge: naming, distribution, Alfred install model

## Status

Accepted

## Context

hermes 3.0 replaces the 2.x Dart CLI (`alfred_jetbrains_cli`) with a Go rewrite on
[`github.com/adaouat/forge`](https://github.com/adaouat/forge), adding first-class support
for multiple launchers (Alfred today, Raycast next). `docs/tasks/roadmap.md` §2.1–§2.2 lists
the open questions that had to be answered before M0 could close. This ADR records those
answers.

## Decisions

### Naming (§2.1)

- **N1 — Binary name.** Renamed from `alfred_jetbrains_cli` to `hermes`. Clean break: no
  back-compat shim binary that execs `hermes` under the old name. Existing Alfred installs
  must be reinstalled (see N4).
- **N2 — GitHub repo.** This local checkout has no GitHub remote configured yet, so there is
  no existing repo to rename. The decision is: publish as a **new** repo at
  `github.com/adaouat/hermes` rather than continuing/renaming the old
  `bchatard/alfred-jetbrains-cli` repo. `.goreleaser.yml` already targets
  `adaouat/homebrew-tap` and `github.com/adaouat/hermes`, consistent with this. The old repo
  is left in place (or archived separately, out of scope here) with a README note pointing
  at the new one — accepted as part of 3.0 rather than deferred to 4.0.
- **N3 — Go module path.** `github.com/adaouat/hermes`. Already reflected in `go.mod`;
  ratified here.
- **N4 — Alfred workflow bundle id.** Changed from `@bchatard-alfred-jetbrains-next` to a new
  id (chosen during M4 install implementation). This orphans every existing Alfred install —
  accepted as part of the clean break; users reinstall via `hermes install --launcher
  alfred`. No migration path is provided.

### Alfred install model (§2.2)

- **A1 — Binary handling.** `install --launcher alfred` never copies or moves the running
  binary. `info.plist` is generated with the absolute resolved path
  (`os.Executable()` + `filepath.EvalSymlinks`). brew/mise own the binary; the Alfred
  workflow shells out to it directly.
- **A2 — `--retain` flag.** Removed immediately, no deprecation warning. This is a clean
  rewrite; there is no 2.x flag-compatibility promise.
- **A3 — Drift detection.** `updatecheck.Hinter` runs on non-JSON commands. `install
  --verify` diffs the embedded `info.plist` template against the installed copy and reports
  schema drift.
- **A4 — Brew prefix variability.** Resolved at `install` time (the live resolved path is
  embedded). Migrating prefixes (e.g. Intel → Apple Silicon) requires re-running `install`;
  `--verify` flags the drift instead of silently breaking.
- **A5 — Alfred not installed.** When `~/Library/Application Support/Alfred/prefs.json` is
  missing, exit with `exitcode.Config` and a message that includes a link to Alfred's install
  page.
- **A6 — Codesigning/notarization.** Deferred for 3.0. The binary ships unsigned; README
  documents the `xattr -d com.apple.quarantine` workaround for Gatekeeper. Revisit before any
  release intended for wider distribution.

## Consequences

- Every existing 2.x Alfred user must manually reinstall (new binary name, new repo, new
  bundle id, unsigned binary needing a quarantine workaround). This is acceptable because the
  rewrite is a clean break, not a patch release.
- Before cutting the first `v3.0.0` tag (M6/M7), the `adaouat/hermes` GitHub repo must
  actually exist (pushed from this local checkout) and the `adaouat/homebrew-tap` repo must
  exist with write access for the release workflow.
- No alfred_jetbrains_cli compatibility shim exists anywhere in the distribution chain.

## References

- `docs/tasks/roadmap.md` §2.1, §2.2, M0, M4, M6
- [`0002-launcher-abstraction.md`](0002-launcher-abstraction.md) — Raycast install model and
  output-format decisions
