"""
Allergy Checker for CAE

Checks for potential allergic reactions and cross-sensitivities
"""

import logging
from typing import Dict, Any, List, Optional

logger = logging.getLogger(__name__)

class AllergyChecker:
    """Check for allergy risks and cross-sensitivities"""
    
    def __init__(self):
        self.allergy_database = self._initialize_allergy_database()
        logger.info("Allergy Checker initialized")
    
    def _initialize_allergy_database(self) -> Dict[str, Dict]:
        """Initialize allergy and cross-sensitivity database"""
        return {
            "penicillin": {
                "cross_sensitivities": ["amoxicillin", "ampicillin", "cephalexin"],
                "risk_level": "high",
                "reaction_types": ["rash", "anaphylaxis", "breathing_difficulty"]
            },
            "sulfa_drugs": {
                "cross_sensitivities": ["sulfamethoxazole", "trimethoprim_sulfamethoxazole", "furosemide"],
                "risk_level": "moderate",
                "reaction_types": ["rash", "stevens_johnson_syndrome"]
            },
            "aspirin": {
                "cross_sensitivities": ["ibuprofen", "naproxen", "diclofenac"],
                "risk_level": "moderate",
                "reaction_types": ["bronchospasm", "urticaria", "angioedema"]
            }
        }
    
    async def check_allergies(self, patient_id: str, medication: str, 
                            known_allergies: List[str]) -> Dict[str, Any]:
        """Check for allergy risks with new medication"""
        
        logger.info(f"Checking allergy risks for {medication}")
        
        # Check direct allergy match
        if medication.lower() in [allergy.lower() for allergy in known_allergies]:
            return {
                "risk_detected": True,
                "risk_level": "CRITICAL",
                "allergy_type": "direct_allergy",
                "details": f"Patient has known allergy to {medication}"
            }
        
        # Check cross-sensitivities
        for allergy in known_allergies:
            allergy_data = self.allergy_database.get(allergy.lower())
            if allergy_data:
                cross_sensitivities = allergy_data.get("cross_sensitivities", [])
                if medication.lower() in [cs.lower() for cs in cross_sensitivities]:
                    return {
                        "risk_detected": True,
                        "risk_level": allergy_data.get("risk_level", "moderate").upper(),
                        "allergy_type": "cross_sensitivity",
                        "details": f"Cross-sensitivity risk: {medication} with known {allergy} allergy"
                    }
        
        # No allergy risk detected
        return {
            "risk_detected": False,
            "risk_level": "NONE",
            "allergy_type": "none",
            "details": "No allergy risks detected"
        }
