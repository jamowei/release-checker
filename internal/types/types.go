package types

import "time"

// AppConfig represents a single application configuration from the YAML file
type AppConfig struct {
	Name     string `yaml:"name"`                // Display name for the app
	Repo     string `yaml:"repo"`                // Repository URL or shorthand
	Enabled  bool   `yaml:"enabled"`             // Whether to check this app
	TagRegex string `yaml:"tag_regex,omitempty"` // Optional regex to filter tags
}

// Config represents the entire configuration file
type Config struct {
	StateFile string      `yaml:"state_file,omitempty"` // Override state file location
	Apps      []AppConfig `yaml:"apps"`                 // List of apps to monitor
}

// AppState represents the stored state for a single app
type AppState struct {
	CurrentVersion string    `json:"current_version"` // Last known version
	LastUpdated    time.Time `json:"last_updated"`    // When version was last updated
	UpdateCount    int       `json:"update_count"`    // Number of times updated
}

// State represents the entire state file
type State struct {
	Version   string              `json:"version"`    // State file format version
	LastCheck time.Time           `json:"last_check"` // When the last check was performed
	Apps      map[string]AppState `json:"apps"`       // Per-app states
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		Version: "1.0",
		Apps:    make(map[string]AppState),
	}
}

// CheckResult holds the result of checking a single app
type CheckResult struct {
	AppName    string
	OldVersion string
	NewVersion string
	Error      error
	Updated    bool
}
