"""
Medication Order Models for Order Management Service

This module provides FHIR-compliant models for medication orders,
implementing the FHIR MedicationRequest resource.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
from datetime import datetime
from enum import Enum
import os
import sys

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import shared FHIR models
try:
    from shared.models import (
        FHIRBaseModel, CodeableConcept, Reference, Identifier, 
        Period, Annotation, Quantity
    )
except ImportError:
    # Fallback if shared models are not available
    from pydantic import BaseModel as FHIRBaseModel
    
    class CodeableConcept(BaseModel):
        coding: Optional[List[Dict[str, Any]]] = None
        text: Optional[str] = None
    
    class Reference(BaseModel):
        reference: Optional[str] = None
        display: Optional[str] = None
    
    class Identifier(BaseModel):
        use: Optional[str] = None
        system: Optional[str] = None
        value: Optional[str] = None
    
    class Period(BaseModel):
        start: Optional[datetime] = None
        end: Optional[datetime] = None
    
    class Annotation(BaseModel):
        text: str
        author_string: Optional[str] = None
        time: Optional[datetime] = None
    
    class Quantity(BaseModel):
        value: Optional[float] = None
        unit: Optional[str] = None
        system: Optional[str] = None
        code: Optional[str] = None

# Medication Request Status
class MedicationRequestStatus(str, Enum):
    """FHIR MedicationRequest status values"""
    ACTIVE = "active"
    ON_HOLD = "on-hold"
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"
    STOPPED = "stopped"
    DRAFT = "draft"
    UNKNOWN = "unknown"

class MedicationRequestIntent(str, Enum):
    """FHIR MedicationRequest intent values"""
    PROPOSAL = "proposal"
    PLAN = "plan"
    ORDER = "order"
    ORIGINAL_ORDER = "original-order"
    REFLEX_ORDER = "reflex-order"
    FILLER_ORDER = "filler-order"
    INSTANCE_ORDER = "instance-order"
    OPTION = "option"

# Dosage Instruction Model
class DosageInstruction(FHIRBaseModel):
    """
    FHIR Dosage datatype for medication dosage instructions.
    """
    sequence: Optional[int] = Field(None, description="The order of the dosage instructions")
    text: Optional[str] = Field(None, description="Free text dosage instructions")
    additional_instruction: Optional[List[CodeableConcept]] = Field(None, alias="additionalInstruction", description="Supplemental instruction or warnings")
    patient_instruction: Optional[str] = Field(None, alias="patientInstruction", description="Patient or consumer oriented instructions")
    timing: Optional[Dict[str, Any]] = Field(None, description="When medication should be administered")
    as_needed_boolean: Optional[bool] = Field(None, alias="asNeededBoolean", description="Take 'as needed'")
    as_needed_codeable_concept: Optional[CodeableConcept] = Field(None, alias="asNeededCodeableConcept", description="Take 'as needed' for x")
    site: Optional[CodeableConcept] = Field(None, description="Body site to administer to")
    route: Optional[CodeableConcept] = Field(None, description="How drug should enter body")
    method: Optional[CodeableConcept] = Field(None, description="Technique for administering medication")
    dose_and_rate: Optional[List[Dict[str, Any]]] = Field(None, alias="doseAndRate", description="Amount of medication administered")
    max_dose_per_period: Optional[Quantity] = Field(None, alias="maxDosePerPeriod", description="Upper limit on medication per unit of time")
    max_dose_per_administration: Optional[Quantity] = Field(None, alias="maxDosePerAdministration", description="Upper limit on medication per administration")
    max_dose_per_lifetime: Optional[Quantity] = Field(None, alias="maxDosePerLifetime", description="Upper limit on medication per lifetime of the patient")
    
    class Config:
        extra = "allow"
        populate_by_name = True

# Dispense Request Model
class DispenseRequest(FHIRBaseModel):
    """
    FHIR MedicationRequest.dispenseRequest component.
    """
    initial_fill: Optional[Dict[str, Any]] = Field(None, alias="initialFill", description="First fill details")
    dispense_interval: Optional[Dict[str, Any]] = Field(None, alias="dispenseInterval", description="Minimum period of time between dispenses")
    validity_period: Optional[Period] = Field(None, alias="validityPeriod", description="Time period supply is authorized for")
    number_of_repeats_allowed: Optional[int] = Field(None, alias="numberOfRepeatsAllowed", description="Number of refills authorized")
    quantity: Optional[Quantity] = Field(None, description="Amount of medication to supply per dispense")
    expected_supply_duration: Optional[Dict[str, Any]] = Field(None, alias="expectedSupplyDuration", description="Number of days supply per dispense")
    performer: Optional[Reference] = Field(None, description="Intended dispenser")
    
    class Config:
        extra = "allow"
        populate_by_name = True

# Medication Request Model
class MedicationOrder(FHIRBaseModel):
    """
    FHIR MedicationRequest resource for medication orders.
    
    This model represents a request for a medication to be dispensed and administered.
    """
    
    # Required FHIR fields
    resourceType: str = Field(default="MedicationRequest", description="FHIR resource type")
    id: Optional[str] = Field(None, description="Logical id of this artifact")
    
    # Core MedicationRequest fields
    status: MedicationRequestStatus = Field(..., description="Current status of the medication request")
    intent: MedicationRequestIntent = Field(..., description="Intent of the medication request")
    category: Optional[List[CodeableConcept]] = Field(None, description="Type of medication usage")
    priority: Optional[str] = Field(None, description="Urgency of the request")
    
    # Medication
    medication_codeable_concept: Optional[CodeableConcept] = Field(None, alias="medicationCodeableConcept", description="Medication to be taken")
    medication_reference: Optional[Reference] = Field(None, alias="medicationReference", description="Medication to be taken")
    
    # Patient and context
    subject: Reference = Field(..., description="Who or group medication request is for")
    encounter: Optional[Reference] = Field(None, description="Encounter created during")
    
    # Timing
    authored_on: Optional[datetime] = Field(None, alias="authoredOn", description="When request was initially authored")
    
    # Participants
    requester: Optional[Reference] = Field(None, description="Who/What requested the medication")
    performer: Optional[Reference] = Field(None, description="Intended performer of administration")
    performer_type: Optional[CodeableConcept] = Field(None, alias="performerType", description="Desired kind of performer of the medication administration")
    recorder: Optional[Reference] = Field(None, description="Person who entered the request")
    
    # Clinical context
    reason_code: Optional[List[CodeableConcept]] = Field(None, alias="reasonCode", description="Reason or indication for ordering or not ordering the medication")
    reason_reference: Optional[List[Reference]] = Field(None, alias="reasonReference", description="Condition or observation that supports why the prescription is being written")
    supporting_information: Optional[List[Reference]] = Field(None, alias="supportingInformation", description="Information to support ordering of the medication")
    
    # Instructions
    note: Optional[List[Annotation]] = Field(None, description="Information about the prescription")
    dosage_instruction: Optional[List[DosageInstruction]] = Field(None, alias="dosageInstruction", description="How the medication should be taken")
    
    # Dispensing
    dispense_request: Optional[DispenseRequest] = Field(None, alias="dispenseRequest", description="Medication supply authorization")
    
    # Substitution
    substitution: Optional[Dict[str, Any]] = Field(None, description="Any restrictions on medication substitution")
    
    # Prior prescription
    prior_prescription: Optional[Reference] = Field(None, alias="priorPrescription", description="An order/prescription that is being replaced")
    
    # Detection of issue
    detected_issue: Optional[List[Reference]] = Field(None, alias="detectedIssue", description="Clinical Issue with action")
    
    # Event history
    event_history: Optional[List[Reference]] = Field(None, alias="eventHistory", description="A list of events of interest in the lifecycle")
    
    # Metadata
    meta: Optional[Dict[str, Any]] = Field(None, description="Metadata about the resource")
    
    class Config:
        extra = "allow"
        populate_by_name = True
        
    def to_fhir_dict(self) -> Dict[str, Any]:
        """Convert to FHIR-compliant dictionary"""
        data = self.model_dump(by_alias=True, exclude_unset=True)
        return data
    
    @classmethod
    def from_fhir_dict(cls, fhir_dict: Dict[str, Any]) -> "MedicationOrder":
        """Create instance from FHIR dictionary"""
        return cls.model_validate(fhir_dict)

# Create and Update models for API endpoints
class MedicationOrderCreate(BaseModel):
    """Model for creating a medication order"""
    status: MedicationRequestStatus = MedicationRequestStatus.DRAFT
    intent: MedicationRequestIntent = MedicationRequestIntent.ORDER
    category: Optional[List[CodeableConcept]] = None
    priority: Optional[str] = "routine"
    medication_codeable_concept: Optional[CodeableConcept] = None
    medication_reference: Optional[Reference] = None
    subject: Reference
    encounter: Optional[Reference] = None
    requester: Optional[Reference] = None
    performer: Optional[Reference] = None
    performer_type: Optional[CodeableConcept] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    dosage_instruction: Optional[List[DosageInstruction]] = None
    dispense_request: Optional[DispenseRequest] = None
    
    def to_medication_order(self) -> MedicationOrder:
        """Convert to a FHIR MedicationOrder."""
        data = self.model_dump(exclude_unset=True)
        data["authored_on"] = datetime.utcnow()
        return MedicationOrder(**data)

class MedicationOrderUpdate(BaseModel):
    """Model for updating a medication order"""
    status: Optional[MedicationRequestStatus] = None
    priority: Optional[str] = None
    performer: Optional[Reference] = None
    note: Optional[List[Annotation]] = None
    dosage_instruction: Optional[List[DosageInstruction]] = None
