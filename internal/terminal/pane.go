package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

const (
	// renderInterval limits redraws to ~20fps to prevent flicker from spinners
	renderInterval = 50 * time.Millisecond

	// defaultScrollback is the max lines to keep in scrollback buffer
	defaultScrollback = 10000

	// readBufferSize is the PTY read buffer size (large to reduce redraws)
	readBufferSize = 65536
)

// Pane represents an embedded terminal pane with PTY and vt10x emulation
type Pane struct {
	// Identity
	id string

	// Core terminal state
	vt      vt10x.Terminal
	pty     *os.File
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	exitErr error

	// Working directory for the command
	workdir string

	// Dimensions
	width  int
	height int

	// Render optimization
	cachedView      string
	lastRender      time.Time
	dirty           bool
	renderScheduled bool

	// Scrollback buffer (vt10x doesn't have built-in scrollback)
	scrollback    []string
	scrollOffset  int
	maxScrollback int
}

// New creates a new terminal pane with the given dimensions
func New(id string, width, height int) *Pane {
	return &Pane{
		id:            id,
		width:         width,
		height:        height,
		maxScrollback: defaultScrollback,
		scrollback:    make([]string, 0),
	}
}

// ID returns the pane's identifier
func (p *Pane) ID() string {
	return p.id
}

// SetWorkdir sets the working directory for commands
func (p *Pane) SetWorkdir(dir string) {
	p.workdir = dir
}

// Running returns whether the pane has a running process
func (p *Pane) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// ExitErr returns any error from the process exit
func (p *Pane) ExitErr() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

// SetSize updates the pane dimensions and resizes PTY/vt10x
func (p *Pane) SetSize(width, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.width = width
	p.height = height
	p.dirty = true
	p.cachedView = ""

	// Reset scroll to live view on resize
	p.scrollOffset = 0

	// Resize virtual terminal
	if p.vt != nil {
		p.vt.Resize(width, height)
	}

	// Resize PTY
	if p.pty != nil && p.running {
		pty.Setsize(p.pty, &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(width),
		})
	}
}

// Size returns the current dimensions
func (p *Pane) Size() (width, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.width, p.height
}

// --- Bubbletea Messages ---

// OutputMsg carries data read from the PTY
type OutputMsg struct {
	PaneID string
	Data   []byte
}

// ExitMsg indicates the process has exited
type ExitMsg struct {
	PaneID string
	Err    error
}

// RenderTickMsg triggers a throttled render
type RenderTickMsg struct {
	PaneID string
}

// ExitFocusMsg signals to return to board view
type ExitFocusMsg struct{}

// --- PTY Lifecycle (Issue #13) ---

// Start launches a command in a PTY and returns a Cmd to begin reading
func (p *Pane) Start(command string, args ...string) tea.Cmd {
	return func() tea.Msg {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Create virtual terminal with current dimensions
		p.vt = vt10x.New(vt10x.WithSize(p.width, p.height))

		// Build command
		p.cmd = exec.Command(command, args...)
		p.cmd.Env = buildCleanEnv()

		// Set working directory if specified
		if p.workdir != "" {
			p.cmd.Dir = p.workdir
		}

		// Start PTY
		ptmx, err := pty.Start(p.cmd)
		if err != nil {
			p.exitErr = err
			return ExitMsg{PaneID: p.id, Err: err}
		}
		p.pty = ptmx
		p.running = true
		p.exitErr = nil

		// Set PTY size
		pty.Setsize(p.pty, &pty.Winsize{
			Rows: uint16(p.height),
			Cols: uint16(p.width),
		})

		// Start read loop
		return p.readOutputUnlocked()()
	}
}

func (p *Pane) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	if p.pty != nil {
		p.pty.Close()
	}
	p.running = false
	return nil
}

var ErrPaneNotRunning = fmt.Errorf("pane is not running")

func (p *Pane) WriteInput(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.pty == nil {
		return 0, ErrPaneNotRunning
	}
	return p.pty.Write(data)
}

