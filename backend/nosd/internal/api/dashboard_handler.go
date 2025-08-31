package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// DashboardResponse aggregates all dashboard data
type DashboardResponse struct {
	System      SystemSummary      `json:"system"`
	Storage     StorageSummary     `json:"storage"`
	Disks       DisksSummary       `json:"disks"`
	Shares      []ShareInfo        `json:"shares"`
	Apps        []AppInfo          `json:"apps"`
	Maintenance MaintenanceStatus  `json:"maintenance"`
	Events      []EventInfo        `json:"events"`
}

// SystemSummary for dashboard widget
type SystemSummary struct {
	Status   string  `json:"status"` // ok, degraded, critical
	CPUPct   float64 `json:"cpuPct"`
	Load1    float64 `json:"load1"`
	Memory   MemInfo `json:"mem"`
	Swap     MemInfo `json:"swap"`
	UptimeSec int64  `json:"uptimeSec"`
}

// MemInfo for memory stats
type MemInfo struct {
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
}

// StorageSummary for dashboard widget
type StorageSummary struct {
	TotalBytes  uint64 `json:"totalBytes"`
	UsedBytes   uint64 `json:"usedBytes"`
	PoolsOnline int    `json:"poolsOnline"`
	PoolsTotal  int    `json:"poolsTotal"`
}

// DisksSummary for dashboard widget
type DisksSummary struct {
	Total       int    `json:"total"`
	Healthy     int    `json:"healthy"`
	Warning     int    `json:"warning"`
	Critical    int    `json:"critical"`
	LastScanISO string `json:"lastScanISO"`
}

// ShareInfo for network shares
type ShareInfo struct {
	Name  string `json:"name"`
	Proto string `json:"proto"` // SMB, NFS, AFP
	Path  string `json:"path"`
	State string `json:"state"` // active, disabled
}

// AppInfo for installed apps
type AppInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Version string `json:"version"`
}

// MaintenanceStatus for scrub/balance operations
type MaintenanceStatus struct {
	Scrub   MaintenanceOp `json:"scrub"`
	Balance MaintenanceOp `json:"balance"`
}

// MaintenanceOp status
type MaintenanceOp struct {
	State   string `json:"state"` // idle, running, scheduled
	NextISO string `json:"nextISO"`
}

// EventInfo for recent activity
type EventInfo struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // info, warning, error
}

// HandleDashboard returns aggregated dashboard data
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
	defer cancel()

	response := DashboardResponse{
		System:      getSystemSummary(ctx),
		Storage:     getStorageSummary(ctx),
		Disks:       getDisksSummary(ctx),
		Shares:      getShares(ctx),
		Apps:        getInstalledApps(ctx),
		Maintenance: getMaintenanceStatus(ctx),
		Events:      getRecentEvents(ctx),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(response)
}

func getSystemSummary(ctx context.Context) SystemSummary {
	summary := SystemSummary{
		Status: "ok",
		Memory: MemInfo{},
		Swap:   MemInfo{},
	}

	// CPU usage (with timeout)
	cpuChan := make(chan float64, 1)
	go func() {
		if percents, err := cpu.Percent(50*time.Millisecond, false); err == nil && len(percents) > 0 {
			cpuChan <- percents[0]
		} else {
			cpuChan <- 0
		}
	}()

	select {
	case summary.CPUPct = <-cpuChan:
	case <-ctx.Done():
		summary.CPUPct = 0
	}

	// Load average
	if l, err := load.Avg(); err == nil {
		summary.Load1 = l.Load1
	}

	// Memory
	if m, err := mem.VirtualMemory(); err == nil {
		summary.Memory.Used = m.Used
		summary.Memory.Total = m.Total
	}

	// Swap
	if s, err := mem.SwapMemory(); err == nil {
		summary.Swap.Used = s.Used
		summary.Swap.Total = s.Total
	}

	// Uptime
	if uptime, err := host.Uptime(); err == nil {
		summary.UptimeSec = int64(uptime)
	}

	// Determine health status
	if summary.CPUPct > 90 || (summary.Memory.Total > 0 && float64(summary.Memory.Used)/float64(summary.Memory.Total) > 0.9) {
		summary.Status = "critical"
	} else if summary.CPUPct > 70 || (summary.Memory.Total > 0 && float64(summary.Memory.Used)/float64(summary.Memory.Total) > 0.7) {
		summary.Status = "degraded"
	}

	return summary
}

