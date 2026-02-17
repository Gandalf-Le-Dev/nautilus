package metrics

import (
	"time"

	"github.com/navica-dev/nautilus/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

// Client manages metrics collection and reporting
type Client struct {
	// Configuration
	config *config.MetricsConfig

	// Prometheus metrics
	jobRuns       *prometheus.CounterVec
	jobRunsFailed *prometheus.CounterVec
	jobRunLatency *prometheus.HistogramVec

	// Labels
	labels map[string]string
}

// NewClient creates a new metrics client
func NewClient(cfg *config.MetricsConfig) *Client {
	c := &Client{
		config: cfg,
		labels: make(map[string]string),
	}

	// Initialize Prometheus if enabled
	if cfg.Prometheus.Enabled {
		c.initPrometheus()
	}

	return c
}

// initPrometheus initializes Prometheus metrics
func (c *Client) initPrometheus() {
	c.jobRuns = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nautilus_job_runs_total",
			Help: "Total number of job executions",
		},
		[]string{"job", "status"},
	)

	c.jobRunsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nautilus_job_runs_failed_total",
			Help: "Total number of failed job executions",
		},
		[]string{"job"},
	)

	c.jobRunLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nautilus_job_run_duration_seconds",
			Help:    "Duration of job executions in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
		},
		[]string{"job", "status"},
	)

	log.Debug().Msg("Prometheus metrics initialized")
}

// RegisterBasicMetrics sets up basic metrics for a job
func (c *Client) RegisterBasicMetrics(jobName string) {
	c.labels["job"] = jobName

	// Initialize counters with 0 values to ensure they appear in Prometheus
	if c.jobRuns != nil {
		c.jobRuns.WithLabelValues(jobName, "success").Add(0)
		c.jobRuns.WithLabelValues(jobName, "failure").Add(0)
	}

	if c.jobRunsFailed != nil {
		c.jobRunsFailed.WithLabelValues(jobName).Add(0)
	}

	log.Debug().Str("job", jobName).Msg("Basic metrics registered")
}

// RecordRunStart records the start of a job run
func (c *Client) RecordRunStart() {
	// Nothing to record at start time for Prometheus
	// This is a hook for other metrics systems that might need it
}

// RecordRunSuccess records a successful job run
func (c *Client) RecordRunSuccess(duration time.Duration) {
	if !c.config.Enabled {
		return
	}

	jobName := c.labels["job"]

	if c.jobRuns != nil {
		c.jobRuns.WithLabelValues(jobName, "success").Inc()
	}

	if c.jobRunLatency != nil {
		c.jobRunLatency.WithLabelValues(jobName, "success").Observe(duration.Seconds())
	}
}

// RecordRunFailure records a failed job run
func (c *Client) RecordRunFailure(duration time.Duration) {
	if !c.config.Enabled {
		return
	}

	jobName := c.labels["job"]

	if c.jobRuns != nil {
		c.jobRuns.WithLabelValues(jobName, "failure").Inc()
	}

	if c.jobRunsFailed != nil {
		c.jobRunsFailed.WithLabelValues(jobName).Inc()
	}

	if c.jobRunLatency != nil {
		c.jobRunLatency.WithLabelValues(jobName, "failure").Observe(duration.Seconds())
	}
}

// CreateCustomCounter creates a custom counter metric
func (c *Client) CreateCustomCounter(name, help string, labelNames ...string) *prometheus.CounterVec {
	if !c.config.Enabled || !c.config.Prometheus.Enabled {
		return nil
	}

	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labelNames,
	)
}

// CreateCustomGauge creates a custom gauge metric
func (c *Client) CreateCustomGauge(name, help string, labelNames ...string) *prometheus.GaugeVec {
	if !c.config.Enabled || !c.config.Prometheus.Enabled {
		return nil
	}

	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labelNames,
	)
}

// CreateCustomHistogram creates a custom histogram metric
func (c *Client) CreateCustomHistogram(
	name, help string,
	buckets []float64,
	labelNames ...string,
) *prometheus.HistogramVec {
	if !c.config.Enabled || !c.config.Prometheus.Enabled {
		return nil
	}

	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		labelNames,
	)
}
