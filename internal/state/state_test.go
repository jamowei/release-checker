package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/release-checker/release-checker/internal/types"
)

func TestManager_NewManager(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.path != statePath {
		t.Errorf("mgr.path = %q, want %q", mgr.path, statePath)
	}
}

func TestManager_Load_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	st := mgr.GetAllApps()
	if st == nil {
		t.Error("GetAllApps returned nil")
	}
	if mgr.state.Version != "1.0" {
		t.Errorf("Version = %q, want %q", mgr.state.Version, "1.0")
	}
	if !mgr.state.LastCheck.IsZero() {
		t.Error("LastCheck should be zero initially")
	}
}

func TestManager_Load_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	now := time.Now().UTC()
	existing := types.State{
		Version:   "1.0",
		LastCheck: now,
		Apps: map[string]types.AppState{
			"test-app": {
				CurrentVersion: "v1.2.3",
				LastUpdated:    now.Add(-24 * time.Hour),
				UpdateCount:    1,
			},
		},
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	err = os.WriteFile(statePath, data, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	mgr := NewManager(statePath)
	err = mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	apps := mgr.GetAllApps()
	if len(apps) != 1 {
		t.Fatalf("GetAllApps length = %d, want 1", len(apps))
	}
	appState, ok := apps["test-app"]
	if !ok {
		t.Fatal("test-app not found in state")
	}
	if appState.CurrentVersion != "v1.2.3" {
		t.Errorf("CurrentVersion = %q, want %q", appState.CurrentVersion, "v1.2.3")
	}
	if appState.UpdateCount != 1 {
		t.Errorf("UpdateCount = %d, want 1", appState.UpdateCount)
	}
}

func TestManager_Load_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	err := os.WriteFile(statePath, []byte(`{invalid json`), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	mgr := NewManager(statePath)
	err = mgr.Load()
	if err == nil {
		t.Fatal("expected error for corrupted file, got nil")
	}

	// Should create backup
	backups, err := filepath.Glob(statePath + ".bak.*")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("backup count = %d, want 1", len(backups))
	}

	// State should be reset
	st := mgr.GetAllApps()
	if st == nil {
		t.Error("GetAllApps returned nil")
	}
}

func TestManager_UpdateApp(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	mgr.UpdateApp("myapp", "v1.5.0")

	apps := mgr.GetAllApps()
	if len(apps) != 1 {
		t.Fatalf("apps length = %d, want 1", len(apps))
	}
	appState := apps["myapp"]
	if appState.CurrentVersion != "v1.5.0" {
		t.Errorf("CurrentVersion = %q, want %q", appState.CurrentVersion, "v1.5.0")
	}
	if appState.UpdateCount != 1 {
		t.Errorf("UpdateCount = %d, want 1", appState.UpdateCount)
	}
	if appState.LastUpdated.IsZero() {
		t.Error("LastUpdated is zero")
	}
}

func TestManager_UpdateApp_IncrementsCount(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	mgr.UpdateApp("myapp", "v1.0.0")
	mgr.UpdateApp("myapp", "v1.1.0")
	mgr.UpdateApp("myapp", "v1.2.0")

	apps := mgr.GetAllApps()
	appState := apps["myapp"]
	if appState.UpdateCount != 3 {
		t.Errorf("UpdateCount = %d, want 3", appState.UpdateCount)
	}
	if appState.CurrentVersion != "v1.2.0" {
		t.Errorf("CurrentVersion = %q, want %q", appState.CurrentVersion, "v1.2.0")
	}
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	mgr.UpdateApp("myapp", "v2.0.0")
	err := mgr.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists and contains correct data
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var saved types.State
	err = json.Unmarshal(data, &saved)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	appState, ok := saved.Apps["myapp"]
	if !ok {
		t.Fatal("myapp not found in saved state")
	}
	if appState.CurrentVersion != "v2.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", appState.CurrentVersion, "v2.0.0")
	}
}

func TestManager_Save_Atomic(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	mgr.UpdateApp("myapp", "v3.0.0")

	err := mgr.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Temp file should be cleaned up
	tmpFiles, _ := filepath.Glob(statePath + ".tmp")
	if len(tmpFiles) != 0 {
		t.Errorf("temp file(s) still exist: %v", tmpFiles)
	}
}

func TestManager_MarkChecked(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	mgr := NewManager(statePath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	mgr.MarkChecked("myapp")

	st := mgr.GetAllApps()
	_, exists := st["myapp"]
	if exists {
		t.Error("MarkChecked should not create an app entry")
	}

	lastCheck := mgr.GetLastCheck()
	if lastCheck.IsZero() {
		t.Error("LastCheck should not be zero after MarkChecked")
	}
}
