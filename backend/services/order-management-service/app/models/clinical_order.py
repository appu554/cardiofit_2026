"""
Clinical Order Models for Order Management Service

This module provides FHIR-compliant models for clinical orders,
implementing the FHIR ServiceRequest resource and related order types.
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

# Order Status Enums
class OrderStatus(str, Enum):
    """FHIR ServiceRequest status values"""
    DRAFT = "draft"
    ACTIVE = "active"
    ON_HOLD = "on-hold"
    REVOKED = "revoked"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"
    UNKNOWN = "unknown"

class OrderIntent(str, Enum):
    """FHIR ServiceRequest intent values"""
    PROPOSAL = "proposal"
    PLAN = "plan"
    DIRECTIVE = "directive"
    ORDER = "order"
    ORIGINAL_ORDER = "original-order"
    REFLEX_ORDER = "reflex-order"
    FILLER_ORDER = "filler-order"
    INSTANCE_ORDER = "instance-order"
    OPTION = "option"

class OrderPriority(str, Enum):
    """FHIR ServiceRequest priority values"""
    ROUTINE = "routine"
    URGENT = "urgent"
    ASAP = "asap"
    STAT = "stat"

# Core Clinical Order Model (ServiceRequest)
class ClinicalOrder(FHIRBaseModel):
    """
    FHIR ServiceRequest resource for clinical orders.
    
    This model represents a request for a service to be performed.
    It is the core model for all types of clinical orders in CPOE.
    """
    
    # Required FHIR fields
    resourceType: str = Field(default="ServiceRequest", description="FHIR resource type")
    id: Optional[str] = Field(None, description="Logical id of this artifact")
    
    # Core ServiceRequest fields
    status: OrderStatus = Field(..., description="Current status of the order")
    intent: OrderIntent = Field(..., description="Intent of the order")
    category: Optional[List[CodeableConcept]] = Field(None, description="Classification of service")
    priority: Optional[OrderPriority] = Field(None, description="Urgency of the order")
    code: CodeableConcept = Field(..., description="What is being requested/ordered")
    
    # Patient and context
    subject: Reference = Field(..., description="Individual or entity the service is ordered for")
    encounter: Optional[Reference] = Field(None, description="Encounter during which the order was created")
    
    # Timing
    occurrence_datetime: Optional[datetime] = Field(None, alias="occurrenceDateTime", description="When service should occur")
    occurrence_period: Optional[Period] = Field(None, alias="occurrencePeriod", description="When service should occur")
    authored_on: Optional[datetime] = Field(None, alias="authoredOn", description="Date request signed")
    
    # Participants
    requester: Optional[Reference] = Field(None, description="Who/what is requesting service")
    performer_type: Optional[List[CodeableConcept]] = Field(None, alias="performerType", description="Performer role")
    performer: Optional[List[Reference]] = Field(None, description="Requested performer")
    
    # Clinical context
    reason_code: Optional[List[CodeableConcept]] = Field(None, alias="reasonCode", description="Explanation/justification for service")
    reason_reference: Optional[List[Reference]] = Field(None, alias="reasonReference", description="Explanation/justification for service")
    supporting_info: Optional[List[Reference]] = Field(None, alias="supportingInfo", description="Additional clinical information")
    
    # Instructions and notes
    note: Optional[List[Annotation]] = Field(None, description="Comments")
    patient_instruction: Optional[str] = Field(None, alias="patientInstruction", description="Patient or consumer-oriented instructions")
    
    # Order management fields
    replaces: Optional[List[Reference]] = Field(None, description="What request replaces")
    based_on: Optional[List[Reference]] = Field(None, alias="basedOn", description="What request fulfills")
    requisition: Optional[Identifier] = Field(None, description="Composite Request ID")
    
    # Metadata
    meta: Optional[Dict[str, Any]] = Field(None, description="Metadata about the resource")
    
    class Config:
        extra = "allow"  # Allow extra fields for FHIR compliance
        populate_by_name = True  # Allow field aliases
        
    def to_fhir_dict(self) -> Dict[str, Any]:
        """Convert to FHIR-compliant dictionary"""
        data = self.model_dump(by_alias=True, exclude_unset=True)
        return data
    
    @classmethod
    def from_fhir_dict(cls, fhir_dict: Dict[str, Any]) -> "ClinicalOrder":
        """Create instance from FHIR dictionary"""
        return cls.model_validate(fhir_dict)

# Order Set Model (RequestGroup)
class OrderSet(FHIRBaseModel):
    """
    FHIR RequestGroup resource for order sets.
    
    This model represents a group of related orders that are typically
    ordered together for a specific condition or protocol.
    """
    
    resourceType: str = Field(default="RequestGroup", description="FHIR resource type")
    id: Optional[str] = Field(None, description="Logical id of this artifact")
    
    # Core RequestGroup fields
    status: str = Field(..., description="Current status of the request group")
    intent: str = Field(..., description="Intent of the request group")
    priority: Optional[str] = Field(None, description="Urgency of the request group")
    
    # Identification
    identifier: Optional[List[Identifier]] = Field(None, description="Business identifier")
    code: Optional[CodeableConcept] = Field(None, description="What's being requested/ordered")
    
    # Context
    subject: Optional[Reference] = Field(None, description="Who the request group is about")
    encounter: Optional[Reference] = Field(None, description="Created during encounter")
    authored_on: Optional[datetime] = Field(None, alias="authoredOn", description="When the request group was authored")
    author: Optional[Reference] = Field(None, description="Device or practitioner that authored the request group")
    
    # Clinical context
    reason_code: Optional[List[CodeableConcept]] = Field(None, alias="reasonCode", description="Why the request group is needed")
    reason_reference: Optional[List[Reference]] = Field(None, alias="reasonReference", description="Why the request group is needed")
    
    # Instructions
    note: Optional[List[Annotation]] = Field(None, description="Additional notes about the request group")
    
    # Actions (the actual orders in the set)
    action: Optional[List[Dict[str, Any]]] = Field(None, description="Proposed actions, if any")
    
    class Config:
        extra = "allow"
        populate_by_name = True

# Order Status History for audit trail
class OrderStatusHistory(FHIRBaseModel):
    """
    Model for tracking order status changes over time.
    This provides an audit trail for order lifecycle management.
    """
    
    id: Optional[str] = Field(None, description="Unique identifier for this history entry")
    order_id: str = Field(..., description="Reference to the clinical order")
    previous_status: Optional[OrderStatus] = Field(None, description="Previous status")
    new_status: OrderStatus = Field(..., description="New status")
    change_datetime: datetime = Field(..., description="When the status changed")
    changed_by: Optional[Reference] = Field(None, description="Who changed the status")
    reason_for_change: Optional[str] = Field(None, description="Reason for the status change")
    note: Optional[str] = Field(None, description="Additional notes about the change")
    
    class Config:
        extra = "allow"

# Create and Update models for API endpoints
class ClinicalOrderCreate(BaseModel):
    """Model for creating a clinical order"""
    status: OrderStatus = OrderStatus.DRAFT
    intent: OrderIntent = OrderIntent.ORDER
    category: Optional[List[CodeableConcept]] = None
    priority: Optional[OrderPriority] = OrderPriority.ROUTINE
    code: CodeableConcept
    subject: Reference
    encounter: Optional[Reference] = None
    occurrence_datetime: Optional[datetime] = None
    occurrence_period: Optional[Period] = None
    requester: Optional[Reference] = None
    performer_type: Optional[List[CodeableConcept]] = None
    performer: Optional[List[Reference]] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    supporting_info: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    patient_instruction: Optional[str] = None

    def to_clinical_order(self) -> ClinicalOrder:
        """Convert to a FHIR ClinicalOrder."""
        data = self.model_dump(exclude_unset=True)
        data["authored_on"] = datetime.utcnow()
        return ClinicalOrder(**data)

class ClinicalOrderUpdate(BaseModel):
    """Model for updating a clinical order"""
    status: Optional[OrderStatus] = None
    priority: Optional[OrderPriority] = None
    occurrence_datetime: Optional[datetime] = None
    occurrence_period: Optional[Period] = None
    performer: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    patient_instruction: Optional[str] = None
