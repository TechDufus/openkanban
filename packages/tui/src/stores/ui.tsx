import { createSignal, createContext, useContext, type ParentComponent } from "solid-js"
import type { ConnectionStatus } from "../client/connection"

export type Mode =
  | "NORMAL"
  | "HELP"
  | "CONFIRM"
  | "CREATE_TICKET"
  | "EDIT_TICKET"
  | "AGENT_VIEW"
  | "SETTINGS"

interface UIContextValue {
  mode: () => Mode
  setMode: (mode: Mode) => void
  connectionStatus: () => ConnectionStatus
  setConnectionStatus: (status: ConnectionStatus) => void
  activeColumn: () => number
  setActiveColumn: (col: number) => void
  activeTicket: () => number
  setActiveTicket: (ticket: number) => void
  selectedTicketId: () => string | null
  setSelectedTicketId: (id: string | null) => void
  notification: () => string | null
  showNotification: (message: string, durationMs?: number) => void
}

const UIContext = createContext<UIContextValue>()

export const UIProvider: ParentComponent = (props) => {
  const [mode, setModeSignal] = createSignal<Mode>("NORMAL")
  const [connectionStatus, setConnectionStatusSignal] = createSignal<ConnectionStatus>("disconnected")
  const [activeColumn, setActiveColumnSignal] = createSignal(0)
  const [activeTicket, setActiveTicketSignal] = createSignal(0)
  const [selectedTicketId, setSelectedTicketIdSignal] = createSignal<string | null>(null)
  const [notification, setNotification] = createSignal<string | null>(null)

  let notificationTimeout: ReturnType<typeof setTimeout> | null = null

  const showNotification = (message: string, durationMs = 3000) => {
    if (notificationTimeout) {
      clearTimeout(notificationTimeout)
    }
    setNotification(message)
    notificationTimeout = setTimeout(() => {
      setNotification(null)
      notificationTimeout = null
    }, durationMs)
  }

  const value: UIContextValue = {
    mode,
    setMode: setModeSignal,
    connectionStatus,
    setConnectionStatus: setConnectionStatusSignal,
    activeColumn,
    setActiveColumn: setActiveColumnSignal,
    activeTicket,
    setActiveTicket: setActiveTicketSignal,
    selectedTicketId,
    setSelectedTicketId: setSelectedTicketIdSignal,
    notification,
    showNotification,
  }

  return (
    <UIContext.Provider value={value}>
      {props.children}
    </UIContext.Provider>
  )
}

export function useUI() {
  const context = useContext(UIContext)
  if (!context) {
    throw new Error("useUI must be used within UIProvider")
  }
  return context
}
