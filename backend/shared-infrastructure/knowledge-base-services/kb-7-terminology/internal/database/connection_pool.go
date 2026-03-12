package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

// ConnectionPoolConfig holds configuration for database connection pooling
type ConnectionPoolConfig struct {
	MaxOpenConnections    int           `json:"max_open_connections"`
	MaxIdleConnections    int           `json:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `json:"connection_max_lifetime"`
	ConnectionMaxIdleTime time.Duration `json:"connection_max_idle_time"`
	
	// Performance tuning
	SlowQueryThreshold    time.Duration `json:"slow_query_threshold"`
	EnableMetrics         bool          `json:"enable_metrics"`
	HealthCheckInterval   time.Duration `json:"health_check_interval"`
}

// DefaultConnectionPoolConfig returns sensible defaults for KB-7 terminology service
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxOpenConnections:    50,  // Handle high concurrent load
		MaxIdleConnections:    10,  // Keep connections ready
		ConnectionMaxLifetime: 30 * time.Minute,
		ConnectionMaxIdleTime: 5 * time.Minute,
		SlowQueryThreshold:    100 * time.Millisecond,
		EnableMetrics:         true,
		HealthCheckInterval:   30 * time.Second,
	}
}

// HighPerformanceConnectionPoolConfig returns configuration optimized for high-throughput operations
func HighPerformanceConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxOpenConnections:    100, // Higher concurrency for batch operations
		MaxIdleConnections:    25,  // More idle connections ready
		ConnectionMaxLifetime: 20 * time.Minute,
		ConnectionMaxIdleTime: 3 * time.Minute,
		SlowQueryThreshold:    50 * time.Millisecond,
		EnableMetrics:         true,
		HealthCheckInterval:   15 * time.Second,
	}
}

// ConnectionPoolManager manages database connection pooling with performance optimization
type ConnectionPoolManager struct {
	db     *sql.DB
	config ConnectionPoolConfig
	logger *logrus.Logger
	
	// Metrics
	connectionsInUse   prometheus.Gauge
	connectionsIdle    prometheus.Gauge
	connectionDuration prometheus.Histogram
	slowQueries       prometheus.Counter
}

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(db *sql.DB, config ConnectionPoolConfig, logger *logrus.Logger) *ConnectionPoolManager {
	manager := &ConnectionPoolManager{
		db:     db,
		config: config,
		logger: logger,
	}
	
	// Configure the connection pool
	manager.configurePool()
	
	// Initialize metrics if enabled
	if config.EnableMetrics {
		manager.initializeMetrics()
	}
	
	// Start health check routine
	go manager.healthCheckRoutine()
	
	return manager
}

// configurePool applies the connection pool configuration
func (m *ConnectionPoolManager) configurePool() {
	m.db.SetMaxOpenConns(m.config.MaxOpenConnections)
	m.db.SetMaxIdleConns(m.config.MaxIdleConnections)
	m.db.SetConnMaxLifetime(m.config.ConnectionMaxLifetime)
	m.db.SetConnMaxIdleTime(m.config.ConnectionMaxIdleTime)
	
	m.logger.WithFields(logrus.Fields{
		"max_open_connections":     m.config.MaxOpenConnections,
		"max_idle_connections":     m.config.MaxIdleConnections,
		"connection_max_lifetime":  m.config.ConnectionMaxLifetime,
		"connection_max_idle_time": m.config.ConnectionMaxIdleTime,
	}).Info("Database connection pool configured")
}

// initializeMetrics sets up Prometheus metrics for connection pool monitoring
func (m *ConnectionPoolManager) initializeMetrics() {
	m.connectionsInUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kb7_db_connections_in_use",
		Help: "Number of database connections currently in use",
	})
	
	m.connectionsIdle = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kb7_db_connections_idle",
		Help: "Number of idle database connections",
	})
	
	m.connectionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "kb7_db_connection_duration_seconds",
		Help: "Duration of database connections",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
	})
	
	m.slowQueries = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kb7_db_slow_queries_total",
		Help: "Total number of slow database queries",
	})
	
	// Register metrics
	prometheus.MustRegister(m.connectionsInUse)
	prometheus.MustRegister(m.connectionsIdle)
	prometheus.MustRegister(m.connectionDuration)
	prometheus.MustRegister(m.slowQueries)
}

// GetStats returns current connection pool statistics
func (m *ConnectionPoolManager) GetStats() map[string]interface{} {
	stats := m.db.Stats()
	
	return map[string]interface{}{
		"max_open_connections":     m.config.MaxOpenConnections,
		"max_idle_connections":     m.config.MaxIdleConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration":           stats.WaitDuration,
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
		"connection_utilization":   float64(stats.InUse) / float64(m.config.MaxOpenConnections) * 100,
		"idle_utilization":        float64(stats.Idle) / float64(m.config.MaxIdleConnections) * 100,
	}
}

// OptimizeForBatchOperations temporarily increases connection pool size for batch operations
func (m *ConnectionPoolManager) OptimizeForBatchOperations(batchSize int) func() {
	// Calculate optimal connection count for batch size
	optimalConnections := m.calculateOptimalConnections(batchSize)
	
	// Store original values
	originalMaxOpen := m.config.MaxOpenConnections
	originalMaxIdle := m.config.MaxIdleConnections
	
	// Increase connections for batch processing
	if optimalConnections > m.config.MaxOpenConnections {
		m.config.MaxOpenConnections = optimalConnections
		m.config.MaxIdleConnections = optimalConnections / 4 // 25% idle
		m.configurePool()
		
		m.logger.WithFields(logrus.Fields{
			"batch_size":          batchSize,
			"original_max_open":   originalMaxOpen,
			"optimized_max_open":  optimalConnections,
		}).Info("Connection pool optimized for batch operation")
	}
	
	// Return cleanup function
	return func() {
		m.config.MaxOpenConnections = originalMaxOpen
		m.config.MaxIdleConnections = originalMaxIdle
		m.configurePool()
		
		m.logger.Info("Connection pool restored to original configuration")
	}
}

// calculateOptimalConnections determines optimal connection count based on batch size
func (m *ConnectionPoolManager) calculateOptimalConnections(batchSize int) int {
	// Rule of thumb: 1 connection per 100 items in batch, with reasonable bounds
	optimal := (batchSize / 100) + 5 // Base connections
	
	// Bounds checking
	if optimal < 10 {
		optimal = 10
	}
	if optimal > 200 {
		optimal = 200
	}
	
	return optimal
}

// ExecuteWithOptimization executes a function with connection pool optimization
func (m *ConnectionPoolManager) ExecuteWithOptimization(batchSize int, fn func() error) error {
	cleanup := m.OptimizeForBatchOperations(batchSize)
	defer cleanup()
	
	// Wait a moment for connections to be established
	time.Sleep(100 * time.Millisecond)
	
	return fn()
}

// healthCheckRoutine periodically checks connection pool health
func (m *ConnectionPoolManager) healthCheckRoutine() {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		m.performHealthCheck()
	}
}

// performHealthCheck checks connection pool health and updates metrics
func (m *ConnectionPoolManager) performHealthCheck() {
	stats := m.db.Stats()
	
	// Update metrics if enabled
	if m.config.EnableMetrics {
		m.connectionsInUse.Set(float64(stats.InUse))
		m.connectionsIdle.Set(float64(stats.Idle))
	}
	
	// Log warnings for potential issues
	utilizationPercent := float64(stats.InUse) / float64(m.config.MaxOpenConnections) * 100
	if utilizationPercent > 80 {
		m.logger.WithFields(logrus.Fields{
			"utilization_percent": utilizationPercent,
			"in_use":             stats.InUse,
			"max_open":           m.config.MaxOpenConnections,
		}).Warn("High database connection utilization")
	}
	
	if stats.WaitCount > 0 && stats.WaitDuration > time.Second {
		m.logger.WithFields(logrus.Fields{
			"wait_count":    stats.WaitCount,
			"wait_duration": stats.WaitDuration,
		}).Warn("Database connection waits detected")
	}
	
	// Store stats in database for historical analysis
	go m.recordConnectionStats(stats)
}

// recordConnectionStats stores connection statistics in the database
func (m *ConnectionPoolManager) recordConnectionStats(stats sql.DBStats) {
	query := `
		INSERT INTO connection_pool_stats (
			active_connections, idle_connections, total_connections, max_connections,
			connection_wait_time_ms
		) VALUES ($1, $2, $3, $4, $5)
	`
	
	waitTimeMs := float64(stats.WaitDuration.Nanoseconds()) / 1e6
	
	_, err := m.db.Exec(query, 
		stats.InUse, 
		stats.Idle, 
		stats.OpenConnections, 
		m.config.MaxOpenConnections,
		waitTimeMs,
	)
	
	if err != nil {
		m.logger.WithError(err).Debug("Failed to record connection pool stats")
	}
}

// ValidateConnection performs a simple ping to validate the connection
func (m *ConnectionPoolManager) ValidateConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return m.db.PingContext(ctx)
}

// GetConnectionForBatch returns an optimized connection configuration for batch operations
func (m *ConnectionPoolManager) GetConnectionForBatch() *sql.DB {
	// This could return a specialized connection with different settings
	// For now, return the main database connection
	return m.db
}

// RecordSlowQuery records metrics for slow queries
func (m *ConnectionPoolManager) RecordSlowQuery(query string, duration time.Duration) {
	if duration > m.config.SlowQueryThreshold {
		if m.config.EnableMetrics {
			m.slowQueries.Inc()
		}
		
		m.logger.WithFields(logrus.Fields{
			"query":    query[:min(len(query), 200)], // Truncate for logging
			"duration": duration,
		}).Warn("Slow query detected")
	}
}

// OptimizeQuery provides query optimization suggestions
func (m *ConnectionPoolManager) OptimizeQuery(query string) []string {
	suggestions := []string{}
	
	queryLower := strings.ToLower(query)
	
	// Basic query optimization suggestions
	if strings.Contains(queryLower, "select *") {
		suggestions = append(suggestions, "Consider selecting specific columns instead of using SELECT *")
	}
	
	if strings.Contains(queryLower, "where") && !strings.Contains(queryLower, "index") {
		suggestions = append(suggestions, "Ensure WHERE clause columns have appropriate indexes")
	}
	
	if strings.Contains(queryLower, "order by") && !strings.Contains(queryLower, "limit") {
		suggestions = append(suggestions, "Consider adding LIMIT clause to ORDER BY queries")
	}
	
	if strings.Count(queryLower, "join") > 3 {
		suggestions = append(suggestions, "Multiple JOINs detected - consider query restructuring or materialized views")
	}
	
	return suggestions
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Shutdown gracefully closes the connection pool
func (m *ConnectionPoolManager) Shutdown() error {
	m.logger.Info("Shutting down connection pool manager")
	
	// Log final statistics
	stats := m.GetStats()
	m.logger.WithFields(logrus.Fields{
		"final_stats": stats,
	}).Info("Final connection pool statistics")
	
	return m.db.Close()
}