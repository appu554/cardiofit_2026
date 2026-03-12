"""
Patient resource model for Clinical Synthesis Hub.

This module provides a Pydantic model for the FHIR Patient resource
used across all microservices in the Clinical Synthesis Hub.
"""

import logging
from typing import Dict, List, Optional, Any, Union, Type, TypeVar
from pydantic import Field, validator, BaseModel

from ..base import FHIRBaseModel, FHIR_AVAILABLE
from ..datatypes import (
    Address, ContactPoint, HumanName, Identifier, Reference, CodeableConcept
)

# Try to import FHIR resources, but make it optional
try:
    from fhir.resources import get_fhir_model_class
    FHIR_IMPORTED = True
except ImportError:
    logging.warning(
        "fhir.resources module not found in patient.py. "
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

class PatientContact(FHIRBaseModel):
    """
    A contact party (e.g. guardian, partner, friend) for the patient.
    """
    relationship: Optional[List[CodeableConcept]] = None
    name: Optional[HumanName] = None
    telecom: Optional[List[ContactPoint]] = None
    address: Optional[Address] = None
    gender: Optional[str] = None
    organization: Optional[Reference] = None
    period: Optional[Dict[str, Any]] = None

class PatientCommunication(FHIRBaseModel):
    """
    A language which may be used to communicate with the patient about his or her health.
    """
    language: CodeableConcept
    preferred: Optional[bool] = None

class PatientLink(FHIRBaseModel):
    """
    Link to another patient resource that concerns the same actual person.
    """
    other: Reference
    type: str  # 'replaced-by' | 'replaces' | 'refer' | 'seealso'

class Patient(FHIRBaseModel):
    """
    Demographics and other administrative information about an individual
    receiving care or other health-related services.
    """
    resourceType: str = "Patient"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Patient demographics
    identifier: Optional[List[Identifier]] = None
    active: Optional[bool] = True
    name: Optional[List[HumanName]] = None
    telecom: Optional[List[ContactPoint]] = None
    gender: Optional[str] = None  # male | female | other | unknown
    birthDate: Optional[str] = None
    deceasedBoolean: Optional[bool] = None
    deceasedDateTime: Optional[str] = None
    address: Optional[List[Address]] = None
    
    # Patient relationships
    maritalStatus: Optional[CodeableConcept] = None
    multipleBirthBoolean: Optional[bool] = None
    multipleBirthInteger: Optional[int] = None
    contact: Optional[List[PatientContact]] = None
    communication: Optional[List[PatientCommunication]] = None
    generalPractitioner: Optional[List[Reference]] = None
    managingOrganization: Optional[Reference] = None
    link: Optional[List[PatientLink]] = None
    
    @validator('gender')
    def validate_gender(cls, v):
        """Validate that gender is one of the allowed values."""
        if v is not None and v not in ['male', 'female', 'other', 'unknown']:
            raise ValueError(f"Invalid gender: {v}. Must be one of: male, female, other, unknown")
        return v
    
    @classmethod
    def get_fhir_model(cls) -> type:
        """
        Get the FHIR model class for this resource.
        
        Returns:
            The FHIR model class for Patient
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE or not FHIR_IMPORTED:
            raise ImportError(
                "fhir.resources is required for get_fhir_model. "
                "Install with 'pip install fhir.resources'"
            )
        return get_fhir_model_class("Patient")

    @classmethod
    def from_fhir_patient(cls, fhir_patient):
        """
        Create a Patient instance from a FHIR Patient resource.
        
        Args:
            fhir_patient: A FHIR Patient resource from fhir.resources
            
        Returns:
            A Patient instance
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE or not FHIR_IMPORTED:
            raise ImportError(
                "fhir.resources is required for from_fhir_patient. "
                "Install with 'pip install fhir.resources'"
            )
            
        if isinstance(fhir_patient, dict):
            return cls.parse_obj(fhir_patient)
        
        # If it's a FHIR model, convert to dict first
        return cls.parse_obj(fhir_patient.dict(exclude_unset=True))
    
    def to_fhir_patient(self):
        """
        Convert this Patient to a FHIR Patient resource.
        
        Returns:
            A FHIR Patient resource from fhir.resources
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE or not FHIR_IMPORTED:
            raise ImportError(
                "fhir.resources is required for to_fhir_patient. "
                "Install with 'pip install fhir.resources'"
            )
            
        PatientResource = get_fhir_model_class("Patient")
        fhir_patient = PatientResource(**self.dict(exclude_none=True))
        return fhir_patient
