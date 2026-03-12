# KB-2 Clinical Context Apollo Federation Integration

## Overview

This integration exposes the KB-2 Clinical Context Service through Apollo Federation, providing unified GraphQL access to clinical phenotyping, risk assessment, and treatment preference capabilities while maintaining type safety and federation compliance.

## Architecture

```
Frontend (GraphQL Client)
       ↓
Apollo Federation Gateway (Port 4000)
       ↓
KB-2 Clinical Context Subgraph (Port 8082/api/federation)
       ↓
KB-2 Go Service (Port 8082/v1/*)
       ↓
MongoDB + Redis + CEL Engine
```

## Core Capabilities

### 1. **Clinical Phenotype Evaluation**
- Batch processing up to 1,000 patients
- CEL-based rule evaluation
- Confidence scoring and implications
- SLA: 100ms p95 latency

### 2. **Risk Assessment**
- Multi-category risk analysis (cardiovascular, diabetes, medication, fall, bleeding)
- Framingham-based cardiovascular risk
- Modifiable vs non-modifiable risk factors
- SLA: 200ms p95 latency

### 3. **Treatment Preferences** 
- Institutional rule-based recommendations
- ADA/EASD 2023, ACC/AHA 2017 guideline compliance
- Cost-effectiveness and formulary preferences
- Conflict resolution for competing recommendations
- SLA: 50ms p95 latency

### 4. **Clinical Context Assembly**
- Unified patient clinical intelligence
- Combines phenotypes, risks, and treatment preferences
- Configurable detail levels (summary, standard, comprehensive, detailed)
- SLA: 200ms p95 latency

## Federation Schema

### Extended Patient Type
```graphql
extend type Patient @key(fields: "id") {
  id: ID! @external
  clinicalContext: ClinicalContext
  phenotypes: [ClinicalPhenotype!]!
  riskAssessments: [RiskAssessment!]!
  treatmentPreferences: [TreatmentPreference!]!
}
```

### Core Types
- **ClinicalContext**: Complete patient clinical intelligence
- **ClinicalPhenotype**: Detected clinical patterns with CEL rules
- **RiskAssessment**: Multi-category risk scores and factors
- **TreatmentPreference**: Institutional treatment recommendations
- **MedicationReference**: Structured medication data

### Query Operations
```graphql
# Direct Operations
evaluatePatientPhenotypes(input: PhenotypeEvaluationInput!): PhenotypeEvaluationResponse!
assessPatientRisk(input: RiskAssessmentInput!): RiskAssessmentResponse!  
getPatientTreatmentPreferences(input: TreatmentPreferencesInput!): TreatmentPreferencesResponse!
assemblePatientContext(input: ClinicalContextInput!): ClinicalContextResponse!

# Meta Operations  
availablePhenotypes(category: String): [PhenotypeDefinition!]!
patientContextHistory(patientId: ID!, limit: Int = 10): [ClinicalContextHistory!]!
```

## Implementation Files

### Core Federation Components

**1. Schema Definition**
- **File**: `apollo-federation/schemas/kb2-clinical-context-schema.js`
- **Purpose**: Complete GraphQL schema with federation directives
- **Key Features**:
  - Patient type extensions with `@key(fields: "id")`
  - Comprehensive clinical intelligence types
  - Input/output type definitions
  - Enumeration types for risk categories, severity levels

**2. Resolvers**  
- **File**: `apollo-federation/resolvers/kb2-clinical-context-resolvers.js`
- **Purpose**: GraphQL resolvers connecting to KB-2 Go service
- **Key Features**:
  - REST API integration with KB-2 service
  - Data transformation between GraphQL and KB-2 formats
  - Error handling and logging
  - Patient extension resolvers for federation

**3. Subgraph Service**
- **File**: `apollo-federation/services/kb2-clinical-context-service.js` 
- **Purpose**: Standalone Apollo subgraph server
- **Key Features**:
  - Health checks and readiness probes
  - Performance monitoring and metrics
  - Graceful shutdown handling
  - Context creation for authentication

### Configuration

**4. Federation Configuration**
- **File**: `apollo-federation/supergraph.yaml` 
- **Updated**: Added KB-2 subgraph routing configuration
- **URL**: `http://localhost:8082/api/federation`

**5. Gateway Integration**
- **File**: `apollo-federation/index.js`
- **Status**: Already includes KB-2 in `federationServices` array
- **Configuration**: Uses `IntrospectAndCompose` for schema composition

### Testing & Validation

