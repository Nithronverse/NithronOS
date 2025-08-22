package config

// AgentSocket returns the agent socket path. Kept as method for future plumb.
func (c Config) AgentSocket() string {
	// Touch fields to avoid staticcheck complaints about empty branches while keeping behavior.
	_ = c.EtcDir
	_ = c.Bind
	// default path; can be overridden via env later if needed
	return "/run/nos-agent.sock"
}
