# Notification Service - Project Structure Summary

## Overview
Complete Go project structure for Module 6 - Component 6C: Multi-Channel Notification System

**Location**: `/Users/apoorvabk/Downloads/cardiofit/backend/services/notification-service`
**Language**: Go 1.21+
**Module**: `github.com/cardiofit/notification-service`

## Directory Structure

```
notification-service/
├── cmd/                                    # Application entry points
│   └── server/
│       └── main.go                        # Main service entry point (HTTP/gRPC servers)
│
├── internal/                              # Private application code
│   ├── config/
│   │   └── config.go                      # Configuration management (Viper)
│   ├── kafka/
│   │   └── consumer.go                    # Kafka consumer implementation
│   ├── routing/
│   │   └── engine.go                      # Notification routing logic
│   ├── delivery/
│   │   ├── manager.go                     # Delivery orchestration
│   │   ├── sendgrid.go                    # Email provider (SendGrid)
│   │   ├── twilio.go                      # SMS provider (Twilio)
│   │   └── firebase.go                    # Push provider (Firebase)
│   ├── escalation/
│   │   └── engine.go                      # Escalation logic
│   ├── fatigue/
│   │   └── manager.go                     # Fatigue management (Redis-based)
│   ├── database/
│   │   ├── postgres.go                    # PostgreSQL client (pgx/v5)
│   │   └── redis.go                       # Redis client
│   └── models/
│       └── alert.go                       # Internal data models
│
├── pkg/                                   # Public packages
│   ├── models/                            # Exported data models
│   └── proto/
│       └── notification.proto             # gRPC service definitions
│
├── configs/
│   └── config.yaml                        # Configuration file
│
├── scripts/
│   └── setup.sh                           # Setup and initialization script
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile                     # Multi-stage Docker build
│   │   └── docker-compose.yml             # Local development stack
│   └── kubernetes/
│       └── deployment.yaml                # K8s deployment manifest
│
├── docs/
│   ├── ARCHITECTURE.md                    # Architecture documentation
│   └── API.md                             # API documentation
│
├── tests/
│   ├── unit/
│   │   └── routing_test.go                # Unit tests
│   ├── integration/                       # Integration tests
│   └── e2e/                               # End-to-end tests
│
├── go.mod                                 # Go module definition
├── go.sum                                 # Dependency checksums
├── Makefile                               # Build and development commands
├── .gitignore                             # Git ignore patterns
├── .env.example                           # Environment variables template
└── README.md                              # Project documentation
```

## Core Components

### 1. Main Server (`cmd/server/main.go`)
- HTTP server (port 8050) for health checks and metrics
- gRPC server (port 9050) for service APIs
- Kafka consumer initialization and management
- Graceful shutdown handling
- Health check endpoints: `/health`, `/ready`, `/metrics`

### 2. Kafka Consumer (`internal/kafka/consumer.go`)
- Consumes clinical alerts from `clinical-alerts` topic
- Manual offset management for at-least-once delivery
- Integrates routing, delivery, and escalation engines
- Error handling and retry logic

### 3. Routing Engine (`internal/routing/engine.go`)
- Priority-based channel selection (critical→SMS, high→push, medium→email)
- Fatigue management integration
- User preference handling
- Retry policy configuration

### 4. Delivery Manager (`internal/delivery/manager.go`)
- Multi-provider orchestration
- Channel-specific delivery (email, SMS, push)
- Delivery result tracking
- Error categorization and handling

### 5. Fatigue Manager (`internal/fatigue/manager.go`)
- Redis-based rate limiting
- Quiet hours enforcement (22:00-07:00 default)
- Per-recipient notification counting
- Priority-based bypass rules

### 6. Escalation Engine (`internal/escalation/engine.go`)
- Failed delivery detection
- Progressive channel escalation (email→push→SMS)
- Configurable delays and max levels
- Critical alert prioritization

### 7. Database Clients
- **PostgreSQL** (`internal/database/postgres.go`): Delivery tracking, preferences
- **Redis** (`internal/database/redis.go`): Fatigue management, caching

### 8. Configuration (`internal/config/config.go`)
- Viper-based configuration management
- YAML file + environment variable support
- Structured configuration for all components
- Sensible defaults

## Key Dependencies

