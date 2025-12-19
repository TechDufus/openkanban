import type { ServerWebSocket } from "bun"
import {
  parseClientMessage,
  serializeServerMessage,
  type ClientMessage,
  type ServerMessage,
} from "@openkanban/shared"
import { boardStore } from "../store/board"
import { ptyManager } from "../pty/manager"

interface WebSocketData {
  id: string
  subscribedToBoard: boolean
  subscribedTerminals: Set<string>
}

const clients = new Map<string, ServerWebSocket<WebSocketData>>()

export function handleWsOpen(ws: ServerWebSocket<WebSocketData>): void {
  const id = crypto.randomUUID()
  ws.data = {
    id,
    subscribedToBoard: false,
    subscribedTerminals: new Set(),
  }
  clients.set(id, ws)
  console.log(`[ws] Client connected: ${id}`)
}

export function handleWsClose(ws: ServerWebSocket<WebSocketData>): void {
  const { id, subscribedTerminals } = ws.data
  
  for (const sessionId of subscribedTerminals) {
    ptyManager.unsubscribe(sessionId, ws)
  }
  
  clients.delete(id)
  console.log(`[ws] Client disconnected: ${id}`)
}

export function handleWsMessage(
  ws: ServerWebSocket<WebSocketData>,
  message: string | Buffer
): void {
  const data = typeof message === "string" ? message : message.toString()
  const msg = parseClientMessage(data)
  
  if (!msg) {
    sendError(ws, "Invalid message format")
    return
  }
  
  routeMessage(ws, msg)
}

function routeMessage(ws: ServerWebSocket<WebSocketData>, msg: ClientMessage): void {
  switch (msg.type) {
    case "board:subscribe":
      handleBoardSubscribe(ws)
      break
      
    case "board:unsubscribe":
      ws.data.subscribedToBoard = false
      break
      
    case "ticket:create":
      handleTicketCreate(ws, msg.ticket)
      break
      
    case "ticket:update":
      handleTicketUpdate(ws, msg.ticketId, msg.patch)
      break
      
    case "ticket:delete":
      handleTicketDelete(ws, msg.ticketId)
      break
      
    case "ticket:move":
      handleTicketMove(ws, msg.ticketId, msg.status)
      break
      
    case "agent:spawn":
      handleAgentSpawn(ws, msg.ticketId)
      break
      
    case "agent:kill":
      handleAgentKill(ws, msg.ticketId)
      break
      
    case "terminal:subscribe":
      handleTerminalSubscribe(ws, msg.sessionId)
      break
      
    case "terminal:unsubscribe":
      handleTerminalUnsubscribe(ws, msg.sessionId)
      break
      
    case "terminal:input":
      handleTerminalInput(ws, msg.sessionId, msg.data)
      break
      
    case "terminal:resize":
      handleTerminalResize(ws, msg.sessionId, msg.cols, msg.rows)
      break
      
    default:
      sendError(ws, `Unknown message type`)
  }
}

function handleBoardSubscribe(ws: ServerWebSocket<WebSocketData>): void {
  ws.data.subscribedToBoard = true
  const board = boardStore.getBoard()
  send(ws, { type: "board:state", board })
}

function handleTicketCreate(
  _ws: ServerWebSocket<WebSocketData>,
  ticketData: Parameters<typeof boardStore.createTicket>[0]
): void {
  const ticket = boardStore.createTicket(ticketData)
  broadcast({ type: "ticket:created", ticket })
}

function handleTicketUpdate(
  ws: ServerWebSocket<WebSocketData>,
  ticketId: string,
  patch: Parameters<typeof boardStore.updateTicket>[1]
): void {
  const ticket = boardStore.updateTicket(ticketId, patch)
  if (ticket) {
    broadcast({ type: "ticket:updated", ticket })
  } else {
    sendError(ws, "Ticket not found")
  }
}

function handleTicketDelete(ws: ServerWebSocket<WebSocketData>, ticketId: string): void {
  const deleted = boardStore.deleteTicket(ticketId)
  if (deleted) {
    broadcast({ type: "ticket:deleted", ticketId })
  } else {
    sendError(ws, "Ticket not found")
  }
}

