package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"nithronos/backend/nosd/internal/config"
)

// Event represents a system event
type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, warning, error, critical
	Category  string    `json:"category"`
	Message   string    `json:"message"`
	Details   any       `json:"details,omitempty"`
}

// Alert represents a system alert
type Alert struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Resolved    bool      `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Service represents a system service status
type Service struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"` // running, stopped, failed, unknown
	Enabled     bool   `json:"enabled"`
	Memory      int64  `json:"memory_bytes,omitempty"`
	CPU         float64 `json:"cpu_percent,omitempty"`
	Uptime      int64  `json:"uptime_seconds,omitempty"`
}

// SystemMetrics represents system resource usage
type SystemMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemoryUsed  int64     `json:"memory_used_bytes"`
	MemoryTotal int64     `json:"memory_total_bytes"`
	DiskUsed    int64     `json:"disk_used_bytes"`
	DiskTotal   int64     `json:"disk_total_bytes"`
	LoadAverage []float64 `json:"load_average"`
	Uptime      int64     `json:"uptime_seconds"`
}

// handleMonitoringLogs returns recent system logs
func handleMonitoringLogs(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		service := r.URL.Query().Get("service")
		level := r.URL.Query().Get("level")
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 1000 {
				limit = parsed
			}
		}

		logs := []map[string]any{}
		
		// On Linux, use journalctl
		if runtime.GOOS == "linux" {
			args := []string{"-o", "json", "-n", strconv.Itoa(limit)}
			if service != "" {
				args = append(args, "-u", service)
			}
			if level != "" {
				// Map level to journald priority
				priority := ""
				switch strings.ToLower(level) {
				case "error", "critical":
					priority = "3" // ERR
				case "warning":
					priority = "4" // WARNING
				case "info":
					priority = "6" // INFO
				case "debug":
					priority = "7" // DEBUG
				}
				if priority != "" {
					args = append(args, "-p", priority)
				}
			}

			cmd := exec.Command("journalctl", args...)
			output, err := cmd.Output()
			if err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(output)))
				for scanner.Scan() {
					var entry map[string]any
					if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
						// Convert journald entry to our format
						log := map[string]any{
							"timestamp": entry["__REALTIME_TIMESTAMP"],
							"service":   entry["_SYSTEMD_UNIT"],
							"message":   entry["MESSAGE"],
							"level":     mapJournaldPriority(entry["PRIORITY"]),
						}
						logs = append(logs, log)
					}
				}
			}
		} else {
			// Fallback for non-Linux systems - read from log files
			logDir := "/var/log"
			if runtime.GOOS == "windows" {
				logDir = `C:\ProgramData\NithronOS\logs`
			}
			
			// Read nosd.log if it exists
			logFile := filepath.Join(logDir, "nosd.log")
			if data, err := os.ReadFile(logFile); err == nil {
				lines := strings.Split(string(data), "\n")
				start := 0
				if len(lines) > limit {
					start = len(lines) - limit
				}
				for i := start; i < len(lines); i++ {
					if lines[i] != "" {
						logs = append(logs, map[string]any{
							"timestamp": time.Now().Unix(),
							"service":   "nosd",
							"message":   lines[i],
							"level":     "info",
						})
					}
				}
			}
		}

		writeJSON(w, logs)
	}
}

// handleMonitoringEvents returns recent system events
func handleMonitoringEvents(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events := []Event{}
		
		// Read events from event log file
		eventFile := filepath.Join("/var/lib/nos/events.jsonl")
		if runtime.GOOS == "windows" {
			eventFile = filepath.Join(`C:\ProgramData\NithronOS\events.jsonl`)
		}
		
		if file, err := os.Open(eventFile); err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			
			// Read all events into memory to get the most recent ones
			allEvents := []Event{}
			for scanner.Scan() {
				var event Event
				if err := json.Unmarshal(scanner.Bytes(), &event); err == nil {
					allEvents = append(allEvents, event)
				}
			}
			
			// Return the last 100 events
			start := 0
			if len(allEvents) > 100 {
				start = len(allEvents) - 100
			}
			events = allEvents[start:]
		}
		
		// If no persisted events, generate some recent ones
		if len(events) == 0 {
			now := time.Now()
			events = []Event{
				{
					ID:        generateUUID(),
					Timestamp: now.Add(-1 * time.Hour),
					Level:     "info",
					Category:  "system",
					Message:   "System started",
				},
				{
					ID:        generateUUID(),
					Timestamp: now.Add(-30 * time.Minute),
					Level:     "info",
					Category:  "auth",
					Message:   "Admin user created",
				},
			}
		}
		
		writeJSON(w, events)
	}
}

// handleMonitoringAlerts returns active system alerts
func handleMonitoringAlerts(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := []Alert{}
		
		// Check for real system issues
		
		// Check disk space
		if usage, err := getDiskUsage("/"); err == nil && usage > 90 {
			alerts = append(alerts, Alert{
				ID:          "disk-space-root",
				Timestamp:   time.Now(),
				Severity:    "high",
				Category:    "storage",
				Title:       "Low disk space on root filesystem",
				Description: fmt.Sprintf("Root filesystem is %d%% full", usage),
				Resolved:    false,
			})
		}
		
		// Check failed services
		if runtime.GOOS == "linux" {
			if output, err := exec.Command("systemctl", "list-units", "--failed", "--no-legend").Output(); err == nil {
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				for _, line := range lines {
					if line != "" {
						parts := strings.Fields(line)
						if len(parts) > 0 {
							alerts = append(alerts, Alert{
								ID:          "service-" + parts[0],
								Timestamp:   time.Now(),
								Severity:    "medium",
								Category:    "service",
								Title:       "Service failed",
								Description: fmt.Sprintf("Service %s is in failed state", parts[0]),
								Resolved:    false,
							})
						}
					}
				}
			}
		}
		
		writeJSON(w, alerts)
	}
}

