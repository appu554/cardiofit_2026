# KB-7 Phase 4 FHIR Integration Requirements

**Date**: 2025-09-20
**Phase**: 4 - FHIR Terminology Service Integration
**Prerequisites**: ✅ Phase 3 Complete (23,337 triples, 2,468 medication concepts)
**Integration Target**: Google Healthcare API + Go Medication Service v2

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   FHIR Clients  │ → │  Go Service v2   │ ↔ │ KB-7 GraphDB    │
│   (Frontend)    │    │  (Orchestrator)  │    │ (Terminology)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                ↓
                       ┌──────────────────┐
                       │ Google FHIR Store│
                       │  (Patient Data)  │
                       └──────────────────┘
```

## Phase 4 Objectives

### Primary Goals
1. **FHIR Terminology Service**: Expose KB-7 as FHIR-compliant terminology service
2. **Go Service Integration**: Implement SPARQL client in medication-service-v2
3. **Clinical Decision Support**: Drug interaction detection via FHIR workflows
4. **Terminology Validation**: FHIR resource validation using KB-7 concepts

### Success Criteria
- ✅ FHIR R4 terminology endpoints operational
- ✅ Go service can query KB-7 for medication concepts
- ✅ Drug interaction detection integrated into FHIR workflows
- ✅ External terminology mappings accessible via FHIR APIs
- ✅ Performance meets clinical workflow requirements (<500ms response)

## 1. FHIR Terminology Service Implementation

### A. FHIR R4 Terminology Operations

#### CodeSystem Resource
```json
{
  "resourceType": "CodeSystem",
  "id": "kb7-medication-concepts",
  "url": "http://cardiofit.ai/kb7/fhir/CodeSystem/medication-concepts",
  "version": "1.0.0",
  "name": "KB7MedicationConcepts",
  "title": "KB-7 Clinical Medication Concepts",
  "status": "active",
  "experimental": false,
  "publisher": "CardioFit Clinical Platform",
  "description": "Clinical medication concepts with external terminology mappings",
  "caseSensitive": true,
  "content": "complete",
  "count": 2468,
  "concept": [
    {
      "code": "aspirin-81mg",
      "display": "Aspirin 81mg Low-Dose",
      "definition": "Low-dose aspirin for cardiovascular protection",
      "property": [
        {
          "code": "snomedMapping",
          "valueCode": "387458008"
        },
        {
          "code": "rxnormMapping",
          "valueCode": "243670"
        },
        {
          "code": "safetyLevel",
          "valueString": "low-risk"
        }
      ]
    }
  ]
}
```

#### ValueSet Resource
```json
{
  "resourceType": "ValueSet",
  "id": "cardiovascular-medications",
  "url": "http://cardiofit.ai/kb7/fhir/ValueSet/cardiovascular-medications",
  "version": "1.0.0",
  "name": "CardiovascularMedications",
  "title": "Cardiovascular Protection Medications",
  "status": "active",
  "experimental": false,
  "description": "Medications used for cardiovascular protection and management",
  "compose": {
    "include": [
      {
        "system": "http://cardiofit.ai/kb7/fhir/CodeSystem/medication-concepts",
        "filter": [
          {
            "property": "concept",
            "op": "in",
            "value": "aspirin-81mg,warfarin-5mg,atorvastatin-20mg,lisinopril-10mg"
          }
        ]
      }
    ]
  }
}
```

### B. Required FHIR Terminology Endpoints

#### Core Operations
```
GET /fhir/CodeSystem/kb7-medication-concepts
GET /fhir/ValueSet/cardiovascular-medications
POST /fhir/CodeSystem/$lookup
POST /fhir/ValueSet/$expand
POST /fhir/CodeSystem/$validate-code
POST /fhir/ConceptMap/$translate
```

#### Advanced Operations
```
POST /fhir/ValueSet/$validate-code
POST /fhir/CodeSystem/$subsumes
POST /fhir/ConceptMap/$closure
GET /fhir/NamingSystem/kb7-identifiers
```

## 2. Go Service Integration Architecture

### A. SPARQL Client Implementation

#### Core Client Structure
```go
// pkg/terminology/kb7_client.go
package terminology

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

