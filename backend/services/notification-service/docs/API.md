# Notification Service API Documentation

## REST API Endpoints

### Health Checks

#### GET /health
Health check endpoint that verifies database and Redis connectivity.

**Response**:
```
Status: 200 OK
Body: "OK"
```

**Failure**:
```
Status: 503 Service Unavailable
Body: "Database unhealthy" or "Redis unhealthy"
```

#### GET /ready
Readiness check that verifies Kafka consumer is operational.

**Response**:
```
Status: 200 OK
Body: "Ready"
```

**Failure**:
```
Status: 503 Service Unavailable
Body: "Kafka consumer not ready"
```

#### GET /metrics
Prometheus metrics endpoint exposing service metrics.

**Response**:
```
Status: 200 OK
Content-Type: text/plain

# HELP notifications_total Total number of notifications processed
# TYPE notifications_total counter
notifications_total{channel="email",priority="high"} 1234
...
```

## gRPC API

### SendNotification

Manually triggers a notification delivery.

**Request**:
```protobuf
message SendNotificationRequest {
  string patient_id = 1;
  string priority = 2;        // "critical", "high", "medium", "low"
  string type = 3;            // Alert type identifier
  string title = 4;
  string message = 5;
  repeated string recipients = 6;
  map<string, string> metadata = 7;
  bool requires_ack = 8;
}
```

**Response**:
```protobuf
message SendNotificationResponse {
  string notification_id = 1;
  string status = 2;         // "queued", "processing", "delivered", "failed"
  google.protobuf.Timestamp created_at = 3;
}
```

**Example**:
```go
req := &pb.SendNotificationRequest{
    PatientId: "patient-123",
    Priority:  "high",
    Type:      "cardiac-alert",
    Title:     "Abnormal Heart Rate",
    Message:   "Patient heart rate exceeded threshold",
    Recipients: []string{"doctor@hospital.com"},
    Metadata: map[string]string{
        "alert_id": "alert-456",
        "threshold": "120bpm",
    },
    RequiresAck: true,
}
```

### GetDeliveryStatus

Retrieves the delivery status of a notification.

**Request**:
```protobuf
message GetDeliveryStatusRequest {
  string notification_id = 1;
}
```

**Response**:
```protobuf
message GetDeliveryStatusResponse {
  string notification_id = 1;
  string status = 2;
  string channel = 3;
  repeated DeliveryAttempt attempts = 4;
  google.protobuf.Timestamp last_updated = 5;
}

message DeliveryAttempt {
  string channel = 1;
  bool success = 2;
  string error = 3;
  google.protobuf.Timestamp attempted_at = 4;
}
```

### UpdatePreferences

Updates user notification preferences.

**Request**:
```protobuf
message UpdatePreferencesRequest {
  string user_id = 1;
  repeated string preferred_channels = 2;  // ["email", "sms", "push"]
  string quiet_hours_start = 3;           // "22:00"
  string quiet_hours_end = 4;             // "07:00"
  repeated string enabled_alert_types = 5;
  string priority_threshold = 6;          // Minimum priority to deliver
}
```

**Response**:
```protobuf
message UpdatePreferencesResponse {
  bool success = 1;
  string message = 2;
}
```

### GetPreferences

Retrieves user notification preferences.

**Request**:
```protobuf
message GetPreferencesRequest {
  string user_id = 1;
}
```

**Response**:
```protobuf
message GetPreferencesResponse {
  string user_id = 1;
  repeated string preferred_channels = 2;
  string quiet_hours_start = 3;
  string quiet_hours_end = 4;
  repeated string enabled_alert_types = 5;
  string priority_threshold = 6;
}
```

## Kafka Message Format

The service consumes clinical alerts from the `clinical-alerts` Kafka topic.

**Message Structure**:
```json
{
  "id": "alert-123",
  "patient_id": "patient-456",
  "priority": "critical",
  "type": "cardiac-alert",
  "title": "Cardiac Arrest Risk",
  "message": "Patient vital signs indicate high risk",
  "metadata": {
    "heart_rate": "145",
    "blood_pressure": "180/110",
    "alert_source": "ml-model"
  },
  "recipients": [
    "doctor@hospital.com",
    "+15551234567"
  ],
  "timestamp": "2024-01-15T14:30:00Z",
  "expires_at": "2024-01-15T15:00:00Z",
  "requires_ack": true
}
```

## Error Codes

### HTTP Status Codes
- `200 OK` - Success
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Missing or invalid authentication
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service dependencies unavailable

### gRPC Status Codes
- `OK (0)` - Success
- `INVALID_ARGUMENT (3)` - Invalid request parameters
- `NOT_FOUND (5)` - Resource not found
- `ALREADY_EXISTS (6)` - Resource already exists
- `PERMISSION_DENIED (7)` - Insufficient permissions
- `RESOURCE_EXHAUSTED (8)` - Rate limit exceeded
- `UNAVAILABLE (14)` - Service unavailable
- `INTERNAL (13)` - Internal server error

## Rate Limits

- **Manual notifications**: 100 requests/minute per user
- **Preference updates**: 10 requests/minute per user
- **Status queries**: 1000 requests/minute per user

## Authentication

All gRPC endpoints require JWT authentication via metadata:

```go
md := metadata.Pairs("authorization", "Bearer "+token)
ctx := metadata.NewOutgoingContext(context.Background(), md)
```

## Examples

### Send Critical Alert via gRPC

```go
import (
    "context"
    pb "github.com/cardiofit/notification-service/pkg/proto"
    "google.golang.org/grpc"
)

conn, _ := grpc.Dial("localhost:9050", grpc.WithInsecure())
client := pb.NewNotificationServiceClient(conn)

resp, err := client.SendNotification(context.Background(), &pb.SendNotificationRequest{
    PatientId: "patient-123",
    Priority:  "critical",
    Type:      "sepsis-alert",
    Title:     "Sepsis Risk Detected",
    Message:   "Immediate intervention required",
    Recipients: []string{"oncall@hospital.com", "+15551234567"},
})
```

### Check Delivery Status

```bash
curl -X GET http://localhost:8050/health

grpcurl -plaintext \
  -d '{"notification_id": "notif-123"}' \
  localhost:9050 \
  notification.NotificationService/GetDeliveryStatus
```

### Update User Preferences

```bash
grpcurl -plaintext \
  -d '{
    "user_id": "user-123",
    "preferred_channels": ["email", "push"],
    "quiet_hours_start": "22:00",
    "quiet_hours_end": "07:00",
    "priority_threshold": "high"
  }' \
  localhost:9050 \
  notification.NotificationService/UpdatePreferences
```
