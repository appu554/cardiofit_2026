"""
Condition resource model for Clinical Synthesis Hub.

This module provides a Pydantic model for the FHIR Condition resource
used across all microservices in the Clinical Synthesis Hub.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import Field, validator
from fhir.resources import get_fhir_model_class

from ..base import FHIRBaseModel
from ..datatypes import (
    CodeableConcept, Reference, Annotation, Identifier, Period
)

class ConditionStage(FHIRBaseModel):
    """
    Stage/grade, usually assessed formally.
    """
    summary: Optional[CodeableConcept] = None
    assessment: Optional[List[Reference]] = None
    type: Optional[CodeableConcept] = None

class ConditionEvidence(FHIRBaseModel):
    """
    Supporting evidence for the condition.
    """
    code: Optional[List[CodeableConcept]] = None
    detail: Optional[List[Reference]] = None

class Condition(FHIRBaseModel):
    """
    Detailed information about a condition, problem, diagnosis, or other event, 
    situation, issue, or clinical concept.
    """
    resourceType: str = "Condition"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers
    identifier: Optional[List[Identifier]] = None
    
    # Clinical status
    clinicalStatus: Optional[CodeableConcept] = None
    verificationStatus: Optional[CodeableConcept] = None
    
    # Category and code
    category: Optional[List[CodeableConcept]] = None
    severity: Optional[CodeableConcept] = None
    code: CodeableConcept
    
    # Body site
    bodySite: Optional[List[CodeableConcept]] = None
    
    # Subject and context
    subject: Reference
    encounter: Optional[Reference] = None
    
    # Onset and abatement
    onsetDateTime: Optional[str] = None
    onsetAge: Optional[Dict[str, Any]] = None
    onsetPeriod: Optional[Period] = None
    onsetRange: Optional[Dict[str, Any]] = None
    onsetString: Optional[str] = None
    
    abatementDateTime: Optional[str] = None
    abatementAge: Optional[Dict[str, Any]] = None
    abatementPeriod: Optional[Period] = None
    abatementRange: Optional[Dict[str, Any]] = None
    abatementString: Optional[str] = None
    
    # Recorded date
    recordedDate: Optional[str] = None
    recorder: Optional[Reference] = None
    asserter: Optional[Reference] = None
    
    # Clinical details
    stage: Optional[List[ConditionStage]] = None
    evidence: Optional[List[ConditionEvidence]] = None
    note: Optional[List[Annotation]] = None
    
    @classmethod
    def from_fhir_condition(cls, fhir_condition):
        """
        Create a Condition instance from a FHIR Condition resource.
        
        Args:
            fhir_condition: A FHIR Condition resource from fhir.resources
            
        Returns:
            A Condition instance
        """
        if isinstance(fhir_condition, dict):
            return cls.parse_obj(fhir_condition)
        
        # If it's a FHIR model, convert to dict first
        return cls.parse_obj(fhir_condition.dict(exclude_unset=True))
    
    def to_fhir_condition(self):
        """
        Convert this Condition to a FHIR Condition resource.
        
        Returns:
            A FHIR Condition resource from fhir.resources
        """
        ConditionResource = get_fhir_model_class("Condition")
        return ConditionResource.parse_obj(self.dict(exclude_none=True))
