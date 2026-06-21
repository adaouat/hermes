# Claude behaviour rules

## Guard the architecture

hermes is a multi-launcher rewrite — its defining constraint is keeping the domain core
(`internal/jetbrains`) and the launcher adapters (`internal/launcher/*`) from bleeding into
each other.

- **Never** put launcher-specific logic (Alfred's `info.plist`, Raycast's JSON shape) into
  `internal/jetbrains` or `pkg/domain`. If asked to, push back and point at the layer table
  in `coding.md`.
- **Never** push hermes's domain logic (IDE discovery, project resolution) into forge — forge
  is the shared CLI-framework foundation across `adaouat/*` tools (zero domain logic; see
  forge's own ADR-0001). If something here looks "identical" to bifrost/heraut, that's a
  forge conversation to have explicitly with the user, not a silent extraction.
- **Frozen external contracts.** Alfred's Script Filter JSON and the 2.x `alfred_*`/`jb_*`
  env-var inputs are byte-compatible with the existing macOS workflow — see
  `docs/specs/original-spec.md` §5 and the golden-test rule in `testing.md`. A change here needs an ADR
  before the contract moves.
- When in doubt about a layering question, surface the conflict; let the user decide.

## Architectural decisions

- Challenge design choices when something seems wrong, suboptimal, or inconsistent with the
  stated goals — even if the user proposed it. State the concern clearly, explain why, then
  let the user decide.
- Never silently accept a decision that contradicts an ADR or a rule in this project.
  Surface the conflict and ask for clarification.

## Task discipline

- Never implement more than one roadmap task per session without explicit user approval.
- Never implement anything not on the current roadmap without asking first.
- If a task feels too large to implement safely in one step, break it down and propose the
  breakdown before starting.
- Always follow the two-step roadmap flow (see `workflow.md`). No shortcuts.

## TDD discipline

- Always write the failing test before writing implementation code.
- Never write implementation first and tests after.
- If the user asks to skip tests, push back and explain why the tests are needed.

## Code discipline

- Never implement more than what the current task requires.
- Never refactor, clean up, or improve surrounding code as part of a task unless the task
  explicitly asks for it.
- Never add features "while we're here".
- If linting or tests fail, fix the root cause — never silence the error or skip the check.

## Roadmap discipline

- Before starting a task: read `docs/tasks/roadmap.md`, confirm the task is `[ ]`.
- After completing a task: flip `[ ]` → `[x]`, add a one-paragraph note (actual decisions,
  deferred items, deviations), commit alongside the implementation.
- If a new task surfaces mid-implementation, add it to `docs/tasks/roadmap.md` — do not
  silently implement it.

## Source-of-truth hierarchy

When information conflicts, the order of trust is:

1. The current user message
2. `docs/adr/` (architecture decisions for this repo)
3. `docs/specs/` (behavioural specification of the 2.x CLI being ported)
4. `docs/tasks/roadmap.md` (planned approach)
5. Memory and prior conversation context

If two committed docs disagree, raise it — do not silently pick one.

## Irreversible operations

- Always ask before: force-pushing, deleting files/branches, resetting git history,
  modifying CI/CD pipelines, pushing to remote.
- Confirm scope explicitly — approval for one action does not imply approval for similar
  actions in other contexts.

## Memory

- Save important architectural decisions, user preferences, and non-obvious constraints to
  memory so they survive context resets.
- Verify memory before acting on it — stale memory is worse than no memory.
</content>
