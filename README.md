# Nautilus

**A lifecycle framework for Go workloads**

Nautilus provides a clean, opinionated framework for building robust Go jobs with built-in observability, health checking, and lifecycle management. Whether you're building migrations, periodic sync jobs, or continuous queue consumers, Nautilus handles the infrastructure so you can focus on business logic.

[![Go Report Card](https://goreportcard.com/badge/github.com/navica-dev/nautilus)](https://goreportcard.com/report/github.com/navica-dev/nautilus)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/navica-dev/nautilus?status.svg)](https://godoc.org/github.com/navica-dev/nautilus)

## Features

- 🎯 **Three execution modes**: OneShot, Periodic, Continuous
- 🔄 **Automatic retries** with exponential backoff
- 🏥 **Health checks** with liveness and readiness probes
- 📊 **Prometheus metrics** out of the box
- 🔌 **Plugin system** for reusable components
- 🪝 **Lifecycle hooks** for custom instrumentation
- 🛡️ **Graceful shutdown** with configurable grace periods
- 🔍 **Structured logging** with zerolog
- ☸️ **Kubernetes-ready** with example manifests

## Installation

```bash
go get github.com/navica-dev/nautilus
```

## Quick Start

Here's a minimal one-shot job:

```go
package main

import (
    "context"
    "fmt"

    "github.com/navica-dev/nautilus/core"
)

type MyJob struct{}

func (m *MyJob) Setup(ctx context.Context) error {
    fmt.Println("Initializing...")
    return nil
}

func (m *MyJob) Execute(ctx context.Context) error {
    fmt.Println("Running job logic...")
    return nil
}

func (m *MyJob) Teardown(ctx context.Context) error {
    fmt.Println("Cleaning up...")
    return nil
}

func main() {
    n, _ := core.New(
        core.WithName("my-job"),
        core.WithOneShot(),
    )

    n.Run(context.Background(), &MyJob{})
}
```

## Execution Modes

Nautilus supports three distinct execution modes:

### OneShot

Execute once and exit. Perfect for migrations, cleanup tasks, or batch processing.

```go
n, _ := core.New(
    core.WithName("migration-job"),
    core.WithOneShot(),
    core.WithTimeout(5*time.Minute),
    core.WithRetry(2, 5*time.Second),
)
```

### Periodic

Execute repeatedly on a schedule using intervals or cron expressions.

```go
// Using interval
n, _ := core.New(
    core.WithName("sync-job"),
    core.WithInterval(30*time.Second),
    core.WithTimeout(25*time.Second),
)

// Using cron schedule
n, _ := core.New(
    core.WithName("hourly-job"),
    core.WithSchedule("0 * * * *"),  // Every hour
)
```

### Continuous

Execute once and block indefinitely. Your `Execute()` method runs its own loop. Ideal for queue consumers and long-running processes.

```go
n, _ := core.New(
    core.WithName("queue-consumer"),
    core.WithContinuous(),
    core.WithGracePeriod(30*time.Second),
)

func (m *MyJob) Execute(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case msg := <-m.queue:
            // Process message
        }
    }
}
```

## RunContext

Access execution metadata within your job:

```go
func (m *MyJob) Execute(ctx context.Context) error {
    rc := core.RunContextFrom(ctx)

    rc.Logger.Info().
        Str("run_id", rc.RunID).
        Int("run_count", rc.RunCount).
        Msg("Processing data")

    // Your logic here
    return nil
}
```

## Lifecycle Hooks

Add custom instrumentation at key lifecycle points:

```go
n, _ := core.New(
    core.WithName("my-job"),
    core.WithOnRunStart(func(rc *core.RunContext) {
        fmt.Printf("🚀 Run #%d started\n", rc.RunCount)
        // Send to observability platform
    }),
    core.WithOnRunComplete(func(rc *core.RunContext, err error) {
        if err != nil {
            // Alert on failure
        }
    }),
    core.WithOnShutdown(func() {
        // Flush buffers, close connections
    }),
)
```

## Health Checks

Implement the optional `HealthCheck` interface:

```go
func (m *MyJob) HealthCheck(ctx context.Context) error {
    if m.queueBacklog > 1000 {
        return fmt.Errorf("queue backlog too high: %d", m.queueBacklog)
    }
    return nil
}

func (m *MyJob) Name() string {
    return "my-job"
}
```

Health endpoints:
- `/health` - Overall health status with component details
- `/ready` - Ready to accept traffic (Ready or Executing states only)
- `/live` - Process is alive (returns error only if Stopped)
- `/metrics` - Prometheus metrics

## Configuration

Nautilus can be configured via code, YAML files, or both:

```go
n, _ := core.New(
    core.WithConfigPath("config.yaml"),  // Load from file
    core.WithName("my-job"),              // Override file config
    core.WithInterval(30*time.Second),
)
```

Example `config.yaml`:

```yaml
job:
  name: data-sync-job
  description: Synchronize data from external API

execution:
  mode: periodic
  interval: 30s
  timeout: 25s
  maxRetries: 2
  retryBackoff: 5s

api:
  enabled: true
  port: 12911

metrics:
  enabled: true
  prometheus:
    enabled: true
    path: /metrics

health:
  checkInterval: 30s
  maxConsecutiveFailures: 3

shutdown:
  gracePeriod: 60s
```

## Plugins

Nautilus includes a PostgreSQL plugin for common database operations:

```go
import plugin "github.com/navica-dev/nautilus/plugins/database"

pgConfig := plugin.PostgresPluginConfig{
    Job:        "my-job",
    ConnString: "postgres://user:pass@localhost:5432/db",
    MaxConns:   25,
}

pgPlugin := plugin.NewPostgresPlugin(&pgConfig, "namespace")

n, _ := core.New(
    core.WithName("db-job"),
    core.WithPlugin(pgPlugin),
)
```

The PostgreSQL plugin provides:
- Connection pooling with pgx
- Health checking
- Automatic Prometheus metrics (connections, query duration, errors)
- Query helpers and transaction support

## Kubernetes Deployment

Nautilus jobs are Kubernetes-native with proper health checks and graceful shutdown.

### Periodic Job (Deployment)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nautilus-periodic
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: nautilus
        image: your-registry/nautilus-job:latest
        ports:
        - containerPort: 12911
        livenessProbe:
          httpGet:
            path: /live
            port: 12911
        readinessProbe:
          httpGet:
            path: /ready
            port: 12911
```

### OneShot Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: nautilus-migration
spec:
  backoffLimit: 0  # Nautilus handles retries
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: nautilus
        image: your-registry/nautilus-migration:latest
```

### CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nautilus-scheduled
spec:
  schedule: "0 * * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
          - name: nautilus
            image: your-registry/nautilus-job:latest
```

See the [`deploy/`](./deploy) directory for complete examples including StatefulSets for consumers.

## Examples

The [`examples/`](./examples) directory contains complete working examples:

- **[oneshot](./examples/oneshot)** - Database migration job
- **[periodic](./examples/periodic)** - Data synchronization job with interval
- **[consumer](./examples/consumer)** - Queue consumer using continuous mode
- **[postgres](./examples/postgres)** - Database operations with the PostgreSQL plugin

To run an example:

```bash
# Start dependencies
docker compose -f .devenv/docker-compose.yml up -d

# Run example
go run ./examples/periodic/main.go
```

## Testing

Nautilus provides a test helper for synchronous job execution:

```go
import "github.com/navica-dev/nautilus/pkg/nautilustest"

func TestMyJob(t *testing.T) {
    job := &MyJob{}

    result, err := nautilustest.RunJob(
        context.Background(),
        job,
        core.WithName("test-job"),
        core.WithTimeout(5*time.Second),
    )

    if err != nil {
        t.Fatalf("Job failed: %v", err)
    }

    if result.ExecuteDuration > 1*time.Second {
        t.Errorf("Job too slow: %v", result.ExecuteDuration)
    }
}
```

## Observability

### Metrics

Nautilus exposes Prometheus metrics at `/metrics`:

- `nautilus_job_runs_total{job, status}` - Total job executions
- `nautilus_job_duration_seconds{job, status}` - Execution duration
- `nautilus_job_errors_total{job}` - Error count
- Plugin metrics (e.g., PostgreSQL connection pool, query performance)

### Logging

Structured JSON logs via zerolog:

```json
{
  "level": "info",
  "component": "nautilus-core",
  "job": "sync-job",
  "run": 42,
  "run_id": "uuid",
  "time": "2026-02-17T12:00:00Z",
  "message": "Job run completed"
}
```

### State Tracking

Monitor job lifecycle state:

```go
state := n.GetState()  // Created, Initializing, Ready, Executing, ShuttingDown, Stopped
```

## Development Setup

### Prerequisites

- Go 1.24+
- Docker and Docker Compose (for running examples)

### Setting Up the Development Environment

1. Clone the repository:

   ```bash
   git clone https://github.com/navica-dev/nautilus.git
   cd nautilus
   ```

2. Start the development environment:

   ```bash
   cd .devenv
   docker-compose up -d
   ```

   This starts:
   - PostgreSQL on port 5432
   - PgAdmin on port 5050
   - Prometheus on port 9090
   - Grafana on port 3000

3. Run examples:

   ```bash
   go run examples/periodic/main.go
   ```

4. Run tests:

   ```bash
   go test ./...
   ```

## Architecture

Nautilus follows a clean architecture with clear separation of concerns:

- **Core**: Lifecycle orchestration, scheduling, state management
- **Interfaces**: Clean contracts for jobs and plugins
- **Plugins**: Reusable components (database, messaging, etc.)
- **API**: Health checks and metrics endpoints
- **Config**: YAML-based configuration with programmatic overrides

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
