"""
Clinical Decision Support endpoints for the Medication Service.

This module provides REST API endpoints for medication-related clinical decision support,
including drug interaction checking, allergy alerts, and clinical recommendations.
"""

from typing import List, Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, Query
from pydantic import BaseModel
import logging
from datetime import datetime

from app.services.fhir_service_factory import get_fhir_service
from shared.auth import get_current_user_from_token
from app.domain.services.clinical_recipe_engine import (
    ClinicalRecipeEngine, RecipeContext, RecipeResult
)

logger = logging.getLogger(__name__)

router = APIRouter()

# Initialize the clinical recipe engine
recipe_engine = ClinicalRecipeEngine()

class DrugInteractionCheck(BaseModel):
    """Model for drug interaction check request."""
    patient_id: str
    new_medication_code: str
    new_medication_system: Optional[str] = "http://www.nlm.nih.gov/research/umls/rxnorm"

class MedicationAlert(BaseModel):
    """Model for medication alert."""
    id: str
    alert_type: str  # interaction, allergy, contraindication, dosage
    severity: str    # high, medium, low
    priority: str    # urgent, high, medium, low
    status: str      # active, acknowledged, resolved
    title: str
    description: str
    medication: Dict[str, Any]
    patient: Dict[str, str]
    triggered_by: Dict[str, Any]
    recommendations: List[Dict[str, str]]
    created_at: str
    updated_at: Optional[str] = None
    acknowledged_by: Optional[Dict[str, str]] = None
    resolved_by: Optional[Dict[str, str]] = None

class AlertAcknowledgment(BaseModel):
    """Model for acknowledging an alert."""
    acknowledged_by: Dict[str, str]
    note: Optional[str] = None
    action: str  # proceed, modify, discontinue, monitor

class ClinicalRecipeRequest(BaseModel):
    """Model for clinical recipe execution request."""
    patient_id: str
    action_type: str  # MEDICATION_PRESCRIBE, MEDICATION_MODIFY, etc.
    medication_data: Dict[str, Any]
    patient_data: Optional[Dict[str, Any]] = {}
    provider_data: Optional[Dict[str, Any]] = {}
    encounter_data: Optional[Dict[str, Any]] = {}
    clinical_data: Optional[Dict[str, Any]] = {}

class DoseCalculationRequest(BaseModel):
    """Model for dose calculation request."""
    patient_id: str
    medication_code: str
    medication_system: Optional[str] = "http://www.nlm.nih.gov/research/umls/rxnorm"
    indication: str
    calculation_type: str  # weight_based, bsa_based, auc_based, fixed, tiered
    patient_context: Dict[str, Any]  # weight, height, age, renal function, etc.
    dosing_parameters: Dict[str, Any]  # dose_per_kg, target_auc, etc.

class PharmaceuticalIntelligenceRequest(BaseModel):
    """Model for pharmaceutical intelligence analysis."""
    patient_id: str
    clinical_scenario: str
    current_medications: List[Dict[str, Any]]
    proposed_medication: Dict[str, Any]
    patient_factors: Dict[str, Any]

