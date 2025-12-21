package ui

import (
	"fmt"
	"path/filepath"
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
	"github.com/techdufus/openkanban/internal/project"
	"github.com/techdufus/openkanban/internal/terminal"
)

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
	columnOverhead = 5

	ticketHeight       = 6
	columnHeaderHeight = 3

	formFieldTitle       = 0
	formFieldDescription = 1
	formFieldBranch      = 2
	formFieldProject     = 3
)

type Model struct {
	config *config.Config

	globalStore     *project.GlobalTicketStore
	columns         []board.Column
	filterProjectID string

	worktreeMgrs map[string]*git.WorktreeManager
	agentMgr     *agent.Manager

	mode          Mode
	activeColumn  int
	activeTicket  int
	width         int
	height        int
	spinner       spinner.Model
	scrollOffset  int
	columnOffsets []int

	dragging         bool
	dragSourceColumn int
	dragSourceTicket int
	dragTargetColumn int

	columnTickets [][]*board.Ticket

	showHelp    bool
	showConfirm bool
	confirmMsg  string
	confirmFn   func() tea.Cmd

	titleInput      textinput.Model
	descInput       textarea.Model
	branchInput     textinput.Model
	projectInput    textinput.Model
	ticketFormField int
	editingTicketID board.TicketID
	branchLocked    bool
	selectedProject *project.Project

	notification string
	notifyTime   time.Time

	panes          map[board.TicketID]*terminal.Pane
	focusedPane    board.TicketID
	statusDetector *agent.StatusDetector

	settingsIndex   int
	settingsEditing bool
	settingsInput   textinput.Model
}

func NewModel(cfg *config.Config, globalStore *project.GlobalTicketStore, agentMgr *agent.Manager, filterProjectID string) *Model {
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

	pi := textinput.New()
	pi.Placeholder = "Select project..."
	pi.CharLimit = 100
	pi.Width = 40

	si := textinput.New()
	si.CharLimit = 200
	si.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Meter
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))

	worktreeMgrs := make(map[string]*git.WorktreeManager)
	for _, p := range globalStore.Projects() {
		worktreeMgrs[p.ID] = git.NewWorktreeManager(p)
	}

	var selectedProject *project.Project
	projects := globalStore.Projects()
	if len(projects) > 0 {
		if filterProjectID != "" {
			selectedProject = globalStore.GetProject(filterProjectID)
		}
		if selectedProject == nil {
			selectedProject = projects[0]
		}
	}

	m := &Model{
		config:          cfg,
		globalStore:     globalStore,
		columns:         board.DefaultColumns(),
		filterProjectID: filterProjectID,
		worktreeMgrs:    worktreeMgrs,
		agentMgr:        agentMgr,
		mode:            ModeNormal,
		titleInput:      ti,
		descInput:       di,
		branchInput:     bi,
		projectInput:    pi,
		settingsInput:   si,
		spinner:         sp,
		panes:           make(map[board.TicketID]*terminal.Pane),
		statusDetector:  agent.NewStatusDetector(),
		selectedProject: selectedProject,
	}
	m.refreshColumnTickets()
	return m
}

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
		if m.mode == ModeAgentView {
			return m.handleAgentViewMouse(msg)
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
		return m, tea.Batch(
			m.pollAgentStatusesAsync(),
			tickAgentStatus(m.agentMgr.StatusPollInterval()),
		)

	case agentStatusResultMsg:
		for ticketID, status := range msg {
			if ticket, _ := m.globalStore.Get(ticketID); ticket != nil {
				ticket.AgentStatus = status
			}
		}

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

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.mode == ModeNormal || m.mode == ModeHelp {
			m.showHelp = !m.showHelp
			return m, nil
		}
	}

	if m.showHelp {
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

func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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
		m.ensureTicketVisible()
	case "G":
		if len(m.columnTickets) > m.activeColumn {
			m.activeTicket = max(len(m.columnTickets[m.activeColumn])-1, 0)
		}
		m.ensureTicketVisible()

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
	case "-", "backspace":
		return m.quickMoveTicketBackward()
	case "s":
		return m.spawnAgent()
	case "S":
		return m.stopAgent()

	case ":":
		m.mode = ModeCommand

	case "p":
		m.cycleProjectFilter()

	case "O":
		m.mode = ModeSettings
		m.settingsIndex = 0
		m.settingsEditing = false
	}

	return m, nil
}

