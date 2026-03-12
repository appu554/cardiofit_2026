"""
Dosing Calculator Reasoner

This module implements real clinical dosing calculations based on
established pharmacokinetic principles and clinical guidelines.
"""

import logging
import math
from typing import Dict, Any, Optional, List, Tuple
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)

class DosingUnit(Enum):
    """Dosing units"""
    MG = "mg"
    MCG = "mcg"
    UNITS = "units"
    ML = "ml"
    MG_KG = "mg/kg"
    MCG_KG = "mcg/kg"
    UNITS_KG = "units/kg"

class RenalFunction(Enum):
    """Renal function categories"""
    NORMAL = "normal"  # CrCl > 90
    MILD_IMPAIRMENT = "mild_impairment"  # CrCl 60-89
    MODERATE_IMPAIRMENT = "moderate_impairment"  # CrCl 30-59
    SEVERE_IMPAIRMENT = "severe_impairment"  # CrCl 15-29
    KIDNEY_FAILURE = "kidney_failure"  # CrCl < 15

class HepaticFunction(Enum):
    """Hepatic function categories"""
    NORMAL = "normal"
    MILD_IMPAIRMENT = "mild_impairment"  # Child-Pugh A
    MODERATE_IMPAIRMENT = "moderate_impairment"  # Child-Pugh B
    SEVERE_IMPAIRMENT = "severe_impairment"  # Child-Pugh C

@dataclass
class DosingRecommendation:
    """Dosing recommendation structure"""
    medication_id: str
    dose: str
    frequency: str
    route: str
    duration: str
    rationale: str
    warnings: List[str]
    adjustments: List['DosingAdjustment']

@dataclass
class DosingAdjustment:
    """Dosing adjustment structure"""
    type: str  # "renal", "hepatic", "age", "weight"
    adjustment: str
    rationale: str
    required: bool

