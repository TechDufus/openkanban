import { Hono } from "hono"

export const healthRoutes = new Hono()

healthRoutes.get("/health", (c) => {
  return c.json({
    status: "ok",
    pid: process.pid,
    uptime: process.uptime(),
    timestamp: new Date().toISOString(),
  })
})
