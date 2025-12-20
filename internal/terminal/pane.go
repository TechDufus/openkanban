package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/techdufus/openkanban/internal/daemon"
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
	exitErr error
	workdir string
	width   int
	height  int

	cachedView string
	lastRender time.Time
	dirty      bool

	daemonClient *daemon.Client
	daemonMode   bool
	sessionID    string

	readerCtx    context.Context
	readerCancel context.CancelFunc
	readerDone   chan struct{}
	lifecycleCh  chan tea.Msg
}

func New(id string, width, height int) *Pane {
	return &Pane{
		id:          id,
		width:       width,
		height:      height,
		lifecycleCh: make(chan tea.Msg, 1),
	}
}

func (p *Pane) ID() string {
	return p.id
}

func (p *Pane) SetWorkdir(dir string) {
	p.workdir = dir
}

func (p *Pane) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

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

	if p.daemonMode && p.daemonClient != nil && p.running {
		p.daemonClient.Resize(uint16(height), uint16(width))
		return
	}

	if p.pty != nil && p.running {
		pty.Setsize(p.pty, &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(width),
		})
	}
}

func (p *Pane) Size() (width, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.width, p.height
}

type PaneStartedMsg struct {
	PaneID string
}

type PaneStoppedMsg struct {
	PaneID string
	Err    error
}

type RenderTickMsg struct {
	PaneID string
}

type ExitFocusMsg struct{}

func (p *Pane) Start(command string, args ...string) tea.Cmd {
	return func() tea.Msg {
		p.mu.Lock()

		width, height := p.width, p.height
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}
		p.width, p.height = width, height

		p.vt = vt10x.New(vt10x.WithSize(width, height))

		p.cmd = exec.Command(command, args...)
		p.cmd.Env = buildCleanEnv()

		if p.workdir != "" {
			p.cmd.Dir = p.workdir
		}

		ptmx, err := pty.Start(p.cmd)
		if err != nil {
			p.exitErr = err
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}
		p.pty = ptmx
		p.running = true
		p.exitErr = nil

		pty.Setsize(p.pty, &pty.Winsize{
			Rows: uint16(p.height),
			Cols: uint16(p.width),
		})

		p.readerCtx, p.readerCancel = context.WithCancel(context.Background())
		p.readerDone = make(chan struct{})

		p.mu.Unlock()

		go p.ptyReaderLoop()

		return PaneStartedMsg{PaneID: p.id}
	}
}

func (p *Pane) StartDaemon(sessionID, workdir, command string, args []string) tea.Cmd {
	return func() tea.Msg {
		p.mu.Lock()

		width, height := p.width, p.height
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}
		p.width, p.height = width, height

		p.vt = vt10x.New(vt10x.WithSize(width, height))

		client := daemon.NewClient("")
		if err := client.Connect(); err != nil {
			p.exitErr = err
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}

		if err := client.CreateSession(sessionID, workdir, command, args); err != nil {
			client.Close()
			p.exitErr = err
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}

		p.daemonClient = client
		p.daemonMode = true
		p.sessionID = sessionID
		p.running = true
		p.exitErr = nil

		if err := client.Resize(uint16(p.height), uint16(p.width)); err != nil {
			p.cleanupDaemon()
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}

		p.readerCtx, p.readerCancel = context.WithCancel(context.Background())
		p.readerDone = make(chan struct{})

		p.mu.Unlock()

		go p.daemonReaderLoop()

		return PaneStartedMsg{PaneID: p.id}
	}
}

func (p *Pane) AttachDaemon(sessionID string) tea.Cmd {
	return func() tea.Msg {
		p.mu.Lock()

		width, height := p.width, p.height
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}
		p.width, p.height = width, height

		p.vt = vt10x.New(vt10x.WithSize(width, height))

		client := daemon.NewClient("")
		if err := client.Connect(); err != nil {
			p.exitErr = err
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}

		if err := client.AttachSession(sessionID); err != nil {
			client.Close()
			p.exitErr = err
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}

		p.daemonClient = client
		p.daemonMode = true
		p.sessionID = sessionID
		p.running = true
		p.exitErr = nil

		if err := client.Resize(uint16(p.height), uint16(p.width)); err != nil {
			p.cleanupDaemon()
			p.mu.Unlock()
			return PaneStoppedMsg{PaneID: p.id, Err: err}
		}

		p.readerCtx, p.readerCancel = context.WithCancel(context.Background())
		p.readerDone = make(chan struct{})

		p.mu.Unlock()

		go p.daemonReaderLoop()

		return PaneStartedMsg{PaneID: p.id}
	}
}

