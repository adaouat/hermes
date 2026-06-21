# Workflow rules

## Branching

**During the build phase (pre-v1.0)**: commits land directly on `main`. The roadmap is the
protection, not branches. One developer, one trunk.

**After v1.0 ships**: every working session starts on a new branch off `main`. Never commit
directly to `main`.

- Branch name (post-v1.0): `<type>/<short-description>` where type matches the
  conventional-commit type (e.g. `feat/raycast-adapter`, `fix/recent-projects-precedence`).
- Fetch with prune before branching:
  ```bash
  git fetch --prune --prune-tags --all --tags
  git checkout -b <type>/<short-description> origin/main
  ```

## Conventional commits

All commits follow [Conventional Commits](https://www.conventionalcommits.org/). Allowed types:

| Type       | Use for                                                    |
|------------|------------------------------------------------------------|
| `feat`     | New user-visible behaviour                                 |
| `fix`      | Bug fix in existing behaviour                               |
| `docs`     | `docs/specs/`, `docs/adr/`, README, in-code doc comments   |
| `chore`    | Tool config, repo housekeeping, dependency bumps           |
| `refactor` | Code change with no behaviour change                       |
| `test`     | Adding or rewriting tests, no production change             |
| `style`    | Formatting, whitespace, lint-only fixes                     |
| `perf`     | Performance-only change                                     |
| `ci`       | `.github/workflows/*`, GoReleaser, release tooling          |
| `build`    | `go.mod`, build system                                      |

**Scope** matches the affected package: `feat(jetbrains): continue past missing settings
dir`, `fix(launcher/alfred): renderItem matches renderItems envelope`, `docs(adr): add 0002
launcher abstraction`. Keep subject lines ≤72 characters. Use the body for the *why*, not
the *what*.

## Two-step roadmap flow

Task status is tracked inline in `docs/tasks/roadmap.md` (the M0–M7 milestone plan) via
`[ ]` / `[x]` checkboxes.

1. **Implement** — confirm the task is `[ ]`, then do the work (TDD: failing test first).
   Commit in logical pieces using the right conventional-commit type.
2. **Complete** — flip `[ ]` → `[x]` and add a one-paragraph note under the task describing
   actual decisions, deferred items, or deviations. Commit the roadmap update alongside the
   final implementation commit.

Never silently mark a task complete without the note. The note is what makes the roadmap a
living document.

## Git hooks (hk)

Hooks live in `.config/hk/config.pkl` and run on every commit (pre-commit linters,
commit-msg conventional-commit validation, prepare-commit-msg `typos`).

**Never** pass `--no-verify`, `--no-gpg-sign`, or any flag that bypasses hooks. If a hook
fails, fix the underlying issue.

## Lint fixes

Fix lint failures through `hk`, never the underlying tool directly (it applies the project's
configured file selection and flags):

```bash
hk fix             # fix everything fixable
hk fix -S <linter> # target one linter (e.g. hk fix -S golangci-lint, hk fix -S yamlfmt)
```

## Version pinning

Pin exact versions everywhere — no `latest` in mise config, `go.mod`, or CI workflows.

**Exceptions** — format/lint or editor tooling with no API surface that could break the
build may use `latest`: `pkl`, `tombi`, `typos`, `yamlfmt`, `gopls` (and similar LSP tools).

## GitHub Actions

Pin every action to a full commit SHA, never a mutable tag. Add the semantic version as a
comment so intent stays readable:

```yaml
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6
```

To update, find the new SHA for the desired tag (`github.com/<owner>/<action>/tags`) and
replace both the SHA and the comment. Never use `@v4`, `@main`, or `@latest`.

## Plans

Plans live in `docs/plans/`. Each captures one discrete unit of work — a phase, a
milestone, a non-trivial task, or a research/design spike. Name them descriptively in
lowercase kebab-case (`m1-domain-core.md`); never keep the auto-generated random name.

## Releases

hermes ships a **binary** distributed via Homebrew + mise + curl install (no self-replacing
binary — see forge ADR-0005's pattern). GoReleaser builds and signs darwin arm64/amd64 on a
`v*` tag. A tag that changes a frozen launcher contract (Alfred Script Filter JSON, Raycast
JSON) is preceded by the ADR that justifies it.
