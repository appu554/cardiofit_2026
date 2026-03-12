"""
GraphQL Types for Order Management Service

This module defines the GraphQL types used in the Order Management Service.
"""

import strawberry
from typing import List, Optional, Dict, Any
from datetime import datetime

from app.models.clinical_order import OrderStatus, OrderIntent, OrderPriority

# GraphQL Enums
@strawberry.enum
class OrderStatusEnum(OrderStatus):
    """Order status enumeration"""
    pass

@strawberry.enum  
class OrderIntentEnum(OrderIntent):
    """Order intent enumeration"""
    pass

@strawberry.enum
class OrderPriorityEnum(OrderPriority):
    """Order priority enumeration"""
    pass

# FHIR Complex Types
@strawberry.type
class Coding:
    """FHIR Coding datatype"""
    system: Optional[str] = None
    version: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    user_selected: Optional[bool] = None

@strawberry.type
class CodeableConcept:
    """FHIR CodeableConcept datatype"""
    coding: Optional[List[Coding]] = None
    text: Optional[str] = None

@strawberry.type
class Reference:
    """FHIR Reference datatype"""
    reference: Optional[str] = None
    type: Optional[str] = None
    identifier: Optional["Identifier"] = None
    display: Optional[str] = None

@strawberry.type
class Annotation:
    """FHIR Annotation datatype"""
    author_reference: Optional[Reference] = None
    author_string: Optional[str] = None
    time: Optional[datetime] = None
    text: str

@strawberry.type
class Identifier:
    """FHIR Identifier datatype"""
    use: Optional[str] = None
    type: Optional[CodeableConcept] = None
    system: Optional[str] = None
    value: Optional[str] = None
    period: Optional["Period"] = None
    assigner: Optional[Reference] = None

@strawberry.type
class Period:
    """FHIR Period datatype"""
    start: Optional[datetime] = None
    end: Optional[datetime] = None

@strawberry.type
class Quantity:
    """FHIR Quantity datatype"""
    value: Optional[float] = None
    comparator: Optional[str] = None
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class Ratio:
    """FHIR Ratio datatype"""
    numerator: Optional[Quantity] = None
    denominator: Optional[Quantity] = None

@strawberry.type
class Range:
    """FHIR Range datatype"""
    low: Optional[Quantity] = None
    high: Optional[Quantity] = None

@strawberry.type
class Timing:
    """FHIR Timing datatype"""
    event: Optional[List[datetime]] = None
    repeat: Optional[Dict[str, Any]] = None
    code: Optional[CodeableConcept] = None

# Core Order Types
@strawberry.federation.type(keys=["id"])
class ClinicalOrder:
    """
    FHIR ServiceRequest resource for clinical orders.
    Federated entity that can be referenced by other services.
    """
    id: strawberry.ID
    resource_type: str = "ServiceRequest"
    status: OrderStatusEnum
    intent: OrderIntentEnum
    category: Optional[List[CodeableConcept]] = None
    priority: Optional[OrderPriorityEnum] = None
    code: CodeableConcept
    subject: Reference  # Patient reference
    encounter: Optional[Reference] = None
    occurrence_datetime: Optional[datetime] = None
    authored_on: Optional[datetime] = None
    requester: Optional[Reference] = None  # Practitioner reference
    performer: Optional[List[Reference]] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    supporting_info: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    patient_instruction: Optional[str] = None

# Medication-specific types
@strawberry.type
class Dosage:
    """FHIR Dosage datatype"""
    sequence: Optional[int] = None
    text: Optional[str] = None
    additional_instruction: Optional[List[CodeableConcept]] = None
    patient_instruction: Optional[str] = None
    timing: Optional[Timing] = None
    as_needed_boolean: Optional[bool] = None
    as_needed_codeable_concept: Optional[CodeableConcept] = None
    site: Optional[CodeableConcept] = None
    route: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None
    dose_and_rate: Optional[List[Dict[str, Any]]] = None
    max_dose_per_period: Optional[Ratio] = None
    max_dose_per_administration: Optional[Quantity] = None
    max_dose_per_lifetime: Optional[Quantity] = None

