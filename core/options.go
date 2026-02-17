package core

import (
	"fmt"
	"time"

	"github.com/navica-dev/nautilus/internal/config"
	"github.com/navica-dev/nautilus/pkg/enums"
	"github.com/navica-dev/nautilus/pkg/logging"
	"github.com/navica-dev/nautilus/pkg/plugin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Option is a functional option for configuring Nautilus
type Option func(*Nautilus) error

// WithConfigPath loads configuration from the specified path and merges with existing config
// File values override current values, but don't replace the entire config
func WithConfigPath(path string) Option {
	return func(n *Nautilus) error {
		fileCfg, err := config.LoadConfig(path)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Merge file config into existing config
		// File values override defaults, but preserve any values set by previous options
		mergeConfigs(n.config, fileCfg)
		return nil
	}
}

// mergeConfigs merges src into dst, with src values taking precedence for non-zero values
func mergeConfigs(dst, src *config.Config) {
	// Merge Job config
	if src.Job.Name != "" {
		dst.Job.Name = src.Job.Name
	}
	if src.Job.Description != "" {
		dst.Job.Description = src.Job.Description
	}
	if src.Job.Parameters != nil {
		dst.Job.Parameters = src.Job.Parameters
	}

	// Merge Execution config
	dst.Execution.Mode = src.Execution.Mode
	if src.Execution.Schedule != "" {
		dst.Execution.Schedule = src.Execution.Schedule
	}
	if src.Execution.Interval > 0 {
		dst.Execution.Interval = src.Execution.Interval
	}
	if src.Execution.Timeout > 0 {
		dst.Execution.Timeout = src.Execution.Timeout
	}
	if src.Execution.MaxRetries > 0 {
		dst.Execution.MaxRetries = src.Execution.MaxRetries
	}
	if src.Execution.RetryBackoff > 0 {
		dst.Execution.RetryBackoff = src.Execution.RetryBackoff
	}

	// Merge API config
	dst.API.Enabled = src.API.Enabled
	if src.API.Port > 0 {
		dst.API.Port = src.API.Port
	}
	if src.API.Path != "" {
		dst.API.Path = src.API.Path
	}
	dst.API.DebugMode = src.API.DebugMode
	dst.API.TLS = src.API.TLS

	// Merge Logging config
	if src.Logging.Level != "" {
		dst.Logging.Level = src.Logging.Level
	}
	if src.Logging.Format != "" {
		dst.Logging.Format = src.Logging.Format
	}

	// Merge Metrics config
	dst.Metrics.Enabled = src.Metrics.Enabled
	dst.Metrics.Prometheus = src.Metrics.Prometheus

	// Merge Secrets config
	if src.Secrets.Provider != "" {
		dst.Secrets.Provider = src.Secrets.Provider
	}
	dst.Secrets.Vault = src.Secrets.Vault

	// Merge Health config
	if src.Health.CheckInterval > 0 {
		dst.Health.CheckInterval = src.Health.CheckInterval
	}
	if src.Health.MaxConsecutiveFailures > 0 {
		dst.Health.MaxConsecutiveFailures = src.Health.MaxConsecutiveFailures
	}

	// Merge Shutdown config
	if src.Shutdown.GracePeriod > 0 {
		dst.Shutdown.GracePeriod = src.Shutdown.GracePeriod
	}
}

// WithConfig directly sets the configuration
func WithConfig(cfg *config.Config) Option {
	return func(n *Nautilus) error {
		n.config = cfg
		return nil
	}
}

// WithName sets the job name
func WithName(name string) Option {
	return func(n *Nautilus) error {
		n.config.Job.Name = name
		return nil
	}
}

// WithDescription sets the job description
func WithDescription(description string) Option {
	return func(n *Nautilus) error {
		n.config.Job.Description = description
		return nil
	}
}

// WithVersion sets the version information
func WithVersion(version string) Option {
	return func(n *Nautilus) error {
		n.version = version
		return nil
	}
}

// WithOneShot configures Nautilus to execute the job once and exit
func WithOneShot() Option {
	return func(n *Nautilus) error {
		n.config.Execution.Mode = enums.ModeOneShot
		return nil
	}
}

// WithContinuous configures Nautilus to execute the job once and block indefinitely
// Used for queue consumers and long-running processes
func WithContinuous() Option {
	return func(n *Nautilus) error {
		n.config.Execution.Mode = enums.ModeContinuous
		return nil
	}
}

// WithSchedule sets a cron schedule for periodic execution
func WithSchedule(cronExpr string) Option {
	return func(n *Nautilus) error {
		n.config.Execution.Mode = enums.ModePeriodic
		n.config.Execution.Schedule = cronExpr
		return nil
	}
}

// WithInterval sets a time interval for periodic execution
func WithInterval(interval time.Duration) Option {
	return func(n *Nautilus) error {
		n.config.Execution.Mode = enums.ModePeriodic
		n.config.Execution.Interval = interval
		return nil
	}
}

// WithRetry configures retry behavior for failed job executions
// Only applies to OneShot and Periodic modes (Continuous mode manages its own loop)
func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(n *Nautilus) error {
		n.config.Execution.MaxRetries = maxRetries
		n.config.Execution.RetryBackoff = backoff
		return nil
	}
}

