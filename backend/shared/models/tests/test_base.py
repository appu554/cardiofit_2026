"""
Unit tests for the base FHIR models.

This module contains tests for the FHIRBaseModel and related functionality.
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

# Import the models to test
from shared.models.base import FHIRBaseModel
from fhir.resources import FHIRAbstractModel
from fhir.resources import get_fhir_model_class

class TestModel(FHIRBaseModel):
    """Test model for unit tests."""
    id: Optional[str] = None
    resourceType: str = "TestResource"
    name: Optional[str] = None
    value: Optional[int] = None
    active: Optional[bool] = None
    tags: Optional[List[str]] = None

class FHIRBaseModelTests(unittest.TestCase):
    """Tests for the FHIRBaseModel class."""

    def setUp(self):
        """Set up test fixtures."""
        self.test_data = {
            "resourceType": "TestResource",
            "id": "test-1",
            "name": "Test Resource",
            "value": 42,
            "active": True,
            "tags": ["test", "example"]
        }
        self.model = TestModel(**self.test_data)

    def test_init(self):
        """Test model initialization."""
        self.assertEqual(self.model.id, "test-1")
        self.assertEqual(self.model.resourceType, "TestResource")
        self.assertEqual(self.model.name, "Test Resource")
        self.assertEqual(self.model.value, 42)
        self.assertTrue(self.model.active)
        self.assertEqual(self.model.tags, ["test", "example"])

    def test_to_fhir(self):
        """Test conversion to FHIR dictionary."""
        fhir_dict = self.model.to_fhir()
        self.assertIsInstance(fhir_dict, dict)
        self.assertEqual(fhir_dict["id"], "test-1")
        self.assertEqual(fhir_dict["resourceType"], "TestResource")
        self.assertEqual(fhir_dict["name"], "Test Resource")
        self.assertEqual(fhir_dict["value"], 42)
        self.assertTrue(fhir_dict["active"])
        self.assertEqual(fhir_dict["tags"], ["test", "example"])

    def test_from_fhir(self):
        """Test creation from FHIR dictionary."""
        model = TestModel.from_fhir(self.test_data)
        self.assertEqual(model.id, "test-1")
        self.assertEqual(model.resourceType, "TestResource")
        self.assertEqual(model.name, "Test Resource")
        self.assertEqual(model.value, 42)
        self.assertTrue(model.active)
        self.assertEqual(model.tags, ["test", "example"])

    def test_model_dump_exclude_none(self):
        """Test that model_dump() with exclude_none=True excludes None values."""
        model = TestModel(id="test-2", name="Test Only")
        model_dict = model.model_dump(exclude_none=True)
        self.assertIn("id", model_dict)
        self.assertIn("resourceType", model_dict)
        self.assertIn("name", model_dict)
        self.assertNotIn("value", model_dict)
        self.assertNotIn("active", model_dict)
        self.assertNotIn("tags", model_dict)

    def test_extra_fields(self):
        """Test that extra fields are allowed."""
        data = self.test_data.copy()
        data["extra_field"] = "extra value"
        model = TestModel(**data)
        self.assertEqual(getattr(model, "extra_field"), "extra value")

if __name__ == "__main__":
    unittest.main()
