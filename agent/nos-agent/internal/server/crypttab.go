package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Minimal crypttab helpers: append/remove lines atomically.
func handleCrypttabEnsure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Line string `json:"line"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Line) == "" {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	path := filepath.Join(etcDir, "crypttab")
	data := ""
	if b, err := os.ReadFile(path); err == nil {
		data = string(b)
	}
	if !strings.Contains(data, body.Line) {
		if !strings.HasSuffix(data, "\n") && len(data) > 0 {
			data += "\n"
		}
		data += body.Line + "\n"
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if f, err := os.OpenFile(path+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
			_, _ = f.WriteString(data)
			_ = f.Sync()
			_ = f.Close()
			_ = fsyncDir(filepath.Dir(path))
			_ = os.Rename(path+".tmp", path)
			_ = fsyncDir(filepath.Dir(path))
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleCrypttabRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Contains string `json:"contains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Contains) == "" {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	path := filepath.Join(etcDir, "crypttab")
	b, err := os.ReadFile(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if !strings.Contains(ln, body.Contains) {
			out = append(out, ln)
		}
	}
	data := strings.Join(out, "\n")
	if f, err := os.OpenFile(path+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
		_, _ = f.WriteString(data)
		_ = f.Sync()
		_ = f.Close()
		_ = fsyncDir(filepath.Dir(path))
		_ = os.Rename(path+".tmp", path)
		_ = fsyncDir(filepath.Dir(path))
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
