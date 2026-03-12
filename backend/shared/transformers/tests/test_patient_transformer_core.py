"""
Unit tests for the Patient transformer core functionality.

These tests focus only on the transformation logic without requiring
full GraphQL type integration, which is part of Phase 4.
"""

import unittest
from unittest.mock import MagicMock, patch
import datetime

from shared.models import Patient
from shared.transformers.base import BaseTransformer
from shared.transformers.exceptions import TransformationError, ValidationError

# Create a simple dictionary-based transformer for testing Phase 2 functionality
class DictTransformer(BaseTransformer):
    """Base class for dictionary-based transformers for testing."""

    def _validate_source(self, source_data):
        """Validate the source data."""
        if not isinstance(source_data, self.source_type):
            raise ValidationError(f"Expected {self.source_type.__name__}, got {type(source_data).__name__}")

    def _validate_target(self, target_data):
        """Validate the target data."""
        if not isinstance(target_data, dict):
            raise ValidationError(f"Expected dict, got {type(target_data).__name__}")

    def _transform(self, source_data):
        """Transform the source data to a dictionary."""
        # Convert to dictionary
        if hasattr(source_data, "model_dump"):
            # Pydantic v2
            data = source_data.model_dump()
        elif hasattr(source_data, "dict"):
            # Pydantic v1
            data = source_data.dict()
        else:
            # Fallback
            data = dict(source_data)

        # Transform nested objects
        data = self._transform_nested_objects(data)

        return data

    def _transform_nested_objects(self, data):
        """Transform nested objects in the data."""
        return data

class PatientToDictTransformer(DictTransformer):
    """Transformer for Patient to dictionary."""

    source_type = Patient
    target_type = dict

    def _transform_nested_objects(self, data):
        """Transform nested objects in the Patient data."""
        # Remove None values to clean up the output
        data = {k: v for k, v in data.items() if v is not None}

        # Transform name
        if 'name' in data and data['name']:
            transformed_names = []
            for name in data['name']:
                transformed_name = dict(name)

                # Handle given names
                if 'given' in transformed_name and isinstance(transformed_name['given'], list):
                    transformed_name['given_name'] = ' '.join(transformed_name['given'])
                    del transformed_name['given']

                # Rename family
                if 'family' in transformed_name:
                    transformed_name['family_name'] = transformed_name['family']
                    del transformed_name['family']

                transformed_names.append(transformed_name)

            data['name'] = transformed_names
        elif 'name' in data:
            # Remove None values
            del data['name']

        # Transform telecom
        if 'telecom' in data and data['telecom']:
            transformed_telecoms = []
            for telecom in data['telecom']:
                transformed_telecom = dict(telecom)

                # Add a display_value field
                if 'system' in telecom and 'value' in telecom:
                    system = telecom['system'].capitalize() if telecom['system'] else ''
                    transformed_telecom['display_value'] = f"{system}: {telecom['value']}"

                transformed_telecoms.append(transformed_telecom)

            data['telecom'] = transformed_telecoms
        elif 'telecom' in data:
            # Remove None values
            del data['telecom']

        # Transform address
        if 'address' in data and data['address']:
            transformed_addresses = []
            for address in data['address']:
                transformed_address = dict(address)

                # Handle address lines
                if 'line' in transformed_address and isinstance(transformed_address['line'], list):
                    transformed_address['street_address'] = '\n'.join(transformed_address['line'])
                    del transformed_address['line']

                # Add a formatted field
                formatted_parts = []

                if 'street_address' in transformed_address:
                    formatted_parts.append(transformed_address['street_address'])

                city_state_zip = []
                if 'city' in transformed_address:
                    city_state_zip.append(transformed_address['city'])
                if 'state' in transformed_address:
                    city_state_zip.append(transformed_address['state'])
                if 'postal_code' in transformed_address:
                    city_state_zip.append(transformed_address['postal_code'])

                if city_state_zip:
                    formatted_parts.append(', '.join(city_state_zip))

                if 'country' in transformed_address:
                    formatted_parts.append(transformed_address['country'])

                transformed_address['formatted'] = '\n'.join(formatted_parts)

                transformed_addresses.append(transformed_address)

            data['address'] = transformed_addresses
        elif 'address' in data:
            # Remove None values
            del data['address']

        return data

