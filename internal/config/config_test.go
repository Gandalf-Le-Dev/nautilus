package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Execution.Timeout != 10*time.Minute {
		t.Fatalf("expected default timeout 10m, got %v", cfg.Execution.Timeout)
	}
	if cfg.API.Port != 12911 {
		t.Fatalf("expected default API port 12911, got %d", cfg.API.Port)
	}
	if !cfg.API.Enabled {
		t.Fatal("expected API enabled by default")
	}
	if !cfg.Metrics.Enabled {
		t.Fatal("expected metrics enabled by default")
	}
	if cfg.Shutdown.GracePeriod != 30*time.Second {
		t.Fatalf("expected default grace period 30s, got %v", cfg.Shutdown.GracePeriod)
	}
	if cfg.Health.MaxConsecutiveFailures != 3 {
		t.Fatalf("expected default max consecutive failures 3, got %d", cfg.Health.MaxConsecutiveFailures)
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	yamlContent := `
job:
  name: test-job
  description: A test job
execution:
  mode: 1
  interval: 5s
  timeout: 30s
api:
  port: 9090
  debugMode: true
shutdown:
  gracePeriod: 10s
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Job.Name != "test-job" {
		t.Fatalf("expected job name 'test-job', got '%s'", cfg.Job.Name)
	}
	if cfg.Execution.Interval != 5*time.Second {
		t.Fatalf("expected interval 5s, got %v", cfg.Execution.Interval)
	}
	if cfg.API.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.API.Port)
	}
	if !cfg.API.DebugMode {
		t.Fatal("expected debugMode true")
	}
	if cfg.Shutdown.GracePeriod != 10*time.Second {
		t.Fatalf("expected grace period 10s, got %v", cfg.Shutdown.GracePeriod)
	}
}

func TestLoadConfigEmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return defaults
	if cfg.API.Port != 12911 {
		t.Fatalf("expected default port 12911, got %d", cfg.API.Port)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(configPath, []byte("{{{{invalid yaml!!!!"), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
