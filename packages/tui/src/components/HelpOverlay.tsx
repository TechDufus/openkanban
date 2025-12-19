import { For } from "solid-js"

const KEYBINDINGS = [
  { key: "h / ←", action: "Move to left column" },
  { key: "l / →", action: "Move to right column" },
  { key: "j / ↓", action: "Move to next ticket" },
  { key: "k / ↑", action: "Move to previous ticket" },
  { key: "g", action: "Jump to first ticket" },
  { key: "G", action: "Jump to last ticket" },
  { key: "n", action: "Create new ticket" },
  { key: "e", action: "Edit selected ticket" },
  { key: "d", action: "Delete selected ticket" },
  { key: "s", action: "Spawn agent on ticket" },
  { key: "S", action: "Stop agent on ticket" },
  { key: "Enter", action: "Open agent view" },
  { key: "O", action: "Open settings" },
  { key: "?", action: "Toggle this help" },
  { key: "q / Esc", action: "Close / Quit" },
]

export function HelpOverlay() {
  return (
    <box
      width="100%"
      height="100%"
      justifyContent="center"
      alignItems="center"
    >
      <box
        backgroundColor="#1e1e2e"
        borderStyle="rounded"
        borderColor="#89b4fa"
        padding={2}
        flexDirection="column"
        width={50}
      >
        <text color="#89b4fa" bold>
          {" Keyboard Shortcuts "}
        </text>
        <box height={1} />
        <box flexDirection="column">
          <For each={KEYBINDINGS}>
            {(binding) => (
              <box flexDirection="row">
                <box width={14}>
                  <text color="#f9e2af">{binding.key}</text>
                </box>
                <text color="#cdd6f4">{binding.action}</text>
              </box>
            )}
          </For>
        </box>
        <box height={1} />
        <text color="#6c7086">Press ? or Esc to close</text>
      </box>
    </box>
  )
}
