package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/navica-dev/nautilus/pkg/enums"
)

// mockJob is a simple mock implementation of interfaces.Job
type mockJob struct {
	setupCalled    bool
	executeCalled  bool
	teardownCalled bool
	executeError   error
	executeDelay   time.Duration
	executeFn      func(ctx context.Context) error
}

func (m *mockJob) Setup(ctx context.Context) error {
	m.setupCalled = true
	return nil
}

func (m *mockJob) Execute(ctx context.Context) error {
	m.executeCalled = true
	if m.executeDelay > 0 {
		time.Sleep(m.executeDelay)
	}
	if m.executeFn != nil {
		return m.executeFn(ctx)
	}
	return m.executeError
}

func (m *mockJob) Teardown(ctx context.Context) error {
	m.teardownCalled = true
	return nil
}

// TestExecutionModeOneShot verifies one-shot execution mode
func TestExecutionModeOneShot(t *testing.T) {
	job := &mockJob{}

	n, err := New(
		WithName("test-oneshot"),
		WithOneShot(),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	if n.config.Execution.Mode != enums.ModeOneShot {
		t.Errorf("Expected ModeOneShot, got %v", n.config.Execution.Mode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = n.Run(ctx, job)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	if !job.setupCalled {
		t.Error("Setup was not called")
	}
	if !job.executeCalled {
		t.Error("Execute was not called")
	}
	if !job.teardownCalled {
		t.Error("Teardown was not called")
	}

	if n.GetState() != enums.StateStopped {
		t.Errorf("Expected StateStopped, got %v", n.GetState())
	}
}

// TestExecutionModePeriodic verifies periodic execution mode with interval
func TestExecutionModePeriodic(t *testing.T) {
	job := &mockJob{}
	execCount := 0

	job.executeFn = func(ctx context.Context) error {
		execCount++
		return nil
	}

	n, err := New(
		WithName("test-periodic"),
		WithInterval(100*time.Millisecond),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	if n.config.Execution.Mode != enums.ModePeriodic {
		t.Errorf("Expected ModePeriodic, got %v", n.config.Execution.Mode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	err = n.Run(ctx, job)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run failed: %v", err)
	}

	// Should have run at least 2-3 times in 350ms with 100ms interval
	if execCount < 2 {
		t.Errorf("Expected at least 2 executions, got %d", execCount)
	}
}

// TestRetryBehavior verifies retry logic on failure
func TestRetryBehavior(t *testing.T) {
	job := &mockJob{}
	attempts := 0

	job.executeFn = func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil // Succeed on 3rd attempt
	}

	n, err := New(
		WithName("test-retry"),
		WithOneShot(),
		WithRetry(5, 10*time.Millisecond),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = n.Run(ctx, job)
	if err != nil {
		t.Errorf("Run should succeed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryExhaustion verifies retry gives up after max attempts
func TestRetryExhaustion(t *testing.T) {
	job := &mockJob{
		executeError: errors.New("persistent failure"),
	}

	n, err := New(
		WithName("test-retry-exhaustion"),
		WithOneShot(),
		WithRetry(2, 10*time.Millisecond),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = n.Run(ctx, job)
	if err == nil {
		t.Error("Expected error after retry exhaustion")
	}

	// Should have tried 3 times (initial + 2 retries)
	runCount := n.GetRunCount()
	if runCount != 1 {
		t.Errorf("Expected 1 run with 3 attempts, got %d runs", runCount)
	}
}

// TestStateTransitions verifies lifecycle state transitions
func TestStateTransitions(t *testing.T) {
	job := &mockJob{}

	n, err := New(
		WithName("test-states"),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	if n.GetState() != enums.StateCreated {
		t.Errorf("Expected StateCreated initially, got %v", n.GetState())
	}

	// We can't easily test state transitions during Run without mocking,
	// but we can verify the final state after one-shot completion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = n.Run(ctx, job)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	if n.GetState() != enums.StateStopped {
		t.Errorf("Expected StateStopped after completion, got %v", n.GetState())
	}
}

// TestGracefulShutdown verifies graceful drain during shutdown
func TestGracefulShutdown(t *testing.T) {
	job := &mockJob{
		executeDelay: 200 * time.Millisecond,
	}

	n, err := New(
		WithName("test-graceful-shutdown"),
		WithInterval(50*time.Millisecond),
		WithGracePeriod(500*time.Millisecond),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Trigger shutdown after 150ms
		time.Sleep(150 * time.Millisecond)
		n.Shutdown(ctx)
		cancel()
	}()

	err = n.Run(ctx, job)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Run failed: %v", err)
	}

	// Should have transitioned to shutting down
	state := n.GetState()
	if state != enums.StateShuttingDown && state != enums.StateStopped {
		t.Errorf("Expected StateShuttingDown or StateStopped, got %v", state)
	}
}

// TestEventHooks verifies event hooks are called
func TestEventHooks(t *testing.T) {
	job := &mockJob{}

	var (
		runStartCalled    bool
		runCompleteCalled bool
		shutdownCalled    bool
	)

	n, err := New(
		WithName("test-hooks"),
		WithOneShot(),
		WithAPI(false, 0),
		WithMetrics(false),
		WithOnRunStart(func(rc *RunContext) {
			runStartCalled = true
			if rc.RunID == "" {
				t.Error("RunContext.RunID is empty")
			}
		}),
		WithOnRunComplete(func(rc *RunContext, err error) {
			runCompleteCalled = true
		}),
		WithOnShutdown(func() {
			shutdownCalled = true
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = n.Run(ctx, job)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	if !runStartCalled {
		t.Error("OnRunStart hook was not called")
	}
	if !runCompleteCalled {
		t.Error("OnRunComplete hook was not called")
	}
	if !shutdownCalled {
		t.Error("OnShutdown hook was not called")
	}
}

// TestRunContext verifies RunContext is available in Execute
func TestRunContext(t *testing.T) {
	job := &mockJob{}

	job.executeFn = func(ctx context.Context) error {
		rc := RunContextFrom(ctx)
		if rc.RunID == "" {
			return errors.New("RunContext.RunID is empty")
		}
		if rc.RunCount != 1 {
			return errors.New("RunContext.RunCount is not 1")
		}
		if rc.StartTime.IsZero() {
			return errors.New("RunContext.StartTime is zero")
		}
		return nil
	}

	n, err := New(
		WithName("test-runcontext"),
		WithOneShot(),
		WithAPI(false, 0),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = n.Run(ctx, job)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
}

// TestConfigOptions verifies option ordering and merging
func TestConfigOptions(t *testing.T) {
	n, err := New(
		WithName("initial-name"),
		WithDescription("test description"),
		WithTimeout(5*time.Minute),
		WithMaxConsecutiveFailures(5),
		WithGracePeriod(1*time.Minute),
	)
	if err != nil {
		t.Fatalf("Failed to create Nautilus: %v", err)
	}

	if n.config.Job.Name != "initial-name" {
		t.Errorf("Expected name 'initial-name', got '%s'", n.config.Job.Name)
	}

	if n.config.Job.Description != "test description" {
		t.Errorf("Expected description 'test description', got '%s'", n.config.Job.Description)
	}

	if n.config.Execution.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", n.config.Execution.Timeout)
	}

	if n.config.Health.MaxConsecutiveFailures != 5 {
		t.Errorf("Expected max failures 5, got %d", n.config.Health.MaxConsecutiveFailures)
	}

	if n.config.Shutdown.GracePeriod != 1*time.Minute {
		t.Errorf("Expected grace period 1m, got %v", n.config.Shutdown.GracePeriod)
	}
}
