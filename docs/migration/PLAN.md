# OpenKanban Migration Plan: Go → TypeScript/OpenTUI

> **Status:** Phase 0 Complete - GO Decision  
> **Created:** 2025-12-19  
> **Updated:** 2025-12-19  
> **Decision:** Client/Server daemon architecture with OpenTUI TUI client
> **Spike Result:** PTY works via `Bun.Terminal` (native) and `@lydell/node-pty` (fallback)

---

## Executive Summary

This document captures research and planning for rewriting OpenKanban from Go/Bubble Tea to TypeScript/OpenTUI with a **client/server daemon architecture**.

### Why Client/Server?

| Benefit | Description |
|---------|-------------|
| **Run Once** | Daemon process - start it and forget |
| **Agent Survival** | Agents keep running when TUI closes |
| **Hot Reload** | Update server code without killing agents |
| **Auto-Update** | Server can self-update, restart gracefully |
| **Multi-Client** | TUI + web + desktop can connect simultaneously |
| **Remote Access** | Eventually connect from mobile/other machines |

### Architecture Overview

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
│  │  │ /api/board  │ │  /ws/term   │ │    (future web UI)      │ ││
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

### Key Discovery

**OpenCode is TypeScript (81.6%), not Go.** This means:
- Complete language change required
- Different runtime (Bun vs Go binary)
- Different TUI paradigm (reactive vs Elm architecture)

### Risk Assessment

| Risk | Severity | Status |
|------|----------|--------|
| PTY embedding uncharted in OpenTUI | ~~High~~ | ✅ **RESOLVED** - `Bun.Terminal` works natively |
| bun-pty may not work | ~~Medium~~ | ✅ **RESOLVED** - `@lydell/node-pty` works as fallback |
| Client/server complexity | Medium | Standard patterns, well understood |
| Young framework, less docs | Medium | Reference OpenCode source |
| No single-binary deployment | Low | Acceptable tradeoff |

### Spike Results (2025-12-19)

| Approach | Status | Notes |
|----------|--------|-------|
| `Bun.Terminal` | ✅ **WORKS** | Native, zero deps, Bun 1.3.5+ |
| `@lydell/node-pty` | ✅ **WORKS** | Cross-platform fallback |
| `@xterm/headless` | ✅ **WORKS** | Full terminal emulation |
| `node-pty` | ❌ FAILS | V8 symbol incompatibility |

---

## Technology Stack Comparison

| Component | Current (Go) | Target (TypeScript) |
|-----------|--------------|---------------------|
| **Language** | Go | TypeScript |
| **Runtime** | Single binary | Bun |
| **Architecture** | Monolithic TUI | Client/Server Daemon |
| **TUI Framework** | Bubble Tea (Charm) | @opentui/core + @opentui/solid |
| **UI Paradigm** | Elm (Model-View-Update) | Reactive (SolidJS signals) |
| **Styling** | Lipgloss | OpenTUI built-in |
| **Layout** | Manual calculation | Yoga (flexbox) |
| **PTY** | creack/pty | `Bun.Terminal` (native) or `@lydell/node-pty` |
| **Terminal Emulation** | hinshun/vt10x | `@xterm/headless` (tested, works) |
| **HTTP Server** | N/A | Hono |
| **Real-time** | N/A | WebSocket (native Bun) |

### Server Stack

```
hono             - Fast HTTP framework for Bun
@xterm/headless  - Terminal emulation (server-side)
Bun.Terminal     - Native PTY (Bun 1.3.5+)
```

### Client Stack (TUI)

```
@opentui/core    - Core rendering engine (Zig-based)
@opentui/solid   - SolidJS bindings
solid-js         - Reactive UI framework
yoga-layout      - Flexbox for terminals (built-in)
ghostty-opentui  - Terminal rendering component
```

---

## Daemon Architecture Deep Dive

### Server Responsibilities

| Component | Responsibility |
|-----------|----------------|
| **PTY Manager** | Spawn, resize, kill terminal sessions |
| **Board Store** | CRUD operations, JSON persistence |
| **Agent Orchestrator** | Spawn agents, monitor status, inject context |
| **WebSocket Hub** | Stream terminal output, broadcast state changes |
| **REST API** | Board/ticket CRUD, agent control |

### Client Responsibilities

| Component | Responsibility |
|-----------|----------------|
| **UI Rendering** | OpenTUI components, layout |
| **Input Handling** | Keyboard, mouse → commands |
| **WebSocket Client** | Receive state, send commands |
| **Terminal Display** | Render PTY output from server |

### Process Lifecycle

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
│ Is daemon running? (check PID file)                              │
├─────────────────────────────────────────────────────────────────┤
│  NO  → Start daemon (detached), then connect TUI                 │
│  YES → Connect TUI to existing daemon                            │
└─────────────────────────────────────────────────────────────────┘
```

### Daemon Lifecycle

```typescript
// Simplified daemon lifecycle
class OpenKanbanDaemon {
  private pidFile = "~/.openkanban/daemon.pid"
  private server: ReturnType<typeof Bun.serve>
  private ptyManager: PtyManager
  private boardStore: BoardStore
  
  async start() {
    // 1. Acquire lock (single instance)
    this.acquireLock()
    
    // 2. Initialize services
    this.boardStore = new BoardStore()
    this.ptyManager = new PtyManager()
    
    // 3. Start HTTP/WebSocket server
    this.server = Bun.serve({
      port: 4200,
      fetch: this.handleRequest.bind(this),
      websocket: this.websocketHandlers,
    })
    
    // 4. Register signal handlers
    process.on("SIGTERM", () => this.shutdown())
    process.on("SIGINT", () => this.shutdown())
    
    console.log(`Daemon started on port ${this.server.port}`)
  }
  
  async shutdown() {
    // 1. Stop accepting connections
    this.server.stop()
    
    // 2. Gracefully terminate PTY sessions
    await this.ptyManager.closeAll()
    
    // 3. Persist state
    await this.boardStore.flush()
    
    // 4. Release lock
    this.releaseLock()
    
    process.exit(0)
  }
}
```

### WebSocket Protocol

```typescript
// Client → Server messages
type ClientMessage = 
  | { type: "terminal:input", ticketId: string, data: string }
  | { type: "terminal:resize", ticketId: string, cols: number, rows: number }
  | { type: "agent:spawn", ticketId: string }
  | { type: "agent:kill", ticketId: string }
  | { type: "board:subscribe" }

