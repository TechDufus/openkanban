package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const wslScriptName = "openkanban-daemon.sh"
const wslProfileMarker = "# openkanban-daemon auto-start"

type WSLManager struct{}

func (m *WSLManager) scriptDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openkanban", "bin")
}

func (m *WSLManager) scriptPath() string {
	return filepath.Join(m.scriptDir(), wslScriptName)
}

func (m *WSLManager) profilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".profile")
}

func (m *WSLManager) pidFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openkanban", "daemon.pid")
}

func (m *WSLManager) Install(binPath string) error {
	if err := os.MkdirAll(m.scriptDir(), 0755); err != nil {
		return fmt.Errorf("create script dir: %w", err)
	}

	script := fmt.Sprintf(`#!/bin/bash
PIDFILE="%s"
DAEMON="%s"

if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE")
    if kill -0 "$PID" 2>/dev/null; then
        exit 0
    fi
fi

nohup "$DAEMON" daemon run > ~/.openkanban/logs/daemon.log 2>&1 &
echo $! > "$PIDFILE"
`, m.pidFile(), binPath)

	if err := os.WriteFile(m.scriptPath(), []byte(script), 0755); err != nil {
		return fmt.Errorf("write startup script: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(m.pidFile()), 0755); err != nil {
		return fmt.Errorf("create pid dir: %w", err)
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".openkanban", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	return m.addToProfile()
}

func (m *WSLManager) addToProfile() error {
	profilePath := m.profilePath()

	content, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read profile: %w", err)
	}

	if strings.Contains(string(content), wslProfileMarker) {
		return nil
	}

	entry := fmt.Sprintf("\n%s\nif [ -x \"%s\" ]; then\n    \"%s\"\nfi\n", wslProfileMarker, m.scriptPath(), m.scriptPath())

	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open profile: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("write to profile: %w", err)
	}

	return nil
}

func (m *WSLManager) removeFromProfile() error {
	content, err := os.ReadFile(m.profilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	skip := false

	for _, line := range lines {
		if strings.Contains(line, wslProfileMarker) {
			skip = true
			continue
		}
		if skip && line == "fi" {
			skip = false
			continue
		}
		if skip {
			continue
		}
		newLines = append(newLines, line)
	}

	return os.WriteFile(m.profilePath(), []byte(strings.Join(newLines, "\n")), 0644)
}

func (m *WSLManager) Uninstall() error {
	_ = m.Stop()
	_ = m.removeFromProfile()

	if err := os.Remove(m.scriptPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove script: %w", err)
	}

	_ = os.Remove(m.pidFile())
	return nil
}

func (m *WSLManager) Start() error {
	return exec.Command("bash", m.scriptPath()).Run()
}

func (m *WSLManager) Stop() error {
	data, err := os.ReadFile(m.pidFile())
	if err != nil {
		return nil
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}

	_ = proc.Signal(syscall.SIGTERM)
	_ = os.Remove(m.pidFile())
	return nil
}

func (m *WSLManager) Status() (string, error) {
	data, err := os.ReadFile(m.pidFile())
	if err != nil {
		return "stopped", nil
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return "stopped", nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return "stopped", nil
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return "stopped", nil
	}

	return "running", nil
}

func (m *WSLManager) IsInstalled() bool {
	_, err := os.Stat(m.scriptPath())
	return err == nil
}
