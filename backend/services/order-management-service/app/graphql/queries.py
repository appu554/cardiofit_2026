"""
GraphQL Queries for Order Management Service

This module defines the GraphQL queries for the Order Management Service.
"""

import strawberry
from typing import List, Optional
import logging

from .types import (
    ClinicalOrder, MedicationOrder, LabOrder, ImagingOrder, OrderSet,
    OrderConnection, OrderSearchResult, OrderStatistics,
    OrderStatusEnum, OrderPriorityEnum
)
from .inputs import OrderSearchFilters, PaginationInput, OrderSortInput
from app.services.order_service import get_order_service

logger = logging.getLogger(__name__)

@strawberry.type
class Query:
    """Root query type for the Order Management Service."""
    
    @strawberry.field
    async def order(self, id: strawberry.ID) -> Optional[ClinicalOrder]:
        """Get a specific order by ID"""
        try:
            order_service = await get_order_service()
            order = await order_service.get_clinical_order(str(id))
            
            if not order:
                return None
                
            from .resolvers import convert_order_to_graphql
            return convert_order_to_graphql(order)
        except Exception as e:
            logger.error(f"Error getting order {id}: {e}")
            return None
    
    @strawberry.field
    async def orders(
        self,
        filters: Optional[OrderSearchFilters] = None,
        pagination: Optional[PaginationInput] = None,
        sort: Optional[OrderSortInput] = None
    ) -> List[ClinicalOrder]:
        """Get orders with optional filtering, pagination, and sorting"""
        try:
            order_service = await get_order_service()
            
            # Build search parameters
            search_params = {}
            
            if filters:
                if filters.patient_id:
                    search_params["subject"] = f"Patient/{filters.patient_id}"
                if filters.practitioner_id:
                    search_params["requester"] = f"Practitioner/{filters.practitioner_id}"
                if filters.encounter_id:
                    search_params["encounter"] = f"Encounter/{filters.encounter_id}"
                if filters.status:
                    search_params["status"] = ",".join([s.value for s in filters.status])
                if filters.priority:
                    search_params["priority"] = ",".join([p.value for p in filters.priority])
                if filters.category:
                    search_params["category"] = ",".join(filters.category)
                if filters.date_from:
                    search_params["date"] = f"ge{filters.date_from.isoformat()}"
                if filters.date_to:
                    search_params["date"] = f"le{filters.date_to.isoformat()}"
                if filters.code_system and filters.code_value:
                    search_params["code"] = f"{filters.code_system}|{filters.code_value}"
            
            # Add pagination
            if pagination:
                search_params["_count"] = str(pagination.limit)
                if pagination.offset:
                    search_params["_offset"] = str(pagination.offset)
            else:
                search_params["_count"] = "50"  # Default limit
            
            # Add sorting
            if sort:
                search_params["_sort"] = f"{'-' if sort.direction == 'desc' else ''}{sort.field}"
            
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting orders: {e}")
            return []
    
    @strawberry.field
    async def orders_by_patient(self, patient_id: str) -> List[ClinicalOrder]:
        """Get all orders for a specific patient"""
        try:
            order_service = await get_order_service()
            search_params = {"subject": f"Patient/{patient_id}"}
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting orders for patient {patient_id}: {e}")
            return []
    
    @strawberry.field
    async def orders_by_practitioner(
        self,
        practitioner_id: str,
        status: Optional[List[OrderStatusEnum]] = None
    ) -> List[ClinicalOrder]:
        """Get orders by practitioner"""
        try:
            order_service = await get_order_service()
            
            search_params = {"requester": f"Practitioner/{practitioner_id}"}
            if status:
                search_params["status"] = ",".join([s.value for s in status])
            
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting orders for practitioner {practitioner_id}: {e}")
            return []
    
    @strawberry.field
    async def orders_by_encounter(self, encounter_id: str) -> List[ClinicalOrder]:
        """Get all orders for a specific encounter"""
        try:
            order_service = await get_order_service()
            search_params = {"encounter": f"Encounter/{encounter_id}"}
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting orders for encounter {encounter_id}: {e}")
            return []
    
    @strawberry.field
    async def pending_orders(self, practitioner_id: str) -> List[ClinicalOrder]:
        """Get pending orders for a practitioner"""
        try:
            order_service = await get_order_service()
            
            search_params = {
                "requester": f"Practitioner/{practitioner_id}",
                "status": "draft"
            }
            
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting pending orders for practitioner {practitioner_id}: {e}")
            return []
    
    @strawberry.field
    async def active_orders(self, patient_id: Optional[str] = None) -> List[ClinicalOrder]:
        """Get active orders, optionally filtered by patient"""
        try:
            order_service = await get_order_service()
            
            search_params = {"status": "active"}
            if patient_id:
                search_params["subject"] = f"Patient/{patient_id}"
            
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting active orders: {e}")
            return []
    
    @strawberry.field
    async def search_orders(
        self,
        query: str,
        filters: Optional[OrderSearchFilters] = None
    ) -> OrderSearchResult:
        """Search orders with text query and filters"""
        try:
            order_service = await get_order_service()
            
            # Build search parameters
            search_params = {"_text": query}
            
            if filters:
                if filters.patient_id:
                    search_params["subject"] = f"Patient/{filters.patient_id}"
                if filters.status:
                    search_params["status"] = ",".join([s.value for s in filters.status])
                if filters.category:
                    search_params["category"] = ",".join(filters.category)
            
            orders = await order_service.search_clinical_orders(search_params)
            
            from .resolvers import convert_order_to_graphql
            return OrderSearchResult(
                orders=[convert_order_to_graphql(order) for order in orders],
                total_count=len(orders),
                search_params=search_params
            )
        except Exception as e:
            logger.error(f"Error searching orders: {e}")
            return OrderSearchResult(orders=[], total_count=0, search_params={})
    
    @strawberry.field
    async def order_statistics(
        self,
        patient_id: Optional[str] = None,
        practitioner_id: Optional[str] = None
    ) -> OrderStatistics:
        """Get order statistics for dashboards"""
        try:
            order_service = await get_order_service()
            
            # Base search parameters
            base_params = {}
            if patient_id:
                base_params["subject"] = f"Patient/{patient_id}"
            if practitioner_id:
                base_params["requester"] = f"Practitioner/{practitioner_id}"
            
            # Get orders by status
            statuses = ["draft", "active", "completed", "cancelled"]
            status_counts = {}
            total_orders = 0
            
            for status in statuses:
                search_params = {**base_params, "status": status}
                orders = await order_service.search_clinical_orders(search_params)
                count = len(orders)
                status_counts[status] = count
                total_orders += count
            
            # Get orders by priority
            priorities = ["routine", "urgent", "asap", "stat"]
            priority_counts = {}
            
            for priority in priorities:
                search_params = {**base_params, "priority": priority}
                orders = await order_service.search_clinical_orders(search_params)
                priority_counts[priority] = len(orders)
            
            # Get orders by category (simplified)
            categories = ["laboratory", "imaging", "medication", "procedure"]
            category_counts = {}
            
            for category in categories:
                search_params = {**base_params, "category": category}
                orders = await order_service.search_clinical_orders(search_params)
                category_counts[category] = len(orders)
            
            return OrderStatistics(
                total_orders=total_orders,
                draft_orders=status_counts.get("draft", 0),
                active_orders=status_counts.get("active", 0),
                completed_orders=status_counts.get("completed", 0),
                cancelled_orders=status_counts.get("cancelled", 0),
                orders_by_priority=priority_counts,
                orders_by_category=category_counts
            )
        except Exception as e:
            logger.error(f"Error getting order statistics: {e}")
            return OrderStatistics(
                total_orders=0,
                draft_orders=0,
                active_orders=0,
                completed_orders=0,
                cancelled_orders=0,
                on_hold_orders=0,
                orders_by_priority={},
                orders_by_category={},
                orders_by_requester={},
                orders_by_date_range={},
                most_common_orders=[]
            )

    # ========================================
    # ORDER SETS & PROTOCOLS
    # ========================================

    @strawberry.field
    async def order_set(self, id: strawberry.ID) -> Optional[OrderSet]:
        """Get a specific order set by ID"""
        try:
            # TODO: Implement order set retrieval
            # For now, return sample order sets
            if str(id) == "hypertension-protocol":
                return OrderSet(
                    id=id,
                    status="active",
                    intent="plan",
                    name="Hypertension Management Protocol",
                    title="Standard Hypertension Treatment Protocol",
                    description="Comprehensive order set for hypertension management including medications, monitoring, and lifestyle interventions",
                    code={"coding": [{"system": "http://snomed.info/sct", "code": "38341003", "display": "Hypertensive disorder"}]},
                    category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/plan-definition-type", "code": "clinical-protocol"}]}],
                    applicable_context=[{"coding": [{"system": "http://snomed.info/sct", "code": "38341003", "display": "Hypertension"}]}]
                )
            return None
        except Exception as e:
            logger.error(f"Error getting order set {id}: {e}")
            return None

    @strawberry.field
    async def order_sets(
        self,
        category: Optional[List[str]] = None,
        context: Optional[List[str]] = None
    ) -> List[OrderSet]:
        """Get available order sets with optional filtering"""
        try:
            # TODO: Implement order set search
            # For now, return sample order sets
            order_sets = [
                OrderSet(
                    id="hypertension-protocol",
                    status="active",
                    intent="plan",
                    name="Hypertension Management Protocol",
                    title="Standard Hypertension Treatment Protocol",
                    description="Comprehensive order set for hypertension management",
                    category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/plan-definition-type", "code": "clinical-protocol"}]}]
                ),
                OrderSet(
                    id="diabetes-protocol",
                    status="active",
                    intent="plan",
                    name="Diabetes Management Protocol",
                    title="Standard Diabetes Treatment Protocol",
                    description="Comprehensive order set for diabetes management",
                    category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/plan-definition-type", "code": "clinical-protocol"}]}]
                ),
                OrderSet(
                    id="pneumonia-protocol",
                    status="active",
                    intent="plan",
                    name="Community Acquired Pneumonia Protocol",
                    title="CAP Treatment Protocol",
                    description="Evidence-based order set for community acquired pneumonia",
                    category=[{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/plan-definition-type", "code": "clinical-protocol"}]}]
                )
            ]

            # Apply filters
            if category:
                # Filter by category (simplified)
                filtered_sets = []
                for order_set in order_sets:
                    if any(cat in str(order_set.category).lower() for cat in category):
                        filtered_sets.append(order_set)
                return filtered_sets

            return order_sets
        except Exception as e:
            logger.error(f"Error getting order sets: {e}")
            return []

    @strawberry.field
    async def order_sets_by_condition(self, condition_code: str) -> List[OrderSet]:
        """Get order sets applicable to a specific condition"""
        try:
            # TODO: Implement condition-based order set retrieval
            # For now, return condition-specific order sets
            condition_sets = {
                "hypertension": [
                    OrderSet(
                        id="hypertension-protocol",
                        status="active",
                        intent="plan",
                        name="Hypertension Management Protocol",
                        description="Comprehensive hypertension management"
                    )
                ],
                "diabetes": [
                    OrderSet(
                        id="diabetes-protocol",
                        status="active",
                        intent="plan",
                        name="Diabetes Management Protocol",
                        description="Comprehensive diabetes management"
                    )
                ],
                "pneumonia": [
                    OrderSet(
                        id="pneumonia-protocol",
                        status="active",
                        intent="plan",
                        name="Community Acquired Pneumonia Protocol",
                        description="Evidence-based pneumonia treatment"
                    )
                ]
            }

            return condition_sets.get(condition_code.lower(), [])
        except Exception as e:
            logger.error(f"Error getting order sets for condition {condition_code}: {e}")
            return []

    # ========================================
    # ORDER HISTORY & AUDIT TRAIL
    # ========================================

    @strawberry.field
    async def order_history(self, order_id: str) -> List[OrderStatusHistory]:
        """Get complete history of an order including all status changes"""
        try:
            # TODO: Implement order history retrieval
            # For now, return sample history
            from datetime import datetime, timedelta

            history = [
                OrderStatusHistory(
                    id=f"history-{order_id}-1",
                    order_id=order_id,
                    previous_status=None,
                    new_status="draft",
                    change_datetime=datetime.now() - timedelta(hours=2),
                    changed_by={"reference": "Practitioner/dr-smith", "display": "Dr. Smith"},
                    reason_for_change="Order created",
                    change_type="creation"
                ),
                OrderStatusHistory(
                    id=f"history-{order_id}-2",
                    order_id=order_id,
                    previous_status="draft",
                    new_status="active",
                    change_datetime=datetime.now() - timedelta(hours=1),
                    changed_by={"reference": "Practitioner/dr-smith", "display": "Dr. Smith"},
                    reason_for_change="Order signed and activated",
                    change_type="status_change"
                )
            ]

            return history
        except Exception as e:
            logger.error(f"Error getting order history for {order_id}: {e}")
            return []

    @strawberry.field
    async def patient_order_timeline(self, patient_id: str) -> List[ClinicalOrder]:
        """Get chronological timeline of all orders for a patient"""
        try:
            order_service = await get_order_service()
            search_params = {
                "subject": f"Patient/{patient_id}",
                "_sort": "-authored_on"  # Sort by date descending
            }
            orders = await order_service.search_clinical_orders(search_params)

            from .resolvers import convert_order_to_graphql
            return [convert_order_to_graphql(order) for order in orders]
        except Exception as e:
            logger.error(f"Error getting order timeline for patient {patient_id}: {e}")
            return []
