# KB-2 Clinical Context Service

A high-performance Go microservice for clinical phenotyping and risk stratification, designed to transform basic patient data into actionable clinical intelligence.

## 🎯 Overview

The KB-2 Clinical Context Service is a production-ready clinical phenotyping and risk stratification engine that provides:

- **Batch Phenotype Evaluation**: Process up to 1,000 patients using CEL-based clinical rules
- **Enhanced Risk Assessment**: Multi-category risk analysis (cardiovascular, diabetes, medication, fall, bleeding)
- **Treatment Preferences**: Institutional rule-based treatment recommendations with conflict resolution
- **Complete Context Assembly**: Unified clinical intelligence combining all assessment types
- **Performance SLAs**: Sub-200ms processing with comprehensive metrics

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- MongoDB 4.4+
- Redis 6.0+
- Docker & Docker Compose (optional)

### Installation

```bash
# Clone and enter the service directory
cd kb-2-clinical-context-go

# Install dependencies
make deps

# Build the service
make build

# Start dependencies (MongoDB + Redis)
make start-deps

# Run the service
make run

# Check health
make health
```

### Using Docker

```bash
# Build Docker image
make docker-build

# Run with Docker
make docker-run

# Check logs
make docker-logs
```

## 📊 API Endpoints

### Core Endpoints

| Endpoint | Method | SLA | Description |
|----------|--------|-----|-------------|
| `/v1/phenotypes/evaluate` | POST | 100ms | Batch phenotype evaluation |
| `/v1/phenotypes/explain` | POST | 150ms | Phenotype reasoning explanation |
| `/v1/risk/assess` | POST | 200ms | Enhanced risk assessment |
| `/v1/treatment/preferences` | POST | 50ms | Treatment recommendations |
| `/v1/context/assemble` | POST | 200ms | Complete context assembly |

### Supporting Endpoints

- `GET /health` - Service health check
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics
- `GET /v1/docs` - API documentation
- `GET /v1/phenotypes` - Available phenotypes
- `GET /v1/context/history/{patient_id}` - Patient context history

## 🧬 Clinical Intelligence Features

### Phenotype Evaluation

Uses CEL (Common Expression Language) for flexible, maintainable clinical rules:

```cel
// Cardiovascular risk phenotype
age >= 65 && 
(has_condition('hypertension') || has_condition('diabetes')) &&
lab_value('total_cholesterol') > 200
```

**Supported Phenotypes:**
- High Cardiovascular Risk
- Heart Failure Risk
- Diabetes Complications
- Medication Interaction Risk
- Fall Risk (elderly patients)

### Risk Assessment

Multi-category risk scoring across:

- **Cardiovascular**: Framingham-based with modern enhancements
- **Diabetes**: Complications risk with glycemic control factors
- **Medication**: Polypharmacy and interaction analysis
- **Fall**: Elderly-specific multi-factorial assessment
- **Bleeding**: HAS-BLED based for anticoagulated patients

### Treatment Preferences

Institutional rule-based recommendations with:

- **Guideline Compliance**: ADA/EASD 2023, ACC/AHA 2017
- **Cost Effectiveness**: Formulary preferences and generic substitution
- **Conflict Resolution**: Priority-based resolution of competing recommendations
- **Patient Preferences**: Dosing frequency, route preferences, cost considerations

## 🏗️ Architecture

### Service Structure

```
kb-2-clinical-context-go/
├── cmd/server/             # Application entry point
├── internal/
│   ├── api/               # HTTP handlers and routing
│   ├── config/            # Configuration management
│   ├── metrics/           # Prometheus metrics
│   ├── models/            # Data models
│   └── services/          # Business logic
│       ├── context_service.go      # Context assembly
│       ├── phenotype_engine.go     # CEL-based evaluation
│       ├── risk_assessment_service.go
│       └── treatment_preference_service.go
├── knowledge-base/        # Clinical knowledge definitions
│   ├── phenotypes/        # CEL-based phenotype rules
│   ├── risk-models/       # Risk calculation models
│   └── treatment-preferences/  # Institutional preferences
├── api/                   # OpenAPI specification
└── Dockerfile            # Container definition
```