@strawberry.type
class DispenseRequest:
    """FHIR MedicationRequest.dispenseRequest"""
    initial_fill: Optional[Dict[str, Any]] = None
    dispense_interval: Optional[Quantity] = None
    validity_period: Optional[Period] = None
    number_of_repeats_allowed: Optional[int] = None
    quantity: Optional[Quantity] = None
    expected_supply_duration: Optional[Quantity] = None
    performer: Optional[Reference] = None

@strawberry.type
class MedicationRequestSubstitution:
    """FHIR MedicationRequest.substitution"""
    allowed_boolean: Optional[bool] = None
    allowed_codeable_concept: Optional[CodeableConcept] = None
    reason: Optional[CodeableConcept] = None

@strawberry.federation.type(keys=["id"])
class MedicationOrder:
    """
    FHIR MedicationRequest resource for medication orders.
    Comprehensive medication ordering with clinical decision support.
    """
    id: strawberry.ID
    resource_type: str = "MedicationRequest"
    identifier: Optional[List[Identifier]] = None
    status: str
    status_reason: Optional[CodeableConcept] = None
    intent: str
    category: Optional[List[CodeableConcept]] = None
    priority: Optional[str] = None
    do_not_perform: Optional[bool] = None
    reported_boolean: Optional[bool] = None
    reported_reference: Optional[Reference] = None
    medication_codeable_concept: Optional[CodeableConcept] = None
    medication_reference: Optional[Reference] = None
    subject: Reference
    encounter: Optional[Reference] = None
    supporting_information: Optional[List[Reference]] = None
    authored_on: Optional[datetime] = None
    requester: Optional[Reference] = None
    performer: Optional[Reference] = None
    performer_type: Optional[CodeableConcept] = None
    recorder: Optional[Reference] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    instantiates_canonical: Optional[List[str]] = None
    instantiates_uri: Optional[List[str]] = None
    based_on: Optional[List[Reference]] = None
    group_identifier: Optional[Identifier] = None
    course_of_therapy_type: Optional[CodeableConcept] = None
    insurance: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    dosage_instruction: Optional[List[Dosage]] = None
    dispense_request: Optional[DispenseRequest] = None
    substitution: Optional[MedicationRequestSubstitution] = None
    prior_prescription: Optional[Reference] = None
    detected_issue: Optional[List[Reference]] = None
    event_history: Optional[List[Reference]] = None

@strawberry.federation.type(keys=["id"])
class LabOrder:
    """
    Laboratory order (ServiceRequest with lab-specific fields).
    Comprehensive lab ordering with specimen requirements and clinical context.
    """
    id: strawberry.ID
    resource_type: str = "ServiceRequest"
    identifier: Optional[List[Identifier]] = None
    instantiates_canonical: Optional[List[str]] = None
    instantiates_uri: Optional[List[str]] = None
    based_on: Optional[List[Reference]] = None
    replaces: Optional[List[Reference]] = None
    requisition: Optional[Identifier] = None
    status: OrderStatusEnum
    intent: OrderIntentEnum
    category: Optional[List[CodeableConcept]] = None
    priority: Optional[OrderPriorityEnum] = None
    do_not_perform: Optional[bool] = None
    code: CodeableConcept
    order_detail: Optional[List[CodeableConcept]] = None
    quantity_quantity: Optional[Quantity] = None
    quantity_ratio: Optional[Ratio] = None
    quantity_range: Optional[Range] = None
    subject: Reference
    encounter: Optional[Reference] = None
    occurrence_datetime: Optional[datetime] = None
    occurrence_period: Optional[Period] = None
    occurrence_timing: Optional[Timing] = None
    as_needed_boolean: Optional[bool] = None
    as_needed_codeable_concept: Optional[CodeableConcept] = None
    authored_on: Optional[datetime] = None
    requester: Optional[Reference] = None
    performer_type: Optional[List[CodeableConcept]] = None
    performer: Optional[List[Reference]] = None
    location_code: Optional[List[CodeableConcept]] = None
    location_reference: Optional[List[Reference]] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    insurance: Optional[List[Reference]] = None
    supporting_info: Optional[List[Reference]] = None
    specimen: Optional[List[Reference]] = None
    body_site: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None
    patient_instruction: Optional[str] = None
    relevant_history: Optional[List[Reference]] = None
    # Lab-specific fields
    test_name: Optional[str] = None
    specimen_source: Optional[str] = None
    fasting_required: Optional[bool] = None
    collection_datetime_preference: Optional[datetime] = None
    clinical_history: Optional[str] = None
    lab_priority: Optional[str] = None
    expected_turnaround_time: Optional[Quantity] = None

