package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/techdufus/openkanban/internal/board"
)

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.mode == ModeAgentView && m.focusedPane != "" {
		return m.renderAgentView()
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	b.WriteString(m.renderBoard())

	if m.showHelp {
		return m.renderWithOverlay(m.renderHelp())
	}
	if m.showConfirm {
		return m.renderWithOverlay(m.renderConfirmDialog())
	}
	if m.mode == ModeCreateTicket || m.mode == ModeEditTicket {
		return m.renderWithOverlay(m.renderTicketForm())
	}
	if m.mode == ModeSettings {
		return m.renderWithOverlay(m.renderSettingsView())
	}

	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

func (m *Model) renderHeader() string {
	logo := lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true).
		Render("◈ OpenKanban")

	var filterBadge string
	if m.filterProjectID == "" {
		filterBadge = lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorMauve).
			Padding(0, 1).
			Render("All Projects")
	} else if p := m.globalStore.GetProject(m.filterProjectID); p != nil {
		filterBadge = lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorMauve).
			Padding(0, 1).
			Render(p.Name)
	}

	projectCount := len(m.globalStore.Projects())
	ticketCount := m.globalStore.Count()
	stats := dimStyle.Render(fmt.Sprintf("%d projects, %d tickets", projectCount, ticketCount))

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", filterBadge, "  ", stats)

	workingCount, waitingCount, idleCount := 0, 0, 0
	for ticketID, pane := range m.panes {
		if !pane.Running() {
			continue
		}
		ticket, _ := m.globalStore.Get(ticketID)
		if ticket == nil {
			workingCount++
			continue
		}

		switch ticket.AgentStatus {
		case board.AgentWorking:
			workingCount++
		case board.AgentWaiting:
			waitingCount++
		case board.AgentIdle:
			idleCount++
		default:
			workingCount++
		}
	}

	var activity string
	totalActive := workingCount + waitingCount + idleCount
	if totalActive > 0 {
		var statusText string
		var bgColor lipgloss.Color

		if waitingCount > 0 {
			bgColor = colorMauve
			statusText = fmt.Sprintf("◐ %d waiting", waitingCount)
			if workingCount > 0 {
				statusText = fmt.Sprintf("◐ %d waiting, %d working", waitingCount, workingCount)
			}
		} else if workingCount > 0 {
			bgColor = colorYellow
			statusText = fmt.Sprintf("%s %d working", m.spinner.View(), workingCount)
		} else {
			bgColor = colorBlue
			statusText = fmt.Sprintf("◆ %d idle", idleCount)
		}

		activityBadge := lipgloss.NewStyle().
			Foreground(colorBase).
			Background(bgColor).
			Bold(true).
			Padding(0, 1).
			Render(statusText)
		activity = activityBadge
	}

	helpStyle := lipgloss.NewStyle().Foreground(colorMuted)
	help := helpStyle.Render("p filter  ? help  q quit")

	right := help
	if activity != "" {
		right = lipgloss.JoinHorizontal(lipgloss.Center, activity, "  ", help)
	}

	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	spacing = max(spacing, 0)

	return lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", spacing), right)
}

