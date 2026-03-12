from fastapi import FastAPI, Request, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from fastapi.responses import HTMLResponse, FileResponse
from strawberry.fastapi import GraphQLRouter
from contextlib import asynccontextmanager
import logging
import os
from app.graphql.schema import schema
from app.config import settings
from app.auth import AuthenticationMiddleware, HeaderAuthMiddleware
from app.middleware import RBACMiddleware, RequestLoggingMiddleware, RateLimitMiddleware, GraphQLRBACMiddleware
from app.api.proxy import router as proxy_router
from app.api.endpoints.raw_fhir import router as raw_fhir_router
from app.api.endpoints.raw_graphql import router as raw_graphql_router
from app.api.endpoints.direct_fhir import router as direct_fhir_router

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup logic
    logger.info("Starting API Gateway")
    logger.info(f"Auth Service URL: {settings.AUTH_SERVICE_URL}")
    logger.info(f"FHIR Service URL: {settings.FHIR_SERVICE_URL}")
    logger.info(f"Patient Service URL: {settings.PATIENT_SERVICE_URL}")
    logger.info(f"Observation Service URL: {settings.OBSERVATION_SERVICE_URL}")
    logger.info(f"Medication Service URL: {settings.MEDICATION_SERVICE_URL}")
    logger.info(f"Condition Service URL: {settings.CONDITION_SERVICE_URL}")
    logger.info(f"Encounter Service URL: {settings.ENCOUNTER_SERVICE_URL}")
    logger.info(f"Timeline Service URL: {settings.TIMELINE_SERVICE_URL}")
    yield
    # Shutdown logic
    logger.info("Shutting down API Gateway")

# Initialize FastAPI app
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="""
    # Clinical Synthesis Hub API Gateway

    This API Gateway provides a centralized entry point for all microservices in the Clinical Synthesis Hub.

    ## Features

    - **Centralized Authentication**: All requests are authenticated through the Auth Service
    - **Role-Based Access Control (RBAC)**: Permissions are enforced based on user roles
    - **Request Routing**: Requests are routed to the appropriate microservice
    - **GraphQL Support**: GraphQL queries are supported for complex data requirements

    ## Authentication

    All API endpoints require authentication using a JWT token in the Authorization header:

    ```
    Authorization: Bearer your-jwt-token
    ```

    To obtain a token, use the `/api/auth/login` or `/api/auth/token` endpoints.
    """,
    version="1.0.0",
    lifespan=lifespan,
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Replace with specific origins in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add rate limiting middleware if enabled
if settings.RATE_LIMIT_ENABLED:
    app.add_middleware(
        RateLimitMiddleware,
        requests_limit=settings.RATE_LIMIT_REQUESTS,
        window_size=settings.RATE_LIMIT_WINDOW,
        exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/graphql"]
    )
    logger.info(f"Added RateLimitMiddleware with limit: {settings.RATE_LIMIT_REQUESTS} requests per {settings.RATE_LIMIT_WINDOW} seconds")

# Add request logging middleware if enabled
if settings.ENABLE_REQUEST_LOGGING:
    app.add_middleware(
        RequestLoggingMiddleware,
        log_request_body=settings.LOG_REQUEST_BODY,
        log_response_body=settings.LOG_RESPONSE_BODY
    )
    logger.info("Added RequestLoggingMiddleware")

# Add authentication middleware
app.add_middleware(
    AuthenticationMiddleware,
    auth_service_url=settings.AUTH_SERVICE_URL,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/graphql", "/graphql-explorer", "/graphql-explorer-v2", "/static", "/favicon.ico", "/api/auth/login", "/api/auth/token", "/api/auth/authorize", "/api/auth/callback", "/api/auth/verify", "/api/graphql/playground", "/api/graphql/explorer", "/api/graphql/explorer/schema", "/api/graphql/explorer/query-builder"]
)
logger.info(f"Added AuthenticationMiddleware with Auth Service URL: {settings.AUTH_SERVICE_URL}")

# Add RBAC middleware
app.add_middleware(
    RBACMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/graphql", "/graphql-explorer", "/graphql-explorer-v2", "/static", "/favicon.ico", "/api/auth/login", "/api/auth/token", "/api/auth/authorize", "/api/auth/callback", "/api/auth/verify", "/api/graphql", "/api/graphql/playground", "/api/graphql/explorer", "/api/graphql/explorer/schema", "/api/graphql/explorer/query-builder"]
)
logger.info("Added RBACMiddleware")

# Add GraphQL RBAC middleware
app.add_middleware(GraphQLRBACMiddleware)
logger.info("Added GraphQLRBACMiddleware")

# Create GraphQL router with context
async def get_context(request: Request):
    # Include user information from request state if available
    context = {"request": request}

    if hasattr(request.state, 'user'):
        context["user"] = request.state.user
        context["user_role"] = getattr(request.state, 'user_role', None)
        context["user_roles"] = getattr(request.state, 'user_roles', [])
        context["user_permissions"] = getattr(request.state, 'user_permissions', [])

        logger.debug(f"GraphQL context includes user: {context['user'].get('id')} with role: {context['user_role']}")

    return context

graphql_app = GraphQLRouter(
    schema,
    context_getter=get_context,
    graphiql=True  # Enable GraphiQL interface
)

# Health check endpoint
@app.get("/health")
async def health_check():
    return {
        "status": "ok",
        "services": {
            "auth": "up",
            "graphql": "up",
            "proxy": "up"
        }
    }

