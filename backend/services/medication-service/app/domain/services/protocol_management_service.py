"""
Protocol Management Service
Handles complex medication protocols and cumulative dose tracking
"""

import logging
from typing import Dict, Any, Optional, List, Tuple
from decimal import Decimal
from datetime import date, datetime
from dataclasses import dataclass
from enum import Enum

from ..value_objects.dose_specification import DoseSpecification, DoseCalculationContext
from ..value_objects.clinical_properties import DosingType

logger = logging.getLogger(__name__)


class ProtocolType(Enum):
    """Types of medication protocols"""
    CHEMOTHERAPY = "chemotherapy"
    ANTIBIOTIC = "antibiotic"
    CHRONIC_DISEASE = "chronic_disease"
    IMMUNOSUPPRESSION = "immunosuppression"
    PAIN_MANAGEMENT = "pain_management"


class DoseModificationReason(Enum):
    """Reasons for protocol dose modifications"""
    TOXICITY = "toxicity"
    ORGAN_DYSFUNCTION = "organ_dysfunction"
    DRUG_INTERACTION = "drug_interaction"
    PATIENT_TOLERANCE = "patient_tolerance"
    EFFICACY = "efficacy"


@dataclass
class CumulativeDoseLimit:
    """Cumulative dose limits for medications"""
    medication_id: str
    lifetime_limit_mg: Optional[Decimal] = None
    annual_limit_mg: Optional[Decimal] = None
    cycle_limit_mg: Optional[Decimal] = None
    monitoring_threshold_mg: Optional[Decimal] = None
    warning_threshold_percent: Decimal = Decimal('80')  # Warn at 80% of limit
    
    def is_approaching_limit(self, current_cumulative: Decimal) -> bool:
        """Check if approaching any cumulative limit"""
        if self.lifetime_limit_mg:
            threshold = self.lifetime_limit_mg * (self.warning_threshold_percent / Decimal('100'))
            if current_cumulative >= threshold:
                return True
        
        return False
    
    def is_limit_exceeded(self, current_cumulative: Decimal) -> bool:
        """Check if any cumulative limit is exceeded"""
        if self.lifetime_limit_mg and current_cumulative > self.lifetime_limit_mg:
            return True
        
        return False


@dataclass
class ProtocolDoseModification:
    """Protocol-specific dose modification"""
    medication_id: str
    cycle_number: int
    modification_percent: Decimal  # 0.75 = 25% reduction
    reason: DoseModificationReason
    applied_date: date
    notes: Optional[str] = None
    
    def apply_to_dose(self, original_dose: Decimal) -> Decimal:
        """Apply modification to dose"""
        return original_dose * self.modification_percent


@dataclass
class ProtocolCycle:
    """Individual cycle within a protocol"""
    cycle_number: int
    cycle_length_days: int
    medications: List[Dict[str, Any]]  # Medication schedule for this cycle
    dose_modifications: Optional[List[ProtocolDoseModification]] = None
    
    def get_modified_dose(self, medication_id: str, original_dose: Decimal) -> Decimal:
        """Get dose with any cycle-specific modifications applied"""
        if not self.dose_modifications:
            return original_dose
        
        modified_dose = original_dose
        for modification in self.dose_modifications:
            if modification.medication_id == medication_id:
                modified_dose = modification.apply_to_dose(modified_dose)
        
        return modified_dose


