"""
Enhanced Workflow Proposals API for Calculate > Validate > Commit integration.

This module provides enhanced endpoints that integrate with the strategic orchestrator
and workflow engine for the complete medication proposal lifecycle.
"""
import logging
import uuid
import asyncio
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, Optional, List
from fastapi import APIRouter, HTTPException, Depends, Body, Path, status
from pydantic import BaseModel, Field

from app.core.auth import get_auth_payload
from app.services.medication_service import get_medication_service
from app.services.google_fhir_service import google_fhir_service

logger = logging.getLogger(__name__)

router = APIRouter()

# Enhanced proposal storage with database integration ready
active_proposals: Dict[str, Dict[str, Any]] = {}
proposal_history: List[Dict[str, Any]] = []


class EnhancedMedicationCommitRequest(BaseModel):
    """Enhanced request model for committing medication proposals."""
    validation_id: str = Field(..., description="Safety Gateway validation ID")
    validation_verdict: str = Field(..., description="Safety validation verdict (SAFE, WARNING, UNSAFE)")
    selected_proposal: Dict[str, Any] = Field(..., description="Selected proposal to commit")
    provider_decision: Dict[str, Any] = Field(..., description="Provider decision context")
    override_token: Optional[str] = Field(None, description="Override token if validation was WARNING/UNSAFE")
    commit_notes: Optional[str] = Field(None, description="Additional commit notes")
    audit_context: Optional[Dict[str, Any]] = Field(None, description="Audit context information")


class CommitResponse(BaseModel):
    """Response model for commit operations."""
    medication_order_id: str
    persistence_status: str
    event_publication_status: str
    audit_trail_id: str
    result: str  # SUCCESS, WARNING, FAILURE


