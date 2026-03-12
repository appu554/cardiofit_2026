from fastapi import APIRouter
from app.api.endpoints import orders, order_sets, order_management, webhooks

api_router = APIRouter()

# Include all endpoint routers
api_router.include_router(orders.router, prefix="/orders", tags=["orders"])
api_router.include_router(order_sets.router, prefix="/order-sets", tags=["order-sets"])
api_router.include_router(order_management.router, prefix="/order-management", tags=["order-management"])
api_router.include_router(webhooks.router, prefix="/webhooks", tags=["Webhooks"])