func (p *Pane) ptyReaderLoop() {
	defer close(p.readerDone)

	buf := make([]byte, readBufferSize)
	for {
		select {
		case <-p.readerCtx.Done():
			return
		default:
		}

		n, err := p.pty.Read(buf)
		if err != nil {
			p.mu.Lock()
			p.running = false
			p.exitErr = err
			p.mu.Unlock()

			select {
			case p.lifecycleCh <- PaneStoppedMsg{PaneID: p.id, Err: err}:
			default:
			}
			return
		}

		if n > 0 {
			p.mu.Lock()
			vt := p.vt
			p.mu.Unlock()

			if vt != nil {
				vt.Write(buf[:n])

				p.mu.Lock()
				p.dirty = true
				p.mu.Unlock()
			}
		}
	}
}

func (p *Pane) daemonReaderLoop() {
	defer close(p.readerDone)

	for {
		select {
		case <-p.readerCtx.Done():
			return
		default:
		}

		p.mu.Lock()
		client := p.daemonClient
		p.mu.Unlock()

		if client == nil {
			return
		}

		msgType, data, err := client.ReadMessage()
		if err != nil {
			p.mu.Lock()
			p.cleanupDaemon()
			p.exitErr = err
			p.mu.Unlock()

			select {
			case p.lifecycleCh <- PaneStoppedMsg{PaneID: p.id, Err: err}:
			default:
			}
			return
		}

		switch msgType {
		case daemon.MsgData:
			if len(data) > 0 {
				p.mu.Lock()
				vt := p.vt
				p.mu.Unlock()

				if vt != nil {
					vt.Write(data)

					p.mu.Lock()
					p.dirty = true
					p.mu.Unlock()
				}
			}
		case daemon.MsgExit:
			p.mu.Lock()
			p.cleanupDaemon()
			p.mu.Unlock()

			select {
			case p.lifecycleCh <- PaneStoppedMsg{PaneID: p.id, Err: io.EOF}:
			default:
			}
			return
		}
	}
}

func (p *Pane) WaitForLifecycle() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-p.lifecycleCh
		if !ok {
			return nil
		}
		return msg
	}
}

func (p *Pane) cleanupDaemon() {
	p.daemonMode = false
	p.running = false
	if p.daemonClient != nil {
		p.daemonClient.Close()
		p.daemonClient = nil
	}
}

func (p *Pane) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	if p.readerCancel != nil {
		p.readerCancel()
	}

	if p.readerDone != nil {
		select {
		case <-p.readerDone:
		case <-time.After(500 * time.Millisecond):
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.daemonMode {
		p.cleanupDaemon()
	} else {
		if p.cmd != nil && p.cmd.Process != nil {
			p.cmd.Process.Kill()
		}
		if p.pty != nil {
			p.pty.Close()
			p.pty = nil
		}
	}

	return nil
}

var ErrPaneNotRunning = fmt.Errorf("pane is not running")

func (p *Pane) WriteInput(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return 0, ErrPaneNotRunning
	}

	if p.daemonMode && p.daemonClient != nil {
		return p.daemonClient.Write(data)
	}

	if p.pty == nil {
		return 0, ErrPaneNotRunning
	}
	return p.pty.Write(data)
}

func (p *Pane) IsDaemonMode() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.daemonMode
}

func (p *Pane) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case RenderTickMsg:
		if msg.PaneID != p.id {
			return nil
		}
		return nil

	case PaneStoppedMsg:
		if msg.PaneID != p.id {
			return nil
		}
		return nil
	}

	return nil
}

func (p *Pane) ScheduleRenderTick() tea.Cmd {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.scheduleRenderTickUnlocked()
}

func (p *Pane) scheduleRenderTickUnlocked() tea.Cmd {
	paneID := p.id
	return tea.Tick(renderInterval, func(time.Time) tea.Msg {
		return RenderTickMsg{PaneID: paneID}
	})
}

