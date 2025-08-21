package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nithronos/backend/nosd/internal/fsatomic"
)

type User struct {
	ID             string   `json:"id"`
	Username       string   `json:"username"`
	PasswordHash   string   `json:"password_hash"`
	Roles          []string `json:"roles"`
	TOTPEnc        string   `json:"totp_enc"`
	RecoveryHashes []string `json:"recovery_hashes"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	LastLoginAt    string   `json:"last_login_at"`
	FailedAttempts int      `json:"failed_attempts"`
	LockedUntil    string   `json:"locked_until"`
}

type dbFile struct {
	Version int    `json:"version"`
	Users   []User `json:"users"`
}

var (
	ErrUserNotFound = errors.New("user not found")
)

type Store struct {
	path  string
	users map[string]User // by username
	mu    sync.RWMutex
	ioMu  sync.Mutex // serialize writers within this process
}

func New(path string) (*Store, error) {
	s := &Store{path: path, users: map[string]User{}}
	if err := s.load(); err != nil {
		// Start empty on missing/invalid file to avoid panics in early flows/tests
		s.users = map[string]User{}
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Clean any crash artifact and load
	var f dbFile
	ok, err := fsatomic.LoadJSON(s.path, &f)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if f.Version != 1 {
		return fmt.Errorf("unsupported users db version: %d", f.Version)
	}
	for _, u := range f.Users {
		s.users[u.Username] = u
	}
	return nil
}

// writeUsers persists the given snapshot without holding s.mu.
func (s *Store) writeUsers(list []User) error {
	data := dbFile{Version: 1, Users: list}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	// In-process serialize
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	return fsatomic.WithLock(s.path, func() error {
		return fsatomic.SaveJSON(context.Background(), s.path, data, fs.FileMode(0o600))
	})
}

func (s *Store) HasAdmin() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		for _, r := range u.Roles {
			if r == "admin" {
				return true
			}
		}
	}
	return false
}

func (s *Store) FindByUsername(username string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[username]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (s *Store) FindByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (s *Store) UpsertUser(u User) error {
	// Update in-memory under write lock and take a snapshot
	s.mu.Lock()
	now := time.Now().UTC().Format(time.RFC3339)
	if u.CreatedAt == "" {
		u.CreatedAt = now
	}
	u.UpdatedAt = now
	s.users[u.Username] = u
	// snapshot current users to avoid holding the mutex during IO
	list := make([]User, 0, len(s.users))
	for _, usr := range s.users {
		list = append(list, usr)
	}
	s.mu.Unlock()
	// Persist snapshot without holding the lock
	return s.writeUsers(list)
}
