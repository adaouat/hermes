# Changelog

<!-- heraut-release: v0.2.0 -->
## [0.2.0](https://github.com/adaouat/hermes/compare/v0.1.0..v0.2.0) - 2026-10-14

### 🚀 Features

- *(cmd)* Wire launcher.Registry with alfred and generic adapters - ([855dc09](https://github.com/adaouat/hermes/commit/855dc09a40b0796b84b54c28e6a53d07e0385f5f)) by @bchatard

- *(cmd)* Thread the debug log file into the launcher registry - ([1cfeff2](https://github.com/adaouat/hermes/commit/1cfeff2a47579d128d99d6909ac59d0328e00859)) by @bchatard

- *(cmd)* Add parseProduct flag validation - ([e04b5fc](https://github.com/adaouat/hermes/commit/e04b5fc5ce21216a1c74a146932fa367c1012913)) by @bchatard

- *(cmd)* Add --config file loading via loadConfig - ([f6f846b](https://github.com/adaouat/hermes/commit/f6f846b96424c76f0843e1537b2d9bbe53a7b80d)) by @bchatard

- *(cmd)* Add resolveLauncher (ADR-0002 O1 precedence) - ([cb501f8](https://github.com/adaouat/hermes/commit/cb501f8cf9b8025aa99ba511d75e42f5b09dd829)) by @bchatard

- *(cmd)* Add setupLogging (stderr + debug temp file) - ([29968ac](https://github.com/adaouat/hermes/commit/29968acd4bd7951ff81aa844e405d1305c439339)) by @bchatard

- *(cmd)* Add the runtime composition root - ([2f97e5b](https://github.com/adaouat/hermes/commit/2f97e5bd38b527d1fd2b8526d15fb439e5270551)) by @bchatard

- *(cmd)* Wire root.go's persistent flags to the runtime - ([ce73552](https://github.com/adaouat/hermes/commit/ce7355237eb465ded7a461e8f8742cca0c073a20)) by @bchatard

- *(cmd)* Add the search command - ([f01caac](https://github.com/adaouat/hermes/commit/f01caac0b26d5b38d994047841f115fc073ac70b)) by @bchatard

- *(cmd)* Add the all command - ([8a2e5e8](https://github.com/adaouat/hermes/commit/8a2e5e8fb070f0af95c4129b6fc61cfdc9a75f8b)) by @bchatard

- *(cmd)* Add the open command - ([94ea5fc](https://github.com/adaouat/hermes/commit/94ea5fcc79ba91db1bdffbf50daff73c45c8ad9f)) by @bchatard

- *(cmd)* Add the configuration command - ([582a61d](https://github.com/adaouat/hermes/commit/582a61d2a279ad718fdd42be43cdc3d4be1d8b0a)) by @bchatard

- *(cmd)* Add HERMES_DEBUG env var trigger per ADR-0002 O2 - ([b4f7368](https://github.com/adaouat/hermes/commit/b4f7368154a3dd6b1983d9e3808c26d01c43f45e)) by @bchatard

- *(cmd)* Add hermes install command (roadmap M4) - ([271624e](https://github.com/adaouat/hermes/commit/271624ed6fbf546ff4db514665d5892ed995425c)) by @bchatard

- *(cmd)* Add hermes doctor command, closing roadmap M4 - ([e0ff3bb](https://github.com/adaouat/hermes/commit/e0ff3bb3a5e06c21fddf6714aefd0d819e8019b5)) by @bchatard

- *(cmd)* Wire the raycast adapter into the launcher registry - ([aba2787](https://github.com/adaouat/hermes/commit/aba27879f8639fb22b3a7d9a3fdc022fdad3ee09)) by @bchatard

- *(jetbrains)* Add Service.Open for the open command - ([6e6cc3a](https://github.com/adaouat/hermes/commit/6e6cc3aa272909ee1dbba85398800323c95aa8c4)) by @bchatard

- *(jetbrains)* Tag ProductDetails for JSON round-tripping - ([449fc85](https://github.com/adaouat/hermes/commit/449fc85c37d905bf14c05d1065b64014461413c3)) by @bchatard

- *(launcher)* Add Launcher interface, InstallOpts, Report - ([ce47f01](https://github.com/adaouat/hermes/commit/ce47f01c97c0869f3561bc55dfa64a6f37692383)) by @bchatard

- *(launcher)* Add Registry with Get/Detect/Default - ([85311b3](https://github.com/adaouat/hermes/commit/85311b35a86226214f3519d227cfefd096445983)) by @bchatard

- *(launcher/alfred)* Add Script Filter JSON renderer - ([bd37f54](https://github.com/adaouat/hermes/commit/bd37f54557fafc564bd3d7c3cadcb63b5d1e508c)) by @bchatard

- *(launcher/alfred)* Add Adapter implementing launcher.Launcher - ([7494333](https://github.com/adaouat/hermes/commit/7494333fd758095548b2c45b22b9ae5281ddcb95)) by @bchatard

- *(launcher/alfred)* Add WithLogFile for the deferred third debug item - ([b1538b7](https://github.com/adaouat/hermes/commit/b1538b7aa37fd0fb6a67b8180cbe582adbd02efc)) by @bchatard

- *(launcher/alfred)* Implement the Alfred installer (roadmap M4) - ([6350270](https://github.com/adaouat/hermes/commit/6350270be19a0e9d3ca6a2de8c2df4b4af84151a)) by @bchatard

- *(launcher/generic)* Add launcher-neutral JSON adapter - ([adbdbbb](https://github.com/adaouat/hermes/commit/adbdbbb267c4cb954b11b3cc83297392547762bd)) by @bchatard

- *(launcher/raycast)* Add the Raycast Launcher adapter (roadmap M5) - ([061d653](https://github.com/adaouat/hermes/commit/061d6538d2e011cd241f275561bcc5283211bf71)) by @bchatard


### 🐛 Bug Fixes

- *(deps)* Bump forge to v0.19.5 - ([d9a89cb](https://github.com/adaouat/hermes/commit/d9a89cb123a06c8800ccca122cc79a815dc502af)) by @bchatard

- *(heraut)* Migrate config to v0.58.0 schema, wire regenerate_changelog - ([062d09c](https://github.com/adaouat/hermes/commit/062d09c415d8294fb6301177aafb83d93d1b9fb1)) by @bchatard

- *(heraut)* Update bump field - ([dddcae0](https://github.com/adaouat/hermes/commit/dddcae094a364e1d8bf8b3ef9b99e5d7ce505535)) by @bchatard

- *(jetbrains)* Wrap Service.Open's error returns - ([fda7e60](https://github.com/adaouat/hermes/commit/fda7e60acd3c62400a2c1c6acdca37aa95ee1f7b)) by @bchatard

- *(jetbrains)* Wrap Service.Search's error returns to match Service.Open - ([44e3697](https://github.com/adaouat/hermes/commit/44e3697c519094d2be2e4ffcb4da79dca5eeb056)) by @bchatard

- *(jetbrains)* Wrap Service.Search's remaining RecentProjects error return - ([1603275](https://github.com/adaouat/hermes/commit/1603275bc9b1f6ca9fb16f6ccd999d462e41ec41)) by @bchatard

- *(launcher/alfred)* Guard empty version in debug item + rename env shadow - ([6a5db62](https://github.com/adaouat/hermes/commit/6a5db627b8bd2f093e749b850fdeeef372789ea0)) by @bchatard

- Strip ANSI escapes from the man page (bump forge to v0.19.3) - ([a5b4c13](https://github.com/adaouat/hermes/commit/a5b4c13a90d1efc88631ae3f6bbb755381245e12)) by @bchatard


### 🚜 Refactor

- *(cmd)* Use updatecheck.Hinter's own PostRun/CacheFile helpers - ([9e80514](https://github.com/adaouat/hermes/commit/9e80514430e3ff83487d37149feeafc70aaa9529)) by @bchatard


### 📚 Documentation

- *(adr)* Add 0004 Alfred Script Filter contract frozen at 2.6.2 - ([f03fc85](https://github.com/adaouat/hermes/commit/f03fc85a30d563411cc97f5c8b301a9eeece1302)) by @bchatard

- *(adr)* Correct golden fixture path in 0004 - ([0c4b4fa](https://github.com/adaouat/hermes/commit/0c4b4faa9aa66c0d5d0ea2ab67eed41dce0b4621)) by @bchatard

- *(adr)* Record the command error-handling break from 2.x - ([fef59e0](https://github.com/adaouat/hermes/commit/fef59e0f6b87c399c10d1c2e84c78f1ebc72ad7d)) by @bchatard

- *(adr)* Freeze the Raycast JSON contract (ADR-0005), close M5 - ([3ed902f](https://github.com/adaouat/hermes/commit/3ed902f198f10390384b0c648c61915ceb52d2b9)) by @bchatard

- *(goreleaser)* Cite heraut ADR-0018, not forge - ([534e053](https://github.com/adaouat/hermes/commit/534e053efc35caf8d2638aea20946d059edfac55)) by @bchatard

- *(plans)* Add M2/M3 planning docs - ([2939eff](https://github.com/adaouat/hermes/commit/2939eff16e160469323ac94afa510e0e065fd2de)) by @bchatard

- *(raycast)* Add the formal JSON Schema for the golden contract (roadmap M7) - ([d2452e0](https://github.com/adaouat/hermes/commit/d2452e066338fc1b164a184dca8a77a5066d2d25)) by @bchatard

- *(roadmap)* Close out M2 tasks 1–4 (launcher + alfred adapter) - ([41818fe](https://github.com/adaouat/hermes/commit/41818feda57e56b119cca57df053de095a0ce676)) by @bchatard

- *(roadmap)* Correct registry bullet — wiring in main deferred to M3 - ([3c3bf66](https://github.com/adaouat/hermes/commit/3c3bf66fe5d9a20022c849b6ad8618458d082d20)) by @bchatard

- *(roadmap)* Close out M2 — launcher abstraction + Alfred adapter - ([146b433](https://github.com/adaouat/hermes/commit/146b433911043d4370fe39d227e0dee15a8c44a6)) by @bchatard

- *(roadmap)* Close out M3 commands + forge wiring - ([a12421c](https://github.com/adaouat/hermes/commit/a12421ca668e8be3b536b28f3fe273fc63d2f683)) by @bchatard

- *(roadmap)* Correct M3 note's --config and update-hint claims - ([7a5baf4](https://github.com/adaouat/hermes/commit/7a5baf4a38345c81767477813a2b3a2b0a24741c)) by @bchatard

- *(roadmap)* Correct M3 note's doctor/install claim, track hermes_debug gap, document cobra-constructor coverage gap - ([d3d2c95](https://github.com/adaouat/hermes/commit/d3d2c95a2861ee32da8d316fdb6154dce6778630)) by @bchatard

- *(roadmap)* Close M6, distribution & update hint - ([390950f](https://github.com/adaouat/hermes/commit/390950f121b90e5dbe8172da388d896486d2864a)) by @bchatard

- *(roadmap)* Close out M7's fixture/parity/CI/QA work - ([092c41d](https://github.com/adaouat/hermes/commit/092c41daa25a3fc4a3878ed2ac8cfd3fc907ea8c)) by @bchatard

- *(rules)* Update golden-fixture path to testdata/ in testing.md - ([b821e06](https://github.com/adaouat/hermes/commit/b821e06827f91ace1dbb2006eb046af564ecdea6)) by @bchatard

- Add README, freeze the distribution model (ADR-0007), close M6 - ([36389ec](https://github.com/adaouat/hermes/commit/36389ece98dad8232f517cdb93d8d04aaf42a717)) by @bchatard


### 🧪 Testing

- *(cmd)* Close cmd/hermes coverage gaps flagged by the M3 final review - ([4ebcbfd](https://github.com/adaouat/hermes/commit/4ebcbfd7ce291e4abf43d645ba1a6ba29a8e3037)) by @bchatard

- *(launcher)* Assert Detect never probes the default adapter - ([3f50bbd](https://github.com/adaouat/hermes/commit/3f50bbd584714db176aa897c729470eb90484247)) by @bchatard

- *(launcher/alfred)* Embed golden fixture instead of os.ReadFile - ([fc390c5](https://github.com/adaouat/hermes/commit/fc390c5183b95315a7f65c6da49fb3ab32c7ad41)) by @bchatard

- *(launcher/alfred)* Use testdata/ for golden fixture, remove test/fixtures/ copy - ([dd9493a](https://github.com/adaouat/hermes/commit/dd9493a028bf8f263535b18d47c31c1d9803a588)) by @bchatard

- Prove Alfred golden parity against the real Dart 2.6.2 CLI (roadmap M7) - ([cc9bd69](https://github.com/adaouat/hermes/commit/cc9bd694c39eab94f23cc852ce58dd02ee43494f)) by @bchatard


### ⚙️ Miscellaneous Tasks

- *(hk)* Pass yamlfmt config to the check command too - ([525eea9](https://github.com/adaouat/hermes/commit/525eea91bc40623edad452dabdf453c8ab41623c)) by @bchatard

- *(macos)* Scope mise install to go only on the macOS matrix - ([3780875](https://github.com/adaouat/hermes/commit/37808755d9b58aaafc0e619592a7e82a7ac921f6)) by @bchatard

- *(mise)* Add build and run tasks - ([023b2d8](https://github.com/adaouat/hermes/commit/023b2d82b9aec08a8523651a1d6946538bf17b79)) by @bchatard

- *(release)* Use heraut release --set-version - ([7be0f5f](https://github.com/adaouat/hermes/commit/7be0f5f2878e2733f1b7732d4e1152ffa9382b18)) by @bchatard

- *(release)* Adopt packslip, SBOMs and cask completions - ([bff74a5](https://github.com/adaouat/hermes/commit/bff74a5b97a4b9d141bf9701bbbdffbbf4627ed8)) by @bchatard

- *(release)* Use forge's shared release actions - ([7a28792](https://github.com/adaouat/hermes/commit/7a2879262d0e40640cde2d7c6c32f56841a30403)) by @bchatard

- Track Claude Code project plugin settings - ([49c535b](https://github.com/adaouat/hermes/commit/49c535b8716e0880e845e0ec84704e3778775b15)) by @bchatard

- Add a real macOS test/build matrix (roadmap M7) - ([14b9eb3](https://github.com/adaouat/hermes/commit/14b9eb3a21111074f8456391c005cc863f79f9e9)) by @bchatard

- Add MIT license - ([b6ac56e](https://github.com/adaouat/hermes/commit/b6ac56e9d222c76a2f4174191dd9ac5fcd6ffbf9)) by @bchatard


### 💼 Other

- *(deps)* Bump forge to v0.17.3 - ([5e238a7](https://github.com/adaouat/hermes/commit/5e238a7faa689df7f6284edb40133dd97d3f3321)) by @bchatard

- *(deps)* Bump forge to v0.18.0 - ([63f2744](https://github.com/adaouat/hermes/commit/63f27449d1fab60318c969a6ae47f68a3baf2f1d)) by @bchatard

- *(deps)* Bump forge to v0.19.0 - ([efcdf9c](https://github.com/adaouat/hermes/commit/efcdf9ca8a5e5e0ce1864ea4b53d0d03c3a9c12c)) by @bchatard

<!-- heraut-release: v0.1.0 -->
## [0.1.0](https://github.com/adaouat/hermes/compare/v0.0.0..v0.1.0) - 2026-06-21

### 🚀 Features

- *(domain)* Add Product enum with Display() - ([551d966](https://github.com/adaouat/hermes/commit/551d9668320d97fdcb4265cc22866427fe136a05)) by @bchatard

- *(domain)* Add launcher-neutral Item struct - ([a708e0f](https://github.com/adaouat/hermes/commit/a708e0fca5eb3d8956183cb341f89c8663b0471a)) by @bchatard

- *(env)* Add Env port with real and envtest fake implementations - ([501c83f](https://github.com/adaouat/hermes/commit/501c83ffdd2a116fa4dfa67805905918181ef94e)) by @bchatard

- *(iofs)* Add FS port with real and iofstest fake implementations - ([76c3098](https://github.com/adaouat/hermes/commit/76c3098ff66481da7e2b36caca38131a17b449f1)) by @bchatard

- *(jetbrains)* Add product config Defaults, Merge, and Load - ([0564f67](https://github.com/adaouat/hermes/commit/0564f672519ffbdccb19561ce7e2ef65f950e8e1)) by @bchatard

- *(jetbrains)* Add Locator for application bundles and binaries - ([062d87b](https://github.com/adaouat/hermes/commit/062d87b7828a5bf1f137effee4cd9a48b4d2532a)) by @bchatard

- *(jetbrains)* Add recent-projects XML extractors - ([65da362](https://github.com/adaouat/hermes/commit/65da3629c12b1771f84db5a5836af000113d2498)) by @bchatard

- *(jetbrains)* Add Repository for settings dir and recent projects - ([b64b363](https://github.com/adaouat/hermes/commit/b64b363aad8d97c7047f8d80008a28faccc69f34)) by @bchatard

- *(jetbrains)* Add ProjectName resolution chain - ([dece9f8](https://github.com/adaouat/hermes/commit/dece9f8866b8a1ca3ba9cfba25a277613e521f38)) by @bchatard

- *(jetbrains)* Add Service orchestrating Search and SearchAll - ([2452a99](https://github.com/adaouat/hermes/commit/2452a996883dad13f10f38d0aaf07f5a2e3afdc6)) by @bchatard


### 📚 Documentation

- *(adr)* Resolve M0 open questions, add ADR-0001 and ADR-0002 - ([8ef6886](https://github.com/adaouat/hermes/commit/8ef6886831b6b8b634e37c6b07aa7c822ab35f1e)) by @bchatard

- *(roadmap)* Close out M1 domain core - ([2329c84](https://github.com/adaouat/hermes/commit/2329c84c164816950fbe27271abf2677355ef74e)) by @bchatard

<!-- heraut-release: v0.0.0 -->
## [0.0.0] - 2026-06-21

### ⚙️ Miscellaneous Tasks

- Bootstrap hermes on forge - ([d66c579](https://github.com/adaouat/hermes/commit/d66c579a8a952a310c8a1642a88f2136870cd7ab)) by @bchatard
<!-- heraut-footer -->

---
_Generated by [heraut](https://github.com/adaouat/heraut) v0.80.0 at 16:17 on 2026-10-14._
