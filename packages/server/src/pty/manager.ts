import type { ServerWebSocket } from "bun"
import { broadcastTerminalOutput, broadcastTerminalExit } from "../ws/handler"

interface PtySession {
  id: string
  ticketId: string
  proc: ReturnType<typeof Bun.spawn>
  buffer: string[]
  subscribers: Set<ServerWebSocket<unknown>>
  cols: number
  rows: number
}

const MAX_BUFFER_LINES = 10000

class PtyManager {
  private sessions = new Map<string, PtySession>()

  async spawn(ticketId: string, command: string[], cwd: string): Promise<string> {
    const sessionId = crypto.randomUUID()

    const proc = Bun.spawn(command, {
      cwd,
      env: { ...process.env, TERM: "xterm-256color" },
      stdin: "pipe",
      stdout: "pipe",
      stderr: "pipe",
    })

    const session: PtySession = {
      id: sessionId,
      ticketId,
      proc,
      buffer: [],
      subscribers: new Set(),
      cols: 80,
      rows: 24,
    }

    this.sessions.set(sessionId, session)

    this.streamOutput(session)

    proc.exited.then((code) => {
      broadcastTerminalExit(sessionId, code)
      this.sessions.delete(sessionId)
    })

    return sessionId
  }

  private async streamOutput(session: PtySession): Promise<void> {
    const { proc, id } = session

    const stdout = proc.stdout
    if (stdout && typeof stdout !== "number") {
      const reader = stdout.getReader()
      const decoder = new TextDecoder()

      const readLoop = async () => {
        try {
          while (true) {
            const { done, value } = await reader.read()
            if (done) break

            const text = decoder.decode(value)
            this.appendToBuffer(session, text)
            broadcastTerminalOutput(id, text)
          }
        } catch {
        }
      }

      readLoop()
    }

    const stderr = proc.stderr
    if (stderr && typeof stderr !== "number") {
      const reader = stderr.getReader()
      const decoder = new TextDecoder()

      const readLoop = async () => {
        try {
          while (true) {
            const { done, value } = await reader.read()
            if (done) break

            const text = decoder.decode(value)
            this.appendToBuffer(session, text)
            broadcastTerminalOutput(id, text)
          }
        } catch {
        }
      }

      readLoop()
    }
  }

  private appendToBuffer(session: PtySession, text: string): void {
    const lines = text.split("\n")
    session.buffer.push(...lines)

    if (session.buffer.length > MAX_BUFFER_LINES) {
      session.buffer = session.buffer.slice(-MAX_BUFFER_LINES)
    }
  }

  write(sessionId: string, data: string): void {
    const session = this.sessions.get(sessionId)
    const stdin = session?.proc.stdin
    if (stdin && typeof stdin !== "number") {
      stdin.write(data)
    }
  }

  resize(sessionId: string, cols: number, rows: number): void {
    const session = this.sessions.get(sessionId)
    if (session) {
      session.cols = cols
      session.rows = rows
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
      session.proc.kill()
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
      session.proc.kill()
    }
    this.sessions.clear()
  }
}

export const ptyManager = new PtyManager()
