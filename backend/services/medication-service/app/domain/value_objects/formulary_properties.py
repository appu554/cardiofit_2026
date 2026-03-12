"""
Formulary Properties Value Objects
Immutable objects representing formulary and insurance concepts
"""

from dataclasses import dataclass
from typing import Optional, List, Dict, Any
from decimal import Decimal
from enum import Enum
from datetime import date


class FormularyStatus(Enum):
    """Formulary status classifications"""
    PREFERRED = "preferred"           # Tier 1 - Lowest cost
    NON_PREFERRED = "non_preferred"   # Tier 2 - Moderate cost
    SPECIALTY = "specialty"           # Tier 3 - High cost specialty drugs
    PRIOR_AUTH_REQUIRED = "prior_auth_required"  # Requires prior authorization
    STEP_THERAPY = "step_therapy"     # Must try other drugs first
    QUANTITY_LIMIT = "quantity_limit" # Limited quantities allowed
    NOT_COVERED = "not_covered"       # Not on formulary
    EXCLUDED = "excluded"             # Explicitly excluded


class CostTier(Enum):
    """Insurance cost tiers"""
    TIER_1 = 1  # Generic drugs - lowest copay
    TIER_2 = 2  # Preferred brand drugs
    TIER_3 = 3  # Non-preferred brand drugs
    TIER_4 = 4  # Specialty drugs - highest copay
    TIER_5 = 5  # Specialty biologics


class PriorAuthStatus(Enum):
    """Prior authorization status"""
    NOT_REQUIRED = "not_required"
    REQUIRED = "required"
    APPROVED = "approved"
    DENIED = "denied"
    PENDING = "pending"
    EXPIRED = "expired"


@dataclass(frozen=True)
class CostInformation:
    """Cost information for a medication"""
    wholesale_price: Optional[Decimal] = None
    average_wholesale_price: Optional[Decimal] = None
    patient_copay: Optional[Decimal] = None
    insurance_coverage_percent: Optional[Decimal] = None
    cost_per_unit: Optional[Decimal] = None
    cost_per_day: Optional[Decimal] = None
    cost_tier: Optional[CostTier] = None
    
    def __post_init__(self):
        """Validate cost information"""
        if self.insurance_coverage_percent is not None:
            if not (Decimal('0') <= self.insurance_coverage_percent <= Decimal('100')):
                raise ValueError("Insurance coverage percent must be between 0 and 100")
    
    def get_patient_cost(self, quantity: Decimal) -> Optional[Decimal]:
        """Calculate patient cost for given quantity"""
        if self.patient_copay:
            return self.patient_copay
        
        if self.cost_per_unit and self.insurance_coverage_percent:
            total_cost = self.cost_per_unit * quantity
            patient_portion = Decimal('100') - self.insurance_coverage_percent
            return total_cost * (patient_portion / Decimal('100'))
        
        return None
    
    def is_cost_effective(self, alternative_cost: 'CostInformation') -> bool:
        """Compare cost effectiveness with alternative"""
        if not self.cost_per_day or not alternative_cost.cost_per_day:
            return False
        
        return self.cost_per_day < alternative_cost.cost_per_day


@dataclass(frozen=True)
class FormularyRestriction:
    """Formulary restriction details"""
    restriction_type: str  # prior_auth, step_therapy, quantity_limit
    description: str
    requirements: List[str]
    override_criteria: Optional[List[str]] = None
    documentation_required: Optional[List[str]] = None
    
    def is_override_possible(self) -> bool:
        """Check if restriction can be overridden"""
        return bool(self.override_criteria)


@dataclass(frozen=True)
class StepTherapyRequirement:
    """Step therapy requirement details"""
    required_medications: List[str]  # Medications that must be tried first
    trial_duration_days: int
    failure_criteria: List[str]
    exceptions: Optional[List[str]] = None
    
    def __post_init__(self):
        """Validate step therapy requirements"""
        if self.trial_duration_days <= 0:
            raise ValueError("Trial duration must be positive")
        if not self.required_medications:
            raise ValueError("Required medications list cannot be empty")


@dataclass(frozen=True)
class QuantityLimit:
    """Quantity limit restrictions"""
    max_quantity_per_fill: Optional[Decimal] = None
    max_days_supply: Optional[int] = None
    max_quantity_per_month: Optional[Decimal] = None
    max_refills_per_year: Optional[int] = None
    limit_reason: Optional[str] = None
    
    def is_quantity_allowed(self, requested_quantity: Decimal, days_supply: int) -> bool:
        """Check if requested quantity is within limits"""
        if self.max_quantity_per_fill and requested_quantity > self.max_quantity_per_fill:
            return False
        
        if self.max_days_supply and days_supply > self.max_days_supply:
            return False
        
        return True


