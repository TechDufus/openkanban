# Agent Integration

This document describes how OpenKanban spawns, monitors, and manages AI coding agents.

## Overview

OpenKanban runs AI agents in embedded PTY terminals within the TUI. This approach provides:

- **Seamless UX**: No context switching to external terminals
- **Integrated view**: See agent output directly in the board
- **Full terminal emulation**: Colors, cursor movement, interactive prompts
- **Easy navigation**: `ctrl+g` returns to board view

## Supported Agents

### Tier 1: Full Support

Agents with native support and session continuation.

| Agent | Command | Session Resume | Notes |
|-------|---------|----------------|-------|
| OpenCode | `opencode` | `--session` flag | Native session lookup |
| Claude Code | `claude` | `--continue` flag | Continues last session |
| Gemini CLI | `gemini` | `--resume` flag | Auto-approve with `--yolo` |
| Codex CLI | `codex` | `resume --last` | Auto-approve with `--full-auto` |
| Aider | `aider` | N/A | Use `--yes` flag |

### Tier 2: Generic Support

Any CLI tool that runs interactively.

```json
{
  "agents": {
    "custom-agent": {
      "command": "/path/to/agent",
      "args": ["--interactive"]
    }
  }
}
```

## Agent Lifecycle

### Spawning an Agent

```
User presses 's' on in-progress ticket
       │
       ▼
┌─────────────────────────────────────────┐
│ 1. Check ticket status                  │
│    Must be "in_progress"                │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 2. Ensure worktree exists               │
│    Create if missing                    │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 3. Create terminal pane                 │
│    terminal.New(ticketID, width, height)│
│    pane.SetWorkdir(worktreePath)        │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 4. Build agent command                  │
│    Add context prompt for new sessions  │
│    Add --continue/--session for resume  │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 5. Start PTY                            │
│    pane.Start(command, args...)         │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 6. Enter agent view                     │
│    mode = ModeAgentView                 │
│    Full-screen terminal display         │
└─────────────────────────────────────────┘
```

### Implementation

```go
// internal/ui/model.go - spawnAgent()

func (m *Model) spawnAgent() (tea.Model, tea.Cmd) {
    ticket := m.selectedTicket()
    if ticket.Status != board.StatusInProgress {
        m.notify("Move ticket to In Progress first")
        return m, nil
    }

    // Ensure worktree exists
    if ticket.WorktreePath == "" {
        if err := m.setupWorktree(ticket); err != nil {
            m.notify("Failed to create worktree: " + err.Error())
            return m, nil
        }
    }

    // Get agent config
    agentType := proj.Settings.DefaultAgent
    agentCfg := m.config.Agents[agentType]

    // Create terminal pane
    pane := terminal.New(string(ticket.ID), m.width, m.height-2)
    pane.SetWorkdir(ticket.WorktreePath)
    m.panes[ticket.ID] = pane

    // Build args with context
    isNewSession := agent.ShouldInjectContext(ticket)
    args := m.buildAgentArgs(agentCfg, ticket, isNewSession)

    // Enter agent view
    m.mode = ModeAgentView
    m.focusedPane = ticket.ID

    return m, pane.Start(agentCfg.Command, args...)
}
```

### Context Injection

For new sessions, OpenKanban injects ticket context:

```go
// internal/agent/context.go

func BuildContextPrompt(template string, ticket *board.Ticket) string {
    // Template variables:
    // {{.Title}}       - Ticket title
    // {{.Description}} - Ticket description
    // {{.BranchName}}  - Git branch name
    // {{.BaseBranch}}  - Base branch (e.g., main)
    
    result := strings.ReplaceAll(template, "{{.Title}}", ticket.Title)
    result = strings.ReplaceAll(result, "{{.Description}}", ticket.Description)
    // ...
    return result
}

func ShouldInjectContext(ticket *board.Ticket) bool {
    // New session if never spawned before
    return ticket.AgentSpawnedAt == nil
}
```

Default prompt template:

```
You have been spawned by OpenKanban to work on a ticket.

**Title:** {{.Title}}

**Description:**
{{.Description}}

**Branch:** {{.BranchName}} (from {{.BaseBranch}})

Focus on completing this ticket. Ask clarifying questions if needed.
```

### Session Continuation

For returning to an existing session:

**OpenCode:**
```go
case "opencode":
    if !isNewSession {
        if sessionID := agent.FindOpencodeSession(ticket.WorktreePath); sessionID != "" {
            args = append(args, "--session", sessionID)
        }
    }
```

**Claude Code:**
```go
case "claude":
    if !isNewSession {
        args = append(args, "--continue")
    }
```

## Terminal Pane

### PTY Architecture

```go
// internal/terminal/pane.go

type Pane struct {
    id      string
    vt      vt10x.Terminal   // Virtual terminal (handles escape sequences)
    pty     *os.File         // PTY master file descriptor
    cmd     *exec.Cmd        // Running process
    workdir string           // Working directory
    width   int
    height  int
}
```

### Starting a Process

```go
func (p *Pane) Start(command string, args ...string) tea.Cmd {
    return func() tea.Msg {
        // Create virtual terminal
        p.vt = vt10x.New(vt10x.WithSize(p.width, p.height))

        // Build command
        p.cmd = exec.Command(command, args...)
        p.cmd.Env = buildCleanEnv()  // Filter OPENCODE_*, CLAUDE_* vars
        p.cmd.Dir = p.workdir

        // Start PTY
        ptmx, err := pty.Start(p.cmd)
        if err != nil {
            return ExitMsg{PaneID: p.id, Err: err}
        }
        p.pty = ptmx

        // Set terminal size
        pty.Setsize(p.pty, &pty.Winsize{
            Rows: uint16(p.height),
            Cols: uint16(p.width),
        })

        // Start reading output
        return p.readOutput()()
    }
}
```

### Input Handling

```go
func (p *Pane) HandleKey(msg tea.KeyMsg) tea.Msg {
    // ctrl+g exits agent view
    if msg.String() == "ctrl+g" {
        return ExitFocusMsg{}
    }

    // Convert key to PTY escape sequence
    input := p.translateKey(msg)
    p.pty.Write(input)
    return nil
}

func (p *Pane) translateKey(msg tea.KeyMsg) []byte {
    switch msg.Type {
    case tea.KeyEnter:
        return []byte("\r")
    case tea.KeyUp:
        return []byte("\x1b[A")
    case tea.KeyDown:
        return []byte("\x1b[B")
    // ... etc
    }
    return []byte(string(msg.Runes))
}
```

### Rendering

```go
func (p *Pane) View() string {
    // Render vt10x buffer with ANSI colors
    // Handles cursor position, colors, attributes
}
```

## Status Detection

### Status Types

```go
type AgentStatus string

const (
    AgentNone      AgentStatus = ""         // No agent
    AgentIdle      AgentStatus = "idle"     // Waiting for input
    AgentWorking   AgentStatus = "working"  // Processing
    AgentWaiting   AgentStatus = "waiting"  // Waiting for user
    AgentCompleted AgentStatus = "completed"
    AgentError     AgentStatus = "error"
)
```

### Detection Methods

**1. Process State**
```go
func (p *Pane) Running() bool {
    return p.running && p.cmd != nil && p.cmd.Process != nil
}
```

**2. Status Files** (for OpenCode/Claude)
```go
func (d *StatusDetector) DetectStatus(agentType, sessionID string, running bool) AgentStatus {
    if !running {
        return AgentNone
    }
    
    // Check agent-specific status file
    switch agentType {
    case "opencode":
        return d.checkOpencodeStatus(sessionID)
    case "claude":
        return d.checkClaudeStatus(sessionID)
    }
    
    return AgentIdle
}
```

### Polling

Status is polled at configurable intervals:

```go
func tickAgentStatus(d time.Duration) tea.Cmd {
    return tea.Tick(d, func(t time.Time) tea.Msg {
        return agentStatusMsg(t)
    })
}

// In Update():
case agentStatusMsg:
    return m, tea.Batch(
        m.pollAgentStatusesAsync(),
        tickAgentStatus(m.agentMgr.StatusPollInterval()),
    )
```