@strawberry.federation.type(keys=["id"])
class ImagingOrder:
    """
    Imaging order (ServiceRequest with imaging-specific fields).
    Comprehensive imaging ordering with modality, contrast, and clinical requirements.
    """
    id: strawberry.ID
    resource_type: str = "ServiceRequest"
    identifier: Optional[List[Identifier]] = None
    instantiates_canonical: Optional[List[str]] = None
    instantiates_uri: Optional[List[str]] = None
    based_on: Optional[List[Reference]] = None
    replaces: Optional[List[Reference]] = None
    requisition: Optional[Identifier] = None
    status: OrderStatusEnum
    intent: OrderIntentEnum
    category: Optional[List[CodeableConcept]] = None
    priority: Optional[OrderPriorityEnum] = None
    do_not_perform: Optional[bool] = None
    code: CodeableConcept
    order_detail: Optional[List[CodeableConcept]] = None
    quantity_quantity: Optional[Quantity] = None
    quantity_ratio: Optional[Ratio] = None
    quantity_range: Optional[Range] = None
    subject: Reference
    encounter: Optional[Reference] = None
    occurrence_datetime: Optional[datetime] = None
    occurrence_period: Optional[Period] = None
    occurrence_timing: Optional[Timing] = None
    as_needed_boolean: Optional[bool] = None
    as_needed_codeable_concept: Optional[CodeableConcept] = None
    authored_on: Optional[datetime] = None
    requester: Optional[Reference] = None
    performer_type: Optional[List[CodeableConcept]] = None
    performer: Optional[List[Reference]] = None
    location_code: Optional[List[CodeableConcept]] = None
    location_reference: Optional[List[Reference]] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    insurance: Optional[List[Reference]] = None
    supporting_info: Optional[List[Reference]] = None
    specimen: Optional[List[Reference]] = None
    body_site: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None
    patient_instruction: Optional[str] = None
    relevant_history: Optional[List[Reference]] = None
    # Imaging-specific fields
    procedure_name: Optional[str] = None
    modality: Optional[str] = None
    contrast_required: Optional[bool] = None
    contrast_agent: Optional[CodeableConcept] = None
    laterality: Optional[str] = None
    transport_mode: Optional[str] = None
    clinical_history_for_radiologist: Optional[str] = None
    clinical_question: Optional[str] = None
    pregnancy_status: Optional[bool] = None
    radiation_dose_estimate: Optional[Quantity] = None

# Clinical Decision Support Types
@strawberry.type
class DrugInteractionAlert:
    """Clinical decision support alert for drug interactions"""
    id: strawberry.ID
    severity: str  # high, moderate, low
    interaction_type: str  # drug-drug, drug-allergy, drug-condition
    description: str
    recommendation: Optional[str] = None
    source: Optional[str] = None
    evidence_level: Optional[str] = None
    affected_medications: List[Reference]
    clinical_significance: Optional[str] = None

@strawberry.type
class ClinicalAlert:
    """General clinical decision support alert"""
    id: strawberry.ID
    alert_type: str  # interaction, contraindication, duplicate, formulary
    severity: str  # critical, warning, info
    title: str
    description: str
    recommendation: Optional[str] = None
    source: Optional[str] = None
    triggered_by: Optional[Reference] = None
    patient_context: Optional[List[str]] = None

@strawberry.type
class FormularyStatus:
    """Medication formulary status and alternatives"""
    medication: Reference
    formulary_status: str  # preferred, non-preferred, not-covered
    tier: Optional[int] = None
    prior_authorization_required: Optional[bool] = None
    step_therapy_required: Optional[bool] = None
    quantity_limits: Optional[str] = None
    alternatives: Optional[List[Reference]] = None
    cost_estimate: Optional[Quantity] = None

