"""
Workflow Proposals API endpoints for Calculate → Validate → Commit flow.
"""
import logging
import uuid
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, Optional
from fastapi import APIRouter, HTTPException, Depends, Body, Path, status
from pydantic import BaseModel, Field

from app.core.auth import get_auth_payload
from app.services.medication_service import get_medication_service
from app.services.google_fhir_service import google_fhir_service

logger = logging.getLogger(__name__)

router = APIRouter()

# Public router for testing (bypasses authentication)
public_router = APIRouter()

# In-memory proposal storage (in production, use database)
active_proposals: Dict[str, Dict[str, Any]] = {}


class MedicationProposalRequest(BaseModel):
    """Request model for creating medication proposals."""
    patient_id: str = Field(..., description="Patient ID")
    medication_code: str = Field(..., description="Medication code (RxNorm)")
    medication_name: str = Field(..., description="Medication name")
    dosage: str = Field(..., description="Dosage instructions")
    frequency: str = Field(..., description="Frequency (e.g., 'twice daily')")
    duration: Optional[str] = Field(None, description="Duration (e.g., '7 days')")
    route: Optional[str] = Field("oral", description="Route of administration")
    priority: Optional[str] = Field("routine", description="Priority level")
    indication: Optional[str] = Field(None, description="Indication for medication")
    provider_id: str = Field(..., description="Prescribing provider ID")
    encounter_id: Optional[str] = Field(None, description="Associated encounter ID")
    notes: Optional[str] = Field(None, description="Additional notes")


class MedicationCommitRequest(BaseModel):
    """Request model for committing medication proposals."""
    safety_validation: Dict[str, Any] = Field(..., description="Safety validation results")
    commit_notes: Optional[str] = Field(None, description="Commit notes")


class ProposalResponse(BaseModel):
    """Response model for proposal operations."""
    proposal_id: str
    status: str
    created_at: datetime
    proposal_data: Dict[str, Any]


