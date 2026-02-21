package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDebugConfigEndpointSanitized(t *testing.T) {
	cfg := &config.APIConfig{
		Enabled:   true,
		Port:      12911,
		DebugMode: true,
		TLS: config.TLSConfig{
			Enabled:  true,
			CertFile: "/secret/cert.pem",
			KeyFile:  "/secret/key.pem",
		},
	}

	s := NewServer(cfg, "test")

	req := httptest.NewRequest("GET", "/debug/config", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "/secret/cert.pem") {
		t.Fatal("config output should not contain TLS cert path")
	}
	if strings.Contains(body, "/secret/key.pem") {
		t.Fatal("config output should not contain TLS key path")
	}
}

func TestDebugConfigEndpointDisabled(t *testing.T) {
	cfg := &config.APIConfig{
		Enabled:   true,
		Port:      12911,
		DebugMode: false,
	}

	s := NewServer(cfg, "test")

	req := httptest.NewRequest("GET", "/debug/config", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when debug disabled, got %d", w.Code)
	}
}

func TestRateLimiting(t *testing.T) {
	cfg := &config.APIConfig{
		Enabled: true,
		Port:    0,
	}

	s := NewServer(cfg, "test")

	// Hammer the endpoint beyond rate limit
	var rejected int
	for i := range 150 {
		_ = i
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		s.handler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			rejected++
		}
	}

	if rejected == 0 {
		t.Fatal("expected some requests to be rate limited")
	}
}
