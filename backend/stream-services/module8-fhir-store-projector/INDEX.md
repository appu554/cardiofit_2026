# Module 8 FHIR Store Projector - Complete Index

## Quick Navigation

- **Getting Started**: [QUICK_START.md](QUICK_START.md)
- **Full Documentation**: [README.md](README.md)
- **Test Plan**: [FIRST_RESOURCE_WRITE_TEST.md](FIRST_RESOURCE_WRITE_TEST.md)
- **Delivery Report**: [DELIVERY_CONFIRMATION.md](DELIVERY_CONFIRMATION.md)
- **Summary**: [SUMMARY.txt](SUMMARY.txt)

## File Structure

```
module8-fhir-store-projector/
├── Documentation/
│   ├── README.md                          424 lines - Complete user guide
│   ├── DELIVERY_CONFIRMATION.md           650 lines - Implementation report
│   ├── QUICK_START.md                     275 lines - 30-second setup
│   ├── FIRST_RESOURCE_WRITE_TEST.md       260 lines - Testing guide
│   ├── SUMMARY.txt                         65 lines - Executive summary
│   └── INDEX.md                           This file
│
├── Core Service/
│   ├── app/
│   │   ├── __init__.py                      2 lines - Package init
│   │   ├── config.py                       84 lines - Configuration
│   │   ├── main.py                        139 lines - FastAPI health endpoints
│   │   └── services/
│   │       ├── __init__.py                  5 lines - Services package
│   │       ├── fhir_store_handler.py      391 lines - Google API handler
│   │       └── projector.py               251 lines - Kafka consumer
│   │
│   ├── run.py                             120 lines - Service launcher
│   └── credentials/
│       └── google-credentials.json       2.3 KB - Service account
│
├── Testing/
│   ├── test_projector.py                  391 lines - Integration tests
│   ├── validate_setup.py                  120 lines - Setup validation
│   ├── SAMPLE_RESOURCES.json               11 KB - Test resources (8 types)
│   └── tests/                                       Test directory
│
├── Configuration/
│   ├── requirements.txt                    18 lines - Dependencies
│   ├── .env.example                        45 lines - Config template
│   ├── .gitignore                          35 lines - Git exclusions
│   └── Dockerfile                          25 lines - Container image
│
└── Total: 20 files, ~2,100 lines of code and documentation
```

## Core Components

### 1. FHIR Store Handler (`app/services/fhir_store_handler.py`)

**Purpose**: Google Cloud Healthcare API integration

**Key Features**:
- Upsert logic (UPDATE → CREATE on 404)
- Retry with exponential backoff (3 attempts, 2x multiplier)
- Validation (resource type, ID, structure)
- Statistics tracking by resource type

**Public Methods**:
```python
FHIRStoreHandler(project_id, location, dataset_id, store_id, credentials_path)
- upsert_resource(fhir_resource_obj) -> Dict[str, Any]
- get_stats() -> Dict[str, Any]
- reset_stats() -> None
```

**Supported Resource Types** (8):
1. Observation
2. RiskAssessment
3. DiagnosticReport
4. Condition
5. MedicationRequest
6. Procedure
7. Encounter
8. Patient

### 2. FHIR Store Projector (`app/services/projector.py`)

**Purpose**: Kafka consumer for batch processing

**Key Features**:
- Extends KafkaConsumerBase from module8-shared
- Batch size: 20 (optimized for API rate limits)
- DLQ support for failed messages
- No transformation (resources pre-transformed by Module 6)

**Public Methods**:
```python
FHIRStoreProjector(config)
- get_projector_name() -> str
- process_batch(messages: List[Dict]) -> None
- get_processing_summary() -> Dict[str, Any]
- close() -> None
```

### 3. Configuration (`app/config.py`)

**Environment Groups**:
- Kafka: Bootstrap servers, credentials, consumer settings
- Google Cloud: Project, location, dataset, FHIR store, credentials
- Batch: Size (20), timeout (10s)
- API: Rate limiting, retry settings
- Service: Port (8056), log level

**Helper Methods**:
```python
Config.get_kafka_config() -> Dict[str, Any]
Config.get_fhir_store_path() -> str
```

### 4. Health Endpoints (`app/main.py`)

**FastAPI Routes**:
- `GET /` - Service info
- `GET /health` - Health status and basic stats
- `GET /metrics` - Detailed processing metrics
- `GET /stats/reset` - Reset statistics counters