@router.post("/medication", response_model=ProposalResponse, status_code=status.HTTP_201_CREATED)
async def create_medication_proposal(
    request: MedicationProposalRequest = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> ProposalResponse:
    """
    Create a medication proposal (CALCULATE phase).
    
    This endpoint generates a medication proposal without committing it to the FHIR store.
    The proposal includes all necessary data for safety validation.
    """
    try:
        logger.info(f"Creating medication proposal for patient {request.patient_id}")
        
        # Generate unique proposal ID
        proposal_id = f"med_proposal_{uuid.uuid4().hex[:12]}"
        
        # Get authorization header
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"
        
        # Fetch patient data for context
        patient_data = await _get_patient_data(request.patient_id, auth_header)
        
        # Create proposal data structure
        proposal_data = {
            "proposal_id": proposal_id,
            "proposal_type": "medication_prescription",
            "status": "proposed",
            "patient_id": request.patient_id,
            "patient_data": patient_data,
            "medication": {
                "code": request.medication_code,
                "name": request.medication_name,
                "dosage": request.dosage,
                "frequency": request.frequency,
                "duration": request.duration,
                "route": request.route,
                "indication": request.indication
            },
            "prescriber": {
                "provider_id": request.provider_id
            },
            "clinical_context": {
                "encounter_id": request.encounter_id,
                "priority": request.priority,
                "notes": request.notes
            },
            "created_at": datetime.now(timezone.utc).isoformat(),
            "created_by": auth_payload.get("user_id", "unknown"),
            "workflow_metadata": {
                "phase": "calculate",
                "requires_safety_validation": True,
                "estimated_cost": await _calculate_medication_cost(request.medication_code),
                "formulary_status": await _check_formulary_status(request.medication_code, patient_data)
            }
        }
        
        # Store proposal in memory (in production, store in database)
        active_proposals[proposal_id] = proposal_data
        
        logger.info(f"Created medication proposal {proposal_id}")
        
        return ProposalResponse(
            proposal_id=proposal_id,
            status="proposed",
            created_at=datetime.now(timezone.utc),
            proposal_data=proposal_data
        )
        
    except Exception as e:
        logger.error(f"Error creating medication proposal: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error creating medication proposal: {str(e)}"
        )


@router.post("/{proposal_id}/commit", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def commit_medication_proposal(
    proposal_id: str = Path(..., description="Proposal ID"),
    request: MedicationCommitRequest = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Commit a medication proposal (COMMIT phase).
    
    This endpoint commits an approved proposal to the FHIR store as a MedicationRequest.
    """
    try:
        logger.info(f"Committing medication proposal {proposal_id}")
        
        # Retrieve proposal
        if proposal_id not in active_proposals:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Proposal {proposal_id} not found"
            )
        
        proposal = active_proposals[proposal_id]
        
        # Check safety validation
        safety_validation = request.safety_validation
        if safety_validation.get("verdict") != "SAFE":
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Cannot commit unsafe proposal. Safety verdict: {safety_validation.get('verdict')}"
            )
        
        # Get authorization header
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"
        
        # Create MedicationRequest from proposal
        medication_request_data = await _convert_proposal_to_medication_request(
            proposal, safety_validation, request.commit_notes
        )
        
        # Commit to FHIR store
        medication_service = get_medication_service()
        fhir_result = await medication_service.create_medication_request(
            medication_request_data, auth_header
        )
        
        # Update proposal status
        proposal["status"] = "committed"
        proposal["committed_at"] = datetime.now(timezone.utc).isoformat()
        proposal["committed_by"] = auth_payload.get("user_id", "unknown")
        proposal["fhir_resource_id"] = fhir_result.get("id")
        proposal["safety_validation"] = safety_validation
        proposal["commit_notes"] = request.commit_notes
        
        logger.info(f"Successfully committed proposal {proposal_id} as FHIR resource {fhir_result.get('id')}")
        
        return {
            "status": "committed",
            "proposal_id": proposal_id,
            "fhir_resource_id": fhir_result.get("id"),
            "committed_at": proposal["committed_at"],
            "safety_validation": safety_validation,
            "fhir_result": fhir_result
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error committing medication proposal {proposal_id}: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error committing proposal: {str(e)}"
        )


@router.get("/{proposal_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def get_proposal(
    proposal_id: str = Path(..., description="Proposal ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """Get a proposal by ID."""
    if proposal_id not in active_proposals:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Proposal {proposal_id} not found"
        )
    
    return active_proposals[proposal_id]


@router.delete("/{proposal_id}", status_code=status.HTTP_204_NO_CONTENT)
async def cancel_proposal(
    proposal_id: str = Path(..., description="Proposal ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
):
    """Cancel a proposal."""
    if proposal_id not in active_proposals:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Proposal {proposal_id} not found"
        )
    
    proposal = active_proposals[proposal_id]
    if proposal["status"] == "committed":
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Cannot cancel committed proposal"
        )
    
    proposal["status"] = "cancelled"
    proposal["cancelled_at"] = datetime.now(timezone.utc).isoformat()
    proposal["cancelled_by"] = auth_payload.get("user_id", "unknown")
    
    logger.info(f"Cancelled proposal {proposal_id}")


# Helper functions

async def _get_patient_data(patient_id: str, auth_header: str) -> Dict[str, Any]:
    """Fetch patient data for proposal context."""
    try:
        # In a real implementation, call the patient service
        # For now, return minimal patient data
        return {
            "patient_id": patient_id,
            "demographics": {
                "age": "unknown",
                "weight": "unknown",
                "allergies": []
            }
        }
    except Exception as e:
        logger.warning(f"Could not fetch patient data for {patient_id}: {e}")
        return {"patient_id": patient_id, "demographics": {}}


async def _calculate_medication_cost(medication_code: str) -> Optional[float]:
    """Calculate estimated medication cost."""
    # In a real implementation, call pricing service
    return None


async def _check_formulary_status(medication_code: str, patient_data: Dict[str, Any]) -> str:
    """Check if medication is on formulary."""
    # In a real implementation, check formulary database
    return "unknown"


async def _convert_proposal_to_medication_request(
    proposal: Dict[str, Any],
    safety_validation: Dict[str, Any],
    commit_notes: Optional[str]
) -> Dict[str, Any]:
    """Convert proposal to MedicationRequest format."""
    medication = proposal["medication"]
    
    return {
        "resourceType": "MedicationRequest",
        "status": "active",
        "intent": "order",
        "medicationCodeableConcept": {
            "coding": [{
                "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                "code": medication["code"],
                "display": medication["name"]
            }]
        },
        "subject": {
            "reference": f"Patient/{proposal['patient_id']}"
        },
        "requester": {
            "reference": f"Practitioner/{proposal['prescriber']['provider_id']}"
        },
        "dosageInstruction": [{
            "text": f"{medication['dosage']} {medication['frequency']}",
            "route": {
                "coding": [{
                    "system": "http://snomed.info/sct",
                    "code": "26643006" if medication["route"] == "oral" else "unknown",
                    "display": medication["route"]
                }]
            }
        }],
        "note": [
            {"text": commit_notes} if commit_notes else None,
            {"text": f"Safety validation: {safety_validation.get('verdict')}"}
        ]
    }

# ========================================
# PUBLIC ENDPOINTS (NO AUTHENTICATION)
# ========================================

@public_router.post("/medication", response_model=ProposalResponse, status_code=status.HTTP_201_CREATED)
async def create_medication_proposal_public(
    request: MedicationProposalRequest = Body(...)
) -> ProposalResponse:
    """
    PUBLIC ENDPOINT: Create a medication proposal (bypasses authentication).

    This endpoint is for testing purposes and bypasses authentication.
    Use the regular /api/proposals/medication endpoint for production.

    This endpoint generates a medication proposal without committing it to the FHIR store.
    The proposal includes all necessary data for safety validation.
    """
    try:
        logger.info(f"PUBLIC: Creating medication proposal for patient {request.patient_id}")

        # Generate unique proposal ID
        proposal_id = f"med_proposal_{uuid.uuid4().hex[:12]}"

        # Use dummy auth for public endpoint
        auth_header = "Bearer public_test_token"

        # Fetch patient data for context
        patient_data = await _get_patient_data(request.patient_id, auth_header)

        # Create proposal data structure
        proposal_data = {
            "proposal_id": proposal_id,
            "proposal_type": "medication_prescription",
            "status": "proposed",
            "patient_id": request.patient_id,
            "patient_data": patient_data,
            "medication": {
                "code": request.medication_code,
                "name": request.medication_name,
                "dosage": request.dosage,
                "frequency": request.frequency,
                "duration": request.duration,
                "route": request.route,
                "indication": request.indication
            },
            "prescriber": {
                "provider_id": request.provider_id
            },
            "clinical_context": {
                "encounter_id": request.encounter_id,
                "priority": request.priority,
                "notes": request.notes
            },
            "created_at": datetime.now(timezone.utc).isoformat(),
            "expires_at": (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat(),
            "source": "public_endpoint",
            "note": "Created via public endpoint - bypasses authentication"
        }

        # Store proposal in cache (1 hour expiry)
        cache_key = f"proposal:{proposal_id}"
        # Note: In a real implementation, you'd use Redis or similar
        # For testing, we'll just return the proposal

        logger.info(f"PUBLIC: Created proposal {proposal_id}")

        return ProposalResponse(
            proposal_id=proposal_id,
            status="success",
            message="Medication proposal created successfully via public endpoint",
            proposal_data=proposal_data,
            created_at=datetime.now(timezone.utc).isoformat()
        )

    except Exception as e:
        logger.error(f"PUBLIC: Error creating medication proposal: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error creating medication proposal: {str(e)}"
        )
