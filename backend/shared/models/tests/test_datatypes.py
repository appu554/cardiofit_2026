"""
Unit tests for FHIR datatypes.

This module contains tests for the FHIR datatypes like CodeableConcept, Reference, etc.
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
from shared.models.datatypes import (
    CodeableConcept, Coding, Reference, Identifier, HumanName,
    ContactPoint, Address, Quantity, Period
)

class CodingTests(unittest.TestCase):
    """Tests for the Coding datatype."""

    def test_init(self):
        """Test Coding initialization."""
        coding = Coding(
            system="http://snomed.info/sct",
            code="123456",
            display="Test Coding"
        )
        self.assertEqual(coding.system, "http://snomed.info/sct")
        self.assertEqual(coding.code, "123456")
        self.assertEqual(coding.display, "Test Coding")

    def test_to_dict(self):
        """Test conversion to dictionary."""
        coding = Coding(
            system="http://snomed.info/sct",
            code="123456",
            display="Test Coding"
        )
        coding_dict = coding.dict(exclude_none=True)
        self.assertEqual(coding_dict["system"], "http://snomed.info/sct")
        self.assertEqual(coding_dict["code"], "123456")
        self.assertEqual(coding_dict["display"], "Test Coding")

class CodeableConceptTests(unittest.TestCase):
    """Tests for the CodeableConcept datatype."""

    def test_init(self):
        """Test CodeableConcept initialization."""
        concept = CodeableConcept(
            coding=[
                Coding(
                    system="http://snomed.info/sct",
                    code="123456",
                    display="Test Coding"
                )
            ],
            text="Test Concept"
        )
        self.assertEqual(len(concept.coding), 1)
        self.assertEqual(concept.coding[0].system, "http://snomed.info/sct")
        self.assertEqual(concept.coding[0].code, "123456")
        self.assertEqual(concept.text, "Test Concept")

    def test_to_dict(self):
        """Test conversion to dictionary."""
        concept = CodeableConcept(
            coding=[
                Coding(
                    system="http://snomed.info/sct",
                    code="123456",
                    display="Test Coding"
                )
            ],
            text="Test Concept"
        )
        concept_dict = concept.dict(exclude_none=True)
        self.assertEqual(len(concept_dict["coding"]), 1)
        self.assertEqual(concept_dict["coding"][0]["system"], "http://snomed.info/sct")
        self.assertEqual(concept_dict["coding"][0]["code"], "123456")
        self.assertEqual(concept_dict["text"], "Test Concept")

class ReferenceTests(unittest.TestCase):
    """Tests for the Reference datatype."""

    def test_init(self):
        """Test Reference initialization."""
        reference = Reference(
            reference="Patient/123",
            display="John Smith"
        )
        self.assertEqual(reference.reference, "Patient/123")
        self.assertEqual(reference.display, "John Smith")

    def test_to_dict(self):
        """Test conversion to dictionary."""
        reference = Reference(
            reference="Patient/123",
            display="John Smith"
        )
        reference_dict = reference.dict(exclude_none=True)
        self.assertEqual(reference_dict["reference"], "Patient/123")
        self.assertEqual(reference_dict["display"], "John Smith")

class HumanNameTests(unittest.TestCase):
    """Tests for the HumanName datatype."""

    def test_init(self):
        """Test HumanName initialization."""
        name = HumanName(
            family="Smith",
            given=["John", "Adam"],
            prefix=["Mr."],
            suffix=["Jr."],
            use="official"
        )
        self.assertEqual(name.family, "Smith")
        self.assertEqual(name.given, ["John", "Adam"])
        self.assertEqual(name.prefix, ["Mr."])
        self.assertEqual(name.suffix, ["Jr."])
        self.assertEqual(name.use, "official")

    def test_to_dict(self):
        """Test conversion to dictionary."""
        name = HumanName(
            family="Smith",
            given=["John", "Adam"],
            prefix=["Mr."],
            suffix=["Jr."],
            use="official"
        )
        name_dict = name.dict(exclude_none=True)
        self.assertEqual(name_dict["family"], "Smith")
        self.assertEqual(name_dict["given"], ["John", "Adam"])
        self.assertEqual(name_dict["prefix"], ["Mr."])
        self.assertEqual(name_dict["suffix"], ["Jr."])
        self.assertEqual(name_dict["use"], "official")

class AddressTests(unittest.TestCase):
    """Tests for the Address datatype."""

    def test_init(self):
        """Test Address initialization."""
        address = Address(
            line=["123 Main St"],
            city="Anytown",
            state="CA",
            postalCode="12345",
            country="USA",
            use="home"
        )
        self.assertEqual(address.line, ["123 Main St"])
        self.assertEqual(address.city, "Anytown")
        self.assertEqual(address.state, "CA")
        self.assertEqual(address.postalCode, "12345")
        self.assertEqual(address.country, "USA")
        self.assertEqual(address.use, "home")

    def test_to_dict(self):
        """Test conversion to dictionary."""
        address = Address(
            line=["123 Main St"],
            city="Anytown",
            state="CA",
            postalCode="12345",
            country="USA",
            use="home"
        )
        address_dict = address.dict(exclude_none=True)
        self.assertEqual(address_dict["line"], ["123 Main St"])
        self.assertEqual(address_dict["city"], "Anytown")
        self.assertEqual(address_dict["state"], "CA")
        self.assertEqual(address_dict["postalCode"], "12345")
        self.assertEqual(address_dict["country"], "USA")
        self.assertEqual(address_dict["use"], "home")

class QuantityTests(unittest.TestCase):
    """Tests for the Quantity datatype."""

    def test_init(self):
        """Test Quantity initialization."""
        quantity = Quantity(
            value=80,
            unit="beats/minute",
            system="http://unitsofmeasure.org",
            code="/min"
        )
        self.assertEqual(quantity.value, 80)
        self.assertEqual(quantity.unit, "beats/minute")
        self.assertEqual(quantity.system, "http://unitsofmeasure.org")
        self.assertEqual(quantity.code, "/min")

    def test_to_dict(self):
        """Test conversion to dictionary."""
        quantity = Quantity(
            value=80,
            unit="beats/minute",
            system="http://unitsofmeasure.org",
            code="/min"
        )
        quantity_dict = quantity.dict(exclude_none=True)
        self.assertEqual(quantity_dict["value"], 80)
        self.assertEqual(quantity_dict["unit"], "beats/minute")
        self.assertEqual(quantity_dict["system"], "http://unitsofmeasure.org")
        self.assertEqual(quantity_dict["code"], "/min")

if __name__ == "__main__":
    unittest.main()
