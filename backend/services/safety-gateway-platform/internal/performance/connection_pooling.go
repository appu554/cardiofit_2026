package performance

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/internal/config"
	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/pkg/logger"
)

// ConnectionPoolManager manages multiple connection pools for different services
type ConnectionPoolManager struct {
	pools                map[string]*ConnectionPool
	config               *config.ConnectionPoolConfig
	logger               *logger.Logger
	
	// Resource management
	resourceMonitor      *ConnectionResourceMonitor
	healthChecker        *PoolHealthChecker
	loadBalancer         *ConnectionLoadBalancer
	
	// State management
	isRunning            int32 // atomic
	poolMetrics          *ConnectionPoolMetrics
	stopCh               chan struct{}
	mu                   sync.RWMutex
}

// ConnectionPool represents a pool of connections to a specific service
type ConnectionPool struct {
	name                 string
	config               *PoolConfiguration
	logger               *logger.Logger
	
	// Connection management
	connections          chan *PooledConnection
	activeConnections    map[string]*PooledConnection
	connectionFactory    ConnectionFactory
	
	// Health and monitoring
	isHealthy            int32 // atomic
	stats                *PoolStatistics
	healthChecker        *ConnectionHealthChecker
	
	// Resource control
	semaphore           chan struct{} // Controls max connections
	lastCleanup         time.Time
	cleanupInterval     time.Duration
	
	// Synchronization
	mu                  sync.RWMutex
	stopCh              chan struct{}
}

// PooledConnection wraps a connection with pooling metadata
type PooledConnection struct {
	id                  string
	conn                net.Conn
	httpClient          *http.Client
	createdAt           time.Time
	lastUsed            time.Time
	usageCount          int64
	isActive            int32 // atomic
	pool                *ConnectionPool
	
	// Connection state
	state               ConnectionState
	healthScore         float64
	responseTime        time.Duration
	errorCount          int32
	
	// Lifecycle management
	maxUsage            int64
	maxIdleTime         time.Duration
	expiresAt           time.Time
	
	mu                  sync.RWMutex
}

type ConnectionState string

const (
	StateIdle           ConnectionState = "idle"
	StateActive         ConnectionState = "active"
	StateFailed         ConnectionState = "failed"
	StateExpired        ConnectionState = "expired"
	StateClosing        ConnectionState = "closing"
)

