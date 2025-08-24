package firstboot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func gen() string { return "123456" }

func TestSaveLoadReuseAndExpire(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "firstboot.json")
	s := New(p)
	// New
	st, reused, err := s.NewOrReuse(15*time.Minute, gen)
	if err != nil || reused || st == nil || st.OTP != "123456" {
		t.Fatalf("new: err=%v reused=%v st=%v", err, reused, st)
	}
	// Reuse
	st2, reused2, err := s.NewOrReuse(15*time.Minute, func() string { return "000000" })
	if err != nil || !reused2 || st2 == nil || st2.OTP != "123456" {
		t.Fatalf("reuse: err=%v reused=%v st=%v", err, reused2, st2)
	}
	// Expire by editing file
	st2.ExpiresAt = time.Now().Add(-1 * time.Minute)
	_ = s.SaveAtomic(context.TODO(), st2, 0o600)
	st3, reused3, err := s.NewOrReuse(15*time.Minute, func() string { return "654321" })
	if err != nil || reused3 || st3 == nil || st3.OTP != "654321" {
		t.Fatalf("remint: err=%v reused=%v st=%v", err, reused3, st3)
	}
}

func TestEACCESStorage(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skip as root")
	}
	// Use an unwritable directory (root-owned temp dir variant is hard to ensure; simulate with file path that cannot be a dir)
	// Create a file where the dir would be
	dir := t.TempDir()
	badDir := filepath.Join(dir, "state")
	// create a file named 'state' so MkdirAll on its parent works, but opening file under it fails
	if err := os.WriteFile(badDir, []byte(""), 0o600); err != nil {
		t.Skip("cannot simulate")
	}
	p := filepath.Join(badDir, "firstboot.json")
	s := New(p)
	if _, _, err := s.NewOrReuse(15*time.Minute, gen); err == nil {
		t.Fatalf("expected error creating under non-directory path")
	}
}