# Order Set Types
@strawberry.type
class OrderSetAction:
    """FHIR RequestGroup.action for order set items"""
    id: Optional[str] = None
    prefix: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    text_equivalent: Optional[str] = None
    priority: Optional[str] = None
    code: Optional[List[CodeableConcept]] = None
    reason: Optional[List[CodeableConcept]] = None
    documentation: Optional[List[Dict[str, Any]]] = None
    goal_id: Optional[List[str]] = None
    subject_codeable_concept: Optional[CodeableConcept] = None
    subject_reference: Optional[Reference] = None
    trigger: Optional[List[Dict[str, Any]]] = None
    condition: Optional[List[Dict[str, Any]]] = None
    input: Optional[List[Dict[str, Any]]] = None
    output: Optional[List[Dict[str, Any]]] = None
    related_action: Optional[List[Dict[str, Any]]] = None
    timing_datetime: Optional[datetime] = None
    timing_age: Optional[Quantity] = None
    timing_period: Optional[Period] = None
    timing_duration: Optional[Quantity] = None
    timing_range: Optional[Range] = None
    timing_timing: Optional[Timing] = None
    participant: Optional[List[Dict[str, Any]]] = None
    type: Optional[CodeableConcept] = None
    grouping_behavior: Optional[str] = None
    selection_behavior: Optional[str] = None
    required_behavior: Optional[str] = None
    precheck_behavior: Optional[str] = None
    cardinality_behavior: Optional[str] = None
    definition_canonical: Optional[str] = None
    definition_uri: Optional[str] = None
    transform: Optional[str] = None
    dynamic_value: Optional[List[Dict[str, Any]]] = None
    action: Optional[List["OrderSetAction"]] = None

@strawberry.federation.type(keys=["id"])
class OrderSet:
    """
    FHIR RequestGroup resource for order sets.
    Comprehensive order set management with clinical protocols.
    """
    id: strawberry.ID
    resource_type: str = "RequestGroup"
    identifier: Optional[List[Identifier]] = None
    instantiates_canonical: Optional[List[str]] = None
    instantiates_uri: Optional[List[str]] = None
    based_on: Optional[List[Reference]] = None
    replaces: Optional[List[Reference]] = None
    group_identifier: Optional[Identifier] = None
    status: str
    intent: str
    priority: Optional[str] = None
    code: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    encounter: Optional[Reference] = None
    authored_on: Optional[datetime] = None
    author: Optional[Reference] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    action: Optional[List[OrderSetAction]] = None
    # Order set specific fields
    name: Optional[str] = None
    title: Optional[str] = None
    description: Optional[str] = None
    version: Optional[str] = None
    category: Optional[List[CodeableConcept]] = None
    applicable_context: Optional[List[CodeableConcept]] = None
    usage_instructions: Optional[str] = None
    clinical_guidelines: Optional[List[str]] = None

# Order Status History and Audit Trail
@strawberry.federation.type(keys=["id"])
class OrderStatusHistory:
    """Order status change history for audit trail"""
    id: strawberry.ID
    order_id: str
    previous_status: Optional[str] = None
    new_status: str
    change_datetime: datetime
    changed_by: Reference
    reason_for_change: Optional[str] = None
    change_type: str  # status_change, modification, cancellation, etc.
    change_details: Optional[Dict[str, Any]] = None
    system_generated: Optional[bool] = None

@strawberry.type
class OrderSignature:
    """Order signature and attestation"""
    id: strawberry.ID
    order_id: str
    signature_type: str  # electronic, digital, wet
    signed_by: Reference
    signed_datetime: datetime
    signature_method: Optional[str] = None
    attestation_text: Optional[str] = None
    co_signature_required: Optional[bool] = None
    co_signed_by: Optional[Reference] = None
    co_signed_datetime: Optional[datetime] = None

# Connection types for pagination
@strawberry.type
class OrderConnection:
    """Connection type for paginated order results"""
    edges: List["OrderEdge"]
    page_info: "PageInfo"
    total_count: int

@strawberry.type
class OrderEdge:
    """Edge type for order connections"""
    node: ClinicalOrder
    cursor: str

@strawberry.type
class PageInfo:
    """Pagination information"""
    has_next_page: bool
    has_previous_page: bool
    start_cursor: Optional[str] = None
    end_cursor: Optional[str] = None

# Search and Filter types
@strawberry.type
class OrderSearchResult:
    """Comprehensive search result for orders"""
    orders: List[ClinicalOrder]
    total_count: int
    search_params: Dict[str, Any]
    facets: Optional[Dict[str, Any]] = None
    suggestions: Optional[List[str]] = None

