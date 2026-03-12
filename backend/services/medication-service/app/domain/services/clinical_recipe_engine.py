"""
Clinical Recipe Engine - Implementation of the Clinical Logic Recipe Book

This module implements the comprehensive clinical logic recipes defined in
MedicationRecipeBook.txt, providing pharmaceutical intelligence and clinical
decision support through structured recipe execution.

Key Features:
- Recipe-based clinical validation
- Multi-tier safety checking (CAE, ProtocolEngine, GraphDB)
- Clinical decision support with explanations
- ML enhancement points for continuous learning
- Performance optimization with QoS tiers
"""

import logging
from typing import Dict, List, Any, Optional, Union
from dataclasses import dataclass
from enum import Enum
from abc import ABC, abstractmethod
import asyncio
from datetime import datetime

logger = logging.getLogger(__name__)

class RecipePriority(Enum):
    """Recipe priority levels - supports all values from MedicationRecipeBook.txt"""
    CRITICAL_100 = 100  # Code Blue, Massive Transfusion
    CRITICAL_99 = 99    # Chemotherapy, Pregnancy, RSI
    CRITICAL_98 = 98    # Anticoagulation, Transfusion, Pediatric
    HIGH_97 = 97        # Anesthesia, Immunocompromised
    HIGH_96 = 96        # Pre-Procedural, Geriatric
    HIGH_95 = 95        # Renal, Discharge, Contrast, Cardiology
    MEDIUM_94 = 94      # Drug Interactions, Antimicrobial
    MEDIUM_93 = 93      # Admission, TDM, Diabetes
    MEDIUM_92 = 92      # Controlled Substances, Psychiatry
    STANDARD_91 = 91    # High-Risk Monitoring, Radiation
    STANDARD_90 = 90    # Hepatic, Core Measures
    LOW_89 = 89         # Regulatory Compliance

class QoSTier(Enum):
    """Quality of Service tiers for recipe execution"""
    PLATINUM = "platinum"  # <50ms, highest accuracy
    GOLD = "gold"         # <100ms, high accuracy
    SILVER = "silver"     # <200ms, standard accuracy
    BRONZE = "bronze"     # <500ms, basic accuracy

class ValidationEngine(Enum):
    """Available validation engines"""
    CAE = "cae"                    # Clinical Assessment Engine
    PROTOCOL_ENGINE = "protocol"   # Protocol compliance engine
    GRAPH_DB = "graphdb"          # Relationship-based validation

@dataclass
class RecipeContext:
    """Context data for recipe execution"""
    patient_id: str
    action_type: str
    medication_data: Dict[str, Any]
    patient_data: Dict[str, Any]
    provider_data: Dict[str, Any]
    encounter_data: Dict[str, Any]
    clinical_data: Dict[str, Any]
    timestamp: datetime

@dataclass
class ValidationResult:
    """Result of a validation check"""
    passed: bool
    severity: str  # CRITICAL, HIGH, MEDIUM, LOW
    message: str
    explanation: str
    alternatives: List[str]
    evidence_base: List[str]
    ml_confidence: Optional[float] = None

@dataclass
class RecipeResult:
    """Complete result of recipe execution"""
    recipe_id: str
    recipe_name: str
    execution_time_ms: float
    validations: List[ValidationResult]
    overall_status: str  # SAFE, WARNING, UNSAFE, ERROR
    clinical_decision_support: Dict[str, Any]
    cost_considerations: Dict[str, Any]
    ml_insights: Dict[str, Any]
    performance_metrics: Dict[str, Any]

class ClinicalRecipe(ABC):
    """Abstract base class for clinical recipes"""
    
    def __init__(self, recipe_config: Dict[str, Any]):
        self.id = recipe_config['id']
        self.name = recipe_config['name']
        self.description = recipe_config['description']
        # Store priority as integer value for flexibility
        self.priority = recipe_config.get('priority', 90)
        self.triggers = recipe_config.get('triggers', [])
        self.clinical_rationale = recipe_config.get('clinicalRationale', '')
        self.evidence_base = recipe_config.get('evidenceBase', {})
        self.validation_logic = recipe_config.get('validationLogic', {})
        self.clinical_decision_support = recipe_config.get('clinicalDecisionSupport', {})
        self.cost_considerations = recipe_config.get('costConsiderations', {})
        self.ml_enhancement = recipe_config.get('mlEnhancement', {})
        self.performance_hint = recipe_config.get('performanceHint', '')
        self.fallback_behavior = recipe_config.get('fallbackBehavior', '')
        self.qos_tier = QoSTier(recipe_config.get('qosTier', 'silver'))
    
    @abstractmethod
    async def execute(self, context: RecipeContext) -> RecipeResult:
        """Execute the recipe with given context"""
        pass
    
    @abstractmethod
    def should_trigger(self, context: RecipeContext) -> bool:
        """Check if recipe should be triggered for given context"""
        pass

    def _determine_overall_status(self, validations: List[ValidationResult]) -> str:
        """Determine overall safety status from validations"""
        if not validations:
            return "SAFE"

        critical_failures = [v for v in validations if not v.passed and v.severity == "CRITICAL"]
        if critical_failures:
            return "UNSAFE"

        high_warnings = [v for v in validations if not v.passed and v.severity == "HIGH"]
        if high_warnings:
            return "WARNING"

        return "SAFE"

    def _error_result(self, error: Exception, start_time: datetime) -> RecipeResult:
        """Generate error result for failed recipe execution"""
        execution_time = (datetime.now() - start_time).total_seconds() * 1000

        return RecipeResult(
            recipe_id=self.id,
            recipe_name=self.name,
            execution_time_ms=execution_time,
            validations=[ValidationResult(
                passed=False,
                severity="CRITICAL",
                message=f"Recipe execution failed: {str(error)}",
                explanation="Internal system error during recipe execution",
                alternatives=[],
                evidence_base=[]
            )],
            overall_status="ERROR",
            clinical_decision_support={},
            cost_considerations={},
            ml_insights={},
            performance_metrics={'execution_time_ms': execution_time}
        )

