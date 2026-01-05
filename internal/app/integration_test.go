//go:build integration

package app_test

import (
	"os"
	"testing"

	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/project"
	"github.com/techdufus/openkanban/internal/testutil"
)

func TestIntegration_ProjectCreation(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.InitGitRepo()

	env.AssertProjectCount(0)

	p := env.CreateProject("test-project")

	env.AssertProjectCount(1)

	reg := env.LoadRegistry()
	loaded, err := reg.Get(p.ID)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if loaded.Name != "test-project" {
		t.Errorf("project name = %q; want %q", loaded.Name, "test-project")
	}

	if loaded.RepoPath != env.RepoDir {
		t.Errorf("project path = %q; want %q", loaded.RepoPath, env.RepoDir)
	}
}

func TestIntegration_ProjectDeletion(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.InitGitRepo()

	p := env.CreateProject("to-delete")
	env.AssertProjectCount(1)

	reg := env.LoadRegistry()
	if err := reg.Delete(p.ID); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	env.AssertProjectCount(0)
}

func TestIntegration_TicketLifecycle(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.InitGitRepo()

	p := env.CreateProject("ticket-test")

	store, err := project.LoadTicketStore(p)
	if err != nil {
		t.Fatalf("failed to load ticket store: %v", err)
	}

	ticket := board.NewTicket("Test Ticket", p.ID)
	ticket.Description = "Test Description"
	store.Add(ticket)

	if err := store.Save(); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	reloaded, err := project.LoadTicketStore(p)
	if err != nil {
		t.Fatalf("failed to reload ticket store: %v", err)
	}

	if reloaded.Count() != 1 {
		t.Errorf("ticket count = %d; want 1", reloaded.Count())
	}

	loaded, err := reloaded.Get(ticket.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}

	if loaded.Title != "Test Ticket" {
		t.Errorf("ticket title = %q; want %q", loaded.Title, "Test Ticket")
	}

	if loaded.Status != board.StatusBacklog {
		t.Errorf("ticket status = %s; want %s", loaded.Status, board.StatusBacklog)
	}
}

func TestIntegration_TicketStatusTransition(t *testing.T) {
	env := testutil.NewTestEnv(t)
	env.InitGitRepo()

	p := env.CreateProject("status-test")

	store, err := project.LoadTicketStore(p)
	if err != nil {
		t.Fatalf("failed to load ticket store: %v", err)
	}

	ticket := board.NewTicket("Status Test", p.ID)
	store.Add(ticket)

	if ticket.Status != board.StatusBacklog {
		t.Errorf("initial status = %s; want %s", ticket.Status, board.StatusBacklog)
	}

	if err := store.Move(ticket.ID, board.StatusInProgress); err != nil {
		t.Fatalf("failed to move ticket: %v", err)
	}

	if ticket.Status != board.StatusInProgress {
		t.Errorf("status after move = %s; want %s", ticket.Status, board.StatusInProgress)
	}

	if err := store.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	reloaded, _ := project.LoadTicketStore(p)
	reloadedTicket, _ := reloaded.Get(ticket.ID)

	if reloadedTicket.Status != board.StatusInProgress {
		t.Errorf("persisted status = %s; want %s", reloadedTicket.Status, board.StatusInProgress)
	}
}

func TestIntegration_MultipleProjects(t *testing.T) {
	env := testutil.NewTestEnv(t)

	baseDir := t.TempDir()
	repo1 := baseDir + "/repo1"
	repo2 := baseDir + "/repo2"

	for _, dir := range []string{repo1, repo2} {
		if err := initGitRepo(dir); err != nil {
			t.Fatalf("failed to init git repo %s: %v", dir, err)
		}
	}

	reg := env.LoadRegistry()

	p1 := project.NewProject("project-1", repo1)
	p2 := project.NewProject("project-2", repo2)

	if err := reg.Add(p1); err != nil {
		t.Fatalf("failed to add project 1: %v", err)
	}
	if err := reg.Add(p2); err != nil {
		t.Fatalf("failed to add project 2: %v", err)
	}

	env.AssertProjectCount(2)

	store1, _ := project.LoadTicketStore(p1)
	store2, _ := project.LoadTicketStore(p2)

	store1.Add(board.NewTicket("Ticket in P1", p1.ID))
	store2.Add(board.NewTicket("Ticket in P2", p2.ID))
	store2.Add(board.NewTicket("Another in P2", p2.ID))

	store1.Save()
	store2.Save()

	reloaded1, _ := project.LoadTicketStore(p1)
	reloaded2, _ := project.LoadTicketStore(p2)

	if reloaded1.Count() != 1 {
		t.Errorf("project 1 ticket count = %d; want 1", reloaded1.Count())
	}
	if reloaded2.Count() != 2 {
		t.Errorf("project 2 ticket count = %d; want 2", reloaded2.Count())
	}
}

func initGitRepo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.MkdirAll(dir+"/.git", 0755)
}
