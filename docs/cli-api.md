# ksrc CLI API Spec

# CLI Flags Reference

This doc mirrors `ksrc --help` for flags and outputs. Architecture decisions and tradeoffs live in `docs/decisions.md`.

## Command Overview

## Global Flags
- `-v, --verbose`: Show verbose output (including Gradle failure output)
- `--version`: Print version and exit

## Resolution Notes
- If Gradle resolution fails, `ksrc` falls back to cache-only resolution and emits a warning.
- Cache-only mode may return results that don't match the current project; it uses the latest cached version if no version is specified.
- With `--all`, cache-only mode scans all cached sources (can be large/slow).

### `ksrc search <pattern>`
Search dependency sources for a pattern, optionally filtered by module/group.

**Usage**
```
ksrc search <pattern> [flags] [-- <rg-args>]
```
If no selector is provided, `ksrc` defaults to `--all`. Use `--module`/`--group`/`--artifact`/`--version` to narrow scope.

**Key Flags**
- `--project <path>`: Project root (default: `.`)
- `--all`: Search across all resolved dependencies (default when no module/group/artifact/version is set)
- `--subproject <name>`: Limit resolution to a subproject (repeatable)
- `--targets <list>`: Limit KMP targets (comma‑separated, e.g. `jvm,android,iosX64`)
- `--config <name>`: Dependency configuration(s) or glob patterns to resolve (comma‑separated; default: inferred)
- `--module <glob>`: Filter by `group:artifact` glob
- `--group <glob>`: Filter by group
- `--artifact <glob>`: Filter by artifact
- `--version <glob>`: Filter by version
- `--scope <compile|runtime|test|all>`: Dependency scope (default: `compile`)
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)
- `--refresh`: Re‑resolve and re‑download sources
- `--offline`: Only use cached sources, error if missing
- `--context <n>`: Show N lines before/after matches (rg `-C`)
- `--max-results <n>`: Limit output
- `--rg-args <args>`: Extra args passed to `rg` (comma‑separated)
- `-- <rg-args>`: Pass through raw `rg` args without CSV encoding
- `--show-extracted-path`: Include temp extracted paths in output (off by default)
- `--emit-id <always|auto|never>`: Include file identifiers (default: `always`)

**Output (default)**
`<file-id> <line>:<col>:<match>` (use `--show-extracted-path` to include temp paths)

**Aliases**
- `ksrc rg` is an alias of `ksrc search`

---

### `ksrc cat <file-id|path>`
Print file contents to stdout. Resolves the file from dependency sources.

**Usage**
```
ksrc cat <file-id|path> [flags]
```

**Path Forms**
- Relative source path: `org/jetbrains/kotlinx/coroutines/flow/Flow.kt`
- Fully qualified path: `group:artifact:version!/org/.../Flow.kt`

**Flags**
- `--project <path>`
- `--module <glob>` (disambiguate)
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)
- `--lines <start,end>`: Output a line range (1‑based, inclusive; sed‑style)

---

### `ksrc open <path>`
Open a file in `$PAGER` (defaults to `less -R`).

**Usage**
```
ksrc open <path> [flags]
```

**Flags**
- `--project <path>`
- `--module <glob>` (disambiguate)
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)
- `--lines <start,end>`: Output a line range (1‑based, inclusive; sed‑style)

---

### `ksrc deps`
List resolved dependencies and source availability.

**Usage**
```
ksrc deps [flags]
```

**Flags**
- `--project <path>`
- `--scope <compile|runtime|test|all>`
- `--config <name>` (glob supported; comma‑separated)
- `--targets <list>` (comma‑separated)
- `--subproject <name>` (repeatable)
- `--offline`
- `--refresh`
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)

**Output (default)**
`group:artifact:version  [sources: yes|no]  [path: <gradle cache path>]`

---

### `ksrc fetch <coord>`
Ensure sources for a coordinate exist in Gradle caches.

**Usage**
```
ksrc fetch org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.1
```

**Flags**
- `--project <path>` (optional, if resolving via project)
- `--refresh`
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)

---

### `ksrc where <path|coord>`
Locate the Gradle cached source artifact or file.

**Usage**
```
ksrc where org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.1
ksrc where org/jetbrains/kotlinx/coroutines/flow/Flow.kt
```

**Flags**
- `--project <path>`
- `--module <glob>` (disambiguate)
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)

---

### `ksrc resolve`
Resolve the dependency graph without search. No project files are modified.

**Usage**
```
ksrc resolve [flags]
```

**Flags**
- `--project <path>`
- `--module <glob>`
- `--group <glob>`
- `--artifact <glob>`
- `--version <glob>`
- `--scope <compile|runtime|test|all>`
- `--config <name>` (glob supported; comma‑separated)
- `--targets <list>` (comma‑separated)
- `--subproject <name>` (repeatable)
- `--offline`
- `--refresh`
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)

---

### `ksrc doctor`
Diagnostics for project detection, Gradle cache accessibility, and source availability.

---

## File Identifier
`<file-id>` is a fully qualified path to a file inside a source JAR:
`group:artifact:version!/path/inside/jar.kt`

`ksrc search` emits `<file-id>` in every result line so clients can call `ksrc cat <file-id>` with no extra resolution steps.