func (m *Model) cycleProjectFilter() {
	projects := m.globalStore.Projects()
	if len(projects) == 0 {
		return
	}

	if m.filterProjectID == "" {
		m.filterProjectID = projects[0].ID
	} else {
		found := false
		for i, p := range projects {
			if p.ID == m.filterProjectID {
				if i+1 < len(projects) {
					m.filterProjectID = projects[i+1].ID
				} else {
					m.filterProjectID = ""
				}
				found = true
				break
			}
		}
		if !found {
			m.filterProjectID = ""
		}
	}

	m.refreshColumnTickets()

	if m.filterProjectID == "" {
		m.notify("Showing all projects")
	} else {
		if p := m.globalStore.GetProject(m.filterProjectID); p != nil {
			m.notify("Filtering: " + p.Name)
		}
	}
}

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

func (m *Model) hitTest(x, y int) (column, ticket int) {
	if m.width == 0 || len(m.columns) == 0 {
		return -1, -1
	}

	headerHeight := 2
	if y < headerHeight {
		return -1, -1
	}

	columnWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(columnWidth)
	numVisible := visibleCols
	if m.scrollOffset+visibleCols > len(m.columns) {
		numVisible = len(m.columns) - m.scrollOffset
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

	ticketY := relativeY - columnHeaderHeight
	if ticketY < 0 {
		return -1
	}

	offset := 0
	if column < len(m.columnOffsets) {
		offset = m.columnOffsets[column]
	}

	ticketIdx := offset + (ticketY / ticketHeight)
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
	targetStatus := m.columns[m.dragTargetColumn].Status

	if targetStatus == board.StatusInProgress && ticket.WorktreePath == "" {
		if err := m.setupWorktree(ticket); err != nil {
			m.notify("Worktree failed: " + err.Error())
			m.dragging = false
			return m, nil
		}
	}

	m.globalStore.Move(ticket.ID, targetStatus)
	m.refreshColumnTickets()
	m.saveTicket(ticket)

	m.activeColumn = m.dragTargetColumn
	m.activeTicket = 0
	m.ensureColumnVisible()
	m.ensureTicketVisible()

	m.notify("Moved to " + string(targetStatus))
	m.dragging = false
	m.dragTargetColumn = 0

	return m, nil
}

func (m *Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.mode = ModeNormal
	case "esc":
		m.mode = ModeNormal
	}
	return m, nil
}

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

	if result := pane.HandleKey(msg); result != nil {
		if _, isExit := result.(terminal.ExitFocusMsg); isExit {
			m.mode = ModeNormal
			m.focusedPane = ""
		}
	}

	return m, nil
}

func (m *Model) handleAgentViewMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	pane, ok := m.panes[m.focusedPane]
	if !ok {
		return m, nil
	}

	pane.HandleMouse(msg)
	return m, nil
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
		return m.nextFormField(isEdit), nil
	case "shift+tab":
		return m.prevFormField(isEdit), nil

	case "ctrl+s", "ctrl+enter":
		return m.saveTicketForm(isEdit)

	case "enter":
		if m.ticketFormField == formFieldTitle {
			return m.saveTicketForm(isEdit)
		}
		if m.ticketFormField == formFieldProject && !isEdit {
			m.cycleSelectedProject()
			return m, nil
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
	case formFieldProject:
		if msg.String() == " " || msg.String() == "enter" {
			m.cycleSelectedProject()
		}
	}
	return m, cmd
}

func (m *Model) cycleSelectedProject() {
	projects := m.globalStore.Projects()
	if len(projects) == 0 {
		return
	}

	if m.selectedProject == nil {
		m.selectedProject = projects[0]
		return
	}

	for i, p := range projects {
		if p.ID == m.selectedProject.ID {
			if i+1 < len(projects) {
				m.selectedProject = projects[i+1]
			} else {
				m.selectedProject = projects[0]
			}
			return
		}
	}
	m.selectedProject = projects[0]
}

