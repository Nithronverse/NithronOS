package firewall

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nithronos/backend/nosd/internal/fsatomic"

	"github.com/google/uuid"
)

// Rule represents a firewall rule
type Rule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Direction   string    `json:"direction"` // inbound, outbound
	Action      string    `json:"action"`    // allow, deny, reject
	Protocol    string    `json:"protocol"`  // tcp, udp, icmp, any
	Source      Address   `json:"source"`
	Destination Address   `json:"destination"`
	Interface   string    `json:"interface,omitempty"`
	Priority    int       `json:"priority"`
	Comment     string    `json:"comment,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Address represents network address with optional port
type Address struct {
	IP   string `json:"ip,omitempty"`   // IP address or CIDR
	Port string `json:"port,omitempty"` // Port or port range
}

// Status represents firewall status
type Status struct {
	Enabled     bool      `json:"enabled"`
	RuleCount   int       `json:"ruleCount"`
	LastUpdated time.Time `json:"lastUpdated"`
	Version     string    `json:"version"`
}

// Manager manages firewall rules using nftables
type Manager struct {
	storePath  string
	rules      map[string]*Rule
	enabled    bool
	mu         sync.RWMutex
	nftPath    string
	configPath string
}

// NewManager creates a new firewall manager
func NewManager(storePath string) (*Manager, error) {
	m := &Manager{
		storePath:  storePath,
		rules:      make(map[string]*Rule),
		enabled:    false,
		configPath: "/etc/nftables.conf",
	}

	// Find nft binary
	nftPath, err := exec.LookPath("nft")
	if err != nil {
		// Try common locations
		for _, path := range []string{"/usr/sbin/nft", "/sbin/nft"} {
			if _, err := os.Stat(path); err == nil {
				nftPath = path
				break
			}
		}
		if nftPath == "" {
			return nil, fmt.Errorf("nftables not found")
		}
	}
	m.nftPath = nftPath

	// Load existing rules
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Check current status
	m.checkStatus()

	// Add default rules if none exist
	if len(m.rules) == 0 {
		m.addDefaultRules()
	}

	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rulesPath := filepath.Join(m.storePath, "firewall_rules.json")
	var rules []*Rule
	if ok, err := fsatomic.LoadJSON(rulesPath, &rules); err != nil {
		return err
	} else if ok {
		for _, rule := range rules {
			m.rules[rule.ID] = rule
		}
	}

	// Load enabled state
	statusPath := filepath.Join(m.storePath, "firewall_status.json")
	var status Status
	if ok, err := fsatomic.LoadJSON(statusPath, &status); err == nil && ok {
		m.enabled = status.Enabled
	}

	return nil
}

func (m *Manager) save() error {
	// Save rules
	rules := make([]*Rule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}

	rulesPath := filepath.Join(m.storePath, "firewall_rules.json")
	if err := fsatomic.SaveJSON(context.Background(), rulesPath, rules, 0600); err != nil {
		return err
	}

	// Save status
	status := Status{
		Enabled:     m.enabled,
		RuleCount:   len(m.rules),
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	statusPath := filepath.Join(m.storePath, "firewall_status.json")
	return fsatomic.SaveJSON(context.Background(), statusPath, status, 0600)
}

func (m *Manager) addDefaultRules() {
	// Allow established connections
	m.rules["default-established"] = &Rule{
		ID:        "default-established",
		Name:      "Allow Established Connections",
		Enabled:   true,
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "any",
		Comment:   "Allow already established connections",
		Priority:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Allow loopback
	m.rules["default-loopback"] = &Rule{
		ID:        "default-loopback",
		Name:      "Allow Loopback",
		Enabled:   true,
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "any",
		Interface: "lo",
		Comment:   "Allow loopback interface",
		Priority:  2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Allow SSH
	m.rules["default-ssh"] = &Rule{
		ID:        "default-ssh",
		Name:      "Allow SSH",
		Enabled:   true,
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "tcp",
		Destination: Address{
			Port: "22",
		},
		Comment:   "Allow SSH access",
		Priority:  10,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Allow HTTP/HTTPS
	m.rules["default-http"] = &Rule{
		ID:        "default-http",
		Name:      "Allow HTTP",
		Enabled:   true,
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "tcp",
		Destination: Address{
			Port: "80",
		},
		Comment:   "Allow HTTP access",
		Priority:  20,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.rules["default-https"] = &Rule{
		ID:        "default-https",
		Name:      "Allow HTTPS",
		Enabled:   true,
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "tcp",
		Destination: Address{
			Port: "443",
		},
		Comment:   "Allow HTTPS access",
		Priority:  21,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Allow ping
	m.rules["default-ping"] = &Rule{
		ID:        "default-ping",
		Name:      "Allow Ping",
		Enabled:   true,
		Direction: "inbound",
		Action:    "allow",
		Protocol:  "icmp",
		Comment:   "Allow ICMP ping",
		Priority:  30,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_ = m.save()
}

func (m *Manager) checkStatus() {
	// Check if nftables service is running
	cmd := exec.Command("systemctl", "is-active", "nftables")
	if err := cmd.Run(); err == nil {
		m.enabled = true
	}
}

// GetStatus returns firewall status
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Status{
		Enabled:     m.enabled,
		RuleCount:   len(m.rules),
		LastUpdated: time.Now(),
		Version:     "1.0",
	}
}

// SetEnabled enables or disables the firewall
func (m *Manager) SetEnabled(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if enabled {
		// Apply rules and start service
		if err := m.applyRules(); err != nil {
			return err
		}

		// Enable and start nftables service
		cmd := exec.Command("systemctl", "enable", "nftables")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable nftables: %w", err)
		}

		cmd = exec.Command("systemctl", "start", "nftables")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start nftables: %w", err)
		}
	} else {
		// Stop and disable service
		cmd := exec.Command("systemctl", "stop", "nftables")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop nftables: %w", err)
		}

		cmd = exec.Command("systemctl", "disable", "nftables")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disable nftables: %w", err)
		}

		// Flush rules
		cmd = exec.Command(m.nftPath, "flush", "ruleset")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to flush rules: %w", err)
		}
	}

	m.enabled = enabled
	return m.save()
}

// ListRules returns all firewall rules
func (m *Manager) ListRules() []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*Rule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}

	return rules
}

// GetRule returns a specific rule
func (m *Manager) GetRule(id string) (*Rule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, ok := m.rules[id]
	return rule, ok
}

// CreateRule creates a new firewall rule
func (m *Manager) CreateRule(rule *Rule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules[rule.ID] = rule

	if err := m.save(); err != nil {
		return err
	}

	// Apply rules if firewall is enabled
	if m.enabled {
		return m.applyRules()
	}

	return nil
}

// UpdateRule updates an existing rule
func (m *Manager) UpdateRule(id string, updates *Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, ok := m.rules[id]
	if !ok {
		return fmt.Errorf("rule not found")
	}

	// Update fields
	if updates.Name != "" {
		rule.Name = updates.Name
	}
	rule.Enabled = updates.Enabled
	rule.Direction = updates.Direction
	rule.Action = updates.Action
	rule.Protocol = updates.Protocol
	rule.Source = updates.Source
	rule.Destination = updates.Destination
	rule.Interface = updates.Interface
	rule.Priority = updates.Priority
	if updates.Comment != "" {
		rule.Comment = updates.Comment
	}
	rule.UpdatedAt = time.Now()

	if err := m.save(); err != nil {
		return err
	}

	// Apply rules if firewall is enabled
	if m.enabled {
		return m.applyRules()
	}

	return nil
}

// DeleteRule deletes a firewall rule
func (m *Manager) DeleteRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.rules[id]; !ok {
		return fmt.Errorf("rule not found")
	}

	delete(m.rules, id)

	if err := m.save(); err != nil {
		return err
	}

	// Apply rules if firewall is enabled
	if m.enabled {
		return m.applyRules()
	}

	return nil
}

// applyRules generates and applies nftables configuration
func (m *Manager) applyRules() error {
	config := m.generateConfig()

	// Write configuration to file
	tmpFile := m.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Test configuration
	cmd := exec.Command(m.nftPath, "-c", "-f", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("invalid configuration: %s", string(output))
	}

	// Move to final location
	if err := os.Rename(tmpFile, m.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Apply configuration
	cmd = exec.Command(m.nftPath, "-f", m.configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to apply rules: %s", string(output))
	}

	return nil
}

// generateConfig generates nftables configuration
func (m *Manager) generateConfig() string {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("#!/usr/sbin/nft -f\n")
	buf.WriteString("# NithronOS Firewall Configuration\n")
	buf.WriteString("# Generated: " + time.Now().Format(time.RFC3339) + "\n\n")

	// Flush existing rules
	buf.WriteString("flush ruleset\n\n")

	// Create table
	buf.WriteString("table inet filter {\n")

	// Create chains
	buf.WriteString("  chain input {\n")
	buf.WriteString("    type filter hook input priority 0; policy drop;\n")

	// Add established connections rule
	buf.WriteString("    ct state established,related accept\n")

	// Add loopback rule
	buf.WriteString("    iif lo accept\n")

	// Add inbound rules
	for _, rule := range m.rules {
		if rule.Enabled && rule.Direction == "inbound" {
			buf.WriteString(m.generateRule(rule))
		}
	}

	// Log dropped packets
	buf.WriteString("    log prefix \"[nftables] dropped: \" level info\n")

	buf.WriteString("  }\n\n")

	buf.WriteString("  chain forward {\n")
	buf.WriteString("    type filter hook forward priority 0; policy drop;\n")
	buf.WriteString("  }\n\n")

	buf.WriteString("  chain output {\n")
	buf.WriteString("    type filter hook output priority 0; policy accept;\n")

	// Add outbound rules
	for _, rule := range m.rules {
		if rule.Enabled && rule.Direction == "outbound" {
			buf.WriteString(m.generateRule(rule))
		}
	}

	buf.WriteString("  }\n")
	buf.WriteString("}\n")

	return buf.String()
}

// generateRule generates nftables rule syntax
func (m *Manager) generateRule(rule *Rule) string {
	var parts []string

	// Add comment
	if rule.Comment != "" {
		parts = append(parts, fmt.Sprintf("    # %s\n", rule.Comment))
	}

	// Build rule
	parts = append(parts, "    ")

	// Protocol
	if rule.Protocol != "any" {
		parts = append(parts, fmt.Sprintf("ip protocol %s", rule.Protocol))
	}

	// Source address
	if rule.Source.IP != "" {
		parts = append(parts, fmt.Sprintf("ip saddr %s", rule.Source.IP))
	}

	// Source port
	if rule.Source.Port != "" && (rule.Protocol == "tcp" || rule.Protocol == "udp") {
		parts = append(parts, fmt.Sprintf("%s sport %s", rule.Protocol, rule.Source.Port))
	}

	// Destination address
	if rule.Destination.IP != "" {
		parts = append(parts, fmt.Sprintf("ip daddr %s", rule.Destination.IP))
	}

	// Destination port
	if rule.Destination.Port != "" && (rule.Protocol == "tcp" || rule.Protocol == "udp") {
		parts = append(parts, fmt.Sprintf("%s dport %s", rule.Protocol, rule.Destination.Port))
	}

	// Interface
	if rule.Interface != "" {
		if rule.Direction == "inbound" {
			parts = append(parts, fmt.Sprintf("iif %s", rule.Interface))
		} else {
			parts = append(parts, fmt.Sprintf("oif %s", rule.Interface))
		}
	}

	// Action
	switch rule.Action {
	case "allow":
		parts = append(parts, "accept")
	case "deny":
		parts = append(parts, "drop")
	case "reject":
		parts = append(parts, "reject")
	}

	return strings.Join(parts, " ") + "\n"
}

// ExportRules exports rules to a file
func (m *Manager) ExportRules(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*Rule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}

	return fsatomic.SaveJSON(context.Background(), path, rules, 0644)
}

// ImportRules imports rules from a file
func (m *Manager) ImportRules(path string) error {
	var rules []*Rule
	ok, err := fsatomic.LoadJSON(path, &rules)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no rules found in file")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing rules
	m.rules = make(map[string]*Rule)

	// Import new rules
	for _, rule := range rules {
		if rule.ID == "" {
			rule.ID = uuid.New().String()
		}
		rule.UpdatedAt = time.Now()
		m.rules[rule.ID] = rule
	}

	if err := m.save(); err != nil {
		return err
	}

	// Apply rules if firewall is enabled
	if m.enabled {
		return m.applyRules()
	}

	return nil
}

// GetLogs returns recent firewall logs
func (m *Manager) GetLogs(lines int) ([]string, error) {
	// Read from system log
	cmd := exec.Command("journalctl", "-u", "nftables", "-n", fmt.Sprintf("%d", lines), "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	var logs []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}

	return logs, nil
}

// TestRule tests a firewall rule
func (m *Manager) TestRule(rule *Rule) error {
	// Generate a test configuration with just this rule
	testConfig := fmt.Sprintf(`#!/usr/sbin/nft -f
table inet test {
  chain test {
    type filter hook input priority 0; policy accept;
%s
  }
}`, m.generateRule(rule))

	// Write to temporary file
	tmpFile := "/tmp/nft-test-" + uuid.New().String()
	defer os.Remove(tmpFile)

	if err := os.WriteFile(tmpFile, []byte(testConfig), 0644); err != nil {
		return fmt.Errorf("failed to write test config: %w", err)
	}

	// Test configuration
	cmd := exec.Command(m.nftPath, "-c", "-f", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("invalid rule: %s", string(output))
	}

	return nil
}