## Configuration

### Agent Config

```json
{
  "agents": {
    "opencode": {
      "command": "opencode",
      "args": [],
      "status_file": ".opencode/status.json",
      "init_prompt": "Custom prompt for OpenCode..."
    },
    "claude": {
      "command": "claude",
      "args": ["--dangerously-skip-permissions"],
      "status_file": ".claude/status.json",
      "init_prompt": "Custom prompt for Claude..."
    },
    "gemini": {
      "command": "gemini",
      "args": ["--yolo"],
      "init_prompt": "Custom prompt for Gemini..."
    },
    "codex": {
      "command": "codex",
      "args": ["--full-auto"],
      "init_prompt": "Custom prompt for Codex..."
    },
    "aider": {
      "command": "aider",
      "args": ["--yes"],
      "init_prompt": "Custom prompt for Aider..."
    }
  },
  "defaults": {
    "default_agent": "opencode",
    "init_prompt": "Default prompt for all agents..."
  }
}
```

### Prompt Priority

1. Agent-specific `init_prompt` in config
2. Global `defaults.init_prompt` in config
3. Built-in default prompt

## Environment Isolation

When spawning agents, OpenKanban filters environment variables to prevent nested session detection:

```go
func buildCleanEnv() []string {
    var env []string
    for _, e := range os.Environ() {
        key := strings.Split(e, "=")[0]
        // Skip agent-specific vars that might cause issues
        if key == "OPENCODE" || strings.HasPrefix(key, "OPENCODE_") {
            continue
        }
        if key == "CLAUDE" || strings.HasPrefix(key, "CLAUDE_") {
            continue
        }
        if key == "GEMINI" || strings.HasPrefix(key, "GEMINI_") {
            continue
        }
        if key == "CODEX" || strings.HasPrefix(key, "CODEX_") {
            continue
        }
        env = append(env, e)
    }
    env = append(env, "TERM=xterm-256color")
    return env
}
```

## Adding New Agents

### 1. Add Configuration

```json
{
  "agents": {
    "new-agent": {
      "command": "new-agent-cli",
      "args": ["--mode", "interactive"],
      "init_prompt": "You are working on: {{.Title}}"
    }
  }
}
```

### 2. Handle Session Resume (Optional)

If the agent supports session continuation, add logic to `buildAgentArgs()`:

```go
case "new-agent":
    if !isNewSession {
        // Add session resume flag
        args = append(args, "--resume", ticket.ID)
    }
```

### 3. Add Status Detection (Optional)

If the agent writes status files:

```go
func (d *StatusDetector) checkNewAgentStatus(sessionID string) AgentStatus {
    path := filepath.Join(os.Getenv("HOME"), ".new-agent", "status", sessionID)
    // Read and parse status
}
```

## Error Handling

### Spawn Failures

```go
// PTY start fails
if err != nil {
    return ExitMsg{PaneID: p.id, Err: err}
}

// Handled in Update():
case terminal.ExitMsg:
    delete(m.panes, board.TicketID(msg.PaneID))
    m.notify("Agent exited")
```

### Agent Crashes

When the agent process exits:

```go
// In pane read loop - EOF means process exited
n, err := ptyFile.Read(buf)
if err != nil {
    return ExitMsg{PaneID: paneID, Err: err}
}
```

### Recovery

User can restart with `s` key on the ticket.

## Security Considerations

### Command Sources

Agent commands come only from config, never user input:

```go
// SAFE: From validated config
agentCfg := m.config.Agents[agentType]
pane.Start(agentCfg.Command, args...)

// NEVER: From user input
// pane.Start(userInput, ...)
```

### Worktree Validation

Worktrees are always within the project's designated directory:

```go
worktreePath := filepath.Join(m.worktreeDir, branchName)
// Path is always under worktreeDir, can't escape
```

### Environment Filtering

Prevents sensitive environment variables from leaking to agents and prevents nested session issues.
