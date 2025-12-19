import type { ServerWebSocket } from "bun"
import { broadcastTerminalOutput, broadcastTerminalExit } from "../ws/handler"

interface BunTerminal {
  write(data: string | Uint8Array): void
  resize(cols: number, rows: number): void
  dispose(): void
}

interface BunTerminalOptions {
  cols: number
  rows: number
  data: (term: BunTerminal, data: Uint8Array) => void
  exit: (term: BunTerminal, exitCode: number) => void
}

interface PtySession {
  id: string
  ticketId: string
  terminal: BunTerminal
  process: ReturnType<typeof Bun.spawn>
  buffer: string[]
  subscribers: Set<ServerWebSocket<unknown>>
  cols: number
  rows: number
}

const MAX_BUFFER_LINES = 10000
const DEFAULT_COLS = 120
const DEFAULT_ROWS = 40

class PtyManager {
  private sessions = new Map<string, PtySession>()

  async spawn(ticketId: string, command: string[], cwd: string): Promise<string> {
    const sessionId = crypto.randomUUID()
    const [cmd = "bash", ...args] = command

    console.log(`[pty] Spawning: ${cmd} ${args.join(" ")} in ${cwd}`)

    const buffer: string[] = []
    const decoder = new TextDecoder()

    const BunTerminalClass = (Bun as unknown as { Terminal: new (opts: BunTerminalOptions) => BunTerminal }).Terminal

    const terminal = new BunTerminalClass({
      cols: DEFAULT_COLS,
      rows: DEFAULT_ROWS,
      data: (_term, data) => {
        const text = decoder.decode(data)
        this.appendToBuffer(buffer, text)
        broadcastTerminalOutput(sessionId, text)
      },
      exit: (_term, exitCode) => {
        console.log(`[pty] Session ${sessionId} exited with code ${exitCode}`)
        console.log(`[pty] Buffer contents: ${buffer.slice(-5).join("\\n")}`)
        broadcastTerminalExit(sessionId, exitCode)
        this.sessions.delete(sessionId)
      },
    })

    const proc = Bun.spawn([cmd, ...args], {
      terminal: terminal as unknown as undefined,
      cwd,
      env: process.env,
    })

    console.log(`[pty] PID: ${proc.pid}`)

    const session: PtySession = {
      id: sessionId,
      ticketId,
      terminal,
      process: proc,
      buffer,
      subscribers: new Set(),
      cols: DEFAULT_COLS,
      rows: DEFAULT_ROWS,
    }

    this.sessions.set(sessionId, session)

    console.log(`[pty] Spawned session ${sessionId} for ticket ${ticketId}: ${cmd} ${args.join(" ")}`)

    return sessionId
  }

  private appendToBuffer(buffer: string[], text: string): void {
    const lines = text.split("\n")
    buffer.push(...lines)

    if (buffer.length > MAX_BUFFER_LINES) {
      buffer.splice(0, buffer.length - MAX_BUFFER_LINES)
    }
  }

  write(sessionId: string, data: string): void {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.terminal.write(data)
    }
  }

  resize(sessionId: string, cols: number, rows: number): void {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.cols = cols
      session.rows = rows
      session.terminal.resize(cols, rows)
    }
  }

  subscribe(sessionId: string, ws: ServerWebSocket<unknown>): void {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.subscribers.add(ws)
    }
  }

  unsubscribe(sessionId: string, ws: ServerWebSocket<unknown>): void {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.subscribers.delete(ws)
    }
  }

  getBuffer(sessionId: string): string | undefined {
    const session = this.sessions.get(sessionId)
    if (session) {
      return session.buffer.join("\n")
    }
    return undefined
  }

  kill(sessionId: string): void {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.process.kill()
      session.terminal.dispose()
      this.sessions.delete(sessionId)
    }
  }

  getSession(ticketId: string): PtySession | undefined {
    for (const session of this.sessions.values()) {
      if (session.ticketId === ticketId) {
        return session
      }
    }
    return undefined
  }

  async closeAll(): Promise<void> {
    for (const session of this.sessions.values()) {
      session.process.kill()
      session.terminal.dispose()
    }
    this.sessions.clear()
  }
}

export const ptyManager = new PtyManager()
