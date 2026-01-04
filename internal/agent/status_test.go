package agent

import (
	"testing"
	"time"

	"github.com/techdufus/openkanban/internal/board"
)

func TestMapOpencodeStatus(t *testing.T) {
	d := NewStatusDetector()

	tests := []struct {
		name     string
		input    opencodeSessionStatus
		expected board.AgentStatus
	}{
		{
			name:     "busy maps to working",
			input:    opencodeSessionStatus{Type: "busy"},
			expected: board.AgentWorking,
		},
		{
			name:     "idle maps to idle",
			input:    opencodeSessionStatus{Type: "idle"},
			expected: board.AgentIdle,
		},
		{
			name:     "retry maps to error",
			input:    opencodeSessionStatus{Type: "retry"},
			expected: board.AgentError,
		},
		{
			name:     "unknown type maps to none",
			input:    opencodeSessionStatus{Type: "unknown"},
			expected: board.AgentNone,
		},
		{
			name:     "empty type maps to none",
			input:    opencodeSessionStatus{Type: ""},
			expected: board.AgentNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.mapOpencodeStatus(tt.input)
			if result != tt.expected {
				t.Errorf("mapOpencodeStatus(%+v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectCodingAgentStatus(t *testing.T) {
	d := NewStatusDetector()

	tests := []struct {
		name        string
		recentLower string
		fullLower   string
		expected    board.AgentStatus
	}{
		{
			name:        "waiting for user input",
			recentLower: "waiting for your response",
			fullLower:   "",
			expected:    board.AgentWaiting,
		},
		{
			name:        "yes/no prompt",
			recentLower: "do you want to proceed? [y/n]",
			fullLower:   "",
			expected:    board.AgentWaiting,
		},
		{
			name:        "permission request",
			recentLower: "approve this change?",
			fullLower:   "",
			expected:    board.AgentWaiting,
		},
		{
			name:        "thinking indicator",
			recentLower: "thinking about the problem...",
			fullLower:   "",
			expected:    board.AgentWorking,
		},
		{
			name:        "processing",
			recentLower: "processing your request",
			fullLower:   "",
			expected:    board.AgentWorking,
		},
		{
			name:        "progress bar",
			recentLower: "downloading ━━━━━━━━",
			fullLower:   "",
			expected:    board.AgentWorking,
		},
		{
			name:        "error message",
			recentLower: "error: failed to compile",
			fullLower:   "",
			expected:    board.AgentError,
		},
		{
			name:        "rate limit error",
			recentLower: "rate limit exceeded, please wait",
			fullLower:   "",
			expected:    board.AgentError,
		},
		{
			name:        "idle at prompt",
			recentLower: "ready for input >",
			fullLower:   "",
			expected:    board.AgentNone,
		},
		{
			name:        "no clear status",
			recentLower: "some random output",
			fullLower:   "",
			expected:    board.AgentNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.detectCodingAgentStatus(tt.recentLower, tt.fullLower)
			if result != tt.expected {
				t.Errorf("detectCodingAgentStatus(%q, %q) = %q; want %q",
					tt.recentLower, tt.fullLower, result, tt.expected)
			}
		})
	}
}

func TestDetectGenericAgentStatus(t *testing.T) {
	d := NewStatusDetector()

	tests := []struct {
		name        string
		recentLower string
		expected    board.AgentStatus
	}{
		{
			name:        "error in output",
			recentLower: "something went wrong, error occurred",
			expected:    board.AgentError,
		},
		{
			name:        "failed message",
			recentLower: "operation failed",
			expected:    board.AgentError,
		},
		{
			name:        "processing dots",
			recentLower: "loading...",
			expected:    board.AgentWorking,
		},
		{
			name:        "processing keyword",
			recentLower: "processing data",
			expected:    board.AgentWorking,
		},
		{
			name:        "normal output",
			recentLower: "completed successfully",
			expected:    board.AgentNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.detectGenericAgentStatus(tt.recentLower)
			if result != tt.expected {
				t.Errorf("detectGenericAgentStatus(%q) = %q; want %q",
					tt.recentLower, result, tt.expected)
			}
		})
	}
}

func TestDetectStatusWithPort_NotRunning(t *testing.T) {
	d := NewStatusDetector()

	result := d.DetectStatusWithPort("opencode", "session-1", "/path", 4097, false, "")
	if result != board.AgentNone {
		t.Errorf("DetectStatusWithPort with processRunning=false should return AgentNone; got %q", result)
	}
}

func TestDetectStatusWithPort_UnknownStatus(t *testing.T) {
	d := NewStatusDetector()

	result := d.DetectStatusWithPort("opencode", "nonexistent-session", "/nonexistent/path", 0, true, "some random output with no patterns")
	if result != board.AgentNone {
		t.Errorf("DetectStatusWithPort with undetermined status should return AgentNone; got %q", result)
	}
}

func TestStatusDetectorCaching(t *testing.T) {
	d := NewStatusDetector()

	d.statusCacheMu.Lock()
	d.statusCache["file:test-session"] = cachedStatus{
		status:    board.AgentWorking,
		timestamp: time.Now(),
	}
	d.statusCacheMu.Unlock()

	result := d.readStatusFile("test-session")
	if result != board.AgentWorking {
		t.Errorf("readStatusFile should return cached status; got %q, want %q", result, board.AgentWorking)
	}
}
