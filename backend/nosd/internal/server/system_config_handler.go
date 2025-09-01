package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// AgentRequest represents a request to the agent
type AgentRequest struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type SystemConfigHandler struct {
	logger      zerolog.Logger
	agentClient AgentClient
}

func NewSystemConfigHandler(logger zerolog.Logger, agentClient AgentClient) *SystemConfigHandler {
	return &SystemConfigHandler{
		logger:      logger.With().Str("component", "system-config").Logger(),
		agentClient: agentClient,
	}
}

func (h *SystemConfigHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Hostname
	r.Get("/hostname", h.GetHostname)
	r.Post("/hostname", h.SetHostname)

	// Timezone
	r.Get("/timezone", h.GetTimezone)
	r.Post("/timezone", h.SetTimezone)
	r.Get("/timezones", h.ListTimezones)

	// NTP
	r.Get("/ntp", h.GetNTP)
	r.Post("/ntp", h.SetNTP)

	// Network interfaces
	r.Get("/network/interfaces", h.ListInterfaces)
	r.Get("/network/interfaces/{iface}", h.GetInterface)
	r.Post("/network/interfaces/{iface}", h.ConfigureInterface)

	// Telemetry consent
	r.Get("/telemetry/consent", h.GetTelemetryConsent)
	r.Post("/telemetry/consent", h.SetTelemetryConsent)

	return r
}

// Hostname management

type HostnameConfig struct {
	Hostname       string `json:"hostname"`
	PrettyHostname string `json:"pretty_hostname,omitempty"`
}

func (h *SystemConfigHandler) GetHostname(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get hostname")
		respondError(w, http.StatusInternalServerError, "Failed to get hostname")
		return
	}

	// Try to get pretty hostname
	prettyHostname := ""
	if data, err := os.ReadFile("/etc/machine-info"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_HOSTNAME=") {
				prettyHostname = strings.TrimPrefix(line, "PRETTY_HOSTNAME=")
				prettyHostname = strings.Trim(prettyHostname, "\"")
				break
			}
		}
	}

	respondJSON(w, http.StatusOK, HostnameConfig{
		Hostname:       hostname,
		PrettyHostname: prettyHostname,
	})
}