// PoolConfiguration defines connection pool settings
type PoolConfiguration struct {
	Name                string        `json:"name"`
	ServiceEndpoint     string        `json:"service_endpoint"`
	
	// Pool sizing
	InitialSize         int           `json:"initial_size"`
	MinConnections      int           `json:"min_connections"`
	MaxConnections      int           `json:"max_connections"`
	
	// Connection lifecycle
	MaxConnectionAge    time.Duration `json:"max_connection_age"`
	MaxIdleTime         time.Duration `json:"max_idle_time"`
	MaxUsageCount       int64         `json:"max_usage_count"`
	
	// Timeouts
	ConnectionTimeout   time.Duration `json:"connection_timeout"`
	ReadTimeout         time.Duration `json:"read_timeout"`
	WriteTimeout        time.Duration `json:"write_timeout"`
	KeepAliveTimeout    time.Duration `json:"keep_alive_timeout"`
	
	// Health checking
	HealthCheckEnabled  bool          `json:"health_check_enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	UnhealthyThreshold  int           `json:"unhealthy_threshold"`
	HealthyThreshold    int           `json:"healthy_threshold"`
	
	// Retry and recovery
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	CircuitBreakerEnabled bool        `json:"circuit_breaker_enabled"`
	
	// TLS and security
	TLSEnabled          bool          `json:"tls_enabled"`
	TLSConfig           *TLSConfiguration `json:"tls_config,omitempty"`
	
	// Advanced features
	LoadBalancing       LoadBalancingStrategy `json:"load_balancing"`
	CompressionEnabled  bool          `json:"compression_enabled"`
	KeepAliveEnabled    bool          `json:"keep_alive_enabled"`
}

// TLSConfiguration defines TLS settings for connections
type TLSConfiguration struct {
	CertFile            string `json:"cert_file,omitempty"`
	KeyFile             string `json:"key_file,omitempty"`
	CAFile              string `json:"ca_file,omitempty"`
	ServerName          string `json:"server_name,omitempty"`
	InsecureSkipVerify  bool   `json:"insecure_skip_verify"`
	MinVersion          string `json:"min_version"`
	MaxVersion          string `json:"max_version"`
}

type LoadBalancingStrategy string

const (
	LoadBalanceRoundRobin    LoadBalancingStrategy = "round_robin"
	LoadBalanceLeastUsed     LoadBalancingStrategy = "least_used"
	LoadBalanceHealthBased   LoadBalancingStrategy = "health_based"
	LoadBalanceResponseTime  LoadBalancingStrategy = "response_time"
)

// ConnectionFactory creates new connections
type ConnectionFactory interface {
	CreateConnection(config *PoolConfiguration) (*PooledConnection, error)
	ValidateConnection(conn *PooledConnection) error
	CloseConnection(conn *PooledConnection) error
}

// HTTPConnectionFactory creates HTTP connections
type HTTPConnectionFactory struct {
	logger *logger.Logger
}

// PoolStatistics tracks connection pool performance
type PoolStatistics struct {
	PoolName            string        `json:"pool_name"`
	
	// Connection counts
	TotalConnections    int           `json:"total_connections"`
	ActiveConnections   int           `json:"active_connections"`
	IdleConnections     int           `json:"idle_connections"`
	FailedConnections   int           `json:"failed_connections"`
	
	// Usage statistics
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	
	// Performance metrics
	AverageResponseTime time.Duration `json:"average_response_time"`
	P95ResponseTime     time.Duration `json:"p95_response_time"`
	RequestsPerSecond   float64       `json:"requests_per_second"`
	ErrorRate           float64       `json:"error_rate"`
	
	// Pool efficiency
	HitRatio            float64       `json:"hit_ratio"`
	MissRatio           float64       `json:"miss_ratio"`
	ConnectionUtilization float64     `json:"connection_utilization"`
	
	// Health metrics
	HealthScore         float64       `json:"health_score"`
	IsHealthy           bool          `json:"is_healthy"`
	LastHealthCheck     time.Time     `json:"last_health_check"`
	
	// Lifecycle metrics
	TotalCreated        int64         `json:"total_created"`
	TotalDestroyed      int64         `json:"total_destroyed"`
	AverageLifetime     time.Duration `json:"average_lifetime"`
	
	mu                  sync.RWMutex
}

// ConnectionResourceMonitor monitors overall connection resource usage
type ConnectionResourceMonitor struct {
	config              *config.ResourceMonitorConfig
	logger              *logger.Logger
	
	// Resource tracking
	totalConnections    int32  // atomic
	totalMemoryUsage    int64  // atomic
	totalBandwidthUsage int64  // atomic
	
	// Resource limits
	maxConnections      int32
	maxMemoryUsage      int64
	maxBandwidthUsage   int64
	
	// Monitoring
	isRunning           bool
	monitoringInterval  time.Duration
	stopCh              chan struct{}
	
	// Alerts and thresholds
	alertThresholds     *ResourceThresholds
	alertCallback       func(alert *ResourceAlert)
	
	mu                  sync.RWMutex
}

// ResourceThresholds defines resource usage thresholds
type ResourceThresholds struct {
	ConnectionWarning   float64 `json:"connection_warning"`   // 80%
	ConnectionCritical  float64 `json:"connection_critical"`  // 95%
	MemoryWarning       float64 `json:"memory_warning"`       // 85%
	MemoryCritical      float64 `json:"memory_critical"`      // 95%
	BandwidthWarning    float64 `json:"bandwidth_warning"`    // 80%
	BandwidthCritical   float64 `json:"bandwidth_critical"`   // 90%
}

// ResourceAlert represents a resource usage alert
type ResourceAlert struct {
	Type                AlertType     `json:"type"`
	Resource            ResourceType  `json:"resource"`
	Level               AlertLevel    `json:"level"`
	CurrentUsage        float64       `json:"current_usage"`
	Threshold           float64       `json:"threshold"`
	Message             string        `json:"message"`
	Timestamp           time.Time     `json:"timestamp"`
	Recommendations     []string      `json:"recommendations"`
}

type AlertType string
type ResourceType string
type AlertLevel string

const (
	AlertTypeResourceUsage AlertType = "resource_usage"
	AlertTypePoolHealth    AlertType = "pool_health"
	AlertTypePerformance   AlertType = "performance"
	
	ResourceTypeConnections ResourceType = "connections"
	ResourceTypeMemory      ResourceType = "memory"
	ResourceTypeBandwidth   ResourceType = "bandwidth"
	
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// PoolHealthChecker monitors pool health
type PoolHealthChecker struct {
	pools               map[string]*ConnectionPool
	config              *config.HealthCheckConfig
	logger              *logger.Logger
	
	// Health checking
	isRunning           bool
	checkInterval       time.Duration
	healthHistory       map[string]*HealthHistory
	
	stopCh              chan struct{}
	mu                  sync.RWMutex
}

// HealthHistory tracks pool health over time
type HealthHistory struct {
	PoolName            string            `json:"pool_name"`
	HealthScores        []HealthScore     `json:"health_scores"`
	Incidents           []*HealthIncident `json:"incidents"`
	LastHealthy         time.Time         `json:"last_healthy"`
	ConsecutiveFailures int               `json:"consecutive_failures"`
	ConsecutiveSuccesses int              `json:"consecutive_successes"`
	
	mu                  sync.RWMutex
}

// HealthScore represents a point-in-time health measurement
type HealthScore struct {
	Timestamp           time.Time `json:"timestamp"`
	Score               float64   `json:"score"`        // 0.0 to 1.0
	ResponseTime        time.Duration `json:"response_time"`
	ErrorRate           float64   `json:"error_rate"`
	ConnectionCount     int       `json:"connection_count"`
	Notes               string    `json:"notes,omitempty"`
}

// HealthIncident represents a health-related incident
type HealthIncident struct {
	ID                  string        `json:"id"`
	Timestamp           time.Time     `json:"timestamp"`
	Severity            IncidentSeverity `json:"severity"`
	Type                IncidentType  `json:"type"`
	Description         string        `json:"description"`
	Resolution          string        `json:"resolution,omitempty"`
	ResolvedAt          *time.Time    `json:"resolved_at,omitempty"`
	Duration            time.Duration `json:"duration"`
}

type IncidentSeverity string
type IncidentType string

const (
	SeverityLow      IncidentSeverity = "low"
	SeverityMedium   IncidentSeverity = "medium"
	SeverityHigh     IncidentSeverity = "high"
	SeverityCritical IncidentSeverity = "critical"
	
	IncidentTypeConnectionFailure IncidentType = "connection_failure"
	IncidentTypeHighLatency      IncidentType = "high_latency"
	IncidentTypeHighErrorRate    IncidentType = "high_error_rate"
	IncidentTypeResourceExhaustion IncidentType = "resource_exhaustion"
)

// ConnectionHealthChecker checks individual connection health
type ConnectionHealthChecker struct {
	config              *config.ConnectionHealthConfig
	logger              *logger.Logger
}

// ConnectionLoadBalancer balances connections across pools
type ConnectionLoadBalancer struct {
	pools               map[string]*ConnectionPool
	strategy            LoadBalancingStrategy
	logger              *logger.Logger
	
	// Load balancing state
	roundRobinIndex     int32  // atomic
	loadHistory         map[string]*LoadMetrics
	
	mu                  sync.RWMutex
}

// LoadMetrics tracks load balancing metrics
type LoadMetrics struct {
	PoolName            string        `json:"pool_name"`
	RequestCount        int64         `json:"request_count"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ErrorRate           float64       `json:"error_rate"`
	LoadScore           float64       `json:"load_score"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// ConnectionPoolMetrics aggregates metrics across all pools
type ConnectionPoolMetrics struct {
	TotalPools          int                        `json:"total_pools"`
	TotalConnections    int                        `json:"total_connections"`
	HealthyPools        int                        `json:"healthy_pools"`
	UnhealthyPools      int                        `json:"unhealthy_pools"`
	
	OverallHealthScore  float64                    `json:"overall_health_score"`
	AverageResponseTime time.Duration              `json:"average_response_time"`
	TotalThroughput     float64                    `json:"total_throughput"`
	OverallErrorRate    float64                    `json:"overall_error_rate"`
	
	PoolMetrics         map[string]*PoolStatistics `json:"pool_metrics"`
	ResourceUsage       *ResourceUsageMetrics      `json:"resource_usage"`
	
	LastUpdated         time.Time                  `json:"last_updated"`
	
	mu                  sync.RWMutex
}

// ResourceUsageMetrics tracks resource consumption
type ResourceUsageMetrics struct {
	TotalConnections    int     `json:"total_connections"`
	ConnectionUsage     float64 `json:"connection_usage"`     // % of max
	MemoryUsageMB       int64   `json:"memory_usage_mb"`
	MemoryUsagePercent  float64 `json:"memory_usage_percent"`
	BandwidthUsageMBps  int64   `json:"bandwidth_usage_mbps"`
	BandwidthUsagePercent float64 `json:"bandwidth_usage_percent"`
	
	Timestamp           time.Time `json:"timestamp"`
}

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(
	config *config.ConnectionPoolConfig,
	logger *logger.Logger,
) *ConnectionPoolManager {
	
	manager := &ConnectionPoolManager{
		pools:           make(map[string]*ConnectionPool),
		config:          config,
		logger:          logger,
		stopCh:          make(chan struct{}),
		poolMetrics:     NewConnectionPoolMetrics(),
		resourceMonitor: NewConnectionResourceMonitor(config.ResourceMonitor, logger),
		healthChecker:   NewPoolHealthChecker(config.HealthCheck, logger),
		loadBalancer:    NewConnectionLoadBalancer(logger),
	}
	
	return manager
}

// Start initializes and starts the connection pool manager
func (m *ConnectionPoolManager) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.isRunning, 0, 1) {
		return fmt.Errorf("connection pool manager is already running")
	}
	
	m.logger.Info("Starting connection pool manager")
	
	// Start resource monitor
	if err := m.resourceMonitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start resource monitor: %w", err)
	}
	
	// Start health checker
	if err := m.healthChecker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}
	
	// Initialize configured pools
	for _, poolConfig := range m.config.Pools {
		if err := m.CreatePool(poolConfig); err != nil {
			m.logger.Error("Failed to create pool", "pool", poolConfig.Name, "error", err)
		}
	}
	
	// Start metrics collection
	go m.metricsCollectionLoop(ctx)
	
	m.logger.Info("Connection pool manager started")
	return nil
}

// Stop gracefully shuts down the connection pool manager
func (m *ConnectionPoolManager) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.isRunning, 1, 0) {
		return nil
	}
	
	m.logger.Info("Stopping connection pool manager")
	
	close(m.stopCh)
	
	// Stop all pools
	m.mu.RLock()
	pools := make([]*ConnectionPool, 0, len(m.pools))
	for _, pool := range m.pools {
		pools = append(pools, pool)
	}
	m.mu.RUnlock()
	
	for _, pool := range pools {
		if err := pool.Close(); err != nil {
			m.logger.Error("Failed to close pool", "pool", pool.name, "error", err)
		}
	}
	
	// Stop components
	m.healthChecker.Stop(ctx)
	m.resourceMonitor.Stop(ctx)
	
	m.logger.Info("Connection pool manager stopped")
	return nil
}

// CreatePool creates and initializes a new connection pool
func (m *ConnectionPoolManager) CreatePool(config *PoolConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.pools[config.Name]; exists {
		return fmt.Errorf("pool %s already exists", config.Name)
	}
	
	factory := &HTTPConnectionFactory{logger: m.logger}
	
	pool, err := NewConnectionPool(config, factory, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create pool %s: %w", config.Name, err)
	}
	
	if err := pool.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize pool %s: %w", config.Name, err)
	}
	
	m.pools[config.Name] = pool
	m.healthChecker.AddPool(pool)
	m.loadBalancer.AddPool(pool)
	
	m.logger.Info("Connection pool created", "pool", config.Name, "endpoint", config.ServiceEndpoint)
	return nil
}

// GetConnection retrieves a connection from the specified pool
func (m *ConnectionPoolManager) GetConnection(poolName string, timeout time.Duration) (*PooledConnection, error) {
	m.mu.RLock()
	pool, exists := m.pools[poolName]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("pool %s not found", poolName)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return pool.GetConnection(ctx)
}

// ReturnConnection returns a connection to its pool
func (m *ConnectionPoolManager) ReturnConnection(conn *PooledConnection) error {
	if conn.pool == nil {
		return fmt.Errorf("connection has no associated pool")
	}
	
	return conn.pool.ReturnConnection(conn)
}

// metricsCollectionLoop collects metrics from all pools
func (m *ConnectionPoolManager) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

// collectMetrics aggregates metrics from all pools
func (m *ConnectionPoolManager) collectMetrics() {
	m.poolMetrics.mu.Lock()
	defer m.poolMetrics.mu.Unlock()
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var totalConnections int
	var healthyPools int
	var totalResponseTime time.Duration
	var totalThroughput float64
	var totalErrorRate float64
	var responseTimeCount int
	
	for _, pool := range m.pools {
		stats := pool.GetStatistics()
		m.poolMetrics.PoolMetrics[pool.name] = stats
		
		totalConnections += stats.TotalConnections
		if stats.IsHealthy {
			healthyPools++
		}
		
		if stats.AverageResponseTime > 0 {
			totalResponseTime += stats.AverageResponseTime
			responseTimeCount++
		}
		
		totalThroughput += stats.RequestsPerSecond
		totalErrorRate += stats.ErrorRate
	}
	
	m.poolMetrics.TotalPools = len(m.pools)
	m.poolMetrics.TotalConnections = totalConnections
	m.poolMetrics.HealthyPools = healthyPools
	m.poolMetrics.UnhealthyPools = len(m.pools) - healthyPools
	m.poolMetrics.TotalThroughput = totalThroughput
	
	if responseTimeCount > 0 {
		m.poolMetrics.AverageResponseTime = totalResponseTime / time.Duration(responseTimeCount)
	}
	
	if len(m.pools) > 0 {
		m.poolMetrics.OverallErrorRate = totalErrorRate / float64(len(m.pools))
		m.poolMetrics.OverallHealthScore = float64(healthyPools) / float64(len(m.pools))
	}
	
	// Update resource usage
	m.poolMetrics.ResourceUsage = m.resourceMonitor.GetCurrentUsage()
	m.poolMetrics.LastUpdated = time.Now()
}

// GetMetrics returns current connection pool metrics
func (m *ConnectionPoolManager) GetMetrics() *ConnectionPoolMetrics {
	m.poolMetrics.mu.RLock()
	defer m.poolMetrics.mu.RUnlock()
	
	// Return a deep copy
	metrics := &ConnectionPoolMetrics{
		TotalPools:          m.poolMetrics.TotalPools,
		TotalConnections:    m.poolMetrics.TotalConnections,
		HealthyPools:        m.poolMetrics.HealthyPools,
		UnhealthyPools:      m.poolMetrics.UnhealthyPools,
		OverallHealthScore:  m.poolMetrics.OverallHealthScore,
		AverageResponseTime: m.poolMetrics.AverageResponseTime,
		TotalThroughput:     m.poolMetrics.TotalThroughput,
		OverallErrorRate:    m.poolMetrics.OverallErrorRate,
		LastUpdated:         m.poolMetrics.LastUpdated,
		PoolMetrics:         make(map[string]*PoolStatistics),
	}
	
	// Copy pool metrics
	for name, stats := range m.poolMetrics.PoolMetrics {
		statsCopy := *stats
		metrics.PoolMetrics[name] = &statsCopy
	}
	
	// Copy resource usage
	if m.poolMetrics.ResourceUsage != nil {
		resourceUsage := *m.poolMetrics.ResourceUsage
		metrics.ResourceUsage = &resourceUsage
	}
	
	return metrics
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(
	config *PoolConfiguration,
	factory ConnectionFactory,
	logger *logger.Logger,
) (*ConnectionPool, error) {
	
	pool := &ConnectionPool{
		name:              config.Name,
		config:            config,
		logger:            logger,
		connections:       make(chan *PooledConnection, config.MaxConnections),
		activeConnections: make(map[string]*PooledConnection),
		connectionFactory: factory,
		semaphore:         make(chan struct{}, config.MaxConnections),
		cleanupInterval:   5 * time.Minute,
		stats:            NewPoolStatistics(config.Name),
		stopCh:           make(chan struct{}),
		healthChecker:    &ConnectionHealthChecker{logger: logger},
	}
	
	return pool, nil
}

// Initialize initializes the connection pool with initial connections
func (p *ConnectionPool) Initialize() error {
	p.logger.Info("Initializing connection pool", "pool", p.name, "initial_size", p.config.InitialSize)
	
	for i := 0; i < p.config.InitialSize; i++ {
		conn, err := p.createConnection()
		if err != nil {
			p.logger.Error("Failed to create initial connection", "pool", p.name, "error", err)
			continue
		}
		
		select {
		case p.connections <- conn:
			// Connection added to pool
		default:
			// Pool is full, close connection
			conn.close()
		}
	}
	
	// Start maintenance goroutines
	go p.maintenanceLoop()
	
	atomic.StoreInt32(&p.isHealthy, 1)
	
	p.logger.Info("Connection pool initialized", "pool", p.name)
	return nil
}

// GetConnection retrieves a connection from the pool
func (p *ConnectionPool) GetConnection(ctx context.Context) (*PooledConnection, error) {
	// Acquire semaphore slot
	select {
	case p.semaphore <- struct{}{}:
		// Slot acquired
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	// Try to get an existing connection
	select {
	case conn := <-p.connections:
		if p.isConnectionValid(conn) {
			p.activateConnection(conn)
			return conn, nil
		}
		// Connection is invalid, close it and create a new one
		conn.close()
	default:
		// No connections available in pool
	}
	
	// Create a new connection
	conn, err := p.createConnection()
	if err != nil {
		<-p.semaphore // Release semaphore slot
		return nil, err
	}
	
	p.activateConnection(conn)
	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (p *ConnectionPool) ReturnConnection(conn *PooledConnection) error {
	if conn == nil {
		return fmt.Errorf("cannot return nil connection")
	}
	
	p.mu.Lock()
	delete(p.activeConnections, conn.id)
	p.mu.Unlock()
	
	// Update connection state
	conn.mu.Lock()
	conn.state = StateIdle
	conn.lastUsed = time.Now()
	atomic.StoreInt32(&conn.isActive, 0)
	conn.mu.Unlock()
	
	// Check if connection should be kept
	if p.shouldKeepConnection(conn) {
		select {
		case p.connections <- conn:
			// Connection returned to pool
		default:
			// Pool is full, close connection
			conn.close()
		}
	} else {
		conn.close()
	}
	
	// Release semaphore slot
	<-p.semaphore
	
	return nil
}

// createConnection creates a new pooled connection
func (p *ConnectionPool) createConnection() (*PooledConnection, error) {
	conn, err := p.connectionFactory.CreateConnection(p.config)
	if err != nil {
		return nil, err
	}
	
	atomic.AddInt64(&p.stats.TotalCreated, 1)
	return conn, nil
}

// isConnectionValid checks if a connection is still valid
func (p *ConnectionPool) isConnectionValid(conn *PooledConnection) bool {
	if conn == nil {
		return false
	}
	
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	
	// Check if connection has expired
	if time.Now().After(conn.expiresAt) {
		return false
	}
	
	// Check if connection exceeded max usage
	if conn.maxUsage > 0 && conn.usageCount >= conn.maxUsage {
		return false
	}
	
	// Check if connection has been idle too long
	if conn.maxIdleTime > 0 && time.Since(conn.lastUsed) > conn.maxIdleTime {
		return false
	}
	
	// Check connection state
	if conn.state == StateFailed || conn.state == StateExpired {
		return false
	}
	
	return true
}

// activateConnection marks a connection as active
func (p *ConnectionPool) activateConnection(conn *PooledConnection) {
	conn.mu.Lock()
	conn.state = StateActive
	conn.lastUsed = time.Now()
	conn.usageCount++
	atomic.StoreInt32(&conn.isActive, 1)
	conn.mu.Unlock()
	
	p.mu.Lock()
	p.activeConnections[conn.id] = conn
	p.mu.Unlock()
	
	atomic.AddInt64(&p.stats.TotalRequests, 1)
}

// shouldKeepConnection determines if a connection should be kept in the pool
func (p *ConnectionPool) shouldKeepConnection(conn *PooledConnection) bool {
	if !p.isConnectionValid(conn) {
		return false
	}
	
	// Don't keep connection if pool has too many idle connections
	currentSize := len(p.connections)
	if currentSize >= p.config.MaxConnections {
		return false
	}
	
	return true
}

// maintenanceLoop performs periodic maintenance on the connection pool
func (p *ConnectionPool) maintenanceLoop() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.performMaintenance()
		}
	}
}

// performMaintenance performs maintenance tasks
func (p *ConnectionPool) performMaintenance() {
	p.logger.Debug("Performing pool maintenance", "pool", p.name)
	
	// Clean up expired connections
	p.cleanupExpiredConnections()
	
	// Ensure minimum connections
	p.ensureMinimumConnections()
	
	// Update statistics
	p.updateStatistics()
}

// cleanupExpiredConnections removes expired connections from the pool
func (p *ConnectionPool) cleanupExpiredConnections() {
	var validConnections []*PooledConnection
	
	// Drain the channel and check each connection
	for {
		select {
		case conn := <-p.connections:
			if p.isConnectionValid(conn) {
				validConnections = append(validConnections, conn)
			} else {
				conn.close()
				atomic.AddInt64(&p.stats.TotalDestroyed, 1)
			}
		default:
			// No more connections in the channel
			goto done
		}
	}
	
done:
	// Put valid connections back
	for _, conn := range validConnections {
		select {
		case p.connections <- conn:
			// Connection returned
		default:
			// Channel is full, close connection
			conn.close()
			atomic.AddInt64(&p.stats.TotalDestroyed, 1)
		}
	}
}

// ensureMinimumConnections ensures the pool has minimum required connections
func (p *ConnectionPool) ensureMinimumConnections() {
	currentSize := len(p.connections)
	needed := p.config.MinConnections - currentSize
	
	for i := 0; i < needed; i++ {
		conn, err := p.createConnection()
		if err != nil {
			p.logger.Error("Failed to create minimum connection", "pool", p.name, "error", err)
			break
		}
		
		select {
		case p.connections <- conn:
			// Connection added
		default:
			// Pool is full
			conn.close()
			break
		}
	}
}

// updateStatistics updates pool statistics
func (p *ConnectionPool) updateStatistics() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	
	p.stats.TotalConnections = len(p.connections) + len(p.activeConnections)
	p.stats.ActiveConnections = len(p.activeConnections)
	p.stats.IdleConnections = len(p.connections)
	
	// Calculate hit ratio
	if p.stats.TotalRequests > 0 {
		p.stats.HitRatio = float64(p.stats.SuccessfulRequests) / float64(p.stats.TotalRequests)
		p.stats.MissRatio = 1.0 - p.stats.HitRatio
		p.stats.ErrorRate = float64(p.stats.FailedRequests) / float64(p.stats.TotalRequests)
	}
	
	// Calculate utilization
	if p.config.MaxConnections > 0 {
		p.stats.ConnectionUtilization = float64(p.stats.ActiveConnections) / float64(p.config.MaxConnections)
	}
	
	// Update health status
	p.stats.IsHealthy = atomic.LoadInt32(&p.isHealthy) == 1
	p.stats.LastHealthCheck = time.Now()
}

// GetStatistics returns current pool statistics
func (p *ConnectionPool) GetStatistics() *PoolStatistics {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	
	// Return a copy
	stats := *p.stats
	return &stats
}

// Close closes the connection pool
func (p *ConnectionPool) Close() error {
	p.logger.Info("Closing connection pool", "pool", p.name)
	
	close(p.stopCh)
	
	// Close all connections in the pool
	for {
		select {
		case conn := <-p.connections:
			conn.close()
		default:
			goto closeActive
		}
	}
	
closeActive:
	// Close all active connections
	p.mu.RLock()
	activeConns := make([]*PooledConnection, 0, len(p.activeConnections))
	for _, conn := range p.activeConnections {
		activeConns = append(activeConns, conn)
	}
	p.mu.RUnlock()
	
	for _, conn := range activeConns {
		conn.close()
	}
	
	p.logger.Info("Connection pool closed", "pool", p.name)
	return nil
}

// close closes a pooled connection
func (c *PooledConnection) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.state == StateClosing {
		return
	}
	
	c.state = StateClosing
	
	if c.conn != nil {
		c.conn.Close()
	}
	
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// Constructor and helper functions

// CreateConnection creates a new HTTP connection
func (f *HTTPConnectionFactory) CreateConnection(config *PoolConfiguration) (*PooledConnection, error) {
	// Create HTTP client with appropriate configuration
	client := &http.Client{
		Timeout: config.ConnectionTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        config.MaxConnections,
			MaxIdleConnsPerHost: config.MaxConnections,
			IdleConnTimeout:     config.MaxIdleTime,
			DisableKeepAlives:   !config.KeepAliveEnabled,
		},
	}
	
	conn := &PooledConnection{
		id:            fmt.Sprintf("%s_%d", config.Name, time.Now().UnixNano()),
		httpClient:    client,
		createdAt:     time.Now(),
		lastUsed:      time.Now(),
		state:         StateIdle,
		healthScore:   1.0,
		maxUsage:      config.MaxUsageCount,
		maxIdleTime:   config.MaxIdleTime,
		expiresAt:     time.Now().Add(config.MaxConnectionAge),
	}
	
	return conn, nil
}

func (f *HTTPConnectionFactory) ValidateConnection(conn *PooledConnection) error {
	// Simplified validation - in production would perform actual health check
	return nil
}

func (f *HTTPConnectionFactory) CloseConnection(conn *PooledConnection) error {
	conn.close()
	return nil
}

func NewPoolStatistics(poolName string) *PoolStatistics {
	return &PoolStatistics{
		PoolName:  poolName,
		CreatedAt: time.Now(),
	}
}

func NewConnectionPoolMetrics() *ConnectionPoolMetrics {
	return &ConnectionPoolMetrics{
		PoolMetrics: make(map[string]*PoolStatistics),
		LastUpdated: time.Now(),
	}
}

// Stub implementations for supporting components
func NewConnectionResourceMonitor(config *config.ResourceMonitorConfig, logger *logger.Logger) *ConnectionResourceMonitor {
	return &ConnectionResourceMonitor{
		config:             config,
		logger:             logger,
		monitoringInterval: 30 * time.Second,
		alertThresholds: &ResourceThresholds{
			ConnectionWarning:  0.8,
			ConnectionCritical: 0.95,
			MemoryWarning:      0.85,
			MemoryCritical:     0.95,
			BandwidthWarning:   0.8,
			BandwidthCritical:  0.9,
		},
		stopCh: make(chan struct{}),
	}
}

func (r *ConnectionResourceMonitor) Start(ctx context.Context) error {
	r.logger.Info("Connection resource monitor started")
	r.isRunning = true
	return nil
}

func (r *ConnectionResourceMonitor) Stop(ctx context.Context) error {
	r.logger.Info("Connection resource monitor stopped")
	r.isRunning = false
	return nil
}

func (r *ConnectionResourceMonitor) GetCurrentUsage() *ResourceUsageMetrics {
	return &ResourceUsageMetrics{
		TotalConnections:      int(atomic.LoadInt32(&r.totalConnections)),
		MemoryUsageMB:         atomic.LoadInt64(&r.totalMemoryUsage) / (1024 * 1024),
		BandwidthUsageMBps:    atomic.LoadInt64(&r.totalBandwidthUsage) / (1024 * 1024),
		Timestamp:            time.Now(),
	}
}

func NewPoolHealthChecker(config *config.HealthCheckConfig, logger *logger.Logger) *PoolHealthChecker {
	return &PoolHealthChecker{
		pools:         make(map[string]*ConnectionPool),
		config:        config,
		logger:        logger,
		checkInterval: 30 * time.Second,
		healthHistory: make(map[string]*HealthHistory),
		stopCh:        make(chan struct{}),
	}
}

func (h *PoolHealthChecker) Start(ctx context.Context) error {
	h.logger.Info("Pool health checker started")
	h.isRunning = true
	return nil
}

func (h *PoolHealthChecker) Stop(ctx context.Context) error {
	h.logger.Info("Pool health checker stopped")
	h.isRunning = false
	return nil
}

func (h *PoolHealthChecker) AddPool(pool *ConnectionPool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.pools[pool.name] = pool
	h.healthHistory[pool.name] = &HealthHistory{
		PoolName:             pool.name,
		HealthScores:         make([]HealthScore, 0),
		Incidents:            make([]*HealthIncident, 0),
		ConsecutiveSuccesses: 0,
	}
}

func NewConnectionLoadBalancer(logger *logger.Logger) *ConnectionLoadBalancer {
	return &ConnectionLoadBalancer{
		pools:       make(map[string]*ConnectionPool),
		strategy:    LoadBalanceRoundRobin,
		logger:      logger,
		loadHistory: make(map[string]*LoadMetrics),
	}
}

func (l *ConnectionLoadBalancer) AddPool(pool *ConnectionPool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.pools[pool.name] = pool
	l.loadHistory[pool.name] = &LoadMetrics{
		PoolName:    pool.name,
		LastUpdated: time.Now(),
	}
}