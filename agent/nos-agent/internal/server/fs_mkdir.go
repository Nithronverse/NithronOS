package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type FSMkdirRequest struct {
	Path  string `json:"path"`
	Mode  string `json:"mode"`
	Owner string `json:"owner"`
	Group string `json:"group"`
}

func handleFSMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runtime.GOOS == "windows" {
		writeErr(w, http.StatusNotImplemented, "not supported on windows")
		return
	}
	var req FSMkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !filepath.IsAbs(req.Path) || req.Path == "/" || req.Path == "" {
		writeErr(w, http.StatusBadRequest, "absolute path required")
		return
	}
	clean := filepath.Clean(req.Path)
	deny := map[string]struct{}{"/": {}, "/etc": {}, "/boot": {}, "/root": {}}
	if _, bad := deny[clean]; bad {
		writeErr(w, http.StatusBadRequest, "path forbidden")
		return
	}

	if err := os.MkdirAll(clean, 0o775); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("mkdir: %v", err))
		return
	}
	// chmod
	if req.Mode != "" {
		if m, err := parseMode(req.Mode); err == nil {
			_ = os.Chmod(clean, m)
		}
	} else {
		_ = os.Chmod(clean, 0o775)
	}
	// chown
	if req.Owner != "" || req.Group != "" {
		if runtimeChownSupported() {
			_ = chownByName(clean, req.Owner, req.Group)
		}
	}

	logAuthPriv("fs.mkdir " + clean)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