func (h *SystemConfigHandler) SetHostname(w http.ResponseWriter, r *http.Request) {
	var config HostnameConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate hostname (RFC 1123)
	if !isValidHostname(config.Hostname) {
		respondError(w, http.StatusBadRequest, "Invalid hostname format")
		return
	}

	// Use agent to set hostname (privileged operation). In tests, allow bypass.
	if os.Getenv("NOS_TEST_BYPASS_AGENT") != "1" {
		req := AgentRequest{
			Action: "system.hostname.set",
			Params: map[string]interface{}{
				"hostname":        config.Hostname,
				"pretty_hostname": config.PrettyHostname,
			},
		}
		var resp interface{}
		if err := h.agentClient.PostJSON(context.Background(), "/execute", req, &resp); err != nil {
			h.logger.Error().Err(err).Msg("Failed to set hostname")
			respondError(w, http.StatusInternalServerError, "Failed to set hostname")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Timezone management

type TimezoneConfig struct {
	Timezone string    `json:"timezone"`
	Time     time.Time `json:"time"`
	UTC      bool      `json:"utc"`
}

func (h *SystemConfigHandler) GetTimezone(w http.ResponseWriter, r *http.Request) {
	// Read /etc/timezone or use timedatectl
	timezone := "UTC"
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		timezone = strings.TrimSpace(string(data))
	} else {
		// Try timedatectl
		if output, err := exec.Command("timedatectl", "show", "--value", "-p", "Timezone").Output(); err == nil {
			timezone = strings.TrimSpace(string(output))
		}
	}

	// Check if hardware clock is UTC
	utc := true
	if data, err := os.ReadFile("/etc/adjtime"); err == nil {
		lines := strings.Split(string(data), "\n")
		if len(lines) >= 3 && strings.Contains(lines[2], "LOCAL") {
			utc = false
		}
	}

	respondJSON(w, http.StatusOK, TimezoneConfig{
		Timezone: timezone,
		Time:     time.Now(),
		UTC:      utc,
	})
}

func (h *SystemConfigHandler) SetTimezone(w http.ResponseWriter, r *http.Request) {
	var config TimezoneConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate timezone exists
	tzPath := fmt.Sprintf("/usr/share/zoneinfo/%s", config.Timezone)
	if _, err := os.Stat(tzPath); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid timezone")
		return
	}

	// Use agent to set timezone; bypass in tests
	if os.Getenv("NOS_TEST_BYPASS_AGENT") != "1" {
		req := AgentRequest{
			Action: "system.timezone.set",
			Params: map[string]interface{}{
				"timezone": config.Timezone,
				"utc":      config.UTC,
			},
		}
		var resp interface{}
		if err := h.agentClient.PostJSON(context.Background(), "/execute", req, &resp); err != nil {
			h.logger.Error().Err(err).Msg("Failed to set timezone")
			respondError(w, http.StatusInternalServerError, "Failed to set timezone")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SystemConfigHandler) ListTimezones(w http.ResponseWriter, r *http.Request) {
	// List all available timezones
	timezones := []string{}

	// Parse tzdata
	cmd := exec.Command("timedatectl", "list-timezones")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				timezones = append(timezones, line)
			}
		}
	} else {
		// Fallback: read from /usr/share/zoneinfo
		// This is more complex, simplified for now
		timezones = []string{
			"UTC",
			"America/New_York",
			"America/Chicago",
			"America/Denver",
			"America/Los_Angeles",
			"Europe/London",
			"Europe/Paris",
			"Europe/Berlin",
			"Asia/Tokyo",
			"Asia/Shanghai",
			"Australia/Sydney",
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"timezones": timezones,
	})
}

// NTP management

type NTPConfig struct {
	Enabled bool     `json:"enabled"`
	Servers []string `json:"servers"`
	Status  string   `json:"status"`
}

func (h *SystemConfigHandler) GetNTP(w http.ResponseWriter, r *http.Request) {
	config := NTPConfig{
		Enabled: false,
		Servers: []string{},
		Status:  "unknown",
	}

	// Check if NTP is enabled
	if output, err := exec.Command("timedatectl", "show", "--value", "-p", "NTP").Output(); err == nil {
		config.Enabled = strings.TrimSpace(string(output)) == "yes"
	}

	// Get NTP servers from systemd-timesyncd
	if data, err := os.ReadFile("/etc/systemd/timesyncd.conf"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "NTP=") {
				servers := strings.TrimPrefix(line, "NTP=")
				config.Servers = strings.Fields(servers)
				break
			}
		}
	}

	// Get sync status
	if output, err := exec.Command("timedatectl", "show", "--value", "-p", "NTPSynchronized").Output(); err == nil {
		if strings.TrimSpace(string(output)) == "yes" {
			config.Status = "synchronized"
		} else {
			config.Status = "not_synchronized"
		}
	}

	respondJSON(w, http.StatusOK, config)
}