function handleTicketMove(
  ws: ServerWebSocket<WebSocketData>,
  ticketId: string,
  status: Parameters<typeof boardStore.moveTicket>[1]
): void {
  const ticket = boardStore.moveTicket(ticketId, status)
  if (ticket) {
    broadcast({ type: "ticket:updated", ticket })
  } else {
    sendError(ws, "Ticket not found")
  }
}

async function handleAgentSpawn(ws: ServerWebSocket<WebSocketData>, ticketId: string): Promise<void> {
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket) {
    sendError(ws, "Ticket not found")
    return
  }
  
  const cwd = ticket.worktreePath ?? process.cwd()
  const sessionId = await ptyManager.spawn(ticketId, ["bash"], cwd)
  
  boardStore.updateTicket(ticketId, {
    terminalSessionId: sessionId,
    agentStatus: "idle",
    agentSpawnedAt: new Date().toISOString(),
  })
  
  const updatedTicket = boardStore.getTicket(ticketId)
  if (updatedTicket) {
    broadcast({ type: "ticket:updated", ticket: updatedTicket })
    broadcast({ type: "agent:status", ticketId, status: "idle", sessionId })
  }
}

function handleAgentKill(ws: ServerWebSocket<WebSocketData>, ticketId: string): void {
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket?.terminalSessionId) {
    sendError(ws, "No agent running for ticket")
    return
  }
  
  ptyManager.kill(ticket.terminalSessionId)
  
  boardStore.updateTicket(ticketId, {
    terminalSessionId: undefined,
    agentStatus: "none",
  })
  
  const updatedTicket = boardStore.getTicket(ticketId)
  if (updatedTicket) {
    broadcast({ type: "ticket:updated", ticket: updatedTicket })
    broadcast({ type: "agent:status", ticketId, status: "none" })
  }
}

function handleTerminalSubscribe(ws: ServerWebSocket<WebSocketData>, sessionId: string): void {
  ws.data.subscribedTerminals.add(sessionId)
  ptyManager.subscribe(sessionId, ws)
  
  const buffer = ptyManager.getBuffer(sessionId)
  if (buffer) {
    send(ws, { type: "terminal:buffer", sessionId, data: buffer })
  }
}

function handleTerminalUnsubscribe(ws: ServerWebSocket<WebSocketData>, sessionId: string): void {
  ws.data.subscribedTerminals.delete(sessionId)
  ptyManager.unsubscribe(sessionId, ws)
}

function handleTerminalInput(
  ws: ServerWebSocket<WebSocketData>,
  sessionId: string,
  data: string
): void {
  ptyManager.write(sessionId, data)
}

function handleTerminalResize(
  ws: ServerWebSocket<WebSocketData>,
  sessionId: string,
  cols: number,
  rows: number
): void {
  ptyManager.resize(sessionId, cols, rows)
}

function send(ws: ServerWebSocket<WebSocketData>, msg: ServerMessage): void {
  ws.send(serializeServerMessage(msg))
}

function sendError(ws: ServerWebSocket<WebSocketData>, message: string): void {
  send(ws, { type: "error", message })
}

export function broadcast(msg: ServerMessage): void {
  const data = serializeServerMessage(msg)
  for (const [, client] of clients) {
    if (client.data.subscribedToBoard) {
      client.send(data)
    }
  }
}

export function broadcastTerminalOutput(sessionId: string, data: string): void {
  const msg = serializeServerMessage({ type: "terminal:output", sessionId, data })
  for (const [, client] of clients) {
    if (client.data.subscribedTerminals.has(sessionId)) {
      client.send(msg)
    }
  }
}

export function broadcastTerminalExit(sessionId: string, code: number): void {
  const msg = serializeServerMessage({ type: "terminal:exit", sessionId, code })
  for (const [, client] of clients) {
    if (client.data.subscribedTerminals.has(sessionId)) {
      client.send(msg)
    }
  }
}
