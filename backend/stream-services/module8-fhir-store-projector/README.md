# Module 8 FHIR Store Projector

Consumes FHIR resources from `prod.ehr.fhir.upsert` and writes to Google Cloud Healthcare API FHIR Store.

## Overview

The FHIR Store Projector is the final storage layer in Module 8's hybrid architecture, persisting FHIR-compliant clinical data to Google Cloud Healthcare API for long-term storage, interoperability, and HIPAA compliance.

### Key Characteristics

- **Input Topic**: `prod.ehr.fhir.upsert` (compacted, 12 partitions, 365-day retention)
- **Input Format**: `FHIRResource` objects (pre-transformed by Module 6)
- **NO TRANSFORMATION**: Resources are already FHIR R4 compliant
- **Output**: Google Cloud Healthcare FHIR Store
- **Batch Size**: 20 (small due to API rate limits)
- **Throughput**: ~200 resources/sec (API limited)
- **API Latency**: 50-100ms per resource

## Architecture

```
prod.ehr.fhir.upsert
       ↓
[FHIR Store Projector]
       ↓
[Validation & Parsing]
       ↓
[Google Healthcare API]
  - Try UPDATE first
  - If 404, CREATE new
  - Retry on errors
       ↓
[FHIR Store] (HIPAA-compliant storage)
```

## Supported Resource Types

- **Observation**: Vital signs, lab results
- **RiskAssessment**: ML predictions, clinical risk scores
- **DiagnosticReport**: Test results, imaging reports
- **Condition**: Diagnoses, problems
- **MedicationRequest**: Medication orders
- **Procedure**: Medical procedures
- **Encounter**: Clinical encounters
- **Patient**: Patient demographics (reference)

## Installation

```bash
cd module8-fhir-store-projector

# Install dependencies
pip install -r requirements.txt

# Copy credentials
cp /path/to/google-credentials.json credentials/

# Configure environment
cp .env.example .env
# Edit .env with your Kafka and GCP settings
```

## Configuration

### Environment Variables

```bash
# Kafka (Confluent Cloud)
KAFKA_BOOTSTRAP_SERVERS=your-cluster.confluent.cloud:9092
KAFKA_SASL_USERNAME=your-api-key
KAFKA_SASL_PASSWORD=your-api-secret

# Consumer Settings
KAFKA_GROUP_ID=module8-fhir-store-projector
BATCH_SIZE=20
BATCH_TIMEOUT_SECONDS=10

# Google Cloud Healthcare API
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=us-central1
GOOGLE_CLOUD_DATASET_ID=cardiofit_fhir_dataset
GOOGLE_CLOUD_FHIR_STORE_ID=cardiofit_fhir_store
GOOGLE_APPLICATION_CREDENTIALS=credentials/google-credentials.json

# API Rate Limiting
MAX_REQUESTS_PER_SECOND=200
RETRY_MAX_ATTEMPTS=3
RETRY_BACKOFF_FACTOR=2
```

### Google Cloud Setup

1. **Create Healthcare Dataset**:
   ```bash
   gcloud healthcare datasets create cardiofit_fhir_dataset \
     --location=us-central1 \
     --project=cardiofit-905a8
   ```

2. **Create FHIR Store**:
   ```bash
   gcloud healthcare fhir-stores create cardiofit_fhir_store \
     --dataset=cardiofit_fhir_dataset \
     --location=us-central1 \
     --version=R4 \
     --enable-update-create
   ```

3. **Grant Service Account Permissions**:
   ```bash
   gcloud healthcare fhir-stores add-iam-policy-binding cardiofit_fhir_store \
     --dataset=cardiofit_fhir_dataset \
     --location=us-central1 \
     --member="serviceAccount:healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com" \
     --role="roles/healthcare.fhirResourceEditor"
   ```

## Running the Service

### Development

```bash
# Run with Python
python run.py

# Service will start on port 8056
# Kafka consumer runs in background thread
```

### Production

```bash
# Use uvicorn with workers
uvicorn app.main:app --host 0.0.0.0 --port 8056 --workers 4
```

### Docker

```bash
# Build image
docker build -t fhir-store-projector .

# Run container
docker run -d \
  --name fhir-store-projector \
  -p 8056:8056 \
  -v $(pwd)/credentials:/app/credentials \
  --env-file .env \
  fhir-store-projector
```

## API Endpoints

### Health Check
```bash
curl http://localhost:8056/health
```

Response:
```json
{
  "status": "healthy",
  "service": "fhir-store-projector",
  "fhir_store": {
    "project_id": "cardiofit-905a8",
    "location": "us-central1",
    "dataset_id": "cardiofit_fhir_dataset",
    "store_id": "cardiofit_fhir_store"
  },
  "stats": {
    "total_upserts": 1247,
    "successful_creates": 523,
    "successful_updates": 698,
    "failed_upserts": 26,
    "success_rate": 0.979
  }
}
```

### Detailed Metrics
```bash
curl http://localhost:8056/metrics
```

Response:
```json
{
  "projector_stats": {
    "total_processed": 1247,
    "successful_upserts": 1221,
    "failed_upserts": 26,
    "validation_errors": 3,
    "resource_type_counts": {
      "Observation": 645,
      "RiskAssessment": 234,
      "Condition": 189,
      "DiagnosticReport": 123,
      "MedicationRequest": 56
    }
  },
  "handler_stats": {
    "total_upserts": 1247,
    "successful_creates": 523,
    "successful_updates": 698,
    "failed_upserts": 26,
    "validation_errors": 3,
    "success_rate": 0.979
  }
}
```

