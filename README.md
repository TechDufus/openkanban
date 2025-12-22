<h1 align="center">
  <br>
  <img src="https://github.com/user-attachments/assets/14cde506-2091-4745-9349-2604d8ec5b32" alt="OpenKanban" width="600">
  <br>
</h1>

<h4 align="center">A TUI kanban board for orchestrating AI coding agents.</h4>

<p align="center">
  <a href="https://github.com/TechDufus/openkanban/releases/latest">
    <img src="https://img.shields.io/github/v/release/TechDufus/openkanban?style=flat-square&color=blue" alt="Release">
  </a>
  <a href="https://github.com/TechDufus/openkanban/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/TechDufus/openkanban?style=flat-square&color=green" alt="License">
  </a>
  <a href="https://github.com/TechDufus/openkanban">
    <img src="https://img.shields.io/github/go-mod/go-version/TechDufus/openkanban?style=flat-square" alt="Go Version">
  </a>
  <a href="https://github.com/TechDufus/openkanban/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/TechDufus/openkanban/release.yaml?style=flat-square&label=build" alt="Build Status">
  </a>
</p>

<p align="center">
  <img src="./docs/assets/demo.gif" alt="OpenKanban Demo" width="800">
</p>

---

## Why?

AI coding agents are powerful, but managing multiple agents across projects gets messy fast. You end up with terminals everywhere, losing track of what's running where, and context-switching between tasks becomes a chore.

OpenKanban gives you a single view of all your work. Each ticket gets its own git worktree and embedded terminal. Spawn an agent, watch it work, jump between tasks. Everything stays organized.

## Features

- **Tickets as worktrees** - Each task gets an isolated git branch
- **Embedded terminals** - Agents run inside the TUI, not in random terminal tabs
- **Any agent** - OpenCode, Claude Code, Aider, or whatever CLI tool you prefer
- **Multi-project** - Manage tickets across all your repositories from one board

## Install

### Homebrew (macOS/Linux)

```bash
brew install TechDufus/tap/openkanban
```

To update:

```bash
brew upgrade openkanban
```

### Go

```bash
go install github.com/TechDufus/openkanban@latest
```

## Quick Start

```bash
cd ~/projects/my-app
openkanban new "My App"
openkanban
```

## Keybindings

| Key | Action |
|-----|--------|
| `n` | New ticket |
| `s` | Spawn agent |
| `enter` | Attach to agent |
| `h/l` | Move ticket between columns |
| `?` | Full help |

## Configuration

OpenKanban is highly configurable. Agents, keybindings, branch naming, cleanup behavior - all customizable in `~/.config/openkanban/config.json`.

See [Configuration Guide](./docs/CONFIGURATION.md) for details.

## License

[AGPL-3.0](LICENSE)
