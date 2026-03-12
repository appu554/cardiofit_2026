#!/usr/bin/env python3
"""
Validate FHIR Store Projector Setup
Tests configuration and Google Cloud connectivity
"""

import sys
import os
sys.path.insert(0, '.')

from app.config import Config
from app.services.fhir_store_handler import FHIRStoreHandler

def main():
    print("\n" + "="*80)
    print("FHIR STORE PROJECTOR SETUP VALIDATION")
    print("="*80)

    # Test 1: Configuration
    print("\n--- Configuration ---")
    print(f"Project ID: {Config.GOOGLE_CLOUD_PROJECT_ID}")
    print(f"Location: {Config.GOOGLE_CLOUD_LOCATION}")
    print(f"Dataset: {Config.GOOGLE_CLOUD_DATASET_ID}")
    print(f"FHIR Store: {Config.GOOGLE_CLOUD_FHIR_STORE_ID}")
    print(f"Credentials: {Config.GOOGLE_APPLICATION_CREDENTIALS}")
    print(f"Full Path: {Config.get_fhir_store_path()}")

    # Test 2: Credentials file exists
    print("\n--- Credentials Check ---")
    if os.path.exists(Config.GOOGLE_APPLICATION_CREDENTIALS):
        print(f"✓ Credentials file found: {Config.GOOGLE_APPLICATION_CREDENTIALS}")
    else:
        print(f"✗ Credentials file NOT found: {Config.GOOGLE_APPLICATION_CREDENTIALS}")
        return False

    # Test 3: Initialize handler
    print("\n--- Handler Initialization ---")
    try:
        handler = FHIRStoreHandler(
            project_id=Config.GOOGLE_CLOUD_PROJECT_ID,
            location=Config.GOOGLE_CLOUD_LOCATION,
            dataset_id=Config.GOOGLE_CLOUD_DATASET_ID,
            store_id=Config.GOOGLE_CLOUD_FHIR_STORE_ID,
            credentials_path=Config.GOOGLE_APPLICATION_CREDENTIALS,
        )
        print(f"✓ FHIR Store Handler initialized")
        print(f"  Path: {handler.fhir_store_path}")
        print(f"  Supported Types: {sorted(handler.SUPPORTED_RESOURCE_TYPES)}")
    except Exception as e:
        print(f"✗ Handler initialization failed: {e}")
        return False

    # Test 4: Validate sample resource structure
    print("\n--- Sample Resource Validation ---")
    from datetime import datetime

    sample_resource = {
        "resourceType": "Observation",
        "resourceId": f"test-obs-{int(datetime.now().timestamp())}",
        "patientId": "patient-test",
        "lastUpdated": int(datetime.now().timestamp() * 1000),
        "fhirData": {
            "resourceType": "Observation",
            "id": f"test-obs-{int(datetime.now().timestamp())}",
            "status": "final",
            "code": {
                "coding": [{
                    "system": "http://loinc.org",
                    "code": "8867-4",
                    "display": "Heart rate"
                }]
            },
            "subject": {"reference": "Patient/patient-test"},
            "effectiveDateTime": datetime.now().isoformat(),
            "valueQuantity": {
                "value": 72,
                "unit": "beats/minute",
                "system": "http://unitsofmeasure.org",
                "code": "/min"
            }
        }
    }

    try:
        handler._validate_resource(
            sample_resource['resourceType'],
            sample_resource['resourceId'],
            sample_resource['fhirData']
        )
        print("✓ Sample resource validation passed")
    except Exception as e:
        print(f"✗ Sample resource validation failed: {e}")
        return False

    # Test 5: Module8-shared integration
    print("\n--- Module8-Shared Integration ---")
    try:
        from module8_shared.models.events import FHIRResource
        print("✓ FHIRResource model imported successfully")

        # Test parsing
        fhir_obj = FHIRResource(**sample_resource)
        print(f"✓ Sample resource parsed successfully")
        print(f"  Resource Type: {fhir_obj.resource_type}")
        print(f"  Resource ID: {fhir_obj.resource_id}")
    except Exception as e:
        print(f"✗ Module8-shared integration failed: {e}")
        return False

    print("\n" + "="*80)
    print("✓ ALL VALIDATION CHECKS PASSED")
    print("="*80)
    print("\nReady for testing with Google Cloud Healthcare API")
    print("Note: Actual API tests require network connectivity and valid credentials")

    return True

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
