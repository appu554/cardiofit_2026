"""
Clinical Context Models - Core data models for the Clinical Context Recipe System
Implements Pillar 2: Clinical Context Recipe System (The "Governance Engine")
"""
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any, Union
from datetime import datetime, timedelta
from enum import Enum
import uuid


class DataSourceType(Enum):
    """Supported data source types for clinical context assembly"""
    PATIENT_SERVICE = "patient_service"
    MEDICATION_SERVICE = "medication_service"
    LAB_SERVICE = "lab_service"
    ALLERGY_SERVICE = "allergy_service"
    CONDITION_SERVICE = "condition_service"
    ENCOUNTER_SERVICE = "encounter_service"
    FHIR_STORE = "fhir_store"
    CAE_SERVICE = "cae_service"
    CONTEXT_SERVICE_INTERNAL = "context_service_internal"
    GRAPH_DB = "graph_db"
    # New data source types from ContextRecipeBook.txt
    APOLLO_FEDERATION = "apollo_federation"
    WORKFLOW_ENGINE = "workflow_engine"
    SAFETY_GATEWAY = "safety_gateway"
    ELASTICSEARCH = "elasticsearch"
    OBSERVATION_SERVICE = "observation_service"
    DEVICE_DATA_SERVICE = "device_data_service"
    CONTEXT_SERVICE = "context_service"


class SafetyFlagType(Enum):
    """Types of safety flags that can be raised during context assembly"""
    DRUG_INTERACTION = "drug_interaction"
    ALLERGY_ALERT = "allergy_alert"
    DOSAGE_WARNING = "dosage_warning"
    CONTRAINDICATION = "contraindication"
    DATA_QUALITY = "data_quality"
    STALE_DATA = "stale_data"
    MISSING_CRITICAL_DATA = "missing_critical_data"
    SERVICE_UNAVAILABLE = "service_unavailable"


class SafetySeverity(Enum):
    """Severity levels for safety flags"""
    INFO = "info"
    WARNING = "warning"
    CRITICAL = "critical"
    FATAL = "fatal"


class ContextStatus(Enum):
    """Status of clinical context assembly"""
    SUCCESS = "success"
    PARTIAL = "partial"
    FAILED = "failed"
    UNAVAILABLE = "unavailable"


@dataclass
class SafetyFlag:
    """Safety flag raised during context assembly"""
    flag_type: SafetyFlagType
    severity: SafetySeverity
    message: str
    data_point: Optional[str] = None
    details: Optional[Dict[str, Any]] = None
    timestamp: datetime = field(default_factory=datetime.utcnow)


@dataclass
class SourceMetadata:
    """Metadata about data source and retrieval"""
    source_type: DataSourceType
    source_endpoint: str
    retrieved_at: datetime
    data_version: str
    completeness: float  # 0.0 to 1.0
    response_time_ms: float
    cache_hit: bool = False
    error_message: Optional[str] = None


@dataclass
class DataPoint:
    """Definition of a data point required by a recipe"""
    name: str
    source_type: DataSourceType
    fields: List[str]
    required: bool = True
    max_age_hours: int = 24
    quality_threshold: float = 0.8
    timeout_ms: int = 5000
    retry_count: int = 2
    fallback_sources: List[DataSourceType] = field(default_factory=list)


@dataclass
class ConditionalRule:
    """Conditional rule for additional data requirements"""
    condition: str  # Python expression to evaluate
    additional_data_points: List[DataPoint]
    description: str


@dataclass
class QualityConstraints:
    """Data quality constraints for recipe"""
    minimum_completeness: float = 0.8
    maximum_age_hours: int = 24
    required_fields: List[str] = field(default_factory=list)
    accuracy_threshold: float = 0.9


@dataclass
class CacheStrategy:
    """Caching strategy for recipe"""
    l1_ttl_seconds: int = 300  # 5 minutes
    l2_ttl_seconds: int = 900  # 15 minutes
    l3_ttl_seconds: int = 3600  # 1 hour
    invalidation_events: List[str] = field(default_factory=list)
    cache_key_pattern: str = "context:{patient_id}:{recipe_id}"
    # New attributes from ContextRecipeBook.txt
    emergency_cache: bool = False
    cache_strategy: str = "l1_l2_l3"


