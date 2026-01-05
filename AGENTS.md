# OpenKanban

**Generated:** 2026-01-03 | **Commit:** 6418109 | **Branch:** main

TUI kanban board for orchestrating AI coding agents. Go 1.25+, Bubbletea, Lipgloss.

## Structure

```
openkanban/
├── cmd/               # CLI (Cobra) - root, new, list, delete
├── internal/
│   ├── ui/            # Bubbletea model/view (HEART OF APP)
│   ├── project/       # Project registry, ticket stores
│   ├── agent/         # Agent config, status detection
│   ├── terminal/      # PTY panes (creack/pty + vt10x)
│   ├── git/           # Worktree management
│   ├── config/        # Config loading/validation
│   ├── board/         # Ticket/Column types
│   ├── app/           # App orchestration
│   └── testutil/      # Test helpers (TestEnv, assertions)
├── docs/              # Design docs
└── main.go            # Entry point
```

## Where to Look

| Task | Location | Notes |
|------|----------|-------|
| Add keybinding | `internal/ui/model.go` → `handleNormalMode()` | Follow existing switch pattern |
| New UI mode | `internal/ui/model.go` → Mode const + handler | Add to Mode enum, create handler |
| Ticket fields | `internal/board/board.go` → Ticket struct | Update JSON tags, add to form |
| Agent config | `internal/config/config.go` → AgentConfig | Add defaults in `defaultAgents()` |
| Git operations | `internal/git/worktree.go` | Uses exec.Command, not go-git |
| PTY rendering | `internal/terminal/pane.go` → `View()` | vt10x cell-by-cell rendering |
| Status detection | `internal/agent/status.go` | Polls OpenCode API/files |

## Code Map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Model` | struct | ui/model.go:61 | Main app state, Bubbletea model |
| `Update()` | method | ui/model.go:250 | Event handler (NEVER BLOCK) |
| `View()` | method | ui/view.go:13 | All rendering |
| `Ticket` | struct | board/board.go:69 | Task unit with worktree/agent |
| `GlobalTicketStore` | struct | project/tickets.go | Multi-project ticket aggregation |
| `Pane` | struct | terminal/pane.go:21 | PTY-based embedded terminal |
| `StatusDetector` | struct | agent/status.go | Agent status polling |
| `WorktreeManager` | struct | git/worktree.go:13 | Git worktree CRUD |
| `Config` | struct | config/config.go:106 | Global config with agents map |
| `TestEnv` | struct | testutil/testutil.go:14 | Isolated test environment helper |

## Conventions

- **Imports**: stdlib, blank, external, blank, internal
- **Errors**: Return last, wrap with `fmt.Errorf("context: %w", err)`
- **Config**: All behavior configurable via `~/.config/openkanban/config.json`
- **Branch naming**: Slugify title with configurable prefix (default `task/`)

## Anti-Patterns (THIS PROJECT)

- **NEVER block in `Update()`** - All I/O via `tea.Cmd`, async
- **NEVER use tmux** - Embedded PTY only (creack/pty + vt10x)
- **NEVER assume config exists** - Use `DefaultConfig()` fallback
- **NEVER suppress type errors** - No `as any` equivalents
- **NEVER add AI attribution to commits**

## Unique Styles

- PTY integration: `terminal.Pane.Start()` → returns `tea.Cmd` for read loop
- Agent status: Poll OpenCode API on port 4096+N per agent
- Worktrees: Sibling dir pattern `{repo}-worktrees/{branch}`
- Modal system: `Mode` enum + dedicated handler per mode

## Commands

```bash
make build                          # Build binary
make test                           # Unit tests (default)
make test-unit                      # Unit tests only
make test-integration               # Integration tests only
make test-all                       # All tests
make coverage                       # Coverage report
make lint                           # Run linters
go test -run TestName ./...         # Single test
goreleaser release --snapshot       # Local release build
```

## Testing

### Test Isolation

Config directory is overridable via environment variable:

```bash
OPENKANBAN_CONFIG_DIR=/tmp/test openkanban    # Use custom config dir
```

Priority: `OPENKANBAN_CONFIG_DIR` > `XDG_CONFIG_HOME/openkanban` > `~/.config/openkanban`

### Test Types

| Type | Location | Build Tag | When to Use |
|------|----------|-----------|-------------|
| Unit | `*_test.go` | none | Pure logic, no I/O |
| Integration | `*_test.go` | `integration` | State persistence, multi-component |

### Writing Integration Tests

Use `testutil.TestEnv` for isolated test environments:

```go
//go:build integration

package mypackage_test

import (
    "testing"
    "github.com/techdufus/openkanban/internal/testutil"
)

func TestSomething_Integration(t *testing.T) {
    env := testutil.NewTestEnv(t)  // Creates temp dirs, sets OPENKANBAN_CONFIG_DIR
    env.InitGitRepo()               // Optional: init git in temp repo

    // Setup
    p := env.CreateProject("test-project")

    // Action
    // ... your test logic ...

    // Verify
    env.AssertProjectCount(1)
    env.AssertTicketCount(0)
}
```

### TestEnv API

| Method | Purpose |
|--------|---------|
| `NewTestEnv(t)` | Create isolated env, auto-cleanup |
| `InitGitRepo()` | Initialize git repo in temp dir |
| `CreateProject(name)` | Create and register project |
| `LoadRegistry()` | Load project registry |
| `LoadTickets()` | Load ticket store |
| `AssertProjectCount(n)` | Verify project count |
| `AssertTicketCount(n)` | Verify ticket count |
| `AssertTicketExists(title)` | Find ticket by title |
| `AssertTicketStatus(id, status)` | Verify ticket status |

### Test Requirements for Changes

**When modifying these areas, ADD or UPDATE tests:**

| Change Area | Required Tests |
|-------------|----------------|
| Config loading/validation | Unit test in `config_test.go` |
| Project CRUD | Integration test using `TestEnv` |
| Ticket CRUD | Integration test using `TestEnv` |
| Status transitions | Integration test verifying persistence |
| New CLI commands | Smoke test + integration test |

**All PRs must pass:**
- `make test-unit`
- `make test-integration`
- `make lint`

## Data Locations

| Data | Path | Format |
|------|------|--------|
| Global config | `~/.config/openkanban/config.json` | JSON |
| Project registry | `~/.config/openkanban/projects.json` | JSON |
| Ticket store | `{repo}/.openkanban/tickets.json` | JSON per project |

## Notes

- **Elm architecture**: Model → Update → View cycle, immutable-ish state
- **Status poll interval**: Configurable via `opencode.poll_interval` (default 1s)
- **Worktree cleanup**: Configurable delete behavior on ticket deletion
- **Agent priority**: opencode > claude > gemini > codex > aider (first available becomes default)
