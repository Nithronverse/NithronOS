package server

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
)

func handleBtrfsScrubStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Mount string `json:"mount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Mount) == "" || !filepath.IsAbs(body.Mount) {
		writeErr(w, http.StatusBadRequest, "absolute mount path required")
		return
	}
	out, err := exec.Command("btrfs", "scrub", "start", "-B", body.Mount).CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, strings.TrimSpace(string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "output": string(out)})
}

func handleBtrfsScrubStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	mount := r.URL.Query().Get("mount")
	if strings.TrimSpace(mount) == "" || !filepath.IsAbs(mount) {
		writeErr(w, http.StatusBadRequest, "absolute mount path required")
		return
	}
	out, err := exec.Command("btrfs", "scrub", "status", mount).CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, strings.TrimSpace(string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": string(out)})
}

func handleBtrfsCheckRepair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Path     string `json:"path"`
		Advanced bool   `json:"advanced"`
		Confirm  string `json:"confirm"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if !body.Advanced || strings.ToUpper(strings.TrimSpace(body.Confirm)) != "REPAIR" {
		writeErr(w, http.StatusForbidden, "repair requires advanced=true and confirm=REPAIR")
		return
	}
	if strings.TrimSpace(body.Path) == "" || !filepath.IsAbs(body.Path) {
		writeErr(w, http.StatusBadRequest, "absolute path required")
		return
	}
	out, err := exec.Command("btrfs", "check", "--repair", body.Path).CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, strings.TrimSpace(string(out)))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "output": string(out)})
}