func (m *Model) nextFormField(isEdit bool) *Model {
	m.blurAllFormFields()
	m.ticketFormField++

	maxField := formFieldBranch
	if !isEdit && len(m.globalStore.Projects()) > 1 {
		maxField = formFieldProject
	}

	if m.ticketFormField > maxField {
		m.ticketFormField = formFieldTitle
	}
	if m.ticketFormField == formFieldBranch && m.branchLocked {
		m.ticketFormField++
		if m.ticketFormField > maxField {
			m.ticketFormField = formFieldTitle
		}
	}
	m.focusCurrentField()
	return m
}

func (m *Model) prevFormField(isEdit bool) *Model {
	m.blurAllFormFields()
	m.ticketFormField--

	maxField := formFieldBranch
	if !isEdit && len(m.globalStore.Projects()) > 1 {
		maxField = formFieldProject
	}

	if m.ticketFormField < formFieldTitle {
		m.ticketFormField = maxField
	}
	if m.ticketFormField == formFieldBranch && m.branchLocked {
		m.ticketFormField--
		if m.ticketFormField < formFieldTitle {
			m.ticketFormField = maxField
		}
	}
	m.focusCurrentField()
	return m
}

func (m *Model) blurAllFormFields() {
	m.titleInput.Blur()
	m.descInput.Blur()
	m.branchInput.Blur()
	m.projectInput.Blur()
}

func (m *Model) focusCurrentField() {
	switch m.ticketFormField {
	case formFieldTitle:
		m.titleInput.Focus()
	case formFieldDescription:
		m.descInput.Focus()
	case formFieldBranch:
		m.branchInput.Focus()
	case formFieldProject:
		m.projectInput.Focus()
	}
}

func (m *Model) saveTicketForm(isEdit bool) (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.notify("Title cannot be empty")
		return m, nil
	}

	if m.selectedProject == nil {
		m.notify("No project selected")
		return m, nil
	}

	desc := strings.TrimSpace(m.descInput.Value())
	branchName := strings.TrimSpace(m.branchInput.Value())
	if branchName == "" {
		branchName = m.generateBranchNameFromTitle(title, m.selectedProject)
	}

	if isEdit && m.editingTicketID != "" {
		ticket, _ := m.globalStore.Get(m.editingTicketID)
		if ticket != nil {
			ticket.Title = title
			ticket.Description = desc
			if !m.branchLocked {
				ticket.BranchName = branchName
			}
			ticket.Touch()
			m.saveTicket(ticket)
			m.refreshColumnTickets()
			m.notify("Updated: " + title)
		}
	} else {
		ticket := board.NewTicket(title, m.selectedProject.ID)
		ticket.Description = desc
		ticket.BranchName = branchName
		ticket.Status = m.columns[m.activeColumn].Status
		m.globalStore.Add(ticket)
		m.refreshColumnTickets()
		m.saveTicket(ticket)
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
	{"filter_project", "Filter Project", "project"},
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
		m.settingsIndex = max(m.settingsIndex, 0)
	case "enter", " ":
		return m.enterSettingsEdit()
	case "esc", "q":
		m.mode = ModeNormal
	}
	return m, nil
}

func (m *Model) handleSettingsEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	field := settingsFields[m.settingsIndex]

	if field.kind == "project" {
		m.cycleProjectFilter()
		m.settingsEditing = false
		return m, nil
	}

	switch msg.String() {
	case "enter":
		m.applySettingsValue(field.key, m.settingsInput.Value())
		m.settingsEditing = false
		m.settingsInput.Blur()
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

	if field.kind == "project" {
		m.cycleProjectFilter()
		return m, nil
	}

	m.settingsEditing = true
	m.settingsInput.SetValue(m.getSettingsValue(field.key))
	m.settingsInput.Focus()
	return m, textinput.Blink
}

func (m *Model) getSettingsValue(key string) string {
	switch key {
	case "filter_project":
		if m.filterProjectID == "" {
			return "All Projects"
		}
		if p := m.globalStore.GetProject(m.filterProjectID); p != nil {
			return p.Name
		}
	}
	return ""
}

func (m *Model) applySettingsValue(key, value string) {
}

