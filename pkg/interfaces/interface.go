package interfaces

import (
	"context"
	"time"

	"github.com/navica-dev/nautilus/pkg/enums"
)

// Core interface for implementing nautilus Jobs
type Job interface {
	// Setup prepares the nautilus job for execution
	// It receives a context that may be used for cancellation
	Setup(ctx context.Context) error

	// Execute performs the job's primary function
	// This method will be called either once or periodically depending on config
	Execute(ctx context.Context) error

	// Teardown performs cleanup when the job is shutting down
	// This allows for graceful release of resources
	Teardown(ctx context.Context) error
}

// Configurable is an optional interface jobs can implement for more complex configuraton needs
type Configurable interface {
	// Configure allows a job to configure itself from structured configuration
	Configure(config map[string]any) error
}

// RunInfo provides informations about a job's execution
type RunInfo struct {
	// RunID is a unique identifier for this execution
	RunID string

	// StartTime is when the execution began
	StartTime time.Time

	// Duration is how long the execution took (or has been running)
	Duration time.Duration

	// Status indicates the current state of the execution
	Status enums.RunStatusEnum

	Error error
}

// HealthCheck is an optional interface jobs can implement
// to provide custom health status information
type HealthCheck interface {
	// Name returns the name of this health check component
	Name() string

	// HealthCheck reports on the health status of the job
	// Returns nil if healthy, or an error describing the issue
	HealthCheck(ctx context.Context) error
}
