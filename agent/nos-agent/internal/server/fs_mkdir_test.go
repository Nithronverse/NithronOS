package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFSMkdir_RejectsUnsafePaths(t *testing.T) {
	body := FSMkdirRequest{Path: "/"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/fs/mkdir", bytes.NewReader(b))
	w := httptest.NewRecorder()
	handleFSMkdir(w, req)
	if runtime.GOOS == "windows" {
		if w.Code != http.StatusNotImplemented {
			t.Fatalf("expected 501 on windows, got %d", w.Code)
		}
		return
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestFSMkdir_SuccessCreatesDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on windows")
	}
	base, err := os.MkdirTemp("", "nos-mkdir-*")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(base)
	target := filepath.Join(base, "newdir", "sub")

	body := FSMkdirRequest{Path: target, Mode: "0775"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/fs/mkdir", bytes.NewReader(b))
	w := httptest.NewRecorder()
	handleFSMkdir(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	fi, err := os.Stat(target)
	if err != nil || !fi.IsDir() {
		t.Fatalf("dir not created: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o775 {
		t.Fatalf("perm mismatch: got %o want 775", got)
	}
}
