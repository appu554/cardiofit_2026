package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/database/models"
)

// Repository handles database operations for the outbox service
type Repository struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger
	config *config.Config
}

// NewRepository creates a new database repository
func NewRepository(config *config.Config, logger *logrus.Logger) (*Repository, error) {
	pool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	repo := &Repository{
		pool:   pool,
		logger: logger,
		config: config,
	}

	// Test the connection
	if err := repo.ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return repo, nil
}

// Close closes the database connection pool
func (r *Repository) Close() {
	r.pool.Close()
}

// ping tests the database connection
func (r *Repository) ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.pool.Ping(ctx)
}

// CreatePartitionedTable creates a partitioned outbox table for a service
func (r *Repository) CreatePartitionedTable(ctx context.Context, serviceName string) error {
	tableName := "outbox_events_" + strings.ReplaceAll(serviceName, "-", "_")
	
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_name VARCHAR(255) NOT NULL,
			event_type VARCHAR(255) NOT NULL,
			event_data JSONB NOT NULL,
			topic VARCHAR(255) NOT NULL,
			correlation_id VARCHAR(255),
			priority INTEGER NOT NULL DEFAULT 5,
			metadata JSONB DEFAULT '{}',
			medical_context VARCHAR(50) NOT NULL DEFAULT 'routine',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			published_at TIMESTAMP WITH TIME ZONE,
			retry_count INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			error_message TEXT,
			next_retry_at TIMESTAMP WITH TIME ZONE
		);
		
		CREATE INDEX IF NOT EXISTS idx_%s_status ON %s (status);
		CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s (created_at);
		CREATE INDEX IF NOT EXISTS idx_%s_priority ON %s (priority DESC);
		CREATE INDEX IF NOT EXISTS idx_%s_medical_context ON %s (medical_context);
		CREATE INDEX IF NOT EXISTS idx_%s_next_retry ON %s (next_retry_at) WHERE next_retry_at IS NOT NULL;
	`, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName)

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create partitioned table for service %s: %w", serviceName, err)
	}

	r.logger.Infof("Created/verified partitioned table for service: %s", serviceName)
	return nil
}

// InsertEvent inserts a new event into the outbox
func (r *Repository) InsertEvent(ctx context.Context, event *models.OutboxEvent) error {
	tableName := "outbox_events_" + strings.ReplaceAll(event.ServiceName, "-", "_")
	
	// Ensure the partitioned table exists
	if err := r.CreatePartitionedTable(ctx, event.ServiceName); err != nil {
		return fmt.Errorf("failed to ensure table exists: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, service_name, event_type, event_data, topic, correlation_id,
			priority, metadata, medical_context, created_at, retry_count, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, tableName)

	_, err := r.pool.Exec(ctx, query,
		event.ID,
		event.ServiceName,
		event.EventType,
		event.EventData,
		event.Topic,
		event.CorrelationID,
		event.Priority,
		event.Metadata,
		event.MedicalContext,
		event.CreatedAt,
		event.RetryCount,
		event.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	r.logger.Debugf("Inserted event %s for service %s", event.ID, event.ServiceName)
	return nil
}

// GetPendingEvents retrieves pending events for publishing
func (r *Repository) GetPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error) {
	var events []*models.OutboxEvent

	// Query all service tables for pending events
	for _, serviceName := range r.config.SupportedServices {
		tableName := "outbox_events_" + strings.ReplaceAll(serviceName, "-", "_")
		
		query := fmt.Sprintf(`
			SELECT id, service_name, event_type, event_data, topic, correlation_id,
				   priority, metadata, medical_context, created_at, published_at,
				   retry_count, status, error_message, next_retry_at
			FROM %s
			WHERE status IN ('pending', 'failed')
			  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
			ORDER BY 
				CASE medical_context
					WHEN 'critical' THEN 1
					WHEN 'urgent' THEN 2
					WHEN 'routine' THEN 3
					ELSE 4
				END,
				priority DESC,
				created_at ASC
			LIMIT $1
		`, tableName)

		rows, err := r.pool.Query(ctx, query, limit)
		if err != nil {
			// Table might not exist yet, skip
			r.logger.Debugf("Skipping table %s: %v", tableName, err)
			continue
		}

		for rows.Next() {
			event := &models.OutboxEvent{}
			err := rows.Scan(
				&event.ID,
				&event.ServiceName,
				&event.EventType,
				&event.EventData,
				&event.Topic,
				&event.CorrelationID,
				&event.Priority,
				&event.Metadata,
				&event.MedicalContext,
				&event.CreatedAt,
				&event.PublishedAt,
				&event.RetryCount,
				&event.Status,
				&event.ErrorMessage,
				&event.NextRetryAt,
			)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan event: %w", err)
			}
			events = append(events, event)
		}
		rows.Close()
	}

	return events, nil
}

// UpdateEventStatus updates the status of an event
func (r *Repository) UpdateEventStatus(ctx context.Context, event *models.OutboxEvent) error {
	tableName := "outbox_events_" + strings.ReplaceAll(event.ServiceName, "-", "_")
	
	query := fmt.Sprintf(`
		UPDATE %s SET
			status = $2,
			published_at = $3,
			retry_count = $4,
			error_message = $5,
			next_retry_at = $6
		WHERE id = $1
	`, tableName)

	result, err := r.pool.Exec(ctx, query,
		event.ID,
		event.Status,
		event.PublishedAt,
		event.RetryCount,
		event.ErrorMessage,
		event.NextRetryAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("event %s not found", event.ID)
	}

	return nil
}

// GetOutboxStats retrieves statistics about the outbox queues
func (r *Repository) GetOutboxStats(ctx context.Context) (*models.OutboxStats, error) {
	stats := &models.OutboxStats{
		QueueDepths:  make(map[string]int64),
		SuccessRates: make(map[string]float64),
	}

	// Get queue depths and success rates for each service
	for _, serviceName := range r.config.SupportedServices {
		tableName := "outbox_events_" + strings.ReplaceAll(serviceName, "-", "_")
		
		// Queue depth
		var queueDepth int64
		queueQuery := fmt.Sprintf(`
			SELECT COUNT(*) FROM %s WHERE status IN ('pending', 'failed')
		`, tableName)
		
		err := r.pool.QueryRow(ctx, queueQuery).Scan(&queueDepth)
		if err != nil {
			// Table might not exist, set to 0
			queueDepth = 0
		}
		stats.QueueDepths[serviceName] = queueDepth

		// Success rate (last 24 hours)
		var totalEvents, successfulEvents int64
		successQuery := fmt.Sprintf(`
			SELECT 
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE status = 'published') as successful
			FROM %s 
			WHERE created_at >= NOW() - INTERVAL '24 hours'
		`, tableName)
		
		err = r.pool.QueryRow(ctx, successQuery).Scan(&totalEvents, &successfulEvents)
		if err != nil {
			stats.SuccessRates[serviceName] = 0.0
		} else if totalEvents > 0 {
			stats.SuccessRates[serviceName] = float64(successfulEvents) / float64(totalEvents)
		} else {
			stats.SuccessRates[serviceName] = 1.0 // No events = 100% success
		}
	}

	// Get total processed in 24h across all services
	var totalProcessed24h int64
	for _, serviceName := range r.config.SupportedServices {
		tableName := "outbox_events_" + strings.ReplaceAll(serviceName, "-", "_")
		totalQuery := fmt.Sprintf(`
			SELECT COUNT(*) FROM %s 
			WHERE status = 'published' AND published_at >= NOW() - INTERVAL '24 hours'
		`, tableName)
		
		var count int64
		err := r.pool.QueryRow(ctx, totalQuery).Scan(&count)
		if err == nil {
			totalProcessed24h += count
		}
	}
	stats.TotalProcessed24h = totalProcessed24h

	// Get dead letter count across all services
	var deadLetterCount int64
	for _, serviceName := range r.config.SupportedServices {
		tableName := "outbox_events_" + strings.ReplaceAll(serviceName, "-", "_")
		dlqQuery := fmt.Sprintf(`
			SELECT COUNT(*) FROM %s WHERE status = 'dead_letter'
		`, tableName)
		
		var count int64
		err := r.pool.QueryRow(ctx, dlqQuery).Scan(&count)
		if err == nil {
			deadLetterCount += count
		}
	}
	stats.DeadLetterCount = deadLetterCount

	return stats, nil
}

// HealthCheck performs a health check on the database
func (r *Repository) HealthCheck(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})
	
	// Test connection
	if err := r.ping(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health
	}

	// Get connection pool stats
	stats := r.pool.Stat()
	health["status"] = "healthy"
	health["total_connections"] = stats.TotalConns()
	health["idle_connections"] = stats.IdleConns()
	health["acquired_connections"] = stats.AcquiredConns()

	return health
}