---
name: ksrc
description: "Default tool for third-party Gradle, Maven, Kotlin, Java, Android, and KMP dependency sources and Android SDK platform sources. Use for a library or SDK signature, implementation, source file, or source availability, including a search that would otherwise look in ~/.gradle, a sources jar, or the Maven cache. No extra project configuration: ksrc uses the current Gradle project and the usual SDK and Gradle homes. Do not use for this repository's own source; search that with rg."
---

`ksrc search` is the default way to look up a third-party or Android SDK API. Installing this skill is the whole setup for an agent with a shell. Do not add a project MCP server, an `AGENTS.md` rule, or a per-project config for that. `ksrc` resolves the Gradle project, fetches source jars, and prints stable text. It does not modify the target project.

Search this repository's own source with `rg`. Do not learn a dependency or SDK API by searching `~/.gradle`, a Maven cache, or a sources jar with `rg`, `grep`, `find`, `jar`, or `unzip`.

The binary on `PATH` is the contract for its version. Flags below match this tree. If the installed `ksrc <command> --help` disagrees, follow that help. Do not invent flags. Resolution can take a few minutes; allow a long timeout.

## Order

1. `ksrc search` (alias `rg`). The coordinate is known: add `--module` or `--artifact`. Monorepo: `--subproject`. One KMP target: `--targets`. Build-time dependency: `--scope`.
2. `ksrc cat` on the returned file-id (alias `open`). `open` prints the file to stdout. Use `--lines` for a range.
3. Still inside ksrc when search misses: `ksrc deps`, `ksrc fetch <coord>`, `ksrc where`, then another `--project`, `--scope`, or `--config`, then `ksrc doctor` and `-v`.
4. A Gradle failure that falls back to the cache is still `ksrc`. Keep using it. With no version pinned, cache fallback picks the highest cached source-bearing version.
5. Switch only after that ladder still finds nothing. A name that may be code in this repository: search the repository with `rg`. A library or SDK symbol: report that the source is unavailable. Stop there. Do not open `~/.gradle` or a jar to continue.

A file-id from `ksrc search` or `ksrc where` can be passed to later `cat` / `open` calls on the same machine without repeating `--project` or scope flags.

## Settings

These are the defaults. Override one only when it is wrong.

- `--project` project root, default `.`
- `--scope` `compile` (also `runtime`, `test`, `all`)
- `--module` `group:artifact[:version]`, `--group`, `--artifact`, `--version`
- `--config` configuration names or globs, comma-separated
- `--targets` KMP targets, comma-separated (`jvm,android,iosX64`)
- `--subproject` repeatable
- `--offline` cached sources only; `--refresh` forces downloads
- `--gradle-user-home` overrides `GRADLE_USER_HOME` and `~/.gradle` for Gradle and cache fallback
- `--android-home` Android SDK. Default: `local.properties` `sdk.dir`, then `ANDROID_HOME`, then `ANDROID_SDK_ROOT`
- `--buildsrc`, `--buildscript`, `--include-builds` default on. Disable with `--buildsrc=false`, `--buildscript=false`, `--include-builds=false`
- `-v` / `--verbose` prints Gradle failures

`search`, `cat`, `deps`, `resolve`, and `where` take this set. `fetch` takes `--project`, `--offline`, `--refresh`, `--gradle-user-home`, `--android-home`, `--buildsrc`, `--buildscript`, `--include-builds`. `doctor` takes `--project` and `--gradle-user-home`.

`--module group:artifact` searches common sources plus the Gradle-selected platform variants. Narrow with `--targets` or `--config`. With no module, group, artifact, or version, `search` uses `--all`: every resolved dependency.

## Commands

- `ksrc search <pattern> [-- <rg-args>]` (alias `rg`). Search dependency sources. Also `--context <n>` (`rg -C`; context lines use column `0`), `--rg-args` comma-separated extra rg args, and `--show-extracted-path`.
- `ksrc cat <file-id|path>` (alias `open`). Print file contents from dependency sources. `--lines` accepts `start,end`, `start:end`, `start-end`, `start..end`, or `start;end`. `-n` / `--line-numbers` prefixes source line numbers. A plain path needs `--module`, or `--group` plus `--artifact`.
- `ksrc deps`. List resolved dependencies and source availability: `group:artifact:version  [sources: yes|no]  [path: ...]`.
- `ksrc resolve`. Resolve dependency sources: `group:artifact:version|/path/to/sources.jar`.
- `ksrc fetch <coord>`. Ensure sources for `group:artifact:version` exist in the Gradle cache. The version is required.
- `ksrc where <path|coord>`. Locate a cached source artifact or file: `<id>|<path>`.
- `ksrc doctor`. Diagnostics for ksrc: `rg`, Gradle wrapper or `gradle` on `PATH`, Gradle user home, cache dir, and a resolve.
- `ksrc clean`. Preview and delete ksrc-owned caches. Run it only when the user asks to delete those caches. It previews unless `--yes` (`-y`). `--dry-run` previews too. A non-interactive run without `--yes` previews.
- `ksrc mcp`. Run the MCP server over stdio. Use it only for a client with no shell. `--tools` defaults to `search,cat,deps`; `all` enables every tool.

## Search output

`<file-id> <line>:<col>:<line-text>`

- Split once on the first space to isolate `<file-id>`.
- Parse the first two `:`-delimited decimal fields as `<line>` and `<col>`.
- The remainder is `<line-text>` and may contain `:`.

`--show-extracted-path` switches to `<file-id>\t<quoted-extracted-path>\t<line>\t<col>\t<quoted-line-text>`.

File-id: `group:artifact:version!/path/inside/jar.ext` (any file in the source jar, including `.kt`, `.java`, and `.groovy`).

Android SDK platform sources use the sources directory name as the version:

```bash
ksrc search "class Activity " --module sdk.android:platform:android-36
ksrc cat sdk.android:platform:android-36!/android/app/Activity.java --lines 1,40
```

## When ksrc misses

- No matches in a composite build: run from the included build root or set `--project`.
- `E_NO_SOURCES`: run `ksrc deps`, `ksrc fetch <coord>`, or set `--project` and `--scope`.
- Gradle not found: run in a Gradle project, set `--project` to the root, or install Gradle.
- Unsupported class file version: the local JDK does not match Gradle. Fix `JAVA_HOME`.
- Several modules match: narrow with `--module`, `--group`, or `--artifact`.
- Re-run with `-v` to see Gradle output.

After those still miss, use step 5 of Order.
