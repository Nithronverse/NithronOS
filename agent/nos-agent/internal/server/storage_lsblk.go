package server

import (
	"net/http"
	"os/exec"
	"strings"
)

// handleStorageLsblk exposes a restricted lsblk for nosd to call via the agent socket.
// Only allows a strict set of lsblk arguments.
func handleStorageLsblk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := exec.LookPath("lsblk"); err != nil {
		writeErr(w, http.StatusNotFound, "lsblk not found")
		return
	}
	// Fixed, allowlisted argument set
	args := []string{"--bytes", "--json", "-O", "-o", "NAME,KNAME,PATH,SIZE,ROTA,TYPE,TRAN,VENDOR,MODEL,SERIAL,MOUNTPOINT,FSTYPE,RM"}
	cmd := exec.Command("lsblk", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, strings.TrimSpace(string(out)))
		return
	}
	// Passthrough JSON (already JSON)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}
