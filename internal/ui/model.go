package ui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/techdufus/openkanban/internal/agent"
	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
	"github.com/techdufus/openkanban/internal/git"
	"github.com/techdufus/openkanban/internal/terminal"
)

// Mode represents the current UI mode
type Mode string

const (
	ModeNormal       Mode = "NORMAL"
	ModeInsert       Mode = "INSERT"
	ModeCommand      Mode = "COMMAND"
	ModeHelp         Mode = "HELP"
	ModeConfirm      Mode = "CONFIRM"
	ModeCreateTicket Mode = "CREATE"
	ModeEditTicket   Mode = "EDIT"
	ModeAgentView    Mode = "AGENT"
	ModeSettings     Mode = "SETTINGS"
)

const (
	minColumnWidth = 20
	columnOverhead = 5 // border (2) + padding (2) + margin (1)

	formFieldTitle       = 0
	formFieldDescription = 1
	formFieldBranch      = 2
)

// Model is the main Bubbletea model
type Model struct {
	// Configuration
	config *config.Config

	// Data
	board    *board.Board
	boardDir string

	// Managers
	agentMgr    *agent.Manager
	worktreeMgr *git.WorktreeManager

	// UI state
	mode         Mode
	activeColumn int
	activeTicket int
	width        int
	height       int
	spinner      spinner.Model
	scrollOffset int

	// Mouse/drag state
	dragging         bool
	dragSourceColumn int
	dragSourceTicket int
	dragTargetColumn int

	// Cached column tickets
	columnTickets [][]*board.Ticket

	// Overlay state
	showHelp    bool
	showConfirm bool
	confirmMsg  string
	confirmFn   func() tea.Cmd

	titleInput      textinput.Model
	descInput       textarea.Model
	branchInput     textinput.Model
	ticketFormField int
	editingTicketID board.TicketID
	branchLocked    bool

	// Error/notification
	notification string
	notifyTime   time.Time

	// Terminal panes for embedded agent sessions
	panes       map[board.TicketID]*terminal.Pane
	focusedPane board.TicketID // "" = board view, otherwise pane is focused

	// Settings UI state
	settingsIndex   int             // which setting field is selected
	settingsEditing bool            // true when editing a field value
	settingsInput   textinput.Model // input for editing string values
}

func NewModel(cfg *config.Config, b *board.Board, boardDir string, agentMgr *agent.Manager, worktreeMgr *git.WorktreeManager) *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter ticket title..."
	ti.CharLimit = 100
	ti.Width = 40

	di := textarea.New()
	di.Placeholder = "Optional description..."
	di.CharLimit = 500
	di.SetWidth(40)
	di.SetHeight(4)
	di.ShowLineNumbers = false

	bi := textinput.New()
	bi.Placeholder = "Auto-generated from title..."
	bi.CharLimit = 100
	bi.Width = 40

	si := textinput.New()
	si.CharLimit = 200
	si.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Meter
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))

	m := &Model{
		config:        cfg,
		board:         b,
		boardDir:      boardDir,
		agentMgr:      agentMgr,
		worktreeMgr:   worktreeMgr,
		mode:          ModeNormal,
		titleInput:    ti,
		descInput:     di,
		branchInput:   bi,
		settingsInput: si,
		spinner:       sp,
		panes:         make(map[board.TicketID]*terminal.Pane),
	}
	m.refreshColumnTickets()
	return m
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickAgentStatus(m.agentMgr.StatusPollInterval()),
		m.spinner.Tick,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.focusedPane != "" {
			if pane, ok := m.panes[m.focusedPane]; ok {
				pane.SetSize(m.width, m.height-2)
			}
		}
		return m, nil

	case tea.MouseMsg:
		if m.mode == ModeNormal {
			return m.handleMouse(msg)
		}
		return m, nil

	case terminal.OutputMsg, terminal.RenderTickMsg:
		return m.handleTerminalMsg(msg)

	case terminal.ExitMsg:
		delete(m.panes, board.TicketID(msg.PaneID))
		if m.focusedPane == board.TicketID(msg.PaneID) {
			m.mode = ModeNormal
			m.focusedPane = ""
			m.notify("Agent exited")
		}
		return m, nil

	case terminal.ExitFocusMsg:
		m.mode = ModeNormal
		m.focusedPane = ""
		return m, nil

	case agentStatusMsg:
		m.agentMgr.PollStatuses(m.board.Tickets)
		return m, tickAgentStatus(m.agentMgr.StatusPollInterval())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case notificationMsg:
		if time.Since(m.notifyTime) > 3*time.Second {
			m.notification = ""
		}
		return m, nil
	}

	return m, nil
}

