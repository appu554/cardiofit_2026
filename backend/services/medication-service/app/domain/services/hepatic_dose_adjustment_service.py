"""
Hepatic Dose Adjustment Service
Implements clinical algorithms for liver function-based dose adjustments
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal, ROUND_HALF_UP
from enum import Enum
from dataclasses import dataclass

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class HepaticFunction(Enum):
    """Hepatic function categories based on Child-Pugh classification"""
    NORMAL = "normal"           # Child-Pugh A or normal liver function
    MILD_IMPAIRMENT = "mild"    # Child-Pugh A (5-6 points)
    MODERATE_IMPAIRMENT = "moderate"  # Child-Pugh B (7-9 points)
    SEVERE_IMPAIRMENT = "severe"      # Child-Pugh C (10-15 points)


class HepaticAdjustmentType(Enum):
    """Types of hepatic dose adjustments"""
    DOSE_REDUCTION = "dose_reduction"
    INTERVAL_EXTENSION = "interval_extension"
    COMBINATION = "combination"
    CONTRAINDICATED = "contraindicated"
    AVOID_HEPATOTOXIC = "avoid_hepatotoxic"


@dataclass
class HepaticAdjustmentRule:
    """Rule for hepatic dose adjustment"""
    medication_class: str
    hepatic_function: HepaticFunction
    adjustment_type: HepaticAdjustmentType
    dose_factor: Optional[Decimal] = None
    interval_multiplier: Optional[Decimal] = None
    max_dose: Optional[Decimal] = None
    contraindicated: bool = False
    monitoring_required: bool = False
    notes: Optional[str] = None


class HepaticDoseAdjustmentService:
    """
    Service for calculating hepatic dose adjustments
    
    Implements clinical guidelines for dose modifications based on liver function
    Uses Child-Pugh classification and hepatic metabolism considerations
    """
    
    def __init__(self):
        self.adjustment_rules = self._load_adjustment_rules()
    
    def calculate_hepatic_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate hepatic dose adjustment
        
        Returns:
            - Adjusted dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        # Determine hepatic function
        hepatic_function = self._assess_hepatic_function(context)
        clinical_notes.append(f"Hepatic function assessed as: {hepatic_function.value}")
        
        # Check if adjustment is needed
        if hepatic_function == HepaticFunction.NORMAL:
            clinical_notes.append("No hepatic dose adjustment required")
            return dose, warnings, clinical_notes
        
        # Get medication-specific adjustment rules
        medication_class = medication_properties.get('pharmacologic_class', 'unknown')
        adjustment_rule = self._get_adjustment_rule(medication_class, hepatic_function)
        
        if not adjustment_rule:
            # Check if medication is hepatically metabolized
            hepatic_metabolism = medication_properties.get('hepatic_metabolism_percent', 0)
            if hepatic_metabolism > 50:  # >50% hepatic metabolism
                warnings.append(f"No specific hepatic adjustment rule found for {medication_class}")
                adjustment_rule = self._get_default_adjustment(hepatic_function)
            else:
                clinical_notes.append("Medication not significantly hepatically metabolized")
                return dose, warnings, clinical_notes
        
        # Check for contraindications
        if adjustment_rule.contraindicated:
            warnings.append("Medication contraindicated in this degree of hepatic impairment")
            clinical_notes.append("Consider alternative therapy")
        
        # Check for hepatotoxicity concerns
        is_hepatotoxic = medication_properties.get('is_hepatotoxic', False)
        if is_hepatotoxic and hepatic_function != HepaticFunction.NORMAL:
            warnings.append("Hepatotoxic medication in patient with hepatic impairment")
            clinical_notes.append("Enhanced liver function monitoring required")
        
        # Apply dose adjustment
        adjusted_dose = self._apply_adjustment(dose, adjustment_rule, context)
        
        # Add clinical notes
        if adjustment_rule.dose_factor and adjustment_rule.dose_factor < Decimal('1.0'):
            reduction_percent = (Decimal('1.0') - adjustment_rule.dose_factor) * 100
            clinical_notes.append(f"Dose reduced by {reduction_percent}% for hepatic impairment")
        
        if adjustment_rule.interval_multiplier and adjustment_rule.interval_multiplier > Decimal('1.0'):
            clinical_notes.append(f"Dosing interval extended by {adjustment_rule.interval_multiplier}x")
        
        if adjustment_rule.monitoring_required:
            clinical_notes.append("Enhanced liver function monitoring recommended")
        
        if adjustment_rule.notes:
            clinical_notes.append(adjustment_rule.notes)
        
        logger.info(f"Hepatic adjustment applied: {dose.value} -> {adjusted_dose.value} {dose.unit.value}")
        
        return adjusted_dose, warnings, clinical_notes
    
    def _assess_hepatic_function(self, context: DoseCalculationContext) -> HepaticFunction:
        """Assess hepatic function based on available data"""
        
        liver_function = context.liver_function
        
        if not liver_function:
            logger.warning("No hepatic function data available, assuming normal function")
            return HepaticFunction.NORMAL
        
        # Map string values to enum
        function_mapping = {
            'normal': HepaticFunction.NORMAL,
            'mild': HepaticFunction.MILD_IMPAIRMENT,
            'moderate': HepaticFunction.MODERATE_IMPAIRMENT,
            'severe': HepaticFunction.SEVERE_IMPAIRMENT
        }
        
        return function_mapping.get(liver_function.lower(), HepaticFunction.NORMAL)
    
    def _get_adjustment_rule(
        self, 
        medication_class: str, 
        hepatic_function: HepaticFunction
    ) -> Optional[HepaticAdjustmentRule]:
        """Get specific adjustment rule for medication class and hepatic function"""
        
        key = f"{medication_class}_{hepatic_function.value}"
        return self.adjustment_rules.get(key)
    
    def _get_default_adjustment(self, hepatic_function: HepaticFunction) -> HepaticAdjustmentRule:
        """Get conservative default adjustment when specific rule not available"""
        
        if hepatic_function == HepaticFunction.MILD_IMPAIRMENT:
            return HepaticAdjustmentRule(
                medication_class="default",
                hepatic_function=hepatic_function,
                adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.75'),
                monitoring_required=True,
                notes="Conservative 25% dose reduction for mild hepatic impairment"
            )
        elif hepatic_function == HepaticFunction.MODERATE_IMPAIRMENT:
            return HepaticAdjustmentRule(
                medication_class="default",
                hepatic_function=hepatic_function,
                adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.5'),
                monitoring_required=True,
                notes="50% dose reduction for moderate hepatic impairment"
            )
        else:  # SEVERE_IMPAIRMENT
            return HepaticAdjustmentRule(
                medication_class="default",
                hepatic_function=hepatic_function,
                adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
                dose_factor=Decimal('0.25'),
                monitoring_required=True,
                notes="75% dose reduction for severe hepatic impairment - consider alternative therapy"
            )
    
    def _apply_adjustment(
        self,
        dose: DoseSpecification,
        rule: HepaticAdjustmentRule,
        context: DoseCalculationContext
    ) -> DoseSpecification:
        """Apply the adjustment rule to the dose"""
        
        adjusted_value = dose.value
        adjustment_method = dose.calculation_method
        adjustment_factors = dict(dose.calculation_factors)
        
        # Apply dose reduction
        if rule.dose_factor:
            adjusted_value = adjusted_value * rule.dose_factor
            adjustment_method += "_hepatic_adjusted"
            adjustment_factors['hepatic_dose_factor'] = float(rule.dose_factor)
        
        # Apply maximum dose limit
        if rule.max_dose and adjusted_value > rule.max_dose:
            adjusted_value = rule.max_dose
            adjustment_factors['hepatic_max_dose_applied'] = float(rule.max_dose)
        
        # Round to appropriate precision
        adjusted_value = adjusted_value.quantize(Decimal('0.1'), rounding=ROUND_HALF_UP)
        
        # Add hepatic function info to calculation factors
        adjustment_factors.update({
            'hepatic_function': rule.hepatic_function.value,
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
    
    def _load_adjustment_rules(self) -> Dict[str, HepaticAdjustmentRule]:
        """Load medication-specific hepatic adjustment rules"""
        
        rules = {}
        
        # Statins
        rules["statin_mild"] = HepaticAdjustmentRule(
            medication_class="statin",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            monitoring_required=True,
            notes="Reduce statin dose by 50%, monitor liver enzymes"
        )
        
        rules["statin_moderate"] = HepaticAdjustmentRule(
            medication_class="statin",
            hepatic_function=HepaticFunction.MODERATE_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.CONTRAINDICATED,
            contraindicated=True,
            notes="Statins contraindicated in moderate-severe hepatic impairment"
        )
        
        # Acetaminophen/Paracetamol
        rules["acetaminophen_mild"] = HepaticAdjustmentRule(
            medication_class="analgesic_antipyretic",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            max_dose=Decimal('2000'),  # 2g/day max
            monitoring_required=True,
            notes="Reduce dose and limit to 2g/day maximum"
        )
        
        rules["acetaminophen_moderate"] = HepaticAdjustmentRule(
            medication_class="analgesic_antipyretic",
            hepatic_function=HepaticFunction.MODERATE_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            max_dose=Decimal('1000'),  # 1g/day max
            monitoring_required=True,
            notes="Reduce dose by 50%, limit to 1g/day maximum"
        )
        
        # Warfarin
        rules["warfarin_mild"] = HepaticAdjustmentRule(
            medication_class="anticoagulant",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            monitoring_required=True,
            notes="Reduce warfarin dose, monitor INR closely"
        )
        
        rules["warfarin_moderate"] = HepaticAdjustmentRule(
            medication_class="anticoagulant",
            hepatic_function=HepaticFunction.MODERATE_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            monitoring_required=True,
            notes="Reduce warfarin dose by 50%, monitor INR very closely"
        )
        
        # Benzodiazepines
        rules["benzodiazepine_mild"] = HepaticAdjustmentRule(
            medication_class="benzodiazepine",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            notes="Reduce benzodiazepine dose by 50% due to decreased metabolism"
        )
        
        rules["benzodiazepine_moderate"] = HepaticAdjustmentRule(
            medication_class="benzodiazepine",
            hepatic_function=HepaticFunction.MODERATE_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.25'),
            notes="Reduce benzodiazepine dose by 75%, consider alternative"
        )
        
        # Opioids
        rules["opioid_mild"] = HepaticAdjustmentRule(
            medication_class="opioid",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            monitoring_required=True,
            notes="Reduce opioid dose by 25%, monitor for increased sedation"
        )
        
        rules["opioid_moderate"] = HepaticAdjustmentRule(
            medication_class="opioid",
            hepatic_function=HepaticFunction.MODERATE_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            monitoring_required=True,
            notes="Reduce opioid dose by 50%, monitor closely for toxicity"
        )
        
        # Proton Pump Inhibitors
        rules["proton_pump_inhibitor_mild"] = HepaticAdjustmentRule(
            medication_class="proton_pump_inhibitor",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            notes="Reduce PPI dose by 50% in hepatic impairment"
        )
        
        # Antiepileptics - Phenytoin
        rules["antiepileptic_mild"] = HepaticAdjustmentRule(
            medication_class="antiepileptic",
            hepatic_function=HepaticFunction.MILD_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.75'),
            monitoring_required=True,
            notes="Reduce dose by 25%, monitor drug levels closely"
        )
        
        rules["antiepileptic_moderate"] = HepaticAdjustmentRule(
            medication_class="antiepileptic",
            hepatic_function=HepaticFunction.MODERATE_IMPAIRMENT,
            adjustment_type=HepaticAdjustmentType.DOSE_REDUCTION,
            dose_factor=Decimal('0.5'),
            monitoring_required=True,
            notes="Reduce dose by 50%, monitor drug levels and liver function"
        )
        
        return rules
    
    def get_hepatic_function_category(self, context: DoseCalculationContext) -> str:
        """Get hepatic function category as string"""
        return self._assess_hepatic_function(context).value
    
    def requires_hepatic_adjustment(
        self, 
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> bool:
        """Check if medication requires hepatic dose adjustment"""
        
        hepatic_function = self._assess_hepatic_function(context)
        if hepatic_function == HepaticFunction.NORMAL:
            return False
        
        # Check if medication is hepatically metabolized
        hepatic_metabolism = medication_properties.get('hepatic_metabolism_percent', 0)
        if hepatic_metabolism > 50:  # >50% hepatic metabolism
            return True
        
        # Check for specific medication classes that require adjustment
        medication_class = medication_properties.get('pharmacologic_class', '')
        high_risk_classes = [
            'statin', 'analgesic_antipyretic', 'anticoagulant', 'benzodiazepine',
            'opioid', 'proton_pump_inhibitor', 'antiepileptic'
        ]
        
        return medication_class in high_risk_classes
    
    def get_monitoring_recommendations(
        self, 
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> List[str]:
        """Get hepatic monitoring recommendations"""
        
        recommendations = []
        hepatic_function = self._assess_hepatic_function(context)
        
        if hepatic_function != HepaticFunction.NORMAL:
            recommendations.append("Monitor liver function tests (ALT, AST, bilirubin) regularly")
            
            if hepatic_function == HepaticFunction.SEVERE_IMPAIRMENT:
                recommendations.append("Consider hepatology consultation")
                recommendations.append("Monitor for signs of drug accumulation and toxicity")
            
            medication_class = medication_properties.get('pharmacologic_class', '')
            if medication_class == 'anticoagulant':
                recommendations.append("Monitor INR/PT closely")
            elif medication_class == 'antiepileptic':
                recommendations.append("Monitor drug levels")
            elif medication_class in ['benzodiazepine', 'opioid']:
                recommendations.append("Monitor for increased sedation and respiratory depression")
            
            is_hepatotoxic = medication_properties.get('is_hepatotoxic', False)
            if is_hepatotoxic:
                recommendations.append("Enhanced monitoring for hepatotoxicity")
                recommendations.append("Consider baseline and periodic liver function tests")
        
        return recommendations
    
    def assess_hepatotoxicity_risk(
        self, 
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> Tuple[str, List[str]]:
        """Assess hepatotoxicity risk level and recommendations"""
        
        hepatic_function = self._assess_hepatic_function(context)
        is_hepatotoxic = medication_properties.get('is_hepatotoxic', False)
        
        if not is_hepatotoxic:
            return "low", []
        
        if hepatic_function == HepaticFunction.NORMAL:
            return "moderate", ["Monitor for signs of hepatotoxicity"]
        elif hepatic_function == HepaticFunction.MILD_IMPAIRMENT:
            return "high", [
                "Enhanced monitoring for hepatotoxicity",
                "Consider alternative non-hepatotoxic medication"
            ]
        else:
            return "very_high", [
                "Avoid hepatotoxic medication if possible",
                "If no alternative, use lowest effective dose with intensive monitoring",
                "Consider hepatology consultation"
            ]
