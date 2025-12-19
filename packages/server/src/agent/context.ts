import type { Ticket, AgentConfig } from "@openkanban/shared"

interface ContextData {
  Title: string
  Description: string
  BranchName: string
  BaseBranch: string
  TicketID: string
  Status: string
  WorktreePath: string
}

export function buildContextPrompt(ticket: Ticket, config: AgentConfig): string {
  const template = config.initPrompt
  if (!template) {
    return ""
  }

  const data: ContextData = {
    Title: ticket.title,
    Description: ticket.description ?? "",
    BranchName: ticket.branchName ?? "",
    BaseBranch: ticket.baseBranch ?? "",
    TicketID: ticket.id,
    Status: ticket.status,
    WorktreePath: ticket.worktreePath ?? "",
  }

  let result = template
  for (const [key, value] of Object.entries(data)) {
    result = result.replaceAll(`{{${key}}}`, value)
  }

  result = result.replace(/\n{3,}/g, "\n\n").trim()

  return result
}

export function shouldInjectContext(ticket: Ticket): boolean {
  return !ticket.agentSpawnedAt
}