type KB7Client struct {
    endpoint string
    client   *http.Client
}

type ConceptResult struct {
    ConceptID       string            `json:"conceptId"`
    PreferredLabel  string            `json:"preferredLabel"`
    AlternativeLabels []string        `json:"alternativeLabels"`
    SnomedMapping   string            `json:"snomedMapping,omitempty"`
    RxNormMapping   string            `json:"rxnormMapping,omitempty"`
    SafetyLevel     string            `json:"safetyLevel"`
    RequiresReview  bool              `json:"requiresReview"`
    Properties      map[string]string `json:"properties"`
}

type DrugInteraction struct {
    InteractionID   string   `json:"interactionId"`
    InvolvedDrugs   []string `json:"involvedDrugs"`
    SafetyLevel     string   `json:"safetyLevel"`
    Description     string   `json:"description"`
    RequiresReview  bool     `json:"requiresReview"`
    EvidenceLevel   string   `json:"evidenceLevel"`
}

func NewKB7Client(endpoint string) *KB7Client {
    return &KB7Client{
        endpoint: endpoint,
        client:   &http.Client{},
    }
}
```

#### Medication Concept Queries
```go
func (c *KB7Client) FindMedicationConcept(ctx context.Context, searchTerm string) (*ConceptResult, error) {
    sparql := `
    PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
    PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
    PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

    SELECT ?concept ?prefLabel ?altLabel ?snomedId ?rxnormId ?safetyLevel ?requiresReview
    WHERE {
        ?concept a kb7:MedicationConcept .
        ?concept rdfs:label ?prefLabel .
        OPTIONAL { ?concept skos:altLabel ?altLabel }
        OPTIONAL {
            ?concept kb7:mapsTo ?snomed .
            FILTER(STRSTARTS(STR(?snomed), "http://snomed.info/id/"))
            BIND(STRAFTER(STR(?snomed), "http://snomed.info/id/") AS ?snomedId)
        }
        OPTIONAL {
            ?concept kb7:mapsTo ?rxnorm .
            FILTER(STRSTARTS(STR(?rxnorm), "http://purl.bioontology.org/ontology/RXNORM/"))
            BIND(STRAFTER(STR(?rxnorm), "http://purl.bioontology.org/ontology/RXNORM/") AS ?rxnormId)
        }
        OPTIONAL { ?concept kb7:safetyLevel ?safetyLevel }
        OPTIONAL { ?concept kb7:requiresClinicalReview ?requiresReview }
        FILTER(CONTAINS(LCASE(STR(?prefLabel)), LCASE("%s")) ||
               CONTAINS(LCASE(STR(?altLabel)), LCASE("%s")))
    }
    LIMIT 10`

    return c.executeSPARQLQuery(ctx, fmt.Sprintf(sparql, searchTerm, searchTerm))
}

func (c *KB7Client) CheckDrugInteractions(ctx context.Context, medicationIDs []string) ([]DrugInteraction, error) {
    sparql := `
    PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
    PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

    SELECT ?interaction ?label ?safetyLevel ?requiresReview ?evidenceLevel ?involvedDrug
    WHERE {
        ?interaction a kb7:DrugInteraction .
        ?interaction rdfs:label ?label .
        ?interaction kb7:involves ?involvedDrug .
        OPTIONAL { ?interaction kb7:safetyLevel ?safetyLevel }
        OPTIONAL { ?interaction kb7:requiresClinicalReview ?requiresReview }
        OPTIONAL { ?interaction kb7:evidenceLevel ?evidenceLevel }
        FILTER(?involvedDrug IN (%s))
    }`

    return c.executeDrugInteractionQuery(ctx, fmt.Sprintf(sparql, strings.Join(medicationIDs, ",")))
}
```

### B. Service Integration Points

#### Medication Service Integration
```go
// internal/services/medication_service.go
package services

