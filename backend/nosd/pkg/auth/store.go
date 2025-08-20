package auth

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type Store struct {
	path  string
	users map[string]User // by email
	mu    sync.RWMutex
}

var ErrUserNotFound = errors.New("user not found")

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, users: map[string]User{}}
	_ = s.load()
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var list []User
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	for _, u := range list {
		s.users[u.Email] = u
	}
	return nil
}

func (s *Store) GetByEmail(email string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (s *Store) GetByEmailOrID(idOrEmail string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.users[idOrEmail]; ok {
		return u, nil
	}
	for _, u := range s.users {
		if u.ID == idOrEmail || u.Email == idOrEmail {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (s *Store) UpdateUser(u User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[u.Email] = u
	// write out as list
	list := make([]User, 0, len(s.users))
	for _, v := range s.users {
		list = append(list, v)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.path, b, 0o600); err != nil {
		return err
	}
	return nil
}
