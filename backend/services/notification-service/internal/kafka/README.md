# Kafka Alert Consumer

High-performance Kafka consumer for clinical alerts with parallel processing, automatic retry, and comprehensive metrics tracking.

## Overview

The Alert Consumer ingests clinical alerts from 3 Kafka topics and routes them to appropriate notification channels through the AlertRouter interface.

## Features

- **Multi-Topic Consumption**: Subscribes to 3 alert topics simultaneously
- **Worker Pool**: Parallel message processing with configurable concurrency (default: 10 workers)
- **Automatic Offset Management**: Configurable auto-commit with manual commit fallback
- **Error Handling**: Comprehensive error tracking and retry logic
- **Metrics Tracking**: Real-time monitoring of consumption, processing, and lag
- **Graceful Shutdown**: Waits for in-flight messages before stopping
- **Health Checks**: Built-in health monitoring and diagnostics

## Topics

| Topic | Source | Alert Types |
|-------|--------|-------------|
| `ml-risk-alerts.v1` | Module 5 ML Inference | Sepsis, Mortality Risk, Deterioration, Readmission |
| `clinical-patterns.v1` | Module 4 CEP | Vital Sign Anomalies, Threshold Violations, Pattern Detection |
| `alert-management.v1` | Module 4 Alert Management | Critical Routing, Manual Triggers, Escalations |

## Alert Schema

```json
{
  "alert_id": "string",
  "patient_id": "string",
  "hospital_id": "string",
  "department_id": "string",
  "alert_type": "SEPSIS_ALERT | MORTALITY_RISK | ...",
  "severity": "CRITICAL | HIGH | MODERATE | LOW",
  "confidence": 0.95,
  "message": "string",
  "recommendations": ["string"],
  "patient_location": {
    "room": "ICU-5",
    "bed": "A"
  },
  "vital_signs": {
    "heart_rate": 125,
    "blood_pressure_systolic": 85,
    "temperature": 39.2
  },
  "timestamp": 1699564800000,
  "metadata": {
    "source_module": "MODULE5_ML_INFERENCE",
    "model_version": "1.2.3",
    "requires_escalation": true
  }
}
```

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/cardiofit/notification-service/internal/kafka"
    "go.uber.org/zap"
)

// Create configuration
config := &kafka.ConsumerConfig{
    Brokers: []string{"pkc-xxxxx.us-east-1.aws.confluent.cloud:9092"},
    GroupID: "notification-service-consumers",
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

// Create router (implements AlertRouter interface)
router := NewAlertRouter(...)

// Create logger
logger, _ := zap.NewProduction()

// Create consumer
consumer, err := kafka.NewAlertConsumer(config, router, logger)
if err != nil {
    log.Fatal(err)
}

// Start consuming
ctx := context.Background()
if err := consumer.Start(ctx); err != nil {
    log.Fatal(err)
}

// Graceful shutdown
defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    consumer.Stop(shutdownCtx)
}()
```

### Implementing AlertRouter

```go
type AlertRouter interface {
    RouteAlert(ctx context.Context, alert *Alert) error
}

type MyAlertRouter struct {
    // Your routing dependencies
    fatigueTracker   *AlertFatigueTracker
    userService      *UserPreferenceService
    deliveryService  *NotificationDeliveryService
}

