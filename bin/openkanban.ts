#!/usr/bin/env bun

import { spawn } from "node:child_process"
import fs from "node:fs"
import path from "node:path"
import { Daemon } from "@openkanban/server"
import { startTui } from "@openkanban/tui"

const DEFAULT_PORT = 4200

type Command = "start" | "daemon" | "status" | "stop" | "restart" | "help"

function parseArgs(): { command: Command; port: number; foreground: boolean } {
  const args = process.argv.slice(2)
  let command: Command = "start"
  let port = DEFAULT_PORT
  let foreground = false

  for (let i = 0; i < args.length; i++) {
    const arg = args[i]
    if (arg === "daemon") command = "daemon"
    else if (arg === "status") command = "status"
    else if (arg === "stop") command = "stop"
    else if (arg === "restart") command = "restart"
    else if (arg === "help" || arg === "--help" || arg === "-h") command = "help"
    else if (arg === "-p" || arg === "--port") {
      const nextArg = args[++i]
      if (nextArg) port = parseInt(nextArg, 10)
    }
    else if (arg === "-f" || arg === "--foreground") foreground = true
  }

  return { command, port, foreground }
}

async function ensureDaemonRunning(port: number): Promise<void> {
  const status = Daemon.isRunning()
  if (status.running) {
    console.log(`[openkanban] Daemon already running (PID ${status.pid})`)
    return
  }

  console.log("[openkanban] Starting daemon...")

  const serverPath = path.resolve(import.meta.dir, "../packages/server/src/index.ts")
  const logDir = path.join(process.env["HOME"] ?? "~", ".openkanban")
  fs.mkdirSync(logDir, { recursive: true })
  const logFile = fs.openSync(path.join(logDir, "daemon.log"), "a")

  const child = spawn("bun", ["run", serverPath], {
    detached: true,
    stdio: ["ignore", logFile, logFile],
    env: { ...process.env, PORT: String(port) },
  })

  child.unref()
  fs.closeSync(logFile)

  await Bun.sleep(500)

  const newStatus = Daemon.isRunning()
  if (newStatus.running) {
    console.log(`[openkanban] Daemon started (PID ${newStatus.pid})`)
  } else {
    console.error("[openkanban] Failed to start daemon")
    process.exit(1)
  }
}

async function runCommand(cmd: Command, port: number, foreground: boolean): Promise<void> {
  switch (cmd) {
    case "start": {
      await ensureDaemonRunning(port)
      startTui(port)
      break
    }

    case "daemon": {
      if (foreground) {
        const daemon = new Daemon()
        await daemon.start(port)
      } else {
        await ensureDaemonRunning(port)
      }
      break
    }

    case "status": {
      const status = Daemon.isRunning()
      if (status.running) {
        console.log(`Daemon running (PID ${status.pid})`)

        try {
          const res = await fetch(`http://localhost:${port}/api/health`)
          const health = await res.json()
          console.log(`  Port: ${port}`)
          console.log(`  Uptime: ${Math.round(health.uptime)}s`)
        } catch {
          console.log(`  Port: ${port} (health check failed)`)
        }
      } else {
        console.log("Daemon not running")
      }
      break
    }

    case "stop": {
      const stopped = await Daemon.stop()
      if (stopped) {
        console.log("Daemon stopped")
      } else {
        console.log("Daemon was not running")
      }
      break
    }

    case "restart": {
      await Daemon.stop()
      await Bun.sleep(500)
      await ensureDaemonRunning(port)
      console.log("Daemon restarted")
      break
    }

    case "help": {
      console.log(`
OpenKanban - TUI kanban board for AI agents

Usage:
  openkanban [command] [options]

Commands:
  start         Start daemon and TUI (default)
  daemon        Run daemon only
  status        Show daemon status
  stop          Stop the daemon
  restart       Restart the daemon
  help          Show this help

Options:
  -p, --port <port>    Server port (default: 4200)
  -f, --foreground     Run daemon in foreground (with daemon command)

Examples:
  openkanban                    Start with TUI
  openkanban daemon -f          Run daemon in foreground
  openkanban status             Check if daemon is running
  openkanban stop               Stop the daemon
`)
      break
    }
  }
}

const { command, port, foreground } = parseArgs()
await runCommand(command, port, foreground)
