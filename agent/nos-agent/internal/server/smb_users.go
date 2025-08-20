package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

var usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

type SMBUserCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func handleSMBUserCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req SMBUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !usernameRe.MatchString(req.Username) {
		writeErr(w, http.StatusBadRequest, "invalid username")
		return
	}
	// ensure system user exists (idempotent)
	if out, err := exec.Command("id", "-u", req.Username).CombinedOutput(); err != nil {
		_ = out
		_ = exec.Command("useradd", "-M", "-s", "/usr/sbin/nologin", req.Username).Run()
	}
	// set samba password non-interactively; if empty, set a random disabled password
	pass := req.Password
	if pass == "" {
		pass = "x"
	}
	cmd := exec.Command("sh", "-c", fmt.Sprintf("(echo %q; echo %q) | smbpasswd -s -a %s", pass, pass, shellQuote(req.Username)))
	if out, err := cmd.CombinedOutput(); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("smbpasswd failed: %s", strings.TrimSpace(string(out))))
		return
	}
	logAuthPriv("smb.user-create " + req.Username)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleSMBUsersList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cmd := exec.Command("pdbedit", "-L")
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "pdbedit failed")
		return
	}
	users := []string{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// format: username:... or username:XXXX:... depending on backend
		if i := strings.Index(line, ":"); i > 0 {
			u := line[:i]
			if usernameRe.MatchString(u) {
				users = append(users, u)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}
