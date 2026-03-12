"""
Unit tests for FHIR resource models.

This module contains tests for the FHIR resource models like Patient, Observation, etc.
"""

import unittest
import sys
import os
import json
from typing import Dict, List, Optional, Any
from datetime import datetime

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../"))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the models to test
from shared.models.resources import (
    Patient, Observation, Condition, Encounter
)
from shared.models.datatypes import (
    CodeableConcept, Coding, Reference, Identifier, HumanName,
    ContactPoint, Address, Quantity, Period
)

class PatientTests(unittest.TestCase):
    """Tests for the Patient resource model."""

    def setUp(self):
        """Set up test fixtures."""
        self.patient_data = {
            "resourceType": "Patient",
            "id": "patient-1",
            "active": True,
            "gender": "male",
            "birthDate": "1970-01-01",
            "name": [
                {
                    "family": "Smith",
                    "given": ["John", "Adam"],
                    "use": "official"
                }
            ],
            "telecom": [
                {
                    "system": "phone",
                    "value": "555-123-4567",
                    "use": "home"
                },
                {
                    "system": "email",
                    "value": "john.smith@example.com"
                }
            ],
            "address": [
                {
                    "line": ["123 Main St"],
                    "city": "Anytown",
                    "state": "CA",
                    "postalCode": "12345",
                    "country": "USA",
                    "use": "home"
                }
            ]
        }
        self.patient = Patient(**self.patient_data)

    def test_init(self):
        """Test Patient initialization."""
        self.assertEqual(self.patient.resourceType, "Patient")
        self.assertEqual(self.patient.id, "patient-1")
        self.assertTrue(self.patient.active)
        self.assertEqual(self.patient.gender, "male")
        self.assertEqual(self.patient.birthDate, "1970-01-01")
        self.assertEqual(len(self.patient.name), 1)
        self.assertEqual(self.patient.name[0].family, "Smith")
        self.assertEqual(self.patient.name[0].given, ["John", "Adam"])
        self.assertEqual(len(self.patient.telecom), 2)
        self.assertEqual(self.patient.telecom[0].system, "phone")
        self.assertEqual(self.patient.telecom[0].value, "555-123-4567")
        self.assertEqual(len(self.patient.address), 1)
        self.assertEqual(self.patient.address[0].city, "Anytown")

    def test_to_fhir(self):
        """Test conversion to FHIR dictionary."""
        fhir_dict = self.patient.to_fhir()
        self.assertEqual(fhir_dict["resourceType"], "Patient")
        self.assertEqual(fhir_dict["id"], "patient-1")
        self.assertTrue(fhir_dict["active"])
        self.assertEqual(fhir_dict["gender"], "male")
        self.assertEqual(fhir_dict["birthDate"], "1970-01-01")

    def test_from_fhir(self):
        """Test creation from FHIR dictionary."""
        patient = Patient.from_fhir(self.patient_data)
        self.assertEqual(patient.resourceType, "Patient")
        self.assertEqual(patient.id, "patient-1")
        self.assertTrue(patient.active)
        self.assertEqual(patient.gender, "male")
        self.assertEqual(patient.birthDate, "1970-01-01")

    def test_to_fhir_model(self):
        """Test conversion to FHIR model."""
        fhir_model = self.patient.to_fhir_model()
        # Convert to dict to check values
        fhir_dict = fhir_model.model_dump()
        self.assertEqual(fhir_dict["resourceType"], "Patient")
        self.assertEqual(fhir_dict["id"], "patient-1")
        self.assertTrue(fhir_dict["active"])
        self.assertEqual(fhir_dict["gender"], "male")
        # The birthDate might be a date object or a string, so we'll check the string representation
        self.assertEqual(str(fhir_dict["birthDate"]), "1970-01-01")

    def test_from_fhir_model(self):
        """Test creation from FHIR model."""
        fhir_model = self.patient.to_fhir_model()
        patient = Patient.from_fhir_model(fhir_model)
        self.assertEqual(patient.resourceType, "Patient")
        self.assertEqual(patient.id, "patient-1")
        self.assertTrue(patient.active)
        self.assertEqual(patient.gender, "male")
        self.assertEqual(patient.birthDate, "1970-01-01")

    def test_validate_gender(self):
        """Test gender validation."""
        # Valid gender
        patient = Patient(gender="male")
        self.assertEqual(patient.gender, "male")

        # Invalid gender
        with self.assertRaises(ValueError):
            Patient(gender="invalid")

