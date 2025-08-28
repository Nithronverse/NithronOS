package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// HealthMonitor monitors app health status
type HealthMonitor struct {
	stateStore  *StateStore
	catalogMgr  *CatalogManager
	helperPath  string
	httpClient  *http.Client
	interval    time.Duration
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	healthCache map[string]HealthStatus
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(stateStore *StateStore, catalogMgr *CatalogManager) *HealthMonitor {
	return &HealthMonitor{
		stateStore: stateStore,
		catalogMgr: catalogMgr,
		helperPath: "/usr/lib/nos/apps/nos-app-helper.sh",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		interval:    10 * time.Second,
		healthCache: make(map[string]HealthStatus),
		stopCh:      make(chan struct{}),
	}
}

// Start begins health monitoring
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.mu.Lock()
	if hm.running {
		hm.mu.Unlock()
		return fmt.Errorf("health monitor already running")
	}
	hm.running = true
	hm.mu.Unlock()

	go hm.monitorLoop(ctx)
	return nil
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		close(hm.stopCh)
		hm.running = false
	}
}

// monitorLoop is the main monitoring loop
func (hm *HealthMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// Initial check
	hm.checkAllApps(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.checkAllApps(ctx)
		}
	}
}

// checkAllApps checks health of all installed apps
func (hm *HealthMonitor) checkAllApps(ctx context.Context) {
	apps := hm.stateStore.GetAllApps()

	var wg sync.WaitGroup
	for _, app := range apps {
		if app.Status == StatusRunning || app.Status == StatusStarting {
			wg.Add(1)
			go func(a InstalledApp) {
				defer wg.Done()
				hm.checkAppHealth(ctx, a)
			}(app)
		}
	}

	wg.Wait()
}

// checkAppHealth checks health of a single app
func (hm *HealthMonitor) checkAppHealth(ctx context.Context, app InstalledApp) {
	health := HealthStatus{
		Status:    "unknown",
		CheckedAt: time.Now(),
	}

	// Get catalog entry for health config
	entry, err := hm.catalogMgr.GetEntry(app.ID)
	if err != nil {
		health.Status = "unknown"
		health.Message = fmt.Sprintf("Catalog entry not found: %v", err)
		hm.updateHealth(app.ID, health)
		return
	}

	// Check container status
	containerHealth, err := hm.getContainerHealth(ctx, app.ID)
	if err != nil {
		health.Status = "unhealthy"
		health.Message = fmt.Sprintf("Failed to get container status: %v", err)
		hm.updateHealth(app.ID, health)
		return
	}

	health.Containers = containerHealth

	// Determine overall health from containers
	allHealthy := true
	anyRunning := false
	for _, c := range containerHealth {
		if c.Status == "running" {
			anyRunning = true
			if c.Health != "healthy" && c.Health != "" {
				allHealthy = false
			}
		} else {
			allHealthy = false
		}
	}

	if !anyRunning {
		health.Status = "unhealthy"
		health.Message = "No containers running"
		hm.updateHealth(app.ID, health)
		return
	}

	// Perform HTTP health check if configured
	if entry.Health.Type == "http" && entry.Health.URL != "" {
		httpHealthy := hm.checkHTTPHealth(ctx, app.ID, entry.Health)
		if !httpHealthy {
			allHealthy = false
			health.Message = "HTTP health check failed"
		}
	}

	// Set final status
	if allHealthy {
		health.Status = "healthy"
		health.Message = "All checks passing"
	} else {
		health.Status = "unhealthy"
	}

	hm.updateHealth(app.ID, health)
}

// getContainerHealth gets container health status
func (hm *HealthMonitor) getContainerHealth(ctx context.Context, appID string) ([]ContainerHealth, error) {
	cmd := exec.CommandContext(ctx, hm.helperPath, "compose-ps", fmt.Sprintf("/srv/apps/%s/config", appID))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %w", err)
	}

	// Parse JSON output
	var containers []map[string]interface{}
	if err := json.Unmarshal(output, &containers); err != nil {
		// Try to parse as single container
		var container map[string]interface{}
		if err := json.Unmarshal(output, &container); err != nil {
			return nil, fmt.Errorf("failed to parse container status: %w", err)
		}
		containers = []map[string]interface{}{container}
	}

	result := []ContainerHealth{}
	for _, c := range containers {
		health := ContainerHealth{
			Name:   getString(c, "Name"),
			Status: getString(c, "State"),
		}

		// Get health status if available
		if healthStatus, ok := c["Health"].(map[string]interface{}); ok {
			health.Health = getString(healthStatus, "Status")
		}

		result = append(result, health)
	}

	return result, nil
}

// checkHTTPHealth performs HTTP health check
func (hm *HealthMonitor) checkHTTPHealth(ctx context.Context, appID string, config HealthConfig) bool {
	// Replace container name in URL with actual container name
	url := strings.ReplaceAll(config.URL, "${CONTAINER}", fmt.Sprintf("nos-app-%s-%s-1", appID, config.Container))

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	// Perform request
	resp, err := hm.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check status code
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// updateHealth updates health status in cache and state
func (hm *HealthMonitor) updateHealth(appID string, health HealthStatus) {
	hm.mu.Lock()
	hm.healthCache[appID] = health
	hm.mu.Unlock()

	// Update in state store
	if err := hm.stateStore.UpdateAppHealth(appID, health); err != nil {
		fmt.Printf("Failed to update app health for %s: %v\n", appID, err)
	}
}

// GetHealth returns cached health status for an app
func (hm *HealthMonitor) GetHealth(appID string) (HealthStatus, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, ok := hm.healthCache[appID]
	return health, ok
}

// GetAllHealth returns health status for all apps
func (hm *HealthMonitor) GetAllHealth() map[string]HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]HealthStatus)
	for k, v := range hm.healthCache {
		result[k] = v
	}
	return result
}

// ForceCheck forces an immediate health check for an app
func (hm *HealthMonitor) ForceCheck(ctx context.Context, appID string) error {
	app, err := hm.stateStore.GetApp(appID)
	if err != nil {
		return err
	}

	hm.checkAppHealth(ctx, *app)
	return nil
}

// Helper function to safely get string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