## Testing Suite

### Validation Script (`validate_setup.py`)

**Tests**:
1. Configuration loading
2. Credentials file existence
3. Handler initialization
4. Sample resource validation
5. Module8-shared integration

**Usage**:
```bash
python3 validate_setup.py
```

### Integration Tests (`test_projector.py`)

**Test Scenarios**:
1. Handler direct test (CREATE + UPDATE)
2. Validation error handling
3. Projector batch processing

**Usage**:
```bash
python3 test_projector.py
```

### Sample Resources (`SAMPLE_RESOURCES.json`)

**Includes** (8 resources):
- Observation (Heart Rate)
- RiskAssessment (Sepsis)
- DiagnosticReport (Lab Panel)
- Condition (Hypertension)
- MedicationRequest (Lisinopril)
- Procedure (BP Measurement)
- Encounter (Emergency Visit)
- Patient (Demographics)

## Configuration Examples

### Minimal `.env`

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=your-cluster.confluent.cloud:9092
KAFKA_SASL_USERNAME=your-api-key
KAFKA_SASL_PASSWORD=your-api-secret

# Google Cloud
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_APPLICATION_CREDENTIALS=credentials/google-credentials.json
```

### Production `.env`

```bash
# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=pkc-prod.us-east-1.aws.confluent.cloud:9092
KAFKA_SASL_USERNAME=prod-api-key
KAFKA_SASL_PASSWORD=prod-api-secret
KAFKA_GROUP_ID=module8-fhir-store-projector-prod
KAFKA_TOPIC_FHIR_UPSERT=prod.ehr.fhir.upsert
KAFKA_TOPIC_DLQ=prod.ehr.dlq.fhir-store-projector

# Google Cloud
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=us-central1
GOOGLE_CLOUD_DATASET_ID=cardiofit_fhir_dataset
GOOGLE_CLOUD_FHIR_STORE_ID=cardiofit_fhir_store
GOOGLE_APPLICATION_CREDENTIALS=credentials/google-credentials.json

# Batch & Performance
BATCH_SIZE=20
BATCH_TIMEOUT_SECONDS=10
MAX_REQUESTS_PER_SECOND=200

# Retry Logic
RETRY_MAX_ATTEMPTS=3
RETRY_BACKOFF_FACTOR=2

# Service
SERVICE_PORT=8056
LOG_LEVEL=INFO
```

## Architecture Context

### Module 8 Hybrid Architecture

```
Stage 1 (Java)
    ↓
prod.ehr.events.validated
    ↓
Stage 2 (Python)
    ↓
prod.ehr.events.enriched
    ↓
[4 Core Projectors] → PostgreSQL, ClickHouse, Neo4j, MongoDB
    ↓
Module 6 (FHIR Transform)
    ↓
prod.ehr.fhir.upsert
    ↓
[FHIR Store Projector] ← This Service
    ↓
Google Cloud Healthcare API
    ↓
FHIR Store (HIPAA-compliant)
```

### Data Flow

1. **Module 6** transforms enriched events to FHIR R4 resources
2. **Kafka Topic** `prod.ehr.fhir.upsert` buffers resources (compacted, 365 days)
3. **FHIR Store Projector** consumes in batches of 20
4. **Validation** checks resource structure and type
5. **Upsert** tries UPDATE, falls back to CREATE on 404
6. **Retry** exponential backoff on transient errors (503, 504, 429, 500)
7. **DLQ** sends failed messages to `prod.ehr.dlq.fhir-store-projector`
8. **FHIR Store** persists resources in HIPAA-compliant storage

## Performance Specifications

| Metric | Specification |
|--------|---------------|
| Throughput | ~200 resources/sec (API limited) |
| Batch Size | 20 resources |
| Batch Timeout | 10 seconds |
| API Latency | 50-100ms per resource |
| Success Rate | >95% (target) |
| Memory Usage | ~256MB per instance |
| CPU Usage | Low (I/O bound) |

## Monitoring

### Health Check

```bash
curl http://localhost:8056/health
```

**Response**:
```json
{
  "status": "healthy",
  "stats": {
    "total_upserts": 1247,
    "successful_creates": 523,
    "successful_updates": 698,
    "success_rate": 0.979
  }
}
```

### Metrics

```bash
curl http://localhost:8056/metrics
```

**Key Metrics**:
- `total_processed`: Total messages consumed
- `successful_upserts`: Successful API writes
- `failed_upserts`: Failed writes (after retries)
- `validation_errors`: Invalid resource structures
- `resource_type_counts`: Breakdown by FHIR type

## Common Commands

### Development

```bash
# Setup
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services/module8-fhir-store-projector
pip install -r requirements.txt
cp .env.example .env

