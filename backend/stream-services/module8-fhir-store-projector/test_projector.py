#!/usr/bin/env python3
"""
Test FHIR Store Projector with sample FHIR resources

Verifies:
1. FHIR resource validation
2. Upsert operations (CREATE and UPDATE)
3. Error handling and DLQ
4. Resource type tracking
"""

import json
import sys
import os
from datetime import datetime
from typing import Dict, Any

# Add module8-shared to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'module8-shared'))

from app.config import Config
from app.services.fhir_store_handler import FHIRStoreHandler
from app.services.projector import FHIRStoreProjector


def create_sample_observation() -> Dict[str, Any]:
    """Create sample Observation FHIR resource"""
    resource_id = f"obs-test-{int(datetime.now().timestamp())}"

    return {
        "resourceType": "Observation",
        "resourceId": resource_id,
        "patientId": "patient-12345",
        "lastUpdated": int(datetime.now().timestamp() * 1000),
        "fhirData": {
            "resourceType": "Observation",
            "id": resource_id,
            "status": "final",
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                            "code": "vital-signs",
                            "display": "Vital Signs"
                        }
                    ]
                }
            ],
            "code": {
                "coding": [
                    {
                        "system": "http://loinc.org",
                        "code": "8867-4",
                        "display": "Heart rate"
                    }
                ],
                "text": "Heart Rate"
            },
            "subject": {
                "reference": "Patient/patient-12345"
            },
            "effectiveDateTime": datetime.now().isoformat(),
            "valueQuantity": {
                "value": 72,
                "unit": "beats/minute",
                "system": "http://unitsofmeasure.org",
                "code": "/min"
            }
        }
    }


def create_sample_risk_assessment() -> Dict[str, Any]:
    """Create sample RiskAssessment FHIR resource"""
    resource_id = f"risk-test-{int(datetime.now().timestamp())}"

    return {
        "resourceType": "RiskAssessment",
        "resourceId": resource_id,
        "patientId": "patient-12345",
        "lastUpdated": int(datetime.now().timestamp() * 1000),
        "fhirData": {
            "resourceType": "RiskAssessment",
            "id": resource_id,
            "status": "final",
            "subject": {
                "reference": "Patient/patient-12345"
            },
            "occurrenceDateTime": datetime.now().isoformat(),
            "prediction": [
                {
                    "outcome": {
                        "coding": [
                            {
                                "system": "http://snomed.info/sct",
                                "code": "91302008",
                                "display": "Sepsis"
                            }
                        ],
                        "text": "Sepsis Risk"
                    },
                    "probabilityDecimal": 0.35,
                    "whenPeriod": {
                        "start": datetime.now().isoformat(),
                        "end": datetime.now().isoformat()
                    }
                }
            ]
        }
    }


def create_sample_condition() -> Dict[str, Any]:
    """Create sample Condition FHIR resource"""
    resource_id = f"cond-test-{int(datetime.now().timestamp())}"

    return {
        "resourceType": "Condition",
        "resourceId": resource_id,
        "patientId": "patient-12345",
        "lastUpdated": int(datetime.now().timestamp() * 1000),
        "fhirData": {
            "resourceType": "Condition",
            "id": resource_id,
            "clinicalStatus": {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
                        "code": "active"
                    }
                ]
            },
            "verificationStatus": {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
                        "code": "confirmed"
                    }
                ]
            },
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                            "code": "encounter-diagnosis"
                        }
                    ]
                }
            ],
            "code": {
                "coding": [
                    {
                        "system": "http://snomed.info/sct",
                        "code": "38341003",
                        "display": "Hypertension"
                    }
                ],
                "text": "Hypertension"
            },
            "subject": {
                "reference": "Patient/patient-12345"
            },
            "onsetDateTime": datetime.now().isoformat()
        }
    }


