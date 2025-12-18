package agent

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/techdufus/openkanban/internal/board"
)

type ContextData struct {
	Title        string
	Description  string
	BranchName   string
	BaseBranch   string
	TicketID     string
	Status       string
	WorktreePath string
}

func BuildContextPrompt(promptTemplate string, ticket *board.Ticket) string {
	if promptTemplate == "" {
		return ""
	}

	data := ContextData{
		Title:        ticket.Title,
		Description:  ticket.Description,
		BranchName:   ticket.BranchName,
		BaseBranch:   ticket.BaseBranch,
		TicketID:     string(ticket.ID),
		Status:       string(ticket.Status),
		WorktreePath: ticket.WorktreePath,
	}

	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return buildFallbackPrompt(ticket)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return buildFallbackPrompt(ticket)
	}

	return buf.String()
}

func buildFallbackPrompt(ticket *board.Ticket) string {
	var sb strings.Builder
	sb.WriteString("Task: ")
	sb.WriteString(ticket.Title)
	if ticket.Description != "" {
		sb.WriteString("\n\n")
		sb.WriteString(ticket.Description)
	}
	return sb.String()
}

func ShouldInjectContext(ticket *board.Ticket) bool {
	return ticket.AgentSpawnedAt == nil
}
