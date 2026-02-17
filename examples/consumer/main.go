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

var name = "message-consumer"

// Message represents a message from a queue
type Message struct {
	ID      string
	Payload string
	Attempt int
}

// MessageConsumer demonstrates a continuous job for queue consumption
type MessageConsumer struct {
	logger         zerolog.Logger
	queue          chan Message
	processed      int
	failed         int
	producerDone   chan struct{}
	producerCancel context.CancelFunc
}

func (m *MessageConsumer) Setup(ctx context.Context) error {
	m.logger = logging.GetLogger("consumer")
	m.queue = make(chan Message, 100)
	m.producerDone = make(chan struct{})

	m.logger.Info().Msg("Message consumer initialized")

	// Start a fake producer goroutine to simulate incoming messages
	producerCtx, cancel := context.WithCancel(ctx)
	m.producerCancel = cancel

	go m.fakeProducer(producerCtx)

	return nil
}

func (m *MessageConsumer) Execute(ctx context.Context) error {
	// Extract run context
	rc := core.RunContextFrom(ctx)
	logger := rc.Logger

	logger.Info().Msg("Starting message consumption")

	// Continuous consumption loop
	for {
		select {
		case <-ctx.Done():
			logger.Info().
				Int("processed", m.processed).
				Int("failed", m.failed).
				Msg("Context cancelled, stopping consumption")
			return ctx.Err()

		case msg, ok := <-m.queue:
			if !ok {
				logger.Info().Msg("Queue closed, stopping")
				return nil
			}

			// Process the message
			if err := m.processMessage(ctx, msg); err != nil {
				logger.Error().
					Err(err).
					Str("message_id", msg.ID).
					Msg("Failed to process message")
				m.failed++
			} else {
				logger.Debug().
					Str("message_id", msg.ID).
					Msg("Message processed successfully")
				m.processed++
			}

			// Log progress every 10 messages
			if (m.processed+m.failed)%10 == 0 {
				logger.Info().
					Int("processed", m.processed).
					Int("failed", m.failed).
					Msg("Processing progress")
			}
		}
	}
}

func (m *MessageConsumer) processMessage(ctx context.Context, msg Message) error {
	// Simulate message processing
	processingTime := time.Duration(rand.Intn(100)+50) * time.Millisecond

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(processingTime):
		// Simulate occasional failures (5% failure rate)
		if rand.Float32() < 0.05 {
			return fmt.Errorf("processing failed for message %s", msg.ID)
		}
		return nil
	}
}

func (m *MessageConsumer) Teardown(ctx context.Context) error {
	m.logger.Info().Msg("Shutting down consumer")

	// Stop the producer
	if m.producerCancel != nil {
		m.producerCancel()
	}

	// Wait for producer to stop
	select {
	case <-m.producerDone:
		m.logger.Debug().Msg("Producer stopped")
	case <-time.After(5 * time.Second):
		m.logger.Warn().Msg("Producer shutdown timeout")
	}

	// Close the queue
	close(m.queue)

	m.logger.Info().
		Int("total_processed", m.processed).
		Int("total_failed", m.failed).
		Msg("Consumer shutdown complete")

	return nil
}

// fakeProducer simulates messages arriving on a queue
func (m *MessageConsumer) fakeProducer(ctx context.Context) {
	defer close(m.producerDone)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	msgCount := 0

	for {
		select {
		case <-ctx.Done():
			m.logger.Info().Msg("Producer stopping")
			return

		case <-ticker.C:
			msgCount++
			msg := Message{
				ID:      fmt.Sprintf("msg-%d", msgCount),
				Payload: fmt.Sprintf("payload-%d", msgCount),
				Attempt: 1,
			}

			select {
			case m.queue <- msg:
				// Message sent
			case <-ctx.Done():
				return
			default:
				m.logger.Warn().Msg("Queue full, dropping message")
			}
		}
	}
}

// HealthCheck implements the optional health check interface
func (m *MessageConsumer) HealthCheck(ctx context.Context) error {
	queueLen := len(m.queue)
	if queueLen > 80 {
		return fmt.Errorf("queue backlog too high: %d messages", queueLen)
	}
	return nil
}

func (m *MessageConsumer) Name() string {
	return "message-consumer"
}

func main() {
	// Setup logging
	logging.Setup()

	// Create the job
	job := &MessageConsumer{}

	// Configure Nautilus for continuous execution
	n, err := core.New(
		core.WithName(name),
		core.WithDescription("Continuous message queue consumer"),
		core.WithContinuous(), // Run continuously
		core.WithAPI(true, 12911),
		core.WithMetrics(true),
		core.WithGracePeriod(30*time.Second),
		core.WithOnRunStart(func(rc *core.RunContext) {
			fmt.Println("🎯 Message consumer started")
		}),
		core.WithOnShutdown(func() {
			fmt.Println("⏸️  Gracefully shutting down consumer...")
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Nautilus: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting message consumer...")
	fmt.Println("Press Ctrl+C to stop gracefully")
	fmt.Println("Listening on :12911 for health checks")

	// Run the job (blocks until interrupted)
	if err := n.Run(context.Background(), job); err != nil {
		fmt.Fprintf(os.Stderr, "Consumer failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Consumer stopped")
}