class DictToPatientTransformer(DictTransformer):
    """Transformer for dictionary to Patient."""

    source_type = dict
    target_type = Patient

    def _validate_source(self, source_data):
        """Validate the source data."""
        if not isinstance(source_data, dict):
            raise ValidationError(f"Expected dict, got {type(source_data).__name__}")

    def _transform(self, source_data):
        """Transform the dictionary to a Patient."""
        # Create a copy of the data
        data = dict(source_data)

        # Transform nested objects
        data = self._transform_nested_objects(data)

        # Create a Patient instance
        return Patient(**data)

    def _transform_nested_objects(self, data):
        """Transform nested objects in the dictionary data."""
        # Transform name
        if 'name' in data and data['name']:
            transformed_names = []
            for name in data['name']:
                transformed_name = dict(name)

                # Handle given name
                if 'given_name' in transformed_name:
                    # Split the given name by spaces
                    transformed_name['given'] = transformed_name['given_name'].split()
                    del transformed_name['given_name']

                # Rename family_name
                if 'family_name' in transformed_name:
                    transformed_name['family'] = transformed_name['family_name']
                    del transformed_name['family_name']

                transformed_names.append(transformed_name)

            data['name'] = transformed_names

        # Transform telecom
        if 'telecom' in data and data['telecom']:
            transformed_telecoms = []
            for telecom in data['telecom']:
                transformed_telecom = dict(telecom)

                # Remove display_value field
                if 'display_value' in transformed_telecom:
                    del transformed_telecom['display_value']

                transformed_telecoms.append(transformed_telecom)

            data['telecom'] = transformed_telecoms

        # Transform address
        if 'address' in data and data['address']:
            transformed_addresses = []
            for address in data['address']:
                transformed_address = dict(address)

                # Handle street address
                if 'street_address' in transformed_address:
                    # Split the street address by newlines
                    transformed_address['line'] = transformed_address['street_address'].split('\n')
                    del transformed_address['street_address']

                # Remove formatted field
                if 'formatted' in transformed_address:
                    del transformed_address['formatted']

                transformed_addresses.append(transformed_address)

            data['address'] = transformed_addresses

        return data

class TestPatientToDictTransformer(unittest.TestCase):
    """Test cases for the Patient to dictionary transformer."""

    def setUp(self):
        """Set up test fixtures."""
        # Create a transformer instance
        self.transformer = PatientToDictTransformer()

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
        """Test transforming a Patient to a dictionary."""
        # Transform the Patient
        result = self.transformer.transform(self.patient)

        # Check that the transformation was successful
        self.assertIsInstance(result, dict)
        self.assertEqual(result["id"], "patient-1")
        self.assertEqual(result["resource_type"], "Patient")
        self.assertTrue(result["active"])
        self.assertEqual(result["gender"], "male")
        self.assertEqual(result["birth_date"], "1970-01-01")

        # Check name transformation
        self.assertEqual(len(result["name"]), 1)
        self.assertEqual(result["name"][0]["family_name"], "Smith")
        self.assertEqual(result["name"][0]["given_name"], "John Adam")
        self.assertEqual(result["name"][0]["use"], "official")

        # Check telecom transformation
        self.assertEqual(len(result["telecom"]), 2)
        self.assertEqual(result["telecom"][0]["system"], "phone")
        self.assertEqual(result["telecom"][0]["value"], "555-123-4567")
        self.assertEqual(result["telecom"][0]["use"], "home")
        self.assertEqual(result["telecom"][0]["display_value"], "Phone: 555-123-4567")

        # Check address transformation
        self.assertEqual(len(result["address"]), 1)
        self.assertEqual(result["address"][0]["street_address"], "123 Main St")
        self.assertEqual(result["address"][0]["city"], "Anytown")
        self.assertEqual(result["address"][0]["state"], "CA")
        self.assertEqual(result["address"][0]["postal_code"], "12345")
        self.assertEqual(result["address"][0]["country"], "USA")
        self.assertEqual(result["address"][0]["use"], "home")
        self.assertEqual(result["address"][0]["formatted"], "123 Main St\nAnytown, CA, 12345\nUSA")

    def test_transform_empty_patient(self):
        """Test transforming an empty Patient."""
        # Create an empty Patient
        empty_patient = Patient(
            id="empty-patient",
            resource_type="Patient"
        )

        # Transform the Patient
        result = self.transformer.transform(empty_patient)

        # Check that the transformation was successful
        self.assertIsInstance(result, dict)
        self.assertEqual(result["id"], "empty-patient")
        self.assertEqual(result["resource_type"], "Patient")

        # Check that optional fields are either None or not present
        if "name" in result:
            self.assertIsNone(result["name"])
        if "telecom" in result:
            self.assertIsNone(result["telecom"])
        if "address" in result:
            self.assertIsNone(result["address"])

    def test_transform_invalid_patient(self):
        """Test transforming an invalid Patient."""
        # Create an invalid object (not a Patient)
        invalid_patient = {"id": "invalid-patient"}

        # Check that transforming raises a ValidationError
        with self.assertRaises(ValidationError):
            self.transformer.transform(invalid_patient)

if __name__ == '__main__':
    unittest.main()
