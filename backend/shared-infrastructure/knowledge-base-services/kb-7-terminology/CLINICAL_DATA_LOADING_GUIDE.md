# KB-7 Clinical Data Loading Guide

**Objective**: Load sample clinical data into KB-7 GraphDB repository and integrate with Google FHIR Healthcare API for complete Phase 3 testing.

## Architecture Overview

```
Google FHIR Store ←→ Go Medication Service v2 ←→ KB-7 GraphDB Repository
     (Patient Data)        (Orchestration)         (Terminology)
```

## Configuration Details

### Google Healthcare API Setup
- **Project ID**: `cardiofit-905a8`
- **Location**: `us-central1` (default)
- **Dataset**: `cardiofit-clinical-dev`
- **FHIR Store**: `medication-fhir-store`
- **Service Account**: `healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com`
- **Base URL**: `https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/us-central1/datasets/cardiofit-clinical-dev/fhirStores/medication-fhir-store/fhir`

### KB-7 GraphDB Repository
- **Repository**: `kb7-terminology`
- **Endpoint**: `http://localhost:7200/repositories/kb7-terminology`
- **Current Status**: ✅ Licensed, 498 triples loaded, OWL 2 RL reasoning active

## 1. Sample Clinical Data Creation

### A. Create Sample FHIR Resources

#### Sample Medications (FHIR R4)
```json
{
  "resourceType": "Medication",
  "id": "aspirin-81mg",
  "code": {
    "coding": [{
      "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
      "code": "243670",
      "display": "Aspirin 81 MG Oral Tablet"
    }, {
      "system": "http://snomed.info/sct",
      "code": "387458008",
      "display": "Aspirin"
    }]
  },
  "status": "active",
  "form": {
    "coding": [{
      "system": "http://snomed.info/sct",
      "code": "385055001",
      "display": "Tablet"
    }]
  },
  "ingredient": [{
    "itemCodeableConcept": {
      "coding": [{
        "system": "http://snomed.info/sct",
        "code": "387458008",
        "display": "Aspirin"
      }]
    },
    "strength": {
      "numerator": {
        "value": 81,
        "unit": "mg",
        "system": "http://unitsofmeasure.org",
        "code": "mg"
      },
      "denominator": {
        "value": 1,
        "unit": "tablet",
        "system": "http://snomed.info/sct",
        "code": "385055001"
      }
    }
  }]
}
```

#### Sample Patient
```json
{
  "resourceType": "Patient",
  "id": "patient-cardio-001",
  "identifier": [{
    "system": "http://cardiofit.ai/patient-id",
    "value": "CARDIO-001"
  }],
  "name": [{
    "family": "Smith",
    "given": ["John", "Robert"]
  }],
  "gender": "male",
  "birthDate": "1965-03-15",
  "active": true
}
```

#### Sample MedicationRequest
```json
{
  "resourceType": "MedicationRequest",
  "id": "aspirin-request-001",
  "status": "active",
  "intent": "order",
  "medicationReference": {
    "reference": "Medication/aspirin-81mg"
  },
  "subject": {
    "reference": "Patient/patient-cardio-001"
  },
  "authoredOn": "2025-09-20T12:00:00Z",
  "dosageInstruction": [{
    "text": "Take 1 tablet by mouth daily for cardiovascular protection",
    "timing": {
      "repeat": {
        "frequency": 1,
        "period": 1,
        "periodUnit": "d"
      }
    },
    "route": {
      "coding": [{
        "system": "http://snomed.info/sct",
        "code": "26643006",
        "display": "Oral route"
      }]
    },
    "doseAndRate": [{
      "doseQuantity": {
        "value": 1,
        "unit": "tablet",
        "system": "http://snomed.info/sct",
        "code": "385055001"
      }
    }]
  }]
}
```

### B. KB-7 Clinical Terminology Data

#### Sample Clinical Concepts (Turtle)
```turtle
@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .
@prefix sct: <http://snomed.info/id/> .
@prefix rxnorm: <http://purl.bioontology.org/ontology/RXNORM/> .

# Aspirin as a Medication Concept
kb7:aspirin-concept a kb7:MedicationConcept ;
    rdfs:label "Aspirin"@en ;
    kb7:mapsTo sct:387458008 ;
    kb7:mapsTo rxnorm:1154 ;
    kb7:safetyLevel "low-risk" ;
    kb7:regulatoryStatus "approved" ;
    kb7:evidenceLevel "high" ;
    kb7:requiresClinicalReview false .

# Cardiovascular Protection as Clinical Condition
kb7:cardiovascular-protection a kb7:ClinicalCondition ;
    rdfs:label "Cardiovascular Protection"@en ;
    kb7:mapsTo sct:182840001 ;
    kb7:evidenceLevel "high" ;
    kb7:requiresClinicalReview false .

# Drug Interaction - Aspirin + Warfarin
kb7:aspirin-warfarin-interaction a kb7:DrugInteraction ;
    rdfs:label "Aspirin-Warfarin Bleeding Risk"@en ;
    kb7:involves kb7:aspirin-concept ;
    kb7:involves kb7:warfarin-concept ;
    kb7:safetyLevel "high-risk" ;
    kb7:requiresClinicalReview true ;
    kb7:evidenceLevel "high" .
```