# Root endpoint
@app.get("/")
async def root():
    return {
        "message": "Welcome to the Clinical Synthesis Hub API Gateway",
        "documentation": "/docs",
        "graphql": "/graphql",
        "graphql_explorer": "/graphql-explorer",
        "graphql_explorer_v2": "/graphql-explorer-v2",
        "graphql_schema_explorer": "/graphql-schema-explorer",
        "graphql_query_builder": "/graphql-query-builder",
        "health": "/health"
    }

# Favicon endpoint
@app.get("/favicon.ico", include_in_schema=False)
async def favicon():
    return {"status": "ok"}

# Mount static files first
try:
    app.mount("/static", StaticFiles(directory="app/static"), name="static")
    logger.info("Mounted static files directory")
except Exception as e:
    logger.error(f"Error mounting static files: {str(e)}")

# GraphQL Explorer endpoint
@app.get("/graphql-explorer", response_class=HTMLResponse)
async def graphql_explorer(token: str = None):
    try:
        with open("app/static/graphql-explorer.html", "r") as f:
            content = f.read()

        # If token is provided in the URL, pass it to the GraphQL Explorer
        if token:
            # Add the token to the URL as a query parameter
            if "?token=" not in content and 'id="token-input"' in content:
                # Log that we're adding the token
                logger.info(f"Adding token to GraphQL Explorer: {token[:10]}...")

                # Update the content to include the token
                content = content.replace(
                    '<input type="text" id="token-input" placeholder="Enter JWT token" />',
                    f'<input type="text" id="token-input" placeholder="Enter JWT token" value="{token}" />'
                )

        return HTMLResponse(content=content)
    except Exception as e:
        logger.error(f"Error serving GraphQL Explorer: {str(e)}")
        raise HTTPException(status_code=500, detail="Error serving GraphQL Explorer")

# GraphQL Explorer V2 endpoint
@app.get("/graphql-explorer-v2", response_class=HTMLResponse)
async def graphql_explorer_v2(token: str = None):
    try:
        with open("app/static/graphql-explorer-v2.html", "r") as f:
            content = f.read()

        # If token is provided in the URL, pass it to the GraphQL Explorer
        if token:
            # Add the token to the URL as a query parameter
            if "?token=" not in content and 'id="token-input"' in content:
                # Log that we're adding the token
                logger.info(f"Adding token to GraphQL Explorer V2: {token[:10]}...")

                # Update the content to include the token
                content = content.replace(
                    '<input type="text" id="token-input" placeholder="Enter JWT token" />',
                    f'<input type="text" id="token-input" placeholder="Enter JWT token" value="{token}" />'
                )

        return HTMLResponse(content=content)
    except Exception as e:
        logger.error(f"Error serving GraphQL Explorer V2: {str(e)}")
        raise HTTPException(status_code=500, detail="Error serving GraphQL Explorer V2")

# GraphQL Schema Explorer endpoint
@app.get("/graphql-schema-explorer", response_class=HTMLResponse)
async def graphql_schema_explorer(token: str = None):
    try:
        with open("app/static/graphql-schema-explorer.html", "r") as f:
            content = f.read()

        # If token is provided in the URL, pass it to the GraphQL Schema Explorer
        if token:
            # Add the token to the URL as a query parameter
            if "?token=" not in content and 'id="token-input"' in content:
                # Log that we're adding the token
                logger.info(f"Adding token to GraphQL Schema Explorer: {token[:10]}...")

                # Update the content to include the token
                content = content.replace(
                    '<input type="text" id="token-input" placeholder="Enter JWT token" />',
                    f'<input type="text" id="token-input" placeholder="Enter JWT token" value="{token}" />'
                )

        return HTMLResponse(content=content)
    except Exception as e:
        logger.error(f"Error serving GraphQL Schema Explorer: {str(e)}")
        raise HTTPException(status_code=500, detail="Error serving GraphQL Schema Explorer")

# GraphQL Query Builder endpoint
@app.get("/graphql-query-builder", response_class=HTMLResponse)
async def graphql_query_builder(token: str = None):
    try:
        with open("app/static/graphql-query-builder.html", "r") as f:
            content = f.read()

        # If token is provided in the URL, pass it to the GraphQL Query Builder
        if token:
            # Add the token to the URL as a query parameter
            if "?token=" not in content and 'id="token-input"' in content:
                # Log that we're adding the token
                logger.info(f"Adding token to GraphQL Query Builder: {token[:10]}...")

                # Update the content to include the token
                content = content.replace(
                    '<input type="text" id="token-input" class="form-control" placeholder="Enter JWT token" />',
                    f'<input type="text" id="token-input" class="form-control" placeholder="Enter JWT token" value="{token}" />'
                )

        return HTMLResponse(content=content)
    except Exception as e:
        logger.error(f"Error serving GraphQL Query Builder: {str(e)}")
        raise HTTPException(status_code=500, detail="Error serving GraphQL Query Builder")

# Mount GraphQL endpoint
app.include_router(graphql_app, prefix="/graphql")

# Mount raw FHIR router for direct FHIR operations without schema validation
app.include_router(raw_fhir_router, prefix="/api")

# Mount raw GraphQL router for direct GraphQL operations without schema validation
app.include_router(raw_graphql_router, prefix="/api")

# Mount direct FHIR router for direct FHIR operations without validation
app.include_router(direct_fhir_router, prefix="/api")

# Mount proxy router for API endpoints last
app.include_router(proxy_router)