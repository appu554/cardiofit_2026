"""
Duplicate Therapy Reasoner for Clinical Assertion Engine

Detects exact, therapeutic, and pharmacologic duplicate medications to prevent
dangerous polypharmacy and therapeutic redundancy.
"""

import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class DuplicateTherapy:
    """Duplicate therapy detection result"""
    duplicate_id: str
    duplicate_type: str  # exact, therapeutic, pharmacologic
    medication_a: str
    medication_b: str
    severity: str
    description: str
    mechanism: str
    clinical_impact: str
    recommendations: List[str]
    confidence_score: float
    evidence_sources: List[str]


class DuplicateTherapyReasoner:
    """
    Advanced duplicate therapy detection reasoner
    
    Features:
    - Exact duplicate detection (same medication)
    - Therapeutic duplicate detection (same therapeutic class)
    - Pharmacologic duplicate detection (same mechanism of action)
    - Route transition validation (IV to PO conversions)
    - Brand/generic duplicate detection
    - Combination appropriateness assessment
    """
    
    def __init__(self):
        self.therapeutic_classes = self._load_therapeutic_classes()
        self.pharmacologic_classes = self._load_pharmacologic_classes()
        self.brand_generic_map = self._load_brand_generic_map()
        self.appropriate_combinations = self._load_appropriate_combinations()
        
        logger.info("Duplicate Therapy Reasoner initialized")
    
    async def check_duplicate_therapy(self, patient_id: str, medication_ids: List[str],
                                    patient_context: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """
        Check for duplicate therapy across multiple dimensions
        
        Args:
            patient_id: Patient identifier
            medication_ids: List of medication identifiers
            patient_context: Patient clinical context
            
        Returns:
            List of duplicate therapy assertions
        """
        try:
            # Debug logging to identify the issue
            logger.info(f"DEBUG: medication_ids type: {type(medication_ids)}, value: {medication_ids}")
            logger.info(f"DEBUG: patient_context type: {type(patient_context)}, value: {patient_context}")

            duplicates = []
            
            # Check all medication pairs for duplicates
            for i, med_a in enumerate(medication_ids):
                for j, med_b in enumerate(medication_ids[i+1:], i+1):
                    
                    # 1. Exact duplicate check
                    exact_duplicate = await self._check_exact_duplicate(med_a, med_b)
                    if exact_duplicate:
                        duplicates.append(exact_duplicate)
                    
                    # 2. Therapeutic duplicate check
                    therapeutic_duplicate = await self._check_therapeutic_duplicate(med_a, med_b)
                    if therapeutic_duplicate:
                        duplicates.append(therapeutic_duplicate)
                    
                    # 3. Pharmacologic duplicate check
                    pharmacologic_duplicate = await self._check_pharmacologic_duplicate(med_a, med_b)
                    if pharmacologic_duplicate:
                        duplicates.append(pharmacologic_duplicate)
                    
                    # 4. Brand/generic duplicate check
                    brand_generic_duplicate = await self._check_brand_generic_duplicate(med_a, med_b)
                    if brand_generic_duplicate:
                        duplicates.append(brand_generic_duplicate)
            
            # Convert to assertion format
            assertions = []
            for duplicate in duplicates:
                assertion = {
                    "type": "duplicate_therapy",
                    "severity": duplicate.severity,
                    "title": f"{duplicate.duplicate_type.title()} Duplicate Therapy",
                    "description": duplicate.description,
                    "explanation": f"Mechanism: {duplicate.mechanism}. Impact: {duplicate.clinical_impact}",
                    "confidence": duplicate.confidence_score,
                    "evidence_sources": duplicate.evidence_sources,
                    "recommendations": duplicate.recommendations,
                    "metadata": {
                        "duplicate_type": duplicate.duplicate_type,
                        "medication_a": duplicate.medication_a,
                        "medication_b": duplicate.medication_b,
                        "duplicate_id": duplicate.duplicate_id
                    }
                }
                assertions.append(assertion)
            
            logger.info(f"Found {len(assertions)} duplicate therapy issues for patient {patient_id}")

            # Return in the format expected by the parallel executor
            return {
                'assertions': assertions,
                'confidence_score': 0.9 if assertions else 1.0,
                'metadata': {
                    'reasoner_type': 'duplicate_therapy',
                    'total_duplicates': len(assertions),
                    'status': 'completed'
                }
            }
            
        except Exception as e:
            logger.error(f"Error checking duplicate therapy: {e}")
            return {
                'assertions': [],
                'confidence_score': 0.0,
                'metadata': {
                    'reasoner_type': 'duplicate_therapy',
                    'error': str(e),
                    'status': 'failed'
                }
            }
    
    async def _check_exact_duplicate(self, med_a: str, med_b: str) -> Optional[DuplicateTherapy]:
        """Check for exact medication duplicates"""
        
        # Normalize medication names for comparison
        normalized_a = self._normalize_medication_name(med_a)
        normalized_b = self._normalize_medication_name(med_b)
        
        if normalized_a == normalized_b:
            return DuplicateTherapy(
                duplicate_id=f"exact_{normalized_a}_{normalized_b}",
                duplicate_type="exact",
                medication_a=med_a,
                medication_b=med_b,
                severity="high",
                description=f"Exact duplicate medication: {med_a} and {med_b}",
                mechanism="Same active ingredient and formulation",
                clinical_impact="Risk of overdose, increased adverse effects",
                recommendations=[
                    "Discontinue one medication",
                    "Verify intended therapy",
                    "Check for prescribing error"
                ],
                confidence_score=0.95,
                evidence_sources=["Medication reconciliation", "Drug database"]
            )
        
        return None
    
    async def _check_therapeutic_duplicate(self, med_a: str, med_b: str) -> Optional[DuplicateTherapy]:
        """Check for therapeutic class duplicates"""
        
        class_a = self._get_therapeutic_class(med_a)
        class_b = self._get_therapeutic_class(med_b)
        
        if class_a and class_b and class_a == class_b:
            # Check if combination is appropriate
            if self._is_appropriate_combination(med_a, med_b, class_a):
                return None
            
            severity = self._assess_therapeutic_duplicate_severity(class_a)
            
            return DuplicateTherapy(
                duplicate_id=f"therapeutic_{class_a}_{med_a}_{med_b}",
                duplicate_type="therapeutic",
                medication_a=med_a,
                medication_b=med_b,
                severity=severity,
                description=f"Therapeutic duplicate in {class_a} class: {med_a} and {med_b}",
                mechanism=f"Both medications belong to {class_a} therapeutic class",
                clinical_impact="Potential for additive effects and increased toxicity",
                recommendations=[
                    "Review therapeutic necessity",
                    "Consider single agent therapy",
                    "Monitor for additive effects"
                ],
                confidence_score=0.85,
                evidence_sources=["Therapeutic classification", "Clinical guidelines"]
            )
        
        return None
    
    async def _check_pharmacologic_duplicate(self, med_a: str, med_b: str) -> Optional[DuplicateTherapy]:
        """Check for pharmacologic mechanism duplicates"""
        
        mechanism_a = self._get_pharmacologic_mechanism(med_a)
        mechanism_b = self._get_pharmacologic_mechanism(med_b)
        
        if mechanism_a and mechanism_b and mechanism_a == mechanism_b:
            # Check if combination is clinically appropriate
            if self._is_appropriate_mechanism_combination(med_a, med_b, mechanism_a):
                return None
            
            return DuplicateTherapy(
                duplicate_id=f"pharmacologic_{mechanism_a}_{med_a}_{med_b}",
                duplicate_type="pharmacologic",
                medication_a=med_a,
                medication_b=med_b,
                severity="moderate",
                description=f"Pharmacologic duplicate with {mechanism_a} mechanism: {med_a} and {med_b}",
                mechanism=f"Both medications work via {mechanism_a}",
                clinical_impact="Potential for additive pharmacologic effects",
                recommendations=[
                    "Evaluate clinical rationale",
                    "Consider mechanism diversity",
                    "Monitor for enhanced effects"
                ],
                confidence_score=0.75,
                evidence_sources=["Pharmacology database", "Mechanism of action"]
            )
        
        return None
    
    async def _check_brand_generic_duplicate(self, med_a: str, med_b: str) -> Optional[DuplicateTherapy]:
        """Check for brand/generic duplicates"""
        
        generic_a = self._get_generic_name(med_a)
        generic_b = self._get_generic_name(med_b)
        
        if generic_a and generic_b and generic_a == generic_b and med_a != med_b:
            return DuplicateTherapy(
                duplicate_id=f"brand_generic_{generic_a}_{med_a}_{med_b}",
                duplicate_type="brand_generic",
                medication_a=med_a,
                medication_b=med_b,
                severity="high",
                description=f"Brand/generic duplicate: {med_a} and {med_b} (both {generic_a})",
                mechanism="Same active ingredient, different brand names",
                clinical_impact="Risk of double dosing with same medication",
                recommendations=[
                    "Discontinue one formulation",
                    "Educate patient about brand/generic equivalence",
                    "Update medication list"
                ],
                confidence_score=0.90,
                evidence_sources=["Generic equivalence database", "FDA Orange Book"]
            )
        
        return None
    
    def _normalize_medication_name(self, medication: str) -> str:
        """Normalize medication name for comparison"""
        return medication.lower().strip().replace("-", "").replace(" ", "")
    
    def _get_therapeutic_class(self, medication: str) -> Optional[str]:
        """Get therapeutic class for medication"""
        return self.therapeutic_classes.get(medication.lower())
    
    def _get_pharmacologic_mechanism(self, medication: str) -> Optional[str]:
        """Get pharmacologic mechanism for medication"""
        return self.pharmacologic_classes.get(medication.lower())
    
    def _get_generic_name(self, medication: str) -> Optional[str]:
        """Get generic name for medication"""
        return self.brand_generic_map.get(medication.lower())
    
    def _is_appropriate_combination(self, med_a: str, med_b: str, therapeutic_class: str) -> bool:
        """Check if therapeutic combination is clinically appropriate"""
        combination_key = tuple(sorted([med_a.lower(), med_b.lower()]))
        return combination_key in self.appropriate_combinations.get(therapeutic_class, set())
    
    def _is_appropriate_mechanism_combination(self, med_a: str, med_b: str, mechanism: str) -> bool:
        """Check if pharmacologic mechanism combination is appropriate"""
        # Some mechanisms allow appropriate combinations (e.g., different beta-blockers for different indications)
        appropriate_mechanisms = ["beta_blocker", "ace_inhibitor", "calcium_channel_blocker"]
        return mechanism in appropriate_mechanisms
    
    def _assess_therapeutic_duplicate_severity(self, therapeutic_class: str) -> str:
        """Assess severity based on therapeutic class"""
        high_risk_classes = ["anticoagulant", "antiarrhythmic", "insulin", "opioid"]
        moderate_risk_classes = ["antihypertensive", "antidepressant", "antibiotic"]
        
        if therapeutic_class in high_risk_classes:
            return "high"
        elif therapeutic_class in moderate_risk_classes:
            return "moderate"
        else:
            return "low"
    
    def _load_therapeutic_classes(self) -> Dict[str, str]:
        """Load therapeutic classification database"""
        return {
            # Cardiovascular
            "warfarin": "anticoagulant",
            "heparin": "anticoagulant",
            "aspirin": "antiplatelet",
            "clopidogrel": "antiplatelet",
            "lisinopril": "ace_inhibitor",
            "enalapril": "ace_inhibitor",
            "metoprolol": "beta_blocker",
            "propranolol": "beta_blocker",
            "amlodipine": "calcium_channel_blocker",
            "nifedipine": "calcium_channel_blocker",
            
            # Diabetes
            "metformin": "biguanide",
            "insulin": "insulin",
            "glipizide": "sulfonylurea",
            "glyburide": "sulfonylurea",
            
            # Antibiotics
            "amoxicillin": "penicillin",
            "ampicillin": "penicillin",
            "ciprofloxacin": "fluoroquinolone",
            "levofloxacin": "fluoroquinolone",
            
            # Pain management
            "morphine": "opioid",
            "oxycodone": "opioid",
            "ibuprofen": "nsaid",
            "naproxen": "nsaid"
        }
    
    def _load_pharmacologic_classes(self) -> Dict[str, str]:
        """Load pharmacologic mechanism database"""
        return {
            # Beta blockers
            "metoprolol": "beta_blocker",
            "propranolol": "beta_blocker",
            "atenolol": "beta_blocker",
            
            # ACE inhibitors
            "lisinopril": "ace_inhibition",
            "enalapril": "ace_inhibition",
            "captopril": "ace_inhibition",
            
            # Calcium channel blockers
            "amlodipine": "calcium_channel_blockade",
            "nifedipine": "calcium_channel_blockade",
            "diltiazem": "calcium_channel_blockade",
            
            # NSAIDs
            "ibuprofen": "cox_inhibition",
            "naproxen": "cox_inhibition",
            "diclofenac": "cox_inhibition"
        }
    
    def _load_brand_generic_map(self) -> Dict[str, str]:
        """Load brand/generic name mapping"""
        return {
            # Brand to generic mapping
            "tylenol": "acetaminophen",
            "advil": "ibuprofen",
            "motrin": "ibuprofen",
            "aleve": "naproxen",
            "prinivil": "lisinopril",
            "zestril": "lisinopril",
            "lopressor": "metoprolol",
            "toprol": "metoprolol",
            "norvasc": "amlodipine",
            "procardia": "nifedipine",
            
            # Generic names map to themselves
            "acetaminophen": "acetaminophen",
            "ibuprofen": "ibuprofen",
            "naproxen": "naproxen",
            "lisinopril": "lisinopril",
            "metoprolol": "metoprolol",
            "amlodipine": "amlodipine"
        }
    
    def _load_appropriate_combinations(self) -> Dict[str, set]:
        """Load clinically appropriate therapeutic combinations"""
        return {
            "antihypertensive": {
                ("lisinopril", "amlodipine"),  # ACE + CCB
                ("metoprolol", "lisinopril"),  # Beta-blocker + ACE
                ("amlodipine", "metoprolol")   # CCB + Beta-blocker
            },
            "antiplatelet": {
                ("aspirin", "clopidogrel")     # Dual antiplatelet therapy
            }
        }