// handleKey processes keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		if m.mode == ModeNormal {
			return m, tea.Quit
		}
	case "esc":
		if m.mode == ModeAgentView {
			break
		}
		m.mode = ModeNormal
		m.showHelp = false
		m.showConfirm = false
		m.titleInput.Blur()
		return m, nil
	case "?":
		if m.mode != ModeAgentView {
			m.showHelp = !m.showHelp
			return m, nil
		}
	}

	// Mode-specific handling
	if m.showHelp {
		// Any key closes help
		m.showHelp = false
		return m, nil
	}

	if m.showConfirm {
		return m.handleConfirm(msg)
	}

	switch m.mode {
	case ModeNormal:
		return m.handleNormalMode(msg)
	case ModeCommand:
		return m.handleCommandMode(msg)
	case ModeCreateTicket:
		return m.handleCreateTicketMode(msg)
	case ModeEditTicket:
		return m.handleEditTicketMode(msg)
	case ModeAgentView:
		return m.handleAgentViewMode(msg)
	case ModeSettings:
		return m.handleSettingsMode(msg)
	}

	return m, nil
}

// handleNormalMode processes keys in normal mode
func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Navigation
	case "h", "left":
		m.moveColumn(-1)
	case "l", "right":
		m.moveColumn(1)
	case "j", "down":
		m.moveTicket(1)
	case "k", "up":
		m.moveTicket(-1)
	case "g":
		m.activeTicket = 0
	case "G":
		if len(m.columnTickets) > m.activeColumn {
			m.activeTicket = len(m.columnTickets[m.activeColumn]) - 1
			if m.activeTicket < 0 {
				m.activeTicket = 0
			}
		}

	case "n":
		return m.createNewTicket()
	case "e":
		return m.editTicket()
	case "enter":
		return m.attachToAgent()
	case "d":
		return m.confirmDeleteTicket()
	case " ":
		return m.quickMoveTicket()
	case "s":
		return m.spawnAgent()
	case "S":
		return m.stopAgent()

	// Command mode
	case ":":
		m.mode = ModeCommand

	// Settings
	case "O":
		m.mode = ModeSettings
		m.settingsIndex = 0
		m.settingsEditing = false
	}

	return m, nil
}

// handleMouse processes mouse events
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			col, ticket := m.hitTest(msg.X, msg.Y)
			if col >= 0 {
				m.activeColumn = col
				if ticket >= 0 {
					m.activeTicket = ticket
					m.dragging = true
					m.dragSourceColumn = col
					m.dragSourceTicket = ticket
					m.dragTargetColumn = col
				}
				m.ensureColumnVisible()
			}
		}

	case tea.MouseActionMotion:
		if m.dragging && msg.Button == tea.MouseButtonLeft {
			col, _ := m.hitTest(msg.X, msg.Y)
			if col >= 0 {
				m.dragTargetColumn = col
			}
		}

	case tea.MouseActionRelease:
		if m.dragging {
			if m.dragTargetColumn != m.dragSourceColumn && m.dragTargetColumn >= 0 {
				return m.dropTicket()
			}
			m.dragging = false
			m.dragTargetColumn = 0
		}

	default:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.moveTicket(-1)
		case tea.MouseButtonWheelDown:
			m.moveTicket(1)
		}
	}

	return m, nil
}

