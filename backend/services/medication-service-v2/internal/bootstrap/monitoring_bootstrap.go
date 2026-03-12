package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"medication-service-v2/internal/infrastructure/monitoring"
)

// MonitoringComponents holds all monitoring-related components
type MonitoringComponents struct {
	Metrics        *monitoring.Metrics
	Logger         *monitoring.Logger
	TracingManager *monitoring.TracingManager
	HealthManager  *monitoring.HealthManager
	AlertManager   *monitoring.AlertManager
}

// MonitoringBootstrap initializes all monitoring components for the medication service
type MonitoringBootstrap struct {
	config     *MonitoringConfig
	components *MonitoringComponents
	zapLogger  *zap.Logger
}

// MonitoringConfig holds all monitoring configuration
type MonitoringConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	
	// Tracing configuration
	Tracing *monitoring.TracingConfig
	
	// Health check configuration
	HealthChecks *monitoring.HealthCheckConfig
	
	// Logging configuration
	Logging *monitoring.LoggingConfig
	
	// Alerting configuration
	Alerting *monitoring.AlertingConfig
	
	// Metrics configuration
	MetricsPort int
	HealthPort  int
}

// NewMonitoringBootstrap creates a new monitoring bootstrap instance
func NewMonitoringBootstrap(config *MonitoringConfig) *MonitoringBootstrap {
	return &MonitoringBootstrap{
		config: config,
	}
}

// Initialize sets up all monitoring components
func (mb *MonitoringBootstrap) Initialize(ctx context.Context) error {
	mb.zapLogger = zap.NewExample() // Temporary logger for bootstrap
	defer mb.zapLogger.Sync()

	mb.zapLogger.Info("Initializing monitoring components",
		zap.String("service", mb.config.ServiceName),
		zap.String("version", mb.config.ServiceVersion),
		zap.String("environment", mb.config.Environment),
	)

	// Initialize components in order
	if err := mb.initializeLogging(); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	if err := mb.initializeMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	if err := mb.initializeTracing(); err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	if err := mb.initializeHealthChecks(); err != nil {
		return fmt.Errorf("failed to initialize health checks: %w", err)
	}

	if err := mb.initializeAlerting(ctx); err != nil {
		return fmt.Errorf("failed to initialize alerting: %w", err)
	}

	mb.components.Logger.Info("All monitoring components initialized successfully",
		zap.String("service", mb.config.ServiceName),
		zap.Int("metrics_port", mb.config.MetricsPort),
		zap.Int("health_port", mb.config.HealthPort),
	)

	return nil
}

// initializeLogging sets up the healthcare-compliant logging system
func (mb *MonitoringBootstrap) initializeLogging() error {
	logger, err := monitoring.NewLogger(mb.config.Logging)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	mb.components = &MonitoringComponents{
		Logger: logger,
	}

	// Log initialization event for audit trail
	auditEvent := &monitoring.AuditEvent{
		EventID:       generateEventID(),
		EventType:     "system_initialization",
		EventCategory: "system_lifecycle",
		Operation:     "logging_system_startup",
		Outcome:       "success",
		Description:   "Healthcare-compliant logging system initialized",
		ComplianceFlags: []string{"hipaa", "audit_trail"},
	}
	
	mb.components.Logger.LogAuditEvent(context.Background(), auditEvent)

	return nil
}

// initializeMetrics sets up Prometheus metrics collection
func (mb *MonitoringBootstrap) initializeMetrics() error {
	metrics := monitoring.NewMetrics()
	mb.components.Metrics = metrics

	mb.components.Logger.Info("Metrics system initialized",
		zap.Int("metrics_count", 48), // Approximate number of metrics defined
		zap.String("registry", "prometheus"),
	)

	// Record initialization metric
	mb.components.Metrics.RecordCounter("service_initializations_total", 1, map[string]string{
		"component": "metrics",
		"status":    "success",
	})

	return nil
}

// initializeTracing sets up OpenTelemetry distributed tracing
func (mb *MonitoringBootstrap) initializeTracing() error {
	tracingManager, err := monitoring.NewTracingManager(mb.config.Tracing, mb.components.Logger)
	if err != nil {
		return fmt.Errorf("failed to create tracing manager: %w", err)
	}

	mb.components.TracingManager = tracingManager

	// Test tracing by creating a startup span
	ctx, span := tracingManager.StartSpan(context.Background(), "service.initialization")
	defer tracingManager.FinishSpan(span, true, map[string]interface{}{
		"component":     "tracing",
		"service":       mb.config.ServiceName,
		"version":       mb.config.ServiceVersion,
		"startup_time":  time.Now().Unix(),
	})

	tracingManager.RecordClinicalEvent(span, "system_startup", "success", map[string]string{
		"initialization_component": "tracing",
		"compliance_level":        "hipaa",
	})

	return nil
}

