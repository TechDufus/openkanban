import { createSignal, createContext, useContext, type ParentComponent } from "solid-js"
import type { Board, Ticket, TicketStatus, ServerMessage } from "@openkanban/shared"

interface BoardContextValue {
  board: () => Board | null
  getTicketsByStatus: (status: TicketStatus) => Ticket[]
  getTicket: (id: string) => Ticket | undefined
  handleServerMessage: (msg: ServerMessage) => void
}

const BoardContext = createContext<BoardContextValue>()

export const BoardProvider: ParentComponent = (props) => {
  const [board, setBoard] = createSignal<Board | null>(null)

  const handleServerMessage = (msg: ServerMessage) => {
    switch (msg.type) {
      case "board:state":
        setBoard(msg.board)
        break

      case "ticket:created":
        setBoard((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            tickets: [...prev.tickets, msg.ticket],
          }
        })
        break

      case "ticket:updated":
        setBoard((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            tickets: prev.tickets.map((t) =>
              t.id === msg.ticket.id ? msg.ticket : t
            ),
          }
        })
        break

      case "ticket:deleted":
        setBoard((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            tickets: prev.tickets.filter((t) => t.id !== msg.ticketId),
          }
        })
        break

      case "agent:status":
        setBoard((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            tickets: prev.tickets.map((t) =>
              t.id === msg.ticketId
                ? { ...t, agentStatus: msg.status, terminalSessionId: msg.sessionId }
                : t
            ),
          }
        })
        break
    }
  }

  const getTicketsByStatus = (status: TicketStatus): Ticket[] => {
    const b = board()
    if (!b) return []
    return b.tickets.filter((t) => t.status === status)
  }

  const getTicket = (id: string): Ticket | undefined => {
    const b = board()
    if (!b) return undefined
    return b.tickets.find((t) => t.id === id)
  }

  const value: BoardContextValue = {
    board,
    getTicketsByStatus,
    getTicket,
    handleServerMessage,
  }

  return (
    <BoardContext.Provider value={value}>
      {props.children}
    </BoardContext.Provider>
  )
}

export function useBoard() {
  const context = useContext(BoardContext)
  if (!context) {
    throw new Error("useBoard must be used within BoardProvider")
  }
  return context
}
