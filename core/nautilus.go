package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/navica-dev/nautilus/internal/api"
	"github.com/navica-dev/nautilus/internal/config"
	"github.com/navica-dev/nautilus/internal/metrics"
	"github.com/navica-dev/nautilus/pkg/enums"
	"github.com/navica-dev/nautilus/pkg/interfaces"
	"github.com/navica-dev/nautilus/pkg/plugin"
)

// Nautilus is the main orchestrator for job execution
type Nautilus struct {
	// Configuration
	config *config.Config

	// Components
	apiServer     *api.Server
	metricsClient *metrics.Client

	// Execution
	cron        *cron.Cron
	runCount    int
	lastRunTime time.Time

	// Run state tracking
	currentRun       *interfaces.RunInfo
	consecutiveFails int

	// Metadata
	version   string
	startTime time.Time

	// State
	mu           sync.RWMutex
	shutdownOnce sync.Once
	stopping     bool
	state        enums.State

	// Execution tracking for graceful shutdown
	execWg sync.WaitGroup

	// Event hooks
	onRunStart    func(*RunContext)
	onRunComplete func(*RunContext, error)
	onShutdown    func()

	// Plugins
	plugins plugin.PluginRegistry

	// Logging
	logger zerolog.Logger
}

// New creates a new Nautilus instance with the provided options
func New(options ...Option) (*Nautilus, error) {
	// Start with default configuration
	n := &Nautilus{
		config:    config.DefaultConfig(),
		startTime: time.Now(),
		version:   "dev",
		plugins:   *plugin.NewPluginRegistry(),
		state:     enums.StateCreated,
	}

	// Apply options in order (later options override earlier ones)
	for _, option := range options {
		if err := option(n); err != nil {
			return nil, err
		}
	}

	n.logger = log.With().Str("component", "nautilus-core").Str("job", n.config.Job.Name).Logger()

	// Validate and initialize components
	if err := n.initialize(); err != nil {
		return nil, err
	}

	return n, nil
}

// initialize performs internal initialization for Nautilus
func (n *Nautilus) initialize() error {
	// Set up scheduler for periodic execution
	n.cron = cron.New(cron.WithSeconds())

	// Set up metrics client
	if n.config.Metrics.Enabled {
		n.metricsClient = metrics.NewClient(&n.config.Metrics)
		n.metricsClient.RegisterBasicMetrics(n.config.Job.Name)
	}

	// Set up API server for health checks
	if n.config.API.Enabled {
		n.apiServer = api.NewServer(&n.config.API, n.version)

		// Add Nautilus itself as a health checker
		n.apiServer.RegisterHealthChecker(n)

		// Provide state information to the API server
		n.apiServer.SetStateProvider(func() string {
			return n.GetState().String()
		})
	}

	return nil
}

func (n *Nautilus) RegisterPlugin(plugin plugin.Plugin) {
	n.plugins.Register(plugin)
}

