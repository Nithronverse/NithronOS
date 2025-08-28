package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LifecycleManager handles app lifecycle operations
type LifecycleManager struct {
	catalogMgr   *CatalogManager
	stateStore   *StateStore
	renderer     *TemplateRenderer
	appsRoot     string
	agentPath    string
	helperPath   string
	snapshotPath string
	caddyPath    string
	eventLogger  EventLogger
}

// EventLogger interface for logging events
type EventLogger interface {
	LogEvent(event Event) error
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(
	catalogMgr *CatalogManager,
	stateStore *StateStore,
	renderer *TemplateRenderer,
	appsRoot string,
	agentPath string,
	eventLogger EventLogger,
) *LifecycleManager {
	return &LifecycleManager{
		catalogMgr:   catalogMgr,
		stateStore:   stateStore,
		renderer:     renderer,
		appsRoot:     appsRoot,
		agentPath:    agentPath,
		helperPath:   "/usr/lib/nos/apps/nos-app-helper.sh",
		snapshotPath: "/usr/lib/nos/apps/nos-app-snapshot.sh",
		caddyPath:    "/etc/caddy/Caddyfile.d",
		eventLogger:  eventLogger,
	}
}

// InstallApp installs a new application
func (lm *LifecycleManager) InstallApp(ctx context.Context, req InstallRequest, userID string) error {
	// Get catalog entry
	entry, err := lm.catalogMgr.GetEntry(req.ID)
	if err != nil {
		return fmt.Errorf("app not found in catalog: %w", err)
	}

	// Check if already installed
	if _, err := lm.stateStore.GetApp(req.ID); err == nil {
		return fmt.Errorf("app already installed: %s", req.ID)
	}

	// Validate parameters
	if err := lm.renderer.ValidateParams(entry, req.Params); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	// Log installation start event
	lm.logEvent("app.install.start", req.ID, userID, map[string]interface{}{
		"version": entry.Version,
		"params":  req.Params,
	})

	// Create app directories
	appDir := filepath.Join(lm.appsRoot, req.ID)
	configDir := filepath.Join(appDir, "config")
	// dataDir := filepath.Join(appDir, "data") // TODO: Use for data persistence

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure data directory is a subvolume if on Btrfs
	if err := lm.ensureDataSubvolume(req.ID); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Render compose file
	composeContent, err := lm.renderer.RenderComposeFile(entry, req.Params)
	if err != nil {
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to render compose file: %w", err)
	}

	composePath := filepath.Join(configDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, composeContent, 0600); err != nil {
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Render environment file
	envContent, err := lm.renderer.RenderEnvFile(req.Params)
	if err != nil {
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to render env file: %w", err)
	}

	envPath := filepath.Join(configDir, ".env")
	if err := os.WriteFile(envPath, envContent, 0600); err != nil {
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to write env file: %w", err)
	}

	// Set ownership
	if err := lm.setAppOwnership(appDir); err != nil {
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	// Create initial snapshot
	snapshotID, err := lm.createSnapshot(req.ID, "post-install")
	if err != nil {
		// Log warning but continue
		fmt.Fprintf(os.Stderr, "Warning: failed to create post-install snapshot: %v\n", err)
	}

	// Start the app
	if err := lm.startApp(ctx, req.ID); err != nil {
		// Rollback on failure
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to start app: %w", err)
	}

	// Setup reverse proxy if ports are exposed
	if len(entry.Defaults.Ports) > 0 {
		if err := lm.setupReverseProxy(req.ID, entry.Defaults.Ports); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to setup reverse proxy: %v\n", err)
		}
	}

	// Save app state
	app := InstalledApp{
		ID:      req.ID,
		Name:    entry.Name,
		Version: entry.Version,
		Status:  StatusRunning,
		Params:  req.Params,
		Ports:   entry.Defaults.Ports,
		URLs:    lm.generateAppURLs(req.ID, entry.Defaults.Ports),
		Health: HealthStatus{
			Status:    "unknown",
			CheckedAt: time.Now(),
		},
		Snapshots: []AppSnapshot{},
	}

	if snapshotID != "" {
		app.Snapshots = append(app.Snapshots, AppSnapshot{
			ID:        snapshotID,
			Timestamp: time.Now(),
			Type:      "btrfs",
			Name:      "post-install",
			Path:      filepath.Join(lm.appsRoot, ".snapshots", req.ID, snapshotID),
		})
	}

	if err := lm.stateStore.AddApp(app); err != nil {
		// Try to clean up
		if stopErr := lm.stopApp(ctx, req.ID); stopErr != nil {
			fmt.Printf("Failed to stop app during cleanup: %v\n", stopErr)
		}
		os.RemoveAll(appDir)
		return fmt.Errorf("failed to save app state: %w", err)
	}

	// Log installation complete event
	lm.logEvent("app.install.complete", req.ID, userID, map[string]interface{}{
		"version": entry.Version,
	})

	// Run post-install script if exists
	if err := lm.runPostInstallScript(entry, appDir); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: post-install script failed: %v\n", err)
	}

	return nil
}

// UpgradeApp upgrades an existing application
func (lm *LifecycleManager) UpgradeApp(ctx context.Context, appID string, req UpgradeRequest, userID string) error {
	// Get current app
	app, err := lm.stateStore.GetApp(appID)
	if err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	// Get catalog entry
	entry, err := lm.catalogMgr.GetEntry(appID)
	if err != nil {
		return fmt.Errorf("app not found in catalog: %w", err)
	}

	// Log upgrade start event
	lm.logEvent("app.upgrade.start", appID, userID, map[string]interface{}{
		"from_version": app.Version,
		"to_version":   req.Version,
	})

	// Create pre-upgrade snapshot
	snapshotID, err := lm.createSnapshot(appID, "pre-upgrade")
	if err != nil {
		return fmt.Errorf("failed to create pre-upgrade snapshot: %w", err)
	}

	// Update app state to upgrading
	if err := lm.stateStore.UpdateAppStatus(appID, StatusUpgrading); err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	// Merge params if provided
	params := app.Params
	if req.Params != nil {
		for k, v := range req.Params {
			params[k] = v
		}
	}

	// Validate new parameters
	if err := lm.renderer.ValidateParams(entry, params); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	// Render new compose file
	configDir := filepath.Join(lm.appsRoot, appID, "config")
	composeContent, err := lm.renderer.RenderComposeFile(entry, params)
	if err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("failed to render compose file: %w", err)
	}

	// Backup current config
	composePath := filepath.Join(configDir, "docker-compose.yml")
	backupPath := filepath.Join(configDir, "docker-compose.yml.backup")
	if err := lm.copyFile(composePath, backupPath); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("failed to backup compose file: %w", err)
	}

	// Write new compose file
	if err := os.WriteFile(composePath, composeContent, 0600); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Pull new images
	if err := lm.pullImages(ctx, appID); err != nil {
		// Restore backup
		if err := lm.copyFile(backupPath, composePath); err != nil {
			fmt.Printf("Failed to restore compose file: %v\n", err)
		}
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// Restart app with new configuration
	if err := lm.restartApp(ctx, appID); err != nil {
		// Rollback on failure
		if err := lm.copyFile(backupPath, composePath); err != nil {
			fmt.Printf("Failed to restore compose file: %v\n", err)
		}
		if err := lm.rollbackSnapshot(ctx, appID, snapshotID); err != nil {
			fmt.Printf("Failed to rollback snapshot: %v\n", err)
		}
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("failed to restart app: %w", err)
	}

	// Wait for health check
	healthy := lm.waitForHealth(ctx, appID, 60*time.Second)
	if !healthy {
		// Rollback if unhealthy
		fmt.Fprintf(os.Stderr, "App unhealthy after upgrade, rolling back...\n")
		if err := lm.copyFile(backupPath, composePath); err != nil {
			fmt.Printf("Failed to restore compose file: %v\n", err)
		}
		if err := lm.rollbackSnapshot(ctx, appID, snapshotID); err != nil {
			fmt.Printf("Failed to rollback snapshot: %v\n", err)
		}
		if err := lm.restartApp(ctx, appID); err != nil {
			fmt.Printf("Failed to restart app after update: %v\n", err)
		}
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return fmt.Errorf("app unhealthy after upgrade, rolled back")
	}

	// Update app state
	app.Version = req.Version
	app.Params = params
	app.Status = StatusRunning
	if err := lm.stateStore.UpdateApp(*app); err != nil {
		return fmt.Errorf("failed to update app state: %w", err)
	}

	// Log upgrade complete event
	lm.logEvent("app.upgrade.complete", appID, userID, map[string]interface{}{
		"version": req.Version,
	})

	// Clean up backup
	os.Remove(backupPath)

	return nil
}

