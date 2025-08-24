package firstboot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	OTP       string    `json:"otp"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Store struct {
	path string
}

func New(path string) *Store { return &Store{path: path} }

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (*State, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		// Fallback: legacy schema {otp, created_at, used}
		var legacy struct {
			OTP       string `json:"otp"`
			CreatedAt string `json:"created_at"`
			Used      bool   `json:"used"`
		}
		if json.Unmarshal(b, &legacy) == nil && legacy.OTP != "" && !legacy.Used {
			if t, e := time.Parse(time.RFC3339, legacy.CreatedAt); e == nil {
				st.OTP = legacy.OTP
				st.IssuedAt = t
				st.ExpiresAt = t.Add(15 * time.Minute)
			}
		} else {
			return nil, err
		}
	} else if st.ExpiresAt.IsZero() {
		// Handle legacy content that partially unmarshalled (otp) but lacks timestamps
		var legacy struct {
			OTP       string `json:"otp"`
			CreatedAt string `json:"created_at"`
			Used      bool   `json:"used"`
		}
		if json.Unmarshal(b, &legacy) == nil && legacy.OTP != "" && !legacy.Used {
			if t, e := time.Parse(time.RFC3339, legacy.CreatedAt); e == nil {
				st.OTP = legacy.OTP
				st.IssuedAt = t
				st.ExpiresAt = t.Add(15 * time.Minute)
			}
		}
	}
	if time.Now().After(st.ExpiresAt) {
		// expired; best-effort delete
		_ = os.Remove(s.path)
		return nil, nil
	}
	return &st, nil
}

func (s *Store) NewOrReuse(ttl time.Duration, gen func() string) (*State, bool, error) {
	if st, err := s.Load(); err != nil {
		return nil, false, err
	} else if st != nil {
		return st, true, nil
	}
	// new
	now := time.Now().UTC()
	st := &State{OTP: gen(), IssuedAt: now, ExpiresAt: now.Add(ttl)}
	if err := s.SaveAtomic(context.Background(), st, 0o600); err != nil {
		return nil, false, err
	}
	return st, false, nil
}

func (s *Store) SaveAtomic(ctx context.Context, st *State, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(st); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// fsync parent dir best-effort
	if dirf, err := os.Open(filepath.Dir(s.path)); err == nil {
		_ = dirf.Sync()
		_ = dirf.Close()
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	return nil
}

func (s *Store) String() string { return fmt.Sprintf("Store(%s)", s.path) }