### Technology Stack

- **Language**: Go 1.21
- **Framework**: Gin (HTTP routing)
- **Rule Engine**: Google CEL (Common Expression Language)
- **Databases**: MongoDB (primary), Redis (caching)
- **Metrics**: Prometheus
- **Documentation**: OpenAPI 3.0

## 🔧 Configuration

### Environment Variables

```bash
# Service Configuration
PORT=8088
ENVIRONMENT=development

# Database URLs
DATABASE_URL=mongodb://localhost:27017
DATABASE_NAME=kb_clinical_context
REDIS_URL=localhost:6379

# Performance Settings
BATCH_SIZE=1000
MAX_CONCURRENT_REQUESTS=100
CACHE_TIMEOUT=3600

# SLA Thresholds (milliseconds)
PHENOTYPE_EVALUATION_SLA=100
PHENOTYPE_EXPLANATION_SLA=150
RISK_ASSESSMENT_SLA=200
TREATMENT_PREFERENCES_SLA=50
CONTEXT_ASSEMBLY_SLA=200

# Feature Flags
ENABLE_CACHING=true
ENABLE_METRICS=true
STRICT_VALIDATION=true
```

## 📈 Performance & Monitoring

### Performance Targets

- **Phenotype Evaluation**: 100ms p95 (batch of 100 patients)
- **Risk Assessment**: 200ms p95 (comprehensive analysis)
- **Treatment Preferences**: 50ms p95 (single condition)
- **Context Assembly**: 200ms p95 (all components)
- **Throughput**: 10,000+ requests/second
- **Cache Hit Rate**: >95%

### Metrics Available

```bash
# View metrics
curl http://localhost:8088/metrics

# Key metrics include:
kb2_requests_total                    # Total HTTP requests
kb2_request_duration_seconds         # Request latency histograms
kb2_phenotype_evaluations_total      # Phenotype evaluations
kb2_risk_assessments_total          # Risk assessments
kb2_cache_hits_total                 # Cache performance
kb2_sla_violations_total             # SLA compliance
```

### Health Checks

```bash
# Basic health check
curl http://localhost:8088/health

# Readiness check
curl http://localhost:8088/ready
```

## 🧪 Testing

### API Testing

```bash
# Test all endpoints
make test-api

# Individual endpoint tests
make test-phenotypes    # Phenotype evaluation
make test-risk         # Risk assessment  
make test-treatment    # Treatment preferences
make test-context      # Context assembly
```

### Performance Testing

```bash
# Load testing
make load-test

# Stress testing
make stress-test

# Full test suite
make full-test
```

### Example API Calls

#### Phenotype Evaluation

```bash
curl -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patients": [{
      "id": "patient-123",
      "age": 65,
      "gender": "male",
      "conditions": ["diabetes", "hypertension"],
      "labs": {
        "hba1c": {"value": 8.5, "unit": "%"},
        "total_cholesterol": {"value": 250, "unit": "mg/dL"}
      }
    }],
    "include_explanation": false
  }'
```

#### Risk Assessment

```bash
curl -X POST http://localhost:8088/v1/risk/assess \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "patient_data": {
      "id": "patient-123",
      "age": 65,
      "gender": "male",
      "conditions": ["diabetes", "hypertension"]
    },
    "risk_categories": ["cardiovascular", "diabetes"],
    "include_factors": true
  }'
```

#### Treatment Preferences

```bash
curl -X POST http://localhost:8088/v1/treatment/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "condition": "diabetes",
    "patient_data": {
      "id": "patient-123", 
      "age": 65,
      "gender": "male",
      "conditions": ["diabetes"]
    },
    "preference_profile": {
      "once_daily": true,
      "injectable": false,
      "cost_conscious": true
    }
  }'
```

#### Complete Context Assembly

```bash
curl -X POST http://localhost:8088/v1/context/assemble \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "patient_data": {
      "id": "patient-123",
      "age": 65,
      "gender": "male", 
      "conditions": ["diabetes", "hypertension"]
    },
    "detail_level": "comprehensive",
    "include_phenotypes": true,
    "include_risks": true,
    "include_treatments": true
  }'
```

