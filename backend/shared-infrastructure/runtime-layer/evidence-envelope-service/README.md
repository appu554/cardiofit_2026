# Evidence Envelope Service

A comprehensive clinical decision evidence management system that provides audit trails, confidence scoring, and regulatory compliance for healthcare AI applications.

## Features

- **Evidence Envelope Management**: Create and manage clinical decision evidence containers
- **Inference Chain Tracking**: Track complete reasoning chains with confidence scores
- **Regulatory Compliance**: HIPAA, FDA 21CFR11, and ISO 13485 compliant audit trails
- **Real-time Monitoring**: Prometheus metrics and structured logging
- **Cryptographic Integrity**: SHA-256 checksums for tamper detection
- **Multi-layer Caching**: Redis and in-memory caching for performance
- **Event Streaming**: Kafka integration for real-time audit events

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   FastAPI App   │    │  MongoDB Store   │    │  Redis Cache    │
│   (Port 8020)   │◄──►│  (Envelopes &    │◄──►│  (Performance   │
│                 │    │   Audit Records) │    │   Optimization) │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                                              │
         ▼                                              ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Kafka Events   │    │   Prometheus     │    │    Grafana      │
│  (Audit Trail)  │    │   (Metrics)      │    │ (Visualization) │
│                 │    │  (Port 9090)     │    │  (Port 3001)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Quick Start

### Using Docker Compose (Recommended)

1. **Start all services:**
   ```bash
   docker-compose -f docker-compose.evidence-envelope.yml up -d
   ```

2. **Check service health:**
   ```bash
   curl http://localhost:8020/health
   ```

3. **View API documentation:**
   - Swagger UI: http://localhost:8020/docs
   - ReDoc: http://localhost:8020/redoc

4. **Access monitoring:**
   - Grafana: http://localhost:3001 (admin/evidence-grafana-admin)
   - Prometheus: http://localhost:9090
   - Kafka UI: http://localhost:8082

### Local Development

1. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Set up environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start external services:**
   ```bash
   # MongoDB, Redis, Kafka need to be running
   docker-compose -f docker-compose.evidence-envelope.yml up mongodb redis kafka -d
   ```

4. **Run the service:**
   ```bash
   python -m uvicorn src.main:app --host 0.0.0.0 --port 8020 --reload
   ```

## API Endpoints

### Health & Monitoring
- `GET /health` - Basic health check
- `GET /health/ready` - Readiness check (dependencies)
- `GET /metrics` - Prometheus metrics

### Evidence Envelopes
- `POST /envelopes` - Create new evidence envelope
- `GET /envelopes/{envelope_id}` - Get envelope by ID
- `GET /envelopes` - Query envelopes with filters
- `POST /envelopes/{envelope_id}/inference-steps` - Add inference step
- `POST /envelopes/{envelope_id}/finalize` - Finalize envelope

### Audit & Compliance
- `GET /envelopes/{envelope_id}/audit-trail` - Complete audit trail
- `GET /envelopes/{envelope_id}/integrity` - Verify envelope integrity
- `POST /envelopes/{envelope_id}/wrap-response` - Wrap service responses

## Usage Examples

### Create Evidence Envelope

```bash
curl -X POST "http://localhost:8020/envelopes" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "proposal_id": "proposal-12345",
    "snapshot_id": "snapshot-67890",
    "knowledge_versions": {
      "clinical_guidelines": "v2.1.0",
      "drug_database": "v1.5.3"
    },
    "clinical_context": {
      "patient_id": "patient-abc123",
      "encounter_id": "encounter-def456",
      "workflow_type": "medication_review",
      "clinical_domain": "cardiology",
      "urgency_level": "routine"
    }
  }'
```

### Add Inference Step

```bash
curl -X POST "http://localhost:8020/envelopes/{envelope_id}/inference-steps" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "step_type": "drug_interaction_check",
    "description": "Check for adverse drug interactions",
    "source_data": {
      "current_medications": ["medication1", "medication2"],
      "proposed_medication": "new_medication"
    },
    "reasoning_logic": "Checked interaction database for contraindications",
    "result_data": {
      "interactions_found": false,
      "safety_score": 0.95
    },
    "confidence": 0.95,
    "execution_time_ms": 150,
    "knowledge_sources": ["drug_interaction_db_v2.1"]
  }'
```

### Query Envelopes

```bash
curl "http://localhost:8020/envelopes?patient_id=patient-123&status=finalized&limit=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Configuration

Key configuration options in `.env`:

| Setting | Default | Description |
|---------|---------|-------------|
| `PORT` | 8020 | Service port |
| `MONGODB_CONNECTION_STRING` | mongodb://localhost:27017 | MongoDB connection |
| `REDIS_HOST` | localhost | Redis host |
| `KAFKA_BOOTSTRAP_SERVERS` | localhost:9092 | Kafka brokers |
| `JWT_SECRET_KEY` | (required) | JWT signing key |
| `ENVELOPE_CACHE_SIZE` | 1000 | In-memory cache size |
| `ENVELOPE_CACHE_TTL` | 300 | Cache TTL in seconds |
| `AUDIT_RETENTION_DAYS` | 2555 | Data retention (7 years) |

## Monitoring

### Prometheus Metrics

The service exposes comprehensive metrics:

- `evidence_envelope_created_total` - Total envelopes created
- `evidence_envelope_finalized_total` - Total envelopes finalized
- `evidence_envelope_inference_steps_total` - Total inference steps
- `evidence_envelope_creation_duration_seconds` - Creation time histogram
- `evidence_envelope_confidence_score` - Current confidence scores

### Structured Logging

All logs are structured JSON with correlation IDs:

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "event": "envelope_created",
  "envelope_id": "env_123456789",
  "proposal_id": "proposal-12345",
  "duration_ms": 45
}
```

## Security

### Authentication

All endpoints (except `/health` and `/metrics`) require JWT authentication:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Data Integrity

- All envelopes include SHA-256 checksums
- Cryptographic verification prevents tampering
- Immutable audit trails for compliance

### CORS Configuration

Configure allowed origins via `ALLOWED_ORIGINS` environment variable.

## Compliance

### HIPAA Compliance
- Comprehensive audit logs
- Data encryption in transit and at rest
- Access controls and authentication
- 7-year data retention policy

### FDA 21CFR Part 11
- Electronic signatures via JWT tokens
- Audit trail completeness
- Data integrity verification
- Secure timestamps

## Development

### Running Tests

```bash
pytest tests/ -v --cov=src
```

### Code Quality

```bash
# Formatting
black src/ tests/
isort src/ tests/

# Linting
flake8 src/ tests/
mypy src/
```

### Database Migrations

MongoDB collections and indexes are automatically created via the initialization script in `scripts/mongo-init.js`.

## Troubleshooting

### Common Issues

1. **Service won't start**
   - Check MongoDB, Redis, and Kafka are running
   - Verify environment variables are set correctly
   - Check logs: `docker-compose logs evidence-envelope-service`

2. **Authentication errors**
   - Verify JWT_SECRET_KEY is configured
   - Check token expiration and format

3. **Database connection issues**
   - Verify MongoDB connection string
   - Check network connectivity
   - Ensure database user permissions

4. **Performance issues**
   - Monitor Redis cache hit rates
   - Check MongoDB indexes
   - Review Prometheus metrics

### Health Check Endpoints

```bash
# Basic health
curl http://localhost:8020/health

# Readiness (dependencies)
curl http://localhost:8020/health/ready
```

## License

Proprietary - CardioFit Clinical Synthesis Hub