"""
Order Service for Order Management Service

This module provides business logic for order management operations,
including order creation, validation, lifecycle management, and clinical decision support.
"""

import logging
from typing import Dict, List, Optional, Any, Union
from datetime import datetime
import os
import sys

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import event publishing mixin
try:
    from services.shared.event_publishing.event_publisher_mixin import EventPublisherMixin
except ImportError:
    # Fallback if event publishing not available
    class EventPublisherMixin:
        def initialize_event_publisher(self, *args, **kwargs):
            pass
        async def publish_order_event(self, *args, **kwargs):
            return None

# Import models
from app.models.clinical_order import (
    ClinicalOrder, ClinicalOrderCreate, ClinicalOrderUpdate,
    OrderStatus, OrderIntent, OrderPriority, OrderStatusHistory
)
from app.models.medication_order import MedicationOrder, MedicationOrderCreate, MedicationOrderUpdate
from app.models.lab_order import LabOrder, LabOrderCreate, LabOrderUpdate
from app.models.imaging_order import ImagingOrder, ImagingOrderCreate, ImagingOrderUpdate
from app.models.order_set import OrderSet, OrderSetCreate, OrderSetUpdate, OrderSetApplication

from .fhir_service_factory import get_fhir_service

logger = logging.getLogger(__name__)

