package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"nithronos/backend/nosd/internal/config"
)

func TestEnsureFirstBootOTP_PrintsAndPersists(t *testing.T) {
	dir := t.TempDir()
	users := filepath.Join(dir, "users.json")
	first := filepath.Join(dir, "firstboot.json")
	t.Setenv("NOS_USERS_PATH", users)
	t.Setenv("NOS_FIRSTBOOT_PATH", first)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	cfg := config.FromEnv()
	ensureFirstBootOTP(cfg)
	_ = w.Close()
	out, _ := io.ReadAll(r)
	if !regexp.MustCompile(`First-boot OTP: \d{6}`).Match(out) {
		t.Fatalf("expected OTP line in stdout, got: %s", string(out))
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("firstboot.json not created: %v", err)
	}

	// Simulate restart: recapture stdout and force regeneration by removing file
	_ = os.Remove(first)
	r2, w2, _ := os.Pipe()
	os.Stdout = w2
	ensureFirstBootOTP(cfg)
	_ = w2.Close()
	out2, _ := io.ReadAll(r2)
	if !bytes.Contains(out2, []byte("First-boot OTP:")) {
		t.Fatalf("expected OTP printed again on restart")
	}
}

// no extra helpers