type MedicationService struct {
    kb7Client      *terminology.KB7Client
    fhirClient     *google_fhir.GoogleFHIRClient
    repository     *repositories.MedicationRepository
}

func (s *MedicationService) ValidateMedicationRequest(ctx context.Context, request *fhir.MedicationRequest) (*ValidationResult, error) {
    // 1. Extract medication coding from FHIR request
    coding := request.MedicationCodeableConcept.Coding[0]

    // 2. Query KB-7 for concept validation
    concept, err := s.kb7Client.FindMedicationConcept(ctx, coding.Display)
    if err != nil {
        return nil, fmt.Errorf("terminology lookup failed: %w", err)
    }

    // 3. Validate external terminology mappings
    if coding.System == "http://snomed.info/sct" && concept.SnomedMapping != coding.Code {
        return &ValidationResult{
            Valid:   false,
            Message: "SNOMED CT code mismatch",
        }, nil
    }

    // 4. Check for drug interactions with patient's current medications
    interactions, err := s.checkPatientDrugInteractions(ctx, request.Subject.Reference, concept.ConceptID)
    if err != nil {
        return nil, fmt.Errorf("drug interaction check failed: %w", err)
    }

    return &ValidationResult{
        Valid:        true,
        Concept:      concept,
        Interactions: interactions,
        RequiresReview: concept.RequiresReview || len(interactions) > 0,
    }, nil
}
```

## 3. Clinical Decision Support Integration

### A. Drug Interaction Detection Workflow

#### FHIR Workflow Integration
```json
{
  "resourceType": "Task",
  "id": "drug-interaction-check",
  "status": "requested",
  "intent": "order",
  "code": {
    "coding": [{
      "system": "http://cardiofit.ai/kb7/fhir/CodeSystem/clinical-tasks",
      "code": "drug-interaction-check",
      "display": "Drug Interaction Safety Check"
    }]
  },
  "for": {
    "reference": "Patient/patient-cardio-001"
  },
  "input": [
    {
      "type": {
        "coding": [{
          "system": "http://cardiofit.ai/kb7/fhir/CodeSystem/task-parameters",
          "code": "medication-request",
          "display": "Medication Request"
        }]
      },
      "valueReference": {
        "reference": "MedicationRequest/aspirin-request-001"
      }
    }
  ],
  "output": [
    {
      "type": {
        "coding": [{
          "system": "http://cardiofit.ai/kb7/fhir/CodeSystem/task-outputs",
          "code": "safety-assessment",
          "display": "Clinical Safety Assessment"
        }]
      },
      "valueCodeableConcept": {
        "coding": [{
          "system": "http://cardiofit.ai/kb7/fhir/CodeSystem/safety-levels",
          "code": "high-risk",
          "display": "High Risk - Clinical Review Required"
        }]
      }
    }
  ]
}
```

### B. Clinical Decision Support Rules Engine

#### Go Implementation
```go
// internal/clinical/decision_support.go
package clinical

type DecisionSupportEngine struct {
    kb7Client *terminology.KB7Client
    ruleEngine *rules.ClinicalRulesEngine
}

type ClinicalDecision struct {
    RecommendationID string                 `json:"recommendationId"`
    PatientID        string                 `json:"patientId"`
    MedicationID     string                 `json:"medicationId"`
    Decision         string                 `json:"decision"` // approve, review, reject
    SafetyLevel      string                 `json:"safetyLevel"`
    Rationale        string                 `json:"rationale"`
    RequiredActions  []string               `json:"requiredActions"`
    EvidenceLevel    string                 `json:"evidenceLevel"`
    Interactions     []DrugInteraction      `json:"interactions"`
    Contraindications []Contraindication    `json:"contraindications"`
}