// StartApp starts an application
func (lm *LifecycleManager) StartApp(ctx context.Context, appID string, userID string) error {
	if err := lm.stateStore.UpdateAppStatus(appID, StatusStarting); err != nil {
		return err
	}

	lm.logEvent("app.start", appID, userID, nil)

	if err := lm.startApp(ctx, appID); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return err
	}

	return lm.stateStore.UpdateAppStatus(appID, StatusRunning)
}

// StopApp stops an application
func (lm *LifecycleManager) StopApp(ctx context.Context, appID string, userID string) error {
	if err := lm.stateStore.UpdateAppStatus(appID, StatusStopping); err != nil {
		return err
	}

	lm.logEvent("app.stop", appID, userID, nil)

	if err := lm.stopApp(ctx, appID); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return err
	}

	return lm.stateStore.UpdateAppStatus(appID, StatusStopped)
}

// RestartApp restarts an application
func (lm *LifecycleManager) RestartApp(ctx context.Context, appID string, userID string) error {
	lm.logEvent("app.restart", appID, userID, nil)
	return lm.restartApp(ctx, appID)
}

// DeleteApp deletes an application
func (lm *LifecycleManager) DeleteApp(ctx context.Context, appID string, keepData bool, userID string) error {
	// Stop the app first
	if err := lm.stopApp(ctx, appID); err != nil {
		fmt.Printf("Failed to stop app during uninstall: %v\n", err)
	}

	lm.logEvent("app.delete", appID, userID, map[string]interface{}{
		"keep_data": keepData,
	})

	// Remove from systemd
	if err := lm.disableSystemdService(appID); err != nil {
		fmt.Printf("Failed to disable systemd service: %v\n", err)
	}

	// Remove Caddy configuration
	caddyPath := filepath.Join(lm.caddyPath, fmt.Sprintf("app-%s.caddy", appID))
	os.Remove(caddyPath)
	if err := lm.reloadCaddy(); err != nil {
		fmt.Printf("Failed to reload Caddy after app removal: %v\n", err)
	}

	// Remove app directory if not keeping data
	if !keepData {
		appDir := filepath.Join(lm.appsRoot, appID)
		if err := lm.removeAppDirectory(appDir); err != nil {
			return fmt.Errorf("failed to remove app directory: %w", err)
		}

		// Remove snapshots
		snapshotDir := filepath.Join(lm.appsRoot, ".snapshots", appID)
		if err := lm.removeAppDirectory(snapshotDir); err != nil {
			fmt.Printf("Failed to remove snapshot directory: %v\n", err)
		}
	}

	// Remove from state
	return lm.stateStore.DeleteApp(appID)
}

