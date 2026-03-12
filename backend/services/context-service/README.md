# Clinical Context Service

**Federated Clinical Data Intelligence Hub**  
*Implementing the Three Pillars of Excellence*

## Overview

The Clinical Context Service is a production-grade microservice that implements the **Clinical Context Recipe System** - a governance-driven approach to clinical data assembly. It serves as the central intelligence hub for the clinical synthesis platform, providing unified access to clinical context through three architectural pillars.

## The Three Pillars of Excellence

### Pillar 1: Federated GraphQL API (The "Unified Data Graph")
- **Purpose**: Single endpoint for all clinical context queries
- **Implementation**: Strawberry GraphQL with federation support
- **Features**: 
  - Recipe-based context assembly
  - Field-specific queries
  - Context availability validation
  - Real-time subscriptions

### Pillar 2: Clinical Context Recipe System (The "Governance Engine")
- **Purpose**: Governance-as-code for clinical data assembly
- **Implementation**: YAML-based recipes with Clinical Governance Board approval
- **Features**:
  - Recipe inheritance and composition
  - Version control and approval workflows
  - Conditional data requirements
  - Safety requirement enforcement

### Pillar 3: Multi-Layer Intelligent Cache (The "Performance Accelerator")
- **Purpose**: Sub-200ms context assembly with intelligent caching
- **Implementation**: L1 (in-process) + L2 (Redis) + L3 (service-level) caching
- **Features**:
  - Event-driven cache invalidation via Kafka
  - Predictive cache warming
  - Performance monitoring and optimization

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Clinical Context Service                     │
├─────────────────────────────────────────────────────────────────┤
│  Pillar 1: GraphQL API     │  Pillar 2: Recipe System          │
│  ┌─────────────────────┐   │  ┌─────────────────────────────┐   │
│  │ • Query Resolvers   │   │  │ • Recipe Management         │   │
│  │ • Mutation Handlers │   │  │ • Governance Approval       │   │
│  │ • Subscriptions     │   │  │ • Version Control           │   │
│  │ • Schema Federation │   │  │ • Conditional Rules         │   │
│  └─────────────────────┘   │  └─────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  Pillar 3: Multi-Layer Cache                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ L1: In-Process Cache │ L2: Redis Cache │ L3: Service Cache │ │
│  │ • Workflow-scoped    │ • Distributed   │ • Long-term       │ │
│  │ • <1ms response      │ • <10ms response│ • Background      │ │
│  └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  Event-Driven Invalidation (Kafka)                             │
│  • Clinical data change events • Cache invalidation patterns   │
└─────────────────────────────────────────────────────────────────┘
```

## Clinical Context Recipes

### Recipe Structure
```yaml
recipe_id: "medication_prescribing_v2"
recipe_name: "Medication Prescribing Context"
version: "2.0"
clinical_scenario: "medication_ordering"
workflow_category: "command_initiated"
execution_pattern: "pessimistic"

required_data_points:
  - name: "patient_demographics"
    source_type: "patient_service"
    fields: ["age", "weight", "gender"]
    required: true
    max_age_hours: 24
    quality_threshold: 0.95

conditional_rules:
  - condition: "patient.age < 18"
    description: "Pediatric patient requires additional data"
    additional_data_points: [...]

governance_metadata:
  approved_by: "Clinical Governance Board"
  approval_date: "2024-01-15T10:00:00Z"
  clinical_board_approval_id: "CGB-MED-PRESCRIBE-20240115"
```

### Available Recipes
- **medication_prescribing_v2**: Comprehensive medication prescribing context
- **clinical_deterioration_response_v1**: Emergency response context (Digital Reflex Arc)
- **routine_medication_refill_v1**: Optimistic refill workflow context
- **base_clinical_context_v1**: Foundation recipe for inheritance

## API Reference

### GraphQL Queries

#### Get Context by Recipe
```graphql
query GetContextByRecipe($patientId: String!, $recipeId: String!) {
  getContextByRecipe(patientId: $patientId, recipeId: $recipeId) {
    contextId
    patientId
    assembledData
    completenessScore
    safetyFlags {
      flagType
      severity
      message
    }
    status
    assemblyDurationMs
  }
}
```

#### Validate Context Availability
```graphql
query ValidateContextAvailability($patientId: String!, $recipeId: String!) {
  validateContextAvailability(patientId: $patientId, recipeId: $recipeId) {
    available
    estimatedCompleteness
    unavailableSources
    estimatedAssemblyTimeMs
    cacheAvailable
  }
}
```

### GraphQL Mutations

#### Invalidate Patient Context
```graphql
mutation InvalidatePatientContext($patientId: String!) {
  invalidatePatientContext(patientId: $patientId)
}
```

## Installation & Setup

### Prerequisites
- Python 3.8+
- Redis (for L2 cache)
- Kafka (for event-driven invalidation)
- Access to clinical microservices

### Installation
```bash
# Clone the repository
cd backend/services/context-service

