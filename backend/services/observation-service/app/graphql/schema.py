"""
GraphQL schema for the Observation Service.

This module provides the GraphQL schema for the Observation Service using Strawberry.
It defines queries and mutations for observation data, following the same pattern
as the Patient Service.
"""

import strawberry
# from strawberry.scalars import JSON # No longer using generic JSON for inputs
from typing import List, Optional, Union, Dict, Any
import logging
from datetime import datetime

# Import GraphQL types
from . import types as gql_types

# Import service
from app.services.observation_service import get_observation_service # For business logic layer
from app.services.fhir_service import get_fhir_service # For direct FHIR operations if ObservationService doesn't cover all needs or for simplicity here

# Import settings and auth
from app.core.config import settings
from app.core.auth import get_token_payload, _validate_and_get_payload
from fastapi import HTTPException, status, Depends

# Configure logging
logger = logging.getLogger(__name__)

@strawberry.type
class Query:
    """GraphQL Query type for Observation Service."""
    
    @strawberry.field(description="Get an observation by ID")
    async def observation(
        self, 
        info: strawberry.Info, 
        id: str
    ) -> Optional[gql_types.Observation]:
        """Resolver for observation query."""
        try:
            request = info.context["request"]
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                logger.warning("Observation query: Missing or malformed Authorization header.")
                raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Not authenticated")
            token = auth_header.split(" ")[1]
            current_user = await _validate_and_get_payload(token)
            
            if not current_user:
                logger.warning("Unauthorized access attempt to observation endpoint")
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )
            
            # Get the observation data
            service = await get_observation_service()
            observation_data = await service.get_observation_by_id(id, current_user)
            
            if not observation_data:
                logger.warning(f"Observation not found: {id}")
                return None
                
            logger.info(f"Successfully retrieved observation: {id}")
            return gql_types.Observation.from_fhir(observation_data)
            
        except HTTPException as he:
            logger.error(f"HTTP error in observation resolver: {str(he)}")
            raise
        except Exception as e:
            logger.error(f"Error in observation resolver: {str(e)}", exc_info=True)
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to retrieve observation: {str(e)}"
            )
    
    @strawberry.field(description="Search for observations with filtering and pagination")
    async def observations(
        self,
        info,
        patient_id: Optional[str] = None,
        category: Optional[str] = None,
        code: Optional[str] = None,
        date: Optional[str] = None,
        status: Optional[str] = None,
        page: int = 1,
        count: int = 10
    ) -> List[gql_types.Observation]:
        """Resolver for observations query."""
        try:
            request = info.context["request"]

            # Check if this is a federation request (bypass authentication)
            is_federation = info.context.get("federation", False)
            current_user = None

            if not is_federation:
                auth_header = request.headers.get("Authorization")
                if not auth_header or not auth_header.startswith("Bearer "):
                    logger.warning("Observations query: Missing or malformed Authorization header.")
                    raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Not authenticated")
                token = auth_header.split(" ")[1]
                current_user = await _validate_and_get_payload(token)

                if not current_user:
                    logger.warning("Unauthorized access attempt to observations endpoint")
                    raise HTTPException(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        detail="Authentication required"
                    )
            else:
                logger.info("Federation request detected, bypassing authentication")
            
            # Build FHIR-compatible search parameters
            search_params_dict = {}
            if patient_id:
                search_params_dict["subject"] = f"Patient/{patient_id}"
            if category:
                # Assuming category is a simple code string. For complex CodeableConcepts, this might need adjustment.
                search_params_dict["category"] = category 
            if code:
                # Assuming code is a simple code string. For complex Coding/CodeableConcepts, format might be system|code.
                search_params_dict["code"] = code
            if date:
                # FHIR date search. For exact match: date=YYYY-MM-DD. Prefixes like 'eq', 'gt', 'lt' can be used.
                search_params_dict["date"] = date 
            if status:
                search_params_dict["status"] = status
            
            # Add pagination parameters
            search_params_dict["_count"] = count
            search_params_dict["_offset"] = (page - 1) * count
            
            # Get paginated results
            service = await get_observation_service() # Ensure service is awaited if get_observation_service is async
            observations_data, total = await service.search_observations(
                search_params=search_params_dict,
                token_payload=current_user
            )
            
            logger.info(f"Successfully retrieved {len(observations_data)} of {total} observations (pagination temporarily removed)")
            return [gql_types.Observation.from_fhir(obs) for obs in observations_data if obs]
            
        except HTTPException as he:
            logger.error(f"HTTP error in observations resolver: {str(he)}")
            raise
        except Exception as e:
            logger.error(f"Error in observations resolver: {str(e)}", exc_info=True)
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to retrieve observations: {str(e)}"
            )


