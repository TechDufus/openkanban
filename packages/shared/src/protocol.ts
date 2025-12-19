import type { AgentStatus, Board, SessionID, Ticket, TicketID, TicketStatus } from "./types"

export type ClientMessage =
  | { type: "terminal:input"; sessionId: SessionID; data: string }
  | { type: "terminal:resize"; sessionId: SessionID; cols: number; rows: number }
  | { type: "terminal:subscribe"; sessionId: SessionID }
  | { type: "terminal:unsubscribe"; sessionId: SessionID }
  | { type: "agent:spawn"; ticketId: TicketID }
  | { type: "agent:kill"; ticketId: TicketID }
  | { type: "board:subscribe" }
  | { type: "board:unsubscribe" }
  | { type: "ticket:create"; ticket: Partial<Ticket> & { title: string } }
  | { type: "ticket:update"; ticketId: TicketID; patch: Partial<Ticket> }
  | { type: "ticket:delete"; ticketId: TicketID }
  | { type: "ticket:move"; ticketId: TicketID; status: TicketStatus }

export type ServerMessage =
  | { type: "terminal:output"; sessionId: SessionID; data: string }
  | { type: "terminal:buffer"; sessionId: SessionID; data: string }
  | { type: "terminal:exit"; sessionId: SessionID; code: number }
  | { type: "board:state"; board: Board }
  | { type: "board:patch"; patch: Partial<Board> }
  | { type: "ticket:created"; ticket: Ticket }
  | { type: "ticket:updated"; ticket: Ticket }
  | { type: "ticket:deleted"; ticketId: TicketID }
  | { type: "agent:status"; ticketId: TicketID; status: AgentStatus; sessionId?: SessionID }
  | { type: "error"; message: string; code?: string }
  | { type: "pong" }

export function parseClientMessage(data: string): ClientMessage | null {
  try {
    return JSON.parse(data) as ClientMessage
  } catch {
    return null
  }
}

export function serializeServerMessage(msg: ServerMessage): string {
  return JSON.stringify(msg)
}

export const WS_CLOSE_CODES = {
  NORMAL: 1000,
  GOING_AWAY: 1001,
  PROTOCOL_ERROR: 1002,
  INVALID_DATA: 1003,
  SERVER_ERROR: 1011,
} as const
