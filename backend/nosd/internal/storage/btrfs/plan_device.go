package btrfs

import (
	"fmt"
	"strings"
)

type DevicePlanRequest struct {
	Action  string `json:"action"` // add|remove|replace
	PoolID  string `json:"poolId"`
	Devices struct {
		Add     []string            `json:"add,omitempty"`
		Remove  []string            `json:"remove,omitempty"`
		Replace []map[string]string `json:"replace,omitempty"` // {old,new}
	} `json:"devices"`
	TargetProfile struct {
		Data string `json:"data,omitempty"` // single|raid1
		Meta string `json:"meta,omitempty"`
	} `json:"targetProfile,omitempty"`
	Force bool `json:"force,omitempty"`
}

type PlanStep struct {
	ID, Description, Command string
	Destructive              bool
}

type DevicePlan struct {
	PlanID          string     `json:"planId"`
	Steps           []PlanStep `json:"steps"`
	Warnings        []string   `json:"warnings"`
	RequiresBalance bool       `json:"requiresBalance,omitempty"`
}

// Planner is a minimal facade for tests; the real implementation should inspect pool state.
type Planner struct {
	// Inject pool facts for validation; simplified for skeleton
	PoolMount          string
	ExistingDevices    []string
	CurrentProfileData string // single|raid1
	CurrentProfileMeta string
	DeviceSizes        map[string]int64 // path -> size bytes
	PoolUsedPct        float64          // 0..100
	MountReadWrite     bool             // true if RW
	Degraded           bool             // degraded state
	SizeThresholdPct   float64          // e.g., 0.90
}

func (p Planner) contains(dev string) bool {
	for _, d := range p.ExistingDevices {
		if d == dev {
			return true
		}
	}
	return false
}

func (p Planner) Plan(req DevicePlanRequest) (DevicePlan, error) {
	plan := DevicePlan{PlanID: "dev-" + randomID(), Steps: []PlanStep{}, Warnings: []string{}}
	mount := p.PoolMount
	switch strings.ToLower(req.Action) {
	case "add":
		if !p.MountReadWrite || p.Degraded {
			return plan, fmt.Errorf("pool not writable or degraded; cannot balance")
		}
		add := unique(req.Devices.Add)
		if len(add) == 0 {
			return plan, fmt.Errorf("no devices to add")
		}
		for _, d := range add {
			if p.contains(d) {
				return plan, fmt.Errorf("device already in pool: %s", d)
			}
		}
		// size checks: new >= threshold * min(existing)
		minExisting := int64(0)
		for _, d := range p.ExistingDevices {
			if sz, ok := p.DeviceSizes[d]; ok {
				if minExisting == 0 || sz < minExisting {
					minExisting = sz
				}
			}
		}
		thr := p.SizeThresholdPct
		if thr <= 0 {
			thr = 0.90
		}
		minAllowed := int64(float64(minExisting) * thr)
		for _, d := range add {
			sz, ok := p.DeviceSizes[d]
			if !ok {
				return plan, fmt.Errorf("unknown device: %s", d)
			}
			if minExisting > 0 && sz < minAllowed {
				return plan, fmt.Errorf("device too small: %s", d)
			}
		}
		profD := req.TargetProfile.Data
		profM := req.TargetProfile.Meta
		if profD == "" {
			profD = p.CurrentProfileData
		}
		if profM == "" {
			profM = p.CurrentProfileMeta
		}
		cmdAdd := "btrfs device add " + strings.Join(quoteAll(add), " ") + " " + shellQuote(mount)
		plan.Steps = append(plan.Steps, PlanStep{ID: "dev-add", Description: "add devices", Command: cmdAdd, Destructive: true})
		cmdBal := fmt.Sprintf("btrfs balance start -dconvert=%s -mconvert=%s %s", profD, profM, shellQuote(mount))
		plan.Steps = append(plan.Steps, PlanStep{ID: "balance", Description: "rebalance data/metadata", Command: cmdBal, Destructive: false})
		plan.RequiresBalance = true
		if p.PoolUsedPct >= 80 {
			plan.Warnings = append(plan.Warnings, "Pool is >80% full; balance may take longer.")
		}
		if strings.ToLower(profD) == "single" && strings.ToLower(p.CurrentProfileData) != "single" {
			plan.Warnings = append(plan.Warnings, "Target profile will be single; redundancy reduced.")
		}
	case "remove":
		rem := unique(req.Devices.Remove)
		if len(rem) == 0 {
			return plan, fmt.Errorf("no devices to remove")
		}
		// Safety: redundancy
		if p.CurrentProfileData == "single" && len(p.ExistingDevices)-len(rem) < 1 && !req.Force {
			return plan, RemoveRedundancyError{Reason: "insufficient redundancy for removal"}
		}
		if p.CurrentProfileData == "raid1" && len(p.ExistingDevices)-len(rem) < 2 && !req.Force {
			return plan, RemoveRedundancyError{Reason: "cannot shrink raid1 below 2 devices without force"}
		}
		cmd := "btrfs device remove " + strings.Join(quoteAll(rem), " ") + " " + shellQuote(mount)
		plan.Steps = append(plan.Steps, PlanStep{ID: "dev-remove", Description: "remove devices", Command: cmd, Destructive: true})
	case "replace":
		if len(req.Devices.Replace) == 0 {
			return plan, fmt.Errorf("no replace pairs")
		}
		for i, pair := range req.Devices.Replace {
			old := pair["old"]
			newd := pair["new"]
			if !p.contains(old) {
				return plan, fmt.Errorf("old not in pool: %s", old)
			}
			if p.contains(newd) {
				return plan, fmt.Errorf("new already in pool: %s", newd)
			}
			// size check new >= old
			if so, ok := p.DeviceSizes[old]; ok {
				if sn, ok2 := p.DeviceSizes[newd]; ok2 {
					if sn < so {
						return plan, fmt.Errorf("replacement too small: %s < %s", newd, old)
					}
				} else {
					return plan, fmt.Errorf("unknown device: %s", newd)
				}
			} else {
				return plan, fmt.Errorf("unknown device: %s", old)
			}
			cmd := fmt.Sprintf("btrfs replace start %s %s %s", shellQuote(old), shellQuote(newd), shellQuote(mount))
			plan.Steps = append(plan.Steps, PlanStep{ID: fmt.Sprintf("replace-%d", i+1), Description: "replace device", Command: cmd, Destructive: true})
		}
	default:
		return plan, fmt.Errorf("invalid action")
	}
	return plan, nil
}

// RemoveRedundancyError indicates remove would violate redundancy.
type RemoveRedundancyError struct{ Reason string }

func (e RemoveRedundancyError) Error() string { return e.Reason }

func unique(in []string) []string {
	m := map[string]struct{}{}
	out := []string{}
	for _, s := range in {
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
func quoteAll(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = shellQuote(v)
	}
	return out
}
func randomID() string { return "x" }
