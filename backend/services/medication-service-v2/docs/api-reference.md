# Medication Service V2 API Reference

Complete API reference for the Go/Rust Medication Service V2 implementing the Recipe & Snapshot architecture pattern.

## Base URL
```
http://localhost:8005/api/v1
```

## Authentication
All API endpoints require JWT authentication via the `Authorization` header:
```http
Authorization: Bearer <jwt_token>
```

## API Overview

### Core Workflow APIs
- [Medication Proposals](#medication-proposals) - 4-phase workflow for medication recommendations
- [Recipe Resolution](#recipe-resolution) - Internal recipe management and resolution
- [Clinical Snapshots](#clinical-snapshots) - Immutable clinical data snapshots
- [Clinical Intelligence](#clinical-intelligence) - Advanced clinical reasoning and scoring

### Supporting APIs
- [Knowledge Base](#knowledge-base) - Clinical knowledge and guidelines access
- [Formulary](#formulary) - Medication formulary and coverage information
- [Health & Monitoring](#health--monitoring) - Service health and metrics

---

## Medication Proposals

### POST /medications/propose
Creates a complete medication proposal using the 4-phase workflow.

#### Request Body
```json
{
  "patient_id": "string",
  "indication": "string",
  "clinical_context": {
    "weight_kg": 70.5,
    "height_cm": 175,
    "age_years": 45,
    "gender": "M|F",
    "pregnancy_status": "boolean",
    "additional_context": {}
  },
  "preferences": {
    "route": "PO|IV|IM|SC",
    "frequency": "daily|BID|TID|QID",
    "duration_days": 30,
    "avoid_generic": false
  },
  "constraints": {
    "max_cost": 100.0,
    "formulary_preferred": true,
    "avoid_interactions": true
  }
}
```

#### Response
```json
{
  "proposal_id": "prop_01HG7XJKM9...",
  "patient_id": "patient-123",
  "status": "PROPOSED",
  "processing_time_ms": 185,
  "proposals": [
    {
      "rank": 1,
      "medication": {
        "medication_id": "med_456",
        "rxnorm_code": "123456",
        "generic_name": "lisinopril",
        "brand_name": "Prinivil",
        "strength": "10mg",
        "dosage_form": "tablet"
      },
      "dose": {
        "value": 10.0,
        "unit": "mg",
        "frequency": "daily",
        "route": "PO"
      },
      "calculation_details": {
        "method": "fixed_dose",
        "factors": {
          "indication": "hypertension",
          "patient_weight": 70.5
        }
      },
      "scoring": {
        "total_score": 8.5,
        "efficacy_score": 9.0,
        "safety_score": 8.0,
        "cost_score": 8.5,
        "formulary_score": 9.0
      },
      "formulary_info": {
        "tier": 1,
        "copay_estimate": 5.0,
        "prior_auth_required": false,
        "quantity_limit": null
      },
      "safety_alerts": [
        {
          "level": "INFO",
          "message": "Monitor potassium levels",
          "category": "laboratory_monitoring"
        }
      ]
    }
  ],
  "snapshot_reference": {
    "snapshot_id": "snap_01HG7XJKM9...",
    "checksum": "sha256:abc123...",
    "created_at": "2024-01-15T10:30:00Z",
    "expires_at": "2024-01-16T10:30:00Z"
  },
  "evidence_envelope": {
    "snapshot_id": "snap_01HG7XJKM9...",
    "kb_versions": {
      "drug_rules": "v2.1.0",
      "guidelines": "v1.8.2"
    },
    "protocol_used": "hypertension-standard-v1.2",
    "generated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Error Responses
```json
{
  "error": {
    "code": "INVALID_PATIENT_CONTEXT",
    "message": "Patient weight is required for dose calculation",
    "details": {
      "missing_fields": ["weight_kg"],
      "validation_errors": []
    }
  }
}
```

### GET /medications/proposals/{proposal_id}
Retrieves an existing medication proposal.

#### Response
```json
{
  "proposal_id": "prop_01HG7XJKM9...",
  "status": "PROPOSED|COMMITTED|CANCELLED",
  "created_at": "2024-01-15T10:30:00Z",
  "committed_at": null,
  "committed_by": null,
  // ... same structure as POST response
}
```

### POST /medications/proposals/{proposal_id}/commit
Commits a medication proposal for implementation.

#### Request Body
```json
{
  "committed_by": "provider_123",
  "selected_proposal_rank": 1,
  "modifications": {
    "dose_adjustment": 0.8,
    "frequency_change": "BID",
    "duration_days": 14
  },
  "commit_context": {
    "encounter_id": "enc_789",
    "prescription_context": {}
  }
}
```

#### Response
```json
{
  "prescription_id": "rx_01HG7XJKM9...",
  "proposal_id": "prop_01HG7XJKM9...",
  "status": "COMMITTED",
  "committed_at": "2024-01-15T10:35:00Z",
  "committed_by": "provider_123",
  "final_prescription": {
    "medication": {...},
    "dose": {...},
    "instructions": "Take 10mg daily with food"
  }
}
```

---

## Recipe Resolution

### POST /recipes/resolve
Resolves a workflow recipe for a specific clinical protocol.

#### Request Body
```json
{
  "protocol_id": "hypertension-standard",
  "context_needs": {
    "calculation_fields": ["weight", "age", "gender"],
    "safety_fields": ["allergies", "conditions", "current_medications"],
    "audit_fields": ["provider", "encounter"],
    "conditional_requirements": {
      "pediatric": "age < 18",
      "pregnancy": "pregnancy_status == true",
      "renal_impairment": "creatinine_clearance < 60"
    }
  },
  "patient_characteristics": {
    "age_years": 45,
    "pregnancy_status": false,
    "renal_function": "normal"
  }
}
```

#### Response
```json
{
  "recipe": {
    "recipe_id": "hypertension-standard_1705320600",
    "protocol_id": "hypertension-standard",
    "version": "1.2",
    "required_fields": [
      "demographics.weight_kg",
      "demographics.height_cm",
      "demographics.age_years",
      "vitals.blood_pressure",
      "allergies.drug_allergies",
      "conditions.current_conditions",
      "medications.current_medications"
    ],
    "freshness_requirements": {
      "vitals": "24h",
      "medications": "7d",
      "allergies": "30d"
    },
    "ttl_seconds": 3600,
    "allow_live_fetch": false,
    "allowed_live_fields": [],
    "conditional_fields": {
      "applied": ["diabetes_monitoring"],
      "skipped": ["pediatric_dosing", "pregnancy_monitoring"]
    }
  },
  "resolution_metadata": {
    "resolution_time_ms": 8,
    "cache_hit": false,
    "rules_evaluated": 12,
    "conditions_matched": ["adult", "normal_renal_function"]
  }
}
```

### GET /recipes/{recipe_id}
Retrieves a previously resolved recipe.

### GET /recipes/protocols
Lists available clinical protocols.

#### Response
```json
{
  "protocols": [
    {
      "protocol_id": "hypertension-standard",
      "name": "Standard Hypertension Management",
      "version": "1.2",
      "description": "Evidence-based hypertension treatment protocol",
      "categories": ["cardiovascular", "primary_care"],
      "supported_indications": ["essential_hypertension", "stage1_hypertension"],
      "last_updated": "2024-01-01T00:00:00Z"
    }
  ],
  "total_count": 25
}
```

---

## Clinical Snapshots

### POST /snapshots/create
Creates an immutable clinical snapshot for a patient.

#### Request Body
```json
{
  "patient_id": "patient-123",
  "recipe": {
    "recipe_id": "hypertension-standard_1705320600",
    "required_fields": [...],
    "freshness_requirements": {...}
  },
  "snapshot_options": {
    "include_historical": true,
    "data_sources": ["ehr", "fhir_store", "external_labs"],
    "validation_level": "strict"
  }
}
```

#### Response
```json
{
  "snapshot": {
    "snapshot_id": "snap_01HG7XJKM9...",
    "patient_id": "patient-123",
    "created_at": "2024-01-15T10:30:00Z",
    "expires_at": "2024-01-16T10:30:00Z",
    "checksum": "sha256:abc123def456...",
    "signature": "ed25519:signature_data...",
    "included_fields": [
      "demographics.weight_kg",
      "demographics.age_years",
      "vitals.blood_pressure",
      "allergies.drug_allergies"
    ],
    "data": {
      "demographics": {
        "weight_kg": 70.5,
        "height_cm": 175,
        "age_years": 45,
        "gender": "M"
      },
      "vitals": {
        "blood_pressure": {
          "systolic": 145,
          "diastolic": 92,
          "recorded_at": "2024-01-15T08:00:00Z"
        }
      },
      "allergies": {
        "drug_allergies": [
          {
            "drug": "penicillin",
            "reaction": "rash",
            "severity": "mild"
          }
        ]
      }
    },
    "metadata": {
      "data_sources_used": ["ehr_primary", "fhir_store"],
      "freshness_status": {
        "vitals": "fresh",
        "medications": "stale",
        "allergies": "fresh"
      },
      "validation_results": {
        "all_required_present": true,
        "data_quality_score": 0.95
      }
    }
  }
}
```

### GET /snapshots/{snapshot_id}
Retrieves an existing clinical snapshot.

### POST /snapshots/{snapshot_id}/validate
Validates snapshot integrity and freshness.

#### Response
```json
{
  "snapshot_id": "snap_01HG7XJKM9...",
  "validation": {
    "is_valid": true,
    "is_expired": false,
    "checksum_valid": true,
    "signature_valid": true,
    "freshness_check": {
      "all_fresh": false,
      "stale_fields": ["medications.current_medications"],
      "expired_fields": []
    }
  },
  "recommended_action": "PROCEED|REFRESH|REJECT"
}
```

---

## Clinical Intelligence

### POST /intelligence/analyze
Performs clinical intelligence analysis on patient data.

#### Request Body
```json
{
  "patient_data": {
    "demographics": {...},
    "conditions": [...],
    "medications": [...],
    "allergies": [...]
  },
  "analysis_type": "drug_interactions|contraindications|dosing_optimization",
  "context": {
    "indication": "hypertension",
    "proposed_medications": ["lisinopril", "hydrochlorothiazide"]
  }
}
```

#### Response
```json
{
  "analysis_results": {
    "drug_interactions": [
      {
        "drug1": "lisinopril",
        "drug2": "potassium_supplement",
        "interaction_type": "pharmacodynamic",
        "severity": "moderate",
        "mechanism": "Both increase potassium levels",
        "management": "Monitor serum potassium levels closely",
        "clinical_significance": "May cause hyperkalemia"
      }
    ],
    "contraindications": [],
    "dosing_recommendations": [
      {
        "medication": "lisinopril",
        "recommendation": "Start with 5mg daily, titrate up",
        "rationale": "Patient age and renal function normal",
        "evidence_level": "A"
      }
    ]
  },
  "risk_assessment": {
    "overall_risk": "low",
    "risk_factors": ["age > 40", "hypertension"],
    "monitoring_recommendations": [
      "Check BP in 2 weeks",
      "Monitor potassium at 1 month"
    ]
  }
}
```

### POST /intelligence/score
Scores and ranks medication options.

#### Request Body
```json
{
  "candidates": [
    {
      "medication_id": "med_456",
      "dose": {...},
      "route": "PO",
      "frequency": "daily"
    }
  ],
  "patient_data": {...},
  "scoring_criteria": {
    "efficacy_weight": 0.4,
    "safety_weight": 0.3,
    "cost_weight": 0.2,
    "formulary_weight": 0.1
  }
}
```

#### Response
```json
{
  "scored_candidates": [
    {
      "rank": 1,
      "medication_id": "med_456",
      "total_score": 8.5,
      "component_scores": {
        "efficacy": 9.0,
        "safety": 8.0,
        "cost": 8.5,
        "formulary": 9.0
      },
      "scoring_details": {
        "efficacy_factors": ["indication_match", "dose_appropriateness"],
        "safety_factors": ["no_contraindications", "minimal_interactions"],
        "cost_factors": ["generic_available", "formulary_tier_1"],
        "formulary_factors": ["preferred_status", "no_pa_required"]
      }
    }
  ]
}
```

---

## Knowledge Base

### GET /knowledge/medications/{rxnorm_code}
Retrieves detailed medication information.

#### Response
```json
{
  "medication": {
    "rxnorm_code": "123456",
    "generic_name": "lisinopril",
    "brand_names": ["Prinivil", "Zestril"],
    "therapeutic_class": "ACE Inhibitor",
    "pharmacologic_class": "Angiotensin Converting Enzyme Inhibitor",
    "indications": ["hypertension", "heart_failure", "post_mi"],
    "contraindications": ["pregnancy", "angioedema_history"],
    "dosing_info": {
      "adult_dose_range": "5-40mg daily",
      "pediatric_dose": "0.1mg/kg daily",
      "renal_adjustment": true,
      "hepatic_adjustment": false
    },
    "monitoring_parameters": ["blood_pressure", "potassium", "creatinine"],
    "drug_interactions": [...],
    "side_effects": [...],
    "last_updated": "2024-01-01T00:00:00Z"
  }
}
```

### GET /knowledge/guidelines
Searches clinical guidelines and evidence.

#### Query Parameters
- `indication`: Clinical indication (e.g., "hypertension")
- `medication`: Medication name or RxNorm code
- `patient_population`: Patient characteristics (e.g., "adult", "pediatric")
- `limit`: Number of results (default: 10)

#### Response
```json
{
  "guidelines": [
    {
      "guideline_id": "acc_aha_hypertension_2023",
      "title": "2023 ACC/AHA Hypertension Guidelines",
      "organization": "American College of Cardiology",
      "version": "2023.1",
      "recommendations": [
        {
          "recommendation": "First-line therapy should include thiazide diuretics, ACE inhibitors, ARBs, or CCBs",
          "strength": "Class I",
          "evidence_level": "A",
          "applicable_populations": ["adults", "stage_1_hypertension"]
        }
      ],
      "last_updated": "2023-11-15T00:00:00Z"
    }
  ],
  "total_count": 12
}
```

### POST /knowledge/interactions
Checks for drug interactions.

#### Request Body
```json
{
  "medications": [
    {"rxnorm_code": "123456", "name": "lisinopril"},
    {"rxnorm_code": "789012", "name": "hydrochlorothiazide"}
  ],
  "patient_factors": {
    "age": 45,
    "gender": "M",
    "conditions": ["hypertension"],
    "renal_function": "normal"
  }
}
```

#### Response
```json
{
  "interactions": [
    {
      "drug1": "lisinopril",
      "drug2": "hydrochlorothiazide",
      "interaction_type": "synergistic",
      "severity": "beneficial",
      "mechanism": "Complementary antihypertensive effects",
      "clinical_management": "Commonly used in combination",
      "evidence_level": "well_established"
    }
  ],
  "interaction_summary": {
    "total_interactions": 1,
    "severe_count": 0,
    "moderate_count": 0,
    "minor_count": 0,
    "beneficial_count": 1
  }
}
```

---

## Formulary

### GET /formulary/coverage
Checks medication coverage and formulary status.

#### Query Parameters
- `medication_id`: Medication identifier
- `rxnorm_code`: RxNorm code
- `insurance_plan`: Insurance plan identifier
- `pharmacy_id`: Specific pharmacy (optional)

#### Response
```json
{
  "coverage": {
    "medication_id": "med_456",
    "rxnorm_code": "123456",
    "formulary_status": "preferred",
    "tier": 1,
    "copay_estimate": 5.0,
    "prior_authorization": {
      "required": false,
      "criteria": null
    },
    "quantity_limits": {
      "limit": "30 tablets per 30 days",
      "override_available": true
    },
    "step_therapy": {
      "required": false,
      "alternatives": []
    },
    "generic_available": true,
    "brand_alternatives": ["Prinivil", "Zestril"]
  },
  "cost_information": {
    "retail_price": 25.99,
    "insurance_copay": 5.00,
    "patient_responsibility": 5.00,
    "deductible_applied": 0.00
  }
}
```

### GET /formulary/alternatives
Finds therapeutic alternatives for a medication.

#### Query Parameters
- `medication_id`: Current medication
- `reason`: Reason for alternative (cost, formulary, allergy, interaction)
- `insurance_plan`: Patient's insurance plan

#### Response
```json
{
  "alternatives": [
    {
      "medication": {
        "medication_id": "med_789",
        "generic_name": "enalapril",
        "therapeutic_equivalence": "AB",
        "same_class": true
      },
      "formulary_comparison": {
        "original_tier": 3,
        "alternative_tier": 1,
        "cost_savings": 15.00
      },
      "conversion_info": {
        "dose_conversion": "lisinopril 10mg = enalapril 10mg",
        "conversion_ratio": 1.0,
        "monitoring_changes": []
      },
      "evidence_level": "established",
      "switching_considerations": [
        "No dose adjustment needed",
        "Same monitoring parameters"
      ]
    }
  ],
  "recommendation": {
    "preferred_alternative": "med_789",
    "rationale": "Lower formulary tier with equivalent efficacy",
    "confidence_level": "high"
  }
}
```

---

## Health & Monitoring

### GET /health/live
Liveness probe endpoint.

#### Response
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "medication-service-v2",
  "version": "1.0.0"
}
```

### GET /health/ready
Readiness probe endpoint.

#### Response
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "dependencies": {
    "database": "healthy",
    "redis": "healthy",
    "rust_engine": "healthy",
    "context_gateway": "healthy",
    "knowledge_bases": "healthy"
  },
  "checks_passed": 5,
  "checks_total": 5
}
```

### GET /health/deps
Detailed dependency health check.

#### Response
```json
{
  "dependencies": [
    {
      "name": "postgresql",
      "status": "healthy",
      "response_time_ms": 12,
      "last_check": "2024-01-15T10:30:00Z",
      "details": {
        "connection_pool": "8/20 connections",
        "query_performance": "normal"
      }
    },
    {
      "name": "rust_clinical_engine",
      "status": "healthy",
      "response_time_ms": 8,
      "last_check": "2024-01-15T10:30:00Z",
      "details": {
        "grpc_status": "serving",
        "calculation_performance": "optimal"
      }
    }
  ]
}
```

### GET /metrics
Prometheus metrics endpoint.

#### Response (Prometheus format)
```
# HELP medication_v2_requests_total Total number of API requests
# TYPE medication_v2_requests_total counter
medication_v2_requests_total{method="POST",endpoint="/medications/propose",status="200"} 1523

# HELP medication_v2_request_duration_seconds Request duration in seconds
# TYPE medication_v2_request_duration_seconds histogram
medication_v2_request_duration_seconds_bucket{method="POST",endpoint="/medications/propose",le="0.1"} 1200
medication_v2_request_duration_seconds_bucket{method="POST",endpoint="/medications/propose",le="0.25"} 1500
medication_v2_request_duration_seconds_sum{method="POST",endpoint="/medications/propose"} 187.5
medication_v2_request_duration_seconds_count{method="POST",endpoint="/medications/propose"} 1523

# HELP medication_v2_clinical_calculations_total Total clinical calculations performed
# TYPE medication_v2_clinical_calculations_total counter
medication_v2_clinical_calculations_total{type="weight_based"} 856
medication_v2_clinical_calculations_total{type="fixed_dose"} 667
```

---

## Error Handling

### Standard Error Response Format
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": {
      "field_errors": [...],
      "validation_errors": [...],
      "additional_context": {...}
    },
    "trace_id": "01HG7XJKM9...",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `INVALID_REQUEST` | Request body validation failed |
| 400 | `INVALID_PATIENT_CONTEXT` | Required patient data missing or invalid |
| 400 | `INVALID_MEDICATION` | Medication not found or invalid |
| 401 | `UNAUTHORIZED` | Authentication token missing or invalid |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `RESOURCE_NOT_FOUND` | Requested resource not found |
| 409 | `PROPOSAL_ALREADY_COMMITTED` | Proposal cannot be modified |
| 422 | `CLINICAL_CONTRAINDICATION` | Clinical safety check failed |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
| 502 | `DEPENDENCY_ERROR` | External service unavailable |
| 503 | `SERVICE_UNAVAILABLE` | Service temporarily unavailable |

---

## Rate Limiting

API requests are subject to rate limiting:

- **Default Limit**: 1000 requests per minute per API key
- **Burst Allowance**: Up to 100 requests in a 10-second window
- **Headers**: Rate limit information included in response headers
  - `X-RateLimit-Limit`: Requests allowed per window
  - `X-RateLimit-Remaining`: Requests remaining in current window
  - `X-RateLimit-Reset`: Unix timestamp when window resets

## Versioning

The API uses URL path versioning:
- Current version: `v1`
- Future versions: `v2`, `v3`, etc.
- Backward compatibility maintained for at least one major version

## Support

For API support and questions:
- Technical documentation: [docs/](docs/)
- Issue tracking: Project issue tracker
- Performance issues: Check [/health/deps](#get-healthdeps) endpoint first