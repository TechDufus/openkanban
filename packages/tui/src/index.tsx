import { render } from "@opentui/solid"
import { App } from "./App"

export function startTui(serverPort = 4200): void {
  console.log(`[openkanban] Connecting to server on port ${serverPort}...`)
  render(() => <App serverPort={serverPort} />)
}

export { App } from "./App"

if (import.meta.main) {
  const port = parseInt(process.env["PORT"] ?? "4200", 10)
  startTui(port)
}
