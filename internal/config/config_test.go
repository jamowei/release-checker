package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeRepo_HTTPSURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://github.com/kubernetes/kubernetes.git", "https://github.com/kubernetes/kubernetes.git"},
		{"https://codeberg.org/ziglang/zig.git", "https://codeberg.org/ziglang/zig.git"},
		{"http://example.com/repo.git", "http://example.com/repo.git"},
	}

	for _, tt := range tests {
		result := NormalizeRepo(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeRepo(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeRepo_GitHubShorthand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"kubernetes/kubernetes", "https://github.com/kubernetes/kubernetes.git"},
		{"hashicorp/terraform", "https://github.com/hashicorp/terraform.git"},
		{"prometheus/prometheus", "https://github.com/prometheus/prometheus.git"},
	}

	for _, tt := range tests {
		result := NormalizeRepo(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeRepo(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeRepo_HostPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux", "https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git"},
		{"codeberg.org/ziglang/zig", "https://codeberg.org/ziglang/zig.git"},
		{"git.example.com/team/project", "https://git.example.com/team/project.git"},
	}

	for _, tt := range tests {
		result := NormalizeRepo(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeRepo(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeRepo_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"owner/repo/extra", "https://github.com/owner/repo/extra.git"},
		{"example.com", "example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		result := NormalizeRepo(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeRepo(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configData := `
state_file: /tmp/state.json
apps:
  - name: test-app
    repo: kubernetes/kubernetes
    enabled: true
`
	err := os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	if cfg.StateFile != "/tmp/state.json" {
		t.Errorf("StateFile = %q, want %q", cfg.StateFile, "/tmp/state.json")
	}
	if len(cfg.Apps) != 1 {
		t.Fatalf("Apps length = %d, want 1", len(cfg.Apps))
	}
	app := cfg.Apps[0]
	if app.Name != "test-app" {
		t.Errorf("App.Name = %q, want %q", app.Name, "test-app")
	}
	expectedRepo := "https://github.com/kubernetes/kubernetes.git"
	if app.Repo != expectedRepo {
		t.Errorf("App.Repo = %q, want %q", app.Repo, expectedRepo)
	}
	if !app.Enabled {
		t.Error("App.Enabled = false, want true")
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`invalid: [}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Error message = %q, should contain 'failed to parse'", err.Error())
	}
}

func TestLoad_MissingConfig(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Logf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg is nil")
	}
	if len(cfg.Apps) != 0 {
		t.Errorf("Apps length = %d, want 0", len(cfg.Apps))
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		env      map[string]string
	}{
		{"$HOME/.config/file", "/home/testuser/.config/file", map[string]string{"HOME": "/home/testuser"}},
		{"/absolute/path", "/absolute/path", nil},
		{"${HOME}/file", "/home/testuser/file", map[string]string{"HOME": "/home/testuser"}},
	}

	for _, tt := range tests {
		// Restore env after test
		if tt.env != nil {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
		}
		result := expandPath(tt.input)
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
