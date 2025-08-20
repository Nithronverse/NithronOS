package snapcfg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type TargetType string

const (
	TargetTypeBtrfs TargetType = "btrfs"
	TargetTypeAuto  TargetType = "auto"
	TargetTypeTar   TargetType = "tar"
)

type RawTarget struct {
	ID           string     `yaml:"id"`
	Path         string     `yaml:"path"`
	Type         TargetType `yaml:"type"`
	StopServices []string   `yaml:"stop_services"`
}

type RawConfig struct {
	Version int         `yaml:"version"`
	Targets []RawTarget `yaml:"targets"`
}

type Target struct {
	ID           string
	Path         string
	DeclaredType TargetType
	Effective    TargetType
	StopServices []string
	IsBtrfs      bool
}

type Config struct {
	Version int
	Targets []Target
	Source  string
}

// DefaultPath returns the snapshots config path.
// If NOS_SNAPSHOTS_PATH is set, use it. Otherwise prefer ./devdata/snapshots.yaml when present, else /etc/nos/snapshots.yaml.
func DefaultPath() string {
	if p := os.Getenv("NOS_SNAPSHOTS_PATH"); strings.TrimSpace(p) != "" {
		return p
	}
	dev := filepath.Clean("./devdata/snapshots.yaml")
	if _, err := os.Stat(dev); err == nil {
		return dev
	}
	return "/etc/nos/snapshots.yaml"
}

// Load reads and validates the snapshots configuration. Invalid or missing paths are skipped.
func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshots config: %w", err)
	}
	var raw RawConfig
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse snapshots config: %w", err)
	}
	out := &Config{Version: raw.Version, Source: path}
	for _, rt := range raw.Targets {
		if strings.TrimSpace(rt.ID) == "" || strings.TrimSpace(rt.Path) == "" {
			continue
		}
		if !filepath.IsAbs(rt.Path) {
			continue
		}
		if fi, err := os.Stat(rt.Path); err != nil || !fi.IsDir() {
			// missing or not a directory: skip
			continue
		}
		isBtrfs := detectBtrfs(rt.Path)
		eff := rt.Type
		if rt.Type == TargetTypeAuto {
			if isBtrfs {
				eff = TargetTypeBtrfs
			} else {
				eff = TargetTypeTar
			}
		}
		t := Target{
			ID:           rt.ID,
			Path:         rt.Path,
			DeclaredType: rt.Type,
			Effective:    eff,
			StopServices: append([]string(nil), rt.StopServices...),
			IsBtrfs:      isBtrfs,
		}
		out.Targets = append(out.Targets, t)
	}
	if out.Version == 0 {
		out.Version = 1
	}
	return out, nil
}

func detectBtrfs(path string) bool {
	// Try findmnt if available
	if _, err := exec.LookPath("findmnt"); err == nil {
		cmd := exec.Command("findmnt", "-n", "-o", "FSTYPE", "--target", path)
		out, err := cmd.Output()
		if err == nil {
			fstype := strings.TrimSpace(string(out))
			return strings.EqualFold(fstype, "btrfs")
		}
	}
	// Fallback: parse /proc/self/mounts (best-effort)
	if b, err := os.ReadFile("/proc/self/mounts"); err == nil {
		lines := strings.Split(string(b), "\n")
		bestLen := 0
		bestFstype := ""
		for _, ln := range lines {
			if strings.TrimSpace(ln) == "" {
				continue
			}
			parts := strings.Fields(ln)
			if len(parts) < 3 {
				continue
			}
			mountPoint := parts[1]
			fstype := parts[2]
			if strings.HasPrefix(path, mountPoint) && len(mountPoint) > bestLen {
				bestLen = len(mountPoint)
				bestFstype = fstype
			}
		}
		return strings.EqualFold(bestFstype, "btrfs")
	}
	return false
}

var ErrNotFound = errors.New("snapshots config not found")
