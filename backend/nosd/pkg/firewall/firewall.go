package firewall

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Status struct {
	Mode             string `json:"mode"`
	NFTPresent       bool   `json:"nft_present"`
	UFWPresent       bool   `json:"ufw_present"`
	FirewalldPresent bool   `json:"firewalld_present"`
	LastAppliedAt    string `json:"last_applied_at,omitempty"`
}

func Detect() (Status, error) {
	s := Status{}
	s.NFTPresent = hasCmd("nft")
	s.UFWPresent = isActive("ufw")
	s.FirewalldPresent = isActive("firewalld")
	return s, nil
}

func hasCmd(name string) bool { _, err := exec.LookPath(name); return err == nil }

func isActive(unit string) bool {
	if !hasCmd("systemctl") {
		return false
	}
	cmd := exec.Command("systemctl", "is-active", unit)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// BuildRules returns an nftables ruleset for the given mode.
// Modes: lan-only, vpn-only, tunnel, direct (for now, lan-only and others use same base with RFC1918 allowances)
func BuildRules(mode string) string {
	base := `table inet filter {
  chains {
    input {
      type filter hook input priority 0; policy drop;
      ct state established,related accept
      iif lo accept
      ip protocol icmp accept
      ip6 nexthdr icmpv6 accept
      ip saddr { 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 } tcp dport { 22, 443 } accept
      ip6 saddr fc00::/7 tcp dport { 22, 443 } accept
    }
    forward { type filter hook forward priority 0; policy drop; }
    output { type filter hook output priority 0; policy accept; }
  }
}`
	// In the future, vary by mode
	switch strings.ToLower(mode) {
	case "lan-only", "vpn-only", "tunnel", "direct", "":
		return base
	default:
		return base
	}
}

// Mode storage helpers
func ReadMode(modePath string) (string, error) {
	b, err := os.ReadFile(modePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func WriteMode(modePath, mode string) error {
	if err := os.MkdirAll(filepath.Dir(modePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(modePath, []byte(strings.ToLower(mode)), 0o644)
}

func BackupPath(baseDir string) string {
	ts := time.Now().UTC().Format("20060102-150405")
	return filepath.Join(baseDir, fmt.Sprintf("backup-%s.nft", ts))
}
