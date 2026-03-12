"""
Federation endpoint for the Encounter Management Service.

This module provides the federation schema with proper Apollo Federation v2 directives.
"""

from fastapi import APIRouter
from strawberry.schema.schema_converter import to_graphql_schema
from graphql import print_schema
import logging

# Import the schema
from app.graphql.federation_schema import schema

logger = logging.getLogger(__name__)

router = APIRouter()

# Generate the SDL with Federation directives
def get_federation_sdl():
    """Generate the SDL with Federation directives."""
    try:
        # Get the base schema as string
        graphql_schema = to_graphql_schema(schema)
        schema_str = print_schema(graphql_schema)

        # Add @key directive to main types
        schema_str = schema_str.replace(
            "type Encounter {",
            "type Encounter @key(fields: \"id\") {"
        )
        
        schema_str = schema_str.replace(
            "type Location {",
            "type Location @key(fields: \"id\") {"
        )

        # Mark shared FHIR types as @shareable to avoid conflicts with other services
        shared_types = [
            "CodeableConcept", "Coding", "Identifier", "Reference", "Period",
            "Quantity", "ContactPoint", "Address"
        ]

        for type_name in shared_types:
            schema_str = schema_str.replace(
                f"type {type_name} {{",
                f"type {type_name} @shareable {{"
            )

        # Add Federation v2 directives
        federation_sdl = f"""extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable", "@external"])

{schema_str}
"""

        logger.info("Generated federation SDL successfully")
        return federation_sdl
        
    except Exception as e:
        logger.error(f"Error generating federation SDL: {e}")
        raise

@router.post("/")
async def federation_endpoint(query: dict):
    """
    Federation endpoint for Apollo Gateway introspection.
    """
    try:
        # Handle _service query for federation
        if query.get("query") and "_service" in query["query"]:
            sdl = get_federation_sdl()
            return {
                "data": {
                    "_service": {
                        "sdl": sdl
                    }
                }
            }
        
        # Handle regular GraphQL queries
        from strawberry.fastapi import GraphQLRouter
        graphql_router = GraphQLRouter(schema)
        
        # Execute the query using the schema
        result = await schema.execute(
            query["query"],
            variable_values=query.get("variables"),
            operation_name=query.get("operationName")
        )
        
        response = {"data": result.data}
        if result.errors:
            response["errors"] = [{"message": str(error)} for error in result.errors]
            
        return response
        
    except Exception as e:
        logger.error(f"Error in federation endpoint: {e}")
        return {
            "errors": [{"message": f"Internal server error: {str(e)}"}]
        }

@router.get("/")
async def federation_info():
    """
    Get federation endpoint information.
    """
    return {
        "service": "Encounter Management Service",
        "federation": "Apollo Federation v2",
        "endpoint": "/api/federation",
        "schema_available": True
    }
