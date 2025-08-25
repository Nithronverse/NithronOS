package server

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemInfo represents system information
type SystemInfo struct {
	Hostname     string  `json:"hostname"`
	Uptime       uint64  `json:"uptime"`
	Kernel       string  `json:"kernel"`
	Version      string  `json:"version"`
	Arch         string  `json:"arch,omitempty"`
	CPUCount     int     `json:"cpuCount,omitempty"`
	MemoryTotal  uint64  `json:"memoryTotal,omitempty"`
	MemoryUsed   uint64  `json:"memoryUsed,omitempty"`
}

// ServiceStatus represents the status of a system service
type ServiceStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // running, stopped, failed, unknown
	Enabled bool   `json:"enabled"`
	Uptime  uint64 `json:"uptime,omitempty"`
}

// SystemHandler handles system-related endpoints
type SystemHandler struct{}

// NewSystemHandler creates a new system handler
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// Routes registers the system routes
func (h *SystemHandler) Routes() chi.Router {
	r := chi.NewRouter()
	
	r.Get("/info", h.GetSystemInfo)
	r.Get("/services", h.GetServices)
	
	return r
}

// GetSystemInfo returns system information
// GET /api/v1/system/info
func (h *SystemHandler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	info := SystemInfo{
		Arch: runtime.GOARCH,
	}
	
	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}
	
	// Get host info (uptime, kernel)
	if hostInfo, err := host.Info(); err == nil {
		info.Uptime = hostInfo.Uptime
		info.Kernel = hostInfo.KernelVersion
		info.Version = hostInfo.Platform + " " + hostInfo.PlatformVersion
	}
	
	// Get CPU count
	if cpuCount, err := cpu.Counts(true); err == nil {
		info.CPUCount = cpuCount
	}
	
	// Get memory info
	if memInfo, err := mem.VirtualMemory(); err == nil {
		info.MemoryTotal = memInfo.Total
		info.MemoryUsed = memInfo.Used
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Error().Err(err).Msg("Failed to encode system info")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetServices returns the status of system services
// GET /api/v1/system/services
func (h *SystemHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	services := []ServiceStatus{
		{
			Name:    "nosd",
			Status:  "running", // We're running if we're serving this request
			Enabled: true,
			Uptime:  uint64(time.Since(startTime).Seconds()),
		},
		{
			Name:    "nos-agent",
			Status:  h.checkServiceStatus("nos-agent"),
			Enabled: true,
		},
		{
			Name:    "caddy",
			Status:  h.checkServiceStatus("caddy"),
			Enabled: true,
		},
		{
			Name:    "docker",
			Status:  h.checkServiceStatus("docker"),
			Enabled: true,
		},
		{
			Name:    "smbd",
			Status:  h.checkServiceStatus("smbd"),
			Enabled: h.isServiceEnabled("smbd"),
		},
		{
			Name:    "nfs-server",
			Status:  h.checkServiceStatus("nfs-server"),
			Enabled: h.isServiceEnabled("nfs-server"),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(services); err != nil {
		log.Error().Err(err).Msg("Failed to encode services")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// checkServiceStatus checks if a systemd service is running
func (h *SystemHandler) checkServiceStatus(service string) string {
	// This is a simplified check - in production, you'd use systemd D-Bus API
	// or parse systemctl output
	
	// For now, return "unknown" for services we can't check
	// In a real implementation, you'd execute:
	// systemctl is-active <service>
	
	return "unknown"
}

// isServiceEnabled checks if a systemd service is enabled
func (h *SystemHandler) isServiceEnabled(service string) bool {
	// This is a simplified check - in production, you'd use systemd D-Bus API
	// or parse systemctl output
	
	// For now, return true for critical services
	return service == "smbd" || service == "nfs-server"
}

var startTime = time.Now()