def test_handler_directly():
    """Test FHIR Store Handler directly"""
    print("\n" + "="*80)
    print("TEST 1: FHIR Store Handler Direct Test")
    print("="*80)

    # Initialize handler
    handler = FHIRStoreHandler(
        project_id=Config.GOOGLE_CLOUD_PROJECT_ID,
        location=Config.GOOGLE_CLOUD_LOCATION,
        dataset_id=Config.GOOGLE_CLOUD_DATASET_ID,
        store_id=Config.GOOGLE_CLOUD_FHIR_STORE_ID,
        credentials_path=Config.GOOGLE_APPLICATION_CREDENTIALS,
    )

    print(f"\nFHIR Store Path: {handler.fhir_store_path}")
    print(f"Supported Resource Types: {handler.SUPPORTED_RESOURCE_TYPES}")

    # Test 1: Create Observation
    print("\n--- Test 1.1: Create Observation ---")
    obs_resource = create_sample_observation()
    result = handler.upsert_resource(obs_resource)

    print(f"Operation: {result['operation']}")
    print(f"Success: {result['success']}")
    print(f"Resource: {result['resource_type']}/{result['resource_id']}")

    # Test 2: Update same Observation
    print("\n--- Test 1.2: Update same Observation (should UPDATE) ---")
    obs_resource['fhirData']['valueQuantity']['value'] = 75  # Change value
    result = handler.upsert_resource(obs_resource)

    print(f"Operation: {result['operation']}")
    print(f"Success: {result['success']}")
    print(f"Resource: {result['resource_type']}/{result['resource_id']}")

    # Test 3: Create RiskAssessment
    print("\n--- Test 1.3: Create RiskAssessment ---")
    risk_resource = create_sample_risk_assessment()
    result = handler.upsert_resource(risk_resource)

    print(f"Operation: {result['operation']}")
    print(f"Success: {result['success']}")
    print(f"Resource: {result['resource_type']}/{result['resource_id']}")

    # Test 4: Create Condition
    print("\n--- Test 1.4: Create Condition ---")
    cond_resource = create_sample_condition()
    result = handler.upsert_resource(cond_resource)

    print(f"Operation: {result['operation']}")
    print(f"Success: {result['success']}")
    print(f"Resource: {result['resource_type']}/{result['resource_id']}")

    # Print handler statistics
    print("\n--- Handler Statistics ---")
    stats = handler.get_stats()
    print(json.dumps(stats, indent=2))

    return handler


def test_validation_errors():
    """Test validation error handling"""
    print("\n" + "="*80)
    print("TEST 2: Validation Error Handling")
    print("="*80)

    handler = FHIRStoreHandler(
        project_id=Config.GOOGLE_CLOUD_PROJECT_ID,
        location=Config.GOOGLE_CLOUD_LOCATION,
        dataset_id=Config.GOOGLE_CLOUD_DATASET_ID,
        store_id=Config.GOOGLE_CLOUD_FHIR_STORE_ID,
        credentials_path=Config.GOOGLE_APPLICATION_CREDENTIALS,
    )

    # Test 1: Unsupported resource type
    print("\n--- Test 2.1: Unsupported Resource Type ---")
    try:
        invalid_resource = {
            "resourceType": "InvalidType",
            "resourceId": "invalid-123",
            "patientId": "patient-12345",
            "fhirData": {
                "resourceType": "InvalidType",
                "id": "invalid-123"
            }
        }
        result = handler.upsert_resource(invalid_resource)
        print(f"Result: {result}")
    except ValueError as e:
        print(f"Caught expected validation error: {e}")

    # Test 2: Missing resourceId
    print("\n--- Test 2.2: Missing resourceId ---")
    try:
        invalid_resource = {
            "resourceType": "Observation",
            "resourceId": None,
            "patientId": "patient-12345",
            "fhirData": {"resourceType": "Observation"}
        }
        result = handler.upsert_resource(invalid_resource)
        print(f"Result: {result}")
    except ValueError as e:
        print(f"Caught expected validation error: {e}")

    # Test 3: ResourceType mismatch
    print("\n--- Test 2.3: ResourceType Mismatch ---")
    try:
        invalid_resource = {
            "resourceType": "Observation",
            "resourceId": "obs-123",
            "patientId": "patient-12345",
            "fhirData": {
                "resourceType": "Condition",  # Mismatch!
                "id": "obs-123"
            }
        }
        result = handler.upsert_resource(invalid_resource)
        print(f"Result: {result}")
    except ValueError as e:
        print(f"Caught expected validation error: {e}")

    print("\n--- Validation Statistics ---")
    stats = handler.get_stats()
    print(f"Validation Errors: {stats['validation_errors']}")


