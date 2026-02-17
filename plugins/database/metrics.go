package plugins

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PostgresMetrics holds all metrics for the PostgreSQL plugin
type PostgresMetrics struct {
	// Query metrics
	queriesTotal       *prometheus.CounterVec
	queryDuration      *prometheus.HistogramVec
	rowsProcessed      *prometheus.CounterVec
	preparedQueryUsage *prometheus.CounterVec

	// Connection pool metrics
	poolActiveConnections  *prometheus.GaugeVec
	poolIdleConnections    *prometheus.GaugeVec
	poolTotalConnections   *prometheus.GaugeVec
	poolMaxConnections     *prometheus.GaugeVec
	poolConnectionRequests *prometheus.CounterVec
	poolConnectionTimeouts *prometheus.CounterVec
	poolConnectionWaitTime *prometheus.HistogramVec
	poolConnectionLifetime *prometheus.HistogramVec

	// Database metrics
	connectionErrors      *prometheus.CounterVec
	connectionAcquireTime *prometheus.HistogramVec
	pingDuration          *prometheus.HistogramVec

	// Transaction metrics
	transactionsTotal       *prometheus.CounterVec
	transactionDuration     *prometheus.HistogramVec
	transactionOperations   *prometheus.CounterVec
	transactionRollbacks    *prometheus.CounterVec
	transactionSavepoints   *prometheus.CounterVec

	// Batch metrics
	batchOperationsTotal   *prometheus.CounterVec
	batchOperationDuration *prometheus.HistogramVec
	batchSize              *prometheus.HistogramVec

	// Copy metrics
	copyOperationsTotal *prometheus.CounterVec
	copyRowsProcessed   *prometheus.CounterVec
	copyDuration        *prometheus.HistogramVec

	// Listen/Notify metrics
	notificationsReceived *prometheus.CounterVec
	listenerErrors        *prometheus.CounterVec

	// Error metrics
	operationErrors *prometheus.CounterVec
}

// NewPostgresMetrics initializes and registers all PostgreSQL metrics
func NewPostgresMetrics(namespace string) *PostgresMetrics {
	m := &PostgresMetrics{}

	// Query metrics
	m.queriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_queries_total",
			Help:      "Total number of PostgreSQL queries executed",
		},
		[]string{"job", "database", "query_type", "status"},
	)

	m.queryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_query_duration_seconds",
			Help:      "Duration of PostgreSQL queries in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"job", "database", "query_type", "table"},
	)

	m.rowsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_rows_processed_total",
			Help:      "Total number of rows processed in PostgreSQL",
		},
		[]string{"job", "database", "operation", "table"},
	)

	m.preparedQueryUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_prepared_queries_total",
			Help:      "Total number of prepared statements used",
		},
		[]string{"job", "database", "query_name"},
	)

	// Connection pool metrics
	m.poolActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "postgres_pool_active_connections",
			Help:      "Current number of active connections in the pool",
		},
		[]string{"job", "database"},
	)

	m.poolIdleConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "postgres_pool_idle_connections",
			Help:      "Current number of idle connections in the pool",
		},
		[]string{"job", "database"},
	)

	m.poolTotalConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "postgres_pool_total_connections",
			Help:      "Total number of connections in the pool",
		},
		[]string{"job", "database"},
	)

	m.poolMaxConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "postgres_pool_max_connections",
			Help:      "Maximum number of connections allowed in the pool",
		},
		[]string{"job", "database"},
	)

	m.poolConnectionRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_pool_connection_requests_total",
			Help:      "Total number of connection requests made to the pool",
		},
		[]string{"job", "database"},
	)

	m.poolConnectionTimeouts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_pool_connection_timeouts_total",
			Help:      "Total number of connection timeouts that occurred",
		},
		[]string{"job", "database"},
	)

	m.poolConnectionWaitTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_pool_connection_wait_seconds",
			Help:      "Time spent waiting for a connection from the pool",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"job", "database"},
	)

	m.poolConnectionLifetime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_pool_connection_lifetime_seconds",
			Help:      "Lifetime of connections in the pool",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 10), // From 1s to ~17min
		},
		[]string{"job", "database"},
	)

	// Database metrics
	m.connectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_connection_errors_total",
			Help:      "Total number of PostgreSQL connection errors",
		},
		[]string{"job", "database", "error_type"},
	)

	m.connectionAcquireTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_connection_acquire_seconds",
			Help:      "Time taken to acquire a database connection",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"job", "database"},
	)

	m.pingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_ping_duration_seconds",
			Help:      "Duration of PostgreSQL ping operations",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 8), // From 1ms to ~0.25s
		},
		[]string{"job", "database"},
	)

	// Transaction metrics
	m.transactionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_transactions_total",
			Help:      "Total number of PostgreSQL transactions",
		},
		[]string{"job", "database", "status"}, // status: committed, rolled_back
	)

	m.transactionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_transaction_duration_seconds",
			Help:      "Duration of PostgreSQL transactions",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12), // From 1ms to ~4s
		},
		[]string{"job", "database"},
	)

	m.transactionOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_transaction_operations_total",
			Help:      "Total number of operations within PostgreSQL transactions",
		},
		[]string{"job", "database", "operation_type"}, // operation_type: query, exec
	)

	m.transactionRollbacks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_transaction_rollbacks_total",
			Help:      "Total number of PostgreSQL transaction rollbacks",
		},
		[]string{"job", "database", "reason"}, // reason: error, explicit
	)

	m.transactionSavepoints = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_transaction_savepoints_total",
			Help:      "Total number of PostgreSQL transaction savepoints",
		},
		[]string{"job", "database", "operation"}, // operation: create, release, rollback_to
	)

	// Batch metrics
	m.batchOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_batch_operations_total",
			Help:      "Total number of PostgreSQL batch operations",
		},
		[]string{"job", "database"},
	)

	m.batchOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_batch_operation_duration_seconds",
			Help:      "Duration of PostgreSQL batch operations",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"job", "database"},
	)

	m.batchSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_batch_size",
			Help:      "Size of PostgreSQL batch operations",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"job", "database"},
	)

	// Copy metrics
	m.copyOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_copy_operations_total",
			Help:      "Total number of PostgreSQL COPY operations",
		},
		[]string{"job", "database", "table", "status"},
	)

	m.copyRowsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_copy_rows_processed_total",
			Help:      "Total number of rows processed in PostgreSQL COPY operations",
		},
		[]string{"job", "database", "table"},
	)

	m.copyDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "postgres_copy_duration_seconds",
			Help:      "Duration of PostgreSQL COPY operations",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 10), // From 10ms to ~10s
		},
		[]string{"job", "database", "table"},
	)

	// Listen/Notify metrics
	m.notificationsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_notifications_received_total",
			Help:      "Total number of PostgreSQL notifications received",
		},
		[]string{"job", "database", "channel"},
	)

	m.listenerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_listener_errors_total",
			Help:      "Total number of PostgreSQL listener errors",
		},
		[]string{"job", "database", "channel", "error_type"},
	)

	// Error metrics
	m.operationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postgres_operation_errors_total",
			Help:      "Total number of errors during PostgreSQL operations",
		},
		[]string{"job", "database", "operation", "error_type"},
	)

	return m
}

