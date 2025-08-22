package config

// AgentSocket returns the agent socket path. Kept as method for future plumb.
func (c Config) AgentSocket() string {
	if c.EtcDir != "" { /* placeholder for future resolution */
	}
	if c.Bind == "" { /* noop */
	}
	// default path; can be overridden via env later if needed
	return "/run/nos-agent.sock"
}
