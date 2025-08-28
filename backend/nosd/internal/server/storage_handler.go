package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Pool represents a storage pool
type Pool struct {
	ID         string       `json:"id"`
	UUID       string       `json:"uuid"`
	Label      string       `json:"label,omitempty"`
	Mountpoint string       `json:"mountpoint"`
	Size       uint64       `json:"size"`
	Used       uint64       `json:"used"`
	Free       uint64       `json:"free"`
	Raid       string       `json:"raid"`
	Status     string       `json:"status"` // online, degraded, offline
	Devices    []PoolDevice `json:"devices"`
	Subvolumes []Subvolume  `json:"subvolumes,omitempty"`
}

// PoolDevice represents a device in a pool
type PoolDevice struct {
	Path   string `json:"path"`
	Size   uint64 `json:"size"`
	Used   uint64 `json:"used,omitempty"`
	Status string `json:"status,omitempty"`
}

// Subvolume represents a Btrfs subvolume
type Subvolume struct {
	ID   string  `json:"id"`
	Path string  `json:"path"`
	Size *uint64 `json:"size,omitempty"`
}

// PoolSummary represents a summary of all pools
type PoolSummary struct {
	TotalPools    int    `json:"totalPools"`
	TotalSize     uint64 `json:"totalSize"`
	TotalUsed     uint64 `json:"totalUsed"`
	PoolsOnline   int    `json:"poolsOnline"`
	PoolsDegraded int    `json:"poolsDegraded"`
}

// Device represents a storage device
type Device struct {
	Path   string `json:"path"`
	Model  string `json:"model,omitempty"`
	Serial string `json:"serial,omitempty"`
	Size   uint64 `json:"size"`
	Type   string `json:"type,omitempty"` // ssd, hdd
	InUse  bool   `json:"inUse"`
	Pool   string `json:"pool,omitempty"`
}

// ScrubStatus represents the status of a scrub operation
type ScrubStatus struct {
	PoolID       string  `json:"poolId"`
	Status       string  `json:"status"` // idle, running, paused, finished, cancelled
	Progress     *int    `json:"progress,omitempty"`
	BytesScanned *uint64 `json:"bytesScanned,omitempty"`
	BytesTotal   *uint64 `json:"bytesTotal,omitempty"`
	ErrorsFixed  *int    `json:"errorsFixed,omitempty"`
	StartedAt    *string `json:"startedAt,omitempty"`
	FinishedAt   *string `json:"finishedAt,omitempty"`
	NextRun      *string `json:"nextRun,omitempty"`
}

// BalanceStatus represents the status of a balance operation
type BalanceStatus struct {
	PoolID        string  `json:"poolId"`
	Status        string  `json:"status"` // idle, running, paused, finished, cancelled
	Progress      *int    `json:"progress,omitempty"`
	BytesBalanced *uint64 `json:"bytesBalanced,omitempty"`
	BytesTotal    *uint64 `json:"bytesTotal,omitempty"`
	StartedAt     *string `json:"startedAt,omitempty"`
	FinishedAt    *string `json:"finishedAt,omitempty"`
}

// StorageHandler handles storage-related endpoints
type StorageHandler struct {
	agentClient AgentClient
}

// NewStorageHandler creates a new storage handler
func NewStorageHandler(agentClient AgentClient) *StorageHandler {
	return &StorageHandler{
		agentClient: agentClient,
	}
}

// Routes registers the storage routes
func (h *StorageHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Pool endpoints
	r.Get("/pools", h.GetPools)
	r.Get("/pools/{uuid}", h.GetPool)
	r.Get("/pools/{uuid}/subvols", h.GetPoolSubvolumes)
	r.Get("/pools/{uuid}/options", h.GetPoolMountOptions)
	r.Put("/pools/{uuid}/options", h.SetPoolMountOptions)

	// Device endpoints
	r.Get("/devices", h.GetDevices)

	return r
}

