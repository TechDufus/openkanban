package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const systemdServiceName = "openkanban-daemon"

const systemdTemplate = `[Unit]
Description=OpenKanban Daemon - Persistent PTY sessions for AI agents
After=network.target

[Service]
Type=simple
ExecStart=%s daemon run
Restart=on-failure
RestartSec=5
Environment=HOME=%s
Environment=PATH=%s

[Install]
WantedBy=default.target
`

type SystemdManager struct{}

func (m *SystemdManager) serviceDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

func (m *SystemdManager) servicePath() string {
	return filepath.Join(m.serviceDir(), systemdServiceName+".service")
}

func (m *SystemdManager) Install(binPath string) error {
	if err := os.MkdirAll(m.serviceDir(), 0755); err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}

	home, _ := os.UserHomeDir()
	path := os.Getenv("PATH")
	content := fmt.Sprintf(systemdTemplate, binPath, home, path)

	if err := os.WriteFile(m.servicePath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "enable", systemdServiceName).Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	return nil
}

func (m *SystemdManager) Uninstall() error {
	_ = exec.Command("systemctl", "--user", "stop", systemdServiceName).Run()
	_ = exec.Command("systemctl", "--user", "disable", systemdServiceName).Run()

	if err := os.Remove(m.servicePath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove service file: %w", err)
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func (m *SystemdManager) Start() error {
	return exec.Command("systemctl", "--user", "start", systemdServiceName).Run()
}

func (m *SystemdManager) Stop() error {
	return exec.Command("systemctl", "--user", "stop", systemdServiceName).Run()
}

func (m *SystemdManager) Status() (string, error) {
	out, err := exec.Command("systemctl", "--user", "status", systemdServiceName).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "could not be found") {
			return "not installed", nil
		}
		if strings.Contains(string(out), "inactive") {
			return "stopped", nil
		}
	}
	if strings.Contains(string(out), "active (running)") {
		return "running", nil
	}
	return "unknown", nil
}

func (m *SystemdManager) IsInstalled() bool {
	_, err := os.Stat(m.servicePath())
	return err == nil
}