// hitTest returns column and ticket indices at screen position, or -1 if not found
func (m *Model) hitTest(x, y int) (column, ticket int) {
	if m.width == 0 || len(m.board.Columns) == 0 {
		return -1, -1
	}

	headerHeight := 2
	if y < headerHeight {
		return -1, -1
	}

	columnWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(columnWidth)
	numVisible := visibleCols
	if m.scrollOffset+visibleCols > len(m.board.Columns) {
		numVisible = len(m.board.Columns) - m.scrollOffset
	}

	baseWidth, remainder := m.distributeWidth(numVisible)

	hasLeftIndicator := m.scrollOffset > 0
	startX := 0
	if hasLeftIndicator {
		startX = 2
	}

	for i := 0; i < numVisible; i++ {
		colWidth := baseWidth + 3
		if i < remainder {
			colWidth++
		}

		if x >= startX && x < startX+colWidth {
			actualCol := m.scrollOffset + i
			ticketIdx := m.hitTestTicket(y-headerHeight, actualCol)
			return actualCol, ticketIdx
		}
		startX += colWidth
	}

	return -1, -1
}

func (m *Model) hitTestTicket(relativeY, column int) int {
	if column < 0 || column >= len(m.columnTickets) {
		return -1
	}

	tickets := m.columnTickets[column]
	if len(tickets) == 0 {
		return -1
	}

	columnHeaderHeight := 3
	ticketHeight := 6

	ticketY := relativeY - columnHeaderHeight
	if ticketY < 0 {
		return -1
	}

	ticketIdx := ticketY / ticketHeight
	if ticketIdx >= len(tickets) {
		return -1
	}

	return ticketIdx
}

func (m *Model) dropTicket() (tea.Model, tea.Cmd) {
	if len(m.columnTickets) <= m.dragSourceColumn {
		m.dragging = false
		return m, nil
	}

	tickets := m.columnTickets[m.dragSourceColumn]
	if len(tickets) <= m.dragSourceTicket {
		m.dragging = false
		return m, nil
	}

	ticket := tickets[m.dragSourceTicket]
	targetStatus := m.board.Columns[m.dragTargetColumn].Status

	if targetStatus == board.StatusInProgress && ticket.WorktreePath == "" {
		if err := m.setupWorktree(ticket); err != nil {
			m.notify("Worktree failed: " + err.Error())
			m.dragging = false
			return m, nil
		}
	}

	m.board.MoveTicket(ticket.ID, targetStatus)
	m.refreshColumnTickets()
	m.saveBoard()

	m.activeColumn = m.dragTargetColumn
	m.activeTicket = 0
	m.ensureColumnVisible()

	m.notify("Moved to " + string(targetStatus))
	m.dragging = false
	m.dragTargetColumn = 0

	return m, nil
}

// handleCommandMode processes keys in command mode
func (m *Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute command
		m.mode = ModeNormal
	case "esc":
		m.mode = ModeNormal
	}
	return m, nil
}

// handleConfirm processes keys in confirm dialog
func (m *Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.showConfirm = false
		if m.confirmFn != nil {
			return m, m.confirmFn()
		}
	case "n", "N", "esc":
		m.showConfirm = false
	}
	return m, nil
}

func (m *Model) handleAgentViewMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pane, ok := m.panes[m.focusedPane]
	if !ok {
		m.mode = ModeNormal
		m.focusedPane = ""
		return m, nil
	}

	switch msg.String() {
	case "tab":
		m.focusedPane = m.nextActivePane()
		return m, nil
	}

	if result := pane.HandleKey(msg); result != nil {
		if _, isExit := result.(terminal.ExitFocusMsg); isExit {
			m.mode = ModeNormal
			m.focusedPane = ""
		}
	}

	return m, nil
}

func (m *Model) nextActivePane() board.TicketID {
	var activePanes []board.TicketID
	for id, pane := range m.panes {
		if pane.Running() {
			activePanes = append(activePanes, id)
		}
	}
	if len(activePanes) == 0 {
		return ""
	}

	currentIdx := -1
	for i, id := range activePanes {
		if id == m.focusedPane {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(activePanes)
	return activePanes[nextIdx]
}

func (m *Model) handleCreateTicketMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleTicketForm(msg, false)
}

func (m *Model) handleEditTicketMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleTicketForm(msg, true)
}

