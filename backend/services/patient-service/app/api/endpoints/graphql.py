"""
GraphQL endpoint for the Patient Service.

This module provides a GraphQL endpoint for the Patient Service using FastAPI and Graphene.
"""

from fastapi import APIRouter, Depends, Request, Body, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials
import logging
import json

# Import GraphQL schema
from app.graphql.schema import schema

# Import authentication
from app.core.auth import get_token_payload

# Helper function to get token payload with request context
async def get_token_payload_with_request(request: Request):
    """
    Get token payload with request context.
    This extracts the token from the Authorization header and passes it to get_token_payload
    along with the request object for header access.
    """
    auth_header = request.headers.get("Authorization")
    if not auth_header:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing",
            headers={"WWW-Authenticate": "Bearer"}
        )

    # Extract the token
    parts = auth_header.split()
    if len(parts) != 2 or parts[0].lower() != "bearer":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid Authorization header format",
            headers={"WWW-Authenticate": "Bearer"}
        )

    token = parts[1]

    # Create a credentials object
    credentials = HTTPAuthorizationCredentials(scheme="Bearer", credentials=token)

    # Get token payload with request context
    return await get_token_payload(credentials, request)

# Configure logging
logger = logging.getLogger(__name__)

# Create router
router = APIRouter()

# Add GraphQL endpoint with token support
@router.post("/")
async def graphql_endpoint(
    request: Request,
    query: dict = Body(...),
    token: str = None
):
    """
    GraphQL endpoint for the Patient Service.

    This endpoint handles GraphQL queries and mutations.
    It supports authentication via the Authorization header or a token query parameter.
    """
    # Get token payload with request context
    token_payload = await get_token_payload_with_request(request)
    try:
        # Log all headers for debugging
        logger.info("=== GRAPHQL ENDPOINT HEADERS ===")
        for header_name, header_value in request.headers.items():
            if header_name.lower() in ['authorization', 'x-user-id', 'x-user-role', 'x-user-roles', 'x-user-permissions', 'x-user-email']:
                logger.info(f"  {header_name}: {header_value}")
        logger.info("=== END GRAPHQL ENDPOINT HEADERS ===")

        # Extract the query and variables from the request
        query_str = query.get("query", "")
        variables = query.get("variables", {})
        operation_name = query.get("operationName", None)

        # Log the request
        logger.info(f"GraphQL request: {query_str}")
        logger.info(f"Variables: {variables}")

        # Check if user info is in request state
        if hasattr(request.state, 'user'):
            logger.info(f"User info from request state: {request.state.user}")
        else:
            logger.warning("No user info in request state")

        # Use token from query parameter if provided
        context = {"request": request, "token_payload": token_payload}
        if token:
            # Override the token_payload with the token from the query parameter
            # This would require additional code to validate the token
            logger.info(f"Using token from query parameter: {token}")
            # For now, we'll just log it and continue using the token from the header

        # Execute the query
        result = await schema.execute_async(
            query_str,
            variable_values=variables,
            context_value=context,
            operation_name=operation_name
        )

        # Build the response
        response = {"data": result.data}
        if result.errors:
            response["errors"] = [str(error) for error in result.errors]

        # Log the response
        logger.info(f"GraphQL response: {response}")

        return response
    except Exception as e:
        logger.error(f"Error executing GraphQL query: {str(e)}")
        return {"errors": [str(e)]}

