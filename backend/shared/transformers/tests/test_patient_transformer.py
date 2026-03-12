"""
Unit tests for the Patient transformer.
"""

import unittest
from unittest.mock import MagicMock, patch
import datetime

from shared.models import Patient
from shared.transformers.fhir_to_graphql.patient import PatientTransformer, PatientType
from shared.transformers.graphql_to_fhir.patient import PatientInputTransformer, PatientInput
from shared.transformers.exceptions import TransformationError, ValidationError

class TestPatientTransformer(unittest.TestCase):
    """Test cases for the Patient transformer."""

    def setUp(self):
        """Set up test fixtures."""
        # Patch the PatientType in the transformer module
        patcher = patch('shared.transformers.fhir_to_graphql.patient.PatientType', PatientType)
        self.addCleanup(patcher.stop)
        patcher.start()

        # Create a transformer instance
        self.transformer = PatientTransformer()

        # Create a sample Patient
        self.patient = Patient(
            id="patient-1",
            resource_type="Patient",
            active=True,
            name=[
                {
                    "family": "Smith",
                    "given": ["John", "Adam"],
                    "use": "official"
                }
            ],
            telecom=[
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
            gender="male",
            birth_date="1970-01-01",
            address=[
                {
                    "line": ["123 Main St"],
                    "city": "Anytown",
                    "state": "CA",
                    "postal_code": "12345",
                    "country": "USA",
                    "use": "home"
                }
            ]
        )

    def test_transform(self):
        """Test transforming a Patient to a PatientType."""
        # Transform the Patient
        patient_type = self.transformer.transform(self.patient)

        # Check that the transformation was successful
        self.assertIsInstance(patient_type, PatientType)
        self.assertEqual(patient_type.id, "patient-1")
        self.assertEqual(patient_type.resourceType, "Patient")
        self.assertTrue(patient_type.active)
        self.assertEqual(patient_type.gender, "male")
        self.assertEqual(patient_type.birthDate, "1970-01-01")

        # Check name transformation
        self.assertEqual(len(patient_type.name), 1)
        self.assertEqual(patient_type.name[0]['familyName'], "Smith")
        self.assertEqual(patient_type.name[0]['givenName'], "John Adam")
        self.assertEqual(patient_type.name[0]['use'], "official")

        # Check telecom transformation
        self.assertEqual(len(patient_type.telecom), 2)
        self.assertEqual(patient_type.telecom[0]['system'], "phone")
        self.assertEqual(patient_type.telecom[0]['value'], "555-123-4567")
        self.assertEqual(patient_type.telecom[0]['use'], "home")
        self.assertEqual(patient_type.telecom[0]['displayValue'], "Phone: 555-123-4567")

        # Check address transformation
        self.assertEqual(len(patient_type.address), 1)
        self.assertEqual(patient_type.address[0]['streetAddress'], "123 Main St")
        self.assertEqual(patient_type.address[0]['city'], "Anytown")
        self.assertEqual(patient_type.address[0]['state'], "CA")
        self.assertEqual(patient_type.address[0]['postalCode'], "12345")
        self.assertEqual(patient_type.address[0]['country'], "USA")
        self.assertEqual(patient_type.address[0]['use'], "home")
        self.assertEqual(patient_type.address[0]['formatted'], "123 Main St\nAnytown, CA, 12345\nUSA")

    def test_transform_empty_patient(self):
        """Test transforming an empty Patient."""
        # Create an empty Patient
        empty_patient = Patient(
            id="empty-patient",
            resource_type="Patient"
        )

        # Transform the Patient
        patient_type = self.transformer.transform(empty_patient)

        # Check that the transformation was successful
        self.assertIsInstance(patient_type, PatientType)
        self.assertEqual(patient_type.id, "empty-patient")
        self.assertEqual(patient_type.resourceType, "Patient")

        # Check that optional fields are not present
        self.assertFalse(hasattr(patient_type, "name"))
        self.assertFalse(hasattr(patient_type, "telecom"))
        self.assertFalse(hasattr(patient_type, "address"))

    def test_transform_invalid_patient(self):
        """Test transforming an invalid Patient."""
        # Create an invalid object (not a Patient)
        invalid_patient = {"id": "invalid-patient"}

        # Check that transforming raises a ValidationError
        with self.assertRaises(ValidationError):
            self.transformer.transform(invalid_patient)


class TestPatientInputTransformer(unittest.TestCase):
    """Test cases for the PatientInput transformer."""

    def setUp(self):
        """Set up test fixtures."""
        # Patch the PatientInput in the transformer module
        patcher = patch('shared.transformers.graphql_to_fhir.patient.PatientInput', PatientInput)
        self.addCleanup(patcher.stop)
        patcher.start()

        # Create a transformer instance
        self.transformer = PatientInputTransformer()

        # Create a sample PatientInput
        self.patient_input = PatientInput(
            id="patient-1",
            active=True,
            name=[
                {
                    "family_name": "Smith",
                    "given_name": "John Adam",
                    "use": "official"
                }
            ],
            telecom=[
                {
                    "system": "phone",
                    "value": "555-123-4567",
                    "use": "home",
                    "display_value": "Phone: 555-123-4567"
                },
                {
                    "system": "email",
                    "value": "john.smith@example.com",
                    "display_value": "Email: john.smith@example.com"
                }
            ],
            gender="male",
            birthDate="1970-01-01",
            address=[
                {
                    "street_address": "123 Main St",
                    "city": "Anytown",
                    "state": "CA",
                    "postal_code": "12345",
                    "country": "USA",
                    "use": "home",
                    "formatted": "123 Main St\nAnytown, CA, 12345\nUSA"
                }
            ]
        )

    def test_transform(self):
        """Test transforming a PatientInput to a Patient."""
        # Transform the PatientInput
        patient = self.transformer.transform(self.patient_input)

        # Check that the transformation was successful
        self.assertIsInstance(patient, Patient)
        self.assertEqual(patient.id, "patient-1")
        self.assertEqual(patient.resource_type, "Patient")
        self.assertTrue(patient.active)
        self.assertEqual(patient.gender, "male")
        self.assertEqual(patient.birth_date, "1970-01-01")

        # Check name transformation
        self.assertEqual(len(patient.name), 1)
        self.assertEqual(patient.name[0].family, "Smith")
        self.assertEqual(patient.name[0].given, ["John", "Adam"])
        self.assertEqual(patient.name[0].use, "official")

        # Check telecom transformation
        self.assertEqual(len(patient.telecom), 2)
        self.assertEqual(patient.telecom[0].system, "phone")
        self.assertEqual(patient.telecom[0].value, "555-123-4567")
        self.assertEqual(patient.telecom[0].use, "home")

        # Check address transformation
        self.assertEqual(len(patient.address), 1)
        self.assertEqual(patient.address[0].line, ["123 Main St"])
        self.assertEqual(patient.address[0].city, "Anytown")
        self.assertEqual(patient.address[0].state, "CA")
        self.assertEqual(patient.address[0].postal_code, "12345")
        self.assertEqual(patient.address[0].country, "USA")
        self.assertEqual(patient.address[0].use, "home")

    def test_transform_empty_patient_input(self):
        """Test transforming an empty PatientInput."""
        # Create an empty PatientInput
        empty_patient_input = PatientInput(
            id="empty-patient"
        )

        # Transform the PatientInput
        patient = self.transformer.transform(empty_patient_input)

        # Check that the transformation was successful
        self.assertIsInstance(patient, Patient)
        self.assertEqual(patient.id, "empty-patient")
        self.assertEqual(patient.resource_type, "Patient")

        # Check that optional fields are not present
        self.assertFalse(hasattr(patient, "name"))
        self.assertFalse(hasattr(patient, "telecom"))
        self.assertFalse(hasattr(patient, "address"))

    def test_transform_invalid_patient_input(self):
        """Test transforming an invalid PatientInput."""
        # Create an invalid object (not a PatientInput)
        invalid_patient_input = {"id": "invalid-patient"}

        # Check that transforming raises a ValidationError
        with self.assertRaises(ValidationError):
            self.transformer.transform(invalid_patient_input)


if __name__ == '__main__':
    unittest.main()