func (m *Model) handleTicketForm(msg tea.KeyMsg, isEdit bool) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		return m.nextFormField(), nil
	case "shift+tab":
		return m.prevFormField(), nil

	case "ctrl+s", "ctrl+enter":
		return m.saveTicketForm(isEdit)

	case "enter":
		if m.ticketFormField == formFieldTitle {
			return m.saveTicketForm(isEdit)
		}

	case "esc":
		m.mode = ModeNormal
		m.blurAllFormFields()
		m.editingTicketID = ""
		m.branchLocked = false
		return m, nil
	}

	var cmd tea.Cmd
	switch m.ticketFormField {
	case formFieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case formFieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case formFieldBranch:
		if !m.branchLocked {
			m.branchInput, cmd = m.branchInput.Update(msg)
		}
	}
	return m, cmd
}

func (m *Model) nextFormField() *Model {
	m.blurAllFormFields()
	m.ticketFormField++
	if m.ticketFormField > formFieldBranch {
		m.ticketFormField = formFieldTitle
	}
	if m.ticketFormField == formFieldBranch && m.branchLocked {
		m.ticketFormField = formFieldTitle
	}
	m.focusCurrentField()
	return m
}

func (m *Model) prevFormField() *Model {
	m.blurAllFormFields()
	m.ticketFormField--
	if m.ticketFormField < formFieldTitle {
		m.ticketFormField = formFieldBranch
	}
	if m.ticketFormField == formFieldBranch && m.branchLocked {
		m.ticketFormField = formFieldDescription
	}
	m.focusCurrentField()
	return m
}

func (m *Model) blurAllFormFields() {
	m.titleInput.Blur()
	m.descInput.Blur()
	m.branchInput.Blur()
}

func (m *Model) focusCurrentField() {
	switch m.ticketFormField {
	case formFieldTitle:
		m.titleInput.Focus()
	case formFieldDescription:
		m.descInput.Focus()
	case formFieldBranch:
		m.branchInput.Focus()
	}
}

func (m *Model) saveTicketForm(isEdit bool) (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.notify("Title cannot be empty")
		return m, nil
	}

	desc := strings.TrimSpace(m.descInput.Value())
	branchName := strings.TrimSpace(m.branchInput.Value())
	if branchName == "" {
		branchName = m.generateBranchNameFromTitle(title)
	}

	if isEdit && m.editingTicketID != "" {
		ticket := m.board.Tickets[m.editingTicketID]
		if ticket != nil {
			ticket.Title = title
			ticket.Description = desc
			if !m.branchLocked {
				ticket.BranchName = branchName
			}
			ticket.UpdatedAt = time.Now()
			m.saveBoard()
			m.refreshColumnTickets()
			m.notify("Updated: " + title)
		}
	} else {
		ticket := board.NewTicket(title)
		ticket.Description = desc
		ticket.BranchName = branchName
		ticket.Status = m.board.Columns[m.activeColumn].Status
		m.board.AddTicket(ticket)
		m.refreshColumnTickets()
		m.saveBoard()
		m.notify("Created: " + title)
	}

	m.mode = ModeNormal
	m.blurAllFormFields()
	m.editingTicketID = ""
	m.branchLocked = false
	return m, nil
}

type settingsField struct {
	key   string
	label string
	kind  string
}

var settingsFields = []settingsField{
	{"default_agent", "Default Agent", "string"},
	{"worktree_base", "Worktree Base", "string"},
	{"auto_spawn_agent", "Auto Spawn Agent", "bool"},
	{"auto_create_branch", "Auto Create Branch", "bool"},
	{"branch_prefix", "Branch Prefix", "string"},
	{"branch_naming", "Branch Naming", "string"},
	{"branch_template", "Branch Template", "string"},
	{"slug_max_length", "Slug Max Length", "int"},
}

func (m *Model) handleSettingsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsEditing {
		return m.handleSettingsEdit(msg)
	}

	switch msg.String() {
	case "j", "down":
		m.settingsIndex++
		if m.settingsIndex >= len(settingsFields) {
			m.settingsIndex = len(settingsFields) - 1
		}
	case "k", "up":
		m.settingsIndex--
		if m.settingsIndex < 0 {
			m.settingsIndex = 0
		}
	case "enter", " ":
		return m.enterSettingsEdit()
	case "esc", "q":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m *Model) handleSettingsEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	field := settingsFields[m.settingsIndex]

	switch msg.String() {
	case "enter":
		m.applySettingsValue(field.key, m.settingsInput.Value())
		m.settingsEditing = false
		m.settingsInput.Blur()
		m.saveBoard()
		m.notify("Settings saved")
		return m, nil
	case "esc":
		m.settingsEditing = false
		m.settingsInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.settingsInput, cmd = m.settingsInput.Update(msg)
	return m, cmd
}

