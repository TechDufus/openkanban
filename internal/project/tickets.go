package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/techdufus/openkanban/internal/board"
)

const ticketsDir = ".openkanban"
const ticketsFile = "tickets.json"

type TicketStore struct {
	ProjectID string                           `json:"project_id"`
	Tickets   map[board.TicketID]*board.Ticket `json:"tickets"`
	UpdatedAt time.Time                        `json:"updated_at"`

	repoPath string
}

func NewTicketStore(projectID, repoPath string) *TicketStore {
	return &TicketStore{
		ProjectID: projectID,
		Tickets:   make(map[board.TicketID]*board.Ticket),
		UpdatedAt: time.Now(),
		repoPath:  repoPath,
	}
}

func LoadTicketStore(project *Project) (*TicketStore, error) {
	store := NewTicketStore(project.ID, project.RepoPath)

	path := store.filePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, err
	}

	if store.Tickets == nil {
		store.Tickets = make(map[board.TicketID]*board.Ticket)
	}
	store.repoPath = project.RepoPath

	return store, nil
}

func (s *TicketStore) filePath() string {
	return filepath.Join(s.repoPath, ticketsDir, ticketsFile)
}

func (s *TicketStore) Save() error {
	dir := filepath.Join(s.repoPath, ticketsDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	s.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	path := s.filePath()
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *TicketStore) Add(ticket *board.Ticket) {
	ticket.ProjectID = s.ProjectID
	s.Tickets[ticket.ID] = ticket
}

func (s *TicketStore) Get(id board.TicketID) (*board.Ticket, error) {
	t, ok := s.Tickets[id]
	if !ok {
		return nil, board.ErrTicketNotFound
	}
	return t, nil
}

func (s *TicketStore) Delete(id board.TicketID) error {
	if _, ok := s.Tickets[id]; !ok {
		return board.ErrTicketNotFound
	}
	delete(s.Tickets, id)
	return nil
}

func (s *TicketStore) Move(id board.TicketID, newStatus board.TicketStatus) error {
	t, ok := s.Tickets[id]
	if !ok {
		return board.ErrTicketNotFound
	}
	t.SetStatus(newStatus)
	return nil
}

func (s *TicketStore) GetByStatus(status board.TicketStatus) []*board.Ticket {
	var result []*board.Ticket
	for _, t := range s.Tickets {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

func (s *TicketStore) All() []*board.Ticket {
	result := make([]*board.Ticket, 0, len(s.Tickets))
	for _, t := range s.Tickets {
		result = append(result, t)
	}
	return result
}

func (s *TicketStore) Count() int {
	return len(s.Tickets)
}

func (s *TicketStore) CountByStatus(status board.TicketStatus) int {
	count := 0
	for _, t := range s.Tickets {
		if t.Status == status {
			count++
		}
	}
	return count
}

// GlobalTicketStore aggregates tickets from all projects
type GlobalTicketStore struct {
	projects     map[string]*Project
	ticketStores map[string]*TicketStore
	allTickets   map[board.TicketID]*board.Ticket
}

func NewGlobalTicketStore() *GlobalTicketStore {
	return &GlobalTicketStore{
		projects:     make(map[string]*Project),
		ticketStores: make(map[string]*TicketStore),
		allTickets:   make(map[board.TicketID]*board.Ticket),
	}
}

func LoadGlobalTicketStore(registry *ProjectRegistry) (*GlobalTicketStore, error) {
	g := NewGlobalTicketStore()

	for _, p := range registry.Projects {
		store, err := LoadTicketStore(p)
		if err != nil {
			continue
		}

		g.projects[p.ID] = p
		g.ticketStores[p.ID] = store

		for id, ticket := range store.Tickets {
			g.allTickets[id] = ticket
		}
	}

	return g, nil
}

func (g *GlobalTicketStore) GetProject(id string) *Project {
	return g.projects[id]
}

func (g *GlobalTicketStore) GetProjectForTicket(ticket *board.Ticket) *Project {
	return g.projects[ticket.ProjectID]
}

func (g *GlobalTicketStore) GetStoreForTicket(ticket *board.Ticket) *TicketStore {
	return g.ticketStores[ticket.ProjectID]
}

func (g *GlobalTicketStore) Get(id board.TicketID) (*board.Ticket, error) {
	t, ok := g.allTickets[id]
	if !ok {
		return nil, board.ErrTicketNotFound
	}
	return t, nil
}

func (g *GlobalTicketStore) Add(ticket *board.Ticket) error {
	store := g.ticketStores[ticket.ProjectID]
	if store == nil {
		return board.ErrTicketNotFound
	}
	store.Add(ticket)
	g.allTickets[ticket.ID] = ticket
	return nil
}

func (g *GlobalTicketStore) Delete(id board.TicketID) error {
	ticket, ok := g.allTickets[id]
	if !ok {
		return board.ErrTicketNotFound
	}

	store := g.ticketStores[ticket.ProjectID]
	if store != nil {
		store.Delete(id)
	}
	delete(g.allTickets, id)
	return nil
}

func (g *GlobalTicketStore) Move(id board.TicketID, newStatus board.TicketStatus) error {
	ticket, ok := g.allTickets[id]
	if !ok {
		return board.ErrTicketNotFound
	}

	store := g.ticketStores[ticket.ProjectID]
	if store != nil {
		store.Move(id, newStatus)
	}
	return nil
}

func (g *GlobalTicketStore) Save(ticket *board.Ticket) error {
	store := g.ticketStores[ticket.ProjectID]
	if store == nil {
		return board.ErrTicketNotFound
	}
	return store.Save()
}

func (g *GlobalTicketStore) SaveAll() error {
	for _, store := range g.ticketStores {
		if err := store.Save(); err != nil {
			return err
		}
	}
	return nil
}

func (g *GlobalTicketStore) GetByStatus(status board.TicketStatus) []*board.Ticket {
	var result []*board.Ticket
	for _, t := range g.allTickets {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

func (g *GlobalTicketStore) All() []*board.Ticket {
	result := make([]*board.Ticket, 0, len(g.allTickets))
	for _, t := range g.allTickets {
		result = append(result, t)
	}
	return result
}

func (g *GlobalTicketStore) Count() int {
	return len(g.allTickets)
}

func (g *GlobalTicketStore) Projects() []*Project {
	result := make([]*Project, 0, len(g.projects))
	for _, p := range g.projects {
		result = append(result, p)
	}
	return result
}

func (g *GlobalTicketStore) HasProjects() bool {
	return len(g.projects) > 0
}

func (g *GlobalTicketStore) AddProject(p *Project) {
	g.projects[p.ID] = p
	g.ticketStores[p.ID] = NewTicketStore(p.ID, p.RepoPath)
}
