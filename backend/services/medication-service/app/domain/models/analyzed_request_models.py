"""
Analyzed Request Models - Data Models for Enhanced Orchestrator

This module defines the data models used by the Request Analyzer and other
Enhanced Orchestrator components for structured data representation.
"""

from typing import Dict, List, Any, Optional, Set, Union
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
import uuid


class RiskLevel(Enum):
    """Clinical risk stratification levels"""
    LOW = "low"
    MODERATE = "moderate"
    HIGH = "high"
    CRITICAL = "critical"


class AgeGroup(Enum):
    """Patient age group classifications for clinical decision making"""
    NEONATE = "neonate"      # 0-28 days
    INFANT = "infant"        # 29 days - 2 years
    CHILD = "child"          # 2-12 years
    ADOLESCENT = "adolescent" # 12-18 years
    ADULT = "adult"          # 18-65 years
    ELDERLY = "elderly"      # 65+ years


class OrganFunction(Enum):
    """Organ function status classifications"""
    NORMAL = "normal"
    MILD_IMPAIRMENT = "mild_impairment"
    MODERATE_IMPAIRMENT = "moderate_impairment"
    SEVERE_IMPAIRMENT = "severe_impairment"
    END_STAGE = "end_stage"


class UrgencyLevel(Enum):
    """Clinical urgency levels"""
    ROUTINE = "routine"
    URGENT = "urgent"
    EMERGENCY = "emergency"
    STAT = "stat"


@dataclass
class MedicationProperties:
    """
    Comprehensive medication properties extracted and derived by Request Analyzer
    
    Includes both direct properties (from request) and derived properties
    (calculated based on clinical knowledge and drug databases)
    """
    # Direct properties from request
    name: str
    rxnorm_code: Optional[str] = None
    ndc: Optional[str] = None
    therapeutic_class: Optional[str] = None
    pharmacologic_class: Optional[str] = None
    indication: Optional[str] = None
    
    # Derived clinical properties
    is_high_alert: bool = False
    is_narrow_therapeutic_index: bool = False
    is_vesicant: bool = False
    is_irritant: bool = False
    requires_therapeutic_monitoring: bool = False
    has_rems_program: bool = False
    has_black_box_warning: bool = False
    
    # Clinical classification flags
    black_box_warnings: List[str] = field(default_factory=list)
    contraindication_categories: List[str] = field(default_factory=list)
    special_population_considerations: List[str] = field(default_factory=list)
    
    # Risk assessment
    interaction_potential: RiskLevel = RiskLevel.LOW
    administration_complexity: RiskLevel = RiskLevel.LOW
    monitoring_intensity: RiskLevel = RiskLevel.LOW
    
    # Dosing considerations
    requires_renal_adjustment: bool = False
    requires_hepatic_adjustment: bool = False
    requires_weight_based_dosing: bool = False
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'name': self.name,
            'rxnorm_code': self.rxnorm_code,
            'therapeutic_class': self.therapeutic_class,
            'is_high_alert': self.is_high_alert,
            'is_narrow_therapeutic_index': self.is_narrow_therapeutic_index,
            'requires_therapeutic_monitoring': self.requires_therapeutic_monitoring,
            'interaction_potential': self.interaction_potential.value,
            'administration_complexity': self.administration_complexity.value,
            'special_considerations': self.special_population_considerations
        }


@dataclass
class PatientProperties:
    """
    Comprehensive patient properties for clinical decision making
    
    Includes demographics, clinical status, and treatment context
    """
    # Demographics
    patient_id: str
    age_years: Optional[int] = None
    age_group: Optional[AgeGroup] = None
    weight_kg: Optional[float] = None
    height_cm: Optional[float] = None
    bmi: Optional[float] = None
    bsa_m2: Optional[float] = None
    gender: Optional[str] = None
    
    # Special populations
    is_pregnant: bool = False
    is_lactating: bool = False
    pregnancy_trimester: Optional[int] = None
    
    # Clinical status
    active_conditions: List[str] = field(default_factory=list)
    condition_severities: Dict[str, str] = field(default_factory=dict)
    
    # Organ function
    renal_function: OrganFunction = OrganFunction.NORMAL
    hepatic_function: OrganFunction = OrganFunction.NORMAL
    cardiac_function: OrganFunction = OrganFunction.NORMAL
    
    # Laboratory values (recent)
    creatinine_mg_dl: Optional[float] = None
    egfr_ml_min: Optional[float] = None
    bilirubin_mg_dl: Optional[float] = None
    alt_u_l: Optional[float] = None
    
    # Allergy and sensitivity profile
    known_allergies: List[str] = field(default_factory=list)
    allergy_severities: Dict[str, str] = field(default_factory=dict)
    cross_sensitivity_risks: List[str] = field(default_factory=list)
    
    # Current treatment context
    current_medications: List[str] = field(default_factory=list)
    recent_procedures: List[str] = field(default_factory=list)
    care_setting: Optional[str] = None
    admission_date: Optional[datetime] = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'patient_id': self.patient_id,
            'age_years': self.age_years,
            'age_group': self.age_group.value if self.age_group else None,
            'weight_kg': self.weight_kg,
            'gender': self.gender,
            'active_conditions': self.active_conditions,
            'renal_function': self.renal_function.value,
            'hepatic_function': self.hepatic_function.value,
            'known_allergies': self.known_allergies,
            'current_medications': self.current_medications,
            'care_setting': self.care_setting
        }


