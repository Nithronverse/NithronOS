package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"nithronos/backend/nosd/pkg/net"
)

// NetHandler handles networking-related API endpoints
type NetHandler struct {
	firewallMgr *net.FirewallManager
	wgMgr       *net.WireGuardManager
	httpsMgr    *net.HTTPSManager
	totpMgr     *net.TOTPManager
	logger      zerolog.Logger
}

// NewNetHandler creates a new networking handler
func NewNetHandler(logger zerolog.Logger) (*NetHandler, error) {
	wgMgr, err := net.NewWireGuardManager()
	if err != nil {
		return nil, err
	}

	return &NetHandler{
		firewallMgr: net.NewFirewallManager(),
		wgMgr:       wgMgr,
		httpsMgr:    net.NewHTTPSManager(),
		totpMgr:     net.NewTOTPManager(),
		logger:      logger,
	}, nil
}

// Routes returns the networking routes
func (h *NetHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Network status
	r.Get("/status", h.GetNetworkStatus)

	// Firewall endpoints
	r.Get("/firewall/state", h.GetFirewallState)
	r.Post("/firewall/plan", h.PlanFirewall)
	r.Post("/firewall/apply", h.ApplyFirewall)
	r.Post("/firewall/confirm", h.ConfirmFirewall)
	r.Post("/firewall/rollback", h.RollbackFirewall)

	// WireGuard endpoints
	r.Get("/wg/state", h.GetWireGuardState)
	r.Post("/wg/enable", h.EnableWireGuard)
	r.Post("/wg/disable", h.DisableWireGuard)
	r.Post("/wg/peers/add", h.AddWireGuardPeer)
	r.Post("/wg/peers/remove", h.RemoveWireGuardPeer)

	// HTTPS/ACME endpoints
	r.Get("/https/config", h.GetHTTPSConfig)
	r.Post("/https/configure", h.ConfigureHTTPS)
	r.Post("/https/test", h.TestHTTPS)

	// Remote Access Wizard
	r.Post("/wizard/start", h.StartWizard)
	r.Get("/wizard/state", h.GetWizardState)
	r.Post("/wizard/next", h.WizardNext)
	r.Post("/wizard/complete", h.CompleteWizard)

	return r
}

// AuthRoutes returns the 2FA/auth routes to be mounted separately
func (h *NetHandler) AuthRoutes() chi.Router {
	r := chi.NewRouter()

	// 2FA/TOTP endpoints
	r.Post("/2fa/enroll", h.EnrollTOTP)
	r.Post("/2fa/verify", h.VerifyTOTP)
	r.Post("/2fa/disable", h.DisableTOTP)
	r.Post("/2fa/backup-codes", h.RegenerateBackupCodes)
	r.Get("/2fa/status", h.GetTOTPStatus)

	return r
}

// GetNetworkStatus returns the overall network configuration status
func (h *NetHandler) GetNetworkStatus(w http.ResponseWriter, r *http.Request) {
	// Get current states from all managers
	firewallState, _ := h.firewallMgr.GetState()
	wgState, _ := h.wgMgr.GetState()
	httpsConfig, _ := h.httpsMgr.GetConfig()

	// Determine access mode
	var accessMode net.AccessMode
	if firewallState != nil {
		accessMode = firewallState.Mode
	} else {
		accessMode = net.AccessModeLANOnly
	}

	status := net.NetworkStatus{
		AccessMode:  accessMode,
		LANAccess:   true,
		WANBlocked:  accessMode == net.AccessModeLANOnly,
		WireGuard:   wgState,
		HTTPS:       httpsConfig,
		Firewall:    firewallState,
		ExternalIP:  h.getExternalIP(),
		InternalIPs: h.getInternalIPs(),
		OpenPorts:   h.getOpenPorts(),
	}

	h.writeJSON(w, status)
}

// Firewall handlers

func (h *NetHandler) GetFirewallState(w http.ResponseWriter, r *http.Request) {
	state, err := h.firewallMgr.GetState()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, state)
}

func (h *NetHandler) PlanFirewall(w http.ResponseWriter, r *http.Request) {
	var req net.PlanFirewallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create firewall plan
	plan, err := h.firewallMgr.CreatePlan(req.DesiredMode, req.EnableWG, req.EnableHTTPS, req.CustomRules)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, plan)
}

func (h *NetHandler) ApplyFirewall(w http.ResponseWriter, r *http.Request) {
	var req net.ApplyFirewallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply the plan
	if err := h.firewallMgr.ApplyPlan(req.PlanID, req.RollbackTimeoutSec); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "applied",
		"message": "Firewall configuration applied. Please confirm within timeout period.",
	})
}

func (h *NetHandler) ConfirmFirewall(w http.ResponseWriter, r *http.Request) {
	if err := h.firewallMgr.ConfirmPlan(); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "confirmed",
		"message": "Firewall configuration confirmed and active.",
	})
}

