---
name: ksrc
description: Search/read 3rd-party Gradle (Kotlin/Java) dependency sources. Avoid directly accessing `.gradle`; instead, proactively use this skill to inspect source code of dependencies to learn API shapes or implementations. 
---

## How to use:

1. Search dependencies to retrieve coordinates and matches using rg-style globs: `ksrc search "class LocalDate\("`

If you want faster execution & less noise, consider adding:
- `--artifact` to limit search to one artifact, (or `--module` to also limit by version)
- `--subproject` to help discovery for monorepos/large modular apps
- `--targets` to limit to specific KMP targets. 

2. Read a file by returned id: `ksrc cat org.jetbrains.kotlinx:kotlinx-datetime:0.6.1!/kotlinx/datetime/LocalDate.kt --lines 1,200`

File-id format: `group:artifact:version!/path/inside/jar.kt`

## Common issues
- If, unexpectedly, no matches are found, try `--project` with app project (not monorepo root), specifying `--scope` (esp. for build-time deps), or `ksrc doctor`.
- `E_NO_SOURCES`: dependency sources not available; try `ksrc deps`, `ksrc fetch <coord>`, specify a project and scope.
- Gradle not found: a) run in a Gradle project dir, b) set `--project` path explicitly, c) install gradle on machine.
- Gradle build script is failing in the repo: `ksrc` falls back to cache-only resolution and warns; re-run with `-v` to see Gradle output for debugging.
- Gradle fails with unresolved class version: User's Local java in env is resolved to something unsupported by gradle. Help them fix Gradle<>JDK incompatibility.
- Ambiguous modules: use `--module`, `--group`, or `--artifact` to narrow scope.
- Give this tool generous timeouts. It can take a few minutes to download sources and set up gradle.

## Commands
### `ksrc search <pattern> [-- <rg-args>]`
Search dependency sources.

Output format: `<file-id> <line>:<col>:<match>`

Common flags:
- `--all` search across all resolved deps (default, slow)
- `--project <path>` project root (default `.`)
- `--subproject <name>` limit to a subproject (repeatable)
- `--targets <list>` limit KMP targets (comma‑separated: `jvm,android,iosX64`)
- `--config <name>` resolve specific configuration(s) (comma‑separated)
- `--scope <compile|runtime|test|all>`
- `--module <glob>` module filter (`group:artifact[:version]`)
- `--group <glob>` / `--artifact <glob>` / `--version <glob>`
- `--offline` only use cached sources
- `--refresh` force dependency refresh
- `--context <n>` shortcut for `rg -C <n>` (context lines emit column `0`)
- `--rg-args <args>` extra rg args (comma‑separated)
- `-- <rg-args>` pass through raw rg args
- `--show-extracted-path` include temp extracted paths in output (off by default)

### `ksrc cat <file-id|path>`
Print file contents.

Common flags:
- `--lines <start,end>` 1‑based inclusive range
- `--module <glob>` / `--group` / `--artifact` / `--version` to disambiguate when using a path

### `ksrc open <file-id|path>`
Open in `$PAGER` (defaults to `less -R`). Same flags as `cat`.

### `ksrc deps`
List resolved dependencies and source availability.

### `ksrc resolve`
Resolve and print source JARs: `group:artifact:version|/path/to/sources.jar`.

### `ksrc fetch <coord>`
Ensure sources for a coordinate exist: `group:artifact:version`.
Usually only needed with offline mode.

### `ksrc where <path|coord>`
Locate cached source JAR or file.

### `ksrc doctor`
Basic diagnostics for environment issues.
