"""
Special Populations Service
Handles dosing adjustments for pregnancy, lactation, and other special populations
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal
from enum import Enum
from dataclasses import dataclass

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class PregnancyCategory(Enum):
    """FDA Pregnancy Categories"""
    A = "A"  # Adequate, well-controlled studies in pregnant women have not shown increased risk
    B = "B"  # Animal studies show no risk, but no adequate human studies
    C = "C"  # Animal studies show adverse effects, no adequate human studies
    D = "D"  # Evidence of human fetal risk, but benefits may outweigh risks
    X = "X"  # Evidence of fetal abnormalities, risks outweigh benefits


class LactationRisk(Enum):
    """Lactation risk categories"""
    COMPATIBLE = "compatible"           # Safe during breastfeeding
    PROBABLY_COMPATIBLE = "probably_compatible"  # Likely safe
    CAUTION = "caution"                # Use with caution, monitor infant
    CONTRAINDICATED = "contraindicated"  # Avoid during breastfeeding
    UNKNOWN = "unknown"                # Insufficient data


class CriticalIllnessType(Enum):
    """Types of critical illness affecting dosing"""
    SEPSIS = "sepsis"
    SHOCK = "shock"
    MULTI_ORGAN_FAILURE = "multi_organ_failure"
    BURNS = "burns"
    TRAUMA = "trauma"


@dataclass
class PregnancyDoseAdjustment:
    """Pregnancy-specific dose adjustment"""
    medication_id: str
    pregnancy_category: PregnancyCategory
    trimester_specific: Dict[int, Decimal]  # Trimester -> dose factor
    contraindicated: bool = False
    alternative_medications: Optional[List[str]] = None
    monitoring_requirements: Optional[List[str]] = None
    notes: Optional[str] = None


@dataclass
class LactationDoseAdjustment:
    """Lactation-specific dose adjustment"""
    medication_id: str
    lactation_risk: LactationRisk
    dose_factor: Optional[Decimal] = None
    contraindicated: bool = False
    alternative_medications: Optional[List[str]] = None
    infant_monitoring: Optional[List[str]] = None
    notes: Optional[str] = None


class SpecialPopulationsService:
    """
    Service for special population dose adjustments
    
    Handles:
    - Pregnancy dosing adjustments
    - Lactation considerations
    - Critical illness dosing
    - Pharmacogenomic adjustments (when available)
    """
    
    def __init__(self):
        self.pregnancy_adjustments = self._load_pregnancy_adjustments()
        self.lactation_adjustments = self._load_lactation_adjustments()
    
    def calculate_pregnancy_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any],
        trimester: int
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate pregnancy-specific dose adjustment
        
        Returns:
            - Adjusted dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        if not context.pregnancy_status:
            clinical_notes.append("No pregnancy dose adjustment needed")
            return dose, warnings, clinical_notes
        
        medication_id = medication_properties.get('medication_id')
        adjustment = self.pregnancy_adjustments.get(medication_id)
        
        if not adjustment:
            # Check pregnancy category from medication properties
            pregnancy_category = medication_properties.get('pregnancy_category')
            if pregnancy_category:
                category_enum = PregnancyCategory(pregnancy_category)
                if category_enum in [PregnancyCategory.D, PregnancyCategory.X]:
                    warnings.append(f"Pregnancy Category {category_enum.value} - Use with extreme caution or avoid")
                    clinical_notes.append("Consider alternative therapy if available")
            
            return dose, warnings, clinical_notes
        
        # Check for contraindications
        if adjustment.contraindicated:
            warnings.append("Medication contraindicated in pregnancy")
            if adjustment.alternative_medications:
                clinical_notes.append(f"Consider alternatives: {', '.join(adjustment.alternative_medications)}")
            return dose, warnings, clinical_notes
        
        # Apply trimester-specific adjustments
        dose_factor = adjustment.trimester_specific.get(trimester, Decimal('1.0'))
        
        if dose_factor != Decimal('1.0'):
            adjusted_value = dose.value * dose_factor
            adjustment_factors = dict(dose.calculation_factors)
            adjustment_factors.update({
                'pregnancy_dose_factor': float(dose_factor),
                'trimester': trimester,
                'pregnancy_category': adjustment.pregnancy_category.value,
                'original_dose': float(dose.value)
            })
            
            adjusted_dose = DoseSpecification(
                value=adjusted_value,
                unit=dose.unit,
                route=dose.route,
                calculation_method=f"{dose.calculation_method}_pregnancy_adjusted",
                calculation_factors=adjustment_factors
            )
            
            change_percent = (dose_factor - Decimal('1.0')) * 100
            if change_percent > 0:
                clinical_notes.append(f"Dose increased by {change_percent}% for pregnancy (trimester {trimester})")
            else:
                clinical_notes.append(f"Dose reduced by {abs(change_percent)}% for pregnancy (trimester {trimester})")
        else:
            adjusted_dose = dose
            clinical_notes.append(f"No dose adjustment needed for trimester {trimester}")
        
        # Add monitoring requirements
        if adjustment.monitoring_requirements:
            clinical_notes.extend(adjustment.monitoring_requirements)
        
        if adjustment.notes:
            clinical_notes.append(adjustment.notes)
        
        return adjusted_dose, warnings, clinical_notes
    
    def calculate_lactation_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate lactation-specific dose adjustment
        
        Returns:
            - Adjusted dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        if not context.breastfeeding_status:
            clinical_notes.append("No lactation dose adjustment needed")
            return dose, warnings, clinical_notes
        
        medication_id = medication_properties.get('medication_id')
        adjustment = self.lactation_adjustments.get(medication_id)
        
        if not adjustment:
            warnings.append("Lactation safety data not available - use with caution")
            clinical_notes.append("Monitor infant for adverse effects")
            return dose, warnings, clinical_notes
        
        # Check for contraindications
        if adjustment.contraindicated:
            warnings.append("Medication contraindicated during breastfeeding")
            if adjustment.alternative_medications:
                clinical_notes.append(f"Consider alternatives: {', '.join(adjustment.alternative_medications)}")
            return dose, warnings, clinical_notes
        
        # Apply dose adjustments if needed
        if adjustment.dose_factor and adjustment.dose_factor != Decimal('1.0'):
            adjusted_value = dose.value * adjustment.dose_factor
            adjustment_factors = dict(dose.calculation_factors)
            adjustment_factors.update({
                'lactation_dose_factor': float(adjustment.dose_factor),
                'lactation_risk': adjustment.lactation_risk.value,
                'original_dose': float(dose.value)
            })
            
            adjusted_dose = DoseSpecification(
                value=adjusted_value,
                unit=dose.unit,
                route=dose.route,
                calculation_method=f"{dose.calculation_method}_lactation_adjusted",
                calculation_factors=adjustment_factors
            )
            
            change_percent = (adjustment.dose_factor - Decimal('1.0')) * 100
            if change_percent > 0:
                clinical_notes.append(f"Dose increased by {change_percent}% for lactation safety")
            else:
                clinical_notes.append(f"Dose reduced by {abs(change_percent)}% for lactation safety")
        else:
            adjusted_dose = dose
        
        # Add risk category information
        clinical_notes.append(f"Lactation risk category: {adjustment.lactation_risk.value}")
        
        # Add infant monitoring requirements
        if adjustment.infant_monitoring:
            clinical_notes.append("Monitor infant for:")
            clinical_notes.extend([f"  - {monitor}" for monitor in adjustment.infant_monitoring])
        
        if adjustment.notes:
            clinical_notes.append(adjustment.notes)
        
        return adjusted_dose, warnings, clinical_notes
    
    def calculate_critical_illness_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any],
        illness_type: CriticalIllnessType
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate critical illness dose adjustment
        
        Critical illness can affect pharmacokinetics significantly
        """
        warnings = []
        clinical_notes = []
        
        # Critical illness adjustments based on pathophysiology
        adjustment_factors = {
            CriticalIllnessType.SEPSIS: {
                'volume_expansion': Decimal('1.3'),  # Increased Vd
                'clearance_reduction': Decimal('0.7')  # Reduced clearance
            },
            CriticalIllnessType.SHOCK: {
                'volume_expansion': Decimal('1.4'),
                'clearance_reduction': Decimal('0.6')
            },
            CriticalIllnessType.BURNS: {
                'volume_expansion': Decimal('1.5'),  # Significant fluid shifts
                'clearance_increase': Decimal('1.2')  # Hypermetabolic state
            }
        }
        
        factors = adjustment_factors.get(illness_type)
        if not factors:
            clinical_notes.append(f"No specific adjustment for {illness_type.value}")
            return dose, warnings, clinical_notes
        
        # Apply volume of distribution changes (affects loading doses)
        if 'loading' in dose.calculation_method.lower():
            if 'volume_expansion' in factors:
                adjusted_value = dose.value * factors['volume_expansion']
                clinical_notes.append(f"Loading dose increased for {illness_type.value} (expanded Vd)")
            else:
                adjusted_value = dose.value
        else:
            # Apply clearance changes (affects maintenance doses)
            if 'clearance_reduction' in factors:
                adjusted_value = dose.value * factors['clearance_reduction']
                clinical_notes.append(f"Maintenance dose reduced for {illness_type.value} (reduced clearance)")
            elif 'clearance_increase' in factors:
                adjusted_value = dose.value * factors['clearance_increase']
                clinical_notes.append(f"Maintenance dose increased for {illness_type.value} (increased clearance)")
            else:
                adjusted_value = dose.value
        
        adjustment_calculation_factors = dict(dose.calculation_factors)
        adjustment_calculation_factors.update({
            'critical_illness': illness_type.value,
            'critical_illness_factor': float(adjusted_value / dose.value),
            'original_dose': float(dose.value)
        })
        
        adjusted_dose = DoseSpecification(
            value=adjusted_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_critical_illness",
            calculation_factors=adjustment_calculation_factors
        )
        
        warnings.append(f"Dose adjusted for {illness_type.value}")
        clinical_notes.extend([
            "Enhanced monitoring recommended in critical illness",
            "Consider therapeutic drug monitoring if available",
            "Reassess dose as clinical status changes"
        ])
        
        return adjusted_dose, warnings, clinical_notes
    
    # === PRIVATE HELPER METHODS ===
    
    def _load_pregnancy_adjustments(self) -> Dict[str, PregnancyDoseAdjustment]:
        """Load pregnancy-specific dose adjustments"""
        
        adjustments = {}
        
        # Antihypertensives safe in pregnancy
        adjustments['methyldopa'] = PregnancyDoseAdjustment(
            medication_id='methyldopa',
            pregnancy_category=PregnancyCategory.B,
            trimester_specific={1: Decimal('1.0'), 2: Decimal('1.0'), 3: Decimal('1.0')},
            monitoring_requirements=["Monitor blood pressure", "Monitor fetal growth"],
            notes="First-line antihypertensive in pregnancy"
        )
        
        # Antibiotics
        adjustments['amoxicillin'] = PregnancyDoseAdjustment(
            medication_id='amoxicillin',
            pregnancy_category=PregnancyCategory.B,
            trimester_specific={1: Decimal('1.0'), 2: Decimal('1.2'), 3: Decimal('1.3')},
            notes="Increased clearance in pregnancy may require dose increase"
        )
        
        # Contraindicated medications
        adjustments['warfarin'] = PregnancyDoseAdjustment(
            medication_id='warfarin',
            pregnancy_category=PregnancyCategory.X,
            trimester_specific={},
            contraindicated=True,
            alternative_medications=['heparin', 'enoxaparin'],
            notes="Teratogenic - use heparin instead"
        )
        
        return adjustments
    
    def _load_lactation_adjustments(self) -> Dict[str, LactationDoseAdjustment]:
        """Load lactation-specific dose adjustments"""
        
        adjustments = {}
        
        # Compatible medications
        adjustments['acetaminophen'] = LactationDoseAdjustment(
            medication_id='acetaminophen',
            lactation_risk=LactationRisk.COMPATIBLE,
            notes="Minimal transfer to breast milk"
        )
        
        adjustments['ibuprofen'] = LactationDoseAdjustment(
            medication_id='ibuprofen',
            lactation_risk=LactationRisk.COMPATIBLE,
            notes="Preferred NSAID during breastfeeding"
        )
        
        # Caution medications
        adjustments['codeine'] = LactationDoseAdjustment(
            medication_id='codeine',
            lactation_risk=LactationRisk.CAUTION,
            dose_factor=Decimal('0.75'),  # Reduce dose
            infant_monitoring=['sedation', 'feeding difficulties', 'breathing problems'],
            notes="Risk of infant sedation, especially in ultrarapid metabolizers"
        )
        
        # Contraindicated medications
        adjustments['lithium'] = LactationDoseAdjustment(
            medication_id='lithium',
            lactation_risk=LactationRisk.CONTRAINDICATED,
            contraindicated=True,
            notes="High milk/plasma ratio, risk of infant toxicity"
        )
        
        return adjustments
    
    def get_pregnancy_category(self, medication_id: str) -> Optional[PregnancyCategory]:
        """Get pregnancy category for medication"""
        adjustment = self.pregnancy_adjustments.get(medication_id)
        return adjustment.pregnancy_category if adjustment else None
    
    def get_lactation_risk(self, medication_id: str) -> Optional[LactationRisk]:
        """Get lactation risk category for medication"""
        adjustment = self.lactation_adjustments.get(medication_id)
        return adjustment.lactation_risk if adjustment else None
    
    def is_safe_in_pregnancy(self, medication_id: str) -> bool:
        """Check if medication is safe in pregnancy"""
        adjustment = self.pregnancy_adjustments.get(medication_id)
        if not adjustment:
            return False  # Unknown safety
        
        return (not adjustment.contraindicated and 
                adjustment.pregnancy_category in [PregnancyCategory.A, PregnancyCategory.B])
    
    def is_safe_in_lactation(self, medication_id: str) -> bool:
        """Check if medication is safe during lactation"""
        adjustment = self.lactation_adjustments.get(medication_id)
        if not adjustment:
            return False  # Unknown safety
        
        return (not adjustment.contraindicated and 
                adjustment.lactation_risk in [LactationRisk.COMPATIBLE, LactationRisk.PROBABLY_COMPATIBLE])
