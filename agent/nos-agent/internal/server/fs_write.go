package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FSWriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"`
	Owner   string `json:"owner"`
	Group   string `json:"group"`
	Atomic  *bool  `json:"atomic"`
}

func handleFSWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req FSWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !filepath.IsAbs(req.Path) || req.Path == "/" || req.Path == "" {
		writeErr(w, http.StatusBadRequest, "absolute path required")
		return
	}
	deny := []string{"/etc/passwd", "/etc/shadow", "/etc/group"}
	for _, d := range deny {
		if filepath.Clean(req.Path) == d {
			writeErr(w, http.StatusBadRequest, "path forbidden")
			return
		}
	}
	atomic := true
	if req.Atomic != nil {
		atomic = *req.Atomic
	}

	target := filepath.Clean(req.Path)
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("mkdir: %v", err))
		return
	}

	writePath := target
	if atomic {
		writePath = target + ".tmp"
	}
	f, err := os.OpenFile(writePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("open: %v", err))
		return
	}
	if _, err := f.WriteString(req.Content); err != nil {
		_ = f.Close()
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("write: %v", err))
		return
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("fsync: %v", err))
		return
	}
	_ = f.Close()

	if req.Mode != "" {
		if m, perr := parseMode(req.Mode); perr == nil {
			_ = os.Chmod(writePath, m)
		}
	}
	if (req.Owner != "" || req.Group != "") && runtimeChownSupported() {
		_ = chownByName(writePath, req.Owner, req.Group)
	}

	if atomic {
		if err := os.Rename(writePath, target); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("rename: %v", err))
			return
		}
	}

	logAuthPriv(fmt.Sprintf("fs.write %s (%d bytes)", target, len(req.Content)))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseMode(s string) (os.FileMode, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	// accept octal like 0644 or "0644" or "644" or "0775"
	if !strings.HasPrefix(s, "0") {
		s = "0" + s
	}
	var v uint64
	_, err := fmt.Sscanf(s, "%o", &v)
	if err != nil {
		return 0, err
	}
	return os.FileMode(v), nil
}
