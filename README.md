# OpenKanban

A TUI kanban board for orchestrating AI coding agents across multiple projects.

```
    BACKLOG (3)          IN PROGRESS (2)         DONE (1)
 +-----------------+  +-----------------+  +-----------------+
 | auth-system     |  | api-endpoints   |  | db-schema       |
 | [idle]          |  | [working]       |  | [done]          |
 |                 |  | opencode        |  | claude          |
 +-----------------+  +-----------------+  +-----------------+
 | user-dashboard  |  | payment-flow    |
 | [idle]          |  | [working]       |
 |                 |  | claude          |
 +-----------------+  +-----------------+
 | notifications   |
 | [idle]          |
 |                 |
 +-----------------+

 [n]ew  [enter]open  [h/l]move  [d]elete  [q]uit  [?]help
```

## What It Does

Each ticket on the board represents a task. When you start working on a ticket:

1. **Git worktree created** - Isolated branch for that task
2. **AI agent spawned** - Claude Code, OpenCode, or your preferred agent
3. **Embedded terminal** - Agent runs in an embedded PTY within the TUI
4. **Status tracked** - See which agents are working/idle/done

Move tickets across columns with keyboard (`h`/`l`, `space`) or mouse drag-and-drop.

## Installation

```bash
# From source
go install github.com/techdufus/openkanban@latest

# Or build locally
git clone https://github.com/techdufus/openkanban
cd openkanban
go build -o openkanban .
```

## Quick Start

```bash
# Register a git repository as a project
cd ~/projects/my-app
openkanban new "My App"

# Launch the board (shows all projects)
openkanban

# Or filter to a specific project
openkanban -p ~/projects/my-app
```

## Configuration

Config lives in `~/.config/openkanban/config.json`:

```json
{
  "defaults": {
    "default_agent": "opencode",
    "branch_prefix": "task/",
    "branch_template": "{prefix}{slug}",
    "slug_max_length": 40
  },
  "agents": {
    "opencode": {
      "command": "opencode",
      "args": []
    },
    "claude": {
      "command": "claude",
      "args": ["--dangerously-skip-permissions"]
    },
    "aider": {
      "command": "aider",
      "args": ["--yes"]
    }
  },
  "cleanup": {
    "delete_worktree": true,
    "delete_branch": false
  }
}
```

## Keybindings

| Key | Action |
|-----|--------|
| `j/k` | Move cursor up/down |
| `h/l` | Move between columns |
| `space` | Move ticket to next column |
| `-` | Move ticket to previous column |
| `enter` | Attach to running agent |
| `n` | Create new ticket |
| `e` | Edit ticket |
| `s` | Spawn agent for ticket |
| `S` | Stop agent |
| `d` | Delete ticket (with confirmation) |
| `p` | Cycle project filter |
| `?` | Show help |
| `q` | Quit |

**In agent view:**
| Key | Action |
|-----|--------|
| `ctrl+g` | Return to board |
| All other keys | Passed to agent |

## How It Works

### Project Registration

OpenKanban manages tickets across multiple git repositories. Each repo is registered as a "project":

```bash
openkanban new "Project Name"    # Register current directory
openkanban list                   # Show all registered projects
```

Projects are stored in `~/.config/openkanban/projects.json`. Tickets for each project are stored in `{repo}/.openkanban/tickets.json`.

### Ticket Lifecycle

1. **Create ticket** (`n`)
   - Enter title and optional description
   - Select project (if multiple registered)
   - Branch name auto-generated from title

2. **Start work** (move to "In Progress")
   - Git worktree created automatically
   - Branch created from default branch

3. **Spawn agent** (`s`)
   - Agent launches in embedded terminal
   - Worktree directory is working directory
   - Agent receives ticket context

4. **Work in agent** (`enter`)
   - Full terminal emulation in TUI
   - `ctrl+g` returns to board

5. **Complete ticket** (move to "Done")
   - Agent can keep running
   - Or stop with `S`

6. **Delete ticket** (`d`)
   - Stops agent if running
   - Removes worktree (configurable)
   - Optionally deletes branch

## Architecture

```
openkanban/
├── cmd/root.go              # CLI commands
├── main.go                  # Entry point
├── internal/
│   ├── app/app.go           # Application orchestration
│   ├── ui/
│   │   ├── model.go         # Bubbletea model + Update
│   │   └── view.go          # Rendering
│   ├── board/board.go       # Ticket model, columns
│   ├── project/
│   │   ├── project.go       # Project model
│   │   ├── store.go         # Project registry
│   │   ├── tickets.go       # Ticket storage
│   │   └── filter.go        # Saved filters
│   ├── terminal/pane.go     # PTY terminal emulation
│   ├── agent/
│   │   ├── agent.go         # Agent spawning
│   │   ├── context.go       # Ticket context injection
│   │   └── status.go        # Status detection
│   ├── git/worktree.go      # Git worktree operations
│   └── config/config.go     # Configuration
└── docs/
```

## Supported Agents

| Agent | Status | Notes |
|-------|--------|-------|
| OpenCode | Full | Native support with session resume |
| Claude Code | Full | Native support with `--continue` |
| Aider | Full | `--yes` flag recommended |

## Tech Stack

- **Go 1.23+**
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** - TUI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Styling (Catppuccin Mocha)
- **[vt10x](https://github.com/hinshun/vt10x)** - Terminal emulation
- **[creack/pty](https://github.com/creack/pty)** - PTY handling

## Prior Art

- [vibe-kanban](https://github.com/BloopAI/vibe-kanban) - Web-based kanban for AI agents (inspiration)

## License

MIT
