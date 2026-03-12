from fastapi import APIRouter
from app.api.routes import router

api_router = APIRouter()
api_router.include_router(router, tags=["organizations"])

# Add webhooks endpoint
try:
    from app.api.endpoints.webhooks import router as webhooks_router
    api_router.include_router(webhooks_router, prefix="/webhooks", tags=["Webhooks"])
except ImportError:
    pass

# Add federation endpoint
try:
    import strawberry
    from strawberry.fastapi import GraphQLRouter
    from app.graphql.federation_schema import schema

    # Create GraphQL router for federation
    graphql_router = GraphQLRouter(schema)
    
    # Add federation endpoint
    api_router.include_router(graphql_router, prefix="/federation")
    
except ImportError:
    pass
except Exception:
    pass
