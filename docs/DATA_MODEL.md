# Data Model & Persistence

This document defines the data structures, persistence strategies, and state management for OpenKanban.

## Core Data Structures

### Ticket

The fundamental unit of work. Each ticket represents a task with an associated git worktree and agent session.

```go
type TicketID string // UUID v4

type TicketStatus string

const (
    StatusBacklog    TicketStatus = "backlog"
    StatusInProgress TicketStatus = "in_progress"
    StatusDone       TicketStatus = "done"
    StatusArchived   TicketStatus = "archived"
)

type AgentStatus string

const (
    AgentIdle      AgentStatus = "idle"      // Session exists, no activity
    AgentWorking   AgentStatus = "working"   // Active output detected
    AgentWaiting   AgentStatus = "waiting"   // Waiting for user input
    AgentCompleted AgentStatus = "completed" // Agent reported done
    AgentError     AgentStatus = "error"     // Agent crashed/errored
    AgentNone      AgentStatus = "none"      // No session spawned
)

type Ticket struct {
    ID          TicketID     `json:"id"`
    ProjectID   string       `json:"project_id"`
    Title       string       `json:"title"`
    Description string       `json:"description,omitempty"`
    Status      TicketStatus `json:"status"`
    
    // Git integration
    WorktreePath string `json:"worktree_path,omitempty"`
    BranchName   string `json:"branch_name,omitempty"`
    BaseBranch   string `json:"base_branch,omitempty"` // e.g., "main"
    
    // Agent integration (embedded PTY terminals, not tmux)
    AgentType      string      `json:"agent_type,omitempty"` // "claude", "opencode", "aider"
    AgentStatus    AgentStatus `json:"agent_status"`
    AgentSpawnedAt *time.Time  `json:"agent_spawned_at,omitempty"`
    AgentPort      int         `json:"agent_port,omitempty"` // Per-ticket opencode port
    
    // Metadata
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    StartedAt   *time.Time `json:"started_at,omitempty"`   // When moved to in_progress
    CompletedAt *time.Time `json:"completed_at,omitempty"` // When moved to done
    
    // User-defined
    Labels   []string          `json:"labels,omitempty"`
    Priority int               `json:"priority,omitempty"` // 1=highest, 5=lowest
    Meta     map[string]string `json:"meta,omitempty"`     // Custom key-value pairs
}
```

### Project

A Project represents a registered git repository. Each git repo is one Project.

```go
type Project struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    RepoPath    string          `json:"repo_path"`    // Absolute path to git repo root
    WorktreeDir string          `json:"worktree_dir"` // Where worktrees go (default: {repo}-worktrees)
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
    Settings    ProjectSettings `json:"settings"`
}

type ProjectSettings struct {
    DefaultAgent     string `json:"default_agent,omitempty"`
    AutoSpawnAgent   bool   `json:"auto_spawn_agent"`
    AutoCreateBranch bool   `json:"auto_create_branch"`
    BranchPrefix     string `json:"branch_prefix,omitempty"`
    BranchNaming     string `json:"branch_naming,omitempty"`   // "template" | "ai" | "prompt"
    BranchTemplate   string `json:"branch_template,omitempty"` // e.g., "{prefix}{slug}"
    SlugMaxLength    int    `json:"slug_max_length,omitempty"` // default: 40
}
```

### Column

Columns define the board layout and map to ticket statuses.

```go
type Column struct {
    ID     string       `json:"id"`
    Name   string       `json:"name"`
    Status TicketStatus `json:"status"` // Maps to ticket status
    Color  string       `json:"color"`  // Hex color for column header
    Limit  int          `json:"limit"`  // WIP limit (0 = unlimited)
}
```

### Application State

Runtime state for the TUI application.

```go
type AppState struct {
    // Current board
    Board *Board
    
    // UI state
    ActiveColumn int        // Currently selected column index
    ActiveTicket int        // Currently selected ticket index within column
    Mode         UIMode     // Normal, Insert, Command, Help
    
    // Filtering/search
    FilterLabels []string
    SearchQuery  string
    
    // Cached views
    ColumnTickets [][]TicketID // Tickets per column (filtered/sorted)
    
    // Agent monitoring
    AgentStatuses map[TicketID]AgentStatus // Real-time status cache
    LastPoll      time.Time
}

type UIMode string

const (
    ModeNormal  UIMode = "normal"
    ModeInsert  UIMode = "insert"  // Editing ticket
    ModeCommand UIMode = "command" // : command mode
    ModeHelp    UIMode = "help"    // Help overlay
    ModeConfirm UIMode = "confirm" // Confirmation dialog
)
```

## Persistence Strategy

### File-Based Storage

OpenKanban uses a multi-file JSON storage approach:

```
~/.config/openkanban/
├── config.json           # Global configuration
└── projects.json         # Project registry (all registered projects)

{repo}/.openkanban/
└── tickets.json          # Tickets for this specific project
```

### Project Registry Format

Stored in `~/.config/openkanban/projects.json`:

