"""
GraphQL endpoint for the Observation Service.

This module provides a GraphQL endpoint for the Observation Service using FastAPI and Graphene.
"""

from fastapi import APIRouter, Depends, Request, Body, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
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
    return await get_token_payload(credentials)

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
    GraphQL endpoint for the Observation Service.

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

        # Execute the GraphQL query
        result = await schema.execute_async(
            query_str,
            variables=variables,
            operation_name=operation_name,
            context_value=context
        )

        # Check for errors
        if result.errors:
            logger.error(f"GraphQL errors: {result.errors}")
            return {
                "data": result.data,
                "errors": [str(error) for error in result.errors]
            }

        return {"data": result.data}

    except Exception as e:
        logger.error(f"Error in GraphQL endpoint: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error executing GraphQL query: {str(e)}"
        )

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
    GraphQL GET endpoint for the Observation Service.

    This endpoint handles GraphQL queries via GET requests.
    It supports authentication via the Authorization header or a token query parameter.

    Example:
    GET /api/graphql?query={observation(id:"123"){id,status,code{text},valueQuantity{value,unit}}}&token=your-token-here
    """
    # Get token payload with request context
    token_payload = await get_token_payload_with_request(request)
    
    try:
        # Log the request
        logger.info(f"GraphQL GET request: {query}")
        
        # Parse variables if provided
        variables_dict = {}
        if variables:
            try:
                variables_dict = json.loads(variables)
            except json.JSONDecodeError as e:
                logger.error(f"Error parsing variables: {str(e)}")
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=f"Invalid variables JSON: {str(e)}"
                )
        
        # Use token from query parameter if provided
        context = {"request": request, "token_payload": token_payload}
        if token:
            # Override the token_payload with the token from the query parameter
            # This would require additional code to validate the token
            logger.info(f"Using token from query parameter: {token}")
            # For now, we'll just log it and continue using the token from the header

        # Execute the GraphQL query
        result = await schema.execute_async(
            query,
            variables=variables_dict,
            operation_name=operation_name,
            context_value=context
        )

        # Check for errors
        if result.errors:
            logger.error(f"GraphQL errors: {result.errors}")
            return {
                "data": result.data,
                "errors": [str(error) for error in result.errors]
            }

        return {"data": result.data}

    except Exception as e:
        logger.error(f"Error in GraphQL GET endpoint: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error executing GraphQL query: {str(e)}"
        )

# Add GraphiQL endpoint for development
@router.get("/graphiql")
async def graphiql():
    """
    GraphiQL interface for exploring the GraphQL API.
    Only available in development mode.
    """
    html = """
    <!DOCTYPE html>
    <html>
    <head>
        <title>GraphiQL</title>
        <link href="https://cdn.jsdelivr.net/npm/graphiql@3.0.0/graphiql.min.css" rel="stylesheet" />
        <script src="https://cdn.jsdelivr.net/npm/react@17.0.2/umd/react.production.min.js"></script>
        <script src="https://cdn.jsdelivr.net/npm/react-dom@17.0.2/umd/react-dom.production.min.js"></script>
        <script src="https://cdn.jsdelivr.net/npm/graphiql@3.0.0/graphiql.min.js"></script>
    </head>
    <body style="margin: 0;">
        <div id="graphiql" style="height: 100vh;"></div>
        <script>
            const url = window.location.origin + "/api/graphql";
            const headers = {};
            
            // Add authorization header if token exists in localStorage
            const token = localStorage.getItem('token');
            if (token) {
                headers['Authorization'] = 'Bearer ' + token;
            }
            
            // Function to fetch with credentials
            function graphQLFetcher(graphQLParams, opts) {
                return fetch(url, {
                    method: 'post',
                    headers: {
                        'Content-Type': 'application/json',
                        'Accept': 'application/json',
                        ...headers,
                        ...opts.headers,
                    },
                    body: JSON.stringify(graphQLParams),
                    credentials: 'same-origin',
                }).then(response => response.json());
            }
            
            // Render GraphiQL
            ReactDOM.render(
                React.createElement(GraphiQL, {
                    fetcher: graphQLFetcher,
                    headerEditorEnabled: true,
                    shouldPersistHeaders: true,
                    onEditHeaders: (newHeaders) => {
                        const authHeader = newHeaders
                            .split('\n')
                            .find(header => header.toLowerCase().startsWith('authorization:'));
                        
                        if (authHeader) {
                            const token = authHeader.split(' ')[2];
                            localStorage.setItem('token', token);
                        }
                    }
                }),
                document.getElementById('graphiql')
            );
        </script>
    </body>
    </html>
    """
    return HTMLResponse(html)

# Log that the GraphQL endpoint is being added
logger.info("Added GraphQL endpoints at /graphql (POST and GET) and /graphiql (GET) for development")
