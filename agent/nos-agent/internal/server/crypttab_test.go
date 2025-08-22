package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrypttabAtomicEnsureAndRemove(t *testing.T) {
	old := etcDir
	dir := t.TempDir()
	etcDir = dir
	defer func() { etcDir = old }()

	// Ensure line
	req := httptest.NewRequest(http.MethodPost, "/v1/crypttab/ensure", strings.NewReader(`{"line":"luks-name UUID=abcd /etc/nos/keys/pool.key luks"}`))
	w := httptest.NewRecorder()
	handleCrypttabEnsure(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ensure status: %d", w.Code)
	}
	p := filepath.Join(dir, "crypttab")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "luks-name") {
		t.Fatalf("line not written: %s", string(b))
	}

	// Remove
	req2 := httptest.NewRequest(http.MethodPost, "/v1/crypttab/remove", strings.NewReader(`{"contains":"luks-name"}`))
	w2 := httptest.NewRecorder()
	handleCrypttabRemove(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("remove status: %d", w2.Code)
	}
	b2, _ := os.ReadFile(p)
	if strings.Contains(string(b2), "luks-name") {
		t.Fatalf("line not removed: %s", string(b2))
	}
}
