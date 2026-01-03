package agent

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
)

type opencodeSession struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Updated   int64  `json:"updated"`
}

func FindOpencodeSession(directory string) string {
	cmd := exec.Command("opencode", "session", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	var sessions []opencodeSession
	if err := json.Unmarshal(output, &sessions); err != nil {
		return ""
	}

	normalizedDir := normalizePath(directory)

	var matches []opencodeSession
	for _, s := range sessions {
		if normalizePath(s.Directory) == normalizedDir {
			matches = append(matches, s)
		}
	}

	if len(matches) == 0 {
		return ""
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Updated > matches[j].Updated
	})

	return matches[0].ID
}

func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		return resolved
	}
	return cleaned
}

// Manager handles AI agent configuration and status polling.
// Agent lifecycle (spawn/stop) is now managed by terminal.Pane via PTY.
type Manager struct {
	config *config.Config
}

// NewManager creates a new agent manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{config: cfg}
}

// GetAgentConfig returns the configuration for a specific agent type
func (m *Manager) GetAgentConfig(agentType string) (*config.AgentConfig, bool) {
	cfg, ok := m.config.Agents[agentType]
	return &cfg, ok
}

// PollStatuses is a no-op placeholder.
// Agent status is now tracked by the UI via terminal.Pane.Running().
func (m *Manager) PollStatuses(tickets map[board.TicketID]*board.Ticket) {
	// Status is now managed by terminal panes, not tmux sessions.
	// This method is kept for interface compatibility but does nothing.
}

func (m *Manager) StatusPollInterval() time.Duration {
	interval := m.config.Opencode.PollInterval
	if interval <= 0 {
		interval = 1
	}
	return time.Duration(interval) * time.Second
}
