"""
Shared models package for Clinical Synthesis Hub.

This package provides shared Pydantic models for all microservices,
ensuring consistent data representation across the platform.
"""

import logging

# Base models
from .base import FHIRBaseModel, FHIR_AVAILABLE

# FHIR datatypes
try:
    from .datatypes import (
        Address, Annotation, Attachment, CodeableConcept, Coding, ContactPoint,
        HumanName, Identifier, Period, Quantity, Range, Ratio, Reference
    )
    DATA_TYPES_AVAILABLE = True
except ImportError as e:
    logging.warning(f"Could not import FHIR datatypes: {e}")
    # Create dummy classes for type hints
    class DummyType:
        pass
    
    Address = CodeableConcept = Coding = ContactPoint = HumanName = DummyType
    Identifier = Period = Quantity = Range = Ratio = Reference = DummyType
    Annotation = Attachment = DummyType
    DATA_TYPES_AVAILABLE = False

# FHIR resources
try:
    from .resources import (
        Patient, Observation, Condition, Encounter,
        EncounterParticipant, EncounterDiagnosis, EncounterLocation, 
        EncounterStatusHistory, EncounterClassHistory,
        Medication, MedicationRequest, MedicationAdministration, MedicationStatement,
        DiagnosticReport
    )
    RESOURCES_AVAILABLE = True
except ImportError as e:
    logging.warning(f"Could not import FHIR resources: {e}")
    # Create dummy classes for type hints
    class DummyResource:
        pass
    
    Patient = Observation = Condition = Encounter = DummyResource
    EncounterParticipant = EncounterDiagnosis = EncounterLocation = DummyResource
    EncounterStatusHistory = EncounterClassHistory = DummyResource
    Medication = MedicationRequest = MedicationAdministration = DummyResource
    MedicationStatement = DiagnosticReport = DummyResource
    RESOURCES_AVAILABLE = False

# Validators
try:
    from .validators import validate_fhir_resource
    VALIDATORS_AVAILABLE = True
except ImportError as e:
    logging.warning(f"Could not import FHIR validators: {e}")
    def validate_fhir_resource(resource: dict) -> tuple[bool, list[str]]:
        """Dummy validator when FHIR validation is not available."""
        return True, []
    VALIDATORS_AVAILABLE = False

__all__ = [
    # Base models
    "FHIRBaseModel", "FHIR_AVAILABLE",

    # FHIR datatypes
    "Address", "Annotation", "Attachment", "CodeableConcept", "Coding", "ContactPoint",
    "HumanName", "Identifier", "Period", "Quantity", "Range", "Ratio", "Reference",
    "DATA_TYPES_AVAILABLE",

    # FHIR resources
    "Patient", "Observation", "Condition", "Encounter",
    "EncounterParticipant", "EncounterDiagnosis", "EncounterLocation", "EncounterStatusHistory", "EncounterClassHistory",
    "Medication", "MedicationRequest", "MedicationAdministration", "MedicationStatement",
    "DiagnosticReport", "RESOURCES_AVAILABLE",

    # Validators
    "validate_fhir_resource", "VALIDATORS_AVAILABLE"
]
