# Architecture

## System Overview

OpenKanban is a TUI application built with Go and Bubbletea that orchestrates AI coding agents across multiple isolated development environments.

```
┌─────────────────────────────────────────────────────────────────┐
│                       OpenKanban TUI                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Backlog   │  │ In Progress │  │    Done     │              │
│  │  ┌───────┐  │  │  ┌───────┐  │  │  ┌───────┐  │              │
│  │  │Ticket │  │  │  │Ticket │  │  │  │Ticket │  │              │
│  │  └───────┘  │  │  └───────┘  │  │  └───────┘  │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Core Engine                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Project/     │  │ Git Manager  │  │Terminal Panes│          │
│  │ TicketStore  │  │  (worktrees) │  │   (PTY)      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                     System Layer                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Filesystem  │  │     Git      │  │  PTY/vt10x   │          │
│  │ (.openkanban)│  │  (worktrees) │  │ (terminals)  │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
openkanban/
├── cmd/root.go              # CLI commands (Cobra)
├── main.go                  # Entry point
├── internal/
│   ├── app/app.go           # Application orchestration
│   ├── ui/
│   │   ├── model.go         # Bubbletea model, Update loop, key handling
│   │   └── view.go          # Rendering logic
│   ├── board/board.go       # Ticket struct, columns, status types
│   ├── project/
│   │   ├── project.go       # Project model
│   │   ├── store.go         # Project registry (~/.config/openkanban/projects.json)
│   │   ├── tickets.go       # TicketStore, GlobalTicketStore
│   │   └── filter.go        # SavedFilter for views
│   ├── terminal/pane.go     # PTY-based embedded terminal (vt10x)
│   ├── agent/
│   │   ├── agent.go         # Agent manager
│   │   ├── context.go       # Ticket context injection
│   │   ├── server.go        # OpenCode server integration
│   │   └── status.go        # Agent status detection
│   ├── git/worktree.go      # Git worktree operations
│   └── config/config.go     # Configuration loading
├── docs/
│   ├── AGENT_INTEGRATION.md
│   ├── CONFIGURATION.md
│   ├── DATA_MODEL.md
│   └── UI_DESIGN.md
├── go.mod
├── go.sum
└── README.md
```

## Component Breakdown

### 1. TUI Layer (`internal/ui/`)

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) (Elm architecture):

```go
type Model struct {
    config          *config.Config
    globalStore     *project.GlobalTicketStore  // All projects & tickets
    columns         []board.Column
    filterProjectID string                       // Current project filter
    worktreeMgrs    map[string]*git.WorktreeManager
    agentMgr        *agent.Manager
    mode            Mode                         // Normal, Create, Edit, AgentView...
    panes           map[board.TicketID]*terminal.Pane
    // ... navigation state, form inputs, etc.
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)
    case tea.MouseMsg:
        return m.handleMouse(msg)
    case terminal.OutputMsg:
        return m.handleTerminalMsg(msg)
    // ...
    }
}
```

**Modes:**
- `ModeNormal` - Board navigation
- `ModeCreateTicket` - Ticket creation form
- `ModeEditTicket` - Ticket edit form
- `ModeAgentView` - Full-screen embedded terminal
- `ModeSettings` - Settings panel
- `ModeHelp` - Help overlay
- `ModeFilter` - Search/filter tickets
- `ModeSpawning` - Agent spawn in progress
- `ModeShuttingDown` - Application shutdown with cleanup
- `ModeConfirm` - Confirmation dialog

### 2. Project Layer (`internal/project/`)

Multi-project architecture:

```go
// Project represents a registered git repository
type Project struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    RepoPath    string          `json:"repo_path"`
    WorktreeDir string          `json:"worktree_dir,omitempty"`
    Settings    ProjectSettings `json:"settings"`
}

// ProjectRegistry manages all registered projects
// Stored in ~/.config/openkanban/projects.json
type ProjectRegistry struct {
    Projects map[string]*Project `json:"projects"`
}

// TicketStore holds tickets for a single project
// Stored in {repo}/.openkanban/tickets.json
type TicketStore struct {
    ProjectID string
    Tickets   map[board.TicketID]*board.Ticket
}

// GlobalTicketStore aggregates tickets from all projects
type GlobalTicketStore struct {
    projects     map[string]*Project
    ticketStores map[string]*TicketStore
    allTickets   map[board.TicketID]*board.Ticket
}
```

### 3. Board Layer (`internal/board/`)

Ticket and column definitions:

```go
type Ticket struct {
    ID            TicketID     `json:"id"`
    ProjectID     string       `json:"project_id"`
    Title         string       `json:"title"`
    Description   string       `json:"description,omitempty"`
    Status        TicketStatus `json:"status"`
    BranchName    string       `json:"branch_name,omitempty"`
    BaseBranch    string       `json:"base_branch,omitempty"`
    WorktreePath  string       `json:"worktree_path,omitempty"`
    AgentType     string       `json:"agent_type,omitempty"`
    AgentStatus   AgentStatus  `json:"agent_status,omitempty"`
    CreatedAt     time.Time    `json:"created_at"`
    UpdatedAt     time.Time    `json:"updated_at"`
}

type TicketStatus string
const (
    StatusBacklog    TicketStatus = "backlog"
    StatusInProgress TicketStatus = "in_progress"
    StatusDone       TicketStatus = "done"
)

type AgentStatus string
const (
    AgentNone      AgentStatus = "none"
    AgentIdle      AgentStatus = "idle"
    AgentWorking   AgentStatus = "working"
    AgentWaiting   AgentStatus = "waiting"
    AgentCompleted AgentStatus = "completed"
    AgentError     AgentStatus = "error"
)
```

### 4. Terminal Layer (`internal/terminal/`)

Embedded PTY-based terminals using `creack/pty` and `hinshun/vt10x`:

```go
type Pane struct {
    id      string
    vt      vt10x.Terminal   // Virtual terminal state
    pty     *os.File         // PTY file descriptor
    cmd     *exec.Cmd
    workdir string
    width   int
    height  int
}

// Start launches a command in the PTY
func (p *Pane) Start(command string, args ...string) tea.Cmd {
    return func() tea.Msg {
        p.vt = vt10x.New(vt10x.WithSize(p.width, p.height))
        p.cmd = exec.Command(command, args...)
        p.cmd.Dir = p.workdir
        
        ptmx, err := pty.Start(p.cmd)
        if err != nil {
            return ExitMsg{PaneID: p.id, Err: err}
        }
        p.pty = ptmx
        
        // Start reading from PTY
        return p.readOutput()()
    }
}

// HandleKey converts Bubbletea key messages to PTY input
func (p *Pane) HandleKey(msg tea.KeyMsg) tea.Msg {
    if msg.String() == "ctrl+g" {
        return ExitFocusMsg{}  // Return to board
    }
    p.pty.Write(translateKey(msg))
    return nil
}

// View renders the vt10x terminal buffer
func (p *Pane) View() string {
    // Render cells with colors from vt10x
}
```

### 5. Git Layer (`internal/git/`)

Worktree management:

```go
type WorktreeManager struct {
    project     *project.Project
    repoPath    string
    worktreeDir string
}

func (m *WorktreeManager) CreateWorktree(branchName, baseBranch string) (string, error) {
    worktreePath := filepath.Join(m.worktreeDir, branchName)
    
    // git worktree add -b <branch> <path> <base>
    cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
    cmd.Dir = m.repoPath
    return worktreePath, cmd.Run()
}

func (m *WorktreeManager) RemoveWorktree(path string) error {
    cmd := exec.Command("git", "worktree", "remove", path)
    cmd.Dir = m.repoPath
    return cmd.Run()
}
```

### 6. Agent Layer (`internal/agent/`)

Agent configuration and context injection:

```go
type Manager struct {
    config *config.Config
}

// BuildContextPrompt creates the initial prompt for an agent
func BuildContextPrompt(template string, ticket *board.Ticket) string {
    // Replace {{.Title}}, {{.Description}}, {{.BranchName}}, etc.
}

// StatusDetector polls agent status
type StatusDetector struct {
    cache map[string]cachedStatus
}

func (d *StatusDetector) DetectStatus(agentType, sessionID string, running bool) board.AgentStatus {
    // Check status files, API endpoints, or process state
}
```

## Data Flow

### Creating a Ticket

```
User presses 'n'
       │
       ▼
┌──────────────┐
│ Show Form    │ ← ModeCreateTicket
└──────────────┘
       │
       ▼ (user enters title, presses Enter)
┌──────────────┐
│ NewTicket()  │ ← board.NewTicket(title, projectID)
└──────────────┘
       │
       ▼
┌──────────────┐
│ Add to Store │ ← globalStore.Add(ticket)
└──────────────┘
       │
       ▼
┌──────────────┐
│ Save JSON    │ ← ticketStore.Save()
└──────────────┘
       │
       ▼
┌──────────────┐
│ Update View  │ ← refreshColumnTickets()
└──────────────┘
```

### Moving Ticket to "In Progress"

```
User presses 'space' on backlog ticket
       │
       ▼
┌──────────────────┐
│ Check Worktree   │ ← ticket.WorktreePath == ""?
└──────────────────┘
       │ (no worktree)
       ▼
┌──────────────────┐
│ Create Worktree  │ ← worktreeMgr.CreateWorktree()
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Update Status    │ ← globalStore.Move(id, StatusInProgress)
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Save & Refresh   │ ← saveTicket(), refreshColumnTickets()
└──────────────────┘
```

### Spawning Agent

```
User presses 's' on in-progress ticket
       │
       ▼
┌──────────────────┐
│ Get Agent Config │ ← config.Agents[agentType]
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Create Pane      │ ← terminal.New(ticketID, width, height)
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Build Args       │ ← buildAgentArgs(cfg, ticket, isNewSession)
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Start PTY        │ ← pane.Start(command, args...)
└──────────────────┘
       │
       ▼
┌──────────────────┐
│ Enter AgentView  │ ← mode = ModeAgentView
└──────────────────┘
```

## Configuration

Stored in `~/.config/openkanban/config.json`:

```json
{
  "defaults": {
    "default_agent": "opencode",
    "branch_prefix": "task/",
    "branch_template": "{prefix}{slug}",
    "slug_max_length": 40,
    "init_prompt": "..."
  },
  "agents": {
    "opencode": {
      "command": "opencode",
      "args": [],
      "status_file": ".opencode/status.json",
      "init_prompt": "..."
    },
    "claude": {
      "command": "claude",
      "args": ["--dangerously-skip-permissions"],
      "status_file": ".claude/status.json"
    }
  },
  "ui": {
    "theme": "catppuccin-mocha",
    "show_agent_status": true,
    "refresh_interval": 5
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

## Testing Strategy

```
internal/
├── board/
│   ├── board.go
│   └── board_test.go       # Ticket CRUD, slugify, status transitions
├── config/
│   ├── config.go
│   └── config_test.go      # Config loading, defaults
├── project/
│   ├── store.go
│   └── store_test.go       # Registry operations
├── git/
│   ├── worktree.go
│   └── worktree_test.go    # Integration tests (requires git)
└── agent/
    ├── status.go
    └── status_test.go      # Status detection logic
```

## Future Considerations

### Plugin System
Custom agents via config with status detection:

```json
{
  "agents": {
    "custom": {
      "command": "/path/to/my-agent",
      "args": ["--mode", "interactive"],
      "status_command": "pgrep -f my-agent"
    }
  }
}
```

### GitHub/GitLab Sync
Bi-directional sync with issue trackers.

### SQLite Backend
Optional SQLite storage for large ticket counts.
