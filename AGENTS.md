## Project Purpose
`ksrc` is a CLI for one‑liner search and file read of Kotlin dependency sources. It resolves dependency versions from the project, ensures source JARs are present in **Gradle caches**, and runs `rg` over those JARs.

## Stack
- Language: Go 1.22+
- CLI: cobra
- Search: external `rg`
- Gradle integration: per‑invocation init script (`-I <temp>`)
- File read: Go `archive/zip` + line slicing

## Philosophy / Rules
- Zero project mutation: no files written to the repo.
- No custom cache/index: use Gradle caches only.
- Minimize agent context pollution, compact outputs.
- Deterministic resolution: prefer project‑resolved versions; only fall back to cache‑latest if absent.
- Keep output stable and parseable.
- Fixed bug? -> Add a regression test.
- New feature? -> Cover with unit tests without asking, ask user if they want integration tests.
- Read `docs/decisions.md` for architecture and decisions; keep it updated with new decisions from the team.

## Directory / Module Structure (planned)
- `cmd/` — CLI entry points and wiring
- `gradle/` — init script generation, Gradle execution, output parsing
- `resolve/` — version selection, module filtering, cache scanning
- `search/` — rg invocation, result formatting, file‑id emission
- `cat/` — zip file read + `--lines` slicing
- `internal/` — shared helpers (logging, error codes)
- `testdata/` — minimal Gradle fixtures
- `docs/` — CLI spec and stack

## Common Tasks
- Add/adjust commands: update cobra wiring in `cmd/`
- Resolution changes: keep init script minimal and compatible with multiple Gradle versions.
- Search changes: must keep `rg` call scoped to resolved JARs only.
- After code changes, rebuild the binary to `./bin/ksrc` so the symlinked CLI updates for the user.
- Brew manipulation: The brew tap with the ksrc formulat is separate repo at https://github.com/respawn-app/homebrew-tap . It's usually cloned at the parent dir of the cwd (./../homebrew-tap/)

## Tests
- Unit: parsing, version selection, file‑id handling.
- Integration: run against `testdata/` Gradle fixture; no repo mutation; asserts on output format.

## Clean Merge Expectations
- Keep changes focused;
- Update ./docs and ./skills when CLI flags, outputs, APIs or formats change.
- Update AGENTS.md (this file) with learnings/rules/memories for future you.
