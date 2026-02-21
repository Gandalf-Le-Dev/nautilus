package api

import (
	"context"
	"testing"
	"time"

	"github.com/navica-dev/nautilus/internal/config"
)

func TestServerStartupError(t *testing.T) {
	// Use an invalid port to force startup failure
	cfg := &config.APIConfig{
		Enabled: true,
		Port:    -1, // Invalid port
	}

	s := NewServer(cfg, "test")

	errCh := s.StartAsync()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected startup error for invalid port")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for startup error")
	}
}

func TestServerStartsSuccessfully(t *testing.T) {
	cfg := &config.APIConfig{
		Enabled: true,
		Port:    0, // OS-assigned port
	}

	s := NewServer(cfg, "test")

	errCh := s.StartAsync()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Should not have errored
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected startup error: %v", err)
		}
	default:
		// No error yet — good, server is running
	}

	// Clean shutdown
	s.Stop(context.Background())
}
