from fastapi import APIRouter
from app.api.endpoints import fhir, hl7, debug

api_router = APIRouter()

api_router.include_router(fhir.router, prefix="/fhir", tags=["FHIR"])
api_router.include_router(hl7.router, prefix="/hl7", tags=["HL7"])
api_router.include_router(debug.router, prefix="/debug", tags=["Debug"])
