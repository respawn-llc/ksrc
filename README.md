# ksrc

**One‑liner search and read for Kotlin 3rd-party dependency sources for AI agents.**

Your AI agents take ~10 steps just to see a single Kotlin function's signature in a third-party library. `ksrc` turns that into two commands and ~4x less tokens.

## What it is

It's a CLI utility to enable efficient source code search for AI agents working with Kotlin.

Ever saw an AI agent find a function's signature in TypeScript/Python? Simple `rg` over `node_modules` and a `sed` call is all it needs to discover APIs and signatures.

With Kotlin/Gradle, agents have to take a 15-step journey to download, locate, unpack and ripgrep source jars.
`ksrc` turns 16k tokens wasted on that into 2 CLI commands.

## 1. Install the tool

Start by installing the **command itself**.

### Homebrew (macOS/Linux) - recommended, auto-updated:

```
brew tap respawn-app/tap
brew install ksrc
```

### Standalone binaries via GitHub Releases:

Install script (macOS/Linux):

```
curl -fsSL https://raw.githubusercontent.com/respawn-app/ksrc/main/scripts/install.sh | sh
```

### Manual install: 

Download the appropriate archive for your OS/arch from [releases](https://github.com/respawn-app/ksrc/releases) and place `ksrc` on your `PATH`.

## 2. Teach agents how to use it

Next up, install the claude code plugin/skill, to let your agents know they _can_ use `ksrc` and how to use it.

### Claude Code

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

Give this tool larger timeouts - it can take a minute to download sources (if needed) and resolve gradle projects. 

1. Start by using ksrc search to get the file identifier and lines you need. Example:

```bash
$ ksrc search "updateState<"
pro.respawn.flowmvi:core:3.3.0-alpha03!/commonMain/pro/respawn/flowmvi/api/StateReceiver.kt 19:8: updateState<State.Subtype, _> { }
```
The tool returns found artifacts, versions, source sets, paths, and lines in a single common format that's chainable with other commands, rg-style.

If you want faster execution & less noise, specify:
- `--artifact` to limit search to one artifact, (or `--module` to also limit by version)
- `--subproject` to help discovery for monorepos/large modular apps
- `--targets` to limit to specific KMP targets. 

2. When you have found the desired artifact, read the file contents:

```bash
$ ksrc cat 'pro.respawn.flowmvi:core:3.3.0-alpha03!/commonMain/pro/respawn/flowmvi/api/StateReceiver.kt' --lines 10,25
```

## License

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License.

See `LICENSE.txt`.