```json
{
  "projects": {
    "proj-uuid-1": {
      "id": "proj-uuid-1",
      "name": "My Project",
      "repo_path": "/home/user/projects/myproject",
      "worktree_dir": "/home/user/projects/myproject-worktrees",
      "created_at": "2025-01-15T10:00:00Z",
      "updated_at": "2025-01-16T14:30:00Z",
      "settings": {
        "default_agent": "opencode",
        "auto_spawn_agent": true,
        "auto_create_branch": true,
        "branch_prefix": "task/",
        "branch_naming": "template",
        "branch_template": "{prefix}{slug}",
        "slug_max_length": 40
      }
    }
  }
}
```

### Per-Project Tickets Format

Stored in `{repo}/.openkanban/tickets.json`:

```json
{
  "tickets": {
    "ticket-uuid-1": {
      "id": "ticket-uuid-1",
      "project_id": "proj-uuid-1",
      "title": "Implement user authentication",
      "description": "Add JWT-based auth to the API",
      "status": "in_progress",
      "worktree_path": "/home/user/projects/myproject-worktrees/task-implement-user-authentication",
      "branch_name": "task/implement-user-authentication",
      "base_branch": "main",
      "agent_type": "opencode",
      "agent_status": "working",
      "agent_port": 4097,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-16T14:30:00Z",
      "started_at": "2025-01-16T09:00:00Z",
      "labels": ["backend", "security"],
      "priority": 1
    }
  }
}
```

### SQLite Storage (Optional, for large boards)

For boards with >1000 tickets or complex querying needs.

```sql
-- Schema
CREATE TABLE boards (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    settings JSON NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE columns (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    color TEXT,
    wip_limit INTEGER DEFAULT 0,
    position INTEGER NOT NULL,
    UNIQUE(board_id, position)
);

CREATE TABLE tickets (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id),
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'backlog',
    worktree_path TEXT,
    branch_name TEXT,
    base_branch TEXT,
    agent_type TEXT,
    agent_status TEXT DEFAULT 'none',
    tmux_session TEXT,
    priority INTEGER DEFAULT 3,
    labels JSON DEFAULT '[]',
    meta JSON DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX idx_tickets_board_status ON tickets(board_id, status);
CREATE INDEX idx_tickets_agent_status ON tickets(agent_status);
```

### Storage Interface

Abstract storage to support both backends:

```go
type Storage interface {
    // Board operations
    CreateBoard(board *Board) error
    GetBoard(id string) (*Board, error)
    UpdateBoard(board *Board) error
    DeleteBoard(id string) error
    ListBoards() ([]*Board, error)
    
    // Ticket operations
    CreateTicket(boardID string, ticket *Ticket) error
    GetTicket(boardID string, ticketID TicketID) (*Ticket, error)
    UpdateTicket(boardID string, ticket *Ticket) error
    DeleteTicket(boardID string, ticketID TicketID) error
    ListTickets(boardID string, filter TicketFilter) ([]*Ticket, error)
    
    // Batch operations
    MoveTicket(boardID string, ticketID TicketID, newStatus TicketStatus) error
    ReorderTickets(boardID string, status TicketStatus, ticketIDs []TicketID) error
}

type TicketFilter struct {
    Status   []TicketStatus
    Labels   []string
    Priority *int
    Search   string
}
```

## State Transitions

### Ticket Lifecycle

```
                    ┌─────────────┐
                    │   Created   │
                    └──────┬──────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────┐
│                      BACKLOG                              │
│  - No worktree                                           │
│  - No agent session                                      │
│  - agent_status = "none"                                 │
└──────────────────────┬───────────────────────────────────┘
                       │ Move to In Progress
                       │ (triggers: create worktree, spawn agent)
                       ▼
┌──────────────────────────────────────────────────────────┐
│                   IN PROGRESS                             │
│  - Worktree created at {worktree_dir}/{branch_name}      │
│  - Branch created: {branch_prefix}{slug}                 │
│  - Embedded PTY terminal with agent process              │
│  - agent_status cycles: idle → working → waiting → ...   │
└──────────────────────┬───────────────────────────────────┘
                       │ Move to Done
                       │ (triggers: optional cleanup prompt)
                       ▼
┌──────────────────────────────────────────────────────────┐
│                       DONE                                │
│  - Worktree can be kept or removed                       │
│  - Agent session terminated                              │
│  - agent_status = "completed" or "none"                  │
│  - Branch ready for PR                                   │
└──────────────────────┬───────────────────────────────────┘
                       │ Archive (optional)
                       ▼
┌──────────────────────────────────────────────────────────┐
│                     ARCHIVED                              │
│  - Hidden from default view                              │
│  - Worktree removed                                      │
│  - Historical record preserved                           │
└──────────────────────────────────────────────────────────┘
```

### Agent Status Transitions

