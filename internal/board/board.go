package board

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)

func Slugify(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 40
	}

	slug := strings.ToLower(s)

	slug = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, slug)

	slug = nonAlphanumericRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	if len(slug) > maxLen {
		slug = slug[:maxLen]
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

type TicketID string

func NewTicketID() TicketID {
	return TicketID(uuid.New().String())
}

type TicketStatus string

const (
	StatusBacklog    TicketStatus = "backlog"
	StatusInProgress TicketStatus = "in_progress"
	StatusDone       TicketStatus = "done"
	StatusArchived   TicketStatus = "archived"
)

type AgentStatus string

const (
	AgentNone      AgentStatus = "none"
	AgentIdle      AgentStatus = "idle"
	AgentWorking   AgentStatus = "working"
	AgentWaiting   AgentStatus = "waiting"
	AgentCompleted AgentStatus = "completed"
	AgentError     AgentStatus = "error"
)

type Ticket struct {
	ID          TicketID     `json:"id"`
	ProjectID   string       `json:"project_id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TicketStatus `json:"status"`

	WorktreePath string `json:"worktree_path,omitempty"`
	BranchName   string `json:"branch_name,omitempty"`
	BaseBranch   string `json:"base_branch,omitempty"`

	AgentType      string      `json:"agent_type,omitempty"`
	AgentStatus    AgentStatus `json:"agent_status"`
	AgentSpawnedAt *time.Time  `json:"agent_spawned_at,omitempty"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Labels   []string          `json:"labels,omitempty"`
	Priority int               `json:"priority,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

func NewTicket(title, projectID string) *Ticket {
	now := time.Now()
	return &Ticket{
		ID:          NewTicketID(),
		ProjectID:   projectID,
		Title:       title,
		Status:      StatusBacklog,
		AgentStatus: AgentNone,
		Priority:    3,
		CreatedAt:   now,
		UpdatedAt:   now,
		Labels:      []string{},
		Meta:        map[string]string{},
	}
}

func (t *Ticket) Touch() {
	t.UpdatedAt = time.Now()
}

func (t *Ticket) SetStatus(status TicketStatus) {
	now := time.Now()
	t.Status = status
	t.UpdatedAt = now

	switch status {
	case StatusInProgress:
		t.StartedAt = &now
	case StatusDone:
		t.CompletedAt = &now
	}
}

type Column struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Status TicketStatus `json:"status"`
	Color  string       `json:"color"`
	Limit  int          `json:"limit"`
}

func DefaultColumns() []Column {
	return []Column{
		{ID: "backlog", Name: "Backlog", Status: StatusBacklog, Color: "#89b4fa", Limit: 0},
		{ID: "in-progress", Name: "In Progress", Status: StatusInProgress, Color: "#f9e2af", Limit: 3},
		{ID: "done", Name: "Done", Status: StatusDone, Color: "#a6e3a1", Limit: 0},
	}
}

var (
	ErrTicketNotFound = &BoardError{Message: "ticket not found"}
)

type BoardError struct {
	Message string
}

func (e *BoardError) Error() string {
	return e.Message
}
