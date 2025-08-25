package server

import (
	"encoding/json"
	"net/http"

	"github.com/NotTekk/nosd/pkg/agentclient"
	"github.com/NotTekk/nosd/pkg/shares"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// SharesHandler handles share-related HTTP requests
type SharesHandler struct {
	manager *shares.Manager
	agent   *agentclient.Client
}

// NewSharesHandler creates a new shares handler
func NewSharesHandler(manager *shares.Manager, agent *agentclient.Client) *SharesHandler {
	return &SharesHandler{
		manager: manager,
		agent:   agent,
	}
}

// Routes registers share routes
func (h *SharesHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListShares)
	r.Post("/", h.CreateShare)
	r.Route("/{name}", func(r chi.Router) {
		r.Patch("/", h.UpdateShare)
		r.Delete("/", h.DeleteShare)
		r.Post("/test", h.TestShare)
	})

	return r
}

// ListShares handles GET /api/shares
func (h *SharesHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	sharesList, err := h.manager.List()
	if err != nil {
		log.Error().Err(err).Msg("failed to list shares")
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to list shares")
		return
	}

	writeJSON(w, http.StatusOK, sharesList)
}

// CreateShare handles POST /api/shares
func (h *SharesHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	var req shares.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_JSON", "Invalid request body")
		return
	}

	// Create share in manager (validates and persists)
	share, err := h.manager.Create(&req)
	if err != nil {
		if shareErr, ok := err.(*shares.Error); ok {
			writeError(w, http.StatusBadRequest, string(shareErr.Code), shareErr.Message)
			return
		}
		log.Error().Err(err).Msg("failed to create share")
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to create share")
		return
	}

	// Apply share via agent (privileged operations)
	if err := h.applyShare(share); err != nil {
		// Rollback
		_ = h.manager.Delete(share.Name)
		log.Error().Err(err).Str("share", share.Name).Msg("failed to apply share")
		writeError(w, http.StatusInternalServerError, "ERR_APPLY_FAILED", "Failed to apply share configuration")
		return
	}

	writeJSON(w, http.StatusCreated, share)
}

// UpdateShare handles PATCH /api/shares/:name
func (h *SharesHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req shares.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_JSON", "Invalid request body")
		return
	}

	// Update share in manager
	share, err := h.manager.Update(name, &req)
	if err != nil {
		if shareErr, ok := err.(*shares.Error); ok {
			status := http.StatusBadRequest
			if shareErr.Code == shares.ErrCodeNotFound {
				status = http.StatusNotFound
			}
			writeError(w, status, string(shareErr.Code), shareErr.Message)
			return
		}
		log.Error().Err(err).Msg("failed to update share")
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to update share")
		return
	}

	// Apply updated configuration
	if err := h.applyShare(share); err != nil {
		log.Error().Err(err).Str("share", share.Name).Msg("failed to apply share update")
		writeError(w, http.StatusInternalServerError, "ERR_APPLY_FAILED", "Failed to apply share configuration")
		return
	}

	writeJSON(w, http.StatusOK, share)
}

// DeleteShare handles DELETE /api/shares/:name
func (h *SharesHandler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Get share details before deletion
	share, err := h.manager.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, "ERR_NOT_FOUND", "Share not found")
		return
	}

	// Remove share configuration via agent
	if err := h.removeShare(share); err != nil {
		log.Error().Err(err).Str("share", name).Msg("failed to remove share")
		writeError(w, http.StatusInternalServerError, "ERR_REMOVE_FAILED", "Failed to remove share configuration")
		return
	}

	// Delete from manager
	if err := h.manager.Delete(name); err != nil {
		if shareErr, ok := err.(*shares.Error); ok {
			status := http.StatusBadRequest
			if shareErr.Code == shares.ErrCodeNotFound {
				status = http.StatusNotFound
			}
			writeError(w, status, string(shareErr.Code), shareErr.Message)
			return
		}
		log.Error().Err(err).Msg("failed to delete share")
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to delete share")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestShare handles POST /api/shares/:name/test
func (h *SharesHandler) TestShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req shares.TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_JSON", "Invalid request body")
		return
	}

	result, err := h.manager.Test(name, req.Config)
	if err != nil {
		log.Error().Err(err).Msg("failed to test share configuration")
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to test configuration")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// applyShare applies share configuration via agent
func (h *SharesHandler) applyShare(share *shares.Share) error {
	// Create share directory and set permissions
	req := &agentclient.CreateShareRequest{
		Path:    share.Path,
		Name:    share.Name,
		Owners:  share.Owners,
		Readers: share.Readers,
	}

	if share.SMB != nil && share.SMB.Recycle != nil && share.SMB.Recycle.Enabled {
		req.RecycleDir = share.SMB.Recycle.Directory
	}

	if err := h.agent.CreateShare(req); err != nil {
		return err
	}

	// Apply ACLs
	if err := h.agent.ApplyACLs(&agentclient.ApplyACLsRequest{
		Path:    share.Path,
		Owners:  share.Owners,
		Readers: share.Readers,
	}); err != nil {
		return err
	}

	// Configure SMB if enabled
	if share.SMB != nil && share.SMB.Enabled {
		config, err := shares.GenerateSambaConfig(share)
		if err != nil {
			return err
		}

		if err := h.agent.WriteSambaConfig(&agentclient.WriteSambaConfigRequest{
			Name:   share.Name,
			Config: config,
		}); err != nil {
			return err
		}

		if err := h.agent.ReloadSamba(); err != nil {
			return err
		}
	}

	// Configure NFS if enabled
	if share.NFS != nil && share.NFS.Enabled {
		// TODO: Get LAN networks from configuration
		lanNetworks := []string{"192.168.0.0/16", "10.0.0.0/8"}

		config, err := shares.GenerateNFSExport(share, lanNetworks)
		if err != nil {
			return err
		}

		if err := h.agent.WriteNFSExport(&agentclient.WriteNFSExportRequest{
			Name:   share.Name,
			Config: config,
		}); err != nil {
			return err
		}

		if err := h.agent.ReloadNFS(); err != nil {
			return err
		}
	}

	// Update Avahi if Time Machine is involved
	allShares, _ := h.manager.List()
	if err := shares.UpdateAvahiTimeMachine(allShares); err != nil {
		log.Warn().Err(err).Msg("failed to update Avahi Time Machine service")
	} else {
		// Reload Avahi if the file changed
		_ = h.agent.ReloadAvahi()
	}

	return nil
}

// removeShare removes share configuration via agent
func (h *SharesHandler) removeShare(share *shares.Share) error {
	// Remove SMB configuration if it exists
	if share.SMB != nil && share.SMB.Enabled {
		if err := h.agent.RemoveSambaConfig(share.Name); err != nil {
			log.Warn().Err(err).Str("share", share.Name).Msg("failed to remove Samba config")
		}
		_ = h.agent.ReloadSamba()
	}

	// Remove NFS export if it exists
	if share.NFS != nil && share.NFS.Enabled {
		if err := h.agent.RemoveNFSExport(share.Name); err != nil {
			log.Warn().Err(err).Str("share", share.Name).Msg("failed to remove NFS export")
		}
		_ = h.agent.ReloadNFS()
	}

	// Note: We don't remove the directory itself to preserve data
	// Admin can manually remove if needed

	return nil
}
