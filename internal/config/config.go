package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration.
type Config struct {
	Monitors  []MonitorConfig  `yaml:"monitors"`
	Notifiers []NotifierConfig `yaml:"notifiers"`
}

// MonitorConfig holds a single monitor's configuration.
type MonitorConfig struct {
	Name          string `yaml:"name"`
	Type          string `yaml:"type"`
	Target        string `yaml:"target"`
	Interval      int    `yaml:"interval"`
	AlertInterval int    `yaml:"alert_interval"`
	Enabled       bool   `yaml:"enabled"`
}

// NotifierConfig holds a single notifier's configuration.
type NotifierConfig struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Webhook string            `yaml:"webhook"`
	Enabled bool              `yaml:"enabled"`
	Extra   map[string]string `yaml:"extra,omitempty"`
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

	// Set defaults
	for i := range cfg.Monitors {
		if cfg.Monitors[i].Interval <= 0 {
			cfg.Monitors[i].Interval = 60
		}
		if cfg.Monitors[i].AlertInterval <= 0 {
			cfg.Monitors[i].AlertInterval = 300
		}
		if !cfg.Monitors[i].Enabled {
			cfg.Monitors[i].Enabled = true
		}
	}

	for i := range cfg.Notifiers {
		if !cfg.Notifiers[i].Enabled {
			cfg.Notifiers[i].Enabled = true
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
		Monitors: []MonitorConfig{
			{
				Name:          "官网",
				Type:          "http",
				Target:        "https://example.com",
				Interval:      60,
				AlertInterval: 300,
				Enabled:       true,
			},
		},
		Notifiers: []NotifierConfig{
			{
				Name:    "运维群",
				Type:    "wechat",
				Webhook: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
				Enabled: true,
			},
		},
	}
}