def test_projector_batch_processing():
    """Test projector batch processing"""
    print("\n" + "="*80)
    print("TEST 3: Projector Batch Processing")
    print("="*80)

    # Build configuration
    config = {
        'kafka': Config.get_kafka_config(),
        'topics': {
            'fhir_upsert': Config.KAFKA_TOPIC_FHIR_UPSERT,
            'dlq': Config.KAFKA_TOPIC_DLQ,
        },
        'batch_size': Config.BATCH_SIZE,
        'batch_timeout_seconds': Config.BATCH_TIMEOUT_SECONDS,
        'fhir_store': {
            'project_id': Config.GOOGLE_CLOUD_PROJECT_ID,
            'location': Config.GOOGLE_CLOUD_LOCATION,
            'dataset_id': Config.GOOGLE_CLOUD_DATASET_ID,
            'store_id': Config.GOOGLE_CLOUD_FHIR_STORE_ID,
            'credentials_path': Config.GOOGLE_APPLICATION_CREDENTIALS,
            'max_retries': Config.RETRY_MAX_ATTEMPTS,
            'retry_backoff_factor': Config.RETRY_BACKOFF_FACTOR,
        },
    }

    projector = FHIRStoreProjector(config)

    # Create batch of messages
    batch = [
        create_sample_observation(),
        create_sample_risk_assessment(),
        create_sample_condition(),
    ]

    print(f"\nProcessing batch of {len(batch)} resources...")

    # Process batch
    projector.process_batch(batch)

    # Print projector statistics
    print("\n--- Projector Statistics ---")
    summary = projector.get_processing_summary()
    print(json.dumps(summary, indent=2, default=str))


def main():
    """Run all tests"""
    print("\n" + "="*80)
    print("FHIR STORE PROJECTOR TEST SUITE")
    print("="*80)
    print(f"\nConfiguration:")
    print(f"  Project ID: {Config.GOOGLE_CLOUD_PROJECT_ID}")
    print(f"  Location: {Config.GOOGLE_CLOUD_LOCATION}")
    print(f"  Dataset: {Config.GOOGLE_CLOUD_DATASET_ID}")
    print(f"  FHIR Store: {Config.GOOGLE_CLOUD_FHIR_STORE_ID}")
    print(f"  Credentials: {Config.GOOGLE_APPLICATION_CREDENTIALS}")

    try:
        # Test 1: Handler direct test
        handler = test_handler_directly()

        # Test 2: Validation errors
        test_validation_errors()

        # Test 3: Projector batch processing
        test_projector_batch_processing()

        print("\n" + "="*80)
        print("ALL TESTS COMPLETED SUCCESSFULLY")
        print("="*80)

        # Final statistics
        print("\n--- Final Handler Statistics ---")
        final_stats = handler.get_stats()
        print(json.dumps(final_stats, indent=2))

        print("\n--- Resource Type Breakdown ---")
        for resource_type, count in final_stats['resource_type_counts'].items():
            print(f"  {resource_type}: {count}")

        print(f"\nTotal Upserts: {final_stats['total_upserts']}")
        print(f"Success Rate: {final_stats['success_rate']:.2%}")

    except Exception as e:
        print(f"\n❌ TEST FAILED: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
