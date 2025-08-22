package pools

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
)

// PoolSpec models a desired btrfs pool configuration.
// Name is a human label; Mountpoint is the desired mount path.
type PoolSpec struct {
	Name       string      `json:"name"`
	Mountpoint string      `json:"mountpoint"`
	Devices    []string    `json:"devices"`
	RaidData   string      `json:"raidData"`
	RaidMeta   string      `json:"raidMeta"`
	Features   []string    `json:"features,omitempty"`
	Encrypt    EncryptSpec `json:"encrypt"`
}

type EncryptSpec struct {
	Enabled bool   `json:"enabled"`
	Keyfile string `json:"keyfile,omitempty"`
}

var (
	ErrNoDevices       = errors.New("at least one device required")
	ErrUnsupportedRAID = errors.New("unsupported raid profile")
	ErrForbiddenRAID   = errors.New("raid5/raid6 are forbidden by default")
)

// ValidateSpec normalizes, applies defaults and validates the spec.
// It returns a copy with defaults applied.
func ValidateSpec(in PoolSpec) (PoolSpec, error) {
	// trim
	sp := in
	sp.Name = strings.TrimSpace(sp.Name)
	sp.Mountpoint = strings.TrimSpace(sp.Mountpoint)
	// dedupe devices and keep stable order
	if len(sp.Devices) == 0 {
		return sp, ErrNoDevices
	}
	seen := map[string]bool{}
	uniq := make([]string, 0, len(sp.Devices))
	for _, d := range sp.Devices {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if !seen[d] {
			seen[d] = true
			uniq = append(uniq, d)
		}
	}
	if len(uniq) == 0 {
		return sp, ErrNoDevices
	}
	sp.Devices = uniq

	// Defaults for RAID profiles
	if strings.TrimSpace(sp.RaidData) == "" {
		if len(sp.Devices) >= 2 {
			sp.RaidData = "raid1"
		} else {
			sp.RaidData = "single"
		}
	}
	if strings.TrimSpace(sp.RaidMeta) == "" {
		if len(sp.Devices) >= 2 {
			sp.RaidMeta = "raid1"
		} else {
			sp.RaidMeta = "single"
		}
	}

	// Forbid raid5/raid6 (advanced override not wired)
	forbidden := map[string]bool{"raid5": true, "raid6": true}
	if forbidden[strings.ToLower(sp.RaidData)] || forbidden[strings.ToLower(sp.RaidMeta)] {
		return sp, ErrForbiddenRAID
	}

	// Validate allowed profiles
	allowed := map[string]bool{"single": true, "raid0": true, "raid1": true, "raid10": true}
	if !allowed[strings.ToLower(sp.RaidData)] || !allowed[strings.ToLower(sp.RaidMeta)] {
		return sp, ErrUnsupportedRAID
	}

	// Normalize features (dedupe, sorted)
	if len(sp.Features) > 0 {
		m := map[string]bool{}
		out := make([]string, 0, len(sp.Features))
		for _, f := range sp.Features {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if !m[f] {
				m[f] = true
				out = append(out, f)
			}
		}
		sort.Strings(out)
		sp.Features = out
	}

	// Defaults for encryption
	if sp.Encrypt.Enabled {
		if strings.TrimSpace(sp.Encrypt.Keyfile) == "" {
			name := sp.Name
			if name == "" {
				name = "pool"
			}
			sp.Encrypt.Keyfile = filepath.Join("/etc/nos/keys", name+".key")
		}
	}

	return sp, nil
}
