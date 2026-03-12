"""
Clinical Properties Value Objects
Immutable objects representing clinical and pharmaceutical properties
"""

from dataclasses import dataclass
from typing import Optional, List, Dict, Any
from decimal import Decimal
from enum import Enum


class TherapeuticClass(Enum):
    """Major therapeutic classifications"""
    ANALGESIC = "analgesic"
    ANTIBIOTIC = "antibiotic"
    ANTINEOPLASTIC = "antineoplastic"
    CARDIOVASCULAR = "cardiovascular"
    ENDOCRINE = "endocrine"
    GASTROINTESTINAL = "gastrointestinal"
    NEUROLOGIC = "neurologic"
    PSYCHIATRIC = "psychiatric"
    RESPIRATORY = "respiratory"
    IMMUNOLOGIC = "immunologic"


class PharmacologicClass(Enum):
    """Specific pharmacologic mechanisms"""
    ACE_INHIBITOR = "ace_inhibitor"
    BETA_BLOCKER = "beta_blocker"
    CALCIUM_CHANNEL_BLOCKER = "calcium_channel_blocker"
    DIURETIC = "diuretic"
    NSAID = "nsaid"
    OPIOID = "opioid"
    ANTICOAGULANT = "anticoagulant"
    PROTON_PUMP_INHIBITOR = "proton_pump_inhibitor"
    STATIN = "statin"
    BENZODIAZEPINE = "benzodiazepine"


class DEASchedule(Enum):
    """DEA controlled substance schedules"""
    SCHEDULE_I = 1
    SCHEDULE_II = 2
    SCHEDULE_III = 3
    SCHEDULE_IV = 4
    SCHEDULE_V = 5


class DosingType(Enum):
    """Types of dosing calculations"""
    FIXED = "fixed"
    WEIGHT_BASED = "weight_based"
    BSA_BASED = "bsa_based"
    AUC_BASED = "auc_based"
    TIERED = "tiered"
    LOADING_DOSE = "loading_dose"
    MAINTENANCE_DOSE = "maintenance_dose"
    PROTOCOL_BASED = "protocol_based"


@dataclass(frozen=True)
class PharmacokineticProperties:
    """
    Pharmacokinetic properties of a medication
    """
    absorption_rate: Optional[Decimal] = None  # hours to peak
    bioavailability: Optional[Decimal] = None  # 0.0 to 1.0
    protein_binding: Optional[Decimal] = None  # percentage
    volume_of_distribution: Optional[Decimal] = None  # L/kg
    half_life_hours: Optional[Decimal] = None
    clearance_ml_min: Optional[Decimal] = None
    metabolism_pathway: Optional[str] = None  # hepatic, renal, other
    active_metabolites: Optional[List[str]] = None
    
    def __post_init__(self):
        """Validate pharmacokinetic properties"""
        if self.bioavailability is not None:
            if not (Decimal('0') <= self.bioavailability <= Decimal('1')):
                raise ValueError("Bioavailability must be between 0.0 and 1.0")
        
        if self.protein_binding is not None:
            if not (Decimal('0') <= self.protein_binding <= Decimal('100')):
                raise ValueError("Protein binding must be between 0 and 100 percent")


@dataclass(frozen=True)
class PharmacodynamicProperties:
    """
    Pharmacodynamic properties of a medication
    """
    mechanism_of_action: str
    target_receptors: Optional[List[str]] = None
    onset_of_action_hours: Optional[Decimal] = None
    duration_of_action_hours: Optional[Decimal] = None
    therapeutic_window: Optional[Dict[str, Decimal]] = None  # min/max levels
    dose_response_relationship: Optional[str] = None  # linear, logarithmic, etc.


@dataclass(frozen=True)
class SafetyProfile:
    """
    Safety and monitoring profile for a medication
    """
    is_high_alert: bool = False
    is_controlled_substance: bool = False
    dea_schedule: Optional[DEASchedule] = None
    black_box_warning: Optional[str] = None
    contraindications: Optional[List[str]] = None
    common_adverse_effects: Optional[List[str]] = None
    serious_adverse_effects: Optional[List[str]] = None
    drug_interactions: Optional[List[str]] = None
    monitoring_requirements: Optional[List[str]] = None
    pregnancy_category: Optional[str] = None
    lactation_safety: Optional[str] = None
    
    def __post_init__(self):
        """Validate safety profile"""
        if self.is_controlled_substance and not self.dea_schedule:
            raise ValueError("Controlled substances must have DEA schedule")