func (n *Nautilus) Run(ctx context.Context, job interfaces.Job) error {
	n.logger.Info().
		Str("version", n.version).
		Msg("Starting job")

	// Create a cancellation context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	defer close(done)

	// Handle signals in a separate goroutine
	go func() {
		select {
		case <-sigChan:
			n.logger.Info().Msg("Received shutdown signal, shutting down ...")
			n.Shutdown(ctx)
		case <-ctx.Done():
			// Context cancelled, clean shutdown
		case <-done:
			// Parent function is exiting, clean up
		}
		// Unregister signal handling to prevent leaks
		signal.Stop(sigChan)
	}()

	// Transition to initializing state
	n.mu.Lock()
	n.setState(enums.StateInitializing)
	n.mu.Unlock()

	// Initialize plugins
	if err := n.plugins.InitializeAll(ctx); err != nil {
		return fmt.Errorf("plugin initialization failed: %w", err)
	}
	defer func() {
		if errs := n.plugins.TerminateAll(ctx); len(errs) > 0 {
			for _, err := range errs {
				n.logger.Error().Err(err).Msg("plugin termination error")
			}
		}
	}()

	// Start API server if enabled
	if n.apiServer != nil {
		apiErrCh := n.apiServer.StartAsync()

		// Monitor for API server errors in background
		go func() {
			if err := <-apiErrCh; err != nil {
				n.logger.Error().Err(err).Msg("API server error")
			}
		}()
	}

	// Initialize the job
	n.logger.Info().Msg("Initializing job")
	if err := job.Setup(ctx); err != nil {
		return fmt.Errorf("job initialization failed: %w", err)
	}

	// Register health check if supported
	if healthChecker, ok := job.(interfaces.HealthCheck); ok && n.apiServer != nil {
		n.apiServer.RegisterHealthChecker(healthChecker)
	}

	// Transition to ready state
	n.mu.Lock()
	n.setState(enums.StateReady)
	n.mu.Unlock()

	// Execute based on mode
	switch n.config.Execution.Mode {
	case enums.ModeOneShot:
		// Execute once, run Teardown, and exit
		execErr := n.executeRun(ctx, job)
		n.Shutdown(ctx)
		teardownErr := n.performTermination(job)
		if execErr != nil {
			return execErr
		}
		return teardownErr

	case enums.ModeContinuous:
		// Execute once and block indefinitely
		// The job's Execute method should run its own loop
		execErr := n.executeRun(ctx, job)

		// Wait for shutdown signal
		<-ctx.Done()

		// Perform clean shutdown
		teardownErr := n.performTermination(job)
		if execErr != nil {
			return execErr
		}
		return teardownErr

	case enums.ModePeriodic:
		// Execute repeatedly on schedule
		if n.config.Execution.Schedule != "" {
			// Use cron scheduler
			_, err := n.cron.AddFunc(n.config.Execution.Schedule, func() {
				if err := n.executeRun(ctx, job); err != nil {
					n.logger.Error().Err(err).Msg("Scheduled run failed")
				}
			})
			if err != nil {
				return fmt.Errorf("failed to schedule run: %w", err)
			}
			n.cron.Start()
		} else if n.config.Execution.Interval > 0 {
			// Use interval-based execution
			go func() {
				ticker := time.NewTicker(n.config.Execution.Interval)
				defer ticker.Stop()

				// Execute immediately on start
				if err := n.executeRun(ctx, job); err != nil {
					n.logger.Error().Err(err).Msg("Initial run failed")
				}

				for {
					select {
					case <-ticker.C:
						// Check if we're shutting down before starting new execution
						n.mu.RLock()
						stopping := n.stopping
						n.mu.RUnlock()
						if stopping {
							return
						}

						if err := n.executeRun(ctx, job); err != nil {
							n.logger.Error().Err(err).Msg("Periodic run failed")
						}
					case <-ctx.Done():
						return
					}
				}
			}()
		} else {
			return fmt.Errorf("periodic mode requires either Schedule or Interval")
		}

	default:
		return fmt.Errorf("invalid execution mode: %v", n.config.Execution.Mode)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Clean shutdown
	return n.performTermination(job)
}

// executeRun performs a single execution of the job with retry support
func (n *Nautilus) executeRun(ctx context.Context, job interfaces.Job) error {
	// Track execution for graceful shutdown
	n.execWg.Add(1)
	defer n.execWg.Done()

	startTime := time.Now()

	// Create run info
	runID := uuid.New().String()
	runInfo := &interfaces.RunInfo{
		RunID:     runID,
		StartTime: startTime,
		Status:    enums.RunStatusRunning,
	}

	n.mu.Lock()
	n.runCount++
	currentRun := n.runCount
	n.currentRun = runInfo
	n.setState(enums.StateExecuting)
	n.mu.Unlock()

	// Create run-specific logger
	runLogger := n.logger.With().
		Int("run", currentRun).
		Str("run_id", runID).
		Logger()

	// Track metrics
	runLogger.Info().
		Time("start_time", startTime).
		Msg("Starting job run")

	if n.metricsClient != nil {
		n.metricsClient.RecordRunStart()
	}

	// Create and embed RunContext
	runCtx := &RunContext{
		RunID:     runID,
		RunCount:  currentRun,
		Logger:    runLogger,
		StartTime: startTime,
	}

	// Call onRunStart hook if registered
	if n.onRunStart != nil {
		n.onRunStart(runCtx)
	}

	// Execute with retry logic (only for OneShot and Periodic modes)
	var err error
	maxAttempts := 1
	if n.config.Execution.Mode != enums.ModeContinuous && n.config.Execution.MaxRetries > 0 {
		maxAttempts = n.config.Execution.MaxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Create run context with timeout if configured
		execCtx := ctx
		if n.config.Execution.Timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, n.config.Execution.Timeout)
			defer cancel()
		}

		// Embed RunContext into the execution context
		execCtx = WithRunContext(execCtx, runCtx)

		// Execute the job
		err = job.Execute(execCtx)

		if err == nil {
			// Success!
			break
		}

		// Failed - check if we should retry
		if attempt < maxAttempts-1 {
			// Calculate backoff with exponential increase
			backoff := n.config.Execution.RetryBackoff * time.Duration(1<<uint(attempt))

			// Cap backoff at interval if in periodic mode
			if n.config.Execution.Mode == enums.ModePeriodic && n.config.Execution.Interval > 0 {
				if backoff > n.config.Execution.Interval {
					backoff = n.config.Execution.Interval
				}
			}

			n.logger.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Int("max_attempts", maxAttempts).
				Dur("backoff", backoff).
				Msg("Job execution failed, retrying")

			// Wait before retry (with context cancellation check)
			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				// Context cancelled, abort retries
				return ctx.Err()
			}
		}
	}

	// Update metrics and run info
	duration := time.Since(startTime)

	n.mu.Lock()
	if n.currentRun != nil && n.currentRun.RunID == runID {
		n.currentRun.Duration = duration
		n.lastRunTime = startTime

		if err != nil {
			n.currentRun.Status = enums.RunStatusFailed
			n.currentRun.Error = err
			n.consecutiveFails++
		} else {
			n.currentRun.Status = enums.RunStatusCompleted
			n.consecutiveFails = 0
		}
	}
	n.mu.Unlock()

	if n.metricsClient != nil {
		if err != nil {
			n.metricsClient.RecordRunFailure(duration)
		} else {
			n.metricsClient.RecordRunSuccess(duration)
		}
	}

	// Log completion
	logEvent := n.logger.Info()
	if err != nil {
		logEvent = n.logger.Error().Err(err)
	}

	logEvent.
		Int("run", currentRun).
		Str("run_id", runID).
		Dur("duration", duration).
		Time("end_time", time.Now()).
		Msg("Job run completed")

	// Return to ready state (unless we're shutting down)
	n.mu.Lock()
	if n.state != enums.StateShuttingDown {
		n.setState(enums.StateReady)
	}
	n.mu.Unlock()

	// Call onRunComplete hook if registered
	if n.onRunComplete != nil {
		n.onRunComplete(runCtx, err)
	}

	return err
}