func (r *MyAlertRouter) RouteAlert(ctx context.Context, alert *Alert) error {
    // 1. Determine target users
    users := r.determineTargetUsers(alert)

    // 2. Check alert fatigue
    // 3. Get user preferences
    // 4. Send notifications
    // 5. Schedule escalations

    return nil
}
```

## Configuration

### Environment Variables

```bash
KAFKA_BROKERS=pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
KAFKA_GROUP_ID=notification-service-consumers
KAFKA_USERNAME=xxx
KAFKA_PASSWORD=xxx
KAFKA_AUTO_COMMIT=true
KAFKA_SESSION_TIMEOUT_MS=30000
KAFKA_WORKER_POOL_SIZE=10
```

### Configuration Options

```go
type ConsumerConfig struct {
    // Required
    Brokers  []string  // Kafka broker addresses
    GroupID  string    // Consumer group ID
    Topics   []string  // Topics to subscribe to

    // Optional (defaults provided)
    Username             string  // SASL username
    Password             string  // SASL password
    AutoCommit           bool    // Enable auto-commit (default: true)
    SessionTimeoutMs     int     // Session timeout (default: 30000)
    MaxPollRecords       int     // Max records per poll (default: 100)
    WorkerPoolSize       int     // Parallel workers (default: 10)
    EnableAutoOffsetStore bool   // Auto store offsets (default: false)
}
```

## Metrics

### Available Metrics

The consumer tracks comprehensive metrics accessible via `GetMetrics()`:

```go
type ConsumerMetrics struct {
    MessagesConsumed      int64              // Total messages consumed
    MessagesProcessed     int64              // Successfully processed
    MessagesFailed        int64              // Failed processing
    ProcessingErrors      map[string]int64   // Error counts by type
    LastMessageTimestamp  time.Time          // Last message received
    ConsumerLag           int64              // Current consumer lag
    ProcessingDurationMs  []int64            // Last 100 processing times
    TopicMessageCounts    map[string]int64   // Messages per topic
}
```

### Monitoring Example

```go
// Get current metrics
metrics := consumer.GetMetrics()

fmt.Printf("Consumed: %d\n", metrics.MessagesConsumed)
fmt.Printf("Processed: %d\n", metrics.MessagesProcessed)
fmt.Printf("Failed: %d\n", metrics.MessagesFailed)
fmt.Printf("Lag: %d\n", metrics.ConsumerLag)

// Calculate average processing time
if len(metrics.ProcessingDurationMs) > 0 {
    var sum int64
    for _, duration := range metrics.ProcessingDurationMs {
        sum += duration
    }
    avg := sum / int64(len(metrics.ProcessingDurationMs))
    fmt.Printf("Avg Processing Time: %dms\n", avg)
}
```

## Health Checks

```go
// Check consumer health
if err := consumer.HealthCheck(); err != nil {
    log.Printf("Consumer unhealthy: %v", err)
}

// Health check validates:
// - Consumer is running
// - Recent message activity (< 5 minutes)
```

## Error Handling

### Error Categories

1. **Deserialization Errors**: Invalid JSON or schema mismatches
2. **Validation Errors**: Missing required fields
3. **Routing Errors**: Failed to route alert to channels
4. **Kafka Errors**: Broker connection issues, offset commit failures

### Error Tracking

All errors are:
- Logged with structured fields (alert_id, patient_id, etc.)
- Tracked in metrics by error type
- Incremented in failure counters

### Retry Behavior

- **Auto-retry**: Kafka client handles connection retries (up to 8 attempts)
- **Manual Commit**: If auto-commit disabled, offsets only committed on success
- **Dead Letter Queue**: Consider implementing DLQ for persistent failures

## Performance

### Benchmarks

```
BenchmarkAlertValidation-8        5000000    250 ns/op      0 B/op    0 allocs/op
BenchmarkAlertDeserialization-8   500000    3200 ns/op   1800 B/op   45 allocs/op
```

### Throughput

- **Target**: 1000 alerts/second
- **Achieved**: Depends on router implementation and external API latency
- **Worker Pool**: Tune `WorkerPoolSize` based on workload (10-50 workers typical)

### Latency Targets

| Metric | Target |
|--------|--------|
| Message Consumption | < 10ms P99 |
| Alert Validation | < 1ms |
| Alert Routing | < 100ms P99 (depends on router) |
| Total End-to-End | < 200ms P99 |

## Testing

### Unit Tests

```bash
go test -v ./internal/kafka
```

### Integration Tests

Requires running Kafka cluster:

```bash
# Start test Kafka (Docker)
docker-compose -f docker-compose.test.yml up -d kafka

# Run integration tests
go test -v -tags=integration ./internal/kafka
```

### Load Testing

```bash
# Produce test messages
go run cmd/load-test/main.go \
  --brokers localhost:9092 \
  --topic ml-risk-alerts.v1 \
  --rate 1000 \
  --duration 60s

