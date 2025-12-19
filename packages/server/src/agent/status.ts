import type { AgentStatus, AgentConfig } from "@openkanban/shared"
import os from "node:os"
import path from "node:path"
import fs from "node:fs"

interface CachedStatus {
  status: AgentStatus
  timestamp: number
}

interface OpencodeSessionStatus {
  type: string
  attempt?: number
  message?: string
  next?: number
}

type OpencodeStatusResponse = Record<string, OpencodeSessionStatus>

class StatusMonitor {
  private cache = new Map<string, CachedStatus>()
  private readonly CACHE_TTL_MS = 500
  private readonly API_TIMEOUT_MS = 2000
  private readonly statusDir: string

  constructor() {
    this.statusDir = path.join(os.homedir(), ".cache", "openkanban-status")
  }

  async getStatus(
    sessionId: string,
    config: AgentConfig,
    processRunning: boolean
  ): Promise<AgentStatus> {
    if (!processRunning) {
      return "none"
    }

    const cacheKey = `${config.name}:${sessionId}`
    const cached = this.cache.get(cacheKey)
    if (cached && Date.now() - cached.timestamp < this.CACHE_TTL_MS) {
      return cached.status
    }

    let status: AgentStatus = "none"

    const fileStatus = this.readStatusFile(sessionId)
    if (fileStatus !== "none") {
      status = fileStatus
    }

    if (status === "none" && config.statusApi && config.name === "opencode") {
      const apiStatus = await this.queryOpencodeApi(config.statusApi, sessionId)
      if (apiStatus !== "none") {
        status = apiStatus
      }
    }

    if (status === "none") {
      status = "idle"
    }

    this.cache.set(cacheKey, { status, timestamp: Date.now() })

    return status
  }

  private readStatusFile(sessionId: string): AgentStatus {
    const statusFile = path.join(this.statusDir, `${sessionId}.status`)

    try {
      const content = fs.readFileSync(statusFile, "utf-8").trim()
      
      switch (content) {
        case "working":
          return "working"
        case "done":
        case "idle":
          return "idle"
        case "waiting":
        case "permission":
          return "waiting"
        case "error":
          return "error"
        case "completed":
          return "completed"
        default:
          return "none"
      }
    } catch {
      return "none"
    }
  }

  private async queryOpencodeApi(
    apiUrl: string,
    sessionId: string
  ): Promise<AgentStatus> {
    try {
      const controller = new AbortController()
      const timeout = setTimeout(() => controller.abort(), this.API_TIMEOUT_MS)

      const response = await fetch(apiUrl, {
        signal: controller.signal,
      })
      clearTimeout(timeout)

      if (!response.ok) {
        return "none"
      }

      const data = (await response.json()) as OpencodeStatusResponse
      const sessionStatus = data[sessionId]

      if (!sessionStatus) {
        return "none"
      }

      return this.mapOpencodeStatus(sessionStatus)
    } catch {
      return "none"
    }
  }

  private mapOpencodeStatus(status: OpencodeSessionStatus): AgentStatus {
    switch (status.type) {
      case "busy":
        return "working"
      case "idle":
        return "idle"
      case "retry":
        return "error"
      default:
        return "none"
    }
  }

  invalidateCache(sessionId?: string): void {
    if (sessionId) {
      for (const key of this.cache.keys()) {
        if (key.endsWith(`:${sessionId}`)) {
          this.cache.delete(key)
        }
      }
    } else {
      this.cache.clear()
    }
  }

  writeStatusFile(sessionId: string, status: AgentStatus): void {
    try {
      fs.mkdirSync(this.statusDir, { recursive: true })

      const statusFile = path.join(this.statusDir, `${sessionId}.status`)
      let statusStr: string

      switch (status) {
        case "working":
          statusStr = "working"
          break
        case "idle":
          statusStr = "idle"
          break
        case "waiting":
          statusStr = "waiting"
          break
        case "completed":
          statusStr = "completed"
          break
        case "error":
          statusStr = "error"
          break
        default:
          statusStr = "idle"
      }

      fs.writeFileSync(statusFile, statusStr + "\n")
    } catch {
      return
    }
  }

  cleanupStatusFile(sessionId: string): void {
    try {
      const statusFile = path.join(this.statusDir, `${sessionId}.status`)
      fs.unlinkSync(statusFile)
    } catch {
      return
    }
  }
}

export const statusMonitor = new StatusMonitor()
