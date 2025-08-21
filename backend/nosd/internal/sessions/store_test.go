package sessions

import (
	"path/filepath"
	"testing"
)

func TestSessionsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	s := New(path)
	sess := Session{ID: "sid1", UserID: "u1", Roles: []string{"user"}, ExpiresAt: NowUTC()}
	if err := s.Upsert(sess); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, ok := s.Get("sid1")
	if !ok || got.UserID != "u1" {
		t.Fatalf("get mismatch: %+v ok=%v", got, ok)
	}
	if err := s.Delete("sid1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := s.Get("sid1"); ok {
		t.Fatalf("expected delete")
	}
}