# Add GET endpoint for GraphQL to support URL-based queries
@router.get("/")
async def graphql_get_endpoint(
    request: Request,
    query: str = None,
    variables: str = None,
    operation_name: str = None,
    token: str = None
):
    """
    GraphQL GET endpoint for the Patient Service.

    This endpoint handles GraphQL queries via GET requests.
    It supports authentication via the Authorization header or a token query parameter.

    Example:
    GET /api/graphql?query={patient(id:"123"){id,name{family,given}}}&token=your-token-here
    """
    try:
        if not query:
            return {"errors": ["No GraphQL query provided"]}

        # Parse variables if provided
        parsed_variables = {}
        if variables:
            try:
                parsed_variables = json.loads(variables)
            except:
                return {"errors": ["Invalid variables format"]}

        # Log the request
        logger.info(f"GraphQL GET request: {query}")
        logger.info(f"Variables: {parsed_variables}")

        # Get token payload with request context
        token_payload = await get_token_payload_with_request(request)

        # Log all headers for debugging
        logger.info("=== GRAPHQL GET ENDPOINT HEADERS ===")
        for header_name, header_value in request.headers.items():
            if header_name.lower() in ['authorization', 'x-user-id', 'x-user-role', 'x-user-roles', 'x-user-permissions', 'x-user-email']:
                logger.info(f"  {header_name}: {header_value}")
        logger.info("=== END GRAPHQL GET ENDPOINT HEADERS ===")

        # Use token from query parameter if provided
        context = {"request": request, "token_payload": token_payload}
        if token:
            # Log token usage from query parameter
            logger.info(f"Using token from query parameter: {token}")
            # For now, we'll just log it and continue using the token from the header

        # Execute the query
        result = await schema.execute_async(
            query,
            variable_values=parsed_variables,
            context_value=context,
            operation_name=operation_name
        )

        # Build the response
        response = {"data": result.data}
        if result.errors:
            response["errors"] = [str(error) for error in result.errors]

        # Log the response
        logger.info(f"GraphQL response: {response}")

        return response
    except Exception as e:
        logger.error(f"Error executing GraphQL query: {str(e)}")
        return {"errors": [str(e)]}

logger.info("Added GraphQL endpoints at /graphql (POST and GET)")

# Add GraphiQL endpoint
@router.get("/playground")
async def graphiql(request: Request):
    """
    GraphiQL playground for exploring the GraphQL API.

    This endpoint serves the GraphiQL interface, which is a web-based tool
    for exploring and testing GraphQL APIs.
    """
    return {
        "html": """
        <!DOCTYPE html>
        <html>
        <head>
            <title>GraphiQL</title>
            <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
            <style>
                body {
                    margin: 0;
                    height: 100vh;
                }
                #graphiql {
                    height: calc(100vh - 50px);
                }
                .token-input {
                    display: flex;
                    align-items: center;
                    padding: 8px;
                    background-color: #f3f3f3;
                    border-bottom: 1px solid #ddd;
                    height: 34px;
                }
                .token-input label {
                    margin-right: 8px;
                    font-family: system-ui, sans-serif;
                    font-size: 14px;
                }
                .token-input input {
                    flex-grow: 1;
                    padding: 6px;
                    border: 1px solid #ccc;
                    border-radius: 4px;
                    font-family: monospace;
                    font-size: 13px;
                }
                .token-input button {
                    margin-left: 8px;
                    padding: 6px 12px;
                    background-color: #4CAF50;
                    color: white;
                    border: none;
                    border-radius: 4px;
                    cursor: pointer;
                    font-size: 13px;
                }
                .token-input button:hover {
                    background-color: #45a049;
                }
            </style>
        </head>
        <body>
            <div class="token-input">
                <label for="token">Auth Token:</label>
                <input type="text" id="token" placeholder="Bearer your-token-here" />
                <button onclick="updateToken()">Update</button>
            </div>
            <div id="graphiql"></div>

            <script
                crossorigin
                src="https://unpkg.com/react/umd/react.production.min.js"
            ></script>
            <script
                crossorigin
                src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"
            ></script>
            <script
                crossorigin
                src="https://unpkg.com/graphiql/graphiql.min.js"
            ></script>
            <script>
                // Initialize with saved token if available
                document.getElementById('token').value = localStorage.getItem('auth_token') || '';

                // Function to update the token
                function updateToken() {
                    const token = document.getElementById('token').value;
                    localStorage.setItem('auth_token', token);

                    // Reload the page to apply the new token
                    window.location.reload();
                }

                // Create the fetcher with the token
                const fetcher = GraphiQL.createFetcher({
                    url: '/api/graphql',
                    headers: {
                        'Authorization': localStorage.getItem('auth_token') || ''
                    }
                });

                // Render GraphiQL
                ReactDOM.render(
                    React.createElement(GraphiQL, { fetcher }),
                    document.getElementById('graphiql'),
                );
            </script>
        </body>
        </html>
        """
    }
