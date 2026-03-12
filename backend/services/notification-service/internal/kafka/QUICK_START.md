# Kafka Consumer Quick Start Guide

## 5-Minute Integration

### 1. Install Dependencies

```bash
# Install librdkafka
brew install librdkafka  # macOS
apt-get install librdkafka-dev  # Ubuntu/Debian

# Install Go dependencies
cd backend/services/notification-service
go mod init github.com/cardiofit/notification-service
go get github.com/confluentinc/confluent-kafka-go/v2/kafka
go get go.uber.org/zap
```

### 2. Set Environment Variables

```bash
export KAFKA_BROKERS="pkc-xxxxx.us-east-1.aws.confluent.cloud:9092"
export KAFKA_GROUP_ID="notification-service-consumers"
export KAFKA_USERNAME="your-api-key"
export KAFKA_PASSWORD="your-api-secret"
```

### 3. Create Main Application

Create `cmd/main.go`:

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    "github.com/cardiofit/notification-service/internal/kafka"
    "go.uber.org/zap"
)

func main() {
    // Create logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // Configure consumer
    config := &kafka.ConsumerConfig{
        Brokers: []string{os.Getenv("KAFKA_BROKERS")},
        GroupID: os.Getenv("KAFKA_GROUP_ID"),
        Topics: []string{
            "ml-risk-alerts.v1",
            "clinical-patterns.v1",
            "alert-management.v1",
        },
        Username: os.Getenv("KAFKA_USERNAME"),
        Password: os.Getenv("KAFKA_PASSWORD"),
        AutoCommit: true,
        WorkerPoolSize: 10,
    }

    // Create simple router
    router := kafka.NewExampleAlertRouter(logger)

    // Create consumer
    consumer, err := kafka.NewAlertConsumer(config, router, logger)
    if err != nil {
        logger.Fatal("Failed to create consumer", zap.Error(err))
    }

    // Start consumer
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := consumer.Start(ctx); err != nil {
        logger.Fatal("Failed to start consumer", zap.Error(err))
    }

    logger.Info("Consumer started successfully")

    // Wait for shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    logger.Info("Shutting down...")
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()
    consumer.Stop(shutdownCtx)
}
```

### 4. Run

```bash
go run cmd/main.go
```

## Custom Router Implementation

Replace `ExampleAlertRouter` with your own:

```go
type MyAlertRouter struct {
    twilioClient  *twilio.RestClient
    sendGridClient *sendgrid.Client
    logger        *zap.Logger
}

func (r *MyAlertRouter) RouteAlert(ctx context.Context, alert *kafka.Alert) error {
    switch alert.Severity {
    case kafka.SeverityCritical:
        // Send SMS + Pager + Voice
        r.sendSMS(alert)
        r.sendPager(alert)
        r.initiateVoiceCall(alert)
    case kafka.SeverityHigh:
        // Send SMS + Push
        r.sendSMS(alert)
        r.sendPush(alert)
    case kafka.SeverityModerate:
        // Send Push only
        r.sendPush(alert)
    }
    return nil
}
```

## Monitoring

Add metrics endpoint to your HTTP server:

```go
http.HandleFunc("/metrics/kafka", func(w http.ResponseWriter, r *http.Request) {
    metrics := consumer.GetMetrics()
    json.NewEncoder(w).Encode(metrics)
})

http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if err := consumer.HealthCheck(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
})
```

## Testing

### Unit Tests

```bash
go test -v ./internal/kafka
```

### Integration Test

Use the example integration:

```bash
go run internal/kafka/example_integration.go
```

## Common Issues

### Issue: Consumer not receiving messages

**Solution**: Check topic names and group ID:

```bash
# List topics
kafka-topics --list --bootstrap-server $KAFKA_BROKERS

# Check consumer group
kafka-consumer-groups --describe --group notification-service-consumers --bootstrap-server $KAFKA_BROKERS
```

### Issue: High lag

**Solution**: Increase worker pool size:

```go
config.WorkerPoolSize = 20  // Increase from 10
```

### Issue: Processing failures

**Solution**: Check router errors:

```go
logger.Error("Router error", zap.Error(err), zap.String("alert_id", alert.AlertID))
```

## Next Steps

1. Implement production AlertRouter with real notification channels
2. Add Prometheus metrics export
3. Set up alerting on consumer lag
4. Deploy to Kubernetes with horizontal pod autoscaling

## Documentation

- Full README: [README.md](./README.md)
- Implementation Guide: [NOTIFICATION_SERVICE_KAFKA_CONSUMER_IMPLEMENTATION.md](/Users/apoorvabk/Downloads/cardiofit/claudedocs/NOTIFICATION_SERVICE_KAFKA_CONSUMER_IMPLEMENTATION.md)
- Service Specification: [NOTIFICATION_SERVICE_SPECIFICATION.md](/Users/apoorvabk/Downloads/cardiofit/claudedocs/NOTIFICATION_SERVICE_SPECIFICATION.md)