# Helper functions for mapping GraphQL input to FHIR dictionary
def _input_to_fhir_coding(coding_input: Optional[gql_types.CodingInput]) -> Optional[Dict[str, Any]]:
    if not coding_input:
        return None
    fhir_coding = {}
    if coding_input.system is not None: fhir_coding["system"] = coding_input.system
    if coding_input.code is not None: fhir_coding["code"] = coding_input.code
    if coding_input.display is not None: fhir_coding["display"] = coding_input.display
    return fhir_coding if fhir_coding else None

def _input_to_fhir_codeable_concept(cc_input: Optional[gql_types.CodeableConceptInput]) -> Optional[Dict[str, Any]]:
    if not cc_input:
        return None
    fhir_cc = {}
    if cc_input.coding:
        codings = [_input_to_fhir_coding(c) for c in cc_input.coding if c]
        valid_codings = [c for c in codings if c]
        if valid_codings: fhir_cc["coding"] = valid_codings
    if cc_input.text is not None: fhir_cc["text"] = cc_input.text
    return fhir_cc if fhir_cc else None

def _input_to_fhir_quantity(quantity_input: Optional[gql_types.QuantityInput]) -> Optional[Dict[str, Any]]:
    if not quantity_input:
        return None
    fhir_quantity = {}
    if quantity_input.value is not None: fhir_quantity["value"] = quantity_input.value
    if quantity_input.unit is not None: fhir_quantity["unit"] = quantity_input.unit
    if quantity_input.system is not None: fhir_quantity["system"] = quantity_input.system
    if quantity_input.code is not None: fhir_quantity["code"] = quantity_input.code
    return fhir_quantity if fhir_quantity else None

def _input_to_fhir_reference(reference_input: Optional[gql_types.ReferenceInput]) -> Optional[Dict[str, Any]]:
    if not reference_input or not reference_input.reference:
        return None
    return {"reference": reference_input.reference}

def _input_to_fhir_observation_dict(input_data: gql_types.CreateObservationInput) -> Dict[str, Any]:
    fhir_obs: Dict[str, Any] = {"resourceType": "Observation"}

    fhir_obs["status"] = input_data.status
    
    code_fhir = _input_to_fhir_codeable_concept(input_data.code)
    if not code_fhir:
        raise ValueError("Observation 'code' is mandatory and could not be mapped.") 
    fhir_obs["code"] = code_fhir

    subject_fhir = _input_to_fhir_reference(input_data.subject)
    if not subject_fhir:
        raise ValueError("Observation 'subject' is mandatory and could not be mapped.")
    fhir_obs["subject"] = subject_fhir

    if input_data.effective_date_time:
        fhir_obs["effectiveDateTime"] = input_data.effective_date_time
    
    if input_data.value_quantity:
        fhir_obs["valueQuantity"] = _input_to_fhir_quantity(input_data.value_quantity)
    elif input_data.value_codeable_concept:
        fhir_obs["valueCodeableConcept"] = _input_to_fhir_codeable_concept(input_data.value_codeable_concept)
    elif input_data.value_string is not None:
        fhir_obs["valueString"] = input_data.value_string
    elif input_data.value_boolean is not None:
        fhir_obs["valueBoolean"] = input_data.value_boolean
    elif input_data.value_integer is not None:
        fhir_obs["valueInteger"] = input_data.value_integer

    if input_data.category:
        categories = [_input_to_fhir_codeable_concept(cat) for cat in input_data.category if cat]
        valid_categories = [c for c in categories if c]
        if valid_categories: fhir_obs["category"] = valid_categories

    if input_data.issued:
        fhir_obs["issued"] = input_data.issued

    if input_data.performer:
        performers = [_input_to_fhir_reference(p) for p in input_data.performer if p]
        valid_performers = [p for p in performers if p]
        if valid_performers: fhir_obs["performer"] = valid_performers
            
    return fhir_obs

