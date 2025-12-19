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
  getBuffer: (sessionId: SessionID) => string
  appendOutput: (sessionId: SessionID, data: string) => void
  setBuffer: (sessionId: SessionID, data: string) => void
  createSession: (sessionId: SessionID, cols?: number, rows?: number) => void
  removeSession: (sessionId: SessionID) => void
  resizeSession: (sessionId: SessionID, cols: number, rows: number) => void
}

const TerminalContext = createContext<TerminalContextValue>()

export const TerminalProvider: ParentComponent = (props) => {
  const [sessions, setSessions] = createSignal<Map<SessionID, TerminalSession>>(new Map())
  const [renderTrigger, setRenderTrigger] = createSignal(0)

  const getSession = (sessionId: SessionID): TerminalSession | undefined => {
    return sessions().get(sessionId)
  }

  const createSession = (sessionId: SessionID, cols = 120, rows = 40): void => {
    setSessions((prev) => {
      if (prev.has(sessionId)) return prev
      
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

  const ensureSession = (sessionId: SessionID): TerminalSession => {
    let session = sessions().get(sessionId)
    
    if (!session) {
      session = { sessionId, buffer: "", cols: 120, rows: 40 }
      setSessions((prev) => {
        const next = new Map(prev)
        next.set(sessionId, session!)
        return next
      })
    }
    
    return session
  }

  const appendOutput = (sessionId: SessionID, data: string): void => {
    const session = ensureSession(sessionId)
    session.buffer += data
    setRenderTrigger((n) => n + 1)
  }

  const setBuffer = (sessionId: SessionID, data: string): void => {
    const session = ensureSession(sessionId)
    session.buffer = data
    setRenderTrigger((n) => n + 1)
  }

  const getBuffer = (sessionId: SessionID): string => {
    renderTrigger()
    
    const session = sessions().get(sessionId)
    if (!session) return ""
    
    return session.buffer
  }

  const resizeSession = (sessionId: SessionID, cols: number, rows: number): void => {
    const session = sessions().get(sessionId)
    if (!session) return
    
    session.cols = cols
    session.rows = rows
    
    setRenderTrigger((n) => n + 1)
  }

  const value: TerminalContextValue = {
    sessions,
    getSession,
    getBuffer,
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