**6. Integration Tests**
- **File**: `apollo-federation/test-kb2-integration.js`
- **Purpose**: Comprehensive federation integration testing
- **Coverage**:
  - Direct KB-2 service health and functionality
  - Federation GraphQL operations
  - Patient extension queries
  - Performance benchmarking
  - SLA compliance validation

## Usage Examples

### 1. Evaluate Patient Phenotypes
```graphql
mutation EvaluatePhenotypes($input: PhenotypeEvaluationInput!) {
  evaluatePatientPhenotypes(input: $input) {
    results {
      patientId
      phenotypes {
        id
        name
        category
        matched
        confidence
        implications {
          severity
          description
          recommendations
        }
      }
      evaluationSummary {
        totalPhenotypes
        matchedPhenotypes
        averageConfidence
      }
    }
    processingTime
    slaCompliant
  }
}
```

**Variables:**
```json
{
  "input": {
    "patients": [{
      "id": "patient-123",
      "age": 65,
      "gender": "male",
      "conditions": ["diabetes", "hypertension"],
      "labs": [
        {
          "name": "HbA1c",
          "value": 8.5,
          "unit": "%"
        }
      ]
    }],
    "includeImplications": true,
    "confidenceThreshold": 0.7
  }
}
```

### 2. Patient with Clinical Context
```graphql
query GetPatientWithContext($patientId: ID!) {
  patient(id: $patientId) {
    id
    name {
      family
      given
    }
    clinicalContext {
      phenotypes {
        name
        matched
        confidence
      }
      riskAssessments {
        category
        score
        category_result
      }
      treatmentPreferences {
        condition
        firstLine {
          medication {
            name
            genericName
          }
        }
      }
      contextMetadata {
        processingTime
        slaCompliant
        confidenceScore
      }
    }
  }
}
```

### 3. Comprehensive Risk Assessment
```graphql
mutation AssessRisk($input: RiskAssessmentInput!) {
  assessPatientRisk(input: $input) {
    riskAssessments {
      model
      category
      score
      category_result
      recommendations {
        priority
        action
        rationale
      }
      riskFactors {
        name
        value
        contribution
        modifiable
      }
    }
    overallRiskProfile {
      overallRisk
      primaryConcerns
      recommendedActions
    }
  }
}
```

### 4. Treatment Preferences
```graphql
mutation GetTreatmentPreferences($input: TreatmentPreferencesInput!) {
  getPatientTreatmentPreferences(input: $input) {
    preferences {
      condition
      firstLine {
        medication {
          name
          drugClass
        }
        preferenceScore
        reasons
      }
      rationale
      guidelineSource
    }
    alternativeOptions {
      medication {
        name
      }
      suitabilityScore
      rationale
    }
  }
}
```

## Performance Characteristics

### SLA Targets
- **Phenotype Evaluation**: 100ms p95 (batch of 100 patients)
- **Risk Assessment**: 200ms p95 (comprehensive analysis)  
- **Treatment Preferences**: 50ms p95 (single condition)
- **Context Assembly**: 200ms p95 (all components)
- **Throughput**: 10,000+ requests/second
- **Cache Hit Rate**: >95%

### Optimization Features
- 3-tier caching strategy (Redis, in-memory, HTTP)
- Batch processing for phenotype evaluation
- Parallel component assembly
- Connection pooling and request queuing
- Performance monitoring and SLA tracking

## Monitoring & Operations

### Health Endpoints
- **Subgraph Health**: `http://localhost:8082/health`
- **Service Readiness**: `http://localhost:8082/ready` 
- **Metrics**: `http://localhost:8082/metrics`
- **Service Info**: `http://localhost:8082/info`

### Key Metrics
```
kb2_subgraph_requests_total           # Total GraphQL requests
kb2_subgraph_response_duration_seconds # Response latency histograms  
kb2_service_dependency_up            # KB-2 service dependency status
```

### Error Handling
- **GraphQL Errors**: Structured error responses with extensions
- **Service Unavailable**: Graceful degradation with proper HTTP status codes
- **Timeout Handling**: Configurable timeouts with fallback responses
- **Authentication**: Token-based authentication with role-based access

## Development Workflow

### 1. Running the Integration

**Start KB-2 Service:**
```bash
cd backend/services/knowledge-base-services/kb-2-clinical-context-go
make run
```

**Start KB-2 Subgraph (Optional - for standalone testing):**
```bash
cd apollo-federation
node services/kb2-clinical-context-service.js
```

