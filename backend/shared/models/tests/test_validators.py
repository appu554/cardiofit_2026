"""
Unit tests for FHIR validators.

This module contains tests for the FHIR validation functionality.
"""

import unittest
import sys
import os
import json
from typing import Dict, List, Optional, Any

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the validators to test
from shared.models.validators import validate_fhir_resource
from shared.models.validators.fhir import FHIRValidationError

class ValidatorTests(unittest.TestCase):
    """Tests for the FHIR validators."""

    def setUp(self):
        """Set up test fixtures."""
        self.valid_patient = {
            "resourceType": "Patient",
            "id": "patient-1",
            "active": True,
            "gender": "male",
            "birthDate": "1970-01-01",
            "name": [
                {
                    "family": "Smith",
                    "given": ["John"]
                }
            ]
        }
        
        self.invalid_patient = {
            "resourceType": "Patient",
            "id": "patient-1",
            "active": True,
            "gender": "invalid",  # Invalid gender
            "birthDate": "1970-01-01",
            "name": [
                {
                    "family": "Smith",
                    "given": ["John"]
                }
            ]
        }
        
        self.valid_observation = {
            "resourceType": "Observation",
            "status": "final",
            "code": {
                "coding": [
                    {
                        "system": "http://loinc.org",
                        "code": "8867-4",
                        "display": "Heart rate"
                    }
                ]
            },
            "subject": {
                "reference": "Patient/patient-1"
            },
            "valueQuantity": {
                "value": 80,
                "unit": "beats/minute"
            }
        }
        
        self.invalid_observation = {
            "resourceType": "Observation",
            "status": "invalid",  # Invalid status
            "code": {
                "coding": [
                    {
                        "system": "http://loinc.org",
                        "code": "8867-4",
                        "display": "Heart rate"
                    }
                ]
            },
            "subject": {
                "reference": "Patient/patient-1"
            },
            "valueQuantity": {
                "value": 80,
                "unit": "beats/minute"
            }
        }

    def test_validate_valid_patient(self):
        """Test validation of a valid Patient resource."""
        try:
            result = validate_fhir_resource(self.valid_patient, "Patient")
            self.assertEqual(result["resourceType"], "Patient")
            self.assertEqual(result["id"], "patient-1")
        except FHIRValidationError as e:
            self.fail(f"validate_fhir_resource raised FHIRValidationError unexpectedly: {e}")

    def test_validate_invalid_patient(self):
        """Test validation of an invalid Patient resource."""
        with self.assertRaises(FHIRValidationError):
            validate_fhir_resource(self.invalid_patient, "Patient")

    def test_validate_valid_observation(self):
        """Test validation of a valid Observation resource."""
        try:
            result = validate_fhir_resource(self.valid_observation, "Observation")
            self.assertEqual(result["resourceType"], "Observation")
            self.assertEqual(result["status"], "final")
        except FHIRValidationError as e:
            self.fail(f"validate_fhir_resource raised FHIRValidationError unexpectedly: {e}")

    def test_validate_invalid_observation(self):
        """Test validation of an invalid Observation resource."""
        with self.assertRaises(FHIRValidationError):
            validate_fhir_resource(self.invalid_observation, "Observation")

    def test_validate_missing_resource_type(self):
        """Test validation with missing resource type."""
        data = self.valid_patient.copy()
        del data["resourceType"]
        
        with self.assertRaises(FHIRValidationError):
            validate_fhir_resource(data)

    def test_validate_unknown_resource_type(self):
        """Test validation with unknown resource type."""
        data = {
            "resourceType": "UnknownResource",
            "id": "test-1"
        }
        
        with self.assertRaises(FHIRValidationError):
            validate_fhir_resource(data)

if __name__ == "__main__":
    unittest.main()
