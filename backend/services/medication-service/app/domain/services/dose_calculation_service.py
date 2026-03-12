"""
Dose Calculation Service - Core Pharmaceutical Intelligence
Implements sophisticated dose calculation strategies with clinical algorithms
"""

import logging
from typing import Dict, Any, Optional, List
from decimal import Decimal, ROUND_HALF_UP
from abc import ABC, abstractmethod
from enum import Enum
import math

from ..value_objects.dose_specification import (
    DoseSpecification, DoseCalculationContext, DoseUnit, RouteOfAdministration
)
from ..value_objects.clinical_properties import DosingType, DosingGuidelines

logger = logging.getLogger(__name__)


class CalculationStrategy(ABC):
    """Abstract base class for dose calculation strategies"""
    
    @abstractmethod
    def calculate(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate dose using this strategy"""
        pass
    
    @abstractmethod
    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate that context has required data for this strategy"""
        pass


class WeightBasedCalculator(CalculationStrategy):
    """
    Weight-based dose calculation strategy
    Used for most medications with mg/kg dosing
    """
    
    def calculate(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate weight-based dose"""
        
        if not context.weight_kg:
            raise ValueError("Weight required for weight-based dosing")
        
        # Get base dose per kg
        dose_mg_kg = guidelines.weight_based_dose_mg_kg
        
        # Apply age-specific dosing if available
        if context.is_pediatric():
            pediatric_dose = guidelines.get_pediatric_dose_mg_kg()
            if pediatric_dose:
                dose_mg_kg = pediatric_dose
                logger.info(f"Applied pediatric dose: {dose_mg_kg} mg/kg")
        
        # Calculate total dose
        calculated_dose = dose_mg_kg * context.weight_kg
        
        # Apply weight-based adjustments for extreme weights
        if context.weight_kg > Decimal('100'):  # Obesity adjustment
            obesity_factor = self._calculate_obesity_adjustment(context.weight_kg)
            calculated_dose = calculated_dose * obesity_factor
            logger.info(f"Applied obesity adjustment factor: {obesity_factor}")
        
        # Select appropriate route and unit
        route = self._determine_route(medication_properties)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="weight_based",
            calculation_factors={
                "weight_kg": float(context.weight_kg),
                "dose_mg_kg": float(dose_mg_kg),
                "total_dose_mg": float(calculated_dose),
                "patient_type": "pediatric" if context.is_pediatric() else "adult",
                "obesity_adjusted": context.weight_kg > Decimal('100')
            }
        )
    
    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate context for weight-based calculation"""
        errors = []
        
        if not context.weight_kg:
            errors.append("Weight (kg) is required for weight-based dosing")
        elif context.weight_kg <= 0:
            errors.append("Weight must be positive")
        elif context.weight_kg > Decimal('300'):
            errors.append("Weight exceeds maximum safe limit (300 kg)")
        
        return errors
    
    def _calculate_obesity_adjustment(self, weight_kg: Decimal) -> Decimal:
        """Calculate dose adjustment for obese patients"""
        # Simplified obesity adjustment - can be made more sophisticated
        if weight_kg > Decimal('150'):
            return Decimal('0.8')  # Reduce dose by 20% for very obese patients
        elif weight_kg > Decimal('120'):
            return Decimal('0.9')  # Reduce dose by 10% for obese patients
        else:
            return Decimal('1.0')  # No adjustment
    
    def _determine_route(self, medication_properties: Dict[str, Any]) -> RouteOfAdministration:
        """Determine appropriate route based on medication properties"""
        # Simplified route determination
        primary_route = medication_properties.get('primary_route', 'PO')
        try:
            return RouteOfAdministration(primary_route)
        except ValueError:
            return RouteOfAdministration.ORAL  # Default fallback


class BSABasedCalculator(CalculationStrategy):
    """
    Body Surface Area (BSA) based dose calculation
    Primarily used for chemotherapy and other high-risk medications
    """
    
    def calculate(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate BSA-based dose"""
        
        if not context.bsa_m2:
            # Calculate BSA if height and weight available
            if context.height_cm and context.weight_kg:
                context = self._calculate_bsa(context)
            else:
                raise ValueError("BSA or height/weight required for BSA-based dosing")
        
        # Get base dose per m²
        dose_mg_m2 = guidelines.bsa_based_dose_mg_m2
        
        # Calculate total dose
        calculated_dose = dose_mg_m2 * context.bsa_m2
        
        # Apply BSA capping for safety (common in chemotherapy)
        max_bsa = Decimal('2.0')  # Standard BSA cap
        if context.bsa_m2 > max_bsa:
            capped_dose = dose_mg_m2 * max_bsa
            logger.warning(f"BSA capped at {max_bsa} m² (actual: {context.bsa_m2} m²)")
            calculated_dose = capped_dose
        
        route = self._determine_route(medication_properties)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="bsa_based",
            calculation_factors={
                "bsa_m2": float(context.bsa_m2),
                "dose_mg_m2": float(dose_mg_m2),
                "total_dose_mg": float(calculated_dose),
                "height_cm": float(context.height_cm) if context.height_cm else None,
                "weight_kg": float(context.weight_kg) if context.weight_kg else None,
                "bsa_capped": context.bsa_m2 > max_bsa
            }
        )
    
    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate context for BSA-based calculation"""
        errors = []
        
        if not context.bsa_m2 and not (context.height_cm and context.weight_kg):
            errors.append("BSA or height/weight required for BSA-based dosing")
        
        if context.height_cm and (context.height_cm < 50 or context.height_cm > 250):
            errors.append("Height must be between 50-250 cm")
        
        if context.weight_kg and (context.weight_kg <= 0 or context.weight_kg > 300):
            errors.append("Weight must be between 0-300 kg")
        
        return errors
    
    def _calculate_bsa(self, context: DoseCalculationContext) -> DoseCalculationContext:
        """Calculate BSA using Mosteller formula"""
        if not context.height_cm or not context.weight_kg:
            return context
        
        # Mosteller formula: BSA (m²) = √[(height_cm × weight_kg) / 3600]
        bsa = Decimal(str(math.sqrt(float(context.height_cm * context.weight_kg) / 3600)))
        bsa = bsa.quantize(Decimal('0.01'), rounding=ROUND_HALF_UP)
        
        # Create new context with calculated BSA
        return DoseCalculationContext(
            patient_id=context.patient_id,
            weight_kg=context.weight_kg,
            height_cm=context.height_cm,
            age_years=context.age_years,
            age_months=context.age_months,
            bsa_m2=bsa,
            creatinine_clearance=context.creatinine_clearance,
            egfr=context.egfr,
            liver_function=context.liver_function,
            pregnancy_status=context.pregnancy_status,
            breastfeeding_status=context.breastfeeding_status
        )
    
    def _determine_route(self, medication_properties: Dict[str, Any]) -> RouteOfAdministration:
        """Determine route - BSA dosing typically IV"""
        primary_route = medication_properties.get('primary_route', 'IV')
        try:
            return RouteOfAdministration(primary_route)
        except ValueError:
            return RouteOfAdministration.INTRAVENOUS


class AUCBasedCalculator(CalculationStrategy):
    """
    Area Under the Curve (AUC) based dose calculation
    Used for medications requiring precise pharmacokinetic targeting
    """
    
    def calculate(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate AUC-based dose using Calvert formula or similar"""
        
        # This is typically used for carboplatin and similar drugs
        # Calvert formula: Dose = Target AUC × (GFR + 25)
        
        target_auc = medication_properties.get('target_auc', Decimal('5'))  # Default AUC
        
        # Use eGFR or creatinine clearance
        gfr = context.egfr or context.creatinine_clearance
        if not gfr:
            raise ValueError("GFR or creatinine clearance required for AUC-based dosing")
        
        # Calvert formula
        calculated_dose = target_auc * (gfr + Decimal('25'))
        
        route = RouteOfAdministration.INTRAVENOUS  # AUC dosing typically IV
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="auc_based",
            calculation_factors={
                "target_auc": float(target_auc),
                "gfr": float(gfr),
                "calculated_dose_mg": float(calculated_dose),
                "formula": "Calvert: Dose = AUC × (GFR + 25)"
            }
        )
    
    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate context for AUC-based calculation"""
        errors = []
        
        if not context.egfr and not context.creatinine_clearance:
            errors.append("eGFR or creatinine clearance required for AUC-based dosing")
        
        gfr = context.egfr or context.creatinine_clearance
        if gfr and gfr < Decimal('10'):
            errors.append("GFR too low for standard AUC-based dosing")
        
        return errors


class FixedDoseCalculator(CalculationStrategy):
    """
    Fixed dose calculation strategy
    Used for medications with standard fixed doses
    """
    
    def calculate(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate fixed dose"""
        
        dose_range = guidelines.standard_dose_range
        if not dose_range:
            raise ValueError("Fixed dosing requires standard dose range")
        
        # Select appropriate dose based on patient factors
        if context.is_geriatric():
            # Use lower end of range for elderly
            calculated_dose = dose_range.get('min', dose_range.get('standard'))
        elif context.is_pediatric():
            # Pediatric fixed dosing should use weight-based instead
            raise ValueError("Fixed dosing not appropriate for pediatric patients")
        else:
            # Use standard dose for adults
            calculated_dose = dose_range.get('standard', dose_range.get('min'))
        
        route = self._determine_route(medication_properties)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="fixed",
            calculation_factors={
                "standard_dose": float(calculated_dose),
                "dose_range": {k: float(v) for k, v in dose_range.items()},
                "patient_category": self._categorize_patient(context)
            }
        )
    
    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate context for fixed dose calculation"""
        errors = []
        
        if context.is_pediatric():
            errors.append("Fixed dosing not recommended for pediatric patients")
        
        return errors
    
    def _determine_route(self, medication_properties: Dict[str, Any]) -> RouteOfAdministration:
        """Determine appropriate route"""
        primary_route = medication_properties.get('primary_route', 'PO')
        try:
            return RouteOfAdministration(primary_route)
        except ValueError:
            return RouteOfAdministration.ORAL
    
    def _categorize_patient(self, context: DoseCalculationContext) -> str:
        """Categorize patient for dosing"""
        if context.is_pediatric():
            return "pediatric"
        elif context.is_geriatric():
            return "geriatric"
        else:
            return "adult"


class TieredDoseCalculator(CalculationStrategy):
    """
    Tiered dose calculation strategy
    Uses patient characteristics to select from predefined dose tiers
    """
    
    def calculate(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate tiered dose based on patient characteristics"""
        
        base_dose = guidelines.standard_dose_range.get('min')
        if not base_dose:
            raise ValueError("Tiered dosing requires base dose")
        
        # Determine tier based on patient factors
        tier_multiplier = self._determine_tier_multiplier(context, guidelines)
        calculated_dose = base_dose * tier_multiplier
        
        route = self._determine_route(medication_properties)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="tiered",
            calculation_factors={
                "base_dose": float(base_dose),
                "tier_multiplier": float(tier_multiplier),
                "calculated_dose": float(calculated_dose),
                "patient_tier": self._determine_patient_tier(context)
            }
        )
    
    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate context for tiered calculation"""
        errors = []
        
        if not context.age_years:
            errors.append("Age required for tiered dosing")
        
        return errors
    
    def _determine_tier_multiplier(
        self, 
        context: DoseCalculationContext, 
        guidelines: DosingGuidelines
    ) -> Decimal:
        """Determine dose multiplier based on patient tier"""
        
        if context.is_pediatric():
            if context.weight_kg:
                # Weight-based adjustment for pediatrics
                return min(Decimal('1.0'), context.weight_kg / Decimal('70'))
            else:
                return Decimal('0.5')  # Conservative pediatric dose
        
        elif context.is_geriatric():
            # Geriatric dose reduction
            geriatric_factor = guidelines.get_geriatric_dose_adjustment()
            return geriatric_factor
        
        elif context.has_renal_impairment():
            # Renal impairment adjustment
            return Decimal('0.75')
        
        elif context.has_hepatic_impairment():
            # Hepatic impairment adjustment
            return Decimal('0.5')
        
        else:
            # Standard adult dose
            return Decimal('1.0')
    
    def _determine_patient_tier(self, context: DoseCalculationContext) -> str:
        """Determine patient tier for logging"""
        if context.is_pediatric():
            return "pediatric"
        elif context.is_geriatric():
            return "geriatric"
        elif context.has_renal_impairment():
            return "renal_impaired"
        elif context.has_hepatic_impairment():
            return "hepatic_impaired"
        else:
            return "standard_adult"
    
    def _determine_route(self, medication_properties: Dict[str, Any]) -> RouteOfAdministration:
        """Determine appropriate route"""
        primary_route = medication_properties.get('primary_route', 'PO')
        try:
            return RouteOfAdministration(primary_route)
        except ValueError:
            return RouteOfAdministration.ORAL


class LoadingDoseCalculator(CalculationStrategy):
    """
    Loading dose calculation strategy
    Calculates initial higher doses to rapidly achieve therapeutic levels
    """

    def calculate(
        self,
        context: DoseCalculationContext,
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """Calculate loading dose"""

        # Get maintenance dose first
        maintenance_dose = self._get_maintenance_dose(context, guidelines)

        # Calculate loading dose multiplier based on half-life and target
        loading_multiplier = self._calculate_loading_multiplier(
            medication_properties, guidelines
        )

        calculated_dose = maintenance_dose * loading_multiplier

        # Apply safety caps for loading doses
        max_loading_dose = guidelines.max_single_dose or (maintenance_dose * Decimal('5'))
        calculated_dose = min(calculated_dose, max_loading_dose)

        route = self._determine_route(medication_properties)

        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="loading_dose",
            calculation_factors={
                "maintenance_dose": float(maintenance_dose),
                "loading_multiplier": float(loading_multiplier),
                "calculated_loading_dose": float(calculated_dose),
                "max_loading_dose": float(max_loading_dose),
                "half_life_hours": medication_properties.get('half_life_hours', 'unknown')
            }
        )

    def validate_context(self, context: DoseCalculationContext) -> List[str]:
        """Validate context for loading dose calculation"""
        errors = []

        if not context.weight_kg:
            errors.append("Weight required for loading dose calculation")

        return errors

    def _get_maintenance_dose(
        self,
        context: DoseCalculationContext,
        guidelines: DosingGuidelines
    ) -> Decimal:
        """Calculate maintenance dose as baseline"""
        if guidelines.weight_based_dose_mg_kg and context.weight_kg:
            return context.weight_kg * guidelines.weight_based_dose_mg_kg
        elif guidelines.standard_dose_range:
            return guidelines.standard_dose_range.get('standard', guidelines.standard_dose_range.get('min'))
        else:
            raise ValueError("Cannot determine maintenance dose for loading calculation")

    def _calculate_loading_multiplier(
        self,
        medication_properties: Dict[str, Any],
        guidelines: DosingGuidelines
    ) -> Decimal:
        """Calculate loading dose multiplier based on pharmacokinetics"""

        # If explicit loading dose multiplier provided
        if hasattr(guidelines, 'loading_dose_multiplier') and guidelines.loading_dose_multiplier:
            return guidelines.loading_dose_multiplier

        # Calculate based on half-life (longer half-life = higher loading dose)
        half_life_hours = medication_properties.get('half_life_hours')
        if half_life_hours:
            if half_life_hours >= 24:  # Long half-life drugs
                return Decimal('3.0')
            elif half_life_hours >= 12:  # Medium half-life drugs
                return Decimal('2.5')
            elif half_life_hours >= 6:   # Short-medium half-life drugs
                return Decimal('2.0')
            else:  # Short half-life drugs
                return Decimal('1.5')

        # Default conservative loading dose
        return Decimal('2.0')

    def _determine_route(self, medication_properties: Dict[str, Any]) -> RouteOfAdministration:
        """Determine appropriate route - loading doses often IV"""
        primary_route = medication_properties.get('primary_route', 'IV')
        try:
            return RouteOfAdministration(primary_route)
        except ValueError:
            return RouteOfAdministration.INTRAVENOUS


class DoseCalculationService:
    """
    Core Dose Calculation Service
    
    Implements the strategy pattern for different calculation methods
    This is the pharmaceutical intelligence engine of the service
    """
    
    def __init__(self):
        self.strategies = {
            DosingType.WEIGHT_BASED: WeightBasedCalculator(),
            DosingType.BSA_BASED: BSABasedCalculator(),
            DosingType.AUC_BASED: AUCBasedCalculator(),
            DosingType.FIXED: FixedDoseCalculator(),
            DosingType.TIERED: TieredDoseCalculator(),
            DosingType.LOADING_DOSE: LoadingDoseCalculator()
        }
    
    def calculate_dose(
        self,
        dosing_type: DosingType,
        context: DoseCalculationContext,
        guidelines: DosingGuidelines,
        medication_properties: Dict[str, Any]
    ) -> DoseSpecification:
        """
        Calculate dose using appropriate strategy
        
        This is the main entry point for dose calculations
        """
        logger.info(f"Calculating dose using {dosing_type.value} strategy for patient {context.patient_id}")
        
        # Get appropriate strategy
        strategy = self.strategies.get(dosing_type)
        if not strategy:
            raise ValueError(f"Unsupported dosing type: {dosing_type}")
        
        # Validate context
        validation_errors = strategy.validate_context(context)
        if validation_errors:
            raise ValueError(f"Context validation failed: {validation_errors}")
        
        # Calculate dose
        try:
            dose_spec = strategy.calculate(context, guidelines, medication_properties)
            logger.info(f"Dose calculated: {dose_spec.to_display_string()}")
            return dose_spec
            
        except Exception as e:
            logger.error(f"Dose calculation failed: {str(e)}")
            raise
    
    def get_supported_dosing_types(self) -> List[DosingType]:
        """Get list of supported dosing types"""
        return list(self.strategies.keys())
    
    def validate_calculation_context(
        self, 
        dosing_type: DosingType, 
        context: DoseCalculationContext
    ) -> List[str]:
        """Validate context for specific dosing type"""
        strategy = self.strategies.get(dosing_type)
        if not strategy:
            return [f"Unsupported dosing type: {dosing_type}"]
        
        return strategy.validate_context(context)