## 📚 Knowledge Base

### Phenotype Definitions

Located in `knowledge-base/phenotypes/`, using CEL syntax:

- `cardiovascular.yaml` - Cardiovascular risk phenotypes
- `diabetes.yaml` - Diabetes complication phenotypes

### Risk Models

Defined in `knowledge-base/risk-models/risk_stratification.yaml`:

- Cardiovascular risk (Framingham-based)
- Diabetes complications
- Medication interactions
- Fall risk (elderly)
- Bleeding risk (HAS-BLED)

### Treatment Preferences

Institutional rules in `knowledge-base/treatment-preferences/institutional_preferences.yaml`:

- ADA/EASD diabetes guidelines
- ACC/AHA hypertension guidelines
- Cost-effectiveness rules
- Conflict resolution hierarchy

## 🛠️ Development

### Development Workflow

```bash
# Start development mode with hot reload
make dev

# Code quality checks
make lint
make format
make vet

# Security scanning
make security-scan

# Full quality pipeline
make all
```

### Adding New Phenotypes

1. Define phenotype in YAML:

```yaml
# knowledge-base/phenotypes/my_condition.yaml
phenotypes:
  - id: "my_new_phenotype"
    name: "My New Phenotype"
    category: "custom"
    cel_rule: "age > 50 && has_condition('my_condition')"
```

2. Load phenotypes in service initialization
3. Test with API endpoints

### Extending Risk Models

1. Add model definition in `risk_stratification.yaml`
2. Implement model interface in `risk_assessment_service.go`
3. Add to model registry
4. Test with risk assessment endpoints

## 🚀 Deployment

### Docker Deployment

```bash
# Build production image
make docker-build

# Run with production settings
docker run -d \
  --name kb2-clinical-context \
  -p 8088:8088 \
  -e ENVIRONMENT=production \
  -e DATABASE_URL=mongodb://prod-mongo:27017 \
  -e REDIS_URL=prod-redis:6379 \
  kb2-clinical-context:latest
```

### Kubernetes

See example Kubernetes manifests in the knowledge-base-services documentation.

### Environment-Specific Configuration

- **Development**: `make run` with local dependencies
- **Testing**: `make test-api` with test fixtures
- **Production**: Docker deployment with external databases

## 🔒 Security & Compliance

### Security Features

- Input validation on all endpoints
- Rate limiting and request size limits
- Secure dependency management
- Container security scanning
- RFC 7807 compliant error responses

### HIPAA Compliance Considerations

- No patient data stored permanently
- All processing in-memory
- Audit logging for all operations
- Encryption in transit
- Access controls via external authentication

## 📖 API Documentation

- **OpenAPI Spec**: `api/openapi.yaml`
- **Interactive Docs**: `http://localhost:8088/v1/docs`
- **Swagger UI**: `make swagger-ui`

## 🤝 Integration

### Flow2 Orchestrator Integration

```go
// Example integration with Flow2
client := &http.Client{Timeout: 5 * time.Second}

// Context assembly for medication decisions
response, err := client.Post(
    "http://kb2:8088/v1/context/assemble",
    "application/json",
    bytes.NewReader(contextRequest),
)
```

### Clinical Assertion Engine Integration

The service provides structured clinical context for CAE decision-making:

- Phenotype results for condition-specific protocols
- Risk scores for safety thresholds
- Treatment preferences for recommendation ranking

### Safety Gateway Integration

Risk assessment results integrate with safety checks:

- High-risk patient identification
- Medication interaction warnings
- Contraindication detection

## 📞 Support

For issues, questions, or contributions:

- **Documentation**: This README and OpenAPI specification
- **Health Endpoint**: `GET /health` for service status
- **Metrics**: `GET /metrics` for operational insights
- **Logs**: Use `make logs-deps` and `make docker-logs`

---

**KB-2 Clinical Context Service** - Transforming patient data into clinical intelligence with production-grade performance and clinical accuracy. 🏥⚡