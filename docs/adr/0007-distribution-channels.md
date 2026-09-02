# 0007 — Distribution channels (Homebrew cask, mise, curl; heraut-owned releases)

## Status

Accepted

## Context

Roadmap M6 describes distribution in terms that predate this repo's actual bootstrap:
a `bchatard/tap` Homebrew **formula**, Apple codesigning in `.goreleaser.yaml`, and a
dedicated `install.sh` script. None of that matches what M0 already scaffolded, what
`docs/adr/0001-go-rewrite-on-forge.md` A6 already decided, or the shared
`github.com/adaouat/*` distribution model documented in forge's
[`docs/guides/distribution.md`](https://github.com/adaouat/forge/blob/main/docs/guides/distribution.md)
(the template every `adaouat/*` CLI — `bifrost`, `heraut`, now `hermes` — copies its
`.goreleaser.yml` from). This ADR records the actual model, correcting the stale roadmap
wording, and confirms it's already working: `v0.1.0` shipped through this exact pipeline
before M6 started (`CHANGELOG.md`, `adaouat/homebrew-tap`'s `Casks/hermes.rb`).

## Decisions

### Homebrew: a cask, not a formula

`.goreleaser.yml`'s `homebrew_casks` block (not `brews`) publishes to `adaouat/homebrew-tap`
— a tap shared across the family, one cask per tool, not `bchatard/tap`. Per forge's guide,
`brews` (the formula form) is deprecated for pre-built binaries; casks are what `bifrost` and
`heraut` already use. No `alfred-jetbrains-cli` formula alias exists — the clean break already
decided in ADR-0001 N1/N2 (new binary name, new repo) extends to distribution naming too.

### No codesigning — reaffirms ADR-0001 A6

`.goreleaser.yml` has no `signs` block. ADR-0001 A6 already deferred codesigning/notarization
for 3.0; this ADR doesn't reopen it. README documents the `xattr -d com.apple.quarantine`
workaround.

### Release ownership: heraut, not goreleaser

`.goreleaser.yml` sets `release.disable: true` — goreleaser only builds binaries and
*generates* the cask file (`--skip=publish`); it never talks to GitHub. `heraut release
--version` (`.github/workflows/release.yml`) owns tagging, changelog generation
(`.config/heraut.yml`), and the actual GitHub Release. A final workflow step pushes the
generated cask to `adaouat/homebrew-tap` once `heraut release` has uploaded the binaries —
skipped gracefully if `HOMEBREW_TAP_TOKEN` isn't set. This is the "heraut-owned" release
model forge's guide describes (as opposed to bifrost's "self-release," where goreleaser
itself creates the release); hermes adopted it at bootstrap, before this roadmap's M6 section
was written against the older, simpler assumption of a self-contained `.goreleaser.yaml`.

### No `install.sh`

Neither `bifrost` nor `heraut` ship a dedicated install script — their READMEs document a
plain curl-download-chmod-move one-liner instead (see their `## Install` → `### Prebuilt
binary` sections). hermes's `README.md` follows the same pattern. A bespoke `install.sh`
would be the only one in the family, with no shared maintenance and no more capability than
the one-liner already provides (raw binaries need no extraction, so there's nothing an
OS/arch-resolving script would do beyond what `mise`/Homebrew already automate).

### Checksums: `checksums.txt`, not `SHA256SUMS`

The roadmap named the file `SHA256SUMS`; the actual (and family-wide) name is
`checksums.txt` (`.goreleaser.yml`'s `checksum.name_template`). Purely a naming
correction — no behavioral difference.

### Update hint

`cmd/hermes/root.go` wires `updatecheck.Hinter{...}.PostRun()` as `PersistentPostRunE`,
gated by `jsonOutputCommands` (`search`/`all`/`open` stay silent — their stdout is a
launcher's parsed JSON envelope). This already matches forge's `Hinter`/`CacheFile` helpers
(no hand-rolled cache-path or opt-out logic) as of this milestone. `--version`'s output does
**not** get the hint: cobra's built-in version-flag handling (`fang.WithVersion`, wrapping
`cobra.Command.Version`) returns before any `Persistent*Run` hook fires, by cobra's own
design (`Command.execute()` checks the `version` flag and returns early, ahead of the
`PersistentPreRunE`/`PersistentPostRunE` loop). Neither `bifrost` nor `heraut` — the two
other `updatecheck.Hinter` adopters in the family — work around this either; it's an accepted
characteristic of the shared fang+cobra stack, not a hermes-specific gap.

## Consequences

- Any change to the cask/release/checksum model should update forge's shared
  `docs/guides/distribution.md` first (or explicitly diverge with a reason recorded here),
  since three tools now depend on that guide staying accurate.
- `docs/tasks/roadmap.md` M6's original bullet list (`brews`, `signs`, `SHA256SUMS`,
  `install.sh`) is superseded by this ADR; the roadmap's checkboxes are annotated
  accordingly rather than silently reinterpreted.
- `--version` staying hint-free is a known, permanent limitation of this stack, not a TODO.

## References

- `docs/tasks/roadmap.md` M6
- [`0001-go-rewrite-on-forge.md`](0001-go-rewrite-on-forge.md) A6 (codesigning deferral), N1/N2
  (clean-break naming this ADR's Homebrew-naming decision extends)
- forge `docs/guides/distribution.md`, `docs/adr/0005-updates-via-package-managers.md`
- `.goreleaser.yml`, `.config/heraut.yml`, `.github/workflows/release.yml`
