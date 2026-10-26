# 0009 — Record a stable binary path at install, not the symlink-resolved one

## Status

Accepted. Supersedes the "resolved path (`os.Executable()` + `filepath.EvalSymlinks`)" clause
of ADR-0001 A1, and A4's "the live resolved path is embedded".

## Context

ADR-0001 A1 had `install` write `info.plist` with the binary's symlink-resolved path. Under
the Homebrew cask (ADR-0007), `$(brew --prefix)/bin/hermes` is a symlink to
`Caskroom/hermes/<version>/hermes`, so A1 recorded the version-specific Caskroom path.
`brew upgrade` deletes the previous version directory, so every upgrade left the Alfred
workflow pointing at a missing binary until the user re-ran `hermes install`. `install
--check` reported only a generic "info.plist differs".

The same resolved path fed Raycast's `$PATH` reachability check (ADR-0002 R1/R4). Its
directory, `Caskroom/hermes/<version>`, is never on `$PATH`, so every Homebrew user was
told hermes is not on `$PATH`.

## Decision

- `cmd/hermes` passes `InstallOpts.BinaryPath` from `stableBinaryPath`: the first `hermes`
  in an absolute `$PATH` entry that is the same file (`os.SameFile`) as the running binary.
  For Homebrew that is `bin/hermes`, which survives upgrades. If no `$PATH` entry matches, the
  path as invoked (`os.Executable()`, made absolute, not symlink-resolved) is used. That covers
  a plain binary in `~/.local/bin` or `$GOBIN`.
- The path stays absolute and is still embedded at install time, so A1's "never copy or move
  the binary" and A4's "migrating prefixes requires re-running `install`" both still hold.
  Resolving `hermes` through `PATH` at run time inside the workflow script was rejected:
  Alfred runs scripts with a minimal `PATH` that excludes the Homebrew prefix, so the script
  would have to embed a `PATH` anyway.
- Alfred's `Verify` reads the binary path the installed `info.plist` records. If it differs
  from the current stable path, `Verify` reports that drift, and says separately when the
  recorded binary no longer exists. Re-running `install` fixes both.

## Consequences

- An existing working workflow keeps working. Re-running `install` changes only the recorded
  path. The Script Filter JSON contract (ADR-0004) is untouched.
- Workflows installed by ≤ 0.2.1 under Homebrew still record a Caskroom path. `install
  --check` now names it, and the fix is one more `hermes install`.
- **Known limitation, mise:** mise puts a shim on `$PATH`, not a link to the hermes binary,
  so no `$PATH` entry is the same file. hermes falls back to mise's versioned
  `installs/hermes/<version>/hermes` path, which is no worse than before.
  `install --check` flags it once that version is pruned.

## References

- `docs/adr/0001-go-rewrite-on-forge.md` A1, A4
- `docs/adr/0002-launcher-abstraction.md`, `docs/adr/0007-distribution-channels.md`
- `cmd/hermes/install.go` (`stableBinaryPath`), `internal/launcher/alfred/installer.go` (`verify`)