@dataclass
class SituationalProperties:
    """
    Situational context properties for workflow and urgency assessment
    """
    # Urgency and timing
    urgency: UrgencyLevel = UrgencyLevel.ROUTINE
    stat_order: bool = False
    emergency_override: bool = False
    time_criticality_score: float = 0.0
    
    # Workflow context
    prescriber_id: Optional[str] = None
    prescriber_specialty: Optional[str] = None
    prescriber_experience_level: Optional[str] = None
    
    # Order context
    order_set_context: Optional[str] = None
    protocol_enrollment: Optional[str] = None
    clinical_trial_participation: bool = False
    
    # Care context
    encounter_type: Optional[str] = None  # inpatient, outpatient, emergency, surgery
    care_unit: Optional[str] = None  # ICU, ward, clinic
    consultation_requested: bool = False
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'urgency': self.urgency.value,
            'stat_order': self.stat_order,
            'emergency_override': self.emergency_override,
            'time_criticality_score': self.time_criticality_score,
            'prescriber_specialty': self.prescriber_specialty,
            'encounter_type': self.encounter_type,
            'care_unit': self.care_unit
        }


@dataclass
class EnrichedContext:
    """
    Enriched clinical context with inferred properties and intelligence
    
    This represents the "clinical intelligence" derived from analyzing
    the medication, patient, and situational properties together
    """
    # Overall risk assessment
    overall_risk_level: RiskLevel = RiskLevel.LOW
    complexity_score: float = 0.0
    safety_score: float = 1.0  # 0.0 = unsafe, 1.0 = safe
    
    # Clinical requirements
    monitoring_requirements: List[str] = field(default_factory=list)
    baseline_requirements: List[str] = field(default_factory=list)
    special_considerations: List[str] = field(default_factory=list)
    
    # Clinical flags and alerts
    clinical_flags: Set[str] = field(default_factory=set)
    warning_flags: Set[str] = field(default_factory=set)
    
    # Inferred clinical context
    likely_indication: Optional[str] = None
    predicted_protocols: List[str] = field(default_factory=list)
    recommended_consultations: List[str] = field(default_factory=list)
    
    # Context recipe hints
    context_complexity_level: str = "standard"  # minimal, standard, comprehensive, intensive
    specialized_context_needs: List[str] = field(default_factory=list)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'overall_risk_level': self.overall_risk_level.value,
            'complexity_score': self.complexity_score,
            'safety_score': self.safety_score,
            'monitoring_requirements': self.monitoring_requirements,
            'clinical_flags': list(self.clinical_flags),
            'warning_flags': list(self.warning_flags),
            'likely_indication': self.likely_indication,
            'context_complexity_level': self.context_complexity_level,
            'specialized_context_needs': self.specialized_context_needs
        }


@dataclass
class AnalyzedRequest:
    """
    Complete analyzed request containing all extracted and derived properties
    
    This is the primary output of the Request Analyzer and input to the
    Context Selection Engine
    """
    # Unique identifier for this analysis
    analysis_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    analysis_timestamp: datetime = field(default_factory=datetime.utcnow)
    
    # Original request reference
    original_request: Any = None
    request_id: Optional[str] = None
    
    # Analyzed properties
    medication_properties: Optional[MedicationProperties] = None
    patient_properties: Optional[PatientProperties] = None
    situational_properties: Optional[SituationalProperties] = None
    enriched_context: Optional[EnrichedContext] = None
    
    # Analysis metadata
    requires_clinical_rules: bool = False
    requires_safety_validation: bool = False
    requires_specialist_review: bool = False
    
    # Performance tracking
    analysis_duration_ms: Optional[float] = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization and logging"""
        return {
            'analysis_id': self.analysis_id,
            'analysis_timestamp': self.analysis_timestamp.isoformat(),
            'request_id': self.request_id,
            'medication_properties': self.medication_properties.to_dict() if self.medication_properties else None,
            'patient_properties': self.patient_properties.to_dict() if self.patient_properties else None,
            'situational_properties': self.situational_properties.to_dict() if self.situational_properties else None,
            'enriched_context': self.enriched_context.to_dict() if self.enriched_context else None,
            'requires_clinical_rules': self.requires_clinical_rules,
            'requires_safety_validation': self.requires_safety_validation,
            'requires_specialist_review': self.requires_specialist_review,
            'analysis_duration_ms': self.analysis_duration_ms
        }
    
    def get_risk_summary(self) -> Dict[str, Any]:
        """Get a summary of risk factors for quick assessment"""
        if not self.enriched_context:
            return {'overall_risk': 'unknown', 'factors': []}
        
        risk_factors = []
        
        if self.medication_properties and self.medication_properties.is_high_alert:
            risk_factors.append('high_alert_medication')
        
        if self.patient_properties and self.patient_properties.age_group == AgeGroup.ELDERLY:
            risk_factors.append('elderly_patient')
        
        if self.situational_properties and self.situational_properties.urgency in [UrgencyLevel.EMERGENCY, UrgencyLevel.STAT]:
            risk_factors.append('urgent_situation')
        
        return {
            'overall_risk': self.enriched_context.overall_risk_level.value,
            'complexity_score': self.enriched_context.complexity_score,
            'safety_score': self.enriched_context.safety_score,
            'risk_factors': risk_factors,
            'clinical_flags': list(self.enriched_context.clinical_flags),
            'monitoring_required': len(self.enriched_context.monitoring_requirements) > 0
        }


# Type aliases for convenience
RequestAnalysisResult = AnalyzedRequest
ClinicalProperties = Dict[str, Any]
RiskAssessment = Dict[str, Union[str, float, List[str]]]