// RollbackApp rolls back an app to a snapshot
func (lm *LifecycleManager) RollbackApp(ctx context.Context, appID string, snapshotTS string, userID string) error {
	lm.logEvent("app.rollback", appID, userID, map[string]interface{}{
		"snapshot": snapshotTS,
	})

	if err := lm.stateStore.UpdateAppStatus(appID, StatusRollback); err != nil {
		return err
	}

	if err := lm.rollbackSnapshot(ctx, appID, snapshotTS); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return err
	}

	// Restart app after rollback
	if err := lm.restartApp(ctx, appID); err != nil {
		if err := lm.stateStore.UpdateAppStatus(appID, StatusError); err != nil {
			fmt.Printf("Failed to update app status: %v\n", err)
		}
		return err
	}

	return lm.stateStore.UpdateAppStatus(appID, StatusRunning)
}

// Helper methods

func (lm *LifecycleManager) startApp(ctx context.Context, appID string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "start", fmt.Sprintf("nos-app@%s.service", appID))
	return cmd.Run()
}

func (lm *LifecycleManager) stopApp(ctx context.Context, appID string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "stop", fmt.Sprintf("nos-app@%s.service", appID))
	return cmd.Run()
}

func (lm *LifecycleManager) restartApp(ctx context.Context, appID string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "restart", fmt.Sprintf("nos-app@%s.service", appID))
	return cmd.Run()
}

func (lm *LifecycleManager) disableSystemdService(appID string) error {
	cmd := exec.Command("systemctl", "disable", fmt.Sprintf("nos-app@%s.service", appID))
	return cmd.Run()
}

