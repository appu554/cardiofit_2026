"""
Strawberry GraphQL schema for the Observation Service.

This module provides the GraphQL schema for the Observation Service using Strawberry.
It defines queries and mutations for observation data.
"""

from typing import List, Optional, Any, Dict
import logging
from datetime import datetime
import strawberry
from strawberry.types import Info
from app.graphql import types as gql_types # Added for specific GraphQL types

# Import service
from app.services.observation_service import get_observation_service

# Import settings and auth
from app.core.config import settings
from app.core.auth import get_token_payload, _validate_and_get_payload
from fastapi import HTTPException, status, Request

# Configure logging
logger = logging.getLogger(__name__)

# Define GraphQL types
# Local ObservationType removed, will use gql_types.Observation from app.graphql.types

@strawberry.type
class PageInfo:
    """Pagination info type."""
    has_next_page: bool
    has_previous_page: bool
    start_cursor: Optional[str] = None
    end_cursor: Optional[str] = None
    total_count: int

@strawberry.type
class ObservationEdge:
    """Edge type for pagination."""
    node: gql_types.Observation
    cursor: str

@strawberry.type
class ObservationConnection:
    """Connection type for paginated observations."""
    edges: List[ObservationEdge]
    page_info: PageInfo

# Input types
# Query type
@strawberry.type
class Query:
    """Query type for Observation Service."""
    
    @strawberry.field
    async def observation(self, info: Info, id: str) -> Optional[gql_types.Observation]:
        """Get an observation by ID."""
        try:
            # Get the current user from the context
            request: Request = info.context["request"]
            # Get the token from the Authorization header
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Missing or invalid Authorization header"
                )
            token = auth_header.split(" ")[1]
            current_user = await _validate_and_get_payload(token)
            
            if not current_user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )

            # Get the observation service
            observation_service = await get_observation_service()
            observation_data = await observation_service.get_observation(id)

            if not observation_data:
                return None

            # Convert to Strawberry type
            return gql_types.Observation.from_fhir(observation_data)
            
        except HTTPException as he:
            raise
        except Exception as e:
            logger.error(f"Error getting observation: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Error retrieving observation: {str(e)}"
            )

    @strawberry.field
    async def observations(
        self,
        info: Info,
        filter: Optional[gql_types.ObservationFilterInput] = None,
        page: int = 1,
        count: int = 10
    ) -> ObservationConnection:
        """Get observations with filtering and pagination."""
        try:
            # Get the current user from the context
            request: Request = info.context["request"]
            # Get the token from the Authorization header
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Missing or invalid Authorization header"
                )
            token = auth_header.split(" ")[1]
            current_user = await _validate_and_get_payload(token)
            
            if not current_user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )

            # Get the observation service
            observation_service = await get_observation_service()

            # Build search parameters
            params = {}
            if filter:
                if filter.patient_id is not None: params["patient_id"] = filter.patient_id
                if filter.category is not None: params["category"] = filter.category
                if filter.code is not None: params["code"] = filter.code
                if filter.date is not None: params["date"] = filter.date
                if filter.status is not None: params["status"] = filter.status
            
            params["_page"] = page
            params["_count"] = count
            
            # Remove None values (though explicit checks above mostly handle this)
            # params = {k: v for k, v in params.items() if v is not None} # This line might be redundant now

            # Search observations
            observations_data, total = await observation_service.search_observations(params)

            # Create edges for pagination
            edges = [
                ObservationEdge(
                    node=gql_types.Observation.from_fhir(obs),
                    cursor=str(i + ((page - 1) * count))
                )
                for i, obs in enumerate(observations_data)
            ]

            # Create page info
            has_next_page = (page * count) < total
            has_previous_page = page > 1
            
            return ObservationConnection(
                edges=edges,
                page_info=PageInfo(
                    has_next_page=has_next_page,
                    has_previous_page=has_previous_page,
                    start_cursor=str((page - 1) * count) if edges else None,
                    end_cursor=str((page * count) - 1) if edges else None,
                    total_count=total
                )
            )
            
        except HTTPException as he:
            raise
        except Exception as e:
            logger.error(f"Error getting observations: {str(e)}")
            return ObservationConnection(
                edges=[],
                page_info=PageInfo(
                    has_next_page=False,
                    has_previous_page=False,
                    start_cursor=None,
                    end_cursor=None,
                    total_count=0
                )
            )