// initializeHealthChecks sets up the health monitoring system
func (mb *MonitoringBootstrap) initializeHealthChecks() error {
	// Create dependencies for health manager
	deps := &monitoring.Dependencies{
		// Database and Redis would be injected here from main application
		Logger:         mb.components.Logger,
		Metrics:        mb.components.Metrics,
		TracingManager: mb.components.TracingManager,
	}

	healthManager := monitoring.NewHealthManager(mb.config.HealthChecks, deps, mb.config.ServiceVersion)
	mb.components.HealthManager = healthManager

	mb.components.Logger.Info("Health monitoring system initialized",
		zap.Duration("check_interval", mb.config.HealthChecks.HealthCheckInterval),
		zap.Duration("database_timeout", mb.config.HealthChecks.DatabaseTimeout),
		zap.Duration("redis_timeout", mb.config.HealthChecks.RedisTimeout),
	)

	return nil
}

// initializeAlerting sets up the alerting system
func (mb *MonitoringBootstrap) initializeAlerting(ctx context.Context) error {
	alertManager := monitoring.NewAlertManager(mb.config.Alerting, mb.components.Metrics, mb.components.Logger)
	mb.components.AlertManager = alertManager

	// Start alert manager
	if err := alertManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	mb.components.Logger.Info("Alerting system initialized",
		zap.Int("alert_rules", len(mb.config.Alerting.Rules)),
		zap.Duration("evaluation_interval", mb.config.Alerting.EvaluationInterval),
	)

	// Fire a test alert to verify system is working
	testAlert := &monitoring.Alert{
		Name:        "Monitoring System Startup",
		Category:    monitoring.CategorySystemHealth,
		Severity:    monitoring.SeverityInfo,
		Description: "Monitoring system has been successfully initialized",
		Metadata: map[string]interface{}{
			"startup_time": time.Now().Format(time.RFC3339),
			"service":      mb.config.ServiceName,
			"version":      mb.config.ServiceVersion,
		},
	}

	alertManager.FireManualAlert(ctx, testAlert)

	return nil
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func (mb *MonitoringBootstrap) StartMetricsServer() error {
	metricsRouter := gin.New()
	metricsRouter.Use(gin.Recovery())
	
	// Add healthcare-specific middleware
	metricsRouter.Use(mb.addHealthcareHeaders())
	metricsRouter.Use(mb.addCorrelationID())

	// Prometheus metrics endpoint
	metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Custom health metrics endpoint
	metricsRouter.GET("/health/metrics", mb.handleHealthMetrics)

	// Service information endpoint
	metricsRouter.GET("/info", mb.handleServiceInfo)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", mb.config.MetricsPort),
		Handler:      metricsRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	mb.components.Logger.Info("Starting metrics server",
		zap.Int("port", mb.config.MetricsPort),
		zap.String("endpoint", "/metrics"),
	)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mb.components.Logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	return nil
}

// StartHealthServer starts the health check HTTP server
func (mb *MonitoringBootstrap) StartHealthServer() error {
	healthRouter := gin.New()
	healthRouter.Use(gin.Recovery())
	
	// Add healthcare-specific middleware
	healthRouter.Use(mb.addHealthcareHeaders())
	healthRouter.Use(mb.addCorrelationID())

	// Register health check routes
	mb.components.HealthManager.RegisterRoutes(healthRouter)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", mb.config.HealthPort),
		Handler:      healthRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	mb.components.Logger.Info("Starting health check server",
		zap.Int("port", mb.config.HealthPort),
		zap.String("liveness", "/health/live"),
		zap.String("readiness", "/health/ready"),
		zap.String("detailed", "/health/status"),
	)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mb.components.Logger.Error("Health server failed", zap.Error(err))
		}
	}()

	return nil
}

// GetComponents returns the initialized monitoring components
func (mb *MonitoringBootstrap) GetComponents() *MonitoringComponents {
	return mb.components
}

// Shutdown gracefully shuts down all monitoring components
func (mb *MonitoringBootstrap) Shutdown(ctx context.Context) error {
	mb.components.Logger.Info("Shutting down monitoring components")

	// Shutdown alert manager
	if mb.components.AlertManager != nil {
		mb.components.AlertManager.Stop()
	}

	// Shutdown tracing
	if mb.components.TracingManager != nil {
		if err := mb.components.TracingManager.Shutdown(ctx); err != nil {
			mb.components.Logger.Error("Failed to shutdown tracing", zap.Error(err))
		}
	}

	// Log shutdown event for audit trail
	auditEvent := &monitoring.AuditEvent{
		EventID:       generateEventID(),
		EventType:     "system_shutdown",
		EventCategory: "system_lifecycle",
		Operation:     "monitoring_system_shutdown",
		Outcome:       "success",
		Description:   "Healthcare monitoring system shut down gracefully",
		ComplianceFlags: []string{"hipaa", "audit_trail"},
	}
	
	mb.components.Logger.LogAuditEvent(ctx, auditEvent)

	// Close logger last
	if mb.components.Logger != nil {
		return mb.components.Logger.Close()
	}

	return nil
}

