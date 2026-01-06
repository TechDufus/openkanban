# OpenKanban Demo
#
# Dependencies: vhs, ffmpeg, ttyd
# Run with: VHS_NO_SANDBOX=true vhs docs/assets/demo.tape
#
# Uses OPENKANBAN_CONFIG_DIR for isolation - safe to run anywhere.
# Uses $OKBIN to reference the locally built binary (run from repo root).
#
# VHS QUIRK: VHS cannot send Ctrl+S (terminal interprets it as XOFF flow control).
# The app accepts both Ctrl+S and Ctrl+X to save forms. Real users use Ctrl+S,
# but this demo uses Ctrl+X. See internal/ui/model.go handleTicketForm().
# DO NOT remove Ctrl+X support from the app - it breaks this demo.

Output docs/assets/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set Padding 20

# Setup isolated environment
Hide
Type "export OPENKANBAN_CONFIG_DIR=$(mktemp -d)"
Enter
Type "export DEMO_REPO=$(mktemp -d)"
Enter
Type "export OKBIN=$(pwd)/openkanban"
Enter
Type "cd $DEMO_REPO && git init -q && echo '# Demo' > README.md && git add . && git commit -q -m 'init'"
Enter
Type "$OKBIN new 'My App'"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 200ms
Show

# Launch OpenKanban
Type "$OKBIN"
Sleep 300ms
Enter
Sleep 2s

# Create first ticket
Type "n"
Sleep 600ms
Type "Add user authentication"
Tab
Sleep 400ms
Type "Implement JWT-based auth with refresh tokens"
Sleep 400ms
Ctrl+x
Sleep 1s

# Create second ticket
Type "n"
Sleep 600ms
Type "Setup database schema"
Tab
Sleep 400ms
Type "PostgreSQL with migrations"
Sleep 400ms
Ctrl+x
Sleep 1s

# Create third ticket
Type "n"
Sleep 600ms
Type "Build REST API endpoints"
Tab
Sleep 400ms
Type "CRUD operations for all resources"
Sleep 400ms
Ctrl+x
Sleep 1s

# Navigate to first ticket and move to In Progress
Type "k"
Sleep 400ms
Type "k"
Sleep 400ms
Type " "
Sleep 800ms

# Spawn agent on first ticket
Type "s"
Sleep 3s
Hide
Sleep 12s
Show
Sleep 2s

# Type a message in the first agent
Type "Hello from ticket ONE!"
Enter
Sleep 2s

# Detach from agent
Ctrl+g
Sleep 1.5s

# Move another ticket to In Progress
Type "h"
Sleep 400ms
Type " "
Sleep 800ms

# Spawn second agent
Type "s"
Sleep 3s
Hide
Sleep 12s
Show
Sleep 2s

# Type a different message in the second agent
Type "This is ticket TWO - a different session!"
Enter
Sleep 2s

# Detach
Ctrl+g
Sleep 1.5s

# Re-attach to FIRST agent to show it's still there
Type "k"
Sleep 400ms
Enter
Sleep 2s

# Detach
Ctrl+g
Sleep 1s

# Attach to SECOND agent to show it's different
Type "j"
Sleep 400ms
Enter
Sleep 2s

# Detach and show final board with both agents running
Ctrl+g
Sleep 2s

# Quit
Type "q"
Sleep 500ms
Type "y"
Sleep 500ms
