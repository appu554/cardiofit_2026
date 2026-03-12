from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> medication-service -> services -> backend
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Print the backend directory for debugging
print(f"Backend directory: {backend_dir}")
print(f"Checking if shared module exists: {os.path.exists(os.path.join(backend_dir, 'shared'))}")
if os.path.exists(os.path.join(backend_dir, 'shared')):
    print(f"Contents of shared directory:")
    for item in os.listdir(os.path.join(backend_dir, 'shared')):
        print(f"  {item}")

    # Check if auth directory exists
    auth_dir = os.path.join(backend_dir, 'shared', 'auth')
    if os.path.exists(auth_dir):
        print(f"Contents of auth directory:")
        for item in os.listdir(auth_dir):
            print(f"  {item}")

# Try to import the shared module using a more direct approach
try:
    # First try the normal import
    from shared.auth import HeaderAuthMiddleware
    print("Successfully imported HeaderAuthMiddleware from shared.auth")
except ImportError as e:
    print(f"Error importing HeaderAuthMiddleware from shared.auth: {e}")
    # If that fails, try the direct import module
    from app.direct_import import HeaderAuthMiddleware
    print("Using HeaderAuthMiddleware from direct_import")

from app.api.api import api_router
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Lifespan event handler
async def lifespan(app: FastAPI):
    # Startup
    logger.info("Using Google Healthcare API for medication data storage")

    # Initialize FHIR service
    try:
        from app.services.fhir_service_factory import initialize_fhir_service
        fhir_service = await initialize_fhir_service()
        logger.info(f"FHIR service initialized: {type(fhir_service).__name__}")
    except Exception as e:
        logger.error(f"Failed to initialize FHIR service: {e}")

    yield

    # Shutdown
    logger.info("Medication service shutdown complete")

app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Medication Service API for Clinical Synthesis Hub",
    version="1.0.0",
    openapi_url=f"{settings.API_PREFIX}/openapi.json",
    docs_url=f"{settings.API_PREFIX}/docs",
    redoc_url=f"{settings.API_PREFIX}/redoc",
    lifespan=lifespan
)

# Set up CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add header-based authentication middleware
# This middleware extracts user information from headers set by the API Gateway
app.add_middleware(
    HeaderAuthMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/", "/api/federation", "/api/webhooks", "/api/public", "/api/clinical-recipes", "/api/dose-calculation", "/api/drug-interactions"]
)

# Log middleware configuration
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

# Add federation endpoint (no authentication required for schema introspection)
try:
    import strawberry
    from strawberry.fastapi import GraphQLRouter
    from app.graphql.federation_schema import schema

    # Create GraphQL router for federation
    graphql_router = GraphQLRouter(schema)

    # Mount the federation endpoint
    app.include_router(graphql_router, prefix="/api/federation")
    logger.info("Federation endpoint mounted at /api/federation")
except ImportError as e:
    logger.warning(f"Could not mount federation endpoint: {e}")
except Exception as e:
    logger.error(f"Error mounting federation endpoint: {e}")

# Add public medication endpoint (no authentication required for Context Service)
@app.get("/api/public/medication-requests/patient/{patient_id}")
async def get_patient_medications_public(patient_id: str):
    """Get medication requests for a patient - public endpoint for Context Service."""
    try:
        from app.services.fhir_service_factory import get_fhir_service

        fhir_service = get_fhir_service()

        # Search for medication requests for this patient
        search_params = {"subject": f"Patient/{patient_id}"}
        resources = await fhir_service.search_resources("MedicationRequest", search_params)

        logger.info(f"Found {len(resources)} medication requests for patient {patient_id}")

        return {
            "patient_id": patient_id,
            "medication_requests": resources,
            "count": len(resources)
        }
    except Exception as e:
        logger.error(f"Error fetching patient medications: {e}")
        return {
            "patient_id": patient_id,
            "medication_requests": [],
            "count": 0,
            "error": str(e)
        }