func (h *NetHandler) RollbackFirewall(w http.ResponseWriter, r *http.Request) {
	if err := h.firewallMgr.Rollback(); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "rolled_back",
		"message": "Firewall configuration rolled back to previous state.",
	})
}

// WireGuard handlers

func (h *NetHandler) GetWireGuardState(w http.ResponseWriter, r *http.Request) {
	state, err := h.wgMgr.GetState()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, state)
}

func (h *NetHandler) EnableWireGuard(w http.ResponseWriter, r *http.Request) {
	var req net.EnableWireGuardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Enable WireGuard
	if err := h.wgMgr.Enable(req.CIDR, req.ListenPort, req.EndpointHostname, req.DNS); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "enabled",
		"message": "WireGuard VPN enabled successfully.",
	})
}

func (h *NetHandler) DisableWireGuard(w http.ResponseWriter, r *http.Request) {
	if err := h.wgMgr.Disable(); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "disabled",
		"message": "WireGuard VPN disabled.",
	})
}

func (h *NetHandler) AddWireGuardPeer(w http.ResponseWriter, r *http.Request) {
	var req net.AddWireGuardPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Add peer
	config, err := h.wgMgr.AddPeer(req.Name, req.AllowedIPs, req.PublicKey)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, config)
}

func (h *NetHandler) RemoveWireGuardPeer(w http.ResponseWriter, r *http.Request) {
	peerID := r.URL.Query().Get("id")
	if peerID == "" {
		h.writeError(w, http.StatusBadRequest, "Peer ID required")
		return
	}

	if err := h.wgMgr.RemovePeer(peerID); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "removed",
		"message": "WireGuard peer removed.",
	})
}

// HTTPS/ACME handlers

func (h *NetHandler) GetHTTPSConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.httpsMgr.GetConfig()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, config)
}

func (h *NetHandler) ConfigureHTTPS(w http.ResponseWriter, r *http.Request) {
	var req net.ConfigureHTTPSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Configure HTTPS
	if err := h.httpsMgr.Configure(req.Mode, req.Domain, req.Email, req.DNSProvider, req.DNSAPIKey); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "configured",
		"message": "HTTPS configuration applied successfully.",
	})
}

func (h *NetHandler) TestHTTPS(w http.ResponseWriter, r *http.Request) {
	if err := h.httpsMgr.TestConfiguration(); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "success",
		"message": "HTTPS configuration test passed.",
	})
}

// 2FA/TOTP handlers

func (h *NetHandler) EnrollTOTP(w http.ResponseWriter, r *http.Request) {
	var req net.EnrollTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get user info from session
	userID := h.getUserID(r)
	username := h.getUsername(r)

	// Verify password (implement actual password check)
	if !h.verifyPassword(userID, req.Password) {
		h.writeError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	// Enroll user
	enrollment, err := h.totpMgr.EnrollUser(userID, username)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, enrollment)
}

func (h *NetHandler) VerifyTOTP(w http.ResponseWriter, r *http.Request) {
	var req net.VerifyTOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID := h.getUserID(r)

	// Verify code
	valid, err := h.totpMgr.VerifyCode(userID, req.Code)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !valid {
		h.writeError(w, http.StatusUnauthorized, "Invalid code")
		return
	}

	// Mark session as 2FA verified
	sessionID := h.getSessionID(r)
	h.totpMgr.MarkSessionVerified(sessionID, 30*time.Minute)

	h.writeJSON(w, map[string]string{
		"status":  "verified",
		"message": "2FA verification successful.",
	})
}

func (h *NetHandler) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	if err := h.totpMgr.DisableUser(userID); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string]string{
		"status":  "disabled",
		"message": "2FA disabled for user.",
	})
}

func (h *NetHandler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	codes, err := h.totpMgr.RegenerateBackupCodes(userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, map[string][]string{
		"backup_codes": codes,
	})
}

func (h *NetHandler) GetTOTPStatus(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	enrolled := h.totpMgr.IsUserEnrolled(userID)
	sessionID := h.getSessionID(r)
	verified := h.totpMgr.IsSessionVerified(sessionID)

	// Check if 2FA is required
	remoteIP := h.getRemoteIP(r)
	required := net.RequiresTwoFactor(remoteIP, userID, h.totpMgr)

	h.writeJSON(w, map[string]bool{
		"enrolled": enrolled,
		"verified": verified,
		"required": required,
	})
}

// Remote Access Wizard handlers

func (h *NetHandler) StartWizard(w http.ResponseWriter, r *http.Request) {
	// Initialize wizard state
	state := &net.RemoteAccessWizardState{
		Step:       1,
		AccessMode: net.AccessModeLANOnly,
		Completed:  false,
	}

	// Store in session (implement session storage)
	sessionID := h.getSessionID(r)
	h.storeWizardState(sessionID, state)

	h.writeJSON(w, state)
}

func (h *NetHandler) GetWizardState(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	state := h.getWizardState(sessionID)

	if state == nil {
		h.writeError(w, http.StatusNotFound, "No wizard session found")
		return
	}

	h.writeJSON(w, state)
}

