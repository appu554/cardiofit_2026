from fastapi import APIRouter, Depends, HTTPException, Body
from typing import List, Optional, Dict, Any
from pydantic import BaseModel
import logging
import os
import sys

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import models and services
from app.models.clinical_order import ClinicalOrder
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

# Request models for order management operations
class OrderActionRequest(BaseModel):
    reason: Optional[str] = None
    note: Optional[str] = None

class SignatureRequest(BaseModel):
    signature_type: str = "electronic"
    signature_data: Optional[str] = None
    reason: Optional[str] = None

@router.post("/sign/{order_id}", response_model=ClinicalOrder)
async def sign_order(
    order_id: str,
    signature_data: SignatureRequest = Body(...),
    user=Depends(get_current_user)
):
    """Sign an order (activate from draft status)"""
    try:
        order_service = await get_order_service()

        # Check user permissions for signing
        if user.get("role") not in ["doctor", "nurse"]:
            raise HTTPException(status_code=403, detail="User does not have permission to sign orders")

        order = await order_service.sign_order(order_id, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error signing order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error signing order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/cosign/{order_id}", response_model=ClinicalOrder)
async def cosign_order(
    order_id: str,
    signature_data: SignatureRequest = Body(...),
    user=Depends(get_current_user)
):
    """Co-sign an order (for resident/attending workflows)"""
    try:
        order_service = await get_order_service()

        # Check user permissions for co-signing
        if user.get("role") != "doctor":
            raise HTTPException(status_code=403, detail="Only doctors can co-sign orders")

        # TODO: Implement co-signing logic
        # For now, treat as regular signing
        order = await order_service.sign_order(order_id, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error co-signing order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error co-signing order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/cancel/{order_id}", response_model=ClinicalOrder)
async def cancel_order(
    order_id: str,
    action_data: OrderActionRequest = Body(...),
    user=Depends(get_current_user)
):
    """Cancel an order"""
    try:
        order_service = await get_order_service()

        # Check user permissions for cancelling
        if user.get("role") not in ["doctor", "nurse"]:
            raise HTTPException(status_code=403, detail="User does not have permission to cancel orders")

        reason = action_data.reason or "Order cancelled by user"
        order = await order_service.cancel_order(order_id, reason, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error cancelling order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error cancelling order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/hold/{order_id}", response_model=ClinicalOrder)
async def hold_order(
    order_id: str,
    action_data: OrderActionRequest = Body(...),
    user=Depends(get_current_user)
):
    """Put an order on hold"""
    try:
        order_service = await get_order_service()

        # Check user permissions for holding
        if user.get("role") not in ["doctor", "nurse"]:
            raise HTTPException(status_code=403, detail="User does not have permission to hold orders")

        reason = action_data.reason or "Order placed on hold"
        order = await order_service.hold_order(order_id, reason, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error holding order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error holding order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/release/{order_id}", response_model=ClinicalOrder)
async def release_order(
    order_id: str,
    action_data: OrderActionRequest = Body(default_factory=OrderActionRequest),
    user=Depends(get_current_user)
):
    """Release an order from hold"""
    try:
        order_service = await get_order_service()

        # Check user permissions for releasing
        if user.get("role") not in ["doctor", "nurse"]:
            raise HTTPException(status_code=403, detail="User does not have permission to release orders")

        order = await order_service.release_order(order_id, user)
        return order

    except ValueError as e:
        logger.error(f"Validation error releasing order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error releasing order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/discontinue/{order_id}", response_model=ClinicalOrder)
async def discontinue_order(
    order_id: str,
    action_data: OrderActionRequest = Body(...),
    user=Depends(get_current_user)
):
    """Discontinue an order (mark as completed)"""
    try:
        order_service = await get_order_service()

        # Check user permissions for discontinuing
        if user.get("role") not in ["doctor", "nurse"]:
            raise HTTPException(status_code=403, detail="User does not have permission to discontinue orders")

        # For discontinuation, we'll update the status to completed
        from app.models.clinical_order import ClinicalOrderUpdate, OrderStatus
        order_update = ClinicalOrderUpdate(status=OrderStatus.COMPLETED)
        order = await order_service.update_clinical_order(order_id, order_update, user)

        return order

    except ValueError as e:
        logger.error(f"Validation error discontinuing order {order_id}: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Error discontinuing order {order_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))
