import { useKeyboard } from "@opentui/solid"
import type { KeyEvent } from "@opentui/core"
import { useBoard } from "../stores/board"
import { useUI } from "../stores/ui"

export interface NavigationActions {
  onQuit?: () => void
  onCreateTicket?: () => void
  onDeleteTicket?: (ticketId: string) => void
  onSpawnAgent?: (ticketId: string) => void
  onKillAgent?: (ticketId: string) => void
}

export function useNavigation(actions: NavigationActions = {}) {
  const { board, getTicketsByStatus } = useBoard()
  const {
    mode,
    setMode,
    activeColumn,
    setActiveColumn,
    activeTicket,
    setActiveTicket,
    selectedTicketId,
    setSelectedTicketId,
    showNotification,
  } = useUI()

  const getColumns = () => board()?.columns ?? []
  
  const getTicketsForActiveColumn = () => {
    const columns = getColumns()
    const col = columns[activeColumn()]
    if (!col) return []
    return getTicketsByStatus(col.key)
  }

  const clamp = (val: number, min: number, max: number) => 
    Math.max(min, Math.min(max, val))

  const updateSelectedTicket = (colIdx: number, ticketIdx: number) => {
    const columns = getColumns()
    const col = columns[colIdx]
    if (!col) {
      setSelectedTicketId(null)
      return
    }
    const tickets = getTicketsByStatus(col.key)
    const ticket = tickets[ticketIdx]
    setSelectedTicketId(ticket?.id ?? null)
  }

  const handleNormalMode = (e: KeyEvent) => {
    const columns = getColumns()
    const tickets = getTicketsForActiveColumn()

    switch (e.name) {
      case "h":
      case "left": {
        const newCol = clamp(activeColumn() - 1, 0, columns.length - 1)
        setActiveColumn(newCol)
        setActiveTicket(0)
        updateSelectedTicket(newCol, 0)
        break
      }
      case "l":
      case "right": {
        const newCol = clamp(activeColumn() + 1, 0, columns.length - 1)
        setActiveColumn(newCol)
        setActiveTicket(0)
        updateSelectedTicket(newCol, 0)
        break
      }

      case "j":
      case "down": {
        const newTicket = clamp(activeTicket() + 1, 0, Math.max(0, tickets.length - 1))
        setActiveTicket(newTicket)
        updateSelectedTicket(activeColumn(), newTicket)
        break
      }
      case "k":
      case "up": {
        const newTicket = clamp(activeTicket() - 1, 0, Math.max(0, tickets.length - 1))
        setActiveTicket(newTicket)
        updateSelectedTicket(activeColumn(), newTicket)
        break
      }

      case "g": {
        setActiveTicket(0)
        updateSelectedTicket(activeColumn(), 0)
        break
      }
      case "G": {
        const lastIdx = Math.max(0, tickets.length - 1)
        setActiveTicket(lastIdx)
        updateSelectedTicket(activeColumn(), lastIdx)
        break
      }

      case "n": {
        setMode("CREATE_TICKET")
        break
      }
      case "e": {
        if (selectedTicketId()) {
          setMode("EDIT_TICKET")
        } else {
          showNotification("No ticket selected")
        }
        break
      }
      case "d": {
        if (selectedTicketId()) {
          setMode("CONFIRM")
        } else {
          showNotification("No ticket selected")
        }
        break
      }
      case "s": {
        const ticketId = selectedTicketId()
        if (ticketId) {
          actions.onSpawnAgent?.(ticketId)
          showNotification("Spawning agent...")
        }
        break
      }
      case "S": {
        const ticketId = selectedTicketId()
        if (ticketId) {
          actions.onKillAgent?.(ticketId)
          showNotification("Stopping agent...")
        }
        break
      }
      case "enter": {
        if (selectedTicketId()) {
          setMode("AGENT_VIEW")
        }
        break
      }

      case "?": {
        setMode("HELP")
        break
      }
      case "O": {
        setMode("SETTINGS")
        break
      }

      case "q": {
        if (actions.onQuit) {
          actions.onQuit()
        } else {
          process.exit(0)
        }
        break
      }
    }
  }

  const handleOverlayMode = (e: KeyEvent) => {
    if (e.name === "escape" || e.name === "q") {
      setMode("NORMAL")
    }
  }

  const handleConfirmMode = (e: KeyEvent) => {
    if (e.name === "y" || e.name === "Y" || e.name === "enter") {
      const ticketId = selectedTicketId()
      if (ticketId) {
        actions.onDeleteTicket?.(ticketId)
        showNotification("Ticket deleted")
      }
      setMode("NORMAL")
    } else if (e.name === "n" || e.name === "N" || e.name === "escape") {
      setMode("NORMAL")
    }
  }

  const handleAgentViewMode = (e: KeyEvent) => {
    if (e.ctrl && e.name === "g") {
      setMode("NORMAL")
    }
  }

  useKeyboard((e: KeyEvent) => {
    const currentMode = mode()

    switch (currentMode) {
      case "NORMAL":
        handleNormalMode(e)
        break
      case "HELP":
      case "SETTINGS":
        handleOverlayMode(e)
        break
      case "CONFIRM":
        handleConfirmMode(e)
        break
      case "AGENT_VIEW":
        handleAgentViewMode(e)
        break
    }
  })

  return {
    activeColumn,
    activeTicket,
    selectedTicketId,
  }
}
