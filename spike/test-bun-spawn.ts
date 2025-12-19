/**
 * Test 2: Bun.spawn with stdio
 * 
 * Test if Bun's native spawn can be used for interactive processes
 */

console.log("=== Testing Bun.spawn for interactive processes ===\n")

async function testBunSpawn() {
  try {
    // Test 1: Simple spawn with pipe
    console.log("Test 1: Bun.spawn with piped stdio")
    
    const proc = Bun.spawn(["bash", "-c", "echo 'Hello from Bun.spawn' && pwd && echo 'Done'"], {
      stdout: "pipe",
      stderr: "pipe",
    })

    const output = await new Response(proc.stdout).text()
    console.log("Output:", output)
    
    const exitCode = await proc.exited
    console.log(`Exit code: ${exitCode}`)
    console.log("SUCCESS: Basic Bun.spawn works\n")

    // Test 2: Interactive spawn (may not work without PTY)
    console.log("Test 2: Attempting interactive Bun.spawn")
    
    const interactiveProc = Bun.spawn(["bash"], {
      stdin: "pipe",
      stdout: "pipe",
      stderr: "pipe",
    })

    // Write commands
    const writer = interactiveProc.stdin.getWriter()
    await writer.write(new TextEncoder().encode("echo 'interactive test'\n"))
    await writer.write(new TextEncoder().encode("exit\n"))
    await writer.close()

    const interactiveOutput = await new Response(interactiveProc.stdout).text()
    console.log("Interactive output:", interactiveOutput)
    
    const interactiveExitCode = await interactiveProc.exited
    console.log(`Interactive exit code: ${interactiveExitCode}`)
    
    if (interactiveOutput.includes("interactive test")) {
      console.log("SUCCESS: Interactive Bun.spawn works (but no PTY features)\n")
    } else {
      console.log("PARTIAL: Bun.spawn works but output buffering differs\n")
    }

  } catch (error) {
    console.error("FAILED:", error)
  }
}

// Test 3: Check if Bun has PTY support in its API
console.log("Test 3: Checking Bun API for PTY features")
console.log("Bun.spawn options:", Object.keys(Bun.spawn))
console.log("Note: As of Bun 1.x, native PTY is not supported - need node-pty\n")

await testBunSpawn()