### Core Libraries
- `github.com/confluentinc/confluent-kafka-go/v2` - Kafka client
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/spf13/viper` - Configuration management
- `go.uber.org/zap` - Structured logging
- `google.golang.org/grpc` - gRPC framework
- `github.com/prometheus/client_golang` - Metrics

### External Providers
- `github.com/sendgrid/sendgrid-go` - Email delivery
- `github.com/twilio/twilio-go` - SMS delivery
- `firebase.google.com/go/v4` - Push notifications

### Testing
- `github.com/stretchr/testify` - Testing utilities

## Configuration

### Environment Variables
Set via `.env` or export:
```bash
SENDGRID_API_KEY=your-key
TWILIO_ACCOUNT_SID=your-sid
TWILIO_AUTH_TOKEN=your-token
TWILIO_FROM_NUMBER=+1234567890
FIREBASE_CREDENTIALS_PATH=./credentials/firebase.json
FIREBASE_PROJECT_ID=your-project-id
```

### Configuration File (`configs/config.yaml`)
- Server settings (ports, environment)
- Database connections (PostgreSQL, Redis)
- Kafka settings (brokers, topics, consumer group)
- Delivery providers (credentials, settings)
- Routing rules (priorities, retry policies)
- Escalation policies (delays, max levels)
- Fatigue management (limits, quiet hours)

## Build and Run

### Local Development
```bash
# Setup
./scripts/setup.sh

# Install dependencies
make deps

# Run service
make run

# Run tests
make test
```

### Docker
```bash
# Build image
make docker-build

# Start stack (PostgreSQL, Redis, Kafka, service)
make docker-up

# Stop stack
make docker-down
```

### Kubernetes
```bash
# Apply manifests
kubectl apply -f deployments/kubernetes/deployment.yaml

# Check status
kubectl get pods -n cardiofit

# View logs
kubectl logs -f -n cardiofit deployment/notification-service
```

## API Endpoints

### REST
- `GET /health` - Health check (database, Redis)
- `GET /ready` - Readiness check (Kafka consumer)
- `GET /metrics` - Prometheus metrics

### gRPC (port 9050)
- `SendNotification` - Manual notification trigger
- `GetDeliveryStatus` - Query delivery status
- `UpdatePreferences` - Update user preferences
- `GetPreferences` - Retrieve user preferences

## Testing Strategy

### Unit Tests (`tests/unit/`)
- Routing logic validation
- Fatigue management rules
- Escalation policies
- Configuration loading

### Integration Tests (`tests/integration/`)
- Database operations
- Redis operations
- Provider integration
- Kafka consumer behavior

### E2E Tests (`tests/e2e/`)
- Complete notification flow
- Multi-channel delivery
- Escalation scenarios
- Error handling

## Monitoring

### Prometheus Metrics
- `notifications_total` - Total notifications processed
- `notifications_delivered` - Successful deliveries by channel
- `notifications_failed` - Failed deliveries by channel
- `notification_delivery_duration` - Delivery latency
- `fatigue_suppressions_total` - Fatigue-based suppressions
- `escalations_total` - Escalation events

### Logging
- Structured JSON logging (Zap)
- Log levels: debug, info, warn, error
- Contextual information (alert ID, patient ID, channel)

## Service Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 8050 | HTTP | Health checks, metrics |
| 9050 | gRPC | Service APIs |

## Integration Points

### Upstream (Producers)
- **Stream Processing** (Module 4): Produces alerts to `clinical-alerts` topic
- **Clinical Reasoning** (Module 3): ML-generated alerts
- **Manual Triggers**: gRPC `SendNotification` API

### Downstream (Dependencies)
- **PostgreSQL**: Delivery tracking, preferences
- **Redis**: Fatigue management, caching
- **Kafka**: Alert consumption
- **SendGrid**: Email delivery
- **Twilio**: SMS delivery
- **Firebase**: Push notification delivery

### Peer Services
- **Patient Service**: User information and preferences
- **Auth Service**: Authentication and authorization
- **Apollo Federation**: GraphQL interface

## Security

- **Authentication**: Service-to-service JWT validation
- **TLS**: All external provider communication encrypted
- **Secrets Management**: Environment variables, Kubernetes secrets
- **PHI Handling**: HIPAA-compliant data handling
- **Audit Logging**: All deliveries tracked

## Performance

- **Throughput**: 1000+ notifications/second
- **Latency**: < 100ms processing time (excluding provider delays)
- **Scalability**: Horizontal scaling via Kafka consumer groups
- **Resilience**: Circuit breakers, retry logic, graceful degradation

## Next Steps

1. **Generate protobuf code**: `make proto`
2. **Run database migrations**: Create migration tool
3. **Implement gRPC services**: Add service implementations
4. **Add comprehensive tests**: Unit, integration, E2E
5. **Set up CI/CD**: Build, test, deploy pipeline
6. **Configure monitoring**: Grafana dashboards, alerts
7. **Load testing**: Validate performance targets

## References

- **CardioFit Architecture**: Module 6 - Notification System
- **API Documentation**: `docs/API.md`
- **Architecture Details**: `docs/ARCHITECTURE.md`
- **Makefile**: Build commands and targets
- **README.md**: Getting started and quick reference