## 2. Data Loading Implementation

### A. Load Sample Data into GraphDB

Create a comprehensive clinical dataset loading script:

```bash
#!/bin/bash
# load-clinical-data.sh

echo "🏥 Loading Clinical Data into KB-7 Repository"

# 1. Load sample medications
curl -X POST \
  -H "Content-Type: text/turtle" \
  --data '@sample-data/medications.ttl' \
  "http://localhost:7200/repositories/kb7-terminology/statements"

# 2. Load clinical conditions
curl -X POST \
  -H "Content-Type: text/turtle" \
  --data '@sample-data/conditions.ttl' \
  "http://localhost:7200/repositories/kb7-terminology/statements"

# 3. Load drug interactions
curl -X POST \
  -H "Content-Type: text/turtle" \
  --data '@sample-data/interactions.ttl' \
  "http://localhost:7200/repositories/kb7-terminology/statements"

# 4. Load concept mappings
curl -X POST \
  -H "Content-Type: text/turtle" \
  --data '@sample-data/mappings.ttl' \
  "http://localhost:7200/repositories/kb7-terminology/statements"

echo "✅ Clinical data loaded successfully"
```

### B. Go Service Integration Test

Create a Go integration test that:
1. Connects to KB-7 GraphDB
2. Queries clinical terminology
3. Creates FHIR resources in Google Healthcare API
4. Validates the complete pipeline

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cardiofit/medication-service-v2/internal/infrastructure/google_fhir"
)

func testCompletePhase3Pipeline() error {
    ctx := context.Background()

    // 1. Test KB-7 SPARQL connectivity
    kb7Result, err := queryKB7Terminology(ctx, "aspirin")
    if err != nil {
        return fmt.Errorf("KB-7 query failed: %w", err)
    }

    // 2. Test Google FHIR integration
    fhirClient := google_fhir.NewGoogleFHIRClient(&google_fhir.Config{
        ProjectID:       "cardiofit-905a8",
        Location:        "us-central1",
        DatasetID:       "cardiofit-clinical-dev",
        FHIRStoreID:     "medication-fhir-store",
        CredentialsPath: "./credentials/google-credentials.json",
    })

    // 3. Create sample medication in FHIR store
    medication := map[string]interface{}{
        "resourceType": "Medication",
        "id":           "aspirin-81mg",
        "code": map[string]interface{}{
            "coding": []map[string]interface{}{{
                "system":  kb7Result.SnomedCode,
                "code":    kb7Result.ConceptID,
                "display": kb7Result.PreferredTerm,
            }},
        },
        "status": "active",
    }

    result, err := fhirClient.CreateResource(ctx, "Medication", medication)
    if err != nil {
        return fmt.Errorf("FHIR creation failed: %w", err)
    }

    log.Printf("✅ Complete pipeline test successful: %+v", result)
    return nil
}
```

## 3. Complete Phase 3 Testing Scenarios

### Scenario 1: Medication Terminology Lookup
1. Query KB-7 for "aspirin" concept
2. Retrieve SNOMED CT and RxNorm mappings
3. Create FHIR Medication resource with proper coding
4. Store in Google Healthcare API

### Scenario 2: Drug Interaction Detection
1. Load patient medication list from FHIR
2. Query KB-7 for drug interactions
3. Trigger clinical review if high-risk interaction found
4. Update medication request with safety warnings

### Scenario 3: Clinical Decision Support
1. Receive medication request
2. Validate against KB-7 clinical guidelines
3. Apply dosing rules from knowledge base
4. Generate clinical recommendations

### Scenario 4: Terminology Mapping Validation
1. Load external terminologies (SNOMED CT subset)
2. Validate KB-7 mappings against authoritative sources
3. Test inference engine with clinical reasoning rules
4. Verify compliance with Australian TGA requirements

## 4. Performance and Scalability Testing

### Data Volume Testing
- Load 1,000+ medication concepts
- Test with 10,000+ patient records
- Validate query performance under load
- Monitor memory and CPU usage

### Integration Testing
- Test Go service ↔ GraphDB communication
- Validate FHIR resource creation/retrieval
- Test error handling and retry logic
- Verify clinical safety workflows

## 5. Next Steps for Complete Phase 3

1. **Create Sample Data Files**: Generate comprehensive clinical datasets
2. **Implement Go Integration**: Build SPARQL client in medication service
3. **Load External Terminologies**: Connect SNOMED CT, RxNorm, LOINC
4. **Test Clinical Workflows**: Validate end-to-end clinical scenarios
5. **Performance Optimization**: Add caching and query optimization
6. **Documentation**: Complete API documentation and integration guides

## 6. Expected Outcomes

After complete Phase 3 implementation:
- ✅ **10,000+ clinical triples** in KB-7 repository
- ✅ **Full FHIR integration** with Google Healthcare API
- ✅ **Clinical reasoning engine** operational
- ✅ **External terminology mappings** validated
- ✅ **Go service integration** functional
- ✅ **Performance benchmarks** met
- ✅ **Ready for Phase 4** production deployment

This represents **TRUE Phase 3 completion** with functional clinical data pipeline, not just infrastructure testing.