## Testing

### Unit Tests

```bash
# Run test suite
python test_projector.py
```

Tests verify:
1. FHIR resource upsert (CREATE and UPDATE)
2. Validation error handling
3. Batch processing
4. Resource type tracking

### Manual Testing

```bash
# Test with sample Observation
curl -X POST http://localhost:8056/test/observation

# Test with sample RiskAssessment
curl -X POST http://localhost:8056/test/risk-assessment

# Verify in FHIR Store
gcloud healthcare fhir-stores search cardiofit_fhir_store \
  --dataset=cardiofit_fhir_dataset \
  --location=us-central1 \
  --resource-type=Observation \
  --search-string="subject=Patient/patient-12345"
```

## Error Handling

### Dead Letter Queue (DLQ)

Failed messages are sent to `prod.ehr.dlq.fhir-store-projector`:

```json
{
  "original_message": {
    "resourceType": "Observation",
    "resourceId": "obs-123",
    "fhirData": {...}
  },
  "error": "Validation error: resourceType mismatch",
  "projector": "fhir-store-projector"
}
```

### Retry Strategy

1. **Exponential Backoff**: 1s, 2s, 4s (configurable)
2. **Retryable Errors**:
   - ServiceUnavailable (503)
   - DeadlineExceeded (504)
   - ResourceExhausted (429)
   - InternalServerError (500)
3. **Non-Retryable Errors**:
   - ValidationError (400)
   - PermissionDenied (403)
   - InvalidArgument (400)

## Performance Tuning

### Batch Size Optimization

```bash
# Small batches for low latency
BATCH_SIZE=10
BATCH_TIMEOUT_SECONDS=5

# Larger batches for throughput
BATCH_SIZE=50
BATCH_TIMEOUT_SECONDS=15
```

**Recommendation**: Keep BATCH_SIZE ≤ 20 due to API rate limits.

### API Rate Limiting

Google Healthcare API has rate limits:
- **Read**: 5,000 requests/minute
- **Write**: 1,000 requests/minute

Monitor with:
```bash
curl http://localhost:8056/metrics | jq '.handler_stats.api_errors'
```

### Parallelization

Run multiple projector instances with different consumer group IDs:

```bash
# Instance 1
KAFKA_GROUP_ID=fhir-store-projector-1 python run.py

# Instance 2
KAFKA_GROUP_ID=fhir-store-projector-2 python run.py
```

Kafka will distribute partitions across instances.

## Monitoring

### Prometheus Metrics

Exposed on `/metrics` endpoint:

- `fhir_store_upserts_total`: Total upsert operations
- `fhir_store_creates_total`: Successful CREATE operations
- `fhir_store_updates_total`: Successful UPDATE operations
- `fhir_store_errors_total`: Failed operations
- `fhir_store_validation_errors_total`: Validation failures
- `fhir_store_api_latency_seconds`: API request duration

### Logging

Structured JSON logs:

```json
{
  "event": "FHIR resource upserted",
  "resource_type": "Observation",
  "resource_id": "obs-12345",
  "operation": "UPDATE",
  "timestamp": "2025-11-15T20:45:32Z",
  "level": "info"
}
```

## Troubleshooting

### Connection Issues

```bash
# Test GCP credentials
gcloud auth application-default login
gcloud healthcare fhir-stores describe cardiofit_fhir_store \
  --dataset=cardiofit_fhir_dataset \
  --location=us-central1
```

### Validation Errors

Check DLQ topic:
```bash
kafka-console-consumer \
  --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS \
  --topic prod.ehr.dlq.fhir-store-projector \
  --from-beginning
```

### Performance Issues

```bash
# Check API latency
curl http://localhost:8056/metrics | jq '.handler_stats |
  {total: .total_upserts, success_rate: .success_rate}'

# Monitor Kafka lag
kafka-consumer-groups \
  --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS \
  --group module8-fhir-store-projector \
  --describe
```

## Architecture Integration

### Module 8 Hybrid Architecture

```
Stage 1 (Java) → Stage 2 (Python) → Module 6 (FHIR Transform)
                                              ↓
                                    prod.ehr.fhir.upsert
                                              ↓
                                    [FHIR Store Projector]
                                              ↓
                                    Google Healthcare FHIR Store
```

### Data Flow

1. **Module 6** transforms enriched events to FHIR R4
2. **Kafka Topic** `prod.ehr.fhir.upsert` buffers resources
3. **FHIR Store Projector** consumes and validates
4. **Google Healthcare API** persists to FHIR Store
5. **HIPAA-compliant storage** for long-term retention

## Security

### Authentication

- Service account credentials (OAuth 2.0)
- Scope: `https://www.googleapis.com/auth/cloud-healthcare`
- Credentials stored in `credentials/google-credentials.json`

### Authorization

- IAM role: `roles/healthcare.fhirResourceEditor`
- Principle of least privilege
- Audit logs enabled

### Encryption

- TLS 1.2+ for API calls
- Data encrypted at rest (GCP default)
- HIPAA compliance enabled on FHIR Store

## License

Copyright 2025 CardioFit Platform. All rights reserved.
