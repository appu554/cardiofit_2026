"""
Dose Specification Value Objects
Immutable objects representing pharmaceutical dosing concepts
"""

from dataclasses import dataclass
from typing import Optional, Dict, Any, List
from decimal import Decimal
from enum import Enum


class DoseUnit(Enum):
    """Standard dose units for medications"""
    MG = "mg"
    G = "g"
    MCG = "mcg"
    UNITS = "units"
    ML = "mL"
    MG_M2 = "mg/m2"
    MG_KG = "mg/kg"
    UNITS_KG = "units/kg"
    MEQ = "mEq"
    MMOL = "mmol"


class RouteOfAdministration(Enum):
    """Routes of medication administration"""
    ORAL = "PO"
    INTRAVENOUS = "IV"
    INTRAMUSCULAR = "IM"
    SUBCUTANEOUS = "SC"
    TOPICAL = "TOP"
    INHALATION = "INH"
    RECTAL = "PR"
    SUBLINGUAL = "SL"
    TRANSDERMAL = "TD"
    INTRATHECAL = "IT"


class FrequencyType(Enum):
    """Types of dosing frequencies"""
    SCHEDULED = "scheduled"  # Regular intervals (e.g., Q8H)
    PRN = "prn"             # As needed
    ONCE = "once"           # Single dose
    CONTINUOUS = "continuous" # Continuous infusion


@dataclass(frozen=True)
class DoseSpecification:
    """
    Immutable specification of a medication dose
    Represents the calculated dose with all parameters
    """
    value: Decimal
    unit: DoseUnit
    route: RouteOfAdministration
    calculation_method: str
    calculation_factors: Dict[str, Any]
    
    def __post_init__(self):
        """Validate dose specification"""
        if self.value <= 0:
            raise ValueError("Dose value must be positive")
        
        if not isinstance(self.value, Decimal):
            # Convert to Decimal for precision
            object.__setattr__(self, 'value', Decimal(str(self.value)))
    
    def to_display_string(self) -> str:
        """Convert to human-readable string"""
        return f"{self.value} {self.unit.value} {self.route.value}"
    
    def is_weight_based(self) -> bool:
        """Check if this is a weight-based dose"""
        return self.unit in [DoseUnit.MG_KG, DoseUnit.UNITS_KG]
    
    def is_bsa_based(self) -> bool:
        """Check if this is a BSA-based dose"""
        return self.unit == DoseUnit.MG_M2


@dataclass(frozen=True)
class Frequency:
    """
    Immutable frequency specification for medication administration
    """
    type: FrequencyType
    interval_hours: Optional[int] = None
    times_per_day: Optional[int] = None
    specific_times: Optional[List[str]] = None  # e.g., ["08:00", "20:00"]
    max_doses_per_day: Optional[int] = None
    conditions: Optional[str] = None  # e.g., "with meals", "for pain"
    
    def __post_init__(self):
        """Validate frequency specification"""
        if self.type == FrequencyType.SCHEDULED:
            if not self.interval_hours and not self.times_per_day:
                raise ValueError("Scheduled frequency requires interval_hours or times_per_day")
        
        if self.type == FrequencyType.PRN:
            if not self.max_doses_per_day:
                raise ValueError("PRN frequency requires max_doses_per_day")
    
    def to_display_string(self) -> str:
        """Convert to human-readable string"""
        if self.type == FrequencyType.ONCE:
            return "Once"
        elif self.type == FrequencyType.CONTINUOUS:
            return "Continuous"
        elif self.type == FrequencyType.SCHEDULED:
            if self.interval_hours:
                return f"Every {self.interval_hours} hours"
            elif self.times_per_day:
                return f"{self.times_per_day} times daily"
        elif self.type == FrequencyType.PRN:
            base = f"As needed (max {self.max_doses_per_day}/day)"
            if self.conditions:
                base += f" {self.conditions}"
            return base
        
        return str(self.type.value)


@dataclass(frozen=True)
class Duration:
    """
    Immutable duration specification for medication therapy
    """
    days: Optional[int] = None
    weeks: Optional[int] = None
    months: Optional[int] = None
    indefinite: bool = False
    until_condition: Optional[str] = None  # e.g., "until symptoms resolve"
    
    def __post_init__(self):
        """Validate duration specification"""
        duration_specified = any([self.days, self.weeks, self.months, self.indefinite, self.until_condition])
        if not duration_specified:
            raise ValueError("Duration must specify days, weeks, months, indefinite, or until_condition")
    
    def to_total_days(self) -> Optional[int]:
        """Convert to total days if possible"""
        if self.indefinite or self.until_condition:
            return None
        
        total_days = 0
        if self.days:
            total_days += self.days
        if self.weeks:
            total_days += self.weeks * 7
        if self.months:
            total_days += self.months * 30  # Approximate
        
        return total_days if total_days > 0 else None
    
    def to_display_string(self) -> str:
        """Convert to human-readable string"""
        if self.indefinite:
            return "Indefinite"
        if self.until_condition:
            return f"Until {self.until_condition}"
        
        parts = []
        if self.months:
            parts.append(f"{self.months} month{'s' if self.months != 1 else ''}")
        if self.weeks:
            parts.append(f"{self.weeks} week{'s' if self.weeks != 1 else ''}")
        if self.days:
            parts.append(f"{self.days} day{'s' if self.days != 1 else ''}")
        
        return " ".join(parts)


