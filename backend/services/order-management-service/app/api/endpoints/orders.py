from fastapi import APIRouter, Depends, HTTPException, Query
from typing import List, Optional, Dict, Any
import logging
import os
import sys

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import models and services
from app.models.clinical_order import ClinicalOrder, ClinicalOrderCreate, ClinicalOrderUpdate
from app.models.medication_order import MedicationOrder, MedicationOrderCreate, MedicationOrderUpdate
from app.models.lab_order import LabOrder, LabOrderCreate, LabOrderUpdate
from app.models.imaging_order import ImagingOrder, ImagingOrderCreate, ImagingOrderUpdate
from app.services.order_service import get_order_service

# Import authentication
try:
    from shared.auth import get_current_user
except ImportError:
    # Fallback for development
    def get_current_user():
        return {"user_id": "dev-user", "role": "doctor", "name": "Development User"}

logger = logging.getLogger(__name__)

router = APIRouter()

@router.get("/", response_model=List[ClinicalOrder])
async def get_orders(
    patient_id: Optional[str] = Query(None, description="Filter by patient ID"),
    status: Optional[str] = Query(None, description="Filter by order status"),
    category: Optional[str] = Query(None, description="Filter by order category"),
    limit: int = Query(50, description="Maximum number of results"),
    user=Depends(get_current_user)
):
    """Get orders with optional filtering"""
    try:
        order_service = await get_order_service()

        # Build search parameters
        search_params = {}
        if patient_id:
            search_params["subject"] = f"Patient/{patient_id}"
        if status:
            search_params["status"] = status
        if category:
            search_params["category"] = category
        search_params["_count"] = str(limit)

        orders = await order_service.search_clinical_orders(search_params)
        return orders

    except Exception as e:
        logger.error(f"Error getting orders: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/", response_model=ClinicalOrder)
async def create_order(
    order_data: ClinicalOrderCreate,
    user=Depends(get_current_user)
):
    """Create a new clinical order"""
    try:
        order_service = await get_order_service()
        order = await order_service.create_clinical_order(order_data, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error creating order: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error creating order: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/{order_id}", response_model=ClinicalOrder)
async def get_order(
    order_id: str,
    user=Depends(get_current_user)
):
    """Get a specific order by ID"""
    try:
        order_service = await get_order_service()
        order = await order_service.get_clinical_order(order_id)

        if not order:
            raise HTTPException(status_code=404, detail=f"Order {order_id} not found")

        return order

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/{order_id}", response_model=ClinicalOrder)
async def update_order(
    order_id: str,
    order_data: ClinicalOrderUpdate,
    user=Depends(get_current_user)
):
    """Update an order"""
    try:
        order_service = await get_order_service()
        order = await order_service.update_clinical_order(order_id, order_data, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error updating order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error updating order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.delete("/{order_id}")
async def delete_order(
    order_id: str,
    user=Depends(get_current_user)
):
    """Delete an order"""
    try:
        order_service = await get_order_service()
        success = await order_service.delete_clinical_order(order_id, user)

        if success:
            return {"message": f"Order {order_id} deleted successfully"}
        else:
            raise HTTPException(status_code=500, detail="Failed to delete order")

    except Exception as e:
        logger.error(f"Error deleting order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))