func (m *Model) renderBoard() string {
	columnWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(columnWidth)

	startCol := m.scrollOffset
	endCol := min(startCol+visibleCols, len(m.columns))

	numVisible := endCol - startCol
	baseWidth, remainder := m.distributeWidth(numVisible)

	var columns []string

	if startCol > 0 {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Render("◀")
		columns = append(columns, indicator)
	}

	for i := startCol; i < endCol; i++ {
		col := m.columns[i]
		isActive := i == m.activeColumn
		isLast := i == endCol-1
		isDragTarget := m.dragging && i == m.dragTargetColumn && i != m.dragSourceColumn

		colWidth := baseWidth
		if i-startCol < remainder {
			colWidth++
		}

		ticketOffset := 0
		if i < len(m.columnOffsets) {
			ticketOffset = m.columnOffsets[i]
		}

		columns = append(columns, m.renderColumn(col, m.columnTickets[i], isActive, isDragTarget, colWidth, isLast, ticketOffset))
	}

	if endCol < len(m.columns) {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Render("▶")
		columns = append(columns, indicator)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m *Model) renderColumn(col board.Column, tickets []*board.Ticket, isActive, isDragTarget bool, width int, isLast bool, ticketOffset int) string {
	headerColor := lipgloss.Color(col.Color)

	icon := "○"
	if isActive {
		icon = "●"
	}

	headerText := fmt.Sprintf("%s %s", icon, col.Name)
	countText := fmt.Sprintf("(%d)", len(tickets))
	if col.Limit > 0 {
		countText = fmt.Sprintf("(%d/%d)", len(tickets), col.Limit)
	}

	header := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true).
		Render(headerText)

	count := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(" " + countText)

	headerLine := header + count

	visibleCount := m.visibleTicketCount()
	endIdx := min(ticketOffset+visibleCount, len(tickets))

	hasMoreAbove := ticketOffset > 0
	hasMoreBelow := endIdx < len(tickets)

	indicatorStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(width - 4).
		Align(lipgloss.Center)

	var ticketViews []string

	if hasMoreAbove {
		ticketViews = append(ticketViews, indicatorStyle.Render(fmt.Sprintf("▲ %d more", ticketOffset)))
	}

	for i := ticketOffset; i < endIdx; i++ {
		ticket := tickets[i]
		isSelected := isActive && i == m.activeTicket
		ticketViews = append(ticketViews, m.renderTicket(ticket, isSelected, width-4, col.Color))
	}

	if hasMoreBelow {
		remaining := len(tickets) - endIdx
		ticketViews = append(ticketViews, indicatorStyle.Render(fmt.Sprintf("▼ %d more", remaining)))
	}

	ticketsView := strings.Join(ticketViews, "\n")
	if len(tickets) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(1, 0)
		ticketsView = emptyStyle.Render("No tickets")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, headerLine, "", ticketsView)

	border := columnBorder
	borderColor := colorSurface
	if isDragTarget {
		border = columnBorderActive
		borderColor = colorGreen
	} else if isActive {
		border = columnBorderActive
		borderColor = headerColor
	}

	style := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Width(width).
		Padding(0, 1)

	if !isLast {
		style = style.MarginRight(1)
	}

	return style.Render(content)
}

func (m *Model) renderTicket(ticket *board.Ticket, isSelected bool, width int, columnColor string) string {
	pane, hasPane := m.panes[ticket.ID]
	isRunning := hasPane && pane.Running()

	effectiveStatus := ticket.AgentStatus
	if isRunning && effectiveStatus == board.AgentNone {
		effectiveStatus = board.AgentWorking
	}

	idStr := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf("#%s", string(ticket.ID)[:4]))

	var projectBadge string
	if m.filterProjectID == "" {
		if proj := m.globalStore.GetProjectForTicket(ticket); proj != nil {
			shortName := proj.Name
			if len(shortName) > 12 {
				shortName = shortName[:10] + ".."
			}
			projectBadge = lipgloss.NewStyle().
				Foreground(colorTeal).
				Render("[" + shortName + "]")
		}
	}

	var sessionBadge string
	switch effectiveStatus {
	case board.AgentWorking:
		sessionBadge = lipgloss.NewStyle().
			Foreground(colorYellow).
			Render(m.spinner.View())
	case board.AgentWaiting:
		sessionBadge = lipgloss.NewStyle().
			Foreground(colorMauve).
			Render("◐")
	case board.AgentIdle:
		if hasPane {
			sessionBadge = lipgloss.NewStyle().
				Foreground(colorBlue).
				Render("◆")
		}
	case board.AgentCompleted:
		sessionBadge = lipgloss.NewStyle().
			Foreground(colorGreen).
			Render("✓")
	case board.AgentError:
		sessionBadge = lipgloss.NewStyle().
			Foreground(colorRed).
			Render("✗")
	}

	headerLine := idStr
	if projectBadge != "" {
		headerLine = fmt.Sprintf("%s %s", idStr, projectBadge)
	}
	if sessionBadge != "" {
		headerLine = fmt.Sprintf("%s  %s", headerLine, sessionBadge)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(colorText).
		Bold(isSelected).
		Width(width)
	wrappedTitle := titleStyle.Render(ticket.Title)

	var descLine string
	if ticket.Description != "" {
		desc := ticket.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		desc = strings.ReplaceAll(desc, "\n", " ")
		descLine = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Width(width).
			Render(desc)
	}

	var statusParts []string
	if ticket.AgentType != "" {
		agentBadge := lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorBlue).
			Padding(0, 1).
			Render(ticket.AgentType)
		statusParts = append(statusParts, agentBadge)
	}

	if effectiveStatus != board.AgentNone {
		var statusIcon, statusText string
		var statusColor lipgloss.Color
		switch effectiveStatus {
		case board.AgentIdle:
			statusIcon = "◆"
			statusText = "idle"
			statusColor = colorBlue
		case board.AgentWorking:
			statusIcon = m.spinner.View()
			statusText = "working"
			statusColor = colorYellow
		case board.AgentWaiting:
			statusIcon = "◐"
			statusText = "waiting"
			statusColor = colorMauve
		case board.AgentCompleted:
			statusIcon = "✓"
			statusText = "done"
			statusColor = colorGreen
		case board.AgentError:
			statusIcon = "✗"
			statusText = "error"
			statusColor = colorRed
		}
		statusStyle := lipgloss.NewStyle().Foreground(statusColor)
		statusParts = append(statusParts, statusStyle.Render(statusIcon+" "+statusText))
	}

	statusLine := strings.Join(statusParts, " ")

	var labelParts []string
	for _, label := range ticket.Labels {
		lbl := lipgloss.NewStyle().
			Foreground(colorSubtext).
			Background(colorOverlay).
			Padding(0, 1).
			Render(label)
		labelParts = append(labelParts, lbl)
	}
	labelsLine := strings.Join(labelParts, " ")

	lines := []string{headerLine, wrappedTitle}
	if descLine != "" {
		lines = append(lines, descLine)
	}
	if statusLine != "" {
		lines = append(lines, statusLine)
	}
	if labelsLine != "" {
		lines = append(lines, labelsLine)
	}

	content := strings.Join(lines, "\n")

	border := ticketBorder
	borderColor := colorSurface

	if isSelected {
		border = ticketBorderSelected
		borderColor = lipgloss.Color(columnColor)
	}

	if isRunning {
		borderColor = colorGreen
	}

	cardStyle := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Padding(0, 1).
		MarginBottom(1).
		Width(width)

	return cardStyle.Render(content)
}