@router.post("/api/v1/medication/commit", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def enhanced_commit_medication_proposal(
    request: EnhancedMedicationCommitRequest = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Enhanced commit endpoint for medication proposals.
    
    This endpoint is called by the strategic orchestrator during the COMMIT phase
    of the Calculate > Validate > Commit workflow. It includes comprehensive
    validation verification, audit trail creation, and event publication.
    """
    start_time = datetime.now(timezone.utc)
    proposal_id = request.selected_proposal.get("proposal_id", "unknown")
    
    try:
        logger.info(f"Enhanced commit starting for proposal {proposal_id}",
                   extra={
                       "validation_id": request.validation_id,
                       "verdict": request.validation_verdict,
                       "provider_decision": request.provider_decision
                   })
        
        # 1. VALIDATION VERIFICATION
        validation_check = await _verify_safety_validation(
            request.validation_id, 
            request.validation_verdict,
            request.override_token
        )
        
        if not validation_check["valid"]:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Validation verification failed: {validation_check['reason']}"
            )
        
        # 2. PROPOSAL VALIDATION
        proposal_check = await _validate_proposal_for_commit(request.selected_proposal)
        if not proposal_check["valid"]:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Proposal validation failed: {proposal_check['reason']}"
            )
        
        # 3. CREATE AUDIT TRAIL
        audit_trail_id = await _create_audit_trail(
            proposal_id=proposal_id,
            validation_id=request.validation_id,
            provider_decision=request.provider_decision,
            commit_context=request.audit_context or {},
            user_id=auth_payload.get("user_id", "system"),
            start_time=start_time
        )
        
        # 4. CONVERT PROPOSAL TO FHIR MEDICATIONREQUEST
        medication_request_data = await _convert_proposal_to_fhir_medication_request(
            proposal=request.selected_proposal,
            validation_context={
                "validation_id": request.validation_id,
                "verdict": request.validation_verdict,
                "override_token": request.override_token
            },
            provider_decision=request.provider_decision,
            commit_notes=request.commit_notes,
            audit_trail_id=audit_trail_id
        )
        
        # 5. PERSIST TO FHIR STORE
        auth_header = f"Bearer {auth_payload.get('token', 'system_token')}"
        
        try:
            medication_service = get_medication_service()
            fhir_result = await medication_service.create_medication_request(
                medication_request_data, auth_header
            )
            
            persistence_status = "SUCCESS"
            medication_order_id = fhir_result.get("id", f"med_order_{uuid.uuid4().hex[:12]}")
            
            logger.info(f"FHIR MedicationRequest created successfully: {medication_order_id}")
            
        except Exception as fhir_error:
            logger.error(f"FHIR persistence failed: {fhir_error}")
            # In production, implement retry logic and transaction rollback
            persistence_status = "FAILED"
            medication_order_id = f"failed_{uuid.uuid4().hex[:12]}"
        
        # 6. PUBLISH EVENTS
        event_publication_status = await _publish_medication_events(
            medication_order_id=medication_order_id,
            proposal=request.selected_proposal,
            validation_context={
                "validation_id": request.validation_id,
                "verdict": request.validation_verdict
            },
            provider_decision=request.provider_decision,
            audit_trail_id=audit_trail_id
        )
        
        # 7. UPDATE PROPOSAL STATUS
        await _update_proposal_status(
            proposal_id=proposal_id,
            status="committed",
            medication_order_id=medication_order_id,
            committed_by=auth_payload.get("user_id", "system"),
            audit_trail_id=audit_trail_id
        )
        
        # 8. CALCULATE PROCESSING TIME
        processing_time_ms = int((datetime.now(timezone.utc) - start_time).total_seconds() * 1000)
        
        # 9. DETERMINE OVERALL RESULT
        if persistence_status == "SUCCESS" and event_publication_status in ["SUCCESS", "PARTIAL"]:
            overall_result = "SUCCESS"
        elif persistence_status == "SUCCESS":
            overall_result = "WARNING"  # Persisted but events failed
        else:
            overall_result = "FAILURE"
        
        logger.info(f"Enhanced commit completed for {proposal_id}",
                   extra={
                       "result": overall_result,
                       "medication_order_id": medication_order_id,
                       "processing_time_ms": processing_time_ms,
                       "audit_trail_id": audit_trail_id
                   })
        
        # 10. RETURN STRATEGIC ORCHESTRATOR COMPATIBLE RESPONSE
        return {
            "medication_order_id": medication_order_id,
            "persistence_status": persistence_status,
            "event_publication_status": event_publication_status,
            "audit_trail_id": audit_trail_id,
            "result": overall_result,
            "processing_time_ms": processing_time_ms,
            "validation_verification": validation_check,
            "fhir_result": fhir_result if persistence_status == "SUCCESS" else None,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "metadata": {
                "proposal_id": proposal_id,
                "validation_id": request.validation_id,
                "committed_by": auth_payload.get("user_id", "system"),
                "workflow_phase": "COMMIT"
            }
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Enhanced commit failed for proposal {proposal_id}: {e}", exc_info=True)
        
        # Create failure audit entry
        try:
            await _create_failure_audit_entry(
                proposal_id=proposal_id,
                error=str(e),
                user_id=auth_payload.get("user_id", "system"),
                start_time=start_time
            )
        except Exception as audit_error:
            logger.error(f"Failed to create failure audit entry: {audit_error}")
        
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Enhanced commit failed: {str(e)}"
        )


# Helper Functions

async def _verify_safety_validation(
    validation_id: str, 
    verdict: str, 
    override_token: Optional[str] = None
) -> Dict[str, Any]:
    """
    Verify that the safety validation is still valid and matches expected criteria.
    
    In production, this would:
    1. Check validation ID exists in Safety Gateway
    2. Verify validation hasn't expired
    3. Confirm verdict matches
    4. Validate override tokens if provided
    """
    # Mock implementation - in production, call Safety Gateway HTTP API
    if not validation_id or not validation_id.startswith("validation_"):
        return {"valid": False, "reason": "Invalid validation ID format"}
    
    if verdict not in ["SAFE", "WARNING", "UNSAFE"]:
        return {"valid": False, "reason": f"Invalid verdict: {verdict}"}
    
    if verdict in ["WARNING", "UNSAFE"] and not override_token:
        return {"valid": False, "reason": f"Override token required for verdict: {verdict}"}
    
    # In production, verify the override token with Safety Gateway
    if override_token:
        logger.info(f"Verifying override token: {override_token}")
    
    return {
        "valid": True, 
        "reason": "Validation verified",
        "validation_age_seconds": 30,  # Mock value
        "override_verified": bool(override_token)
    }


async def _validate_proposal_for_commit(proposal: Dict[str, Any]) -> Dict[str, Any]:
    """Validate that the proposal is ready for commit."""
    required_fields = ["proposal_id", "medication", "patient_id", "prescriber"]
    
    for field in required_fields:
        if field not in proposal:
            return {"valid": False, "reason": f"Missing required field: {field}"}
    
    # Check proposal isn't already committed
    proposal_id = proposal.get("proposal_id")
    if proposal_id in active_proposals:
        existing_proposal = active_proposals[proposal_id]
        if existing_proposal.get("status") == "committed":
            return {"valid": False, "reason": "Proposal already committed"}
    
    return {"valid": True, "reason": "Proposal valid for commit"}


async def _create_audit_trail(
    proposal_id: str,
    validation_id: str,
    provider_decision: Dict[str, Any],
    commit_context: Dict[str, Any],
    user_id: str,
    start_time: datetime
) -> str:
    """Create comprehensive audit trail for the commit operation."""
    audit_trail_id = f"audit_{uuid.uuid4().hex[:16]}"
    
    audit_entry = {
        "audit_trail_id": audit_trail_id,
        "proposal_id": proposal_id,
        "validation_id": validation_id,
        "operation": "MEDICATION_COMMIT",
        "user_id": user_id,
        "timestamp": start_time.isoformat(),
        "provider_decision": provider_decision,
        "commit_context": commit_context,
        "workflow_phase": "COMMIT",
        "status": "IN_PROGRESS"
    }
    
    # In production, persist to audit database
    proposal_history.append(audit_entry)
    
    logger.info(f"Audit trail created: {audit_trail_id}")
    return audit_trail_id


async def _convert_proposal_to_fhir_medication_request(
    proposal: Dict[str, Any],
    validation_context: Dict[str, Any],
    provider_decision: Dict[str, Any],
    commit_notes: Optional[str],
    audit_trail_id: str
) -> Dict[str, Any]:
    """Convert proposal to FHIR MedicationRequest with enhanced metadata."""
    medication = proposal["medication"]
    
    # Base FHIR MedicationRequest structure
    fhir_request = {
        "resourceType": "MedicationRequest",
        "id": f"med_req_{uuid.uuid4().hex[:12]}",
        "status": "active",
        "intent": "order",
        "category": [{
            "coding": [{
                "system": "http://terminology.hl7.org/CodeSystem/medicationrequest-category",
                "code": "outpatient",
                "display": "Outpatient"
            }]
        }],
        "medicationCodeableConcept": {
            "coding": [{
                "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                "code": medication.get("code", "unknown"),
                "display": medication.get("name", "Unknown Medication")
            }],
            "text": medication.get("name", "Unknown Medication")
        },
        "subject": {
            "reference": f"Patient/{proposal['patient_id']}"
        },
        "requester": {
            "reference": f"Practitioner/{proposal['prescriber']['provider_id']}"
        },
        "dosageInstruction": [{
            "text": f"{medication.get('dosage', 'As directed')} {medication.get('frequency', '')}".strip(),
            "route": {
                "coding": [{
                    "system": "http://snomed.info/sct",
                    "code": "26643006" if medication.get("route") == "oral" else "unknown",
                    "display": medication.get("route", "unknown")
                }]
            }
        }],
        "note": []
    }
    
    # Add workflow metadata as notes
    workflow_notes = [
        {
            "text": f"Generated via Calculate > Validate > Commit workflow"
        },
        {
            "text": f"Safety validation: {validation_context['verdict']} (ID: {validation_context['validation_id']})"
        },
        {
            "text": f"Audit trail: {audit_trail_id}"
        }
    ]
    
    if commit_notes:
        workflow_notes.append({"text": f"Commit notes: {commit_notes}"})
    
    if validation_context.get("override_token"):
        workflow_notes.append({
            "text": f"Safety override applied: {validation_context['override_token']}"
        })
    
    fhir_request["note"] = workflow_notes
    
    # Add extensions for enhanced metadata
    fhir_request["extension"] = [
        {
            "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/workflow-metadata",
            "extension": [
                {
                    "url": "proposal_id",
                    "valueString": proposal.get("proposal_id", "unknown")
                },
                {
                    "url": "validation_id", 
                    "valueString": validation_context["validation_id"]
                },
                {
                    "url": "audit_trail_id",
                    "valueString": audit_trail_id
                },
                {
                    "url": "workflow_phase",
                    "valueString": "COMMIT"
                }
            ]
        }
    ]
    
    return fhir_request


async def _publish_medication_events(
    medication_order_id: str,
    proposal: Dict[str, Any],
    validation_context: Dict[str, Any],
    provider_decision: Dict[str, Any],
    audit_trail_id: str
) -> str:
    """Publish medication events for downstream systems."""
    try:
        # Event payload for medication ordering
        event_payload = {
            "event_type": "MEDICATION_ORDERED",
            "medication_order_id": medication_order_id,
            "proposal_id": proposal.get("proposal_id"),
            "patient_id": proposal.get("patient_id"),
            "provider_id": proposal.get("prescriber", {}).get("provider_id"),
            "medication_code": proposal.get("medication", {}).get("code"),
            "medication_name": proposal.get("medication", {}).get("name"),
            "validation_context": validation_context,
            "provider_decision": provider_decision,
            "audit_trail_id": audit_trail_id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": "medication_service_enhanced_commit"
        }
        
        # In production, publish to Kafka, event store, or notification service
        logger.info(f"Publishing medication event for order {medication_order_id}", 
                   extra=event_payload)
        
        # Mock successful publication
        await asyncio.sleep(0.01)  # Simulate async operation
        return "SUCCESS"
        
    except Exception as e:
        logger.error(f"Event publication failed for order {medication_order_id}: {e}")
        return "FAILED"


async def _update_proposal_status(
    proposal_id: str,
    status: str,
    medication_order_id: str,
    committed_by: str,
    audit_trail_id: str
):
    """Update proposal status after successful commit."""
    if proposal_id in active_proposals:
        active_proposals[proposal_id].update({
            "status": status,
            "committed_at": datetime.now(timezone.utc).isoformat(),
            "committed_by": committed_by,
            "medication_order_id": medication_order_id,
            "audit_trail_id": audit_trail_id
        })
    
    # In production, update database record
    logger.info(f"Proposal {proposal_id} status updated to {status}")


async def _create_failure_audit_entry(
    proposal_id: str,
    error: str,
    user_id: str,
    start_time: datetime
):
    """Create audit entry for failed commit operations."""
    failure_audit = {
        "audit_trail_id": f"failure_audit_{uuid.uuid4().hex[:16]}",
        "proposal_id": proposal_id,
        "operation": "MEDICATION_COMMIT_FAILURE",
        "user_id": user_id,
        "timestamp": start_time.isoformat(),
        "error": error,
        "workflow_phase": "COMMIT",
        "status": "FAILED",
        "failure_time": datetime.now(timezone.utc).isoformat()
    }
    
    proposal_history.append(failure_audit)
    logger.warning(f"Failure audit entry created for proposal {proposal_id}")


# Health and status endpoints

@router.get("/api/v1/proposals/health", response_model=Dict[str, Any])
async def enhanced_proposals_health():
    """Health check for enhanced proposals endpoint."""
    return {
        "status": "healthy",
        "service": "enhanced_workflow_proposals",
        "version": "1.0.0",
        "active_proposals": len(active_proposals),
        "proposal_history": len(proposal_history),
        "capabilities": [
            "enhanced_commit",
            "validation_verification", 
            "audit_trail_creation",
            "event_publication",
            "fhir_integration"
        ],
        "timestamp": datetime.now(timezone.utc).isoformat()
    }


@router.get("/api/v1/proposals/stats", response_model=Dict[str, Any])
async def enhanced_proposals_stats():
    """Get proposal statistics."""
    committed_count = len([p for p in active_proposals.values() if p.get("status") == "committed"])
    pending_count = len([p for p in active_proposals.values() if p.get("status") == "proposed"])
    
    return {
        "total_proposals": len(active_proposals),
        "committed_proposals": committed_count,
        "pending_proposals": pending_count,
        "audit_entries": len(proposal_history),
        "success_rate": committed_count / len(active_proposals) if active_proposals else 0,
        "timestamp": datetime.now(timezone.utc).isoformat()
    }