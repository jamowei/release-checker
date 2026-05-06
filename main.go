package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/release-checker/release-checker/internal/config"
	"github.com/release-checker/release-checker/internal/fetcher"
	"github.com/release-checker/release-checker/internal/state"
)

var (
	configPath  string
	statePath   string
	dryRun      bool
	verbose     bool
	showVersion bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "Path to config file (overrides defaults)")
	flag.StringVar(&statePath, "state", "", "Path to state file (overrides config and defaults)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would change without updating state")
	flag.BoolVar(&verbose, "verbose", false, "Show all checks, not just updates")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Println("release-checker version 1.0.0")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Determine state file path
	var stateFilePath string
	if statePath != "" {
		stateFilePath = statePath
	} else if cfg.StateFile != "" {
		stateFilePath = cfg.StateFile
	} else {
		// Use default: same directory as executable
		stateFilePath = config.GetDefaultStatePath("")
	}

	// Ensure state directory exists
	stateDir := filepath.Dir(stateFilePath)
	if stateDir != "." {
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating state directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Initialize state manager
	mgr := state.NewManager(stateFilePath)
	if err := mgr.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	// Initialize fetcher
	f := fetcher.NewFetcher()

	// Track if any errors occurred
	hasErrors := false

	// Process each app
	for _, app := range cfg.Apps {
		if !app.Enabled {
			if verbose {
				fmt.Fprintf(os.Stdout, "[SKIP] %s: disabled\n", app.Name)
			}
			continue
		}

		// Get current state
		currentState := mgr.GetAppState(app.Name)

		// Fetch latest version
		result := f.CheckApp(&app, currentState)

		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] %s: %v\n", app.Name, result.Error)
			hasErrors = true
			continue
		}

		// Version unchanged
		if !result.Updated {
			if verbose {
				fmt.Fprintf(os.Stdout, "[OK] %s: current=%s, latest=%s\n", app.Name, currentState.CurrentVersion, result.NewVersion)
			}
			// Still mark as checked
			if !dryRun {
				mgr.MarkChecked(app.Name)
			}
			continue
		}

		// New version available
		oldVer := currentState.CurrentVersion
		if oldVer == "" {
			oldVer = "(none)"
		}
		fmt.Fprintf(os.Stdout, "%s: %s -> %s\n", app.Name, oldVer, result.NewVersion)

		// Update state (unless dry-run)
		if !dryRun {
			mgr.UpdateApp(app.Name, result.NewVersion)
		}
	}

	// Save state if not dry-run
	if !dryRun {
		if err := mgr.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving state: %v\n", err)
			hasErrors = true
		}
	}

	// Exit with appropriate code
	if hasErrors {
		os.Exit(1)
	}
	os.Exit(0)
}
