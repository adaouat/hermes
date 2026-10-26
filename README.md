# hermes

Locate JetBrains IDEs and their recent projects, and hand the result to a launcher — Alfred
today, Raycast next — so you can fuzzy-search and open a recent project in the right IDE.

Go rewrite of the [Alfred JetBrains workflow](https://github.com/bchatard/alfred-jetbrains)'s
CLI, on [`github.com/adaouat/forge`](https://github.com/adaouat/forge).

## Install

### Homebrew (recommended)

```bash
brew install --cask adaouat/tap/hermes
```

The simplest choice for the Alfred workflow: `hermes install` records the stable
`$(brew --prefix)/bin/hermes`, which survives `brew upgrade`, and the cask clears the
Gatekeeper quarantine flag for you.

### `go install`

```bash
go install github.com/adaouat/hermes/cmd/hermes@latest
```

### mise

```bash
mise use github:adaouat/hermes
```

Or declare it in your `.mise.toml` / `mise.toml`:

```toml
[tools]
"github:adaouat/hermes" = "latest"
```

The Alfred workflow runs hermes through mise's `installs/github-adaouat-hermes/latest` link,
so it survives `mise upgrade` and `mise prune`. It runs the newest *installed* version,
though, even if you pinned an older one.

### Prebuilt binary

Download the raw binary for your platform from the
[releases page](https://github.com/adaouat/hermes/releases). Assets are named
`hermes_<version>_<os>_<arch>` (no archive wrapper), alongside a `checksums.txt` for
verification.

```bash
# example: macOS arm64 — replace <version> with the release tag
curl -L -o hermes "https://github.com/adaouat/hermes/releases/download/<version>/hermes_<version>_darwin_arm64"
chmod +x hermes
mkdir -p ~/.local/bin && mv hermes ~/.local/bin/   # a stable directory on your $PATH
hermes --version
```

Move the binary to its permanent location *before* running `hermes install`. The workflow
records the path hermes runs from, so installing from `~/Downloads` ties it to that file.

> **macOS / Gatekeeper:** the binary isn't codesigned yet (`docs/adr/0001-go-rewrite-on-forge.md`
> A6). If it's blocked after download, clear the quarantine flag:
> ```bash
> xattr -d com.apple.quarantine hermes
> ```

hermes prints a one-line upgrade hint when a newer release exists (skipped for JSON-output
commands, so it never lands in a launcher's parsed output); re-run your install method
(`brew upgrade hermes`, `mise upgrade hermes`, `go install …@latest`, or the curl command)
to upgrade. Set
`HERMES_CHECK_UPDATE=false` to disable the check entirely.

## Usage

```
hermes search --product <p>                   # find recent projects for one IDE
hermes all                                     # aggregate across every installed IDE
hermes open --product <p> --path <existing>    # resolve one project → launcher item
hermes install --launcher <alfred|raycast>     # launcher-specific setup
hermes configuration                           # dump merged JetBrains product config
hermes doctor                                  # explain why a product can't be located
```

Every command accepts `--launcher <alfred|raycast|generic>` to pick the output format;
without it, hermes auto-detects from the environment (`alfred_version`/`RAYCAST_VERSION`) and
falls back to `generic` — launcher-neutral JSON, useful for scripts and debugging.

### Alfred

```bash
hermes install --launcher alfred
```

Writes the Alfred workflow (`info.plist` + icon) into your Alfred preferences. The workflow
points at a stable path to your `hermes` binary (see [Install](#install)), never a
version-specific one, so it keeps working across upgrades. hermes never copies or moves the
binary itself. Use `--check` for a non-mutating drift report, or `--verify` to confirm after
installing.

> **Upgrading from 0.2.1 or earlier:** those versions recorded a version-specific path that
> `brew upgrade` or `mise prune` deletes. Run `hermes install --launcher alfred` once more.
> `hermes install --check` reports the stale path if you're affected.

### Raycast

The Go-side JSON contract exists (`internal/launcher/raycast`,
[`docs/adr/0005`](docs/adr/0005-raycast-extension-architecture.md)); the Raycast extension
itself isn't published yet. See [`extensions/raycast/README.md`](extensions/raycast/README.md)
for the contract in the meantime.

## Docs

- [`docs/specs/`](docs/specs/) — behavioural specification (the 2.x Dart CLI's documented
  behaviour, the compatibility baseline)
- [`docs/adr/`](docs/adr/) — architecture decision records
- [`docs/tasks/roadmap.md`](docs/tasks/roadmap.md) — the build plan
