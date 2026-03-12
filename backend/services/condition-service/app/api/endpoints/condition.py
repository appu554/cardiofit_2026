from fastapi import APIRouter, Depends, HTTPException, Query
from typing import Dict, List, Any, Optional
from app.models.condition import ConditionCreate, ConditionUpdate
from app.services.condition_service import ConditionService
from app.core.auth import get_token_payload
from shared.models import Condition, CodeableConcept, Coding
import logging

# Constants for condition categories
PROBLEM_LIST_CATEGORY = "problem-list-item"
ENCOUNTER_DIAGNOSIS_CATEGORY = "encounter-diagnosis"
HEALTH_CONCERN_CATEGORY = "health-concern"

logger = logging.getLogger(__name__)

router = APIRouter()
condition_service = ConditionService()

@router.post("/", response_model=Dict[str, Any])
async def create_condition(
    condition: ConditionCreate,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Create a new condition.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"
        return await condition_service.create_condition(condition, authorization)
    except Exception as e:
        logger.error(f"Error creating condition: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/problems", response_model=Dict[str, Any])
async def create_problem(
    condition: ConditionCreate,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Create a new problem list item.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        # Ensure the category is set to problem-list-item
        # First, check if category is already set
        has_problem_category = False
        if condition.category:
            for cat in condition.category:
                if cat.coding:
                    for coding in cat.coding:
                        if isinstance(coding, dict) and coding.get("code") == PROBLEM_LIST_CATEGORY:
                            has_problem_category = True
                            break
                        elif hasattr(coding, "code") and coding.code == PROBLEM_LIST_CATEGORY:
                            has_problem_category = True
                            break
                if has_problem_category:
                    break

        # If not, add the problem-list-item category
        if not has_problem_category:
            if not condition.category:
                condition.category = []

            problem_category = CodeableConcept(
                coding=[
                    Coding(
                        system="http://terminology.hl7.org/CodeSystem/condition-category",
                        code=PROBLEM_LIST_CATEGORY,
                        display="Problem List Item"
                    )
                ],
                text="Problem List Item"
            )
            condition.category.append(problem_category)

        return await condition_service.create_condition(condition, authorization)
    except Exception as e:
        logger.error(f"Error creating problem: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/diagnoses", response_model=Dict[str, Any])
async def create_diagnosis(
    condition: ConditionCreate,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Create a new encounter diagnosis.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        # Ensure the category is set to encounter-diagnosis
        # First, check if category is already set
        has_diagnosis_category = False
        if condition.category:
            for cat in condition.category:
                if cat.coding:
                    for coding in cat.coding:
                        if isinstance(coding, dict) and coding.get("code") == ENCOUNTER_DIAGNOSIS_CATEGORY:
                            has_diagnosis_category = True
                            break
                        elif hasattr(coding, "code") and coding.code == ENCOUNTER_DIAGNOSIS_CATEGORY:
                            has_diagnosis_category = True
                            break
                if has_diagnosis_category:
                    break

        # If not, add the encounter-diagnosis category
        if not has_diagnosis_category:
            if not condition.category:
                condition.category = []

            diagnosis_category = CodeableConcept(
                coding=[
                    Coding(
                        system="http://terminology.hl7.org/CodeSystem/condition-category",
                        code=ENCOUNTER_DIAGNOSIS_CATEGORY,
                        display="Encounter Diagnosis"
                    )
                ],
                text="Encounter Diagnosis"
            )
            condition.category.append(diagnosis_category)

        return await condition_service.create_condition(condition, authorization)
    except Exception as e:
        logger.error(f"Error creating diagnosis: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/health-concerns", response_model=Dict[str, Any])
async def create_health_concern(
    condition: ConditionCreate,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Create a new health concern.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        # Ensure the category is set to health-concern
        # First, check if category is already set
        has_health_concern_category = False
        if condition.category:
            for cat in condition.category:
                if cat.coding:
                    for coding in cat.coding:
                        if isinstance(coding, dict) and coding.get("code") == HEALTH_CONCERN_CATEGORY:
                            has_health_concern_category = True
                            break
                        elif hasattr(coding, "code") and coding.code == HEALTH_CONCERN_CATEGORY:
                            has_health_concern_category = True
                            break
                if has_health_concern_category:
                    break

        # If not, add the health-concern category
        if not has_health_concern_category:
            if not condition.category:
                condition.category = []

            health_concern_category = CodeableConcept(
                coding=[
                    Coding(
                        system="http://terminology.hl7.org/CodeSystem/condition-category",
                        code=HEALTH_CONCERN_CATEGORY,
                        display="Health Concern"
                    )
                ],
                text="Health Concern"
            )
            condition.category.append(health_concern_category)

        return await condition_service.create_condition(condition, authorization)
    except Exception as e:
        logger.error(f"Error creating health concern: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/{condition_id}", response_model=Dict[str, Any])
async def get_condition(
    condition_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Get a condition by ID.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"
        return await condition_service.get_condition(condition_id, authorization)
    except Exception as e:
        logger.error(f"Error getting condition: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/{condition_id}", response_model=Dict[str, Any])
async def update_condition(
    condition_id: str,
    condition: ConditionUpdate,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Update a condition.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"
        return await condition_service.update_condition(condition_id, condition, authorization)
    except Exception as e:
        logger.error(f"Error updating condition: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.delete("/{condition_id}", response_model=Dict[str, Any])
async def delete_condition(
    condition_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Delete a condition.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"
        return await condition_service.delete_condition(condition_id, authorization)
    except Exception as e:
        logger.error(f"Error deleting condition: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/", response_model=List[Dict[str, Any]])
async def search_conditions(
    code: Optional[str] = Query(None, description="Condition code"),
    category: Optional[str] = Query(None, description="Condition category"),
    clinical_status: Optional[str] = Query(None, description="Clinical status"),
    verification_status: Optional[str] = Query(None, description="Verification status"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Search for conditions.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        params = {}
        if code:
            params["code"] = code
        if category:
            params["category"] = category
        if clinical_status:
            params["clinical-status"] = clinical_status
        if verification_status:
            params["verification-status"] = verification_status

        return await condition_service.search_conditions(params, authorization)
    except Exception as e:
        logger.error(f"Error searching conditions: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]])
async def get_patient_conditions(
    patient_id: str,
    code: Optional[str] = Query(None, description="Condition code"),
    category: Optional[str] = Query(None, description="Condition category"),
    clinical_status: Optional[str] = Query(None, description="Clinical status"),
    verification_status: Optional[str] = Query(None, description="Verification status"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Get conditions for a patient.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        params = {}
        if code:
            params["code"] = code
        if category:
            params["category"] = category
        if clinical_status:
            params["clinical-status"] = clinical_status
        if verification_status:
            params["verification-status"] = verification_status

        return await condition_service.get_patient_conditions(patient_id, params, authorization)
    except Exception as e:
        logger.error(f"Error getting patient conditions: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/patient/{patient_id}/problems", response_model=List[Dict[str, Any]])
async def get_patient_problems(
    patient_id: str,
    code: Optional[str] = Query(None, description="Problem code"),
    clinical_status: Optional[str] = Query(None, description="Clinical status"),
    verification_status: Optional[str] = Query(None, description="Verification status"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Get problem list items for a patient.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        params = {
            "category": PROBLEM_LIST_CATEGORY
        }
        if code:
            params["code"] = code
        if clinical_status:
            params["clinical-status"] = clinical_status
        if verification_status:
            params["verification-status"] = verification_status

        return await condition_service.get_patient_conditions(patient_id, params, authorization)
    except Exception as e:
        logger.error(f"Error getting patient problems: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/patient/{patient_id}/diagnoses", response_model=List[Dict[str, Any]])
async def get_patient_diagnoses(
    patient_id: str,
    code: Optional[str] = Query(None, description="Diagnosis code"),
    clinical_status: Optional[str] = Query(None, description="Clinical status"),
    verification_status: Optional[str] = Query(None, description="Verification status"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Get encounter diagnoses for a patient.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        params = {
            "category": ENCOUNTER_DIAGNOSIS_CATEGORY
        }
        if code:
            params["code"] = code
        if clinical_status:
            params["clinical-status"] = clinical_status
        if verification_status:
            params["verification-status"] = verification_status

        return await condition_service.get_patient_conditions(patient_id, params, authorization)
    except Exception as e:
        logger.error(f"Error getting patient diagnoses: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/patient/{patient_id}/health-concerns", response_model=List[Dict[str, Any]])
async def get_patient_health_concerns(
    patient_id: str,
    code: Optional[str] = Query(None, description="Health concern code"),
    clinical_status: Optional[str] = Query(None, description="Clinical status"),
    verification_status: Optional[str] = Query(None, description="Verification status"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Get health concerns for a patient.
    """
    try:
        # Extract the token from the payload
        token = token_payload.get("token")
        # Create the authorization header
        authorization = f"Bearer {token}"

        params = {
            "category": HEALTH_CONCERN_CATEGORY
        }
        if code:
            params["code"] = code
        if clinical_status:
            params["clinical-status"] = clinical_status
        if verification_status:
            params["verification-status"] = verification_status

        return await condition_service.get_patient_conditions(patient_id, params, authorization)
    except Exception as e:
        logger.error(f"Error getting patient health concerns: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))
