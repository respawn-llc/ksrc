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
- `--rg-args <args>`: Extra args passed to `rg` (comma‑separated)
- `-- <rg-args>`: Pass through raw `rg` args without CSV encoding
- `--show-extracted-path`: Include temp extracted paths in output (off by default)

**Output (default)**
`<file-id> <line>:<col>:<line-text>` (use `--show-extracted-path` to include temp paths)

`<line-text>` is the full source line with the trailing newline stripped. It may contain literal `:` characters. When `--context` is set, context lines use the same shape with `<col>` set to `0`.

Parse contract:
- Split once on the first space to separate `<file-id>` from the location/text payload.
- Parse the first two `:`-delimited decimal fields in the payload as `<line>` and `<col>`.
- Treat the remainder verbatim as `<line-text>`.

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
- Relative source path: `com/example/http/HttpClient.java`
- Fully qualified path: `group:artifact:version!/com/example/http/HttpClient.java`

**Flags**
- `--project <path>`
- `--module <glob>` (disambiguate)
- `--buildsrc`: Include buildSrc dependencies (default: `true`; set `--buildsrc=false` to disable)
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)
- `--lines <start,end>`: Output a line range (1‑based, inclusive; sed‑style)

---

### `ksrc open <file-id|path>`
Open a file in `$PAGER` (defaults to `less -R`).

**Usage**
```
ksrc open <file-id|path> [flags]
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
ksrc where com/example/http/HttpClient.java --group com.example --artifact http
```

**Path Forms**
- Coordinate: `group:artifact[:version]`
- Relative source path: `com/example/http/HttpClient.java` (requires `--module`, or `--group` plus `--artifact`)
- File ID: `group:artifact:version!/com/example/http/HttpClient.java`

**Output**
- Coordinate lookup: `<coord>|<jar-path>`
- File-id lookup: `<file-id>|<jar-path>`
- Path lookup: `<file-id>|<jar-path>`

For path lookups, the emitted `<file-id>` always uses the resolved `group:artifact:version`, even if the input only provided `--group`/`--artifact` or `--module` without a version. This makes the result directly reusable with `ksrc cat <file-id>` and `ksrc open <file-id>`.

**Flags**
- `--project <path>`
- `--module <glob>` (disambiguate)
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
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)

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
- `--buildscript`: Include buildscript classpath deps (default: `true`; set `--buildscript=false` to disable)
- `--include-builds`: Include composite builds (includeBuild) (default: `true`; set `--include-builds=false` to disable)

---

### `ksrc doctor`
Diagnostics for project detection, Gradle cache accessibility, and source availability.

---

### `ksrc mcp`
Run an MCP server over stdio for tool integrations.

**Usage**
```
ksrc mcp [flags]
```

**Flags**
- `--tools <list>`: Comma-separated tool list (default: `search,cat,deps`; use `all` for all tools)

**Default tools**
- `search`
- `cat`
- `deps`

**Optional tools (enable via --tools)**
- `fetch`
- `resolve`
- `where`

**Notes**
- Transport is stdio only; clients should spawn `ksrc mcp` via `mcp.json`.
- Outputs are plain text matching CLI formats.

---

## File Identifier
`<file-id>` is a fully qualified path to a file inside a source JAR:
`group:artifact:version!/path/inside/jar.ext`

Parse `<file-id>` by splitting once on `!/`. The left side is `group:artifact:version`; the right side is the slash-normalized path inside the source jar.

`ksrc search` and `ksrc where <path>` emit `<file-id>` in reusable form so clients can call `ksrc cat <file-id>` with no extra resolution steps.