// Server → Client messages
type ServerMessage =
  | { type: "terminal:output", ticketId: string, data: string }
  | { type: "terminal:exit", ticketId: string, code: number }
  | { type: "board:state", board: Board }
  | { type: "board:update", patch: BoardPatch }
  | { type: "agent:status", ticketId: string, status: AgentStatus }
```

### REST API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/board` | Get full board state |
| `POST` | `/api/tickets` | Create ticket |
| `PATCH` | `/api/tickets/:id` | Update ticket |
| `DELETE` | `/api/tickets/:id` | Delete ticket |
| `POST` | `/api/tickets/:id/move` | Move ticket to column |
| `POST` | `/api/tickets/:id/agent/spawn` | Spawn agent |
| `POST` | `/api/tickets/:id/agent/kill` | Kill agent |
| `GET` | `/api/agents/status` | All agent statuses |

### Hot Reload Strategy

```typescript
// Server supports hot reload without killing agents
// Using Bun's native --hot mode with global state preservation

declare global {
  var __openkanban_pty_sessions: Map<string, PtySession>
  var __openkanban_board_state: Board
}

// Preserve across hot reloads
globalThis.__openkanban_pty_sessions ??= new Map()
globalThis.__openkanban_board_state ??= loadFromDisk()

// Hot reload handler
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    console.log("Hot reload - preserving", 
      globalThis.__openkanban_pty_sessions.size, "PTY sessions")
  })
  import.meta.hot.accept()
}
```

### Key Differences

| Aspect | Bubble Tea | OpenTUI |
|--------|------------|---------|
| **State** | Immutable Model struct | SolidJS stores/signals |
| **Updates** | `Update(msg) → (Model, Cmd)` | Reactive mutations |
| **Rendering** | `View() → string` | JSX components |
| **Async** | `tea.Cmd` (goroutines) | Promises/async-await |
| **Components** | Bubbles library | Built-in primitives |

---

## Current Implementation (Go)

### File Structure (~4,300 LOC)

```
internal/
├── ui/
│   ├── model.go      # 1,353 lines - State machine, 10 modes
│   └── view.go       #   845 lines - Rendering, Catppuccin theme
├── terminal/
│   └── pane.go       #   751 lines - PTY + vt10x (HARDEST PART)
├── agent/
│   ├── agent.go      #    81 lines - Config lookup
│   ├── status.go     #   243 lines - API/file status detection
│   └── context.go    #    63 lines - Prompt injection
├── board/
│   └── board.go      #   281 lines - Data model, JSON persistence
├── git/
│   └── worktree.go   #   181 lines - Worktree management
├── config/
│   └── config.go     #   276 lines - Configuration
└── app/
    └── app.go        #   156 lines - Entry point
```

### Features Implemented

- [x] Kanban board with 3 columns (Backlog, In Progress, Done)
- [x] Ticket CRUD with JSON persistence
- [x] Git worktree creation per ticket
- [x] Embedded PTY terminals running AI agents
- [x] Agent status detection (OpenCode API, file hooks)
- [x] Vim-style keyboard navigation (h/j/k/l/g/G)
- [x] Mouse support with drag-and-drop
- [x] 10 UI modes (Normal, Help, Confirm, Create, Edit, Agent, Settings, etc.)
- [x] Scrollback buffer (10,000 lines)
- [x] Catppuccin Mocha theme
- [x] Board settings UI

### The Hard Parts (Already Solved in Go)

1. **PTY Terminal Embedding** - `creack/pty` + `hinshun/vt10x`
2. **Key Translation** - Ctrl sequences, arrow keys → ANSI escapes
3. **Render Throttling** - 20fps to prevent flicker
4. **Agent Status Detection** - Multi-source with caching

---

## Phase 0: PTY Feasibility Spike (CRITICAL)

> **Do this BEFORE committing to full migration**

### Goal

Prove that embedded PTY terminals work in Bun/OpenTUI.

### Duration

2-4 hours

### Test Plan

#### Test 1: bun-pty Package

```bash
mkdir pty-spike && cd pty-spike
bun init -y
bun add bun-pty
```

```typescript
// spike-pty.ts
import { spawn } from "bun-pty"

const pty = spawn("bash", [], {
  name: "xterm-256color",
  cols: 80,
  rows: 24,
  cwd: process.cwd(),
})

pty.onData((data) => console.log("Output:", data))
pty.write("echo hello\r")

setTimeout(() => {
  pty.write("exit\r")
  pty.kill()
}, 3000)
```

```bash
bun run spike-pty.ts
```

#### Test 2: ghostty-opentui Rendering

```bash
bun add @opentui/core @opentui/solid ghostty-opentui solid-js
```

```typescript
// spike-terminal.tsx
import { render } from "@opentui/solid"
import { GhosttyTerminal } from "ghostty-opentui"

function App() {
  return (
    <box flexDirection="column" width="100%" height="100%">
      <text>OpenKanban Terminal Test</text>
      <GhosttyTerminal /* ... */ />
    </box>
  )
}

render(() => <App />)
```

#### Test 3: Node Subprocess Fallback (if bun-pty fails)

```typescript
// pty-worker.mjs (Node.js)
import pty from "@lydell/node-pty"

const term = pty.spawn("bash", [], {
  name: "xterm-256color",
  cols: 80,
  rows: 24,
})

process.on("message", (msg) => {
  if (msg.type === "input") term.write(msg.data)
  if (msg.type === "resize") term.resize(msg.cols, msg.rows)
})

term.onData((data) => process.send({ type: "output", data }))
```

```typescript
// main.ts (Bun)
const worker = Bun.spawn(["node", "pty-worker.mjs"], {
  ipc: (message) => handlePtyOutput(message),
})
```

### Decision Gate

| Result | Action |
|--------|--------|
| bun-pty works | Proceed with migration |
| bun-pty fails, Node subprocess works | Proceed with fallback pattern |
| Both fail | Abort migration, stay with Go |

---

## Phase 1: Project Setup (1-2 days)

### Goal

Monorepo scaffold with server and client packages.

### Directory Structure