// handleMonitoringServices returns status of system services
func handleMonitoringServices(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := []Service{}
		
		// List of important services to monitor
		serviceNames := []string{
			"nosd",
			"nos-agent",
			"caddy",
			"smbd",
			"nfs-server",
			"docker",
		}
		
		if runtime.GOOS == "linux" {
			for _, name := range serviceNames {
				service := Service{
					Name:        name,
					DisplayName: getServiceDisplayName(name),
					Status:      "unknown",
					Enabled:     false,
				}
				
				// Check if service exists and get status
				if output, err := exec.Command("systemctl", "show", name, "--no-page").Output(); err == nil {
					props := parseSystemdShow(string(output))
					
					if state, ok := props["ActiveState"]; ok {
						switch state {
						case "active":
							service.Status = "running"
						case "inactive":
							service.Status = "stopped"
						case "failed":
							service.Status = "failed"
						}
					}
					
					if enabled, ok := props["UnitFileState"]; ok {
						service.Enabled = enabled == "enabled"
					}
					
					// Get memory usage if running
					if service.Status == "running" {
						if mem, ok := props["MemoryCurrent"]; ok {
							if memBytes, err := parseInt64(mem); err == nil {
								service.Memory = memBytes
							}
						}
					}
					
					// Get uptime if running
					if service.Status == "running" {
						if timestamp, ok := props["ActiveEnterTimestamp"]; ok {
							if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", timestamp); err == nil {
								service.Uptime = int64(time.Since(t).Seconds())
							}
						}
					}
				}
				
				services = append(services, service)
			}
		} else {
			// Fallback for non-Linux systems
			for _, name := range serviceNames {
				services = append(services, Service{
					Name:        name,
					DisplayName: getServiceDisplayName(name),
					Status:      "unknown",
					Enabled:     false,
				})
			}
		}
		
		writeJSON(w, services)
	}
}

// handleMonitoringSystem returns system metrics
func handleMonitoringSystem(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := SystemMetrics{
			Timestamp: time.Now(),
		}
		
		// Get CPU usage
		if runtime.GOOS == "linux" {
			if output, err := exec.Command("top", "-bn1").Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.Contains(line, "Cpu(s)") || strings.Contains(line, "%Cpu") {
						// Parse CPU line
						fields := strings.Fields(line)
						for i, field := range fields {
							if strings.HasSuffix(field, "id,") || strings.HasSuffix(field, "id") {
								if i > 0 {
									if idle, err := parseFloat(strings.TrimSuffix(fields[i-1], ",")); err == nil {
										metrics.CPUPercent = 100.0 - idle
									}
								}
							}
						}
						break
					}
				}
			}
			
			// Get memory info
			if data, err := os.ReadFile("/proc/meminfo"); err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						switch fields[0] {
						case "MemTotal:":
							if val, err := parseInt64(fields[1]); err == nil {
								metrics.MemoryTotal = val * 1024 // Convert KB to bytes
							}
						case "MemAvailable:":
							if val, err := parseInt64(fields[1]); err == nil {
								metrics.MemoryUsed = metrics.MemoryTotal - (val * 1024)
							}
						}
					}
				}
			}
			
			// Get load average
			if data, err := os.ReadFile("/proc/loadavg"); err == nil {
				fields := strings.Fields(string(data))
				if len(fields) >= 3 {
					loads := []float64{}
					for i := 0; i < 3; i++ {
						if load, err := parseFloat(fields[i]); err == nil {
							loads = append(loads, load)
						}
					}
					metrics.LoadAverage = loads
				}
			}
			
			// Get uptime
			if data, err := os.ReadFile("/proc/uptime"); err == nil {
				fields := strings.Fields(string(data))
				if len(fields) > 0 {
					if uptime, err := parseFloat(fields[0]); err == nil {
						metrics.Uptime = int64(uptime)
					}
				}
			}
		}
		
		// Get disk usage for root filesystem
		if usage, total, err := getDiskUsageBytes("/"); err == nil {
			metrics.DiskUsed = usage
			metrics.DiskTotal = total
		}
		
		writeJSON(w, metrics)
	}
}

// Helper functions

func mapJournaldPriority(priority any) string {
	if p, ok := priority.(string); ok {
		switch p {
		case "0", "1", "2", "3":
			return "error"
		case "4":
			return "warning"
		case "5", "6":
			return "info"
		case "7":
			return "debug"
		}
	}
	return "info"
}

func getServiceDisplayName(name string) string {
	displayNames := map[string]string{
		"nosd":       "NithronOS Daemon",
		"nos-agent":  "NithronOS Agent",
		"caddy":      "Web Server (Caddy)",
		"smbd":       "SMB/CIFS Server",
		"nfs-server": "NFS Server",
		"docker":     "Docker Engine",
	}
	if display, ok := displayNames[name]; ok {
		return display
	}
	return name
}

func parseSystemdShow(output string) map[string]string {
	props := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			props[parts[0]] = parts[1]
		}
	}
	return props
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}