func (h *SystemConfigHandler) SetNTP(w http.ResponseWriter, r *http.Request) {
	var config NTPConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Use agent to configure NTP; bypass in tests
	if os.Getenv("NOS_TEST_BYPASS_AGENT") != "1" {
		req := AgentRequest{
			Action: "system.ntp.configure",
			Params: map[string]interface{}{
				"enabled": config.Enabled,
				"servers": config.Servers,
			},
		}
		var resp interface{}
		if err := h.agentClient.PostJSON(context.Background(), "/execute", req, &resp); err != nil {
			h.logger.Error().Err(err).Msg("Failed to configure NTP")
			respondError(w, http.StatusInternalServerError, "Failed to configure NTP")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Network interface management

type NetworkInterface struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	State       string   `json:"state"`
	MACAddress  string   `json:"mac_address"`
	MTU         int      `json:"mtu"`
	IPv4Address []string `json:"ipv4_address"`
	IPv6Address []string `json:"ipv6_address"`
	Gateway     string   `json:"gateway,omitempty"`
	DNS         []string `json:"dns,omitempty"`
	DHCP        bool     `json:"dhcp"`
}

type NetworkConfig struct {
	DHCP        bool     `json:"dhcp"`
	IPv4Address string   `json:"ipv4_address,omitempty"`
	IPv4Gateway string   `json:"ipv4_gateway,omitempty"`
	DNS         []string `json:"dns,omitempty"`
}

func (h *SystemConfigHandler) ListInterfaces(w http.ResponseWriter, r *http.Request) {
	interfaces := []NetworkInterface{}

	ifaces, err := net.Interfaces()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list interfaces")
		respondError(w, http.StatusInternalServerError, "Failed to list interfaces")
		return
	}

	for _, iface := range ifaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		ni := NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			MTU:        iface.MTU,
			State:      "down",
		}

		// Check if interface is up
		if iface.Flags&net.FlagUp != 0 {
			ni.State = "up"
		}

		// Determine type
		if strings.HasPrefix(iface.Name, "eth") || strings.HasPrefix(iface.Name, "enp") {
			ni.Type = "ethernet"
		} else if strings.HasPrefix(iface.Name, "wlan") || strings.HasPrefix(iface.Name, "wlp") {
			ni.Type = "wireless"
		} else if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "br") {
			ni.Type = "bridge"
		} else {
			ni.Type = "unknown"
		}

		// Get addresses
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipnet.IP.To4() != nil {
				ni.IPv4Address = append(ni.IPv4Address, addr.String())
			} else {
				ni.IPv6Address = append(ni.IPv6Address, addr.String())
			}
		}

		// Try to determine if DHCP is in use (simplified)
		ni.DHCP = h.isDHCP(iface.Name)

		interfaces = append(interfaces, ni)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"interfaces": interfaces,
	})
}

func (h *SystemConfigHandler) GetInterface(w http.ResponseWriter, r *http.Request) {
	ifaceName := chi.URLParam(r, "iface")

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		respondError(w, http.StatusNotFound, "Interface not found")
		return
	}

	ni := NetworkInterface{
		Name:       iface.Name,
		MACAddress: iface.HardwareAddr.String(),
		MTU:        iface.MTU,
		State:      "down",
		DHCP:       h.isDHCP(iface.Name),
	}

	if iface.Flags&net.FlagUp != 0 {
		ni.State = "up"
	}

	// Get addresses
	addrs, _ := iface.Addrs()
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		if ipnet.IP.To4() != nil {
			ni.IPv4Address = append(ni.IPv4Address, addr.String())
		} else {
			ni.IPv6Address = append(ni.IPv6Address, addr.String())
		}
	}

	respondJSON(w, http.StatusOK, ni)
}

