import type { ClientMessage, ServerMessage } from "@openkanban/shared"

export type ConnectionStatus = "connecting" | "connected" | "disconnected"

export interface ConnectionCallbacks {
  onStatusChange: (status: ConnectionStatus) => void
  onMessage: (msg: ServerMessage) => void
}

export class Connection {
  private ws: WebSocket | null = null
  private url: string
  private callbacks: ConnectionCallbacks
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 10
  private baseReconnectDelay = 1000

  constructor(port: number, callbacks: ConnectionCallbacks) {
    this.url = `ws://localhost:${port}/ws`
    this.callbacks = callbacks
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return

    this.callbacks.onStatusChange("connecting")

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        this.reconnectAttempts = 0
        this.callbacks.onStatusChange("connected")
        this.send({ type: "board:subscribe" })
      }

      this.ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as ServerMessage
          this.callbacks.onMessage(msg)
        } catch {
          console.error("[ws] Failed to parse message")
        }
      }

      this.ws.onclose = () => {
        this.callbacks.onStatusChange("disconnected")
        this.scheduleReconnect()
      }

      this.ws.onerror = () => {
        this.ws?.close()
      }
    } catch {
      this.callbacks.onStatusChange("disconnected")
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return

    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts),
      30000
    )
    this.reconnectAttempts++

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, delay)
  }

  send(msg: ClientMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
    }
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
  }
}
