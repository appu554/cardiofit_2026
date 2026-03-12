"""
Therapeutic Drug Monitoring (TDM) Service
Implements Bayesian dosing and target level calculations
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass
import math

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class TDMType(Enum):
    """Types of therapeutic drug monitoring"""
    PEAK_TROUGH = "peak_trough"      # Peak and trough levels
    STEADY_STATE = "steady_state"    # Steady-state levels
    AUC_MONITORING = "auc_monitoring" # Area under curve monitoring
    RANDOM_LEVEL = "random_level"    # Random level monitoring


class SamplingTime(Enum):
    """Drug level sampling times"""
    PEAK = "peak"                    # Peak concentration
    TROUGH = "trough"                # Trough concentration
    RANDOM = "random"                # Random time point
    STEADY_STATE = "steady_state"    # At steady state


@dataclass
class DrugLevel:
    """Individual drug level measurement"""
    medication_id: str
    patient_id: str
    level_value: Decimal             # Drug concentration
    level_unit: str                  # mg/L, mcg/mL, etc.
    sampling_time: SamplingTime
    collection_datetime: datetime
    dose_given: Decimal              # Dose that produced this level
    time_since_dose_hours: Decimal   # Hours since last dose
    
    def is_therapeutic(self, target_range: Tuple[Decimal, Decimal]) -> bool:
        """Check if level is within therapeutic range"""
        min_level, max_level = target_range
        return min_level <= self.level_value <= max_level
    
    def is_toxic(self, toxic_threshold: Decimal) -> bool:
        """Check if level is toxic"""
        return self.level_value > toxic_threshold


@dataclass
class TDMTarget:
    """Therapeutic drug monitoring targets"""
    medication_id: str
    target_peak_min: Optional[Decimal] = None
    target_peak_max: Optional[Decimal] = None
    target_trough_min: Optional[Decimal] = None
    target_trough_max: Optional[Decimal] = None
    toxic_threshold: Optional[Decimal] = None
    target_auc: Optional[Decimal] = None
    level_unit: str = "mg/L"
    
    def get_peak_range(self) -> Optional[Tuple[Decimal, Decimal]]:
        """Get peak therapeutic range"""
        if self.target_peak_min and self.target_peak_max:
            return (self.target_peak_min, self.target_peak_max)
        return None
    
    def get_trough_range(self) -> Optional[Tuple[Decimal, Decimal]]:
        """Get trough therapeutic range"""
        if self.target_trough_min and self.target_trough_max:
            return (self.target_trough_min, self.target_trough_max)
        return None


@dataclass
class PharmacokineticParameters:
    """Individual patient pharmacokinetic parameters"""
    patient_id: str
    medication_id: str
    clearance_l_hr: Optional[Decimal] = None      # Clearance (L/hr)
    volume_distribution_l: Optional[Decimal] = None # Volume of distribution (L)
    half_life_hours: Optional[Decimal] = None     # Half-life (hours)
    bioavailability: Optional[Decimal] = None     # F (0-1)
    calculated_from_levels: bool = False          # Calculated vs population values
    last_updated: Optional[datetime] = None
    
    def calculate_elimination_constant(self) -> Optional[Decimal]:
        """Calculate elimination rate constant (ke)"""
        if self.half_life_hours:
            return Decimal(str(math.log(2))) / self.half_life_hours
        elif self.clearance_l_hr and self.volume_distribution_l:
            return self.clearance_l_hr / self.volume_distribution_l
        return None


class TherapeuticDrugMonitoringService:
    """
    Service for therapeutic drug monitoring and Bayesian dosing
    
    Implements:
    - Target level calculations
    - Bayesian dose adjustments
    - Population pharmacokinetics
    - Individual kinetic parameter estimation
    """
    
    def __init__(self, tdm_repository, lab_service):
        self.tdm_repository = tdm_repository
        self.lab_service = lab_service
        self.tdm_targets = self._load_tdm_targets()
        self.population_pk = self._load_population_pk_parameters()
    
    async def calculate_tdm_dose_adjustment(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any],
        recent_levels: List[DrugLevel],
        target_level: Optional[Decimal] = None
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate dose adjustment based on drug levels using Bayesian approach
        
        Returns:
            - Adjusted dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        medication_id = medication_properties.get('medication_id')
        
        if not recent_levels:
            clinical_notes.append("No recent drug levels available for TDM adjustment")
            return dose, warnings, clinical_notes
        
        # Get TDM targets for this medication
        tdm_target = self.tdm_targets.get(medication_id)
        if not tdm_target:
            clinical_notes.append("No TDM targets defined for this medication")
            return dose, warnings, clinical_notes
        
        # Get or calculate individual PK parameters
        pk_params = await self._get_individual_pk_parameters(
            context.patient_id, medication_id, recent_levels
        )
        
        # Determine target level if not provided
        if not target_level:
            target_level = self._determine_target_level(recent_levels, tdm_target)
        
        # Calculate dose adjustment using Bayesian approach
        adjusted_dose_value = self._calculate_bayesian_dose_adjustment(
            dose.value, recent_levels, target_level, pk_params, tdm_target
        )
        
        # Create adjusted dose specification
        adjustment_factors = dict(dose.calculation_factors)
        adjustment_factors.update({
            'tdm_adjustment_factor': float(adjusted_dose_value / dose.value),
            'target_level': float(target_level),
            'current_level': float(recent_levels[-1].level_value),
            'pk_clearance': float(pk_params.clearance_l_hr) if pk_params.clearance_l_hr else None,
            'pk_half_life': float(pk_params.half_life_hours) if pk_params.half_life_hours else None,
            'original_dose': float(dose.value)
        })
        
        adjusted_dose = DoseSpecification(
            value=adjusted_dose_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_tdm_adjusted",
            calculation_factors=adjustment_factors
        )
        
        # Generate clinical notes
        change_percent = ((adjusted_dose_value / dose.value) - Decimal('1.0')) * 100
        if abs(change_percent) > 5:  # Only note significant changes
            if change_percent > 0:
                clinical_notes.append(f"Dose increased by {change_percent:.1f}% based on drug levels")
            else:
                clinical_notes.append(f"Dose reduced by {abs(change_percent):.1f}% based on drug levels")
        else:
            clinical_notes.append("Minimal dose adjustment based on drug levels")
        
        # Add level interpretation
        latest_level = recent_levels[-1]
        level_interpretation = self._interpret_drug_level(latest_level, tdm_target)
        clinical_notes.append(level_interpretation)
        
        # Add monitoring recommendations
        monitoring_recs = self._get_tdm_monitoring_recommendations(
            medication_id, adjusted_dose_value, pk_params
        )
        clinical_notes.extend(monitoring_recs)
        
        # Check for toxic levels
        if tdm_target.toxic_threshold and latest_level.level_value > tdm_target.toxic_threshold:
            warnings.append(f"Current level ({latest_level.level_value} {tdm_target.level_unit}) exceeds toxic threshold")
            clinical_notes.append("Consider dose hold and repeat level")
        
        logger.info(f"TDM dose adjustment: {dose.value} -> {adjusted_dose_value} {dose.unit.value}")
        
        return adjusted_dose, warnings, clinical_notes
    
    async def predict_drug_levels(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_id: str,
        prediction_times: List[int]  # Hours after dose
    ) -> List[Dict[str, Any]]:
        """
        Predict drug levels at specified times after dose
        
        Returns list of predicted levels with timestamps
        """
        predictions = []
        
        # Get individual PK parameters
        pk_params = await self._get_individual_pk_parameters(
            context.patient_id, medication_id, []
        )
        
        if not pk_params.clearance_l_hr or not pk_params.volume_distribution_l:
            logger.warning("Insufficient PK parameters for level prediction")
            return predictions
        
        # Calculate elimination rate constant
        ke = pk_params.calculate_elimination_constant()
        if not ke:
            return predictions
        
        # Predict levels using one-compartment model
        for hours_after_dose in prediction_times:
            # C(t) = (Dose/Vd) * e^(-ke*t)
            predicted_level = (dose.value / pk_params.volume_distribution_l) * Decimal(str(math.exp(-float(ke * Decimal(str(hours_after_dose))))))
            
            predictions.append({
                'hours_after_dose': hours_after_dose,
                'predicted_level': float(predicted_level),
                'level_unit': self.tdm_targets.get(medication_id, TDMTarget(medication_id)).level_unit
            })
        
        return predictions
    
    def get_next_sampling_recommendation(
        self,
        medication_id: str,
        last_dose_time: datetime,
        pk_params: PharmacokineticParameters
    ) -> Dict[str, Any]:
        """Get recommendation for next drug level sampling"""
        
        tdm_target = self.tdm_targets.get(medication_id)
        if not tdm_target:
            return {'recommendation': 'No TDM targets defined'}
        
        # Calculate time to steady state (5 half-lives)
        if pk_params.half_life_hours:
            steady_state_hours = pk_params.half_life_hours * 5
            steady_state_time = last_dose_time + timedelta(hours=float(steady_state_hours))
            
            return {
                'recommendation': 'Sample at steady state',
                'recommended_time': steady_state_time.isoformat(),
                'hours_from_last_dose': float(steady_state_hours),
                'sampling_type': 'trough',
                'notes': 'Collect trough level just before next dose'
            }
        
        # Default recommendation
        return {
            'recommendation': 'Sample in 24-48 hours',
            'sampling_type': 'trough',
            'notes': 'Collect trough level just before next dose'
        }
    
    # === PRIVATE HELPER METHODS ===
    
    async def _get_individual_pk_parameters(
        self,
        patient_id: str,
        medication_id: str,
        recent_levels: List[DrugLevel]
    ) -> PharmacokineticParameters:
        """Get or calculate individual PK parameters"""
        
        # Try to get existing individual parameters
        individual_pk = await self.tdm_repository.get_pk_parameters(patient_id, medication_id)
        
        if individual_pk and individual_pk.calculated_from_levels:
            return individual_pk
        
        # Calculate from recent levels if available
        if len(recent_levels) >= 2:
            calculated_pk = self._calculate_pk_from_levels(recent_levels)
            if calculated_pk:
                # Save calculated parameters
                await self.tdm_repository.save_pk_parameters(calculated_pk)
                return calculated_pk
        
        # Fall back to population parameters
        population_pk = self.population_pk.get(medication_id)
        if population_pk:
            return PharmacokineticParameters(
                patient_id=patient_id,
                medication_id=medication_id,
                clearance_l_hr=population_pk['clearance_l_hr'],
                volume_distribution_l=population_pk['volume_distribution_l'],
                half_life_hours=population_pk['half_life_hours'],
                bioavailability=population_pk.get('bioavailability', Decimal('1.0')),
                calculated_from_levels=False
            )
        
        # Default parameters if nothing available
        return PharmacokineticParameters(
            patient_id=patient_id,
            medication_id=medication_id,
            half_life_hours=Decimal('12'),  # Default 12-hour half-life
            calculated_from_levels=False
        )
    
    def _calculate_pk_from_levels(self, levels: List[DrugLevel]) -> Optional[PharmacokineticParameters]:
        """Calculate PK parameters from drug levels using first-order kinetics"""
        
        if len(levels) < 2:
            return None
        
        # Sort levels by collection time
        sorted_levels = sorted(levels, key=lambda x: x.collection_datetime)
        
        # Use first two levels for calculation
        level1 = sorted_levels[0]
        level2 = sorted_levels[1]
        
        # Calculate time difference in hours
        time_diff = (level2.collection_datetime - level1.collection_datetime).total_seconds() / 3600
        
        if time_diff <= 0:
            return None
        
        # Calculate elimination rate constant: ke = ln(C1/C2) / (t2-t1)
        if level2.level_value > 0:
            ke = Decimal(str(math.log(float(level1.level_value / level2.level_value)))) / Decimal(str(time_diff))
            
            # Calculate half-life: t1/2 = ln(2) / ke
            half_life = Decimal(str(math.log(2))) / ke
            
            # Estimate volume of distribution: Vd = Dose / C0
            # Extrapolate back to C0 using: C0 = C1 * e^(ke * t1)
            time_to_level1 = float(level1.time_since_dose_hours)
            c0 = level1.level_value * Decimal(str(math.exp(float(ke) * time_to_level1)))
            vd = level1.dose_given / c0
            
            # Calculate clearance: Cl = ke * Vd
            clearance = ke * vd
            
            return PharmacokineticParameters(
                patient_id=level1.patient_id,
                medication_id=level1.medication_id,
                clearance_l_hr=clearance,
                volume_distribution_l=vd,
                half_life_hours=half_life,
                calculated_from_levels=True,
                last_updated=datetime.utcnow()
            )
        
        return None
    
    def _calculate_bayesian_dose_adjustment(
        self,
        current_dose: Decimal,
        recent_levels: List[DrugLevel],
        target_level: Decimal,
        pk_params: PharmacokineticParameters,
        tdm_target: TDMTarget
    ) -> Decimal:
        """Calculate dose adjustment using Bayesian approach"""
        
        latest_level = recent_levels[-1]
        
        # Simple linear adjustment: New Dose = Current Dose * (Target Level / Current Level)
        if latest_level.level_value > 0:
            adjustment_factor = target_level / latest_level.level_value
            
            # Apply safety limits (don't change dose by more than 50% at once)
            if adjustment_factor > Decimal('1.5'):
                adjustment_factor = Decimal('1.5')
            elif adjustment_factor < Decimal('0.5'):
                adjustment_factor = Decimal('0.5')
            
            return current_dose * adjustment_factor
        
        return current_dose
    
    def _determine_target_level(self, recent_levels: List[DrugLevel], tdm_target: TDMTarget) -> Decimal:
        """Determine appropriate target level based on sampling type"""
        
        latest_level = recent_levels[-1]
        
        if latest_level.sampling_time == SamplingTime.PEAK:
            if tdm_target.target_peak_min and tdm_target.target_peak_max:
                # Target middle of peak range
                return (tdm_target.target_peak_min + tdm_target.target_peak_max) / 2
        elif latest_level.sampling_time == SamplingTime.TROUGH:
            if tdm_target.target_trough_min and tdm_target.target_trough_max:
                # Target middle of trough range
                return (tdm_target.target_trough_min + tdm_target.target_trough_max) / 2
        
        # Default to current level if no specific target
        return latest_level.level_value
    
    def _interpret_drug_level(self, level: DrugLevel, tdm_target: TDMTarget) -> str:
        """Interpret drug level relative to therapeutic targets"""
        
        if level.sampling_time == SamplingTime.PEAK:
            peak_range = tdm_target.get_peak_range()
            if peak_range:
                if level.is_therapeutic(peak_range):
                    return f"Peak level ({level.level_value} {tdm_target.level_unit}) is therapeutic"
                elif level.level_value < peak_range[0]:
                    return f"Peak level ({level.level_value} {tdm_target.level_unit}) is subtherapeutic"
                else:
                    return f"Peak level ({level.level_value} {tdm_target.level_unit}) is supratherapeutic"
        
        elif level.sampling_time == SamplingTime.TROUGH:
            trough_range = tdm_target.get_trough_range()
            if trough_range:
                if level.is_therapeutic(trough_range):
                    return f"Trough level ({level.level_value} {tdm_target.level_unit}) is therapeutic"
                elif level.level_value < trough_range[0]:
                    return f"Trough level ({level.level_value} {tdm_target.level_unit}) is subtherapeutic"
                else:
                    return f"Trough level ({level.level_value} {tdm_target.level_unit}) is supratherapeutic"
        
        return f"Drug level: {level.level_value} {tdm_target.level_unit}"
    
    def _get_tdm_monitoring_recommendations(
        self,
        medication_id: str,
        adjusted_dose: Decimal,
        pk_params: PharmacokineticParameters
    ) -> List[str]:
        """Get TDM monitoring recommendations"""
        
        recommendations = []
        
        if pk_params.half_life_hours:
            steady_state_days = (pk_params.half_life_hours * 5) / 24
            recommendations.append(f"Repeat level at steady state (~{steady_state_days:.1f} days)")
        else:
            recommendations.append("Repeat level in 3-5 days")
        
        recommendations.append("Collect trough level just before next dose")
        
        # Medication-specific recommendations
        if medication_id in ['vancomycin', 'gentamicin', 'tobramycin']:
            recommendations.append("Monitor renal function")
        elif medication_id in ['digoxin']:
            recommendations.append("Monitor electrolytes (K+, Mg2+)")
        elif medication_id in ['phenytoin', 'carbamazepine']:
            recommendations.append("Monitor for signs of toxicity")
        
        return recommendations
    
    def _load_tdm_targets(self) -> Dict[str, TDMTarget]:
        """Load therapeutic drug monitoring targets"""
        
        targets = {}
        
        # Vancomycin
        targets['vancomycin'] = TDMTarget(
            medication_id='vancomycin',
            target_trough_min=Decimal('10'),
            target_trough_max=Decimal('20'),
            toxic_threshold=Decimal('30'),
            level_unit='mg/L'
        )
        
        # Gentamicin
        targets['gentamicin'] = TDMTarget(
            medication_id='gentamicin',
            target_peak_min=Decimal('5'),
            target_peak_max=Decimal('10'),
            target_trough_min=Decimal('0.5'),
            target_trough_max=Decimal('2'),
            toxic_threshold=Decimal('12'),
            level_unit='mg/L'
        )
        
        # Digoxin
        targets['digoxin'] = TDMTarget(
            medication_id='digoxin',
            target_trough_min=Decimal('1.0'),
            target_trough_max=Decimal('2.0'),
            toxic_threshold=Decimal('2.5'),
            level_unit='mcg/L'
        )
        
        # Phenytoin
        targets['phenytoin'] = TDMTarget(
            medication_id='phenytoin',
            target_trough_min=Decimal('10'),
            target_trough_max=Decimal('20'),
            toxic_threshold=Decimal('25'),
            level_unit='mg/L'
        )
        
        return targets
    
    def _load_population_pk_parameters(self) -> Dict[str, Dict[str, Decimal]]:
        """Load population pharmacokinetic parameters"""
        
        parameters = {}
        
        # Vancomycin (adult population)
        parameters['vancomycin'] = {
            'clearance_l_hr': Decimal('4.0'),
            'volume_distribution_l': Decimal('70'),
            'half_life_hours': Decimal('6'),
            'bioavailability': Decimal('1.0')
        }
        
        # Gentamicin
        parameters['gentamicin'] = {
            'clearance_l_hr': Decimal('5.4'),
            'volume_distribution_l': Decimal('18'),
            'half_life_hours': Decimal('2.5'),
            'bioavailability': Decimal('1.0')
        }
        
        # Digoxin
        parameters['digoxin'] = {
            'clearance_l_hr': Decimal('1.2'),
            'volume_distribution_l': Decimal('420'),
            'half_life_hours': Decimal('36'),
            'bioavailability': Decimal('0.7')
        }
        
        return parameters