@dataclass(frozen=True)
class Quantity:
    """
    Immutable quantity specification for medication dispensing
    """
    amount: Decimal
    unit: str  # tablets, capsules, mL, etc.
    days_supply: Optional[int] = None
    
    def __post_init__(self):
        """Validate quantity specification"""
        if self.amount <= 0:
            raise ValueError("Quantity amount must be positive")
        
        if not isinstance(self.amount, Decimal):
            object.__setattr__(self, 'amount', Decimal(str(self.amount)))
    
    def to_display_string(self) -> str:
        """Convert to human-readable string"""
        base = f"{self.amount} {self.unit}"
        if self.days_supply:
            base += f" ({self.days_supply} day supply)"
        return base


@dataclass(frozen=True)
class DoseCalculationContext:
    """
    Context information used for dose calculations
    Immutable snapshot of patient and medication factors
    """
    patient_id: str
    weight_kg: Optional[Decimal] = None
    height_cm: Optional[Decimal] = None
    age_years: Optional[int] = None
    age_months: Optional[int] = None
    bsa_m2: Optional[Decimal] = None
    creatinine_clearance: Optional[Decimal] = None
    egfr: Optional[Decimal] = None
    liver_function: Optional[str] = None  # normal, mild, moderate, severe
    pregnancy_status: Optional[bool] = None
    breastfeeding_status: Optional[bool] = None
    trimester: Optional[int] = None  # Pregnancy trimester (1, 2, or 3)

    # Advanced features (100% Pharmaceutical Intelligence)
    pgx_results: Optional[List[Any]] = None      # Pharmacogenomic test results
    recent_drug_levels: Optional[List[Any]] = None # Recent drug level measurements
    target_auc: Optional[Decimal] = None         # Target AUC for PK-guided dosing
    target_peak: Optional[Decimal] = None        # Target peak concentration
    target_trough: Optional[Decimal] = None      # Target trough concentration

    def __post_init__(self):
        """Validate and calculate derived values"""
        # Calculate BSA if height and weight available
        if self.height_cm and self.weight_kg and not self.bsa_m2:
            # Mosteller formula: sqrt((height_cm * weight_kg) / 3600)
            import math
            bsa = Decimal(str(math.sqrt(float(self.height_cm * self.weight_kg) / 3600)))
            object.__setattr__(self, 'bsa_m2', bsa.quantize(Decimal('0.01')))
    
    def is_pediatric(self) -> bool:
        """Check if patient is pediatric"""
        return self.age_years is not None and self.age_years < 18
    
    def is_geriatric(self) -> bool:
        """Check if patient is geriatric"""
        return self.age_years is not None and self.age_years >= 65
    
    def has_renal_impairment(self) -> bool:
        """Check if patient has renal impairment"""
        if self.creatinine_clearance:
            return self.creatinine_clearance < Decimal('60')
        if self.egfr:
            return self.egfr < Decimal('60')
        return False
    
    def has_hepatic_impairment(self) -> bool:
        """Check if patient has hepatic impairment"""
        return self.liver_function in ['mild', 'moderate', 'severe']


@dataclass(frozen=True)
class DoseProposal:
    """
    Immutable proposal for a medication dose
    Result of dose calculation with all supporting information
    """
    medication_id: str
    patient_id: str
    calculated_dose: DoseSpecification
    frequency: Frequency
    duration: Duration
    quantity: Quantity
    indication: str
    calculation_context: DoseCalculationContext
    calculation_timestamp: str
    confidence_score: Decimal  # 0.0 to 1.0
    warnings: List[str]
    clinical_notes: List[str]
    
    def __post_init__(self):
        """Validate dose proposal"""
        if not (Decimal('0') <= self.confidence_score <= Decimal('1')):
            raise ValueError("Confidence score must be between 0.0 and 1.0")
        
        if not isinstance(self.confidence_score, Decimal):
            object.__setattr__(self, 'confidence_score', Decimal(str(self.confidence_score)))
    
    def has_warnings(self) -> bool:
        """Check if proposal has warnings"""
        return len(self.warnings) > 0
    
    def is_high_confidence(self) -> bool:
        """Check if proposal has high confidence"""
        return self.confidence_score >= Decimal('0.8')
    
    def to_summary_string(self) -> str:
        """Convert to summary string"""
        return (f"{self.calculated_dose.to_display_string()} "
                f"{self.frequency.to_display_string()} "
                f"for {self.duration.to_display_string()}")
