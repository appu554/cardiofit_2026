// Package outboxsdk provides a standardized SDK for microservices to implement
// the transactional outbox pattern with the Global Outbox Service
package outboxsdk

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "global-outbox-service-go/pkg/proto"
)

// OutboxClient provides a simple interface for services to publish events transactionally
type OutboxClient struct {
	serviceName      string
	pool            *pgxpool.Pool
	grpcClient      pb.OutboxServiceClient
	grpcConn        *grpc.ClientConn
	logger          *logrus.Logger
	config          *ClientConfig
	registrationID  string
}

// ClientConfig configures the outbox client
type ClientConfig struct {
	ServiceName           string        `json:"service_name"`
	DatabaseURL           string        `json:"database_url"`
	OutboxServiceGRPCURL  string        `json:"outbox_service_grpc_url"`
	DefaultTopic          string        `json:"default_topic"`
	DefaultPriority       int32         `json:"default_priority"`
	DefaultMedicalContext string        `json:"default_medical_context"`
	EnableTracing         bool          `json:"enable_tracing"`
	Timeout               time.Duration `json:"timeout"`
	RetryAttempts         int           `json:"retry_attempts"`
	CircuitBreakerEnabled bool          `json:"circuit_breaker_enabled"`
}

// EventOptions provides options for publishing events
type EventOptions struct {
	Topic           string            `json:"topic,omitempty"`
	Priority        int32             `json:"priority,omitempty"`
	MedicalContext  string            `json:"medical_context,omitempty"`
	CorrelationID   string            `json:"correlation_id,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// TransactionFunc represents a function that performs business logic within a transaction
type TransactionFunc func(ctx context.Context, tx pgx.Tx) error

// NewOutboxClient creates a new outbox client
func NewOutboxClient(config *ClientConfig, logger *logrus.Logger) (*OutboxClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Apply defaults
	applyDefaults(config)

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize database connection
	pool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test database connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize gRPC connection
	conn, err := grpc.Dial(config.OutboxServiceGRPCURL, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to outbox service: %w", err)
	}

	grpcClient := pb.NewOutboxServiceClient(conn)

	client := &OutboxClient{
		serviceName: config.ServiceName,
		pool:        pool,
		grpcClient:  grpcClient,
		grpcConn:    conn,
		logger:      logger,
		config:      config,
	}

	// Initialize the service (create table, register, etc.)
	if err := client.initialize(context.Background()); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	logger.Infof("Outbox client initialized for service: %s", config.ServiceName)
	return client, nil
}

// Close closes the outbox client and cleans up resources
func (c *OutboxClient) Close() error {
	if c.grpcConn != nil {
		c.grpcConn.Close()
	}
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// SaveAndPublish performs business logic and publishes an event in a single transaction
// This is the main method that services should use
func (c *OutboxClient) SaveAndPublish(ctx context.Context, eventType string, eventData interface{}, options *EventOptions, businessLogic TransactionFunc) error {
	// Start a database transaction
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute business logic first
	if businessLogic != nil {
		if err := businessLogic(ctx, tx); err != nil {
			return fmt.Errorf("business logic failed: %w", err)
		}
	}

	// Serialize event data
	eventDataJSON, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Apply defaults for options
	if options == nil {
		options = &EventOptions{}
	}
	c.applyEventDefaults(eventType, options)

	// Create outbox event record
	event := &OutboxEvent{
		ID:             uuid.New(),
		ServiceName:    c.serviceName,
		EventType:      eventType,
		EventData:      string(eventDataJSON),
		Topic:          options.Topic,
		CorrelationID:  options.CorrelationID,
		Priority:       options.Priority,
		MedicalContext: options.MedicalContext,
		Metadata:       options.Metadata,
		CreatedAt:      time.Now().UTC(),
		Status:         "pending",
	}

	// Insert into outbox table
	if err := c.insertOutboxEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	c.logger.Debugf("Successfully saved and queued event %s of type %s", event.ID, eventType)
	return nil
}

// PublishEvent publishes an event immediately via gRPC (for non-transactional scenarios)
func (c *OutboxClient) PublishEvent(ctx context.Context, eventType string, eventData interface{}, options *EventOptions) error {
	// Serialize event data
	eventDataJSON, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Apply defaults
	if options == nil {
		options = &EventOptions{}
	}
	c.applyEventDefaults(eventType, options)

	// Create gRPC request
	req := &pb.PublishEventRequest{
		ServiceName:    c.serviceName,
		EventType:      eventType,
		EventData:      string(eventDataJSON),
		Topic:          options.Topic,
		CorrelationId:  options.CorrelationID,
		Priority:       options.Priority,
		MedicalContext: options.MedicalContext,
		Metadata:       options.Metadata,
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Publish via gRPC
	resp, err := c.grpcClient.PublishEvent(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish event via gRPC: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("event publishing failed: %s", resp.Message)
	}

	c.logger.Debugf("Successfully published event %s of type %s", resp.EventId, eventType)
	return nil
}

// SaveAndPublishBatch publishes multiple events in a single transaction
func (c *OutboxClient) SaveAndPublishBatch(ctx context.Context, events []EventRequest, businessLogic TransactionFunc) error {
	// Start a database transaction
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute business logic first
	if businessLogic != nil {
		if err := businessLogic(ctx, tx); err != nil {
			return fmt.Errorf("business logic failed: %w", err)
		}
	}

	// Insert all events
	for i, eventReq := range events {
		eventDataJSON, err := json.Marshal(eventReq.EventData)
		if err != nil {
			return fmt.Errorf("failed to marshal event data for event %d: %w", i, err)
		}

		// Apply defaults
		if eventReq.Options == nil {
			eventReq.Options = &EventOptions{}
		}
		c.applyEventDefaults(eventReq.EventType, eventReq.Options)

		event := &OutboxEvent{
			ID:             uuid.New(),
			ServiceName:    c.serviceName,
			EventType:      eventReq.EventType,
			EventData:      string(eventDataJSON),
			Topic:          eventReq.Options.Topic,
			CorrelationID:  eventReq.Options.CorrelationID,
			Priority:       eventReq.Options.Priority,
			MedicalContext: eventReq.Options.MedicalContext,
			Metadata:       eventReq.Options.Metadata,
			CreatedAt:      time.Now().UTC(),
			Status:         "pending",
		}

		if err := c.insertOutboxEvent(ctx, tx, event); err != nil {
			return fmt.Errorf("failed to insert outbox event %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	c.logger.Debugf("Successfully saved and queued %d events", len(events))
	return nil
}

// HealthCheck checks if the outbox service is available
func (c *OutboxClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{
		Service: c.serviceName,
	}

	resp, err := c.grpcClient.HealthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Status != "SERVING" {
		return fmt.Errorf("outbox service is not healthy: %s - %s", resp.Status, resp.Message)
	}

	return nil
}

// GetStats returns outbox statistics for this service
func (c *OutboxClient) GetStats(ctx context.Context) (*pb.GetOutboxStatsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &pb.GetOutboxStatsRequest{
		ServiceName: c.serviceName,
	}

	return c.grpcClient.GetOutboxStats(ctx, req)
}

// Helper types and methods

// EventRequest represents a single event to be published
type EventRequest struct {
	EventType string
	EventData interface{}
	Options   *EventOptions
}

// OutboxEvent represents an event in the outbox table
type OutboxEvent struct {
	ID             uuid.UUID         `json:"id"`
	ServiceName    string            `json:"service_name"`
	EventType      string            `json:"event_type"`
	EventData      string            `json:"event_data"`
	Topic          string            `json:"topic"`
	CorrelationID  string            `json:"correlation_id,omitempty"`
	Priority       int32             `json:"priority"`
	MedicalContext string            `json:"medical_context"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	PublishedAt    *time.Time        `json:"published_at,omitempty"`
	RetryCount     int32             `json:"retry_count"`
	Status         string            `json:"status"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	NextRetryAt    *time.Time        `json:"next_retry_at,omitempty"`
}

// initialize sets up the client (creates table, registers with service, etc.)
func (c *OutboxClient) initialize(ctx context.Context) error {
	// Create outbox table if it doesn't exist
	if err := c.createOutboxTable(ctx); err != nil {
		return fmt.Errorf("failed to create outbox table: %w", err)
	}

	// Register with the global outbox service
	// This is optional - the service can auto-discover tables
	c.logger.Debugf("Outbox client initialized for service %s", c.serviceName)
	
	return nil
}

// createOutboxTable creates the outbox table for this service
func (c *OutboxClient) createOutboxTable(ctx context.Context) error {
	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(c.serviceName, "-", "_"))
	
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

	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	c.logger.Debugf("Created/verified outbox table: %s", tableName)
	return nil
}

// insertOutboxEvent inserts an event into the outbox table within a transaction
func (c *OutboxClient) insertOutboxEvent(ctx context.Context, tx pgx.Tx, event *OutboxEvent) error {
	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(c.serviceName, "-", "_"))
	
	metadataJSON := "{}"
	if event.Metadata != nil {
		if data, err := json.Marshal(event.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, service_name, event_type, event_data, topic, correlation_id,
			priority, metadata, medical_context, created_at, retry_count, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, tableName)

	_, err := tx.Exec(ctx, query,
		event.ID,
		event.ServiceName,
		event.EventType,
		event.EventData,
		event.Topic,
		nullString(event.CorrelationID),
		event.Priority,
		metadataJSON,
		event.MedicalContext,
		event.CreatedAt,
		event.RetryCount,
		event.Status,
	)

	return err
}

// applyEventDefaults applies default values to event options
func (c *OutboxClient) applyEventDefaults(eventType string, options *EventOptions) {
	if options.Topic == "" {
		if c.config.DefaultTopic != "" {
			options.Topic = c.config.DefaultTopic
		} else {
			// Generate default topic from service name and event type
			servicePart := strings.ReplaceAll(c.serviceName, "-", "_")
			eventPart := strings.ReplaceAll(eventType, ".", "_")
			options.Topic = fmt.Sprintf("clinical.%s.%s", servicePart, eventPart)
		}
	}

	if options.Priority == 0 {
		options.Priority = c.config.DefaultPriority
	}

	if options.MedicalContext == "" {
		options.MedicalContext = c.config.DefaultMedicalContext
	}

	if options.Metadata == nil {
		options.Metadata = make(map[string]string)
	}

	// Add SDK metadata
	options.Metadata["sdk_version"] = "1.0.0"
	options.Metadata["service_name"] = c.serviceName
}

// Utility functions

func applyDefaults(config *ClientConfig) {
	if config.DefaultPriority == 0 {
		config.DefaultPriority = 5
	}
	if config.DefaultMedicalContext == "" {
		config.DefaultMedicalContext = "routine"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.OutboxServiceGRPCURL == "" {
		config.OutboxServiceGRPCURL = "localhost:50052"
	}
}

func validateConfig(config *ClientConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if config.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}
	if config.DefaultPriority < 1 || config.DefaultPriority > 10 {
		return fmt.Errorf("default_priority must be between 1 and 10")
	}
	return nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}