package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/techdufus/openkanban/internal/board"
)

// StatusDetector polls status files and analyzes terminal content to determine
// whether an AI agent is actively working, idle, or waiting for user input.
type StatusDetector struct {
	statusCache     map[string]cachedStatus
	statusCacheMu   sync.RWMutex
	cacheExpiration time.Duration
	statusDirs      []string
}

type cachedStatus struct {
	status    board.AgentStatus
	timestamp time.Time
}

// NewStatusDetector creates a StatusDetector configured to read from standard
// status file locations (~/.cache/claude-status, ~/.cache/openkanban-status).
func NewStatusDetector() *StatusDetector {
	homeDir, _ := os.UserHomeDir()

	return &StatusDetector{
		statusCache:     make(map[string]cachedStatus),
		cacheExpiration: 500 * time.Millisecond,
		statusDirs: []string{
			filepath.Join(homeDir, ".cache", "claude-status"),
			filepath.Join(homeDir, ".cache", "openkanban-status"),
		},
	}
}

// DetectStatus returns the current agent status using:
// 1. Status files written by agent hooks (most reliable)
// 2. Terminal content heuristics (fallback)
func (d *StatusDetector) DetectStatus(sessionName string, terminalContent string, processRunning bool) board.AgentStatus {
	if !processRunning {
		return board.AgentNone
	}

	if status := d.readStatusFile(sessionName); status != board.AgentNone {
		return status
	}

	return d.analyzeTerminalContent(terminalContent)
}

func (d *StatusDetector) readStatusFile(sessionName string) board.AgentStatus {
	if sessionName == "" {
		return board.AgentNone
	}

	d.statusCacheMu.RLock()
	cached, exists := d.statusCache[sessionName]
	d.statusCacheMu.RUnlock()

	if exists && time.Since(cached.timestamp) < d.cacheExpiration {
		return cached.status
	}

	var status board.AgentStatus = board.AgentNone

	for _, dir := range d.statusDirs {
		statusFile := filepath.Join(dir, sessionName+".status")
		content, err := os.ReadFile(statusFile)
		if err != nil {
			continue
		}

		statusStr := strings.TrimSpace(string(content))
		switch statusStr {
		case "working":
			status = board.AgentWorking
		case "done", "idle":
			status = board.AgentIdle
		case "waiting", "permission":
			status = board.AgentWaiting
		case "error":
			status = board.AgentError
		case "completed":
			status = board.AgentCompleted
		}

		if status != board.AgentNone {
			break
		}
	}

	d.statusCacheMu.Lock()
	d.statusCache[sessionName] = cachedStatus{
		status:    status,
		timestamp: time.Now(),
	}
	d.statusCacheMu.Unlock()

	return status
}

func (d *StatusDetector) analyzeTerminalContent(content string) board.AgentStatus {
	if content == "" {
		return board.AgentIdle
	}

	lines := strings.Split(content, "\n")
	recentContent := content
	if len(lines) > 10 {
		recentContent = strings.Join(lines[len(lines)-10:], "\n")
	}

	workingIndicators := []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		"◐", "◓", "◑", "◒",
		"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█",
		"...",
		"Thinking", "Writing", "Reading", "Analyzing", "Processing",
		"Working", "Loading", "Searching", "Generating",
		"Executing", "Running",
	}

	for _, indicator := range workingIndicators {
		if strings.Contains(recentContent, indicator) {
			return board.AgentWorking
		}
	}

	waitingIndicators := []string{
		"[Y/n]", "[y/N]", "(y/n)",
		"Allow?", "Approve?", "Confirm?",
		"Press", "Enter to",
		"permission",
	}

	for _, indicator := range waitingIndicators {
		if strings.ContainsAny(recentContent, indicator) || strings.Contains(strings.ToLower(recentContent), strings.ToLower(indicator)) {
			return board.AgentWaiting
		}
	}

	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			lastLine = trimmed
			break
		}
	}

	idlePrompts := []string{
		"> ", "$ ", "❯ ", "→ ", ">> ", "% ",
		"claude>", "opencode>", "aider>",
		"What would you like",
		"How can I help",
		"Enter your",
	}

	for _, prompt := range idlePrompts {
		if strings.HasSuffix(lastLine, prompt) || strings.Contains(strings.ToLower(lastLine), strings.ToLower(prompt)) {
			return board.AgentIdle
		}
	}

	return board.AgentWorking
}

// InvalidateCache clears cached status for a session, or all sessions if empty.
func (d *StatusDetector) InvalidateCache(sessionName string) {
	d.statusCacheMu.Lock()
	defer d.statusCacheMu.Unlock()

	if sessionName == "" {
		d.statusCache = make(map[string]cachedStatus)
	} else {
		delete(d.statusCache, sessionName)
	}
}

// WriteStatusFile persists agent status to disk for external monitoring.
func WriteStatusFile(sessionName string, status board.AgentStatus) error {
	homeDir, _ := os.UserHomeDir()
	statusDir := filepath.Join(homeDir, ".cache", "openkanban-status")

	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return err
	}

	statusFile := filepath.Join(statusDir, sessionName+".status")
	var statusStr string

	switch status {
	case board.AgentWorking:
		statusStr = "working"
	case board.AgentIdle:
		statusStr = "idle"
	case board.AgentWaiting:
		statusStr = "waiting"
	case board.AgentCompleted:
		statusStr = "completed"
	case board.AgentError:
		statusStr = "error"
	default:
		statusStr = "idle"
	}

	return os.WriteFile(statusFile, []byte(statusStr+"\n"), 0644)
}

// CleanupStatusFile removes status files for a session from all known directories.
func CleanupStatusFile(sessionName string) error {
	homeDir, _ := os.UserHomeDir()

	statusDirs := []string{
		filepath.Join(homeDir, ".cache", "claude-status"),
		filepath.Join(homeDir, ".cache", "openkanban-status"),
	}

	for _, dir := range statusDirs {
		statusFile := filepath.Join(dir, sessionName+".status")
		os.Remove(statusFile)
	}

	return nil
}
