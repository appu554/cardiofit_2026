from typing import List, Dict, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status
from app.core.auth import get_token_payload
from app.services.observation_service import get_observation_service
from app.models.observation import ObservationCategory

router = APIRouter()
observation_service = get_observation_service()

@router.get("", response_model=List[Dict[str, Any]])
async def get_lab_results(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    code: Optional[str] = Query(None, description="Lab test code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get laboratory results."""
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
        
        # Get laboratory results
        lab_results = await observation_service.get_observations_by_category(ObservationCategory.LABORATORY, params)
        
        return lab_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting laboratory results: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]])
async def get_patient_lab_results(
    patient_id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    code: Optional[str] = Query(None, description="Lab test code"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get laboratory results for a patient."""
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
        
        # Get laboratory results for the patient
        lab_results = await observation_service.get_patient_observations_by_category(patient_id, ObservationCategory.LABORATORY, params)
        
        return lab_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient laboratory results: {str(e)}"
        )

@router.get("/cbc", response_model=List[Dict[str, Any]])
async def get_cbc_results(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get CBC (Complete Blood Count) results."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "58410-2",  # LOINC code for CBC panel
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get CBC results
        cbc_results = await observation_service.get_observations_by_category(ObservationCategory.LABORATORY, params)
        
        return cbc_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting CBC results: {str(e)}"
        )

@router.get("/bmp", response_model=List[Dict[str, Any]])
async def get_bmp_results(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get BMP (Basic Metabolic Panel) results."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "51990-0",  # LOINC code for BMP panel
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get BMP results
        bmp_results = await observation_service.get_observations_by_category(ObservationCategory.LABORATORY, params)
        
        return bmp_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting BMP results: {str(e)}"
        )

@router.get("/cmp", response_model=List[Dict[str, Any]])
async def get_cmp_results(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get CMP (Comprehensive Metabolic Panel) results."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "24323-8",  # LOINC code for CMP panel
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get CMP results
        cmp_results = await observation_service.get_observations_by_category(ObservationCategory.LABORATORY, params)
        
        return cmp_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting CMP results: {str(e)}"
        )

@router.get("/lipid-panel", response_model=List[Dict[str, Any]])
async def get_lipid_panel_results(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get Lipid Panel results."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "57698-3",  # LOINC code for Lipid Panel
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get Lipid Panel results
        lipid_panel_results = await observation_service.get_observations_by_category(ObservationCategory.LABORATORY, params)
        
        return lipid_panel_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting Lipid Panel results: {str(e)}"
        )

@router.get("/urinalysis", response_model=List[Dict[str, Any]])
async def get_urinalysis_results(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get Urinalysis results."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "code": "24356-8",  # LOINC code for Urinalysis
            "date": date,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get Urinalysis results
        urinalysis_results = await observation_service.get_observations_by_category(ObservationCategory.LABORATORY, params)
        
        return urinalysis_results
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting Urinalysis results: {str(e)}"
        )
