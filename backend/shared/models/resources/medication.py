"""
Medication-related resource models for Clinical Synthesis Hub.

This module provides Pydantic models for the FHIR Medication, MedicationRequest,
MedicationAdministration, and MedicationStatement resources used across all 
microservices in the Clinical Synthesis Hub.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import Field, validator
from fhir.resources import get_fhir_model_class

from ..base import FHIRBaseModel
from ..datatypes import (
    CodeableConcept, Reference, Annotation, Identifier, Ratio, Quantity
)

class DosageInstruction(FHIRBaseModel):
    """
    How the medication should be taken.
    """
    sequence: Optional[int] = None
    text: Optional[str] = None
    additionalInstruction: Optional[List[CodeableConcept]] = None
    patientInstruction: Optional[str] = None
    timing: Optional[Dict[str, Any]] = None
    asNeededBoolean: Optional[bool] = None
    asNeededCodeableConcept: Optional[CodeableConcept] = None
    site: Optional[CodeableConcept] = None
    route: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None
    doseAndRate: Optional[List[Dict[str, Any]]] = None
    maxDosePerPeriod: Optional[Ratio] = None
    maxDosePerAdministration: Optional[Quantity] = None
    maxDosePerLifetime: Optional[Quantity] = None

class MedicationIngredient(FHIRBaseModel):
    """
    Active or inactive ingredient.
    """
    itemCodeableConcept: Optional[CodeableConcept] = None
    itemReference: Optional[Reference] = None
    isActive: Optional[bool] = None
    strength: Optional[Ratio] = None

class Medication(FHIRBaseModel):
    """
    Definition of a Medication.
    """
    resourceType: str = "Medication"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers and status
    identifier: Optional[List[Identifier]] = None
    code: Optional[CodeableConcept] = None
    status: Optional[str] = None  # active | inactive | entered-in-error
    
    # Manufacturer
    manufacturer: Optional[Reference] = None
    
    # Form and amount
    form: Optional[CodeableConcept] = None
    amount: Optional[Ratio] = None
    
    # Ingredients and batch
    ingredient: Optional[List[MedicationIngredient]] = None
    batch: Optional[Dict[str, Any]] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        if v is not None and v not in ['active', 'inactive', 'entered-in-error']:
            raise ValueError(f"Invalid status: {v}. Must be one of: active, inactive, entered-in-error")
        return v

class MedicationRequest(FHIRBaseModel):
    """
    An order or request for both supply of the medication and the instructions for 
    administration of the medication to a patient.
    """
    resourceType: str = "MedicationRequest"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers
    identifier: Optional[List[Identifier]] = None
    
    # Status and intent
    status: str  # active | on-hold | cancelled | completed | entered-in-error | stopped | draft | unknown
    intent: str  # proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option
    
    # Medication
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    
    # Subject and context
    subject: Reference
    encounter: Optional[Reference] = None
    
    # Supporting information
    supportingInformation: Optional[List[Reference]] = None
    
    # Authored date
    authoredOn: Optional[str] = None
    
    # Requester
    requester: Optional[Reference] = None
    
    # Performer
    performer: Optional[Reference] = None
    performerType: Optional[CodeableConcept] = None
    
    # Recorder
    recorder: Optional[Reference] = None
    
    # Reason
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    
    # Insurance
    insurance: Optional[List[Reference]] = None
    
    # Notes
    note: Optional[List[Annotation]] = None
    
    # Dosage
    dosageInstruction: Optional[List[DosageInstruction]] = None
    
    # Dispense
    dispenseRequest: Optional[Dict[str, Any]] = None
    
    # Substitution
    substitution: Optional[Dict[str, Any]] = None
    
    # Prior prescription
    priorPrescription: Optional[Reference] = None
    
    # Detected issues
    detectedIssue: Optional[List[Reference]] = None
    
    # Event history
    eventHistory: Optional[List[Reference]] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        allowed_values = [
            'active', 'on-hold', 'cancelled', 'completed', 
            'entered-in-error', 'stopped', 'draft', 'unknown'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid status: {v}. Must be one of: {', '.join(allowed_values)}")
        return v
    
    @validator('intent')
    def validate_intent(cls, v):
        """Validate that intent is one of the allowed values."""
        allowed_values = [
            'proposal', 'plan', 'order', 'original-order', 
            'reflex-order', 'filler-order', 'instance-order', 'option'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid intent: {v}. Must be one of: {', '.join(allowed_values)}")
        return v

class MedicationAdministration(FHIRBaseModel):
    """
    Administration of medication to a patient.
    """
    resourceType: str = "MedicationAdministration"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers
    identifier: Optional[List[Identifier]] = None
    
    # Status
    status: str  # in-progress | not-done | on-hold | completed | entered-in-error | stopped | unknown
    statusReason: Optional[List[CodeableConcept]] = None
    
    # Medication
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    
    # Subject and context
    subject: Reference
    context: Optional[Reference] = None
    
    # Supporting information
    supportingInformation: Optional[List[Reference]] = None
    
    # Effective time
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    
    # Performer
    performer: Optional[List[Dict[str, Any]]] = None
    
    # Reason
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    
    # Request
    request: Optional[Reference] = None
    
    # Device
    device: Optional[List[Reference]] = None
    
    # Notes
    note: Optional[List[Annotation]] = None
    
    # Dosage
    dosage: Optional[Dict[str, Any]] = None
    
    # Event history
    eventHistory: Optional[List[Reference]] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        allowed_values = [
            'in-progress', 'not-done', 'on-hold', 'completed', 
            'entered-in-error', 'stopped', 'unknown'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid status: {v}. Must be one of: {', '.join(allowed_values)}")
        return v

class MedicationStatement(FHIRBaseModel):
    """
    Record of medication being taken by a patient.
    """
    resourceType: str = "MedicationStatement"
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    
    # Identifiers
    identifier: Optional[List[Identifier]] = None
    
    # Status
    status: str  # active | completed | entered-in-error | intended | stopped | on-hold | unknown | not-taken
    statusReason: Optional[List[CodeableConcept]] = None
    
    # Medication
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    
    # Subject and context
    subject: Reference
    context: Optional[Reference] = None
    
    # Effective time
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    
    # Date asserted
    dateAsserted: Optional[str] = None
    
    # Information source
    informationSource: Optional[Reference] = None
    
    # Derived from
    derivedFrom: Optional[List[Reference]] = None
    
    # Reason
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    
    # Notes
    note: Optional[List[Annotation]] = None
    
    # Dosage
    dosage: Optional[List[DosageInstruction]] = None
    
    @validator('status')
    def validate_status(cls, v):
        """Validate that status is one of the allowed values."""
        allowed_values = [
            'active', 'completed', 'entered-in-error', 'intended', 
            'stopped', 'on-hold', 'unknown', 'not-taken'
        ]
        if v not in allowed_values:
            raise ValueError(f"Invalid status: {v}. Must be one of: {', '.join(allowed_values)}")
        return v
