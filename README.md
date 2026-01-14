# ksrc

**One‑liner search and read for Kotlin 3rd-party dependency sources for AI agents.**

Your AI agents take ~10 steps just to see a single Kotlin function's signature in a third-party library. `ksrc` turns that into two commands and ~4x less tokens.

## What it is

It's a CLI utility to enable efficient source code search for AI agents working with Kotlin.

Ever saw an AI agent find a function's signature in TypeScript/Python? Simple `rg` over `node_modules` and a `sed` call is all it needs to discover APIs and
signatures.

With Kotlin/Gradle, agents have to take a 15-step journey to download, locate, unpack and ripgrep source jars.
`ksrc` turns 16k tokens wasted on that into 2 CLI commands.

## Install

Homebrew (macOS/Linux):

```
brew tap respawn-app/tap
brew install ksrc
```

Standalone binaries via GitHub Releases:

Install script (macOS/Linux):

```
curl -fsSL https://raw.githubusercontent.com/respawn-app/ksrc/main/scripts/install.sh | sh
```

Manual install: download the appropriate archive for your OS/arch and place `ksrc` on your `PATH`.

Next up, install the claude code plugin/skill, to let your agents know they can use `ksrc` and how to use it.

### Claude Code plugin

Add the Respawn marketplace, then install the plugin:

```
/plugin marketplace add respawn-app/claude-plugin-marketplace
/plugin install ksrc@respawn-tools
```

### Codex skill

Install from the public GitHub path:

```
$skill-installer install https://github.com/respawn-app/ksrc/tree/main/skills/ksrc
```

### AGENTS.md prompt

> Use `ksrc` bash command to discover Kotlin/gradle library dependency sources. Start with `ksrc --help`.

## Usage

Search defaults to all dependencies when no selector is provided:

```
ksrc search "LocalDate"
ksrc search "LocalDate" --module org.jetbrains.kotlinx:kotlinx-datetime
ksrc cat <file-id> --lines 1,120
```

## License

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License.

See `LICENSE.txt`.
