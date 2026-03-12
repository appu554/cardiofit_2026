"""
Recipe Models for Medication Service V2 Recipe Resolver

This module defines models for recipe templates, resolution contexts, and field requirements
used by the internal Recipe Resolver system.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field, validator
from enum import Enum
from datetime import datetime, timedelta
from dataclasses import dataclass


class FieldType(str, Enum):
    """Types of fields in recipe resolution"""
    CALCULATION = "calculation"
    SAFETY = "safety"
    AUDIT = "audit"
    CONDITIONAL = "conditional"
    DEMOGRAPHIC = "demographic"
    LABORATORY = "laboratory"
    VITAL = "vital"
    MEDICATION = "medication"
    ALLERGY = "allergy"
    CONDITION = "condition"


class ConditionalOperator(str, Enum):
    """Operators for conditional rules"""
    EQUALS = "eq"
    NOT_EQUALS = "neq"
    GREATER_THAN = "gt"
    GREATER_THAN_EQUAL = "gte"
    LESS_THAN = "lt"
    LESS_THAN_EQUAL = "lte"
    IN = "in"
    NOT_IN = "not_in"
    CONTAINS = "contains"
    IS_TRUE = "is_true"
    IS_FALSE = "is_false"
    IS_NULL = "is_null"
    IS_NOT_NULL = "is_not_null"


class FreshnessRequirement(str, Enum):
    """Freshness requirements for different field types"""
    IMMEDIATE = "immediate"  # <1 minute
    RECENT = "recent"        # <15 minutes
    CURRENT = "current"      # <1 hour
    TODAY = "today"          # <24 hours
    WEEKLY = "weekly"        # <7 days
    STATIC = "static"        # No freshness requirement


class ResolutionStrategy(str, Enum):
    """Strategies for resolving field conflicts"""
    MOST_RECENT = "most_recent"
    HIGHEST_PRIORITY = "highest_priority" 
    MERGE_ALL = "merge_all"
    FAIL_ON_CONFLICT = "fail_on_conflict"
    CUSTOM_LOGIC = "custom_logic"


class RecipePhase(str, Enum):
    """Phases of medication workflow requiring different fields"""
    CALCULATION = "calculation"
    SAFETY_CHECK = "safety_check"
    AUDIT_REVIEW = "audit_review"
    FINALIZATION = "finalization"


@dataclass
class ConditionalRule:
    """Represents a conditional rule for field inclusion"""
    field_path: str
    operator: ConditionalOperator
    value: Any
    description: Optional[str] = None
    
    def evaluate(self, context_data: Dict[str, Any]) -> bool:
        """Evaluate the conditional rule against context data"""
        try:
            field_value = self._get_nested_value(context_data, self.field_path)
            
            if self.operator == ConditionalOperator.EQUALS:
                return field_value == self.value
            elif self.operator == ConditionalOperator.NOT_EQUALS:
                return field_value != self.value
            elif self.operator == ConditionalOperator.GREATER_THAN:
                return field_value > self.value
            elif self.operator == ConditionalOperator.GREATER_THAN_EQUAL:
                return field_value >= self.value
            elif self.operator == ConditionalOperator.LESS_THAN:
                return field_value < self.value
            elif self.operator == ConditionalOperator.LESS_THAN_EQUAL:
                return field_value <= self.value
            elif self.operator == ConditionalOperator.IN:
                return field_value in self.value
            elif self.operator == ConditionalOperator.NOT_IN:
                return field_value not in self.value
            elif self.operator == ConditionalOperator.CONTAINS:
                return self.value in field_value
            elif self.operator == ConditionalOperator.IS_TRUE:
                return bool(field_value) is True
            elif self.operator == ConditionalOperator.IS_FALSE:
                return bool(field_value) is False
            elif self.operator == ConditionalOperator.IS_NULL:
                return field_value is None
            elif self.operator == ConditionalOperator.IS_NOT_NULL:
                return field_value is not None
                
            return False
        except (KeyError, TypeError, AttributeError):
            return False
    
    def _get_nested_value(self, data: Dict[str, Any], path: str) -> Any:
        """Get value from nested dictionary using dot notation"""
        keys = path.split('.')
        value = data
        for key in keys:
            if isinstance(value, dict):
                value = value.get(key)
            else:
                raise KeyError(f"Path not found: {path}")
        return value


class FieldRequirement(BaseModel):
    """Represents a field requirement in a recipe"""
    field_path: str = Field(..., description="Dot notation path to field")
    field_type: FieldType = Field(..., description="Type of field")
    required: bool = Field(default=True, description="Whether field is required")
    freshness: FreshnessRequirement = Field(default=FreshnessRequirement.CURRENT)
    priority: int = Field(default=100, description="Priority for conflict resolution")
    validation_rules: Optional[Dict[str, Any]] = Field(default=None)
    conditional_rules: Optional[List[Dict[str, Any]]] = Field(default=None)
    description: Optional[str] = None
    
    def is_conditionally_required(self, context_data: Dict[str, Any]) -> bool:
        """Check if field is required based on conditional rules"""
        if not self.conditional_rules:
            return self.required
            
        # All conditional rules must pass for field to be required
        for rule_data in self.conditional_rules:
            rule = ConditionalRule(
                field_path=rule_data['field_path'],
                operator=ConditionalOperator(rule_data['operator']),
                value=rule_data['value'],
                description=rule_data.get('description')
            )
            if not rule.evaluate(context_data):
                return False
                
        return self.required
    
    def validate_freshness(self, timestamp: datetime) -> bool:
        """Validate if data meets freshness requirements"""
        now = datetime.utcnow()
        age = now - timestamp
        
        if self.freshness == FreshnessRequirement.IMMEDIATE:
            return age <= timedelta(minutes=1)
        elif self.freshness == FreshnessRequirement.RECENT:
            return age <= timedelta(minutes=15)
        elif self.freshness == FreshnessRequirement.CURRENT:
            return age <= timedelta(hours=1)
        elif self.freshness == FreshnessRequirement.TODAY:
            return age <= timedelta(days=1)
        elif self.freshness == FreshnessRequirement.WEEKLY:
            return age <= timedelta(days=7)
        elif self.freshness == FreshnessRequirement.STATIC:
            return True
            
        return False


class RecipeTemplate(BaseModel):
    """Template defining field requirements for specific clinical scenarios"""
    recipe_id: str = Field(..., description="Unique recipe identifier")
    name: str = Field(..., description="Human-readable recipe name")
    version: str = Field(default="1.0", description="Recipe version")
    description: Optional[str] = None
    clinical_scenario: str = Field(..., description="Clinical scenario this recipe addresses")
    
    # Field requirements organized by phase
    calculation_fields: List[FieldRequirement] = Field(default_factory=list)
    safety_fields: List[FieldRequirement] = Field(default_factory=list)
    audit_fields: List[FieldRequirement] = Field(default_factory=list)
    conditional_fields: List[FieldRequirement] = Field(default_factory=list)
    
    # Recipe configuration
    priority: int = Field(default=100, description="Recipe selection priority")
    triggers: List[Dict[str, Any]] = Field(default_factory=list)
    cache_ttl_seconds: int = Field(default=300, description="Cache TTL in seconds")
    resolution_strategy: ResolutionStrategy = Field(default=ResolutionStrategy.MOST_RECENT)
    
    # Performance targets
    target_resolution_time_ms: int = Field(default=10, description="Target resolution time")
    
    def get_all_fields(self) -> List[FieldRequirement]:
        """Get all field requirements across all phases"""
        return (
            self.calculation_fields + 
            self.safety_fields + 
            self.audit_fields + 
            self.conditional_fields
        )
    
    def get_fields_for_phase(self, phase: RecipePhase) -> List[FieldRequirement]:
        """Get field requirements for a specific phase"""
        if phase == RecipePhase.CALCULATION:
            return self.calculation_fields
        elif phase == RecipePhase.SAFETY_CHECK:
            return self.safety_fields
        elif phase == RecipePhase.AUDIT_REVIEW:
            return self.audit_fields
        elif phase == RecipePhase.FINALIZATION:
            return self.conditional_fields
        return []
    
    def evaluate_triggers(self, context_data: Dict[str, Any]) -> bool:
        """Evaluate if recipe triggers match the context"""
        if not self.triggers:
            return False
            
        for trigger in self.triggers:
            field_path = trigger.get('field_path')
            operator = ConditionalOperator(trigger.get('operator', 'eq'))
            value = trigger.get('value')
            
            rule = ConditionalRule(field_path, operator, value)
            if not rule.evaluate(context_data):
                return False
                
        return True


class ResolutionContext(BaseModel):
    """Context for recipe resolution request"""
    patient_id: str = Field(..., description="Patient identifier")
    medication_code: Optional[str] = None
    medication_name: Optional[str] = None
    indication: Optional[str] = None
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    
    # Patient characteristics for conditional rules
    patient_age: Optional[int] = None
    patient_weight_kg: Optional[float] = None
    patient_pregnancy_status: Optional[bool] = None
    patient_renal_function: Optional[str] = None
    patient_conditions: List[str] = Field(default_factory=list)
    patient_allergies: List[str] = Field(default_factory=list)
    
    # Request metadata
    priority: str = Field(default="routine")
    force_refresh: bool = Field(default=False)
    requested_phases: List[RecipePhase] = Field(default_factory=list)
    
    class Config:
        use_enum_values = True


class ResolvedField(BaseModel):
    """Represents a resolved field with metadata"""
    field_path: str
    value: Any
    field_type: FieldType
    source: str = Field(description="Data source that provided this field")
    timestamp: datetime = Field(description="When this data was captured")
    freshness_valid: bool = Field(description="Whether data meets freshness requirements")
    priority: int = Field(description="Priority used for conflict resolution")
    
    class Config:
        use_enum_values = True


class FieldConflict(BaseModel):
    """Represents a conflict between field values"""
    field_path: str
    conflicting_values: List[ResolvedField]
    resolution_strategy: ResolutionStrategy
    resolved_value: Optional[ResolvedField] = None
    
    class Config:
        use_enum_values = True


class RecipeResolutionResult(BaseModel):
    """Result of recipe resolution process"""
    recipe_id: str
    recipe_version: str
    resolution_context: ResolutionContext
    
    # Resolved fields organized by phase
    calculation_fields: Dict[str, ResolvedField] = Field(default_factory=dict)
    safety_fields: Dict[str, ResolvedField] = Field(default_factory=dict)
    audit_fields: Dict[str, ResolvedField] = Field(default_factory=dict)
    conditional_fields: Dict[str, ResolvedField] = Field(default_factory=dict)
    
    # Resolution metadata
    resolution_time_ms: float
    cache_hit: bool = Field(default=False)
    field_conflicts: List[FieldConflict] = Field(default_factory=list)
    missing_required_fields: List[str] = Field(default_factory=list)
    freshness_violations: List[str] = Field(default_factory=list)
    
    # Completeness metrics
    completeness_score: float = Field(description="Percentage of required fields resolved")
    total_fields_requested: int
    total_fields_resolved: int
    
    # Validation results
    validation_passed: bool = Field(default=True)
    validation_errors: List[str] = Field(default_factory=list)
    validation_warnings: List[str] = Field(default_factory=list)
    
    created_at: datetime = Field(default_factory=datetime.utcnow)
    
    def get_all_resolved_fields(self) -> Dict[str, ResolvedField]:
        """Get all resolved fields across all phases"""
        all_fields = {}
        all_fields.update(self.calculation_fields)
        all_fields.update(self.safety_fields)
        all_fields.update(self.audit_fields)
        all_fields.update(self.conditional_fields)
        return all_fields
    
    def get_fields_for_phase(self, phase: RecipePhase) -> Dict[str, ResolvedField]:
        """Get resolved fields for a specific phase"""
        if phase == RecipePhase.CALCULATION:
            return self.calculation_fields
        elif phase == RecipePhase.SAFETY_CHECK:
            return self.safety_fields
        elif phase == RecipePhase.AUDIT_REVIEW:
            return self.audit_fields
        elif phase == RecipePhase.FINALIZATION:
            return self.conditional_fields
        return {}
    
    class Config:
        use_enum_values = True


class RecipeCache(BaseModel):
    """Cached recipe resolution result"""
    cache_key: str
    result: RecipeResolutionResult
    created_at: datetime = Field(default_factory=datetime.utcnow)
    ttl_seconds: int = Field(default=300)
    access_count: int = Field(default=0)
    
    def is_expired(self) -> bool:
        """Check if cache entry has expired"""
        age = datetime.utcnow() - self.created_at
        return age.total_seconds() > self.ttl_seconds
    
    def increment_access(self):
        """Increment access counter"""
        self.access_count += 1


class RecipeResolutionError(Exception):
    """Base exception for recipe resolution errors"""
    pass


class RecipeNotFoundError(RecipeResolutionError):
    """Recipe template not found"""
    pass


class ConditionalRuleError(RecipeResolutionError):
    """Error in conditional rule evaluation"""
    pass


class FieldResolutionError(RecipeResolutionError):
    """Error resolving required fields"""
    pass


class FreshnessViolationError(RecipeResolutionError):
    """Data freshness requirement violation"""
    pass