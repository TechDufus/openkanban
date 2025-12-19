console.log("=== Testing Bun.Terminal (Native PTY) ===\n")

async function testBunTerminal() {
  const output: string[] = []
  
  // @ts-ignore - Bun.Terminal might not be in types yet
  if (typeof Bun.Terminal === "undefined") {
    console.log("Bun.Terminal not available in this version")
    console.log("Bun version:", Bun.version)
    return
  }

  // @ts-ignore
  await using terminal = new Bun.Terminal({
    cols: 80,
    rows: 24,
    data(_term: unknown, data: Uint8Array) {
      const text = new TextDecoder().decode(data)
      output.push(text)
      process.stdout.write(text)
    },
    exit(_term: unknown, exitCode: number) {
      console.log(`\nPTY exited with code ${exitCode}`)
    },
  })

  const proc = Bun.spawn(["bash", "--norc", "--noprofile"], {
    // @ts-ignore
    terminal,
    env: { ...process.env, PS1: "$ " },
  })

  terminal.write("echo 'Hello from Bun.Terminal!'\n")
  terminal.write("pwd\n")
  terminal.write("exit\n")

  await proc.exited

  console.log("\n=== SUCCESS: Bun.Terminal works! ===")
  console.log("Captured lines:", output.length)
}

testBunTerminal().catch(console.error)