// readOutput returns a Cmd that reads from the PTY
func (p *Pane) readOutput() tea.Cmd {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.readOutputUnlocked()
}

// readOutputUnlocked must be called with mu held
func (p *Pane) readOutputUnlocked() tea.Cmd {
	if p.pty == nil {
		return nil
	}

	ptyFile := p.pty
	paneID := p.id

	return func() tea.Msg {
		buf := make([]byte, readBufferSize)
		n, err := ptyFile.Read(buf)
		if err != nil {
			return ExitMsg{PaneID: paneID, Err: err}
		}
		return OutputMsg{PaneID: paneID, Data: buf[:n]}
	}
}

// --- Update Handler ---

// Update handles messages for this pane, returns commands to execute
func (p *Pane) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case OutputMsg:
		if msg.PaneID != p.id {
			return nil
		}
		p.handleOutput(msg.Data)
		return tea.Batch(p.readOutput(), p.scheduleRenderTick())

	case RenderTickMsg:
		if msg.PaneID != p.id {
			return nil
		}
		p.mu.Lock()
		p.renderScheduled = false
		p.mu.Unlock()
		return nil

	case ExitMsg:
		if msg.PaneID != p.id {
			return nil
		}
		p.mu.Lock()
		p.running = false
		p.exitErr = msg.Err
		if p.pty != nil {
			p.pty.Close()
		}
		p.mu.Unlock()
		return nil
	}

	return nil
}

// handleOutput writes data to vt10x, capturing scrollback
func (p *Pane) handleOutput(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.vt == nil {
		return
	}

	// Capture top line before write for scrollback
	prevTopLine := p.getTopLineUnlocked()

	// Write to virtual terminal (parses ANSI sequences)
	p.vt.Write(data)
	p.dirty = true

	// Check if content scrolled
	newTopLine := p.getTopLineUnlocked()
	if newTopLine != prevTopLine && strings.TrimSpace(prevTopLine) != "" {
		p.addToScrollbackUnlocked(prevTopLine)
	}
}

// scheduleRenderTick returns a Cmd to trigger render after throttle interval
func (p *Pane) scheduleRenderTick() tea.Cmd {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.renderScheduled {
		return nil
	}
	p.renderScheduled = true

	timeSinceLastRender := time.Since(p.lastRender)
	delay := renderInterval - timeSinceLastRender
	if delay < 0 {
		delay = 0
	}

	paneID := p.id
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return RenderTickMsg{PaneID: paneID}
	})
}

// --- Key Handling (Issue #15) ---