func (m *Model) enterSettingsEdit() (tea.Model, tea.Cmd) {
	field := settingsFields[m.settingsIndex]
	s := &m.board.Settings

	if field.kind == "bool" {
		switch field.key {
		case "auto_spawn_agent":
			s.AutoSpawnAgent = !s.AutoSpawnAgent
		case "auto_create_branch":
			s.AutoCreateBranch = !s.AutoCreateBranch
		}
		m.saveBoard()
		m.notify("Setting toggled")
		return m, nil
	}

	m.settingsEditing = true
	m.settingsInput.SetValue(m.getSettingsValue(field.key))
	m.settingsInput.Focus()
	return m, textinput.Blink
}

func (m *Model) getSettingsValue(key string) string {
	s := &m.board.Settings
	switch key {
	case "default_agent":
		return s.DefaultAgent
	case "worktree_base":
		return s.WorktreeBase
	case "branch_prefix":
		return s.BranchPrefix
	case "branch_naming":
		return s.BranchNaming
	case "branch_template":
		return s.BranchTemplate
	case "slug_max_length":
		return fmt.Sprintf("%d", s.SlugMaxLength)
	}
	return ""
}

func (m *Model) applySettingsValue(key, value string) {
	s := &m.board.Settings
	switch key {
	case "default_agent":
		s.DefaultAgent = value
	case "worktree_base":
		s.WorktreeBase = value
	case "branch_prefix":
		s.BranchPrefix = value
	case "branch_naming":
		s.BranchNaming = value
	case "branch_template":
		s.BranchTemplate = value
	case "slug_max_length":
		if n, err := strconv.Atoi(value); err == nil && n > 0 {
			s.SlugMaxLength = n
		}
	}
}

// Navigation helpers
func (m *Model) moveColumn(delta int) {
	m.activeColumn += delta
	if m.activeColumn < 0 {
		m.activeColumn = 0
	}
	if m.activeColumn >= len(m.board.Columns) {
		m.activeColumn = len(m.board.Columns) - 1
	}
	m.activeTicket = 0
	m.ensureColumnVisible()
}

