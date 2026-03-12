"""
GraphQL Resolvers for Order Management Service

This module contains helper functions for converting between FHIR models
and GraphQL types, and other resolver utilities.
"""

import logging
from typing import Dict, List, Optional, Any

from .types import (
    ClinicalOrder, CodeableConcept, Reference, Annotation,
    OrderStatusEnum, OrderIntentEnum, OrderPriorityEnum
)
from .inputs import ClinicalOrderInput

logger = logging.getLogger(__name__)

def convert_order_to_graphql(order) -> ClinicalOrder:
    """Convert FHIR order model to GraphQL type"""
    try:
        return ClinicalOrder(
            id=order.id,
            resource_type=getattr(order, "resourceType", "ServiceRequest"),
            status=OrderStatusEnum(order.status),
            intent=OrderIntentEnum(order.intent),
            category=[_convert_codeable_concept_to_graphql(cat) for cat in (order.category or [])],
            priority=OrderPriorityEnum(order.priority) if order.priority else None,
            code=_convert_codeable_concept_to_graphql(order.code),
            subject=_convert_reference_to_graphql(order.subject),
            encounter=_convert_reference_to_graphql(order.encounter) if order.encounter else None,
            occurrence_datetime=order.occurrence_datetime,
            authored_on=order.authored_on,
            requester=_convert_reference_to_graphql(order.requester) if order.requester else None,
            performer=[_convert_reference_to_graphql(perf) for perf in (order.performer or [])],
            reason_code=[_convert_codeable_concept_to_graphql(reason) for reason in (order.reason_code or [])],
            reason_reference=[_convert_reference_to_graphql(ref) for ref in (order.reason_reference or [])],
            supporting_info=[_convert_reference_to_graphql(info) for info in (order.supporting_info or [])],
            note=[_convert_annotation_to_graphql(note) for note in (order.note or [])],
            patient_instruction=order.patient_instruction
        )
    except Exception as e:
        logger.error(f"Error converting order to GraphQL: {e}")
        # Return a minimal order on error
        return ClinicalOrder(
            id=getattr(order, "id", "unknown"),
            status=OrderStatusEnum.UNKNOWN,
            intent=OrderIntentEnum.ORDER,
            code=CodeableConcept(text="Error loading order"),
            subject=Reference(reference="Patient/unknown")
        )

def convert_graphql_input_to_model(order_input: ClinicalOrderInput):
    """Convert GraphQL input to FHIR order model"""
    from app.models.clinical_order import ClinicalOrderCreate
    
    return ClinicalOrderCreate(
        status=order_input.status.value,
        intent=order_input.intent.value,
        category=[_convert_codeable_concept_input_to_dict(cat) for cat in (order_input.category or [])],
        priority=order_input.priority.value if order_input.priority else None,
        code=_convert_codeable_concept_input_to_dict(order_input.code),
        subject=_convert_reference_input_to_dict(order_input.subject),
        encounter=_convert_reference_input_to_dict(order_input.encounter) if order_input.encounter else None,
        occurrence_datetime=order_input.occurrence_datetime,
        requester=_convert_reference_input_to_dict(order_input.requester) if order_input.requester else None,
        performer=[_convert_reference_input_to_dict(perf) for perf in (order_input.performer or [])],
        reason_code=[_convert_codeable_concept_input_to_dict(reason) for reason in (order_input.reason_code or [])],
        reason_reference=[_convert_reference_input_to_dict(ref) for ref in (order_input.reason_reference or [])],
        supporting_info=[_convert_reference_input_to_dict(info) for info in (order_input.supporting_info or [])],
        note=[_convert_annotation_input_to_dict(note) for note in (order_input.note or [])],
        patient_instruction=order_input.patient_instruction
    )

# Helper functions for converting FHIR complex types
def _convert_codeable_concept_to_graphql(cc) -> CodeableConcept:
    """Convert FHIR CodeableConcept to GraphQL type"""
    if not cc:
        return CodeableConcept()
    
    if isinstance(cc, dict):
        return CodeableConcept(
            coding=cc.get("coding", []),
            text=cc.get("text")
        )
    else:
        return CodeableConcept(
            coding=getattr(cc, "coding", []),
            text=getattr(cc, "text", None)
        )

def _convert_reference_to_graphql(ref) -> Reference:
    """Convert FHIR Reference to GraphQL type"""
    if not ref:
        return Reference()
    
    if isinstance(ref, dict):
        return Reference(
            reference=ref.get("reference"),
            display=ref.get("display")
        )
    else:
        return Reference(
            reference=getattr(ref, "reference", None),
            display=getattr(ref, "display", None)
        )

def _convert_annotation_to_graphql(ann) -> Annotation:
    """Convert FHIR Annotation to GraphQL type"""
    if not ann:
        return Annotation(text="")
    
    if isinstance(ann, dict):
        return Annotation(
            text=ann.get("text", ""),
            author_string=ann.get("author_string"),
            time=ann.get("time")
        )
    else:
        return Annotation(
            text=getattr(ann, "text", ""),
            author_string=getattr(ann, "author_string", None),
            time=getattr(ann, "time", None)
        )

def _convert_codeable_concept_input_to_dict(cc_input) -> Dict[str, Any]:
    """Convert CodeableConceptInput to dictionary"""
    if not cc_input:
        return {}
    
    return {
        "coding": cc_input.coding or [],
        "text": cc_input.text
    }

