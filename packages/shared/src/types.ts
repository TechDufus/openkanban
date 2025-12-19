export type TicketID = string
export type SessionID = string

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

  worktreePath?: string
  branchName?: string
  baseBranch?: string

  agentType?: string
  agentStatus: AgentStatus
  agentSpawnedAt?: string
  terminalSessionId?: SessionID

  createdAt: string
  updatedAt: string
  startedAt?: string
  completedAt?: string

  labels: string[]
  priority: number
  meta: Record<string, string>
}

export interface Column {
  id: string
  name: string
  key: TicketStatus
}

export interface BoardSettings {
  defaultAgent: string
  worktreeBase: string
  baseBranch: string
  autoSpawn: boolean
}

export interface Board {
  id: string
  name: string
  columns: Column[]
  tickets: Ticket[]
  settings: BoardSettings
}

export interface AgentConfig {
  name: string
  command: string
  args: string[]
  env: Record<string, string>
  statusApi?: string
  statusFile?: string
  initPrompt?: string
}

export const DEFAULT_COLUMNS: Column[] = [
  { id: "backlog", name: "Backlog", key: "backlog" },
  { id: "in_progress", name: "In Progress", key: "in_progress" },
  { id: "done", name: "Done", key: "done" },
]

export const DEFAULT_SETTINGS: BoardSettings = {
  defaultAgent: "opencode",
  worktreeBase: ".worktrees",
  baseBranch: "main",
  autoSpawn: false,
}

export function createTicket(partial: Partial<Ticket> & { title: string }): Ticket {
  const now = new Date().toISOString()
  return {
    id: partial.id ?? crypto.randomUUID(),
    title: partial.title,
    description: partial.description,
    status: partial.status ?? "backlog",
    agentStatus: partial.agentStatus ?? "none",
    createdAt: partial.createdAt ?? now,
    updatedAt: partial.updatedAt ?? now,
    startedAt: partial.startedAt,
    completedAt: partial.completedAt,
    worktreePath: partial.worktreePath,
    branchName: partial.branchName,
    baseBranch: partial.baseBranch,
    agentType: partial.agentType,
    agentSpawnedAt: partial.agentSpawnedAt,
    terminalSessionId: partial.terminalSessionId,
    labels: partial.labels ?? [],
    priority: partial.priority ?? 0,
    meta: partial.meta ?? {},
  }
}

export function createBoard(name: string): Board {
  return {
    id: crypto.randomUUID(),
    name,
    columns: [...DEFAULT_COLUMNS],
    tickets: [],
    settings: { ...DEFAULT_SETTINGS },
  }
}
