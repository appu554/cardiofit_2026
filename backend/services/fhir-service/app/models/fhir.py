from datetime import datetime
from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field

# Import shared FHIR models
from shared.models import (
    FHIRBaseModel,
    Coding, CodeableConcept, Reference, Identifier, HumanName,
    ContactPoint, Address, Quantity, Period,
    Patient, Observation, Condition, Encounter,
    Medication, MedicationRequest, MedicationAdministration, MedicationStatement,
    DiagnosticReport
)

# Base FHIR Resource
class FHIRResource(BaseModel):
    resourceType: str
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None

    class Config:
        extra = "allow"  # Allow extra fields for FHIR resources

# FHIR Resources - Using shared models for most resources
# Only keeping DocumentReference as it's not in shared models yet

class DocumentReference(FHIRResource):
    resourceType: str = "DocumentReference"
    status: str  # current | superseded | entered-in-error
    type: Optional[CodeableConcept] = None
    category: Optional[List[CodeableConcept]] = None
    subject: Reference
    date: Optional[str] = None
    author: Optional[List[Reference]] = None
    content: List[Dict[str, Any]]