// Middleware functions

// addHealthcareHeaders adds healthcare-specific HTTP headers
func (mb *MonitoringBootstrap) addHealthcareHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Healthcare-Service", mb.config.ServiceName)
		c.Header("X-Healthcare-Version", mb.config.ServiceVersion)
		c.Header("X-Compliance-Level", "HIPAA")
		c.Header("X-Data-Classification", "PHI")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// addCorrelationID adds correlation ID to requests for tracing
func (mb *MonitoringBootstrap) addCorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = monitoring.GetCorrelationID(c.Request.Context())
		}
		
		c.Header("X-Correlation-ID", correlationID)
		c.Set("correlation_id", correlationID)
		c.Next()
	}
}

// Handler functions

// handleHealthMetrics provides health-related metrics
func (mb *MonitoringBootstrap) handleHealthMetrics(c *gin.Context) {
	metrics := mb.components.HealthManager.GetActiveAlerts()
	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now(),
		"service":   mb.config.ServiceName,
		"version":   mb.config.ServiceVersion,
		"active_alerts_count": len(metrics),
	})
}

// handleServiceInfo provides service information
func (mb *MonitoringBootstrap) handleServiceInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     mb.config.ServiceName,
		"version":     mb.config.ServiceVersion,
		"environment": mb.config.Environment,
		"compliance":  "HIPAA",
		"monitoring": gin.H{
			"metrics_enabled":    true,
			"tracing_enabled":    true,
			"health_enabled":     true,
			"alerting_enabled":   true,
			"logging_enabled":    true,
		},
		"endpoints": gin.H{
			"metrics": fmt.Sprintf(":%d/metrics", mb.config.MetricsPort),
			"health":  fmt.Sprintf(":%d/health/status", mb.config.HealthPort),
		},
		"timestamp": time.Now(),
	})
}

// Utility functions

// generateEventID generates a unique event ID for audit events
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), mb.config.ServiceName)
}

// LoadDefaultConfig creates a default monitoring configuration
func LoadDefaultConfig() *MonitoringConfig {
	return &MonitoringConfig{
		ServiceName:    "medication-service-v2",
		ServiceVersion: "2.0.0",
		Environment:    "production",
		MetricsPort:    9090,
		HealthPort:     8080,
		
		Tracing: &monitoring.TracingConfig{
			ServiceName:     "medication-service-v2",
			ServiceVersion:  "2.0.0",
			JaegerEndpoint:  "http://jaeger-collector:14268/api/traces",
			SamplingRate:    0.1,
			Environment:     "production",
			InstanceID:      "medication-service-v2-instance",
		},
		
		HealthChecks: &monitoring.HealthCheckConfig{
			DatabaseTimeout:        5 * time.Second,
			RedisTimeout:          2 * time.Second,
			ExternalServiceTimeout: 10 * time.Second,
			HealthCheckInterval:    30 * time.Second,
			DegradedThreshold:     100 * time.Millisecond,
			UnhealthyThreshold:    250 * time.Millisecond,
			CriticalThreshold:     500 * time.Millisecond,
		},
		
		Logging: &monitoring.LoggingConfig{
			Level:             monitoring.InfoLevel,
			Environment:       "production",
			ServiceName:       "medication-service-v2",
			ServiceVersion:    "2.0.0",
			EnableConsole:     false,
			EnableFile:        true,
			EnableAuditFile:   true,
			LogDirectory:      "/var/log/medication-service",
			MaxFileSize:       100,
			MaxBackups:        30,
			MaxAge:            90,
			CompressBackups:   true,
			EnableHIPAA:       true,
			EnableAuditTrail:  true,
			RetentionDays:     2555, // 7 years
			EnableSanitization: true,
			SanitizeFields: []string{
				"patient_id", "ssn", "medical_record_number",
				"auth_token", "session_id", "api_key",
			},
		},
		
		Alerting: &monitoring.AlertingConfig{
			EvaluationInterval: 15 * time.Second,
			DefaultSeverity:    monitoring.SeverityMedium,
			MaxAlerts:          1000,
			RetentionDays:      30,
			NotificationChannels: map[string]monitoring.NotificationChannelConfig{
				"email": {
					Type:    "email",
					Enabled: true,
					Settings: map[string]interface{}{
						"smtp_server": "smtp.hospital.com:587",
						"from":        "alerts@hospital.com",
					},
				},
				"slack": {
					Type:    "slack",
					Enabled: true,
					Settings: map[string]interface{}{
						"webhook_url": "${SLACK_WEBHOOK_URL}",
					},
				},
			},
		},
	}
}