from fastapi import APIRouter
from app.api.endpoints import encounter, hl7, fhir, webhooks

api_router = APIRouter()

api_router.include_router(encounter.router, prefix="/encounters", tags=["Encounters"])
api_router.include_router(hl7.router, prefix="/hl7", tags=["HL7"])
api_router.include_router(fhir.router, prefix="/fhir/Encounter", tags=["FHIR"])
api_router.include_router(webhooks.router, prefix="/webhooks", tags=["Webhooks"])
