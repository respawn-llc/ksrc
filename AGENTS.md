## Context
`ksrc` is a Go CLI for searching and reading Kotlin/Java dependency sources resolved from Gradle projects.
It resolves dependencies via Gradle, ensures `-sources.jar` artifacts exist in Gradle caches, then uses `rg` for search and zip reads for `cat/open`.
The tool is designed for scriptable/agent workflows: stable output, file-id chaining, and zero mutation of the target project.

Primary docs to keep aligned with code:
- `README.md`
- `docs/cli-api.md`
- `docs/decisions.md`
- `skills/ksrc/SKILL.md`

## Commands/workflow guidance

- Run the local quality gate used in CI before pushing:
```bash
./scripts/ci-format.sh
go vet ./...
go test ./...
```

- Bump dependency sources with:
```bash
./scripts/update-dependencies.sh
```
This updates Go modules/toolchain, sample and integration Gradle wrappers, the sample Gradle version catalog, Gradle fixture dependency pins, and pinned GitHub Actions.
Add `--verify` to run format/vet/test/build plus a sample smoke check after updating.

- Rebuild local CLI binary after code changes:
```bash
./scripts/build.sh
```
Use `./scripts/build.sh --out <path>` or `KSRC_BUILD_OUT=<path> ./scripts/build.sh` to choose a non-default output path.

- Run a fast functional smoke check against the sample project:
```bash
./bin/ksrc search LocalDate --module org.jetbrains.kotlinx:kotlinx-datetime --project ./sample
```

- For release-related changes, follow `docs/release-workflow.md` and use `scripts/update-brew-tap.sh` instead of manual formula edits.

## Testing instructions
- Add a regression test for every bug fix.
- Add unit tests for every new feature.
- Keep tests in the package being changed (`internal/<pkg>/*_test.go`) and prefer deterministic fakes/stubs over networked/real-project dependencies.

Test layers used in this repo:
- Default test suite (`go test ./...`): unit tests plus lightweight integration tests using `testdata/fixture/gradlew` fake wrapper and temporary jars.
- Real Gradle integration suite: gated by `KSRC_INTEGRATION=1` in `internal/cli/integration_gradle_test.go`; uses `testdata/integration` and `sample`.
- MCP integration test: `internal/mcpserver/integration_test.go` builds and launches `ksrc mcp` and exercises tool calls.

Useful test commands:
```bash
go test ./...
go test ./internal/cli -run Integration
KSRC_INTEGRATION=1 go test ./internal/cli -run IntegrationWithRealGradle
go test -cover ./...
```

## Project Layout & Module Map
- `cmd/ksrc/main.go`: CLI entrypoint.
- `internal/adapter/`: shared CLI/MCP adapter for resolution request building, diagnostics, source lookup, and plaintext output formatting.
- `internal/cli/`: Cobra command wiring, flags, and CLI-specific hints.
- `internal/resolution/`: orchestration layer from CLI to Gradle/cache resolution.
- `internal/gradle/`: init script generation from versioned embedded Groovy templates, Gradle invocation, traversal (`root -> buildSrc -> included builds`).
- `internal/gradlehome/`: effective Gradle user home resolution shared by Gradle invocation diagnostics and cache fallback.
- `internal/resolve/`: cache scanning, coordinate/file-id parsing, filtering/version selection.
- `internal/search/`: persistent source-jar extraction cache plus `rg --json` output parsing.
- `internal/cat/`: zip file reads and `--lines` range parsing.
- `internal/mcpserver/`: stdio MCP server and tool handlers.
- `internal/executil/`: command execution abstraction used for testability.
- `testdata/fixture/`: fake Gradle wrapper test fixture.
- `testdata/integration/`: real Gradle integration fixture.
- `sample/`: KMP/Android sample with Configuration Cache and Isolated Projects enabled, used for smoke coverage.
- `docs/`: API spec, decisions, release workflow.
- `scripts/`: local/CI build script, CI format check, install script, brew tap update automation.

## Dev environment tips
- Build output version is injected from `VERSION` via ldflags; without that, CLI reports `dev`.
- Keep `./bin/ksrc` current if your local shell/tooling points to it.

## Architecture Notes
- Keep command wiring thin in `internal/cli` and `internal/mcpserver`; keep shared source-tool behavior in `internal/adapter` and domain behavior in `resolution`, `gradle`, `resolve`, `search`, and `cat`.
- Keep Gradle init scripts as versioned embedded template files under `internal/gradle/templates/`; test template rendering directly and avoid large inline Go string literals.
- Keep ksrc Gradle resolution compatible with Configuration Cache and Isolated Projects; do not disable Configuration Cache globally.
- Preserve zero-mutation behavior for target Gradle projects; only temporary files are allowed.
- Resolution behavior is intentional and documented in `docs/decisions.md`.
- Gradle user home behavior is API surface: `--gradle-user-home` / MCP `gradleUserHome` override `GRADLE_USER_HOME`; cache fallback must use the same effective home as Gradle invocation.
- Gradle invocation failures fall back to cache-only resolution with warnings; this is part of UX contract.
- Search output and file-id contract are API surfaces; keep formats stable and parseable:
  - Search: `<file-id> <line>:<col>:<match>`
  - File-id: `group:artifact:version!/path/inside/jar.ext`
- MCP tools are plaintext-first; keep tool outputs in `content` and avoid introducing structured payload dependencies.

## Code style
- Follow `gofmt` strictly (`./scripts/ci-format.sh` is authoritative).
- Keep CLI stdout machine-parseable; send diagnostics/warnings/verbose lines to stderr.
- Reuse `executil.Runner` instead of direct `os/exec` in logic that needs test coverage.
- Keep changes cohesive and package-local; avoid leaking command concerns into resolution/search internals.
- When flags/output/API surface changes, update all affected docs and skill files in the same change.
- Follow the commit style used by changelog automation: scoped/typed subject lines such as `cli: ...`, `mcp: ...`, `docs: ...`, `release: ...`, `fix: ...`.

Always keep this file up-to-date when you change project behavior, workflows, flags, outputs, release process, or architecture; remove stale facts promptly, and do not add temporary notes, generic boilerplate, or frequently changing product details.