@dataclass
class GovernanceMetadata:
    """Governance metadata for recipe approval and tracking"""
    approved_by: str
    approval_date: datetime
    version: str
    effective_date: datetime
    expiry_date: Optional[datetime] = None
    clinical_board_approval_id: str = ""
    tags: List[str] = field(default_factory=list)
    change_log: List[str] = field(default_factory=list)


@dataclass
class SafetyRequirements:
    """Safety requirements for recipe"""
    minimum_completeness_score: float = 0.85
    absolute_required_enforcement: str = "STRICT"  # STRICT, LENIENT
    preferred_data_handling: str = "GRACEFUL_DEGRADE"  # GRACEFUL_DEGRADE, FAIL
    critical_missing_data_action: str = "FAIL_WORKFLOW"  # FAIL_WORKFLOW, FLAG_FOR_REVIEW
    stale_data_action: str = "FLAG_FOR_REVIEW"  # FLAG_FOR_REVIEW, REJECT, ACCEPT
    mock_data_policy: str = "STRICTLY_PROHIBITED"  # STRICTLY_PROHIBITED, ALLOWED_FOR_TESTING


@dataclass
class AssemblyRules:
    """Rules for assembling clinical context"""
    parallel_execution: bool = True
    timeout_budget_ms: int = 200
    circuit_breaker_enabled: bool = True
    retry_failed_sources: bool = True
    validate_data_freshness: bool = True
    enforce_quality_constraints: bool = True
    # New attributes from ContextRecipeBook.txt
    fail_fast: bool = False
    fail_closed: bool = False
    emergency_mode: bool = False


@dataclass
class ContextRecipe:
    """
    Clinical Context Recipe - Defines what data to gather and how to assemble it
    Implements the governance-as-code pattern with version control
    """
    recipe_id: str
    recipe_name: str
    version: str
    clinical_scenario: str
    workflow_category: str  # command_initiated, event_triggered
    execution_pattern: str  # pessimistic, optimistic, digital_reflex_arc
    
    # Core recipe definition
    required_data_points: List[DataPoint]
    conditional_rules: List[ConditionalRule] = field(default_factory=list)
    quality_constraints: QualityConstraints = field(default_factory=QualityConstraints)
    safety_requirements: SafetyRequirements = field(default_factory=SafetyRequirements)
    cache_strategy: CacheStrategy = field(default_factory=CacheStrategy)
    assembly_rules: AssemblyRules = field(default_factory=AssemblyRules)
    
    # Governance and metadata
    governance_metadata: GovernanceMetadata = None
    
    # Performance requirements
    sla_ms: int = 200  # SLA budget for context assembly
    cache_duration_seconds: int = 300
    real_data_only: bool = True
    mock_data_detection: bool = True
    
    # Recipe inheritance
    base_recipe_id: Optional[str] = None
    extends_recipes: List[str] = field(default_factory=list)
    
    def validate_governance(self) -> bool:
        """Ensure recipe is approved by Clinical Governance Board"""
        if not self.governance_metadata:
            return False
        return self.governance_metadata.approved_by == "Clinical Governance Board"
    
    def is_expired(self) -> bool:
        """Check if recipe has expired"""
        if not self.governance_metadata or not self.governance_metadata.expiry_date:
            return False

        # Ensure both datetimes are timezone-naive for comparison
        now = datetime.utcnow()
        expiry = self.governance_metadata.expiry_date

        # Convert to naive datetime if timezone-aware
        if expiry.tzinfo is not None:
            expiry = expiry.replace(tzinfo=None)

        return now > expiry
    
    def get_cache_key(self, patient_id: str, provider_id: Optional[str] = None) -> str:
        """Generate cache key for this recipe"""
        base_key = self.cache_strategy.cache_key_pattern.format(
            patient_id=patient_id,
            recipe_id=self.recipe_id
        )
        if provider_id:
            base_key += f":{provider_id}"
        return base_key