// Shutdown initiates a graceful shutdown of Nautilus
func (n *Nautilus) Shutdown(ctx context.Context) {
	n.shutdownOnce.Do(func() {
		n.mu.Lock()
		n.stopping = true
		n.setState(enums.StateShuttingDown)
		gracePeriod := n.config.Shutdown.GracePeriod
		n.mu.Unlock()

		n.logger.Info().Msg("Shutting down Nautilus")

		// Call onShutdown hook if registered
		if n.onShutdown != nil {
			n.onShutdown()
		}

		// Stop the scheduler if running (prevents new executions)
		if n.cron != nil {
			n.cron.Stop()
		}

		// Wait for current execution to complete (with timeout)
		done := make(chan struct{})
		go func() {
			n.execWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			n.logger.Info().Msg("All job executions completed gracefully")
		case <-time.After(gracePeriod):
			n.logger.Warn().
				Dur("grace_period", gracePeriod).
				Msg("Grace period exceeded, forcing shutdown")
		}

		// Stop the API server
		if n.apiServer != nil {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := n.apiServer.Stop(shutdownCtx); err != nil {
				n.logger.Error().Err(err).Msg("Failed to stop API server gracefully")
			}
		}
	})
}

// performTermination handles job termination
func (n *Nautilus) performTermination(job interfaces.Job) error {
	terminateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	n.logger.Info().Msgf("Terminating job")
	err := job.Teardown(terminateCtx)

	// Mark as stopped
	n.mu.Lock()
	n.setState(enums.StateStopped)
	n.mu.Unlock()

	return err
}

// GetCurrentRun return information about the current run
func (n *Nautilus) GetCurrentRun() *interfaces.RunInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.currentRun == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	run := *n.currentRun
	return &run
}

// GetRunCount returns the total number of runs executed
func (n *Nautilus) GetRunCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.runCount
}

// GetLastRunTime returns the time of the last run
func (n *Nautilus) GetLastRunTime() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastRunTime
}

// GetUptime returns the uptime of Nautilus
func (n *Nautilus) GetUptime() time.Duration {
	return time.Since(n.startTime)
}

// GetState returns the current lifecycle state
func (n *Nautilus) GetState() enums.State {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

// setState updates the current state (must hold lock)
func (n *Nautilus) setState(state enums.State) {
	n.state = state
	n.logger.Debug().Str("state", state.String()).Msg("State transition")
}

// --- Heath Checker ---
var _ interfaces.HealthCheck = (*Nautilus)(nil)

func (n *Nautilus) Name() string {
	return "nautilus-core"
}

func (n *Nautilus) HealthCheck(ctx context.Context) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Report unhealthy if we're shutting down
	if n.stopping {
		return fmt.Errorf("shutting down")
	}

	// Report unhealthy if too many consecutive failures
	maxFails := n.config.Health.MaxConsecutiveFailures
	if maxFails > 0 && n.consecutiveFails >= maxFails {
		return fmt.Errorf("too many consecutive failures: %d", n.consecutiveFails)
	}

	// Include basic health info
	return nil
}