func (e *DecisionSupportEngine) EvaluateMedicationRequest(ctx context.Context, request *MedicationRequest) (*ClinicalDecision, error) {
    // 1. Get patient medication history
    currentMeds, err := e.getPatientMedications(ctx, request.PatientID)
    if err != nil {
        return nil, err
    }

    // 2. Check drug interactions
    interactions, err := e.kb7Client.CheckDrugInteractions(ctx, append(currentMeds, request.MedicationID))
    if err != nil {
        return nil, err
    }

    // 3. Apply clinical decision rules
    decision := &ClinicalDecision{
        RecommendationID: generateID(),
        PatientID:        request.PatientID,
        MedicationID:     request.MedicationID,
        Interactions:     interactions,
    }

    // 4. Determine safety level and required actions
    if len(interactions) > 0 {
        highRiskFound := false
        for _, interaction := range interactions {
            if interaction.SafetyLevel == "high-risk" {
                highRiskFound = true
                break
            }
        }

        if highRiskFound {
            decision.Decision = "review"
            decision.SafetyLevel = "high-risk"
            decision.RequiredActions = []string{"clinical-pharmacist-review", "patient-monitoring-plan"}
            decision.Rationale = "High-risk drug interaction detected requiring clinical review"
        } else {
            decision.Decision = "approve"
            decision.SafetyLevel = "moderate-risk"
            decision.RequiredActions = []string{"patient-education", "monitoring-reminder"}
            decision.Rationale = "Moderate drug interaction - approved with monitoring"
        }
    } else {
        decision.Decision = "approve"
        decision.SafetyLevel = "low-risk"
        decision.Rationale = "No significant drug interactions detected"
    }

    return decision, nil
}
```

## 4. Performance and Scalability Requirements

### A. Performance Targets
```yaml
terminology_lookup:
  target_response_time: "< 200ms"
  concurrent_users: 100
  queries_per_second: 500

drug_interaction_check:
  target_response_time: "< 300ms"
  concurrent_checks: 50
  batch_processing: true

concept_validation:
  target_response_time: "< 150ms"
  cache_hit_ratio: "> 90%"
  cache_ttl: "1 hour"
```

### B. Caching Strategy
```go
// internal/cache/terminology_cache.go
package cache

type TerminologyCache struct {
    redis        *redis.Client
    kb7Client    *terminology.KB7Client
    defaultTTL   time.Duration
}

func (c *TerminologyCache) GetConcept(ctx context.Context, conceptID string) (*ConceptResult, error) {
    // 1. Check Redis cache first
    cached, err := c.redis.Get(ctx, fmt.Sprintf("concept:%s", conceptID)).Result()
    if err == nil {
        var concept ConceptResult
        json.Unmarshal([]byte(cached), &concept)
        return &concept, nil
    }

    // 2. Query KB-7 if not cached
    concept, err := c.kb7Client.FindMedicationConcept(ctx, conceptID)
    if err != nil {
        return nil, err
    }

    // 3. Cache result
    conceptJSON, _ := json.Marshal(concept)
    c.redis.Set(ctx, fmt.Sprintf("concept:%s", conceptID), conceptJSON, c.defaultTTL)

    return concept, nil
}
```

## 5. Integration Testing Strategy

### A. End-to-End Test Scenarios

#### Scenario 1: Medication Validation Workflow
```go
func TestMedicationValidationWorkflow(t *testing.T) {
    // 1. Create test patient in Google FHIR store
    patient := createTestPatient()

    // 2. Submit medication request with SNOMED CT coding
    medicationRequest := &fhir.MedicationRequest{
        Status: "active",
        Intent: "order",
        MedicationCodeableConcept: &fhir.CodeableConcept{
            Coding: []fhir.Coding{{
                System:  "http://snomed.info/sct",
                Code:    "387458008",
                Display: "Aspirin",
            }},
        },
        Subject: fhir.Reference{Reference: fmt.Sprintf("Patient/%s", patient.ID)},
    }

    // 3. Validate via KB-7 terminology service
    result, err := medicationService.ValidateMedicationRequest(context.Background(), medicationRequest)
    assert.NoError(t, err)
    assert.True(t, result.Valid)
    assert.Equal(t, "aspirin-81mg", result.Concept.ConceptID)
    assert.Equal(t, "243670", result.Concept.RxNormMapping)
}
```

#### Scenario 2: Drug Interaction Detection
```go
func TestDrugInteractionDetection(t *testing.T) {
    // 1. Create patient with existing warfarin prescription
    patient := createPatientWithMedication("warfarin-5mg")

    // 2. Attempt to add aspirin (should trigger high-risk interaction)
    aspirinRequest := createMedicationRequest(patient.ID, "aspirin-81mg")

    // 3. Check decision support response
    decision, err := decisionEngine.EvaluateMedicationRequest(context.Background(), aspirinRequest)
    assert.NoError(t, err)
    assert.Equal(t, "review", decision.Decision)
    assert.Equal(t, "high-risk", decision.SafetyLevel)
    assert.Contains(t, decision.RequiredActions, "clinical-pharmacist-review")
}
```

### B. Performance Testing
```bash
# Load testing with artillery.io
npx artillery run terminology-load-test.yml

