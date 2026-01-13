package terminal

import (
	"bytes"
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
	renderInterval = 50 * time.Millisecond
	readBufferSize = 65536
)

type Pane struct {
	id      string
	vt      vt10x.Terminal
	pty     *os.File
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	exitErr     error
	workdir     string
	sessionName string
	width       int
	height      int

	cachedView      string
	lastRender      time.Time
	dirty           bool
	renderScheduled bool

	mouseEnabled bool // tracks if child process has enabled mouse tracking
}

func New(id string, width, height int) *Pane {
	return &Pane{
		id:     id,
		width:  width,
		height: height,
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

func (p *Pane) GetWorkdir() string {
	return p.workdir
}

// SetSessionName sets the session name for OPENKANBAN_SESSION env var
func (p *Pane) SetSessionName(name string) {
	p.sessionName = name
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

func (p *Pane) SetSize(width, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.width = width
	p.height = height
	p.dirty = true
	p.cachedView = ""

	if p.vt != nil {
		p.vt.Resize(width, height)
	}

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
		p.cmd.Env = buildCleanEnv(p.sessionName)

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

// StopGraceful sends SIGTERM, waits for timeout, then SIGKILL if needed.
func (p *Pane) StopGraceful(timeout time.Duration) error {
	p.mu.Lock()
	if !p.running || p.cmd == nil || p.cmd.Process == nil {
		p.mu.Unlock()
		return nil
	}

	proc := p.cmd.Process
	p.mu.Unlock()

	if err := proc.Signal(os.Interrupt); err != nil {
		return p.Stop()
	}

	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		proc.Kill()
	}

	p.mu.Lock()
	if p.pty != nil {
		p.pty.Close()
	}
	p.running = false
	p.mu.Unlock()

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

func (p *Pane) handleOutput(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.vt == nil {
		return
	}

	p.detectMouseModeChanges(data)
	p.vt.Write(data)
	p.dirty = true
}

// detectMouseModeChanges scans output for mouse tracking mode escape sequences.
// Called with mutex held.
func (p *Pane) detectMouseModeChanges(data []byte) {
	// Mouse tracking enable sequences (any of these enables mouse mode)
	enableSeqs := [][]byte{
		[]byte("\x1b[?1000h"), // X10 mouse tracking
		[]byte("\x1b[?1002h"), // Button-event tracking
		[]byte("\x1b[?1003h"), // Any-event tracking
		[]byte("\x1b[?1006h"), // SGR extended mode
	}

	// Mouse tracking disable sequences
	disableSeqs := [][]byte{
		[]byte("\x1b[?1000l"),
		[]byte("\x1b[?1002l"),
		[]byte("\x1b[?1003l"),
		[]byte("\x1b[?1006l"),
	}

	// Check for enable sequences
	for _, seq := range enableSeqs {
		if bytes.Contains(data, seq) {
			p.mouseEnabled = true
			return
		}
	}

	// Check for disable sequences
	for _, seq := range disableSeqs {
		if bytes.Contains(data, seq) {
			p.mouseEnabled = false
			return
		}
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

func (p *Pane) HandleMouse(msg tea.MouseMsg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.pty == nil {
		return
	}

	// Block all motion events
	if msg.Action == tea.MouseActionMotion {
		return
	}

	var seq []byte

	// Handle scroll wheel - convert to Page Up/Down when mouse mode not enabled
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if p.mouseEnabled {
			x, y := msg.X+1, msg.Y+1
			if x > 223 {
				x = 223
			}
			if y > 223 {
				y = 223
			}
			seq = []byte{'\x1b', '[', 'M', byte(64 + 32), byte(x + 32), byte(y + 32)}
		} else {
			// Page Up - works in most CLI apps for scrolling
			seq = []byte{'\x1b', '[', '5', '~'}
		}
	case tea.MouseButtonWheelDown:
		if p.mouseEnabled {
			x, y := msg.X+1, msg.Y+1
			if x > 223 {
				x = 223
			}
			if y > 223 {
				y = 223
			}
			seq = []byte{'\x1b', '[', 'M', byte(65 + 32), byte(x + 32), byte(y + 32)}
		} else {
			// Page Down - works in most CLI apps for scrolling
			seq = []byte{'\x1b', '[', '6', '~'}
		}
	// Button clicks: only forward when mouse mode enabled
	case tea.MouseButtonLeft, tea.MouseButtonRight, tea.MouseButtonMiddle:
		if p.mouseEnabled && msg.Action == tea.MouseActionPress {
			x, y := msg.X+1, msg.Y+1
			if x > 223 {
				x = 223
			}
			if y > 223 {
				y = 223
			}
			btn := byte(0) // left
			if msg.Button == tea.MouseButtonMiddle {
				btn = 1
			} else if msg.Button == tea.MouseButtonRight {
				btn = 2
			}
			seq = []byte{'\x1b', '[', 'M', byte(btn + 32), byte(x + 32), byte(y + 32)}
		}
	}

	if len(seq) > 0 {
		p.pty.Write(seq)
	}
}

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

// GetContent returns the current terminal content as plain text for analysis.
func (p *Pane) GetContent() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.vt == nil {
		return ""
	}

	p.vt.Lock()
	defer p.vt.Unlock()

	cols, rows := p.vt.Size()
	if cols <= 0 || rows <= 0 {
		return ""
	}

	var result strings.Builder
	for row := 0; row < rows; row++ {
		if row > 0 {
			result.WriteByte('\n')
		}
		for col := 0; col < cols; col++ {
			ch := p.vt.Cell(col, row).Char
			if ch == 0 {
				ch = ' '
			}
			result.WriteRune(ch)
		}
	}

	return result.String()
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

func buildCleanEnv(sessionName string) []string {
	var env []string
	for _, e := range os.Environ() {
		key := strings.Split(e, "=")[0]
		if key == "OPENCODE" || strings.HasPrefix(key, "OPENCODE_") {
			continue
		}
		if key == "CLAUDE" || strings.HasPrefix(key, "CLAUDE_") {
			continue
		}
		if key == "GEMINI" || strings.HasPrefix(key, "GEMINI_") {
			continue
		}
		if key == "CODEX" || strings.HasPrefix(key, "CODEX_") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "TERM=xterm-256color")
	if sessionName != "" {
		env = append(env, "OPENKANBAN_SESSION="+sessionName)
	}
	return env
}