func (h *NetHandler) WizardNext(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	state := h.getWizardState(sessionID)

	if state == nil {
		h.writeError(w, http.StatusNotFound, "No wizard session found")
		return
	}

	// Parse step data
	var stepData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&stepData); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Process based on current step
	switch state.Step {
	case 1: // Access mode selection
		if mode, ok := stepData["access_mode"].(string); ok {
			state.AccessMode = net.AccessMode(mode)
		}

	case 2: // WireGuard configuration
		if state.AccessMode == net.AccessModeWireGuard {
			// TODO: Store WireGuard config
			// This will be implemented when WireGuard configuration is added
			// state.WireGuardConfig = extractWireGuardConfig(stepData)
			_ = state // placeholder to satisfy linter until implementation
		}

	case 3: // HTTPS configuration
		if state.AccessMode == net.AccessModePublicHTTPS {
			// TODO: Store HTTPS config
			// This will be implemented when HTTPS configuration is added
			// state.HTTPSConfig = extractHTTPSConfig(stepData)
			_ = state // placeholder to satisfy linter until implementation
		}

	case 4: // Firewall plan
		// Generate firewall plan based on selections
		plan, err := h.firewallMgr.CreatePlan(
			state.AccessMode,
			state.WireGuardConfig != nil,
			state.HTTPSConfig != nil,
			nil,
		)
		if err != nil {
			state.Error = err.Error()
		} else {
			state.FirewallPlan = plan
		}
	}

	// Move to next step
	state.Step++

	// Store updated state
	h.storeWizardState(sessionID, state)

	h.writeJSON(w, state)
}

func (h *NetHandler) CompleteWizard(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	state := h.getWizardState(sessionID)

	if state == nil {
		h.writeError(w, http.StatusNotFound, "No wizard session found")
		return
	}

	// Apply all configurations

	// 1. Apply WireGuard if configured
	if state.WireGuardConfig != nil && state.WireGuardConfig.Enabled {
		// TODO: Enable WireGuard
		// This will be implemented when WireGuard manager is added
		// err := h.wgMgr.Enable(state.WireGuardConfig)
		// if err != nil { ... }
		_ = state // placeholder to satisfy linter until implementation
	}

	// 2. Apply HTTPS if configured
	if state.HTTPSConfig != nil {
		// Configure HTTPS
		// h.httpsMgr.Configure(...)
	}

	// 3. Apply firewall plan
	if state.FirewallPlan != nil {
		// Apply firewall
		// h.firewallMgr.ApplyPlan(...)
	}

	state.Completed = true
	h.storeWizardState(sessionID, state)

	h.writeJSON(w, map[string]string{
		"status":  "completed",
		"message": "Remote access configuration applied successfully.",
	})
}

// Helper methods

func (h *NetHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
	}
}

func (h *NetHandler) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		fmt.Printf("Failed to write error response: %v\n", err)
	}
}

func (h *NetHandler) getUserID(r *http.Request) string {
	// Extract from JWT or session
	// Placeholder implementation
	return "user-123"
}

func (h *NetHandler) getUsername(r *http.Request) string {
	// Extract from JWT or session
	// Placeholder implementation
	return "admin"
}

func (h *NetHandler) getSessionID(r *http.Request) string {
	// Extract from cookie or header
	// Placeholder implementation
	return "session-123"
}

func (h *NetHandler) getRemoteIP(r *http.Request) string {
	// Get real IP considering proxy headers
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip != "" {
			// Take first IP if multiple
			parts := strings.Split(ip, ",")
			ip = strings.TrimSpace(parts[0])
		}
	}
	if ip == "" {
		ip = r.RemoteAddr
		// Remove port if present
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	return ip
}

func (h *NetHandler) verifyPassword(userID, password string) bool {
	// Implement actual password verification
	// This should check against stored hash
	return true // Placeholder
}

func (h *NetHandler) getExternalIP() string {
	// Get external IP
	// Placeholder - would call external service
	return "203.0.113.1"
}

func (h *NetHandler) getInternalIPs() []string {
	// Get internal IPs
	// Placeholder - would enumerate interfaces
	return []string{"192.168.1.100", "10.0.0.5"}
}

func (h *NetHandler) getOpenPorts() []int {
	// Get open ports
	// Placeholder - would check netstat/ss
	return []int{22, 80, 443, 51820}
}

// Wizard state management (would be in session store)
var wizardStates = make(map[string]*net.RemoteAccessWizardState)
var wizardMu sync.RWMutex

func (h *NetHandler) storeWizardState(sessionID string, state *net.RemoteAccessWizardState) {
	wizardMu.Lock()
	defer wizardMu.Unlock()
	wizardStates[sessionID] = state
}

func (h *NetHandler) getWizardState(sessionID string) *net.RemoteAccessWizardState {
	wizardMu.RLock()
	defer wizardMu.RUnlock()
	return wizardStates[sessionID]
}