// HandleKey processes a key event and sends to PTY
func (p *Pane) HandleKey(msg tea.KeyMsg) tea.Msg {
	if msg.String() == "ctrl+g" {
		return ExitFocusMsg{}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.pty == nil {
		return nil
	}

	input := p.translateKey(msg)
	if len(input) > 0 {
		p.pty.Write(input)
	}

	return nil
}

// translateKey converts Bubbletea KeyMsg to PTY byte sequences
func (p *Pane) translateKey(msg tea.KeyMsg) []byte {
	key := msg.String()

	// Handle modifier combinations
	switch {
	// Ctrl+A through Ctrl+Z → 0x01-0x1A
	case len(key) == 6 && key[:5] == "ctrl+" && key[5] >= 'a' && key[5] <= 'z':
		return []byte{byte(key[5] - 'a' + 1)}

	// Alt+letter → ESC + letter
	case len(key) == 5 && key[:4] == "alt+" && key[4] >= 'a' && key[4] <= 'z':
		return []byte{27, key[4]}
	}

	// Handle special keys
	switch msg.Type {
	case tea.KeyEnter:
		return []byte("\r")
	case tea.KeyBackspace:
		return []byte{127}
	case tea.KeyTab:
		if msg.Alt {
			return []byte("\x1b[Z") // Shift+Tab
		}
		return []byte("\t")
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	case tea.KeyEscape:
		return []byte{27}
	case tea.KeyHome:
		return []byte("\x1b[H")
	case tea.KeyEnd:
		return []byte("\x1b[F")
	case tea.KeyPgUp:
		return []byte("\x1b[5~")
	case tea.KeyPgDown:
		return []byte("\x1b[6~")
	case tea.KeyDelete:
		return []byte("\x1b[3~")
	case tea.KeySpace:
		return []byte(" ")
	case tea.KeyRunes:
		return []byte(string(msg.Runes))
	}

	return nil
}

// --- Scrollback (Issue #20) ---

// getTopLineUnlocked returns the top line of the terminal (must hold mu)
func (p *Pane) getTopLineUnlocked() string {
	if p.vt == nil {
		return ""
	}

	p.vt.Lock()
	defer p.vt.Unlock()

	cols, _ := p.vt.Size()
	var line strings.Builder
	for x := 0; x < cols; x++ {
		ch := p.vt.Cell(x, 0).Char
		if ch == 0 {
			ch = ' '
		}
		line.WriteRune(ch)
	}
	return strings.TrimRight(line.String(), " ")
}

// addToScrollbackUnlocked adds a line to scrollback with deduplication (must hold mu)
func (p *Pane) addToScrollbackUnlocked(line string) {
	// Deduplicate against recent entries
	checkCount := 20
	if checkCount > len(p.scrollback) {
		checkCount = len(p.scrollback)
	}
	for i := len(p.scrollback) - checkCount; i < len(p.scrollback); i++ {
		if i >= 0 && p.scrollback[i] == line {
			return
		}
	}

	p.scrollback = append(p.scrollback, line)
	if len(p.scrollback) > p.maxScrollback {
		// Trim from front
		p.scrollback = p.scrollback[len(p.scrollback)-p.maxScrollback:]
	}
}

// ScrollUp scrolls the view up by n lines
func (p *Pane) ScrollUp(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.scrollOffset += n
	if p.scrollOffset > len(p.scrollback) {
		p.scrollOffset = len(p.scrollback)
	}
	p.dirty = true
	p.cachedView = ""
}

// ScrollDown scrolls the view down by n lines (toward live view)
func (p *Pane) ScrollDown(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.scrollOffset -= n
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
	p.dirty = true
	p.cachedView = ""
}

// ScrollToBottom returns to live view
func (p *Pane) ScrollToBottom() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.scrollOffset = 0
	p.dirty = true
	p.cachedView = ""
}

// --- Rendering (Issue #14) ---

// View returns the rendered terminal content
func (p *Pane) View() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return cached view if not dirty
	if !p.dirty && p.cachedView != "" {
		return p.cachedView
	}

	p.cachedView = p.renderVTUnlocked()
	p.lastRender = time.Now()
	p.dirty = false
	return p.cachedView
}

// renderVTUnlocked renders the vt10x state to a string (must hold mu)
func (p *Pane) renderVTUnlocked() string {
	if p.vt == nil {
		return "Terminal not initialized"
	}

	p.vt.Lock()
	defer p.vt.Unlock()

	cols, rows := p.vt.Size()
	if cols <= 0 || rows <= 0 {
		return ""
	}

	// If scrolled up, show scrollback + partial screen
	if p.scrollOffset > 0 && len(p.scrollback) > 0 {
		return p.renderWithScrollbackUnlocked(cols, rows)
	}

	// Live view - render current vt screen
	return p.renderLiveScreenUnlocked(cols, rows)
}