@dataclass
class ClinicalContext:
    """
    Assembled clinical context containing all required data for a workflow
    """
    context_id: str
    patient_id: str
    recipe_used: str
    
    # Core context data
    assembled_data: Dict[str, Any]
    completeness_score: float
    data_freshness: Dict[str, datetime]
    source_metadata: Dict[str, SourceMetadata]
    
    # Safety and governance
    safety_flags: List[SafetyFlag] = field(default_factory=list)
    governance_tags: List[str] = field(default_factory=list)
    
    # Assembly metadata
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    assembled_at: datetime = field(default_factory=datetime.utcnow)
    assembly_duration_ms: float = 0.0
    status: ContextStatus = ContextStatus.SUCCESS
    
    # Error tracking
    connection_errors: List[Dict[str, str]] = field(default_factory=list)
    
    # Cache information
    cache_hit: bool = False
    cache_key: str = ""
    ttl_seconds: int = 300
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            "context_id": self.context_id,
            "patient_id": self.patient_id,
            "recipe_used": self.recipe_used,
            "assembled_data": self.assembled_data,
            "completeness_score": self.completeness_score,
            "data_freshness": {k: v.isoformat() for k, v in self.data_freshness.items()},
            "source_metadata": {k: {
                "source_type": v.source_type.value,
                "source_endpoint": v.source_endpoint,
                "retrieved_at": v.retrieved_at.isoformat(),
                "data_version": v.data_version,
                "completeness": v.completeness,
                "response_time_ms": v.response_time_ms,
                "cache_hit": v.cache_hit
            } for k, v in self.source_metadata.items()},
            "safety_flags": [{
                "flag_type": flag.flag_type.value,
                "severity": flag.severity.value,
                "message": flag.message,
                "data_point": flag.data_point,
                "details": flag.details,
                "timestamp": flag.timestamp.isoformat()
            } for flag in self.safety_flags],
            "governance_tags": self.governance_tags,
            "provider_id": self.provider_id,
            "encounter_id": self.encounter_id,
            "assembled_at": self.assembled_at.isoformat(),
            "assembly_duration_ms": self.assembly_duration_ms,
            "status": self.status.value,
            "connection_errors": self.connection_errors,
            "cache_hit": self.cache_hit,
            "cache_key": self.cache_key,
            "ttl_seconds": self.ttl_seconds
        }
    
    def to_json(self) -> str:
        """Convert to JSON string"""
        import json
        return json.dumps(self.to_dict())
    
    @classmethod
    def from_json(cls, json_str: str) -> 'ClinicalContext':
        """Create from JSON string"""
        import json
        data = json.loads(json_str)
        
        # Convert datetime strings back to datetime objects
        data_freshness = {k: datetime.fromisoformat(v) for k, v in data["data_freshness"].items()}
        
        source_metadata = {}
        for k, v in data["source_metadata"].items():
            source_metadata[k] = SourceMetadata(
                source_type=DataSourceType(v["source_type"]),
                source_endpoint=v["source_endpoint"],
                retrieved_at=datetime.fromisoformat(v["retrieved_at"]),
                data_version=v["data_version"],
                completeness=v["completeness"],
                response_time_ms=v["response_time_ms"],
                cache_hit=v["cache_hit"]
            )
        
        safety_flags = []
        for flag_data in data["safety_flags"]:
            safety_flags.append(SafetyFlag(
                flag_type=SafetyFlagType(flag_data["flag_type"]),
                severity=SafetySeverity(flag_data["severity"]),
                message=flag_data["message"],
                data_point=flag_data["data_point"],
                details=flag_data["details"],
                timestamp=datetime.fromisoformat(flag_data["timestamp"])
            ))
        
        return cls(
            context_id=data["context_id"],
            patient_id=data["patient_id"],
            recipe_used=data["recipe_used"],
            assembled_data=data["assembled_data"],
            completeness_score=data["completeness_score"],
            data_freshness=data_freshness,
            source_metadata=source_metadata,
            safety_flags=safety_flags,
            governance_tags=data["governance_tags"],
            provider_id=data["provider_id"],
            encounter_id=data["encounter_id"],
            assembled_at=datetime.fromisoformat(data["assembled_at"]),
            assembly_duration_ms=data["assembly_duration_ms"],
            status=ContextStatus(data["status"]),
            connection_errors=data["connection_errors"],
            cache_hit=data["cache_hit"],
            cache_key=data["cache_key"],
            ttl_seconds=data["ttl_seconds"]
        )


class ClinicalDataError(Exception):
    """Exception raised when clinical data cannot be retrieved or is invalid"""
    pass


class RecipeValidationError(Exception):
    """Exception raised when recipe validation fails"""
    pass


class GovernanceError(Exception):
    """Exception raised when governance requirements are not met"""
    pass
