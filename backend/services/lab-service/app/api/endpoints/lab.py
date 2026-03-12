from typing import Dict, List, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Body, Query, Path, status
from app.core.auth import get_token_payload
from app.services.lab_service import get_lab_service
from app.models.lab import LabTest, LabPanel, LabTestCreate, LabPanelCreate

router = APIRouter()
lab_service = get_lab_service()

@router.post("/tests", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_lab_test(
    lab_test: LabTestCreate = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new lab test."""
    try:
        # Convert Pydantic model to dict
        lab_test_dict = lab_test.dict()
        
        # Create the lab test
        created_test = await lab_service.create_lab_test(lab_test_dict)
        
        return created_test
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating lab test: {str(e)}"
        )

@router.get("/tests/{test_id}", response_model=Dict[str, Any])
async def get_lab_test(
    test_id: str = Path(..., description="Lab test ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Get a lab test by ID."""
    try:
        # Get the lab test
        lab_test = await lab_service.get_lab_test(test_id)
        
        if not lab_test:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Lab test with ID {test_id} not found"
            )
        
        return lab_test
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting lab test: {str(e)}"
        )

@router.get("/tests", response_model=List[Dict[str, Any]])
async def search_lab_tests(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    test_code: Optional[str] = Query(None, description="Test code"),
    status: Optional[str] = Query(None, description="Test status"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Search for lab tests."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "test_code": test_code,
            "status": status,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Search for lab tests
        lab_tests = await lab_service.search_lab_tests(params)
        
        return lab_tests
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error searching lab tests: {str(e)}"
        )

@router.post("/panels", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_lab_panel(
    lab_panel: LabPanelCreate = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new lab panel."""
    try:
        # Convert Pydantic model to dict
        lab_panel_dict = lab_panel.dict()
        
        # Create the lab panel
        created_panel = await lab_service.create_lab_panel(lab_panel_dict)
        
        return created_panel
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating lab panel: {str(e)}"
        )

@router.get("/panels/{panel_id}", response_model=Dict[str, Any])
async def get_lab_panel(
    panel_id: str = Path(..., description="Lab panel ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Get a lab panel by ID."""
    try:
        # Get the lab panel
        lab_panel = await lab_service.get_lab_panel(panel_id)
        
        if not lab_panel:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Lab panel with ID {panel_id} not found"
            )
        
        return lab_panel
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting lab panel: {str(e)}"
        )

@router.get("/panels", response_model=List[Dict[str, Any]])
async def search_lab_panels(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    panel_code: Optional[str] = Query(None, description="Panel code"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Search for lab panels."""
    try:
        # Build search parameters
        params = {
            "patient_id": patient_id,
            "panel_code": panel_code,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Search for lab panels
        lab_panels = await lab_service.search_lab_panels(params)
        
        return lab_panels
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error searching lab panels: {str(e)}"
        )

@router.get("/patient/{patient_id}/tests", response_model=List[Dict[str, Any]])
async def get_patient_lab_tests(
    patient_id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    test_code: Optional[str] = Query(None, description="Test code"),
    status: Optional[str] = Query(None, description="Test status"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get lab tests for a patient."""
    try:
        # Build search parameters
        params = {
            "test_code": test_code,
            "status": status,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get lab tests for the patient
        lab_tests = await lab_service.get_patient_lab_tests(patient_id, params)
        
        return lab_tests
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient lab tests: {str(e)}"
        )

@router.get("/patient/{patient_id}/panels", response_model=List[Dict[str, Any]])
async def get_patient_lab_panels(
    patient_id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    panel_code: Optional[str] = Query(None, description="Panel code"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number")
) -> List[Dict[str, Any]]:
    """Get lab panels for a patient."""
    try:
        # Build search parameters
        params = {
            "panel_code": panel_code,
            "_count": _count,
            "_page": _page
        }
        
        # Remove None values
        params = {k: v for k, v in params.items() if v is not None}
        
        # Get lab panels for the patient
        lab_panels = await lab_service.get_patient_lab_panels(patient_id, params)
        
        return lab_panels
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient lab panels: {str(e)}"
        )
