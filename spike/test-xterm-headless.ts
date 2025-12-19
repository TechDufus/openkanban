/**
 * Test 3: @xterm/headless for terminal emulation
 * 
 * xterm-headless provides terminal state management without a DOM
 * Combined with node-pty, this gives us full terminal emulation
 */

import * as pty from "@lydell/node-pty"
import { Terminal } from "@xterm/headless"

console.log("=== Testing @xterm/headless + node-pty ===\n")

try {
  // Create headless terminal
  const term = new Terminal({
    cols: 80,
    rows: 24,
    allowProposedApi: true,
  })

  console.log(`Created Terminal: ${term.cols}x${term.rows}`)

  // Create PTY
  const ptyProcess = pty.spawn("bash", [], {
    name: "xterm-256color",
    cols: term.cols,
    rows: term.rows,
    cwd: process.cwd(),
    env: process.env as Record<string, string>,
  })

  console.log(`Spawned PTY with PID: ${ptyProcess.pid}`)

  // Connect PTY output to terminal
  ptyProcess.onData((data: string) => {
    term.write(data)
  })

  // Helper to get terminal content
  function getTerminalContent(): string {
    const buffer = term.buffer.active
    let content = ""
    for (let i = 0; i < buffer.length; i++) {
      const line = buffer.getLine(i)
      if (line) {
        content += line.translateToString(true) + "\n"
      }
    }
    return content.trim()
  }

  // Send commands
  setTimeout(() => {
    console.log("\n--- Sending commands ---")
    ptyProcess.write("echo 'Hello from xterm-headless'\r")
  }, 500)

  setTimeout(() => {
    ptyProcess.write("pwd\r")
  }, 1000)

  setTimeout(() => {
    ptyProcess.write("ls -la | head -5\r")
  }, 1500)

  // Read terminal state
  setTimeout(() => {
    console.log("\n=== Terminal Buffer Content ===")
    console.log(getTerminalContent())
    console.log("=== End Buffer ===\n")
    
    // Get cursor position
    console.log(`Cursor position: (${term.buffer.active.cursorX}, ${term.buffer.active.cursorY})`)
    
    ptyProcess.write("exit\r")
  }, 2500)

  ptyProcess.onExit(({ exitCode }) => {
    console.log(`\nPTY exited with code ${exitCode}`)
    console.log("SUCCESS: @xterm/headless + node-pty integration works!")
    
    // Final buffer state
    console.log("\n=== Final Terminal State ===")
    console.log(getTerminalContent())
    
    term.dispose()
    process.exit(0)
  })

  // Timeout
  setTimeout(() => {
    console.log("Timeout - killing")
    ptyProcess.kill()
    term.dispose()
    process.exit(1)
  }, 10000)

} catch (error) {
  console.error("FAILED:", error)
  process.exit(1)
}
