import type { Ticket, TicketStatus } from "@openkanban/shared"
import { boardStore } from "./board"
import { agentSpawner } from "../agent/spawner"
import { createWorktree, removeWorktree } from "../git/worktree"
import { getDefaultBranch, hasUncommittedChanges, deleteBranch } from "../git/branch"
import { slugify } from "../git/utils"

function getRepoPath(): string {
  return process.cwd()
}

function getWorktreeBase(): string {
  return boardStore.getBoard()?.settings.worktreeBase ?? ".worktrees"
}

export async function moveTicketWithWorkflow(
  ticketId: string,
  newStatus: TicketStatus
): Promise<Ticket | undefined> {
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket) {
    return undefined
  }

  if (newStatus === "in_progress" && ticket.status !== "in_progress") {
    return await startTicketWork(ticketId)
  }

  return boardStore.moveTicket(ticketId, newStatus)
}

async function startTicketWork(ticketId: string): Promise<Ticket | undefined> {
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket) {
    return undefined
  }

  const settings = boardStore.getBoard()?.settings
  const repoPath = getRepoPath()
  const worktreeBase = getWorktreeBase()

  if (!ticket.worktreePath) {
    const baseBranch = ticket.baseBranch || await getDefaultBranch(repoPath)
    const branchName = `task/${slugify(ticket.title)}-${ticket.id.slice(0, 8)}`

    try {
      const worktreePath = await createWorktree(
        repoPath,
        branchName,
        baseBranch,
        worktreeBase
      )

      boardStore.updateTicket(ticketId, {
        worktreePath,
        branchName,
        baseBranch,
        status: "in_progress",
        startedAt: new Date().toISOString(),
      })
    } catch (error) {
      boardStore.updateTicket(ticketId, {
        status: "in_progress",
        startedAt: new Date().toISOString(),
      })
      console.error("[workflow] Failed to create worktree:", error)
    }
  } else {
    boardStore.updateTicket(ticketId, {
      status: "in_progress",
      startedAt: new Date().toISOString(),
    })
  }

  if (settings?.autoSpawn && !agentSpawner.isRunning(ticketId)) {
    const agentType = ticket.agentType || settings.defaultAgent
    try {
      await agentSpawner.spawn(ticketId, agentType)
    } catch (error) {
      console.error("[workflow] Failed to auto-spawn agent:", error)
    }
  }

  return boardStore.getTicket(ticketId)
}

export async function deleteTicketWithWorkflow(
  ticketId: string,
  force = false
): Promise<boolean> {
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket) {
    return false
  }

  await agentSpawner.kill(ticketId)

  if (ticket.worktreePath) {
    if (!force) {
      try {
        const hasChanges = await hasUncommittedChanges(ticket.worktreePath)
        if (hasChanges) {
          throw new Error("Ticket has uncommitted changes. Use force to delete.")
        }
      } catch (error) {
        if (error instanceof Error && error.message.includes("uncommitted")) {
          throw error
        }
      }
    }

    try {
      await removeWorktree(getRepoPath(), ticket.worktreePath, force)
    } catch (error) {
      console.error("[workflow] Failed to remove worktree:", error)
    }

    if (ticket.branchName && force) {
      try {
        await deleteBranch(getRepoPath(), ticket.branchName, true)
      } catch {
      }
    }
  }

  return boardStore.deleteTicket(ticketId)
}

export async function cleanupTicketWorktree(ticketId: string): Promise<void> {
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket?.worktreePath) {
    return
  }

  try {
    await removeWorktree(getRepoPath(), ticket.worktreePath, true)
    boardStore.updateTicket(ticketId, {
      worktreePath: undefined,
    })
  } catch (error) {
    console.error("[workflow] Failed to cleanup worktree:", error)
  }
}
