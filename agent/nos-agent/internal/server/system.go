package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AgentRequest represents a request to execute an action
type AgentRequest struct {
	Action string                 `json:"Action"`
	Params map[string]interface{} `json:"Params"`
}

// handleExecute handles the /execute endpoint for system configuration
func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	switch req.Action {
	case "system.hostname.set":
		handleSetHostname(w, req.Params)
	case "system.timezone.set":
		handleSetTimezone(w, req.Params)
	case "system.ntp.set":
		handleSetNTP(w, req.Params)
	case "system.network.configure":
		handleConfigureNetwork(w, req.Params)
	default:
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("unknown action: %s", req.Action))
	}
}

func handleSetHostname(w http.ResponseWriter, params map[string]interface{}) {
	hostname, ok := params["hostname"].(string)
	if !ok || hostname == "" {
		writeErr(w, http.StatusBadRequest, "hostname required")
		return
	}

	// Set hostname using hostnamectl
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "hostnamectl", "set-hostname", hostname)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Fallback: write directly to /etc/hostname
		if err := os.WriteFile("/etc/hostname", []byte(hostname+"\n"), 0644); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("failed to set hostname: %v", err))
			return
		}
		// Also try to set it immediately
		_ = exec.Command("hostname", hostname).Run()
	}

	// Update /etc/hosts
	hostsPath := "/etc/hosts"
	hostsData, _ := os.ReadFile(hostsPath)
	lines := strings.Split(string(hostsData), "\n")
	newLines := []string{}

	for _, line := range lines {
		if strings.Contains(line, "127.0.1.1") {
			newLines = append(newLines, fmt.Sprintf("127.0.1.1\t%s", hostname))
		} else {
			newLines = append(newLines, line)
		}
	}

	_ = os.WriteFile(hostsPath, []byte(strings.Join(newLines, "\n")), 0644)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "result": "success"})
}

func handleSetTimezone(w http.ResponseWriter, params map[string]interface{}) {
	timezone, ok := params["timezone"].(string)
	if !ok || timezone == "" {
		writeErr(w, http.StatusBadRequest, "timezone required")
		return
	}

	// Verify timezone exists
	tzPath := filepath.Join("/usr/share/zoneinfo", timezone)
	if _, err := os.Stat(tzPath); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid timezone: %s", timezone))
		return
	}

	// Set timezone using timedatectl
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "timedatectl", "set-timezone", timezone)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Fallback: create symlink manually
		_ = os.Remove("/etc/localtime")
		if err := os.Symlink(tzPath, "/etc/localtime"); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("failed to set timezone: %v", err))
			return
		}
		// Also write to /etc/timezone
		_ = os.WriteFile("/etc/timezone", []byte(timezone+"\n"), 0644)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "result": "success"})
}

func handleSetNTP(w http.ResponseWriter, params map[string]interface{}) {
	enabled, _ := params["enabled"].(bool)
	servers, _ := params["servers"].([]interface{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if enabled {
		// Enable NTP
		_ = exec.CommandContext(ctx, "timedatectl", "set-ntp", "true").Run()

		// Configure NTP servers if provided
		if len(servers) > 0 {
			// Create timesyncd config
			configDir := "/etc/systemd/timesyncd.conf.d"
			_ = os.MkdirAll(configDir, 0755)

			var serverList []string
			for _, s := range servers {
				if str, ok := s.(string); ok {
					serverList = append(serverList, str)
				}
			}

			if len(serverList) > 0 {
				config := "[Time]\n"
				config += fmt.Sprintf("NTP=%s\n", strings.Join(serverList, " "))
				_ = os.WriteFile(filepath.Join(configDir, "nithronos.conf"), []byte(config), 0644)
			}
		}

		// Restart timesyncd
		_ = exec.CommandContext(ctx, "systemctl", "restart", "systemd-timesyncd").Run()
	} else {
		// Disable NTP
		_ = exec.CommandContext(ctx, "timedatectl", "set-ntp", "false").Run()
		_ = exec.CommandContext(ctx, "systemctl", "stop", "systemd-timesyncd").Run()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "result": "success"})
}

func handleConfigureNetwork(w http.ResponseWriter, params map[string]interface{}) {
	// Placeholder for network configuration
	// This would need more complex implementation based on the network manager in use
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "result": "success"})
}
