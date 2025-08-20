package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ComposeReq struct {
	ID  string `json:"id"`
	Dir string `json:"dir"`
}

func handleComposeUp(w http.ResponseWriter, r *http.Request) {
	var req ComposeReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if !validID(req.ID) || !validDir(req.Dir) {
		writeErr(w, http.StatusBadRequest, "invalid id/dir")
		return
	}
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = req.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("compose up failed: %s", string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func handleComposeDown(w http.ResponseWriter, r *http.Request) {
	var req ComposeReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if !validID(req.ID) || !validDir(req.Dir) {
		writeErr(w, http.StatusBadRequest, "invalid id/dir")
		return
	}
	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = req.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("compose down failed: %s", string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func handleSystemdInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		UnitText string `json:"unit_text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if !validID(req.ID) || req.UnitText == "" {
		writeErr(w, http.StatusBadRequest, "invalid request")
		return
	}
	unitPath := filepath.Join("/etc/systemd/system", fmt.Sprintf("nos-app-%s.service", req.ID))
	if err := os.WriteFile(unitPath, []byte(req.UnitText), 0o644); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// enable now
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		writeErr(w, http.StatusInternalServerError, string(out))
		return
	}
	if out, err := exec.Command("systemctl", "enable", "--now", fmt.Sprintf("nos-app-%s.service", req.ID)).CombinedOutput(); err != nil {
		writeErr(w, http.StatusInternalServerError, string(out))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func validID(id string) bool { return id != "" && !strings.ContainsAny(id, "/.. ") }
func validDir(dir string) bool {
	return strings.HasPrefix(dir, "/opt/") && !strings.Contains(dir, "..")
}
