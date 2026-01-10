package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	// Check for XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bullmq-tui", "config.yaml")
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "bullmq-tui", "config.yaml")
}

// Load reads and parses the config file
func Load(path string) (*Config, error) {
	if path == "" {
		path = GetConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s (run 'bullmq-tui config init' to create it)", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Expand environment variables in connection passwords
	for _, conn := range cfg.Connections {
		conn.Password = expandEnvVars(conn.Password)
	}

	// Set defaults
	if cfg.Settings.RefreshIntervalMs == 0 {
		cfg.Settings.RefreshIntervalMs = 1000
	}
	if cfg.Settings.StatsWindowMinutes == 0 {
		cfg.Settings.StatsWindowMinutes = 30
	}
	if cfg.Settings.MaxJobsDisplay == 0 {
		cfg.Settings.MaxJobsDisplay = 100
	}
	if cfg.Settings.DateFormat == "" {
		cfg.Settings.DateFormat = "2006-01-02 15:04:05"
	}

	return &cfg, nil
}

// Save writes the config to disk
func Save(cfg *Config, path string) error {
	if path == "" {
		path = GetConfigPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateDefault creates a default configuration file
func CreateDefault(path string) (*Config, error) {
	if path == "" {
		path = GetConfigPath()
	}

	cfg := &Config{
		Version:           1,
		DefaultConnection: "local",
		Connections: map[string]*Connection{
			"local": {
				Name:     "Local Development",
				Host:     "localhost",
				Port:     6379,
				Password: "",
				DB:       0,
				TLS:      false,
				Prefix:   "bull",
			},
		},
		Settings: Settings{
			RefreshIntervalMs:  1000,
			StatsWindowMinutes: 30,
			MaxJobsDisplay:     100,
			Theme:              "default",
			DateFormat:         "2006-01-02 15:04:05",
		},
	}

	if err := Save(cfg, path); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetConnection retrieves a connection by name
func (c *Config) GetConnection(name string) (*Connection, error) {
	if name == "" {
		name = c.DefaultConnection
	}

	conn, ok := c.Connections[name]
	if !ok {
		return nil, fmt.Errorf("connection '%s' not found", name)
	}

	return conn, nil
}

// AddConnection adds a new connection to the config
func (c *Config) AddConnection(id string, conn *Connection) {
	if c.Connections == nil {
		c.Connections = make(map[string]*Connection)
	}
	c.Connections[id] = conn
}

// RemoveConnection removes a connection from the config
func (c *Config) RemoveConnection(name string) error {
	if _, ok := c.Connections[name]; !ok {
		return fmt.Errorf("connection '%s' not found", name)
	}

	delete(c.Connections, name)

	// Update default if needed
	if c.DefaultConnection == name {
		// Set to first available connection or empty
		for id := range c.Connections {
			c.DefaultConnection = id
			break
		}
	}

	return nil
}

// SetDefault sets the default connection
func (c *Config) SetDefault(name string) error {
	if _, ok := c.Connections[name]; !ok {
		return fmt.Errorf("connection '%s' not found", name)
	}

	c.DefaultConnection = name
	return nil
}

// expandEnvVars replaces ${VAR} patterns with environment variable values
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]

		// Get value from environment
		value := os.Getenv(varName)

		// Return value or original if not found
		if value == "" {
			return match
		}
		return value
	})
}

// ListConnections returns a list of connection names
func (c *Config) ListConnections() []string {
	names := make([]string, 0, len(c.Connections))
	for name := range c.Connections {
		names = append(names, name)
	}
	return names
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Connections) == 0 {
		return fmt.Errorf("no connections defined")
	}

	if c.DefaultConnection == "" {
		return fmt.Errorf("no default connection set")
	}

	if _, ok := c.Connections[c.DefaultConnection]; !ok {
		return fmt.Errorf("default connection '%s' not found", c.DefaultConnection)
	}

	for id, conn := range c.Connections {
		if conn.Host == "" {
			return fmt.Errorf("connection '%s': host is required", id)
		}
		if conn.Port <= 0 || conn.Port > 65535 {
			return fmt.Errorf("connection '%s': invalid port %d", id, conn.Port)
		}
		if conn.Prefix == "" {
			conn.Prefix = "bull"
		}
	}

	return nil
}
