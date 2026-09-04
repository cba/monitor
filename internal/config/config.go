package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Defaults holds default values inherited by all monitors.
type Defaults struct {
	Interval      int  `yaml:"interval"`
	AlertInterval int  `yaml:"alert_interval"`
	Enabled       bool `yaml:"enabled"`
}

// SSHConfig holds SSH connection parameters for remote monitors.
type SSHConfig struct {
	Host     string `yaml:"host,omitempty"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
	CertFile string `yaml:"cert_file,omitempty"`
}

// Config is the root configuration.
type Config struct {
	Defaults  Defaults         `yaml:"defaults"`
	Monitors  []MonitorConfig  `yaml:"monitors"`
	Notifiers []NotifierConfig `yaml:"notifiers"`
	Reporter  *ReporterConfig  `yaml:"reporter,omitempty"`
}

// MonitorConfig holds a single monitor's configuration.
type MonitorConfig struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Target  string `yaml:"target,omitempty"`
	Enabled *bool  `yaml:"enabled,omitempty"`

	Interval      int `yaml:"interval,omitempty"`
	AlertInterval int `yaml:"alert_interval,omitempty"`

	URL     string  `yaml:"url,omitempty"`
	Keyword string  `yaml:"keyword,omitempty"`
	Host    string  `yaml:"host,omitempty"`
	Port    string  `yaml:"port,omitempty"`
	DSN     string  `yaml:"dsn,omitempty"`
	Path    string  `yaml:"path,omitempty"`
	Warn    float64 `yaml:"warn,omitempty"`
	Crit    float64 `yaml:"crit,omitempty"`

	ProcessName   string `yaml:"process_name,omitempty"`
	ContainerName string `yaml:"container_name,omitempty"`
	Password      string `yaml:"password,omitempty"`

	SSH *SSHConfig `yaml:"ssh,omitempty"`
}

// NotifierConfig holds a single notifier's configuration.
type NotifierConfig struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Webhook string `yaml:"webhook,omitempty"`
	Enabled *bool  `yaml:"enabled,omitempty"`

	CorpID  string `yaml:"corp_id,omitempty"`
	AgentID string `yaml:"agent_id,omitempty"`
	Secret  string `yaml:"secret,omitempty"`
	ToUsers string `yaml:"to_users,omitempty"`
}

// ReporterConfig holds daily report configuration.
type ReporterConfig struct {
	Enabled bool     `yaml:"enabled"`
	Cron    string   `yaml:"cron"`
	Title   string   `yaml:"title"`
	Targets []string `yaml:"targets,omitempty"` // empty = all monitors
}

// Load reads a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	d := cfg.Defaults
	if d.Interval <= 0 {
		d.Interval = 60
	}
	if d.AlertInterval <= 0 {
		d.AlertInterval = 300
	}
	if !d.Enabled {
		d.Enabled = true
	}

	for i := range cfg.Monitors {
		m := &cfg.Monitors[i]
		if m.Interval <= 0 {
			m.Interval = d.Interval
		}
		if m.AlertInterval <= 0 {
			m.AlertInterval = d.AlertInterval
		}
		if m.Enabled == nil {
			enable := d.Enabled
			m.Enabled = &enable
		}
	}

	for i := range cfg.Notifiers {
		if cfg.Notifiers[i].Enabled == nil {
			v := true
			cfg.Notifiers[i].Enabled = &v
		}
	}

	return cfg, nil
}

// Save writes the config to a YAML file.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// DefaultConfig returns a sample configuration.
func DefaultConfig() *Config {
	return &Config{
		Defaults: Defaults{
			Interval:      60,
			AlertInterval: 300,
		},
		Monitors: []MonitorConfig{
			{
				Name: "官网",
				Type: "http",
				URL:  "https://example.com",
			},
		},
		Notifiers: []NotifierConfig{
			{
				Name:    "通知群",
				Type:    "wechat",
				Webhook: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
			},
		},
	}
}
