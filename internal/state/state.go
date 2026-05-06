package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jamowei/release-checker/internal/types"
)

// Manager handles state persistence for the release checker
type Manager struct {
	mu    sync.RWMutex
	path  string
	state *types.State
	dirty bool
}

// NewManager creates a new state manager for the given file path
func NewManager(path string) *Manager {
	return &Manager{
		path:  path,
		state: types.NewState(),
	}
}

// Load reads the state file from disk, creating a new one if missing
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If file doesn't exist, start fresh
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		m.state = types.NewState()
		return nil
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var st types.State
	if err := json.Unmarshal(data, &st); err != nil {
		// Corrupted state file? Backup and start fresh
		backupPath := m.path + ".bak." + time.Now().Format("20060102-150405")
		_ = os.Rename(m.path, backupPath)
		m.state = types.NewState()
		return fmt.Errorf("state file corrupted, backed up to %s: %w", backupPath, err)
	}

	m.state = &st
	return nil
}

// Save writes the state to disk atomically
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.dirty {
		return nil
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first
	tmpPath := m.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomically replace the original
	if err := os.Rename(tmpPath, m.path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	m.dirty = false
	return nil
}

// GetAppState returns the stored state for an app (or empty if not seen before)
func (m *Manager) GetAppState(name string) types.AppState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st, ok := m.state.Apps[name]
	if !ok {
		return types.AppState{}
	}
	return st
}

// UpdateApp updates an app's state after a version change
func (m *Manager) UpdateApp(name, newVersion string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	st, ok := m.state.Apps[name]
	if !ok {
		st = types.AppState{}
	}

	now := time.Now().UTC()

	st.CurrentVersion = newVersion
	st.LastUpdated = now
	st.UpdateCount++

	m.state.Apps[name] = st
	m.state.LastCheck = now
	m.dirty = true

	// Also update global last check time
	if m.state.LastCheck.IsZero() || now.After(m.state.LastCheck) {
		m.state.LastCheck = now
	}
}

// MarkChecked marks an app as checked (even if no update)
func (m *Manager) MarkChecked(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if m.state.LastCheck.IsZero() || now.After(m.state.LastCheck) {
		m.state.LastCheck = now
	}
	m.dirty = true
}

// GetAllApps returns all configured apps' states
func (m *Manager) GetAllApps() map[string]types.AppState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]types.AppState)
	for k, v := range m.state.Apps {
		result[k] = v
	}
	return result
}

// GetLastCheck returns when the last check was performed
func (m *Manager) GetLastCheck() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.LastCheck
}

// SetStatePath allows changing the state file path (for initialization)
func (m *Manager) SetStatePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.path = path
}
