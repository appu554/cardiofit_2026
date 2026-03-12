from typing import List, Dict, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status
from app.core.auth import get_token_payload
from app.services.observation_service import get_observation_service
from app.models.observation import ObservationCategory

router = APIRouter()
observation_service = get_observation_service()

@router.get("", response_model=List[Dict[str, Any]])
async def get_vital_signs(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    code: Optional[str] = Query(None, description="Vital sign code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get vital signs."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": code,
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get vital signs
        vital_signs = await observation_service.get_observations_by_category(ObservationCategory.VITAL_SIGNS, params)
        
        return vital_signs
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting vital signs: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]])
async def get_patient_vital_signs(
    patient_id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    code: Optional[str] = Query(None, description="Vital sign code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get vital signs for a patient."""
    try:
        # Build search parameters
        params = {
            "code": code,
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get vital signs for the patient
        vital_signs = await observation_service.get_patient_observations_by_category(patient_id, ObservationCategory.VITAL_SIGNS, params)
        
        return vital_signs
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient vital signs: {str(e)}"
        )

@router.get("/blood-pressure", response_model=List[Dict[str, Any]])
async def get_blood_pressure(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get blood pressure observations."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "85354-9",  # LOINC code for blood pressure panel
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get blood pressure observations
        blood_pressure = await observation_service.get_observations_by_category(ObservationCategory.VITAL_SIGNS, params)
        
        return blood_pressure
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting blood pressure observations: {str(e)}"
        )

@router.get("/heart-rate", response_model=List[Dict[str, Any]])
async def get_heart_rate(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get heart rate observations."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "8867-4",  # LOINC code for heart rate
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get heart rate observations
        heart_rate = await observation_service.get_observations_by_category(ObservationCategory.VITAL_SIGNS, params)
        
        return heart_rate
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting heart rate observations: {str(e)}"
        )

@router.get("/respiratory-rate", response_model=List[Dict[str, Any]])
async def get_respiratory_rate(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get respiratory rate observations."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "9279-1",  # LOINC code for respiratory rate
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get respiratory rate observations
        respiratory_rate = await observation_service.get_observations_by_category(ObservationCategory.VITAL_SIGNS, params)
        
        return respiratory_rate
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting respiratory rate observations: {str(e)}"
        )

@router.get("/temperature", response_model=List[Dict[str, Any]])
async def get_temperature(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get temperature observations."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "8310-5",  # LOINC code for body temperature
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get temperature observations
        temperature = await observation_service.get_observations_by_category(ObservationCategory.VITAL_SIGNS, params)
        
        return temperature
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting temperature observations: {str(e)}"
        )

@router.get("/oxygen-saturation", response_model=List[Dict[str, Any]])
async def get_oxygen_saturation(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get oxygen saturation observations."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "2708-6",  # LOINC code for oxygen saturation
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get oxygen saturation observations
        oxygen_saturation = await observation_service.get_observations_by_category(ObservationCategory.VITAL_SIGNS, params)
        
        return oxygen_saturation
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting oxygen saturation observations: {str(e)}"
        )
