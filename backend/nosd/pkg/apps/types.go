package apps

import (
	"encoding/json"
	"time"
)

// CatalogEntry represents an app in the catalog
type CatalogEntry struct {
	ID              string       `json:"id" yaml:"id"`
	Name            string       `json:"name" yaml:"name"`
	Version         string       `json:"version" yaml:"version"`
	Description     string       `json:"description" yaml:"description"`
	Categories      []string     `json:"categories" yaml:"categories"`
	Icon            string       `json:"icon" yaml:"icon"`
	Compose         string       `json:"compose" yaml:"compose"`
	Schema          string       `json:"schema" yaml:"schema"`
	Defaults        AppDefaults  `json:"defaults" yaml:"defaults"`
	Health          HealthConfig `json:"health" yaml:"health"`
	NeedsPrivileged bool         `json:"needs_privileged" yaml:"needs_privileged"`
	Notes           string       `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// AppDefaults contains default configuration for an app
type AppDefaults struct {
	Env       map[string]string `json:"env" yaml:"env"`
	Volumes   []VolumeMount     `json:"volumes" yaml:"volumes"`
	Ports     []PortMapping     `json:"ports" yaml:"ports"`
	Resources ResourceLimits    `json:"resources" yaml:"resources"`
}

// VolumeMount represents a volume mount configuration
type VolumeMount struct {
	Host      string `json:"host" yaml:"host"`
	Container string `json:"container" yaml:"container"`
	ReadOnly  bool   `json:"read_only,omitempty" yaml:"read_only,omitempty"`
}

// PortMapping represents a port mapping configuration
type PortMapping struct {
	Host      int    `json:"host" yaml:"host"`
	Container int    `json:"container" yaml:"container"`
	Protocol  string `json:"protocol" yaml:"protocol"` // tcp or udp
}

// ResourceLimits defines resource constraints for an app
type ResourceLimits struct {
	CPULimit    string `json:"cpu_limit,omitempty" yaml:"cpu_limit,omitempty"`       // e.g., "2.0"
	MemoryLimit string `json:"memory_limit,omitempty" yaml:"memory_limit,omitempty"` // e.g., "512m"
	CPURequest  string `json:"cpu_request,omitempty" yaml:"cpu_request,omitempty"`
	MemRequest  string `json:"mem_request,omitempty" yaml:"mem_request,omitempty"`
}

// HealthConfig defines health check configuration
type HealthConfig struct {
	Type           string `json:"type" yaml:"type"` // "container" or "http"
	Container      string `json:"container,omitempty" yaml:"container,omitempty"`
	URL            string `json:"url,omitempty" yaml:"url,omitempty"`
	IntervalSec    int    `json:"interval_s" yaml:"interval_s"`
	TimeoutSec     int    `json:"timeout_s" yaml:"timeout_s"`
	HealthyAfter   int    `json:"healthy_after" yaml:"healthy_after"`
	UnhealthyAfter int    `json:"unhealthy_after" yaml:"unhealthy_after"`
}

// Catalog represents the complete app catalog
type Catalog struct {
	Version   string         `json:"version" yaml:"version"`
	Entries   []CatalogEntry `json:"entries" yaml:"entries"`
	Source    string         `json:"source,omitempty" yaml:"source,omitempty"`
	UpdatedAt time.Time      `json:"updated_at" yaml:"updated_at"`
}

// InstalledApp represents an installed application
type InstalledApp struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Status      AppStatus              `json:"status"`
	Params      map[string]interface{} `json:"params"`
	Ports       []PortMapping          `json:"ports"`
	URLs        []string               `json:"urls"`
	Health      HealthStatus           `json:"health"`
	InstalledAt time.Time              `json:"installed_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Snapshots   []AppSnapshot          `json:"snapshots"`
}

// AppStatus represents the current status of an app
type AppStatus string

const (
	StatusStopped   AppStatus = "stopped"
	StatusStarting  AppStatus = "starting"
	StatusRunning   AppStatus = "running"
	StatusStopping  AppStatus = "stopping"
	StatusError     AppStatus = "error"
	StatusUpgrading AppStatus = "upgrading"
	StatusRollback  AppStatus = "rollback"
	StatusUnknown   AppStatus = "unknown"
)

// HealthStatus represents the health status of an app
type HealthStatus struct {
	Status     string            `json:"status"` // "healthy", "unhealthy", "unknown"
	CheckedAt  time.Time         `json:"checked_at"`
	Message    string            `json:"message,omitempty"`
	Containers []ContainerHealth `json:"containers,omitempty"`
}

// ContainerHealth represents health of a single container
type ContainerHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Health string `json:"health,omitempty"`
}

// AppSnapshot represents a snapshot of app data
type AppSnapshot struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "btrfs" or "rsync"
	Name      string    `json:"name"` // e.g., "pre-upgrade"
	Path      string    `json:"path"`
}

// InstallRequest represents a request to install an app
type InstallRequest struct {
	ID      string                 `json:"id" validate:"required,alphanum"`
	Version string                 `json:"version,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// UpgradeRequest represents a request to upgrade an app
type UpgradeRequest struct {
	Version string                 `json:"version" validate:"required"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// RollbackRequest represents a request to rollback an app
type RollbackRequest struct {
	SnapshotTimestamp string `json:"snapshot_ts" validate:"required"`
}

// DeleteRequest represents a request to delete an app
type DeleteRequest struct {
	KeepData bool `json:"keep_data"`
}

// AppState represents the persistent state of all apps
type AppState struct {
	Version   string         `json:"version"`
	Apps      []InstalledApp `json:"items"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CatalogSource represents a remote catalog source
type CatalogSource struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"` // "git" or "http"
	URL       string `yaml:"url"`
	Branch    string `yaml:"branch,omitempty"`
	SHA256    string `yaml:"sha256,omitempty"`
	Signature string `yaml:"signature,omitempty"`
	Enabled   bool   `yaml:"enabled"`
}

// Event represents an app lifecycle event
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	AppID     string          `json:"app_id"`
	Timestamp time.Time       `json:"timestamp"`
	User      string          `json:"user,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
}

// LogStreamOptions configures log streaming
type LogStreamOptions struct {
	Follow     bool   `json:"follow"`
	Tail       int    `json:"tail"`
	Timestamps bool   `json:"timestamps"`
	Container  string `json:"container,omitempty"`
}
