@.claude/rules/workflow.md
@.claude/rules/testing.md
@.claude/rules/coding.md
@.claude/rules/claude.md

# CLAUDE.md — Hermes

Hermes is a Go CLI rewrite of the JetBrains-IDE project launcher behind the
[Alfred JetBrains workflow](https://github.com/bchatard/alfred-jetbrains) (predecessor:
`bchatard/alfred-jetbrains-cli`, Dart). Named after the Greek messenger god — the carrier of
messages between worlds, root of "hermeneutics" (interpretation/translation) — because the
tool's job is translating one domain (installed JetBrains IDEs + their recent projects) into
whichever launcher front-end is asking (Alfred today, Raycast next).

## What this tool does

Locates installed JetBrains IDEs, reads their "recent projects" preference files, and
renders the result for a launcher front-end so the user can fuzzy-search and open a recent
project in the right IDE.

```
hermes search --product <p>   # find recent projects for one IDE
hermes all                    # aggregate across every installed IDE
hermes open --product <p> --path <existing>  # resolve one project → launcher item
hermes install --launcher <alfred|raycast>   # launcher-specific setup
hermes configuration          # dump merged JetBrains product config
hermes doctor                 # explain why a product can't be located
```

One domain core (`internal/jetbrains`), multiple launcher adapters
(`internal/launcher/{alfred,raycast}`) behind a `launcher.Launcher` interface. See
`docs/tasks/roadmap.md` §1 for the full architecture diagram.

## Docs

- [`docs/specs/`](docs/specs/) — behavioural specification, starting with the legacy Dart
  CLI's documented behaviour (the compatibility baseline)
- [`docs/adr/`](docs/adr/) — architecture decision records
- [`docs/tasks/roadmap.md`](docs/tasks/roadmap.md) — the M0–M7 build plan. **Read this
  before starting any work** and follow the two-step roadmap flow.

## Tech stack

Go on [`github.com/adaouat/forge`](https://github.com/adaouat/forge) — forge supplies the
CLI framework (`cli.Run`, fang, theme), `exec`, `exitcode`, and `updatecheck`. hermes never
re-implements what forge ships, and never pushes its own domain logic (JetBrains discovery,
launcher adapters) into forge.
