package fetcher

import (
	"strings"
	"testing"
	"time"

	"github.com/release-checker/release-checker/internal/types"
)

func TestFetcher_NewFetcher(t *testing.T) {
	f := NewFetcher()
	if f == nil {
		t.Fatal("NewFetcher returned nil")
	}
	if f.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", f.MaxAttempts)
	}
	if f.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", f.BaseDelay)
	}
}

func TestCheckApp_Disabled(t *testing.T) {
	f := NewFetcher()
	app := &types.AppConfig{
		Name:    "test",
		Repo:    "https://example.com/repo.git",
		Enabled: false,
	}
	result := f.CheckApp(app, types.AppState{})
	if result.Error == nil {
		t.Error("expected error for disabled app, got nil")
	}
	if !strings.Contains(result.Error.Error(), "disabled") {
		t.Errorf("Error = %q, should contain 'disabled'", result.Error.Error())
	}
	if result.Updated {
		t.Error("Updated should be false")
	}
}

// Note: Additional integration tests that require network access
// are included but will be skipped in short mode
func TestFetchLatestTag_RealGitHub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	f := NewFetcher()
	tag, err := f.FetchLatestTag("https://github.com/git/git.git")
	if err != nil {
		t.Fatalf("FetchLatestTag failed: %v", err)
	}
	if tag == "" {
		t.Error("tag is empty")
	}
	// Should not contain slashes or spaces
	if strings.Contains(tag, "/") || strings.Contains(tag, " ") {
		t.Errorf("tag contains invalid characters: %q", tag)
	}
}