class DosingCalculator:
    """
    Real dosing calculator with clinical pharmacokinetic principles
    
    This implementation uses established dosing guidelines and
    pharmacokinetic calculations for common medications.
    """
    
    def __init__(self):
        self.dosing_database = self._load_dosing_database()
        logger.info("Dosing Calculator initialized")
    
    def _load_dosing_database(self) -> Dict[str, Dict[str, Any]]:
        """
        Load medication dosing database
        
        In production, this would connect to clinical dosing databases like:
        - Lexicomp Dosing Guidelines
        - Micromedex Dosing
        - Clinical Pharmacology Database
        """
        return {
            "warfarin": {
                "standard_dose": {"value": 5, "unit": DosingUnit.MG},
                "frequency": "once daily",
                "route": "oral",
                "indication_specific": {
                    "atrial_fibrillation": {"initial_dose": 5, "target_inr": "2.0-3.0"},
                    "venous_thromboembolism": {"initial_dose": 5, "target_inr": "2.0-3.0"},
                    "mechanical_valve": {"initial_dose": 5, "target_inr": "2.5-3.5"}
                },
                "adjustments": {
                    "age": {
                        ">75": {"factor": 0.8, "rationale": "Increased sensitivity in elderly"},
                        "65-75": {"factor": 0.9, "rationale": "Moderate dose reduction for age"}
                    },
                    "weight": {
                        "<50": {"factor": 0.8, "rationale": "Lower body weight"},
                        ">100": {"factor": 1.1, "rationale": "Higher body weight"}
                    },
                    "hepatic": {
                        "mild_impairment": {"factor": 0.8, "rationale": "Reduced metabolism"},
                        "moderate_impairment": {"factor": 0.6, "rationale": "Significantly reduced metabolism"},
                        "severe_impairment": {"factor": 0.4, "rationale": "Severely reduced metabolism"}
                    }
                },
                "monitoring": "INR every 2-3 days initially, then weekly until stable",
                "warnings": [
                    "Monitor for bleeding signs",
                    "Avoid vitamin K rich foods in large quantities",
                    "Many drug interactions - check before adding new medications"
                ]
            },
            
            "lisinopril": {
                "standard_dose": {"value": 10, "unit": DosingUnit.MG},
                "frequency": "once daily",
                "route": "oral",
                "indication_specific": {
                    "hypertension": {"initial_dose": 10, "max_dose": 40, "titration": "weekly"},
                    "heart_failure": {"initial_dose": 5, "max_dose": 40, "titration": "bi-weekly"},
                    "post_mi": {"initial_dose": 5, "max_dose": 10, "titration": "careful"}
                },
                "adjustments": {
                    "renal": {
                        "mild_impairment": {"factor": 1.0, "rationale": "No adjustment needed"},
                        "moderate_impairment": {"factor": 0.75, "rationale": "Reduced clearance"},
                        "severe_impairment": {"factor": 0.5, "rationale": "Significantly reduced clearance"},
                        "kidney_failure": {"factor": 0.25, "rationale": "Minimal clearance"}
                    },
                    "age": {
                        ">75": {"factor": 0.8, "rationale": "Start low in elderly"}
                    }
                },
                "monitoring": "Blood pressure, serum creatinine, potassium",
                "warnings": [
                    "Monitor for hyperkalemia",
                    "Check kidney function before starting",
                    "May cause dry cough in 10-15% of patients"
                ]
            },
            
            "metformin": {
                "standard_dose": {"value": 500, "unit": DosingUnit.MG},
                "frequency": "twice daily",
                "route": "oral",
                "indication_specific": {
                    "type2_diabetes": {"initial_dose": 500, "max_dose": 2000, "titration": "weekly"}
                },
                "adjustments": {
                    "renal": {
                        "mild_impairment": {"factor": 1.0, "rationale": "No adjustment if eGFR >45"},
                        "moderate_impairment": {"factor": 0.5, "rationale": "Reduce dose if eGFR 30-45"},
                        "severe_impairment": {"contraindicated": True, "rationale": "Risk of lactic acidosis"},
                        "kidney_failure": {"contraindicated": True, "rationale": "Risk of lactic acidosis"}
                    },
                    "age": {
                        ">80": {"factor": 0.75, "rationale": "Increased risk of lactic acidosis"}
                    }
                },
                "monitoring": "HbA1c, kidney function, vitamin B12 levels",
                "warnings": [
                    "Risk of lactic acidosis with kidney impairment",
                    "Discontinue before contrast procedures",
                    "May cause vitamin B12 deficiency with long-term use"
                ]
            },
            
            "digoxin": {
                "standard_dose": {"value": 0.25, "unit": DosingUnit.MG},
                "frequency": "once daily",
                "route": "oral",
                "indication_specific": {
                    "heart_failure": {"initial_dose": 0.125, "max_dose": 0.25},
                    "atrial_fibrillation": {"initial_dose": 0.25, "max_dose": 0.5}
                },
                "adjustments": {
                    "renal": {
                        "mild_impairment": {"factor": 0.8, "rationale": "Reduced clearance"},
                        "moderate_impairment": {"factor": 0.6, "rationale": "Significantly reduced clearance"},
                        "severe_impairment": {"factor": 0.4, "rationale": "Severely reduced clearance"},
                        "kidney_failure": {"factor": 0.25, "rationale": "Minimal clearance"}
                    },
                    "age": {
                        ">70": {"factor": 0.8, "rationale": "Reduced clearance with age"},
                        ">80": {"factor": 0.6, "rationale": "Significantly reduced clearance"}
                    },
                    "weight": {
                        "<60": {"factor": 0.8, "rationale": "Lower volume of distribution"}
                    }
                },
                "monitoring": "Digoxin level, kidney function, electrolytes",
                "warnings": [
                    "Narrow therapeutic window",
                    "Monitor for toxicity signs (nausea, visual changes, arrhythmias)",
                    "Many drug interactions"
                ]
            }
        }
    
    async def calculate_dosing(
        self,
        patient_id: str,
        medication_id: str,
        patient_parameters: Dict[str, Any],
        indication: Optional[str] = None
    ) -> DosingRecommendation:
        """
        Calculate medication dosing based on patient parameters
        
        Args:
            patient_id: Patient identifier
            medication_id: Medication identifier
            patient_parameters: Patient parameters (weight, age, kidney function, etc.)
            indication: Clinical indication for the medication
            
        Returns:
            DosingRecommendation with calculated dose and adjustments
        """
        logger.info(f"Calculating dosing for {medication_id} for patient {patient_id}")
        
        # Normalize medication name
        medication_id = medication_id.lower().strip()
        
        if medication_id not in self.dosing_database:
            return self._create_generic_recommendation(medication_id, "Unknown medication - consult clinical pharmacist")
        
        drug_data = self.dosing_database[medication_id]
        
        # Get base dose
        base_dose = drug_data["standard_dose"]["value"]
        dose_unit = drug_data["standard_dose"]["unit"]
        
        # Apply indication-specific dosing
        if indication and indication in drug_data.get("indication_specific", {}):
            indication_data = drug_data["indication_specific"][indication]
            base_dose = indication_data.get("initial_dose", base_dose)
        
        # Calculate adjustments
        adjustments = []
        final_dose = base_dose
        adjustment_factor = 1.0
        
        # Age adjustments
        age = patient_parameters.get("age", 0)
        if "age" in drug_data.get("adjustments", {}):
            age_adjustments = drug_data["adjustments"]["age"]
            for age_range, adjustment in age_adjustments.items():
                if self._age_in_range(age, age_range):
                    adjustment_factor *= adjustment["factor"]
                    adjustments.append(DosingAdjustment(
                        type="age",
                        adjustment=f"Dose reduced by {int((1-adjustment['factor'])*100)}%",
                        rationale=adjustment["rationale"],
                        required=True
                    ))
        
        # Weight adjustments
        weight = patient_parameters.get("weight", 70)
        if "weight" in drug_data.get("adjustments", {}):
            weight_adjustments = drug_data["adjustments"]["weight"]
            for weight_range, adjustment in weight_adjustments.items():
                if self._weight_in_range(weight, weight_range):
                    adjustment_factor *= adjustment["factor"]
                    adjustments.append(DosingAdjustment(
                        type="weight",
                        adjustment=f"Dose adjusted by {int((adjustment['factor']-1)*100):+}%",
                        rationale=adjustment["rationale"],
                        required=True
                    ))
        
        # Renal function adjustments
        kidney_function = patient_parameters.get("kidney_function", "normal")
        creatinine_clearance = patient_parameters.get("creatinine_clearance")
        
        if "renal" in drug_data.get("adjustments", {}):
            renal_adjustments = drug_data["adjustments"]["renal"]
            renal_category = self._categorize_renal_function(kidney_function, creatinine_clearance)
            
            if renal_category in renal_adjustments:
                renal_adj = renal_adjustments[renal_category]
                if renal_adj.get("contraindicated"):
                    return self._create_contraindicated_recommendation(
                        medication_id, renal_adj["rationale"]
                    )
                else:
                    adjustment_factor *= renal_adj["factor"]
                    adjustments.append(DosingAdjustment(
                        type="renal",
                        adjustment=f"Dose reduced by {int((1-renal_adj['factor'])*100)}%",
                        rationale=renal_adj["rationale"],
                        required=True
                    ))
        
        # Hepatic function adjustments
        liver_function = patient_parameters.get("liver_function", "normal")
        if "hepatic" in drug_data.get("adjustments", {}):
            hepatic_adjustments = drug_data["adjustments"]["hepatic"]
            if liver_function in hepatic_adjustments:
                hepatic_adj = hepatic_adjustments[liver_function]
                adjustment_factor *= hepatic_adj["factor"]
                adjustments.append(DosingAdjustment(
                    type="hepatic",
                    adjustment=f"Dose reduced by {int((1-hepatic_adj['factor'])*100)}%",
                    rationale=hepatic_adj["rationale"],
                    required=True
                ))
        
        # Calculate final dose
        final_dose = base_dose * adjustment_factor
        final_dose = self._round_dose(final_dose, dose_unit)
        
        # Create recommendation
        return DosingRecommendation(
            medication_id=medication_id,
            dose=f"{final_dose} {dose_unit.value}",
            frequency=drug_data["frequency"],
            route=drug_data["route"],
            duration="As directed by physician",
            rationale=self._create_dosing_rationale(base_dose, final_dose, adjustments),
            warnings=drug_data.get("warnings", []),
            adjustments=adjustments
        )
    
    def _age_in_range(self, age: int, age_range: str) -> bool:
        """Check if age falls in specified range"""
        if age_range.startswith(">"):
            return age > int(age_range[1:])
        elif "-" in age_range:
            min_age, max_age = map(int, age_range.split("-"))
            return min_age <= age <= max_age
        return False
    
    def _weight_in_range(self, weight: float, weight_range: str) -> bool:
        """Check if weight falls in specified range"""
        if weight_range.startswith("<"):
            return weight < float(weight_range[1:])
        elif weight_range.startswith(">"):
            return weight > float(weight_range[1:])
        return False
    
    def _categorize_renal_function(self, kidney_function: str, creatinine_clearance: Optional[float]) -> str:
        """Categorize renal function"""
        if creatinine_clearance:
            if creatinine_clearance >= 90:
                return "normal"
            elif creatinine_clearance >= 60:
                return "mild_impairment"
            elif creatinine_clearance >= 30:
                return "moderate_impairment"
            elif creatinine_clearance >= 15:
                return "severe_impairment"
            else:
                return "kidney_failure"
        
        return kidney_function
    
    def _round_dose(self, dose: float, unit: DosingUnit) -> float:
        """Round dose to appropriate precision"""
        if unit in [DosingUnit.MCG, DosingUnit.MCG_KG]:
            return round(dose, 1)
        elif unit in [DosingUnit.MG, DosingUnit.MG_KG]:
            return round(dose, 2)
        else:
            return round(dose, 1)
    
    def _create_dosing_rationale(self, base_dose: float, final_dose: float, adjustments: List[DosingAdjustment]) -> str:
        """Create rationale for dosing recommendation"""
        rationale = f"Starting dose: {base_dose} mg"
        
        if adjustments:
            rationale += ". Adjustments applied: "
            adj_reasons = [f"{adj.type} ({adj.rationale})" for adj in adjustments]
            rationale += "; ".join(adj_reasons)
        
        if abs(final_dose - base_dose) > 0.01:
            rationale += f". Final dose: {final_dose} mg"
        
        return rationale
    
    def _create_generic_recommendation(self, medication_id: str, warning: str) -> DosingRecommendation:
        """Create generic recommendation for unknown medications"""
        return DosingRecommendation(
            medication_id=medication_id,
            dose="Consult clinical pharmacist",
            frequency="As directed",
            route="As prescribed",
            duration="As directed",
            rationale="Medication not in dosing database",
            warnings=[warning],
            adjustments=[]
        )
    
    def _create_contraindicated_recommendation(self, medication_id: str, reason: str) -> DosingRecommendation:
        """Create recommendation for contraindicated medications"""
        return DosingRecommendation(
            medication_id=medication_id,
            dose="CONTRAINDICATED",
            frequency="N/A",
            route="N/A",
            duration="N/A",
            rationale=f"Contraindicated: {reason}",
            warnings=[f"CONTRAINDICATED: {reason}", "Consider alternative therapy"],
            adjustments=[]
        )
