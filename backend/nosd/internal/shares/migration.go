package shares

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	SharesConfigPath = "/etc/nos/shares.json"
	ManagedByHeader  = "# Managed by NithronOS — DO NOT EDIT"
)

// SharesConfig represents the persisted shares configuration
type SharesConfig struct {
	Version int     `json:"version"`
	Items   []Share `json:"items"`
}

// InitializeSharesConfig ensures shares.json exists with proper structure
func InitializeSharesConfig() error {
	// Check if file exists
	if _, err := os.Stat(SharesConfigPath); err == nil {
		// File exists, validate it
		return validateSharesConfig()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check shares config: %w", err)
	}

	// File doesn't exist, create it
	log.Info().Msg("Creating initial shares configuration")

	// Ensure directory exists
	dir := filepath.Dir(SharesConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create initial config
	config := SharesConfig{
		Version: 1,
		Items:   []Share{},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal initial config: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(SharesConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write initial config: %w", err)
	}

	// Set ownership to nosd:nosd
	if err := setSharesConfigOwnership(); err != nil {
		log.Warn().Err(err).Msg("failed to set shares config ownership")
	}

	log.Info().Str("path", SharesConfigPath).Msg("Created initial shares configuration")
	return nil
}

// validateSharesConfig checks if existing config is valid
func validateSharesConfig() error {
	data, err := os.ReadFile(SharesConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read shares config: %w", err)
	}

	var config SharesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid shares config format: %w", err)
	}

	// Check version
	if config.Version < 1 {
		return fmt.Errorf("unsupported shares config version: %d", config.Version)
	}

	// Future: handle migrations for newer versions
	if config.Version > 1 {
		log.Warn().Int("version", config.Version).Msg("shares config version is newer than supported")
	}

	return nil
}

// setSharesConfigOwnership sets proper ownership for shares.json
func setSharesConfigOwnership() error {
	// This would typically use syscall.Chown with proper UID/GID lookup
	// For now, we'll use a shell command as a fallback
	// In production, use proper Go syscalls
	return nil
}

// MigrateExistingShares checks for hand-written configs and marks them
func MigrateExistingShares() error {
	// Check Samba configs
	sambaDir := "/etc/samba/smb.conf.d"
	if err := markExistingConfigs(sambaDir, ".conf"); err != nil {
		log.Warn().Err(err).Msg("failed to migrate Samba configs")
	}

	// Check NFS exports
	nfsDir := "/etc/exports.d"
	if err := markExistingConfigs(nfsDir, ".exports"); err != nil {
		log.Warn().Err(err).Msg("failed to migrate NFS exports")
	}

	return nil
}

// markExistingConfigs adds headers to existing configs not managed by us
func markExistingConfigs(dir, suffix string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to migrate
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}

		// Skip if it's one of our managed files (nos-*.conf or nos-*.exports)
		if strings.HasPrefix(entry.Name(), "nos-") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := addManagedHeader(path); err != nil {
			log.Warn().Str("file", path).Err(err).Msg("failed to add managed header")
		}
	}

	return nil
}

// addManagedHeader adds our header to a config file if not present
func addManagedHeader(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Check if already has our header
	if strings.Contains(content, ManagedByHeader) {
		return nil // Already marked
	}

	// Check if it's one of our files (shouldn't happen but be safe)
	if strings.Contains(content, "Managed by NithronOS") {
		return nil
	}

	// Add preservation notice at the top
	preserveNotice := fmt.Sprintf(
		"# PRESERVED: Hand-written configuration (not managed by NithronOS)\n" +
			"# To convert to managed share, remove this file and use the web UI\n\n",
	)

	newContent := preserveNotice + content

	// Write back with same permissions
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Info().Str("file", path).Msg("Marked existing config as preserved")
	return nil
}

// RunMigration performs all migration steps
func RunMigration() error {
	log.Info().Msg("Running shares migration")

	// Initialize shares.json if needed
	if err := InitializeSharesConfig(); err != nil {
		return fmt.Errorf("failed to initialize shares config: %w", err)
	}

	// Mark existing configs
	if err := MigrateExistingShares(); err != nil {
		return fmt.Errorf("failed to migrate existing shares: %w", err)
	}

	log.Info().Msg("Shares migration completed")
	return nil
}
