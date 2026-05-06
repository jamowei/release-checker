package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jamowei/release-checker/internal/types"
)

// Config paths to search (in order of precedence)
var configPaths = []string{
	"$HOME/.config/release-checker/config.yaml",
	"./config.yaml",
}

// Load searches for config file at the given path and falls back to defaults
func Load(path string) (*types.Config, error) {
	// If explicit path provided, use it
	if path != "" {
		cfg, err := readConfig(path)
		if err != nil {
			// If file doesn't exist, fall back to default (treat as "not found")
			if os.IsNotExist(err) {
				return &types.Config{
					StateFile: "",
					Apps:      []types.AppConfig{},
				}, nil
			}
			return nil, err
		}
		return cfg, nil
	}

	// Check environment variable
	if envPath := os.Getenv("RELEASE_CHECKER_CONFIG"); envPath != "" {
		cfg, err := readConfig(envPath)
		if err != nil {
			if os.IsNotExist(err) {
				return &types.Config{
					StateFile: "",
					Apps:      []types.AppConfig{},
				}, nil
			}
			return nil, err
		}
		return cfg, nil
	}

	// Search standard paths
	for _, p := range configPaths {
		expanded := expandPath(p)
		if _, err := os.Stat(expanded); err == nil {
			return readConfig(expanded)
		}
	}

	// No config file found - return default config
	return &types.Config{
		StateFile: "",
		Apps:      []types.AppConfig{},
	}, nil
}

// readConfig parses a YAML config file
func readConfig(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return unwrapped os.ErrNotExist so callers can check with os.IsNotExist
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Normalize repo URLs for all apps
	for i := range cfg.Apps {
		cfg.Apps[i].Repo = NormalizeRepo(cfg.Apps[i].Repo)
	}

	return &cfg, nil
}

// NormalizeRepo converts various repo formats into a full git HTTPS URL
// Supported formats:
//  1. Full URL (http:// or https://) → used as-is
//  2. GitHub shorthand: "owner/repo" → "https://github.com/owner/repo.git"
//  3. Host/path: "host.name/path" → "https://host.name/path" (+ .git if needed)
func NormalizeRepo(repo string) string {
	repo = strings.TrimSpace(repo)

	// Already a full URL? Use as-is
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		return ensureGitSuffix(repo)
	}

	parts := strings.SplitN(repo, "/", 2)

	// Need at least owner/repo (two parts)
	if len(parts) < 2 {
		// Invalid format, return as-is (will fail later with git error)
		return repo
	}

	firstPart := parts[0]

	// If first part contains a dot, it's likely a hostname (e.g., "codeberg.org", "git.kernel.org")
	// OR if it contains a colon (port), treat as host/path
	if strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") {
		// Host/path format - assume HTTPS
		return ensureGitSuffix("https://" + repo)
	}

	// Otherwise treat as GitHub shorthand: owner/repo
	return "https://github.com/" + repo + ".git"
}

// ensureGitSuffix adds ".git" suffix if not present
func ensureGitSuffix(url string) string {
	if !strings.HasSuffix(url, ".git") {
		return url + ".git"
	}
	return url
}

// expandPath expands $HOME and environment variables in a path
func expandPath(path string) string {
	// Handle $HOME/ prefix
	if strings.HasPrefix(path, "$HOME/") {
		home := os.Getenv("HOME")
		if home == "" {
			home = "/"
		}
		rest := strings.TrimPrefix(path[6:], "/")
		return filepath.Join(home, rest)
	}
	// Handle ${HOME}/ prefix
	if strings.HasPrefix(path, "${HOME}/") {
		home := os.Getenv("HOME")
		if home == "" {
			home = "/"
		}
		rest := strings.TrimPrefix(path[7:], "/")
		return filepath.Join(home, rest)
	}
	// Expand other env vars using standard library
	return os.ExpandEnv(path)
}

// GetDefaultStatePath returns the default state file location based on executable path
// If running from /usr/local/bin/release-checker, state goes to /usr/local/bin/release-checker-state.json
func GetDefaultStatePath(executablePath string) string {
	// Try to find the actual executable path
	if exe, err := os.Executable(); err == nil {
		return exe + "-state.json"
	}
	// Fallback to current directory
	return "./release-checker-state.json"
}