// QueryTotal increments the query counter
func (m *PostgresMetrics) QueryTotal(job, database, queryType, status string) {
	m.queriesTotal.WithLabelValues(job, database, queryType, status).Inc()
}

// ObserveQueryDuration records the duration of a query
func (m *PostgresMetrics) ObserveQueryDuration(job, database, queryType, table string, durationSeconds float64) {
	m.queryDuration.WithLabelValues(job, database, queryType, table).Observe(durationSeconds)
}

// RowsProcessed increments the rows processed counter
func (m *PostgresMetrics) RowsProcessed(job, database, operation, table string, count int) {
	m.rowsProcessed.WithLabelValues(job, database, operation, table).Add(float64(count))
}

// PreparedQueryUsage increments the prepared query usage counter
func (m *PostgresMetrics) PreparedQueryUsage(job, database, queryName string) {
	m.preparedQueryUsage.WithLabelValues(job, database, queryName).Inc()
}

// SetPoolActiveConnections sets the active connections gauge
func (m *PostgresMetrics) SetPoolActiveConnections(job, database string, count int) {
	m.poolActiveConnections.WithLabelValues(job, database).Set(float64(count))
}

// SetPoolIdleConnections sets the idle connections gauge
func (m *PostgresMetrics) SetPoolIdleConnections(job, database string, count int) {
	m.poolIdleConnections.WithLabelValues(job, database).Set(float64(count))
}

// SetPoolTotalConnections sets the total connections gauge
func (m *PostgresMetrics) SetPoolTotalConnections(job, database string, count int) {
	m.poolTotalConnections.WithLabelValues(job, database).Set(float64(count))
}

// SetPoolMaxConnections sets the max connections gauge
func (m *PostgresMetrics) SetPoolMaxConnections(job, database string, count int) {
	m.poolMaxConnections.WithLabelValues(job, database).Set(float64(count))
}

// PoolConnectionRequest increments the connection requests counter
func (m *PostgresMetrics) PoolConnectionRequest(job, database string) {
	m.poolConnectionRequests.WithLabelValues(job, database).Inc()
}