func (m *Model) renderStatusBar() string {
	modeStr := modeStyle.Render(string(m.mode))

	sep := lipgloss.NewStyle().Foreground(colorOverlay).Render(" │ ")

	hintStyle := lipgloss.NewStyle().Foreground(colorSubtext)
	hints := hintStyle.Render("h/l") + dimStyle.Render(": move") + sep +
		hintStyle.Render("n") + dimStyle.Render(": new") + sep +
		hintStyle.Render("e") + dimStyle.Render(": edit") + sep +
		hintStyle.Render("s") + dimStyle.Render(": spawn") + sep +
		hintStyle.Render("p") + dimStyle.Render(": filter")

	notif := ""
	if m.notification != "" {
		notifBadge := lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorGreen).
			Padding(0, 1).
			Render("✓ " + m.notification)
		notif = notifBadge
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, modeStr, sep, hints)
	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(notif)
	spacing = max(spacing, 0)

	return lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", spacing), notif)
}

func (m *Model) renderHelp() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(colorTeal).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(colorSubtext)

	sepStyle := lipgloss.NewStyle().
		Foreground(colorOverlay)

	sep := sepStyle.Render("─────────────────────────────")

	help := titleStyle.Render("  ◈ Keyboard Shortcuts") + "\n\n" +
		"  " + keyStyle.Render("Navigation") + "                    " + keyStyle.Render("Actions") + "\n" +
		"  " + sep + "\n" +
		"  " + keyStyle.Render("h/l") + descStyle.Render("   Move between columns  ") + keyStyle.Render("n") + descStyle.Render("       New ticket") + "\n" +
		"  " + keyStyle.Render("j/k") + descStyle.Render("   Move between tickets  ") + keyStyle.Render("e") + descStyle.Render("       Edit ticket") + "\n" +
		"  " + keyStyle.Render("g") + descStyle.Render("     Go to first ticket    ") + keyStyle.Render("d") + descStyle.Render("       Delete ticket") + "\n" +
		"  " + keyStyle.Render("G") + descStyle.Render("     Go to last ticket     ") + keyStyle.Render("Space") + descStyle.Render("   Move forward") + "\n" +
		"  " + keyStyle.Render(" ") + descStyle.Render("                            ") + keyStyle.Render("-") + descStyle.Render("       Move backward") + "\n\n" +
		"  " + keyStyle.Render("Agent") + "                         " + keyStyle.Render("View") + "\n" +
		"  " + sep + "\n" +
		"  " + keyStyle.Render("s") + descStyle.Render("     Spawn agent           ") + keyStyle.Render("p") + descStyle.Render("       Cycle project filter") + "\n" +
		"  " + keyStyle.Render("S") + descStyle.Render("     Stop agent            ") + keyStyle.Render("O") + descStyle.Render("       Settings") + "\n" +
		"  " + keyStyle.Render("Enter") + descStyle.Render(" Attach to agent       ") + keyStyle.Render("?") + descStyle.Render("       Toggle help") + "\n" +
		"  " + keyStyle.Render("Ctrl+g") + descStyle.Render(" Exit agent view       ") + keyStyle.Render("q") + descStyle.Render("       Quit") + "\n\n" +
		"  " + dimStyle.Render("Tip: Hold Shift to select text in agent view") + "\n\n" +
		"  " + dimStyle.Render("Press any key to close")

	return lipgloss.NewStyle().
		Border(columnBorder).
		BorderForeground(colorBlue).
		Padding(1, 2).
		Render(help)
}

