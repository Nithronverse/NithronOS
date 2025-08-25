package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// SharesHandler handles privileged share operations
type SharesHandler struct{}

// NewSharesHandler creates a new shares handler
func NewSharesHandler() *SharesHandler {
	return &SharesHandler{}
}

// Register registers share routes
func (h *SharesHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/shares/create", h.CreateShare)
	mux.HandleFunc("/shares/acls", h.ApplyACLs)
	mux.HandleFunc("/shares/samba/write", h.WriteSambaConfig)
	mux.HandleFunc("/shares/samba/remove", h.RemoveSambaConfig)
	mux.HandleFunc("/shares/samba/reload", h.ReloadSamba)
	mux.HandleFunc("/shares/nfs/write", h.WriteNFSExport)
	mux.HandleFunc("/shares/nfs/remove", h.RemoveNFSExport)
	mux.HandleFunc("/shares/nfs/reload", h.ReloadNFS)
	mux.HandleFunc("/shares/avahi/reload", h.ReloadAvahi)
	mux.HandleFunc("/shares/subvol", h.CreateSubvol)
	mux.HandleFunc("/shares/group", h.EnsureGroup)
}

// CreateShareRequest represents a request to create a share directory
type CreateShareRequest struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Owners     []string `json:"owners"`
	Readers    []string `json:"readers"`
	Mode       uint32   `json:"mode,omitempty"`
	RecycleDir string   `json:"recycle_dir,omitempty"`
}

