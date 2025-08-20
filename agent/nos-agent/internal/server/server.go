package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const SocketPath = "/run/nos-agent.sock"

// Start creates the unix socket listener and serves the HTTP API.
func Start() error {
	if err := mustBeRoot(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(SocketPath), 0o755); err != nil {
		return fmt.Errorf("mkdir socket dir: %w", err)
	}
	_ = os.Remove(SocketPath)

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	// Restrict perms; systemd unit is expected to manage ownership/group.
	if runtime.GOOS != "windows" {
		_ = os.Chmod(SocketPath, 0o660)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/btrfs/create", handleBtrfsCreate)
	mux.HandleFunc("/v1/service/reload", handleServiceReload)

	return http.Serve(l, mux)
}

type PlanResponse struct {
	Plan []string `json:"plan"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type BtrfsCreateRequest struct {
	Devices []string `json:"devices"`
	Raid    string   `json:"raid"`
	Label   string   `json:"label"`
	Encrypt bool     `json:"encrypt"`
}

func handleBtrfsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req BtrfsCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Devices) == 0 {
		writeErr(w, http.StatusBadRequest, "devices required")
		return
	}
	allowedRaids := map[string]bool{"single": true, "raid1": true, "raid10": true}
	if req.Raid == "" {
		req.Raid = "single"
	}
	if !allowedRaids[strings.ToLower(req.Raid)] {
		writeErr(w, http.StatusBadRequest, "invalid raid")
		return
	}
	if req.Label == "" {
		req.Label = "pool"
	}

	plan := []string{}
	if req.Encrypt {
		for idx, dev := range req.Devices {
			name := fmt.Sprintf("nos%d", idx)
			plan = append(plan,
				fmt.Sprintf("cryptsetup luksFormat %s", shellQuote(dev)),
				fmt.Sprintf("cryptsetup open %s %s", shellQuote(dev), shellQuote(name)),
			)
		}
		mapped := []string{}
		for idx := range req.Devices {
			mapped = append(mapped, "/dev/mapper/"+fmt.Sprintf("nos%d", idx))
		}
		plan = append(plan, mkfsBtrfsCommand(req.Label, req.Raid, mapped...))
	} else {
		plan = append(plan, mkfsBtrfsCommand(req.Label, req.Raid, req.Devices...))
	}

	writeJSON(w, http.StatusOK, PlanResponse{Plan: plan})
}

type ServiceReloadRequest struct {
	Name string `json:"name"`
}

func handleServiceReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req ServiceReloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	allowed := map[string]string{
		"smb": "smb",
	}
	unit, ok := allowed[strings.ToLower(req.Name)]
	if !ok {
		writeErr(w, http.StatusBadRequest, "service not allowed")
		return
	}
	plan := []string{fmt.Sprintf("systemctl reload %s", shellQuote(unit))}
	writeJSON(w, http.StatusOK, PlanResponse{Plan: plan})
}

func mkfsBtrfsCommand(label, raid string, devices ...string) string {
	args := []string{"mkfs.btrfs", "-L", label}
	if raid != "single" {
		args = append(args, "-m", raid, "-d", raid)
	}
	args = append(args, devices...)
	return strings.Join(quoteAll(args), " ")
}

func quoteAll(items []string) []string {
	res := make([]string, len(items))
	for i, v := range items {
		res[i] = shellQuote(v)
	}
	return res
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// umask and root checks are handled in OS-specific files
