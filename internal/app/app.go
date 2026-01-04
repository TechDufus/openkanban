package app

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/techdufus/openkanban/internal/agent"
	"github.com/techdufus/openkanban/internal/config"
	"github.com/techdufus/openkanban/internal/git"
	"github.com/techdufus/openkanban/internal/project"
	"github.com/techdufus/openkanban/internal/ui"
	"github.com/techdufus/openkanban/internal/update"
)

func Run(cfg *config.Config, filterPath, version string) error {
	registry, err := project.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load project registry: %w", err)
	}

	globalStore, err := project.LoadGlobalTicketStore(registry)
	if err != nil {
		return fmt.Errorf("failed to load tickets: %w", err)
	}

	if !globalStore.HasProjects() {
		return fmt.Errorf("no projects registered. Create one with: openkanban new")
	}

	var filterProjectID string
	if filterPath != "" {
		absPath, _ := filepath.Abs(filterPath)
		absPath = git.ResolveMainRepo(absPath)
		if p, err := registry.FindByPath(absPath); err == nil {
			filterProjectID = p.ID
		}
	}

	agentMgr := agent.NewManager(cfg)

	opencodeServer := agent.NewOpencodeServer(cfg)
	if err := opencodeServer.Start(); err != nil {
		return fmt.Errorf("failed to start opencode server: %w", err)
	}
	defer opencodeServer.Stop()

	updateChecker := update.NewChecker(version)
	model := ui.NewModel(cfg, globalStore, registry, agentMgr, opencodeServer, filterProjectID, updateChecker)

	defer model.Cleanup()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion())

	go func() {
		<-sigChan
		model.Cleanup()
		program.Quit()
	}()

	_, err = program.Run()
	return err
}

func CreateProject(cfg *config.Config, name, repoPath string) error {
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	registry, err := project.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load project registry: %w", err)
	}

	if existing, _ := registry.FindByPath(repoPath); existing != nil {
		return fmt.Errorf("project already exists for %s: %s", repoPath, existing.Name)
	}

	p := project.NewProject(name, repoPath)

	if cfg.Defaults.DefaultAgent != "" {
		p.Settings.DefaultAgent = cfg.Defaults.DefaultAgent
	}
	if cfg.Defaults.BranchPrefix != "" {
		p.Settings.BranchPrefix = cfg.Defaults.BranchPrefix
	}

	if err := registry.Add(p); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	fmt.Printf("Created project '%s' for %s\n", name, repoPath)
	fmt.Printf("Project ID: %s\n", p.ID)
	return nil
}

func ListProjects() error {
	registry, err := project.LoadRegistry()
	if err != nil {
		return err
	}

	projects := registry.List()
	if len(projects) == 0 {
		fmt.Println("No projects found. Create one with: openkanban new")
		return nil
	}

	fmt.Println("Available projects:")
	fmt.Println()

	for _, p := range projects {
		tickets, err := project.LoadTicketStore(p)
		if err != nil {
			continue
		}

		total := tickets.Count()
		inProgress := tickets.CountByStatus("in_progress")

		fmt.Printf("  %s (%s)\n", p.Name, p.ID[:8])
		fmt.Printf("    Path: %s\n", p.RepoPath)
		fmt.Printf("    Tickets: %d total, %d in progress\n", total, inProgress)
		fmt.Println()
	}

	return nil
}

func DeleteProject(nameOrID string) error {
	registry, err := project.LoadRegistry()
	if err != nil {
		return err
	}

	var target *project.Project
	for _, p := range registry.List() {
		if p.Name == nameOrID || p.ID == nameOrID || (len(p.ID) >= 8 && p.ID[:8] == nameOrID) {
			target = p
			break
		}
	}

	if target == nil {
		return fmt.Errorf("project not found: %s", nameOrID)
	}

	if err := registry.Delete(target.ID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	fmt.Printf("Deleted project '%s' (%s)\n", target.Name, target.RepoPath)
	return nil
}
