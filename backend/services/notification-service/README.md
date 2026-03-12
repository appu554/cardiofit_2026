# Notification Service

Multi-channel notification delivery system for clinical alerts in the CardioFit platform.

## Overview

The Notification Service (Module 6 - Component 6C) provides intelligent, multi-channel notification delivery with fatigue management, escalation policies, and delivery tracking. It consumes clinical alerts from Kafka and routes them to appropriate channels (email, SMS, push) based on priority, user preferences, and fatigue rules.

## Features

- **Multi-Channel Delivery**: Email (SendGrid), SMS (Twilio), Push (Firebase)
- **Intelligent Routing**: Priority-based channel selection with user preferences
- **Fatigue Management**: Prevents notification overload with configurable limits and quiet hours
- **Escalation Engine**: Automatic escalation for failed critical notifications
- **Delivery Tracking**: Comprehensive logging and metrics for all deliveries
- **High Availability**: Kafka consumer with offset management and graceful shutdown
- **Health Monitoring**: Health and readiness endpoints with Prometheus metrics

## Architecture

```
┌─────────────┐
│   Kafka     │
│ (Alerts)    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────┐
│      Notification Service               │
│                                         │
│  ┌──────────────┐   ┌───────────────┐  │
│  │   Routing    │──▶│   Fatigue     │  │
│  │   Engine     │   │   Manager     │  │
│  └──────┬───────┘   └───────────────┘  │
│         │                               │
│         ▼                               │
│  ┌──────────────┐   ┌───────────────┐  │
│  │   Delivery   │──▶│  Escalation   │  │
│  │   Manager    │   │   Engine      │  │
│  └──────┬───────┘   └───────────────┘  │
│         │                               │
└─────────┼───────────────────────────────┘
          │
     ┌────┴────┬─────────┬─────────┐
     ▼         ▼         ▼         ▼
┌─────────┐ ┌──────┐ ┌──────┐ ┌──────────┐
│SendGrid │ │Twilio│ │Firebase│ │PostgreSQL│
└─────────┘ └──────┘ └──────┘ └──────────┘
```

## Technology Stack

- **Language**: Go 1.21+
- **Message Queue**: Apache Kafka (Confluent)
- **Database**: PostgreSQL (delivery tracking)
- **Cache**: Redis (fatigue management)
- **Email**: SendGrid
- **SMS**: Twilio
- **Push**: Firebase Cloud Messaging
- **Observability**: Prometheus, Zap logging

## Project Structure

```
notification-service/
├── cmd/
│   └── server/
│       └── main.go              # Service entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── kafka/
│   │   └── consumer.go          # Kafka consumer implementation
│   ├── routing/
│   │   └── engine.go            # Notification routing logic
│   ├── delivery/
│   │   ├── manager.go           # Delivery orchestration
│   │   ├── sendgrid.go          # Email provider
│   │   ├── twilio.go            # SMS provider
│   │   └── firebase.go          # Push provider
│   ├── escalation/
│   │   └── engine.go            # Escalation logic
│   ├── fatigue/
│   │   └── manager.go           # Fatigue management
│   ├── database/
│   │   ├── postgres.go          # PostgreSQL client
│   │   └── redis.go             # Redis client
│   └── models/
│       └── alert.go             # Data models
├── pkg/
│   ├── models/                  # Public data models
│   └── proto/                   # gRPC protobuf definitions
├── configs/
│   └── config.yaml              # Configuration file
├── scripts/
│   └── setup.sh                 # Setup script
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── kubernetes/              # K8s manifests (future)
├── docs/                        # Documentation
├── tests/
│   ├── unit/                    # Unit tests
│   ├── integration/             # Integration tests
│   └── e2e/                     # End-to-end tests
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- Redis 7+
- Kafka (or Confluent Cloud)
- SendGrid API key
- Twilio credentials
- Firebase credentials

### Local Development

1. **Clone and setup**:
```bash
cd backend/services/notification-service
./scripts/setup.sh
```

2. **Configure environment variables**:
```bash
export SENDGRID_API_KEY="your-sendgrid-api-key"
export TWILIO_ACCOUNT_SID="your-twilio-sid"
export TWILIO_AUTH_TOKEN="your-twilio-token"
export TWILIO_FROM_NUMBER="+1234567890"
export FIREBASE_CREDENTIALS_PATH="./credentials/firebase.json"
export FIREBASE_PROJECT_ID="your-firebase-project"
```

3. **Start dependencies** (PostgreSQL, Redis, Kafka):
```bash
make docker-up
```

4. **Run the service**:
```bash
make run
```

### Docker Deployment

```bash
# Build Docker image
make docker-build

