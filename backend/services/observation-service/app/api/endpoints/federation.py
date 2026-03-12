"""
Federation endpoint for the Observation Service.

This module provides a dedicated endpoint for Apollo Federation that bypasses authentication
and exposes the GraphQL schema with Federation directives.
"""

from fastapi import APIRouter, Request, Body, HTTPException, status
from graphql import print_schema
import logging

# Import GraphQL schema
from app.graphql.schema import schema

# Configure logging
logger = logging.getLogger(__name__)

# Create router
router = APIRouter()

# Generate the SDL with Federation directives
def get_federation_sdl():
    """Generate the SDL with Federation directives."""
    # Get the base schema as string - Strawberry schemas use different method
    from strawberry.schema.schema_converter import to_graphql_schema
    graphql_schema = to_graphql_schema(schema)
    schema_str = print_schema(graphql_schema)

    # Add @key directive to Observation type
    schema_str = schema_str.replace(
        "type Observation {",
        "type Observation @key(fields: \"id\") {"
    )

    # Mark shared FHIR types as @shareable to avoid conflicts with Patient service
    shared_types = [
        "CodeableConcept", "Coding", "Identifier", "Reference", "Period",
        "Quantity", "Annotation", "HumanName", "ContactPoint", "Address"
    ]

    for type_name in shared_types:
        schema_str = schema_str.replace(
            f"type {type_name} {{",
            f"type {type_name} @shareable {{"
        )

    # Add Federation directives
    federation_sdl = f"""
    extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable"])

    {schema_str}
    """

    return federation_sdl

@router.post("")
async def federation_endpoint(
    request: Request,
    query: dict = Body(...)
):
    """
    GraphQL endpoint for Apollo Federation.
    This endpoint bypasses authentication and is used by the Apollo Gateway
    to introspect the schema and execute federated queries.
    """
    try:
        # Extract the query and variables from the request
        query_str = query.get("query", "")
        variables = query.get("variables", {})
        operation_name = query.get("operationName", None)

        # Log the federation request
        logger.info(f"Federation request: {query_str}")

        # Create context without authentication for federation
        context = {
            "request": request,
            "token_payload": None,  # No authentication for federation
            "federation": True  # Flag to indicate this is a federation request
        }

        # Execute the GraphQL query - Strawberry federation schema handles _service automatically
        result = await schema.execute(
            query_str,
            variable_values=variables,
            operation_name=operation_name,
            context_value=context
        )

        # Check for errors
        if result.errors:
            logger.error(f"Federation GraphQL errors: {result.errors}")
            return {
                "data": result.data,
                "errors": [str(error) for error in result.errors]
            }

        return {"data": result.data}

    except Exception as e:
        logger.error(f"Error in federation endpoint: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error executing federation GraphQL query: {str(e)}"
        )

@router.get("")
async def federation_get_endpoint(
    request: Request,
    query: str = None,
    variables: str = None,
    operation_name: str = None
):
    """
    GraphQL GET endpoint for Apollo Federation.
    This endpoint handles federation queries via GET requests.
    """
    try:
        # Log the federation GET request
        logger.info(f"Federation GET request: {query}")

        # Parse variables if provided
        variables_dict = {}
        if variables:
            import json
            try:
                variables_dict = json.loads(variables)
            except json.JSONDecodeError as e:
                logger.error(f"Error parsing variables: {str(e)}")
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=f"Invalid variables JSON: {str(e)}"
                )

        # Create context without authentication for federation
        context = {
            "request": request,
            "token_payload": None,  # No authentication for federation
            "federation": True  # Flag to indicate this is a federation request
        }

        # Execute the GraphQL query - Strawberry federation schema handles _service automatically
        result = await schema.execute(
            query,
            variable_values=variables_dict,
            operation_name=operation_name,
            context_value=context
        )

        # Check for errors
        if result.errors:
            logger.error(f"Federation GraphQL GET errors: {result.errors}")
            return {
                "data": result.data,
                "errors": [str(error) for error in result.errors]
            }

        return {"data": result.data}

    except Exception as e:
        logger.error(f"Error in federation GET endpoint: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error executing federation GraphQL GET query: {str(e)}"
        )

# Log that the federation endpoint is being added
logger.info("Added federation endpoints at /federation (POST and GET) for Apollo Federation")
