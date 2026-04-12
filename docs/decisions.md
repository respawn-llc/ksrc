# Decisions and Architecture

This file records non-obvious decisions, tradeoffs, and architecture notes. Update it whenever we make a new call.

## Purpose
Provide single-command search and file read for Gradle dependency sources, without mutating the project and without a ksrc cache/index.

## Goals
- One-liner search (`ksrc search "<pattern>" --module group:artifact` or `ksrc search "<pattern>"`).
- One-liner file read (`ksrc cat <file-id>` or `ksrc open <file-id>`).
- Deterministic dependency resolution using the project graph.
- Use Gradle caches only; no repo writes.
- No extra setup or discovery, out-of-the box "good enough" search results with no fiddling or per-project setup, with broad support. 
- Prioritize KMP projects (all targets), then Android projects, then JVM Gradle projects, when deciding tradeoffs.

## Non-goals
- IDE integration.
- Build/test/run tasks.
- Source search outside Gradle dependency sources.

## Resolution Order (as of 2026-01-07)
Order by likelihood of success and cost. Stop after the first stage that yields sources.
1) Root build
2) buildSrc (only if root produced no sources, buildSrc exists, and not offline)
3) Included builds (only if still no sources and include-builds enabled; BFS traversal)

Rationale: avoid expensive Gradle runs unless needed; prioritize the most likely resolution source.

## buildSrc Handling
- buildSrc is attempted only when it exists and is a Gradle build.
- buildSrc failures are non-fatal; warnings are emitted.
- When buildSrc yields sources, emit a warning to make provenance explicit.

## Composite Builds (includeBuild)
- Included builds are resolved only as a fallback (after root and buildSrc).
- Included build failures are non-fatal; warnings are emitted.
- Access to included builds depends on Gradle lifecycle; init script prints included builds when available.
- Wrapper selection: prefer local wrapper; fallback to root wrapper; then PATH.

## Buildscript Dependencies
- buildscript classpath dependencies are included by default (can be disabled).
- Rationale: many build tool artifacts (AGP, etc.) live on buildscript classpaths.

## Config Selection & Progressive Retry
- `--config` accepts glob patterns (e.g., `*debugCompileClasspath`).
- When `--config` is omitted and no sources are found, retry with Android debug classpaths:
  - `*debugCompileClasspath` for compile scope
  - `*debugRuntimeClasspath` for runtime scope
- This provides a better first-try UX for Android repos without making the default slow for every run.

## Error Handling & Warnings
- Root build failures are fatal.
- Fallback failures (buildSrc/included builds) are warnings; the command continues.
- Warnings are emitted to stderr.

## Performance Notes
- Each resolution stage starts Gradle and can be slow.

## 2026-01-24: Resolution orchestration split
- CLI delegates resolution to `internal/resolution` to keep command wiring thin.
- Gradle traversal is separated from invocation/parsing with an injectable resolver for tests.
- Search execution is separated from rg output parsing so search backends can evolve without changing CLI formatting.

## 2026-01-31: MCP tools return plaintext only (no structuredContent)
- Problem: some MCP harnesses (e.g. Codex) prefer `structuredContent` over `content`. The Go SDK auto-populates `structuredContent` for typed handlers (`ToolHandlerFor`), and when the output type was `struct{}`, it serialized to `{}`. That caused harnesses to ignore the real plaintext output in `content`.
- Decision: switch MCP tools to untyped handlers (`ToolHandler`) and supply explicit `InputSchema` to keep validation while ensuring we only emit `content` text. Errors are returned as `IsError=true` with a text payload in `content`.
- Tradeoff: we lose auto-generated output schemas/structured output, but avoid empty `{}` structured payloads and keep cross-harness behavior consistent for plaintext tools.

## 2026-04-12: Internal parsers use machine-readable records
- `internal/search` invokes `rg --json` and decodes typed `match`/`context` events instead of parsing human-oriented `path:line:col:text` output.
- `internal/gradle` init script emits `KSRCJSON\t<json>` records for deps, source jars, and included builds; the Go side ignores all non-prefixed Gradle log lines.
- External `ksrc search` output remains plaintext: `<file-id> <line>:<col>:<line-text>`. `<line-text>` is raw line content with trailing newline stripped and may contain literal `:`.

## 2026-04-12: Search always extracts into a persistent cache
- `rg --search-zip` cannot provide stable archive-entry provenance for `<file-id>` mapping, so search no longer uses it.
- `internal/search` extracts source jars into a persistent cache under the user cache dir, keyed by canonical absolute jar path.
- This intentionally models production Gradle cache behavior, where artifact paths are already checksum-addressed, and avoids extra hashing or metadata churn in the hot search path.
- `KSRC_EXTRACT_CACHE_DIR` overrides the cache root for tests and local debugging.
