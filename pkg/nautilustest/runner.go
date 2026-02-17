package nautilustest

import (
	"context"
	"time"

	"github.com/navica-dev/nautilus/core"
	"github.com/navica-dev/nautilus/pkg/interfaces"
)

// Result contains the outcome of a test job execution
type Result struct {
	// SetupError is the error from Setup, if any
	SetupError error

	// ExecuteError is the error from Execute, if any
	ExecuteError error

	// TeardownError is the error from Teardown, if any
	TeardownError error

	// Duration is the total time taken for Setup + Execute + Teardown
	Duration time.Duration

	// ExecuteDuration is the time taken for Execute only
	ExecuteDuration time.Duration
}

// RunJob executes a job synchronously for testing purposes
// It calls Setup -> Execute -> Teardown in sequence
// No HTTP server, no metrics, no signal handling
func RunJob(ctx context.Context, job interfaces.Job, opts ...core.Option) (*Result, error) {
	start := time.Now()
	result := &Result{}

	// Create minimal config options for testing
	testOpts := []core.Option{
		core.WithAPI(false, 0),      // Disable API server
		core.WithMetrics(false),      // Disable metrics
		core.WithOneShot(),           // Run once
		core.WithName("test-job"),    // Default test name
	}

	// Append user-provided options
	testOpts = append(testOpts, opts...)

	// Create Nautilus instance
	n, err := core.New(testOpts...)
	if err != nil {
		return nil, err
	}

	// Setup
	result.SetupError = job.Setup(ctx)
	if result.SetupError != nil {
		result.Duration = time.Since(start)
		return result, result.SetupError
	}

	// Execute
	execStart := time.Now()
	result.ExecuteError = job.Execute(ctx)
	result.ExecuteDuration = time.Since(execStart)

	// Teardown (always run, even if Execute failed)
	result.TeardownError = job.Teardown(ctx)

	result.Duration = time.Since(start)

	// Return the first error encountered
	if result.ExecuteError != nil {
		return result, result.ExecuteError
	}
	if result.TeardownError != nil {
		return result, result.TeardownError
	}

	// Suppress unused warning
	_ = n

	return result, nil
}