# Validate
python3 validate_setup.py

# Test
python3 test_projector.py

# Run
python3 run.py
```

### Production

```bash
# Docker build
docker build -t fhir-store-projector .

# Docker run
docker run -d \
  --name fhir-store-projector \
  -p 8056:8056 \
  -v $(pwd)/credentials:/app/credentials \
  --env-file .env \
  fhir-store-projector

# Check logs
docker logs -f fhir-store-projector

# Check health
curl http://localhost:8056/health
```

### Google Cloud Verification

```bash
# Search resources
gcloud healthcare fhir-stores search cardiofit_fhir_store \
  --dataset=cardiofit_fhir_dataset \
  --location=us-central1 \
  --resource-type=Observation \
  --search-string="subject=Patient/patient-12345"

# Get resource
gcloud healthcare fhir-stores get-resource cardiofit_fhir_store \
  --dataset=cardiofit_fhir_dataset \
  --location=us-central1 \
  --resource-type=Observation \
  --resource-id=obs-12345
```

## Dependencies

### Python Packages

```
kafka-python==2.0.2           # Kafka consumer
fastapi==0.109.0              # Web framework
uvicorn[standard]==0.27.0     # ASGI server
pydantic==2.5.3               # Data validation
google-cloud-healthcare==1.13.0  # FHIR Store API
google-auth==2.27.0           # Authentication
structlog==24.1.0             # Structured logging
prometheus-client==0.19.0     # Metrics
-e ../module8-shared          # Shared event models
```

### System Dependencies

- Python 3.11+
- gcc, g++ (for compilation)
- Network access to Confluent Cloud (Kafka)
- Network access to Google Cloud Healthcare API

## Security

### Authentication

- **Google Cloud**: Service account with OAuth 2.0
- **Kafka**: SASL_SSL with username/password
- **Credentials**: Stored in `credentials/google-credentials.json`

### Authorization

- **IAM Role**: `roles/healthcare.fhirResourceEditor`
- **Scope**: `https://www.googleapis.com/auth/cloud-healthcare`

### Encryption

- **In Transit**: TLS 1.2+ for all API calls
- **At Rest**: GCP default encryption for FHIR Store
- **HIPAA**: Compliance enabled on FHIR Store

## Troubleshooting

### Common Issues

1. **Credentials Error**
   - Check: `credentials/google-credentials.json` exists
   - Verify: JSON is valid
   - Test: `gcloud auth application-default login`

2. **Permission Denied**
   - Grant: `roles/healthcare.fhirResourceEditor`
   - Verify: IAM policy binding

3. **FHIR Store Not Found**
   - Create: FHIR Store in GCP Console
   - Verify: Store name and dataset match `.env`

4. **Validation Error**
   - Check: Resource type in SUPPORTED_RESOURCE_TYPES
   - Verify: resourceType matches fhirData.resourceType
   - Confirm: ID matches fhirData.id

## Support Resources

- **Module 8 Architecture**: `../MODULE_8_HYBRID_ARCHITECTURE_IMPLEMENTATION_PLAN.md`
- **Module8-Shared**: `../module8-shared/`
- **Google Healthcare API Docs**: https://cloud.google.com/healthcare-api/docs/how-tos/fhir
- **FHIR R4 Spec**: https://hl7.org/fhir/R4/

## Delivery Status

**Status**: ✅ COMPLETE

**Deliverables**:
- [x] Core service implementation (872 lines)
- [x] Google Cloud Healthcare API integration
- [x] Kafka consumer with batch processing
- [x] Health and metrics endpoints
- [x] Comprehensive testing suite
- [x] Complete documentation (1,600+ lines)
- [x] Docker support
- [x] Sample resources for all 8 types

**Ready For**:
- Dependency installation (`pip install -r requirements.txt`)
- Environment configuration (`.env` setup)
- Testing (`python3 test_projector.py`)
- Production deployment (`python3 run.py`)

---

**Created**: November 15, 2025
**Service**: Module 8 FHIR Store Projector
**Version**: 1.0.0
**Port**: 8056