func (m *Model) renderConfirmDialog() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)

	content := titleStyle.Render("⚠ Confirm") + "\n\n" +
		"  " + lipgloss.NewStyle().Foreground(colorText).Render(m.confirmMsg) + "\n\n" +
		"  " + lipgloss.NewStyle().Foreground(colorGreen).Render("[y]") + dimStyle.Render(" Yes    ") +
		lipgloss.NewStyle().Foreground(colorRed).Render("[n]") + dimStyle.Render(" No    ") +
		lipgloss.NewStyle().Foreground(colorMuted).Render("[Esc]") + dimStyle.Render(" Cancel")

	return lipgloss.NewStyle().
		Border(columnBorder).
		BorderForeground(colorRed).
		Padding(1, 2).
		Render(content)
}

func (m *Model) renderTicketForm() string {
	isEdit := m.mode == ModeEditTicket
	formTitle := "New Ticket"
	actionText := "Create"
	if isEdit {
		formTitle = "Edit Ticket"
		actionText = "Save"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)

	labelStyle := lipgloss.NewStyle().Foreground(colorSubtext)
	activeLabelStyle := lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
	lockedStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	titleLabel := labelStyle
	descLabel := labelStyle
	branchLabel := labelStyle
	projectLabel := labelStyle

	switch m.ticketFormField {
	case formFieldTitle:
		titleLabel = activeLabelStyle
	case formFieldDescription:
		descLabel = activeLabelStyle
	case formFieldBranch:
		branchLabel = activeLabelStyle
	case formFieldProject:
		projectLabel = activeLabelStyle
	}

	var branchField string
	if m.branchLocked {
		branchLabel = lockedStyle
		branchField = lockedStyle.Render(m.branchInput.Value() + " (locked)")
	} else {
		branchField = m.branchInput.View()
	}

	var projectField string
	if m.selectedProject != nil {
		projectField = lipgloss.NewStyle().Foreground(colorTeal).Render(m.selectedProject.Name)
		if m.ticketFormField == formFieldProject {
			projectField += dimStyle.Render(" (Enter to cycle)")
		}
	} else {
		projectField = dimStyle.Render("No project selected")
	}

	content := titleStyle.Render("◈ "+formTitle) + "\n\n" +
		"  " + titleLabel.Render("Title:") + "\n" +
		"  " + m.titleInput.View() + "\n\n" +
		"  " + descLabel.Render("Description:") + "\n" +
		"  " + m.descInput.View() + "\n\n" +
		"  " + branchLabel.Render("Branch:") + "\n" +
		"  " + branchField + "\n"

	if !isEdit && len(m.globalStore.Projects()) > 1 {
		content += "\n  " + projectLabel.Render("Project:") + "\n" +
			"  " + projectField + "\n"
	}

	content += "\n  " + lipgloss.NewStyle().Foreground(colorTeal).Render("[Tab]") + dimStyle.Render(" Switch    ") +
		lipgloss.NewStyle().Foreground(colorGreen).Render("[Ctrl+S]") + dimStyle.Render(" "+actionText+"    ") +
		lipgloss.NewStyle().Foreground(colorMuted).Render("[Esc]") + dimStyle.Render(" Cancel")

	return lipgloss.NewStyle().
		Border(columnBorder).
		BorderForeground(colorGreen).
		Padding(1, 2).
		Render(content)
}

func (m *Model) renderWithOverlay(overlay string) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(colorBase),
	)
}

