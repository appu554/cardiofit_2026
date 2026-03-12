"""
Order Set Models for Order Management Service

This module provides FHIR-compliant models for order sets,
implementing the FHIR RequestGroup resource for grouped orders.
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

# Order Set Status
class OrderSetStatus(str, Enum):
    """FHIR RequestGroup status values"""
    DRAFT = "draft"
    ACTIVE = "active"
    ON_HOLD = "on-hold"
    REVOKED = "revoked"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"
    UNKNOWN = "unknown"

class OrderSetIntent(str, Enum):
    """FHIR RequestGroup intent values"""
    PROPOSAL = "proposal"
    PLAN = "plan"
    DIRECTIVE = "directive"
    ORDER = "order"
    ORIGINAL_ORDER = "original-order"
    REFLEX_ORDER = "reflex-order"
    FILLER_ORDER = "filler-order"
    INSTANCE_ORDER = "instance-order"
    OPTION = "option"

class OrderSetPriority(str, Enum):
    """FHIR RequestGroup priority values"""
    ROUTINE = "routine"
    URGENT = "urgent"
    ASAP = "asap"
    STAT = "stat"

class ActionSelectionBehavior(str, Enum):
    """How actions should be selected"""
    ANY = "any"
    ALL = "all"
    ALL_OR_NONE = "all-or-none"
    EXACTLY_ONE = "exactly-one"
    AT_MOST_ONE = "at-most-one"
    ONE_OR_MORE = "one-or-more"

class ActionRequiredBehavior(str, Enum):
    """Whether actions are required"""
    MUST = "must"
    COULD = "could"
    MUST_UNLESS_DOCUMENTED = "must-unless-documented"

class ActionPrecheckBehavior(str, Enum):
    """Whether actions should be preselected"""
    YES = "yes"
    NO = "no"

# Order Set Action Model
class OrderSetAction(FHIRBaseModel):
    """
    Individual action within an order set.
    """
    id: Optional[str] = Field(None, description="Unique id for action in order set")
    prefix: Optional[str] = Field(None, description="User-visible prefix for the action")
    title: Optional[str] = Field(None, description="User-visible title")
    description: Optional[str] = Field(None, description="Brief description of the action")
    text_equivalent: Optional[str] = Field(None, alias="textEquivalent", description="Static text equivalent of the action")
    
    # Action behavior
    priority: Optional[str] = Field(None, description="Urgency of the action")
    code: Optional[List[CodeableConcept]] = Field(None, description="Code representing the meaning of the action")
    reason: Optional[List[CodeableConcept]] = Field(None, description="Why the action should be performed")
    documentation: Optional[List[Reference]] = Field(None, description="Supporting documentation for the action")
    
    # Selection behavior
    selection_behavior: Optional[ActionSelectionBehavior] = Field(None, alias="selectionBehavior", description="How actions should be selected")
    required_behavior: Optional[ActionRequiredBehavior] = Field(None, alias="requiredBehavior", description="Whether the action is required")
    precheck_behavior: Optional[ActionPrecheckBehavior] = Field(None, alias="precheckBehavior", description="Whether the action should be preselected")
    
    # Timing
    timing_datetime: Optional[datetime] = Field(None, alias="timingDateTime", description="When the action should take place")
    timing_age: Optional[Dict[str, Any]] = Field(None, alias="timingAge", description="When the action should take place")
    timing_period: Optional[Period] = Field(None, alias="timingPeriod", description="When the action should take place")
    timing_duration: Optional[Dict[str, Any]] = Field(None, alias="timingDuration", description="When the action should take place")
    timing_range: Optional[Dict[str, Any]] = Field(None, alias="timingRange", description="When the action should take place")
    timing_timing: Optional[Dict[str, Any]] = Field(None, alias="timingTiming", description="When the action should take place")
    
    # Participants
    participant: Optional[List[Reference]] = Field(None, description="Who should participate in the action")
    
    # Action type
    type: Optional[CodeableConcept] = Field(None, description="The type of action to perform")
    grouping_behavior: Optional[str] = Field(None, alias="groupingBehavior", description="Defines the grouping behavior for the action")
    
    # Resource reference
    resource: Optional[Reference] = Field(None, description="The target of the action")
    
    # Nested actions
    action: Optional[List["OrderSetAction"]] = Field(None, description="Sub actions")
    
    # Conditions
    condition: Optional[List[Dict[str, Any]]] = Field(None, description="Whether or not the action is applicable")
    
    # Related actions
    related_action: Optional[List[Dict[str, Any]]] = Field(None, alias="relatedAction", description="Relationship to another action")
    
    # Transform
    transform: Optional[Reference] = Field(None, description="Transform to apply the template")
    
    # Dynamic values
    dynamic_value: Optional[List[Dict[str, Any]]] = Field(None, alias="dynamicValue", description="Dynamic aspects of the definition")
    
    class Config:
        extra = "allow"
        populate_by_name = True

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
    status: OrderSetStatus = Field(..., description="Current status of the request group")
    intent: OrderSetIntent = Field(..., description="Intent of the request group")
    priority: Optional[OrderSetPriority] = Field(None, description="Urgency of the request group")
    
    # Identification
    identifier: Optional[List[Identifier]] = Field(None, description="Business identifier")
    instantiates_canonical: Optional[List[str]] = Field(None, alias="instantiatesCanonical", description="Instantiates FHIR protocol or definition")
    instantiates_uri: Optional[List[str]] = Field(None, alias="instantiatesUri", description="Instantiates external protocol or definition")
    based_on: Optional[List[Reference]] = Field(None, alias="basedOn", description="Fulfills plan, proposal, or order")
    replaces: Optional[List[Reference]] = Field(None, description="Request(s) replaced by this request")
    group_identifier: Optional[Identifier] = Field(None, alias="groupIdentifier", description="Composite request this is part of")
    
    # Classification
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
    action: Optional[List[OrderSetAction]] = Field(None, description="Proposed actions, if any")
    
    # Order set specific fields
    name: Optional[str] = Field(None, description="Human-readable name for the order set")
    title: Optional[str] = Field(None, description="Name for this order set (human friendly)")
    description: Optional[str] = Field(None, description="Natural language description of the order set")
    use_context: Optional[List[Dict[str, Any]]] = Field(None, alias="useContext", description="The context that the content is intended to support")
    jurisdiction: Optional[List[CodeableConcept]] = Field(None, description="Intended jurisdiction for order set")
    purpose: Optional[str] = Field(None, description="Why this order set is defined")
    usage: Optional[str] = Field(None, description="Describes the clinical usage of the order set")
    copyright: Optional[str] = Field(None, description="Use and/or publishing restrictions")
    approval_date: Optional[datetime] = Field(None, alias="approvalDate", description="When the order set was approved by publisher")
    last_review_date: Optional[datetime] = Field(None, alias="lastReviewDate", description="When the order set was last reviewed")
    effective_period: Optional[Period] = Field(None, alias="effectivePeriod", description="When the order set is expected to be used")
    topic: Optional[List[CodeableConcept]] = Field(None, description="E.g. Education, Treatment, Assessment")
    contributor: Optional[List[Dict[str, Any]]] = Field(None, description="Who contributed to the content")
    related_artifact: Optional[List[Dict[str, Any]]] = Field(None, alias="relatedArtifact", description="Additional documentation, citations")
    
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
    def from_fhir_dict(cls, fhir_dict: Dict[str, Any]) -> "OrderSet":
        """Create instance from FHIR dictionary"""
        return cls.model_validate(fhir_dict)

# Create and Update models for API endpoints
class OrderSetCreate(BaseModel):
    """Model for creating an order set"""
    status: OrderSetStatus = OrderSetStatus.DRAFT
    intent: OrderSetIntent = OrderSetIntent.PLAN
    priority: Optional[OrderSetPriority] = None
    code: Optional[CodeableConcept] = None
    name: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    purpose: Optional[str] = None
    usage: Optional[str] = None
    topic: Optional[List[CodeableConcept]] = None
    action: Optional[List[OrderSetAction]] = None
    reason_code: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None
    
    def to_order_set(self) -> OrderSet:
        """Convert to a FHIR OrderSet."""
        data = self.model_dump(exclude_unset=True)
        data["authored_on"] = datetime.utcnow()
        return OrderSet(**data)

class OrderSetUpdate(BaseModel):
    """Model for updating an order set"""
    status: Optional[OrderSetStatus] = None
    priority: Optional[OrderSetPriority] = None
    name: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    purpose: Optional[str] = None
    usage: Optional[str] = None
    action: Optional[List[OrderSetAction]] = None
    note: Optional[List[Annotation]] = None

# Order Set Application Model
class OrderSetApplication(BaseModel):
    """Model for applying an order set to a patient"""
    order_set_id: str = Field(..., description="ID of the order set to apply")
    patient_id: str = Field(..., description="ID of the patient")
    encounter_id: Optional[str] = Field(None, description="ID of the encounter")
    requester_id: Optional[str] = Field(None, description="ID of the requesting practitioner")
    selected_actions: Optional[List[str]] = Field(None, description="IDs of selected actions (if not all)")
    customizations: Optional[Dict[str, Any]] = Field(None, description="Customizations to apply")
    reason_code: Optional[List[CodeableConcept]] = Field(None, description="Reason for applying the order set")
    note: Optional[str] = Field(None, description="Additional notes")

# Update the forward reference
OrderSetAction.model_rebuild()
