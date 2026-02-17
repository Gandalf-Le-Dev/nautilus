package core

import (
	"context"

	"github.com/rs/zerolog"
	"time"
)

// RunContext contains metadata about the current job execution
type RunContext struct {
	// RunID is a unique identifier for this execution
	RunID string

	// RunCount is the sequential number of this execution
	RunCount int

	// Logger is a logger pre-configured with run metadata
	Logger zerolog.Logger

	// StartTime is when this execution started
	StartTime time.Time
}

// runContextKey is the context key for RunContext
type runContextKey struct{}

// WithRunContext returns a new context with the RunContext embedded
func WithRunContext(ctx context.Context, rc *RunContext) context.Context {
	return context.WithValue(ctx, runContextKey{}, rc)
}

// RunContextFrom extracts the RunContext from the context
// Returns a zero-value RunContext if not present
func RunContextFrom(ctx context.Context) RunContext {
	if rc, ok := ctx.Value(runContextKey{}).(*RunContext); ok && rc != nil {
		return *rc
	}
	// Return zero-value struct, never nil
	return RunContext{}
}
