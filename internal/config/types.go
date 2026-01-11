package config

type Config struct {
	Version           int                    `yaml:"version"`
	DefaultConnection string                 `yaml:"default_connection"`
	Connections       map[string]*Connection `yaml:"connections"`
	Settings          Settings               `yaml:"settings"`
}

type Connection struct {
	Name          string          `yaml:"name"`
	Host          string          `yaml:"host"`
	Port          int             `yaml:"port"`
	Password      string          `yaml:"password"`
	DB            int             `yaml:"db"`
	TLS           bool            `yaml:"tls"`
	TLSSkipVerify bool            `yaml:"tls_skip_verify"`
	Prefix        string          `yaml:"prefix"`
	Sentinel      *SentinelConfig `yaml:"sentinel,omitempty"`
	Cluster       *ClusterConfig  `yaml:"cluster,omitempty"`
}

type SentinelConfig struct {
	Enabled    bool     `yaml:"enabled"`
	MasterName string   `yaml:"master_name"`
	Addresses  []string `yaml:"addresses"`
}

type ClusterConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Addresses []string `yaml:"addresses"`
}

type Settings struct {
	RefreshIntervalMs  int    `yaml:"refresh_interval_ms"`
	StatsWindowMinutes int    `yaml:"stats_window_minutes"`
	MaxJobsDisplay     int    `yaml:"max_jobs_display"`
	Theme              string `yaml:"theme"`
	DateFormat         string `yaml:"date_format"`
}