func (lm *LifecycleManager) pullImages(ctx context.Context, appID string) error {
	configDir := filepath.Join(lm.appsRoot, appID, "config")
	cmd := exec.CommandContext(ctx, lm.helperPath, "compose-pull", configDir)
	return cmd.Run()
}

func (lm *LifecycleManager) ensureDataSubvolume(appID string) error {
	cmd := exec.Command(lm.snapshotPath, "ensure-subvolume", appID)
	return cmd.Run()
}

func (lm *LifecycleManager) createSnapshot(appID, name string) (string, error) {
	cmd := exec.Command(lm.snapshotPath, "snapshot-pre", appID, name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Extract snapshot ID from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/srv/apps/.snapshots/") {
			parts := strings.Split(line, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1], nil
			}
		}
	}

	return "", fmt.Errorf("snapshot ID not found in output")
}

func (lm *LifecycleManager) rollbackSnapshot(ctx context.Context, appID, snapshotTS string) error {
	cmd := exec.CommandContext(ctx, lm.snapshotPath, "rollback", appID, snapshotTS)
	return cmd.Run()
}

func (lm *LifecycleManager) setAppOwnership(appDir string) error {
	cmd := exec.Command("chown", "-R", "nos:nos", appDir)
	return cmd.Run()
}

func (lm *LifecycleManager) setupReverseProxy(appID string, ports []PortMapping) error {
	// Generate Caddy snippet
	snippet, err := lm.renderer.RenderCaddySnippet(appID, ports)
	if err != nil {
		return err
	}

	if snippet == nil {
		return nil // No proxy needed
	}

	// Write to Caddyfile.d
	snippetPath := filepath.Join(lm.caddyPath, fmt.Sprintf("app-%s.caddy", appID))
	if err := os.WriteFile(snippetPath, snippet, 0644); err != nil {
		return fmt.Errorf("failed to write Caddy snippet: %w", err)
	}

	// Reload Caddy
	return lm.reloadCaddy()
}

func (lm *LifecycleManager) reloadCaddy() error {
	cmd := exec.Command("systemctl", "reload", "caddy")
	return cmd.Run()
}

func (lm *LifecycleManager) generateAppURLs(appID string, ports []PortMapping) []string {
	urls := []string{}

	// Add path-based URL
	urls = append(urls, fmt.Sprintf("/apps/%s/", appID))

	// Add port-based URLs if any
	for _, port := range ports {
		if port.Protocol == "tcp" {
			urls = append(urls, fmt.Sprintf("http://localhost:%d", port.Host))
		}
	}

	return urls
}

func (lm *LifecycleManager) waitForHealth(ctx context.Context, appID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check container status
		cmd := exec.CommandContext(ctx, lm.helperPath, "app-status", appID)
		output, err := cmd.Output()
		if err == nil {
			var status map[string]interface{}
			if json.Unmarshal(output, &status) == nil {
				if status["status"] == "running" {
					return true
				}
			}
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(2 * time.Second):
			// Continue checking
		}
	}

	return false
}

func (lm *LifecycleManager) copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600)
}

func (lm *LifecycleManager) removeAppDirectory(dir string) error {
	// Check if it's a Btrfs subvolume
	cmd := exec.Command(lm.snapshotPath, "is-btrfs", dir)
	output, _ := cmd.Output()

	if strings.TrimSpace(string(output)) == "yes" {
		// Remove as Btrfs subvolume
		cmd = exec.Command("btrfs", "subvolume", "delete", dir)
		if err := cmd.Run(); err != nil {
			// Fall back to regular removal
			return os.RemoveAll(dir)
		}
		return nil
	}

	return os.RemoveAll(dir)
}

func (lm *LifecycleManager) runPostInstallScript(entry *CatalogEntry, appDir string) error {
	scriptPath := filepath.Join(lm.catalogMgr.builtinPath, "templates", entry.ID, "post_install.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil // No script
	}

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("APP_ID=%s", entry.ID),
		fmt.Sprintf("APP_DIR=%s", appDir),
	)

	return cmd.Run()
}

func (lm *LifecycleManager) logEvent(eventType, appID, userID string, details interface{}) {
	if lm.eventLogger == nil {
		return
	}

	detailsJSON, _ := json.Marshal(details)

	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		AppID:     appID,
		Timestamp: time.Now(),
		User:      userID,
		Details:   detailsJSON,
	}

	if err := lm.eventLogger.LogEvent(event); err != nil {
		fmt.Printf("Failed to log event: %v\n", err)
	}
}