# Performance baseline validation
./scripts/benchmark-kb7-queries.sh

# Integration stress testing
go test -bench=BenchmarkTerminologyIntegration ./tests/integration/...
```

## 6. Deployment and Operations

### A. Container Configuration
```dockerfile
# Dockerfile.fhir-terminology
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o fhir-terminology-service ./cmd/fhir-terminology

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/fhir-terminology-service .
EXPOSE 8015
CMD ["./fhir-terminology-service"]
```

### B. Service Configuration
```yaml
# config/fhir-terminology.yml
server:
  port: 8015
  read_timeout: 30s
  write_timeout: 30s

kb7:
  endpoint: "http://localhost:7200/repositories/kb7-terminology"
  timeout: 10s
  retry_attempts: 3

cache:
  redis_url: "redis://localhost:6379/1"
  default_ttl: "1h"
  max_memory: "512mb"

google_fhir:
  project_id: "cardiofit-905a8"
  location: "us-central1"
  dataset_id: "cardiofit-clinical-dev"
  fhir_store_id: "medication-fhir-store"
  credentials_path: "./credentials/google-credentials.json"

logging:
  level: "info"
  format: "json"
```

## 7. Success Metrics and Validation

### A. Technical Metrics
- ✅ FHIR R4 compliance validation passes
- ✅ Terminology lookup response time < 200ms
- ✅ Drug interaction detection < 300ms
- ✅ External terminology mapping accuracy > 95%
- ✅ Cache hit ratio > 90%

### B. Clinical Metrics
- ✅ Drug interaction detection accuracy validated by clinical experts
- ✅ False positive rate < 5%
- ✅ Clinical workflow integration seamless
- ✅ Safety alerts properly prioritized
- ✅ External terminology mappings clinically validated

## Conclusion

Phase 4 FHIR integration builds upon the solid Phase 3 foundation to create a production-ready clinical terminology service. The combination of KB-7's comprehensive medication knowledge base with FHIR-compliant APIs enables seamless integration into existing healthcare workflows while providing robust clinical decision support capabilities.

**Key Deliverables**:
1. FHIR R4 terminology service endpoints
2. Go service SPARQL client integration
3. Clinical decision support engine
4. Drug interaction detection workflows
5. Performance optimization with caching
6. Comprehensive integration testing

**Next Phase**: Production deployment with monitoring, security hardening, and clinical user training.

---
**Requirements Document**: Phase 4 FHIR Integration
**Target Completion**: End of Phase 4 development cycle
**Dependencies**: Phase 3 complete, Google Healthcare API access, clinical validation team