// PoolConnectionTimeout increments the connection timeout counter
func (m *PostgresMetrics) PoolConnectionTimeout(job, database string) {
	m.poolConnectionTimeouts.WithLabelValues(job, database).Inc()
}

// ObservePoolConnectionWaitTime records the time spent waiting for a connection
func (m *PostgresMetrics) ObservePoolConnectionWaitTime(job, database string, durationSeconds float64) {
	m.poolConnectionWaitTime.WithLabelValues(job, database).Observe(durationSeconds)
}

// ObservePoolConnectionLifetime records the lifetime of a connection
func (m *PostgresMetrics) ObservePoolConnectionLifetime(job, database string, durationSeconds float64) {
	m.poolConnectionLifetime.WithLabelValues(job, database).Observe(durationSeconds)
}

// ConnectionError increments the connection error counter
func (m *PostgresMetrics) ConnectionError(job, database, errorType string) {
	m.connectionErrors.WithLabelValues(job, database, errorType).Inc()
}

// ObserveConnectionAcquireTime records the time taken to acquire a connection
func (m *PostgresMetrics) ObserveConnectionAcquireTime(job, database string, durationSeconds float64) {
	m.connectionAcquireTime.WithLabelValues(job, database).Observe(durationSeconds)
}

// ObservePingDuration records the duration of a ping operation
func (m *PostgresMetrics) ObservePingDuration(job, database string, durationSeconds float64) {
	m.pingDuration.WithLabelValues(job, database).Observe(durationSeconds)
}

// TransactionTotal increments the transaction counter
func (m *PostgresMetrics) TransactionTotal(job, database, status string) {
	m.transactionsTotal.WithLabelValues(job, database, status).Inc()
}

// ObserveTransactionDuration records the duration of a transaction
func (m *PostgresMetrics) ObserveTransactionDuration(job, database string, durationSeconds float64) {
	m.transactionDuration.WithLabelValues(job, database).Observe(durationSeconds)
}

// TransactionOperation increments the transaction operation counter
func (m *PostgresMetrics) TransactionOperation(job, database, operationType string) {
	m.transactionOperations.WithLabelValues(job, database, operationType).Inc()
}

// TransactionRollback increments the transaction rollback counter
func (m *PostgresMetrics) TransactionRollback(job, database, reason string) {
	m.transactionRollbacks.WithLabelValues(job, database, reason).Inc()
}

// TransactionSavepoint increments the transaction savepoint counter
func (m *PostgresMetrics) TransactionSavepoint(job, database, operation string) {
	m.transactionSavepoints.WithLabelValues(job, database, operation).Inc()
}

// BatchOperationTotal increments the batch operation counter
func (m *PostgresMetrics) BatchOperationTotal(job, database string) {
	m.batchOperationsTotal.WithLabelValues(job, database).Inc()
}

// ObserveBatchOperationDuration records the duration of a batch operation
func (m *PostgresMetrics) ObserveBatchOperationDuration(job, database string, durationSeconds float64) {
	m.batchOperationDuration.WithLabelValues(job, database).Observe(durationSeconds)
}

// ObserveBatchSize records the size of a batch operation
func (m *PostgresMetrics) ObserveBatchSize(job, database string, size int) {
	m.batchSize.WithLabelValues(job, database).Observe(float64(size))
}

// CopyOperationTotal increments the copy operation counter
func (m *PostgresMetrics) CopyOperationTotal(job, database, table, status string) {
	m.copyOperationsTotal.WithLabelValues(job, database, table, status).Inc()
}

// CopyRowsProcessed increments the copy rows processed counter
func (m *PostgresMetrics) CopyRowsProcessed(job, database, table string, count int64) {
	m.copyRowsProcessed.WithLabelValues(job, database, table).Add(float64(count))
}

// ObserveCopyDuration records the duration of a copy operation
func (m *PostgresMetrics) ObserveCopyDuration(job, database, table string, durationSeconds float64) {
	m.copyDuration.WithLabelValues(job, database, table).Observe(durationSeconds)
}

// NotificationReceived increments the notification received counter
func (m *PostgresMetrics) NotificationReceived(job, database, channel string) {
	m.notificationsReceived.WithLabelValues(job, database, channel).Inc()
}

// ListenerError increments the listener error counter
func (m *PostgresMetrics) ListenerError(job, database, channel, errorType string) {
	m.listenerErrors.WithLabelValues(job, database, channel, errorType).Inc()
}

// OperationError increments the operation error counter
func (m *PostgresMetrics) OperationError(job, database, operation, errorType string) {
	m.operationErrors.WithLabelValues(job, database, operation, errorType).Inc()
}