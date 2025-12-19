import { createSignal, createContext, useContext, type ParentComponent } from "solid-js"
import type { SessionID } from "@openkanban/shared"

interface TerminalSession {
  sessionId: SessionID
  buffer: string
  cols: number
  rows: number
}

interface TerminalContextValue {
  sessions: () => Map<SessionID, TerminalSession>
  getSession: (sessionId: SessionID) => TerminalSession | undefined
  appendOutput: (sessionId: SessionID, data: string) => void
  setBuffer: (sessionId: SessionID, data: string) => void
  createSession: (sessionId: SessionID, cols?: number, rows?: number) => void
  removeSession: (sessionId: SessionID) => void
  resizeSession: (sessionId: SessionID, cols: number, rows: number) => void
}

const TerminalContext = createContext<TerminalContextValue>()

export const TerminalProvider: ParentComponent = (props) => {
  const [sessions, setSessions] = createSignal<Map<SessionID, TerminalSession>>(new Map())

  const getSession = (sessionId: SessionID): TerminalSession | undefined => {
    return sessions().get(sessionId)
  }

  const createSession = (sessionId: SessionID, cols = 120, rows = 40): void => {
    setSessions((prev) => {
      const next = new Map(prev)
      next.set(sessionId, { sessionId, buffer: "", cols, rows })
      return next
    })
  }

  const removeSession = (sessionId: SessionID): void => {
    setSessions((prev) => {
      const next = new Map(prev)
      next.delete(sessionId)
      return next
    })
  }

  const appendOutput = (sessionId: SessionID, data: string): void => {
    setSessions((prev) => {
      const session = prev.get(sessionId)
      if (!session) {
        const next = new Map(prev)
        next.set(sessionId, { sessionId, buffer: data, cols: 120, rows: 40 })
        return next
      }
      const next = new Map(prev)
      next.set(sessionId, { ...session, buffer: session.buffer + data })
      return next
    })
  }

  const setBuffer = (sessionId: SessionID, data: string): void => {
    setSessions((prev) => {
      const session = prev.get(sessionId)
      if (!session) {
        const next = new Map(prev)
        next.set(sessionId, { sessionId, buffer: data, cols: 120, rows: 40 })
        return next
      }
      const next = new Map(prev)
      next.set(sessionId, { ...session, buffer: data })
      return next
    })
  }

  const resizeSession = (sessionId: SessionID, cols: number, rows: number): void => {
    setSessions((prev) => {
      const session = prev.get(sessionId)
      if (!session) return prev
      const next = new Map(prev)
      next.set(sessionId, { ...session, cols, rows })
      return next
    })
  }

  const value: TerminalContextValue = {
    sessions,
    getSession,
    appendOutput,
    setBuffer,
    createSession,
    removeSession,
    resizeSession,
  }

  return (
    <TerminalContext.Provider value={value}>
      {props.children}
    </TerminalContext.Provider>
  )
}

export function useTerminal() {
  const context = useContext(TerminalContext)
  if (!context) {
    throw new Error("useTerminal must be used within TerminalProvider")
  }
  return context
}