```
openkanban/
├── package.json              # Workspace root
├── tsconfig.json             # Shared TS config
├── bunfig.toml
├── packages/
│   ├── shared/               # Shared types and utilities
│   │   ├── package.json
│   │   └── src/
│   │       ├── types.ts      # Board, Ticket, Message types
│   │       └── protocol.ts   # WebSocket message types
│   ├── server/               # Daemon server
│   │   ├── package.json
│   │   └── src/
│   │       ├── index.ts      # Entry point
│   │       ├── daemon.ts     # Daemon lifecycle
│   │       ├── api/          # Hono routes
│   │       ├── pty/          # PTY management
│   │       └── store/        # Board persistence
│   └── tui/                  # TUI client
│       ├── package.json
│       └── src/
│           ├── index.tsx     # Entry point
│           ├── App.tsx       # Root component
│           └── client/       # Server connection
├── bin/
│   └── openkanban.ts         # CLI entry point
└── .gitignore
```

### Dependencies

```json
// packages/server/package.json
{
  "dependencies": {
    "hono": "^4.0.0",
    "@xterm/headless": "^5.5.0",
    "@openkanban/shared": "workspace:*"
  }
}

// packages/tui/package.json
{
  "dependencies": {
    "@opentui/core": "^0.1.62",
    "@opentui/solid": "^0.1.62",
    "solid-js": "^1.9.0",
    "ghostty-opentui": "latest",
    "@openkanban/shared": "workspace:*"
  }
}
```

### Tasks

- [ ] Initialize Bun workspace
- [ ] Create shared, server, tui packages
- [ ] Define shared types (Board, Ticket, WebSocket protocol)
- [ ] Configure TypeScript (strict mode, paths)
- [ ] Create CLI entry point skeleton
- [ ] Verify workspace dependencies resolve

---

## Phase 2: Server Core (3-4 days)

### Goal

Daemon server with Hono HTTP/WebSocket, PTY management, and board persistence.

### Directory Structure

```
packages/server/src/
├── index.ts                # Entry point
├── daemon.ts               # Daemon lifecycle (PID, signals, hot reload)
├── api/
│   ├── index.ts            # Hono app setup
│   ├── routes/
│   │   ├── health.ts       # GET /api/health
│   │   ├── board.ts        # Board CRUD
│   │   └── agents.ts       # Agent control
│   └── websocket.ts        # WebSocket upgrade handler
├── pty/
│   ├── manager.ts          # PTY session lifecycle
│   ├── session.ts          # Single PTY session
│   └── emulator.ts         # @xterm/headless wrapper
├── store/
│   ├── board.ts            # Board state + persistence
│   └── persistence.ts      # Atomic JSON read/write
└── lib/
    └── slug.ts             # Slugify function
```

### Daemon Implementation

```typescript
// daemon.ts
import fs from "node:fs"
import path from "node:path"

const CONFIG_DIR = path.join(process.env.HOME!, ".openkanban")
const PID_FILE = path.join(CONFIG_DIR, "daemon.pid")
const DEFAULT_PORT = 4200

export class Daemon {
  private server?: ReturnType<typeof Bun.serve>
  private isShuttingDown = false

  async start(port = DEFAULT_PORT) {
    // Ensure config dir exists
    fs.mkdirSync(CONFIG_DIR, { recursive: true })
    
    // Single instance check
    if (this.isAlreadyRunning()) {
      throw new Error(`Daemon already running (PID ${this.getRunningPid()})`)
    }
    
    // Write PID
    fs.writeFileSync(PID_FILE, process.pid.toString())
    
    // Initialize services
    const app = createApp()
    
    // Start server
    this.server = Bun.serve({
      port,
      fetch: app.fetch,
      websocket: {
        message: this.handleWsMessage.bind(this),
        open: this.handleWsOpen.bind(this),
        close: this.handleWsClose.bind(this),
      },
    })
    
    // Signal handlers
    process.on("SIGTERM", () => this.shutdown())
    process.on("SIGINT", () => this.shutdown())
    
    console.log(`[openkanban] Daemon started on port ${port} (PID ${process.pid})`)
  }

  private isAlreadyRunning(): boolean {
    if (!fs.existsSync(PID_FILE)) return false
    const pid = parseInt(fs.readFileSync(PID_FILE, "utf-8"), 10)
    try {
      process.kill(pid, 0) // Check if process exists
      return true
    } catch {
      fs.unlinkSync(PID_FILE) // Stale PID file
      return false
    }
  }

  async shutdown() {
    if (this.isShuttingDown) return
    this.isShuttingDown = true
    
    console.log("[openkanban] Shutting down...")
    
    // Stop server
    this.server?.stop()
    
    // Close PTY sessions gracefully
    await ptyManager.closeAll()
    
    // Persist state
    await boardStore.flush()
    
    // Remove PID file
    if (fs.existsSync(PID_FILE)) {
      fs.unlinkSync(PID_FILE)
    }
    
    process.exit(0)
  }
}
```

### Hono App Setup

```typescript
// api/index.ts
import { Hono } from "hono"
import { cors } from "hono/cors"
import { healthRoutes } from "./routes/health"
import { boardRoutes } from "./routes/board"
import { agentRoutes } from "./routes/agents"

export function createApp() {
  const app = new Hono()
  
  // Middleware
  app.use("*", cors())
  
  // Routes
  app.route("/api/health", healthRoutes)
  app.route("/api/board", boardRoutes)
  app.route("/api/tickets", boardRoutes)
  app.route("/api/agents", agentRoutes)
  
  // WebSocket upgrade
  app.get("/ws", (c) => {
    const upgraded = c.env.upgrade(c.req.raw)
    if (!upgraded) {
      return c.text("WebSocket upgrade failed", 400)
    }
  })
  
  return app
}
```

### PTY Manager

