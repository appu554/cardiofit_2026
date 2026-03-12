"""
Dose Banding & Practical Considerations Service
Implements dose banding for chemotherapy and practical dosing considerations
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal
from enum import Enum
from dataclasses import dataclass

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext

logger = logging.getLogger(__name__)


class DoseBandingType(Enum):
    """Types of dose banding"""
    CHEMOTHERAPY = "chemotherapy"        # Chemotherapy dose banding
    PEDIATRIC = "pediatric"             # Pediatric dose banding
    INSULIN = "insulin"                 # Insulin dose banding
    STANDARD = "standard"               # Standard dose banding


class RoundingStrategy(Enum):
    """Dose rounding strategies"""
    NEAREST_UNIT = "nearest_unit"       # Round to nearest whole unit
    NEAREST_HALF = "nearest_half"       # Round to nearest 0.5
    NEAREST_QUARTER = "nearest_quarter" # Round to nearest 0.25
    TABLET_STRENGTH = "tablet_strength" # Round to available tablet strengths
    VIAL_SIZE = "vial_size"            # Round to minimize vial waste


@dataclass
class DoseBand:
    """Individual dose band definition"""
    band_id: str
    min_calculated_dose: Decimal
    max_calculated_dose: Decimal
    banded_dose: Decimal
    variance_percent: Decimal           # Acceptable variance from calculated dose
    
    def is_dose_in_band(self, calculated_dose: Decimal) -> bool:
        """Check if calculated dose falls within this band"""
        return self.min_calculated_dose <= calculated_dose <= self.max_calculated_dose
    
    def get_dose_variance(self, calculated_dose: Decimal) -> Decimal:
        """Calculate variance percentage from calculated dose"""
        if calculated_dose == 0:
            return Decimal('0')
        
        variance = abs(self.banded_dose - calculated_dose) / calculated_dose * 100
        return variance


@dataclass
class AvailableStrength:
    """Available medication strength"""
    strength: Decimal
    unit: str
    dosage_form: str                    # tablet, capsule, vial, etc.
    splittable: bool = False            # Can be split/crushed
    volume_ml: Optional[Decimal] = None # For liquid formulations
    
    def can_achieve_dose(self, target_dose: Decimal, max_units: int = 4) -> bool:
        """Check if target dose can be achieved with this strength"""
        if self.splittable:
            # Can use half tablets
            min_achievable = self.strength / 2
            max_achievable = self.strength * max_units
        else:
            min_achievable = self.strength
            max_achievable = self.strength * max_units
        
        return min_achievable <= target_dose <= max_achievable


class DoseBandingService:
    """
    Service for dose banding and practical dosing considerations
    
    Implements:
    - Chemotherapy dose banding for preparation efficiency
    - Tablet splitting feasibility
    - Vial size optimization
    - IV rate calculations
    - Practical rounding strategies
    """
    
    def __init__(self):
        self.dose_bands = self._load_dose_bands()
        self.available_strengths = self._load_available_strengths()
        self.rounding_rules = self._load_rounding_rules()
    
    def apply_dose_banding(
        self,
        dose: DoseSpecification,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any],
        banding_type: DoseBandingType = DoseBandingType.STANDARD
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Apply dose banding to calculated dose
        
        Returns:
            - Banded dose specification
            - List of warnings
            - List of clinical notes
        """
        warnings = []
        clinical_notes = []
        
        medication_id = medication_properties.get('medication_id')
        
        # Get dose bands for this medication and type
        bands = self.dose_bands.get(f"{medication_id}_{banding_type.value}", [])
        
        if not bands:
            clinical_notes.append("No dose banding available for this medication")
            return dose, warnings, clinical_notes
        
        # Find appropriate dose band
        selected_band = None
        for band in bands:
            if band.is_dose_in_band(dose.value):
                selected_band = band
                break
        
        if not selected_band:
            clinical_notes.append("Calculated dose outside available dose bands")
            return dose, warnings, clinical_notes
        
        # Check if banding variance is acceptable
        variance = selected_band.get_dose_variance(dose.value)
        if variance > selected_band.variance_percent:
            warnings.append(f"Dose banding variance ({variance:.1f}%) exceeds acceptable limit")
            clinical_notes.append("Consider individual dose calculation")
        
        # Create banded dose specification
        adjustment_factors = dict(dose.calculation_factors)
        adjustment_factors.update({
            'dose_band_id': selected_band.band_id,
            'calculated_dose': float(dose.value),
            'banded_dose': float(selected_band.banded_dose),
            'banding_variance_percent': float(variance),
            'banding_type': banding_type.value
        })
        
        banded_dose = DoseSpecification(
            value=selected_band.banded_dose,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_dose_banded",
            calculation_factors=adjustment_factors
        )
        
        # Generate clinical notes
        if variance > 5:  # Note significant variances
            if selected_band.banded_dose > dose.value:
                clinical_notes.append(f"Dose increased by {variance:.1f}% for banding efficiency")
            else:
                clinical_notes.append(f"Dose reduced by {variance:.1f}% for banding efficiency")
        
        clinical_notes.append(f"Using dose band: {selected_band.band_id}")
        
        # Add banding-specific notes
        if banding_type == DoseBandingType.CHEMOTHERAPY:
            clinical_notes.append("Dose banding applied for preparation efficiency and safety")
        elif banding_type == DoseBandingType.PEDIATRIC:
            clinical_notes.append("Pediatric dose banding applied for accurate measurement")
        
        logger.info(f"Dose banding applied: {dose.value} -> {selected_band.banded_dose} {dose.unit.value}")
        
        return banded_dose, warnings, clinical_notes
    
    def optimize_tablet_dosing(
        self,
        dose: DoseSpecification,
        medication_id: str
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Optimize dose for available tablet strengths
        
        Considers tablet splitting, multiple tablets, and available strengths
        """
        warnings = []
        clinical_notes = []
        
        # Get available strengths for this medication
        strengths = self.available_strengths.get(medication_id, [])
        tablet_strengths = [s for s in strengths if s.dosage_form in ['tablet', 'capsule']]
        
        if not tablet_strengths:
            clinical_notes.append("No tablet formulations available")
            return dose, warnings, clinical_notes
        
        # Find optimal combination of tablets
        best_combination = self._find_optimal_tablet_combination(dose.value, tablet_strengths)
        
        if not best_combination:
            warnings.append("Cannot achieve exact dose with available tablet strengths")
            clinical_notes.append("Consider liquid formulation or dose adjustment")
            return dose, warnings, clinical_notes
        
        # Calculate optimized dose
        optimized_dose_value = best_combination['total_dose']
        tablet_instructions = best_combination['instructions']
        
        # Create optimized dose specification
        adjustment_factors = dict(dose.calculation_factors)
        adjustment_factors.update({
            'tablet_optimization': True,
            'calculated_dose': float(dose.value),
            'optimized_dose': float(optimized_dose_value),
            'tablet_instructions': tablet_instructions
        })
        
        optimized_dose = DoseSpecification(
            value=optimized_dose_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_tablet_optimized",
            calculation_factors=adjustment_factors
        )
        
        # Generate clinical notes
        variance = abs(optimized_dose_value - dose.value) / dose.value * 100
        if variance > 5:
            clinical_notes.append(f"Dose adjusted by {variance:.1f}% for available tablet strengths")
        
        clinical_notes.append(f"Tablet instructions: {tablet_instructions}")
        
        # Add splitting warnings if needed
        if any('half' in instruction for instruction in tablet_instructions):
            clinical_notes.append("Tablet splitting required - ensure tablets are scored")
            warnings.append("Verify tablet can be safely split")
        
        return optimized_dose, warnings, clinical_notes
    
    def calculate_iv_infusion_rate(
        self,
        dose: DoseSpecification,
        infusion_time_hours: Decimal,
        concentration_mg_ml: Decimal,
        patient_weight_kg: Optional[Decimal] = None
    ) -> Dict[str, Any]:
        """
        Calculate IV infusion rate and volume
        
        Returns infusion parameters for IV administration
        """
        # Calculate total volume needed
        total_volume_ml = dose.value / concentration_mg_ml
        
        # Calculate infusion rate
        infusion_rate_ml_hr = total_volume_ml / infusion_time_hours
        
        # Calculate weight-based rate if weight provided
        weight_based_rate = None
        if patient_weight_kg:
            weight_based_rate = infusion_rate_ml_hr / patient_weight_kg
        
        # Round to practical values
        infusion_rate_ml_hr = self._round_infusion_rate(infusion_rate_ml_hr)
        total_volume_ml = self._round_volume(total_volume_ml)
        
        return {
            'total_dose_mg': float(dose.value),
            'total_volume_ml': float(total_volume_ml),
            'concentration_mg_ml': float(concentration_mg_ml),
            'infusion_time_hours': float(infusion_time_hours),
            'infusion_rate_ml_hr': float(infusion_rate_ml_hr),
            'weight_based_rate_ml_kg_hr': float(weight_based_rate) if weight_based_rate else None,
            'instructions': f"Infuse {total_volume_ml}mL over {infusion_time_hours} hours at {infusion_rate_ml_hr}mL/hr"
        }
    
    def optimize_vial_usage(
        self,
        dose: DoseSpecification,
        medication_id: str
    ) -> Tuple[DoseSpecification, List[str], Dict[str, Any]]:
        """
        Optimize dose to minimize vial waste
        
        Considers available vial sizes and minimizes waste
        """
        clinical_notes = []
        
        # Get available vial sizes
        strengths = self.available_strengths.get(medication_id, [])
        vial_strengths = [s for s in strengths if s.dosage_form == 'vial']
        
        if not vial_strengths:
            clinical_notes.append("No vial formulations available")
            return dose, clinical_notes, {}
        
        # Find optimal vial combination
        optimal_combination = self._find_optimal_vial_combination(dose.value, vial_strengths)
        
        if not optimal_combination:
            clinical_notes.append("Cannot optimize vial usage")
            return dose, clinical_notes, {}
        
        # Calculate optimized dose (may be slightly different to minimize waste)
        optimized_dose_value = optimal_combination['achievable_dose']
        vial_usage = optimal_combination['vial_usage']
        waste_amount = optimal_combination['waste_amount']
        cost_efficiency = optimal_combination['cost_efficiency']
        
        # Create optimized dose specification
        adjustment_factors = dict(dose.calculation_factors)
        adjustment_factors.update({
            'vial_optimization': True,
            'calculated_dose': float(dose.value),
            'optimized_dose': float(optimized_dose_value),
            'waste_amount': float(waste_amount),
            'cost_efficiency': cost_efficiency
        })
        
        optimized_dose = DoseSpecification(
            value=optimized_dose_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_vial_optimized",
            calculation_factors=adjustment_factors
        )
        
        # Generate clinical notes
        if waste_amount > 0:
            waste_percent = (waste_amount / optimized_dose_value) * 100
            clinical_notes.append(f"Vial waste: {waste_amount}mg ({waste_percent:.1f}%)")
        else:
            clinical_notes.append("No vial waste - optimal usage achieved")
        
        return optimized_dose, clinical_notes, vial_usage
    
    # === PRIVATE HELPER METHODS ===
    
    def _find_optimal_tablet_combination(
        self,
        target_dose: Decimal,
        tablet_strengths: List[AvailableStrength]
    ) -> Optional[Dict[str, Any]]:
        """Find optimal combination of tablets to achieve target dose"""
        
        best_combination = None
        min_variance = Decimal('100')  # Start with 100% variance
        
        # Try different combinations
        for strength in tablet_strengths:
            # Try whole tablets only
            for num_tablets in range(1, 5):  # Up to 4 tablets
                dose_achieved = strength.strength * num_tablets
                variance = abs(dose_achieved - target_dose) / target_dose * 100
                
                if variance < min_variance:
                    min_variance = variance
                    best_combination = {
                        'total_dose': dose_achieved,
                        'instructions': [f"{num_tablets} x {strength.strength}mg {strength.dosage_form}"],
                        'variance_percent': variance
                    }
            
            # Try with half tablets if splittable
            if strength.splittable:
                for num_whole in range(0, 4):
                    for num_half in [0, 1]:  # 0 or 1 half tablet
                        dose_achieved = (strength.strength * num_whole) + (strength.strength * num_half / 2)
                        variance = abs(dose_achieved - target_dose) / target_dose * 100
                        
                        if variance < min_variance:
                            min_variance = variance
                            instructions = []
                            if num_whole > 0:
                                instructions.append(f"{num_whole} x {strength.strength}mg {strength.dosage_form}")
                            if num_half > 0:
                                instructions.append(f"{num_half} x {strength.strength/2}mg {strength.dosage_form} (half)")
                            
                            best_combination = {
                                'total_dose': dose_achieved,
                                'instructions': instructions,
                                'variance_percent': variance
                            }
        
        # Only return if variance is acceptable (< 10%)
        if best_combination and best_combination['variance_percent'] < 10:
            return best_combination
        
        return None
    
    def _find_optimal_vial_combination(
        self,
        target_dose: Decimal,
        vial_strengths: List[AvailableStrength]
    ) -> Optional[Dict[str, Any]]:
        """Find optimal vial combination to minimize waste"""
        
        best_combination = None
        min_waste = Decimal('999999')  # Start with high waste
        
        # Try different vial combinations
        for strength in vial_strengths:
            for num_vials in range(1, 4):  # Up to 3 vials
                total_available = strength.strength * num_vials
                
                if total_available >= target_dose:
                    waste = total_available - target_dose
                    
                    if waste < min_waste:
                        min_waste = waste
                        best_combination = {
                            'achievable_dose': target_dose,
                            'waste_amount': waste,
                            'vial_usage': {
                                'strength_mg': float(strength.strength),
                                'num_vials': num_vials,
                                'total_available_mg': float(total_available)
                            },
                            'cost_efficiency': float((target_dose / total_available) * 100)
                        }
        
        return best_combination
    
    def _round_infusion_rate(self, rate_ml_hr: Decimal) -> Decimal:
        """Round infusion rate to practical pump settings"""
        
        if rate_ml_hr < 1:
            # Round to nearest 0.1 mL/hr for very slow rates
            return (rate_ml_hr * 10).quantize(Decimal('1')) / 10
        elif rate_ml_hr < 10:
            # Round to nearest 0.5 mL/hr
            return (rate_ml_hr * 2).quantize(Decimal('1')) / 2
        else:
            # Round to nearest whole number
            return rate_ml_hr.quantize(Decimal('1'))
    
    def _round_volume(self, volume_ml: Decimal) -> Decimal:
        """Round volume to practical measurement"""
        
        if volume_ml < 10:
            # Round to nearest 0.1 mL for small volumes
            return (volume_ml * 10).quantize(Decimal('1')) / 10
        else:
            # Round to nearest whole mL
            return volume_ml.quantize(Decimal('1'))
    
    def _load_dose_bands(self) -> Dict[str, List[DoseBand]]:
        """Load dose banding definitions"""
        
        bands = {}
        
        # Chemotherapy dose bands for doxorubicin
        bands['doxorubicin_chemotherapy'] = [
            DoseBand('dox_band_1', Decimal('45'), Decimal('55'), Decimal('50'), Decimal('10')),
            DoseBand('dox_band_2', Decimal('55'), Decimal('65'), Decimal('60'), Decimal('10')),
            DoseBand('dox_band_3', Decimal('65'), Decimal('75'), Decimal('70'), Decimal('10')),
            DoseBand('dox_band_4', Decimal('75'), Decimal('85'), Decimal('80'), Decimal('10')),
            DoseBand('dox_band_5', Decimal('85'), Decimal('95'), Decimal('90'), Decimal('10'))
        ]
        
        # Pediatric dose bands for acetaminophen
        bands['acetaminophen_pediatric'] = [
            DoseBand('acet_ped_1', Decimal('80'), Decimal('120'), Decimal('100'), Decimal('20')),
            DoseBand('acet_ped_2', Decimal('120'), Decimal('180'), Decimal('150'), Decimal('20')),
            DoseBand('acet_ped_3', Decimal('180'), Decimal('220'), Decimal('200'), Decimal('20')),
            DoseBand('acet_ped_4', Decimal('220'), Decimal('280'), Decimal('250'), Decimal('20'))
        ]
        
        return bands
    
    def _load_available_strengths(self) -> Dict[str, List[AvailableStrength]]:
        """Load available medication strengths"""
        
        strengths = {}
        
        # Acetaminophen tablets
        strengths['acetaminophen'] = [
            AvailableStrength(Decimal('325'), 'mg', 'tablet', splittable=True),
            AvailableStrength(Decimal('500'), 'mg', 'tablet', splittable=True),
            AvailableStrength(Decimal('650'), 'mg', 'tablet', splittable=False)
        ]
        
        # Doxorubicin vials
        strengths['doxorubicin'] = [
            AvailableStrength(Decimal('10'), 'mg', 'vial', volume_ml=Decimal('5')),
            AvailableStrength(Decimal('50'), 'mg', 'vial', volume_ml=Decimal('25')),
            AvailableStrength(Decimal('200'), 'mg', 'vial', volume_ml=Decimal('100'))
        ]
        
        return strengths
    
    def _load_rounding_rules(self) -> Dict[str, Dict[str, Any]]:
        """Load medication-specific rounding rules"""
        
        rules = {
            'insulin': {
                'strategy': RoundingStrategy.NEAREST_UNIT,
                'precision': Decimal('1')
            },
            'chemotherapy': {
                'strategy': RoundingStrategy.NEAREST_UNIT,
                'precision': Decimal('1'),
                'max_variance_percent': Decimal('10')
            },
            'pediatric_liquid': {
                'strategy': RoundingStrategy.NEAREST_QUARTER,
                'precision': Decimal('0.25')
            }
        }
        
        return rules
