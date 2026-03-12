from typing import List, Dict, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status
from fastapi import FastAPI
from app.core.auth import get_token_payload
from app.models.observation import ObservationCreate, ObservationUpdate, ObservationCategory
from app.services.observation_service import get_observation_service

router = APIRouter()

# Public router for testing (bypasses authentication)
public_router = APIRouter()

# This will store the initialized service
_observation_service = None

async def get_observation_service_instance():
    """Get the observation service instance, initializing it if needed."""
    global _observation_service
    if _observation_service is None:
        _observation_service = await get_observation_service()
    return _observation_service

def init_app(app: FastAPI):
    """Initialize the observation service when the app starts."""
    @app.on_event("startup")
    async def startup_event():
        await get_observation_service_instance()

@router.post("", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_observation(
    observation: ObservationCreate = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new observation."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
        # Convert Pydantic model to dict
        observation_dict = observation.dict(exclude_unset=True)
        
        # Create the observation
        created_observation = await service.create_observation(observation_dict)
        
        return created_observation
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating observation: {str(e)}"
        )

@router.get("/{id}", response_model=Dict[str, Any])
async def get_observation(
    id: str = Path(..., description="Observation ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Get an observation by ID."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
        # Get the observation
        observation = await service.get_observation(id)
        
        if not observation:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Observation with ID {id} not found"
            )
        
        return observation
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error retrieving observation: {str(e)}"
        )

@router.put("/{id}", response_model=Dict[str, Any])
async def update_observation(
    id: str = Path(..., description="Observation ID"),
    observation: ObservationUpdate = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Update an observation."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
        # Convert Pydantic model to dict
        observation_dict = observation.dict(exclude_unset=True)
        
        # Update the observation
        updated_observation = await service.update_observation(id, observation_dict)
        
        if not updated_observation:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Observation with ID {id} not found"
            )
        
        return updated_observation
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error updating observation: {str(e)}"
        )

@router.delete("/{id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_observation(
    id: str = Path(..., description="Observation ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> None:
    """Delete an observation."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
        # Delete the observation
        success = await service.delete_observation(id)
        
        if not success:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Observation with ID {id} not found"
            )
        
        return None
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error deleting observation: {str(e)}"
        )

@router.get("", response_model=List[Dict[str, Any]])
async def search_observations(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    category: Optional[str] = Query(None, description="Observation category"),
    code: Optional[str] = Query(None, description="Observation code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Search for observations."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "category": category,
            "code": code,
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Search for observations
        observations = await service.search_observations(params)
        
        return observations
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching observations: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]])
async def get_patient_observations(
    patient_id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    category: Optional[str] = Query(None, description="Observation category"),
    code: Optional[str] = Query(None, description="Observation code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get observations for a patient."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
        # Build search parameters
        params = {
            "category": category,
            "code": code,
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get observations for the patient
        observations = await service.get_patient_observations(patient_id, params)
        
        return observations
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting patient observations: {str(e)}"
        )

@router.get("/category/{category}", response_model=List[Dict[str, Any]])
async def get_observations_by_category(
    category: str = Path(..., description="Observation category"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    code: Optional[str] = Query(None, description="Observation code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get observations by category."""
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()
        
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
        
        # Get observations by category
        observations = await service.get_observations_by_category(category, params)
        
        return observations
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting observations by category: {str(e)}"
        )

# ========================================
# PUBLIC ENDPOINTS (NO AUTHENTICATION)
# ========================================

@public_router.get("/patient/{patient_id}", response_model=Dict[str, Any])
async def get_patient_observations_public(
    patient_id: str = Path(..., description="Patient ID"),
    category: Optional[str] = Query(None, description="Filter by observation category"),
    limit: Optional[int] = Query(100, description="Maximum number of observations to return"),
    offset: Optional[int] = Query(0, description="Number of observations to skip")
) -> Dict[str, Any]:
    """
    PUBLIC ENDPOINT: Get observations for a patient (bypasses authentication).

    This endpoint is for testing purposes and bypasses authentication.
    Use the regular /api/observations/patient/{patient_id} endpoint for production.
    """
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()

        # Build query parameters
        params = {
            "limit": limit,
            "offset": offset
        }

        if category:
            params["category"] = category

        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}

        # Get observations for patient
        observations = await service.get_observations_by_patient(patient_id, params)

        return {
            "patient_id": patient_id,
            "observations": observations,
            "count": len(observations) if isinstance(observations, list) else 0,
            "note": "This is a public endpoint for testing - bypasses authentication"
        }
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting patient observations: {str(e)}"
        )