```typescript
// pty/manager.ts
import { Terminal } from "@xterm/headless"

interface PtySession {
  id: string
  ticketId: string
  pty: ReturnType<typeof Bun.spawn>
  terminal: Terminal  // For screen buffer
  subscribers: Set<WebSocket>
}

class PtyManager {
  private sessions = new Map<string, PtySession>()

  async spawn(ticketId: string, command: string[], cwd: string): Promise<string> {
    const sessionId = crypto.randomUUID()
    
    const terminal = new Terminal({ cols: 80, rows: 24 })
    
    const pty = Bun.spawn(command, {
      cwd,
      env: { ...process.env, TERM: "xterm-256color" },
      stdin: "pipe",
      stdout: "pipe",
      stderr: "pipe",
    })
    
    // Stream output to terminal emulator and subscribers
    this.streamOutput(sessionId, pty, terminal)
    
    this.sessions.set(sessionId, {
      id: sessionId,
      ticketId,
      pty,
      terminal,
      subscribers: new Set(),
    })
    
    return sessionId
  }

  write(sessionId: string, data: string) {
    const session = this.sessions.get(sessionId)
    if (session?.pty.stdin) {
      session.pty.stdin.write(data)
    }
  }

  resize(sessionId: string, cols: number, rows: number) {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.terminal.resize(cols, rows)
      // Note: Bun.spawn doesn't support resize yet, may need Bun.Terminal
    }
  }

  subscribe(sessionId: string, ws: WebSocket) {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.subscribers.add(ws)
      // Send current screen buffer
      ws.send(JSON.stringify({
        type: "terminal:buffer",
        sessionId,
        data: this.getScreenBuffer(session.terminal),
      }))
    }
  }

  async closeAll() {
    for (const session of this.sessions.values()) {
      session.pty.kill()
    }
    this.sessions.clear()
  }
}

export const ptyManager = new PtyManager()
```

### Tasks

- [ ] Implement Daemon class (PID, signals, lifecycle)
- [ ] Create Hono app with CORS middleware
- [ ] Implement health check endpoint
- [ ] Implement WebSocket upgrade handler
- [ ] Create PtyManager with spawn/write/resize/close
- [ ] Integrate @xterm/headless for screen buffer
- [ ] Implement board persistence (atomic JSON write)
- [ ] Add CLI commands: `start`, `stop`, `status`
- [ ] Test daemon lifecycle (start, connect, disconnect, reconnect, stop)

---

## Phase 3: Data Layer (2-3 days)

### Goal

Shared types and board store with full CRUD.

### Directory Structure

```
packages/shared/src/
├── types.ts                # Board, Ticket, Column, Settings
├── protocol.ts             # WebSocket message types
└── index.ts                # Re-exports

packages/server/src/store/
├── board.ts                # BoardStore class
└── persistence.ts          # Atomic JSON read/write
```

### Types (Shared)

```typescript
// packages/shared/src/types.ts
export type TicketID = string

export type TicketStatus = "backlog" | "in_progress" | "done" | "archived"

export type AgentStatus = 
  | "none" 
  | "idle" 
  | "working" 
  | "waiting" 
  | "completed" 
  | "error"

export interface Ticket {
  id: TicketID
  title: string
  description?: string
  status: TicketStatus
  
  // Git integration
  worktreePath?: string
  branchName?: string
  baseBranch?: string
  
  // Agent integration
  agentType?: string
  agentStatus: AgentStatus
  agentSpawnedAt?: Date
  
  // Timestamps
  createdAt: Date
  updatedAt: Date
  startedAt?: Date
  completedAt?: Date
  
  // Metadata
  labels: string[]
  priority: number
  meta: Record<string, string>
}

export interface Column {
  id: string
  name: string
  key: TicketStatus
}

export interface Board {
  id: string
  name: string
  columns: Column[]
  tickets: Ticket[]
  settings: BoardSettings
}

export interface BoardSettings {
  defaultAgent: string
  worktreeBase: string
  baseBranch: string
  autoSpawn: boolean
}
```

### WebSocket Protocol (Shared)

```typescript
// packages/shared/src/protocol.ts

// Client → Server
export type ClientMessage = 
  | { type: "terminal:input", sessionId: string, data: string }
  | { type: "terminal:resize", sessionId: string, cols: number, rows: number }
  | { type: "terminal:subscribe", sessionId: string }
  | { type: "agent:spawn", ticketId: string }
  | { type: "agent:kill", ticketId: string }
  | { type: "board:subscribe" }
  | { type: "ticket:create", ticket: Partial<Ticket> }
  | { type: "ticket:update", ticketId: string, patch: Partial<Ticket> }
  | { type: "ticket:delete", ticketId: string }
  | { type: "ticket:move", ticketId: string, status: TicketStatus }

// Server → Client
export type ServerMessage =
  | { type: "terminal:output", sessionId: string, data: string }
  | { type: "terminal:buffer", sessionId: string, data: string }
  | { type: "terminal:exit", sessionId: string, code: number }
  | { type: "board:state", board: Board }
  | { type: "board:patch", patch: Partial<Board> }
  | { type: "ticket:created", ticket: Ticket }
  | { type: "ticket:updated", ticket: Ticket }
  | { type: "ticket:deleted", ticketId: string }
  | { type: "agent:status", ticketId: string, status: AgentStatus }
  | { type: "error", message: string }
```

### Tasks

- [ ] Define shared types (Board, Ticket, Column, Settings)
- [ ] Define WebSocket protocol types
- [ ] Implement BoardStore class with CRUD
- [ ] Implement atomic JSON persistence
- [ ] Port Slugify function
- [ ] Add board REST endpoints (GET, POST, PATCH, DELETE)
- [ ] Broadcast changes to WebSocket subscribers
- [ ] Ensure backward compatibility with existing .openkanban/ files

---

## Phase 4: TUI Client Shell (2-3 days)

### Goal

TUI client that connects to server and renders basic UI.

### Directory Structure

```
packages/tui/src/
├── index.tsx               # Entry point
├── App.tsx                 # Root component
├── client/
│   ├── connection.ts       # WebSocket connection manager
│   └── api.ts              # REST API client
├── stores/
│   ├── board.ts            # SolidJS store (synced from server)
│   └── ui.ts               # Local UI state (mode, selection)
├── components/
│   ├── Board.tsx           # Main board layout
│   ├── Column.tsx          # Single column
│   ├── Ticket.tsx          # Ticket card
│   ├── Header.tsx          # Logo, board name
│   └── StatusBar.tsx       # Mode, hints
└── theme/
    └── catppuccin.ts       # Color palette
```

### Connection Manager

