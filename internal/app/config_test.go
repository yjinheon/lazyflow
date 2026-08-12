package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAppliesFilesInPrecedenceOrder(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	writeConfig(t, filepath.Join(projectDir, "configs", "default.yaml"), `
airflow:
  base_url: http://project.example
  timeout: 15s
  auth:
    username: project-user
ui:
  rollup_window: 24h
`)
	writeConfig(t, filepath.Join(homeDir, ".config", "lazyflow", "config.yaml"), `
airflow:
  base_url: http://user.example
  auth:
    username: user-name
ui:
  refresh_intervals:
    tasks: 9s
`)

	t.Chdir(projectDir)
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("AIRFLOW_BASE_URL", "")
	t.Setenv("AIRFLOW_USERNAME", "")
	t.Setenv("AIRFLOW_PASSWORD", "")
	t.Setenv("AIRFLOW_TOKEN", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got, want := cfg.Airflow.BaseURL, "http://user.example"; got != want {
		t.Errorf("Airflow.BaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.Airflow.Auth.Username, "user-name"; got != want {
		t.Errorf("Airflow.Auth.Username = %q, want %q", got, want)
	}
	if got, want := cfg.Airflow.Timeout, "15s"; got != want {
		t.Errorf("Airflow.Timeout = %q, want project value %q", got, want)
	}
	if got, want := cfg.UI.RollupWindow, "24h"; got != want {
		t.Errorf("UI.RollupWindow = %q, want project value %q", got, want)
	}
	if got, want := cfg.UI.RefreshIntervals.Tasks, "9s"; got != want {
		t.Errorf("UI.RefreshIntervals.Tasks = %q, want %q", got, want)
	}
	if got, want := cfg.UI.RefreshIntervals.DAGs, "5s"; got != want {
		t.Errorf("UI.RefreshIntervals.DAGs = %q, want default value %q", got, want)
	}
}

func writeConfig(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
