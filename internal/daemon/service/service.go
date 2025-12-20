package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Platform int

const (
	PlatformUnknown Platform = iota
	PlatformLinux
	PlatformMacOS
	PlatformWSL
)

func (p Platform) String() string {
	switch p {
	case PlatformLinux:
		return "linux"
	case PlatformMacOS:
		return "macos"
	case PlatformWSL:
		return "wsl"
	default:
		return "unknown"
	}
}

type Manager interface {
	Install(binPath string) error
	Uninstall() error
	Start() error
	Stop() error
	Status() (string, error)
	IsInstalled() bool
}

func DetectPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS
	case "linux":
		if isWSL() {
			return PlatformWSL
		}
		return PlatformLinux
	default:
		return PlatformUnknown
	}
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	if err != nil {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	return false
}

func NewManager() (Manager, error) {
	platform := DetectPlatform()

	switch platform {
	case PlatformMacOS:
		return &LaunchdManager{}, nil
	case PlatformLinux:
		if hasSystemd() {
			return &SystemdManager{}, nil
		}
		return nil, fmt.Errorf("systemd not available on this Linux system")
	case PlatformWSL:
		if hasSystemd() {
			return &SystemdManager{}, nil
		}
		return &WSLManager{}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func GetBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func userConfigDir() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library")
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}
