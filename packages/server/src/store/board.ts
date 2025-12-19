import fs from "node:fs"
import path from "node:path"
import {
  createBoard,
  createTicket,
  DEFAULT_COLUMNS,
  DEFAULT_SETTINGS,
  type Board,
  type Ticket,
  type TicketStatus,
} from "@openkanban/shared"

const CONFIG_DIR = path.join(process.env["HOME"] ?? "~", ".openkanban")
const BOARD_FILE = path.join(CONFIG_DIR, "board.json")

class BoardStore {
  private board: Board
  private saveTimeout: Timer | null = null
  private readonly SAVE_DEBOUNCE_MS = 500

  constructor() {
    this.board = this.load()
  }

  private load(): Board {
    try {
      if (fs.existsSync(BOARD_FILE)) {
        const data = fs.readFileSync(BOARD_FILE, "utf-8")
        const parsed = JSON.parse(data) as Board
        
        if (!parsed.columns || parsed.columns.length === 0) {
          parsed.columns = [...DEFAULT_COLUMNS]
        }
        if (!parsed.settings) {
          parsed.settings = { ...DEFAULT_SETTINGS }
        }
        if (!parsed.tickets) {
          parsed.tickets = []
        }
        
        return parsed
      }
    } catch (err) {
      console.error("[store] Failed to load board:", err)
    }
    
    return createBoard("OpenKanban")
  }

  private scheduleSave(): void {
    if (this.saveTimeout) {
      clearTimeout(this.saveTimeout)
    }
    this.saveTimeout = setTimeout(() => this.flush(), this.SAVE_DEBOUNCE_MS)
  }

  flush(): void {
    try {
      fs.mkdirSync(CONFIG_DIR, { recursive: true })
      
      const tempFile = `${BOARD_FILE}.tmp`
      fs.writeFileSync(tempFile, JSON.stringify(this.board, null, 2))
      fs.renameSync(tempFile, BOARD_FILE)
    } catch (err) {
      console.error("[store] Failed to save board:", err)
    }
  }

  getBoard(): Board {
    return this.board
  }

  getTicket(ticketId: string): Ticket | undefined {
    return this.board.tickets.find((t) => t.id === ticketId)
  }

  createTicket(data: Partial<Ticket> & { title: string }): Ticket {
    const ticket = createTicket(data)
    this.board.tickets.push(ticket)
    this.scheduleSave()
    return ticket
  }

  updateTicket(ticketId: string, patch: Partial<Ticket>): Ticket | undefined {
    const index = this.board.tickets.findIndex((t) => t.id === ticketId)
    if (index === -1) return undefined

    const existing = this.board.tickets[index]
    if (!existing) return undefined

    const updated: Ticket = {
      ...existing,
      ...patch,
      id: existing.id,
      createdAt: existing.createdAt,
      updatedAt: new Date().toISOString(),
    }

    this.board.tickets[index] = updated
    this.scheduleSave()
    return updated
  }

  deleteTicket(ticketId: string): boolean {
    const index = this.board.tickets.findIndex((t) => t.id === ticketId)
    if (index === -1) return false

    this.board.tickets.splice(index, 1)
    this.scheduleSave()
    return true
  }

  moveTicket(ticketId: string, status: TicketStatus): Ticket | undefined {
    const ticket = this.getTicket(ticketId)
    if (!ticket) return undefined

    const patch: Partial<Ticket> = { status }

    if (status === "in_progress" && !ticket.startedAt) {
      patch.startedAt = new Date().toISOString()
    }
    if (status === "done" && !ticket.completedAt) {
      patch.completedAt = new Date().toISOString()
    }

    return this.updateTicket(ticketId, patch)
  }

  updateSettings(settings: Partial<Board["settings"]>): void {
    this.board.settings = { ...this.board.settings, ...settings }
    this.scheduleSave()
  }
}

export const boardStore = new BoardStore()
