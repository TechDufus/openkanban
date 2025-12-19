# Architecture

## System Overview

OpenKanban is a TUI application for orchestrating AI coding agents. It uses a **client/server daemon architecture** built with TypeScript and Bun.

```
┌──────────────────────────────────────────────────────────────────┐
│                    OpenKanban Daemon (Server)                     │
│                     Runs ONCE, survives reconnects                │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                      Core Services                            ││
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ ││
│  │  │ PTY Manager │ │ Board Store │ │   Agent Orchestrator    │ ││
│  │  │ Bun.Terminal│ │   (State)   │ │ (Spawn, Monitor, Kill)  │ ││
│  │  └──────┬──────┘ └──────┬──────┘ └───────────┬─────────────┘ ││
│  │         └───────────────┼────────────────────┘               ││
│  └─────────────────────────┼────────────────────────────────────┘│
│                            ▼                                      │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                    Hono HTTP Server                           ││
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ ││
│  │  │  REST API   │ │  WebSocket  │ │    Static Files         │ ││
│  │  │ /api/board  │ │  /ws        │ │    (future web UI)      │ ││
│  │  └─────────────┘ └─────────────┘ └─────────────────────────┘ ││
│  └──────────────────────────┬───────────────────────────────────┘│
└─────────────────────────────┼────────────────────────────────────┘
                              │ localhost:4200
             ┌────────────────┼────────────────┐
             ▼                ▼                ▼
        ┌────────┐      ┌──────────┐     ┌────────┐
        │  TUI   │      │ Desktop  │     │  Web   │
        │ Client │      │ (Tauri)  │     │ Client │
        │ (MVP)  │      │ (Future) │     │(Future)│
        └────────┘      └──────────┘     └────────┘
```

## Why Client/Server?

| Benefit | Description |
|---------|-------------|
| **Run Once** | Daemon process - start it and forget |
| **Agent Survival** | Agents keep running when TUI closes |
| **Hot Reload** | Update server code without killing agents |
| **Auto-Update** | Server can self-update, restart gracefully |
| **Multi-Client** | TUI + web + desktop can connect simultaneously |

## Project Structure

```
openkanban/
├── bin/
│   └── openkanban.ts           # CLI entry point
├── packages/
│   ├── shared/                 # Shared types and utilities
│   │   └── src/
│   │       ├── types.ts        # Board, Ticket, Column types
│   │       ├── protocol.ts     # WebSocket message types
│   │       └── index.ts
│   ├── server/                 # Daemon server
│   │   └── src/
│   │       ├── index.ts        # Entry point
│   │       ├── daemon.ts       # Daemon lifecycle (PID, signals)
│   │       └── api/
│   │           ├── index.ts    # Hono app
│   │           └── routes/     # REST endpoints
│   └── tui/                    # TUI client
│       └── src/
│           ├── index.tsx       # Entry point
│           ├── App.tsx         # Root component
│           └── jsx.d.ts        # OpenTUI type declarations
├── docs/
│   └── migration/
│       └── PLAN.md             # Full migration plan
├── package.json                # Workspace root
├── tsconfig.json               # Shared TypeScript config
└── bunfig.toml                 # Bun configuration
```

## Technology Stack

| Component | Technology |
|-----------|------------|
| Runtime | Bun 1.3.5+ |
| Language | TypeScript (strict mode) |
| HTTP Server | Hono |
| TUI Framework | @opentui/core + @opentui/solid |
| UI Reactivity | SolidJS |
| PTY | Bun.Terminal (native) |
| Terminal Emulation | @xterm/headless |

## Package Architecture

### @openkanban/shared

Shared types and WebSocket protocol definitions:

```typescript
export type TicketStatus = "backlog" | "in_progress" | "done" | "archived"
export type AgentStatus = "none" | "idle" | "working" | "waiting" | "completed" | "error"

export interface Ticket {
  id: string
  title: string
  status: TicketStatus
  agentStatus: AgentStatus
  // ...
}

export interface Board {
  id: string
  name: string
  columns: Column[]
  tickets: Ticket[]
  settings: BoardSettings
}
```

### @openkanban/server

Daemon server with Hono HTTP/WebSocket:

```typescript
export class Daemon {
  private server?: ReturnType<typeof Bun.serve>
  
  async start(port = 4200): Promise<void> {
    // Single instance enforcement via PID file
    if (this.isAlreadyRunning()) {
      throw new Error("Daemon already running")
    }
    
    // Start Hono server
    this.server = Bun.serve({
      port,
      fetch: app.fetch,
      websocket: { /* handlers */ },
    })
    
    // Signal handlers for graceful shutdown
    process.on("SIGTERM", () => this.shutdown())
    process.on("SIGINT", () => this.shutdown())
  }
}
```

