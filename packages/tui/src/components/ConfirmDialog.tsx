interface ConfirmDialogProps {
  title: string
  message: string
}

export function ConfirmDialog(props: ConfirmDialogProps) {
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
        borderColor="#f38ba8"
        padding={1}
        flexDirection="column"
        alignItems="center"
        width={40}
      >
        <text color="#f38ba8" bold>
          {props.title}
        </text>
        <box height={1} />
        <text color="#cdd6f4">{props.message}</text>
        <box height={1} />
        <text color="#6c7086">[y]es  [n]o</text>
      </box>
    </box>
  )
}