func (m *Model) moveColumn(delta int) {
	m.activeColumn += delta
	m.activeColumn = max(m.activeColumn, 0)
	if m.activeColumn >= len(m.columns) {
		m.activeColumn = len(m.columns) - 1
	}
	m.activeTicket = 0
	m.ensureColumnVisible()
	m.ensureTicketVisible()
}

func (m *Model) ensureColumnVisible() {
	colWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(colWidth)

	if m.activeColumn < m.scrollOffset {
		m.scrollOffset = m.activeColumn
	} else if m.activeColumn >= m.scrollOffset+visibleCols {
		m.scrollOffset = m.activeColumn - visibleCols + 1
	}

	maxOffset := max(len(m.columns)-visibleCols, 0)
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

func (m *Model) calcColumnWidth() int {
	if m.width == 0 || len(m.columns) == 0 {
		return minColumnWidth
	}

	numCols := len(m.columns)
	totalOverhead := numCols * columnOverhead
	colWidth := (m.width - totalOverhead) / numCols

	return max(colWidth, minColumnWidth)
}

func (m *Model) visibleColumnCount(colWidth int) int {
	if m.width == 0 {
		return len(m.columns)
	}
	visible := m.width / (colWidth + columnOverhead)
	visible = max(visible, 1)
	if visible > len(m.columns) {
		visible = len(m.columns)
	}
	return visible
}

func (m *Model) distributeWidth(numCols int) (baseWidth, remainder int) {
	if numCols == 0 || m.width == 0 {
		return minColumnWidth, 0
	}
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
	m.activeTicket = max(m.activeTicket, 0)
	if m.activeTicket >= len(tickets) {
		m.activeTicket = max(len(tickets)-1, 0)
	}
	m.ensureTicketVisible()
}

func (m *Model) visibleTicketCount() int {
	availableHeight := m.columnContentHeight()
	if availableHeight <= 0 {
		return 1
	}
	count := availableHeight / ticketHeight
	return max(count, 1)
}

func (m *Model) columnContentHeight() int {
	boardHeight := m.height - 4
	contentHeight := boardHeight - columnHeaderHeight - 4
	return contentHeight
}

func (m *Model) ensureTicketVisible() {
	if m.activeColumn < 0 || m.activeColumn >= len(m.columnOffsets) {
		return
	}

	offset := m.columnOffsets[m.activeColumn]
	visible := m.visibleTicketCount()

	if m.activeTicket < offset {
		m.columnOffsets[m.activeColumn] = m.activeTicket
	} else if m.activeTicket >= offset+visible {
		m.columnOffsets[m.activeColumn] = m.activeTicket - visible + 1
	}

	m.columnOffsets[m.activeColumn] = max(m.columnOffsets[m.activeColumn], 0)
}

func (m *Model) createNewTicket() (tea.Model, tea.Cmd) {
	m.mode = ModeCreateTicket
	m.ticketFormField = formFieldTitle
	m.editingTicketID = ""
	m.branchLocked = false

	if m.filterProjectID != "" {
		m.selectedProject = m.globalStore.GetProject(m.filterProjectID)
	} else if m.selectedProject == nil {
		projects := m.globalStore.Projects()
		if len(projects) > 0 {
			m.selectedProject = projects[0]
		}
	}

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
	m.selectedProject = m.globalStore.GetProjectForTicket(ticket)
	m.titleInput.SetValue(ticket.Title)
	m.descInput.SetValue(ticket.Description)
	if ticket.BranchName != "" {
		m.branchInput.SetValue(ticket.BranchName)
	} else if m.selectedProject != nil {
		m.branchInput.SetValue(m.generateBranchNameFromTitle(ticket.Title, m.selectedProject))
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

	proj := m.globalStore.GetProjectForTicket(ticket)
	hasUncommitted := false
	if ticket.WorktreePath != "" && m.config.Cleanup.DeleteWorktree && proj != nil {
		if mgr := m.worktreeMgrs[proj.ID]; mgr != nil {
			var err error
			hasUncommitted, err = mgr.HasUncommittedChanges(ticket.WorktreePath)
			if err != nil {
				hasUncommitted = false
			}
		}
	}

	if hasUncommitted && !m.config.Cleanup.ForceWorktreeRemoval {
		m.showConfirm = true
		m.confirmMsg = "Worktree has uncommitted changes. Force delete?"
		m.confirmFn = func() tea.Cmd {
			m.performTicketCleanup(ticket)
			return nil
		}
	} else {
		m.showConfirm = true
		m.confirmMsg = "Delete ticket: " + ticket.Title + "?"
		m.confirmFn = func() tea.Cmd {
			m.performTicketCleanup(ticket)
			return nil
		}
	}
	return m, nil
}

func (m *Model) performTicketCleanup(ticket *board.Ticket) {
	if pane, ok := m.panes[ticket.ID]; ok {
		pane.Stop()
		delete(m.panes, ticket.ID)
	}

	proj := m.globalStore.GetProjectForTicket(ticket)
	if proj != nil {
		mgr := m.worktreeMgrs[proj.ID]
		if mgr != nil {
			if ticket.WorktreePath != "" && m.config.Cleanup.DeleteWorktree {
				err := mgr.RemoveWorktree(ticket.WorktreePath)
				if err != nil {
					m.notify("Worktree removal failed: " + err.Error())
				}
			}

			if ticket.BranchName != "" && m.config.Cleanup.DeleteBranch {
				err := mgr.DeleteBranch(ticket.BranchName)
				if err != nil {
					m.notify("Branch deletion failed: " + err.Error())
				}
			}
		}
	}

	m.globalStore.Delete(ticket.ID)
	m.refreshColumnTickets()
	m.globalStore.SaveAll()
	m.notify("Deleted ticket")
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

	m.globalStore.Move(ticket.ID, nextStatus)
	m.refreshColumnTickets()
	m.saveTicket(ticket)
	m.notify("Moved to " + string(nextStatus))

	return m, nil
}

func (m *Model) quickMoveTicketBackward() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	prevStatus := m.previousStatus(ticket.Status)
	if prevStatus == ticket.Status {
		return m, nil
	}

	m.globalStore.Move(ticket.ID, prevStatus)
	m.refreshColumnTickets()
	m.saveTicket(ticket)
	m.notify("Moved to " + string(prevStatus))

	return m, nil
}

func (m *Model) setupWorktree(ticket *board.Ticket) error {
	proj := m.globalStore.GetProjectForTicket(ticket)
	if proj == nil {
		return fmt.Errorf("project not found for ticket")
	}

	mgr := m.worktreeMgrs[proj.ID]
	if mgr == nil {
		return fmt.Errorf("worktree manager not found")
	}

	branchName := m.generateBranchName(ticket, proj)
	baseBranch, _ := mgr.GetDefaultBranch()

	path, err := mgr.CreateWorktree(branchName, baseBranch)
	if err != nil {
		return err
	}

	ticket.WorktreePath = path
	ticket.BranchName = branchName
	ticket.BaseBranch = baseBranch
	return nil
}

func (m *Model) generateBranchNameFromTitle(title string, proj *project.Project) string {
	maxLen := proj.GetSlugMaxLength()
	slug := board.Slugify(title, maxLen)

	template := proj.GetBranchTemplate()
	prefix := proj.GetBranchPrefix()

	result := strings.ReplaceAll(template, "{prefix}", prefix)
	result = strings.ReplaceAll(result, "{slug}", slug)

	return result
}

func (m *Model) generateBranchName(ticket *board.Ticket, proj *project.Project) string {
	if ticket.BranchName != "" {
		return ticket.BranchName
	}
	return m.generateBranchNameFromTitle(ticket.Title, proj)
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

	proj := m.globalStore.GetProjectForTicket(ticket)
	if proj == nil {
		m.notify("Project not found")
		return m, nil
	}

	if ticket.WorktreePath == "" {
		if err := m.setupWorktree(ticket); err != nil {
			m.notify("Failed to create worktree: " + err.Error())
			return m, nil
		}
	}

	agentType := proj.Settings.DefaultAgent
	if agentType == "" {
		agentType = m.config.Defaults.DefaultAgent
	}
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

	isNewSession := agent.ShouldInjectContext(ticket)
	args := m.buildAgentArgs(agentCfg, ticket, isNewSession)

	if isNewSession {
		now := time.Now()
		ticket.AgentSpawnedAt = &now
	}

	m.saveTicket(ticket)

	if isNewSession {
		m.notify("Starting " + agentType)
	} else {
		m.notify("Resuming " + agentType)
	}

	m.mode = ModeAgentView
	m.focusedPane = ticket.ID

	return m, pane.Start(agentCfg.Command, args...)
}

func (m *Model) buildAgentArgs(cfg config.AgentConfig, ticket *board.Ticket, isNewSession bool) []string {
	args := make([]string, len(cfg.Args))
	copy(args, cfg.Args)

	agentType := cfg.Command
	if strings.Contains(agentType, "/") {
		agentType = filepath.Base(agentType)
	}

	promptTemplate := m.config.GetEffectiveInitPrompt(agentType)

	switch agentType {
	case "claude":
		if isNewSession && promptTemplate != "" {
			prompt := agent.BuildContextPrompt(promptTemplate, ticket)
			if prompt != "" {
				args = append(args, "--append-system-prompt", prompt)
			}
		} else if !isNewSession {
			if !containsFlag(args, "--continue", "-c") {
				args = append(args, "--continue")
			}
		}
	case "opencode":
		args = append([]string{ticket.WorktreePath}, args...)
		if isNewSession && promptTemplate != "" {
			prompt := agent.BuildContextPrompt(promptTemplate, ticket)
			if prompt != "" {
				args = append(args, "-p", prompt)
			}
		} else if !isNewSession {
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
	m.saveTicket(ticket)
	m.notify("Agent stopped")
	return m, nil
}

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
	m.columnTickets = make([][]*board.Ticket, len(m.columns))
	for i, col := range m.columns {
		allForStatus := m.globalStore.GetByStatus(col.Status)
		if m.filterProjectID != "" {
			var filtered []*board.Ticket
			for _, t := range allForStatus {
				if t.ProjectID == m.filterProjectID {
					filtered = append(filtered, t)
				}
			}
			m.columnTickets[i] = filtered
		} else {
			m.columnTickets[i] = allForStatus
		}
	}

	if len(m.columnOffsets) != len(m.columns) {
		m.columnOffsets = make([]int, len(m.columns))
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

func (m *Model) previousStatus(current board.TicketStatus) board.TicketStatus {
	switch current {
	case board.StatusDone:
		return board.StatusInProgress
	case board.StatusInProgress:
		return board.StatusBacklog
	default:
		return current
	}
}

func (m *Model) notify(msg string) {
	m.notification = msg
	m.notifyTime = time.Now()
}

func (m *Model) saveTicket(ticket *board.Ticket) {
	if err := m.globalStore.Save(ticket); err != nil {
		m.notify("Failed to save: " + err.Error())
	}
}

func (m *Model) pollAgentStatusesAsync() tea.Cmd {
	type paneInfo struct {
		ticketID     board.TicketID
		agentType    string
		worktreePath string
		branchName   string
		running      bool
	}

	var panes []paneInfo
	for ticketID, pane := range m.panes {
		ticket, _ := m.globalStore.Get(ticketID)
		if ticket == nil {
			continue
		}
		panes = append(panes, paneInfo{
			ticketID:     ticketID,
			agentType:    ticket.AgentType,
			worktreePath: ticket.WorktreePath,
			branchName:   ticket.BranchName,
			running:      pane.Running(),
		})
	}

	detector := m.statusDetector

	return func() tea.Msg {
		results := make(agentStatusResultMsg)
		for _, p := range panes {
			if !p.running {
				results[p.ticketID] = board.AgentNone
				continue
			}

			sessionID := p.branchName
			if sessionID == "" {
				sessionID = string(p.ticketID)
			}
			if p.agentType == "opencode" && p.worktreePath != "" {
				if id := agent.FindOpencodeSession(p.worktreePath); id != "" {
					sessionID = id
				}
			}

			results[p.ticketID] = detector.DetectStatus(p.agentType, sessionID, true)
		}
		return results
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
type agentStatusResultMsg map[board.TicketID]board.AgentStatus
type notificationMsg time.Time

func tickAgentStatus(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return agentStatusMsg(t)
	})
}
