# PTY Spike Results

> **Date:** 2025-12-19  
> **Verdict:** GO - Multiple working PTY solutions found

## Executive Summary

We tested multiple approaches for spawning and managing terminal processes in Bun/TypeScript for the OpenKanban OpenTUI migration. **Two approaches work reliably.**

## Test Results

| Approach | Status | Pros | Cons |
|----------|--------|------|------|
| `Bun.Terminal` | ✅ **WORKS** | Native, no deps, built into Bun 1.3.5+ | POSIX only (no Windows yet) |
| `@lydell/node-pty` | ✅ **WORKS** | Cross-platform, well-tested | Extra dependency |
| `@xterm/headless` | ✅ **WORKS** | Full terminal emulation, buffer access | Needs PTY backend |
| `node-pty` | ❌ FAILS | - | V8 symbol incompatibility with Bun |
| `Bun.spawn` | ⚠️ LIMITED | Simple, no deps | No PTY features |

## Recommended Stack

```
┌─────────────────────────────────────────────┐
│            OpenKanban TypeScript            │
├─────────────────────────────────────────────┤
│  Option A: Bun.Terminal (Native)            │
│  - Zero dependencies                        │
│  - Built into Bun 1.3.5+                    │
│  - POSIX only (Linux/macOS)                 │
├─────────────────────────────────────────────┤
│  Option B: @lydell/node-pty + xterm-headless│
│  - Cross-platform including Windows         │
│  - Full terminal emulation                  │
│  - Buffer access for rendering              │
└─────────────────────────────────────────────┘
```

## Test Files

| File | Purpose |
|------|---------|
| `test-bun-terminal.ts` | Native Bun.Terminal PTY |
| `test-lydell-pty.ts` | @lydell/node-pty with Bun |
| `test-xterm-headless.ts` | Full terminal emulation |
| `test-bun-spawn.ts` | Basic process spawning |
| `test-node-pty.ts` | Original node-pty (fails) |

## Running Tests

```bash
bun run test-bun-terminal.ts    # Native PTY
bun run test-lydell-pty.ts      # @lydell/node-pty
bun run test-xterm-headless.ts  # Full emulation
```

## Key Findings

### 1. Bun.Terminal (Native PTY)

Bun 1.3.5+ has **built-in PTY support**:

```typescript
await using terminal = new Bun.Terminal({
  cols: 80,
  rows: 24,
  data(term, data) {
    console.log(new TextDecoder().decode(data))
  },
  exit(term, exitCode) {
    console.log(`Exited: ${exitCode}`)
  },
})

const proc = Bun.spawn(["bash"], { terminal })
terminal.write("echo hello\n")
await proc.exited
```

### 2. @lydell/node-pty

Fork of node-pty that works with Bun:

```typescript
import * as pty from "@lydell/node-pty"

const ptyProcess = pty.spawn("bash", [], {
  name: "xterm-256color",
  cols: 80,
  rows: 24,
})

ptyProcess.onData((data) => console.log(data))
ptyProcess.write("echo hello\r")
```

### 3. @xterm/headless (Terminal Emulation)

For rendering terminal state without a DOM:

```typescript
import { Terminal } from "@xterm/headless"
import * as pty from "@lydell/node-pty"

const term = new Terminal({ cols: 80, rows: 24 })
const ptyProcess = pty.spawn("bash", [])

ptyProcess.onData((data) => term.write(data))

// Get terminal buffer content
const buffer = term.buffer.active
for (let i = 0; i < buffer.length; i++) {
  console.log(buffer.getLine(i)?.translateToString())
}
```

## OpenCode's Approach

From research, OpenCode uses:

1. **Backend**: `bun-pty` (Rust FFI) for PTY spawning
2. **Transport**: WebSocket via Hono server
3. **Frontend**: `ghostty-web` (WASM) for rendering

For OpenKanban, we can use a simpler approach since we're not doing client/server:

- **Recommended**: `Bun.Terminal` (native, zero deps)
- **Fallback**: `@lydell/node-pty` (if Windows needed)

## Next Steps

1. ✅ PTY spike complete - GO decision
2. Proceed with Phase 1: Project Setup
3. Use `Bun.Terminal` as primary PTY solution
4. Keep `@lydell/node-pty` as fallback for cross-platform