# Add public clinical recipe endpoints (no authentication required for testing)
@app.get("/api/clinical-recipes/catalog")
async def get_clinical_recipe_catalog_public():
    """Get clinical recipe catalog - public endpoint for testing."""
    try:
        from app.domain.services.clinical_recipe_engine import ClinicalRecipeEngine

        recipe_engine = ClinicalRecipeEngine()
        catalog = recipe_engine.get_recipe_catalog()

        logger.info(f"Retrieved {len(catalog)} clinical recipes")

        return {
            "status": "success",
            "total_recipes": len(catalog),
            "recipes": catalog
        }
    except Exception as e:
        logger.error(f"Error getting recipe catalog: {e}")
        return {
            "status": "error",
            "total_recipes": 0,
            "recipes": {},
            "error": str(e)
        }

@app.post("/api/clinical-recipes/execute")
async def execute_clinical_recipes_public(request: dict):
    """Execute clinical recipes - public endpoint for testing."""
    try:
        from app.domain.services.clinical_recipe_engine import ClinicalRecipeEngine, RecipeContext
        from datetime import datetime

        recipe_engine = ClinicalRecipeEngine()

        # Create recipe context
        context = RecipeContext(
            patient_id=request.get("patient_id"),
            action_type=request.get("action_type"),
            medication_data=request.get("medication_data", {}),
            patient_data=request.get("patient_data", {}),
            provider_data=request.get("provider_data", {}),
            encounter_data=request.get("encounter_data", {}),
            clinical_data=request.get("clinical_data", {}),
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

        logger.info(f"Executed {len(results)} clinical recipes for {context.action_type}")

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
        logger.error(f"Error executing clinical recipes: {e}")
        return {
            "status": "error",
            "overall_safety_status": "ERROR",
            "error": str(e)
        }

@app.post("/api/dose-calculation/calculate")
async def calculate_dose_public(request: dict):
    """Calculate medication dose - public endpoint for testing."""
    try:
        from app.domain.services.dose_calculation_service import DoseCalculationService
        from app.domain.value_objects.dose_specification import DoseCalculationContext
        from app.domain.value_objects.clinical_properties import DosingType, DosingGuidelines
        from decimal import Decimal

        # Create dose calculation service
        dose_service = DoseCalculationService()

        # Create calculation context
        patient_context = request.get("patient_context", {})
        context = DoseCalculationContext(
            patient_id=request.get("patient_id"),
            weight_kg=Decimal(str(patient_context.get('weight_kg', 0))) if patient_context.get('weight_kg') else None,
            height_cm=Decimal(str(patient_context.get('height_cm', 0))) if patient_context.get('height_cm') else None,
            age_years=patient_context.get('age_years'),
            creatinine_clearance=Decimal(str(patient_context.get('creatinine_clearance', 0))) if patient_context.get('creatinine_clearance') else None,
            egfr=Decimal(str(patient_context.get('egfr', 0))) if patient_context.get('egfr') else None,
            liver_function=patient_context.get('liver_function'),
            pregnancy_status=patient_context.get('pregnancy_status'),
            breastfeeding_status=patient_context.get('breastfeeding_status')
        )

        # Create dosing guidelines (simplified for demo)
        dosing_parameters = request.get("dosing_parameters", {})
        guidelines = DosingGuidelines(
            weight_based_dose_mg_kg=Decimal(str(dosing_parameters.get('dose_per_kg', 10))),
            bsa_based_dose_mg_m2=Decimal(str(dosing_parameters.get('dose_per_m2', 50))),
            standard_dose_range={'standard': Decimal(str(dosing_parameters.get('standard_dose', 100)))},
            renal_adjustment_required=patient_context.get('creatinine_clearance', 100) < 60,
            hepatic_adjustment_required=patient_context.get('liver_function') in ['moderate', 'severe']
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

        dosing_type = dosing_type_map.get(request.get("calculation_type"), DosingType.WEIGHT_BASED)

        # Calculate dose
        dose_spec = dose_service.calculate_dose(
            dosing_type=dosing_type,
            context=context,
            guidelines=guidelines,
            medication_properties={
                'target_auc': dosing_parameters.get('target_auc', 5),
                'indication': request.get('indication')
            }
        )

        logger.info(f"Dose calculated: {dose_spec.to_display_string()}")

        return {
            "status": "success",
            "calculation_type": request.get("calculation_type"),
            "patient_id": request.get("patient_id"),
            "medication_code": request.get("medication_code"),
            "indication": request.get("indication"),
            "calculated_dose": {
                "value": float(dose_spec.value),
                "unit": dose_spec.unit.value,
                "route": dose_spec.route.value,
                "calculation_method": dose_spec.calculation_method,
                "calculation_factors": dose_spec.calculation_factors,
                "display_string": dose_spec.to_display_string()
            },
            "patient_context": patient_context,
            "dosing_parameters": dosing_parameters,
            "clinical_notes": [
                f"Dose calculated using {request.get('calculation_type')} method",
                f"Patient weight: {patient_context.get('weight_kg', 'N/A')} kg",
                f"Indication: {request.get('indication')}"
            ]
        }

    except Exception as e:
        logger.error(f"Error calculating dose: {e}")
        return {
            "status": "error",
            "error": str(e)
        }

@app.post("/api/drug-interactions/check")
async def check_drug_interactions_public(request: dict):
    """Check drug interactions - public endpoint for testing."""
    try:
        from app.services.fhir_service_factory import get_fhir_service

        fhir_service = get_fhir_service()
        patient_id = request.get("patient_id")
        new_medication_code = request.get("new_medication_code")

        # Get patient's current medications
        search_params = {
            "patient": f"Patient/{patient_id}",
            "status": "active"
        }
        current_medications = await fhir_service.search_resources("MedicationRequest", search_params)

        # Mock drug interaction analysis
        interactions = []

        # Check for known interactions (simplified)
        high_risk_combinations = {
            "1049502": ["warfarin", "aspirin"],  # Acetaminophen
            "197696": ["digoxin", "furosemide"],  # Ibuprofen
            "11124": ["warfarin", "furosemide"]   # Vancomycin
        }

        for medication in current_medications:
            med_concept = medication.get("medicationCodeableConcept", {})
            if med_concept.get("coding"):
                current_code = med_concept["coding"][0].get("code")
                current_name = med_concept["coding"][0].get("display", current_code)

                if current_code in high_risk_combinations.get(new_medication_code, []) or \
                   any(risk_med in current_name.lower() for risk_med in high_risk_combinations.get(new_medication_code, [])):
                    interactions.append({
                        "severity": "moderate",
                        "interaction_type": "drug-drug",
                        "description": f"Potential interaction between {new_medication_code} and {current_name}",
                        "mechanism": "Competitive inhibition or additive effects",
                        "management": "Monitor patient closely for adverse effects",
                        "medications": [
                            {
                                "code": new_medication_code,
                                "display": "New medication"
                            },
                            {
                                "code": current_code,
                                "display": current_name
                            }
                        ],
                        "clinical_consequence": "Increased risk of adverse effects",
                        "evidence_level": "moderate"
                    })

        logger.info(f"Found {len(interactions)} potential interactions for patient {patient_id}")

        return {
            "status": "success",
            "patient_id": patient_id,
            "new_medication_code": new_medication_code,
            "has_interactions": len(interactions) > 0,
            "interactions": interactions,
            "total_interactions": len(interactions),
            "current_medications_count": len(current_medications)
        }

    except Exception as e:
        logger.error(f"Error checking drug interactions: {e}")
        return {
            "status": "error",
            "error": str(e)
        }

@app.get("/")
async def root():
    return {"message": "Welcome to the Medication Service API"}

@app.get("/health")
async def health_check():
    return {"status": "healthy"}
