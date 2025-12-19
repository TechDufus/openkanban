import { extend, useKeyboard } from "@opentui/solid"
import { GhosttyTerminalRenderable } from "ghostty-opentui/terminal-buffer"
import { useTerminal } from "../stores/terminal"
import { translateKey, isTerminalKey } from "../lib/keys"
import type { SessionID } from "@openkanban/shared"

extend({ "ghostty-terminal": GhosttyTerminalRenderable })

interface TerminalProps {
  sessionId: SessionID
  onInput: (data: string) => void
  onExit: () => void
  cols?: number
  rows?: number
}

export function Terminal(props: TerminalProps) {
  const { getSession } = useTerminal()
  
  const session = () => getSession(props.sessionId)
  const buffer = () => session()?.buffer ?? ""
  const cols = () => props.cols ?? session()?.cols ?? 120
  const rows = () => props.rows ?? session()?.rows ?? 40

  useKeyboard((event) => {
    if (!isTerminalKey(event)) {
      if (event.ctrl && event.name.toLowerCase() === "g") {
        props.onExit()
      }
      return
    }

    const translated = translateKey(event)
    if (translated) {
      props.onInput(translated)
    }
  })

  return (
    <box flexDirection="column" width="100%" height="100%">
      <ghostty-terminal 
        ansi={buffer()} 
        cols={cols()} 
        rows={rows()}
        persistent={true}
      />
    </box>
  )
}

declare module "solid-js" {
  namespace JSX {
    interface IntrinsicElements {
      "ghostty-terminal": {
        ansi?: string
        cols?: number
        rows?: number
        persistent?: boolean
        limit?: number
        trimEnd?: boolean
      }
    }
  }
}
