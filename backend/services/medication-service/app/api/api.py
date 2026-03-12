from fastapi import APIRouter
from app.api.endpoints import (
    medication, medication_request, medication_administration, medication_statement,
    allergy, clinical_decision_support, hl7, fhir, fhir_medication_request, webhooks,
    workflow_proposals, flow2_medication_safety
)

api_router = APIRouter()

# Regular authenticated endpoints
api_router.include_router(medication.router, prefix="/medications", tags=["Medications"])
api_router.include_router(medication_request.router, prefix="/medication-requests", tags=["Medication Requests"])
api_router.include_router(medication_administration.router, prefix="/medication-administrations", tags=["Medication Administrations"])
api_router.include_router(medication_statement.router, prefix="/medication-statements", tags=["Medication Statements"])
api_router.include_router(allergy.router, prefix="/allergies", tags=["Allergies"])
api_router.include_router(clinical_decision_support.router, prefix="/clinical-decision-support", tags=["Clinical Decision Support"])
api_router.include_router(hl7.router, prefix="/hl7", tags=["HL7"])
api_router.include_router(workflow_proposals.router, prefix="/proposals", tags=["Workflow Proposals"])
api_router.include_router(fhir.router, prefix="/fhir/Medication", tags=["FHIR"])
api_router.include_router(fhir_medication_request.router, prefix="/fhir/MedicationRequest", tags=["FHIR"])
api_router.include_router(webhooks.router, prefix="/webhooks", tags=["Webhooks"])
api_router.include_router(flow2_medication_safety.router, prefix="/flow2/medication-safety", tags=["Flow 2 Medication Safety"])

# Public endpoints (no authentication) for testing
api_router.include_router(allergy.public_router, prefix="/public/allergies", tags=["Public Allergies"])
api_router.include_router(workflow_proposals.public_router, prefix="/public/proposals", tags=["Public Proposals"])
