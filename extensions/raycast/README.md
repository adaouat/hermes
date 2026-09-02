# hermes Raycast extension

**Status:** directory reserved. No TypeScript code lives here yet — building and publishing
the extension is out of scope for hermes 3.0 (`docs/tasks/roadmap.md` §7); it's a follow-up
project, estimated ~3 days.

This file documents the JSON contract the `hermes` binary already emits for `--launcher
raycast`, so the extension (whenever it's built) has a stable target to code against. The
contract itself is frozen by [`docs/adr/0005-raycast-extension-architecture.md`](../../docs/adr/0005-raycast-extension-architecture.md)
— any shape change needs a new ADR.

## What the extension will do

A Raycast "List" command that shells out to the `hermes` binary (found via `$PATH`), parses
its JSON output, and renders one row per recent project. Selecting a row opens the project in
its JetBrains IDE.

```
Raycast extension (TypeScript)
        │  child_process.execFile("hermes", ["all", "--launcher", "raycast"])
        ▼
   hermes binary (this repo)
        │  internal/launcher/raycast — the only Go code that knows this shape
        ▼
   {"items": [...]}  (stdout, JSON)
```

## Invoking hermes

```sh
hermes all --launcher raycast          # every installed product
hermes search --product phpStorm --launcher raycast   # one product
```

Per-product configuration (which binaries/settings dirs to use) flows through the existing,
frozen `jb_*` env vars (`jb_custom_config`, `jb_binaries`, `jb_application`, `jb_settings` —
see `docs/specs/original-spec.md` §5) — Raycast's native preferences UI sets these before
invoking the binary. There is no Raycast-specific configuration surface.

## Output shape

```json
{
  "items": [
    {
      "id": "MyProject",
      "title": "MyProject",
      "subtitle": "/Users/x/projects/my-project",
      "icon": "/Applications/PhpStorm.app",
      "actions": [
        {
          "type": "open",
          "binary": "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
          "path": "/Users/x/projects/my-project",
          "isModernBinary": true
        }
      ]
    }
  ]
}
```

- `id` / `title` — the project name.
- `subtitle` — the absolute project path.
- `icon` — the located IDE's `.app` bundle path (or `.icns` file), usable directly as a
  Raycast `Image.ImageLike` source.
- `actions` — empty when hermes couldn't locate a launchable binary for this item (nothing to
  open); otherwise exactly one `"open"` action today. The array shape leaves room for more
  action types later without a breaking change.
- `actions[].isModernBinary` — **the extension must branch on this** when invoking `binary`:
  - `true` (post-2023 DMG layout, binary path contains `MacOS`): launch via
    `open -na "<binary>" --args "<path>"`.
  - `false` (legacy `bin/` script layout): exec `<binary> <path>` directly.

  This mirrors the branch the Alfred workflow's own open script takes
  (`internal/launcher/alfred/assets/info.plist.tmpl`) — same underlying binaries, same
  invocation split, different launcher.

## Detecting Raycast

`internal/launcher/raycast`'s `Detect()` checks for a `RAYCAST_VERSION` environment variable.
Unlike Alfred (which sets `alfred_version` on every Script Filter invocation automatically),
Raycast does not auto-inject anything into a spawned child process — **the extension itself
must set `RAYCAST_VERSION` in the child's environment** before invoking `hermes`, e.g.:

```ts
execFile("hermes", ["all", "--launcher", "raycast"], {
  env: { ...process.env, RAYCAST_VERSION: environment.raycastVersion },
});
```

Passing `--launcher raycast` explicitly (as above) is simpler and doesn't depend on this at
all — auto-detect is a fallback for scripts/tooling that can't pass the flag, per ADR-0002
O1's precedence chain (`--launcher` flag → `HERMES_LAUNCHER` env var → auto-detect →
`generic` fallback).

## Installing

`hermes install --launcher raycast` never writes a bundle — Raycast Store extensions install
themselves. It only checks whether `hermes` is reachable on `$PATH` and reports the Raycast
Store URL once the extension is published. Use `--check` for a non-mutating preflight or
`--verify` to confirm after installing the binary via brew/mise.
