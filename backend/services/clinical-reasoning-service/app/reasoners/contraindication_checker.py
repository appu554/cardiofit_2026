"""
Contraindication Checker for CAE

Checks for medical contraindications based on patient conditions
"""

import logging
from typing import Dict, Any, List, Optional

logger = logging.getLogger(__name__)

class ContraindicationChecker:
    """Check for medical contraindications"""
    
    def __init__(self):
        self.contraindication_database = self._initialize_contraindication_database()
        logger.info("Contraindication Checker initialized")
    
    def _initialize_contraindication_database(self) -> Dict[str, Dict]:
        """Initialize contraindication database"""
        return {
            "ibuprofen": {
                "absolute_contraindications": [
                    "active_gi_bleeding",
                    "severe_heart_failure",
                    "severe_kidney_disease"
                ],
                "relative_contraindications": [
                    "mild_kidney_impairment",
                    "hypertension",
                    "coronary_artery_disease",
                    "elderly_over_65"
                ],
                "warnings": [
                    "Increased bleeding risk with anticoagulants",
                    "May worsen kidney function",
                    "Cardiovascular risk in elderly"
                ]
            },
            "ciprofloxacin": {
                "absolute_contraindications": [
                    "tendon_disorders",
                    "myasthenia_gravis"
                ],
                "relative_contraindications": [
                    "elderly_over_60",
                    "kidney_impairment",
                    "seizure_disorder"
                ],
                "warnings": [
                    "QT prolongation risk",
                    "Tendon rupture risk in elderly",
                    "Drug interactions with warfarin"
                ]
            },
            "amiodarone": {
                "absolute_contraindications": [
                    "severe_bradycardia",
                    "av_block_high_grade"
                ],
                "relative_contraindications": [
                    "thyroid_disease",
                    "liver_disease",
                    "lung_disease"
                ],
                "warnings": [
                    "Multiple drug interactions",
                    "Thyroid monitoring required",
                    "Pulmonary toxicity risk"
                ]
            }
        }
    
    async def check_contraindications(self, patient_id: str, medication: str,
                                    patient_conditions: List[str],
                                    patient_context: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Check for contraindications based on patient conditions"""
        
        logger.info(f"Checking contraindications for {medication}")
        
        contraindications = []
        medication_data = self.contraindication_database.get(medication.lower())
        
        if not medication_data:
            return contraindications
        
        # Check absolute contraindications
        absolute_contras = medication_data.get("absolute_contraindications", [])
        for condition in patient_conditions:
            if condition.lower() in [ac.lower() for ac in absolute_contras]:
                contraindications.append({
                    "type": "absolute",
                    "severity": "CRITICAL",
                    "condition": condition,
                    "description": f"Absolute contraindication: {medication} with {condition}"
                })
        
        # Check relative contraindications
        relative_contras = medication_data.get("relative_contraindications", [])
        for condition in patient_conditions:
            if condition.lower() in [rc.lower() for rc in relative_contras]:
                contraindications.append({
                    "type": "relative",
                    "severity": "MODERATE",
                    "condition": condition,
                    "description": f"Relative contraindication: {medication} with {condition}"
                })
        
        # Check age-related contraindications
        age = patient_context.get("age", 0)
        if age >= 65 and "elderly_over_65" in relative_contras:
            contraindications.append({
                "type": "age_related",
                "severity": "MODERATE",
                "condition": "elderly",
                "description": f"Age-related caution: {medication} in patient aged {age}"
            })
        
        # Check kidney function
        kidney_function = patient_context.get("kidney_function", "normal")
        if kidney_function != "normal" and "kidney_impairment" in relative_contras:
            contraindications.append({
                "type": "organ_function",
                "severity": "MODERATE",
                "condition": "kidney_impairment",
                "description": f"Kidney function concern: {medication} with {kidney_function}"
            })
        
        return contraindications
