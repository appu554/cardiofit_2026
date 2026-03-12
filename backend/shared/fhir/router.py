"""
FHIR Router Factory for Clinical Synthesis Hub.

This module provides a factory function for creating FastAPI routers that handle
FHIR API endpoints for different resource types.
"""
from typing import Type, Callable, Dict, Any, Optional, List
from fastapi import APIRouter, Depends, HTTPException, Request, Response, status
from fastapi.routing import APIRoute
from pydantic import BaseModel

class FHIRRouterConfig(BaseModel):
    """Configuration for the FHIR router."""
    resource_type: str
    service_class: Type
    get_token_payload: Callable
    prefix: str = ""
    tags: Optional[List[str]] = None


def create_fhir_router(config: FHIRRouterConfig) -> APIRouter:
    """
    Create a FastAPI router with standard FHIR endpoints for the given resource type.
    
    Args:
        config: Configuration for the FHIR router
        
    Returns:
        APIRouter: Configured FastAPI router
    """
    router = APIRouter(prefix=config.prefix, tags=config.tags or [config.resource_type])
    service = config.service_class()
    
    @router.get("/{resource_id}")
    async def read_resource(
        resource_id: str,
        token_payload: Dict[str, Any] = Depends(config.get_token_payload)
    ):
        """Read a single resource by ID."""
        try:
            return await service.read(resource_id, token_payload)
        except Exception as e:
            raise HTTPException(status_code=404, detail=str(e))
    
    @router.get("")
    async def search_resources(
        request: Request,
        token_payload: Dict[str, Any] = Depends(config.get_token_payload)
    ):
        """Search for resources with query parameters."""
        try:
            params = dict(request.query_params)
            return await service.search(params, token_payload)
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @router.post("")
    async def create_resource(
        resource: Dict[str, Any],
        token_payload: Dict[str, Any] = Depends(config.get_token_payload)
    ):
        """Create a new resource."""
        try:
            return await service.create(resource, token_payload)
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @router.put("/{resource_id}")
    async def update_resource(
        resource_id: str,
        resource: Dict[str, Any],
        token_payload: Dict[str, Any] = Depends(config.get_token_payload)
    ):
        """Update an existing resource."""
        try:
            return await service.update(resource_id, resource, token_payload)
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @router.delete("/{resource_id}")
    async def delete_resource(
        resource_id: str,
        token_payload: Dict[str, Any] = Depends(config.get_token_payload)
    ):
        """Delete a resource."""
        try:
            await service.delete(resource_id, token_payload)
            return {"status": "success", "message": f"{config.resource_type}/{resource_id} deleted"}
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    return router
