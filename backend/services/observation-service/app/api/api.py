from fastapi import APIRouter
from app.api.endpoints import observation, hl7, vital_signs, laboratory, fhir, graphql, federation

api_router = APIRouter()

# Regular authenticated endpoints
api_router.include_router(observation.router, prefix="/observations", tags=["Observations"])
api_router.include_router(vital_signs.router, prefix="/vital-signs", tags=["Vital Signs"])
api_router.include_router(laboratory.router, prefix="/laboratory", tags=["Laboratory"])
api_router.include_router(hl7.router, prefix="/hl7", tags=["HL7"])
api_router.include_router(fhir.router, prefix="/fhir", tags=["FHIR"])
api_router.include_router(graphql.router, prefix="/graphql", tags=["GraphQL"])
api_router.include_router(federation.router, prefix="/federation", tags=["Federation"])

# Public endpoints (no authentication) for testing
api_router.include_router(observation.public_router, prefix="/public/observations", tags=["Public Observations"])