class ProtocolManagementService:
    """
    Service for managing complex medication protocols
    
    Handles:
    - Cumulative dose tracking (especially for cardiotoxic drugs)
    - Protocol cycle management
    - Dose modification rules
    - Safety monitoring for protocol medications
    """
    
    def __init__(self, protocol_repository, cumulative_dose_repository):
        self.protocol_repository = protocol_repository
        self.cumulative_dose_repository = cumulative_dose_repository
        
        # Load cumulative dose limits for high-risk medications
        self.cumulative_limits = self._load_cumulative_dose_limits()
    
    async def calculate_protocol_dose(
        self,
        protocol_id: str,
        patient_id: str,
        medication_id: str,
        cycle_number: int,
        context: DoseCalculationContext
    ) -> Tuple[DoseSpecification, List[str], List[str]]:
        """
        Calculate dose for medication within a protocol
        
        Returns:
            - Calculated dose with protocol modifications
            - List of warnings
            - List of clinical notes
        """
        try:
            warnings = []
            clinical_notes = []
            
            # Get protocol definition
            protocol = await self.protocol_repository.get_by_id(protocol_id)
            if not protocol:
                raise ValueError(f"Protocol {protocol_id} not found")
            
            # Get base dose calculation (using standard dose calculation service)
            base_dose = await self._calculate_base_protocol_dose(
                protocol, medication_id, context
            )
            
            # Apply protocol-specific modifications
            modified_dose = await self._apply_protocol_modifications(
                base_dose, protocol, medication_id, cycle_number, patient_id
            )
            
            # Check cumulative dose limits
            cumulative_warnings, cumulative_notes = await self._check_cumulative_limits(
                medication_id, patient_id, modified_dose.value
            )
            warnings.extend(cumulative_warnings)
            clinical_notes.extend(cumulative_notes)
            
            # Add protocol-specific monitoring
            protocol_monitoring = self._get_protocol_monitoring_requirements(
                protocol, medication_id, cycle_number
            )
            clinical_notes.extend(protocol_monitoring)
            
            logger.info(f"Protocol dose calculated: {modified_dose.to_display_string()}")
            
            return modified_dose, warnings, clinical_notes
            
        except Exception as e:
            logger.error(f"Error calculating protocol dose: {str(e)}")
            raise
    
    async def track_cumulative_dose(
        self,
        patient_id: str,
        medication_id: str,
        administered_dose: Decimal,
        administration_date: date
    ) -> Dict[str, Any]:
        """
        Track cumulative dose administration
        
        Returns cumulative dose information and warnings
        """
        try:
            # Record the dose
            await self.cumulative_dose_repository.record_dose(
                patient_id, medication_id, administered_dose, administration_date
            )
            
            # Get updated cumulative totals
            lifetime_total = await self.cumulative_dose_repository.get_lifetime_total(
                patient_id, medication_id
            )
            
            annual_total = await self.cumulative_dose_repository.get_annual_total(
                patient_id, medication_id, administration_date.year
            )
            
            # Check against limits
            warnings = []
            if medication_id in self.cumulative_limits:
                limit = self.cumulative_limits[medication_id]
                
                if limit.is_limit_exceeded(lifetime_total):
                    warnings.append(f"Lifetime cumulative dose limit exceeded: {lifetime_total}mg")
                elif limit.is_approaching_limit(lifetime_total):
                    warnings.append(f"Approaching lifetime cumulative dose limit: {lifetime_total}mg")
            
            return {
                'lifetime_total_mg': float(lifetime_total),
                'annual_total_mg': float(annual_total),
                'warnings': warnings,
                'monitoring_required': len(warnings) > 0
            }
            
        except Exception as e:
            logger.error(f"Error tracking cumulative dose: {str(e)}")
            raise
    
    async def get_protocol_status(
        self,
        protocol_id: str,
        patient_id: str
    ) -> Dict[str, Any]:
        """Get current status of patient's protocol"""
        try:
            enrollment = await self.protocol_repository.get_patient_enrollment(
                protocol_id, patient_id
            )
            
            if not enrollment:
                return {'status': 'not_enrolled'}
            
            protocol = await self.protocol_repository.get_by_id(protocol_id)
            
            # Calculate progress
            progress_percent = (enrollment.current_cycle / protocol.total_cycles) * 100
            
            # Get dose modifications
            modifications = await self.protocol_repository.get_dose_modifications(
                protocol_id, patient_id
            )
            
            return {
                'status': enrollment.status,
                'current_cycle': enrollment.current_cycle,
                'total_cycles': protocol.total_cycles,
                'progress_percent': progress_percent,
                'dose_modifications': len(modifications),
                'next_cycle_date': enrollment.next_cycle_date,
                'completion_date': enrollment.estimated_completion_date
            }
            
        except Exception as e:
            logger.error(f"Error getting protocol status: {str(e)}")
            return {'status': 'error', 'error': str(e)}
    
    # === PRIVATE HELPER METHODS ===
    
    async def _calculate_base_protocol_dose(
        self,
        protocol: Any,
        medication_id: str,
        context: DoseCalculationContext
    ) -> DoseSpecification:
        """Calculate base dose for protocol medication"""
        
        # Get medication from protocol definition
        protocol_medication = None
        for med in protocol.medications:
            if med['medication_id'] == medication_id:
                protocol_medication = med
                break
        
        if not protocol_medication:
            raise ValueError(f"Medication {medication_id} not found in protocol")
        
        # Calculate based on protocol dosing method
        if 'dose_mg_m2' in protocol_medication and context.bsa_m2:
            calculated_dose = Decimal(str(protocol_medication['dose_mg_m2'])) * context.bsa_m2
        elif 'dose_mg_kg' in protocol_medication and context.weight_kg:
            calculated_dose = Decimal(str(protocol_medication['dose_mg_kg'])) * context.weight_kg
        elif 'fixed_dose_mg' in protocol_medication:
            calculated_dose = Decimal(str(protocol_medication['fixed_dose_mg']))
        else:
            raise ValueError("Cannot determine protocol dose calculation method")
        
        from ..value_objects.dose_specification import DoseUnit, RouteOfAdministration
        
        return DoseSpecification(
            value=calculated_dose,
            unit=DoseUnit.MG,
            route=RouteOfAdministration(protocol_medication.get('route', 'IV')),
            calculation_method="protocol_based",
            calculation_factors={
                'protocol_id': protocol.protocol_id,
                'base_dose': float(calculated_dose),
                'calculation_method': 'protocol_defined'
            }
        )
    
    async def _apply_protocol_modifications(
        self,
        base_dose: DoseSpecification,
        protocol: Any,
        medication_id: str,
        cycle_number: int,
        patient_id: str
    ) -> DoseSpecification:
        """Apply protocol-specific dose modifications"""
        
        # Get any dose modifications for this patient/cycle
        modifications = await self.protocol_repository.get_dose_modifications(
            protocol.protocol_id, patient_id, cycle_number
        )
        
        modified_value = base_dose.value
        modification_factors = dict(base_dose.calculation_factors)
        
        for modification in modifications:
            if modification.medication_id == medication_id:
                modified_value = modification.apply_to_dose(modified_value)
                modification_factors[f'modification_{modification.reason.value}'] = float(modification.modification_percent)
        
        return DoseSpecification(
            value=modified_value,
            unit=base_dose.unit,
            route=base_dose.route,
            calculation_method=f"{base_dose.calculation_method}_modified",
            calculation_factors=modification_factors
        )
    
    async def _check_cumulative_limits(
        self,
        medication_id: str,
        patient_id: str,
        proposed_dose: Decimal
    ) -> Tuple[List[str], List[str]]:
        """Check cumulative dose limits"""
        warnings = []
        clinical_notes = []
        
        if medication_id not in self.cumulative_limits:
            return warnings, clinical_notes
        
        limit = self.cumulative_limits[medication_id]
        
        # Get current cumulative dose
        current_cumulative = await self.cumulative_dose_repository.get_lifetime_total(
            patient_id, medication_id
        )
        
        # Check what cumulative would be after this dose
        projected_cumulative = current_cumulative + proposed_dose
        
        if limit.is_limit_exceeded(projected_cumulative):
            warnings.append(f"Proposed dose would exceed lifetime cumulative limit")
            clinical_notes.append(f"Current cumulative: {current_cumulative}mg, Limit: {limit.lifetime_limit_mg}mg")
        elif limit.is_approaching_limit(projected_cumulative):
            warnings.append(f"Approaching lifetime cumulative dose limit")
            clinical_notes.append(f"Current cumulative: {current_cumulative}mg, Limit: {limit.lifetime_limit_mg}mg")
        
        return warnings, clinical_notes
    
    def _get_protocol_monitoring_requirements(
        self,
        protocol: Any,
        medication_id: str,
        cycle_number: int
    ) -> List[str]:
        """Get protocol-specific monitoring requirements"""
        monitoring = []
        
        if protocol.protocol_type == ProtocolType.CHEMOTHERAPY:
            monitoring.extend([
                "Monitor CBC with differential before each cycle",
                "Assess performance status",
                "Monitor for signs of toxicity"
            ])
            
            # Cycle-specific monitoring
            if cycle_number == 1:
                monitoring.append("Baseline cardiac function assessment")
            elif cycle_number % 3 == 0:  # Every 3rd cycle
                monitoring.append("Repeat cardiac function assessment")
        
        return monitoring
    
    def _load_cumulative_dose_limits(self) -> Dict[str, CumulativeDoseLimit]:
        """Load cumulative dose limits for high-risk medications"""
        
        # Anthracyclines (cardiotoxic)
        limits = {
            'doxorubicin': CumulativeDoseLimit(
                medication_id='doxorubicin',
                lifetime_limit_mg=Decimal('450'),  # 450 mg/m² lifetime limit
                monitoring_threshold_mg=Decimal('300'),
                warning_threshold_percent=Decimal('80')
            ),
            'daunorubicin': CumulativeDoseLimit(
                medication_id='daunorubicin',
                lifetime_limit_mg=Decimal('600'),  # 600 mg/m² lifetime limit
                monitoring_threshold_mg=Decimal('400'),
                warning_threshold_percent=Decimal('80')
            ),
            'epirubicin': CumulativeDoseLimit(
                medication_id='epirubicin',
                lifetime_limit_mg=Decimal('900'),  # 900 mg/m² lifetime limit
                monitoring_threshold_mg=Decimal('600'),
                warning_threshold_percent=Decimal('80')
            )
        }
        
        return limits
