import * as pty from "@lydell/node-pty"

console.log("=== Testing @lydell/node-pty with Bun ===\n")

try {
  const shell = process.platform === "win32" ? "powershell.exe" : "bash"
  
  const ptyProcess = pty.spawn(shell, [], {
    name: "xterm-256color",
    cols: 80,
    rows: 24,
    cwd: process.cwd(),
    env: process.env as Record<string, string>,
  })

  console.log(`Spawned PTY with PID: ${ptyProcess.pid}`)

  ptyProcess.onData((data: string) => {
    process.stdout.write(data)
  })

  setTimeout(() => {
    ptyProcess.write("echo 'Hello from @lydell/node-pty'\r")
  }, 500)

  setTimeout(() => {
    ptyProcess.write("pwd\r")
  }, 1000)

  setTimeout(() => {
    ptyProcess.write("exit\r")
  }, 1500)

  ptyProcess.onExit(({ exitCode, signal }) => {
    console.log(`\n=== PTY exited with code ${exitCode}, signal ${signal} ===`)
    console.log("SUCCESS: @lydell/node-pty works with Bun!")
    process.exit(0)
  })

  setTimeout(() => {
    ptyProcess.kill()
    process.exit(1)
  }, 5000)

} catch (error) {
  console.error("FAILED:", error)
  process.exit(1)
}
