"""
Advanced Pharmacokinetics Service
Implements population pharmacokinetics and complex PK modeling
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal
from enum import Enum
from dataclasses import dataclass
import math

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class PKModel(Enum):
    """Pharmacokinetic model types"""
    ONE_COMPARTMENT = "one_compartment"
    TWO_COMPARTMENT = "two_compartment"
    MICHAELIS_MENTEN = "michaelis_menten"
    FLIP_FLOP = "flip_flop"


class DosingRegimen(Enum):
    """Dosing regimen types"""
    SINGLE_DOSE = "single_dose"
    MULTIPLE_DOSE = "multiple_dose"
    CONTINUOUS_INFUSION = "continuous_infusion"
    INTERMITTENT_INFUSION = "intermittent_infusion"


@dataclass
class PopulationPKParameters:
    """Population pharmacokinetic parameters"""
    medication_id: str
    clearance_l_hr_kg: Decimal       # Clearance per kg
    volume_central_l_kg: Decimal     # Central volume per kg
    volume_peripheral_l_kg: Optional[Decimal] = None  # Peripheral volume per kg
    ka_hr: Optional[Decimal] = None  # Absorption rate constant
    bioavailability: Decimal = Decimal('1.0')
    protein_binding_percent: Decimal = Decimal('0')
    
    # Covariates effects
    age_effect_clearance: Optional[Decimal] = None
    creatinine_effect_clearance: Optional[Decimal] = None
    weight_effect_volume: Optional[Decimal] = None


@dataclass
class PKPrediction:
    """Pharmacokinetic prediction result"""
    time_hours: Decimal
    concentration: Decimal
    auc_to_time: Optional[Decimal] = None
    peak_time: Optional[Decimal] = None
    peak_concentration: Optional[Decimal] = None


class AdvancedPharmacokineticsService:
    """
    Service for advanced pharmacokinetic calculations
    
    Implements:
    - Population pharmacokinetics
    - Multi-compartment modeling
    - Covariate effects
    - AUC calculations
    - Bioavailability adjustments
    """
    
    def __init__(self):
        self.population_pk = self._load_population_pk_parameters()
        self.covariate_models = self._load_covariate_models()
    
    def calculate_pk_guided_dose(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any],
        target_auc: Optional[Decimal] = None,
        target_peak: Optional[Decimal] = None,
        target_trough: Optional[Decimal] = None
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate dose using advanced pharmacokinetic modeling
        
        Returns:
            - PK-optimized dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        medication_id = medication_properties.get('medication_id')
        
        # Get population PK parameters
        pop_pk = self.population_pk.get(medication_id)
        if not pop_pk:
            clinical_notes.append("No population PK model available")
            return dose, warnings, clinical_notes
        
        # Calculate individual PK parameters with covariate effects
        individual_pk = self._calculate_individual_pk_parameters(pop_pk, context)
        
        # Determine dosing strategy based on targets
        if target_auc:
            adjusted_dose_value = self._calculate_auc_targeted_dose(
                target_auc, individual_pk, medication_properties
            )
            clinical_notes.append(f"Dose calculated to achieve target AUC: {target_auc}")
        elif target_peak:
            adjusted_dose_value = self._calculate_peak_targeted_dose(
                target_peak, individual_pk, medication_properties
            )
            clinical_notes.append(f"Dose calculated to achieve target peak: {target_peak}")
        elif target_trough:
            adjusted_dose_value = self._calculate_trough_targeted_dose(
                target_trough, individual_pk, medication_properties
            )
            clinical_notes.append(f"Dose calculated to achieve target trough: {target_trough}")
        else:
            # Use standard clearance-based dosing
            adjusted_dose_value = self._calculate_clearance_based_dose(
                dose.value, individual_pk, context
            )
            clinical_notes.append("Dose adjusted based on individual clearance")
        
        # Apply bioavailability correction for oral dosing
        if dose.route.value in ['PO', 'ORAL'] and pop_pk.bioavailability != Decimal('1.0'):
            adjusted_dose_value = adjusted_dose_value / pop_pk.bioavailability
            clinical_notes.append(f"Dose adjusted for bioavailability ({pop_pk.bioavailability * 100}%)")
        
        # Create adjusted dose specification
        adjustment_factors = dict(dose.calculation_factors)
        adjustment_factors.update({
            'pk_clearance_l_hr': float(individual_pk['clearance_l_hr']),
            'pk_volume_l': float(individual_pk['volume_central_l']),
            'pk_half_life_hr': float(individual_pk['half_life_hours']),
            'bioavailability': float(pop_pk.bioavailability),
            'pk_adjustment_factor': float(adjusted_dose_value / dose.value),
            'original_dose': float(dose.value)
        })
        
        adjusted_dose = DoseSpecification(
            value=adjusted_dose_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_pk_optimized",
            calculation_factors=adjustment_factors
        )
        
        # Generate clinical notes
        change_percent = ((adjusted_dose_value / dose.value) - Decimal('1.0')) * 100
        if abs(change_percent) > 10:  # Only note significant changes
            if change_percent > 0:
                clinical_notes.append(f"Dose increased by {change_percent:.1f}% based on PK modeling")
            else:
                clinical_notes.append(f"Dose reduced by {abs(change_percent):.1f}% based on PK modeling")
        
        # Add covariate effects notes
        covariate_notes = self._generate_covariate_notes(pop_pk, context)
        clinical_notes.extend(covariate_notes)
        
        logger.info(f"PK-guided dose adjustment: {dose.value} -> {adjusted_dose_value} {dose.unit.value}")
        
        return adjusted_dose, warnings, clinical_notes
    
    def predict_concentration_time_profile(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_id: str,
        prediction_hours: int = 24,
        dosing_interval_hours: Optional[int] = None
    ) -> List[PKPrediction]:
        """
        Predict concentration-time profile using population PK
        
        Returns list of concentration predictions over time
        """
        predictions = []
        
        pop_pk = self.population_pk.get(medication_id)
        if not pop_pk:
            return predictions
        
        # Calculate individual PK parameters
        individual_pk = self._calculate_individual_pk_parameters(pop_pk, context)
        
        # Generate predictions every hour
        for hour in range(prediction_hours + 1):
            time_point = Decimal(str(hour))
            
            if dosing_interval_hours and hour > 0 and hour % dosing_interval_hours == 0:
                # Multiple dose - add contribution from each dose
                concentration = self._calculate_multiple_dose_concentration(
                    dose.value, time_point, individual_pk, dosing_interval_hours
                )
            else:
                # Single dose concentration
                concentration = self._calculate_single_dose_concentration(
                    dose.value, time_point, individual_pk
                )
            
            # Calculate AUC to this time point
            auc_to_time = self._calculate_auc_to_time(
                dose.value, time_point, individual_pk
            )
            
            predictions.append(PKPrediction(
                time_hours=time_point,
                concentration=concentration,
                auc_to_time=auc_to_time
            ))
        
        # Find peak concentration and time
        if predictions:
            peak_prediction = max(predictions, key=lambda p: p.concentration)
            for pred in predictions:
                pred.peak_time = peak_prediction.time_hours
                pred.peak_concentration = peak_prediction.concentration
        
        return predictions
    
    def calculate_bioequivalence_adjustment(
        self,
        dose: DoseSpecification,
        reference_formulation: Dict[str, Any],
        test_formulation: Dict[str, Any]
    ) -> Tuple[DoseSpecification, List[str]]:
        """
        Calculate dose adjustment for bioequivalence differences
        
        Adjusts dose when switching between formulations with different bioavailability
        """
        clinical_notes = []
        
        ref_bioavailability = Decimal(str(reference_formulation.get('bioavailability', 1.0)))
        test_bioavailability = Decimal(str(test_formulation.get('bioavailability', 1.0)))
        
        if ref_bioavailability == test_bioavailability:
            clinical_notes.append("No bioequivalence adjustment needed")
            return dose, clinical_notes
        
        # Adjust dose to maintain same systemic exposure
        adjustment_factor = ref_bioavailability / test_bioavailability
        adjusted_dose_value = dose.value * adjustment_factor
        
        adjustment_factors = dict(dose.calculation_factors)
        adjustment_factors.update({
            'bioequivalence_factor': float(adjustment_factor),
            'reference_bioavailability': float(ref_bioavailability),
            'test_bioavailability': float(test_bioavailability),
            'original_dose': float(dose.value)
        })
        
        adjusted_dose = DoseSpecification(
            value=adjusted_dose_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_bioequivalence_adjusted",
            calculation_factors=adjustment_factors
        )
        
        change_percent = (adjustment_factor - Decimal('1.0')) * 100
        if change_percent > 0:
            clinical_notes.append(f"Dose increased by {change_percent:.1f}% for lower bioavailability formulation")
        else:
            clinical_notes.append(f"Dose reduced by {abs(change_percent):.1f}% for higher bioavailability formulation")
        
        clinical_notes.append("Monitor for efficacy/toxicity changes with formulation switch")
        
        return adjusted_dose, clinical_notes
    
    # === PRIVATE HELPER METHODS ===
    
    def _calculate_individual_pk_parameters(
        self,
        pop_pk: PopulationPKParameters,
        context: DoseCalculationContext
    ) -> Dict[str, Decimal]:
        """Calculate individual PK parameters from population parameters and covariates"""
        
        # Base parameters scaled by weight
        weight = context.weight_kg or Decimal('70')  # Default 70kg
        
        clearance = pop_pk.clearance_l_hr_kg * weight
        volume_central = pop_pk.volume_central_l_kg * weight
        
        # Apply covariate effects
        if pop_pk.age_effect_clearance and context.age_years:
            age_factor = self._calculate_age_effect(context.age_years, pop_pk.age_effect_clearance)
            clearance = clearance * age_factor
        
        if pop_pk.creatinine_effect_clearance and context.creatinine_clearance:
            renal_factor = self._calculate_renal_effect(
                context.creatinine_clearance, pop_pk.creatinine_effect_clearance
            )
            clearance = clearance * renal_factor
        
        # Calculate derived parameters
        half_life = (Decimal(str(math.log(2))) * volume_central) / clearance
        
        return {
            'clearance_l_hr': clearance,
            'volume_central_l': volume_central,
            'half_life_hours': half_life,
            'elimination_constant': clearance / volume_central
        }
    
    def _calculate_auc_targeted_dose(
        self,
        target_auc: Decimal,
        individual_pk: Dict[str, Decimal],
        medication_properties: Dict[str, Any]
    ) -> Decimal:
        """Calculate dose to achieve target AUC"""
        
        # AUC = Dose / Clearance (for IV dosing)
        clearance = individual_pk['clearance_l_hr']
        return target_auc * clearance
    
    def _calculate_peak_targeted_dose(
        self,
        target_peak: Decimal,
        individual_pk: Dict[str, Decimal],
        medication_properties: Dict[str, Any]
    ) -> Decimal:
        """Calculate dose to achieve target peak concentration"""
        
        # Cmax = Dose / Vd (for IV bolus)
        volume = individual_pk['volume_central_l']
        return target_peak * volume
    
    def _calculate_trough_targeted_dose(
        self,
        target_trough: Decimal,
        individual_pk: Dict[str, Decimal],
        medication_properties: Dict[str, Any]
    ) -> Decimal:
        """Calculate dose to achieve target trough concentration"""
        
        # Assume 24-hour dosing interval
        dosing_interval = Decimal('24')
        ke = individual_pk['elimination_constant']
        volume = individual_pk['volume_central_l']
        
        # Ctrough = (Dose/Vd) * e^(-ke*tau)
        # Rearranging: Dose = Ctrough * Vd / e^(-ke*tau)
        elimination_factor = Decimal(str(math.exp(-float(ke * dosing_interval))))
        return target_trough * volume / elimination_factor
    
    def _calculate_clearance_based_dose(
        self,
        standard_dose: Decimal,
        individual_pk: Dict[str, Decimal],
        context: DoseCalculationContext
    ) -> Decimal:
        """Calculate dose based on individual clearance vs population average"""
        
        # Assume population average clearance of 4 L/hr for 70kg adult
        population_clearance = Decimal('4.0')
        individual_clearance = individual_pk['clearance_l_hr']
        
        # Adjust dose proportionally to clearance
        return standard_dose * (individual_clearance / population_clearance)
    
    def _calculate_single_dose_concentration(
        self,
        dose: Decimal,
        time: Decimal,
        individual_pk: Dict[str, Decimal]
    ) -> Decimal:
        """Calculate concentration at time t after single dose"""
        
        volume = individual_pk['volume_central_l']
        ke = individual_pk['elimination_constant']
        
        # C(t) = (Dose/Vd) * e^(-ke*t)
        c0 = dose / volume
        concentration = c0 * Decimal(str(math.exp(-float(ke * time))))
        
        return concentration
    
    def _calculate_multiple_dose_concentration(
        self,
        dose: Decimal,
        time: Decimal,
        individual_pk: Dict[str, Decimal],
        dosing_interval: int
    ) -> Decimal:
        """Calculate concentration considering multiple doses"""
        
        ke = individual_pk['elimination_constant']
        tau = Decimal(str(dosing_interval))
        
        # Number of doses given by this time
        num_doses = int(time / tau) + 1
        
        total_concentration = Decimal('0')
        
        # Sum contribution from each dose
        for dose_num in range(num_doses):
            dose_time = Decimal(str(dose_num)) * tau
            time_since_dose = time - dose_time
            
            if time_since_dose >= 0:
                concentration = self._calculate_single_dose_concentration(
                    dose, time_since_dose, individual_pk
                )
                total_concentration += concentration
        
        return total_concentration
    
    def _calculate_auc_to_time(
        self,
        dose: Decimal,
        time: Decimal,
        individual_pk: Dict[str, Decimal]
    ) -> Decimal:
        """Calculate AUC from 0 to time t"""
        
        clearance = individual_pk['clearance_l_hr']
        ke = individual_pk['elimination_constant']
        
        # AUC(0-t) = (Dose/Cl) * (1 - e^(-ke*t))
        elimination_factor = Decimal(str(math.exp(-float(ke * time))))
        auc = (dose / clearance) * (Decimal('1') - elimination_factor)
        
        return auc
    
    def _calculate_age_effect(self, age_years: int, age_coefficient: Decimal) -> Decimal:
        """Calculate age effect on clearance"""
        
        # Typical age effect: Cl = Cl_typical * (Age/40)^age_coefficient
        reference_age = Decimal('40')
        age_factor = (Decimal(str(age_years)) / reference_age) ** age_coefficient
        
        return age_factor
    
    def _calculate_renal_effect(
        self,
        creatinine_clearance: Decimal,
        renal_coefficient: Decimal
    ) -> Decimal:
        """Calculate renal function effect on clearance"""
        
        # Typical renal effect: Cl = Cl_typical * (CrCl/120)^renal_coefficient
        normal_crcl = Decimal('120')
        renal_factor = (creatinine_clearance / normal_crcl) ** renal_coefficient
        
        return renal_factor
    
    def _generate_covariate_notes(
        self,
        pop_pk: PopulationPKParameters,
        context: DoseCalculationContext
    ) -> List[str]:
        """Generate notes about covariate effects"""
        
        notes = []
        
        if pop_pk.age_effect_clearance and context.age_years:
            if context.age_years < 18:
                notes.append("Pediatric age effect applied to clearance")
            elif context.age_years > 65:
                notes.append("Geriatric age effect applied to clearance")
        
        if pop_pk.creatinine_effect_clearance and context.creatinine_clearance:
            if context.creatinine_clearance < 60:
                notes.append("Renal impairment effect applied to clearance")
        
        if context.weight_kg and (context.weight_kg < 50 or context.weight_kg > 100):
            notes.append("Weight-based scaling applied to PK parameters")
        
        return notes
    
    def _load_population_pk_parameters(self) -> Dict[str, PopulationPKParameters]:
        """Load population pharmacokinetic parameters"""
        
        parameters = {}
        
        # Vancomycin
        parameters['vancomycin'] = PopulationPKParameters(
            medication_id='vancomycin',
            clearance_l_hr_kg=Decimal('0.057'),  # 4 L/hr for 70kg
            volume_central_l_kg=Decimal('1.0'),  # 70 L for 70kg
            bioavailability=Decimal('1.0'),
            age_effect_clearance=Decimal('-0.3'),
            creatinine_effect_clearance=Decimal('0.8')
        )
        
        # Digoxin
        parameters['digoxin'] = PopulationPKParameters(
            medication_id='digoxin',
            clearance_l_hr_kg=Decimal('0.017'),  # 1.2 L/hr for 70kg
            volume_central_l_kg=Decimal('6.0'),  # 420 L for 70kg
            bioavailability=Decimal('0.7'),
            age_effect_clearance=Decimal('-0.2'),
            creatinine_effect_clearance=Decimal('0.9')
        )
        
        return parameters
    
    def _load_covariate_models(self) -> Dict[str, Dict[str, Any]]:
        """Load covariate effect models"""
        
        models = {
            'age_on_clearance': {
                'pediatric': {'coefficient': Decimal('0.75'), 'reference_age': 40},
                'geriatric': {'coefficient': Decimal('-0.3'), 'reference_age': 40}
            },
            'weight_on_volume': {
                'allometric': {'coefficient': Decimal('1.0'), 'reference_weight': 70}
            },
            'renal_on_clearance': {
                'linear': {'coefficient': Decimal('0.8'), 'reference_crcl': 120}
            }
        }
        
        return models