@dataclass(frozen=True)
class FormularyEntry:
    """Complete formulary entry for a medication"""
    medication_id: str
    formulary_id: str
    insurance_plan_id: str
    
    # Status and Tier
    formulary_status: FormularyStatus
    cost_tier: CostTier
    
    # Cost Information
    cost_info: CostInformation
    
    # Effective Dates (required fields first)
    effective_date: date
    last_updated: date

    # Restrictions
    prior_auth_required: bool = False
    step_therapy_required: bool = False
    quantity_limits: Optional[QuantityLimit] = None
    restrictions: Optional[List[FormularyRestriction]] = None
    step_therapy: Optional[StepTherapyRequirement] = None

    # Optional Dates
    expiration_date: Optional[date] = None

    # Metadata
    notes: Optional[str] = None
    
    def is_active(self, check_date: Optional[date] = None) -> bool:
        """Check if formulary entry is currently active"""
        if check_date is None:
            check_date = date.today()
        
        if check_date < self.effective_date:
            return False
        
        if self.expiration_date and check_date > self.expiration_date:
            return False
        
        return True
    
    def is_preferred(self) -> bool:
        """Check if medication is preferred on formulary"""
        return self.formulary_status == FormularyStatus.PREFERRED
    
    def requires_prior_authorization(self) -> bool:
        """Check if prior authorization is required"""
        return (self.prior_auth_required or 
                self.formulary_status == FormularyStatus.PRIOR_AUTH_REQUIRED)
    
    def has_restrictions(self) -> bool:
        """Check if medication has any restrictions"""
        return (self.prior_auth_required or 
                self.step_therapy_required or 
                self.quantity_limits is not None or
                bool(self.restrictions))
    
    def get_restriction_summary(self) -> List[str]:
        """Get summary of all restrictions"""
        restrictions = []
        
        if self.prior_auth_required:
            restrictions.append("Prior authorization required")
        
        if self.step_therapy_required and self.step_therapy:
            restrictions.append(f"Step therapy required: try {', '.join(self.step_therapy.required_medications)} first")
        
        if self.quantity_limits:
            if self.quantity_limits.max_quantity_per_fill:
                restrictions.append(f"Quantity limit: {self.quantity_limits.max_quantity_per_fill} per fill")
            if self.quantity_limits.max_days_supply:
                restrictions.append(f"Days supply limit: {self.quantity_limits.max_days_supply} days")
        
        if self.restrictions:
            for restriction in self.restrictions:
                restrictions.append(restriction.description)
        
        return restrictions


@dataclass(frozen=True)
class TherapeuticAlternative:
    """Therapeutic alternative medication"""
    medication_id: str
    medication_name: str
    therapeutic_equivalence: str  # AB, AA, etc. (Orange Book ratings)
    formulary_status: FormularyStatus
    cost_comparison: str  # "lower", "similar", "higher"
    cost_savings_percent: Optional[Decimal] = None
    clinical_notes: Optional[str] = None
    
    def is_therapeutically_equivalent(self) -> bool:
        """Check if medication is therapeutically equivalent (AB rated)"""
        return self.therapeutic_equivalence.startswith('AB')
    
    def is_cost_effective(self) -> bool:
        """Check if alternative is more cost effective"""
        return self.cost_comparison == "lower"


@dataclass(frozen=True)
class FormularyRecommendation:
    """Formulary-based recommendation"""
    original_medication_id: str
    recommended_medication_id: str
    recommendation_type: str  # "preferred_alternative", "generic_substitution", "therapeutic_alternative"
    reason: str
    cost_savings: Optional[Decimal] = None
    clinical_considerations: Optional[List[str]] = None
    formulary_advantages: Optional[List[str]] = None
    
    def get_recommendation_summary(self) -> str:
        """Get human-readable recommendation summary"""
        base_message = f"Consider {self.recommendation_type}: {self.reason}"
        
        if self.cost_savings:
            base_message += f" (Potential savings: ${self.cost_savings})"
        
        return base_message


@dataclass(frozen=True)
class InsurancePlan:
    """Insurance plan information"""
    plan_id: str
    plan_name: str
    plan_type: str  # HMO, PPO, Medicare, Medicaid, etc.
    formulary_id: str
    
    # Coverage Details
    deductible: Optional[Decimal] = None
    out_of_pocket_max: Optional[Decimal] = None
    
    # Tier Copays
    tier_1_copay: Optional[Decimal] = None
    tier_2_copay: Optional[Decimal] = None
    tier_3_copay: Optional[Decimal] = None
    tier_4_copay: Optional[Decimal] = None
    tier_5_copay: Optional[Decimal] = None
    
    # Special Programs
    mail_order_available: bool = False
    specialty_pharmacy_required: bool = False
    
    def get_copay_for_tier(self, tier: CostTier) -> Optional[Decimal]:
        """Get copay amount for specific tier"""
        copay_map = {
            CostTier.TIER_1: self.tier_1_copay,
            CostTier.TIER_2: self.tier_2_copay,
            CostTier.TIER_3: self.tier_3_copay,
            CostTier.TIER_4: self.tier_4_copay,
            CostTier.TIER_5: self.tier_5_copay
        }
        return copay_map.get(tier)
    
    def is_specialty_plan(self) -> bool:
        """Check if this is a specialty insurance plan"""
        return self.specialty_pharmacy_required


@dataclass(frozen=True)
class FormularySearchCriteria:
    """Criteria for formulary searches"""
    insurance_plan_id: Optional[str] = None
    formulary_status: Optional[FormularyStatus] = None
    cost_tier: Optional[CostTier] = None
    max_copay: Optional[Decimal] = None
    therapeutic_class: Optional[str] = None
    exclude_prior_auth: bool = False
    exclude_step_therapy: bool = False
    preferred_only: bool = False
    
    def matches_entry(self, entry: FormularyEntry) -> bool:
        """Check if formulary entry matches search criteria"""
        if self.insurance_plan_id and entry.insurance_plan_id != self.insurance_plan_id:
            return False
        
        if self.formulary_status and entry.formulary_status != self.formulary_status:
            return False
        
        if self.cost_tier and entry.cost_tier != self.cost_tier:
            return False
        
        if self.max_copay and entry.cost_info.patient_copay:
            if entry.cost_info.patient_copay > self.max_copay:
                return False
        
        if self.exclude_prior_auth and entry.requires_prior_authorization():
            return False
        
        if self.exclude_step_therapy and entry.step_therapy_required:
            return False
        
        if self.preferred_only and not entry.is_preferred():
            return False
        
        return True
