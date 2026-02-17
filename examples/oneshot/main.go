package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/navica-dev/nautilus/core"
	"github.com/navica-dev/nautilus/pkg/logging"
	"github.com/rs/zerolog"
)

var name = "migration-job"

// MigrationJob demonstrates a one-shot job for database migrations or cleanup tasks
type MigrationJob struct {
	logger  zerolog.Logger
	version string
}

func (m *MigrationJob) Setup(ctx context.Context) error {
	m.logger = logging.GetLogger("migration-job")
	m.version = os.Getenv("MIGRATION_VERSION")
	if m.version == "" {
		m.version = "v1.0.0"
	}

	m.logger.Info().
		Str("version", m.version).
		Msg("Migration job initialized")

	return nil
}

func (m *MigrationJob) Execute(ctx context.Context) error {
	// Extract run context for logging
	rc := core.RunContextFrom(ctx)
	logger := rc.Logger

	logger.Info().Msg("Starting database migration")

	// Simulate migration steps
	steps := []string{
		"Creating new tables",
		"Migrating data",
		"Creating indexes",
		"Updating schema version",
	}

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			logger.Info().
				Int("step", i+1).
				Int("total", len(steps)).
				Str("action", step).
				Msg("Executing migration step")

			// Simulate work
			time.Sleep(500 * time.Millisecond)
		}
	}

	logger.Info().Msg("Migration completed successfully")
	return nil
}

func (m *MigrationJob) Teardown(ctx context.Context) error {
	m.logger.Info().Msg("Migration job cleanup complete")
	return nil
}

func main() {
	// Setup logging
	logging.Setup()

	// Create the job
	job := &MigrationJob{}

	// Configure Nautilus for one-shot execution
	n, err := core.New(
		core.WithName(name),
		core.WithDescription("Database migration job"),
		core.WithOneShot(),
		core.WithTimeout(5*time.Minute),
		core.WithRetry(2, 5*time.Second),
		core.WithAPI(true, 12911),
		core.WithMetrics(true),
		core.WithOnRunStart(func(rc *core.RunContext) {
			fmt.Printf("🚀 Starting migration run %d (ID: %s)\n", rc.RunCount, rc.RunID)
		}),
		core.WithOnRunComplete(func(rc *core.RunContext, err error) {
			if err != nil {
				fmt.Printf("❌ Migration failed: %v\n", err)
			} else {
				fmt.Printf("✅ Migration completed in %v\n", time.Since(rc.StartTime))
			}
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Nautilus: %v\n", err)
		os.Exit(1)
	}

	// Run the job
	if err := n.Run(context.Background(), job); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration job completed successfully")
}
