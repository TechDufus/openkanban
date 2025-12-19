import type { KeyEvent } from "@opentui/core"

export function translateKey(event: KeyEvent): string {
  const { name, ctrl, option: alt } = event

  if (ctrl && name.length === 1) {
    const code = name.toLowerCase().charCodeAt(0)
    if (code >= 97 && code <= 122) {
      return String.fromCharCode(code - 96)
    }
  }

  if (ctrl) {
    switch (name) {
      case "[": return "\x1b"
      case "\\": return "\x1c"
      case "]": return "\x1d"
      case "^": return "\x1e"
      case "_": return "\x1f"
      case "?": return "\x7f"
    }
  }

  const prefix = alt ? "\x1b" : ""

  switch (name) {
    case "up": return prefix + "\x1b[A"
    case "down": return prefix + "\x1b[B"
    case "right": return prefix + "\x1b[C"
    case "left": return prefix + "\x1b[D"
    case "f1": return "\x1bOP"
    case "f2": return "\x1bOQ"
    case "f3": return "\x1bOR"
    case "f4": return "\x1bOS"
    case "f5": return "\x1b[15~"
    case "f6": return "\x1b[17~"
    case "f7": return "\x1b[18~"
    case "f8": return "\x1b[19~"
    case "f9": return "\x1b[20~"
    case "f10": return "\x1b[21~"
    case "f11": return "\x1b[23~"
    case "f12": return "\x1b[24~"
    case "home": return "\x1b[H"
    case "end": return "\x1b[F"
    case "pageup": return "\x1b[5~"
    case "pagedown": return "\x1b[6~"
    case "insert": return "\x1b[2~"
    case "delete": return "\x1b[3~"
    case "return":
    case "enter": return "\r"
    case "tab": return "\t"
    case "backspace": return "\x7f"
    case "escape": return "\x1b"
    case "space": return " "
  }

  if (name.length === 1) {
    return prefix + name
  }

  return ""
}

export function isTerminalKey(event: KeyEvent): boolean {
  if (event.ctrl && event.name.toLowerCase() === "g") {
    return false
  }
  return true
}

export type { KeyEvent }