@router.post("/drug-interaction-check")
async def check_drug_interactions(
    check_data: DrugInteractionCheck,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Check for drug interactions for a new medication.
    
    Args:
        check_data: Drug interaction check data
        current_user: Current authenticated user
        
    Returns:
        Drug interaction analysis results
    """
    try:
        fhir_service = get_fhir_service()
        
        # Get patient's current medications
        search_params = {
            "patient": f"Patient/{check_data.patient_id}",
            "status": "active"
        }
        current_medications = await fhir_service.search_resources("MedicationRequest", search_params)
        
        # Get patient allergies
        allergy_params = {"patient": f"Patient/{check_data.patient_id}"}
        allergies = await fhir_service.search_resources("AllergyIntolerance", allergy_params)
        
        # Mock drug interaction analysis (in production, integrate with drug interaction database)
        interactions = []
        contraindications = []
        allergic_reactions = []
        dosage_adjustments = []
        
        # Check for known interactions (mock data)
        high_risk_combinations = {
            "1049502": ["warfarin", "aspirin"],  # Acetaminophen
            "197696": ["digoxin", "furosemide"]  # Ibuprofen
        }
        
        for medication in current_medications:
            med_concept = medication.get("medicationCodeableConcept", {})
            if med_concept.get("coding"):
                current_code = med_concept["coding"][0].get("code")
                if current_code in high_risk_combinations.get(check_data.new_medication_code, []):
                    interactions.append({
                        "severity": "moderate",
                        "interaction_type": "drug-drug",
                        "description": f"Potential interaction between {check_data.new_medication_code} and {current_code}",
                        "mechanism": "Competitive inhibition",
                        "management": "Monitor patient closely",
                        "medications": [
                            {
                                "code": check_data.new_medication_code,
                                "display": "New medication",
                                "system": check_data.new_medication_system
                            },
                            {
                                "code": current_code,
                                "display": med_concept["coding"][0].get("display", current_code),
                                "system": med_concept["coding"][0].get("system", "")
                            }
                        ],
                        "clinical_consequence": "Increased risk of bleeding",
                        "evidence_level": "moderate",
                        "references": [
                            {
                                "title": "Drug Interaction Database",
                                "url": "https://example.com/interactions",
                                "source": "Clinical Database"
                            }
                        ]
                    })
        
        # Check for allergies
        for allergy in allergies:
            allergy_code = allergy.get("code", {})
            if allergy_code.get("coding"):
                allergen_code = allergy_code["coding"][0].get("code")
                # Mock allergy check
                if allergen_code == check_data.new_medication_code:
                    allergic_reactions.append({
                        "allergen": {
                            "code": allergen_code,
                            "display": allergy_code["coding"][0].get("display", allergen_code),
                            "system": allergy_code["coding"][0].get("system", "")
                        },
                        "severity": allergy.get("criticality", "unknown"),
                        "reaction": "Allergic reaction documented",
                        "cross_sensitivity": False
                    })
        
        # Mock contraindications and dosage adjustments
        if check_data.new_medication_code == "1049502":  # Acetaminophen
            dosage_adjustments.append({
                "reason": "Hepatic impairment",
                "adjustment": "Reduce dose by 50%",
                "new_dosage": "325mg every 8 hours",
                "monitoring": "Monitor liver function tests"
            })
        
        return {
            "has_interactions": len(interactions) > 0,
            "interactions": interactions,
            "contraindications": contraindications,
            "allergic_reactions": allergic_reactions,
            "dosage_adjustments": dosage_adjustments
        }
        
    except Exception as e:
        logger.error(f"Error checking drug interactions: {e}")
        raise HTTPException(status_code=500, detail=f"Error checking interactions: {str(e)}")

@router.get("/alerts/patient/{patient_id}")
async def get_patient_medication_alerts(
    patient_id: str,
    status: Optional[str] = Query(None, description="Filter by alert status"),
    current_user: dict = Depends(get_current_user_from_token)
) -> List[MedicationAlert]:
    """
    Get medication alerts for a specific patient.
    
    Args:
        patient_id: The patient ID
        status: Filter by alert status
        current_user: Current authenticated user
        
    Returns:
        List of medication alerts
    """
    try:
        # Mock alerts (in production, retrieve from database)
        mock_alerts = [
            MedicationAlert(
                id="alert-001",
                alert_type="interaction",
                severity="high",
                priority="urgent",
                status="active",
                title="Drug-Drug Interaction Alert",
                description="Potential interaction between Warfarin and Aspirin",
                medication={
                    "code": "1049502",
                    "display": "Acetaminophen 325 MG Oral Tablet",
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm"
                },
                patient={
                    "reference": f"Patient/{patient_id}",
                    "display": "Patient"
                },
                triggered_by={
                    "type": "new_medication",
                    "reference": "MedicationRequest/med-123",
                    "display": "New medication order"
                },
                recommendations=[
                    {
                        "action": "monitor",
                        "description": "Monitor INR levels closely",
                        "priority": "high"
                    },
                    {
                        "action": "consider_alternative",
                        "description": "Consider alternative pain management",
                        "priority": "medium"
                    }
                ],
                created_at=datetime.now().isoformat(),
                updated_at=None,
                acknowledged_by=None,
                resolved_by=None
            ),
            MedicationAlert(
                id="alert-002",
                alert_type="allergy",
                severity="high",
                priority="urgent",
                status="active",
                title="Allergy Alert",
                description="Patient has documented allergy to Penicillin",
                medication={
                    "code": "7980",
                    "display": "Penicillin",
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm"
                },
                patient={
                    "reference": f"Patient/{patient_id}",
                    "display": "Patient"
                },
                triggered_by={
                    "type": "medication_order",
                    "reference": "MedicationRequest/med-456",
                    "display": "Penicillin order"
                },
                recommendations=[
                    {
                        "action": "discontinue",
                        "description": "Discontinue Penicillin immediately",
                        "priority": "urgent"
                    },
                    {
                        "action": "alternative",
                        "description": "Consider Cephalexin as alternative",
                        "priority": "high"
                    }
                ],
                created_at=datetime.now().isoformat()
            )
        ]
        
        # Filter by status if provided
        if status:
            mock_alerts = [alert for alert in mock_alerts if alert.status == status]
        
        logger.info(f"Retrieved {len(mock_alerts)} alerts for patient {patient_id}")
        return mock_alerts
        
    except Exception as e:
        logger.error(f"Error fetching medication alerts: {e}")
        raise HTTPException(status_code=500, detail=f"Error fetching alerts: {str(e)}")

@router.post("/alerts/{alert_id}/acknowledge")
async def acknowledge_medication_alert(
    alert_id: str,
    acknowledgment: AlertAcknowledgment,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Acknowledge a medication alert.
    
    Args:
        alert_id: The alert ID
        acknowledgment: Acknowledgment data
        current_user: Current authenticated user
        
    Returns:
        Updated alert status
    """
    try:
        # In production, update the alert in database
        logger.info(f"Alert {alert_id} acknowledged by {acknowledgment.acknowledged_by}")
        
        return {
            "id": alert_id,
            "status": "acknowledged",
            "acknowledged_by": acknowledgment.acknowledged_by,
            "acknowledgment_note": acknowledgment.note,
            "action": acknowledgment.action,
            "timestamp": datetime.now().isoformat()
        }
        
    except Exception as e:
        logger.error(f"Error acknowledging alert: {e}")
        raise HTTPException(status_code=500, detail=f"Error acknowledging alert: {str(e)}")

@router.get("/recommendations/patient/{patient_id}")
async def get_medication_recommendations(
    patient_id: str,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Get medication recommendations for a patient.
    
    Args:
        patient_id: The patient ID
        current_user: Current authenticated user
        
    Returns:
        Medication recommendations
    """
    try:
        fhir_service = get_fhir_service()
        
        # Get patient medications and conditions for recommendations
        med_params = {"patient": f"Patient/{patient_id}", "status": "active"}
        medications = await fhir_service.search_resources("MedicationRequest", med_params)
        
        # Mock recommendations based on current medications
        recommendations = {
            "optimization": [
                {
                    "type": "dosage_optimization",
                    "medication": "Acetaminophen",
                    "current_dose": "650mg q6h",
                    "recommended_dose": "500mg q6h",
                    "rationale": "Lower dose may be equally effective with reduced hepatotoxicity risk"
                }
            ],
            "monitoring": [
                {
                    "type": "lab_monitoring",
                    "medication": "Warfarin",
                    "test": "INR",
                    "frequency": "Weekly",
                    "rationale": "Monitor anticoagulation effectiveness"
                }
            ],
            "alternatives": [
                {
                    "current_medication": "Ibuprofen",
                    "alternative": "Acetaminophen",
                    "rationale": "Reduced GI bleeding risk in elderly patients"
                }
            ],
            "adherence": [
                {
                    "medication": "Metformin",
                    "suggestion": "Consider extended-release formulation",
                    "rationale": "Improve adherence with once-daily dosing"
                }
            ]
        }
        
        return recommendations
        
    except Exception as e:
        logger.error(f"Error getting recommendations: {e}")
        raise HTTPException(status_code=500, detail=f"Error getting recommendations: {str(e)}")

# Clinical Recipe Endpoints - The Real Medication Service Purpose

@router.post("/clinical-recipes/execute")
async def execute_clinical_recipes(
    request: ClinicalRecipeRequest,
    current_user: dict = Depends(get_current_user_from_token)
):
    """
    Execute applicable clinical recipes for medication safety and decision support.

    This is the main endpoint for the Clinical Pharmacist's Digital Twin functionality.
    It executes all applicable clinical logic recipes based on the medication action.
    """
    try:
        # Create recipe context
        context = RecipeContext(
            patient_id=request.patient_id,
            action_type=request.action_type,
            medication_data=request.medication_data,
            patient_data=request.patient_data,
            provider_data=request.provider_data,
            encounter_data=request.encounter_data,
            clinical_data=request.clinical_data,
            timestamp=datetime.now()
        )

        # Execute applicable recipes
        results = await recipe_engine.execute_applicable_recipes(context)

        # Aggregate results
        overall_status = "SAFE"
        total_validations = 0
        critical_issues = []
        warnings = []

        for result in results:
            total_validations += len(result.validations)

            if result.overall_status == "UNSAFE":
                overall_status = "UNSAFE"
            elif result.overall_status == "WARNING" and overall_status != "UNSAFE":
                overall_status = "WARNING"

            # Collect critical issues and warnings
            for validation in result.validations:
                if not validation.passed:
                    if validation.severity == "CRITICAL":
                        critical_issues.append(validation)
                    else:
                        warnings.append(validation)

        return {
            "status": "success",
            "overall_safety_status": overall_status,
            "total_recipes_executed": len(results),
            "total_validations": total_validations,
            "critical_issues": len(critical_issues),
            "warnings": len(warnings),
            "execution_summary": {
                "total_time_ms": sum(r.execution_time_ms for r in results),
                "fastest_recipe_ms": min(r.execution_time_ms for r in results) if results else 0,
                "slowest_recipe_ms": max(r.execution_time_ms for r in results) if results else 0
            },
            "recipe_results": [
                {
                    "recipe_id": r.recipe_id,
                    "recipe_name": r.recipe_name,
                    "status": r.overall_status,
                    "execution_time_ms": r.execution_time_ms,
                    "validations": [
                        {
                            "passed": v.passed,
                            "severity": v.severity,
                            "message": v.message,
                            "explanation": v.explanation,
                            "alternatives": v.alternatives
                        } for v in r.validations
                    ],
                    "clinical_decision_support": r.clinical_decision_support
                } for r in results
            ],
            "critical_issues": [
                {
                    "severity": issue.severity,
                    "message": issue.message,
                    "explanation": issue.explanation,
                    "alternatives": issue.alternatives
                } for issue in critical_issues
            ],
            "warnings": [
                {
                    "severity": warning.severity,
                    "message": warning.message,
                    "explanation": warning.explanation,
                    "alternatives": warning.alternatives
                } for warning in warnings
            ]
        }

    except Exception as e:
        logger.error(f"Error executing clinical recipes: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Clinical recipe execution failed: {str(e)}")

@router.get("/clinical-recipes/catalog")
async def get_recipe_catalog(
    current_user: dict = Depends(get_current_user_from_token)
):
    """
    Get catalog of all available clinical recipes.

    Returns information about all registered clinical logic recipes,
    including their priorities, descriptions, and clinical rationale.
    """
    try:
        catalog = recipe_engine.get_recipe_catalog()

        return {
            "status": "success",
            "total_recipes": len(catalog),
            "recipes": catalog
        }

    except Exception as e:
        logger.error(f"Error getting recipe catalog: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Failed to get recipe catalog: {str(e)}")

@router.post("/dose-calculation/calculate")
async def calculate_dose(
    request: DoseCalculationRequest,
    current_user: dict = Depends(get_current_user_from_token)
):
    """
    Calculate medication dose using pharmaceutical intelligence.

    This endpoint implements the core dose calculation functionality
    from the Clinical Pharmacist's Digital Twin, supporting multiple
    calculation strategies (weight-based, BSA-based, AUC-based, etc.).
    """
    try:
        # Import dose calculation service
        from app.domain.services.dose_calculation_service import DoseCalculationService
        from app.domain.value_objects.dose_specification import DoseCalculationContext
        from app.domain.value_objects.clinical_properties import DosingType, DosingGuidelines
        from decimal import Decimal

        # Create dose calculation service
        dose_service = DoseCalculationService()

        # Create calculation context
        context = DoseCalculationContext(
            patient_id=request.patient_id,
            weight_kg=Decimal(str(request.patient_context.get('weight_kg', 0))) if request.patient_context.get('weight_kg') else None,
            height_cm=Decimal(str(request.patient_context.get('height_cm', 0))) if request.patient_context.get('height_cm') else None,
            age_years=request.patient_context.get('age_years'),
            creatinine_clearance=Decimal(str(request.patient_context.get('creatinine_clearance', 0))) if request.patient_context.get('creatinine_clearance') else None,
            egfr=Decimal(str(request.patient_context.get('egfr', 0))) if request.patient_context.get('egfr') else None,
            liver_function=request.patient_context.get('liver_function'),
            pregnancy_status=request.patient_context.get('pregnancy_status'),
            breastfeeding_status=request.patient_context.get('breastfeeding_status')
        )

        # Create dosing guidelines (simplified for demo)
        guidelines = DosingGuidelines(
            weight_based_dose_mg_kg=Decimal(str(request.dosing_parameters.get('dose_per_kg', 10))),
            bsa_based_dose_mg_m2=Decimal(str(request.dosing_parameters.get('dose_per_m2', 50))),
            standard_dose_range={'standard': Decimal(str(request.dosing_parameters.get('standard_dose', 100)))},
            renal_adjustment_required=request.patient_context.get('creatinine_clearance', 100) < 60,
            hepatic_adjustment_required=request.patient_context.get('liver_function') in ['moderate', 'severe']
        )

        # Determine dosing type
        dosing_type_map = {
            'weight_based': DosingType.WEIGHT_BASED,
            'bsa_based': DosingType.BSA_BASED,
            'auc_based': DosingType.AUC_BASED,
            'fixed': DosingType.FIXED,
            'tiered': DosingType.TIERED,
            'loading_dose': DosingType.LOADING_DOSE
        }

        dosing_type = dosing_type_map.get(request.calculation_type, DosingType.WEIGHT_BASED)

        # Calculate dose
        dose_spec = dose_service.calculate_dose(
            dosing_type=dosing_type,
            context=context,
            guidelines=guidelines,
            medication_properties={
                'target_auc': request.dosing_parameters.get('target_auc', 5),
                'indication': request.indication
            }
        )

        return {
            "status": "success",
            "calculation_type": request.calculation_type,
            "patient_id": request.patient_id,
            "medication_code": request.medication_code,
            "indication": request.indication,
            "calculated_dose": {
                "value": float(dose_spec.value),
                "unit": dose_spec.unit.value,
                "route": dose_spec.route.value,
                "calculation_method": dose_spec.calculation_method,
                "calculation_factors": dose_spec.calculation_factors,
                "display_string": dose_spec.to_display_string()
            },
            "patient_context": request.patient_context,
            "dosing_parameters": request.dosing_parameters,
            "clinical_notes": [
                f"Dose calculated using {request.calculation_type} method",
                f"Patient weight: {request.patient_context.get('weight_kg', 'N/A')} kg",
                f"Indication: {request.indication}"
            ]
        }

    except Exception as e:
        logger.error(f"Error calculating dose: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Dose calculation failed: {str(e)}")