@strawberry.type
class OrderStatistics:
    """Comprehensive order statistics for dashboards"""
    total_orders: int
    draft_orders: int
    active_orders: int
    completed_orders: int
    cancelled_orders: int
    on_hold_orders: int
    orders_by_priority: Dict[str, int]
    orders_by_category: Dict[str, int]
    orders_by_requester: Dict[str, int]
    orders_by_date_range: Dict[str, int]
    average_completion_time: Optional[float] = None
    most_common_orders: List[Dict[str, Any]]

# Union types for polymorphic order details
@strawberry.union
class OrderDetails:
    """Union type for different order detail types"""
    types = (MedicationOrder, LabOrder, ImagingOrder)

# Clinical Context Types
@strawberry.type
class PatientContext:
    """Patient clinical context for order decision support"""
    patient_id: str
    age: Optional[int] = None
    weight: Optional[Quantity] = None
    height: Optional[Quantity] = None
    allergies: Optional[List[Reference]] = None
    current_medications: Optional[List[Reference]] = None
    active_conditions: Optional[List[Reference]] = None
    recent_lab_results: Optional[List[Reference]] = None
    pregnancy_status: Optional[bool] = None
    renal_function: Optional[str] = None
    hepatic_function: Optional[str] = None

# Extended entities from other services
@strawberry.federation.type(keys=["id"], extend=True)
class Patient:
    """
    Extended Patient entity from Patient Service.
    Adds order-related fields to the Patient type.
    """
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def orders(self) -> List[ClinicalOrder]:
        """Get all orders for this patient"""
        try:
            from app.services.order_service import get_order_service
            from .resolvers import convert_order_to_graphql

            order_service = await get_order_service()
            search_params = {"subject": f"Patient/{self.id}"}
            orders = await order_service.search_clinical_orders(search_params)
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.error(f"Error getting orders for patient {self.id}: {e}")
            return []

    @strawberry.field
    async def active_orders(self) -> List[ClinicalOrder]:
        """Get active orders for this patient"""
        try:
            from app.services.order_service import get_order_service
            from .resolvers import convert_order_to_graphql

            order_service = await get_order_service()
            search_params = {
                "subject": f"Patient/{self.id}",
                "status": "active"
            }
            orders = await order_service.search_clinical_orders(search_params)
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.error(f"Error getting active orders for patient {self.id}: {e}")
            return []

@strawberry.federation.type(keys=["id"], extend=True)
class Practitioner:
    """
    Extended Practitioner entity from Organization Service.
    Adds order-related fields to the Practitioner type.
    """
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def orders_requested(self) -> List[ClinicalOrder]:
        """Get orders requested by this practitioner"""
        try:
            from app.services.order_service import get_order_service
            from .resolvers import convert_order_to_graphql

            order_service = await get_order_service()
            search_params = {"requester": f"Practitioner/{self.id}"}
            orders = await order_service.search_clinical_orders(search_params)
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.error(f"Error getting orders for practitioner {self.id}: {e}")
            return []

    @strawberry.field
    async def pending_orders(self) -> List[ClinicalOrder]:
        """Get pending orders for this practitioner"""
        try:
            from app.services.order_service import get_order_service
            from .resolvers import convert_order_to_graphql

            order_service = await get_order_service()
            search_params = {
                "requester": f"Practitioner/{self.id}",
                "status": "draft"
            }
            orders = await order_service.search_clinical_orders(search_params)
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.error(f"Error getting pending orders for practitioner {self.id}: {e}")
            return []

@strawberry.federation.type(keys=["id"], extend=True)
class Encounter:
    """
    Extended Encounter entity from Encounter Service.
    Adds order-related fields to the Encounter type.
    """
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def orders(self) -> List[ClinicalOrder]:
        """Get orders for this encounter"""
        try:
            from app.services.order_service import get_order_service
            from .resolvers import convert_order_to_graphql

            order_service = await get_order_service()
            search_params = {"encounter": f"Encounter/{self.id}"}
            orders = await order_service.search_clinical_orders(search_params)
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.error(f"Error getting orders for encounter {self.id}: {e}")
            return []
