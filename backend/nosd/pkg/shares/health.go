package shares

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// HealthCheck represents the result of a service health check
type HealthCheck struct {
	Service string   `json:"service"`
	Healthy bool     `json:"healthy"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

// CheckSambaHealth tests the Samba configuration
func CheckSambaHealth() *HealthCheck {
	result := &HealthCheck{
		Service: "samba",
		Healthy: true,
	}

	// Test configuration with testparm
	cmd := exec.Command("testparm", "-s", "--suppress-prompt")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Healthy = false
		result.Message = "Samba configuration invalid"
		result.Errors = []string{stderr.String()}
		return result
	}

	// Check if service is running
	cmd = exec.Command("systemctl", "is-active", "smbd")
	output, _ := cmd.Output()
	if strings.TrimSpace(string(output)) != "active" {
		result.Healthy = false
		result.Message = "Samba service not running"
		return result
	}

	result.Message = "Samba configuration valid and service running"
	return result
}

// CheckNFSHealth tests the NFS configuration
func CheckNFSHealth() *HealthCheck {
	result := &HealthCheck{
		Service: "nfs",
		Healthy: true,
	}

	// Check exports syntax (dry-run)
	cmd := exec.Command("exportfs", "-ra")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run exportfs to check syntax
	if err := cmd.Run(); err != nil {
		result.Healthy = false
		result.Message = "NFS exports configuration invalid"
		result.Errors = []string{stderr.String()}
		return result
	}

	// Check if service is running
	cmd = exec.Command("systemctl", "is-active", "nfs-server")
	output, _ := cmd.Output()
	if strings.TrimSpace(string(output)) != "active" {
		result.Healthy = false
		result.Message = "NFS server not running"
		return result
	}

	// List current exports
	cmd = exec.Command("exportfs", "-v")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		result.Message = fmt.Sprintf("NFS server healthy with %d exports",
			len(strings.Split(string(output), "\n"))-1)
	} else {
		result.Message = "NFS server healthy (no exports)"
	}

	return result
}

// CheckAvahiHealth tests the Avahi daemon
func CheckAvahiHealth() *HealthCheck {
	result := &HealthCheck{
		Service: "avahi",
		Healthy: true,
	}

	// Check if service is running
	cmd := exec.Command("systemctl", "is-active", "avahi-daemon")
	output, _ := cmd.Output()
	if strings.TrimSpace(string(output)) != "active" {
		result.Healthy = false
		result.Message = "Avahi daemon not running"
		return result
	}

	// Check if we can browse services
	cmd = exec.Command("avahi-browse", "-a", "-t", "-r", "-p")
	if err := cmd.Run(); err != nil {
		result.Healthy = false
		result.Message = "Avahi browse failed"
		result.Errors = []string{err.Error()}
		return result
	}

	result.Message = "Avahi daemon healthy"
	return result
}

// ReloadSambaServices reloads or restarts SMB services
func ReloadSambaServices() error {
	// First test configuration
	cmd := exec.Command("testparm", "-s", "--suppress-prompt")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Samba configuration invalid: %w", err)
	}

	// Reload smbd
	cmd = exec.Command("systemctl", "reload-or-restart", "smbd")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload smbd: %w", err)
	}

	// Reload nmbd if it exists
	cmd = exec.Command("systemctl", "reload-or-restart", "nmbd")
	_ = cmd.Run() // Ignore error as nmbd might not be installed

	return nil
}

// ReloadNFSServices reloads NFS exports and restarts the service
func ReloadNFSServices() error {
	// Re-export all filesystems
	cmd := exec.Command("exportfs", "-ra")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload NFS exports: %w", err)
	}

	// Reload NFS server
	cmd = exec.Command("systemctl", "reload-or-restart", "nfs-server")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload NFS server: %w", err)
	}

	return nil
}

// ReloadAvahiService reloads the Avahi daemon
func ReloadAvahiService() error {
	cmd := exec.Command("systemctl", "reload-or-restart", "avahi-daemon")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload Avahi: %w", err)
	}
	return nil
}
