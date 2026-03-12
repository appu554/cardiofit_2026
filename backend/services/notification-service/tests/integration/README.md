# Integration Tests

Comprehensive integration tests for the CardioFit Notification Service. These tests verify the service works correctly with real infrastructure (PostgreSQL, Redis, Kafka).

## Overview

The integration test suite includes:

1. **Database Integration Tests** (`database_integration_test.go`) - PostgreSQL operations
2. **Redis Integration Tests** (`redis_integration_test.go`) - Cache and alert fatigue tracking
3. **Notification Flow Tests** (`notification_flow_test.go`) - End-to-end notification pipeline
4. **Usage Examples** (`examples/`) - Code examples for developers

## Prerequisites

### Required Infrastructure

1. **PostgreSQL Database**
   - Version: 14+
   - Database: `cardiofit_db`
   - Schema: `notification_service`
   - Tables: `user_preferences`, `users`

2. **Redis Cache**
   - Version: 6+
   - Default port: 6379
   - Recommended: Use DB 1 for testing to isolate from production data

3. **Optional: Apache Kafka**
   - Version: 3.0+
   - Required for Kafka consumer integration tests
   - Can skip if `KAFKA_BROKERS` environment variable is not set

### Setup Instructions

#### 1. Start PostgreSQL

```bash
# Using Docker
docker run --name cardiofit-postgres \
  -e POSTGRES_USER=cardiofit_user \
  -e POSTGRES_PASSWORD=cardiofit_pass \
  -e POSTGRES_DB=cardiofit_db \
  -p 5432:5432 \
  -d postgres:14

# Or use existing PostgreSQL instance
# Update DATABASE_URL environment variable accordingly
```

#### 2. Run Database Migrations

```bash
cd backend/services/notification-service
make migrate-up
```

This creates the required schema and tables:
```sql
CREATE SCHEMA IF NOT EXISTS notification_service;

CREATE TABLE notification_service.user_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    channel_preferences JSONB NOT NULL DEFAULT '{}',
    severity_channels JSONB NOT NULL DEFAULT '{}',
    quiet_hours_enabled BOOLEAN DEFAULT false,
    quiet_hours_start INTEGER,
    quiet_hours_end INTEGER,
    max_alerts_per_hour INTEGER DEFAULT 20,
    phone_number VARCHAR(50),
    email VARCHAR(255),
    pager_number VARCHAR(50),
    fcm_token TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_user_preferences_updated ON notification_service.user_preferences(updated_at);
```

#### 3. Start Redis

```bash
# Using Docker
docker run --name cardiofit-redis \
  -p 6379:6379 \
  -d redis:7-alpine

# Or use existing Redis instance
```

#### 4. (Optional) Start Kafka

```bash
# Using Docker Compose
cd deployments/docker
docker-compose up -d kafka zookeeper

# Kafka integration tests will be skipped if not available
```

## Running Tests

### Run All Integration Tests

```bash
cd backend/services/notification-service

# Run all integration tests
go test -v ./tests/integration/...

# Run with coverage
go test -v ./tests/integration/... -cover

# Skip Kafka tests (if Kafka not available)
unset KAFKA_BROKERS
go test -v ./tests/integration/...
```

### Run Specific Test Suites

```bash
# Database integration tests only
go test -v ./tests/integration/database_integration_test.go

# Redis integration tests only
go test -v ./tests/integration/redis_integration_test.go

# Notification flow tests only
go test -v ./tests/integration/notification_flow_test.go
```

### Run in Short Mode

```bash
# Skip long-running integration tests
go test -v -short ./tests/integration/...
```

### Run Specific Test Cases

```bash
# Run specific test function
go test -v ./tests/integration/ -run TestDatabaseIntegration/UserPreferences_CreateAndRetrieve

# Run all database user preference tests
go test -v ./tests/integration/ -run TestDatabaseIntegration/UserPreferences
```

## Environment Variables

Configure infrastructure connections via environment variables:

```bash
# PostgreSQL connection
export DATABASE_URL="postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"

# Redis connection
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""  # Leave empty if no password

# Kafka connection (optional)
export KAFKA_BROKERS="localhost:9092"

# External service credentials (for delivery tests)
export TWILIO_ACCOUNT_SID="your_account_sid"
export TWILIO_AUTH_TOKEN="your_auth_token"
export TWILIO_FROM_NUMBER="+1234567890"
export SENDGRID_API_KEY="your_api_key"
export SENDGRID_FROM_EMAIL="notifications@cardiofit.com"
export FIREBASE_CREDENTIALS_PATH="path/to/firebase-credentials.json"
```

### Using .env File

Create a `.env.test` file in the project root:

```bash
# .env.test
DATABASE_URL=postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
KAFKA_BROKERS=localhost:9092

# External services (optional - use mocks if not provided)
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_FROM_NUMBER=
SENDGRID_API_KEY=
SENDGRID_FROM_EMAIL=
FIREBASE_CREDENTIALS_PATH=
```

Load environment variables before running tests:

```bash
source .env.test
go test -v ./tests/integration/...
```

## Test Coverage

The integration tests cover:

### Database Integration (database_integration_test.go)