# Monitor consumer metrics
go run cmd/consumer-monitor/main.go
```

## Troubleshooting

### High Consumer Lag

**Symptoms**: `ConsumerLag` metric increasing over time

**Solutions**:
- Increase `WorkerPoolSize` for more parallelism
- Optimize AlertRouter implementation
- Scale horizontally (more consumer instances)
- Check for external API bottlenecks

### Processing Failures

**Symptoms**: High `MessagesFailed` count

**Solutions**:
- Check logs for error patterns
- Review `ProcessingErrors` metrics by error type
- Validate alert schema matches producers
- Verify router dependencies are healthy

### No Messages Received

**Symptoms**: `LastMessageTimestamp` not updating

**Solutions**:
- Verify Kafka connectivity and credentials
- Check topic names match producers
- Confirm consumer group has assigned partitions
- Review Kafka broker logs for issues

### Slow Message Processing

**Symptoms**: High processing duration in metrics

**Solutions**:
- Profile AlertRouter implementation
- Check external API latency
- Review database query performance
- Consider async notification delivery

## Best Practices

1. **Idempotency**: Ensure alert processing is idempotent (may receive duplicates)
2. **Observability**: Monitor all metrics continuously
3. **Graceful Shutdown**: Always call `Stop()` with timeout context
4. **Error Logging**: Log all errors with structured context
5. **Resource Limits**: Set appropriate worker pool size for your workload
6. **Testing**: Test with production-like message volumes
7. **Schema Evolution**: Handle backward-compatible schema changes gracefully

## Dependencies

```go
require (
    github.com/confluentinc/confluent-kafka-go/v2 v2.3.0
    go.uber.org/zap v1.26.0
    github.com/stretchr/testify v1.8.4 // for tests
)
```

## Example Alert Events

### Sepsis Alert (ml-risk-alerts.v1)

```json
{
  "alert_id": "alert-sep-001",
  "patient_id": "PAT-001",
  "hospital_id": "HOSP-001",
  "department_id": "ICU",
  "alert_type": "SEPSIS_ALERT",
  "severity": "CRITICAL",
  "confidence": 0.92,
  "message": "Patient PAT-001 sepsis risk elevated to 92%",
  "recommendations": [
    "Immediate physician review",
    "Blood culture within 30 minutes",
    "Broad-spectrum antibiotics within 1 hour"
  ],
  "patient_location": {"room": "ICU-5", "bed": "A"},
  "vital_signs": {
    "heart_rate": 125,
    "blood_pressure_systolic": 85,
    "temperature": 39.2,
    "respiratory_rate": 28
  },
  "timestamp": 1699564800000,
  "metadata": {
    "source_module": "MODULE5_ML_INFERENCE",
    "model_version": "sepsis-v1.2.3",
    "requires_escalation": true,
    "priority": 1
  }
}
```

### Vital Sign Anomaly (clinical-patterns.v1)

```json
{
  "alert_id": "alert-vs-002",
  "patient_id": "PAT-002",
  "hospital_id": "HOSP-001",
  "department_id": "CARDIOLOGY",
  "alert_type": "VITAL_SIGN_ANOMALY",
  "severity": "HIGH",
  "confidence": 0.88,
  "message": "Sustained tachycardia detected for 15 minutes",
  "recommendations": [
    "Check patient status",
    "Review cardiac medications",
    "Consider EKG"
  ],
  "patient_location": {"room": "CARD-12", "bed": "B"},
  "vital_signs": {
    "heart_rate": 135,
    "blood_pressure_systolic": 145,
    "blood_pressure_diastolic": 92
  },
  "timestamp": 1699564860000,
  "metadata": {
    "source_module": "MODULE4_CEP",
    "requires_escalation": false,
    "priority": 2,
    "alert_tags": ["cardiology", "vital_signs", "tachycardia"]
  }
}
```

### Manual Escalation (alert-management.v1)

```json
{
  "alert_id": "alert-esc-003",
  "patient_id": "PAT-003",
  "hospital_id": "HOSP-001",
  "department_id": "EMERGENCY",
  "alert_type": "ESCALATION",
  "severity": "CRITICAL",
  "confidence": 1.0,
  "message": "Manual escalation: Rapid response team requested",
  "recommendations": [
    "Rapid response team activation",
    "ICU transfer evaluation"
  ],
  "patient_location": {"room": "ER-8", "bed": "A"},
  "vital_signs": {},
  "timestamp": 1699564920000,
  "metadata": {
    "source_module": "ALERT_MANAGEMENT",
    "requires_escalation": true,
    "priority": 1,
    "triggered_by": "nurse-001"
  }
}
```

## License

Copyright 2025 CardioFit Platform. All rights reserved.
