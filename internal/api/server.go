package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/navica-dev/nautilus/internal/config"
	"github.com/navica-dev/nautilus/pkg/interfaces"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP API server
type Server struct {
	mux    *http.ServeMux
	server *http.Server
	config *config.APIConfig
	logger zerolog.Logger

	// Health check handlers
	healthCheckers []interfaces.HealthCheck

	// State provider
	stateProvider func() string

	// Version info
	version string

	// Start time
	startTime time.Time
}

// HealthResponse represents the health check response format
type HealthResponse struct {
	Status     string                     `json:"status"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
}

// ComponentHealth represents a single component's health
type ComponentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const requestIDKey contextKey = "requestID"

// NewServer creates a new API server
func NewServer(cfg *config.APIConfig, version string) *Server {
	mux := http.NewServeMux()

	// Setup logger
	logger := log.With().Str("component", "api-server").Logger()

	s := &Server{
		mux:            mux,
		config:         cfg,
		logger:         logger,
		healthCheckers: make([]interfaces.HealthCheck, 0),
		version:        version,
		startTime:      time.Now(),
	}

	// Register routes
	s.registerRoutes()

	// Create handler chain with middleware
	handler := s.recoverer(s.requestID(s.realIP(mux)))

	// Wrap with timeout
	handler = http.TimeoutHandler(handler, 30*time.Second, "request timeout")

	// Create server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: handler,
	}

	return s
}

// RegisterHealthChecker adds a health checker to the server
func (s *Server) RegisterHealthChecker(checker interfaces.HealthCheck) {
	s.healthCheckers = append(s.healthCheckers, checker)
}

// SetStateProvider sets the function that provides the current state
func (s *Server) SetStateProvider(provider func() string) {
	s.stateProvider = provider
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info().Int("port", s.config.Port).Msg("Starting API server")

	if s.config.TLS.Enabled {
		return s.server.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	}

	return s.server.ListenAndServe()
}

// StartAsync starts the server in a goroutine and returns a channel
// that will receive any startup/runtime error (or nil on clean shutdown).
func (s *Server) StartAsync() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()
	return errCh
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info().Msg("Stopping API server")
	return s.server.Shutdown(ctx)
}

// registerRoutes sets up the HTTP routes
func (s *Server) registerRoutes() {
	// Health check endpoint
	s.mux.HandleFunc("GET /health", s.handleHealth())

	// Readiness check endpoint
	s.mux.HandleFunc("GET /ready", s.handleReady())

	// Liveness check endpoint
	s.mux.HandleFunc("GET /live", s.handleLive())

	// Metrics endpoint
	s.mux.Handle("GET /metrics", promhttp.Handler())

	// Version endpoint
	s.mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": s.version})
	})

	// Debug endpoints
	s.mux.HandleFunc("GET /debug/config", s.handleConfig())
}

// Middleware functions

// requestID adds a unique request ID to the context and response headers
func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		w.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// realIP extracts the real IP address from proxy headers
func (s *Server) realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check X-Forwarded-For header
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Use the first IP in the list (before the first comma)
			for idx := 0; idx < len(xff); idx++ {
				if xff[idx] == ',' {
					r.RemoteAddr = xff[:idx]
					next.ServeHTTP(w, r)
					return
				}
			}
			r.RemoteAddr = xff
		} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
			r.RemoteAddr = xri
		}

		next.ServeHTTP(w, r)
	})
}

// recoverer recovers from panics and logs them
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				s.logger.Error().
					Interface("panic", rvr).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("Request panic recovered")

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Handler functions

// handleHealth creates the health check handler
func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		response := HealthResponse{
			Status:     "ok",
			Components: make(map[string]ComponentHealth),
			Timestamp:  time.Now(),
			Version:    s.version,
			Uptime:     time.Since(s.startTime).String(),
		}

		// Check all health checkers
		for _, checker := range s.healthCheckers {
			err := checker.HealthCheck(ctx)

			status := "ok"
			message := ""

			if err != nil {
				status = "error"
				message = err.Error()
				response.Status = "error" // Overall status is error if any component fails
			}

			response.Components[checker.Name()] = ComponentHealth{
				Status:  status,
				Message: message,
			}
		}

		w.Header().Set("Content-Type", "application/json")

		if response.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// handleReady creates the readiness check handler
func (s *Server) handleReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check state - ready only in "ready" or "executing" states
		if s.stateProvider != nil {
			state := s.stateProvider()
			if state != "ready" && state != "executing" {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("not ready: " + state))
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	}
}

// handleLive creates the liveness check handler
func (s *Server) handleLive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check state - alive unless stopped
		if s.stateProvider != nil {
			state := s.stateProvider()
			if state == "stopped" {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("stopped"))
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	}
}

// sanitizedAPIConfig is a copy of APIConfig with sensitive fields redacted.
type sanitizedAPIConfig struct {
	Enabled   bool   `json:"enabled"`
	Port      int    `json:"port"`
	Path      string `json:"path"`
	DebugMode bool   `json:"debugMode"`
	TLS       struct {
		Enabled bool `json:"enabled"`
	} `json:"tls"`
}

// handleConfig creates the config debug handler
func (s *Server) handleConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.config.DebugMode {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Debug endpoints disabled in production"))
			return
		}
		sanitized := sanitizedAPIConfig{
			Enabled:   s.config.Enabled,
			Port:      s.config.Port,
			Path:      s.config.Path,
			DebugMode: s.config.DebugMode,
		}
		sanitized.TLS.Enabled = s.config.TLS.Enabled
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sanitized)
	}
}
