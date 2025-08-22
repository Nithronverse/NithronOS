package pools

// CreatePlan describes a sequence of steps to create a pool.
type CreatePlan struct {
	Steps []PlanStep `json:"steps"`
}

type PlanStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Destructive bool   `json:"destructive"`
}

// ApplyResult captures the outcome of executing a plan.
type ApplyResult struct {
	Success  bool     `json:"success"`
	Logs     []string `json:"logs"`
	Errors   []string `json:"errors"`
	Fstab    []string `json:"fstab,omitempty"`
	Crypttab []string `json:"crypttab,omitempty"`
}
