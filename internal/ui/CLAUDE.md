# UI Package

BubbleTea-based terminal UI with vim-style navigation.

## Model Structure

Single `Model` struct implements `tea.Model`:
- `Init()` - startup commands
- `Update(msg) (Model, Cmd)` - message handling
- `View() string` - render output

## Mode State Machine

`Mode` type controls behavior routing:
```go
type Mode string
const (
    ModeNormal    Mode = "normal"
    ModeAgentView Mode = "agent_view"
    ModeSettings  Mode = "settings"
    // ...
)
```

Key handlers dispatch by mode: `handleNormalMode()`, `handleAgentViewMode()`

## Key Bindings

Vim-style navigation:
- `h/j/k/l` - movement
- `g/G` - jump to start/end
- `n` - new item
- `d` - delete
- `Enter` - select/confirm
- `Esc` - cancel/back

## View Composition

Separate render methods composed in `View()`:
- `renderHeader()`, `renderBoard()`, `renderColumn()`
- `renderTicket()`, `renderStatusBar()`

## Styling

All styling via lipgloss with theme-based `uiColors` struct.
Never use raw ANSI codes in UI rendering.

## Messages

Custom messages for async operations:
```go
type spawnReadyMsg struct {...}
type agentStatusMsg struct {...}
```

Return `tea.Cmd` from `Update()` for async work.

## Terminal Panes

`panes map[board.TicketID]*terminal.Pane` - one per spawned agent

## Anti-Patterns

- Don't block in Update() - use Cmd for async
- Don't render directly in Update() - only in View()
- Don't store computed strings - recompute in View()
- Don't access panes without nil check
