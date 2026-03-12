"""
Observation resource model for Clinical Synthesis Hub.

This module provides a Pydantic model for the FHIR Observation resource
used across all microservices in the Clinical Synthesis Hub.
"""

import logging
from typing import Dict, List, Optional, Any, Union, Type, TypeVar
from pydantic import Field, validator, BaseModel

from ..base import FHIRBaseModel, FHIR_AVAILABLE
from ..datatypes import (
    CodeableConcept, Reference, Quantity, Period, Range, Ratio, Annotation
)

# Try to import FHIR resources, but make it optional
try:
    from fhir.resources import get_fhir_model_class
    FHIR_IMPORTED = True
except ImportError:
    logging.warning(
        "fhir.resources module not found in observation.py. "
        "FHIR-specific functionality will be limited. "
        "Install with 'pip install fhir.resources' for full FHIR support."
    )
    FHIR_IMPORTED = False
    
    # Dummy function for when FHIR is not available
    def get_fhir_model_class(resource_type: str) -> Type[BaseModel]:
        raise ImportError(
            "fhir.resources is required for get_fhir_model_class. "
            "Install with 'pip install fhir.resources'"
        )

class ObservationComponent(FHIRBaseModel):
    """
    Component observation values.
    """
    code: CodeableConcept
    valueQuantity: Optional[Quantity] = None
    valueCodeableConcept: Optional[CodeableConcept] = None
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
    referenceRange: Optional[List[Dict[str, Any]]] = None

class ObservationReferenceRange(FHIRBaseModel):
    """
    Provides guide for interpretation of a measurement or observation value.
    """
    low: Optional[Quantity] = None
    high: Optional[Quantity] = None
    type: Optional[CodeableConcept] = None
    appliesTo: Optional[List[CodeableConcept]] = None
    age: Optional[Range] = None
    text: Optional[str] = None

class Observation(FHIRBaseModel):
    """
    Measurements and simple assertions made about a patient, device or other subject.
    """
    resourceType: str = "Observation"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Status and category
    status: str  # registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown
    category: Optional[List[CodeableConcept]] = None
    code: CodeableConcept
    
    # Subject and context
    subject: Reference
    encounter: Optional[Reference] = None
    
    # Effective time
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Period] = None
    effectiveInstant: Optional[str] = None
    effectiveTiming: Optional[Dict[str, Any]] = None
    
    # Issued timestamp
    issued: Optional[str] = None
    
    # Performers
    performer: Optional[List[Reference]] = None
    
    # Value
    valueQuantity: Optional[Quantity] = None
    valueCodeableConcept: Optional[CodeableConcept] = None
    valueString: Optional[str] = None
    valueBoolean: Optional[bool] = None
    valueInteger: Optional[int] = None
    valueRange: Optional[Range] = None
    valueRatio: Optional[Ratio] = None
    valueTime: Optional[str] = None
    valueDateTime: Optional[str] = None
    valuePeriod: Optional[Period] = None
    
    # Data absent reason
    dataAbsentReason: Optional[CodeableConcept] = None
    
    # Interpretation and notes
    interpretation: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None
    
    # Physiologically relevant time/time-period
    bodySite: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None
    
    # Reference ranges
    referenceRange: Optional[List[ObservationReferenceRange]] = None
    
    # Components
    component: Optional[List[ObservationComponent]] = None
    
    # Derived from and has member
    derivedFrom: Optional[List[Reference]] = None
    hasMember: Optional[List[Reference]] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        allowed_values = [
            'registered', 'preliminary', 'final', 'amended', 
            'corrected', 'cancelled', 'entered-in-error', 'unknown'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid status: {v}. Must be one of: {', '.join(allowed_values)}")
        return v
    
    @classmethod
    def get_fhir_model(cls) -> type:
        """
        Get the FHIR model class for this resource.
        
        Returns:
            The FHIR model class for Observation
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE or not FHIR_IMPORTED:
            raise ImportError(
                "fhir.resources is required for get_fhir_model. "
                "Install with 'pip install fhir.resources'"
            )
        return get_fhir_model_class("Observation")

    @classmethod
    def from_fhir_observation(cls, fhir_observation):
        """
        Create an Observation instance from a FHIR Observation resource.
        
        Args:
            fhir_observation: A FHIR Observation resource from fhir.resources
            
        Returns:
            An Observation instance
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE or not FHIR_IMPORTED:
            raise ImportError(
                "fhir.resources is required for from_fhir_observation. "
                "Install with 'pip install fhir.resources'"
            )
            
        if isinstance(fhir_observation, dict):
            return cls.parse_obj(fhir_observation)
        
        # If it's a FHIR model, convert to dict first
        return cls.parse_obj(fhir_observation.dict(exclude_unset=True))
    
    def to_fhir_observation(self):
        """
        Convert this Observation to a FHIR Observation resource.
        
        Returns:
            A FHIR Observation resource from fhir.resources
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE or not FHIR_IMPORTED:
            raise ImportError(
                "fhir.resources is required for to_fhir_observation. "
                "Install with 'pip install fhir.resources'"
            )
            
        ObservationResource = get_fhir_model_class("Observation")
        fhir_observation = ObservationResource(**self.dict(exclude_none=True))
        return fhir_observation
