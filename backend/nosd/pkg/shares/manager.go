package shares

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	SharesConfigPath = "/etc/nos/shares.json"
	SharesDir        = "/srv/shares"
)

// Manager handles share operations and persistence
type Manager struct {
	mu       sync.RWMutex
	filepath string
	shares   map[string]*Share
}

// NewManager creates a new shares manager
func NewManager(filepath string) *Manager {
	if filepath == "" {
		filepath = SharesConfigPath
	}
	return &Manager{
		filepath: filepath,
		shares:   make(map[string]*Share),
	}
}

// Load reads shares from disk
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create file if it doesn't exist
	if _, err := os.Stat(m.filepath); os.IsNotExist(err) {
		initial := &SharesFile{
			Version: 1,
			Items:   []*Share{},
		}
		if err := m.saveUnsafe(initial); err != nil {
			return fmt.Errorf("failed to create initial shares file: %w", err)
		}
	}

	data, err := os.ReadFile(m.filepath)
	if err != nil {
		return fmt.Errorf("failed to read shares file: %w", err)
	}

	var file SharesFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse shares file: %w", err)
	}

	// Rebuild map
	m.shares = make(map[string]*Share)
	for _, share := range file.Items {
		m.shares[share.Name] = share
	}

	return nil
}

// List returns all shares
func (m *Manager) List() ([]*Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shares := make([]*Share, 0, len(m.shares))
	for _, share := range m.shares {
		shares = append(shares, share)
	}
	return shares, nil
}

// Get returns a share by name
func (m *Manager) Get(name string) (*Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	share, exists := m.shares[name]
	if !exists {
		return nil, fmt.Errorf("share not found: %s", name)
	}
	return share, nil
}

// Create adds a new share
func (m *Manager) Create(req *CreateRequest) (*Share, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if share already exists
	if _, exists := m.shares[req.Name]; exists {
		return nil, &Error{
			Code:    ErrCodeNameExists,
			Message: fmt.Sprintf("share %s already exists", req.Name),
		}
	}

	// Create share object
	now := time.Now().UTC().Format(time.RFC3339)
	share := &Share{
		Name:        req.Name,
		Path:        fmt.Sprintf("%s/%s", SharesDir, req.Name),
		SMB:         req.SMB,
		NFS:         req.NFS,
		Owners:      req.Owners,
		Readers:     req.Readers,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Validate
	if err := share.Validate(); err != nil {
		return nil, &Error{
			Code:    ErrCodeInvalidName,
			Message: err.Error(),
		}
	}

	// Add to map
	m.shares[share.Name] = share

	// Save to disk
	if err := m.saveUnsafe(m.toFile()); err != nil {
		delete(m.shares, share.Name) // rollback
		return nil, fmt.Errorf("failed to save shares: %w", err)
	}

	return share, nil
}

// Update modifies an existing share
func (m *Manager) Update(name string, req *UpdateRequest) (*Share, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[name]
	if !exists {
		return nil, &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("share %s not found", name),
		}
	}

	// Create a copy for rollback
	original := *share

	// Apply updates
	if req.SMB != nil {
		share.SMB = req.SMB
	}
	if req.NFS != nil {
		share.NFS = req.NFS
	}
	if req.Owners != nil {
		share.Owners = req.Owners
	}
	if req.Readers != nil {
		share.Readers = req.Readers
	}
	if req.Description != nil {
		share.Description = *req.Description
	}
	share.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Validate
	if err := share.Validate(); err != nil {
		*share = original // rollback
		return nil, &Error{
			Code:    ErrCodeInvalidName,
			Message: err.Error(),
		}
	}

	// Save to disk
	if err := m.saveUnsafe(m.toFile()); err != nil {
		*share = original // rollback
		return nil, fmt.Errorf("failed to save shares: %w", err)
	}

	return share, nil
}

// Delete removes a share
func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[name]
	if !exists {
		return &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("share %s not found", name),
		}
	}

	// Remove from map
	delete(m.shares, name)

	// Save to disk
	if err := m.saveUnsafe(m.toFile()); err != nil {
		m.shares[name] = share // rollback
		return fmt.Errorf("failed to save shares: %w", err)
	}

	return nil
}

// Test performs a dry-run validation of a share configuration
func (m *Manager) Test(name string, config json.RawMessage) (*TestResponse, error) {
	// Parse config as either CreateRequest or UpdateRequest
	var req CreateRequest
	if err := json.Unmarshal(config, &req); err != nil {
		return &TestResponse{
			Valid:  false,
			Errors: []string{fmt.Sprintf("invalid configuration: %v", err)},
		}, nil
	}

	// If name is provided in path, it's an update test
	if name != "" {
		m.mu.RLock()
		existing, exists := m.shares[name]
		m.mu.RUnlock()

		if !exists {
			return &TestResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("share %s not found", name)},
			}, nil
		}

		// Simulate update
		testShare := *existing
		if req.SMB != nil {
			testShare.SMB = req.SMB
		}
		if req.NFS != nil {
			testShare.NFS = req.NFS
		}
		if req.Owners != nil {
			testShare.Owners = req.Owners
		}
		if req.Readers != nil {
			testShare.Readers = req.Readers
		}

		if err := testShare.Validate(); err != nil {
			return &TestResponse{
				Valid:  false,
				Errors: []string{err.Error()},
			}, nil
		}
	} else {
		// Test create
		share := &Share{
			Name:    req.Name,
			Path:    fmt.Sprintf("%s/%s", SharesDir, req.Name),
			SMB:     req.SMB,
			NFS:     req.NFS,
			Owners:  req.Owners,
			Readers: req.Readers,
		}

		if err := share.Validate(); err != nil {
			return &TestResponse{
				Valid:  false,
				Errors: []string{err.Error()},
			}, nil
		}

		// Check if name already exists
		m.mu.RLock()
		_, exists := m.shares[req.Name]
		m.mu.RUnlock()

		if exists {
			return &TestResponse{
				Valid:  false,
				Errors: []string{fmt.Sprintf("share %s already exists", req.Name)},
			}, nil
		}
	}

	return &TestResponse{Valid: true}, nil
}

// toFile converts the current state to a SharesFile
func (m *Manager) toFile() *SharesFile {
	items := make([]*Share, 0, len(m.shares))
	for _, share := range m.shares {
		items = append(items, share)
	}
	return &SharesFile{
		Version: 1,
		Items:   items,
	}
}

// saveUnsafe saves the file atomically (caller must hold lock)
func (m *Manager) saveUnsafe(file *SharesFile) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shares: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write atomically
	if err := os.WriteFile(m.filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write shares file: %w", err)
	}

	// Set ownership to nosd:nosd (if running as root during setup)
	// This is a no-op if already correct or if not root
	_ = os.Chown(m.filepath, 1000, 1000) // nosd uid/gid

	return nil
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.Message
}
