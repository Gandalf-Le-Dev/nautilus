package metrics

import (
	"testing"
	"time"

	"github.com/navica-dev/nautilus/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

// testClient is a shared client to avoid duplicate metric registration
// with promauto's global registry.
var testClient *Client

func init() {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Prometheus: config.PrometheusConfig{
			Enabled: true,
			Path:    "/metrics",
		},
	}
	testClient = NewClient(cfg)
	testClient.RegisterBasicMetrics("test-job")
}

func TestNewClient(t *testing.T) {
	if testClient == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestRecordRunSuccess(t *testing.T) {
	// Should not panic
	testClient.RecordRunSuccess(100 * time.Millisecond)
}

func TestRecordRunFailure(t *testing.T) {
	// Should not panic
	testClient.RecordRunFailure(50 * time.Millisecond)
}

func TestRecordRunStart(t *testing.T) {
	// Should not panic (no-op hook)
	testClient.RecordRunStart()
}

func TestDisabledClient(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: false,
		Prometheus: config.PrometheusConfig{
			Enabled: false,
		},
	}
	c := NewClient(cfg)

	// Should not panic even when disabled
	c.RecordRunSuccess(100 * time.Millisecond)
	c.RecordRunFailure(50 * time.Millisecond)
	c.RecordRunStart()
}

func TestCreateCustomCounter(t *testing.T) {
	counter := testClient.CreateCustomCounter("test_custom_counter", "A test counter", "label1")
	if counter == nil {
		t.Fatal("expected non-nil counter")
	}
}

func TestCreateCustomGauge(t *testing.T) {
	gauge := testClient.CreateCustomGauge("test_custom_gauge", "A test gauge", "label1")
	if gauge == nil {
		t.Fatal("expected non-nil gauge")
	}
}

func TestCreateCustomHistogram(t *testing.T) {
	histogram := testClient.CreateCustomHistogram("test_custom_histogram", "A test histogram", prometheus.DefBuckets, "label1")
	if histogram == nil {
		t.Fatal("expected non-nil histogram")
	}
}

func TestCreateCustomMetricsDisabled(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: false,
		Prometheus: config.PrometheusConfig{
			Enabled: false,
		},
	}
	c := NewClient(cfg)

	if counter := c.CreateCustomCounter("x", "x", "l"); counter != nil {
		t.Fatal("expected nil counter when disabled")
	}
	if gauge := c.CreateCustomGauge("x", "x", "l"); gauge != nil {
		t.Fatal("expected nil gauge when disabled")
	}
	if histogram := c.CreateCustomHistogram("x", "x", nil, "l"); histogram != nil {
		t.Fatal("expected nil histogram when disabled")
	}
}