func (m *Model) ensureColumnVisible() {
	colWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(colWidth)

	if m.activeColumn < m.scrollOffset {
		m.scrollOffset = m.activeColumn
	} else if m.activeColumn >= m.scrollOffset+visibleCols {
		m.scrollOffset = m.activeColumn - visibleCols + 1
	}

	maxOffset := len(m.board.Columns) - visibleCols
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

func (m *Model) calcColumnWidth() int {
	if m.width == 0 || len(m.board.Columns) == 0 {
		return minColumnWidth
	}

	numCols := len(m.board.Columns)
	totalOverhead := numCols * columnOverhead
	colWidth := (m.width - totalOverhead) / numCols

	if colWidth < minColumnWidth {
		colWidth = minColumnWidth
	}
	return colWidth
}

func (m *Model) visibleColumnCount(colWidth int) int {
	if m.width == 0 {
		return len(m.board.Columns)
	}
	visible := m.width / (colWidth + columnOverhead)
	if visible < 1 {
		visible = 1
	}
	if visible > len(m.board.Columns) {
		visible = len(m.board.Columns)
	}
	return visible
}

func (m *Model) distributeWidth(numCols int) (baseWidth, remainder int) {
	if numCols == 0 || m.width == 0 {
		return minColumnWidth, 0
	}
	// lipgloss Width() includes padding, so only border (2) and margin (1) are outside
	borders := numCols * 2
	margins := numCols - 1
	available := m.width - borders - margins
	baseWidth = available / numCols
	remainder = available % numCols
	if baseWidth < minColumnWidth {
		baseWidth = minColumnWidth
		remainder = 0
	}
	return baseWidth, remainder
}

func (m *Model) moveTicket(delta int) {
	if len(m.columnTickets) <= m.activeColumn {
		return
	}
	tickets := m.columnTickets[m.activeColumn]
	m.activeTicket += delta
	if m.activeTicket < 0 {
		m.activeTicket = 0
	}
	if m.activeTicket >= len(tickets) {
		m.activeTicket = len(tickets) - 1
		if m.activeTicket < 0 {
			m.activeTicket = 0
		}
	}
}

func (m *Model) createNewTicket() (tea.Model, tea.Cmd) {
	m.mode = ModeCreateTicket
	m.ticketFormField = formFieldTitle
	m.editingTicketID = ""
	m.branchLocked = false
	m.titleInput.Reset()
	m.descInput.Reset()
	m.branchInput.Reset()
	m.blurAllFormFields()
	m.titleInput.Focus()
	return m, m.titleInput.Cursor.BlinkCmd()
}

func (m *Model) editTicket() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		m.notify("No ticket selected")
		return m, nil
	}

	m.mode = ModeEditTicket
	m.ticketFormField = formFieldTitle
	m.editingTicketID = ticket.ID
	m.branchLocked = ticket.WorktreePath != ""
	m.titleInput.SetValue(ticket.Title)
	m.descInput.SetValue(ticket.Description)
	if ticket.BranchName != "" {
		m.branchInput.SetValue(ticket.BranchName)
	} else {
		m.branchInput.SetValue(m.generateBranchNameFromTitle(ticket.Title))
	}
	m.blurAllFormFields()
	m.titleInput.Focus()
	return m, m.titleInput.Cursor.BlinkCmd()
}

func (m *Model) attachToAgent() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		m.notify("No ticket selected")
		return m, nil
	}

	pane, ok := m.panes[ticket.ID]
	if !ok || !pane.Running() {
		m.notify("No active agent for this ticket")
		return m, nil
	}

	m.mode = ModeAgentView
	m.focusedPane = ticket.ID
	paneHeight := m.height - 2
	pane.SetSize(m.width, paneHeight)
	return m, nil
}

func (m *Model) confirmDeleteTicket() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	m.showConfirm = true
	m.confirmMsg = "Delete ticket: " + ticket.Title + "?"
	m.confirmFn = func() tea.Cmd {
		m.board.DeleteTicket(ticket.ID)
		m.refreshColumnTickets()
		m.saveBoard()
		m.notify("Deleted ticket")
		return nil
	}
	return m, nil
}

func (m *Model) quickMoveTicket() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	nextStatus := m.nextStatus(ticket.Status)
	if nextStatus == ticket.Status {
		return m, nil
	}

	if nextStatus == board.StatusInProgress && ticket.WorktreePath == "" {
		if err := m.setupWorktree(ticket); err != nil {
			m.notify("Worktree failed: " + err.Error())
			return m, nil
		}
	}

	m.board.MoveTicket(ticket.ID, nextStatus)
	m.refreshColumnTickets()
	m.saveBoard()
	m.notify("Moved to " + string(nextStatus))

	return m, nil
}

func (m *Model) setupWorktree(ticket *board.Ticket) error {
	branchName := m.generateBranchName(ticket)
	baseBranch, _ := m.worktreeMgr.GetDefaultBranch()

	path, err := m.worktreeMgr.CreateWorktree(branchName, baseBranch)
	if err != nil {
		return err
	}

	ticket.WorktreePath = path
	ticket.BranchName = branchName
	ticket.BaseBranch = baseBranch
	return nil
}

func (m *Model) generateBranchNameFromTitle(title string) string {
	settings := &m.board.Settings

	maxLen := settings.SlugMaxLength
	if maxLen <= 0 {
		maxLen = 40
	}

	slug := board.Slugify(title, maxLen)

	template := settings.BranchTemplate
	if template == "" {
		template = "{prefix}{slug}"
	}

	prefix := settings.BranchPrefix
	if prefix == "" {
		prefix = "task/"
	}

	result := strings.ReplaceAll(template, "{prefix}", prefix)
	result = strings.ReplaceAll(result, "{slug}", slug)

	return result
}