```typescript
// client/connection.ts
import { createSignal, onCleanup } from "solid-js"
import type { ClientMessage, ServerMessage } from "@openkanban/shared"

export function createConnection(url: string) {
  const [connected, setConnected] = createSignal(false)
  const [board, setBoard] = createSignal<Board | null>(null)
  
  let ws: WebSocket | null = null
  let reconnectTimer: Timer | null = null
  
  function connect() {
    ws = new WebSocket(url)
    
    ws.onopen = () => {
      setConnected(true)
      send({ type: "board:subscribe" })
    }
    
    ws.onmessage = (event) => {
      const msg: ServerMessage = JSON.parse(event.data)
      handleMessage(msg)
    }
    
    ws.onclose = () => {
      setConnected(false)
      scheduleReconnect()
    }
  }
  
  function handleMessage(msg: ServerMessage) {
    switch (msg.type) {
      case "board:state":
        setBoard(msg.board)
        break
      case "ticket:created":
      case "ticket:updated":
      case "ticket:deleted":
        // Update local store
        break
      case "terminal:output":
        // Route to terminal component
        break
    }
  }
  
  function send(msg: ClientMessage) {
    ws?.send(JSON.stringify(msg))
  }
  
  function scheduleReconnect() {
    reconnectTimer = setTimeout(connect, 1000)
  }
  
  connect()
  
  onCleanup(() => {
    ws?.close()
    if (reconnectTimer) clearTimeout(reconnectTimer)
  })
  
  return { connected, board, send }
}
```

### Catppuccin Mocha Palette

```typescript
// theme/catppuccin.ts
export const colors = {
  base: "#1e1e2e",      // Background
  surface: "#313244",   // Elevated surfaces
  overlay: "#45475a",   // Borders, separators
  text: "#cdd6f4",      // Primary text
  subtext: "#a6adc8",   // Secondary text
  muted: "#6c7086",     // Disabled, hints
  blue: "#89b4fa",      // Primary accent, Backlog
  green: "#a6e3a1",     // Success, Done
  yellow: "#f9e2af",    // Warning, Working, In Progress
  red: "#f38ba8",       // Error
  mauve: "#cba6f7",     // Waiting, Settings
  teal: "#94e2d5",      // Key hints
  peach: "#fab387",     // Accent
}
```

### Tasks

- [ ] Create WebSocket connection manager with auto-reconnect
- [ ] Create REST API client for initial load
- [ ] Create SolidJS store synced from server
- [ ] Build root App component
- [ ] Implement connection status indicator
- [ ] Handle disconnection gracefully (show overlay)
- [ ] Test: start server, start TUI, see board state

---

## Phase 5: Board UI & Navigation (3-4 days)

### Goal

Complete board UI with keyboard navigation and modal system.

### Directory Structure

```
packages/tui/src/
├── components/
│   ├── Board.tsx           # Main board layout
│   ├── Column.tsx          # Column with header + tickets
│   ├── Ticket.tsx          # Ticket card
│   ├── Header.tsx          # Logo, board name, activity
│   ├── StatusBar.tsx       # Mode, hints, notifications
│   ├── HelpOverlay.tsx     # Help modal
│   └── ConfirmDialog.tsx   # Confirmation modal
├── hooks/
│   ├── useKeyboard.ts      # Keyboard event handling
│   └── useMode.ts          # Modal state machine
└── stores/
    └── ui.ts               # Mode, selection, scroll state
```

### Mode State Machine

```typescript
type Mode = 
  | "NORMAL"
  | "INSERT" 
  | "COMMAND"
  | "HELP"
  | "CONFIRM"
  | "CREATE_TICKET"
  | "EDIT_TICKET"
  | "AGENT_VIEW"
  | "SETTINGS"
```

### Keybindings

| Key | Normal Mode | Action |
|-----|-------------|--------|
| `h` / `l` | Move between columns | `setActiveColumn(+/-1)` |
| `j` / `k` | Move between tickets | `setActiveTicket(+/-1)` |
| `g` / `G` | First/last ticket | `setActiveTicket(0/last)` |
| `n` | New ticket | `setMode("CREATE_TICKET")` |
| `e` | Edit ticket | `setMode("EDIT_TICKET")` |
| `d` | Delete ticket | `setMode("CONFIRM")` |
| `s` | Spawn agent | Spawn agent on ticket |
| `S` | Stop agent | Kill agent process |
| `Enter` | Attach to agent | `setMode("AGENT_VIEW")` |
| `O` | Settings | `setMode("SETTINGS")` |
| `?` | Toggle help | `setMode("HELP")` |
| `q` | Quit | Exit application |
| `Esc` | Cancel/back | Return to NORMAL |

### UI Components

```typescript
// Board.tsx - Horizontal flexbox layout
<box flexDirection="row" width="100%" height="100%">
  <For each={columns()}>
    {(column) => <Column column={column} />}
  </For>
</box>

// Column.tsx - Vertical list with scroll
<box flexDirection="column" flex={1}>
  <ColumnHeader name={column.name} count={tickets().length} />
  <box flexDirection="column" overflow="scroll">
    <For each={tickets()}>
      {(ticket) => <Ticket ticket={ticket} selected={isSelected(ticket)} />}
    </For>
  </box>
  <ScrollIndicator hasMore={hasMoreBelow()} />
</box>

// Ticket.tsx - Card with status badge
<box borderStyle="rounded" padding={1}>
  <text color={colors.muted}>{ticket.id.slice(0, 8)}</text>
  <text bold>{ticket.title}</text>
  <AgentStatusBadge status={ticket.agentStatus} />
</box>
```

### Tasks

- [ ] Build Header component (logo, board name, activity badges)
- [ ] Build Column component (header, ticket list, scroll)
- [ ] Build Ticket card component (ID, title, status badge)
- [ ] Build Board layout (horizontal flexbox)
- [ ] Build StatusBar component (mode, hints)
- [ ] Implement mode state machine
- [ ] h/l - move between columns
- [ ] j/k - move between tickets
- [ ] g/G - first/last ticket
- [ ] ? - toggle help overlay
- [ ] Esc - return to normal mode
- [ ] Build HelpOverlay component
- [ ] Build ConfirmDialog component
- [ ] Implement scroll indicators

---

## Phase 6: Terminal Integration (3-4 days)

### Goal

