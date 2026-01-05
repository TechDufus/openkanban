package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
	"github.com/techdufus/openkanban/internal/project"
)

type TestEnv struct {
	ConfigDir string
	RepoDir   string
	T         *testing.T
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "config")
	repoDir := filepath.Join(baseDir, "repo")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	t.Setenv("OPENKANBAN_CONFIG_DIR", configDir)

	return &TestEnv{
		ConfigDir: configDir,
		RepoDir:   repoDir,
		T:         t,
	}
}

func (e *TestEnv) InitGitRepo() {
	e.T.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = e.RepoDir
	if err := cmd.Run(); err != nil {
		e.T.Fatalf("failed to init git repo: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = e.RepoDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = e.RepoDir
	cmd.Run()
}

func (e *TestEnv) WriteConfig(cfg *config.Config) {
	e.T.Helper()
	path := filepath.Join(e.ConfigDir, "config.json")
	if err := cfg.Save(path); err != nil {
		e.T.Fatalf("failed to write test config: %v", err)
	}
}

func (e *TestEnv) LoadRegistry() *project.ProjectRegistry {
	e.T.Helper()
	reg, err := project.LoadRegistry()
	if err != nil {
		e.T.Fatalf("failed to load registry: %v", err)
	}
	return reg
}

func (e *TestEnv) LoadTickets() *project.TicketStore {
	e.T.Helper()
	p := &project.Project{ID: "test", RepoPath: e.RepoDir}
	store, err := project.LoadTicketStore(p)
	if err != nil {
		e.T.Fatalf("failed to load tickets: %v", err)
	}
	return store
}

func (e *TestEnv) CreateProject(name string) *project.Project {
	e.T.Helper()
	reg := e.LoadRegistry()

	p := project.NewProject(name, e.RepoDir)
	if err := reg.Add(p); err != nil {
		e.T.Fatalf("failed to add project: %v", err)
	}
	return p
}

func (e *TestEnv) AssertProjectCount(expected int) {
	e.T.Helper()
	reg := e.LoadRegistry()
	if len(reg.Projects) != expected {
		e.T.Errorf("expected %d projects, got %d", expected, len(reg.Projects))
	}
}

func (e *TestEnv) AssertTicketCount(expected int) {
	e.T.Helper()
	store := e.LoadTickets()
	if store.Count() != expected {
		e.T.Errorf("expected %d tickets, got %d", expected, store.Count())
	}
}

func (e *TestEnv) AssertTicketExists(title string) *board.Ticket {
	e.T.Helper()
	store := e.LoadTickets()
	for _, t := range store.All() {
		if t.Title == title {
			return t
		}
	}
	e.T.Errorf("expected ticket with title %q to exist", title)
	return nil
}

func (e *TestEnv) AssertTicketStatus(ticketID board.TicketID, expected board.TicketStatus) {
	e.T.Helper()
	store := e.LoadTickets()
	ticket, err := store.Get(ticketID)
	if err != nil {
		e.T.Fatalf("ticket %s not found: %v", ticketID, err)
	}
	if ticket.Status != expected {
		e.T.Errorf("ticket %s status = %s; want %s", ticketID, ticket.Status, expected)
	}
}

func (e *TestEnv) RunCLI(args ...string) ([]byte, error) {
	e.T.Helper()
	cmd := exec.Command("go", append([]string{"run", "../../main.go"}, args...)...)
	cmd.Env = append(os.Environ(), "OPENKANBAN_CONFIG_DIR="+e.ConfigDir)
	return cmd.CombinedOutput()
}
