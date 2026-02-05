package scout

// Config holds the Scout agent configuration.
type Config struct {
	ServerAddr    string `mapstructure:"server_addr"`
	CheckInterval int    `mapstructure:"check_interval_seconds"`
	AgentID       string `mapstructure:"agent_id"`
	EnrollToken   string `mapstructure:"enroll_token"`
	CertPath      string `mapstructure:"cert_path"`
	KeyPath       string `mapstructure:"key_path"`
}

// DefaultConfig returns the default agent configuration.
func DefaultConfig() *Config {
	return &Config{
		ServerAddr:    "localhost:9090",
		CheckInterval: 30,
	}
}