class ObservationTests(unittest.TestCase):
    """Tests for the Observation resource model."""

    def setUp(self):
        """Set up test fixtures."""
        self.observation_data = {
            "resourceType": "Observation",
            "id": "observation-1",
            "status": "final",
            "code": {
                "coding": [
                    {
                        "system": "http://loinc.org",
                        "code": "8867-4",
                        "display": "Heart rate"
                    }
                ],
                "text": "Heart rate"
            },
            "subject": {
                "reference": "Patient/patient-1",
                "display": "John Smith"
            },
            "effectiveDateTime": "2023-07-01T12:00:00Z",
            "valueQuantity": {
                "value": 80,
                "unit": "beats/minute",
                "system": "http://unitsofmeasure.org",
                "code": "/min"
            }
        }
        self.observation = Observation(**self.observation_data)

    def test_init(self):
        """Test Observation initialization."""
        self.assertEqual(self.observation.resourceType, "Observation")
        self.assertEqual(self.observation.id, "observation-1")
        self.assertEqual(self.observation.status, "final")
        self.assertEqual(self.observation.code.coding[0].code, "8867-4")
        self.assertEqual(self.observation.subject.reference, "Patient/patient-1")
        self.assertEqual(self.observation.effectiveDateTime, "2023-07-01T12:00:00Z")
        self.assertEqual(self.observation.valueQuantity.value, 80)

    def test_to_fhir(self):
        """Test conversion to FHIR dictionary."""
        fhir_dict = self.observation.to_fhir()
        self.assertEqual(fhir_dict["resourceType"], "Observation")
        self.assertEqual(fhir_dict["id"], "observation-1")
        self.assertEqual(fhir_dict["status"], "final")
        self.assertEqual(fhir_dict["code"]["coding"][0]["code"], "8867-4")
        self.assertEqual(fhir_dict["subject"]["reference"], "Patient/patient-1")
        self.assertEqual(fhir_dict["effectiveDateTime"], "2023-07-01T12:00:00Z")
        self.assertEqual(fhir_dict["valueQuantity"]["value"], 80)

    def test_validate_status(self):
        """Test status validation."""
        # Valid status
        observation = Observation(
            status="final",
            code=CodeableConcept(
                coding=[Coding(system="http://loinc.org", code="8867-4")]
            ),
            subject=Reference(reference="Patient/1")
        )
        self.assertEqual(observation.status, "final")

        # Invalid status
        with self.assertRaises(ValueError):
            Observation(
                status="invalid",
                code=CodeableConcept(
                    coding=[Coding(system="http://loinc.org", code="8867-4")]
                ),
                subject=Reference(reference="Patient/1")
            )

class ConditionTests(unittest.TestCase):
    """Tests for the Condition resource model."""

    def setUp(self):
        """Set up test fixtures."""
        self.condition_data = {
            "resourceType": "Condition",
            "id": "condition-1",
            "clinicalStatus": {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
                        "code": "active",
                        "display": "Active"
                    }
                ]
            },
            "verificationStatus": {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
                        "code": "confirmed",
                        "display": "Confirmed"
                    }
                ]
            },
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                            "code": "problem-list-item",
                            "display": "Problem List Item"
                        }
                    ]
                }
            ],
            "severity": {
                "coding": [
                    {
                        "system": "http://snomed.info/sct",
                        "code": "24484000",
                        "display": "Severe"
                    }
                ]
            },
            "code": {
                "coding": [
                    {
                        "system": "http://snomed.info/sct",
                        "code": "44054006",
                        "display": "Diabetes mellitus type 2"
                    }
                ],
                "text": "Type 2 diabetes mellitus"
            },
            "subject": {
                "reference": "Patient/patient-1",
                "display": "John Smith"
            },
            "onsetDateTime": "2020-01-01",
            "recordedDate": "2023-07-01T12:00:00Z"
        }
        self.condition = Condition(**self.condition_data)

    def test_init(self):
        """Test Condition initialization."""
        self.assertEqual(self.condition.resourceType, "Condition")
        self.assertEqual(self.condition.id, "condition-1")
        self.assertEqual(self.condition.clinicalStatus.coding[0].code, "active")
        self.assertEqual(self.condition.verificationStatus.coding[0].code, "confirmed")
        self.assertEqual(self.condition.category[0].coding[0].code, "problem-list-item")
        self.assertEqual(self.condition.severity.coding[0].code, "24484000")
        self.assertEqual(self.condition.code.coding[0].code, "44054006")
        self.assertEqual(self.condition.subject.reference, "Patient/patient-1")
        self.assertEqual(self.condition.onsetDateTime, "2020-01-01")
        self.assertEqual(self.condition.recordedDate, "2023-07-01T12:00:00Z")

    def test_to_fhir(self):
        """Test conversion to FHIR dictionary."""
        fhir_dict = self.condition.to_fhir()
        self.assertEqual(fhir_dict["resourceType"], "Condition")
        self.assertEqual(fhir_dict["id"], "condition-1")
        self.assertEqual(fhir_dict["clinicalStatus"]["coding"][0]["code"], "active")
        self.assertEqual(fhir_dict["verificationStatus"]["coding"][0]["code"], "confirmed")
        self.assertEqual(fhir_dict["category"][0]["coding"][0]["code"], "problem-list-item")
        self.assertEqual(fhir_dict["severity"]["coding"][0]["code"], "24484000")
        self.assertEqual(fhir_dict["code"]["coding"][0]["code"], "44054006")
        self.assertEqual(fhir_dict["subject"]["reference"], "Patient/patient-1")
        self.assertEqual(fhir_dict["onsetDateTime"], "2020-01-01")
        self.assertEqual(fhir_dict["recordedDate"], "2023-07-01T12:00:00Z")

if __name__ == "__main__":
    unittest.main()
