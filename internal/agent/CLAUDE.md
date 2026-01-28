# Agent Package

Agent configuration, status detection, and spawn preparation.

## Agent Types

Configured in `config.json` under `agents` map:
- `claude`, `opencode`, `gemini`, `codex`, etc.
- Each has: `command`, `args`, `env`, `init_prompt`

## Session Detection

Find existing sessions to resume:
```go
FindOpencodeSession(workdir) string
FindGeminiSession(workdir) string
FindCodexSession(workdir) string
```

Returns session ID or empty string.

## Context Prompts

`BuildContextPrompt()` uses Go templates:
```go
type ContextData struct {
    TicketTitle       string
    TicketDescription string
    ProjectName       string
    // ...
}
```

Template in config: `"init_prompt": "Work on: {{.TicketTitle}}"`

## Status Detection

`StatusDetector` monitors agent state:
- File-based detection (marker files)
- API-based (HTTP to localhost)
- Terminal content parsing (keyword matching)

Keywords: "waiting", "thinking", "error", etc.

## OpenCode Server

Lifecycle management for opencode:
- `Start()` / `Stop()`
- `waitForReady()` with timeout
- HTTP client queries status API

## Thread Safety

Use `sync.RWMutex` for cache access in StatusDetector.

## Anti-Patterns

- Don't hardcode agent commands - use config
- Don't block on session detection - return empty on timeout
- Don't skip mutex for shared cache
- Don't assume agent process exists - check first