# Install dependencies
pip install -r requirements.txt

# Set environment variables
export REDIS_URL="redis://localhost:6379"
export KAFKA_BOOTSTRAP_SERVERS="your-kafka-servers"

# Start the service
python app/main.py
```

### Configuration
```yaml
# config/service.yaml
service:
  port: 8016
  host: "0.0.0.0"
  
cache:
  redis_url: "redis://localhost:6379"
  l1_max_entries: 1000
  default_ttl_seconds: 300

kafka:
  bootstrap_servers: ["localhost:9092"]
  group_id: "context-service-cache-invalidation"
  
data_sources:
  patient_service: "http://localhost:8003"
  medication_service: "http://localhost:8009"
  # ... other services
```

## Testing

### Run All Tests
```bash
# Run comprehensive test suite
python run_tests.py

# Run specific test suites
pytest tests/test_recipe_system_integration.py -v
pytest tests/test_graphql_api.py -v

# Run with coverage
pytest tests/ --cov=app --cov-report=html
```

### Test Categories
- **Recipe System Integration**: Tests recipe loading, validation, and governance
- **GraphQL API**: Tests all GraphQL queries, mutations, and subscriptions
- **Cache Performance**: Tests multi-layer caching and invalidation
- **End-to-End Workflows**: Tests complete clinical workflows

## Performance Characteristics

### SLA Targets
- **Context Assembly**: <200ms (pessimistic), <100ms (optimistic), <30ms (digital reflex arc)
- **Cache Hit Response**: <1ms (L1), <10ms (L2), <50ms (L3)
- **GraphQL Query Response**: <50ms (cached), <250ms (fresh assembly)

### Scalability
- **Concurrent Requests**: 1000+ concurrent context assemblies
- **Cache Capacity**: 10,000+ contexts in L1, unlimited in L2/L3
- **Event Processing**: 10,000+ Kafka events/second

## Monitoring & Observability

### Health Endpoints
- `GET /health` - Service health check
- `GET /status` - Detailed service status
- `GET /metrics` - Performance metrics

### Key Metrics
- Cache hit ratios (L1, L2, L3)
- Context assembly times
- Recipe validation success rates
- Event processing rates
- Data source health status

### Logging
- Structured logging with correlation IDs
- Clinical audit trails
- Performance tracking
- Error reporting with context

## Security & Compliance

### Clinical Data Protection
- HIPAA-compliant data handling
- Audit logging for all data access
- Encryption in transit and at rest
- Role-based access control

### Governance Compliance
- Clinical Governance Board approval required for all recipes
- Version control with change tracking
- Expiry management for recipes
- Mock data detection and prevention

## Deployment

### Docker Deployment
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY app/ ./app/
COPY config/ ./config/

EXPOSE 8016
CMD ["python", "app/main.py"]
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: context-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: context-service
  template:
    metadata:
      labels:
        app: context-service
    spec:
      containers:
      - name: context-service
        image: context-service:latest
        ports:
        - containerPort: 8016
        env:
        - name: REDIS_URL
          value: "redis://redis-service:6379"
```

## Contributing

### Development Setup
```bash
# Install development dependencies
pip install -r requirements-dev.txt

# Run pre-commit hooks
pre-commit install

# Run linting
black app/ tests/
isort app/ tests/
flake8 app/ tests/

# Run type checking
mypy app/
```

### Recipe Development
1. Create recipe YAML file in `app/config/recipes/`
2. Validate recipe structure
3. Submit for Clinical Governance Board approval
4. Test with integration test suite
5. Deploy to production

## Support & Documentation

- **API Documentation**: Available at `/graphql` (GraphQL Playground)
- **Recipe Documentation**: See `docs/recipes/`
- **Architecture Documentation**: See `docs/architecture/`
- **Troubleshooting Guide**: See `docs/troubleshooting.md`

## License

Copyright (c) 2024 Clinical Synthesis Hub. All rights reserved.

---

**Clinical Context Service v2.0**  
*Federated Clinical Data Intelligence Hub*  
*Implementing the Three Pillars of Excellence*
