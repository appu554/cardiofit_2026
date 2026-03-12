"""
Federation endpoint for the Patient Service.

This module provides a GraphQL endpoint specifically for Apollo Federation.
It bypasses authentication to allow the Federation Gateway to introspect the schema.
"""

from fastapi import APIRouter, Request, Body
from fastapi.responses import HTMLResponse, JSONResponse
import logging
from graphql import graphql, graphql_sync, print_schema

# Import GraphQL schema
from app.graphql.schema import schema

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create router
router = APIRouter()

# Generate the SDL with Federation directives
def get_federation_sdl():
    """Generate the SDL with Federation directives."""
    # Get the base schema as string
    schema_str = print_schema(schema.graphql_schema)

    # Add @key directive to Patient type
    schema_str = schema_str.replace(
        "type Patient {",
        "type Patient @key(fields: \"id\") {"
    )

    # Mark shared FHIR types as @shareable to avoid conflicts with other services
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

    This endpoint bypasses authentication to allow the Federation Gateway to introspect the schema.
    It should only be used by the Federation Gateway, not by clients directly.
    """
    try:
        # Log the request
        logger.info(f"Federation GraphQL request received")

        # Extract the GraphQL query and variables
        graphql_query = query.get("query", "")
        variables = query.get("variables", {})
        operation_name = query.get("operationName")

        # Check if this is a Federation _service query
        is_federation_service_query = "_service" in graphql_query

        # For Federation _service queries, directly return the SDL
        if is_federation_service_query:
            logger.info(f"Processing Federation _service query")

            # Generate the Federation SDL
            sdl = get_federation_sdl()

            # Return the SDL in the expected format
            response = {
                "data": {
                    "_service": {
                        "sdl": sdl
                    }
                }
            }

            logger.info(f"Federation _service response sent")
            return response

        # Check if this is a reference resolver query
        is_reference_resolver = "__resolveReference" in graphql_query

        if is_reference_resolver:
            logger.info(f"Processing Federation reference resolver query")

            # Extract the ID from the query
            # This is a simple extraction and might need to be more robust
            import re
            id_match = re.search(r'id\s*:\s*"([^"]+)"', graphql_query)
            if id_match:
                patient_id = id_match.group(1)
                logger.info(f"Resolving reference for Patient with ID: {patient_id}")

                # Import the patient service
                from app.services.patient_service import get_patient_service

                # Get the patient service
                patient_service = await get_patient_service()

                # Get the patient
                patient_data = await patient_service.get_patient(patient_id)

                if patient_data:
                    # Convert to GraphQL type
                    from app.graphql.types import Patient
                    patient = Patient.from_fhir(patient_data)

                    # Return the patient
                    return {
                        "data": {
                            "_entities": [
                                patient
                            ]
                        }
                    }
                else:
                    logger.warning(f"Patient with ID {patient_id} not found")
                    return {
                        "data": {
                            "_entities": [
                                None
                            ]
                        }
                    }

        # For regular queries, use the regular schema with federation context
        logger.info(f"Executing regular GraphQL query: {operation_name}")

        # Create context that bypasses authentication for federation
        context = {
            "request": request,
            "federation": True,  # Flag to indicate this is a federation request
            "token_payload": None  # No authentication for federation
        }

        result = await graphql(
            schema.graphql_schema,
            graphql_query,
            variable_values=variables,
            operation_name=operation_name,
            context_value=context
        )

        # Convert the result to a dict
        response = {"data": result.data}
        if result.errors:
            response["errors"] = [{"message": str(error)} for error in result.errors]

        # Log the response
        logger.info(f"Federation GraphQL response sent")

        return response
    except Exception as e:
        logger.error(f"Error executing Federation GraphQL query: {str(e)}")
        return {"errors": [{"message": str(e)}]}

# Add GraphiQL playground for Federation
@router.get("/playground")
async def federation_playground(request: Request):
    """
    GraphiQL playground for exploring the Federation GraphQL API.
    """
    html_content = """
    <!DOCTYPE html>
    <html>
    <head>
        <title>Federation GraphQL Playground</title>
        <meta charset="utf-8" />
        <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
        <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
    </head>
    <body style="margin: 0; overflow: hidden; height: 100vh;">
        <div id="graphiql" style="height: 100vh;"></div>
        <script src="https://unpkg.com/react@17/umd/react.development.js"></script>
        <script src="https://unpkg.com/react-dom@17/umd/react-dom.development.js"></script>
        <script src="https://unpkg.com/graphiql/graphiql.min.js"></script>
        <script>
            const fetcher = GraphiQL.createFetcher({
                url: '/api/federation',
            });
            ReactDOM.render(
                React.createElement(GraphiQL, { fetcher }),
                document.getElementById('graphiql'),
            );
        </script>
    </body>
    </html>
    """
    return HTMLResponse(content=html_content)

logger.info("Added Federation GraphQL endpoints at /federation (POST) and /federation/playground (GET)")
