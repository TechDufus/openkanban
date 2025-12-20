package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const launchdLabel = "com.openkanban.daemon"

const launchdTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>HOME</key>
        <string>%s</string>
    </dict>
</dict>
</plist>
`

type LaunchdManager struct{}

func (m *LaunchdManager) launchAgentsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents")
}

func (m *LaunchdManager) plistPath() string {
	return filepath.Join(m.launchAgentsDir(), launchdLabel+".plist")
}

func (m *LaunchdManager) logDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openkanban", "logs")
}

func (m *LaunchdManager) Install(binPath string) error {
	if err := os.MkdirAll(m.launchAgentsDir(), 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	if err := os.MkdirAll(m.logDir(), 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	home, _ := os.UserHomeDir()
	stdoutLog := filepath.Join(m.logDir(), "daemon.log")
	stderrLog := filepath.Join(m.logDir(), "daemon.error.log")

	content := fmt.Sprintf(launchdTemplate, launchdLabel, binPath, stdoutLog, stderrLog, home)

	if err := os.WriteFile(m.plistPath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	return nil
}

func (m *LaunchdManager) Uninstall() error {
	_ = m.Stop()

	if err := os.Remove(m.plistPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}

	return nil
}

func (m *LaunchdManager) Start() error {
	return exec.Command("launchctl", "load", m.plistPath()).Run()
}

func (m *LaunchdManager) Stop() error {
	return exec.Command("launchctl", "unload", m.plistPath()).Run()
}

func (m *LaunchdManager) Status() (string, error) {
	out, err := exec.Command("launchctl", "list", launchdLabel).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Could not find service") {
			return "stopped", nil
		}
		return "not installed", nil
	}
	return "running", nil
}

func (m *LaunchdManager) IsInstalled() bool {
	_, err := os.Stat(m.plistPath())
	return err == nil
}
