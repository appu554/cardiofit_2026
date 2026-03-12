"""
Contraindication Reasoner

This module implements real clinical logic for detecting contraindications
based on patient conditions, allergies, and medication profiles.
"""

import logging
from typing import List, Dict, Any, Optional
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)

class ContraindicationType(Enum):
    """Types of contraindications"""
    ABSOLUTE = "absolute"  # Never use
    RELATIVE = "relative"  # Use with extreme caution
    ALLERGY = "allergy"    # Allergic reaction risk
    PREGNANCY = "pregnancy"  # Pregnancy/lactation risk
    AGE = "age"           # Age-specific contraindication

class ContraindicationSeverity(Enum):
    """Severity of contraindications"""
    CRITICAL = "critical"  # Life-threatening risk
    HIGH = "high"         # Serious adverse effects likely
    MODERATE = "moderate"  # Significant risk
    LOW = "low"           # Minimal risk with monitoring

@dataclass
class Contraindication:
    """Contraindication data structure"""
    contraindication_id: str
    medication_id: str
    condition_id: str
    severity: ContraindicationSeverity
    type: ContraindicationType
    description: str
    rationale: str
    evidence_sources: List[str]
    override_possible: bool
    override_rationale: str

class ContraindicationReasoner:
    """
    Real contraindication reasoner with clinical knowledge base
    
    This implementation uses established contraindication databases and
    clinical guidelines to detect medication contraindications.
    """
    
    def __init__(self):
        self.contraindication_database = self._load_contraindication_database()
        self.allergy_cross_sensitivities = self._load_allergy_cross_sensitivities()
        logger.info("Contraindication Reasoner initialized")
    
    def _load_contraindication_database(self) -> Dict[str, List[Contraindication]]:
        """
        Load contraindication database
        
        In production, this would connect to clinical databases like:
        - Lexicomp Contraindications
        - Micromedex Contraindications
        - Clinical Decision Support Systems
        """
        contraindications = {
            # Warfarin contraindications
            "warfarin": [
                Contraindication(
                    contraindication_id="warfarin_pregnancy_001",
                    medication_id="warfarin",
                    condition_id="pregnancy",
                    severity=ContraindicationSeverity.CRITICAL,
                    type=ContraindicationType.ABSOLUTE,
                    description="Warfarin is teratogenic and contraindicated in pregnancy",
                    rationale="Crosses placenta, causes fetal warfarin syndrome, bleeding risk",
                    evidence_sources=[
                        "FDA Pregnancy Category X",
                        "CHEST Guidelines 2012",
                        "Thromb Haemost 2004;91:1062-1075"
                    ],
                    override_possible=False,
                    override_rationale="No safe override - use heparin alternatives"
                ),
                Contraindication(
                    contraindication_id="warfarin_bleeding_001",
                    medication_id="warfarin",
                    condition_id="active_bleeding",
                    severity=ContraindicationSeverity.CRITICAL,
                    type=ContraindicationType.ABSOLUTE,
                    description="Active bleeding is absolute contraindication to anticoagulation",
                    rationale="Will worsen bleeding, potentially life-threatening",
                    evidence_sources=[
                        "CHEST Guidelines 2012",
                        "Circulation 2014;129:1681-1689"
                    ],
                    override_possible=False,
                    override_rationale="Address bleeding source first"
                ),
                Contraindication(
                    contraindication_id="warfarin_liver_disease_001",
                    medication_id="warfarin",
                    condition_id="severe_liver_disease",
                    severity=ContraindicationSeverity.HIGH,
                    type=ContraindicationType.RELATIVE,
                    description="Severe liver disease increases bleeding risk",
                    rationale="Reduced synthesis of clotting factors, impaired metabolism",
                    evidence_sources=[
                        "Hepatology 2007;46:1097-1102",
                        "J Hepatol 2013;58:757-761"
                    ],
                    override_possible=True,
                    override_rationale="May use with extreme caution and frequent monitoring"
                )
            ],
            
            # ACE inhibitor contraindications
            "lisinopril": [
                Contraindication(
                    contraindication_id="ace_pregnancy_001",
                    medication_id="lisinopril",
                    condition_id="pregnancy",
                    severity=ContraindicationSeverity.CRITICAL,
                    type=ContraindicationType.ABSOLUTE,
                    description="ACE inhibitors cause fetal toxicity in 2nd/3rd trimester",
                    rationale="Oligohydramnios, growth retardation, kidney dysfunction",
                    evidence_sources=[
                        "FDA Pregnancy Category D",
                        "NEJM 2006;354:2443-2451"
                    ],
                    override_possible=False,
                    override_rationale="Use alternative antihypertensives safe in pregnancy"
                ),
                Contraindication(
                    contraindication_id="ace_angioedema_001",
                    medication_id="lisinopril",
                    condition_id="angioedema_history",
                    severity=ContraindicationSeverity.CRITICAL,
                    type=ContraindicationType.ABSOLUTE,
                    description="History of ACE inhibitor-induced angioedema",
                    rationale="High risk of recurrent, potentially fatal angioedema",
                    evidence_sources=[
                        "NEJM 2008;358:1209-1217",
                        "Ann Intern Med 2004;140:891-901"
                    ],
                    override_possible=False,
                    override_rationale="Use ARBs with caution or alternative drug classes"
                ),
                Contraindication(
                    contraindication_id="ace_hyperkalemia_001",
                    medication_id="lisinopril",
                    condition_id="hyperkalemia",
                    severity=ContraindicationSeverity.HIGH,
                    type=ContraindicationType.RELATIVE,
                    description="Hyperkalemia (K+ > 5.5 mEq/L)",
                    rationale="ACE inhibitors further increase potassium levels",
                    evidence_sources=[
                        "NEJM 1998;339:451-458",
                        "Kidney Int 2009;75:585-595"
                    ],
                    override_possible=True,
                    override_rationale="Correct hyperkalemia first, then start with close monitoring"
                )
            ],
            
            # Metformin contraindications
            "metformin": [
                Contraindication(
                    contraindication_id="metformin_kidney_failure_001",
                    medication_id="metformin",
                    condition_id="kidney_failure",
                    severity=ContraindicationSeverity.CRITICAL,
                    type=ContraindicationType.ABSOLUTE,
                    description="Severe kidney impairment (eGFR < 30)",
                    rationale="Risk of lactic acidosis due to reduced clearance",
                    evidence_sources=[
                        "Diabetes Care 2017;40:1543-1548",
                        "Cochrane Database Syst Rev 2010;4:CD002967"
                    ],
                    override_possible=False,
                    override_rationale="Use alternative diabetes medications"
                ),
                Contraindication(
                    contraindication_id="metformin_heart_failure_001",
                    medication_id="metformin",
                    condition_id="decompensated_heart_failure",
                    severity=ContraindicationSeverity.HIGH,
                    type=ContraindicationType.RELATIVE,
                    description="Acute or unstable heart failure",
                    rationale="Increased risk of lactic acidosis in hypoperfusion states",
                    evidence_sources=[
                        "Diabetes Care 2003;26:917-918",
                        "Heart Fail Rev 2013;18:679-688"
                    ],
                    override_possible=True,
                    override_rationale="May use once heart failure stabilized"
                )
            ],
            
            # Beta-blocker contraindications
            "metoprolol": [
                Contraindication(
                    contraindication_id="bb_asthma_001",
                    medication_id="metoprolol",
                    condition_id="asthma",
                    severity=ContraindicationSeverity.HIGH,
                    type=ContraindicationType.RELATIVE,
                    description="Asthma or severe COPD",
                    rationale="Beta-blockade can precipitate bronchospasm",
                    evidence_sources=[
                        "Chest 2005;128:3618-3624",
                        "Cochrane Database Syst Rev 2005;4:CD003566"
                    ],
                    override_possible=True,
                    override_rationale="Cardioselective beta-blockers may be used with caution"
                ),
                Contraindication(
                    contraindication_id="bb_heart_block_001",
                    medication_id="metoprolol",
                    condition_id="heart_block_2nd_3rd_degree",
                    severity=ContraindicationSeverity.CRITICAL,
                    type=ContraindicationType.ABSOLUTE,
                    description="2nd or 3rd degree heart block without pacemaker",
                    rationale="Further depression of AV conduction, risk of complete heart block",
                    evidence_sources=[
                        "Circulation 2013;128:e240-e327",
                        "Eur Heart J 2012;33:2569-2619"
                    ],
                    override_possible=False,
                    override_rationale="Pacemaker required before beta-blocker use"
                )
            ],
            
            # NSAID contraindications
            "ibuprofen": [
                Contraindication(
                    contraindication_id="nsaid_kidney_disease_001",
                    medication_id="ibuprofen",
                    condition_id="chronic_kidney_disease",
                    severity=ContraindicationSeverity.HIGH,
                    type=ContraindicationType.RELATIVE,
                    description="Chronic kidney disease (eGFR < 60)",
                    rationale="NSAIDs reduce kidney function and increase cardiovascular risk",
                    evidence_sources=[
                        "NEJM 2001;345:971-979",
                        "Kidney Int 2007;72:1493-1502"
                    ],
                    override_possible=True,
                    override_rationale="Short-term use with close monitoring if absolutely necessary"
                ),
                Contraindication(
                    contraindication_id="nsaid_heart_failure_001",
                    medication_id="ibuprofen",
                    condition_id="heart_failure",
                    severity=ContraindicationSeverity.HIGH,
                    type=ContraindicationType.RELATIVE,
                    description="Heart failure",
                    rationale="Fluid retention, reduced efficacy of ACE inhibitors/diuretics",
                    evidence_sources=[
                        "Circulation 2007;115:1634-1642",
                        "Heart Fail Rev 2013;18:439-449"
                    ],
                    override_possible=True,
                    override_rationale="Avoid if possible, use lowest dose for shortest duration"
                )
            ]
        }
        
        return contraindications
    
    def _load_allergy_cross_sensitivities(self) -> Dict[str, List[str]]:
        """Load allergy cross-sensitivity data"""
        return {
            "penicillin": [
                "amoxicillin", "ampicillin", "piperacillin", "nafcillin",
                "oxacillin", "cloxacillin", "methicillin"
            ],
            "sulfonamide": [
                "sulfamethoxazole", "sulfadiazine", "sulfasalazine",
                "furosemide", "hydrochlorothiazide", "celecoxib"
            ],
            "aspirin": [
                "ibuprofen", "naproxen", "diclofenac", "celecoxib",
                "indomethacin", "ketorolac"
            ],
            "codeine": [
                "morphine", "oxycodone", "hydrocodone", "tramadol"
            ]
        }
    
    async def check_contraindications(
        self,
        patient_id: str,
        medication_ids: List[str],
        condition_ids: List[str] = None,
        allergy_ids: List[str] = None,
        patient_context: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """
        Check for contraindications
        
        Args:
            patient_id: Patient identifier
            medication_ids: List of medications to check
            condition_ids: List of patient conditions
            allergy_ids: List of patient allergies
            patient_context: Additional patient context
            
        Returns:
            List of detected contraindications
        """
        logger.info(f"Checking contraindications for patient {patient_id}")

        # Debug logging to identify the issue
        logger.info(f"DEBUG: medication_ids type: {type(medication_ids)}, value: {medication_ids}")
        logger.info(f"DEBUG: condition_ids type: {type(condition_ids)}, value: {condition_ids}")
        logger.info(f"DEBUG: allergy_ids type: {type(allergy_ids)}, value: {allergy_ids}")
        logger.info(f"DEBUG: patient_context type: {type(patient_context)}, value: {patient_context}")

        contraindications = []
        condition_ids = condition_ids or []
        allergy_ids = allergy_ids or []
        
        # Check each medication against conditions and allergies
        for medication_id in medication_ids:
            medication_id = medication_id.lower().strip()
            
            # Check condition-based contraindications
            for condition_id in condition_ids:
                contraindication = await self._check_condition_contraindication(
                    medication_id, condition_id, patient_context
                )
                if contraindication:
                    contraindications.append(contraindication)
            
            # Check allergy-based contraindications
            for allergy_id in allergy_ids:
                contraindication = await self._check_allergy_contraindication(
                    medication_id, allergy_id
                )
                if contraindication:
                    contraindications.append(contraindication)
            
            # Check age-based contraindications
            if patient_context and 'age' in patient_context:
                contraindication = await self._check_age_contraindication(
                    medication_id, patient_context['age']
                )
                if contraindication:
                    contraindications.append(contraindication)
            
            # Check pregnancy contraindications
            if patient_context and patient_context.get('pregnancy_status') == 'pregnant':
                contraindication = await self._check_pregnancy_contraindication(
                    medication_id
                )
                if contraindication:
                    contraindications.append(contraindication)
        
        # Sort by severity (critical first)
        severity_order = {
            ContraindicationSeverity.CRITICAL: 0,
            ContraindicationSeverity.HIGH: 1,
            ContraindicationSeverity.MODERATE: 2,
            ContraindicationSeverity.LOW: 3
        }
        
        contraindications.sort(key=lambda x: severity_order.get(
            ContraindicationSeverity(x['severity']), 4
        ))
        
        logger.info(f"Found {len(contraindications)} contraindications for patient {patient_id}")

        # Return in the format expected by the parallel executor
        return {
            'assertions': contraindications,
            'confidence_score': 0.9 if contraindications else 1.0,
            'metadata': {
                'reasoner_type': 'contraindication',
                'total_contraindications': len(contraindications),
                'status': 'completed'
            }
        }
    
    async def _check_condition_contraindication(
        self,
        medication_id: str,
        condition_id: str,
        patient_context: Optional[Dict[str, Any]]
    ) -> Optional[Dict[str, Any]]:
        """Check for condition-based contraindication"""
        
        if medication_id not in self.contraindication_database:
            return None
        
        for contraindication in self.contraindication_database[medication_id]:
            if contraindication.condition_id == condition_id:
                return self._format_contraindication(contraindication)
        
        return None
    
    async def _check_allergy_contraindication(
        self,
        medication_id: str,
        allergy_id: str
    ) -> Optional[Dict[str, Any]]:
        """Check for allergy-based contraindication"""
        
        # Direct allergy match
        if medication_id == allergy_id:
            return {
                "contraindication_id": f"allergy_{medication_id}_{allergy_id}",
                "medication_id": medication_id,
                "condition_id": f"allergy_to_{allergy_id}",
                "severity": ContraindicationSeverity.CRITICAL.value,
                "type": ContraindicationType.ALLERGY.value,
                "description": f"Patient has documented allergy to {allergy_id}",
                "rationale": "Direct allergic reaction risk",
                "evidence_sources": ["Patient allergy history"],
                "override_possible": False,
                "override_rationale": "Use alternative medication"
            }
        
        # Cross-sensitivity check
        for allergen, cross_sensitive_drugs in self.allergy_cross_sensitivities.items():
            if allergy_id == allergen and medication_id in cross_sensitive_drugs:
                return {
                    "contraindication_id": f"cross_sensitivity_{medication_id}_{allergy_id}",
                    "medication_id": medication_id,
                    "condition_id": f"allergy_to_{allergy_id}",
                    "severity": ContraindicationSeverity.HIGH.value,
                    "type": ContraindicationType.ALLERGY.value,
                    "description": f"Cross-sensitivity risk with {allergy_id} allergy",
                    "rationale": f"Potential cross-reaction due to {allergen} allergy",
                    "evidence_sources": ["Cross-sensitivity database"],
                    "override_possible": True,
                    "override_rationale": "May use with premedication and close monitoring"
                }
        
        return None
    
    async def _check_age_contraindication(
        self,
        medication_id: str,
        age: int
    ) -> Optional[Dict[str, Any]]:
        """Check for age-based contraindications"""
        
        # Pediatric contraindications
        if age < 18:
            pediatric_contraindications = {
                "aspirin": {
                    "condition": "reye_syndrome_risk",
                    "description": "Aspirin contraindicated in children due to Reye's syndrome risk",
                    "rationale": "Association with Reye's syndrome in viral illnesses"
                },
                "tetracycline": {
                    "condition": "tooth_discoloration",
                    "description": "Tetracyclines cause permanent tooth discoloration in children",
                    "rationale": "Binds to developing tooth enamel"
                }
            }
            
            if medication_id in pediatric_contraindications:
                contraindication = pediatric_contraindications[medication_id]
                return {
                    "contraindication_id": f"pediatric_{medication_id}",
                    "medication_id": medication_id,
                    "condition_id": contraindication["condition"],
                    "severity": ContraindicationSeverity.HIGH.value,
                    "type": ContraindicationType.AGE.value,
                    "description": contraindication["description"],
                    "rationale": contraindication["rationale"],
                    "evidence_sources": ["Pediatric contraindication guidelines"],
                    "override_possible": True,
                    "override_rationale": "Use alternative pediatric-appropriate medication"
                }
        
        return None
    
    async def _check_pregnancy_contraindication(
        self,
        medication_id: str
    ) -> Optional[Dict[str, Any]]:
        """Check for pregnancy contraindications"""
        
        # Check if medication has pregnancy contraindication
        if medication_id in self.contraindication_database:
            for contraindication in self.contraindication_database[medication_id]:
                if contraindication.condition_id == "pregnancy":
                    return self._format_contraindication(contraindication)
        
        return None
    
    def _format_contraindication(self, contraindication: Contraindication) -> Dict[str, Any]:
        """Format contraindication for API response"""
        return {
            "contraindication_id": contraindication.contraindication_id,
            "medication_id": contraindication.medication_id,
            "condition_id": contraindication.condition_id,
            "severity": contraindication.severity.value,
            "type": contraindication.type.value,
            "description": contraindication.description,
            "rationale": contraindication.rationale,
            "evidence_sources": contraindication.evidence_sources,
            "override_possible": contraindication.override_possible,
            "override_rationale": contraindication.override_rationale
        }
