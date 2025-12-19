import { Hono } from "hono"
import type { Ticket } from "@openkanban/shared"
import { boardStore } from "../../store/board"

export const boardRoutes = new Hono()

boardRoutes.get("/", (c) => {
  return c.json(boardStore.getBoard())
})

boardRoutes.get("/:id", (c) => {
  const ticketId = c.req.param("id")
  const ticket = boardStore.getTicket(ticketId)
  if (!ticket) {
    return c.json({ error: "Ticket not found" }, 404)
  }
  return c.json(ticket)
})

boardRoutes.post("/tickets", async (c) => {
  const body = await c.req.json<{ title: string; description?: string }>()
  const ticket = boardStore.createTicket({
    title: body.title,
    description: body.description,
  })
  return c.json(ticket, 201)
})

boardRoutes.patch("/:id", async (c) => {
  const ticketId = c.req.param("id")
  const patch = await c.req.json<Partial<Ticket>>()
  const ticket = boardStore.updateTicket(ticketId, patch)
  if (!ticket) {
    return c.json({ error: "Ticket not found" }, 404)
  }
  return c.json(ticket)
})

boardRoutes.delete("/:id", (c) => {
  const ticketId = c.req.param("id")
  const deleted = boardStore.deleteTicket(ticketId)
  if (!deleted) {
    return c.json({ error: "Ticket not found" }, 404)
  }
  return c.json({ success: true })
})

boardRoutes.post("/:id/move", async (c) => {
  const ticketId = c.req.param("id")
  const body = await c.req.json<{ status: Ticket["status"] }>()
  const ticket = boardStore.moveTicket(ticketId, body.status)
  if (!ticket) {
    return c.json({ error: "Ticket not found" }, 404)
  }
  return c.json(ticket)
})
