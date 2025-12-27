package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/techdufus/openkanban/internal/board"
)

func TestNewTicketStore(t *testing.T) {
	store := NewTicketStore("project-1", "/path/to/repo")

	if store.ProjectID != "project-1" {
		t.Errorf("ProjectID = %q; want %q", store.ProjectID, "project-1")
	}

	if store.Tickets == nil {
		t.Error("Tickets map should not be nil")
	}

	if len(store.Tickets) != 0 {
		t.Errorf("new store should have 0 tickets; got %d", len(store.Tickets))
	}
}

func TestTicketStore_AddAndGet(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	ticket := board.NewTicket("Test Ticket", "project-1")
	store.Add(ticket)

	retrieved, err := store.Get(ticket.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if retrieved.Title != ticket.Title {
		t.Errorf("retrieved.Title = %q; want %q", retrieved.Title, ticket.Title)
	}

	if retrieved.ProjectID != "project-1" {
		t.Errorf("Add should set ProjectID; got %q", retrieved.ProjectID)
	}
}

func TestTicketStore_GetNotFound(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	_, err := store.Get("nonexistent-id")
	if err != board.ErrTicketNotFound {
		t.Errorf("Get() error = %v; want ErrTicketNotFound", err)
	}
}

func TestTicketStore_Delete(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	ticket := board.NewTicket("Test", "project-1")
	store.Add(ticket)

	if err := store.Delete(ticket.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := store.Get(ticket.ID)
	if err != board.ErrTicketNotFound {
		t.Error("ticket should not exist after delete")
	}
}

func TestTicketStore_DeleteNotFound(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	err := store.Delete("nonexistent")
	if err != board.ErrTicketNotFound {
		t.Errorf("Delete() error = %v; want ErrTicketNotFound", err)
	}
}

func TestTicketStore_Move(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	ticket := board.NewTicket("Test", "project-1")
	store.Add(ticket)

	if err := store.Move(ticket.ID, board.StatusInProgress); err != nil {
		t.Fatalf("Move() error: %v", err)
	}

	retrieved, _ := store.Get(ticket.ID)
	if retrieved.Status != board.StatusInProgress {
		t.Errorf("Status = %q; want %q", retrieved.Status, board.StatusInProgress)
	}

	if retrieved.StartedAt == nil {
		t.Error("StartedAt should be set after moving to in_progress")
	}
}

func TestTicketStore_GetByStatus(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	t1 := board.NewTicket("Backlog 1", "project-1")
	t2 := board.NewTicket("Backlog 2", "project-1")
	t3 := board.NewTicket("In Progress", "project-1")
	t3.Status = board.StatusInProgress

	store.Add(t1)
	store.Add(t2)
	store.Add(t3)

	backlog := store.GetByStatus(board.StatusBacklog)
	if len(backlog) != 2 {
		t.Errorf("GetByStatus(backlog) returned %d tickets; want 2", len(backlog))
	}

	inProgress := store.GetByStatus(board.StatusInProgress)
	if len(inProgress) != 1 {
		t.Errorf("GetByStatus(in_progress) returned %d tickets; want 1", len(inProgress))
	}

	done := store.GetByStatus(board.StatusDone)
	if len(done) != 0 {
		t.Errorf("GetByStatus(done) returned %d tickets; want 0", len(done))
	}
}

func TestTicketStore_All(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	store.Add(board.NewTicket("T1", "project-1"))
	store.Add(board.NewTicket("T2", "project-1"))
	store.Add(board.NewTicket("T3", "project-1"))

	all := store.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d tickets; want 3", len(all))
	}
}

func TestTicketStore_Count(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	if store.Count() != 0 {
		t.Errorf("Count() = %d; want 0", store.Count())
	}

	store.Add(board.NewTicket("T1", "project-1"))
	store.Add(board.NewTicket("T2", "project-1"))

	if store.Count() != 2 {
		t.Errorf("Count() = %d; want 2", store.Count())
	}
}

func TestTicketStore_CountByStatus(t *testing.T) {
	store := NewTicketStore("project-1", "/path")

	t1 := board.NewTicket("T1", "project-1")
	t2 := board.NewTicket("T2", "project-1")
	t3 := board.NewTicket("T3", "project-1")
	t3.Status = board.StatusDone

	store.Add(t1)
	store.Add(t2)
	store.Add(t3)

	if store.CountByStatus(board.StatusBacklog) != 2 {
		t.Errorf("CountByStatus(backlog) = %d; want 2", store.CountByStatus(board.StatusBacklog))
	}

	if store.CountByStatus(board.StatusDone) != 1 {
		t.Errorf("CountByStatus(done) = %d; want 1", store.CountByStatus(board.StatusDone))
	}
}

func TestTicketStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewTicketStore("project-1", tmpDir)
	ticket := board.NewTicket("Persistent Ticket", "project-1")
	ticket.Description = "This should persist"
	ticket.Status = board.StatusInProgress
	store.Add(ticket)

	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	ticketsPath := filepath.Join(tmpDir, ".openkanban", "tickets.json")
	if _, err := os.Stat(ticketsPath); os.IsNotExist(err) {
		t.Fatalf("tickets file should exist at %s", ticketsPath)
	}

	project := &Project{ID: "project-1", RepoPath: tmpDir}
	loaded, err := LoadTicketStore(project)
	if err != nil {
		t.Fatalf("LoadTicketStore() error: %v", err)
	}

	if loaded.Count() != 1 {
		t.Fatalf("loaded store should have 1 ticket; got %d", loaded.Count())
	}

	loadedTicket, err := loaded.Get(ticket.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if loadedTicket.Title != "Persistent Ticket" {
		t.Errorf("Title = %q; want %q", loadedTicket.Title, "Persistent Ticket")
	}

	if loadedTicket.Description != "This should persist" {
		t.Errorf("Description = %q; want %q", loadedTicket.Description, "This should persist")
	}

	if loadedTicket.Status != board.StatusInProgress {
		t.Errorf("Status = %q; want %q", loadedTicket.Status, board.StatusInProgress)
	}
}

func TestLoadTicketStore_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	project := &Project{ID: "project-1", RepoPath: tmpDir}

	store, err := LoadTicketStore(project)
	if err != nil {
		t.Fatalf("LoadTicketStore() should not error for nonexistent file: %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("loaded store should be empty; got %d tickets", store.Count())
	}
}

func TestTicketStore_AtomicSave(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewTicketStore("project-1", tmpDir)
	store.Add(board.NewTicket("Test", "project-1"))

	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	tmpPath := filepath.Join(tmpDir, ".openkanban", "tickets.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}