func (m *Model) generateBranchName(ticket *board.Ticket) string {
	if ticket.BranchName != "" {
		return ticket.BranchName
	}
	return m.generateBranchNameFromTitle(ticket.Title)
}

func (m *Model) spawnAgent() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	if ticket.Status != board.StatusInProgress {
		m.notify("Move ticket to In Progress first")
		return m, nil
	}

	if _, exists := m.panes[ticket.ID]; exists {
		m.notify("Agent already running")
		return m, nil
	}

	if ticket.WorktreePath == "" {
		if err := m.setupWorktree(ticket); err != nil {
			m.notify("Failed to create worktree: " + err.Error())
			return m, nil
		}
	}

	agentType := m.board.Settings.DefaultAgent
	agentCfg, ok := m.config.Agents[agentType]
	if !ok {
		m.notify("Unknown agent: " + agentType)
		return m, nil
	}

	pane := terminal.New(string(ticket.ID), m.width, m.height-2)
	pane.SetWorkdir(ticket.WorktreePath)
	m.panes[ticket.ID] = pane

	ticket.AgentType = agentType
	ticket.AgentStatus = board.AgentIdle

	isResume := ticket.AgentSpawnedAt != nil
	args := m.buildAgentArgs(agentCfg, ticket)

	if !isResume {
		now := time.Now()
		ticket.AgentSpawnedAt = &now
	}

	m.saveBoard()

	if isResume {
		m.notify("Resuming " + agentType)
	} else {
		m.notify("Starting " + agentType)
	}

	m.mode = ModeAgentView
	m.focusedPane = ticket.ID

	return m, pane.Start(agentCfg.Command, args...)
}

func (m *Model) buildAgentArgs(cfg config.AgentConfig, ticket *board.Ticket) []string {
	args := make([]string, len(cfg.Args))
	copy(args, cfg.Args)

	agentType := cfg.Command
	if strings.Contains(agentType, "/") {
		agentType = filepath.Base(agentType)
	}

	hasSession := ticket.AgentSpawnedAt != nil

	switch agentType {
	case "claude":
		if hasSession && !containsFlag(args, "--continue", "-c") {
			args = append(args, "--continue")
		}
	case "opencode":
		args = append([]string{ticket.WorktreePath}, args...)
		if hasSession {
			if sessionID := agent.FindOpencodeSession(ticket.WorktreePath); sessionID != "" {
				args = append(args, "--session", sessionID)
			}
		}
	}

	return args
}

func containsFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

func (m *Model) stopAgent() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	if pane, ok := m.panes[ticket.ID]; ok {
		pane.Stop()
		delete(m.panes, ticket.ID)
	}

	ticket.AgentStatus = board.AgentNone
	m.saveBoard()
	m.notify("Agent stopped")
	return m, nil
}

// Helper methods
func (m *Model) selectedTicket() *board.Ticket {
	if len(m.columnTickets) <= m.activeColumn {
		return nil
	}
	tickets := m.columnTickets[m.activeColumn]
	if len(tickets) <= m.activeTicket {
		return nil
	}
	return tickets[m.activeTicket]
}

func (m *Model) refreshColumnTickets() {
	m.columnTickets = make([][]*board.Ticket, len(m.board.Columns))
	for i, col := range m.board.Columns {
		m.columnTickets[i] = m.board.GetTicketsByStatus(col.Status)
	}
}

func (m *Model) nextStatus(current board.TicketStatus) board.TicketStatus {
	switch current {
	case board.StatusBacklog:
		return board.StatusInProgress
	case board.StatusInProgress:
		return board.StatusDone
	default:
		return current
	}
}

func (m *Model) notify(msg string) {
	m.notification = msg
	m.notifyTime = time.Now()
}

func (m *Model) saveBoard() {
	if err := m.board.Save(m.boardDir); err != nil {
		m.notify("Failed to save: " + err.Error())
	}
}

func (m *Model) handleTerminalMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for _, pane := range m.panes {
		if cmd := pane.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

type agentStatusMsg time.Time
type notificationMsg time.Time

func tickAgentStatus(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return agentStatusMsg(t)
	})
}