@strawberry.type
class Mutation:
    """GraphQL Mutation type for Observation Service.
    
    This class defines all the available mutations for the Observation service.
    Each mutation requires authentication and performs the corresponding CRUD operation.
    """
    
    @strawberry.mutation(description="Create a new observation.")
    async def create_observation(
        self,
        info: strawberry.Info,
        input: gql_types.CreateObservationInput
    ) -> gql_types.ObservationResponse:
        """Resolver for create_observation mutation."""
        try:
            request = info.context["request"]

            # Check if this is a federation request (bypass authentication)
            is_federation = info.context.get("federation", False)
            current_user = None

            if not is_federation:
                auth_header = request.headers.get("Authorization")
                if not auth_header or not auth_header.startswith("Bearer "):
                    logger.warning("Create observation: Missing or malformed Authorization header.")
                    return gql_types.ObservationResponse.from_error("Not authenticated")
                token = auth_header.split(" ")[1]
                current_user = await _validate_and_get_payload(token)

                if not current_user:
                    logger.warning("Unauthorized access attempt to create_observation endpoint")
                    return gql_types.ObservationResponse.from_error("Authentication required")
            else:
                logger.info("Federation request detected for create_observation, bypassing authentication")

            try:
                observation_fhir_dict = _input_to_fhir_observation_dict(input)
            except ValueError as ve:
                logger.error(f"Validation error during input mapping: {str(ve)}")
                return gql_types.ObservationResponse.from_error(f"Invalid input: {str(ve)}")
            
            fhir_service = get_fhir_service() 
            created_observation_fhir = await fhir_service.create(resource=observation_fhir_dict, token_payload=current_user)

            if not created_observation_fhir:
                logger.error("Failed to create observation, FHIR service returned None or empty response")
                return gql_types.ObservationResponse.from_error("Failed to create observation in backend")

            strawberry_observation = gql_types.Observation.from_fhir(created_observation_fhir)
            if not strawberry_observation:
                logger.error("Failed to map created FHIR observation back to GraphQL type")
                return gql_types.ObservationResponse.from_error("Failed to process created observation data after creation")

            logger.info(f"Successfully created observation: {strawberry_observation.id}")
            return gql_types.ObservationResponse.from_observation(
                observation=strawberry_observation, 
                success=True, 
                message="Observation created successfully."
            )
            
        except HTTPException as he:
            logger.error(f"HTTP error in create_observation resolver: {str(he.detail)}")
            raise 
        except Exception as e:
            logger.error(f"Error in create_observation resolver: {str(e)}", exc_info=True)
            return gql_types.ObservationResponse.from_error(f"An unexpected error occurred: {str(e)}")

    # @strawberry.mutation(description="Update an existing observation.")
    # async def update_observation(
    #     self, 
    #     info: strawberry.Info, 
    #     id: str, 
    #     input: gql_types.UpdateObservationInput 
    # ) -> gql_types.ObservationResponse:
    #     """Resolver for update_observation mutation."""
    #     logger.error("update_observation not yet implemented with new input types")
    #     return gql_types.ObservationResponse.from_error("Update observation not implemented")

    # @strawberry.mutation(description="Delete an observation by ID.")
    # async def delete_observation(
    #     self, 
    #     info: strawberry.Info, 
    #     id: str
    # ) -> gql_types.ObservationResponse:
    #     """Resolver for delete_observation mutation."""
    #     logger.error("delete_observation not yet implemented")
    #     return gql_types.ObservationResponse.from_error("Delete observation not implemented")

# Create schema
schema = strawberry.federation.Schema(query=Query, mutation=Mutation)
