"""
Clinical Context Reasoner for Clinical Assertion Engine

Evaluates medications against clinical context including pregnancy/lactation,
disease contraindications, procedure conflicts, and special populations.
"""

import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)


class PregnancyCategory(Enum):
    """FDA Pregnancy Categories"""
    A = "A"  # Adequate studies show no risk
    B = "B"  # Animal studies show no risk, no human studies
    C = "C"  # Animal studies show risk, no human studies
    D = "D"  # Human studies show risk, benefits may warrant use
    X = "X"  # Studies show risk, risks outweigh benefits


@dataclass
class ClinicalContextAssertion:
    """Clinical context assertion result"""
    assertion_id: str
    context_type: str  # pregnancy, lactation, disease, procedure, age
    medication: str
    severity: str
    description: str
    clinical_rationale: str
    recommendations: List[str]
    confidence_score: float
    evidence_sources: List[str]
    special_considerations: List[str]


class ClinicalContextReasoner:
    """
    Advanced clinical context reasoner for special populations and conditions
    
    Features:
    - Pregnancy and lactation safety assessment
    - Disease-specific contraindications
    - Procedure-related medication conflicts
    - Age-specific considerations (pediatric, geriatric)
    - Renal/hepatic impairment adjustments
    - Drug-disease interactions
    """
    
    def __init__(self):
        self.pregnancy_categories = self._load_pregnancy_categories()
        self.lactation_safety = self._load_lactation_safety()
        self.disease_contraindications = self._load_disease_contraindications()
        self.procedure_conflicts = self._load_procedure_conflicts()
        self.age_considerations = self._load_age_considerations()
        self.organ_impairment_adjustments = self._load_organ_impairment_adjustments()
        
        logger.info("Clinical Context Reasoner initialized")
    
    async def check_clinical_context(self, patient_id: str, medication_ids: List[str],
                                   patient_context: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """
        Check medications against clinical context
        
        Args:
            patient_id: Patient identifier
            medication_ids: List of medication identifiers
            patient_context: Patient clinical context
            
        Returns:
            List of clinical context assertions
        """
        try:
            logger.info(f"Starting clinical context check for patient {patient_id}")
            logger.info(f"Patient context type: {type(patient_context)}, value: {patient_context}")
            logger.info(f"Medication IDs: {medication_ids}")
            
            assertions = []
            
            # Ensure patient_context is a dictionary
            if not patient_context:
                logger.info("No patient context provided, using empty dict")
                patient_context = {}
            elif isinstance(patient_context, list):
                logger.warning(f"Received patient_context as list instead of dict, converting to empty dict")
                patient_context = {}
                
            # Log the structure of patient_context
            logger.info(f"Patient context keys: {list(patient_context.keys()) if isinstance(patient_context, dict) else 'No keys (not a dict)'}")
            if 'demographics' in patient_context:
                logger.info(f"Demographics type: {type(patient_context['demographics'])}, value: {patient_context['demographics']}")
            if 'conditions' in patient_context:
                logger.info(f"Conditions type: {type(patient_context['conditions'])}, value: {patient_context['conditions']}")
            if 'medications' in patient_context:
                logger.info(f"Medications type: {type(patient_context['medications'])}, value: {patient_context['medications']}")
                
            for medication in medication_ids:
                
                # 1. Pregnancy safety check
                pregnancy_assertions = await self._check_pregnancy_safety(
                    medication, patient_context
                )
                assertions.extend(pregnancy_assertions)
                
                # 2. Lactation safety check
                lactation_assertions = await self._check_lactation_safety(
                    medication, patient_context
                )
                assertions.extend(lactation_assertions)
                
                # 3. Disease contraindications
                disease_assertions = await self._check_disease_contraindications(
                    medication, patient_context
                )
                assertions.extend(disease_assertions)
                
                # 4. Procedure conflicts
                procedure_assertions = await self._check_procedure_conflicts(
                    medication, patient_context
                )
                assertions.extend(procedure_assertions)
                
                # 5. Age-specific considerations
                age_assertions = await self._check_age_considerations(
                    medication, patient_context
                )
                assertions.extend(age_assertions)
                
                # 6. Organ impairment adjustments
                impairment_assertions = await self._check_organ_impairment(
                    medication, patient_context
                )
                assertions.extend(impairment_assertions)
            
            # Convert to assertion format
            formatted_assertions = []
            for assertion in assertions:
                formatted_assertion = {
                    "type": "clinical_context",
                    "severity": assertion.severity,
                    "title": f"{assertion.context_type.title()} Consideration",
                    "description": assertion.description,
                    "explanation": assertion.clinical_rationale,
                    "confidence": assertion.confidence_score,
                    "evidence_sources": assertion.evidence_sources,
                    "recommendations": assertion.recommendations,
                    "metadata": {
                        "context_type": assertion.context_type,
                        "medication": assertion.medication,
                        "special_considerations": assertion.special_considerations
                    }
                }
                formatted_assertions.append(formatted_assertion)
            
            logger.info(f"Found {len(formatted_assertions)} clinical context issues for patient {patient_id}")
            return {
                'assertions': formatted_assertions,
                'confidence_score': 0.9,
                'metadata': {'reasoner_type': 'clinical_context'}
            }
            
        except Exception as e:
            logger.error(f"Error checking clinical context: {e}")
            return {
                'assertions': [],
                'confidence_score': 0.0,
                'metadata': {'reasoner_type': 'clinical_context', 'error': str(e)}
            }
    
    async def _check_pregnancy_safety(self, medication: str, 
                                    patient_context: Dict[str, Any]) -> List[ClinicalContextAssertion]:
        """Check pregnancy safety for medication"""
        logger.info(f"_check_pregnancy_safety for {medication} - context type: {type(patient_context)}")
        assertions = []
        
        # Ensure patient_context is a dictionary
        if not isinstance(patient_context, dict):
            logger.warning(f"Received non-dict patient_context in _check_pregnancy_safety: {type(patient_context)}")
            return assertions
        
        # Extract demographics from nested structure if available
        demographics = {}
        if "demographics" in patient_context:
            demographics = patient_context.get('demographics', {})
        elif "patient_demographics" in patient_context:
            demographics = patient_context.get('patient_demographics', {})
        
        # Check if patient is pregnant or of childbearing age
        is_pregnant = patient_context.get("pregnancy_status") == "pregnant"
        # Also check in demographics
        if not is_pregnant and demographics.get("pregnancy_status") == "pregnant":
            is_pregnant = True
        
        is_childbearing_age = (
            demographics.get("gender") == "female" and
            18 <= demographics.get("age", 0) <= 45
        )
        
        if not (is_pregnant or is_childbearing_age):
            return assertions
        
        pregnancy_info = self.pregnancy_categories.get(medication.lower())
        if not pregnancy_info:
            return assertions
        
        category = pregnancy_info["category"]
        severity = self._get_pregnancy_severity(category, is_pregnant)
        
        if severity in ["high", "critical"]:
            assertion = ClinicalContextAssertion(
                assertion_id=f"pregnancy_{medication}_{patient_context.get('patient_id', 'unknown')}",
                context_type="pregnancy",
                medication=medication,
                severity=severity,
                description=f"Pregnancy Category {category.value}: {pregnancy_info['description']}",
                clinical_rationale=pregnancy_info["rationale"],
                recommendations=pregnancy_info["recommendations"],
                confidence_score=0.9,
                evidence_sources=["FDA Pregnancy Categories", "Clinical Studies"],
                special_considerations=pregnancy_info.get("special_considerations", [])
            )
            assertions.append(assertion)
        
        return assertions
    
    async def _check_lactation_safety(self, medication: str,
                                    patient_context: Dict[str, Any]) -> List[ClinicalContextAssertion]:
        """Check lactation safety for medication"""
        logger.info(f"_check_lactation_safety for {medication} - context type: {type(patient_context)}")
        assertions = []
        
        # Ensure patient_context is a dictionary
        if not isinstance(patient_context, dict):
            logger.warning(f"Received non-dict patient_context in _check_lactation_safety: {type(patient_context)}")
            return assertions
        
        # Extract demographics from nested structure if available
        demographics = {}
        if "demographics" in patient_context:
            demographics = patient_context.get('demographics', {})
        elif "patient_demographics" in patient_context:
            demographics = patient_context.get('patient_demographics', {})
    
        # Check lactation status in multiple possible locations
        is_lactating = (patient_context.get("lactation_status") == "lactating" or 
                        demographics.get("lactation_status") == "lactating")
        if not is_lactating:
            return assertions
        
        lactation_info = self.lactation_safety.get(medication.lower())
        if not lactation_info:
            return assertions
        
        if lactation_info["risk_level"] in ["high", "contraindicated"]:
            assertion = ClinicalContextAssertion(
                assertion_id=f"lactation_{medication}_{patient_context.get('patient_id', 'unknown')}",
                context_type="lactation",
                medication=medication,
                severity="high" if lactation_info["risk_level"] == "high" else "critical",
                description=f"Lactation risk: {lactation_info['description']}",
                clinical_rationale=lactation_info["rationale"],
                recommendations=lactation_info["recommendations"],
                confidence_score=0.85,
                evidence_sources=["LactMed Database", "AAP Guidelines"],
                special_considerations=lactation_info.get("monitoring", [])
            )
            assertions.append(assertion)
        
        return assertions
    
    async def _check_disease_contraindications(self, medication: str,
                                             patient_context: Dict[str, Any]) -> List[ClinicalContextAssertion]:
        """Check disease-specific contraindications"""
        logger.info(f"_check_disease_contraindications for {medication} - context type: {type(patient_context)}")
        assertions = []
        
        # Ensure patient_context is a dictionary
        if not isinstance(patient_context, dict):
            logger.warning(f"Received non-dict patient_context in _check_disease_contraindications: {type(patient_context)}")
            return assertions
        
        # Try all possible locations for conditions data
        patient_conditions = patient_context.get("active_conditions", [])
        
        # If no conditions found at top level, check in the 'conditions' key
        if not patient_conditions and "conditions" in patient_context:
            patient_conditions = patient_context.get("conditions", [])
            
        # Also check for active_diagnoses which is the structure we're actually getting
        if not patient_conditions and "active_diagnoses" in patient_context:
            patient_conditions = patient_context.get("active_diagnoses", [])
            
        if not patient_conditions:
            return assertions
        
        contraindications = self.disease_contraindications.get(medication.lower(), {})
        
        for condition in patient_conditions:
            # Handle condition as either a string or a dictionary with 'code' and 'display' fields
            condition_match = None
            condition_display_name = ""
            
            if isinstance(condition, dict):
                condition_code = condition.get('code', '').lower()
                condition_display = condition.get('display', '').lower()
                condition_display_name = condition.get('display', condition_code)
                
                # Check both code and display name against contraindications
                if condition_code in contraindications:
                    condition_match = condition_code
                elif condition_display in contraindications:
                    condition_match = condition_display
            else:
                # Handle condition as a simple string
                condition_lower = str(condition).lower()
                condition_display_name = str(condition)
                if condition_lower in contraindications:
                    condition_match = condition_lower
            
            # If we found a match, create an assertion
            if condition_match:
                contraindication_info = contraindications[condition_match]
                
                assertion = ClinicalContextAssertion(
                    assertion_id=f"disease_{medication}_{condition_match}_{patient_context.get('patient_id', 'unknown')}",
                    context_type="disease_contraindication",
                    medication=medication,
                    severity=contraindication_info["severity"],
                    description=f"Contraindicated in {condition_display_name}: {contraindication_info['description']}",
                    clinical_rationale=contraindication_info["rationale"],
                    recommendations=contraindication_info["recommendations"],
                    confidence_score=contraindication_info["confidence"],
                    evidence_sources=contraindication_info["evidence_sources"],
                    special_considerations=contraindication_info.get("monitoring", [])
                )
                assertions.append(assertion)
        
        return assertions
    
    async def _check_procedure_conflicts(self, medication: str,
                                   patient_context: Dict[str, Any]) -> List[ClinicalContextAssertion]:
        """Check procedure-related medication conflicts"""
        logger.info(f"_check_procedure_conflicts for {medication} - context type: {type(patient_context)}")
        assertions = []
        
        # Ensure patient_context is a dictionary
        if not isinstance(patient_context, dict):
            logger.warning(f"Received non-dict patient_context in _check_procedure_conflicts: {type(patient_context)}")
            return assertions
        
        # Check for procedures in all possible locations
        upcoming_procedures = patient_context.get("upcoming_procedures", [])
        
        # If no procedures found at top level, check in the 'procedures' key
        if not upcoming_procedures and "procedures" in patient_context:
            upcoming_procedures = patient_context.get("procedures", [])
            
        # Also look in scheduled_procedures if available
        if not upcoming_procedures and "scheduled_procedures" in patient_context:
            upcoming_procedures = patient_context.get("scheduled_procedures", [])
            
        if not upcoming_procedures:
            return assertions
        
        procedure_conflicts = self.procedure_conflicts.get(medication.lower(), {})
        
        for procedure in upcoming_procedures:
            # Handle procedure as either a string or a dictionary
            procedure_name = procedure
            if isinstance(procedure, dict):
                procedure_name = procedure.get('display', procedure.get('code', ''))
                procedure_lower = procedure_name.lower()
            else:
                procedure_lower = str(procedure).lower()
                
            if procedure_lower in procedure_conflicts:
                conflict_info = procedure_conflicts[procedure_lower]
                
                assertion = ClinicalContextAssertion(
                    assertion_id=f"procedure_{medication}_{procedure}_{patient_context.get('patient_id', 'unknown')}",
                    context_type="procedure_conflict",
                    medication=medication,
                    severity=conflict_info["severity"],
                    description=f"Procedure conflict with {procedure}: {conflict_info['description']}",
                    clinical_rationale=conflict_info["rationale"],
                    recommendations=conflict_info["recommendations"],
                    confidence_score=conflict_info["confidence"],
                    evidence_sources=conflict_info["evidence_sources"],
                    special_considerations=conflict_info.get("timing", [])
                )
                assertions.append(assertion)
        
        return assertions
    
    async def _check_age_considerations(self, medication: str,
                                  patient_context: Dict[str, Any]) -> List[ClinicalContextAssertion]:
        """Check age-specific considerations"""
        logger.info(f"_check_age_considerations for {medication} - context type: {type(patient_context)}")
        assertions = []
        
        # Ensure patient_context is a dictionary
        if not isinstance(patient_context, dict):
            logger.warning(f"Received non-dict patient_context in _check_age_considerations: {type(patient_context)}")
            return assertions
        
        # Try to get age from either top-level or patient_demographics
        age = patient_context.get("age")
        
        # If age not found at top level, check in patient_demographics (which is the structure we're actually getting)
        if age is None and "patient_demographics" in patient_context:
            patient_demographics = patient_context.get("patient_demographics", {})
            age = patient_demographics.get("age")
            
        # Also check in demographics as a fallback
        if age is None and "demographics" in patient_context:
            demographics = patient_context.get("demographics", {})
            age = demographics.get("age")
            
        if age is None:
            logger.debug(f"No age information found in patient context for medication {medication}")
            return assertions
            
        age_info = self.age_considerations.get(medication.lower())
        if not age_info:
            return assertions
        
        # Check pediatric considerations
        if age < 18 and "pediatric" in age_info:
            pediatric_info = age_info["pediatric"]
            if pediatric_info.get("contraindicated", False):
                assertion = ClinicalContextAssertion(
                    assertion_id=f"pediatric_{medication}_{patient_context.get('patient_id', 'unknown')}",
                    context_type="pediatric",
                    medication=medication,
                    severity=pediatric_info["severity"],
                    description=f"Pediatric consideration: {pediatric_info['description']}",
                    clinical_rationale=pediatric_info["rationale"],
                    recommendations=pediatric_info["recommendations"],
                    confidence_score=pediatric_info["confidence"],
                    evidence_sources=pediatric_info["evidence_sources"],
                    special_considerations=pediatric_info.get("monitoring", [])
                )
                assertions.append(assertion)
        
        # Check geriatric considerations
        if age >= 65 and "geriatric" in age_info:
            geriatric_info = age_info["geriatric"]
            if geriatric_info.get("high_risk", False):
                assertion = ClinicalContextAssertion(
                    assertion_id=f"geriatric_{medication}_{patient_context.get('patient_id', 'unknown')}",
                    context_type="geriatric",
                    medication=medication,
                    severity=geriatric_info["severity"],
                    description=f"Geriatric consideration: {geriatric_info['description']}",
                    clinical_rationale=geriatric_info["rationale"],
                    recommendations=geriatric_info["recommendations"],
                    confidence_score=geriatric_info["confidence"],
                    evidence_sources=geriatric_info["evidence_sources"],
                    special_considerations=geriatric_info.get("monitoring", [])
                )
                assertions.append(assertion)
        
        return assertions
    
    async def _check_organ_impairment(self, medication: str,
                                     patient_context: Dict[str, Any]) -> List[ClinicalContextAssertion]:
        """Check organ impairment adjustments"""
        logger.info(f"_check_organ_impairment for {medication} - context type: {type(patient_context)}")
        assertions = []
        
        # Ensure patient_context is a dictionary
        if not isinstance(patient_context, dict):
            logger.warning(f"Received non-dict patient_context in _check_organ_impairment: {type(patient_context)}")
            return assertions
        
        # Try to get organ function data from all possible locations
        kidney_function = patient_context.get("kidney_function", "normal")
        liver_function = patient_context.get("liver_function", "normal")
        
        # Check in laboratory_values which is the structure we're actually getting
        if "laboratory_values" in patient_context:
            lab_values = patient_context.get("laboratory_values", {})
            if isinstance(lab_values, dict):
                if "kidney_function" in lab_values:
                    kidney_function = lab_values.get("kidney_function", kidney_function)
                if "liver_function" in lab_values:
                    liver_function = lab_values.get("liver_function", liver_function)
    
        # If not found at top level, check within other nested structures
        if isinstance(patient_context.get("lab_results"), list):
            for result in patient_context["lab_results"]:
                if isinstance(result, dict):
                    if "kidney_function" in result:
                        kidney_function = result.get("kidney_function", kidney_function)
                    if "liver_function" in result:
                        liver_function = result.get("liver_function", liver_function)

        if isinstance(patient_context.get("clinical_data"), dict):
            clinical_data = patient_context["clinical_data"]
            if "kidney_function" in clinical_data:
                kidney_function = clinical_data.get("kidney_function", kidney_function)
            if "liver_function" in clinical_data:
                liver_function = clinical_data.get("liver_function", liver_function)
        
        impairment_info = self.organ_impairment_adjustments.get(medication.lower())
        if not impairment_info:
            return assertions
        
        # Check renal impairment
        if kidney_function in ["mild_impairment", "moderate_impairment", "severe_impairment"] and "renal" in impairment_info:
            renal_info = impairment_info["renal"]
            if renal_info.get("adjustment_required", False):
                assertion = ClinicalContextAssertion(
                    assertion_id=f"renal_{medication}_{patient_context.get('patient_id', 'unknown')}",
                    context_type="renal_impairment",
                    medication=medication,
                    severity=renal_info["severity"],
                    description=f"Renal adjustment needed: {renal_info['description']}",
                    clinical_rationale=renal_info["rationale"],
                    recommendations=renal_info["recommendations"],
                    confidence_score=renal_info["confidence"],
                    evidence_sources=renal_info["evidence_sources"],
                    special_considerations=renal_info.get("monitoring", [])
                )
                assertions.append(assertion)
        
        # Check hepatic impairment
        if liver_function in ["mild_impairment", "moderate_impairment", "severe_impairment"] and "hepatic" in impairment_info:
            hepatic_info = impairment_info["hepatic"]
            if hepatic_info.get("adjustment_required", False):
                assertion = ClinicalContextAssertion(
                    assertion_id=f"hepatic_{medication}_{patient_context.get('patient_id', 'unknown')}",
                    context_type="hepatic_impairment",
                    medication=medication,
                    severity=hepatic_info["severity"],
                    description=f"Hepatic adjustment needed: {hepatic_info['description']}",
                    clinical_rationale=hepatic_info["rationale"],
                    recommendations=hepatic_info["recommendations"],
                    confidence_score=hepatic_info["confidence"],
                    evidence_sources=hepatic_info["evidence_sources"],
                    special_considerations=hepatic_info.get("monitoring", [])
                )
                assertions.append(assertion)
        
        return assertions
    
    def _get_pregnancy_severity(self, category: PregnancyCategory, is_pregnant: bool) -> str:
        """Determine severity based on pregnancy category"""
        if category == PregnancyCategory.X:
            return "critical"
        elif category == PregnancyCategory.D:
            return "high"
        elif category == PregnancyCategory.C and is_pregnant:
            return "moderate"
        else:
            return "low"
    
    def _load_pregnancy_categories(self) -> Dict[str, Dict[str, Any]]:
        """Load pregnancy category database"""
        return {
            "warfarin": {
                "category": PregnancyCategory.X,
                "description": "Contraindicated in pregnancy - teratogenic",
                "rationale": "Causes fetal warfarin syndrome and bleeding complications",
                "recommendations": ["Discontinue immediately", "Switch to heparin", "Contraception counseling"],
                "special_considerations": ["Teratogenic throughout pregnancy"]
            },
            "lisinopril": {
                "category": PregnancyCategory.D,
                "description": "Avoid in pregnancy - fetal toxicity",
                "rationale": "ACE inhibitors cause fetal renal dysfunction and oligohydramnios",
                "recommendations": ["Discontinue", "Switch to methyldopa or labetalol"],
                "special_considerations": ["Especially dangerous in 2nd/3rd trimester"]
            },
            "metformin": {
                "category": PregnancyCategory.B,
                "description": "Generally safe in pregnancy",
                "rationale": "No evidence of teratogenicity, may reduce gestational diabetes",
                "recommendations": ["Continue if indicated", "Monitor glucose closely"],
                "special_considerations": ["Preferred oral antidiabetic in pregnancy"]
            }
        }
    
    def _load_lactation_safety(self) -> Dict[str, Dict[str, Any]]:
        """Load lactation safety database"""
        return {
            "warfarin": {
                "risk_level": "low",
                "description": "Compatible with breastfeeding",
                "rationale": "Minimal transfer to breast milk",
                "recommendations": ["Continue breastfeeding", "Monitor infant for bleeding"],
                "monitoring": ["INR monitoring", "Infant bleeding assessment"]
            },
            "lithium": {
                "risk_level": "high",
                "description": "Significant risk to nursing infant",
                "rationale": "High milk-to-plasma ratio, infant toxicity reported",
                "recommendations": ["Avoid breastfeeding", "Consider alternative mood stabilizer"],
                "monitoring": ["Infant lithium levels if breastfeeding continues"]
            }
        }
    
    def _load_disease_contraindications(self) -> Dict[str, Dict[str, Dict[str, Any]]]:
        """Load disease-specific contraindications"""
        return {
            "metformin": {
                "kidney_disease": {
                    "severity": "high",
                    "description": "Contraindicated in severe renal impairment",
                    "rationale": "Risk of lactic acidosis due to drug accumulation",
                    "recommendations": ["Discontinue if eGFR <30", "Use alternative antidiabetic"],
                    "confidence": 0.95,
                    "evidence_sources": ["FDA labeling", "Clinical guidelines"],
                    "monitoring": ["Renal function", "Lactate levels"]
                }
            },
            "aspirin": {
                "peptic_ulcer": {
                    "severity": "high",
                    "description": "Contraindicated in active peptic ulcer disease",
                    "rationale": "Increased risk of GI bleeding and ulcer perforation",
                    "recommendations": ["Avoid aspirin", "Use alternative antiplatelet if needed"],
                    "confidence": 0.90,
                    "evidence_sources": ["Clinical studies", "GI guidelines"],
                    "monitoring": ["GI symptoms", "Hemoglobin"]
                }
            }
        }
    
    def _load_procedure_conflicts(self) -> Dict[str, Dict[str, Dict[str, Any]]]:
        """Load procedure-related conflicts"""
        return {
            "warfarin": {
                "surgery": {
                    "severity": "high",
                    "description": "Bleeding risk with surgical procedures",
                    "rationale": "Anticoagulation increases perioperative bleeding risk",
                    "recommendations": ["Bridge with heparin", "Hold 5 days before surgery"],
                    "confidence": 0.95,
                    "evidence_sources": ["Surgical guidelines", "Anticoagulation protocols"],
                    "timing": ["Hold 5 days pre-op", "Resume 24-48 hours post-op"]
                }
            }
        }
    
    def _load_age_considerations(self) -> Dict[str, Dict[str, Dict[str, Any]]]:
        """Load age-specific considerations"""
        return {
            "aspirin": {
                "pediatric": {
                    "contraindicated": True,
                    "severity": "critical",
                    "description": "Risk of Reye's syndrome in children",
                    "rationale": "Aspirin linked to Reye's syndrome in viral illnesses",
                    "recommendations": ["Use acetaminophen or ibuprofen instead"],
                    "confidence": 0.95,
                    "evidence_sources": ["FDA warnings", "Pediatric guidelines"]
                }
            },
            "diphenhydramine": {
                "geriatric": {
                    "high_risk": True,
                    "severity": "moderate",
                    "description": "Anticholinergic effects in elderly",
                    "rationale": "Increased risk of confusion, falls, and cognitive impairment",
                    "recommendations": ["Avoid in elderly", "Use alternative antihistamine"],
                    "confidence": 0.85,
                    "evidence_sources": ["Beers Criteria", "Geriatric guidelines"],
                    "monitoring": ["Cognitive function", "Fall risk"]
                }
            }
        }
    
    def _load_organ_impairment_adjustments(self) -> Dict[str, Dict[str, Dict[str, Any]]]:
        """Load organ impairment adjustment requirements"""
        return {
            "digoxin": {
                "renal": {
                    "adjustment_required": True,
                    "severity": "high",
                    "description": "Dose reduction required in renal impairment",
                    "rationale": "Primarily renally eliminated, risk of toxicity",
                    "recommendations": ["Reduce dose by 50% if CrCl <50", "Monitor levels closely"],
                    "confidence": 0.95,
                    "evidence_sources": ["Pharmacokinetic studies", "Clinical guidelines"],
                    "monitoring": ["Digoxin levels", "Renal function", "ECG"]
                }
            },
            "acetaminophen": {
                "hepatic": {
                    "adjustment_required": True,
                    "severity": "moderate",
                    "description": "Caution in hepatic impairment",
                    "rationale": "Hepatic metabolism, risk of hepatotoxicity",
                    "recommendations": ["Reduce dose", "Limit duration", "Monitor liver function"],
                    "confidence": 0.85,
                    "evidence_sources": ["Hepatology guidelines", "Drug labeling"],
                    "monitoring": ["Liver function tests", "Total daily dose"]
                }
            }
        }