// WithTimeout sets the maximum execution time for each run
func WithTimeout(timeout time.Duration) Option {
	return func(n *Nautilus) error {
		n.config.Execution.Timeout = timeout
		return nil
	}
}

// WithAPI enables or disables the API server
func WithAPI(enabled bool, port int) Option {
	return func(n *Nautilus) error {
		n.config.API.Enabled = enabled
		if port > 0 {
			n.config.API.Port = port
		}
		return nil
	}
}

// WithMetrics enables or disables metrics collection
func WithMetrics(enabled bool) Option {
	return func(n *Nautilus) error {
		n.config.Metrics.Enabled = enabled
		return nil
	}
}

// WithLogLevel sets the logging level
func WithLogLevel(level string) Option {
	return func(n *Nautilus) error {
		lvl, err := zerolog.ParseLevel(level)
		if err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}

		n.config.Logging.Level = level

		// Apply the log level
		zerolog.SetGlobalLevel(lvl)
		return nil
	}
}

// WithLogFormat sets the logging format
func WithLogFormat(format enums.LogFormatEnum) Option {
	return func(n *Nautilus) error {
		n.config.Logging.Format = format

		// Apply the log format
		switch format {
		case enums.LogFormatJson:
			// JSON is the default for zerolog
		case enums.LogFormatConsole:
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: logging.DefaultOutput})
		default:
			return fmt.Errorf("unsupported log format: %s", format)
		}

		return nil
	}
}

// WithLogger sets a custom logger
func WithLogger(logger zerolog.Logger) Option {
	return func(n *Nautilus) error {
		n.logger = logger
		return nil
	}
}

// WithMaxConsecutiveFailures sets the maximum allowed consecutive failures
func WithMaxConsecutiveFailures(max int) Option {
	return func(n *Nautilus) error {
		n.config.Health.MaxConsecutiveFailures = max
		return nil
	}
}

// WithHealthcheckDelay sets the delay between health checks
func WithHealthcheckDelay(delay time.Duration) Option {
	return func(n *Nautilus) error {
		n.config.Health.CheckInterval = delay
		return nil
	}
}

// WithPlugin adds a plugin to Nautilus
func WithPlugin(plugin plugin.Plugin) Option {
	return func(n *Nautilus) error {
		n.plugins.Register(plugin)
		return nil
	}
}

// WithGracePeriod sets the maximum time to wait for a running job during shutdown
func WithGracePeriod(period time.Duration) Option {
	return func(n *Nautilus) error {
		n.config.Shutdown.GracePeriod = period
		return nil
	}
}

// WithOnRunStart registers a callback that is invoked when a job run starts
func WithOnRunStart(hook func(*RunContext)) Option {
	return func(n *Nautilus) error {
		n.onRunStart = hook
		return nil
	}
}

// WithOnRunComplete registers a callback that is invoked when a job run completes
func WithOnRunComplete(hook func(*RunContext, error)) Option {
	return func(n *Nautilus) error {
		n.onRunComplete = hook
		return nil
	}
}

// WithOnShutdown registers a callback that is invoked when shutdown begins
func WithOnShutdown(hook func()) Option {
	return func(n *Nautilus) error {
		n.onShutdown = hook
		return nil
	}
}