# Start all services
make docker-up

# Check service health
curl http://localhost:8050/health

# View logs
docker-compose -f deployments/docker/docker-compose.yml logs -f notification-service

# Stop services
make docker-down
```

## Configuration

Configuration is managed through `configs/config.yaml` and environment variables. Key configuration sections:

- **Server**: HTTP/gRPC ports, environment
- **Database**: PostgreSQL connection settings
- **Redis**: Cache configuration for fatigue management
- **Kafka**: Broker addresses, consumer group, topics
- **Delivery**: Provider credentials and settings
- **Routing**: Channel priorities, retry policies
- **Escalation**: Escalation policies and delays
- **Fatigue**: Notification limits, quiet hours

See `configs/config.yaml` for all available options.

## API Endpoints

### HTTP Endpoints

- `GET /health` - Health check (database, Redis connectivity)
- `GET /ready` - Readiness check (Kafka consumer status)
- `GET /metrics` - Prometheus metrics

### gRPC Services

gRPC service definitions will be added in `pkg/proto/` for:
- Manual notification triggering
- Delivery status queries
- Preference management

## Notification Flow

1. **Alert Reception**: Kafka consumer receives clinical alert
2. **Fatigue Check**: Check recipient notification limits and quiet hours
3. **Routing Decision**: Select channel based on priority and preferences
4. **Delivery**: Send notification through selected channel
5. **Escalation**: If delivery fails and alert is critical, escalate to next channel
6. **Tracking**: Log delivery result to PostgreSQL

## Fatigue Management

Prevents notification overload:

- **Window-based limits**: Max notifications per time window (default: 10/hour)
- **Quiet hours**: Suppress non-critical notifications during off-hours (22:00-07:00)
- **Priority bypass**: Critical alerts always delivered regardless of fatigue
- **Per-recipient tracking**: Individual limits per user

## Escalation Policy

Automatic escalation for failed critical notifications:

1. **Email** (initial) → **Push** (5min) → **SMS** (10min)
2. Configurable delays and max escalation levels
3. Critical channels: SMS and Push for highest priority

## Monitoring

### Metrics

Prometheus metrics available at `/metrics`:

- `notifications_total` - Total notifications processed
- `notifications_delivered` - Successful deliveries by channel
- `notifications_failed` - Failed deliveries by channel
- `notification_delivery_duration` - Delivery latency
- `fatigue_suppressions_total` - Notifications suppressed by fatigue
- `escalations_total` - Escalations triggered

### Logging

Structured JSON logging with Zap:
- Alert processing events
- Delivery attempts and results
- Escalation triggers
- Error conditions

## Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Generate coverage report
make test
go tool cover -html=coverage.out
```

## Development

```bash
# Install dependencies
make deps

# Run linters
make lint

# Build binary
make build

# Clean build artifacts
make clean
```

## Integration with CardioFit

This service integrates with:

- **Stream Processing** (Module 4): Consumes alerts from `clinical-alerts` topic
- **Clinical Reasoning** (Module 3): Receives ML-generated alerts
- **Patient Service**: Retrieves user preferences and contact information
- **Auth Service**: Validates notification permissions
- **Apollo Federation**: Exposes GraphQL queries for delivery status

## Security

- **Authentication**: Service-to-service JWT validation
- **Encryption**: TLS for external provider communication
- **Secrets Management**: Environment variables for credentials
- **PII Protection**: PHI data handling compliant with HIPAA
- **Audit Logging**: All notification deliveries tracked

## Performance

- **Throughput**: 1000+ notifications/second
- **Latency**: < 100ms processing time (excluding provider delays)
- **Concurrency**: Parallel delivery to multiple channels
- **Retry Logic**: Exponential backoff for failed deliveries

## Troubleshooting

### Service won't start
- Check PostgreSQL and Redis connectivity
- Verify Kafka broker availability
- Ensure provider credentials are set

### Notifications not delivered
- Check provider API credentials
- Review fatigue management limits
- Verify Kafka consumer is reading messages

### High delivery failures
- Check provider service status
- Review error logs for specific failures
- Verify recipient contact information

## Contributing

Follow CardioFit Go service patterns:
1. Use structured logging (Zap)
2. Implement health checks
3. Add comprehensive tests
4. Document all exported functions
5. Follow Go best practices and project conventions

## License

Proprietary - CardioFit Platform

## Support

For issues or questions:
- Internal: CardioFit Platform Team
- Documentation: `/docs` directory
- Architecture: See Module 6 specifications