### @openkanban/tui

TUI client using OpenTUI + SolidJS:

```tsx
export function App(props: { serverPort: number }) {
  const [connected, setConnected] = createSignal(false)
  const [board, setBoard] = createSignal<Board | null>(null)
  
  // WebSocket connection with auto-reconnect
  onMount(() => {
    const ws = new WebSocket(`ws://localhost:${props.serverPort}/ws`)
    ws.onopen = () => {
      setConnected(true)
      ws.send(JSON.stringify({ type: "board:subscribe" }))
    }
    // ...
  })
  
  return (
    <box flexDirection="column" width="100%" height="100%">
      <Header connected={connected()} />
      <BoardView board={board()} />
      <StatusBar />
    </box>
  )
}
```

## WebSocket Protocol

### Client → Server

```typescript
type ClientMessage =
  | { type: "terminal:input"; sessionId: string; data: string }
  | { type: "terminal:resize"; sessionId: string; cols: number; rows: number }
  | { type: "board:subscribe" }
  | { type: "ticket:create"; ticket: Partial<Ticket> }
  | { type: "ticket:move"; ticketId: string; status: TicketStatus }
  | { type: "agent:spawn"; ticketId: string }
  | { type: "agent:kill"; ticketId: string }
```

### Server → Client

```typescript
type ServerMessage =
  | { type: "terminal:output"; sessionId: string; data: string }
  | { type: "board:state"; board: Board }
  | { type: "ticket:created"; ticket: Ticket }
  | { type: "ticket:updated"; ticket: Ticket }
  | { type: "agent:status"; ticketId: string; status: AgentStatus }
  | { type: "error"; message: string }
```

## REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/health | Health check |
| GET | /api/board | Get full board state |
| POST | /api/tickets | Create ticket |
| PATCH | /api/tickets/:id | Update ticket |
| DELETE | /api/tickets/:id | Delete ticket |
| POST | /api/tickets/:id/move | Move ticket to column |

## CLI Commands

```bash
openkanban                # Start daemon + TUI (default)
openkanban daemon         # Run daemon only
openkanban daemon -f      # Run daemon in foreground
openkanban status         # Show daemon status
openkanban stop           # Stop the daemon
openkanban restart        # Restart the daemon
```

## Data Flow

### Creating a Ticket

```
TUI Client                         Server
    │                                 │
    │  WS: {type: "ticket:create"}   │
    │ ─────────────────────────────► │
    │                                 │ Create ticket
    │                                 │ Persist to JSON
    │                                 │
    │  WS: {type: "ticket:created"}  │
    │ ◄───────────────────────────── │
    │                                 │
    │  Update local state             │
    │  Re-render board                │
```

### Spawning an Agent

```
TUI Client                         Server
    │                                 │
    │  WS: {type: "agent:spawn"}     │
    │ ─────────────────────────────► │
    │                                 │ Create PTY session
    │                                 │ Start agent process
    │                                 │
    │  WS: {type: "agent:status"}    │
    │ ◄───────────────────────────── │
    │                                 │
    │  WS: {type: "terminal:output"} │
    │ ◄───────────────────────────── │ (streaming)
    │                                 │
```

## Daemon Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Entry Point                          │
│                        openkanban [command]                      │
└─────────────────────────┬───────────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │  start   │    │  status  │    │   stop   │
    │ (daemon) │    │  (info)  │    │  (kill)  │
    └────┬─────┘    └──────────┘    └──────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│ Is daemon running? (check ~/.openkanban/daemon.pid)              │
├─────────────────────────────────────────────────────────────────┤
│  NO  → Start daemon (detached), then connect TUI                 │
│  YES → Connect TUI to existing daemon                            │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration

Configuration is stored in `~/.openkanban/config.yaml`:

```yaml
port: 4200

default_agent: opencode
worktree_base: .worktrees
base_branch: main
auto_spawn: false

agents:
  opencode:
    command: opencode
    args: []
  claude:
    command: claude
    args: []
  aider:
    command: aider
    args: ["--yes"]
```

## Future Considerations

### Desktop App (Tauri)

Same server, Tauri frontend instead of TUI:

```
openkanban daemon &        # Start server
openkanban-desktop         # Launch Tauri app connecting to localhost:4200
```

### Web Interface

Serve static files from the server:

```
packages/
└── web/                   # Future web client
    └── src/
        └── index.tsx      # SolidJS web app
```

### Remote Access

Expose server on network for remote control:

```yaml
bind: 0.0.0.0:4200
auth:
  enabled: true
  token: ${OPENKANBAN_TOKEN}
```
