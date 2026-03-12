"""
Medication Entity - Core Domain Entity
Represents the pharmaceutical intelligence and business rules for medications
"""

from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from uuid import UUID, uuid4
from datetime import datetime
from decimal import Decimal

from ..value_objects.clinical_properties import (
    ClinicalProperties, MedicationIdentifiers, FormulationProperties,
    DosingType, TherapeuticClass, PharmacologicClass
)
from ..value_objects.dose_specification import (
    DoseSpecification, DoseCalculationContext, DoseProposal,
    Frequency, Duration, Quantity, DoseUnit, RouteOfAdministration
)
from ..services.dose_calculation_service import DoseCalculationService
from ..services.renal_dose_adjustment_service import RenalDoseAdjustmentService
from ..services.hepatic_dose_adjustment_service import HepaticDoseAdjustmentService


@dataclass
class Medication:
    """
    Core Medication Entity - The Domain Expert for Pharmaceutical Intelligence
    
    This entity embodies the "Clinical Pharmacist's Digital Twin" philosophy,
    containing all pharmaceutical knowledge and business rules for a medication.
    
    Pillar 1: Pure Domain Expert - Contains ONLY pharmaceutical intelligence,
    NO safety validation (delegated to Safety Gateway)
    """
    
    # Identity
    medication_id: UUID = field(default_factory=uuid4)
    identifiers: MedicationIdentifiers = field(default=None)
    
    # Clinical Intelligence
    clinical_properties: ClinicalProperties = field(default=None)
    
    # Operational Data
    formulary_status: Optional[str] = None
    average_cost: Optional[Decimal] = None
    availability_status: str = "available"
    
    # Metadata
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
    version: int = 1

    # Services (injected)
    _dose_calculation_service: DoseCalculationService = field(default_factory=DoseCalculationService, init=False)
    _renal_adjustment_service: RenalDoseAdjustmentService = field(default_factory=RenalDoseAdjustmentService, init=False)
    _hepatic_adjustment_service: HepaticDoseAdjustmentService = field(default_factory=HepaticDoseAdjustmentService, init=False)

    # Advanced Services (100% Pharmaceutical Intelligence)
    _pharmacogenomics_service: Optional[Any] = field(default=None, init=False)  # PharmacogenomicsService
    _tdm_service: Optional[Any] = field(default=None, init=False)               # TherapeuticDrugMonitoringService
    _advanced_pk_service: Optional[Any] = field(default=None, init=False)       # AdvancedPharmacokineticsService
    _dose_banding_service: Optional[Any] = field(default=None, init=False)      # DoseBandingService
    _special_populations_service: Optional[Any] = field(default=None, init=False) # SpecialPopulationsService

    def __post_init__(self):
        """Validate medication entity"""
        if not self.identifiers:
            raise ValueError("Medication must have identifiers")
        if not self.clinical_properties:
            raise ValueError("Medication must have clinical properties")
    
    # === PHARMACEUTICAL INTELLIGENCE METHODS ===
    
    def calculate_dose(
        self,
        context: DoseCalculationContext,
        indication: str,
        frequency: Frequency,
        duration: Duration
    ) -> DoseProposal:
        """
        Calculate appropriate dose based on patient context and clinical guidelines

        This is the core pharmaceutical intelligence - the "how" of medication therapy
        NO safety validation performed here (that's Safety Gateway's responsibility)

        Enhanced with sophisticated calculation services for clinical accuracy
        """
        dosing_guidelines = self.clinical_properties.dosing_guidelines
        warnings = []
        clinical_notes = []

        # Step 1: Core dose calculation using sophisticated calculation service
        medication_properties = self._get_medication_properties_dict()

        calculated_dose = self._dose_calculation_service.calculate_dose(
            dosing_type=dosing_guidelines.dosing_type,
            context=context,
            guidelines=dosing_guidelines,
            medication_properties=medication_properties
        )

        clinical_notes.append(f"Base dose calculated using {calculated_dose.calculation_method}")

        # Step 2: Apply renal adjustments if needed
        if dosing_guidelines.renal_adjustment_required or self._hepatic_adjustment_service.requires_renal_adjustment(context, medication_properties):
            adjusted_dose, renal_warnings, renal_notes = self._renal_adjustment_service.calculate_renal_adjustment(
                calculated_dose, context, medication_properties
            )
            calculated_dose = adjusted_dose
            warnings.extend(renal_warnings)
            clinical_notes.extend(renal_notes)

        # Step 3: Apply hepatic adjustments if needed
        if dosing_guidelines.hepatic_adjustment_required or self._hepatic_adjustment_service.requires_hepatic_adjustment(context, medication_properties):
            adjusted_dose, hepatic_warnings, hepatic_notes = self._hepatic_adjustment_service.calculate_hepatic_adjustment(
                calculated_dose, context, medication_properties
            )
            calculated_dose = adjusted_dose
            warnings.extend(hepatic_warnings)
            clinical_notes.extend(hepatic_notes)

        # Step 4: Apply pharmacogenomic adjustments if available
        if self._pharmacogenomics_service and hasattr(context, 'pgx_results') and context.pgx_results:
            adjusted_dose, pgx_warnings, pgx_notes = self._pharmacogenomics_service.calculate_pgx_adjustment(
                calculated_dose, context, medication_properties, context.pgx_results
            )
            calculated_dose = adjusted_dose
            warnings.extend(pgx_warnings)
            clinical_notes.extend(pgx_notes)

        # Step 5: Apply TDM adjustments if recent levels available
        if self._tdm_service and hasattr(context, 'recent_drug_levels') and context.recent_drug_levels:
            adjusted_dose, tdm_warnings, tdm_notes = self._tdm_service.calculate_tdm_dose_adjustment(
                calculated_dose, context, medication_properties, context.recent_drug_levels
            )
            calculated_dose = adjusted_dose
            warnings.extend(tdm_warnings)
            clinical_notes.extend(tdm_notes)

        # Step 6: Apply special population adjustments
        if self._special_populations_service:
            # Pregnancy adjustments
            if context.pregnancy_status and context.trimester:
                adjusted_dose, preg_warnings, preg_notes = self._special_populations_service.calculate_pregnancy_adjustment(
                    calculated_dose, context, medication_properties, context.trimester
                )
                calculated_dose = adjusted_dose
                warnings.extend(preg_warnings)
                clinical_notes.extend(preg_notes)

            # Lactation adjustments
            if context.breastfeeding_status:
                adjusted_dose, lact_warnings, lact_notes = self._special_populations_service.calculate_lactation_adjustment(
                    calculated_dose, context, medication_properties
                )
                calculated_dose = adjusted_dose
                warnings.extend(lact_warnings)
                clinical_notes.extend(lact_notes)

        # Step 7: Apply advanced PK optimization if available
        if self._advanced_pk_service and hasattr(context, 'target_auc') and context.target_auc:
            adjusted_dose, pk_warnings, pk_notes = self._advanced_pk_service.calculate_pk_guided_dose(
                calculated_dose, context, medication_properties, target_auc=context.target_auc
            )
            calculated_dose = adjusted_dose
            warnings.extend(pk_warnings)
            clinical_notes.extend(pk_notes)

        # Step 8: Apply dose banding if appropriate
        if self._dose_banding_service and self._should_apply_dose_banding(medication_properties):
            from ..services.dose_banding_service import DoseBandingType
            banding_type = self._determine_banding_type(medication_properties)
            adjusted_dose, band_warnings, band_notes = self._dose_banding_service.apply_dose_banding(
                calculated_dose, context, medication_properties, banding_type
            )
            calculated_dose = adjusted_dose
            warnings.extend(band_warnings)
            clinical_notes.extend(band_notes)

        # Step 9: Apply final dose limits and rounding
        final_dose = self._apply_dose_limits_and_rounding(calculated_dose, dosing_guidelines)

        # Step 10: Calculate quantity
        quantity = self._calculate_quantity(final_dose, frequency, duration)

        # Step 6: Assess confidence and generate additional warnings
        confidence_score, additional_warnings, additional_notes = self._assess_dose_confidence(
            final_dose, context, dosing_guidelines
        )
        warnings.extend(additional_warnings)
        clinical_notes.extend(additional_notes)

        # Step 7: Add monitoring recommendations
        monitoring_recommendations = self._get_monitoring_recommendations(context, medication_properties)
        clinical_notes.extend(monitoring_recommendations)

        return DoseProposal(
            medication_id=str(self.medication_id),
            patient_id=context.patient_id,
            calculated_dose=final_dose,
            frequency=frequency,
            duration=duration,
            quantity=quantity,
            indication=indication,
            calculation_context=context,
            calculation_timestamp=datetime.utcnow().isoformat(),
            confidence_score=confidence_score,
            warnings=warnings,
            clinical_notes=clinical_notes
        )

    # === ENHANCED HELPER METHODS ===

    def _get_medication_properties_dict(self) -> Dict[str, Any]:
        """Convert medication properties to dictionary for calculation services"""
        primary_formulation = self.clinical_properties.get_primary_formulation()

        return {
            'pharmacologic_class': self.clinical_properties.pharmacologic_class.value,
            'therapeutic_class': self.clinical_properties.therapeutic_class.value,
            'primary_route': primary_formulation.route_of_administration if primary_formulation else 'PO',
            'is_high_alert': self.clinical_properties.safety_profile.is_high_alert,
            'is_controlled_substance': self.clinical_properties.safety_profile.is_controlled_substance,
            'is_hepatotoxic': bool(self.clinical_properties.safety_profile.black_box_warning),
            'hepatic_metabolism_percent': 70,  # Default - should be in clinical properties
            'renal_elimination_percent': 30,   # Default - should be in clinical properties
            'target_auc': Decimal('5'),        # Default for AUC-based dosing
            'formulations': [
                {
                    'dosage_form': f.dosage_form,
                    'strength': f.strength,
                    'route': f.route_of_administration
                } for f in self.clinical_properties.formulations
            ]
        }

    def _get_monitoring_recommendations(
        self,
        context: DoseCalculationContext,
        medication_properties: Dict[str, Any]
    ) -> List[str]:
        """Get comprehensive monitoring recommendations"""
        recommendations = []

        # Renal monitoring
        renal_recs = self._renal_adjustment_service.get_monitoring_recommendations(
            context, medication_properties
        )
        recommendations.extend(renal_recs)

        # Hepatic monitoring
        hepatic_recs = self._hepatic_adjustment_service.get_monitoring_recommendations(
            context, medication_properties
        )
        recommendations.extend(hepatic_recs)

        # Medication-specific monitoring
        if self.clinical_properties.dosing_guidelines.therapeutic_drug_monitoring:
            recommendations.append("Therapeutic drug level monitoring required")

        if self.clinical_properties.safety_profile.is_high_alert:
            recommendations.append("High-alert medication - enhanced monitoring required")

        if self.clinical_properties.safety_profile.monitoring_requirements:
            recommendations.extend(self.clinical_properties.safety_profile.monitoring_requirements)

        # Remove duplicates while preserving order
        seen = set()
        unique_recommendations = []
        for rec in recommendations:
            if rec not in seen:
                seen.add(rec)
                unique_recommendations.append(rec)

        return unique_recommendations

    # === LEGACY METHODS (Kept for compatibility) ===

    def _calculate_weight_based_dose(
        self, 
        context: DoseCalculationContext, 
        guidelines: Any
    ) -> DoseSpecification:
        """Calculate weight-based dose"""
        if not context.weight_kg:
            raise ValueError("Weight required for weight-based dosing")
        
        dose_mg_kg = guidelines.weight_based_dose_mg_kg
        if context.is_pediatric():
            pediatric_dose = guidelines.get_pediatric_dose_mg_kg()
            if pediatric_dose:
                dose_mg_kg = pediatric_dose
        
        calculated_dose = dose_mg_kg * context.weight_kg
        
        # Select appropriate route and unit
        primary_formulation = self.clinical_properties.get_primary_formulation()
        route = RouteOfAdministration(primary_formulation.route_of_administration)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="weight_based",
            calculation_factors={
                "weight_kg": float(context.weight_kg),
                "dose_mg_kg": float(dose_mg_kg),
                "patient_type": "pediatric" if context.is_pediatric() else "adult"
            }
        )
    
    def _calculate_bsa_based_dose(
        self, 
        context: DoseCalculationContext, 
        guidelines: Any
    ) -> DoseSpecification:
        """Calculate BSA-based dose (typically for chemotherapy)"""
        if not context.bsa_m2:
            raise ValueError("BSA required for BSA-based dosing")
        
        dose_mg_m2 = guidelines.bsa_based_dose_mg_m2
        calculated_dose = dose_mg_m2 * context.bsa_m2
        
        primary_formulation = self.clinical_properties.get_primary_formulation()
        route = RouteOfAdministration(primary_formulation.route_of_administration)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="bsa_based",
            calculation_factors={
                "bsa_m2": float(context.bsa_m2),
                "dose_mg_m2": float(dose_mg_m2),
                "height_cm": float(context.height_cm) if context.height_cm else None,
                "weight_kg": float(context.weight_kg) if context.weight_kg else None
            }
        )
    
    def _calculate_fixed_dose(
        self, 
        context: DoseCalculationContext, 
        guidelines: Any
    ) -> DoseSpecification:
        """Calculate fixed dose"""
        dose_range = guidelines.standard_dose_range
        if not dose_range:
            raise ValueError("Fixed dosing requires standard dose range")
        
        # Use minimum of range as starting dose, can be adjusted based on patient factors
        calculated_dose = dose_range.get('min', dose_range.get('standard'))
        
        primary_formulation = self.clinical_properties.get_primary_formulation()
        route = RouteOfAdministration(primary_formulation.route_of_administration)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="fixed",
            calculation_factors={
                "standard_dose": float(calculated_dose),
                "dose_range": {k: float(v) for k, v in dose_range.items()}
            }
        )
    
    def _calculate_tiered_dose(
        self, 
        context: DoseCalculationContext, 
        guidelines: Any
    ) -> DoseSpecification:
        """Calculate tiered dose based on patient factors"""
        # Simplified tiered dosing - can be expanded based on specific criteria
        base_dose = guidelines.standard_dose_range.get('min')
        
        # Adjust based on age
        if context.is_geriatric():
            adjustment_factor = guidelines.get_geriatric_dose_adjustment()
            calculated_dose = base_dose * adjustment_factor
        elif context.is_pediatric() and context.weight_kg:
            # Use weight-based calculation for pediatrics
            pediatric_dose_mg_kg = guidelines.get_pediatric_dose_mg_kg()
            if pediatric_dose_mg_kg:
                calculated_dose = pediatric_dose_mg_kg * context.weight_kg
            else:
                calculated_dose = base_dose * Decimal('0.5')  # Conservative pediatric dose
        else:
            calculated_dose = base_dose
        
        primary_formulation = self.clinical_properties.get_primary_formulation()
        route = RouteOfAdministration(primary_formulation.route_of_administration)
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=route,
            calculation_method="tiered",
            calculation_factors={
                "base_dose": float(base_dose),
                "adjustment_factor": float(calculated_dose / base_dose),
                "patient_category": self._categorize_patient(context)
            }
        )
    
    def _apply_age_adjustments(
        self, 
        dose: DoseSpecification, 
        context: DoseCalculationContext,
        guidelines: Any
    ) -> DoseSpecification:
        """Apply age-specific dose adjustments"""
        if context.is_geriatric():
            adjustment_factor = guidelines.get_geriatric_dose_adjustment()
            if adjustment_factor != Decimal('1.0'):
                adjusted_value = dose.value * adjustment_factor
                return DoseSpecification(
                    value=adjusted_value,
                    unit=dose.unit,
                    route=dose.route,
                    calculation_method=f"{dose.calculation_method}_geriatric_adjusted",
                    calculation_factors={
                        **dose.calculation_factors,
                        "geriatric_adjustment": float(adjustment_factor)
                    }
                )
        
        return dose
    
    def _apply_organ_function_adjustments(
        self, 
        dose: DoseSpecification, 
        context: DoseCalculationContext,
        guidelines: Any
    ) -> DoseSpecification:
        """Apply renal and hepatic function adjustments"""
        adjusted_dose = dose
        adjustment_factors = {}
        
        # Renal adjustment
        if guidelines.renal_adjustment_required and context.has_renal_impairment():
            renal_factor = self._calculate_renal_adjustment_factor(context)
            adjusted_dose = DoseSpecification(
                value=adjusted_dose.value * renal_factor,
                unit=adjusted_dose.unit,
                route=adjusted_dose.route,
                calculation_method=f"{adjusted_dose.calculation_method}_renal_adjusted",
                calculation_factors={**adjusted_dose.calculation_factors}
            )
            adjustment_factors['renal_adjustment'] = float(renal_factor)
        
        # Hepatic adjustment
        if guidelines.hepatic_adjustment_required and context.has_hepatic_impairment():
            hepatic_factor = self._calculate_hepatic_adjustment_factor(context)
            adjusted_dose = DoseSpecification(
                value=adjusted_dose.value * hepatic_factor,
                unit=adjusted_dose.unit,
                route=adjusted_dose.route,
                calculation_method=f"{adjusted_dose.calculation_method}_hepatic_adjusted",
                calculation_factors={**adjusted_dose.calculation_factors}
            )
            adjustment_factors['hepatic_adjustment'] = float(hepatic_factor)
        
        if adjustment_factors:
            adjusted_dose = DoseSpecification(
                value=adjusted_dose.value,
                unit=adjusted_dose.unit,
                route=adjusted_dose.route,
                calculation_method=adjusted_dose.calculation_method,
                calculation_factors={
                    **adjusted_dose.calculation_factors,
                    **adjustment_factors
                }
            )
        
        return adjusted_dose
    
    def _calculate_renal_adjustment_factor(self, context: DoseCalculationContext) -> Decimal:
        """Calculate renal dose adjustment factor"""
        if context.creatinine_clearance:
            crcl = context.creatinine_clearance
        elif context.egfr:
            crcl = context.egfr
        else:
            return Decimal('0.5')  # Conservative default
        
        # Simplified renal adjustment - can be made more sophisticated
        if crcl >= 60:
            return Decimal('1.0')
        elif crcl >= 30:
            return Decimal('0.75')
        elif crcl >= 15:
            return Decimal('0.5')
        else:
            return Decimal('0.25')
    
    def _calculate_hepatic_adjustment_factor(self, context: DoseCalculationContext) -> Decimal:
        """Calculate hepatic dose adjustment factor"""
        # Simplified hepatic adjustment based on Child-Pugh classification
        if context.liver_function == 'mild':
            return Decimal('0.75')
        elif context.liver_function == 'moderate':
            return Decimal('0.5')
        elif context.liver_function == 'severe':
            return Decimal('0.25')
        else:
            return Decimal('0.5')  # Conservative default
    
    def _apply_dose_limits_and_rounding(
        self, 
        dose: DoseSpecification, 
        guidelines: Any
    ) -> DoseSpecification:
        """Apply maximum dose limits and rounding rules"""
        final_value = dose.value
        
        # Apply maximum dose limits
        if guidelines.max_single_dose and final_value > guidelines.max_single_dose:
            final_value = guidelines.max_single_dose
        
        # Apply rounding rules (round to nearest 25mg for most oral medications)
        if dose.route in [RouteOfAdministration.ORAL] and final_value > 25:
            final_value = (final_value / 25).quantize(Decimal('1')) * 25
        else:
            # Round to 1 decimal place for other routes
            final_value = final_value.quantize(Decimal('0.1'))
        
        return DoseSpecification(
            value=final_value,
            unit=dose.unit,
            route=dose.route,
            calculation_method=f"{dose.calculation_method}_rounded",
            calculation_factors={
                **dose.calculation_factors,
                "pre_rounding_dose": float(dose.value),
                "rounding_applied": float(dose.value) != float(final_value)
            }
        )
    
    def _calculate_quantity(
        self, 
        dose: DoseSpecification, 
        frequency: Frequency, 
        duration: Duration
    ) -> Quantity:
        """Calculate dispensing quantity"""
        # Simplified quantity calculation
        total_days = duration.to_total_days() or 30  # Default 30 days
        
        if frequency.times_per_day:
            daily_doses = frequency.times_per_day
        elif frequency.interval_hours:
            daily_doses = 24 / frequency.interval_hours
        else:
            daily_doses = 1
        
        total_quantity = dose.value * Decimal(str(daily_doses)) * Decimal(str(total_days))
        
        return Quantity(
            amount=total_quantity,
            unit="tablets",  # Simplified - should be based on formulation
            days_supply=total_days
        )
    
    def _assess_dose_confidence(
        self, 
        dose: DoseSpecification, 
        context: DoseCalculationContext,
        guidelines: Any
    ) -> tuple[Decimal, List[str], List[str]]:
        """Assess confidence in dose calculation and generate warnings"""
        confidence = Decimal('1.0')
        warnings = []
        clinical_notes = []
        
        # Reduce confidence for missing data
        if not context.weight_kg and guidelines.dosing_type == DosingType.WEIGHT_BASED:
            confidence -= Decimal('0.3')
            warnings.append("Weight-based dosing without current weight")
        
        if context.has_renal_impairment() and not guidelines.renal_adjustment_required:
            confidence -= Decimal('0.2')
            warnings.append("Patient has renal impairment but medication may not require adjustment")
        
        # Add clinical notes
        if context.is_pediatric():
            clinical_notes.append("Pediatric dosing applied")
        
        if context.is_geriatric():
            clinical_notes.append("Geriatric dose adjustment considered")
        
        return max(confidence, Decimal('0.1')), warnings, clinical_notes
    
    def _categorize_patient(self, context: DoseCalculationContext) -> str:
        """Categorize patient for dosing purposes"""
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
    
    # === FORMULARY AND OPERATIONAL METHODS ===
    
    def is_formulary_preferred(self) -> bool:
        """Check if medication is formulary preferred"""
        return self.formulary_status in ['preferred', 'tier_1']
    
    def requires_prior_authorization(self) -> bool:
        """Check if medication requires prior authorization"""
        return self.formulary_status in ['non_preferred', 'tier_3', 'prior_auth_required']
    
    def is_available(self) -> bool:
        """Check if medication is currently available"""
        return self.availability_status == 'available'
    
    def get_display_name(self) -> str:
        """Get display name for medication"""
        return self.identifiers.get_display_name()
    
    def get_therapeutic_class(self) -> str:
        """Get therapeutic class"""
        return self.clinical_properties.therapeutic_class.value
    
    def is_high_risk(self) -> bool:
        """Check if medication is high risk"""
        return self.clinical_properties.is_high_risk_medication()
    
    def __str__(self) -> str:
        return f"Medication({self.get_display_name()})"
    
    def __repr__(self) -> str:
        return f"Medication(id={self.medication_id}, name={self.get_display_name()})"

    def _should_apply_dose_banding(self, medication_properties: Dict[str, Any]) -> bool:
        """Determine if dose banding should be applied"""
        # Apply dose banding for chemotherapy drugs
        if medication_properties.get('therapeutic_class') == 'antineoplastic':
            return True

        # Apply dose banding for high-alert medications
        if medication_properties.get('is_high_alert'):
            return True

        # Apply dose banding for pediatric patients
        # This would be determined from context in real implementation
        return False

    def _determine_banding_type(self, medication_properties: Dict[str, Any]):
        """Determine appropriate dose banding type"""
        from ..services.dose_banding_service import DoseBandingType

        if medication_properties.get('therapeutic_class') == 'antineoplastic':
            return DoseBandingType.CHEMOTHERAPY

        return DoseBandingType.STANDARD

    def inject_advanced_services(
        self,
        pharmacogenomics_service=None,
        tdm_service=None,
        advanced_pk_service=None,
        dose_banding_service=None,
        special_populations_service=None
    ):
        """Inject advanced pharmaceutical intelligence services"""
        if pharmacogenomics_service:
            self._pharmacogenomics_service = pharmacogenomics_service
        if tdm_service:
            self._tdm_service = tdm_service
        if advanced_pk_service:
            self._advanced_pk_service = advanced_pk_service
        if dose_banding_service:
            self._dose_banding_service = dose_banding_service
        if special_populations_service:
            self._special_populations_service = special_populations_service
