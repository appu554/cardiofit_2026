"""
Renal Dose Adjustment Service
Implements clinical algorithms for kidney function-based dose adjustments
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal, ROUND_HALF_UP
from enum import Enum
from dataclasses import dataclass

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class RenalFunction(Enum):
    """Renal function categories based on eGFR/CrCl"""
    NORMAL = "normal"           # ≥90 mL/min/1.73m²
    MILD_IMPAIRMENT = "mild"    # 60-89 mL/min/1.73m²
    MODERATE_IMPAIRMENT = "moderate"  # 30-59 mL/min/1.73m²
    SEVERE_IMPAIRMENT = "severe"      # 15-29 mL/min/1.73m²
    KIDNEY_FAILURE = "failure"        # <15 mL/min/1.73m² or dialysis


class RenalAdjustmentType(Enum):
    """Types of renal dose adjustments"""
    DOSE_REDUCTION = "dose_reduction"
    INTERVAL_EXTENSION = "interval_extension"
    COMBINATION = "combination"
    CONTRAINDICATED = "contraindicated"


@dataclass
class RenalAdjustmentRule:
    """Rule for renal dose adjustment"""
    medication_class: str
    renal_function: RenalFunction
    adjustment_type: RenalAdjustmentType
    dose_factor: Optional[Decimal] = None
    interval_multiplier: Optional[Decimal] = None
    max_dose: Optional[Decimal] = None
    contraindicated: bool = False
    monitoring_required: bool = False
    notes: Optional[str] = None


class RenalDoseAdjustmentService:
    """
    Service for calculating renal dose adjustments
    
    Implements clinical guidelines for dose modifications based on kidney function
    Uses eGFR, creatinine clearance, and patient-specific factors
    """
    
    def __init__(self):
        self.adjustment_rules = self._load_adjustment_rules()
    
    def calculate_renal_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate renal dose adjustment
        
        Returns:
            - Adjusted dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        # Determine renal function
        renal_function = self._assess_renal_function(context)
        clinical_notes.append(f"Renal function assessed as: {renal_function.value}")
        
        # Check if adjustment is needed
        if renal_function == RenalFunction.NORMAL:
            clinical_notes.append("No renal dose adjustment required")
            return dose, warnings, clinical_notes
        
        # Get medication-specific adjustment rules
        medication_class = medication_properties.get('pharmacologic_class', 'unknown')
        adjustment_rule = self._get_adjustment_rule(medication_class, renal_function)
        
        if not adjustment_rule:
            warnings.append(f"No specific renal adjustment rule found for {medication_class}")
            # Apply conservative default adjustment
            adjustment_rule = self._get_default_adjustment(renal_function)
        
        # Check for contraindications
        if adjustment_rule.contraindicated:
            warnings.append("Medication contraindicated in this degree of renal impairment")
            clinical_notes.append(f"Consider alternative therapy")
        
        # Apply dose adjustment
        adjusted_dose = self._apply_adjustment(dose, adjustment_rule, context)
        
        # Add clinical notes
        if adjustment_rule.dose_factor and adjustment_rule.dose_factor < Decimal('1.0'):
            reduction_percent = (Decimal('1.0') - adjustment_rule.dose_factor) * 100
            clinical_notes.append(f"Dose reduced by {reduction_percent}% for renal impairment")
        
        if adjustment_rule.interval_multiplier and adjustment_rule.interval_multiplier > Decimal('1.0'):
            clinical_notes.append(f"Dosing interval extended by {adjustment_rule.interval_multiplier}x")
        
        if adjustment_rule.monitoring_required:
            clinical_notes.append("Enhanced renal function monitoring recommended")
        
        if adjustment_rule.notes:
            clinical_notes.append(adjustment_rule.notes)
        
        logger.info(f"Renal adjustment applied: {dose.value} -> {adjusted_dose.value} {dose.unit.value}")
        
        return adjusted_dose, warnings, clinical_notes
    
    def _assess_renal_function(self, context: DoseCalculationContext) -> RenalFunction:
        """Assess renal function based on available data"""
        
        # Prefer eGFR over creatinine clearance
        gfr = context.egfr or context.creatinine_clearance
        
        if not gfr:
            logger.warning("No renal function data available, assuming normal function")
            return RenalFunction.NORMAL
        
        # Classify based on eGFR/CrCl (mL/min/1.73m²)
        if gfr >= 90:
            return RenalFunction.NORMAL
        elif gfr >= 60:
            return RenalFunction.MILD_IMPAIRMENT
        elif gfr >= 30:
            return RenalFunction.MODERATE_IMPAIRMENT
        elif gfr >= 15:
            return RenalFunction.SEVERE_IMPAIRMENT
        else:
            return RenalFunction.KIDNEY_FAILURE
    
    def _get_adjustment_rule(
        self, 
        medication_class: str, 
        renal_function: RenalFunction
    ) -> Optional[RenalAdjustmentRule]:
        """Get specific adjustment rule for medication class and renal function"""
        
        key = f"{medication_class}_{renal_function.value}"
        return self.adjustment_rules.get(key)
    
    def _get_default_adjustment(self, renal_function: RenalFunction) -> RenalAdjustmentRule:
        """Get conservative default adjustment when specific rule not available"""
        
        if renal_function == RenalFunction.MILD_IMPAIRMENT:
            return RenalAdjustmentRule(
                medication_class="default",
                renal_function=renal_function,
                adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.9'),
                notes="Conservative 10% dose reduction for mild renal impairment"
            )
        elif renal_function == RenalFunction.MODERATE_IMPAIRMENT:
            return RenalAdjustmentRule(
                medication_class="default",
                renal_function=renal_function,
                adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.75'),
                monitoring_required=True,
                notes="25% dose reduction for moderate renal impairment"
            )
        elif renal_function == RenalFunction.SEVERE_IMPAIRMENT:
            return RenalAdjustmentRule(
                medication_class="default",
                renal_function=renal_function,
                adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.5'),
                monitoring_required=True,
                notes="50% dose reduction for severe renal impairment"
            )
        else:  # KIDNEY_FAILURE
            return RenalAdjustmentRule(
                medication_class="default",
                renal_function=renal_function,
                adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.25'),
                monitoring_required=True,
                notes="75% dose reduction for kidney failure - consider alternative therapy"
            )
    
    def _apply_adjustment(
        self,
        dose: DoseSpecification,
        rule: RenalAdjustmentRule,
        context: DoseCalculationContext
    ) -> DoseSpecification:
        """Apply the adjustment rule to the dose"""
        
        adjusted_value = dose.value
        adjustment_method = dose.calculation_method
        adjustment_factors = dict(dose.calculation_factors)
        
        # Apply dose reduction
        if rule.dose_factor:
            adjusted_value = adjusted_value * rule.dose_factor
            adjustment_method += "_renal_adjusted"
            adjustment_factors['renal_dose_factor'] = float(rule.dose_factor)
        
        # Apply maximum dose limit
        if rule.max_dose and adjusted_value > rule.max_dose:
            adjusted_value = rule.max_dose
            adjustment_factors['renal_max_dose_applied'] = float(rule.max_dose)
        
        # Round to appropriate precision
        adjusted_value = adjusted_value.quantize(Decimal('0.1'), rounding=ROUND_HALF_UP)
        
        # Add renal function info to calculation factors
        gfr = context.egfr or context.creatinine_clearance
        adjustment_factors.update({
            'renal_function': rule.renal_function.value,
            'gfr_egfr': float(gfr) if gfr else None,
            'adjustment_type': rule.adjustment_type.value,
            'original_dose': float(dose.value)
        })
        
        return DoseSpecification(
            value=adjusted_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=adjustment_method,
            calculation_factors=adjustment_factors
        )
    
    def _load_adjustment_rules(self) -> Dict[str, RenalAdjustmentRule]:
        """Load medication-specific renal adjustment rules"""
        
        rules = {}
        
        # ACE Inhibitors
        rules["ace_inhibitor_moderate"] = RenalAdjustmentRule(
            medication_class="ace_inhibitor",
            renal_function=RenalFunction.MODERATE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            monitoring_required=True,
            notes="Monitor potassium and creatinine closely"
        )
        
        rules["ace_inhibitor_severe"] = RenalAdjustmentRule(
            medication_class="ace_inhibitor",
            renal_function=RenalFunction.SEVERE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            monitoring_required=True,
            notes="Start with 50% dose reduction, monitor closely"
        )
        
        # NSAIDs
        rules["nsaid_moderate"] = RenalAdjustmentRule(
            medication_class="nsaid",
            renal_function=RenalFunction.MODERATE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.CONTRAINDICATED,
            contraindicated=True,
            notes="NSAIDs contraindicated in moderate-severe renal impairment"
        )
        
        rules["nsaid_severe"] = RenalAdjustmentRule(
            medication_class="nsaid",
            renal_function=RenalFunction.SEVERE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.CONTRAINDICATED,
            contraindicated=True,
            notes="NSAIDs contraindicated in severe renal impairment"
        )
        
        # Antibiotics - Aminoglycosides
        rules["aminoglycoside_mild"] = RenalAdjustmentRule(
            medication_class="aminoglycoside",
            renal_function=RenalFunction.MILD_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.INTERVAL_EXTENSION,
            interval_multiplier=Decimal('1.5'),
            monitoring_required=True,
            notes="Extend dosing interval, monitor drug levels"
        )
        
        rules["aminoglycoside_moderate"] = RenalAdjustmentRule(
            medication_class="aminoglycoside",
            renal_function=RenalFunction.MODERATE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.COMBINATION,
            dose_factor=Decimal('0.75'),
            interval_multiplier=Decimal('2.0'),
            monitoring_required=True,
            notes="Reduce dose by 25% and double interval, monitor levels closely"
        )
        
        # Digoxin
        rules["digoxin_mild"] = RenalAdjustmentRule(
            medication_class="cardiac_glycoside",
            renal_function=RenalFunction.MILD_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            monitoring_required=True,
            notes="Reduce dose by 25%, monitor digoxin levels"
        )
        
        rules["digoxin_moderate"] = RenalAdjustmentRule(
            medication_class="cardiac_glycoside",
            renal_function=RenalFunction.MODERATE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            monitoring_required=True,
            notes="Reduce dose by 50%, monitor digoxin levels closely"
        )
        
        # Metformin
        rules["metformin_moderate"] = RenalAdjustmentRule(
            medication_class="biguanide",
            renal_function=RenalFunction.MODERATE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.CONTRAINDICATED,
            contraindicated=True,
            notes="Metformin contraindicated when eGFR <30 mL/min/1.73m²"
        )
        
        # Anticoagulants - Dabigatran
        rules["dabigatran_moderate"] = RenalAdjustmentRule(
            medication_class="direct_thrombin_inhibitor",
            renal_function=RenalFunction.MODERATE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            monitoring_required=True,
            notes="Reduce dose for CrCl 30-50 mL/min"
        )
        
        rules["dabigatran_severe"] = RenalAdjustmentRule(
            medication_class="direct_thrombin_inhibitor",
            renal_function=RenalFunction.SEVERE_IMPAIRMENT,
            adjustment_type=RenalAdjustmentType.CONTRAINDICATED,
            contraindicated=True,
            notes="Dabigatran contraindicated when CrCl <30 mL/min"
        )
        
        return rules
    
    def get_renal_function_category(self, context: DoseCalculationContext) -> str:
        """Get renal function category as string"""
        return self._assess_renal_function(context).value
    
    def requires_renal_adjustment(
        self, 
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> bool:
        """Check if medication requires renal dose adjustment"""
        
        renal_function = self._assess_renal_function(context)
        if renal_function == RenalFunction.NORMAL:
            return False
        
        # Check if medication is renally eliminated
        renal_elimination = medication_properties.get('renal_elimination_percent', 0)
        if renal_elimination > 50:  # >50% renal elimination
            return True
        
        # Check for specific medication classes that require adjustment
        medication_class = medication_properties.get('pharmacologic_class', '')
        high_risk_classes = [
            'ace_inhibitor', 'aminoglycoside', 'cardiac_glycoside', 
            'biguanide', 'direct_thrombin_inhibitor'
        ]
        
        return medication_class in high_risk_classes
    
    def get_monitoring_recommendations(
        self, 
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> List[str]:
        """Get renal monitoring recommendations"""
        
        recommendations = []
        renal_function = self._assess_renal_function(context)
        
        if renal_function != RenalFunction.NORMAL:
            recommendations.append("Monitor renal function (SCr, eGFR) regularly")
            
            if renal_function in [RenalFunction.SEVERE_IMPAIRMENT, RenalFunction.KIDNEY_FAILURE]:
                recommendations.append("Consider nephrology consultation")
                recommendations.append("Monitor for signs of drug accumulation")
            
            medication_class = medication_properties.get('pharmacologic_class', '')
            if medication_class == 'aminoglycoside':
                recommendations.append("Monitor drug levels (peak and trough)")
            elif medication_class == 'cardiac_glycoside':
                recommendations.append("Monitor digoxin levels")
            elif medication_class == 'ace_inhibitor':
                recommendations.append("Monitor potassium levels")
        
        return recommendations
