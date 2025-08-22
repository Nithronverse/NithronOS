package btrfs

import "testing"

func TestPlanAddGeneratesBalance(t *testing.T) {
	p := Planner{PoolMount: "/mnt/p", ExistingDevices: []string{"/dev/sda"}, CurrentProfileData: "raid1", CurrentProfileMeta: "raid1", DeviceSizes: map[string]int64{"/dev/sda": 1000, "/dev/sdb": 1000}}
	p.MountReadWrite = true
	p.SizeThresholdPct = 0.9
	req := DevicePlanRequest{Action: "add"}
	req.Devices.Add = []string{"/dev/sdb"}
	plan, err := p.Plan(req)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Steps) < 2 {
		t.Fatalf("steps too few: %d", len(plan.Steps))
	}
	if !plan.RequiresBalance {
		t.Fatalf("requiresBalance should be true")
	}
}

func TestPlanRemoveRefusesSingleLoss(t *testing.T) {
	p := Planner{PoolMount: "/mnt/p", ExistingDevices: []string{"/dev/sda"}, CurrentProfileData: "single", CurrentProfileMeta: "single"}
	req := DevicePlanRequest{Action: "remove"}
	req.Devices.Remove = []string{"/dev/sda"}
	if _, err := p.Plan(req); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlanReplacePairs(t *testing.T) {
	p := Planner{PoolMount: "/mnt/p", ExistingDevices: []string{"/dev/sda", "/dev/sdb"}, CurrentProfileData: "raid1", CurrentProfileMeta: "raid1", DeviceSizes: map[string]int64{"/dev/sda": 1000, "/dev/sdb": 1000, "/dev/sdc": 1000}}
	req := DevicePlanRequest{Action: "replace"}
	req.Devices.Replace = []map[string]string{{"old": "/dev/sda", "new": "/dev/sdc"}}
	plan, err := p.Plan(req)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Steps) == 0 {
		t.Fatalf("no steps")
	}
}

func TestPlanAddWarnsHighUsageAndSingle(t *testing.T) {
	p := Planner{PoolMount: "/mnt/p", ExistingDevices: []string{"/dev/sda"}, CurrentProfileData: "raid1", CurrentProfileMeta: "raid1", PoolUsedPct: 85, MountReadWrite: true, SizeThresholdPct: 0.9, DeviceSizes: map[string]int64{"/dev/sda": 1000, "/dev/sdb": 1000}}
	req := DevicePlanRequest{Action: "add"}
	req.Devices.Add = []string{"/dev/sdb"}
	req.TargetProfile.Data = "single"
	plan, err := p.Plan(req)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected warnings")
	}
}

func TestPlanRemoveRedundancyTypedError(t *testing.T) {
	p := Planner{PoolMount: "/mnt/p", ExistingDevices: []string{"/dev/sda", "/dev/sdb"}, CurrentProfileData: "raid1", CurrentProfileMeta: "raid1"}
	req := DevicePlanRequest{Action: "remove"}
	req.Devices.Remove = []string{"/dev/sda", "/dev/sdb"}
	if _, err := p.Plan(req); err == nil {
		t.Fatalf("expected redundancy error")
	}
}