# Mutation type
@strawberry.type
class Mutation:
    """Mutation type for Observation Service."""
    
    @strawberry.mutation
    async def create_observation(
        self,
        info: Info,
        input: gql_types.ObservationInput
    ) -> Optional[gql_types.Observation]:
        """Create a new observation."""
        try:
            # Get the current user from the context
            request: Request = info.context["request"]
            # Get the token from the Authorization header
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Missing or invalid Authorization header"
                )
            token = auth_header.split(" ")[1]
            current_user = await _validate_and_get_payload(token)
            
            if not current_user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )

            # Get the observation service
            observation_service = await get_observation_service()
            
            # Convert input to dict and add metadata
            observation_data = input.__dict__
            observation_data["meta"] = {
                "lastUpdated": datetime.utcnow().isoformat(),
                "versionId": "1"
            }
            
            # Create the observation
            created_observation = await observation_service.create_observation(observation_data)
            
            # Convert to Strawberry type
            return gql_types.Observation.from_fhir(created_observation)
            
        except HTTPException as he:
            raise
        except Exception as e:
            logger.error(f"Error creating observation: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Error creating observation: {str(e)}"
            )
    
    @strawberry.mutation
    async def update_observation(
        self,
        info: Info,
        id: str,
        input: gql_types.UpdateObservationInput
    ) -> Optional[gql_types.Observation]:
        """Update an existing observation."""
        try:
            # Get the current user from the context
            request: Request = info.context["request"]
            # Get the token from the Authorization header
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Missing or invalid Authorization header"
                )
            token = auth_header.split(" ")[1]
            current_user = await _validate_and_get_payload(token)
            
            if not current_user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )

            # Get the observation service
            observation_service = await get_observation_service()
            
            # Convert input to dict and add metadata
            update_data = {k: v for k, v in input.__dict__.items() if v is not None}
            update_data["meta"] = {
                "lastUpdated": datetime.utcnow().isoformat()
            }
            
            # Update the observation
            updated_observation = await observation_service.update_observation(id, update_data)
            
            if not updated_observation:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"Observation with ID {id} not found"
                )
            
            # Convert to Strawberry type
            return gql_types.Observation.from_fhir(updated_observation)
            
        except HTTPException as he:
            raise
        except Exception as e:
            logger.error(f"Error updating observation: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Error updating observation: {str(e)}"
            )
    
    @strawberry.mutation
    async def delete_observation(
        self,
        info: Info,
        id: str
    ) -> bool:
        """Delete an observation by marking it as 'entered-in-error'."""
        try:
            # Get the current user from the context
            request: Request = info.context["request"]
            # Get the token from the Authorization header
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Missing or invalid Authorization header"
                )
            token = auth_header.split(" ")[1]
            current_user = await _validate_and_get_payload(token)
            
            if not current_user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )

            # Get the observation service
            observation_service = await get_observation_service()
            
            # Mark as entered-in-error (soft delete)
            update_data = {
                "status": "entered-in-error",
                "meta": {
                    "lastUpdated": datetime.utcnow().isoformat()
                }
            }
            
            # Update the observation
            success = await observation_service.update_observation(id, update_data)
            
            if not success:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"Observation with ID {id} not found"
                )
            
            return True
            
        except HTTPException as he:
            raise
        except Exception as e:
            logger.error(f"Error deleting observation: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Error deleting observation: {str(e)}"
            )

# Create schema
schema = strawberry.federation.Schema(query=Query, mutation=Mutation, types=[gql_types.Observation])
