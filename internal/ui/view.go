package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/techdufus/openkanban/internal/board"
)

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		loadingStyle := lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)
		return lipgloss.Place(
			80, 24,
			lipgloss.Center, lipgloss.Center,
			loadingStyle.Render("◈ Initializing..."),
		)
	}

	if m.mode == ModeShuttingDown {
		return m.renderShuttingDown()
	}

	if m.mode == ModeSpawning {
		return m.renderSpawning()
	}

	if m.mode == ModeAgentView && m.focusedPane != "" {
		return m.renderAgentView()
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	sidebar := m.renderSidebar()
	board := m.renderBoard()
	if sidebar != "" {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, board))
	} else {
		b.WriteString(board)
	}

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

	var filterSection string
	if m.mode == ModeFilter {
		filterSection = m.renderFilterInput()
	} else if m.filterQuery != "" || m.filterProjectID != "" {
		filterSection = m.renderActiveFilter()
	} else {
		filterSection = m.renderFilterHint()
	}

	projectCount := len(m.globalStore.Projects())
	ticketCount := m.globalStore.Count()
	visibleCount := m.countVisibleTickets()
	var stats string
	if m.filterQuery != "" || m.filterProjectID != "" {
		stats = dimStyle.Render(fmt.Sprintf("showing %d of %d", visibleCount, ticketCount))
	} else {
		stats = dimStyle.Render(fmt.Sprintf("%d projects, %d tickets", projectCount, ticketCount))
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", filterSection, "  ", stats)

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
	help := helpStyle.Render("? help  q quit")

	right := help
	if activity != "" {
		right = lipgloss.JoinHorizontal(lipgloss.Center, activity, "  ", help)
	}

	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	spacing = max(spacing, 0)

	header := lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", spacing), right)

	return lipgloss.NewStyle().
		PaddingTop(1).
		PaddingBottom(1).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorSurface).
		Width(m.width).
		Render(header)
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
			Foreground(colorMuted).
			Background(colorSurface).
			Padding(0, 1).
			Render(fmt.Sprintf("◀ %d", startCol))
		columns = append(columns, indicator)
	}

	for i := startCol; i < endCol; i++ {
		col := m.columns[i]
		isActive := i == m.activeColumn && !m.sidebarFocused
		isLast := i == endCol-1
		isDragTarget := m.dragging && i == m.dragTargetColumn && i != m.dragSourceColumn
		isHovered := i == m.hoverColumn && !m.dragging

		colWidth := baseWidth
		if i-startCol < remainder {
			colWidth++
		}

		ticketOffset := 0
		if i < len(m.columnOffsets) {
			ticketOffset = m.columnOffsets[i]
		}

		columns = append(columns, m.renderColumn(col, m.columnTickets[i], isActive, isDragTarget, isHovered, colWidth, isLast, ticketOffset))
	}

	if endCol < len(m.columns) {
		remaining := len(m.columns) - endCol
		indicator := lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorSurface).
			Padding(0, 1).
			Render(fmt.Sprintf("%d ▶", remaining))
		columns = append(columns, indicator)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m *Model) renderColumn(col board.Column, tickets []*board.Ticket, isActive, isDragTarget, isHovered bool, width int, isLast bool, ticketOffset int) string {
	headerColor := lipgloss.Color(col.Color)

	columnIcons := map[board.TicketStatus]string{
		board.StatusBacklog:    "📋",
		board.StatusInProgress: "⚡",
		board.StatusDone:       "✅",
	}
	icon := columnIcons[col.Status]
	if icon == "" {
		icon = "○"
	}
	if isActive {
		icon = "▸ " + icon
	}

	headerText := fmt.Sprintf("%s %s", icon, col.Name)

	countStyle := lipgloss.NewStyle().Foreground(colorMuted)
	countText := fmt.Sprintf("(%d)", len(tickets))
	if col.Limit > 0 {
		countText = fmt.Sprintf("(%d/%d)", len(tickets), col.Limit)
		if len(tickets) >= col.Limit {
			countStyle = lipgloss.NewStyle().
				Foreground(colorBase).
				Background(colorRed).
				Padding(0, 1)
		}
	}

	header := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true).
		Render(headerText)

	count := countStyle.Render(" " + countText)

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
		isTicketHovered := isHovered && i == m.hoverTicket
		ticketViews = append(ticketViews, m.renderTicket(ticket, isSelected, isTicketHovered, width-4, col.Color))
	}

	if hasMoreBelow {
		remaining := len(tickets) - endIdx
		ticketViews = append(ticketViews, indicatorStyle.Render(fmt.Sprintf("▼ %d more", remaining)))
	}

	ticketsView := strings.Join(ticketViews, "\n")
	if len(tickets) == 0 {
		emptyIcon := "○"
		emptyText := "Drop tickets here"
		if col.Status == board.StatusBacklog {
			emptyIcon = "+"
			emptyText = "Press 'n' to create"
		} else if col.Status == board.StatusDone {
			emptyIcon = "✓"
			emptyText = "Completed work appears here"
		}
		emptyStyle := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(2, 0).
			Width(width - 4).
			Align(lipgloss.Center)
		ticketsView = emptyStyle.Render(emptyIcon + "\n" + emptyText)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, headerLine, "", ticketsView)

	border := columnBorder
	borderColor := colorSurface
	if isDragTarget {
		border = dragTargetBorder
		borderColor = colorGreen
	} else if isActive {
		border = columnBorderActive
		borderColor = headerColor
	} else if isHovered {
		borderColor = colorOverlay
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

func (m *Model) renderTicket(ticket *board.Ticket, isSelected, isHovered bool, width int, columnColor string) string {
	pane, hasPane := m.panes[ticket.ID]
	isRunning := hasPane && pane.Running()

	effectiveStatus := ticket.AgentStatus
	if isRunning && effectiveStatus == board.AgentNone {
		effectiveStatus = board.AgentWorking
	}

	var projectBadge string
	if proj := m.globalStore.GetProjectForTicket(ticket); proj != nil {
		shortName := proj.Name
		if len(shortName) > 12 {
			shortName = shortName[:10] + ".."
		}
		bracketStyle := lipgloss.NewStyle().Foreground(colorTeal)
		textStyle := lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
		projectBadge = bracketStyle.Render("❨") + textStyle.Render(shortName) + bracketStyle.Render("❩")
	}

	var sessionBadge string
	switch effectiveStatus {
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

	var headerParts []string
	if projectBadge != "" {
		headerParts = append(headerParts, projectBadge)
	}
	if sessionBadge != "" {
		headerParts = append(headerParts, sessionBadge)
	}
	headerLine := strings.Join(headerParts, "  ")

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

	var accentColor lipgloss.Color = colorSurface
	switch effectiveStatus {
	case board.AgentWorking:
		accentColor = colorYellow
	case board.AgentWaiting:
		accentColor = colorMauve
	case board.AgentIdle:
		if hasPane {
			accentColor = colorBlue
		}
	case board.AgentCompleted:
		accentColor = colorGreen
	case board.AgentError:
		accentColor = colorRed
	}
	if isRunning {
		accentColor = colorGreen
	}

	border := ticketBorder
	borderColor := colorSurface

	if isHovered && !isSelected {
		borderColor = colorOverlay
	}

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
		BorderLeftForeground(accentColor).
		Padding(0, 1).
		MarginBottom(1).
		Width(width)

	return cardStyle.Render(content)
}

func (m *Model) renderStatusBar() string {
	type modeConfig struct {
		icon string
		bg   lipgloss.Color
	}
	modeConfigs := map[Mode]modeConfig{
		ModeNormal:       {"◆", colorBlue},
		ModeInsert:       {"✎", colorGreen},
		ModeCommand:      {":", colorMauve},
		ModeCreateTicket: {"+", colorGreen},
		ModeEditTicket:   {"✎", colorYellow},
		ModeAgentView:    {"▶", colorTeal},
		ModeSettings:     {"⚙", colorMauve},
		ModeHelp:         {"?", colorBlue},
		ModeConfirm:      {"!", colorRed},
	}
	cfg := modeConfigs[m.mode]
	if cfg.bg == "" {
		cfg = modeConfig{"◆", colorBlue}
	}
	modeStr := lipgloss.NewStyle().
		Foreground(colorBase).
		Background(cfg.bg).
		Bold(true).
		Padding(0, 1).
		Render(cfg.icon + " " + string(m.mode))

	sep := lipgloss.NewStyle().Foreground(colorOverlay).Render(" │ ")

	hintStyle := lipgloss.NewStyle().Foreground(colorSubtext)
	hints := hintStyle.Render("h/l") + dimStyle.Render(": move") + sep +
		hintStyle.Render("n") + dimStyle.Render(": new") + sep +
		hintStyle.Render("e") + dimStyle.Render(": edit") + sep +
		hintStyle.Render("s") + dimStyle.Render(": spawn") + sep +
		hintStyle.Render("/") + dimStyle.Render(": search")

	notif := ""
	if m.notification != "" {
		isError := strings.HasPrefix(m.notification, "Failed") ||
			strings.HasPrefix(m.notification, "Error") ||
			strings.Contains(m.notification, "failed")
		bgColor := colorGreen
		icon := "✓"
		if isError {
			bgColor = colorRed
			icon = "✗"
		}
		notifBadge := lipgloss.NewStyle().
			Foreground(colorBase).
			Background(bgColor).
			Padding(0, 1).
			Render(icon + " " + m.notification)
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

	sectionStyle := lipgloss.NewStyle().
		Foreground(colorMauve).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(colorTeal).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(colorSubtext)

	sepStyle := lipgloss.NewStyle().
		Foreground(colorSurface)

	sep := sepStyle.Render("────────────────────────────────────────────")

	help := titleStyle.Render("◈ Keyboard Shortcuts") + "\n\n" +
		sep + "\n" +
		sectionStyle.Render("  🧭 Navigation") + "                 " + sectionStyle.Render("📝 Actions") + "\n" +
		sep + "\n" +
		"  " + keyStyle.Render("h/l") + descStyle.Render("   Move between columns  ") + keyStyle.Render("n") + descStyle.Render("       New ticket") + "\n" +
		"  " + keyStyle.Render("j/k") + descStyle.Render("   Move between tickets  ") + keyStyle.Render("e") + descStyle.Render("       Edit ticket") + "\n" +
		"  " + keyStyle.Render("g") + descStyle.Render("     Go to first ticket    ") + keyStyle.Render("d") + descStyle.Render("       Delete ticket") + "\n" +
		"  " + keyStyle.Render("G") + descStyle.Render("     Go to last ticket     ") + keyStyle.Render("Space") + descStyle.Render("   Move forward") + "\n" +
		"  " + keyStyle.Render(" ") + descStyle.Render("                            ") + keyStyle.Render("-") + descStyle.Render("       Move backward") + "\n\n" +
		sep + "\n" +
		sectionStyle.Render("  📂 Sidebar") + "                    " + sectionStyle.Render("🤖 Agent") + "\n" +
		sep + "\n" +
		"  " + keyStyle.Render("[") + descStyle.Render("     Toggle sidebar        ") + keyStyle.Render("s") + descStyle.Render("       Spawn agent") + "\n" +
		"  " + keyStyle.Render("h") + descStyle.Render("     Enter sidebar         ") + keyStyle.Render("S") + descStyle.Render("       Stop agent") + "\n" +
		"  " + keyStyle.Render("l") + descStyle.Render("     Exit sidebar          ") + keyStyle.Render("Enter") + descStyle.Render("   Attach to agent") + "\n" +
		"  " + keyStyle.Render("j/k") + descStyle.Render("   Navigate projects     ") + keyStyle.Render("Ctrl+g") + descStyle.Render("  Exit agent view") + "\n\n" +
		sep + "\n" +
		sectionStyle.Render("  👁 View") + "\n" +
		sep + "\n" +
		"  " + keyStyle.Render("/") + descStyle.Render("     Search/filter         ") + keyStyle.Render("O") + descStyle.Render("       Settings") + "\n" +
		"  " + keyStyle.Render("?") + descStyle.Render("     Toggle help           ") + keyStyle.Render("q") + descStyle.Render("       Quit") + "\n\n" +
		sep + "\n" +
		"  " + lipgloss.NewStyle().Foreground(colorYellow).Render("💡") + dimStyle.Render(" Tip: Hold Shift to select text in agent view") + "\n\n" +
		"  " + dimStyle.Render("Press any key to close")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
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

func (m *Model) renderShuttingDown() string {
	count := m.RunningAgentCount()
	msg := fmt.Sprintf("Stopping %d agent(s)...", count)

	titleStyle := lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	content := titleStyle.Render(m.spinner.View()+" Shutting Down") + "\n\n" +
		"  " + lipgloss.NewStyle().Foreground(colorText).Render(msg)

	dialog := lipgloss.NewStyle().
		Border(columnBorder).
		BorderForeground(colorYellow).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (m *Model) renderSpawning() string {
	agentName := m.spawningAgent
	if agentName == "" {
		agentName = "agent"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)

	content := titleStyle.Render(m.spinner.View()+" Starting "+agentName) + "\n\n" +
		"  " + dimStyle.Render("[Esc] Cancel")

	dialog := lipgloss.NewStyle().
		Border(columnBorder).
		BorderForeground(colorGreen).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
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

	projectField := m.renderProjectSelector()

	titleCharCount := fmt.Sprintf("%d/100", len(m.titleInput.Value()))
	titleCharStyle := lipgloss.NewStyle().Foreground(colorMuted)
	if len(m.titleInput.Value()) > 80 {
		titleCharStyle = lipgloss.NewStyle().Foreground(colorYellow)
	}
	if len(m.titleInput.Value()) >= 100 {
		titleCharStyle = lipgloss.NewStyle().Foreground(colorRed)
	}

	focusIndicator := lipgloss.NewStyle().Foreground(colorTeal).Render("▸ ")
	noFocus := "  "

	titleFocus, descFocus, branchFocus, projectFocus := noFocus, noFocus, noFocus, noFocus
	switch m.ticketFormField {
	case formFieldTitle:
		titleFocus = focusIndicator
	case formFieldDescription:
		descFocus = focusIndicator
	case formFieldBranch:
		branchFocus = focusIndicator
	case formFieldProject:
		projectFocus = focusIndicator
	}

	content := titleStyle.Render("◈ "+formTitle) + "\n\n" +
		titleFocus + titleLabel.Render("Title") + "  " + titleCharStyle.Render(titleCharCount) + "\n" +
		"  " + m.titleInput.View() + "\n\n" +
		descFocus + descLabel.Render("Description") + "\n" +
		"  " + m.descInput.View() + "\n\n" +
		branchFocus + branchLabel.Render("Branch") + "\n" +
		"  " + branchField + "\n"

	if !isEdit {
		content += "\n" + projectFocus + projectLabel.Render("Project") + "\n" +
			"  " + projectField + "\n"
	}

	content += "\n  " + lipgloss.NewStyle().Foreground(colorTeal).Render("[Tab]") + dimStyle.Render(" Switch    ") +
		lipgloss.NewStyle().Foreground(colorGreen).Render("[Ctrl+S]") + dimStyle.Render(" "+actionText+"    ") +
		lipgloss.NewStyle().Foreground(colorMuted).Render("[Esc]") + dimStyle.Render(" Cancel")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
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
	lines = append(lines, "  "+lipgloss.NewStyle().Foreground(colorTeal).Render("[Enter]")+dimStyle.Render(" Open search  ")+
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
	var sessionDuration string
	if ticket != nil {
		title = ticket.Title
		agentType = ticket.AgentType
		if proj := m.globalStore.GetProjectForTicket(ticket); proj != nil {
			projectName = proj.Name
		}
		if ticket.AgentSpawnedAt != nil {
			duration := time.Since(*ticket.AgentSpawnedAt)
			sessionDuration = formatDuration(duration)
		}
	}

	breadcrumbStyle := lipgloss.NewStyle().Foreground(colorMuted)
	titleStyle := lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true)

	header := breadcrumbStyle.Render("Board → ") + titleStyle.Render(title)

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

	if sessionDuration != "" {
		durationBadge := lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("⏱ " + sessionDuration)
		header = header + "  " + durationBadge
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

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func (m *Model) renderFilterInput() string {
	inputStyle := lipgloss.NewStyle().
		Foreground(colorBase).
		Background(colorTeal).
		Padding(0, 1)
	return inputStyle.Render("/ " + m.filterInput.View())
}

func (m *Model) renderActiveFilter() string {
	filterStyle := lipgloss.NewStyle().
		Foreground(colorBase).
		Background(colorYellow).
		Bold(true).
		Padding(0, 1)

	clearStyle := lipgloss.NewStyle().
		Foreground(colorBase).
		Background(colorRed).
		Padding(0, 1)

	filterText := m.filterQuery
	if m.filterProjectID != "" && m.filterQuery == "" {
		if p := m.globalStore.GetProject(m.filterProjectID); p != nil {
			filterText = "@" + p.Name
		}
	}

	return filterStyle.Render("FILTERED: "+filterText) + " " + clearStyle.Render("× clear")
}

func (m *Model) renderFilterHint() string {
	return lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("/ search (@project to filter)")
}

func (m *Model) countVisibleTickets() int {
	count := 0
	for _, tickets := range m.columnTickets {
		count += len(tickets)
	}
	return count
}

func (m *Model) renderProjectSelector() string {
	projects := m.globalStore.Projects()
	if len(projects) == 0 {
		return dimStyle.Render("No projects available")
	}

	if m.ticketFormField != formFieldProject {
		if m.selectedProject != nil {
			return lipgloss.NewStyle().Foreground(colorTeal).Render(m.selectedProject.Name)
		}
		return dimStyle.Render("Select project...")
	}

	if m.showAddProjectForm {
		return m.renderAddProjectForm()
	}

	var lines []string
	for i, p := range projects {
		name := p.Name
		path := shortenPath(p.RepoPath)

		nameStyle := lipgloss.NewStyle().Foreground(colorText)
		pathStyle := lipgloss.NewStyle().Foreground(colorMuted)
		prefix := "  "

		if i == m.projectListIndex {
			nameStyle = nameStyle.Foreground(colorTeal).Bold(true)
			pathStyle = pathStyle.Foreground(colorSubtext)
			prefix = lipgloss.NewStyle().Foreground(colorTeal).Render("● ")
		} else {
			prefix = "○ "
		}

		line := prefix + nameStyle.Render(name) + "  " + pathStyle.Render(path)
		lines = append(lines, line)
	}

	addOption := "○ " + lipgloss.NewStyle().Foreground(colorGreen).Render("+ Add project...")
	if m.projectListIndex == len(projects) {
		addOption = lipgloss.NewStyle().Foreground(colorTeal).Render("● ") +
			lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("+ Add project...")
	}
	lines = append(lines, addOption)
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("j/k navigate  Enter select  d delete"))

	return strings.Join(lines, "\n  ")
}

func (m *Model) renderAddProjectForm() string {
	titleStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	return titleStyle.Render("Add Project") + "\n\n" +
		"  " + lipgloss.NewStyle().Foreground(colorSubtext).Render("Repository path:") + "\n" +
		"  " + m.addProjectPath.View() + "\n\n" +
		"  " + dimStyle.Render("[Enter] Add  [Esc] Cancel")
}

func shortenPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func (m *Model) renderSidebar() string {
	if !m.sidebarVisible {
		return ""
	}

	projects := m.globalStore.Projects()
	headerHeight := 5
	statusHeight := 1
	availableHeight := m.height - headerHeight - statusHeight

	titleStyle := lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(colorBase).
		Background(colorBlue).
		Bold(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(colorText).
		Padding(0, 1)

	mutedStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Padding(0, 1)

	var lines []string

	lines = append(lines, titleStyle.Render("  Projects"))
	lines = append(lines, "")

	allCount := m.globalStore.Count()
	allLabel := fmt.Sprintf("All (%d)", allCount)
	if m.sidebarIndex == 0 {
		if m.sidebarFocused {
			lines = append(lines, selectedStyle.Render("● "+allLabel))
		} else {
			lines = append(lines, normalStyle.Render("● "+allLabel))
		}
	} else {
		if m.filterProjectID == "" {
			lines = append(lines, normalStyle.Render("● "+allLabel))
		} else {
			lines = append(lines, mutedStyle.Render("  "+allLabel))
		}
	}

	lines = append(lines, "")

	for i, p := range projects {
		idx := i + 1
		count := 0
		for _, t := range m.globalStore.All() {
			if t.ProjectID == p.ID {
				count++
			}
		}
		label := fmt.Sprintf("%s (%d)", p.Name, count)

		isFiltered := m.filterProjectID == p.ID

		if m.sidebarIndex == idx && m.sidebarFocused {
			lines = append(lines, selectedStyle.Render("● "+label))
		} else if isFiltered {
			lines = append(lines, normalStyle.Render("● "+label))
		} else {
			lines = append(lines, mutedStyle.Render("  "+label))
		}
	}

	lines = append(lines, "")
	addIndex := len(projects) + 1
	if m.sidebarIndex == addIndex && m.sidebarFocused {
		lines = append(lines, selectedStyle.Render("+ Add project"))
	} else {
		addStyle := lipgloss.NewStyle().Foreground(colorGreen).Padding(0, 1)
		lines = append(lines, addStyle.Render("+ Add project"))
	}

	for len(lines) < availableHeight-2 {
		lines = append(lines, "")
	}

	hintStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	if m.sidebarFocused {
		lines = append(lines, hintStyle.Render("  j/k ⏎select l→exit"))
	} else {
		lines = append(lines, hintStyle.Render("  h→focus  [hide"))
	}

	content := strings.Join(lines, "\n")

	style := lipgloss.NewStyle().
		Width(m.sidebarWidth).
		Height(availableHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder())

	if m.sidebarFocused {
		style = style.BorderForeground(colorBlue)
	} else {
		style = style.BorderForeground(colorSurface)
	}

	return style.Render(content)
}

func (m *Model) boardWidth() int {
	if m.sidebarVisible {
		return m.width - m.sidebarWidth - 1
	}
	return m.width
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

	dragTargetBorder = lipgloss.Border{
		Top:         "═",
		Bottom:      "═",
		Left:        "║",
		Right:       "║",
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
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
