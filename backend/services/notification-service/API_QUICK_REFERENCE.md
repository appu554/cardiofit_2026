# Notification Service API Quick Reference

## Service Ports
- **HTTP**: 8060
- **gRPC**: 50060

---

## HTTP API Endpoints

### Health & Monitoring
```bash
# Liveness probe
curl http://localhost:8060/health

# Readiness probe
curl http://localhost:8060/ready

# Prometheus metrics
curl http://localhost:8060/metrics
```

### Acknowledge Alert
```bash
curl -X POST http://localhost:8060/api/v1/notifications/acknowledge \
  -H "Content-Type: application/json" \
  -d '{
    "alert_id": "alert-123",
    "user_id": "user-456",
    "notification_id": "notif-789",
    "acknowledgment_note": "Acknowledged by clinician"
  }'
```

### Get Notification Status
```bash
curl http://localhost:8060/api/v1/notifications/{notification_id}
```

### Get Escalation History
```bash
curl http://localhost:8060/api/v1/escalations/{alert_id}
```

---

## gRPC API Methods

### Send Notification
```bash
grpcurl -plaintext -d '{
  "patient_id": "patient-123",
  "priority": "high",
  "type": "sepsis_alert",
  "message": "Patient shows signs of sepsis",
  "recipients": ["doctor@hospital.com"],
  "requires_ack": true
}' localhost:50060 notification.NotificationService/SendNotification
```

### Get Delivery Status
```bash
grpcurl -plaintext -d '{
  "notification_id": "notif-789"
}' localhost:50060 notification.NotificationService/GetDeliveryStatus
```

### Acknowledge Alert
```bash
grpcurl -plaintext -d '{
  "alert_id": "alert-123",
  "user_id": "user-456",
  "notification_id": "notif-789"
}' localhost:50060 notification.NotificationService/AcknowledgeAlert
```

### Update User Preferences
```bash
grpcurl -plaintext -d '{
  "user_id": "user-123",
  "preferred_channels": ["email", "sms"],
  "quiet_hours_start": "22:00",
  "quiet_hours_end": "07:00"
}' localhost:50060 notification.NotificationService/UpdatePreferences
```

### Get User Preferences
```bash
grpcurl -plaintext -d '{
  "user_id": "user-123"
}' localhost:50060 notification.NotificationService/GetPreferences
```

---

## Common Response Codes

### HTTP
- **200 OK**: Success
- **400 Bad Request**: Invalid request body or missing required fields
- **404 Not Found**: Resource not found
- **405 Method Not Allowed**: Wrong HTTP method
- **408 Request Timeout**: Request exceeded 30 second timeout
- **500 Internal Server Error**: Server error (check logs)
- **503 Service Unavailable**: Service dependencies unhealthy

### gRPC
- **OK**: Success
- **INVALID_ARGUMENT**: Missing or invalid required fields
- **NOT_FOUND**: Resource not found
- **UNAUTHENTICATED**: Missing or invalid authentication
- **INTERNAL**: Internal server error

---

## Key Metrics

### Request Metrics
- `notification_http_requests_total{method, path, status}`
- `notification_grpc_requests_total{method, status}`
- `notification_http_request_duration_seconds{method, path}`
- `notification_grpc_request_duration_seconds{method}`

### Business Metrics
- `notification_deliveries_total{channel, status}`
- `notification_escalation_events_total{level}`
- `notification_alert_fatigue_suppressions_total{reason}`
- `notification_kafka_messages_processed_total{topic, status}`

---

## Development Commands

### Start Service
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/services/notification-service
go run ./cmd/server
```

### Run Tests
```bash
# All tests
go test ./internal/server/... -v

# With coverage
go test ./internal/server/... -v -cover

# Specific test
go test ./internal/server/ -v -run TestHandleHealth
```

### Generate Protobuf Code
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       pkg/proto/notification.proto
```

### Build Service
```bash
go build -o notification-service ./cmd/server
```

---

## Troubleshooting

### Service won't start
1. Check port conflicts: `lsof -i :8060` and `lsof -i :50060`
2. Verify database connectivity: Check logs for PostgreSQL errors
3. Verify Redis connectivity: Check logs for Redis errors
4. Check Kafka broker: Ensure Kafka is running and accessible

### Health check failing
1. Check `/ready` endpoint for specific dependency failures
2. Verify database connection: `psql -h localhost -U notification_user -d notification_db`
3. Verify Redis connection: `redis-cli -h localhost -p 6379 ping`
4. Check Kafka: `kafka-topics --list --bootstrap-server localhost:9092`

### API returns 500 errors
1. Check service logs for stack traces
2. Verify database schema is up to date
3. Check for missing environment variables
4. Verify all required tables exist

### Metrics not appearing
1. Check `/metrics` endpoint is accessible
2. Verify Prometheus is scraping the correct port (8060)
3. Check metrics are being recorded (add debug logging)

---

## Configuration

### Required Environment Variables
```bash
export DATABASE_HOST=localhost
export DATABASE_PORT=5432
export DATABASE_USER=notification_user
export DATABASE_PASSWORD=secure_password
export DATABASE_DBNAME=notification_db

export REDIS_HOST=localhost
export REDIS_PORT=6379

export KAFKA_BROKERS=localhost:9092
```

### Optional Environment Variables
```bash
export SERVER_HTTP_PORT=8060
export SERVER_GRPC_PORT=50060
export LOG_LEVEL=info
export LOG_FORMAT=json
```

---

## Testing with curl

### Full workflow example
```bash
# 1. Check service health
curl http://localhost:8060/health

# 2. Send notification (via gRPC)
grpcurl -plaintext -d '{
  "patient_id": "patient-123",
  "priority": "high",
  "message": "Test alert",
  "recipients": ["test@example.com"]
}' localhost:50060 notification.NotificationService/SendNotification
# Response: {"notification_id": "notif-abc", ...}

# 3. Check delivery status
curl http://localhost:8060/api/v1/notifications/notif-abc

# 4. Acknowledge alert
curl -X POST http://localhost:8060/api/v1/notifications/acknowledge \
  -H "Content-Type: application/json" \
  -d '{"alert_id": "alert-123", "user_id": "user-456", "notification_id": "notif-abc"}'

# 5. View escalation history
curl http://localhost:8060/api/v1/escalations/alert-123

# 6. Check metrics
curl http://localhost:8060/metrics | grep notification_deliveries_total
```

---

## Files Reference

### Implementation Files
- `internal/server/http_server.go` - HTTP API server
- `internal/server/grpc_server.go` - gRPC service implementation
- `internal/server/middleware.go` - Middleware and interceptors
- `pkg/proto/notification.proto` - Protocol buffer definitions
- `cmd/server/main.go` - Service entry point

### Test Files
- `internal/server/http_server_test.go` - HTTP tests (561 lines)
- `internal/server/grpc_server_test.go` - gRPC tests (545 lines)

### Documentation
- `PHASE2_5_COMPLETION_REPORT.md` - Full implementation report
- `API_QUICK_REFERENCE.md` - This file

---

## Support

For detailed documentation, see:
- Full API documentation: `PHASE2_5_COMPLETION_REPORT.md`
- Protocol buffer schema: `pkg/proto/notification.proto`
- Configuration options: `internal/config/config.go`

For issues or questions:
1. Check service logs for error messages
2. Verify configuration is correct
3. Ensure all dependencies are running
4. Review test files for usage examples
