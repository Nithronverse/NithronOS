package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"nithronos/backend/nosd/internal/fsatomic"
)

func TestSaveAtomicConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	// Seed with one user
	_ = s.UpsertUser(User{ID: "u0", Username: "admin", PasswordHash: "plain:x", Roles: []string{"admin"}})

	// Start concurrent writers
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := User{ID: "u" + string(rune('a'+i)), Username: "user" + string(rune('a'+i)), PasswordHash: "plain:x"}
			_ = s.UpsertUser(u)
		}(i)
	}
	wg.Wait()

	// Ensure file exists and is valid JSON
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read users: %v", err)
	}
	var f dbFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
}

func TestCrashRecoveryFromTmp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	s, _ := New(path)

	// Write a good file first
	good := dbFile{Version: 1, Users: []User{{ID: "x", Username: "x"}}}
	if err := fsatomic.SaveJSON(context.TODO(), path, good, 0o600); err != nil {
		t.Fatal(err)
	}
	// Create a broken crash artifact
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Next load should ignore tmp and keep good file
	if err := s.load(); err != nil {
		t.Fatalf("load after tmp: %v", err)
	}
	// users.json should still exist
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(path); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("users.json missing: %v", err)
	}
}
