package fetcher

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jamowei/release-checker/internal/types"
)

// Fetcher handles fetching the latest git tag from a repository
type Fetcher struct {
	MaxAttempts int
	BaseDelay   time.Duration
}

// NewFetcher creates a new fetcher with default retry settings
func NewFetcher() *Fetcher {
	return &Fetcher{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
	}
}

// FetchLatestTag fetches the highest semantic version tag from a git repository
// The repo URL should already be normalized (full HTTPS URL)
func (f *Fetcher) FetchLatestTag(repo string) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= f.MaxAttempts; attempt++ {
		tag, err := f.doFetch(repo)
		if err == nil {
			return tag, nil
		}
		lastErr = err

		// If this isn't the last attempt, wait before retrying
		if attempt < f.MaxAttempts {
			delay := f.BaseDelay * time.Duration(attempt*attempt) // exponential backoff
			time.Sleep(delay)
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", f.MaxAttempts, lastErr)
}

// doFetch performs a single git ls-remote call
func (f *Fetcher) doFetch(repo string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "--sort=v:refname", repo)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return "", fmt.Errorf("no tags found in repository")
	}

	// Get the last line (highest version due to --sort=v:refname)
	lastLine := lines[len(lines)-1]
	parts := strings.SplitN(lastLine, "\t", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected git output format: %s", lastLine)
	}

	// Extract tag name: refs/tags/v1.2.3 -> v1.2.3 (or 1.2.3)
	tagRef := parts[1]
	const tagPrefix = "refs/tags/"
	if !strings.HasPrefix(tagRef, tagPrefix) {
		return "", fmt.Errorf("unexpected ref format: %s", tagRef)
	}
	tag := strings.TrimPrefix(tagRef, tagPrefix)

	// Strip leading "v" for consistency (optional, configurable in future)
	tag = strings.TrimPrefix(tag, "v")

	return tag, nil
}

// CheckApp checks a single app and returns the result
func (f *Fetcher) CheckApp(app *types.AppConfig, currentState types.AppState) types.CheckResult {
	result := types.CheckResult{
		AppName:    app.Name,
		OldVersion: currentState.CurrentVersion,
	}

	// Skip disabled apps
	if !app.Enabled {
		result.Error = fmt.Errorf("app is disabled")
		return result
	}

	tag, err := f.FetchLatestTag(app.Repo)
	if err != nil {
		result.Error = fmt.Errorf("fetch failed: %w", err)
		return result
	}

	result.NewVersion = tag

	// Compare versions
	if tag != currentState.CurrentVersion {
		result.Updated = true
	}

	return result
}
