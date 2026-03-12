package kafka

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ExampleAlertRouter is a sample implementation of AlertRouter interface
type ExampleAlertRouter struct {
	logger *zap.Logger
}

func NewExampleAlertRouter(logger *zap.Logger) *ExampleAlertRouter {
	return &ExampleAlertRouter{
		logger: logger,
	}
}

// RouteAlert implements the AlertRouter interface
func (r *ExampleAlertRouter) RouteAlert(ctx context.Context, alert *Alert) error {
	r.logger.Info("Routing alert",
		zap.String("alert_id", alert.AlertID),
		zap.String("patient_id", alert.PatientID),
		zap.String("alert_type", string(alert.AlertType)),
		zap.String("severity", string(alert.Severity)),
		zap.Float64("confidence", alert.Confidence))

	// Example routing logic
	switch alert.Severity {
	case SeverityCritical:
		return r.routeCriticalAlert(ctx, alert)
	case SeverityHigh:
		return r.routeHighAlert(ctx, alert)
	case SeverityModerate:
		return r.routeModerateAlert(ctx, alert)
	case SeverityLow:
		return r.routeLowAlert(ctx, alert)
	default:
		return fmt.Errorf("unknown severity: %s", alert.Severity)
	}
}

func (r *ExampleAlertRouter) routeCriticalAlert(ctx context.Context, alert *Alert) error {
	r.logger.Warn("CRITICAL alert - sending to pager + SMS + voice",
		zap.String("alert_id", alert.AlertID),
		zap.String("patient_id", alert.PatientID))

	// Send to multiple channels
	// - Pager notification
	// - SMS notification
	// - Voice call
	// - Push notification

	return nil
}

func (r *ExampleAlertRouter) routeHighAlert(ctx context.Context, alert *Alert) error {
	r.logger.Info("HIGH alert - sending to SMS + push",
		zap.String("alert_id", alert.AlertID),
		zap.String("patient_id", alert.PatientID))

	// Send to:
	// - SMS notification
	// - Push notification

	return nil
}

func (r *ExampleAlertRouter) routeModerateAlert(ctx context.Context, alert *Alert) error {
	r.logger.Info("MODERATE alert - sending to push + in-app",
		zap.String("alert_id", alert.AlertID),
		zap.String("patient_id", alert.PatientID))

	// Send to:
	// - Push notification
	// - In-app notification

	return nil
}

func (r *ExampleAlertRouter) routeLowAlert(ctx context.Context, alert *Alert) error {
	r.logger.Debug("LOW alert - sending to in-app only",
		zap.String("alert_id", alert.AlertID),
		zap.String("patient_id", alert.PatientID))

	// Send to:
	// - In-app notification only

	return nil
}

// ExampleConsumerIntegration demonstrates how to integrate the Kafka consumer
func ExampleConsumerIntegration() {
	// Create logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	defer logger.Sync()

	// Create consumer configuration from environment variables
	config := &ConsumerConfig{
		Brokers: []string{
			getEnv("KAFKA_BROKERS", "localhost:9092"),
		},
		GroupID: getEnv("KAFKA_GROUP_ID", "notification-service-consumers"),
		Topics: []string{
			"ml-risk-alerts.v1",
			"clinical-patterns.v1",
			"alert-management.v1",
		},
		Username:       getEnv("KAFKA_USERNAME", ""),
		Password:       getEnv("KAFKA_PASSWORD", ""),
		AutoCommit:     true,
		WorkerPoolSize: 10,
	}

	// Create alert router
	router := NewExampleAlertRouter(logger)

	// Create consumer
	consumer, err := NewAlertConsumer(config, router, logger)
	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}

	// Start consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		logger.Fatal("Failed to start consumer", zap.Error(err))
	}

	logger.Info("Kafka consumer started successfully",
		zap.Strings("topics", config.Topics),
		zap.String("group_id", config.GroupID))

	// Setup metrics reporting
	go reportMetrics(consumer, logger)

	// Setup health check
	go healthCheckLoop(consumer, logger)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received, stopping consumer...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := consumer.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
	} else {
		logger.Info("Consumer stopped successfully")
	}
}

// reportMetrics periodically logs consumer metrics
func reportMetrics(consumer *AlertConsumer, logger *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := consumer.GetMetrics()

		logger.Info("Consumer metrics",
			zap.Int64("messages_consumed", metrics.MessagesConsumed),
			zap.Int64("messages_processed", metrics.MessagesProcessed),
			zap.Int64("messages_failed", metrics.MessagesFailed),
			zap.Int64("consumer_lag", metrics.ConsumerLag),
			zap.Time("last_message", metrics.LastMessageTimestamp))

		// Log per-topic counts
		for topic, count := range metrics.TopicMessageCounts {
			logger.Debug("Topic message count",
				zap.String("topic", topic),
				zap.Int64("count", count))
		}

		// Calculate and log average processing time
		if len(metrics.ProcessingDurationMs) > 0 {
			var sum int64
			for _, duration := range metrics.ProcessingDurationMs {
				sum += duration
			}
			avg := sum / int64(len(metrics.ProcessingDurationMs))
			logger.Info("Processing performance",
				zap.Int64("avg_duration_ms", avg),
				zap.Int("sample_size", len(metrics.ProcessingDurationMs)))
		}

		// Log errors by type
		if len(metrics.ProcessingErrors) > 0 {
			for errorType, count := range metrics.ProcessingErrors {
				logger.Warn("Processing errors",
					zap.String("error_type", errorType),
					zap.Int64("count", count))
			}
		}
	}
}

// healthCheckLoop periodically checks consumer health
func healthCheckLoop(consumer *AlertConsumer, logger *zap.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := consumer.HealthCheck(); err != nil {
			logger.Error("Health check failed", zap.Error(err))
		} else {
			logger.Debug("Health check passed")
		}
	}
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Example usage in main.go:
//
// func main() {
//     kafka.ExampleConsumerIntegration()
// }
