package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/techdufus/openkanban/internal/board"
)

const (
	opencodeDefaultPort = 4096
	opencodeAPITimeout  = 2 * time.Second
)

type opencodeStatusResponse map[string]opencodeSessionStatus

type opencodeSessionStatus struct {
	Type    string `json:"type"`
	Attempt int    `json:"attempt,omitempty"`
	Message string `json:"message,omitempty"`
	Next    int    `json:"next,omitempty"`
}

type StatusDetector struct {
	statusCache     map[string]cachedStatus
	statusCacheMu   sync.RWMutex
	cacheExpiration time.Duration
	statusDirs      []string
	httpClient      *http.Client
}

type cachedStatus struct {
	status    board.AgentStatus
	timestamp time.Time
}

func NewStatusDetector() *StatusDetector {
	homeDir, _ := os.UserHomeDir()

	return &StatusDetector{
		statusCache:     make(map[string]cachedStatus),
		cacheExpiration: 500 * time.Millisecond,
		statusDirs: []string{
			filepath.Join(homeDir, ".cache", "openkanban-status"),
		},
		httpClient: &http.Client{
			Timeout: opencodeAPITimeout,
		},
	}
}

// DetectStatus returns the current agent status.
// For OpenCode: queries the native REST API on port 4096.
// For other agents: checks hook-written status files only.
// No terminal heuristics - they're unreliable.
func (d *StatusDetector) DetectStatus(agentType, sessionID string, processRunning bool) board.AgentStatus {
	if !processRunning {
		return board.AgentNone
	}

	// Check hook-written status files first (works for any agent with configured hooks)
	if status := d.readStatusFile(sessionID); status != board.AgentNone {
		return status
	}

	// For OpenCode, query the native API
	if agentType == "opencode" && sessionID != "" {
		if status := d.queryOpencodeAPI(sessionID); status != board.AgentNone {
			return status
		}
	}

	// For agents without native API support or when API is unavailable,
	// return AgentIdle as a safe default when process is running.
	// Users can configure hooks to write status files for accurate tracking.
	return board.AgentIdle
}

func (d *StatusDetector) queryOpencodeAPI(sessionID string) board.AgentStatus {
	cacheKey := "opencode:" + sessionID

	d.statusCacheMu.RLock()
	cached, exists := d.statusCache[cacheKey]
	d.statusCacheMu.RUnlock()

	if exists && time.Since(cached.timestamp) < d.cacheExpiration {
		return cached.status
	}

	url := fmt.Sprintf("http://localhost:%d/session/status", opencodeDefaultPort)
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return board.AgentNone
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return board.AgentNone
	}

	var statusResp opencodeStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return board.AgentNone
	}

	sessionStatus, found := statusResp[sessionID]
	if !found {
		return board.AgentNone
	}

	status := d.mapOpencodeStatus(sessionStatus)

	d.statusCacheMu.Lock()
	d.statusCache[cacheKey] = cachedStatus{
		status:    status,
		timestamp: time.Now(),
	}
	d.statusCacheMu.Unlock()

	return status
}

func (d *StatusDetector) mapOpencodeStatus(s opencodeSessionStatus) board.AgentStatus {
	switch s.Type {
	case "busy":
		return board.AgentWorking
	case "idle":
		return board.AgentIdle
	case "retry":
		return board.AgentError
	default:
		return board.AgentNone
	}
}

func (d *StatusDetector) readStatusFile(sessionName string) board.AgentStatus {
	if sessionName == "" {
		return board.AgentNone
	}

	cacheKey := "file:" + sessionName

	d.statusCacheMu.RLock()
	cached, exists := d.statusCache[cacheKey]
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
	d.statusCache[cacheKey] = cachedStatus{
		status:    status,
		timestamp: time.Now(),
	}
	d.statusCacheMu.Unlock()

	return status
}

func (d *StatusDetector) InvalidateCache(sessionName string) {
	d.statusCacheMu.Lock()
	defer d.statusCacheMu.Unlock()

	if sessionName == "" {
		d.statusCache = make(map[string]cachedStatus)
	} else {
		delete(d.statusCache, "file:"+sessionName)
		delete(d.statusCache, "opencode:"+sessionName)
	}
}

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

func CleanupStatusFile(sessionName string) error {
	homeDir, _ := os.UserHomeDir()
	statusDir := filepath.Join(homeDir, ".cache", "openkanban-status")
	statusFile := filepath.Join(statusDir, sessionName+".status")
	os.Remove(statusFile)
	return nil
}
