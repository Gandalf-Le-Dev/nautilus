package config

import (
	"fmt"
	"os"
	"time"

	"github.com/navica-dev/nautilus/pkg/enums"
	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for Nautilus
type Config struct {
	Job       JobConfig       `yaml:"job"`
	Execution ExecutionConfig `yaml:"execution"`
	API       APIConfig       `yaml:"api"`
	Logging   LoggingConfig   `yaml:"logging"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Secrets   SecretsConfig   `yaml:"secrets"`
	Health    HealthConfig    `yaml:"health"`
	Shutdown  ShutdownConfig  `yaml:"shutdown"`
}

// JobConfig contains job-specific configuration
type JobConfig struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Parameters  map[string]interface{} `yaml:"parameters"`
}

// ExecutionConfig controls how the job is executed
type ExecutionConfig struct {
	// Mode specifies the execution mode (oneshot, periodic, continuous)
	Mode enums.ExecutionMode `yaml:"mode"`

	// Schedule using cron expression (only for ModePeriodic)
	Schedule string `yaml:"schedule"`

	// Interval for periodic execution (only for ModePeriodic)
	Interval time.Duration `yaml:"interval"`

	// Timeout for each execution
	Timeout time.Duration `yaml:"timeout"`

	// Retry configuration
	MaxRetries   int           `yaml:"maxRetries"`
	RetryBackoff time.Duration `yaml:"retryBackoff"`
}

// APIConfig controls the HTTP API server
type APIConfig struct {
	Enabled   bool      `yaml:"enabled"`
	Port      int       `yaml:"port"`
	Path      string    `yaml:"path"`
	DebugMode bool      `yaml:"debugMode"`
	TLS       TLSConfig `yaml:"tls"`
}

// TLSConfig contains TLS configuration for the API server
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

// LoggingConfig controls logging behavior
type LoggingConfig struct {
	Level  string              `yaml:"level"`
	Format enums.LogFormatEnum `yaml:"format"` // json, console
}

// MetricsConfig controls metrics collection
type MetricsConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

// PrometheusConfig controls Prometheus metrics
type PrometheusConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// SecretsConfig controls how secrets are managed
type SecretsConfig struct {
	Provider string      `yaml:"provider"` // vault, aws, env, etc.
	Vault    VaultConfig `yaml:"vault"`
}

// VaultConfig controls HashiCorp Vault integration
type VaultConfig struct {
	Address   string `yaml:"address"`
	TokenPath string `yaml:"tokenPath"`
}

// HealthConfig controls health checking behavior
type HealthConfig struct {
	// CheckInterval is how frequently to run health checks
	CheckInterval time.Duration `yaml:"checkInterval"`

	// MaxConsecutiveFailures is the maximum number of consecutive failures allowed
	MaxConsecutiveFailures int `yaml:"maxConsecutiveFailures"`
}

// ShutdownConfig controls graceful shutdown behavior
type ShutdownConfig struct {
	// GracePeriod is the maximum time to wait for a running job to complete during shutdown
	GracePeriod time.Duration `yaml:"gracePeriod"`
}

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() *Config {
	return &Config{
		Job: JobConfig{
			Name: "nautilus-job",
		},
		Execution: ExecutionConfig{
			Mode:         enums.ModeOneShot,
			Timeout:      10 * time.Minute,
			MaxRetries:   0,
			RetryBackoff: 1 * time.Second,
		},
		API: APIConfig{
			Enabled:   true,
			Port:      12911,
			Path:      "/health",
			DebugMode: false,
			TLS: TLSConfig{
				Enabled: false,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "console",
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Prometheus: PrometheusConfig{
				Enabled: true,
				Path:    "/metrics",
			},
		},
		Secrets: SecretsConfig{
			Provider: "env",
		},
		Health: HealthConfig{
			CheckInterval:          30 * time.Second,
			MaxConsecutiveFailures: 3,
		},
		Shutdown: ShutdownConfig{
			GracePeriod: 30 * time.Second,
		},
	}
}

// LoadConfig loads configuration from a YAML file
// If path is empty, returns DefaultConfig()
func LoadConfig(configPath string) (*Config, error) {
	// Return default config if no path specified
	if configPath == "" {
		return DefaultConfig(), nil
	}

	// Read the YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with defaults
	config := DefaultConfig()

	// Unmarshal YAML into config (will override defaults)
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
