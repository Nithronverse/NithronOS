package wireguard

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"nithronos/backend/nosd/internal/fsatomic"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/curve25519"
)

// Config represents WireGuard interface configuration
type Config struct {
	Enabled     bool      `json:"enabled"`
	Interface   string    `json:"interface"`
	PrivateKey  string    `json:"privateKey"`
	PublicKey   string    `json:"publicKey"`
	ListenPort  int       `json:"listenPort"`
	Address     string    `json:"address"`
	DNS         []string  `json:"dns,omitempty"`
	MTU         int       `json:"mtu,omitempty"`
	PreUp       string    `json:"preUp,omitempty"`
	PostUp      string    `json:"postUp,omitempty"`
	PreDown     string    `json:"preDown,omitempty"`
	PostDown    string    `json:"postDown,omitempty"`
	SaveConfig  bool      `json:"saveConfig"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// Peer represents a WireGuard peer
type Peer struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description,omitempty"`
	PublicKey           string     `json:"publicKey"`
	PresharedKey        string     `json:"presharedKey,omitempty"`
	AllowedIPs          []string   `json:"allowedIPs"`
	Endpoint            string     `json:"endpoint,omitempty"`
	PersistentKeepalive int        `json:"persistentKeepalive,omitempty"`
	LastHandshake       *time.Time `json:"lastHandshake,omitempty"`
	TransferRx          int64      `json:"transferRx"`
	TransferTx          int64      `json:"transferTx"`
	Enabled             bool       `json:"enabled"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// Manager manages WireGuard configuration and peers
type Manager struct {
	storePath  string
	config     *Config
	peers      map[string]*Peer
	mu         sync.RWMutex
	wgPath     string
	configPath string
}

// NewManager creates a new WireGuard manager
func NewManager(storePath string) (*Manager, error) {
	m := &Manager{
		storePath:  storePath,
		peers:      make(map[string]*Peer),
		configPath: "/etc/wireguard/wg0.conf",
	}

	// Find wg binary
	wgPath, err := exec.LookPath("wg")
	if err != nil {
		// Try common locations
		for _, path := range []string{"/usr/bin/wg", "/usr/local/bin/wg"} {
			if _, err := os.Stat(path); err == nil {
				wgPath = path
				break
			}
		}
		if wgPath == "" {
			return nil, fmt.Errorf("wireguard tools not found")
		}
	}
	m.wgPath = wgPath

	// Ensure wireguard directory exists
	if err := os.MkdirAll("/etc/wireguard", 0700); err != nil {
		return nil, fmt.Errorf("failed to create wireguard directory: %w", err)
	}

	// Load existing configuration
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Initialize config if not exists
	if m.config == nil {
		m.initializeConfig()
	}

	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load config
	configPath := filepath.Join(m.storePath, "wireguard_config.json")
	var config Config
	if ok, err := fsatomic.LoadJSON(configPath, &config); err != nil {
		return err
	} else if ok {
		m.config = &config
	}

	// Load peers
	peersPath := filepath.Join(m.storePath, "wireguard_peers.json")
	var peers []*Peer
	if ok, err := fsatomic.LoadJSON(peersPath, &peers); err != nil {
		return err
	} else if ok {
		for _, peer := range peers {
			m.peers[peer.ID] = peer
		}
	}

	return nil
}

func (m *Manager) save() error {
	// Save config
	configPath := filepath.Join(m.storePath, "wireguard_config.json")
	if err := fsatomic.SaveJSON(context.Background(), configPath, m.config, 0600); err != nil {
		return err
	}

	// Save peers
	peers := make([]*Peer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}

	peersPath := filepath.Join(m.storePath, "wireguard_peers.json")
	return fsatomic.SaveJSON(context.Background(), peersPath, peers, 0600)
}

func (m *Manager) initializeConfig() {
	// Generate private key
	privateKey, publicKey := generateKeyPair()

	m.config = &Config{
		Enabled:     false,
		Interface:   "wg0",
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		ListenPort:  51820,
		Address:     "10.0.0.1/24",
		DNS:         []string{"1.1.1.1", "1.0.0.1"},
		MTU:         1420,
		SaveConfig:  false,
		LastUpdated: time.Now(),
	}

	// Add iptables rules for NAT
	m.config.PostUp = "iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE"
	m.config.PostDown = "iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE"

	m.save()
}

// GetConfig returns the WireGuard configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config
}

// UpdateConfig updates the WireGuard configuration
func (m *Manager) UpdateConfig(updates *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update fields
	if updates.ListenPort > 0 {
		m.config.ListenPort = updates.ListenPort
	}
	if updates.Address != "" {
		m.config.Address = updates.Address
	}
	if updates.DNS != nil {
		m.config.DNS = updates.DNS
	}
	if updates.MTU > 0 {
		m.config.MTU = updates.MTU
	}
	m.config.LastUpdated = time.Now()

	if err := m.save(); err != nil {
		return err
	}

	// Apply configuration if enabled
	if m.config.Enabled {
		return m.applyConfig()
	}

	return nil
}

// SetEnabled enables or disables WireGuard
func (m *Manager) SetEnabled(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if enabled {
		// Apply configuration
		if err := m.applyConfig(); err != nil {
			return err
		}

		// Start interface
		cmd := exec.Command("wg-quick", "up", m.config.Interface)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start wireguard: %s", string(output))
		}

		// Enable systemd service
		cmd = exec.Command("systemctl", "enable", fmt.Sprintf("wg-quick@%s", m.config.Interface))
		cmd.Run()
	} else {
		// Stop interface
		cmd := exec.Command("wg-quick", "down", m.config.Interface)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Interface might not be up
			log.Warn().Err(err).Str("output", string(output)).Msg("Failed to stop wireguard interface")
		}

		// Disable systemd service
		cmd = exec.Command("systemctl", "disable", fmt.Sprintf("wg-quick@%s", m.config.Interface))
		cmd.Run()
	}

	m.config.Enabled = enabled
	return m.save()
}

// ListPeers returns all peers
func (m *Manager) ListPeers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Update peer statistics if interface is up
	if m.config.Enabled {
		m.updatePeerStats()
	}

	peers := make([]*Peer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}

	return peers
}

// GetPeer returns a specific peer
func (m *Manager) GetPeer(id string) (*Peer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[id]
	return peer, ok
}

// CreatePeer creates a new peer
func (m *Manager) CreatePeer(peer *Peer) error {
	if peer.ID == "" {
		peer.ID = uuid.New().String()
	}

	// Generate keys if not provided
	if peer.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}

	// Set default allowed IPs if not provided
	if len(peer.AllowedIPs) == 0 {
		// Assign next available IP
		peer.AllowedIPs = []string{m.getNextPeerIP()}
	}

	peer.CreatedAt = time.Now()
	peer.UpdatedAt = time.Now()
	peer.Enabled = true

	m.mu.Lock()
	defer m.mu.Unlock()

	m.peers[peer.ID] = peer

	if err := m.save(); err != nil {
		return err
	}

	// Apply configuration if WireGuard is enabled
	if m.config.Enabled {
		return m.applyConfig()
	}

	return nil
}

// UpdatePeer updates an existing peer
func (m *Manager) UpdatePeer(id string, updates *Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	peer, ok := m.peers[id]
	if !ok {
		return fmt.Errorf("peer not found")
	}

	// Update fields
	if updates.Name != "" {
		peer.Name = updates.Name
	}
	if updates.Description != "" {
		peer.Description = updates.Description
	}
	if updates.AllowedIPs != nil {
		peer.AllowedIPs = updates.AllowedIPs
	}
	if updates.Endpoint != "" {
		peer.Endpoint = updates.Endpoint
	}
	peer.PersistentKeepalive = updates.PersistentKeepalive
	peer.Enabled = updates.Enabled
	peer.UpdatedAt = time.Now()

	if err := m.save(); err != nil {
		return err
	}

	// Apply configuration if WireGuard is enabled
	if m.config.Enabled {
		return m.applyConfig()
	}

	return nil
}

// DeletePeer deletes a peer
func (m *Manager) DeletePeer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.peers[id]; !ok {
		return fmt.Errorf("peer not found")
	}

	delete(m.peers, id)

	if err := m.save(); err != nil {
		return err
	}

	// Apply configuration if WireGuard is enabled
	if m.config.Enabled {
		return m.applyConfig()
	}

	return nil
}

// GeneratePeerConfig generates configuration for a peer
func (m *Manager) GeneratePeerConfig(peerID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peer, ok := m.peers[peerID]
	if !ok {
		return "", fmt.Errorf("peer not found")
	}

	// Get server's public IP
	serverEndpoint := m.getServerEndpoint()

	var buf bytes.Buffer

	// Interface section
	buf.WriteString("[Interface]\n")
	buf.WriteString(fmt.Sprintf("# Name: %s\n", peer.Name))
	buf.WriteString("PrivateKey = <PEER_PRIVATE_KEY>\n")
	buf.WriteString(fmt.Sprintf("Address = %s\n", strings.Join(peer.AllowedIPs, ", ")))

	if len(m.config.DNS) > 0 {
		buf.WriteString(fmt.Sprintf("DNS = %s\n", strings.Join(m.config.DNS, ", ")))
	}

	buf.WriteString("\n")

	// Peer section (server)
	buf.WriteString("[Peer]\n")
	buf.WriteString("# NithronOS Server\n")
	buf.WriteString(fmt.Sprintf("PublicKey = %s\n", m.config.PublicKey))

	if peer.PresharedKey != "" {
		buf.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
	}

	buf.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	buf.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", serverEndpoint, m.config.ListenPort))

	if peer.PersistentKeepalive > 0 {
		buf.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))
	}

	return buf.String(), nil
}

// GenerateQRCode generates a QR code for peer configuration
func (m *Manager) GenerateQRCode(peerID string) (string, error) {
	config, err := m.GeneratePeerConfig(peerID)
	if err != nil {
		return "", err
	}

	// For QR code, we need to include the private key
	// In production, this should be generated on the client side
	// Here we'll return the config as base64 for the frontend to generate QR

	return base64.StdEncoding.EncodeToString([]byte(config)), nil
}

// applyConfig generates and applies WireGuard configuration
func (m *Manager) applyConfig() error {
	config := m.generateConfigFile()

	// Write configuration
	if err := os.WriteFile(m.configPath, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Reload interface if it's running
	if m.isInterfaceUp() {
		// Use wg syncconf to apply changes without disrupting connections
		cmd := exec.Command(m.wgPath, "syncconf", m.config.Interface, m.configPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to sync config: %s", string(output))
		}
	}

	return nil
}

// generateConfigFile generates WireGuard configuration file
func (m *Manager) generateConfigFile() string {
	var buf bytes.Buffer

	// Interface section
	buf.WriteString("[Interface]\n")
	buf.WriteString(fmt.Sprintf("PrivateKey = %s\n", m.config.PrivateKey))
	buf.WriteString(fmt.Sprintf("Address = %s\n", m.config.Address))
	buf.WriteString(fmt.Sprintf("ListenPort = %d\n", m.config.ListenPort))

	if len(m.config.DNS) > 0 {
		buf.WriteString(fmt.Sprintf("DNS = %s\n", strings.Join(m.config.DNS, ", ")))
	}

	if m.config.MTU > 0 {
		buf.WriteString(fmt.Sprintf("MTU = %d\n", m.config.MTU))
	}

	if m.config.PreUp != "" {
		buf.WriteString(fmt.Sprintf("PreUp = %s\n", m.config.PreUp))
	}

	if m.config.PostUp != "" {
		buf.WriteString(fmt.Sprintf("PostUp = %s\n", m.config.PostUp))
	}

	if m.config.PreDown != "" {
		buf.WriteString(fmt.Sprintf("PreDown = %s\n", m.config.PreDown))
	}

	if m.config.PostDown != "" {
		buf.WriteString(fmt.Sprintf("PostDown = %s\n", m.config.PostDown))
	}

	// Peer sections
	for _, peer := range m.peers {
		if !peer.Enabled {
			continue
		}

		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("# Peer: %s\n", peer.Name))
		buf.WriteString("[Peer]\n")
		buf.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))

		if peer.PresharedKey != "" {
			buf.WriteString(fmt.Sprintf("PresharedKey = %s\n", peer.PresharedKey))
		}

		if len(peer.AllowedIPs) > 0 {
			buf.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(peer.AllowedIPs, ", ")))
		}

		if peer.Endpoint != "" {
			buf.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
		}

		if peer.PersistentKeepalive > 0 {
			buf.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))
		}
	}

	return buf.String()
}

// isInterfaceUp checks if the WireGuard interface is up
func (m *Manager) isInterfaceUp() bool {
	cmd := exec.Command(m.wgPath, "show", m.config.Interface)
	return cmd.Run() == nil
}

// updatePeerStats updates peer statistics from wg show
func (m *Manager) updatePeerStats() {
	cmd := exec.Command(m.wgPath, "show", m.config.Interface, "dump")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		// Find peer by public key
		publicKey := fields[0]
		for _, peer := range m.peers {
			if peer.PublicKey == publicKey {
				// Update statistics
				if rx, err := strconv.ParseInt(fields[5], 10, 64); err == nil {
					peer.TransferRx = rx
				}
				if tx, err := strconv.ParseInt(fields[6], 10, 64); err == nil {
					peer.TransferTx = tx
				}

				// Update last handshake
				if ts, err := strconv.ParseInt(fields[4], 10, 64); err == nil && ts > 0 {
					handshake := time.Unix(ts, 0)
					peer.LastHandshake = &handshake
				}

				// Update endpoint
				if fields[2] != "(none)" {
					peer.Endpoint = fields[2]
				}

				break
			}
		}
	}
}

// getNextPeerIP generates the next available peer IP
func (m *Manager) getNextPeerIP() string {
	// Parse the server's network
	_, network, err := net.ParseCIDR(m.config.Address)
	if err != nil {
		return "10.0.0.2/32"
	}

	// Find used IPs
	usedIPs := make(map[string]bool)
	for _, peer := range m.peers {
		for _, allowedIP := range peer.AllowedIPs {
			ip, _, _ := net.ParseCIDR(allowedIP)
			if ip != nil {
				usedIPs[ip.String()] = true
			}
		}
	}

	// Find next available IP
	ip := network.IP
	for i := 2; i < 255; i++ {
		ip = ip.To4()
		ip[3] = byte(i)

		if !usedIPs[ip.String()] {
			return fmt.Sprintf("%s/%d", ip.String(), 32)
		}
	}

	return "10.0.0.2/32"
}

// getServerEndpoint returns the server's public endpoint
func (m *Manager) getServerEndpoint() string {
	// Try to get public IP
	cmd := exec.Command("curl", "-s", "https://api.ipify.org")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// Fallback to hostname
	hostname, err := os.Hostname()
	if err == nil {
		return hostname
	}

	return "your-server.example.com"
}

// GetStatus returns WireGuard status information
func (m *Manager) GetStatus() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"enabled":    m.config.Enabled,
		"interface":  m.config.Interface,
		"publicKey":  m.config.PublicKey,
		"listenPort": m.config.ListenPort,
		"address":    m.config.Address,
		"peerCount":  len(m.peers),
	}

	if m.config.Enabled && m.isInterfaceUp() {
		// Get interface statistics
		cmd := exec.Command(m.wgPath, "show", m.config.Interface, "transfer")
		if output, err := cmd.Output(); err == nil {
			status["transfer"] = string(output)
		}

		// Count active peers
		activePeers := 0
		for _, peer := range m.peers {
			if peer.LastHandshake != nil && time.Since(*peer.LastHandshake) < 3*time.Minute {
				activePeers++
			}
		}
		status["activePeers"] = activePeers
	}

	return status, nil
}

// generateKeyPair generates a WireGuard key pair
func generateKeyPair() (string, string) {
	// Generate private key
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		panic(err)
	}

	// Clamp private key
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Generate public key
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	// Encode to base64
	privateKeyStr := base64.StdEncoding.EncodeToString(privateKey[:])
	publicKeyStr := base64.StdEncoding.EncodeToString(publicKey[:])

	return privateKeyStr, publicKeyStr
}

// GenerateKeyPair generates a new WireGuard key pair (exported)
func GenerateKeyPair() (string, string) {
	return generateKeyPair()
}