- ✅ User preference CRUD operations
- ✅ PostgreSQL transaction handling
- ✅ Concurrent database writes
- ✅ Nullable field handling (quiet hours)
- ✅ User service with real database
- ✅ JSON field serialization/deserialization

**Coverage**: 76.8% of user service statements

### Redis Integration (redis_integration_test.go)

- ✅ Basic cache operations (set, get, delete)
- ✅ Key expiration and TTL management
- ✅ Alert counter tracking
- ✅ Recent alerts list management
- ✅ Alert fatigue tracker integration
- ✅ User preference caching
- ✅ Concurrent Redis access
- ✅ Alert deduplication
- ✅ Session management
- ✅ Rate limiting logic

**Coverage**: All Redis-based components

### Notification Flow (notification_flow_test.go)

- ✅ End-to-end critical alert flow
- ✅ Quiet hours enforcement
- ✅ Alert fatigue rate limiting
- ✅ Multi-recipient notification
- ✅ Critical alert bypass of rate limits
- ✅ Alert routing through full pipeline
- ✅ User preference lookup
- ✅ Delivery service integration
- ✅ Performance benchmarks

**Coverage**: Complete notification pipeline

## Test Data Cleanup

All integration tests automatically clean up test data after execution:

- PostgreSQL: Test records are deleted from `user_preferences` table
- Redis: Test keys are deleted using pattern matching
- Kafka: Test topics are created with cleanup policy (if applicable)

### Manual Cleanup

If tests are interrupted and cleanup doesn't complete:

```bash
# Clean up PostgreSQL test data
psql -U cardiofit_user -d cardiofit_db -c "DELETE FROM notification_service.user_preferences WHERE user_id LIKE 'test_%' OR user_id LIKE 'benchmark_%';"

# Clean up Redis test keys
redis-cli KEYS "*test*" | xargs redis-cli DEL
redis-cli KEYS "*benchmark*" | xargs redis-cli DEL
```

## Benchmarks

Run performance benchmarks to measure throughput:

```bash
# Run all benchmarks
go test -v ./tests/integration/ -bench=. -benchmem

# Run specific benchmark
go test -v ./tests/integration/ -bench=BenchmarkNotificationFlow -benchmem

# Run with custom duration
go test -v ./tests/integration/ -bench=. -benchtime=30s
```

Example benchmark output:
```
BenchmarkNotificationFlow-8    5000    245670 ns/op    12864 B/op    142 allocs/op
```

This means:
- 5000 iterations completed
- Each notification flow takes ~246 microseconds
- Uses ~13KB of memory per operation
- 142 memory allocations per operation

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_USER: cardiofit_user
          POSTGRES_PASSWORD: cardiofit_pass
          POSTGRES_DB: cardiofit_db
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run migrations
        run: make migrate-up
        env:
          DATABASE_URL: postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable

      - name: Run integration tests
        run: go test -v ./tests/integration/... -cover
        env:
          DATABASE_URL: postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable
          REDIS_ADDR: localhost:6379
```

## Troubleshooting

### PostgreSQL Connection Issues

**Error**: `Failed to connect to database: connection refused`

**Solution**:
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check connection settings
psql -U cardiofit_user -d cardiofit_db -h localhost

# Verify DATABASE_URL environment variable
echo $DATABASE_URL
```

### Redis Connection Issues

**Error**: `Failed to connect to Redis: connection refused`

**Solution**:
```bash
# Check Redis is running
docker ps | grep redis

# Test Redis connection
redis-cli ping

# Verify REDIS_ADDR environment variable
echo $REDIS_ADDR
```

### Migration Issues

**Error**: `relation "notification_service.user_preferences" does not exist`

**Solution**:
```bash
# Run migrations
cd backend/services/notification-service
make migrate-up

# Or manually create schema
psql -U cardiofit_user -d cardiofit_db -f migrations/001_create_notification_schema.up.sql
```

### Test Data Conflicts

**Error**: `duplicate key value violates unique constraint`

**Solution**:
```bash
# Clean up test data manually
make test-cleanup

# Or remove specific test records
psql -U cardiofit_user -d cardiofit_db -c "DELETE FROM notification_service.user_preferences WHERE user_id LIKE 'test_%';"
```

## Code Examples

See [`examples/basic_usage_example.go`](./examples/basic_usage_example.go) for comprehensive usage examples:

- `ExampleBasicNotification()` - Send a simple notification
- `ExampleSetupUserPreferences()` - Configure user preferences
- `ExampleSendDirectNotification()` - Send direct SMS/email
- `ExampleCheckAlertFatigue()` - Use the fatigue tracker
- `ExampleGetUsersByRole()` - Query users by role
- `ExampleAlertWithEscalation()` - Create escalation policies

## Contributing

When adding new integration tests:

1. **Follow naming conventions**: `Test<Component>Integration`
2. **Use test cleanup**: Always clean up test data in teardown
3. **Check for infrastructure**: Use `testing.Short()` to skip when infrastructure unavailable
4. **Document requirements**: Update this README with any new prerequisites
5. **Add examples**: Include usage examples for new features

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Assertions](https://pkg.go.dev/github.com/stretchr/testify/assert)
- [pgx PostgreSQL Driver](https://github.com/jackc/pgx)
- [go-redis Client](https://github.com/redis/go-redis)
- [Notification Service Architecture](../../docs/architecture/notification-service.md)