// CreateShare creates a share directory with proper permissions
func (h *SharesHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate path is under /srv/shares
	if !strings.HasPrefix(req.Path, "/srv/shares/") {
		http.Error(w, "Invalid share path", http.StatusBadRequest)
		return
	}

	// Set default mode
	if req.Mode == 0 {
		req.Mode = 02770 // setgid + rwxrwx---
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(req.Path), 0755); err != nil {
		log.Error().Err(err).Str("path", req.Path).Msg("failed to create parent directory")
		http.Error(w, "Failed to create parent directory", http.StatusInternalServerError)
		return
	}

	// Check if path exists and is Btrfs
	if isBtrfs(filepath.Dir(req.Path)) {
		// Try to create as Btrfs subvolume
		cmd := exec.Command("btrfs", "subvolume", "create", req.Path)
		if err := cmd.Run(); err != nil {
			// Fall back to regular directory
			if err := os.MkdirAll(req.Path, os.FileMode(req.Mode)); err != nil {
				log.Error().Err(err).Str("path", req.Path).Msg("failed to create directory")
				http.Error(w, "Failed to create share directory", http.StatusInternalServerError)
				return
			}
		} else {
			// Set permissions on subvolume
			if err := os.Chmod(req.Path, os.FileMode(req.Mode)); err != nil {
				log.Error().Err(err).Str("path", req.Path).Msg("failed to set permissions")
				http.Error(w, "Failed to set permissions", http.StatusInternalServerError)
				return
			}
		}
	} else {
		// Create regular directory
		if err := os.MkdirAll(req.Path, os.FileMode(req.Mode)); err != nil {
			log.Error().Err(err).Str("path", req.Path).Msg("failed to create directory")
			http.Error(w, "Failed to create share directory", http.StatusInternalServerError)
			return
		}
	}

	// Create group for this share
	groupName := fmt.Sprintf("nos-share-%s", req.Name)
	if err := h.ensureGroup(groupName); err != nil {
		log.Warn().Err(err).Str("group", groupName).Msg("failed to create share group")
	}

	// Set ownership to root:root (ACLs will handle actual permissions)
	if err := os.Chown(req.Path, 0, 0); err != nil {
		log.Error().Err(err).Str("path", req.Path).Msg("failed to set ownership")
		http.Error(w, "Failed to set ownership", http.StatusInternalServerError)
		return
	}

	// Create recycle directory if requested
	if req.RecycleDir != "" {
		recyclePath := filepath.Join(req.Path, req.RecycleDir)
		if err := os.MkdirAll(recyclePath, 0770); err != nil {
			log.Warn().Err(err).Str("path", recyclePath).Msg("failed to create recycle directory")
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ApplyACLsRequest represents a request to apply POSIX ACLs
type ApplyACLsRequest struct {
	Path    string   `json:"path"`
	Owners  []string `json:"owners"`
	Readers []string `json:"readers"`
}

// ApplyACLs applies POSIX ACLs to a share
func (h *SharesHandler) ApplyACLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApplyACLsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate path
	if !strings.HasPrefix(req.Path, "/srv/shares/") {
		http.Error(w, "Invalid share path", http.StatusBadRequest)
		return
	}

	// Clear existing ACLs (except base)
	cmd := exec.Command("setfacl", "-b", req.Path)
	if err := cmd.Run(); err != nil {
		log.Warn().Err(err).Str("path", req.Path).Msg("failed to clear ACLs")
	}

	// Apply owner ACLs (rwx)
	for _, owner := range req.Owners {
		parts := strings.Split(owner, ":")
		if len(parts) != 2 {
			continue
		}

		aclType := "u"
		if parts[0] == "group" {
			aclType = "g"
		}

		// Set both access and default ACLs
		acl := fmt.Sprintf("%s:%s:rwx", aclType, parts[1])
		cmd := exec.Command("setfacl", "-m", acl, req.Path)
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).Str("acl", acl).Msg("failed to apply owner ACL")
		}

		// Default ACL for new files/dirs
		defaultAcl := fmt.Sprintf("d:%s", acl)
		cmd = exec.Command("setfacl", "-m", defaultAcl, req.Path)
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).Str("acl", defaultAcl).Msg("failed to apply default owner ACL")
		}
	}

	// Apply reader ACLs (rx)
	for _, reader := range req.Readers {
		parts := strings.Split(reader, ":")
		if len(parts) != 2 {
			continue
		}

		aclType := "u"
		if parts[0] == "group" {
			aclType = "g"
		}

		// Set both access and default ACLs
		acl := fmt.Sprintf("%s:%s:rx", aclType, parts[1])
		cmd := exec.Command("setfacl", "-m", acl, req.Path)
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).Str("acl", acl).Msg("failed to apply reader ACL")
		}

		// Default ACL for new files/dirs
		defaultAcl := fmt.Sprintf("d:%s", acl)
		cmd = exec.Command("setfacl", "-m", defaultAcl, req.Path)
		if err := cmd.Run(); err != nil {
			log.Error().Err(err).Str("acl", defaultAcl).Msg("failed to apply default reader ACL")
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// WriteSambaConfigRequest represents a request to write Samba config
type WriteSambaConfigRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// WriteSambaConfig writes a Samba configuration snippet
func (h *SharesHandler) WriteSambaConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WriteSambaConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure directory exists
	confDir := "/etc/samba/smb.conf.d"
	if err := os.MkdirAll(confDir, 0755); err != nil {
		log.Error().Err(err).Str("dir", confDir).Msg("failed to create Samba config directory")
		http.Error(w, "Failed to create config directory", http.StatusInternalServerError)
		return
	}

	// Write config file atomically
	confPath := filepath.Join(confDir, fmt.Sprintf("nos-%s.conf", req.Name))
	tmpPath := confPath + ".tmp"

	if err := os.WriteFile(tmpPath, []byte(req.Config), 0644); err != nil {
		log.Error().Err(err).Str("path", tmpPath).Msg("failed to write Samba config")
		http.Error(w, "Failed to write config", http.StatusInternalServerError)
		return
	}

	// Test configuration
	cmd := exec.Command("testparm", "-s", tmpPath)
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		log.Error().Err(err).Msg("Samba config validation failed")
		http.Error(w, "Invalid Samba configuration", http.StatusBadRequest)
		return
	}

	// Move to final location
	if err := os.Rename(tmpPath, confPath); err != nil {
		os.Remove(tmpPath)
		log.Error().Err(err).Str("path", confPath).Msg("failed to install Samba config")
		http.Error(w, "Failed to install config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// RemoveSambaConfig removes a Samba configuration snippet
func (h *SharesHandler) RemoveSambaConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	confPath := filepath.Join("/etc/samba/smb.conf.d", fmt.Sprintf("nos-%s.conf", req.Name))
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("path", confPath).Msg("failed to remove Samba config")
		http.Error(w, "Failed to remove config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ReloadSamba reloads the Samba service
func (h *SharesHandler) ReloadSamba(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Test configuration first
	cmd := exec.Command("testparm", "-s")
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("Samba config validation failed")
		http.Error(w, "Invalid Samba configuration", http.StatusInternalServerError)
		return
	}

	// Reload service
	cmd = exec.Command("systemctl", "reload", "smbd")
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("failed to reload Samba")
		http.Error(w, "Failed to reload Samba", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// WriteNFSExportRequest represents a request to write NFS export
type WriteNFSExportRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// WriteNFSExport writes an NFS export configuration
func (h *SharesHandler) WriteNFSExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WriteNFSExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure directory exists
	exportDir := "/etc/exports.d"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Error().Err(err).Str("dir", exportDir).Msg("failed to create exports directory")
		http.Error(w, "Failed to create exports directory", http.StatusInternalServerError)
		return
	}

	// Write export file
	exportPath := filepath.Join(exportDir, fmt.Sprintf("nos-%s.exports", req.Name))
	if err := os.WriteFile(exportPath, []byte(req.Config), 0644); err != nil {
		log.Error().Err(err).Str("path", exportPath).Msg("failed to write NFS export")
		http.Error(w, "Failed to write export", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// RemoveNFSExport removes an NFS export configuration
func (h *SharesHandler) RemoveNFSExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	exportPath := filepath.Join("/etc/exports.d", fmt.Sprintf("nos-%s.exports", req.Name))
	if err := os.Remove(exportPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("path", exportPath).Msg("failed to remove NFS export")
		http.Error(w, "Failed to remove export", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ReloadNFS reloads the NFS exports
func (h *SharesHandler) ReloadNFS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Re-export all filesystems
	cmd := exec.Command("exportfs", "-ra")
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("failed to reload NFS exports")
		http.Error(w, "Failed to reload NFS exports", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// CreateSubvol creates a Btrfs subvolume if the filesystem supports it
func (h *SharesHandler) CreateSubvol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate path
	if !strings.HasPrefix(req.Path, "/srv/shares/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Check if parent is Btrfs
	if !isBtrfs(filepath.Dir(req.Path)) {
		http.Error(w, "Parent is not Btrfs", http.StatusBadRequest)
		return
	}

	// Create subvolume
	cmd := exec.Command("btrfs", "subvolume", "create", req.Path)
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Str("path", req.Path).Msg("failed to create subvolume")
		http.Error(w, "Failed to create subvolume", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// EnsureGroup ensures a system group exists
func (h *SharesHandler) EnsureGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.ensureGroup(req.Name); err != nil {
		log.Error().Err(err).Str("group", req.Name).Msg("failed to ensure group")
		http.Error(w, "Failed to ensure group", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ensureGroup ensures a system group exists (internal helper)
func (h *SharesHandler) ensureGroup(name string) error {
	// Check if group exists
	cmd := exec.Command("getent", "group", name)
	if err := cmd.Run(); err == nil {
		return nil // Group already exists
	}

	// Create group
	cmd = exec.Command("groupadd", "-r", name)
	return cmd.Run()
}

// ReloadAvahi reloads the Avahi daemon
func (h *SharesHandler) ReloadAvahi(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reload Avahi daemon
	cmd := exec.Command("systemctl", "reload-or-restart", "avahi-daemon")
	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("failed to reload Avahi")
		http.Error(w, "Failed to reload Avahi", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// isBtrfs checks if a path is on a Btrfs filesystem
func isBtrfs(path string) bool {
	cmd := exec.Command("stat", "-f", "-c", "%T", path)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "btrfs"
}
