"""
Encounter resource model for Clinical Synthesis Hub.

This module provides a Pydantic model for the FHIR Encounter resource
used across all microservices in the Clinical Synthesis Hub.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import Field, validator
from fhir.resources import get_fhir_model_class

from ..base import FHIRBaseModel
from ..datatypes import (
    CodeableConcept, Reference, Period, Identifier
)

class EncounterParticipant(FHIRBaseModel):
    """
    List of participants involved in the encounter.
    """
    type: Optional[List[CodeableConcept]] = None
    period: Optional[Period] = None
    individual: Optional[Reference] = None

class EncounterDiagnosis(FHIRBaseModel):
    """
    The list of diagnosis relevant to this encounter.
    """
    condition: Reference
    use: Optional[CodeableConcept] = None
    rank: Optional[int] = None

class EncounterLocation(FHIRBaseModel):
    """
    List of locations where the patient has been.
    """
    location: Reference
    status: Optional[str] = None  # planned | active | reserved | completed
    physicalType: Optional[CodeableConcept] = None
    period: Optional[Period] = None

class EncounterStatusHistory(FHIRBaseModel):
    """
    List of past encounter statuses.
    """
    status: str
    period: Period

class EncounterClassHistory(FHIRBaseModel):
    """
    List of past encounter classes.
    """
    class_: Dict[str, str] = Field(..., alias="class")
    period: Period

class Encounter(FHIRBaseModel):
    """
    An interaction between a patient and healthcare provider(s) for the purpose of 
    providing healthcare service(s) or assessing the health status of a patient.
    """
    resourceType: str = "Encounter"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers
    identifier: Optional[List[Identifier]] = None
    
    # Status
    status: str  # planned | arrived | triaged | in-progress | onleave | finished | cancelled
    statusHistory: Optional[List[EncounterStatusHistory]] = None
    
    # Class
    class_: Dict[str, str] = Field(..., alias="class")
    classHistory: Optional[List[EncounterClassHistory]] = None
    
    # Type and priority
    type: Optional[List[CodeableConcept]] = None
    serviceType: Optional[CodeableConcept] = None
    priority: Optional[CodeableConcept] = None
    
    # Subject and participants
    subject: Reference
    episodeOfCare: Optional[List[Reference]] = None
    basedOn: Optional[List[Reference]] = None
    participant: Optional[List[EncounterParticipant]] = None
    
    # Timing
    period: Optional[Period] = None
    length: Optional[Dict[str, Any]] = None  # Duration
    
    # Reasons
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    
    # Diagnoses
    diagnosis: Optional[List[EncounterDiagnosis]] = None
    
    # Accounts and hospitalization
    account: Optional[List[Reference]] = None
    hospitalization: Optional[Dict[str, Any]] = None
    
    # Locations
    location: Optional[List[EncounterLocation]] = None
    
    # Service provider
    serviceProvider: Optional[Reference] = None
    
    # Part of
    partOf: Optional[Reference] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        allowed_values = [
            'planned', 'arrived', 'triaged', 'in-progress', 
            'onleave', 'finished', 'cancelled', 'entered-in-error', 'unknown'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid status: {v}. Must be one of: {', '.join(allowed_values)}")
        return v
    
    @classmethod
    def from_fhir_encounter(cls, fhir_encounter):
        """
        Create an Encounter instance from a FHIR Encounter resource.
        
        Args:
            fhir_encounter: A FHIR Encounter resource from fhir.resources
            
        Returns:
            An Encounter instance
        """
        if isinstance(fhir_encounter, dict):
            return cls.parse_obj(fhir_encounter)
        
        # If it's a FHIR model, convert to dict first
        return cls.parse_obj(fhir_encounter.dict(exclude_unset=True))
    
    def to_fhir_encounter(self):
        """
        Convert this Encounter to a FHIR Encounter resource.
        
        Returns:
            A FHIR Encounter resource from fhir.resources
        """
        EncounterResource = get_fhir_model_class("Encounter")
        return EncounterResource.parse_obj(self.dict(exclude_none=True, by_alias=True))
