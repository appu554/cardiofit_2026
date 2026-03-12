"""
Diagnostic resource models for Clinical Synthesis Hub.

This module provides Pydantic models for the FHIR DiagnosticReport resource
used across all microservices in the Clinical Synthesis Hub.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import Field, validator
from fhir.resources import get_fhir_model_class

from ..base import FHIRBaseModel
from ..datatypes import (
    CodeableConcept, Reference, Identifier, Attachment
)

class DiagnosticReportMedia(FHIRBaseModel):
    """
    Key images associated with this report.
    """
    comment: Optional[str] = None
    link: Reference

class DiagnosticReport(FHIRBaseModel):
    """
    The findings and interpretation of diagnostic tests performed on patients, 
    groups of patients, devices, and locations, and/or specimens derived from these.
    """
    resourceType: str = "DiagnosticReport"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers
    identifier: Optional[List[Identifier]] = None
    
    # Status
    status: str  # registered | partial | preliminary | final | amended | corrected | appended | cancelled | entered-in-error | unknown
    
    # Category and code
    category: Optional[List[CodeableConcept]] = None
    code: CodeableConcept
    
    # Subject and context
    subject: Reference
    encounter: Optional[Reference] = None
    
    # Effective time
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    
    # Issued timestamp
    issued: Optional[str] = None
    
    # Performers
    performer: Optional[List[Reference]] = None
    
    # Results
    resultsInterpreter: Optional[List[Reference]] = None
    specimen: Optional[List[Reference]] = None
    result: Optional[List[Reference]] = None
    imagingStudy: Optional[List[Reference]] = None
    media: Optional[List[DiagnosticReportMedia]] = None
    
    # Conclusion
    conclusion: Optional[str] = None
    conclusionCode: Optional[List[CodeableConcept]] = None
    
    # Presentation
    presentedForm: Optional[List[Attachment]] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        allowed_values = [
            'registered', 'partial', 'preliminary', 'final', 'amended', 
            'corrected', 'appended', 'cancelled', 'entered-in-error', 'unknown'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid status: {v}. Must be one of: {', '.join(allowed_values)}")
        return v
    
    @classmethod
    def from_fhir_diagnostic_report(cls, fhir_diagnostic_report):
        """
        Create a DiagnosticReport instance from a FHIR DiagnosticReport resource.
        
        Args:
            fhir_diagnostic_report: A FHIR DiagnosticReport resource from fhir.resources
            
        Returns:
            A DiagnosticReport instance
        """
        if isinstance(fhir_diagnostic_report, dict):
            return cls.parse_obj(fhir_diagnostic_report)
        
        # If it's a FHIR model, convert to dict first
        return cls.parse_obj(fhir_diagnostic_report.dict(exclude_unset=True))
    
    def to_fhir_diagnostic_report(self):
        """
        Convert this DiagnosticReport to a FHIR DiagnosticReport resource.
        
        Returns:
            A FHIR DiagnosticReport resource from fhir.resources
        """
        DiagnosticReportResource = get_fhir_model_class("DiagnosticReport")
        return DiagnosticReportResource.parse_obj(self.dict(exclude_none=True))
