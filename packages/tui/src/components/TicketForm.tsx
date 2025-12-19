import { createSignal } from "solid-js"
import { useKeyboard } from "@opentui/solid"

interface TicketFormProps {
  mode: "create" | "edit"
  initialTitle?: string
  onSubmit: (title: string) => void
  onCancel: () => void
}

export function TicketForm(props: TicketFormProps) {
  const [title, setTitle] = createSignal(props.initialTitle ?? "")
  const [cursorPos, setCursorPos] = createSignal(props.initialTitle?.length ?? 0)

  useKeyboard((e) => {
    if (e.name === "escape") {
      props.onCancel()
      return
    }

    if (e.name === "enter" || e.name === "return") {
      const t = title().trim()
      if (t) {
        props.onSubmit(t)
      }
      return
    }

    if (e.name === "backspace") {
      const pos = cursorPos()
      if (pos > 0) {
        setTitle((t) => t.slice(0, pos - 1) + t.slice(pos))
        setCursorPos(pos - 1)
      }
      return
    }

    if (e.name === "delete") {
      const pos = cursorPos()
      setTitle((t) => t.slice(0, pos) + t.slice(pos + 1))
      return
    }

    if (e.name === "left") {
      setCursorPos((p) => Math.max(0, p - 1))
      return
    }

    if (e.name === "right") {
      setCursorPos((p) => Math.min(title().length, p + 1))
      return
    }

    if (e.name === "home" || (e.ctrl && e.name === "a")) {
      setCursorPos(0)
      return
    }

    if (e.name === "end" || (e.ctrl && e.name === "e")) {
      setCursorPos(title().length)
      return
    }

    if (e.sequence && e.sequence.length === 1 && !e.ctrl && !e.meta) {
      const pos = cursorPos()
      setTitle((t) => t.slice(0, pos) + e.sequence + t.slice(pos))
      setCursorPos(pos + 1)
    }
  })

  const displayTitle = () => {
    const t = title()
    const pos = cursorPos()
    const before = t.slice(0, pos)
    const cursor = t[pos] ?? " "
    const after = t.slice(pos + 1)
    return { before, cursor, after }
  }

  const headerText = () => props.mode === "create" ? "New Ticket" : "Edit Ticket"

  return (
    <box
      width="100%"
      height="100%"
      justifyContent="center"
      alignItems="center"
    >
      <box
        borderStyle="rounded"
        borderColor="#89b4fa"
        padding={1}
        width={60}
        flexDirection="column"
      >
        <text color="#89b4fa" bold>{headerText()}</text>
        <text color="#6c7086">{"\n"}Title:</text>
        <box marginTop={1}>
          <text color="#cdd6f4">{displayTitle().before}</text>
          <text color="#1e1e2e" backgroundColor="#cdd6f4">{displayTitle().cursor}</text>
          <text color="#cdd6f4">{displayTitle().after}</text>
        </box>
        <text color="#6c7086">{"\n"}[Enter] Save  [Esc] Cancel</text>
      </box>
    </box>
  )
}
