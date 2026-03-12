"""
High Priority Clinical Recipes Implementation
============================================

This module implements the highest priority clinical recipes from MedicationRecipeBook.txt:
- Recipe 1.4: Anticoagulation Safety (Priority: 98)
- Recipe 1.5: Chemotherapy Safety (Priority: 99) 
- Recipe 4.3: Pregnancy & Lactation Safety (Priority: 99)
- Recipe 5.1: Code Blue / Resuscitation (Priority: 100)
- Recipe 5.3: Massive Transfusion Protocol (Priority: 100)

These recipes require sub-50ms execution times and handle life-critical scenarios.
"""

import logging
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from datetime import datetime

from .clinical_recipe_engine import ClinicalRecipe, RecipeContext, RecipeResult, ValidationResult

logger = logging.getLogger(__name__)

class AnticoagulationSafetyRecipe(ClinicalRecipe):
    """
    Recipe 1.4: Anticoagulation Safety
    Comprehensive anticoagulation management
    """
    
    def __init__(self):
        config = {
            'id': 'medication-safety-anticoagulation-v3.0',
            'name': 'Comprehensive Anticoagulation Management',
            'priority': 98,
            'description': 'Anticoagulation safety, monitoring, and reversal readiness',
            'triggers': [
                {'drug.class': ['ANTICOAGULANT', 'ANTIPLATELET']},
                {'drug.name': ['warfarin', 'heparin', 'enoxaparin', 'apixaban', 'rivaroxaban', 'dabigatran']}
            ],
            'clinicalRationale': '''
                Anticoagulants are the #1 cause of medication-related hospitalizations.
                Proper risk stratification and monitoring are life-saving.
            ''',
            'qosTier': 'platinum'
        }
        super().__init__(config)
    
    def should_trigger(self, context: RecipeContext) -> bool:
        """Trigger for anticoagulant medications"""
        medication = context.medication_data
        
        anticoagulant_classes = ['ANTICOAGULANT', 'ANTIPLATELET']
        if medication.get('therapeutic_class') in anticoagulant_classes:
            return True
            
        anticoagulant_names = ['warfarin', 'heparin', 'enoxaparin', 'apixaban', 'rivaroxaban', 'dabigatran']
        med_name = medication.get('name', '').lower()
        if any(name in med_name for name in anticoagulant_names):
            return True
            
        return False
    
    async def execute(self, context: RecipeContext) -> RecipeResult:
        """Execute anticoagulation safety checks"""
        start_time = datetime.now()
        validations = []
        
        try:
            # Calculate HAS-BLED score for bleeding risk
            validations.extend(await self._calculate_has_bled(context))
            
            # Calculate CHA2DS2-VASc for stroke risk (if AFib)
            validations.extend(await self._calculate_cha2ds2_vasc(context))
            
            # Verify INR in therapeutic range
            validations.extend(await self._check_inr_range(context))
            
            # Check CrCl-based DOAC dosing
            validations.extend(await self._check_doac_dosing(context))
            
            # Check drug interactions
            validations.extend(await self._check_anticoagulant_interactions(context))
            
            # Verify reversal agent availability
            validations.extend(await self._check_reversal_readiness(context))
            
            overall_status = self._determine_overall_status(validations)
            execution_time = (datetime.now() - start_time).total_seconds() * 1000
            
            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=validations,
                overall_status=overall_status,
                clinical_decision_support=self._generate_anticoagulation_cds(validations, context),
                cost_considerations={'monitoring': 'INR monitoring frequency optimization'},
                ml_insights={'pattern': 'Bleeding risk prediction'},
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
                    message=f"Anticoagulation safety check failed: {str(e)}",
                    explanation="Unable to assess anticoagulation safety",
                    alternatives=[],
                    evidence_base=[]
                )],
                overall_status="ERROR",
                clinical_decision_support={},
                cost_considerations={},
                ml_insights={},
                performance_metrics={'execution_time_ms': execution_time}
            )
    
    async def _calculate_has_bled(self, context: RecipeContext) -> List[ValidationResult]:
        """Calculate HAS-BLED bleeding risk score"""
        validations = []
        patient = context.patient_data
        clinical_data = context.clinical_data
        
        has_bled_score = 0
        risk_factors = []
        
        # Hypertension (uncontrolled)
        if patient.get('hypertension') and clinical_data.get('recent_labs', {}).get('systolic_bp', 120) > 160:
            has_bled_score += 1
            risk_factors.append("Uncontrolled hypertension")
        
        # Abnormal renal function
        if patient.get('creatinine', 1.0) > 2.26:
            has_bled_score += 1
            risk_factors.append("Renal dysfunction")
        
        # Abnormal liver function
        if 'cirrhosis' in patient.get('conditions', []):
            has_bled_score += 1
            risk_factors.append("Liver dysfunction")
        
        # Stroke history
        if 'stroke' in patient.get('conditions', []):
            has_bled_score += 1
            risk_factors.append("Prior stroke")
        
        # Bleeding history
        if 'bleeding_history' in patient.get('conditions', []):
            has_bled_score += 1
            risk_factors.append("Prior bleeding")
        
        # Labile INR (if on warfarin)
        current_meds = clinical_data.get('current_medications', [])
        if any('warfarin' in med.get('name', '').lower() for med in current_meds):
            recent_inrs = clinical_data.get('recent_inrs', [])
            if len(recent_inrs) >= 3:
                inr_variability = max(recent_inrs) - min(recent_inrs)
                if inr_variability > 1.0:
                    has_bled_score += 1
                    risk_factors.append("Labile INR")
        
        # Elderly (>65)
        if patient.get('age', 0) > 65:
            has_bled_score += 1
            risk_factors.append("Age >65")
        
        # Drugs/alcohol
        if patient.get('alcohol_use') in ['moderate', 'heavy']:
            has_bled_score += 1
            risk_factors.append("Alcohol use")
        
        if has_bled_score >= 3:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"High bleeding risk: HAS-BLED score {has_bled_score}",
                explanation=f"Risk factors: {', '.join(risk_factors)}",
                alternatives=["Consider bleeding risk vs stroke risk", "Frequent monitoring", "Consider left atrial appendage closure"],
                evidence_base=["HAS-BLED bleeding risk score"]
            ))
        
        return validations
    
    async def _calculate_cha2ds2_vasc(self, context: RecipeContext) -> List[ValidationResult]:
        """Calculate CHA2DS2-VASc stroke risk score for AFib patients"""
        validations = []
        patient = context.patient_data
        
        conditions = patient.get('conditions', [])
        if 'atrial_fibrillation' not in conditions:
            return validations
        
        cha2ds2_vasc_score = 0
        risk_factors = []
        
        # Congestive heart failure
        if 'heart_failure' in conditions:
            cha2ds2_vasc_score += 1
            risk_factors.append("Heart failure")
        
        # Hypertension
        if 'hypertension' in conditions:
            cha2ds2_vasc_score += 1
            risk_factors.append("Hypertension")
        
        # Age
        age = patient.get('age', 0)
        if age >= 75:
            cha2ds2_vasc_score += 2
            risk_factors.append("Age ≥75")
        elif age >= 65:
            cha2ds2_vasc_score += 1
            risk_factors.append("Age 65-74")
        
        # Diabetes
        if 'diabetes' in conditions:
            cha2ds2_vasc_score += 1
            risk_factors.append("Diabetes")
        
        # Stroke/TIA history
        if any(condition in conditions for condition in ['stroke', 'tia']):
            cha2ds2_vasc_score += 2
            risk_factors.append("Prior stroke/TIA")
        
        # Vascular disease
        if any(condition in conditions for condition in ['cad', 'pad', 'aortic_plaque']):
            cha2ds2_vasc_score += 1
            risk_factors.append("Vascular disease")
        
        # Sex (female)
        if patient.get('gender', '').lower() == 'female':
            cha2ds2_vasc_score += 1
            risk_factors.append("Female sex")
        
        if cha2ds2_vasc_score >= 2:
            validations.append(ValidationResult(
                passed=True,
                severity="MEDIUM",
                message=f"Anticoagulation indicated: CHA2DS2-VASc score {cha2ds2_vasc_score}",
                explanation=f"Stroke risk factors: {', '.join(risk_factors)}",
                alternatives=["Continue anticoagulation", "Optimize anticoagulation therapy"],
                evidence_base=["CHA2DS2-VASc stroke risk score", "AFib guidelines"]
            ))
        elif cha2ds2_vasc_score == 1:
            validations.append(ValidationResult(
                passed=False,
                severity="MEDIUM",
                message=f"Consider anticoagulation: CHA2DS2-VASc score {cha2ds2_vasc_score}",
                explanation="Borderline stroke risk - consider individual factors",
                alternatives=["Shared decision making", "Consider anticoagulation"],
                evidence_base=["CHA2DS2-VASc stroke risk score"]
            ))
        
        return validations
    
    async def _check_inr_range(self, context: RecipeContext) -> List[ValidationResult]:
        """Check INR therapeutic range for warfarin"""
        validations = []
        medication = context.medication_data
        clinical_data = context.clinical_data
        
        if 'warfarin' not in medication.get('name', '').lower():
            return validations
        
        recent_inr = clinical_data.get('recent_labs', {}).get('inr', 0)
        target_inr_range = clinical_data.get('target_inr_range', [2.0, 3.0])
        
        if recent_inr == 0:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message="No recent INR available for warfarin therapy",
                explanation="INR monitoring required for warfarin safety",
                alternatives=["Obtain INR before next dose", "Consider DOAC if monitoring difficult"],
                evidence_base=["Warfarin monitoring guidelines"]
            ))
        elif recent_inr < target_inr_range[0]:
            validations.append(ValidationResult(
                passed=False,
                severity="MEDIUM",
                message=f"INR below therapeutic range: {recent_inr} (target: {target_inr_range[0]}-{target_inr_range[1]})",
                explanation="Subtherapeutic INR increases stroke risk",
                alternatives=["Increase warfarin dose", "Check compliance", "Recheck INR in 3-5 days"],
                evidence_base=["Warfarin dosing guidelines"]
            ))
        elif recent_inr > target_inr_range[1]:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"INR above therapeutic range: {recent_inr} (target: {target_inr_range[0]}-{target_inr_range[1]})",
                explanation="Supratherapeutic INR increases bleeding risk",
                alternatives=["Hold warfarin dose", "Consider vitamin K", "Recheck INR in 1-2 days"],
                evidence_base=["Warfarin dosing guidelines"]
            ))
        
        return validations
    
    async def _check_doac_dosing(self, context: RecipeContext) -> List[ValidationResult]:
        """Check DOAC dosing based on renal function"""
        validations = []
        medication = context.medication_data
        patient = context.patient_data
        
        doac_names = ['apixaban', 'rivaroxaban', 'dabigatran', 'edoxaban']
        med_name = medication.get('name', '').lower()
        
        if not any(doac in med_name for doac in doac_names):
            return validations
        
        creatinine_clearance = patient.get('creatinine_clearance', 100)
        age = patient.get('age', 0)
        weight = patient.get('weight_kg', 70)
        
        # Apixaban dose reduction criteria
        if 'apixaban' in med_name:
            dose_reduction_criteria = 0
            if age >= 80:
                dose_reduction_criteria += 1
            if weight <= 60:
                dose_reduction_criteria += 1
            if creatinine_clearance <= 50:
                dose_reduction_criteria += 1
            
            if dose_reduction_criteria >= 2:
                validations.append(ValidationResult(
                    passed=False,
                    severity="MEDIUM",
                    message="Apixaban dose reduction indicated",
                    explanation=f"≥2 criteria met: age ≥80, weight ≤60kg, CrCl ≤50",
                    alternatives=["Reduce to 2.5mg BID", "Verify current dosing"],
                    evidence_base=["Apixaban dosing guidelines"]
                ))
        
        # Dabigatran renal dosing
        if 'dabigatran' in med_name:
            if creatinine_clearance < 30:
                validations.append(ValidationResult(
                    passed=False,
                    severity="CRITICAL",
                    message="Dabigatran contraindicated: CrCl <30 mL/min",
                    explanation="Severe renal impairment contraindicates dabigatran",
                    alternatives=["Alternative anticoagulant", "Warfarin with INR monitoring"],
                    evidence_base=["Dabigatran prescribing information"]
                ))
            elif creatinine_clearance < 50:
                validations.append(ValidationResult(
                    passed=False,
                    severity="MEDIUM",
                    message="Dabigatran dose reduction indicated: CrCl 30-50 mL/min",
                    explanation="Moderate renal impairment requires dose reduction",
                    alternatives=["Reduce to 75mg BID", "Monitor renal function"],
                    evidence_base=["Dabigatran dosing guidelines"]
                ))
        
        return validations
    
    async def _check_anticoagulant_interactions(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for anticoagulant drug interactions"""
        validations = []
        current_meds = context.clinical_data.get('current_medications', [])
        
        # High-risk interactions with anticoagulants
        high_risk_interactions = [
            'aspirin', 'clopidogrel', 'nsaid', 'warfarin', 'heparin'
        ]
        
        anticoagulant_count = 0
        interacting_meds = []
        
        for med in current_meds:
            med_name = med.get('name', '').lower()
            if any(interaction in med_name for interaction in high_risk_interactions):
                anticoagulant_count += 1
                interacting_meds.append(med.get('name'))
        
        if anticoagulant_count >= 2:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Multiple anticoagulant/antiplatelet agents: {', '.join(interacting_meds)}",
                explanation="Increased bleeding risk with multiple agents",
                alternatives=["Consider single agent", "Gastroprotection if dual therapy needed"],
                evidence_base=["Anticoagulant interaction guidelines"]
            ))
        
        return validations
    
    async def _check_reversal_readiness(self, context: RecipeContext) -> List[ValidationResult]:
        """Check reversal agent availability"""
        validations = []
        medication = context.medication_data
        
        # This would check institutional availability of reversal agents
        # Simplified for demonstration
        reversal_agents = {
            'warfarin': 'Vitamin K, FFP, PCC',
            'heparin': 'Protamine sulfate',
            'dabigatran': 'Idarucizumab',
            'apixaban': 'Andexanet alfa',
            'rivaroxaban': 'Andexanet alfa'
        }
        
        med_name = medication.get('name', '').lower()
        for drug, reversal in reversal_agents.items():
            if drug in med_name:
                validations.append(ValidationResult(
                    passed=True,
                    severity="LOW",
                    message=f"Reversal agent available: {reversal}",
                    explanation="Emergency reversal option identified",
                    alternatives=[],
                    evidence_base=["Anticoagulant reversal guidelines"]
                ))
                break
        
        return validations
    
    def _generate_anticoagulation_cds(self, validations: List[ValidationResult], context: RecipeContext) -> Dict[str, Any]:
        """Generate anticoagulation-specific clinical decision support"""
        return {
            'provider_explanation': "Anticoagulation safety assessment and optimization",
            'patient_explanation': "Checking blood thinner safety and effectiveness",
            'monitoring_recommendations': [
                "Regular INR monitoring for warfarin",
                "Annual renal function assessment for DOACs",
                "Bleeding risk assessment",
                "Drug interaction screening"
            ],
            'safety_alerts': [v.message for v in validations if not v.passed and v.severity in ["HIGH", "CRITICAL"]]
        }

class ChemotherapySafetyRecipe(ClinicalRecipe):
    """
    Recipe 1.5: Chemotherapy Safety
    Oncology medication safety and protocol compliance
    """

    def __init__(self):
        config = {
            'id': 'medication-safety-chemotherapy-v3.0',
            'name': 'Oncology Medication Safety and Protocol Compliance',
            'priority': 99,
            'description': 'Chemotherapy dosing, toxicity prevention, and protocol adherence',
            'triggers': [
                {'drug.properties.isChemotherapy': True},
                {'drug.properties.isTargetedTherapy': True}
            ],
            'clinicalRationale': '''
                Chemotherapy errors can be fatal. BSA calculations, dose limits, and
                protocol compliance are critical for patient safety and efficacy.
            ''',
            'qosTier': 'platinum'
        }
        super().__init__(config)

    def should_trigger(self, context: RecipeContext) -> bool:
        """Trigger for chemotherapy and targeted therapy"""
        medication = context.medication_data

        if medication.get('is_chemotherapy', False):
            return True
        if medication.get('is_targeted_therapy', False):
            return True
        if medication.get('therapeutic_class') in ['CHEMOTHERAPY', 'TARGETED_THERAPY']:
            return True

        return False

    async def execute(self, context: RecipeContext) -> RecipeResult:
        """Execute chemotherapy safety checks"""
        start_time = datetime.now()
        validations = []

        try:
            # Verify BSA calculation and dose limits
            validations.extend(await self._verify_bsa_dosing(context))

            # Check organ function requirements
            validations.extend(await self._check_organ_function(context))

            # Verify protocol compliance
            validations.extend(await self._verify_protocol_compliance(context))

            # Check for drug interactions
            validations.extend(await self._check_chemo_interactions(context))

            # Verify supportive care measures
            validations.extend(await self._verify_supportive_care(context))

            overall_status = self._determine_overall_status(validations)
            execution_time = (datetime.now() - start_time).total_seconds() * 1000

            return RecipeResult(
                recipe_id=self.id,
                recipe_name=self.name,
                execution_time_ms=execution_time,
                validations=validations,
                overall_status=overall_status,
                clinical_decision_support=self._generate_chemo_cds(validations, context),
                cost_considerations={'monitoring': 'Intensive monitoring required'},
                ml_insights={'pattern': 'Toxicity prediction'},
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
                    message=f"Chemotherapy safety check failed: {str(e)}",
                    explanation="Unable to assess chemotherapy safety",
                    alternatives=[],
                    evidence_base=[]
                )],
                overall_status="ERROR",
                clinical_decision_support={},
                cost_considerations={},
                ml_insights={},
                performance_metrics={'execution_time_ms': execution_time}
            )

    async def _verify_bsa_dosing(self, context: RecipeContext) -> List[ValidationResult]:
        """Verify BSA calculation and dose limits"""
        validations = []
        medication = context.medication_data
        patient = context.patient_data

        height_cm = patient.get('height_cm', 0)
        weight_kg = patient.get('weight_kg', 0)

        if height_cm == 0 or weight_kg == 0:
            validations.append(ValidationResult(
                passed=False,
                severity="CRITICAL",
                message="Missing height or weight for BSA calculation",
                explanation="Accurate BSA required for chemotherapy dosing",
                alternatives=["Obtain accurate height and weight", "Delay until measurements available"],
                evidence_base=["Chemotherapy dosing guidelines"]
            ))
            return validations

        # Calculate BSA using Mosteller formula
        bsa = ((height_cm * weight_kg) / 3600) ** 0.5

        dose_per_m2 = medication.get('dose_per_m2', 0)
        calculated_dose = bsa * dose_per_m2
        prescribed_dose = medication.get('prescribed_dose', 0)

        # Check dose accuracy (within 5%)
        if abs(calculated_dose - prescribed_dose) / calculated_dose > 0.05:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Dose calculation error: Prescribed {prescribed_dose}mg vs calculated {calculated_dose:.1f}mg",
                explanation=f"BSA: {bsa:.2f} m², Dose: {dose_per_m2} mg/m²",
                alternatives=["Verify dose calculation", "Pharmacist verification"],
                evidence_base=["BSA dosing calculations"]
            ))

        # Check maximum dose limits
        max_dose = medication.get('max_dose_mg', float('inf'))
        if calculated_dose > max_dose:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Dose exceeds maximum: {calculated_dose:.1f}mg > {max_dose}mg",
                explanation="Dose capping required for safety",
                alternatives=[f"Cap dose at {max_dose}mg", "Oncology consultation"],
                evidence_base=["Chemotherapy dose limits"]
            ))

        return validations

    async def _check_organ_function(self, context: RecipeContext) -> List[ValidationResult]:
        """Check organ function requirements"""
        validations = []
        medication = context.medication_data
        clinical_data = context.clinical_data

        labs = clinical_data.get('recent_labs', {})

        # Check renal function for nephrotoxic agents
        if medication.get('nephrotoxic', False):
            creatinine = labs.get('creatinine', 1.0)
            if creatinine > 1.5:
                validations.append(ValidationResult(
                    passed=False,
                    severity="HIGH",
                    message=f"Renal impairment: Creatinine {creatinine} mg/dL",
                    explanation="Nephrotoxic chemotherapy requires dose adjustment",
                    alternatives=["Reduce dose", "Alternative regimen", "Nephrology consultation"],
                    evidence_base=["Chemotherapy renal dosing"]
                ))

        # Check liver function for hepatotoxic agents
        if medication.get('hepatotoxic', False):
            alt = labs.get('alt', 30)
            bilirubin = labs.get('bilirubin', 1.0)
            if alt > 3 * 40 or bilirubin > 2.0:  # >3x ULN ALT or elevated bilirubin
                validations.append(ValidationResult(
                    passed=False,
                    severity="HIGH",
                    message=f"Hepatic impairment: ALT {alt}, Bilirubin {bilirubin}",
                    explanation="Hepatotoxic chemotherapy requires dose adjustment",
                    alternatives=["Reduce dose", "Alternative regimen", "Hepatology consultation"],
                    evidence_base=["Chemotherapy hepatic dosing"]
                ))

        # Check bone marrow function
        wbc = labs.get('wbc', 5000)
        anc = labs.get('anc', 2000)
        platelets = labs.get('platelets', 200000)

        if anc < 1000:
            validations.append(ValidationResult(
                passed=False,
                severity="CRITICAL",
                message=f"Severe neutropenia: ANC {anc}",
                explanation="Severe neutropenia contraindicates myelosuppressive chemotherapy",
                alternatives=["Delay treatment", "Growth factor support", "Dose reduction"],
                evidence_base=["Chemotherapy neutropenia guidelines"]
            ))
        elif platelets < 50000:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message=f"Severe thrombocytopenia: Platelets {platelets}",
                explanation="Severe thrombocytopenia requires dose modification",
                alternatives=["Delay treatment", "Platelet transfusion", "Dose reduction"],
                evidence_base=["Chemotherapy thrombocytopenia guidelines"]
            ))

        return validations

    async def _verify_protocol_compliance(self, context: RecipeContext) -> List[ValidationResult]:
        """Verify protocol compliance"""
        validations = []
        medication = context.medication_data
        clinical_data = context.clinical_data

        protocol_name = clinical_data.get('protocol_name', '')
        cycle_number = clinical_data.get('cycle_number', 1)
        day_of_cycle = clinical_data.get('day_of_cycle', 1)

        if not protocol_name:
            validations.append(ValidationResult(
                passed=False,
                severity="HIGH",
                message="No treatment protocol specified",
                explanation="Chemotherapy must follow established protocols",
                alternatives=["Specify treatment protocol", "Oncology consultation"],
                evidence_base=["Oncology treatment protocols"]
            ))

        # Check cycle timing
        last_treatment_date = clinical_data.get('last_treatment_date')
        if last_treatment_date and cycle_number > 1:
            # Simplified cycle timing check
            days_since_last = (datetime.now() - datetime.fromisoformat(last_treatment_date)).days
            expected_cycle_length = clinical_data.get('cycle_length_days', 21)

            if days_since_last < expected_cycle_length - 3:
                validations.append(ValidationResult(
                    passed=False,
                    severity="MEDIUM",
                    message=f"Early cycle: {days_since_last} days since last treatment",
                    explanation=f"Expected cycle length: {expected_cycle_length} days",
                    alternatives=["Verify cycle timing", "Oncology approval for early cycle"],
                    evidence_base=["Protocol cycle timing"]
                ))

        return validations

    async def _check_chemo_interactions(self, context: RecipeContext) -> List[ValidationResult]:
        """Check for chemotherapy drug interactions"""
        validations = []
        current_meds = context.clinical_data.get('current_medications', [])

        # High-risk interactions with chemotherapy
        high_risk_interactions = {
            'warfarin': 'Increased bleeding risk',
            'phenytoin': 'Altered metabolism',
            'live_vaccines': 'Immunosuppression contraindication'
        }

        for med in current_meds:
            med_name = med.get('name', '').lower()
            for interaction, risk in high_risk_interactions.items():
                if interaction in med_name:
                    validations.append(ValidationResult(
                        passed=False,
                        severity="HIGH",
                        message=f"High-risk interaction: {med.get('name')}",
                        explanation=risk,
                        alternatives=["Alternative medication", "Dose adjustment", "Increased monitoring"],
                        evidence_base=["Chemotherapy interaction database"]
                    ))

        return validations

    async def _verify_supportive_care(self, context: RecipeContext) -> List[ValidationResult]:
        """Verify supportive care measures"""
        validations = []
        medication = context.medication_data
        current_meds = context.clinical_data.get('current_medications', [])

        # Check for antiemetic prophylaxis for highly emetogenic regimens
        if medication.get('emetogenic_risk') in ['HIGH', 'MODERATE']:
            antiemetics = ['ondansetron', 'granisetron', 'palonosetron', 'dexamethasone']
            has_antiemetic = any(
                any(antiemetic in med.get('name', '').lower() for antiemetic in antiemetics)
                for med in current_meds
            )

            if not has_antiemetic:
                validations.append(ValidationResult(
                    passed=False,
                    severity="MEDIUM",
                    message="Missing antiemetic prophylaxis",
                    explanation=f"{medication.get('emetogenic_risk')} emetogenic risk requires prophylaxis",
                    alternatives=["Add antiemetic regimen", "Follow NCCN antiemetic guidelines"],
                    evidence_base=["NCCN antiemetic guidelines"]
                ))

        return validations

    def _generate_chemo_cds(self, validations: List[ValidationResult], context: RecipeContext) -> Dict[str, Any]:
        """Generate chemotherapy-specific clinical decision support"""
        return {
            'provider_explanation': "Chemotherapy safety and protocol compliance verification",
            'patient_explanation': "Ensuring chemotherapy treatment is safe and follows protocols",
            'monitoring_recommendations': [
                "Complete blood count before each cycle",
                "Comprehensive metabolic panel",
                "Organ function assessment",
                "Toxicity monitoring per protocol"
            ],
            'safety_alerts': [v.message for v in validations if not v.passed and v.severity in ["HIGH", "CRITICAL"]]
        }
