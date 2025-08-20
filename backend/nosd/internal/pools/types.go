package pools

type Pool struct {
	ID      string   `json:"id"` // uuid or label
	Label   string   `json:"label"`
	UUID    string   `json:"uuid"`
	Mount   string   `json:"mount,omitempty"`
	Devices []string `json:"devices"`
	Size    uint64   `json:"size"`
	Used    uint64   `json:"used"`
	Free    uint64   `json:"free"`
	RAID    string   `json:"raid"`
}

type PlanRequest struct {
	Label   string   `json:"label"`
	Devices []string `json:"devices"`
	Raid    string   `json:"raid"`
}
