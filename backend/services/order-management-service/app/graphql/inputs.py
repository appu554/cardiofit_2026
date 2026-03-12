"""
GraphQL Input Types for Order Management Service

This module defines the GraphQL input types used for mutations and queries.
"""

import strawberry
from typing import List, Optional, Dict, Any
from datetime import datetime

from app.models.clinical_order import OrderStatus, OrderIntent, OrderPriority

# Import enums from types
from .types import OrderStatusEnum, OrderIntentEnum, OrderPriorityEnum

# FHIR Complex Input Types
@strawberry.input
class CodeableConceptInput:
    """Input type for FHIR CodeableConcept"""
    coding: Optional[List[Dict[str, Any]]] = None
    text: Optional[str] = None

@strawberry.input
class ReferenceInput:
    """Input type for FHIR Reference"""
    reference: Optional[str] = None
    display: Optional[str] = None

@strawberry.input
class AnnotationInput:
    """Input type for FHIR Annotation"""
    text: str
    author_string: Optional[str] = None

@strawberry.input
class IdentifierInput:
    """Input type for FHIR Identifier"""
    use: Optional[str] = None
    system: Optional[str] = None
    value: Optional[str] = None

@strawberry.input
class PeriodInput:
    """Input type for FHIR Period"""
    start: Optional[datetime] = None
    end: Optional[datetime] = None

@strawberry.input
class QuantityInput:
    """Input type for FHIR Quantity"""
    value: Optional[float] = None
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None

# Order Input Types
@strawberry.input
class ClinicalOrderInput:
    """Input type for creating clinical orders"""
    status: OrderStatusEnum = OrderStatusEnum.DRAFT
    intent: OrderIntentEnum = OrderIntentEnum.ORDER
    category: Optional[List[CodeableConceptInput]] = None
    priority: Optional[OrderPriorityEnum] = OrderPriorityEnum.ROUTINE
    code: CodeableConceptInput
    subject: ReferenceInput
    encounter: Optional[ReferenceInput] = None
    occurrence_datetime: Optional[datetime] = None
    requester: Optional[ReferenceInput] = None
    performer: Optional[List[ReferenceInput]] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    reason_reference: Optional[List[ReferenceInput]] = None
    supporting_info: Optional[List[ReferenceInput]] = None
    note: Optional[List[AnnotationInput]] = None
    patient_instruction: Optional[str] = None

@strawberry.input
class ClinicalOrderUpdateInput:
    """Input type for updating clinical orders"""
    status: Optional[OrderStatusEnum] = None
    priority: Optional[OrderPriorityEnum] = None
    occurrence_datetime: Optional[datetime] = None
    performer: Optional[List[ReferenceInput]] = None
    note: Optional[List[AnnotationInput]] = None
    patient_instruction: Optional[str] = None

