# ksrc

**One‑liner search and read for Gradle 3rd-party dependency sources for AI agents.**

Your AI agents take ~10 steps just to see a single function signature in a third-party library. `ksrc` turns that into two commands and ~4x less tokens.

## What it is

It's a CLI utility to enable efficient dependency source search for AI agents working with Gradle projects.

Ever saw an AI agent find a function's signature in TypeScript/Python? Simple `rg` over `node_modules` and a `sed` call is all it needs to discover APIs and signatures.

With Gradle ecosystems, agents have to take a 15-step journey to download, locate, unpack and ripgrep source jars.
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

### MCP (Model Context Protocol)

Use when your agent doesn't have `bash` tool. Configure your MCP client to spawn the stdio server:

```json
{
  "mcpServers": {
    "ksrc": {
      "command": "ksrc",
      "args": ["mcp"]
    }
  }
}
```

Default tools: `search`, `cat`, `deps`. Enable more via `--tools=<list>` (e.g., `--tools=search,cat,deps,resolve` or `--tools=all`).

You shouldn't need the skill if you use mcp, but if your agent has access to `bash` tool, prefer CLI+bash instead of the mcp.

### AGENTS.md prompt

> Avoid directly accessing `.gradle`; instead, proactively use `ksrc` bash tool to inspect source code of dependencies to learn API shapes or implementations. Start with `ksrc --help`.

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

This program is licensed under the Apache License, Version 2.0.

See `LICENSE.txt`.

Copyright 2026 Respawn LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