**Start Federation Gateway:**
```bash
cd apollo-federation  
npm start
```

### 2. Testing the Integration
```bash
cd apollo-federation

# Run comprehensive integration tests
node test-kb2-integration.js

# Test specific functionality
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { availablePhenotypes { id name category } }"
  }'
```

### 3. Development & Debugging

**Check Service Health:**
```bash
# KB-2 service
curl http://localhost:8082/health

# Federation gateway
curl http://localhost:4000/health

# Subgraph (if running standalone)
curl http://localhost:8082/health
```

**View GraphQL Schema:**
- Apollo Studio: `http://localhost:4000/graphql`
- Schema introspection available in development mode

## Error Scenarios & Troubleshooting

### Common Issues

**1. KB-2 Service Unavailable**
- **Symptoms**: 503 errors, connection refused
- **Resolution**: Start KB-2 service, check MongoDB/Redis dependencies
- **Federation Behavior**: Returns null for patient extensions, errors for direct queries

**2. Schema Composition Errors**
- **Symptoms**: Federation startup fails, schema conflicts
- **Resolution**: Check federation directives, ensure types match
- **Debug**: Use `rover subgraph check` for schema validation

**3. Performance Issues**
- **Symptoms**: High latency, SLA violations  
- **Resolution**: Check KB-2 service metrics, Redis cache hit rates
- **Monitoring**: Use `/metrics` endpoints and performance headers

**4. Authentication Failures**
- **Symptoms**: 401 errors, access denied
- **Resolution**: Check JWT tokens, user roles/permissions headers
- **Context**: Verify authentication context creation in resolvers

### Debug Commands
```bash
# Check federation schema composition
npm run generate-supergraph

# Test direct KB-2 service
curl -X POST http://localhost:8082/v1/phenotypes/evaluate \
  -H "Content-Type: application/json" \
  -d '{"patients": [{"id": "test", "age": 65}]}'

# Validate GraphQL schema
rover subgraph check my-graph@current --name kb2-clinical-context \
  --schema ./schemas/kb2-clinical-context-schema.js
```

## Security Considerations

### Authentication & Authorization
- JWT token validation through federation context
- User role and permission checking
- Service-to-service authentication headers
- Request ID tracking for audit trails

### Data Privacy
- No patient data stored permanently in KB-2 service
- All processing in-memory with configurable TTL
- HIPAA-compliant audit logging
- Encryption in transit (HTTPS/TLS)

### Input Validation
- GraphQL schema validation for all inputs
- CEL expression validation and sandboxing
- Rate limiting and request size limits
- SQL injection prevention in database queries

## Future Enhancements

### Planned Features
1. **Real-time Updates**: WebSocket subscriptions for context changes
2. **Machine Learning**: Integration with ML models for risk prediction
3. **Clinical Decision Support**: Enhanced treatment recommendations
4. **Multi-tenant Support**: Organization-specific phenotypes and rules
5. **FHIR Integration**: Native FHIR resource support

### Performance Optimizations
1. **GraphQL DataLoader**: Batch and cache resolver calls
2. **Query Complexity Analysis**: Prevent expensive queries
3. **Response Caching**: HTTP-level caching for stable data
4. **Connection Pooling**: Optimize database connections

### Monitoring Enhancements
1. **Distributed Tracing**: OpenTelemetry integration
2. **Custom Metrics**: Business-specific metrics and dashboards
3. **Alerting**: SLA violation and error rate alerts
4. **Performance Profiling**: Query-level performance analysis

---

## Quick Reference

### Service URLs
- **Federation Gateway**: `http://localhost:4000/graphql`
- **KB-2 Subgraph**: `http://localhost:8082/api/federation`
- **KB-2 Service**: `http://localhost:8082/v1/*`

### Key Files
- Schema: `schemas/kb2-clinical-context-schema.js`
- Resolvers: `resolvers/kb2-clinical-context-resolvers.js`  
- Service: `services/kb2-clinical-context-service.js`
- Tests: `test-kb2-integration.js`
- Config: `supergraph.yaml`

### Test Commands
```bash
# Integration tests
node test-kb2-integration.js

# Health checks  
curl http://localhost:8082/health
curl http://localhost:4000/health

# GraphQL query
curl -X POST http://localhost:4000/graphql -H "Content-Type: application/json" -d '{"query":"query { availablePhenotypes { name } }"}'
```

This integration provides a comprehensive, production-ready GraphQL interface to KB-2's clinical intelligence capabilities while maintaining federation compliance and performance requirements.