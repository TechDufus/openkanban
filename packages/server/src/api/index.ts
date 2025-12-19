import { Hono } from "hono"
import { cors } from "hono/cors"
import { healthRoutes } from "./routes/health"
import { boardRoutes } from "./routes/board"

type Env = {
  Bindings: {
    server?: ReturnType<typeof Bun.serve>
  }
}

export function createApp() {
  const app = new Hono<Env>()

  app.use("*", cors())

  app.route("/api", healthRoutes)
  app.route("/api/board", boardRoutes)
  app.route("/api/tickets", boardRoutes)

  app.get("/ws", (c) => {
    return c.text("WebSocket upgrade handled by Bun.serve", 200)
  })

  return app
}
