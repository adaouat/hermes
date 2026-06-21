# Spec: alfred_jetbrains_cli

> **Version covered:** 2.6.2 of the predecessor Dart CLI (`alfred_jetbrains_cli`, snapshot of
> behavior at `3c4eefe`).
> **Status:** Descriptive — captures *what the legacy CLI does*, kept as the behavioural
> baseline for the Go rewrite. The companion `docs/tasks/roadmap.md` describes what 3.0
> (hermes) does differently.

---

## 1. Objective

`alfred_jetbrains_cli` is a **macOS-only Dart CLI** that backs the
[Alfred JetBrains workflow](https://github.com/bchatard/alfred-jetbrains).
It locates installed JetBrains IDEs, reads their "recent projects"
preference files, and emits **Alfred Script Filter JSON** so users can
fuzzy-search and open recent JetBrains projects from Alfred.

**Primary user:** macOS power-users who use Alfred and JetBrains IDEs.

**Success looks like:** typing an Alfred keyword (`phps`, `webs`, `idea`,
`all`, …) instantly shows recent projects with the correct IDE icon and
opens the project in the right IDE on Enter.

---

## 2. Tech Stack

| Layer            | Choice                                   |
|------------------|------------------------------------------|
| Language         | Dart `^3.10.0`                           |
| CLI framework    | `args` + `args/command_runner`           |
| JSON             | `json_annotation` + `json_serializable`  |
| Logging          | `logger` (file + console multi-output)   |
| XML parsing      | `xml` (`XmlDocument`, XPath)             |
| File globbing    | `glob` + `glob/list_local_fs`            |
| Asset embedding  | `embed`, `embed_annotation`              |
| Versioning       | `build_version`                          |
| Lints            | `lints/recommended.yaml`                 |
| Test framework   | `test`                                   |
| Build runner     | `build_runner`, `json_serializable`      |

---

## 3. Commands

Build:        `dart compile exe bin/alfred_jetbrains_cli.dart -o bin/alfred_jetbrains_cli`
Run (dev):    `dart run bin/alfred_jetbrains_cli.dart <command> [flags]`
Tests:        `dart test --coverage-path=coverage/lcov.info`
Format check: `dart format --output=none --set-exit-if-changed .`
Analyze:      `dart analyze`
Codegen:      `dart run build_runner build --delete-conflicting-outputs`

CLI sub-commands:

| Command         | Required flags                 | Output                          |
|-----------------|--------------------------------|---------------------------------|
| `install`       | `--retain` (optional)          | Human-readable install log      |
| `search`        | `--product <enum>`, `--filter` | Alfred Script Filter JSON       |
| `all`           | `--filter` (optional)          | Alfred Script Filter JSON       |
| `open`          | `--product`, `--path` (req.)   | Single Alfred item JSON         |
| `configuration` | —                              | Pretty-printed merged JSON      |
| (top-level)     | `--version` / `-v`             | `packageVersion` to stdout      |

---

## 4. Project Structure

```
bin/
  alfred_jetbrains_cli.dart        → entry point; flushes stdio then exits
lib/
  alfred_jetbrains_cli.dart        → CommandRunner wiring + top-level flags
  helper.dart                      → global constants, debugMode/alfredMode flags, parsePath
  logger.dart                      → file + console logger setup
  version.dart                     → generated package version
  command/
    command.dart                   → barrel exports
    install.dart                   → write workflow files into Alfred prefs
    search.dart                    → search one product
    search_all.dart                → search all products, ignore failures
    open_project.dart              → resolve one project path → Alfred item
    configuration.dart             → dump merged configuration JSON
  jetbrains/
    jetbrains.dart                 → barrel
    product.dart                   → enum + String.toJbName extension
    product_details.dart           → JsonSerializable models
    product_config.dart            → static defaults + custom-config merge
    product_locator.dart           → find Application + binary
    projects.dart                  → find settings dir + recent projects file
    projects_extractor.dart        → XPath extractors for each XML format
    project_name.dart              → resolve human-readable name for a project
  alfred/
    alfred.dart                    → barrel
    response.dart                  → AlfredResponse — wraps stdout JSON
    result_item.dart               → JsonSerializable Alfred item models + Builder
  assets/
    assets.dart                    → @EmbedStr info.plist, @EmbedBinary icon.png
  exception/
    not_found.dart                 → NotFoundException with troubleshoot field
test/
  helper_test.dart
  alfred/result_item_test.dart
  exception/not_found_test.dart
  jetbrains/
    product_test.dart
    product_config_test.dart
    project_name_test.dart
    projects_extractor_test.dart
  TESTABILITY_RECOMMENDATIONS.md   → existing doc; see §10
.github/workflows/                 → release.yml, validate.yml (macos-latest)
analysis_options.yaml              → lints/recommended
pubspec.yaml                       → deps + lint_staged config
cog.toml                           → cocogitto (commit/version automation)
sonar-project.properties           → SonarCloud config
```

---

## 5. Behavioral Contract

### 5.1 Environment variables (inputs)

| Variable           | Effect                                                                                  |
|--------------------|-----------------------------------------------------------------------------------------|
| `alfred_debug`     | If present → debug mode: pretty-print JSON, write log to temp file, append debug items  |
| `alfred_version`   | If present → "alfred mode": disable colors/emojis, silence console output of `logger`   |
| `jb_custom_config` | JSON string overriding defaults per-product (any of `applicationNames`, `preferencePrefix`, `binaries`) |
| `jb_binaries`      | `:`-separated path list to search for binaries (overrides `$PATH`)                      |
| `jb_application`   | `:`-separated path list to search for `.app` bundles (overrides `/Applications` defaults) |
| `jb_settings`      | `:`-separated path list to search for preference dirs (overrides `~/Library/...` defaults) |
| `HOME`             | Required; used to expand `~` and `$USER_HOME$` placeholders                              |
| `PATH`             | Used when `jb_binaries` is unset                                                         |

### 5.2 Output formats

- `search`, `all` → `{ "cache": { "seconds": 86400, "loosereload": true }, "items": [...] }`
- `open` → single `ResultItem` JSON object (no envelope)
- `install`, `configuration` → human-readable / pretty JSON via `consoleLogger`/`print`
- `--version` → bare version string via `print`

Each `ResultItem` carries: `uid`, `title`, `match`, `subtitle`, `arg`,
`autocomplete`, `text { copy, largetype }`, `icon { path, type? }`, and
optional `variables { jb_project_name, jb_bin, jb_search_basename, jb_is_new_bin }`.

`jb_is_new_bin` is `true` when the binary path contains `MacOS` (the
post-2023 DMG layout where `bin/` scripts are no longer shipped).

### 5.3 Discovery rules

**Application bundle:** searches `jb_application` (if set) or, by default,
`/Applications`, `~/Applications`, `~/Applications/JetBrains Toolbox`.
Match is exact equality of `basenameWithoutExtension` against the
configured `applicationNames`.

**Binary:** searches `jb_binaries` (if set) or `$PATH`, then *appends*
`<.app>/Contents/MacOS` from `locateApplication()`. Match is exact
basename equality against the configured `binaries` list.

**Settings directory:** searches `jb_settings` (if set) or, by default,
`~/Library/Application Support/Google`, `~/Library/Application Support/JetBrains`,
`~/Library/Preferences`. Match is a regex on the basename:

- Default: `^<prefix>(\d{1,4}\.\d)$`
- Android Studio: `^<prefix>(\d{1,4}\.\d(\.\d))$` (allows year.quarter.fix)
- Fleet: `^<prefix>$` (no version suffix)

Multiple versioned directories are sorted desc and the **first non-empty**
(`>1` child) directory wins.

**Recent projects file precedence** (under `<settingsDir>/options/`):
1. `recentProjectDirectories.xml`
2. `recentProjects.xml`
3. `recentSolutions.xml` (Rider)
4. Fleet only: glob `backend/**/trusted-paths.xml`

**Project name resolution** (per project path):
1. Contents of `<project>/.idea/name`
2. Contents of `<project>/.idea/.name`
3. Basename of the first `<project>/.idea/*.iml` file
4. XPath probes against `<project>/.idea/workspace.xml`
5. Fallback: `basenameWithoutExtension(projectPath)` (also handles `.sln`)

### 5.4 Supported products (19)

`androidStudio`, `appCode`, `aqua`, `cLion`, `cLionNova`, `dataGrip`,
`dataSpell`, `fleet`, `goLand`, `intelliJIdeaCommunity`,
`intelliJIdeaUltimate`, `phpStorm`, `pyCharmProfessional`,
`pyCharmCommunity`, `rider`, `rubyMine`, `rustRover`, `webStorm`,
`writerside`. (`gateway` is commented out in `product.dart` and
`product_config.dart`.)

### 5.5 Failure semantics

- `search` catches `NotFoundException` → renders **one** result item with the error icon and the `troubleshoot` string in the subtitle. Any other exception → one result item with `iconBod` ("nuclear PC") icon.
- `search_all` catches **every** exception per-product silently — failing products are skipped.
- `open` mirrors `search`'s error handling but with a single item.
- `install` does **not** catch errors; missing `prefs.json` or write failures terminate the process with an unhandled stack trace.

### 5.6 Install command behavior

1. Reads `~/Library/Application Support/Alfred/prefs.json`.
2. Reads `current` key as the Alfred preferences root.
3. Creates `<root>/workflows/@bchatard-alfred-jetbrains-next/`.
4. Writes embedded `info.plist` (with `{{WORKFLOW_VERSION}}` replaced) and `icon.png`.
5. Unless `--retain` is passed (or `Platform.executable` ends with `bin/dart[.exe]`), **renames the running binary** into `<workflow>/bin/`. After this, the just-installed binary is the only copy.

---

## 6. Code Style

The project follows `lints/recommended.yaml` and `dart format`. Example
of the existing style:

```dart
class JetBrainsProductLocator {
  FileSystemEntity? _bin;
  FileSystemEntity? _application;
  final JetBrainsProduct product;

  JetBrainsProductLocator(this.product);

  FileSystemEntity locateBin() {
    if (_bin != null) return _bin!;
    final env = Platform.environment;
    final paths = (env['jb_binaries']?.isEmpty ?? true)
        ? env['PATH']!.split(':')
        : env['jb_binaries']!.split(':');
    // ...
  }
}
```

Conventions observed:
- `lowerCamelCase` for symbols, `UpperCamelCase` for types.
- Barrel files (`*/jetbrains.dart`, `*/alfred.dart`, `command/command.dart`).
- Generated code in `*.g.dart` (committed via `build_runner`).
- Conventional Commits enforced by `cog.toml`.

---

## 7. Testing Strategy

- Framework: `package:test`.
- Tests mirror `lib/` structure under `test/`.
- Current coverage: **pure models, helpers, and extractors only**. Command classes, `JetBrainsProductLocator`, `JetBrainsProjects`, `AlfredResponse`, and the logger are **not unit-tested** (see `test/TESTABILITY_RECOMMENDATIONS.md` for why).
- File-system-touching tests use `Directory.systemTemp.createTempSync` + `tearDown` cleanup.
- CI runs on `macos-latest`: format check → `dart analyze` → `dart test` → SonarCloud scan.

---

## 8. Boundaries

**Always:**
- Run `dart format`, `dart analyze`, and the full test suite before tagging a release.
- Follow Conventional Commits (`cog.toml`).
- Keep CLI command names, flag names, and JSON output keys backwards-compatible within a major version.
- Bump `version` in `pubspec.yaml` *and* let `build_version` regenerate `lib/version.dart`.

**Ask first:**
- Adding/removing a `JetBrainsProduct` enum value.
- Changing default search paths (`/Applications`, `~/Library/...`).
- Changing the Alfred JSON envelope or `ResultItemVariables` keys.
- Adding a new env-var input.

**Never:**
- Hardcode user paths or absolute home directories.
- Print to stdout outside `AlfredResponse` / explicit `consoleLogger` paths (it pollutes Alfred Script Filter JSON).
- Catch and silently swallow exceptions in new code (see Known Issues §10).

---

## 9. Success Criteria

The CLI is "working as specified" when, on a clean macOS machine:

1. `alfred_jetbrains_cli install` writes the workflow into Alfred's prefs dir without prompting and the resulting workflow opens in Alfred.
2. For each installed JetBrains product, `alfred_jetbrains_cli search --product <p>` returns at least one item whose `arg` is an existing project path and whose `variables.jb_bin` is executable.
3. `alfred_jetbrains_cli all` aggregates results across products and never crashes when one product is missing.
4. `alfred_jetbrains_cli open --product <p> --path <existing>` returns exactly one item whose `variables.jb_bin` matches what `search` returns.
5. `alfred_debug=1 alfred_jetbrains_cli search --product phpStorm` appends version/log/timer debug items and writes a log file under `$TMPDIR`.
6. `alfred_jetbrains_cli configuration` emits a valid JSON document round-trippable through `JetBrainsProductsDetails.fromJson`.
7. `dart test` is green and `dart analyze` is silent.

---

## 10. Known Issues & Defects (to address in 3.0)

Tagged `[bug]`, `[smell]`, `[risk]`, `[dx]`. See `ROADMAP.md` for the
fix plan.

### Correctness

- **[bug] `locateSettingsDirectory` throws on first missing path** — `lib/jetbrains/projects.dart:39-43` raises `FileSystemException` instead of continuing to the next candidate, so users who only have `~/Library/Application Support/JetBrains` (the common case) hit a failure path that depends on iteration order.
- **[bug] `singleWhereOrNull` throws on duplicates** — `product_locator.dart:49` and `:95` throw if two matching files exist in the same directory (e.g. both `IntelliJ IDEA.app` and `IntelliJ IDEA Ultimate.app` after the rename). Should be `firstWhereOrNull`.
- **[bug] `toJbName()` mishandles already-capitalised input** — `'PhpStorm'.toJbName()` returns `'PHpStorm'` (a test pins this incorrect behavior).
- **[bug] Force-unwrap of `Platform.environment['HOME']!`** — `helper.dart:14`, `projects_extractor.dart:11`. Crashes with an unhelpful `Null check operator used on a null value` if `HOME` is unset (CI containers, launchd contexts).
- **[bug] Force-unwrap of `env['PATH']!`** — `product_locator.dart:30`. Same risk.
- **[bug] `_addDebug` mutates caller's list** — `alfred/response.dart:40` appends to the `items` list passed in by reference. Callers like `SearchCommand` re-render the same list (not currently re-used, but a foot-gun).
- **[bug] `renderItem` is inconsistent with `renderItems`** — does not add debug items, does not wrap in `{ items: [...] }`. `open` therefore produces a different shape than `search`.
- **[bug] `gateway` is dead code** — Commented out in both `product.dart` and `product_config.dart`. Either delete or restore.
- **[bug] Bare `catch (e)` blocks swallow real errors** — `search_all.dart:39` ("die silently"), `project_name.dart:75` (XPath fallback). Should at least log at `warning`.
- **[bug] `Logger.level = Level.debug`** — `alfred_jetbrains_cli.dart:36` mutates the **global static** in the `logger` package. Bleeds across logger instances and is untestable.
- **[bug] `File('/dev/null')` is POSIX-specific** — `logger.dart:46`. Not portable; acceptable for now since target is macOS but should be a `NullOutput` log adapter instead.

### Smells

- **[smell] Static cache `JetBrainsProductConfiguration._config`** — Global mutable state; can't be reset between tests. Acknowledged in TESTABILITY_RECOMMENDATIONS.md.
- **[smell] Mutable JSON model fields** — `ResultItem`, `ResultItemText`, `ResultItemIcon`, `ResultItemVariables`, `JetBrainsProductDetails` all use non-`final` fields with public setters. Should be immutable.
- **[smell] Duplicated logic between `SearchCommand` and `SearchAllCommand`** — Both build `allowedProducts`, both iterate projects with the same `map` body. Extract a `ProjectSearchService`.
- **[smell] All file I/O is synchronous (`*Sync`)** — Acceptable for a CLI but blocks the event loop; harder to instrument.
- **[smell] Dual logger setup (`logger` + `consoleLogger`)** — Two top-level instances with overlapping concerns. Replace with a single sink that picks output based on context.
- **[smell] `// @todo: check if it's necessary (should not)` in `project_name.dart:42`** — Dead `.sln` branch; the `basenameWithoutExtension` fallback below already handles it.
- **[smell] Hardcoded cache TTL (`60 * 60 * 24`)** — `response.dart:21`. Magic number; should be a constant or env-var.
- **[smell] `'completion'` fast-track in `runCommand` but no `completion` command registered** — `alfred_jetbrains_cli.dart:64`. Dead code path.

### Architecture / DX

- **[risk] Direct dependency on `Platform.environment`, `Directory`, `File` everywhere** — Untestable without spinning up real fixtures. Whole file covered in `TESTABILITY_RECOMMENDATIONS.md`.
- **[risk] `install` is destructive and irreversible** — Renames the running binary into the workflow dir on first run with no confirmation, no backup, and no `--dry-run`. Users invoking the CLI directly (not via npm) silently lose the binary they just ran.
- **[risk] No structured exit codes beyond `success`/`usage`** — Every command path returns `ExitCode.success.code`, even when an Alfred error item was rendered. Scripts cannot tell "no projects" from "failed to locate IDE".
- **[dx] No top-level `--help` example in README** — README only documents `install`.
- **[dx] `CHANGELOG.md` mixes 2025-12 and "0.0.27 - 2025-12-06" entries** — Cocogitto rewrite happened recently; the early history is collapsed.
- **[dx] No integration test for the actual JSON envelope** — `result_item_test.dart` covers models but never asserts the full `{ "cache": ..., "items": [...] }` shape that Alfred consumes.

---

## 11. Open Questions

1. Should 3.0 drop support for the pre-2023 `bin/` script layout (the `jbIsNewBin` flag) now that JetBrains DMGs no longer ship bin scripts?
2. Should the CLI ship a `doctor` / `diagnose` subcommand that explains *why* a product can't be located (paths checked, files seen, regex used)?
3. Is there appetite for an embedded HTTP-free fast path (e.g. a long-lived background process that Alfred talks to over a Unix socket) given that JSON generation is dominated by disk I/O?
4. Should the install command stop being destructive and instead leave the binary where it is, writing only the `info.plist` + symlinking?
5. Is JetBrains Gateway still worth re-enabling or should the commented-out code be deleted permanently?
