/**
 * Test 1: node-pty with Bun
 * 
 * node-pty is the most common PTY library for Node.js
 * Let's see if it works with Bun
 */

import * as pty from "node-pty"

console.log("=== Testing node-pty with Bun ===\n")

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

  // Collect output
  let output = ""
  ptyProcess.onData((data: string) => {
    output += data
    process.stdout.write(data)
  })

  // Send a simple command
  setTimeout(() => {
    console.log("\n--- Sending 'echo hello from pty' ---")
    ptyProcess.write("echo hello from pty\r")
  }, 500)

  // Send another command
  setTimeout(() => {
    console.log("\n--- Sending 'pwd' ---")
    ptyProcess.write("pwd\r")
  }, 1000)

  // Exit and cleanup
  setTimeout(() => {
    console.log("\n--- Sending 'exit' ---")
    ptyProcess.write("exit\r")
  }, 1500)

  // Wait for exit
  ptyProcess.onExit(({ exitCode, signal }) => {
    console.log(`\n=== PTY exited with code ${exitCode}, signal ${signal} ===`)
    console.log("SUCCESS: node-pty works with Bun!")
    process.exit(0)
  })

  // Timeout fallback
  setTimeout(() => {
    console.log("\n=== Timeout - killing PTY ===")
    ptyProcess.kill()
    process.exit(1)
  }, 5000)

} catch (error) {
  console.error("FAILED:", error)
  process.exit(1)
}