func (p *Pane) HandleMouse(msg tea.MouseMsg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	var seq []byte
	x, y := msg.X+1, msg.Y+1
	if x > 223 {
		x = 223
	}
	if y > 223 {
		y = 223
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		seq = []byte{'\x1b', '[', 'M', byte(64 + 32), byte(x + 32), byte(y + 32)}
	case tea.MouseButtonWheelDown:
		seq = []byte{'\x1b', '[', 'M', byte(65 + 32), byte(x + 32), byte(y + 32)}
	case tea.MouseButtonLeft:
		seq = []byte{'\x1b', '[', 'M', byte(0 + 32), byte(x + 32), byte(y + 32)}
	case tea.MouseButtonRight:
		seq = []byte{'\x1b', '[', 'M', byte(2 + 32), byte(x + 32), byte(y + 32)}
	case tea.MouseButtonMiddle:
		seq = []byte{'\x1b', '[', 'M', byte(1 + 32), byte(x + 32), byte(y + 32)}
	}

	if len(seq) > 0 {
		if p.daemonMode && p.daemonClient != nil {
			p.daemonClient.Write(seq)
		} else if p.pty != nil {
			p.pty.Write(seq)
		}
	}
}

func (p *Pane) HandleKey(msg tea.KeyMsg) tea.Msg {
	if msg.String() == "ctrl+g" {
		return ExitFocusMsg{}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	input := p.translateKey(msg)
	if len(input) > 0 {
		if p.daemonMode && p.daemonClient != nil {
			p.daemonClient.Write(input)
		} else if p.pty != nil {
			p.pty.Write(input)
		}
	}

	return nil
}

func (p *Pane) translateKey(msg tea.KeyMsg) []byte {
	key := msg.String()

	switch {
	case len(key) == 6 && key[:5] == "ctrl+" && key[5] >= 'a' && key[5] <= 'z':
		return []byte{byte(key[5] - 'a' + 1)}
	case len(key) == 5 && key[:4] == "alt+" && key[4] >= 'a' && key[4] <= 'z':
		return []byte{27, key[4]}
	}

	switch msg.Type {
	case tea.KeyEnter:
		return []byte("\r")
	case tea.KeyBackspace:
		return []byte{127}
	case tea.KeyTab:
		if msg.Alt {
			return []byte("\x1b[Z")
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

func (p *Pane) View() string {
	if !p.mu.TryLock() {
		return p.cachedView
	}
	defer p.mu.Unlock()

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

func (p *Pane) renderLiveScreenUnlocked(cols, rows int) string {
	cursor := p.vt.Cursor()
	cursorVisible := p.vt.CursorVisible()

	var result strings.Builder
	result.Grow(rows * cols * 2)

	for row := 0; row < rows; row++ {
		if row > 0 {
			result.WriteByte('\n')
		}

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

			if !firstCell && (glyph.FG != currentFG || glyph.BG != currentBG ||
				glyph.Mode != currentMode || isCursor) {
				flushBatch()
			}

			if isCursor {
				result.WriteString("\x1b[7m")
				result.WriteRune(ch)
				result.WriteString("\x1b[27m")
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

func buildANSI(fg, bg vt10x.Color, mode int16, isCursor bool) string {
	var parts []string

	if fgCode := colorToANSI(fg, true); fgCode != "" {
		parts = append(parts, fgCode)
	}

	if bgCode := colorToANSI(bg, false); bgCode != "" {
		parts = append(parts, bgCode)
	}

	if mode&0x04 != 0 {
		parts = append(parts, "1")
	}
	if mode&0x10 != 0 {
		parts = append(parts, "3")
	}
	if mode&0x02 != 0 {
		parts = append(parts, "4")
	}
	if mode&0x01 != 0 {
		parts = append(parts, "7")
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("\x1b[%sm", strings.Join(parts, ";"))
}

func colorToANSI(c vt10x.Color, isFG bool) string {
	if c >= 0x01000000 {
		return ""
	}

	base := 38
	if !isFG {
		base = 48
	}

	if c < 256 {
		return fmt.Sprintf("%d;5;%d", base, c)
	}

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