def _convert_reference_input_to_dict(ref_input) -> Dict[str, Any]:
    """Convert ReferenceInput to dictionary"""
    if not ref_input:
        return {}
    
    return {
        "reference": ref_input.reference,
        "display": ref_input.display
    }

def _convert_annotation_input_to_dict(ann_input) -> Dict[str, Any]:
    """Convert AnnotationInput to dictionary"""
    if not ann_input:
        return {}
    
    return {
        "text": ann_input.text,
        "author_string": ann_input.author_string
    }

# Entity resolution functions for Apollo Federation
async def resolve_patient_orders(patient_id: str) -> List[ClinicalOrder]:
    """Resolve orders for a patient entity"""
    try:
        from app.services.order_service import get_order_service
        
        order_service = await get_order_service()
        search_params = {"subject": f"Patient/{patient_id}"}
        orders = await order_service.search_clinical_orders(search_params)
        
        return [convert_order_to_graphql(order) for order in orders]
    except Exception as e:
        logger.error(f"Error resolving orders for patient {patient_id}: {e}")
        return []

async def resolve_practitioner_orders(practitioner_id: str) -> List[ClinicalOrder]:
    """Resolve orders for a practitioner entity"""
    try:
        from app.services.order_service import get_order_service
        
        order_service = await get_order_service()
        search_params = {"requester": f"Practitioner/{practitioner_id}"}
        orders = await order_service.search_clinical_orders(search_params)
        
        return [convert_order_to_graphql(order) for order in orders]
    except Exception as e:
        logger.error(f"Error resolving orders for practitioner {practitioner_id}: {e}")
        return []

async def resolve_encounter_orders(encounter_id: str) -> List[ClinicalOrder]:
    """Resolve orders for an encounter entity"""
    try:
        from app.services.order_service import get_order_service
        
        order_service = await get_order_service()
        search_params = {"encounter": f"Encounter/{encounter_id}"}
        orders = await order_service.search_clinical_orders(search_params)
        
        return [convert_order_to_graphql(order) for order in orders]
    except Exception as e:
        logger.error(f"Error resolving orders for encounter {encounter_id}: {e}")
        return []

# Validation helpers
def validate_order_input(order_input: ClinicalOrderInput) -> List[str]:
    """Validate order input and return list of errors"""
    errors = []
    
    if not order_input.code or not order_input.code.text:
        errors.append("Order code is required")
    
    if not order_input.subject or not order_input.subject.reference:
        errors.append("Patient reference is required")
    
    if order_input.subject and order_input.subject.reference:
        if not order_input.subject.reference.startswith("Patient/"):
            errors.append("Subject must be a Patient reference")
    
    if order_input.requester and order_input.requester.reference:
        if not order_input.requester.reference.startswith("Practitioner/"):
            errors.append("Requester must be a Practitioner reference")
    
    if order_input.encounter and order_input.encounter.reference:
        if not order_input.encounter.reference.startswith("Encounter/"):
            errors.append("Encounter must be an Encounter reference")
    
    return errors

# Search helpers
def build_search_params(filters: Dict[str, Any]) -> Dict[str, str]:
    """Build FHIR search parameters from GraphQL filters"""
    search_params = {}
    
    if filters.get("patient_id"):
        search_params["subject"] = f"Patient/{filters['patient_id']}"
    
    if filters.get("practitioner_id"):
        search_params["requester"] = f"Practitioner/{filters['practitioner_id']}"
    
    if filters.get("encounter_id"):
        search_params["encounter"] = f"Encounter/{filters['encounter_id']}"
    
    if filters.get("status"):
        if isinstance(filters["status"], list):
            search_params["status"] = ",".join([s.value if hasattr(s, 'value') else str(s) for s in filters["status"]])
        else:
            search_params["status"] = filters["status"].value if hasattr(filters["status"], 'value') else str(filters["status"])
    
    if filters.get("priority"):
        if isinstance(filters["priority"], list):
            search_params["priority"] = ",".join([p.value if hasattr(p, 'value') else str(p) for p in filters["priority"]])
        else:
            search_params["priority"] = filters["priority"].value if hasattr(filters["priority"], 'value') else str(filters["priority"])
    
    if filters.get("category"):
        if isinstance(filters["category"], list):
            search_params["category"] = ",".join(filters["category"])
        else:
            search_params["category"] = filters["category"]
    
    if filters.get("date_from"):
        search_params["date"] = f"ge{filters['date_from'].isoformat()}"
    
    if filters.get("date_to"):
        existing_date = search_params.get("date", "")
        if existing_date:
            search_params["date"] = f"{existing_date}&date=le{filters['date_to'].isoformat()}"
        else:
            search_params["date"] = f"le{filters['date_to'].isoformat()}"
    
    if filters.get("code_system") and filters.get("code_value"):
        search_params["code"] = f"{filters['code_system']}|{filters['code_value']}"
    
    return search_params

# Error handling helpers
def handle_graphql_error(error: Exception, operation: str) -> str:
    """Handle GraphQL errors and return user-friendly messages"""
    logger.error(f"GraphQL error in {operation}: {error}")
    
    if "not found" in str(error).lower():
        return f"Resource not found"
    elif "permission" in str(error).lower() or "unauthorized" in str(error).lower():
        return f"Insufficient permissions for {operation}"
    elif "validation" in str(error).lower():
        return f"Validation error: {str(error)}"
    else:
        return f"An error occurred during {operation}"
