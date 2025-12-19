import { Daemon } from "./daemon"

const DEFAULT_PORT = 4200

export async function startServer(port = DEFAULT_PORT): Promise<void> {
  const daemon = new Daemon()
  await daemon.start(port)
}

export { Daemon } from "./daemon"
export { createApp } from "./api"

if (import.meta.main) {
  const port = parseInt(process.env["PORT"] ?? String(DEFAULT_PORT), 10)
  startServer(port)
}
