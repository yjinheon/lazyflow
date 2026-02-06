package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Airflow AirflowConfig `yaml:"airflow"`
	UI      UIConfig      `yaml:"ui"`
}

type AirflowConfig struct {
	BaseURL string     `yaml:"base_url"`
	Timeout string     `yaml:"timeout"`
	Auth    AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Type     string `yaml:"type"` // "basic" or "token"
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
}

type UIConfig struct {
	Theme            string           `yaml:"theme"`
	RefreshIntervals RefreshIntervals `yaml:"refresh_intervals"`
}

type RefreshIntervals struct {
	DAGs   string `yaml:"dags"`
	Runs   string `yaml:"runs"`
	Tasks  string `yaml:"tasks"`
	Logs   string `yaml:"logs"`
	Health string `yaml:"health"`
}

func DefaultConfig() Config {
	return Config{
		Airflow: AirflowConfig{
			BaseURL: "http://localhost:8080",
			Timeout: "30s",
			Auth: AuthConfig{
				Type: "basic",
			},
		},
		UI: UIConfig{
			Theme: "dark",
			RefreshIntervals: RefreshIntervals{
				DAGs:   "5s",
				Runs:   "3s",
				Tasks:  "2s",
				Logs:   "1s",
				Health: "10s",
			},
		},
	}
}

// LoadConfig loads configuration from file, then overlays environment variables.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()

	// Try config file paths in order
	paths := configPaths()
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config %s: %w", p, err)
		}
		break
	}

	// Environment variable overrides
	if v := os.Getenv("AIRFLOW_BASE_URL"); v != "" {
		cfg.Airflow.BaseURL = v
	}
	if v := os.Getenv("AIRFLOW_USERNAME"); v != "" {
		cfg.Airflow.Auth.Username = v
	}
	if v := os.Getenv("AIRFLOW_PASSWORD"); v != "" {
		cfg.Airflow.Auth.Password = v
	}
	if v := os.Getenv("AIRFLOW_TOKEN"); v != "" {
		cfg.Airflow.Auth.Token = v
		cfg.Airflow.Auth.Type = "token"
	}

	return cfg, nil
}

func configPaths() []string {
	var paths []string

	// ./configs/default.yaml (project local)
	paths = append(paths, "configs/default.yaml")

	// ~/.config/lazyflow/config.yaml
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "lazyflow", "config.yaml"))
	}

	return paths
}

// ParseDuration parses a duration string, returning a fallback on error.
func ParseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
