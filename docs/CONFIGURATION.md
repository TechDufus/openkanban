# Configuration Guide

OpenKanban configuration lives in `~/.config/openkanban/config.json`.

## Default Configuration

```json
{
  "defaults": {
    "default_agent": "opencode",
    "branch_prefix": "task/",
    "branch_naming": "template",
    "branch_template": "{prefix}{slug}",
    "slug_max_length": 40,
    "auto_spawn_agent": true,
    "auto_create_branch": true
  },
  "agents": {
    "opencode": {
      "command": "opencode",
      "args": [],
      "status_file": ".opencode/status.json"
    },
    "claude": {
      "command": "claude",
      "args": ["--dangerously-skip-permissions"],
      "status_file": ".claude/status.json"
    },
    "aider": {
      "command": "aider",
      "args": ["--yes"]
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

## Agents

Define any CLI-based agent. The command runs in the ticket's worktree directory.

```json
{
  "agents": {
    "my-agent": {
      "command": "my-agent-cli",
      "args": ["--flag", "value"],
      "env": {
        "CUSTOM_VAR": "value"
      },
      "init_prompt": "Custom prompt template with {{.Title}} and {{.Description}}"
    }
  }
}
```

### Init Prompt Variables

When spawning an agent, OpenKanban can inject ticket context:

- `{{.Title}}` - Ticket title
- `{{.Description}}` - Ticket description
- `{{.BranchName}}` - Git branch name
- `{{.BaseBranch}}` - Base branch (e.g., main)

## Branch Naming

Control how branches are named:

```json
{
  "defaults": {
    "branch_prefix": "feature/",
    "branch_template": "{prefix}{slug}",
    "slug_max_length": 30
  }
}
```

A ticket titled "Add user authentication" becomes branch `feature/add-user-authentication`.

## Cleanup Behavior

When deleting tickets:

```json
{
  "cleanup": {
    "delete_worktree": true,
    "delete_branch": false,
    "force_worktree_removal": false
  }
}
```

- `delete_worktree` - Remove the git worktree directory
- `delete_branch` - Also delete the git branch
- `force_worktree_removal` - Force removal even with uncommitted changes

## Behavior

Application behavior preferences:

```json
{
  "behavior": {
    "confirm_quit_with_agents": true
  }
}
```

- `confirm_quit_with_agents` - Prompt before quitting when agents are running (default: true). Set to false to auto-close agents without confirmation.

## UI

Display preferences:

```json
{
  "ui": {
    "sidebar_visible": true
  }
}
```

- `sidebar_visible` - Show project sidebar on startup (default: true). Toggle with `[` key during use.

## OpenCode Integration

OpenKanban has deep integration with OpenCode. When enabled, it starts an OpenCode server and connects ticket terminals to it for accurate status detection.

```json
{
  "opencode": {
    "server_enabled": true,
    "server_port": 4096,
    "poll_interval": 1
  }
}
```

- `server_enabled` - Start OpenCode server for enhanced status detection (default: true). When enabled, ticket terminals use `opencode attach` to connect to the shared server.
- `server_port` - Port for the OpenCode server (default: 4096). If a server is already running on this port, OpenKanban will reuse it.
- `poll_interval` - Agent status polling interval in seconds (default: 1).

When `server_enabled` is false, OpenCode runs in standalone mode per-ticket with basic status detection.

## Keybindings

All keybindings are shown in-app with `?`. Custom keybindings coming soon.

## Full Keybindings Reference

### Board View

| Key | Action |
|-----|--------|
| `j/k` | Move cursor up/down |
| `h/l` | Move between columns |
| `g` | Go to first ticket |
| `G` | Go to last ticket |
| `space` | Move ticket to next column |
| `-` | Move ticket to previous column |
| `enter` | Attach to running agent |
| `n` | Create new ticket |
| `e` | Edit ticket |
| `s` | Spawn agent for ticket |
| `S` | Stop agent |
| `d` | Delete ticket |
| `/` | Search/filter tickets |
| `esc` | Clear filter |
| `tab` | Toggle sidebar focus |
| `[` | Toggle sidebar visibility |
| `O` | Open settings |
| `?` | Show help |
| `q` | Quit |

### Sidebar

| Key | Action |
|-----|--------|
| `h` | Focus sidebar (from column 0) |
| `l` | Return to board |
| `j/k` | Navigate projects |
| `enter` | Select project filter |

### Agent View

| Key | Action |
|-----|--------|
| `ctrl+g` | Return to board |
| All other keys | Passed to agent |
