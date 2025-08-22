package sessions

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nithronos/backend/nosd/internal/fsatomic"
)

// Session represents an authenticated session (opaque cookie)
type Session struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	Roles     []string `json:"roles"`
	ExpiresAt string   `json:"expires_at"`
}

type diskFile struct {
	Version  int       `json:"version"`
	Sessions []Session `json:"sessions"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	mem  map[string]Session // by ID
}

func New(path string) *Store {
	s := &Store{path: path, mem: map[string]Session{}}
	_ = s.load()
	return s
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var f diskFile
	ok, err := fsatomic.LoadJSON(s.path, &f)
	if err != nil || !ok {
		return err
	}
	s.mem = map[string]Session{}
	for _, it := range f.Sessions {
		s.mem[it.ID] = it
	}
	return nil
}

func (s *Store) snapshot() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Session, 0, len(s.mem))
	for _, v := range s.mem {
		out = append(out, v)
	}
	return out
}

func (s *Store) Upsert(sess Session) error {
	s.mu.Lock()
	s.mem[sess.ID] = sess
	list := make([]Session, 0, len(s.mem))
	for _, v := range s.mem {
		list = append(list, v)
	}
	s.mu.Unlock()
	// persist 0600
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	return fsatomic.WithLock(s.path, func() error {
		return fsatomic.SaveJSON(context.TODO(), s.path, diskFile{Version: 1, Sessions: list}, fs.FileMode(0o600))
	})
}

func (s *Store) Get(id string) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.mem[id]
	return v, ok
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	delete(s.mem, id)
	list := make([]Session, 0, len(s.mem))
	for _, v := range s.mem {
		list = append(list, v)
	}
	s.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	return fsatomic.WithLock(s.path, func() error {
		return fsatomic.SaveJSON(context.TODO(), s.path, diskFile{Version: 1, Sessions: list}, fs.FileMode(0o600))
	})
}

func NowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

// List returns a snapshot of sessions.
func (s *Store) List() []Session {
	return s.snapshot()
}

// DeleteByUserID removes all sessions for the given user and persists the change.
func (s *Store) DeleteByUserID(userID string) error {
	s.mu.Lock()
	for id, v := range s.mem {
		if v.UserID == userID {
			delete(s.mem, id)
		}
	}
	list := make([]Session, 0, len(s.mem))
	for _, v := range s.mem {
		list = append(list, v)
	}
	s.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	return fsatomic.WithLock(s.path, func() error {
		return fsatomic.SaveJSON(context.TODO(), s.path, diskFile{Version: 1, Sessions: list}, fs.FileMode(0o600))
	})
}