@public_router.post("", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_observation_public(
    observation: ObservationCreate = Body(...)
) -> Dict[str, Any]:
    """
    PUBLIC ENDPOINT: Create a new observation (bypasses authentication).

    This endpoint is for testing purposes and bypasses authentication.
    Use the regular /api/observations endpoint for production.
    """
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()

        # Convert Pydantic model to dict
        observation_dict = observation.dict(exclude_unset=True)

        # Create the observation (pass None for token_payload since this is public endpoint)
        created_observation = await service.create_observation(observation_dict, token_payload=None)

        return {
            **created_observation,
            "note": "This observation was created via public endpoint - bypasses authentication"
        }
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating observation: {str(e)}"
        )

@public_router.get("/test/demographics/{patient_id}", response_model=Dict[str, Any])
async def get_patient_demographics_observations_public(
    patient_id: str = Path(..., description="Patient ID")
) -> Dict[str, Any]:
    """
    PUBLIC ENDPOINT: Get weight and height observations for demographics.

    This endpoint specifically returns weight and height observations
    needed for patient demographics enhancement.
    """
    try:
        # Get the observation service instance
        service = await get_observation_service_instance()

        # Get all observations for patient
        all_observations = await service.get_observations_by_patient(patient_id, {"limit": 1000})

        # Filter for weight and height observations
        weight_observations = []
        height_observations = []

        if isinstance(all_observations, list):
            for obs in all_observations:
                if isinstance(obs, dict):
                    # Check if it's a weight observation
                    if is_weight_observation(obs):
                        weight_observations.append(obs)
                    # Check if it's a height observation
                    elif is_height_observation(obs):
                        height_observations.append(obs)

        # Extract latest values
        latest_weight = None
        latest_height = None

        if weight_observations:
            # Get the most recent weight
            latest_weight_obs = max(weight_observations, key=lambda x: x.get('effectiveDateTime', ''))
            latest_weight = extract_observation_value(latest_weight_obs)

        if height_observations:
            # Get the most recent height
            latest_height_obs = max(height_observations, key=lambda x: x.get('effectiveDateTime', ''))
            latest_height = extract_observation_value(latest_height_obs)

        return {
            "patient_id": patient_id,
            "weight": latest_weight,
            "height": latest_height,
            "weight_observations_count": len(weight_observations),
            "height_observations_count": len(height_observations),
            "note": "Demographics observations for Flow 2 testing"
        }
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting demographics observations: {str(e)}"
        )

def is_weight_observation(observation: Dict[str, Any]) -> bool:
    """Check if observation is a weight measurement"""
    code = observation.get('code', {})

    # Check LOINC codes for weight
    if isinstance(code, dict) and 'coding' in code:
        for coding in code['coding']:
            if isinstance(coding, dict):
                loinc_code = coding.get('code', '')
                if loinc_code in ['29463-7', '3141-9']:  # Body weight LOINC codes
                    return True

    # Check display text
    display = code.get('text', '').lower()
    weight_keywords = ['weight', 'body weight', 'mass']
    return any(keyword in display for keyword in weight_keywords)

def is_height_observation(observation: Dict[str, Any]) -> bool:
    """Check if observation is a height measurement"""
    code = observation.get('code', {})

    # Check LOINC codes for height
    if isinstance(code, dict) and 'coding' in code:
        for coding in code['coding']:
            if isinstance(coding, dict):
                loinc_code = coding.get('code', '')
                if loinc_code in ['8302-2', '8306-3']:  # Body height LOINC codes
                    return True

    # Check display text
    display = code.get('text', '').lower()
    height_keywords = ['height', 'body height', 'stature']
    return any(keyword in display for keyword in height_keywords)

def extract_observation_value(observation: Dict[str, Any]) -> Optional[float]:
    """Extract numeric value from observation"""
    try:
        value_quantity = observation.get('valueQuantity', {})
        if isinstance(value_quantity, dict) and 'value' in value_quantity:
            return float(value_quantity['value'])
    except Exception:
        pass

    return None
