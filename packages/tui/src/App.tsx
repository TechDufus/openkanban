import { onMount, onCleanup, Show, For, createMemo } from "solid-js"
import type { Board, Column, Ticket } from "@openkanban/shared"
import { Connection } from "./client/connection"
import { BoardProvider, useBoard } from "./stores/board"
import { UIProvider, useUI } from "./stores/ui"
import { useNavigation } from "./hooks/useNavigation"
import { HelpOverlay } from "./components/HelpOverlay"
import { ConfirmDialog } from "./components/ConfirmDialog"

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
  const { connectionStatus, setConnectionStatus, mode } = useUI()
  const { board, handleServerMessage, getTicket } = useBoard()
  
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

  const { selectedTicketId } = useNavigation({
    onDeleteTicket: (ticketId) => {
      connection?.send({ type: "ticket:delete", ticketId })
    },
    onSpawnAgent: (ticketId) => {
      connection?.send({ type: "agent:spawn", ticketId })
    },
    onKillAgent: (ticketId) => {
      connection?.send({ type: "agent:kill", ticketId })
    },
  })

  const selectedTicket = createMemo(() => {
    const id = selectedTicketId()
    return id ? getTicket(id) : undefined
  })

  return (
    <box flexDirection="column" width="100%" height="100%">
      <Header status={connectionStatus()} />
      
      <Show when={board()} fallback={<LoadingView status={connectionStatus()} />}>
        <Show when={mode() === "HELP"}>
          <HelpOverlay />
        </Show>
        
        <Show when={mode() === "CONFIRM"}>
          <ConfirmDialog 
            title="Delete Ticket?"
            message={selectedTicket()?.title ?? "Selected ticket"}
          />
        </Show>
        
        <Show when={mode() !== "HELP" && mode() !== "CONFIRM"}>
          <BoardView board={board()!} selectedTicketId={selectedTicketId()} />
        </Show>
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

interface BoardViewProps {
  board: Board
  selectedTicketId: string | null
}

function BoardView(props: BoardViewProps) {
  const { activeColumn } = useUI()
  
  return (
    <box flex={1} flexDirection="row" padding={1}>
      <For each={props.board.columns}>
        {(column, index) => (
          <ColumnView
            column={column}
            tickets={props.board.tickets.filter((t) => t.status === column.key)}
            isActive={activeColumn() === index()}
            selectedTicketId={props.selectedTicketId}
          />
        )}
      </For>
    </box>
  )
}

interface ColumnViewProps {
  column: Column
  tickets: Ticket[]
  isActive: boolean
  selectedTicketId: string | null
}

function ColumnView(props: ColumnViewProps) {
  const headerColor = () => props.isActive ? "#89b4fa" : "#45475a"
  
  return (
    <box flex={1} flexDirection="column" marginRight={1}>
      <box backgroundColor={headerColor()} padding={1}>
        <text color="#cdd6f4" bold>
          {props.column.name} ({props.tickets.length})
        </text>
      </box>
      <box flexDirection="column" flex={1}>
        <For each={props.tickets}>
          {(ticket) => (
            <TicketCard 
              ticket={ticket} 
              isSelected={props.selectedTicketId === ticket.id}
            />
          )}
        </For>
      </box>
    </box>
  )
}

interface TicketCardProps {
  ticket: Ticket
  isSelected: boolean
}

function TicketCard(props: TicketCardProps) {
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

  const borderColor = () => props.isSelected ? "#89b4fa" : statusColor()
  const bgColor = () => props.isSelected ? "#313244" : undefined

  return (
    <box
      borderStyle="rounded"
      borderColor={borderColor()}
      backgroundColor={bgColor()}
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

  const modeHints = () => {
    switch (mode()) {
      case "HELP": return "Press ? or Esc to close help"
      case "CONFIRM": return "Press [y]es or [n]o"
      case "AGENT_VIEW": return "Ctrl+g to exit agent view"
      default: return "[q]uit [n]ew [h/l]columns [j/k]tickets [?]help"
    }
  }

  return (
    <box height={1} backgroundColor="#313244">
      <Show when={notification()} fallback={
        <text color="#6c7086">{" "}{modeHints()}{" "}</text>
      }>
        <text color="#a6e3a1">{" "}{notification()}</text>
      </Show>
      <box flex={1} />
      <text color="#6c7086">{mode()}{" "}</text>
    </box>
  )
}