Terminal display in TUI client, streaming from server via WebSocket.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          SERVER                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │ Bun.Terminal│───▶│  @xterm/    │───▶│    WebSocket        │  │
│  │   (PTY)     │    │  headless   │    │  (binary frames)    │  │
│  └─────────────┘    └─────────────┘    └──────────┬──────────┘  │
└───────────────────────────────────────────────────┼─────────────┘
                                                    │
                                                    ▼
┌───────────────────────────────────────────────────┼─────────────┐
│                          CLIENT                   │              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────┴───────────┐  │
│  │  ghostty-   │◀───│   Store     │◀───│    WebSocket        │  │
│  │  opentui    │    │ (terminal)  │    │    Client           │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
packages/tui/src/
├── components/
│   └── Terminal.tsx        # ghostty-opentui wrapper
├── stores/
│   └── terminals.ts        # Terminal sessions state
└── lib/
    └── keys.ts             # Key translation

packages/server/src/pty/
├── manager.ts              # PTY lifecycle
├── session.ts              # Single PTY + xterm/headless
└── stream.ts               # WebSocket streaming
```

### Terminal Component (Client)

```typescript
// components/Terminal.tsx
import { GhosttyTerminal } from "ghostty-opentui"
import { createEffect, onCleanup } from "solid-js"

interface TerminalProps {
  sessionId: string
  onInput: (data: string) => void
}

export function Terminal(props: TerminalProps) {
  let terminalRef: GhosttyTerminal | undefined
  
  // Subscribe to terminal output from server
  createEffect(() => {
    const unsubscribe = terminalStore.subscribe(props.sessionId, (data) => {
      terminalRef?.write(data)
    })
    onCleanup(unsubscribe)
  })
  
  // Handle resize
  createEffect(() => {
    const { cols, rows } = dimensions()
    connection.send({
      type: "terminal:resize",
      sessionId: props.sessionId,
      cols,
      rows,
    })
  })
  
  return (
    <GhosttyTerminal
      ref={terminalRef}
      onData={(data) => {
        connection.send({
          type: "terminal:input",
          sessionId: props.sessionId,
          data,
        })
      }}
    />
  )
}
```

### Key Translation

```typescript
// lib/keys.ts
export function translateKey(key: KeyEvent): string {
  // Ctrl+A-Z → 0x01-0x1A
  if (key.ctrl && key.key >= 'a' && key.key <= 'z') {
    return String.fromCharCode(key.key.charCodeAt(0) - 96)
  }
  
  // Arrow keys → ANSI sequences
  const arrowMap: Record<string, string> = {
    ArrowUp: "\x1b[A",
    ArrowDown: "\x1b[B",
    ArrowRight: "\x1b[C",
    ArrowLeft: "\x1b[D",
  }
  if (arrowMap[key.key]) return arrowMap[key.key]
  
  // Special keys
  if (key.key === "Enter") return "\r"
  if (key.key === "Backspace") return "\x7f"
  if (key.key === "Tab") return "\t"
  if (key.key === "Escape") return "\x1b"
  
  return key.key
}
```

### Agent View Mode

```typescript
// Full-screen terminal when viewing agent
function AgentView(props: { ticketId: string }) {
  const ticket = () => boardStore.getTicket(props.ticketId)
  const sessionId = () => ticket()?.terminalSessionId
  
  return (
    <box width="100%" height="100%">
      <Show when={sessionId()} fallback={<NoAgentMessage />}>
        <Terminal
          sessionId={sessionId()!}
          onInput={(data) => {
            connection.send({
              type: "terminal:input",
              sessionId: sessionId()!,
              data,
            })
          }}
        />
      </Show>
      <StatusBar hint="Ctrl+g to exit" />
    </box>
  )
}
```

### Tasks

- [ ] Create Terminal component using ghostty-opentui
- [ ] Create terminal sessions store
- [ ] Implement key translation utility
- [ ] Subscribe to terminal output via WebSocket
- [ ] Send terminal input via WebSocket
- [ ] Handle terminal resize
- [ ] Implement Agent View mode (fullscreen)
- [ ] Ctrl+g to exit agent view
- [ ] PgUp/PgDn for scrollback navigation
- [ ] Render throttling on client (~60fps)

---

## Phase 7: Agent Integration (2-3 days)

### Goal

Spawn agents in PTY sessions with context injection, monitor status.

### Directory Structure

```
packages/server/src/
├── agent/
│   ├── spawner.ts          # Agent spawning logic
│   ├── status.ts           # Status detection (API + files)
│   ├── context.ts          # Prompt template injection
│   └── config.ts           # Agent configurations
└── store/
    └── agents.ts           # Agent state per ticket
```

### Agent Config

```typescript
// Agent configuration (from project or global config)
interface AgentConfig {
  name: string
  command: string
  args: string[]
  env: Record<string, string>
  statusApi?: string        // e.g., "http://localhost:4096/session/status"
  statusFile?: string       // e.g., "~/.cache/{session}.status"
  initPrompt?: string       // Template with {{Title}}, {{Description}}, etc.
}

// Default agents
const defaultAgents: AgentConfig[] = [
  {
    name: "opencode",
    command: "opencode",
    args: [],
    statusApi: "http://localhost:4096/session/status",
  },
  {
    name: "claude",
    command: "claude",
    args: [],
  },
  {
    name: "aider",
    command: "aider",
    args: ["--yes"],
  },
]
```

### Agent Spawner (Server)

```typescript
// agent/spawner.ts
class AgentSpawner {
  async spawn(ticketId: string, agentName: string): Promise<string> {
    const ticket = boardStore.getTicket(ticketId)
    const config = this.getAgentConfig(agentName)
    
    // Build context prompt
    const contextPrompt = this.buildContextPrompt(ticket, config)
    
    // Spawn PTY session
    const sessionId = await ptyManager.spawn(
      ticketId,
      [config.command, ...config.args],
      ticket.worktreePath || process.cwd(),
    )
    
    // Inject initial prompt after shell ready
    if (contextPrompt) {
      setTimeout(() => {
        ptyManager.write(sessionId, contextPrompt + "\n")
      }, 500)
    }
    
    // Start status monitoring
    this.monitorStatus(ticketId, sessionId, config)
    
    return sessionId
  }
  
