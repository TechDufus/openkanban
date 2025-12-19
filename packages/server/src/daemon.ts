import fs from "node:fs"
import path from "node:path"
import { createApp } from "./api"
import { handleWsOpen, handleWsClose, handleWsMessage } from "./ws/handler"
import { boardStore } from "./store/board"
import { ptyManager } from "./pty/manager"

const CONFIG_DIR = path.join(process.env["HOME"] ?? "~", ".openkanban")
const PID_FILE = path.join(CONFIG_DIR, "daemon.pid")

export class Daemon {
  private server?: ReturnType<typeof Bun.serve>
  private isShuttingDown = false
  private port = 4200

  async start(port = 4200): Promise<void> {
    this.port = port

    fs.mkdirSync(CONFIG_DIR, { recursive: true })

    if (this.isAlreadyRunning()) {
      const runningPid = this.getRunningPid()
      throw new Error(`Daemon already running (PID ${runningPid})`)
    }

    fs.writeFileSync(PID_FILE, process.pid.toString())

    const app = createApp()

    this.server = Bun.serve({
      port,
      fetch: (req, server) => {
        if (req.headers.get("upgrade") === "websocket") {
          const success = server.upgrade(req, {
            data: {
              id: "",
              subscribedToBoard: false,
              subscribedTerminals: new Set<string>(),
            },
          })
          if (success) return undefined
          return new Response("WebSocket upgrade failed", { status: 400 })
        }
        return app.fetch(req)
      },
      websocket: {
        message: handleWsMessage,
        open: handleWsOpen,
        close: handleWsClose,
      },
    })

    process.on("SIGTERM", () => this.shutdown())
    process.on("SIGINT", () => this.shutdown())

    console.log(`[openkanban] Daemon started on port ${port} (PID ${process.pid})`)
  }

  private isAlreadyRunning(): boolean {
    if (!fs.existsSync(PID_FILE)) return false

    const pid = this.getRunningPid()
    if (pid === null) {
      fs.unlinkSync(PID_FILE)
      return false
    }

    try {
      process.kill(pid, 0)
      return true
    } catch {
      fs.unlinkSync(PID_FILE)
      return false
    }
  }

  private getRunningPid(): number | null {
    try {
      const content = fs.readFileSync(PID_FILE, "utf-8")
      const pid = parseInt(content.trim(), 10)
      return isNaN(pid) ? null : pid
    } catch {
      return null
    }
  }

  async shutdown(): Promise<void> {
    if (this.isShuttingDown) return
    this.isShuttingDown = true

    console.log("[openkanban] Shutting down...")

    this.server?.stop()

    await ptyManager.closeAll()
    boardStore.flush()

    if (fs.existsSync(PID_FILE)) {
      fs.unlinkSync(PID_FILE)
    }

    process.exit(0)
  }

  getPort(): number {
    return this.port
  }

  static isRunning(): { running: boolean; pid?: number } {
    if (!fs.existsSync(PID_FILE)) {
      return { running: false }
    }

    try {
      const content = fs.readFileSync(PID_FILE, "utf-8")
      const pid = parseInt(content.trim(), 10)
      if (isNaN(pid)) {
        return { running: false }
      }
      process.kill(pid, 0)
      return { running: true, pid }
    } catch {
      return { running: false }
    }
  }

  static async stop(): Promise<boolean> {
    const status = Daemon.isRunning()
    if (!status.running || !status.pid) {
      return false
    }

    process.kill(status.pid, "SIGTERM")

    const start = Date.now()
    const timeout = 5000
    while (Date.now() - start < timeout) {
      try {
        process.kill(status.pid, 0)
        await Bun.sleep(100)
      } catch {
        if (fs.existsSync(PID_FILE)) {
          fs.unlinkSync(PID_FILE)
        }
        return true
      }
    }

    try {
      process.kill(status.pid, "SIGKILL")
    } catch {
    }

    if (fs.existsSync(PID_FILE)) {
      fs.unlinkSync(PID_FILE)
    }
    return true
  }
}
