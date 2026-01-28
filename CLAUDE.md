# OpenKanban

Terminal-based kanban board with integrated AI agent spawning for ticket work.

## Stack

Go 1.21+, BubbleTea (TUI), creack/pty, vt10x (terminal emulation)

## Development

```bash
go build ./...    # Build
go test ./...     # Test
go run .          # Run
```

## Where to Look

| Task | Location |
|------|----------|
| Add CLI command | cmd/ |
| Modify UI/keybindings | internal/ui/ |
| Change agent behavior | internal/agent/ |
| Terminal/PTY handling | internal/terminal/ |
| Board/ticket logic | internal/board/ |
| Project management | internal/project/ |
| Configuration | internal/config/ |
| Git operations | internal/git/ |

## Architecture

```
cmd/           CLI entry (cobra)
internal/
  ui/          BubbleTea Model - central orchestrator
  agent/       Agent config, status detection, spawning prep
  terminal/    PTY management, vt10x rendering, scrollback
  board/       Ticket/column data structures
  project/     Multi-project registry, settings cascade
  config/      JSON config, validation, themes
  git/         Worktree operations
```

## Key Flows

**Ticket → Agent spawn:**
ui.spawnAgent() → terminal.New() → pty.Start() → agent process

**Settings cascade:**
ticket.Field → project.Settings.Field → config.Defaults.Field

## Agent Workflow

Scout finds → Librarian reads → You plan → Worker implements → Validator checks

## Guidance

Context-specific guidance lives in nested CLAUDE.md files:
- internal/CLAUDE.md - Go patterns, imports, testing
- internal/ui/CLAUDE.md - BubbleTea patterns
- internal/agent/CLAUDE.md - Agent integration
- internal/terminal/CLAUDE.md - PTY/terminal handling
