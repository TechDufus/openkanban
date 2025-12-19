import type { AgentStatus } from "@openkanban/shared"
import { ptyManager } from "../pty/manager"
import { boardStore } from "../store/board"
import { agentConfigManager } from "./config"
import { buildContextPrompt, shouldInjectContext } from "./context"
import { statusMonitor } from "./status"

const CONTEXT_INJECTION_DELAY_MS = 500
const STATUS_POLL_INTERVAL_MS = 2000

interface AgentSession {
  ticketId: string
  sessionId: string
  agentName: string
  lastStatus: AgentStatus
}

class AgentSpawner {
  private sessions = new Map<string, AgentSession>()
  private statusPollingInterval: ReturnType<typeof setInterval> | null = null
  private statusChangeCallbacks: Set<(ticketId: string, status: AgentStatus) => void> = new Set()

  async spawn(ticketId: string, agentName?: string): Promise<string> {
    const ticket = boardStore.getTicket(ticketId)
    if (!ticket) {
      throw new Error(`Ticket not found: ${ticketId}`)
    }

    const existingSession = this.sessions.get(ticketId)
    if (existingSession) {
      return existingSession.sessionId
    }

    const resolvedAgentName = agentName ?? ticket.agentType ?? boardStore.getBoard()?.settings.defaultAgent ?? "opencode"
    const config = agentConfigManager.getOrDefault(resolvedAgentName)

    const cwd = ticket.worktreePath || process.cwd()
    const sessionId = await ptyManager.spawn(ticketId, [config.command, ...config.args], cwd)

    const session: AgentSession = {
      ticketId,
      sessionId,
      agentName: config.name,
      lastStatus: "idle",
    }
    this.sessions.set(ticketId, session)

    boardStore.updateTicket(ticketId, {
      agentType: config.name,
      agentStatus: "idle",
      agentSpawnedAt: new Date().toISOString(),
      terminalSessionId: sessionId,
    })

    if (shouldInjectContext(ticket)) {
      const contextPrompt = buildContextPrompt(ticket, config)
      if (contextPrompt) {
        setTimeout(() => {
          ptyManager.write(sessionId, contextPrompt + "\n")
        }, CONTEXT_INJECTION_DELAY_MS)
      }
    }

    this.ensureStatusPolling()

    return sessionId
  }

  async kill(ticketId: string): Promise<void> {
    const session = this.sessions.get(ticketId)
    if (!session) {
      return
    }

    ptyManager.kill(session.sessionId)
    statusMonitor.cleanupStatusFile(session.sessionId)
    this.sessions.delete(ticketId)

    boardStore.updateTicket(ticketId, {
      agentStatus: "none",
      terminalSessionId: undefined,
    })
  }

  getSession(ticketId: string): AgentSession | undefined {
    return this.sessions.get(ticketId)
  }

  isRunning(ticketId: string): boolean {
    return this.sessions.has(ticketId)
  }

  onStatusChange(callback: (ticketId: string, status: AgentStatus) => void): () => void {
    this.statusChangeCallbacks.add(callback)
    return () => this.statusChangeCallbacks.delete(callback)
  }

  private ensureStatusPolling(): void {
    if (this.statusPollingInterval) {
      return
    }

    this.statusPollingInterval = setInterval(() => {
      this.pollAllStatuses()
    }, STATUS_POLL_INTERVAL_MS)
  }

  private async pollAllStatuses(): Promise<void> {
    for (const [ticketId, session] of this.sessions) {
      const config = agentConfigManager.getOrDefault(session.agentName)
      const ptySession = ptyManager.getSession(ticketId)
      const processRunning = ptySession !== undefined

      const status = await statusMonitor.getStatus(session.sessionId, config, processRunning)

      if (status !== session.lastStatus) {
        session.lastStatus = status
        boardStore.updateTicket(ticketId, { agentStatus: status })

        for (const callback of this.statusChangeCallbacks) {
          callback(ticketId, status)
        }
      }

      if (!processRunning) {
        this.sessions.delete(ticketId)
        statusMonitor.cleanupStatusFile(session.sessionId)
      }
    }

    if (this.sessions.size === 0 && this.statusPollingInterval) {
      clearInterval(this.statusPollingInterval)
      this.statusPollingInterval = null
    }
  }

  async shutdown(): Promise<void> {
    if (this.statusPollingInterval) {
      clearInterval(this.statusPollingInterval)
      this.statusPollingInterval = null
    }

    for (const [ticketId] of this.sessions) {
      await this.kill(ticketId)
    }
  }
}

export const agentSpawner = new AgentSpawner()
