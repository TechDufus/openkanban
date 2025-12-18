package board

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// TicketID is a unique identifier for a ticket
type TicketID string

// NewTicketID generates a new unique ticket ID
func NewTicketID() TicketID {
	return TicketID(uuid.New().String())
}

// TicketStatus represents the workflow status of a ticket
type TicketStatus string

const (
	StatusBacklog    TicketStatus = "backlog"
	StatusInProgress TicketStatus = "in_progress"
	StatusDone       TicketStatus = "done"
	StatusArchived   TicketStatus = "archived"
)

// AgentStatus represents the current state of an AI agent
type AgentStatus string

const (
	AgentNone      AgentStatus = "none"
	AgentIdle      AgentStatus = "idle"
	AgentWorking   AgentStatus = "working"
	AgentWaiting   AgentStatus = "waiting"
	AgentCompleted AgentStatus = "completed"
	AgentError     AgentStatus = "error"
)

// Ticket represents a unit of work
type Ticket struct {
	ID          TicketID     `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TicketStatus `json:"status"`

	// Git integration
	WorktreePath string `json:"worktree_path,omitempty"`
	BranchName   string `json:"branch_name,omitempty"`
	BaseBranch   string `json:"base_branch,omitempty"`

	// Agent integration
	AgentType      string      `json:"agent_type,omitempty"`
	AgentStatus    AgentStatus `json:"agent_status"`
	AgentSpawnedAt *time.Time  `json:"agent_spawned_at,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Metadata
	Labels   []string          `json:"labels,omitempty"`
	Priority int               `json:"priority,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// NewTicket creates a new ticket with defaults
func NewTicket(title string) *Ticket {
	now := time.Now()
	return &Ticket{
		ID:          NewTicketID(),
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

// Column represents a kanban column
type Column struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Status TicketStatus `json:"status"`
	Color  string       `json:"color"`
	Limit  int          `json:"limit"`
}

// BoardSettings contains board-specific settings
type BoardSettings struct {
	DefaultAgent     string `json:"default_agent"`
	WorktreeBase     string `json:"worktree_base"`
	AutoSpawnAgent   bool   `json:"auto_spawn_agent"`
	AutoCreateBranch bool   `json:"auto_create_branch"`
	BranchPrefix     string `json:"branch_prefix"`
	BranchNaming     string `json:"branch_naming"`   // "template" | "ai" | "prompt"
	BranchTemplate   string `json:"branch_template"` // e.g., "{prefix}{slug}"
	SlugMaxLength    int    `json:"slug_max_length"` // default: 40
}

// Board represents a kanban board
type Board struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	RepoPath  string               `json:"repo_path"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	Columns   []Column             `json:"columns"`
	Tickets   map[TicketID]*Ticket `json:"tickets"`
	Settings  BoardSettings        `json:"settings"`
}

// NewBoard creates a new board with default columns
func NewBoard(name, repoPath string, settings BoardSettings) *Board {
	now := time.Now()

	// Default worktree base if not set
	if settings.WorktreeBase == "" {
		settings.WorktreeBase = repoPath + "-worktrees"
	}

	return &Board{
		ID:        uuid.New().String(),
		Name:      name,
		RepoPath:  repoPath,
		CreatedAt: now,
		UpdatedAt: now,
		Columns: []Column{
			{ID: "backlog", Name: "Backlog", Status: StatusBacklog, Color: "#89b4fa", Limit: 0},
			{ID: "in-progress", Name: "In Progress", Status: StatusInProgress, Color: "#f9e2af", Limit: 3},
			{ID: "done", Name: "Done", Status: StatusDone, Color: "#a6e3a1", Limit: 0},
		},
		Tickets:  make(map[TicketID]*Ticket),
		Settings: settings,
	}
}

// GetTicketsByStatus returns all tickets with the given status
func (b *Board) GetTicketsByStatus(status TicketStatus) []*Ticket {
	var tickets []*Ticket
	for _, t := range b.Tickets {
		if t.Status == status {
			tickets = append(tickets, t)
		}
	}
	return tickets
}

// AddTicket adds a ticket to the board
func (b *Board) AddTicket(t *Ticket) {
	b.Tickets[t.ID] = t
	b.UpdatedAt = time.Now()
}

// MoveTicket changes a ticket's status
func (b *Board) MoveTicket(id TicketID, newStatus TicketStatus) error {
	t, ok := b.Tickets[id]
	if !ok {
		return ErrTicketNotFound
	}

	now := time.Now()
	t.Status = newStatus
	t.UpdatedAt = now

	switch newStatus {
	case StatusInProgress:
		t.StartedAt = &now
	case StatusDone:
		t.CompletedAt = &now
	}

	b.UpdatedAt = now
	return nil
}

// DeleteTicket removes a ticket from the board
func (b *Board) DeleteTicket(id TicketID) error {
	if _, ok := b.Tickets[id]; !ok {
		return ErrTicketNotFound
	}
	delete(b.Tickets, id)
	b.UpdatedAt = time.Now()
	return nil
}

// Errors
var (
	ErrTicketNotFound = &BoardError{Message: "ticket not found"}
	ErrBoardNotFound  = &BoardError{Message: "board not found"}
)

type BoardError struct {
	Message string
}

func (e *BoardError) Error() string {
	return e.Message
}

// Save writes the board to a JSON file
func (b *Board) Save(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, b.ID+".json")
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// LoadBoard reads a board from a JSON file
func LoadBoard(path string) (*Board, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b Board
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if b.Tickets == nil {
		b.Tickets = make(map[TicketID]*Ticket)
	}

	return &b, nil
}
