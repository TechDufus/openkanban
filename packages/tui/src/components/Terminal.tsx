import { useKeyboard } from "@opentui/solid"
import { useTerminal } from "../stores/terminal"
import { translateKey, isTerminalKey } from "../lib/keys"
import type { SessionID } from "@openkanban/shared"

interface TerminalProps {
  sessionId: SessionID
  onInput: (data: string) => void
  onExit: () => void
  cols?: number
  rows?: number
}

export function Terminal(props: TerminalProps) {
  const { getBuffer } = useTerminal()

  const buffer = () => getBuffer(props.sessionId)

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
    <scroll-box 
      flexDirection="column" 
      width="100%" 
      height="100%"
      stickyScroll={true}
      stickyStart="bottom"
    >
      <text>{buffer()}</text>
    </scroll-box>
  )
}