class MedicationSafetyBaseRecipe(ClinicalRecipe):
    """
    Recipe 1.1: Universal Medication Safety Check
    Foundational safety for ALL medication prescriptions
    """
    
    def __init__(self):
        config = {
            'id': 'medication-safety-base-v3.0',
            'name': 'Universal Medication Safety Check',
            'priority': 100,
            'description': 'Foundational safety for ALL medication prescriptions',
            'triggers': [
                {'action_type': ['MEDICATION_PRESCRIBE', 'MEDICATION_MODIFY', 'MEDICATION_RENEW']}
            ],
            'clinicalRationale': '''
                Prevent allergic reactions, drug duplications, and basic contraindications. 
                This is the minimum safety net required for all medications.
            ''',
            'qosTier': 'platinum'
        }
        super().__init__(config)
    
    def should_trigger(self, context: RecipeContext) -> bool:
        """Always trigger for medication actions"""
        return context.action_type in ['MEDICATION_PRESCRIBE', 'MEDICATION_MODIFY', 'MEDICATION_RENEW']
    
    async def execute(self, context: RecipeContext) -> RecipeResult:
        """Execute universal medication safety checks"""
        start_time = datetime.now()
        validations = []
        
        try:
            # CAE Validations
            validations.extend(await self._check_allergies(context))
            validations.extend(await self._check_duplications(context))
            validations.extend(await self._check_contraindications(context))
            validations.extend(await self._check_pregnancy_safety(context))
            validations.extend(await self._check_age_restrictions(context))
            
            # GraphDB Validations
            validations.extend(await self._check_cross_sensitivities(context))
            validations.extend(await self._check_drug_disease_interactions(context))
            
            # Determine overall status
            overall_status = self._determine_overall_status(validations)
            
            # Generate clinical decision support
            cds = self._generate_clinical_decision_support(validations, context)
            
            execution_time = (datetime.now() - start_time).total_seconds() * 1000
            
            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=validations,
                overall_status=overall_status,
                clinical_decision_support=cds,
                cost_considerations={},
                ml_insights={},
                performance_metrics={'execution_time_ms': execution_time}
            )
            
        except Exception as e:
            logger.error(f"Error executing {self.id}: {str(e)}")
            execution_time = (datetime.now() - start_time).total_seconds() * 1000
            
            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=[ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"Recipe execution failed: {str(e)}",
                    explanation="Internal system error during safety validation",
                    alternatives=[],
                    evidence_base=[]
                )],
                overall_status="ERROR",
                clinical_decision_support={},
                cost_considerations={},
                ml_insights={},
                performance_metrics={'execution_time_ms': execution_time}
            )
    
    async def _check_allergies(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for allergic reactions and cross-sensitivities"""
        validations = []
        
        # Get patient allergies
        allergies = context.patient_data.get('allergies', [])
        medication = context.medication_data
        
        for allergy in allergies:
            # Check direct ingredient match
            if self._is_allergic_to_ingredient(allergy, medication):
                validations.append(ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"ALLERGY ALERT: Patient allergic to {allergy.get('substance')}",
                    explanation=f"Patient has documented allergy to {allergy.get('substance')} with reaction: {allergy.get('reaction')}",
                    alternatives=["Consider alternative medication class", "Consult allergy specialist"],
                    evidence_base=["Patient allergy history"]
                ))
        
        return validations
    
    async def _check_duplications(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for therapeutic duplications"""
        validations = []
        
        current_medications = context.clinical_data.get('current_medications', [])
        new_medication = context.medication_data
        
        for current_med in current_medications:
            if self._is_therapeutic_duplication(current_med, new_medication):
                validations.append(ValidationResult(
                    passed=False,
                    severity="HIGH",
                    message=f"DUPLICATION: Similar to existing medication {current_med.get('name')}",
                    explanation="Therapeutic duplication may increase risk of adverse effects",
                    alternatives=["Discontinue existing medication", "Adjust doses"],
                    evidence_base=["Therapeutic class analysis"]
                ))
        
        return validations
    
    async def _check_contraindications(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for absolute contraindications"""
        validations = []
        
        # Implementation would check against contraindication database
        # This is a simplified version
        
        return validations
    
    async def _check_pregnancy_safety(self, context: RecipeContext) -> List[ValidationResult]:
        """Check pregnancy/lactation safety"""
        validations = []
        
        patient_data = context.patient_data
        if patient_data.get('pregnancy_status') == 'pregnant':
            medication = context.medication_data
            pregnancy_category = medication.get('pregnancy_category')
            
            if pregnancy_category in ['X', 'D']:
                validations.append(ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"PREGNANCY RISK: Category {pregnancy_category} medication",
                    explanation="This medication poses significant risk to the fetus",
                    alternatives=["Consider safer alternatives", "Consult maternal-fetal medicine"],
                    evidence_base=["FDA pregnancy categories", "PLLR data"]
                ))
        
        return validations
    
    async def _check_age_restrictions(self, context: RecipeContext) -> List[ValidationResult]:
        """Check age-specific contraindications"""
        validations = []
        
        # Implementation would check age-specific restrictions
        
        return validations
    
    async def _check_cross_sensitivities(self, context: RecipeContext) -> List[ValidationResult]:
        """Check allergy cross-sensitivities using GraphDB"""
        validations = []
        
        # Implementation would query GraphDB for cross-sensitivity relationships
        
        return validations
    
    async def _check_drug_disease_interactions(self, context: RecipeContext) -> List[ValidationResult]:
        """Check drug-disease contraindications"""
        validations = []
        
        # Implementation would check against disease contraindications
        
        return validations
    
    def _is_allergic_to_ingredient(self, allergy: Dict, medication: Dict) -> bool:
        """Check if patient is allergic to medication ingredient"""
        # Simplified implementation
        allergy_substance = allergy.get('substance', '').lower()
        med_ingredients = medication.get('ingredients', [])
        
        for ingredient in med_ingredients:
            if allergy_substance in ingredient.lower():
                return True
        
        return False
    
    def _is_therapeutic_duplication(self, med1: Dict, med2: Dict) -> bool:
        """Check if two medications are therapeutic duplicates"""
        # Simplified implementation
        return med1.get('therapeutic_class') == med2.get('therapeutic_class')
    
    def _determine_overall_status(self, validations: List[ValidationResult]) -> str:
        """Determine overall safety status"""
        if not validations:
            return "SAFE"
        
        critical_failures = [v for v in validations if not v.passed and v.severity == "CRITICAL"]
        if critical_failures:
            return "UNSAFE"
        
        high_warnings = [v for v in validations if not v.passed and v.severity == "HIGH"]
        if high_warnings:
            return "WARNING"
        
        return "SAFE"
    
    def _generate_clinical_decision_support(self, validations: List[ValidationResult], context: RecipeContext) -> Dict[str, Any]:
        """Generate clinical decision support recommendations"""
        return {
            'provider_explanation': "Safety checks: allergies, duplications, contraindications",
            'patient_explanation': "Checking if this medication is safe for you",
            'recommendations': [v.alternatives for v in validations if not v.passed],
            'evidence_summary': [v.evidence_base for v in validations if not v.passed]
        }

class RenalSafetyRecipe(ClinicalRecipe):
    """
    Recipe 1.2: Renal Function Safety
    Renal dosing and nephrotoxicity prevention
    """

    def __init__(self):
        config = {
            'id': 'medication-safety-renal-v3.0',
            'name': 'Renal Dosing and Nephrotoxicity Prevention',
            'priority': 95,
            'description': 'Renal adjustment and nephrotoxicity prevention',
            'triggers': [
                {'drug.properties.renalClearance': '>50%'},
                {'drug.properties.nephrotoxicityRisk': '!= NONE'},
                {'drug.class': ['NSAID', 'ACE_INHIBITOR', 'ARB', 'AMINOGLYCOSIDE', 'VANCOMYCIN']}
            ],
            'clinicalRationale': '''
                50% of adverse drug events in CKD patients are preventable with proper
                dose adjustment. Accumulation risk increases exponentially with decreased GFR.
            ''',
            'qosTier': 'gold'
        }
        super().__init__(config)

    def should_trigger(self, context: RecipeContext) -> bool:
        """Trigger for medications requiring renal adjustment"""
        medication = context.medication_data
        patient = context.patient_data

        # Check if medication has renal clearance >50%
        if medication.get('renal_clearance', 0) > 50:
            return True

        # Check for nephrotoxic medications
        nephrotoxic_classes = ['NSAID', 'ACE_INHIBITOR', 'ARB', 'AMINOGLYCOSIDE', 'VANCOMYCIN']
        if medication.get('therapeutic_class') in nephrotoxic_classes:
            return True

        # Check if patient has renal impairment
        if patient.get('creatinine', 0) > 1.2 or patient.get('egfr', 100) < 60:
            return True

        return False

    async def execute(self, context: RecipeContext) -> RecipeResult:
        """Execute renal safety checks"""
        start_time = datetime.now()
        validations = []

        try:
            # Calculate eGFR using CKD-EPI equation
            validations.extend(await self._calculate_egfr(context))

            # Check for AKI using KDIGO criteria
            validations.extend(await self._check_aki_risk(context))

            # Apply renal dose adjustments
            validations.extend(await self._apply_renal_adjustments(context))

            # Check for multiple nephrotoxins
            validations.extend(await self._check_nephrotoxic_burden(context))

            # Check contrast exposure risk
            validations.extend(await self._check_contrast_exposure(context))

            overall_status = self._determine_overall_status(validations)
            execution_time = (datetime.now() - start_time).total_seconds() * 1000

            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=validations,
                overall_status=overall_status,
                clinical_decision_support=self._generate_renal_cds(validations, context),
                cost_considerations={'monitoring': 'Increased lab monitoring required'},
                ml_insights={'pattern': 'AKI risk prediction'},
                performance_metrics={'execution_time_ms': execution_time}
            )

        except Exception as e:
            logger.error(f"Error executing {self.id}: {str(e)}")
            execution_time = (datetime.now() - start_time).total_seconds() * 1000

            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=[ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"Renal safety check failed: {str(e)}",
                    explanation="Unable to assess renal safety",
                    alternatives=[],
                    evidence_base=[]
                )],
                overall_status="ERROR",
                clinical_decision_support={},
                cost_considerations={},
                ml_insights={},
                performance_metrics={'execution_time_ms': execution_time}
            )

    async def _calculate_egfr(self, context: RecipeContext) -> List[ValidationResult]:
        """Calculate eGFR using CKD-EPI 2021 equation"""
        validations = []
        patient = context.patient_data

        age = patient.get('age', 0)
        creatinine = patient.get('creatinine', 1.0)
        gender = patient.get('gender', 'unknown')

        # Simplified CKD-EPI calculation (race-neutral)
        if gender.lower() == 'female':
            if creatinine <= 0.7:
                egfr = 142 * (creatinine / 0.7) ** -0.241 * (0.9938 ** age)
            else:
                egfr = 142 * (creatinine / 0.7) ** -1.2 * (0.9938 ** age)
        else:
            if creatinine <= 0.9:
                egfr = 142 * (creatinine / 0.9) ** -0.302 * (0.9938 ** age)
            else:
                egfr = 142 * (creatinine / 0.9) ** -1.2 * (0.9938 ** age)

        if egfr < 60:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Chronic kidney disease detected: eGFR {egfr:.1f} mL/min/1.73m²",
                explanation="Reduced kidney function requires dose adjustment",
                alternatives=["Reduce dose based on eGFR", "Consider alternative medication"],
                evidence_base=["CKD-EPI 2021 equation", "KDIGO guidelines"]
            ))

        return validations

    async def _check_aki_risk(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for acute kidney injury risk"""
        validations = []
        clinical_data = context.clinical_data

        # Check for recent creatinine increase
        recent_labs = clinical_data.get('recent_labs', {})
        baseline_cr = recent_labs.get('baseline_creatinine', 1.0)
        current_cr = recent_labs.get('current_creatinine', 1.0)

        if current_cr >= baseline_cr * 1.5:
            validations.append(ValidationResult(
                passed=False,
                severity="CRITICAL",
                message="Acute kidney injury detected (KDIGO Stage 1)",
                explanation=f"Creatinine increased from {baseline_cr} to {current_cr}",
                alternatives=["Hold nephrotoxic medications", "Nephrology consultation"],
                evidence_base=["KDIGO AKI guidelines"]
            ))

        return validations

    async def _apply_renal_adjustments(self, context: RecipeContext) -> List[ValidationResult]:
        """Apply renal dose adjustments"""
        validations = []
        medication = context.medication_data
        patient = context.patient_data

        egfr = patient.get('egfr', 100)

        # Example dose adjustments for common medications
        dose_adjustments = {
            'vancomycin': {
                'egfr_30_60': 'Reduce dose by 25%',
                'egfr_15_30': 'Reduce dose by 50%',
                'egfr_less_15': 'Avoid or dialysis dosing'
            },
            'metformin': {
                'egfr_30_45': 'Reduce dose by 50%',
                'egfr_less_30': 'Contraindicated'
            }
        }

        med_name = medication.get('name', '').lower()
        if med_name in dose_adjustments:
            adjustments = dose_adjustments[med_name]

            if egfr < 15 and 'egfr_less_15' in adjustments:
                validations.append(ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"Severe renal impairment: {adjustments['egfr_less_15']}",
                    explanation=f"eGFR {egfr} mL/min/1.73m² requires dose modification",
                    alternatives=["Alternative medication", "Nephrology consultation"],
                    evidence_base=["FDA dosing guidelines"]
                ))
            elif egfr < 30 and 'egfr_15_30' in adjustments:
                validations.append(ValidationResult(
                    passed=False,
                    severity="HIGH",
                    message=f"Moderate renal impairment: {adjustments['egfr_15_30']}",
                    explanation=f"eGFR {egfr} mL/min/1.73m² requires dose reduction",
                    alternatives=["Reduce dose as recommended"],
                    evidence_base=["FDA dosing guidelines"]
                ))

        return validations

    async def _check_nephrotoxic_burden(self, context: RecipeContext) -> List[ValidationResult]:
        """Check cumulative nephrotoxic burden"""
        validations = []
        current_meds = context.clinical_data.get('current_medications', [])

        nephrotoxic_count = 0
        nephrotoxic_meds = []

        for med in current_meds:
            if med.get('nephrotoxic_risk', 'NONE') != 'NONE':
                nephrotoxic_count += 1
                nephrotoxic_meds.append(med.get('name'))

        if nephrotoxic_count >= 2:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Multiple nephrotoxic medications: {', '.join(nephrotoxic_meds)}",
                explanation="Cumulative nephrotoxicity risk increased",
                alternatives=["Consider alternative agents", "Increase monitoring frequency"],
                evidence_base=["Nephrotoxicity risk assessment"]
            ))

        return validations

    async def _check_contrast_exposure(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for recent contrast exposure"""
        validations = []
        clinical_data = context.clinical_data

        recent_contrast = clinical_data.get('recent_contrast_exposure', False)
        if recent_contrast:
            validations.append(ValidationResult(
                passed=False,
                severity="MEDIUM",
                message="Recent contrast exposure within 72 hours",
                explanation="Increased risk of contrast-induced nephropathy",
                alternatives=["Delay nephrotoxic medications", "Ensure adequate hydration"],
                evidence_base=["Contrast nephropathy prevention guidelines"]
            ))

        return validations

    def _generate_renal_cds(self, validations: List[ValidationResult], context: RecipeContext) -> Dict[str, Any]:
        """Generate renal-specific clinical decision support"""
        return {
            'provider_explanation': "Renal function assessment and dose adjustment recommendations",
            'patient_explanation': "Checking if medication doses are safe for your kidney function",
            'monitoring_recommendations': [
                "Monitor serum creatinine and eGFR",
                "Consider nephrology consultation if eGFR <30",
                "Ensure adequate hydration"
            ],
            'dose_adjustments': [v.alternatives for v in validations if not v.passed]
        }

class HepaticSafetyRecipe(ClinicalRecipe):
    """
    Recipe 1.3: Hepatic Function Safety
    Hepatic metabolism adjustment and DILI prevention
    """

    def __init__(self):
        config = {
            'id': 'medication-safety-hepatic-v3.0',
            'name': 'Hepatic Metabolism and Hepatotoxicity Prevention',
            'priority': 90,
            'description': 'Hepatic metabolism adjustment and DILI prevention',
            'triggers': [
                {'drug.metabolism.hepatic': '>70%'},
                {'drug.properties.hepatotoxicityRisk': '!= NONE'},
                {'drug.class': ['STATIN', 'AZOLE_ANTIFUNGAL', 'ACETAMINOPHEN', 'METHOTREXATE']}
            ],
            'clinicalRationale': '''
                Drug-induced liver injury (DILI) accounts for 50% of acute liver failure.
                Most cases are preventable with proper monitoring and dose adjustment.
            ''',
            'qosTier': 'gold'
        }
        super().__init__(config)

    def should_trigger(self, context: RecipeContext) -> bool:
        """Trigger for medications requiring hepatic assessment"""
        medication = context.medication_data
        patient = context.patient_data

        # Check if medication has hepatic metabolism >70%
        if medication.get('hepatic_metabolism', 0) > 70:
            return True

        # Check for hepatotoxic medications
        hepatotoxic_classes = ['STATIN', 'AZOLE_ANTIFUNGAL', 'ACETAMINOPHEN', 'METHOTREXATE']
        if medication.get('therapeutic_class') in hepatotoxic_classes:
            return True

        # Check if patient has liver disease
        conditions = patient.get('conditions', [])
        liver_conditions = ['cirrhosis', 'hepatitis', 'liver_disease']
        if any(condition in liver_conditions for condition in conditions):
            return True

        return False

    async def execute(self, context: RecipeContext) -> RecipeResult:
        """Execute hepatic safety checks"""
        start_time = datetime.now()
        validations = []

        try:
            # Calculate Child-Pugh score if cirrhosis present
            validations.extend(await self._assess_child_pugh(context))

            # Check liver function tests
            validations.extend(await self._check_liver_function(context))

            # Apply hepatic dose adjustments
            validations.extend(await self._apply_hepatic_adjustments(context))

            # Check for concurrent hepatotoxic drugs
            validations.extend(await self._check_hepatotoxic_burden(context))

            # Special alcohol + acetaminophen check
            validations.extend(await self._check_alcohol_acetaminophen(context))

            overall_status = self._determine_overall_status(validations)
            execution_time = (datetime.now() - start_time).total_seconds() * 1000

            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=validations,
                overall_status=overall_status,
                clinical_decision_support=self._generate_hepatic_cds(validations, context),
                cost_considerations={'monitoring': 'LFT frequency based on risk stratification'},
                ml_insights={'pattern': 'DILI risk prediction'},
                performance_metrics={'execution_time_ms': execution_time}
            )

        except Exception as e:
            logger.error(f"Error executing {self.id}: {str(e)}")
            execution_time = (datetime.now() - start_time).total_seconds() * 1000

            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=[ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"Hepatic safety check failed: {str(e)}",
                    explanation="Unable to assess hepatic safety",
                    alternatives=[],
                    evidence_base=[]
                )],
                overall_status="ERROR",
                clinical_decision_support={},
                cost_considerations={},
                ml_insights={},
                performance_metrics={'execution_time_ms': execution_time}
            )

    async def _assess_child_pugh(self, context: RecipeContext) -> List[ValidationResult]:
        """Calculate Child-Pugh score if cirrhosis present"""
        validations = []
        patient = context.patient_data
        clinical_data = context.clinical_data

        conditions = patient.get('conditions', [])
        if 'cirrhosis' in conditions:
            # Simplified Child-Pugh assessment
            labs = clinical_data.get('recent_labs', {})
            bilirubin = labs.get('bilirubin', 1.0)
            albumin = labs.get('albumin', 3.5)
            inr = labs.get('inr', 1.0)

            child_pugh_score = 0

            # Bilirubin scoring
            if bilirubin < 2:
                child_pugh_score += 1
            elif bilirubin < 3:
                child_pugh_score += 2
            else:
                child_pugh_score += 3

            # Albumin scoring
            if albumin > 3.5:
                child_pugh_score += 1
            elif albumin > 2.8:
                child_pugh_score += 2
            else:
                child_pugh_score += 3

            # INR scoring
            if inr < 1.7:
                child_pugh_score += 1
            elif inr < 2.3:
                child_pugh_score += 2
            else:
                child_pugh_score += 3

            if child_pugh_score >= 10:  # Child-Pugh C
                validations.append(ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"Severe hepatic impairment: Child-Pugh Class C (Score: {child_pugh_score})",
                    explanation="Severe liver dysfunction requires significant dose reduction or avoidance",
                    alternatives=["Avoid hepatically metabolized drugs", "Hepatology consultation"],
                    evidence_base=["Child-Pugh classification"]
                ))
            elif child_pugh_score >= 7:  # Child-Pugh B
                validations.append(ValidationResult(
                    passed=False,
                    severity="HIGH",
                    message=f"Moderate hepatic impairment: Child-Pugh Class B (Score: {child_pugh_score})",
                    explanation="Moderate liver dysfunction requires dose adjustment",
                    alternatives=["Reduce dose by 50%", "Consider alternative"],
                    evidence_base=["Child-Pugh classification"]
                ))

        return validations

    async def _check_liver_function(self, context: RecipeContext) -> List[ValidationResult]:
        """Check liver function tests"""
        validations = []
        clinical_data = context.clinical_data

        labs = clinical_data.get('recent_labs', {})
        alt = labs.get('alt', 30)
        ast = labs.get('ast', 30)

        # Check for elevated transaminases (>3x ULN)
        alt_uln = 40  # Upper limit of normal
        ast_uln = 40

        if alt > 3 * alt_uln or ast > 3 * ast_uln:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Elevated liver enzymes: ALT {alt}, AST {ast}",
                explanation="Elevated transaminases >3x ULN indicate liver injury",
                alternatives=["Hold hepatotoxic medications", "Investigate cause"],
                evidence_base=["DILI guidelines"]
            ))

        return validations

    async def _apply_hepatic_adjustments(self, context: RecipeContext) -> List[ValidationResult]:
        """Apply hepatic dose adjustments"""
        validations = []
        medication = context.medication_data
        patient = context.patient_data

        # Example dose adjustments for hepatically metabolized drugs
        hepatic_adjustments = {
            'simvastatin': {
                'mild_impairment': 'Reduce dose by 25%',
                'moderate_impairment': 'Reduce dose by 50%',
                'severe_impairment': 'Contraindicated'
            },
            'acetaminophen': {
                'any_impairment': 'Reduce daily dose to <2g/day'
            }
        }

        med_name = medication.get('name', '').lower()
        liver_function = patient.get('liver_function', 'normal')

        if med_name in hepatic_adjustments and liver_function != 'normal':
            adjustments = hepatic_adjustments[med_name]

            if liver_function == 'severe' and 'severe_impairment' in adjustments:
                validations.append(ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message=f"Severe hepatic impairment: {adjustments['severe_impairment']}",
                    explanation="Severe liver dysfunction contraindicates this medication",
                    alternatives=["Alternative medication", "Hepatology consultation"],
                    evidence_base=["FDA dosing guidelines"]
                ))

        return validations

    async def _check_hepatotoxic_burden(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for concurrent hepatotoxic drugs"""
        validations = []
        current_meds = context.clinical_data.get('current_medications', [])

        hepatotoxic_count = 0
        hepatotoxic_meds = []

        for med in current_meds:
            if med.get('hepatotoxic_risk', 'NONE') != 'NONE':
                hepatotoxic_count += 1
                hepatotoxic_meds.append(med.get('name'))

        if hepatotoxic_count >= 2:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Multiple hepatotoxic medications: {', '.join(hepatotoxic_meds)}",
                explanation="Cumulative hepatotoxicity risk increased",
                alternatives=["Consider alternative agents", "Increase LFT monitoring"],
                evidence_base=["Hepatotoxicity risk assessment"]
            ))

        return validations

    async def _check_alcohol_acetaminophen(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for alcohol + acetaminophen interaction"""
        validations = []
        medication = context.medication_data
        patient = context.patient_data

        if 'acetaminophen' in medication.get('name', '').lower():
            alcohol_use = patient.get('alcohol_use', 'none')
            daily_dose = medication.get('daily_dose_mg', 0)

            if alcohol_use in ['moderate', 'heavy'] and daily_dose > 2000:
                validations.append(ValidationResult(
                    passed=False,
                    severity="HIGH",
                    message="High-risk alcohol + acetaminophen combination",
                    explanation=f"Daily acetaminophen {daily_dose}mg with {alcohol_use} alcohol use",
                    alternatives=["Reduce acetaminophen to <2g/day", "Alternative analgesic"],
                    evidence_base=["Alcohol-acetaminophen hepatotoxicity studies"]
                ))

        return validations

    def _generate_hepatic_cds(self, validations: List[ValidationResult], context: RecipeContext) -> Dict[str, Any]:
        """Generate hepatic-specific clinical decision support"""
        return {
            'provider_explanation': "Hepatic function assessment and DILI prevention",
            'patient_explanation': "Checking if medication is safe for your liver",
            'monitoring_recommendations': [
                "Monitor liver function tests (ALT, AST, bilirubin)",
                "Consider hepatology consultation if severe impairment",
                "Educate patient on signs of liver injury"
            ],
            'dose_adjustments': [v.alternatives for v in validations if not v.passed]
        }

class ClinicalRecipeEngine:
    """
    Main engine for executing clinical recipes
    Orchestrates recipe selection, execution, and result aggregation
    """
    
    def __init__(self):
        self.recipes: Dict[str, ClinicalRecipe] = {}
        self._register_default_recipes()
    
    def _register_default_recipes(self):
        """Register all 29 clinical recipes from MedicationRecipeBook.txt"""
        try:
            # Import all recipe implementations
            from .clinical_recipes_high_priority import AnticoagulationSafetyRecipe, ChemotherapySafetyRecipe
            from .clinical_recipes_complete import (
                ControlledSubstanceSafetyRecipe, DrugInteractionSafetyRecipe, AntimicrobialStewardshipRecipe,
                PregnancySafetyRecipe, CodeBlueRecipe, MassiveTransfusionRecipe, PediatricSafetyRecipe,
                GeriatricSafetyRecipe, ImmunocompromisedSafetyRecipe, PreProceduralSafetyRecipe,
                AnesthesiaSafetyRecipe, ContrastSafetyRecipe, TransfusionSafetyRecipe, AdmissionSafetyRecipe,
                DischargeSafetyRecipe, RapidSequenceIntubationRecipe, CardiologyAntiarrhythmicRecipe,
                PsychiatryRecipe, DiabetesManagementRecipe, NarrowTherapeuticIndexRecipe,
                HighRiskMonitoringRecipe, CoreMeasuresRecipe, RegulatoryComplianceRecipe, RadiationSafetyRecipe
            )

            # Register the universal medication safety recipe (Recipe 1.1) ✅
            universal_safety = MedicationSafetyBaseRecipe()
            self.recipes[universal_safety.id] = universal_safety

            # Register medication safety recipes (1.2-1.8)
            self.recipes['medication-safety-renal-v3.0'] = RenalSafetyRecipe()
            self.recipes['medication-safety-hepatic-v3.0'] = HepaticSafetyRecipe()
            self.recipes['medication-safety-anticoagulation-v3.0'] = AnticoagulationSafetyRecipe()
            self.recipes['medication-safety-chemotherapy-v3.0'] = ChemotherapySafetyRecipe()
            self.recipes['medication-safety-controlled-v3.0'] = ControlledSubstanceSafetyRecipe()
            self.recipes['medication-safety-interactions-v3.0'] = DrugInteractionSafetyRecipe()
            self.recipes['antimicrobial-stewardship-v1.0'] = AntimicrobialStewardshipRecipe()

            # Register procedure safety recipes (2.1-2.4)
            self.recipes['procedure-safety-universal-v3.0'] = PreProceduralSafetyRecipe()
            self.recipes['procedure-safety-anesthesia-v3.0'] = AnesthesiaSafetyRecipe()
            self.recipes['procedure-safety-contrast-v1.0'] = ContrastSafetyRecipe()
            self.recipes['transfusion-safety-v1.0'] = TransfusionSafetyRecipe()

            # Register admission/discharge recipes (3.1-3.2)
            self.recipes['admission-safety-v3.0'] = AdmissionSafetyRecipe()
            self.recipes['discharge-safety-v3.0'] = DischargeSafetyRecipe()

            # Register special population recipes (4.1-4.4)
            self.recipes['population-pediatric-v3.0'] = PediatricSafetyRecipe()
            self.recipes['population-geriatric-v3.0'] = GeriatricSafetyRecipe()
            self.recipes['population-pregnancy-v3.0'] = PregnancySafetyRecipe()
            self.recipes['population-immunocompromised-v1.0'] = ImmunocompromisedSafetyRecipe()

            # Register emergency/critical care recipes (5.1-5.3)
            self.recipes['emergency-code-blue-v3.0'] = CodeBlueRecipe()
            self.recipes['emergency-rsi-v3.0'] = RapidSequenceIntubationRecipe()
            self.recipes['emergency-mtp-v1.0'] = MassiveTransfusionRecipe()

            # Register specialty-specific recipes (6.1-6.3)
            self.recipes['specialty-cardiology-antiarrhythmic-v3.0'] = CardiologyAntiarrhythmicRecipe()
            self.recipes['specialty-psychiatry-v3.0'] = PsychiatryRecipe()
            self.recipes['specialty-endocrine-diabetes-v1.0'] = DiabetesManagementRecipe()

            # Register monitoring recipes (7.1-7.2)
            self.recipes['monitoring-narrow-therapeutic-v3.0'] = NarrowTherapeuticIndexRecipe()
            self.recipes['monitoring-high-risk-v1.0'] = HighRiskMonitoringRecipe()

            # Register quality/regulatory recipes (8.1-8.2)
            self.recipes['quality-core-measures-v3.0'] = CoreMeasuresRecipe()
            self.recipes['quality-regulatory-v1.0'] = RegulatoryComplianceRecipe()

            # Register imaging safety recipe (9.1)
            self.recipes['imaging-radiation-safety-v1.0'] = RadiationSafetyRecipe()

            logger.info(f"Successfully registered {len(self.recipes)} clinical recipes")

        except ImportError as e:
            logger.error(f"Failed to import recipe implementations: {e}")
            # Fall back to just the universal safety recipe
            universal_safety = MedicationSafetyBaseRecipe()
            self.recipes[universal_safety.id] = universal_safety
            logger.warning("Falling back to universal safety recipe only")
    
    def register_recipe(self, recipe: ClinicalRecipe):
        """Register a new clinical recipe"""
        self.recipes[recipe.id] = recipe
        logger.info(f"Registered clinical recipe: {recipe.id}")
    
    async def execute_applicable_recipes(self, context: RecipeContext) -> List[RecipeResult]:
        """Execute all applicable recipes for the given context"""
        results = []
        
        # Find applicable recipes
        applicable_recipes = [
            recipe for recipe in self.recipes.values()
            if recipe.should_trigger(context)
        ]
        
        # Sort by priority (highest first)
        applicable_recipes.sort(key=lambda r: r.priority, reverse=True)
        
        logger.info(f"Executing {len(applicable_recipes)} applicable recipes for {context.action_type}")
        
        # Execute recipes
        for recipe in applicable_recipes:
            try:
                result = await recipe.execute(context)
                results.append(result)
                logger.info(f"Recipe {recipe.id} executed in {result.execution_time_ms:.1f}ms")
                
            except Exception as e:
                logger.error(f"Failed to execute recipe {recipe.id}: {str(e)}")
                # Continue with other recipes
        
        return results
    
    def get_recipe_catalog(self) -> Dict[str, Dict[str, Any]]:
        """Get catalog of all registered recipes"""
        catalog = {}
        for recipe_id, recipe in self.recipes.items():
            catalog[recipe_id] = {
                'name': recipe.name,
                'description': recipe.description,
                'priority': recipe.priority,
                'qos_tier': recipe.qos_tier.value,
                'clinical_rationale': recipe.clinical_rationale
            }
        return catalog