// renderLiveScreenUnlocked renders the live terminal screen (must hold mu and vt.Lock)
func (p *Pane) renderLiveScreenUnlocked(cols, rows int) string {
	cursor := p.vt.Cursor()
	cursorVisible := p.vt.CursorVisible()

	var result strings.Builder
	result.Grow(rows * cols * 2)

	for row := 0; row < rows; row++ {
		if row > 0 {
			result.WriteByte('\n')
		}

		// Track current style for batching
		var currentFG, currentBG vt10x.Color
		var currentMode int16
		var batch strings.Builder
		firstCell := true

		flushBatch := func() {
			if batch.Len() == 0 {
				return
			}
			result.WriteString(buildANSI(currentFG, currentBG, currentMode, false))
			result.WriteString(batch.String())
			result.WriteString("\x1b[0m")
			batch.Reset()
		}

		for col := 0; col < cols; col++ {
			glyph := p.vt.Cell(col, row)
			ch := glyph.Char
			if ch == 0 {
				ch = ' '
			}

			isCursor := cursorVisible && col == cursor.X && row == cursor.Y

			// Style changed? Flush batch
			if !firstCell && (glyph.FG != currentFG || glyph.BG != currentBG ||
				glyph.Mode != currentMode || isCursor) {
				flushBatch()
			}

			// Handle cursor with reverse video
			if isCursor {
				result.WriteString("\x1b[7m") // Reverse
				result.WriteRune(ch)
				result.WriteString("\x1b[27m") // Un-reverse
				firstCell = true
				continue
			}

			currentFG = glyph.FG
			currentBG = glyph.BG
			currentMode = glyph.Mode
			firstCell = false

			batch.WriteRune(ch)
		}
		flushBatch()
	}

	return result.String()
}

// renderWithScrollbackUnlocked renders scrollback + partial live view (must hold mu and vt.Lock)
func (p *Pane) renderWithScrollbackUnlocked(cols, rows int) string {
	var lines []string

	// Calculate which scrollback lines to show
	scrollbackStart := len(p.scrollback) - p.scrollOffset
	if scrollbackStart < 0 {
		scrollbackStart = 0
	}

	// Add scrollback lines
	for i := scrollbackStart; i < len(p.scrollback) && len(lines) < rows; i++ {
		lines = append(lines, p.scrollback[i])
	}

	// Fill remaining with live screen if needed
	liveRows := rows - len(lines)
	if liveRows > 0 {
		_, vtRows := p.vt.Size()
		for row := 0; row < liveRows && row < vtRows; row++ {
			var line strings.Builder
			for col := 0; col < cols; col++ {
				ch := p.vt.Cell(col, row).Char
				if ch == 0 {
					ch = ' '
				}
				line.WriteRune(ch)
			}
			lines = append(lines, strings.TrimRight(line.String(), " "))
		}
	}

	// Pad to full height
	for len(lines) < rows {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// buildANSI constructs ANSI escape sequence for given colors/mode
func buildANSI(fg, bg vt10x.Color, mode int16, isCursor bool) string {
	var parts []string

	// Foreground
	if fgCode := colorToANSI(fg, true); fgCode != "" {
		parts = append(parts, fgCode)
	}

	// Background
	if bgCode := colorToANSI(bg, false); bgCode != "" {
		parts = append(parts, bgCode)
	}

	// Attributes
	if mode&0x04 != 0 { // Bold
		parts = append(parts, "1")
	}
	if mode&0x10 != 0 { // Italic
		parts = append(parts, "3")
	}
	if mode&0x02 != 0 { // Underline
		parts = append(parts, "4")
	}
	if mode&0x01 != 0 { // Reverse
		parts = append(parts, "7")
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("\x1b[%sm", strings.Join(parts, ";"))
}

// colorToANSI converts vt10x.Color to ANSI escape sequence component
func colorToANSI(c vt10x.Color, isFG bool) string {
	// Default color (special value)
	if c >= 0x01000000 {
		return ""
	}

	base := 38 // Foreground
	if !isFG {
		base = 48 // Background
	}

	// ANSI 256-color palette (0-255)
	if c < 256 {
		return fmt.Sprintf("%d;5;%d", base, c)
	}

	// True color RGB (encoded as r<<16 | g<<8 | b)
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF
	return fmt.Sprintf("%d;2;%d;%d;%d", base, r, g, b)
}

func buildCleanEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		key := strings.Split(e, "=")[0]
		if key == "OPENCODE" || strings.HasPrefix(key, "OPENCODE_") {
			continue
		}
		if key == "CLAUDE" || strings.HasPrefix(key, "CLAUDE_") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "TERM=xterm-256color")
	return env
}
