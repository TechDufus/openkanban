import { onMount, onCleanup, Show, For } from "solid-js"
import type { Board, Column, Ticket } from "@openkanban/shared"
import { Connection } from "./client/connection"
import { BoardProvider, useBoard } from "./stores/board"
import { UIProvider, useUI } from "./stores/ui"

interface AppProps {
  serverPort: number
}

export function App(props: AppProps) {
  return (
    <UIProvider>
      <BoardProvider>
        <AppContent serverPort={props.serverPort} />
      </BoardProvider>
    </UIProvider>
  )
}

function AppContent(props: { serverPort: number }) {
  const { connectionStatus, setConnectionStatus } = useUI()
  const { board, handleServerMessage } = useBoard()
  
  let connection: Connection | null = null

  onMount(() => {
    connection = new Connection(props.serverPort, {
      onStatusChange: setConnectionStatus,
      onMessage: handleServerMessage,
    })
    connection.connect()
  })

  onCleanup(() => {
    connection?.disconnect()
  })

  return (
    <box flexDirection="column" width="100%" height="100%">
      <Header status={connectionStatus()} />
      
      <Show when={board()} fallback={<LoadingView status={connectionStatus()} />}>
        <BoardView board={board()!} />
      </Show>

      <StatusBar />
    </box>
  )
}

function Header(props: { status: string }) {
  const statusColor = () => {
    switch (props.status) {
      case "connected": return "#a6e3a1"
      case "connecting": return "#f9e2af"
      default: return "#f38ba8"
    }
  }

  const statusText = () => {
    switch (props.status) {
      case "connected": return "Connected"
      case "connecting": return "Connecting..."
      default: return "Disconnected"
    }
  }

  return (
    <box height={1} backgroundColor="#313244">
      <text color="#89b4fa" bold>{" OpenKanban "}</text>
      <text color="#6c7086">{" | "}</text>
      <text color={statusColor()}>{statusText()}</text>
    </box>
  )
}

function LoadingView(props: { status: string }) {
  const message = () => {
    switch (props.status) {
      case "connecting": return "Connecting to server..."
      case "disconnected": return "Server disconnected. Reconnecting..."
      default: return "Loading..."
    }
  }

  return (
    <box flex={1} justifyContent="center" alignItems="center">
      <text color="#6c7086">{message()}</text>
    </box>
  )
}

function BoardView(props: { board: Board }) {
  return (
    <box flex={1} flexDirection="row" padding={1}>
      <For each={props.board.columns}>
        {(column) => (
          <ColumnView
            column={column}
            tickets={props.board.tickets.filter((t) => t.status === column.key)}
          />
        )}
      </For>
    </box>
  )
}

function ColumnView(props: { column: Column; tickets: Ticket[] }) {
  return (
    <box flex={1} flexDirection="column" marginRight={1}>
      <box backgroundColor="#45475a" padding={1}>
        <text color="#cdd6f4" bold>
          {props.column.name} ({props.tickets.length})
        </text>
      </box>
      <box flexDirection="column" flex={1}>
        <For each={props.tickets}>
          {(ticket) => <TicketCard ticket={ticket} />}
        </For>
      </box>
    </box>
  )
}

function TicketCard(props: { ticket: Ticket }) {
  const statusColor = () => {
    switch (props.ticket.agentStatus) {
      case "working": return "#f9e2af"
      case "waiting": return "#cba6f7"
      case "completed": return "#a6e3a1"
      case "error": return "#f38ba8"
      case "idle": return "#89b4fa"
      default: return "#45475a"
    }
  }

  return (
    <box
      borderStyle="rounded"
      borderColor={statusColor()}
      padding={1}
      marginTop={1}
    >
      <text color="#6c7086">{props.ticket.id.slice(0, 8)}</text>
      <text color="#cdd6f4">{" "}{props.ticket.title}</text>
      <Show when={props.ticket.agentStatus !== "none"}>
        <text color={statusColor()}>{" ["}{props.ticket.agentStatus}{"]"}</text>
      </Show>
    </box>
  )
}

function StatusBar() {
  const { mode, notification } = useUI()

  return (
    <box height={1} backgroundColor="#313244">
      <Show when={notification()} fallback={
        <text color="#6c7086">
          {" [q]uit [n]ew [h/l]columns [j/k]tickets [?]help "}
        </text>
      }>
        <text color="#a6e3a1">{" "}{notification()}</text>
      </Show>
      <box flex={1} />
      <text color="#6c7086">{mode()}</text>
    </box>
  )
}