func (h *SystemConfigHandler) ConfigureInterface(w http.ResponseWriter, r *http.Request) {
	ifaceName := chi.URLParam(r, "iface")

	var config NetworkConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate interface exists
	if _, err := net.InterfaceByName(ifaceName); err != nil {
		respondError(w, http.StatusNotFound, "Interface not found")
		return
	}

	// Validate IP address if static
	if !config.DHCP {
		if _, _, err := net.ParseCIDR(config.IPv4Address); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid IP address format")
			return
		}

		if config.IPv4Gateway != "" {
			if net.ParseIP(config.IPv4Gateway) == nil {
				respondError(w, http.StatusBadRequest, "Invalid gateway address")
				return
			}
		}
	}

	// Use agent to configure interface; bypass in tests
	if os.Getenv("NOS_TEST_BYPASS_AGENT") != "1" {
		req := AgentRequest{
			Action: "network.interface.configure",
			Params: map[string]interface{}{
				"interface":    ifaceName,
				"dhcp":         config.DHCP,
				"ipv4_address": config.IPv4Address,
				"ipv4_gateway": config.IPv4Gateway,
				"dns":          config.DNS,
			},
		}
		var resp interface{}
		if err := h.agentClient.PostJSON(context.Background(), "/execute", req, &resp); err != nil {
			h.logger.Error().Err(err).Msg("Failed to configure interface")
			respondError(w, http.StatusInternalServerError, "Failed to configure interface")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Telemetry consent

type TelemetryConsent struct {
	Enabled      bool      `json:"enabled"`
	ConsentedAt  time.Time `json:"consented_at,omitempty"`
	DataTypes    []string  `json:"data_types,omitempty"`
	LastReportAt time.Time `json:"last_report_at,omitempty"`
}

func (h *SystemConfigHandler) GetTelemetryConsent(w http.ResponseWriter, r *http.Request) {
	consent := TelemetryConsent{
		Enabled: false,
	}

	// Read consent from file
	consentPath := "/etc/nos/telemetry/consent.json"
	if data, err := os.ReadFile(consentPath); err == nil {
		_ = json.Unmarshal(data, &consent)
	}

	respondJSON(w, http.StatusOK, consent)
}

func (h *SystemConfigHandler) SetTelemetryConsent(w http.ResponseWriter, r *http.Request) {
	var consent TelemetryConsent
	if err := json.NewDecoder(r.Body).Decode(&consent); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set consent timestamp
	if consent.Enabled && consent.ConsentedAt.IsZero() {
		consent.ConsentedAt = time.Now()
	}

	// Default data types if not specified
	if consent.Enabled && len(consent.DataTypes) == 0 {
		consent.DataTypes = []string{
			"system_info",
			"usage_stats",
			"error_reports",
		}
	}

	// Save consent
	consentPath := "/etc/nos/telemetry/consent.json"
	_ = os.MkdirAll(filepath.Dir(consentPath), 0700)

	data, _ := json.MarshalIndent(consent, "", "  ")
	if err := os.WriteFile(consentPath, data, 0600); err != nil {
		h.logger.Error().Err(err).Msg("Failed to save telemetry consent")
		respondError(w, http.StatusInternalServerError, "Failed to save consent")
		return
	}

	// Enable/disable telemetry service
	action := "stop"
	if consent.Enabled {
		action = "start"
	}

	if os.Getenv("NOS_TEST_BYPASS_AGENT") != "1" {
		_ = exec.Command("systemctl", action, "nos-telemetry").Run()
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Helper functions

func isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// RFC 1123 validation
	for _, part := range strings.Split(hostname, ".") {
		if len(part) == 0 || len(part) > 63 {
			return false
		}

		for i, ch := range part {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
				(ch >= '0' && ch <= '9') || (i > 0 && i < len(part)-1 && ch == '-')) {
				return false
			}
		}
	}

	return true
}

func (h *SystemConfigHandler) isDHCP(ifaceName string) bool {
	// Check systemd-networkd configuration
	networkdPath := fmt.Sprintf("/etc/systemd/network/10-%s.network", ifaceName)
	if data, err := os.ReadFile(networkdPath); err == nil {
		return strings.Contains(string(data), "DHCP=yes") ||
			strings.Contains(string(data), "DHCP=ipv4")
	}

	// Check NetworkManager (if present)
	cmd := exec.Command("nmcli", "-t", "-f", "ipv4.method", "c", "show", ifaceName)
	if output, err := cmd.Output(); err == nil {
		return strings.Contains(string(output), "auto")
	}

	// Check /etc/network/interfaces (Debian style)
	if data, err := os.ReadFile("/etc/network/interfaces"); err == nil {
		lines := strings.Split(string(data), "\n")
		inIface := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "iface "+ifaceName) {
				inIface = true
			} else if inIface && strings.HasPrefix(line, "iface ") {
				break
			}

			if inIface && strings.Contains(line, "dhcp") {
				return true
			}
		}
	}

	// Default assumption
	return true
}
