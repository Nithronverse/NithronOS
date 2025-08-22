package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFstabAtomicEnsureAndRemove(t *testing.T) {
	old := etcDir
	dir := t.TempDir()
	etcDir = dir
	defer func() { etcDir = old }()

	// Ensure line
	req := httptest.NewRequest(http.MethodPost, "/v1/fstab/ensure", strings.NewReader(`{"line":"UUID=abcd /mnt btrfs defaults 0 0"}`))
	w := httptest.NewRecorder()
	handleFstabEnsure(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ensure status: %d", w.Code)
	}
	p := filepath.Join(dir, "fstab")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "UUID=abcd") {
		t.Fatalf("line not written: %s", string(b))
	}

	// Remove
	req2 := httptest.NewRequest(http.MethodPost, "/v1/fstab/remove", strings.NewReader(`{"contains":"UUID=abcd"}`))
	w2 := httptest.NewRecorder()
	handleFstabRemove(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("remove status: %d", w2.Code)
	}
	b2, _ := os.ReadFile(p)
	if strings.Contains(string(b2), "UUID=abcd") {
		t.Fatalf("line not removed: %s", string(b2))
	}
}
