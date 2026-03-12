from typing import Dict, List, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status, Request
from app.core.auth import get_token_payload
from app.services.timeline_service import get_timeline_service
from app.models.timeline import PatientTimeline, TimelineFilter, TimelineEvent
import logging
from shared.models import (
    Patient, Observation, Condition, Encounter,
    Medication, MedicationRequest, MedicationAdministration, MedicationStatement
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create router
router = APIRouter()
timeline_service = get_timeline_service()

@router.get("/patients/{patient_id}", response_model=PatientTimeline)
async def get_patient_timeline(
    request: Request,
    patient_id: str = Path(..., description="Patient ID"),
    start_date: Optional[str] = Query(None, description="Filter events after this date (ISO format)"),
    end_date: Optional[str] = Query(None, description="Filter events before this date (ISO format)"),
    event_types: Optional[List[str]] = Query(None, description="Filter by event types (observation, condition, medication, encounter, document)"),
    resource_types: Optional[List[str]] = Query(None, description="Filter by resource types (Observation, Condition, MedicationRequest, etc.)"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> PatientTimeline:
    """
    Get a patient's timeline with optional filtering.

    This endpoint aggregates events from various clinical data sources (observations, conditions,
    medications, encounters, documents) to construct a comprehensive patient timeline.

    The timeline can be filtered by date range, event types, and resource types.
    """
    # Add very visible logging
    print(f"\n\n==== TIMELINE API RECEIVED REQUEST FOR PATIENT {patient_id} ====")
    print(f"Query params: {dict(request.query_params)}")
    print(f"Headers: {dict(request.headers)}")
    print(f"==== END TIMELINE API REQUEST ====\n\n")

    logger.info(f"Timeline API received request for patient {patient_id}")

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

        # Get the patient timeline
        logger.info(f"Calling timeline service to get patient timeline")
        timeline = await timeline_service.get_patient_timeline(patient_id, auth_header, filter_params)
        logger.info(f"Successfully retrieved timeline with {len(timeline.events)} events")
        return timeline
    except Exception as e:
        logger.error(f"Error getting patient timeline: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient timeline: {str(e)}"
        )

@router.post("/patients/{patient_id}/filter", response_model=PatientTimeline)
async def filter_patient_timeline(
    patient_id: str = Path(..., description="Patient ID"),
    filter_params: TimelineFilter = Body(..., description="Timeline filter parameters"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> PatientTimeline:
    """
    Filter a patient's timeline using a JSON body for complex filtering.

    This endpoint allows more complex filtering options than the GET endpoint,
    by accepting a JSON body with filter parameters.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get the filtered patient timeline
        return await timeline_service.get_patient_timeline(patient_id, auth_header, filter_params)
    except Exception as e:
        logger.error(f"Error filtering patient timeline: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error filtering patient timeline: {str(e)}"
        )