  private buildContextPrompt(ticket: Ticket, config: AgentConfig): string {
    if (!config.initPrompt) return ""
    
    return config.initPrompt
      .replace("{{Title}}", ticket.title)
      .replace("{{Description}}", ticket.description || "")
      .replace("{{Branch}}", ticket.branchName || "")
  }
}
```

### Status Detection Strategy

```typescript
// agent/status.ts
class StatusMonitor {
  private cache = new Map<string, { status: AgentStatus; timestamp: number }>()
  private CACHE_TTL = 500 // ms
  
  async getStatus(ticketId: string, config: AgentConfig): Promise<AgentStatus> {
    // Check cache
    const cached = this.cache.get(ticketId)
    if (cached && Date.now() - cached.timestamp < this.CACHE_TTL) {
      return cached.status
    }
    
    let status: AgentStatus = "none"
    
    // Strategy 1: API endpoint
    if (config.statusApi) {
      status = await this.checkApi(config.statusApi, ticketId)
    }
    
    // Strategy 2: Status file
    if (status === "none" && config.statusFile) {
      status = await this.checkFile(config.statusFile, ticketId)
    }
    
    // Strategy 3: Process state fallback
    if (status === "none") {
      const session = ptyManager.getSession(ticketId)
      status = session?.running ? "idle" : "none"
    }
    
    this.cache.set(ticketId, { status, timestamp: Date.now() })
    return status
  }
}
```

### Tasks

- [ ] Create AgentConfig type and loader
- [ ] Implement AgentSpawner with PTY integration
- [ ] Implement context prompt injection
- [ ] Create StatusMonitor with caching
- [ ] Add OpenCode API status detection
- [ ] Add file-based status detection
- [ ] Broadcast status changes via WebSocket
- [ ] Add agent spawn/kill REST endpoints
- [ ] Add agent status polling (configurable interval)

---

## Phase 8: Git Integration (2-3 days)

### Goal

Worktree creation and cleanup per ticket (server-side).

### Directory Structure

```
packages/server/src/
└── git/
    ├── worktree.ts         # Worktree operations
    ├── branch.ts           # Branch operations
    └── utils.ts            # Helpers
```

### Functions

```typescript
// git/worktree.ts
export async function createWorktree(
  repoPath: string,
  branchName: string,
  baseBranch: string,
  worktreeBase: string
): Promise<string>  // Returns worktree path

export async function removeWorktree(
  repoPath: string,
  worktreePath: string,
  force: boolean
): Promise<void>

export async function listWorktrees(repoPath: string): Promise<WorktreeInfo[]>

// git/branch.ts
export async function getDefaultBranch(repoPath: string): Promise<string>

export async function hasUncommittedChanges(worktreePath: string): Promise<boolean>

export function sanitizeBranchName(name: string): string

export async function branchExists(repoPath: string, branch: string): Promise<boolean>
```

### Worktree Lifecycle

```typescript
// Integrated with ticket lifecycle
class TicketWorkflow {
  async moveToInProgress(ticketId: string) {
    const ticket = boardStore.getTicket(ticketId)
    
    // Create worktree if needed
    if (!ticket.worktreePath) {
      const branchName = sanitizeBranchName(`task/${ticket.id}-${slugify(ticket.title)}`)
      const baseBranch = ticket.baseBranch || await getDefaultBranch(repoPath)
      
      const worktreePath = await createWorktree(
        repoPath,
        branchName,
        baseBranch,
        settings.worktreeBase
      )
      
      await boardStore.updateTicket(ticketId, {
        worktreePath,
        branchName,
        baseBranch,
        status: "in_progress",
        startedAt: new Date(),
      })
    }
    
    // Auto-spawn agent if enabled
    if (settings.autoSpawn) {
      await agentSpawner.spawn(ticketId, settings.defaultAgent)
    }
  }
  
  async deleteTicket(ticketId: string, force = false) {
    const ticket = boardStore.getTicket(ticketId)
    
    // Kill agent if running
    await agentSpawner.kill(ticketId)
    
    // Check for uncommitted changes
    if (ticket.worktreePath && !force) {
      if (await hasUncommittedChanges(ticket.worktreePath)) {
        throw new Error("Ticket has uncommitted changes. Use force to delete.")
      }
    }
    
    // Remove worktree
    if (ticket.worktreePath) {
      await removeWorktree(repoPath, ticket.worktreePath, force)
    }
    
    // Delete from store
    await boardStore.deleteTicket(ticketId)
  }
}
```

### Tasks

- [ ] Implement createWorktree (branch + worktree atomically)
- [ ] Implement removeWorktree (force cleanup)
- [ ] Implement getDefaultBranch detection
- [ ] Implement hasUncommittedChanges check
- [ ] Implement branch name sanitization
- [ ] Implement listWorktrees
- [ ] Integrate with ticket move workflow
- [ ] Integrate with ticket delete workflow
- [ ] Add REST endpoints for worktree operations

---

## Phase 9: Polish & CLI (3-4 days)

### Goal

Complete feature parity, CLI, and refinement.

### Directory Structure

```
packages/tui/src/
└── components/
    ├── TicketForm.tsx      # Create/edit ticket form
    ├── Settings.tsx        # Board settings UI
    ├── Toast.tsx           # Notification toasts
    └── ErrorBoundary.tsx   # Error handling

bin/
└── openkanban.ts           # CLI entry point
```

### CLI Commands

```typescript
// bin/openkanban.ts
import { Command } from "commander"

const program = new Command()
  .name("openkanban")
  .description("TUI kanban board for AI agents")
  .version("1.0.0")

// Start daemon + TUI (default)
program
  .command("start", { isDefault: true })
  .description("Start OpenKanban (daemon + TUI)")
  .option("-p, --port <port>", "Server port", "4200")
  .action(async (opts) => {
    await ensureDaemonRunning(opts.port)
    await startTui(opts.port)
  })

// Daemon-only mode
program
  .command("daemon")
  .description("Run daemon only (no TUI)")
  .option("-p, --port <port>", "Server port", "4200")
  .option("-f, --foreground", "Run in foreground")
  .action(async (opts) => {
    if (opts.foreground) {
      await runDaemonForeground(opts.port)
    } else {
      await startDaemonBackground(opts.port)
    }
  })

