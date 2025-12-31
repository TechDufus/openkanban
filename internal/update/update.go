package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	githubRepo = "TechDufus/openkanban"
	apiTimeout = 5 * time.Second
)

type InstallMethod int

const (
	InstallUnknown InstallMethod = iota
	InstallHomebrew
	InstallGo
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckResult contains the result of an update check
type CheckResult struct {
	UpdateAvailable bool
	LatestVersion   string
	CurrentVersion  string
	ReleaseURL      string
	InstallMethod   InstallMethod
	Error           error
}

// UpdateHint returns a user-friendly update command based on install method
func (r CheckResult) UpdateHint() string {
	switch r.InstallMethod {
	case InstallHomebrew:
		return "brew upgrade openkanban"
	case InstallGo:
		return "go install github.com/techdufus/openkanban@latest"
	default:
		return r.ReleaseURL
	}
}

// DetectInstallMethod determines how openkanban was installed
func DetectInstallMethod() InstallMethod {
	exe, err := os.Executable()
	if err != nil {
		return InstallUnknown
	}

	if strings.Contains(exe, "Cellar") || strings.Contains(exe, "linuxbrew") {
		return InstallHomebrew
	}

	if strings.Contains(exe, "/go/bin") {
		return InstallGo
	}

	return InstallUnknown
}

// Checker provides update checking functionality
type Checker struct {
	CurrentVersion string
}

// NewChecker creates a new update checker for the given version
func NewChecker(currentVersion string) *Checker {
	return &Checker{CurrentVersion: currentVersion}
}

// Check compares the current version against the latest GitHub release.
// Returns immediately if current version is "dev" (development build).
func (c *Checker) Check() CheckResult {
	return Check(c.CurrentVersion)
}

// Check compares the current version against the latest GitHub release.
func Check(currentVersion string) CheckResult {
	result := CheckResult{
		CurrentVersion: currentVersion,
	}

	if currentVersion == "dev" || currentVersion == "" {
		return result
	}

	client := &http.Client{Timeout: apiTimeout}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	resp, err := client.Get(url)
	if err != nil {
		result.Error = fmt.Errorf("failed to check for updates: %w", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		return result
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		result.Error = fmt.Errorf("failed to parse release info: %w", err)
		return result
	}

	result.LatestVersion = release.TagName
	result.ReleaseURL = release.HTMLURL
	result.InstallMethod = DetectInstallMethod()

	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(release.TagName, "v")

	if latest != current && latest != "" {
		result.UpdateAvailable = true
	}

	return result
}
