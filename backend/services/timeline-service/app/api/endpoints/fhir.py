from typing import Dict, List, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status, Request
from app.core.auth import get_token_payload
from app.services.timeline_service import get_timeline_service
from app.models.timeline import PatientTimeline, TimelineFilter
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

router = APIRouter()

@router.get("/Patient/{patient_id}/timeline", response_model=Dict[str, Any])
async def get_patient_timeline_fhir(
    request: Request,
    patient_id: str = Path(..., description="Patient ID"),
    start_date: Optional[str] = Query(None, description="Filter events after this date (ISO format)"),
    end_date: Optional[str] = Query(None, description="Filter events before this date (ISO format)"),
    event_types: Optional[List[str]] = Query(None, description="Filter by event types"),
    resource_types: Optional[List[str]] = Query(None, description="Filter by resource types"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Get a patient's timeline in FHIR format.
    This endpoint is called by the FHIR service.
    """
    # Add very visible logging
    print(f"\n\n==== TIMELINE SERVICE RECEIVED FHIR REQUEST FOR PATIENT {patient_id} ====")
    print(f"Query params: {dict(request.query_params)}")
    print(f"Headers: {dict(request.headers)}")
    print(f"==== END TIMELINE SERVICE FHIR REQUEST ====\n\n")
    
    logger.info(f"Timeline FHIR endpoint received request for patient {patient_id}")
    
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"
        logger.info(f"Using token from payload for authorization")

        # Create filter parameters if any filters are provided
        filter_params = None
        if any([start_date, end_date, event_types, resource_types]):
            logger.info(f"Creating filter parameters with: start_date={start_date}, end_date={end_date}, event_types={event_types}, resource_types={resource_types}")
            
            filter_params = TimelineFilter(
                start_date=start_date,
                end_date=end_date,
                event_types=event_types,
                resource_types=resource_types
            )

        # Get the timeline service
        timeline_service = get_timeline_service()
        logger.info(f"Retrieved timeline service instance")
        
        # Get the patient timeline
        logger.info(f"Calling timeline service to get patient timeline")
        timeline = await timeline_service.get_patient_timeline(patient_id, auth_header, filter_params)
        logger.info(f"Successfully retrieved timeline with {len(timeline.events)} events")
        
        # Convert to FHIR format
        return {
            "resourceType": "Bundle",
            "type": "collection",
            "id": f"timeline-{patient_id}",
            "meta": {
                "lastUpdated": timeline.last_updated.isoformat() if timeline.last_updated else None
            },
            "entry": [event.dict() for event in timeline.events]
        }
    except Exception as e:
        logger.error(f"Error getting patient timeline: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient timeline: {str(e)}"
        )
