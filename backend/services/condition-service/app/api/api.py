from fastapi import APIRouter
from app.api.endpoints import condition, hl7, fhir

api_router = APIRouter()

api_router.include_router(condition.router, prefix="/conditions", tags=["Conditions"])
api_router.include_router(hl7.router, prefix="/hl7", tags=["HL7"])
api_router.include_router(fhir.router, prefix="/fhir", tags=["FHIR"])
