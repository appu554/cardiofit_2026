# KB-2 Clinical Context Service - Quick Start Guide

## Summary

**KB-2 Clinical Context** is a Go-based microservice (port 8082) that provides clinical context aggregation, phenotype detection, risk assessment, and care gap identification using MongoDB and Redis.

---

## Prerequisites

- **Go 1.24.0** or compatible
- **MongoDB** (port 27017)
- **Redis** (port 6380 or 6379)
- **Docker** (optional, recommended)

---

## Quick Start (3 Options)

### Option 1: Docker (Recommended - Easiest)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services
make run-kb-docker
```

This starts all infrastructure (MongoDB, Redis) and KB services automatically.

**Verify it's running:**
```bash
curl http://localhost:8082/health
```

---

### Option 2: Using Start Script

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-2-clinical-context
./start-service.sh
```

The script will:
- Check if MongoDB and Redis are running
- Build the binary if needed
- Configure environment variables
- Start the service

**Note**: MongoDB and Redis must be running first.

---

### Option 3: Manual Start

```bash
# 1. Build the service
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-2-clinical-context
go build -o bin/kb-2-clinical-context

# 2. Start MongoDB (if not running)
docker run -d -p 27017:27017 --name kb2-mongo mongo:latest

# 3. Start Redis (if not running)
docker run -d -p 6380:6379 --name kb2-redis redis:latest

# 4. Create .env file (optional)
cat > .env <<EOF
PORT=8082
ENVIRONMENT=development
DEBUG=true
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=clinical_context
REDIS_URL=localhost:6380
METRICS_ENABLED=true
EOF

# 5. Run the service
./bin/kb-2-clinical-context
```

---

## Verify Service is Running

### 1. Health Check
```bash
curl http://localhost:8082/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-20T09:41:00Z",
  "service": "kb-2-clinical-context",
  "version": "1.0.0",
  "checks": {
    "database": {"status": "healthy"},
    "cache": {"status": "healthy"}
  }
}
```

### 2. Metrics Check
```bash
curl http://localhost:8082/metrics
```

### 3. GraphQL Introspection
```bash
curl http://localhost:8082/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __schema { types { name } } }"}'
```

---

## Common Issues

### Issue: MongoDB Connection Failed
**Symptom**: Service logs show "Failed to connect to MongoDB"

**Solutions:**
```bash
# Check if MongoDB is running
docker ps | grep mongo

# Start MongoDB
docker run -d -p 27017:27017 --name kb2-mongo mongo:latest

# Or set custom URI
export MONGODB_URI=mongodb://your-mongodb-host:27017
```

---

### Issue: Redis Connection Failed
**Symptom**: Service logs show "Failed to connect to cache"

**Solutions:**
```bash
# Check if Redis is running
docker ps | grep redis

# Start Redis on port 6380
docker run -d -p 6380:6379 --name kb2-redis redis:latest

# Or use existing Redis on 6379
export REDIS_URL=localhost:6379
```

---

### Issue: Port 8082 Already in Use
**Symptom**: "bind: address already in use"

**Solutions:**
```bash
# Find what's using port 8082
lsof -i :8082

# Use different port
export PORT=8083
./bin/kb-2-clinical-context

# Or kill the conflicting process
kill -9 <PID>
```

---

## Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8082 | HTTP server port |
| `ENVIRONMENT` | development | Environment mode |
| `DEBUG` | true | Enable debug logging |
| `MONGODB_URI` | mongodb://localhost:27017 | MongoDB connection string |
| `MONGODB_DATABASE` | clinical_context | MongoDB database name |
| `REDIS_URL` | localhost:6380 | Redis server address |
| `METRICS_ENABLED` | true | Enable Prometheus metrics |
| `L1_CACHE_MAX_SIZE` | 10000 | In-memory cache max entries |
| `L1_CACHE_TTL` | 5m | In-memory cache TTL |

---

## API Endpoints

### Health & Monitoring
- `GET /health` - Service health check
- `GET /metrics` - Prometheus metrics

### GraphQL Federation
- `POST /api/federation` - GraphQL Federation endpoint
- `POST /graphql` - Direct GraphQL access

### Context Management
- `POST /api/v1/context/build` - Build clinical context
- `GET /api/v1/context/{patient_id}/history` - Context history
- `GET /api/v1/context/statistics` - Context statistics

### Phenotype Detection
- `POST /api/v1/phenotypes/detect` - Detect phenotypes
- `GET /api/v1/phenotypes/definitions` - Get definitions
- `GET /api/v1/phenotypes/validate` - Validate phenotypes
- `POST /api/v1/phenotypes/test` - Test expressions

### Risk Assessment
- `POST /api/v1/risk/assess` - Assess patient risk

### Care Gaps
- `GET /api/v1/care-gaps/{patient_id}` - Identify care gaps

### Administration
- `GET /api/v1/admin/health` - System health
- `POST /api/v1/admin/cache/clear` - Clear cache

---

## Testing

### Run All Tests
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-2-clinical-context
go test ./...
```

### Run Specific Test Types
```bash
# Unit tests only
go test ./tests/unit/...

# Integration tests (requires running services)
go test -tags=integration ./tests/integration/...

# Performance tests
go test ./tests/performance/...

# Clinical scenario tests
go test ./tests/clinical/...
```

---

## Stopping the Service

### If using Docker
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services
make stop-kb
```

### If running manually
```bash
# Find the process
ps aux | grep kb-2-clinical-context

# Kill it
kill <PID>

# Or use Ctrl+C in the terminal where it's running
```

---

## Development Workflow

1. **Make code changes**
2. **Rebuild**: `go build -o bin/kb-2-clinical-context`
3. **Restart service**: `./bin/kb-2-clinical-context`
4. **Test changes**: `curl http://localhost:8082/health`
5. **Run tests**: `go test ./...`

---

## Files Created

### Verification Report
`TEST_VERIFICATION.md` - Comprehensive service verification and documentation

### Startup Script
`start-service.sh` - Automated startup script with dependency checks

### Binary
`bin/kb-2-clinical-context` - Compiled 29 MB Go binary

---

## Next Steps

1. ✅ **Service Built**: Binary created successfully
2. 🚀 **Start Infrastructure**: Run MongoDB and Redis
3. ▶️  **Start Service**: Use one of the three options above
4. ✅ **Verify Health**: Check `/health` endpoint
5. 🧪 **Run Tests**: Validate functionality with `go test ./...`
6. 📊 **Monitor Metrics**: Check `/metrics` for Prometheus data

---

## Support & Documentation

- **Full Documentation**: See `TEST_VERIFICATION.md`
- **Service Architecture**: See `CLAUDE.md` in knowledge-base-services directory
- **API Documentation**: Available at startup in service logs
- **Test Suite**: Located in `tests/` directory
- **Configuration**: See `internal/config/config.go`

---

## Service Architecture

```
KB-2 Clinical Context Service (Port 8082)
│
├── MongoDB (Port 27017)
│   └── Database: clinical_context
│       ├── phenotype_definitions
│       └── patient_contexts
│
├── Redis (Port 6380, DB 2)
│   └── Multi-tier Cache
│       ├── L1: In-memory (10K entries, 5m TTL)
│       ├── L2: Redis (1h TTL)
│       └── L3: CDN (24h TTL)
│
└── Integrations
    ├── Apollo Federation (GraphQL)
    ├── Prometheus (Metrics)
    └── CEL Rule Engine (Phenotypes)
```

---

**Last Updated**: 2025-11-20 09:41:00 UTC
**Service Version**: 1.0.0
**Go Version**: 1.24.0
**Status**: Build Successful ✅
