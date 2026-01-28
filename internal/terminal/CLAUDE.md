# Terminal Package

PTY management and terminal emulation for agent processes.

## Core Components

- **Pane** - manages single PTY + virtual terminal
- **ScrollbackBuffer** - ring buffer for history (default 10k lines)
- **SelectionState** - text selection state machine

## PTY Handling

Uses `creack/pty`:
```go
pty.Start(cmd)      // spawn with PTY
pty.Setsize(f, ws)  // resize
```

## Terminal Emulation

Uses `vt10x` for escape sequence parsing:
- Cursor management
- Cell-based rendering
- Color/attribute handling

## Message Types

BubbleTea integration:
- `OutputMsg` - new terminal output
- `ExitMsg` - process terminated
- `RenderTickMsg` - throttled render trigger

## Rendering

- Throttled at 50ms intervals
- `dirty` flag tracks when re-render needed
- Cached view string until dirty

## Key Translation

`translateKey()` converts BubbleTea `KeyMsg` to PTY bytes:
- Arrow keys → escape sequences
- Ctrl+C → 0x03
- Enter → \r

## Environment

`buildCleanEnv()`:
- Sets `TERM=xterm-256color`
- Strips agent-related env vars
- Preserves PATH, HOME, USER

## Escape Sequence Detection

Byte scanning for mode switches:
- Mouse mode: `\x1b[?1000h`
- Alt screen: `\x1b[?1049h`

## Anti-Patterns

- Don't write to PTY without checking if alive
- Don't skip resize handling - causes display corruption
- Don't render on every output - use throttling
- Don't leak PTY file descriptors - always close
- Don't assume vt10x handles all sequences - some need manual parsing