func getStorageSummary(ctx context.Context) StorageSummary {
	summary := StorageSummary{
		PoolsOnline: 1, // Default to 1 for now
		PoolsTotal:  1,
	}

	// Get disk usage for all mount points
	partitions, err := disk.Partitions(false)
	if err != nil {
		return summary
	}

	for _, partition := range partitions {
		// Skip special filesystems
		if partition.Fstype == "tmpfs" || partition.Fstype == "devtmpfs" {
			continue
		}

		if usage, err := disk.Usage(partition.Mountpoint); err == nil {
			summary.TotalBytes += usage.Total
			summary.UsedBytes += usage.Used
		}
	}

	return summary
}

func getDisksSummary(ctx context.Context) DisksSummary {
	summary := DisksSummary{
		LastScanISO: time.Now().Format(time.RFC3339),
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		return summary
	}

	for _, partition := range partitions {
		// Skip special filesystems
		if partition.Fstype == "tmpfs" || partition.Fstype == "devtmpfs" {
			continue
		}

		summary.Total++

		// Check usage to determine health
		if usage, err := disk.Usage(partition.Mountpoint); err == nil {
			if usage.UsedPercent > 90 {
				summary.Critical++
			} else if usage.UsedPercent > 80 {
				summary.Warning++
			} else {
				summary.Healthy++
			}
		} else {
			summary.Healthy++ // Assume healthy if can't check
		}
	}

	return summary
}

func getShares(ctx context.Context) []ShareInfo {
	// Return mock data for now - would integrate with actual shares system
	return []ShareInfo{
		{
			Name:  "Documents",
			Proto: "SMB",
			Path:  "/mnt/pool/documents",
			State: "active",
		},
		{
			Name:  "Media",
			Proto: "SMB",
			Path:  "/mnt/pool/media",
			State: "active",
		},
	}
}

func getInstalledApps(ctx context.Context) []AppInfo {
	// Return mock data for now - would integrate with actual apps system
	return []AppInfo{
		{
			ID:      "plex",
			Name:    "Plex Media Server",
			State:   "running",
			Version: "1.32.8",
		},
		{
			ID:      "nextcloud",
			Name:    "Nextcloud",
			State:   "running",
			Version: "28.0.1",
		},
	}
}

func getMaintenanceStatus(ctx context.Context) MaintenanceStatus {
	// Return default idle status - would integrate with actual maintenance system
	nextWeek := time.Now().Add(7 * 24 * time.Hour)
	return MaintenanceStatus{
		Scrub: MaintenanceOp{
			State:   "idle",
			NextISO: nextWeek.Format(time.RFC3339),
		},
		Balance: MaintenanceOp{
			State:   "idle",
			NextISO: nextWeek.Add(24 * time.Hour).Format(time.RFC3339),
		},
	}
}

func getRecentEvents(ctx context.Context) []EventInfo {
	// Return recent events - would integrate with actual event system
	now := time.Now()
	events := []EventInfo{
		{
			ID:        "evt_001",
			Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339),
			Type:      "system",
			Message:   "System health check completed",
			Severity:  "info",
		},
		{
			ID:        "evt_002",
			Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339),
			Type:      "storage",
			Message:   "Pool scrub scheduled",
			Severity:  "info",
		},
		{
			ID:        "evt_003",
			Timestamp: now.Add(-1 * time.Hour).Format(time.RFC3339),
			Type:      "app",
			Message:   "Plex Media Server updated to 1.32.8",
			Severity:  "info",
		},
	}
	return events
}

// Individual endpoint handlers for granular access

// HandleStorageSummary returns storage summary
func HandleStorageSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
	defer cancel()

	summary := getStorageSummary(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// HandleDisksSummary returns disks summary
func HandleDisksSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
	defer cancel()

	summary := getDisksSummary(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// HandleRecentEvents returns recent events
func HandleRecentEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
	defer cancel()

	events := getRecentEvents(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// HandleMaintenanceStatus returns maintenance status
func HandleMaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
	defer cancel()

	status := getMaintenanceStatus(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
