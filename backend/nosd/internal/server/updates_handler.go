package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"nithronos/backend/nosd/pkg/updates"
)

// UpdatesHandler handles update-related API endpoints
type UpdatesHandler struct {
	updater *updates.Updater
	logger  zerolog.Logger
}

// NewUpdatesHandler creates a new updates handler
func NewUpdatesHandler(logger zerolog.Logger) (*UpdatesHandler, error) {
	// Default configuration
	config := &updates.UpdateConfig{
		Channel:           updates.ChannelStable,
		AutoCheck:         true,
		AutoCheckInterval: 24 * time.Hour,
		AutoApply:         false,
		SnapshotRetention: 3,
		RepoURL:           "https://apt.nithronos.com",
		GPGKeyID:          "nithronos-release",
		Telemetry:         false,
	}

	// Create updater
	updater, err := updates.NewUpdater(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	return &UpdatesHandler{
		updater: updater,
		logger:  logger,
	}, nil
}

// Routes returns the update routes
func (h *UpdatesHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Version and channel
	r.Get("/version", h.GetVersion)
	r.Get("/channel", h.GetChannel)
	r.Post("/channel", h.SetChannel)

	// Update operations
	r.Get("/check", h.CheckForUpdates)
	r.Post("/apply", h.ApplyUpdate)
	r.Get("/progress", h.GetProgress)
	r.Post("/rollback", h.Rollback)

	// Snapshots
	r.Get("/snapshots", h.ListSnapshots)
	r.Delete("/snapshots/{id}", h.DeleteSnapshot)

	// Progress stream (SSE)
	r.Get("/progress/stream", h.StreamProgress)

	return r
}

// GetVersion returns the current system version
// GET /api/v1/updates/version
func (h *UpdatesHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	version, err := h.updater.GetVersion()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get version")
		h.writeError(w, http.StatusInternalServerError, "Failed to get version")
		return
	}

	h.writeJSON(w, version)
}

// GetChannel returns the current update channel
// GET /api/v1/updates/channel
func (h *UpdatesHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	config := h.updater.GetConfig()
	h.writeJSON(w, map[string]string{
		"channel": string(config.Channel),
	})
}

// SetChannel changes the update channel
// POST /api/v1/updates/channel
func (h *UpdatesHandler) SetChannel(w http.ResponseWriter, r *http.Request) {
	var req updates.ChannelChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate channel
	if req.Channel != updates.ChannelStable && req.Channel != updates.ChannelBeta {
		h.writeError(w, http.StatusBadRequest, "Invalid channel")
		return
	}

	// Set channel
	if err := h.updater.SetChannel(req.Channel); err != nil {
		h.logger.Error().Err(err).Msg("Failed to set channel")
		h.writeError(w, http.StatusInternalServerError, "Failed to set channel")
		return
	}

	h.logger.Info().Str("channel", string(req.Channel)).Msg("Update channel changed")

	h.writeJSON(w, map[string]string{
		"status":  "success",
		"channel": string(req.Channel),
	})
}

// CheckForUpdates checks for available updates
// GET /api/v1/updates/check
func (h *UpdatesHandler) CheckForUpdates(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Msg("Checking for updates")

	response, err := h.updater.CheckForUpdates()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to check for updates")
		h.writeError(w, http.StatusInternalServerError, "Failed to check for updates")
		return
	}

	h.logger.Info().
		Bool("update_available", response.UpdateAvailable).
		Msg("Update check complete")

	h.writeJSON(w, response)
}

// ApplyUpdate applies an available update
// POST /api/v1/updates/apply
func (h *UpdatesHandler) ApplyUpdate(w http.ResponseWriter, r *http.Request) {
	var req updates.UpdateApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is OK, use defaults
		req = updates.UpdateApplyRequest{}
	}

	h.logger.Info().
		Str("version", req.Version).
		Bool("skip_snapshot", req.SkipSnapshot).
		Msg("Applying update")

	// Apply update
	if err := h.updater.ApplyUpdate(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to apply update")
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "started",
		"message": "Update process started",
	})
}

// GetProgress returns the current update progress
// GET /api/v1/updates/progress
func (h *UpdatesHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	progress := h.updater.GetProgress()
	h.writeJSON(w, progress)
}

// StreamProgress streams update progress via Server-Sent Events
// GET /api/v1/updates/progress/stream
func (h *UpdatesHandler) StreamProgress(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Create event channel
	progressChan := h.updater.GetProgressChannel()
	defer h.updater.ReleaseProgressChannel(progressChan)

	// Create ticker for keepalive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Flusher for SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Send initial progress
	if progress := h.updater.GetProgress(); progress != nil {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Stream updates
	for {
		select {
		case <-r.Context().Done():
			return

		case progress := <-progressChan:
			data, _ := json.Marshal(progress)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Stop streaming if update is complete
			if progress.State == updates.UpdateStateSuccess ||
				progress.State == updates.UpdateStateFailed ||
				progress.State == updates.UpdateStateRolledBack {
				return
			}

		case <-ticker.C:
			// Send keepalive ping
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// Rollback performs a rollback to a previous snapshot
// POST /api/v1/updates/rollback
func (h *UpdatesHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	var req updates.RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SnapshotID == "" {
		h.writeError(w, http.StatusBadRequest, "Snapshot ID required")
		return
	}

	h.logger.Info().
		Str("snapshot_id", req.SnapshotID).
		Bool("force", req.Force).
		Msg("Rolling back to snapshot")

	// Perform rollback
	if err := h.updater.Rollback(&req); err != nil {
		h.logger.Error().Err(err).Msg("Rollback failed")
		h.writeError(w, http.StatusInternalServerError, "Rollback failed")
		return
	}

	h.logger.Info().Msg("Rollback completed successfully")

	h.writeJSON(w, map[string]string{
		"status":  "success",
		"message": "System rolled back successfully",
	})
}

// ListSnapshots returns all available update snapshots
// GET /api/v1/updates/snapshots
func (h *UpdatesHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := h.updater.ListSnapshots()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list snapshots")
		h.writeError(w, http.StatusInternalServerError, "Failed to list snapshots")
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"snapshots": snapshots,
		"total":     len(snapshots),
	})
}

// DeleteSnapshot deletes a snapshot
// DELETE /api/v1/updates/snapshots/{id}
func (h *UpdatesHandler) DeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := chi.URLParam(r, "id")
	if snapshotID == "" {
		h.writeError(w, http.StatusBadRequest, "Snapshot ID required")
		return
	}

	h.logger.Info().Str("snapshot_id", snapshotID).Msg("Deleting snapshot")

	if err := h.updater.DeleteSnapshot(snapshotID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete snapshot")
		h.writeError(w, http.StatusInternalServerError, "Failed to delete snapshot")
		return
	}

	h.logger.Info().Msg("Snapshot deleted successfully")

	h.writeJSON(w, map[string]string{
		"status":      "deleted",
		"snapshot_id": snapshotID,
	})
}

// Helper methods

func (h *UpdatesHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
	}
}

func (h *UpdatesHandler) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		fmt.Printf("Failed to write error response: %v\n", err)
	}
}