@strawberry.input
class MedicationOrderInput:
    """Input type for creating medication orders"""
    status: str = "draft"
    intent: str = "order"
    medication_codeable_concept: Optional[CodeableConceptInput] = None
    subject: ReferenceInput
    encounter: Optional[ReferenceInput] = None
    requester: Optional[ReferenceInput] = None
    dosage_instruction: Optional[List[Dict[str, Any]]] = None
    dispense_request: Optional[Dict[str, Any]] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class LabOrderInput:
    """Input type for creating laboratory orders"""
    status: OrderStatusEnum = OrderStatusEnum.DRAFT
    intent: OrderIntentEnum = OrderIntentEnum.ORDER
    priority: Optional[OrderPriorityEnum] = OrderPriorityEnum.ROUTINE
    code: CodeableConceptInput
    subject: ReferenceInput
    encounter: Optional[ReferenceInput] = None
    requester: Optional[ReferenceInput] = None
    test_name: Optional[str] = None
    specimen_source: Optional[str] = None
    fasting_required: Optional[bool] = False
    collection_datetime_preference: Optional[datetime] = None
    clinical_history: Optional[str] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class ImagingOrderInput:
    """Input type for creating imaging orders"""
    status: OrderStatusEnum = OrderStatusEnum.DRAFT
    intent: OrderIntentEnum = OrderIntentEnum.ORDER
    priority: Optional[OrderPriorityEnum] = OrderPriorityEnum.ROUTINE
    code: CodeableConceptInput
    subject: ReferenceInput
    encounter: Optional[ReferenceInput] = None
    requester: Optional[ReferenceInput] = None
    procedure_name: Optional[str] = None
    modality: Optional[str] = None
    body_site: Optional[str] = None
    contrast_required: Optional[bool] = False
    clinical_history_for_radiologist: Optional[str] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class OrderSetInput:
    """Input type for creating order sets"""
    status: str = "draft"
    intent: str = "plan"
    name: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    code: Optional[CodeableConceptInput] = None
    purpose: Optional[str] = None
    usage: Optional[str] = None
    topic: Optional[List[CodeableConceptInput]] = None
    action: Optional[List[Dict[str, Any]]] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class OrderSetUpdateInput:
    """Input type for updating order sets"""
    status: Optional[str] = None
    name: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    purpose: Optional[str] = None
    usage: Optional[str] = None
    action: Optional[List[Dict[str, Any]]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class OrderSetApplicationInput:
    """Input type for applying an order set to a patient"""
    order_set_id: str
    patient_id: str
    encounter_id: Optional[str] = None
    requester_id: Optional[str] = None
    selected_actions: Optional[List[str]] = None
    customizations: Optional[Dict[str, Any]] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    note: Optional[str] = None

# Order Management Action Inputs
@strawberry.input
class OrderActionInput:
    """Input type for order management actions"""
    reason: Optional[str] = None
    note: Optional[str] = None

@strawberry.input
class SignatureInput:
    """Input type for order signatures"""
    signature_type: str = "electronic"
    signature_data: Optional[str] = None
    reason: Optional[str] = None

# Search and Filter Inputs
@strawberry.input
class OrderSearchFilters:
    """Input type for order search filters"""
    patient_id: Optional[str] = None
    practitioner_id: Optional[str] = None
    encounter_id: Optional[str] = None
    status: Optional[List[OrderStatusEnum]] = None
    priority: Optional[List[OrderPriorityEnum]] = None
    category: Optional[List[str]] = None
    date_from: Optional[datetime] = None
    date_to: Optional[datetime] = None
    code_system: Optional[str] = None
    code_value: Optional[str] = None

@strawberry.input
class OrderSortInput:
    """Input type for order sorting"""
    field: str = "authored_on"
    direction: str = "desc"  # "asc" or "desc"

@strawberry.input
class PaginationInput:
    """Input type for pagination"""
    limit: int = 50
    offset: int = 0
    cursor: Optional[str] = None

# Bulk Operations
@strawberry.input
class BulkOrderActionInput:
    """Input type for bulk order actions"""
    order_ids: List[str]
    action: str  # "sign", "cancel", "hold", "release"
    reason: Optional[str] = None
    note: Optional[str] = None

@strawberry.input
class OrderBatchInput:
    """Input type for creating multiple orders at once"""
    orders: List[ClinicalOrderInput]
    encounter_id: Optional[str] = None
    requester_id: Optional[str] = None

# Clinical Decision Support Inputs
@strawberry.input
class CDSCheckInput:
    """Input type for clinical decision support checks"""
    patient_id: str
    order_data: ClinicalOrderInput
    check_types: Optional[List[str]] = None  # ["drug_interactions", "allergies", "contraindications"]

# Order Template Inputs
@strawberry.input
class OrderTemplateInput:
    """Input type for order templates"""
    name: str
    description: Optional[str] = None
    category: str
    template_data: ClinicalOrderInput
    is_active: bool = True
    specialty: Optional[str] = None