// Status
program
  .command("status")
  .description("Show daemon status")
  .action(async () => {
    const status = await getDaemonStatus()
    console.log(status.running 
      ? `Daemon running (PID ${status.pid}) on port ${status.port}`
      : "Daemon not running")
  })

// Stop daemon
program
  .command("stop")
  .description("Stop the daemon")
  .action(async () => {
    await stopDaemon()
    console.log("Daemon stopped")
  })

// Restart daemon
program
  .command("restart")
  .description("Restart the daemon")
  .action(async () => {
    await stopDaemon()
    await startDaemonBackground()
    console.log("Daemon restarted")
  })

program.parse()
```

### TUI Polish

```typescript
// components/TicketForm.tsx
function TicketForm(props: { mode: "create" | "edit", ticket?: Ticket }) {
  const [title, setTitle] = createSignal(props.ticket?.title || "")
  const [description, setDescription] = createSignal(props.ticket?.description || "")
  const [branch, setBranch] = createSignal(props.ticket?.branchName || "")
  
  const branchLocked = () => !!props.ticket?.worktreePath
  
  return (
    <box flexDirection="column" padding={1}>
      <Input label="Title" value={title()} onChange={setTitle} />
      <Input label="Description" value={description()} onChange={setDescription} multiline />
      <Input 
        label="Branch" 
        value={branch()} 
        onChange={setBranch}
        disabled={branchLocked()}
        hint={branchLocked() ? "(locked - worktree exists)" : ""}
      />
      <box flexDirection="row" gap={1}>
        <Button onClick={handleSubmit}>Save</Button>
        <Button onClick={handleCancel} variant="secondary">Cancel</Button>
      </box>
    </box>
  )
}
```

### Tasks

- [ ] Implement CLI with commander
- [ ] `openkanban` - start daemon + TUI (default)
- [ ] `openkanban daemon` - daemon only mode
- [ ] `openkanban status` - show daemon status
- [ ] `openkanban stop` - stop daemon
- [ ] `openkanban restart` - restart daemon
- [ ] Ticket create form (title, description, branch)
- [ ] Ticket edit form (with branch lock)
- [ ] Settings UI (default agent, worktree base, auto-spawn)
- [ ] Mouse click selection
- [ ] Mouse drag-and-drop between columns
- [ ] Toast notification system
- [ ] Error boundary and display
- [ ] Final styling/theme polish
- [ ] Backward compatibility testing

---

## Timeline Summary

| Phase | Duration | Risk | Dependencies |
|-------|----------|------|--------------|
| 0. PTY Spike | ✅ DONE | ~~Critical~~ | None |
| 1. Project Setup | 1-2 days | Low | Phase 0 |
| 2. Server Core | 3-4 days | Medium | Phase 1 |
| 3. Data Layer | 2-3 days | Low | Phase 2 |
| 4. TUI Client Shell | 2-3 days | Medium | Phase 2, 3 |
| 5. Board UI & Nav | 3-4 days | Low | Phase 4 |
| 6. Terminal Integration | 3-4 days | Medium | Phase 2, 5 |
| 7. Agent Integration | 2-3 days | Medium | Phase 6 |
| 8. Git Integration | 2-3 days | Low | Phase 3 |
| 9. Polish & CLI | 3-4 days | Low | Phase 7, 8 |

**Total Estimate: 4-6 weeks**

### Critical Path

```
Phase 0 (done)
    │
    ▼
Phase 1: Project Setup
    │
    ├──────────────────┐
    ▼                  ▼
Phase 2: Server    Phase 3: Data
    │                  │
    ├──────────────────┤
    ▼                  │
Phase 4: TUI Shell ◀───┘
    │
    ▼
Phase 5: Board UI
    │
    ▼
Phase 6: Terminal
    │
    ├──────────────────┐
    ▼                  ▼
Phase 7: Agents    Phase 8: Git
    │                  │
    └──────────────────┤
                       ▼
               Phase 9: Polish
```

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-12-19 | Explore OpenTUI rewrite | User preference, learning opportunity |
| 2025-12-19 | Keep Go version in git history | Fallback if migration fails |
| 2025-12-19 | PTY: Use `Bun.Terminal` + `@lydell/node-pty` fallback | Spike proved both work |
| 2025-12-19 | **Client/Server Daemon Architecture** | Agent survival, hot reload, multi-client support |

---

## References

### Frameworks & Libraries
- [OpenCode Repository](https://github.com/sst/opencode) - Reference implementation
- [OpenTUI](https://github.com/sst/opentui) - TUI framework
- [SolidJS](https://www.solidjs.com/) - Reactive UI framework
- [Hono](https://hono.dev/) - Fast HTTP framework for Bun
- [ghostty-opentui](https://www.npmjs.com/package/ghostty-opentui) - Terminal rendering

### PTY & Terminal
- [Bun.Terminal](https://bun.sh/docs/api/spawn#terminal) - Native Bun PTY (1.3.5+)
- [@lydell/node-pty](https://www.npmjs.com/package/@lydell/node-pty) - Cross-platform PTY fallback
- [@xterm/headless](https://www.npmjs.com/package/@xterm/headless) - Server-side terminal emulation

### Current Implementation
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Current Go framework
- [creack/pty](https://github.com/creack/pty) - Current Go PTY library

---

## Appendix: Planned Dependencies

### Workspace Root
```json
{
  "workspaces": ["packages/*"],
  "devDependencies": {
    "typescript": "^5.0.0",
    "@types/bun": "latest"
  }
}
```

### packages/shared
```json
{
  "name": "@openkanban/shared",
  "dependencies": {}
}
```

### packages/server
```json
{
  "name": "@openkanban/server",
  "dependencies": {
    "hono": "^4.0.0",
    "@xterm/headless": "^5.5.0",
    "@openkanban/shared": "workspace:*"
  }
}
```

### packages/tui
```json
{
  "name": "@openkanban/tui",
  "dependencies": {
    "@opentui/core": "^0.1.62",
    "@opentui/solid": "^0.1.62",
    "solid-js": "^1.9.0",
    "ghostty-opentui": "latest",
    "@openkanban/shared": "workspace:*"
  }
}
```

### bin/openkanban (CLI)
```json
{
  "dependencies": {
    "commander": "^12.0.0",
    "@openkanban/server": "workspace:*",
    "@openkanban/tui": "workspace:*"
  }
}
```