@dataclass(frozen=True)
class DosingGuidelines:
    """
    Dosing guidelines and calculation parameters
    """
    dosing_type: DosingType
    standard_dose_range: Optional[Dict[str, Decimal]] = None  # min/max
    weight_based_dose_mg_kg: Optional[Decimal] = None
    bsa_based_dose_mg_m2: Optional[Decimal] = None
    max_single_dose: Optional[Decimal] = None
    max_daily_dose: Optional[Decimal] = None
    renal_adjustment_required: bool = False
    hepatic_adjustment_required: bool = False
    age_specific_dosing: Optional[Dict[str, Any]] = None
    loading_dose_required: bool = False
    therapeutic_drug_monitoring: bool = False
    
    def get_pediatric_dose_mg_kg(self) -> Optional[Decimal]:
        """Get pediatric dose if available"""
        if self.age_specific_dosing and 'pediatric' in self.age_specific_dosing:
            return self.age_specific_dosing['pediatric'].get('dose_mg_kg')
        return self.weight_based_dose_mg_kg
    
    def get_geriatric_dose_adjustment(self) -> Optional[Decimal]:
        """Get geriatric dose adjustment factor"""
        if self.age_specific_dosing and 'geriatric' in self.age_specific_dosing:
            return self.age_specific_dosing['geriatric'].get('adjustment_factor', Decimal('1.0'))
        return Decimal('1.0')


@dataclass(frozen=True)
class FormulationProperties:
    """
    Properties of a specific medication formulation
    """
    dosage_form: str  # tablet, capsule, injection, etc.
    strength: Decimal
    strength_unit: str
    route_of_administration: str
    release_mechanism: Optional[str] = None  # immediate, extended, delayed
    excipients: Optional[List[str]] = None
    storage_requirements: Optional[str] = None
    stability_data: Optional[Dict[str, Any]] = None
    bioequivalence_data: Optional[Dict[str, Any]] = None
    
    def is_extended_release(self) -> bool:
        """Check if formulation is extended release"""
        return self.release_mechanism in ['extended', 'sustained', 'controlled']
    
    def requires_refrigeration(self) -> bool:
        """Check if formulation requires refrigeration"""
        return self.storage_requirements and 'refrigerat' in self.storage_requirements.lower()


@dataclass(frozen=True)
class ClinicalProperties:
    """
    Complete clinical properties of a medication
    Combines all clinical and pharmaceutical information
    """
    therapeutic_class: TherapeuticClass
    pharmacologic_class: PharmacologicClass
    pharmacokinetics: PharmacokineticProperties
    pharmacodynamics: PharmacodynamicProperties
    safety_profile: SafetyProfile
    dosing_guidelines: DosingGuidelines
    formulations: List[FormulationProperties]
    clinical_indications: List[str]
    off_label_uses: Optional[List[str]] = None
    
    def get_primary_formulation(self) -> Optional[FormulationProperties]:
        """Get the primary (first) formulation"""
        return self.formulations[0] if self.formulations else None
    
    def get_formulation_by_route(self, route: str) -> Optional[FormulationProperties]:
        """Get formulation by route of administration"""
        for formulation in self.formulations:
            if formulation.route_of_administration.lower() == route.lower():
                return formulation
        return None
    
    def is_high_risk_medication(self) -> bool:
        """Check if medication is high risk"""
        return (self.safety_profile.is_high_alert or 
                self.safety_profile.is_controlled_substance or
                self.safety_profile.black_box_warning is not None)
    
    def requires_special_monitoring(self) -> bool:
        """Check if medication requires special monitoring"""
        return (self.dosing_guidelines.therapeutic_drug_monitoring or
                bool(self.safety_profile.monitoring_requirements))
    
    def get_monitoring_requirements(self) -> List[str]:
        """Get all monitoring requirements"""
        requirements = []
        
        if self.safety_profile.monitoring_requirements:
            requirements.extend(self.safety_profile.monitoring_requirements)
        
        if self.dosing_guidelines.therapeutic_drug_monitoring:
            requirements.append("Therapeutic drug level monitoring")
        
        if self.dosing_guidelines.renal_adjustment_required:
            requirements.append("Renal function monitoring")
        
        if self.dosing_guidelines.hepatic_adjustment_required:
            requirements.append("Hepatic function monitoring")
        
        return list(set(requirements))  # Remove duplicates


@dataclass(frozen=True)
class MedicationIdentifiers:
    """
    Standard identifiers for a medication
    """
    rxnorm_code: str
    ndc_codes: List[str]
    generic_name: str
    brand_names: Optional[List[str]] = None
    synonyms: Optional[List[str]] = None
    
    def get_primary_brand_name(self) -> Optional[str]:
        """Get the primary brand name"""
        return self.brand_names[0] if self.brand_names else None
    
    def get_display_name(self) -> str:
        """Get the best display name"""
        primary_brand = self.get_primary_brand_name()
        if primary_brand:
            return f"{primary_brand} ({self.generic_name})"
        return self.generic_name