class OrderService(EventPublisherMixin):
    """
    Service class for order management operations.

    This service provides high-level business logic for:
    - Order creation and validation
    - Order lifecycle management
    - Clinical decision support
    - Order sets management
    - Event publishing for order state changes
    """

    def __init__(self):
        """Initialize the Order Service."""
        super().__init__()
        self.fhir_service = None
        
    async def initialize(self) -> bool:
        """
        Initialize the order service.

        Returns:
            bool: True if initialization successful
        """
        try:
            self.fhir_service = get_fhir_service()
            if self.fhir_service is None:
                logger.error("FHIR service not available")
                return False

            # Initialize event publishing
            self.initialize_event_publisher("order-management-service", enabled=True)

            logger.info("Order service initialized successfully")
            return True
        except Exception as e:
            logger.error(f"Error initializing order service: {e}")
            return False
    
    # Clinical Order Operations
    async def create_clinical_order(self, order_data: ClinicalOrderCreate, user_context: Dict[str, Any]) -> ClinicalOrder:
        """
        Create a new clinical order.
        
        Args:
            order_data: The order creation data
            user_context: User context from authentication
            
        Returns:
            The created clinical order
        """
        try:
            # Convert to FHIR clinical order
            clinical_order = order_data.to_clinical_order()
            
            # Set requester from user context
            if user_context.get("user_id"):
                clinical_order.requester = {
                    "reference": f"Practitioner/{user_context['user_id']}",
                    "display": user_context.get("name", "Unknown Practitioner")
                }
            
            # Validate the order
            await self._validate_order(clinical_order, user_context)
            
            # Create in FHIR store
            fhir_dict = clinical_order.to_fhir_dict()
            created_resource = await self.fhir_service.create_order(fhir_dict, "ServiceRequest")
            
            # Convert back to model
            result = ClinicalOrder.from_fhir_dict(created_resource)
            
            # Create status history entry
            await self._create_status_history(result.id, None, result.status, user_context, "Order created")

            # Publish order created event
            await self.publish_order_event(
                order_id=result.id,
                patient_id=result.subject.get("reference", "").replace("Patient/", "") if result.subject else "unknown",
                order_type=result.code.text if result.code else "unknown",
                operation="created",
                order_data=result.model_dump(),
                status=result.status.value if result.status else None,
                correlation_id=user_context.get("correlation_id")
            )

            logger.info(f"Created clinical order {result.id}")
            return result
            
        except Exception as e:
            logger.error(f"Error creating clinical order: {e}")
            raise
    
    async def get_clinical_order(self, order_id: str) -> Optional[ClinicalOrder]:
        """
        Get a clinical order by ID.
        
        Args:
            order_id: The order ID
            
        Returns:
            The clinical order or None if not found
        """
        try:
            resource = await self.fhir_service.get_order(order_id, "ServiceRequest")
            if resource:
                return ClinicalOrder.from_fhir_dict(resource)
            return None
        except Exception as e:
            logger.error(f"Error getting clinical order {order_id}: {e}")
            raise
    
    async def update_clinical_order(self, order_id: str, order_data: ClinicalOrderUpdate, user_context: Dict[str, Any]) -> ClinicalOrder:
        """
        Update a clinical order.
        
        Args:
            order_id: The order ID
            order_data: The order update data
            user_context: User context from authentication
            
        Returns:
            The updated clinical order
        """
        try:
            # Get existing order
            existing_order = await self.get_clinical_order(order_id)
            if not existing_order:
                raise ValueError(f"Order {order_id} not found")
            
            # Track status change
            old_status = existing_order.status
            
            # Update fields
            update_dict = order_data.model_dump(exclude_unset=True)
            for field, value in update_dict.items():
                setattr(existing_order, field, value)
            
            # Validate the updated order
            await self._validate_order(existing_order, user_context)
            
            # Update in FHIR store
            fhir_dict = existing_order.to_fhir_dict()
            updated_resource = await self.fhir_service.update_order(order_id, fhir_dict, "ServiceRequest")
            
            # Convert back to model
            result = ClinicalOrder.from_fhir_dict(updated_resource)
            
            # Create status history entry if status changed
            if old_status != result.status:
                await self._create_status_history(order_id, old_status, result.status, user_context, "Order updated")

            # Publish order updated event
            await self.publish_order_event(
                order_id=order_id,
                patient_id=result.subject.get("reference", "").replace("Patient/", "") if result.subject else "unknown",
                order_type=result.code.text if result.code else "unknown",
                operation="updated",
                order_data=result.model_dump(),
                status=result.status.value if result.status else None,
                correlation_id=user_context.get("correlation_id")
            )

            logger.info(f"Updated clinical order {order_id}")
            return result
            
        except Exception as e:
            logger.error(f"Error updating clinical order {order_id}: {e}")
            raise
    
    async def delete_clinical_order(self, order_id: str, user_context: Dict[str, Any]) -> bool:
        """
        Delete a clinical order.
        
        Args:
            order_id: The order ID
            user_context: User context from authentication
            
        Returns:
            True if deletion successful
        """
        try:
            # Get existing order for status history
            existing_order = await self.get_clinical_order(order_id)
            if existing_order:
                await self._create_status_history(order_id, existing_order.status, OrderStatus.ENTERED_IN_ERROR, user_context, "Order deleted")
            
            success = await self.fhir_service.delete_order(order_id, "ServiceRequest")
            if success:
                logger.info(f"Deleted clinical order {order_id}")
            return success
        except Exception as e:
            logger.error(f"Error deleting clinical order {order_id}: {e}")
            raise
    
    async def search_clinical_orders(self, search_params: Dict[str, Any]) -> List[ClinicalOrder]:
        """
        Search for clinical orders.
        
        Args:
            search_params: FHIR search parameters
            
        Returns:
            List of matching clinical orders
        """
        try:
            resources = await self.fhir_service.search_orders(search_params, "ServiceRequest")
            return [ClinicalOrder.from_fhir_dict(resource) for resource in resources]
        except Exception as e:
            logger.error(f"Error searching clinical orders: {e}")
            raise
    
    # Order Lifecycle Management
    async def sign_order(self, order_id: str, user_context: Dict[str, Any]) -> ClinicalOrder:
        """
        Sign an order (change status from draft to active).
        
        Args:
            order_id: The order ID
            user_context: User context from authentication
            
        Returns:
            The signed order
        """
        try:
            order_update = ClinicalOrderUpdate(status=OrderStatus.ACTIVE)
            result = await self.update_clinical_order(order_id, order_update, user_context)
            
            # Create specific status history for signing
            await self._create_status_history(order_id, OrderStatus.DRAFT, OrderStatus.ACTIVE, user_context, "Order signed")

            # Publish order signed event
            await self.publish_order_event(
                order_id=order_id,
                patient_id=result.subject.get("reference", "").replace("Patient/", "") if result.subject else "unknown",
                order_type=result.code.text if result.code else "unknown",
                operation="signed",
                order_data=result.model_dump(),
                status=result.status.value if result.status else None,
                correlation_id=user_context.get("correlation_id")
            )

            logger.info(f"Signed order {order_id}")
            return result
        except Exception as e:
            logger.error(f"Error signing order {order_id}: {e}")
            raise
    
    async def cancel_order(self, order_id: str, reason: str, user_context: Dict[str, Any]) -> ClinicalOrder:
        """
        Cancel an order.
        
        Args:
            order_id: The order ID
            reason: Reason for cancellation
            user_context: User context from authentication
            
        Returns:
            The cancelled order
        """
        try:
            order_update = ClinicalOrderUpdate(status=OrderStatus.REVOKED)
            result = await self.update_clinical_order(order_id, order_update, user_context)
            
            # Create specific status history for cancellation
            await self._create_status_history(order_id, result.status, OrderStatus.REVOKED, user_context, f"Order cancelled: {reason}")
            
            logger.info(f"Cancelled order {order_id}: {reason}")
            return result
        except Exception as e:
            logger.error(f"Error cancelling order {order_id}: {e}")
            raise
    
    async def hold_order(self, order_id: str, reason: str, user_context: Dict[str, Any]) -> ClinicalOrder:
        """
        Put an order on hold.
        
        Args:
            order_id: The order ID
            reason: Reason for hold
            user_context: User context from authentication
            
        Returns:
            The held order
        """
        try:
            order_update = ClinicalOrderUpdate(status=OrderStatus.ON_HOLD)
            result = await self.update_clinical_order(order_id, order_update, user_context)
            
            # Create specific status history for hold
            await self._create_status_history(order_id, result.status, OrderStatus.ON_HOLD, user_context, f"Order held: {reason}")
            
            logger.info(f"Put order {order_id} on hold: {reason}")
            return result
        except Exception as e:
            logger.error(f"Error putting order {order_id} on hold: {e}")
            raise
    
    async def release_order(self, order_id: str, user_context: Dict[str, Any]) -> ClinicalOrder:
        """
        Release an order from hold.
        
        Args:
            order_id: The order ID
            user_context: User context from authentication
            
        Returns:
            The released order
        """
        try:
            order_update = ClinicalOrderUpdate(status=OrderStatus.ACTIVE)
            result = await self.update_clinical_order(order_id, order_update, user_context)
            
            # Create specific status history for release
            await self._create_status_history(order_id, OrderStatus.ON_HOLD, OrderStatus.ACTIVE, user_context, "Order released from hold")
            
            logger.info(f"Released order {order_id} from hold")
            return result
        except Exception as e:
            logger.error(f"Error releasing order {order_id} from hold: {e}")
            raise
    
    # Private helper methods
    async def _validate_order(self, order: ClinicalOrder, user_context: Dict[str, Any]) -> None:
        """
        Validate an order before creation or update.
        
        Args:
            order: The order to validate
            user_context: User context from authentication
        """
        try:
            # Basic validation
            if not order.code:
                raise ValueError("Order must have a code")
            if not order.subject:
                raise ValueError("Order must have a subject (patient)")
            
            # Check user permissions
            user_role = user_context.get("role", "")
            if user_role not in ["doctor", "nurse", "pharmacist"]:
                raise ValueError("User does not have permission to create orders")
            
            # TODO: Add clinical decision support validation
            # - Drug interactions
            # - Allergies
            # - Contraindications
            # - Duplicate therapy
            
            logger.debug(f"Order validation passed for user {user_context.get('user_id')}")
            
        except Exception as e:
            logger.error(f"Order validation failed: {e}")
            raise
    
    async def _create_status_history(self, order_id: str, old_status: Optional[OrderStatus], new_status: OrderStatus, user_context: Dict[str, Any], reason: str) -> None:
        """
        Create a status history entry for audit trail.
        
        Args:
            order_id: The order ID
            old_status: Previous status
            new_status: New status
            user_context: User context
            reason: Reason for change
        """
        try:
            history_entry = OrderStatusHistory(
                order_id=order_id,
                previous_status=old_status,
                new_status=new_status,
                change_datetime=datetime.utcnow(),
                changed_by={
                    "reference": f"Practitioner/{user_context.get('user_id', 'unknown')}",
                    "display": user_context.get("name", "Unknown User")
                },
                reason_for_change=reason
            )
            
            # TODO: Store status history in a separate collection or as FHIR AuditEvent
            logger.info(f"Status history created for order {order_id}: {old_status} -> {new_status}")
            
        except Exception as e:
            logger.error(f"Error creating status history: {e}")
            # Don't raise - this is for audit purposes only

# Global service instance
_order_service = None

async def get_order_service() -> OrderService:
    """
    Get the global order service instance.
    
    Returns:
        OrderService instance
    """
    global _order_service
    
    if _order_service is None:
        _order_service = OrderService()
        await _order_service.initialize()
    
    return _order_service