func (m *Model) renderSettingsView() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(colorMauve).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(colorSubtext)

	valueStyle := lipgloss.NewStyle().
		Foreground(colorText)

	selectedLabelStyle := lipgloss.NewStyle().
		Foreground(colorMauve).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render("◈ Global Settings"))
	lines = append(lines, "")

	for i, field := range settingsFields {
		label := field.label
		value := ""

		switch field.key {
		case "filter_project":
			if m.filterProjectID == "" {
				value = "All Projects"
			} else if p := m.globalStore.GetProject(m.filterProjectID); p != nil {
				value = p.Name
			}
		}

		cursor := "  "
		lStyle := labelStyle
		vStyle := valueStyle

		if i == m.settingsIndex {
			cursor = lipgloss.NewStyle().Foreground(colorMauve).Render("▸ ")
			lStyle = selectedLabelStyle
			vStyle = lipgloss.NewStyle().Foreground(colorTeal)
		}

		line := cursor + lStyle.Render(fmt.Sprintf("%-18s", label)) + " " + vStyle.Render(value)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, "  "+lipgloss.NewStyle().Foreground(colorTeal).Render("[Enter/Space]")+dimStyle.Render(" Cycle  ")+
		lipgloss.NewStyle().Foreground(colorMuted).Render("[Esc]")+dimStyle.Render(" Close"))

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Border(columnBorder).
		BorderForeground(colorMauve).
		Padding(1, 2).
		Render(content)
}

func (m *Model) renderAgentView() string {
	pane, ok := m.panes[m.focusedPane]
	if !ok {
		return "No pane focused"
	}

	var b strings.Builder

	ticket, _ := m.globalStore.Get(m.focusedPane)
	title := "Agent"
	agentType := ""
	projectName := ""
	if ticket != nil {
		title = ticket.Title
		agentType = ticket.AgentType
		if proj := m.globalStore.GetProjectForTicket(ticket); proj != nil {
			projectName = proj.Name
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true)

	header := titleStyle.Render("◈ " + title)

	if projectName != "" {
		projBadge := lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorTeal).
			Padding(0, 1).
			Render(projectName)
		header = header + "  " + projBadge
	}

	if agentType != "" {
		agentBadge := lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorBlue).
			Padding(0, 1).
			Render(agentType)
		header = header + "  " + agentBadge
	}

	activePaneCount := 0
	paneIndex := 0
	for id, p := range m.panes {
		if p.Running() {
			activePaneCount++
			if id == m.focusedPane {
				paneIndex = activePaneCount
			}
		}
	}

	paneIndicator := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf("[%d/%d]", paneIndex, activePaneCount))

	keyStyle := lipgloss.NewStyle().Foreground(colorTeal)
	hints := paneIndicator + "  " +
		keyStyle.Render("Ctrl+g") + dimStyle.Render(" Board")

	spacing := m.width - lipgloss.Width(header) - lipgloss.Width(hints)
	spacing = max(spacing, 0)

	b.WriteString(header)
	b.WriteString(strings.Repeat(" ", spacing))
	b.WriteString(hints)
	b.WriteString("\n")

	b.WriteString(pane.View())

	return b.String()
}

var (
	colorBase    = lipgloss.Color("#1e1e2e")
	colorSurface = lipgloss.Color("#313244")
	colorOverlay = lipgloss.Color("#45475a")
	colorText    = lipgloss.Color("#cdd6f4")
	colorSubtext = lipgloss.Color("#a6adc8")
	colorMuted   = lipgloss.Color("#6c7086")
	colorBlue    = lipgloss.Color("#89b4fa")
	colorGreen   = lipgloss.Color("#a6e3a1")
	colorYellow  = lipgloss.Color("#f9e2af")
	colorRed     = lipgloss.Color("#f38ba8")
	colorMauve   = lipgloss.Color("#cba6f7")
	colorTeal    = lipgloss.Color("#94e2d5")
)

var (
	columnBorder = lipgloss.Border{
		Top:         "━",
		Bottom:      "━",
		Left:        "┃",
		Right:       "┃",
		TopLeft:     "┏",
		TopRight:    "┓",
		BottomLeft:  "┗",
		BottomRight: "┛",
	}

	columnBorderActive = lipgloss.Border{
		Top:         "━",
		Bottom:      "━",
		Left:        "┃",
		Right:       "┃",
		TopLeft:     "┏",
		TopRight:    "┓",
		BottomLeft:  "┗",
		BottomRight: "┛",
	}

	ticketBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	ticketBorderSelected = lipgloss.Border{
		Top:         "═",
		Bottom:      "═",
		Left:        "║",
		Right:       "║",
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
	}
)

var (
	dimStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	modeStyle = lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorBlue).
			Bold(true).
			Padding(0, 1)
)
