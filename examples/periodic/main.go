package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/navica-dev/nautilus/core"
	"github.com/navica-dev/nautilus/pkg/logging"
	"github.com/rs/zerolog"
)

var name = "data-sync-job"

// DataSyncJob demonstrates a periodic job that runs on an interval
type DataSyncJob struct {
	logger     zerolog.Logger
	syncCount  int
	lastSyncAt time.Time
}

func (d *DataSyncJob) Setup(ctx context.Context) error {
	d.logger = logging.GetLogger("data-sync")
	d.lastSyncAt = time.Now()

	d.logger.Info().Msg("Data sync job initialized")
	return nil
}

func (d *DataSyncJob) Execute(ctx context.Context) error {
	// Extract run context
	rc := core.RunContextFrom(ctx)
	logger := rc.Logger

	logger.Info().
		Time("last_sync", d.lastSyncAt).
		Msg("Starting data synchronization")

	// Simulate fetching data from external API
	itemsToSync := rand.Intn(50) + 10 // Random between 10-60 items

	for i := 0; i < itemsToSync; i++ {
		select {
		case <-ctx.Done():
			logger.Warn().
				Int("synced", i).
				Int("total", itemsToSync).
				Msg("Sync interrupted")
			return ctx.Err()
		default:
			// Simulate processing item
			time.Sleep(20 * time.Millisecond)
		}
	}

	d.syncCount++
	d.lastSyncAt = time.Now()

	logger.Info().
		Int("items_synced", itemsToSync).
		Int("total_syncs", d.syncCount).
		Msg("Data sync completed")

	return nil
}

func (d *DataSyncJob) Teardown(ctx context.Context) error {
	d.logger.Info().
		Int("total_syncs", d.syncCount).
		Msg("Data sync job shutting down")
	return nil
}

// HealthCheck implements the optional health check interface
func (d *DataSyncJob) HealthCheck(ctx context.Context) error {
	// Report unhealthy if last sync was more than 2 minutes ago
	if time.Since(d.lastSyncAt) > 2*time.Minute {
		return fmt.Errorf("last sync was %v ago", time.Since(d.lastSyncAt))
	}
	return nil
}

func (d *DataSyncJob) Name() string {
	return "data-sync"
}

func main() {
	// Setup logging
	logging.Setup()

	// Create the job
	job := &DataSyncJob{}

	// Configure Nautilus for periodic execution
	n, err := core.New(
		core.WithName(name),
		core.WithDescription("Periodic data synchronization job"),
		core.WithInterval(30*time.Second), // Run every 30 seconds
		core.WithTimeout(25*time.Second),  // Each run has 25 seconds max
		core.WithRetry(1, 2*time.Second),  // Retry once if failed
		core.WithAPI(true, 12911),
		core.WithMetrics(true),
		core.WithGracePeriod(1*time.Minute),
		core.WithOnRunStart(func(rc *core.RunContext) {
			fmt.Printf("🔄 Sync run #%d started\n", rc.RunCount)
		}),
		core.WithOnRunComplete(func(rc *core.RunContext, err error) {
			if err != nil {
				fmt.Printf("❌ Sync #%d failed: %v\n", rc.RunCount, err)
			} else {
				fmt.Printf("✅ Sync #%d completed in %v\n", rc.RunCount, time.Since(rc.StartTime))
			}
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Nautilus: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting periodic data sync job...")
	fmt.Println("Press Ctrl+C to stop")

	// Run the job (blocks until interrupted)
	if err := n.Run(context.Background(), job); err != nil {
		fmt.Fprintf(os.Stderr, "Job failed: %v\n", err)
		os.Exit(1)
	}
}
