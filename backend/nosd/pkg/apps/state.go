package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateStore manages persistent app state
type StateStore struct {
	path  string
	mu    sync.RWMutex
	state *AppState
}

// NewStateStore creates a new state store
func NewStateStore(path string) (*StateStore, error) {
	ss := &StateStore{
		path: path,
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Load existing state or initialize new
	if err := ss.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Initialize empty state
		ss.state = &AppState{
			Version:   "1.0",
			Apps:      []InstalledApp{},
			UpdatedAt: time.Now(),
		}
		// Save initial state
		if err := ss.save(); err != nil {
			return nil, err
		}
	}

	return ss, nil
}

// load reads state from disk
func (ss *StateStore) load() error {
	data, err := os.ReadFile(ss.path)
	if err != nil {
		return err
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	ss.state = &state
	return nil
}

// save writes state to disk atomically
func (ss *StateStore) save() error {
	ss.state.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(ss.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file
	tmpPath := ss.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to disk
	tmpFile, err := os.Open(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to open temp file for sync: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	tmpFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, ss.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// GetApp returns an installed app by ID
func (ss *StateStore) GetApp(id string) (*InstalledApp, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for _, app := range ss.state.Apps {
		if app.ID == id {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("app not found: %s", id)
}

// GetAllApps returns all installed apps
func (ss *StateStore) GetAllApps() []InstalledApp {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	apps := make([]InstalledApp, len(ss.state.Apps))
	copy(apps, ss.state.Apps)
	return apps
}

// AddApp adds a new installed app
func (ss *StateStore) AddApp(app InstalledApp) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Check if app already exists
	for _, existing := range ss.state.Apps {
		if existing.ID == app.ID {
			return fmt.Errorf("app already installed: %s", app.ID)
		}
	}

	app.InstalledAt = time.Now()
	app.UpdatedAt = time.Now()
	ss.state.Apps = append(ss.state.Apps, app)

	return ss.save()
}

// UpdateApp updates an existing app
func (ss *StateStore) UpdateApp(app InstalledApp) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	found := false
	for i, existing := range ss.state.Apps {
		if existing.ID == app.ID {
			app.UpdatedAt = time.Now()
			ss.state.Apps[i] = app
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("app not found: %s", app.ID)
	}

	return ss.save()
}

// UpdateAppStatus updates just the status of an app
func (ss *StateStore) UpdateAppStatus(id string, status AppStatus) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	found := false
	for i, app := range ss.state.Apps {
		if app.ID == id {
			ss.state.Apps[i].Status = status
			ss.state.Apps[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("app not found: %s", id)
	}

	return ss.save()
}

// UpdateAppHealth updates the health status of an app
func (ss *StateStore) UpdateAppHealth(id string, health HealthStatus) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	found := false
	for i, app := range ss.state.Apps {
		if app.ID == id {
			ss.state.Apps[i].Health = health
			ss.state.Apps[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("app not found: %s", id)
	}

	return ss.save()
}

// AddSnapshot adds a snapshot to an app
func (ss *StateStore) AddSnapshot(id string, snapshot AppSnapshot) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	found := false
	for i, app := range ss.state.Apps {
		if app.ID == id {
			ss.state.Apps[i].Snapshots = append(ss.state.Apps[i].Snapshots, snapshot)
			ss.state.Apps[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("app not found: %s", id)
	}

	return ss.save()
}

// RemoveSnapshot removes a snapshot from an app
func (ss *StateStore) RemoveSnapshot(appID, snapshotID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	found := false
	for i, app := range ss.state.Apps {
		if app.ID == appID {
			newSnapshots := []AppSnapshot{}
			for _, snap := range app.Snapshots {
				if snap.ID != snapshotID {
					newSnapshots = append(newSnapshots, snap)
				}
			}
			ss.state.Apps[i].Snapshots = newSnapshots
			ss.state.Apps[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("app not found: %s", appID)
	}

	return ss.save()
}

// DeleteApp removes an app from state
func (ss *StateStore) DeleteApp(id string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	newApps := []InstalledApp{}
	found := false

	for _, app := range ss.state.Apps {
		if app.ID != id {
			newApps = append(newApps, app)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("app not found: %s", id)
	}

	ss.state.Apps = newApps
	return ss.save()
}

// GetSnapshot returns a specific snapshot
func (ss *StateStore) GetSnapshot(appID, snapshotID string) (*AppSnapshot, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for _, app := range ss.state.Apps {
		if app.ID == appID {
			for _, snap := range app.Snapshots {
				if snap.ID == snapshotID || snap.Timestamp.Format("20060102-150405") == snapshotID {
					return &snap, nil
				}
			}
			return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
		}
	}

	return nil, fmt.Errorf("app not found: %s", appID)
}
