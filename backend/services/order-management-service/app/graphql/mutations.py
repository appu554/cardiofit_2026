"""
GraphQL Mutations for Order Management Service

This module defines the GraphQL mutations for the Order Management Service.
"""

import strawberry
from typing import List, Optional
import logging

from .types import ClinicalOrder, MedicationOrder, LabOrder, ImagingOrder, OrderSet
from .inputs import (
    ClinicalOrderInput, ClinicalOrderUpdateInput, MedicationOrderInput,
    LabOrderInput, ImagingOrderInput, OrderSetInput, OrderSetUpdateInput,
    OrderSetApplicationInput, OrderActionInput, SignatureInput,
    BulkOrderActionInput, OrderBatchInput
)
from app.services.order_service import get_order_service

# Import authentication
try:
    from shared.auth import get_current_user
except ImportError:
    def get_current_user():
        return {"user_id": "dev-user", "role": "doctor", "name": "Development User"}

logger = logging.getLogger(__name__)

@strawberry.type
class Mutation:
    """Root mutation type for the Order Management Service."""
    
    # Clinical Order Mutations
    @strawberry.field
    async def create_order(self, order_data: ClinicalOrderInput) -> ClinicalOrder:
        """Create a new clinical order"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            from .resolvers import convert_graphql_input_to_model, convert_order_to_graphql
            order_create = convert_graphql_input_to_model(order_data)
            order = await order_service.create_clinical_order(order_create, user_context)
            
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error creating order: {e}")
            raise Exception(f"Failed to create order: {str(e)}")
    
    @strawberry.field
    async def update_order(
        self, 
        id: strawberry.ID, 
        order_data: ClinicalOrderUpdateInput
    ) -> ClinicalOrder:
        """Update an existing order"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            from app.models.clinical_order import ClinicalOrderUpdate
            from .resolvers import convert_order_to_graphql
            
            # Convert GraphQL input to model
            update_data = ClinicalOrderUpdate(
                status=order_data.status.value if order_data.status else None,
                priority=order_data.priority.value if order_data.priority else None,
                occurrence_datetime=order_data.occurrence_datetime,
                performer=[{
                    "reference": perf.reference,
                    "display": perf.display
                } for perf in (order_data.performer or [])],
                note=[{
                    "text": note.text,
                    "author_string": note.author_string
                } for note in (order_data.note or [])],
                patient_instruction=order_data.patient_instruction
            )
            
            order = await order_service.update_clinical_order(str(id), update_data, user_context)
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error updating order {id}: {e}")
            raise Exception(f"Failed to update order: {str(e)}")
    
    @strawberry.field
    async def delete_order(self, id: strawberry.ID) -> bool:
        """Delete an order"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            success = await order_service.delete_clinical_order(str(id), user_context)
            return success
        except Exception as e:
            logger.error(f"Error deleting order {id}: {e}")
            raise Exception(f"Failed to delete order: {str(e)}")
    
    # Order Lifecycle Management
    @strawberry.field
    async def sign_order(
        self, 
        id: strawberry.ID,
        signature_data: Optional[SignatureInput] = None
    ) -> ClinicalOrder:
        """Sign an order (activate from draft status)"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            order = await order_service.sign_order(str(id), user_context)
            
            from .resolvers import convert_order_to_graphql
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error signing order {id}: {e}")
            raise Exception(f"Failed to sign order: {str(e)}")
    
    @strawberry.field
    async def cancel_order(
        self, 
        id: strawberry.ID, 
        action_data: OrderActionInput
    ) -> ClinicalOrder:
        """Cancel an order"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            reason = action_data.reason or "Order cancelled"
            order = await order_service.cancel_order(str(id), reason, user_context)
            
            from .resolvers import convert_order_to_graphql
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error cancelling order {id}: {e}")
            raise Exception(f"Failed to cancel order: {str(e)}")
    
    @strawberry.field
    async def hold_order(
        self, 
        id: strawberry.ID, 
        action_data: OrderActionInput
    ) -> ClinicalOrder:
        """Put an order on hold"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            reason = action_data.reason or "Order placed on hold"
            order = await order_service.hold_order(str(id), reason, user_context)
            
            from .resolvers import convert_order_to_graphql
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error holding order {id}: {e}")
            raise Exception(f"Failed to hold order: {str(e)}")
    
    @strawberry.field
    async def release_order(self, id: strawberry.ID) -> ClinicalOrder:
        """Release an order from hold"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            order = await order_service.release_order(str(id), user_context)
            
            from .resolvers import convert_order_to_graphql
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error releasing order {id}: {e}")
            raise Exception(f"Failed to release order: {str(e)}")
    
    @strawberry.field
    async def complete_order(self, id: strawberry.ID) -> ClinicalOrder:
        """Mark an order as completed"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            from app.models.clinical_order import ClinicalOrderUpdate, OrderStatus
            order_update = ClinicalOrderUpdate(status=OrderStatus.COMPLETED)
            order = await order_service.update_clinical_order(str(id), order_update, user_context)
            
            from .resolvers import convert_order_to_graphql
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error completing order {id}: {e}")
            raise Exception(f"Failed to complete order: {str(e)}")
    
    # Specialized Order Types
    @strawberry.field
    async def create_medication_order(self, order_data: MedicationOrderInput) -> MedicationOrder:
        """Create a new medication order"""
        try:
            # TODO: Implement medication order creation
            # For now, create as a clinical order with medication category
            clinical_order_data = ClinicalOrderInput(
                code=order_data.medication_codeable_concept or {"text": "Medication Order"},
                subject=order_data.subject,
                encounter=order_data.encounter,
                requester=order_data.requester,
                category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "medication"}]}],
                note=order_data.note
            )
            
            order_service = await get_order_service()
            user_context = get_current_user()
            
            from .resolvers import convert_graphql_input_to_model
            order_create = convert_graphql_input_to_model(clinical_order_data)
            order = await order_service.create_clinical_order(order_create, user_context)
            
            # Convert to MedicationOrder type
            return MedicationOrder(
                id=order.id,
                status=order_data.status,
                intent=order_data.intent,
                medication_codeable_concept=order_data.medication_codeable_concept,
                subject=order_data.subject,
                encounter=order_data.encounter,
                authored_on=order.authored_on,
                requester=order_data.requester,
                dosage_instruction=order_data.dosage_instruction,
                dispense_request=order_data.dispense_request
            )
        except Exception as e:
            logger.error(f"Error creating medication order: {e}")
            raise Exception(f"Failed to create medication order: {str(e)}")
    
    @strawberry.field
    async def create_lab_order(self, order_data: LabOrderInput) -> LabOrder:
        """Create a new laboratory order"""
        try:
            # Convert to clinical order with lab category
            clinical_order_data = ClinicalOrderInput(
                status=order_data.status,
                intent=order_data.intent,
                priority=order_data.priority,
                code=order_data.code,
                subject=order_data.subject,
                encounter=order_data.encounter,
                requester=order_data.requester,
                category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
                reason_code=order_data.reason_code,
                note=order_data.note
            )
            
            order_service = await get_order_service()
            user_context = get_current_user()
            
            from .resolvers import convert_graphql_input_to_model
            order_create = convert_graphql_input_to_model(clinical_order_data)
            order = await order_service.create_clinical_order(order_create, user_context)
            
            # Convert to LabOrder type
            return LabOrder(
                id=order.id,
                status=order_data.status,
                intent=order_data.intent,
                code=order_data.code,
                subject=order_data.subject,
                test_name=order_data.test_name,
                specimen_source=order_data.specimen_source,
                fasting_required=order_data.fasting_required,
                collection_datetime_preference=order_data.collection_datetime_preference,
                clinical_history=order_data.clinical_history
            )
        except Exception as e:
            logger.error(f"Error creating lab order: {e}")
            raise Exception(f"Failed to create lab order: {str(e)}")
    
    @strawberry.field
    async def create_imaging_order(self, order_data: ImagingOrderInput) -> ImagingOrder:
        """Create a new imaging order"""
        try:
            # Convert to clinical order with imaging category
            clinical_order_data = ClinicalOrderInput(
                status=order_data.status,
                intent=order_data.intent,
                priority=order_data.priority,
                code=order_data.code,
                subject=order_data.subject,
                encounter=order_data.encounter,
                requester=order_data.requester,
                category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "imaging"}]}],
                reason_code=order_data.reason_code,
                note=order_data.note
            )
            
            order_service = await get_order_service()
            user_context = get_current_user()
            
            from .resolvers import convert_graphql_input_to_model
            order_create = convert_graphql_input_to_model(clinical_order_data)
            order = await order_service.create_clinical_order(order_create, user_context)
            
            # Convert to ImagingOrder type
            return ImagingOrder(
                id=order.id,
                status=order_data.status,
                intent=order_data.intent,
                code=order_data.code,
                subject=order_data.subject,
                procedure_name=order_data.procedure_name,
                modality=order_data.modality,
                body_site=order_data.body_site,
                contrast_required=order_data.contrast_required,
                clinical_history_for_radiologist=order_data.clinical_history_for_radiologist
            )
        except Exception as e:
            logger.error(f"Error creating imaging order: {e}")
            raise Exception(f"Failed to create imaging order: {str(e)}")
    
    # Bulk Operations
    @strawberry.field
    async def bulk_order_action(self, action_data: BulkOrderActionInput) -> List[ClinicalOrder]:
        """Perform bulk actions on multiple orders"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            results = []
            for order_id in action_data.order_ids:
                try:
                    if action_data.action == "sign":
                        order = await order_service.sign_order(order_id, user_context)
                    elif action_data.action == "cancel":
                        reason = action_data.reason or "Bulk cancellation"
                        order = await order_service.cancel_order(order_id, reason, user_context)
                    elif action_data.action == "hold":
                        reason = action_data.reason or "Bulk hold"
                        order = await order_service.hold_order(order_id, reason, user_context)
                    elif action_data.action == "release":
                        order = await order_service.release_order(order_id, user_context)
                    else:
                        continue
                    
                    from .resolvers import convert_order_to_graphql
                    results.append(convert_order_to_graphql(order))
                except Exception as e:
                    logger.error(f"Error performing bulk action on order {order_id}: {e}")
                    continue
            
            return results
        except Exception as e:
            logger.error(f"Error performing bulk order action: {e}")
            raise Exception(f"Failed to perform bulk action: {str(e)}")
    
    @strawberry.field
    async def create_order_batch(self, batch_data: OrderBatchInput) -> List[ClinicalOrder]:
        """Create multiple orders at once"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()
            
            results = []
            for order_data in batch_data.orders:
                try:
                    # Set common fields if provided
                    if batch_data.encounter_id and not order_data.encounter:
                        order_data.encounter = {"reference": f"Encounter/{batch_data.encounter_id}"}
                    if batch_data.requester_id and not order_data.requester:
                        order_data.requester = {"reference": f"Practitioner/{batch_data.requester_id}"}
                    
                    from .resolvers import convert_graphql_input_to_model, convert_order_to_graphql
                    order_create = convert_graphql_input_to_model(order_data)
                    order = await order_service.create_clinical_order(order_create, user_context)
                    
                    results.append(convert_order_to_graphql(order))
                except Exception as e:
                    logger.error(f"Error creating order in batch: {e}")
                    continue
            
            return results
        except Exception as e:
            logger.error(f"Error creating order batch: {e}")
            raise Exception(f"Failed to create order batch: {str(e)}")

    # ========================================
    # ORDER SETS & PROTOCOLS
    # ========================================

    @strawberry.field
    async def create_order_set(self, order_set_data: OrderSetInput) -> OrderSet:
        """Create a new order set for standardized clinical protocols"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()

            # TODO: Implement order set creation
            # For now, return a placeholder
            return OrderSet(
                id=f"orderset-{order_set_data.name}",
                status=order_set_data.status,
                intent=order_set_data.intent,
                name=order_set_data.name,
                title=order_set_data.title,
                description=order_set_data.description,
                code=order_set_data.code
            )
        except Exception as e:
            logger.error(f"Error creating order set: {e}")
            raise Exception(f"Failed to create order set: {str(e)}")

    @strawberry.field
    async def apply_order_set(self, application_data: OrderSetApplicationInput) -> List[ClinicalOrder]:
        """Apply an order set to a patient with customizations"""
        try:
            order_service = await get_order_service()
            user_context = get_current_user()

            # TODO: Implement order set application
            # For now, create sample orders based on order set
            results = []

            # Example: Hypertension Protocol
            if "hypertension" in application_data.order_set_id.lower():
                # Create medication order
                med_order_data = ClinicalOrderInput(
                    code={"text": "Lisinopril 10mg daily"},
                    subject={"reference": f"Patient/{application_data.patient_id}"},
                    encounter={"reference": f"Encounter/{application_data.encounter_id}"} if application_data.encounter_id else None,
                    requester={"reference": f"Practitioner/{application_data.requester_id}"} if application_data.requester_id else None,
                    category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "medication"}]}],
                    note=[{"text": "Part of hypertension protocol"}] if application_data.note else None
                )

                from .resolvers import convert_graphql_input_to_model, convert_order_to_graphql
                order_create = convert_graphql_input_to_model(med_order_data)
                order = await order_service.create_clinical_order(order_create, user_context)
                results.append(convert_order_to_graphql(order))

            return results
        except Exception as e:
            logger.error(f"Error applying order set: {e}")
            raise Exception(f"Failed to apply order set: {str(e)}")

    # ========================================
    # CLINICAL DECISION SUPPORT
    # ========================================

    @strawberry.field
    async def check_drug_interactions(self, cds_data: CDSCheckInput) -> List[DrugInteractionAlert]:
        """Check for drug interactions and clinical alerts"""
        try:
            # TODO: Implement actual CDS checking
            # For now, return sample alerts
            alerts = []

            # Sample drug interaction alert
            if cds_data.order_data.code and "warfarin" in str(cds_data.order_data.code).lower():
                alerts.append(DrugInteractionAlert(
                    id="alert-warfarin-1",
                    severity="high",
                    interaction_type="drug-drug",
                    description="Warfarin may interact with other medications. Check INR levels.",
                    recommendation="Monitor INR closely and adjust dose as needed.",
                    source="Clinical Decision Support System",
                    evidence_level="high",
                    affected_medications=[{"reference": "Medication/warfarin"}],
                    clinical_significance="Major interaction - requires monitoring"
                ))

            return alerts
        except Exception as e:
            logger.error(f"Error checking drug interactions: {e}")
            raise Exception(f"Failed to check drug interactions: {str(e)}")

    @strawberry.field
    async def validate_order_appropriateness(self, cds_data: CDSCheckInput) -> List[ClinicalAlert]:
        """Validate order appropriateness based on clinical guidelines"""
        try:
            # TODO: Implement actual appropriateness checking
            # For now, return sample alerts
            alerts = []

            # Sample appropriateness alert
            if cds_data.order_data.code and "mri" in str(cds_data.order_data.code).lower():
                alerts.append(ClinicalAlert(
                    id="alert-mri-1",
                    alert_type="appropriateness",
                    severity="warning",
                    title="MRI Appropriateness Check",
                    description="Consider if MRI is appropriate based on clinical guidelines.",
                    recommendation="Review clinical indication and consider alternative imaging if appropriate.",
                    source="Appropriateness Criteria",
                    triggered_by={"reference": f"Patient/{cds_data.patient_id}"}
                ))

            return alerts
        except Exception as e:
            logger.error(f"Error validating order appropriateness: {e}")
            raise Exception(f"Failed to validate order appropriateness: {str(e)}")

    @strawberry.field
    async def check_formulary_status(self, medication_reference: str) -> FormularyStatus:
        """Check medication formulary status and alternatives"""
        try:
            # TODO: Implement actual formulary checking
            # For now, return sample status
            return FormularyStatus(
                medication={"reference": medication_reference},
                formulary_status="preferred",
                tier=1,
                prior_authorization_required=False,
                step_therapy_required=False,
                quantity_limits="30-day supply",
                alternatives=[
                    {"reference": "Medication/alternative-1"},
                    {"reference": "Medication/alternative-2"}
                ],
                cost_estimate={"value": 25.00, "unit": "USD"}
            )
        except Exception as e:
            logger.error(f"Error checking formulary status: {e}")
            raise Exception(f"Failed to check formulary status: {str(e)}")