```
     spawn agent
         │
         ▼
      ┌──────┐
      │ idle │◄─────────────────────┐
      └──┬───┘                      │
         │ activity detected        │ no activity (30s)
         ▼                          │
    ┌─────────┐                     │
    │ working │─────────────────────┘
    └────┬────┘
         │ waiting for input
         ▼
    ┌─────────┐
    │ waiting │ (prompt detected)
    └────┬────┘
         │ user responds OR
         │ agent continues
         ▼
    ┌─────────────┐
    │ working/idle│
    └─────────────┘

    Error states:
    - Process exits unexpectedly → "error"
    - Status file says "done" → "completed"
    - PTY process killed → "none"
```

## Global Configuration

```go
type Config struct {
    Defaults BoardSettings          `json:"defaults"`
    Agents   map[string]AgentConfig `json:"agents"`
    UI       UIConfig               `json:"ui"`
    Cleanup  CleanupSettings        `json:"cleanup"`
    Behavior BehaviorSettings       `json:"behavior"`
    Opencode OpencodeSettings       `json:"opencode"`
    Keys     map[string]string      `json:"keys,omitempty"`
}

type BoardSettings struct {
    DefaultAgent     string `json:"default_agent"`
    WorktreeBase     string `json:"worktree_base"`
    AutoSpawnAgent   bool   `json:"auto_spawn_agent"`
    AutoCreateBranch bool   `json:"auto_create_branch"`
    BranchPrefix     string `json:"branch_prefix"`
    BranchNaming     string `json:"branch_naming"`   // "template" | "ai" | "prompt"
    BranchTemplate   string `json:"branch_template"` // e.g., "{prefix}{slug}"
    SlugMaxLength    int    `json:"slug_max_length"` // default: 40
    InitPrompt       string `json:"init_prompt"`
}

type AgentConfig struct {
    Command    string            `json:"command"`
    Args       []string          `json:"args"`
    Env        map[string]string `json:"env"`
    StatusFile string            `json:"status_file"`
    InitPrompt string            `json:"init_prompt"`
}

type UIConfig struct {
    Theme           string `json:"theme"`
    ShowAgentStatus bool   `json:"show_agent_status"`
    RefreshInterval int    `json:"refresh_interval"`
    ColumnWidth     int    `json:"column_width"`
    TicketHeight    int    `json:"ticket_height"`
    SidebarVisible  bool   `json:"sidebar_visible"`
}

type CleanupSettings struct {
    DeleteWorktree       bool `json:"delete_worktree"`
    DeleteBranch         bool `json:"delete_branch"`
    ForceWorktreeRemoval bool `json:"force_worktree_removal"`
}

type BehaviorSettings struct {
    ConfirmQuitWithAgents bool `json:"confirm_quit_with_agents"`
}

type OpencodeSettings struct {
    ServerEnabled bool `json:"server_enabled"`
    ServerPort    int  `json:"server_port"`
    PollInterval  int  `json:"poll_interval"`
}
```

### Default Configuration File

```json
{
  "defaults": {
    "default_agent": "opencode",
    "worktree_base": "",
    "auto_spawn_agent": true,
    "auto_create_branch": true,
    "branch_prefix": "task/",
    "branch_naming": "template",
    "branch_template": "{prefix}{slug}",
    "slug_max_length": 40
  },
  "agents": {
    "claude": {
      "command": "claude",
      "args": ["--dangerously-skip-permissions"],
      "env": {},
      "status_file": ".claude/status.json"
    },
    "opencode": {
      "command": "opencode",
      "args": [],
      "env": {},
      "status_file": ".opencode/status.json"
    },
    "aider": {
      "command": "aider",
      "args": ["--yes"],
      "env": {},
      "status_file": ""
    }
  },
  "ui": {
    "theme": "catppuccin-mocha",
    "show_agent_status": true,
    "refresh_interval": 5,
    "column_width": 40,
    "ticket_height": 4,
    "sidebar_visible": true
  },
  "cleanup": {
    "delete_worktree": true,
    "delete_branch": false,
    "force_worktree_removal": false
  },
  "behavior": {
    "confirm_quit_with_agents": true
  },
  "opencode": {
    "server_enabled": true,
    "server_port": 4096,
    "poll_interval": 1
  }
}
```

## File Paths

| Purpose | Path | Notes |
|---------|------|-------|
| Global config | `~/.config/openkanban/config.json` | User preferences |
| Project registry | `~/.config/openkanban/projects.json` | All registered projects |
| Project tickets | `{repo}/.openkanban/tickets.json` | Per-project ticket storage |
| Worktrees | `{repo}-worktrees/` | Default sibling to repo |
| Status cache | `~/.cache/openkanban-status/` | Agent status files |

## Concurrency Considerations

1. **File locking**: Use `flock` or similar when writing JSON files
2. **Atomic writes**: Write to temp file, then rename
3. **Agent polling**: Run in separate goroutine, update state via channels
4. **PTY operations**: Terminal panes managed per-ticket with mutex protection

```go
// Atomic write pattern
func (s *JSONStorage) SaveBoard(board *Board) error {
    data, err := json.MarshalIndent(board, "", "  ")
    if err != nil {
        return err
    }
    
    tmpFile := s.boardPath(board.ID) + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return err
    }
    
    return os.Rename(tmpFile, s.boardPath(board.ID))
}
```
