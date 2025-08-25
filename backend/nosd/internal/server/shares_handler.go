package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Share represents a network share
type Share struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Protocol    string   `json:"protocol"` // smb, nfs, afp
	Enabled     bool     `json:"enabled"`
	GuestOk     bool     `json:"guestOk,omitempty"`
	ReadOnly    bool     `json:"readOnly,omitempty"`
	Users       []string `json:"users,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	ModifiedAt  string   `json:"modifiedAt,omitempty"`
}

// SharesHandlerV1 handles share-related endpoints for API v1
type SharesHandlerV1 struct {
	// In a real implementation, this would interface with Samba/NFS configs
	shares []Share
}

// NewSharesHandlerV1 creates a new shares handler for API v1
func NewSharesHandlerV1() *SharesHandlerV1 {
	now := time.Now().Format(time.RFC3339)

	return &SharesHandlerV1{
		shares: []Share{
			{
				Name:        "Documents",
				Path:        "/mnt/main/documents",
				Protocol:    "smb",
				Enabled:     true,
				GuestOk:     false,
				ReadOnly:    false,
				Users:       []string{"admin", "user1"},
				Groups:      []string{"staff"},
				Description: "Shared documents and files",
				CreatedAt:   now,
				ModifiedAt:  now,
			},
			{
				Name:        "Media",
				Path:        "/mnt/main/media",
				Protocol:    "smb",
				Enabled:     true,
				GuestOk:     true,
				ReadOnly:    true,
				Description: "Media library",
				CreatedAt:   now,
				ModifiedAt:  now,
			},
			{
				Name:        "TimeMachine",
				Path:        "/mnt/main/timemachine",
				Protocol:    "afp",
				Enabled:     true,
				GuestOk:     false,
				ReadOnly:    false,
				Users:       []string{"mac_user"},
				Description: "Time Machine backup target",
				CreatedAt:   now,
				ModifiedAt:  now,
			},
		},
	}
}

// Routes registers the shares routes
func (h *SharesHandlerV1) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetShares)
	r.Post("/", h.CreateShare)
	r.Get("/{name}", h.GetShare)
	r.Put("/{name}", h.UpdateShare)
	r.Delete("/{name}", h.DeleteShare)
	r.Post("/{name}/test", h.TestShare)

	return r
}

// GetShares returns all shares
// GET /api/v1/shares
func (h *SharesHandlerV1) GetShares(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.shares); err != nil {
		log.Error().Err(err).Msg("Failed to encode shares")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetShare returns a specific share
// GET /api/v1/shares/{name}
func (h *SharesHandlerV1) GetShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	for _, share := range h.shares {
		if share.Name == name {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(share)
			return
		}
	}

	http.Error(w, "Share not found", http.StatusNotFound)
}

// CreateShare creates a new share
// POST /api/v1/shares
func (h *SharesHandlerV1) CreateShare(w http.ResponseWriter, r *http.Request) {
	var share Share
	if err := json.NewDecoder(r.Body).Decode(&share); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if share already exists
	for _, existing := range h.shares {
		if existing.Name == share.Name {
			http.Error(w, "Share already exists", http.StatusConflict)
			return
		}
	}

	// Validate protocol
	validProtocols := map[string]bool{
		"smb": true,
		"nfs": true,
		"afp": true,
	}

	if !validProtocols[share.Protocol] {
		http.Error(w, "Invalid protocol", http.StatusBadRequest)
		return
	}

	// Set timestamps
	now := time.Now().Format(time.RFC3339)
	share.CreatedAt = now
	share.ModifiedAt = now

	// Add to shares
	h.shares = append(h.shares, share)

	// In real implementation, this would update Samba/NFS configs
	log.Info().Str("name", share.Name).Str("protocol", share.Protocol).Msg("Created share")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(share)
}

// UpdateShare updates an existing share
// PUT /api/v1/shares/{name}
func (h *SharesHandlerV1) UpdateShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var updates Share
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for i, share := range h.shares {
		if share.Name == name {
			// Update fields
			if updates.Path != "" {
				h.shares[i].Path = updates.Path
			}
			if updates.Protocol != "" {
				h.shares[i].Protocol = updates.Protocol
			}
			h.shares[i].Enabled = updates.Enabled
			h.shares[i].GuestOk = updates.GuestOk
			h.shares[i].ReadOnly = updates.ReadOnly
			if updates.Users != nil {
				h.shares[i].Users = updates.Users
			}
			if updates.Groups != nil {
				h.shares[i].Groups = updates.Groups
			}
			if updates.Description != "" {
				h.shares[i].Description = updates.Description
			}
			h.shares[i].ModifiedAt = time.Now().Format(time.RFC3339)

			// In real implementation, this would update Samba/NFS configs
			log.Info().Str("name", name).Msg("Updated share")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(h.shares[i])
			return
		}
	}

	http.Error(w, "Share not found", http.StatusNotFound)
}

// DeleteShare deletes a share
// DELETE /api/v1/shares/{name}
func (h *SharesHandlerV1) DeleteShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	for i, share := range h.shares {
		if share.Name == name {
			// Remove from slice
			h.shares = append(h.shares[:i], h.shares[i+1:]...)

			// In real implementation, this would update Samba/NFS configs
			log.Info().Str("name", name).Msg("Deleted share")

			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	http.Error(w, "Share not found", http.StatusNotFound)
}

// TestShare tests share configuration
// POST /api/v1/shares/{name}/test
func (h *SharesHandlerV1) TestShare(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Find the share
	var share *Share
	for _, s := range h.shares {
		if s.Name == name {
			share = &s
			break
		}
	}

	if share == nil {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	// In real implementation, this would test SMB/NFS connectivity
	// For now, just return success

	result := map[string]interface{}{
		"status":  "success",
		"message": "Share configuration is valid",
		"tests": map[string]interface{}{
			"path_exists":     true,
			"permissions_ok":  true,
			"service_running": true,
		},
	}

	log.Info().Str("name", name).Msg("Tested share configuration")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