// GetPools returns all storage pools or a summary
// GET /api/v1/storage/pools?summary=1
func (h *StorageHandler) GetPools(w http.ResponseWriter, r *http.Request) {
	summary := r.URL.Query().Get("summary") == "1"

	// Get pool data (in real implementation, this would use btrfs fi show, etc.)
	pools := h.getPools()

	if summary {
		// Return summary
		poolSummary := PoolSummary{
			TotalPools:    len(pools),
			TotalSize:     0,
			TotalUsed:     0,
			PoolsOnline:   0,
			PoolsDegraded: 0,
		}

		for _, pool := range pools {
			poolSummary.TotalSize += pool.Size
			poolSummary.TotalUsed += pool.Used

			switch pool.Status {
			case "online":
				poolSummary.PoolsOnline++
			case "degraded":
				poolSummary.PoolsDegraded++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(poolSummary); err != nil {
			fmt.Printf("Failed to write response: %v\n", err)
		}
	} else {
		// Return full pool list
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(pools); err != nil {
			fmt.Printf("Failed to write response: %v\n", err)
		}
	}
}

// GetPool returns a specific pool by UUID
// GET /api/v1/storage/pools/{uuid}
func (h *StorageHandler) GetPool(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	pools := h.getPools()
	for _, pool := range pools {
		if pool.UUID == uuid || pool.ID == uuid {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pool)
			return
		}
	}

	http.Error(w, "Pool not found", http.StatusNotFound)
}

// GetPoolSubvolumes returns subvolumes for a pool
// GET /api/v1/storage/pools/{uuid}/subvols
func (h *StorageHandler) GetPoolSubvolumes(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	// In real implementation, this would use btrfs subvolume list
	subvols := []Subvolume{
		{
			ID:   "256",
			Path: "@",
		},
		{
			ID:   "257",
			Path: "@home",
		},
		{
			ID:   "258",
			Path: "@snapshots",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subvols)

	// Log for debugging
	log.Info().Str("uuid", uuid).Msg("Returned subvolumes for pool")
}

// GetPoolMountOptions returns mount options for a pool
// GET /api/v1/storage/pools/{uuid}/options
func (h *StorageHandler) GetPoolMountOptions(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	// In real implementation, this would read from /proc/mounts or findmnt
	options := map[string]string{
		"mountOptions": "compress=zstd:3,noatime,ssd,discard=async",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)

	log.Info().Str("uuid", uuid).Msg("Returned mount options for pool")
}

// SetPoolMountOptions updates mount options for a pool
// PUT /api/v1/storage/pools/{uuid}/options
func (h *StorageHandler) SetPoolMountOptions(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	var req struct {
		MountOptions string `json:"mountOptions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In real implementation, this would update /etc/fstab and remount
	response := map[string]interface{}{
		"ok":             true,
		"mountOptions":   req.MountOptions,
		"rebootRequired": false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Info().Str("uuid", uuid).Str("options", req.MountOptions).Msg("Updated mount options for pool")
}

// GetDevices returns all storage devices
// GET /api/v1/storage/devices
func (h *StorageHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices := h.getDevices()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// Helper function to get pools (mock data for now)
func (h *StorageHandler) getPools() []Pool {
	// In real implementation, this would use btrfs commands
	return []Pool{
		{
			ID:         "main-pool",
			UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Label:      "main",
			Mountpoint: "/mnt/main",
			Size:       2000000000000, // 2TB
			Used:       1200000000000, // 1.2TB
			Free:       800000000000,  // 800GB
			Raid:       "raid1",
			Status:     "online",
			Devices: []PoolDevice{
				{
					Path:   "/dev/sda",
					Size:   1000000000000,
					Used:   600000000000,
					Status: "online",
				},
				{
					Path:   "/dev/sdb",
					Size:   1000000000000,
					Used:   600000000000,
					Status: "online",
				},
			},
		},
	}
}

// Helper function to get devices (mock data for now)
func (h *StorageHandler) getDevices() []Device {
	// In real implementation, this would use lsblk, smartctl, etc.
	return []Device{
		{
			Path:   "/dev/sda",
			Model:  "Samsung SSD 870",
			Serial: "S5Y2NF0NA12345",
			Size:   1000000000000,
			Type:   "ssd",
			InUse:  true,
			Pool:   "main-pool",
		},
		{
			Path:   "/dev/sdb",
			Model:  "Samsung SSD 870",
			Serial: "S5Y2NF0NA12346",
			Size:   1000000000000,
			Type:   "ssd",
			InUse:  true,
			Pool:   "main-pool",
		},
		{
			Path:   "/dev/sdc",
			Model:  "WD Red Plus",
			Serial: "WX21A23C4567",
			Size:   4000000000000,
			Type:   "hdd",
			InUse:  false,
		},
		{
			Path:   "/dev/sdd",
			Model:  "WD Red Plus",
			Serial: "WX21A23C4568",
			Size:   4000000000000,
			Type:   "hdd",
			InUse:  false,
		},
	}
}

// BtrfsHandler handles Btrfs-specific endpoints
type BtrfsHandler struct {
	agentClient AgentClient
}

// NewBtrfsHandler creates a new Btrfs handler
func NewBtrfsHandler(agentClient AgentClient) *BtrfsHandler {
	return &BtrfsHandler{
		agentClient: agentClient,
	}
}

// Routes registers the Btrfs routes
func (h *BtrfsHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Scrub endpoints
	r.Get("/scrub/status", h.GetScrubStatus)
	r.Post("/scrub/start", h.StartScrub)
	r.Post("/scrub/cancel", h.CancelScrub)

	// Balance endpoints
	r.Get("/balance/status", h.GetBalanceStatus)
	r.Post("/balance/start", h.StartBalance)
	r.Post("/balance/cancel", h.CancelBalance)

	return r
}

// GetScrubStatus returns scrub status for all pools
// GET /api/v1/btrfs/scrub/status
func (h *BtrfsHandler) GetScrubStatus(w http.ResponseWriter, r *http.Request) {
	// In real implementation, this would parse btrfs scrub status
	status := []ScrubStatus{
		{
			PoolID:  "main-pool",
			Status:  "idle",
			NextRun: strPtr("2024-01-07T03:00:00Z"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartScrub starts a scrub operation
// POST /api/v1/btrfs/scrub/start
func (h *BtrfsHandler) StartScrub(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID string `json:"poolId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In real implementation, this would call btrfs scrub start via nos-agent
	log.Info().Str("pool", req.PoolID).Msg("Starting scrub")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "started",
		"poolId": req.PoolID,
	})
}

// CancelScrub cancels a running scrub operation
// POST /api/v1/btrfs/scrub/cancel
func (h *BtrfsHandler) CancelScrub(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID string `json:"poolId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In real implementation, this would call btrfs scrub cancel via nos-agent
	log.Info().Str("pool", req.PoolID).Msg("Cancelling scrub")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "cancelled",
		"poolId": req.PoolID,
	})
}

// GetBalanceStatus returns balance status for all pools
// GET /api/v1/btrfs/balance/status
func (h *BtrfsHandler) GetBalanceStatus(w http.ResponseWriter, r *http.Request) {
	// In real implementation, this would parse btrfs balance status
	status := []BalanceStatus{
		{
			PoolID: "main-pool",
			Status: "idle",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartBalance starts a balance operation
// POST /api/v1/btrfs/balance/start
func (h *BtrfsHandler) StartBalance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID string `json:"poolId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In real implementation, this would call btrfs balance start via nos-agent
	log.Info().Str("pool", req.PoolID).Msg("Starting balance")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "started",
		"poolId": req.PoolID,
	})
}

// CancelBalance cancels a running balance operation
// POST /api/v1/btrfs/balance/cancel
func (h *BtrfsHandler) CancelBalance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID string `json:"poolId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In real implementation, this would call btrfs balance cancel via nos-agent
	log.Info().Str("pool", req.PoolID).Msg("Cancelling balance")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "cancelled",
		"poolId": req.PoolID,
	})
}
