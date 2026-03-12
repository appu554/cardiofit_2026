"""
FHIR resources for Clinical Synthesis Hub.

This module provides Pydantic models for FHIR resources used across
all microservices in the Clinical Synthesis Hub.
"""

import logging
from typing import Any, Type, TypeVar

# Try to import FHIR resources, but make it optional
try:
    from fhir.resources import get_fhir_model_class
    FHIR_IMPORTED = True
except ImportError:
    logging.warning(
        "fhir.resources module not found in resources/__init__.py. "
        "FHIR-specific functionality will be limited. "
        "Install with 'pip install fhir.resources' for full FHIR support."
    )
    FHIR_IMPORTED = False
    
    # Dummy function for when FHIR is not available
    def get_fhir_model_class(resource_type: str) -> Type[Any]:
        raise ImportError(
            "fhir.resources is required for get_fhir_model_class. "
            "Install with 'pip install fhir.resources'"
        )

# Import all resource models
try:
    from .patient import Patient
    from .observation import Observation
    from .condition import Condition
    from .encounter import Encounter, EncounterParticipant, EncounterDiagnosis, EncounterLocation, EncounterStatusHistory, EncounterClassHistory
    from .medication import Medication, MedicationRequest, MedicationAdministration, MedicationStatement
    from .diagnostic import DiagnosticReport
    
    __all__ = [
        "Patient",
        "Observation",
        "Condition",
        "Encounter",
        "EncounterParticipant",
        "EncounterDiagnosis",
        "EncounterLocation",
        "EncounterStatusHistory",
        "EncounterClassHistory",
        "Medication",
        "MedicationRequest",
        "MedicationAdministration",
        "MedicationStatement",
        "DiagnosticReport"
    ]
    
except ImportError as e:
    logging.warning(f"Error importing resource models: {e}")
    
    # Create dummy classes if imports fail
    class DummyResource:
        pass
    
    # Create dummy instances of all resources
    Patient = Observation = Condition = Encounter = DummyResource
    EncounterParticipant = EncounterDiagnosis = EncounterLocation = DummyResource
    EncounterStatusHistory = EncounterClassHistory = Medication = DummyResource
    MedicationRequest = MedicationAdministration = MedicationStatement = DummyResource
    DiagnosticReport = DummyResource
    
    __all__ = [
        "Patient",
        "Observation",
        "Condition",
        "Encounter",
        "EncounterParticipant",
        "EncounterDiagnosis",
        "EncounterLocation",
        "EncounterStatusHistory",
        "EncounterClassHistory",
        "Medication",
        "MedicationRequest",
        "MedicationAdministration",
        "MedicationStatement",
        "DiagnosticReport"
    ]
