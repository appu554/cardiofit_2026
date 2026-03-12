from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
from datetime import datetime
from enum import Enum

# Import shared models
from shared.models import Observation as SharedObservation
from shared.models import (
    CodeableConcept, Reference, Quantity, Period, Range, Ratio, Annotation
)

# Re-export the shared Observation model
Observation = SharedObservation

class ObservationCategory(str, Enum):
    """Observation categories based on FHIR standard"""
    LABORATORY = "laboratory"
    VITAL_SIGNS = "vital-signs"
    IMAGING = "imaging"
    SOCIAL_HISTORY = "social-history"
    EXAM = "exam"
    SURVEY = "survey"
    THERAPY = "therapy"
    ACTIVITY = "activity"

class ObservationStatus(str, Enum):
    """Observation status based on FHIR standard"""
    REGISTERED = "registered"
    PRELIMINARY = "preliminary"
    FINAL = "final"
    AMENDED = "amended"
    CORRECTED = "corrected"
    CANCELLED = "cancelled"
    ENTERED_IN_ERROR = "entered-in-error"
    UNKNOWN = "unknown"

# These models are now imported from shared.models

# Import ObservationReferenceRange from shared models
from shared.models.resources.observation import ObservationReferenceRange, ObservationComponent as SharedObservationComponent

# Re-export the shared ObservationComponent model
ObservationComponent = SharedObservationComponent

# Observation is now imported from shared.models

class ObservationCreate(BaseModel):
    """Model for creating an observation"""
    status: str = "final"  # registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown
    category: List[CodeableConcept]
    code: CodeableConcept
    subject: Reference
    encounter: Optional[Reference] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Period] = None
    issued: Optional[str] = None
    performer: Optional[List[Reference]] = None
    valueQuantity: Optional[Quantity] = None
    valueString: Optional[str] = None
    valueBoolean: Optional[bool] = None
    valueInteger: Optional[int] = None
    valueRange: Optional[Range] = None
    valueRatio: Optional[Ratio] = None
    valueTime: Optional[str] = None
    valueDateTime: Optional[str] = None
    valuePeriod: Optional[Period] = None
    dataAbsentReason: Optional[CodeableConcept] = None
    interpretation: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None
    bodySite: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None
    specimen: Optional[Reference] = None
    device: Optional[Reference] = None
    referenceRange: Optional[List[ObservationReferenceRange]] = None
    hasMember: Optional[List[Reference]] = None
    derivedFrom: Optional[List[Reference]] = None
    component: Optional[List[ObservationComponent]] = None

    def to_fhir_observation(self):
        """Convert to a FHIR Observation."""
        data = self.model_dump(exclude_unset=True)
        return Observation(**data)

class ObservationUpdate(BaseModel):
    """Model for updating an observation"""
    status: Optional[str] = None  # registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown
    category: Optional[List[CodeableConcept]] = None
    code: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    encounter: Optional[Reference] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Period] = None
    issued: Optional[str] = None
    performer: Optional[List[Reference]] = None
    valueQuantity: Optional[Quantity] = None
    valueString: Optional[str] = None
    valueBoolean: Optional[bool] = None
    valueInteger: Optional[int] = None
    valueRange: Optional[Range] = None
    valueRatio: Optional[Ratio] = None
    valueTime: Optional[str] = None
    valueDateTime: Optional[str] = None
    valuePeriod: Optional[Period] = None
    dataAbsentReason: Optional[CodeableConcept] = None
    interpretation: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None
    bodySite: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None
    specimen: Optional[Reference] = None
    device: Optional[Reference] = None
    referenceRange: Optional[List[ObservationReferenceRange]] = None
    hasMember: Optional[List[Reference]] = None
    derivedFrom: Optional[List[Reference]] = None
    component: Optional[List[ObservationComponent]] = None

    def to_fhir_observation_update(self):
        """Convert to a FHIR Observation update."""
        data = self.model_dump(exclude_unset=True)
        return data
