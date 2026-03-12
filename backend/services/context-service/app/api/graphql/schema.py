"""
GraphQL Schema for Clinical Context Service
Implements Pillar 1: Federated GraphQL API (The "Unified Data Graph")
"""
import strawberry
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum

from app.models.context_models import (
    ContextStatus, SafetyFlagType, SafetySeverity, DataSourceType
)


@strawberry.enum
class ContextStatusEnum(Enum):
    """GraphQL enum for context status"""
    SUCCESS = "success"
    PARTIAL = "partial"
    FAILED = "failed"
    UNAVAILABLE = "unavailable"


@strawberry.enum
class SafetyFlagTypeEnum(Enum):
    """GraphQL enum for safety flag types"""
    DRUG_INTERACTION = "drug_interaction"
    ALLERGY_ALERT = "allergy_alert"
    DOSAGE_WARNING = "dosage_warning"
    CONTRAINDICATION = "contraindication"
    DATA_QUALITY = "data_quality"
    STALE_DATA = "stale_data"
    MISSING_CRITICAL_DATA = "missing_critical_data"
    SERVICE_UNAVAILABLE = "service_unavailable"


@strawberry.enum
class SafetySeverityEnum(Enum):
    """GraphQL enum for safety severity levels"""
    INFO = "info"
    WARNING = "warning"
    CRITICAL = "critical"
    FATAL = "fatal"


@strawberry.enum
class DataSourceTypeEnum(Enum):
    """GraphQL enum for data source types"""
    PATIENT_SERVICE = "patient_service"
    MEDICATION_SERVICE = "medication_service"
    LAB_SERVICE = "lab_service"
    ALLERGY_SERVICE = "allergy_service"
    CONDITION_SERVICE = "condition_service"
    ENCOUNTER_SERVICE = "encounter_service"
    FHIR_STORE = "fhir_store"
    CAE_SERVICE = "cae_service"
    GRAPH_DB = "graph_db"


@strawberry.type
class SafetyFlag:
    """Safety flag raised during context assembly"""
    flag_type: SafetyFlagTypeEnum
    severity: SafetySeverityEnum
    message: str
    data_point: Optional[str] = None
    details: Optional[strawberry.scalars.JSON] = None
    timestamp: datetime


@strawberry.type
class SourceMetadata:
    """Metadata about data source and retrieval"""
    source_type: DataSourceTypeEnum
    source_endpoint: str
    retrieved_at: datetime
    data_version: str
    completeness: float
    response_time_ms: float
    cache_hit: bool = False
    error_message: Optional[str] = None


@strawberry.type
class ConnectionError:
    """Connection error information"""
    data_point: str
    source: str
    error: str
    timestamp: datetime


@strawberry.type
class ClinicalContext:
    """
    Assembled clinical context containing all required data for a workflow.
    Primary response type for context queries.
    """
    context_id: str
    patient_id: str
    recipe_used: str
    
    # Core context data
    assembled_data: strawberry.scalars.JSON
    completeness_score: float
    data_freshness: strawberry.scalars.JSON
    source_metadata: strawberry.scalars.JSON
    
    # Safety and governance
    safety_flags: List[SafetyFlag]
    governance_tags: List[str]
    
    # Assembly metadata
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    assembled_at: datetime
    assembly_duration_ms: float
    status: ContextStatusEnum
    
    # Error tracking
    connection_errors: List[ConnectionError]
    
    # Cache information
    cache_hit: bool = False
    cache_key: str = ""
    ttl_seconds: int = 300


@strawberry.type
class ContextFieldsResponse:
    """Response for field-specific context queries"""
    data: strawberry.scalars.JSON
    completeness: float
    metadata: strawberry.scalars.JSON
    status: ContextStatusEnum


@strawberry.type
class ContextAvailabilityResponse:
    """Response for context availability validation"""
    available: bool
    recipe_id: str
    patient_id: str
    estimated_completeness: float
    unavailable_sources: List[str]
    estimated_assembly_time_ms: int
    cache_available: bool


@strawberry.type
class RecipeInfo:
    """Information about a clinical context recipe"""
    recipe_id: str
    recipe_name: str
    version: str
    clinical_scenario: str
    workflow_category: str
    execution_pattern: str
    sla_ms: int
    governance_approved: bool
    effective_date: Optional[datetime] = None
    expiry_date: Optional[datetime] = None


@strawberry.type
class CacheStats:
    """Cache performance statistics"""
    total_entries: int
    hit_ratio: float
    l1_entries: int
    l2_entries: int
    last_updated: datetime
    performance_metrics: strawberry.scalars.JSON


@strawberry.type
class ServiceHealth:
    """Health status of a data source service"""
    service_type: DataSourceTypeEnum
    endpoint: str
    healthy: bool
    response_time_ms: int
    last_check: datetime
    error_message: Optional[str] = None


@strawberry.input
class ContextRequest:
    """Input for context assembly requests"""
    patient_id: str
    recipe_id: str
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    force_refresh: bool = False
    workflow_id: Optional[str] = None


@strawberry.input
class FieldsRequest:
    """Input for field-specific context requests"""
    patient_id: str
    fields: List[str]
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    max_age_hours: Optional[int] = 24


@strawberry.type
class Query:
    """
    GraphQL Query root for Clinical Context Service.
    Implements the unified data graph pattern.
    """
    
    @strawberry.field
    async def get_context_by_recipe(
        self,
        info,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None,
        force_refresh: bool = False,
        workflow_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Primary method for recipe-based context assembly.
        Used by Workflow Engine and other consumers for comprehensive context retrieval.
        """
        from app.api.graphql.resolvers import context_resolver
        return await context_resolver.get_context_by_recipe(
            patient_id, recipe_id, provider_id, encounter_id, force_refresh, workflow_id
        )
    
    @strawberry.field
    async def get_context_fields(
        self,
        info,
        patient_id: str,
        fields: List[str],
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None,
        max_age_hours: int = 24
    ) -> ContextFieldsResponse:
        """
        Field-specific context queries for targeted data retrieval.
        Optimized for domain services that need specific data points.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.field
    async def validate_context_availability(
        self,
        info,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None
    ) -> ContextAvailabilityResponse:
        """
        Validate context availability before workflow execution.
        Provides pre-flight checks for workflow orchestration.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.field
    async def get_available_recipes(
        self,
        info,
        clinical_scenario: Optional[str] = None,
        workflow_category: Optional[str] = None
    ) -> List[RecipeInfo]:
        """
        Get available clinical context recipes.
        Filtered by clinical scenario and workflow category.
        """
        from app.api.graphql.resolvers import context_resolver
        return await context_resolver.get_available_recipes(clinical_scenario, workflow_category)
    
    @strawberry.field
    async def get_recipe_info(
        self,
        info,
        recipe_id: str,
        version: str = "latest"
    ) -> Optional[RecipeInfo]:
        """
        Get detailed information about a specific recipe.
        """
        from app.api.graphql.resolvers import context_resolver
        return await context_resolver.get_recipe_info(recipe_id, version)
    
    @strawberry.field
    async def get_cache_stats(self, info) -> CacheStats:
        """
        Get cache performance statistics.
        Used for monitoring and optimization.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.field
    async def get_service_health(self, info) -> List[ServiceHealth]:
        """
        Get health status of all data source services.
        Used for system monitoring and circuit breaker decisions.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.field
    async def get_patient_context_summary(
        self,
        info,
        patient_id: str,
        include_cache_info: bool = False
    ) -> strawberry.scalars.JSON:
        """
        Get summary of all available context for a patient.
        Useful for debugging and clinical review.
        """
        # This will be implemented in the resolver
        pass


@strawberry.type
class Mutation:
    """
    GraphQL Mutation root for Clinical Context Service.
    Handles context invalidation and recipe management.
    """
    
    @strawberry.mutation
    async def invalidate_patient_context(
        self,
        info,
        patient_id: str,
        recipe_ids: Optional[List[str]] = None
    ) -> bool:
        """
        Invalidate cached context for a patient.
        Used when patient data changes.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.mutation
    async def invalidate_context_cache(
        self,
        info,
        cache_key: str
    ) -> bool:
        """
        Invalidate specific cache entry.
        Used for targeted cache invalidation.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.mutation
    async def warm_context_cache(
        self,
        info,
        patient_ids: List[str],
        recipe_ids: List[str]
    ) -> bool:
        """
        Warm cache with context for specified patients and recipes.
        Used for predictive caching.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.mutation
    async def submit_recipe_for_approval(
        self,
        info,
        recipe_data: strawberry.scalars.JSON,
        requested_by: str,
        justification: str,
        priority: str = "normal"
    ) -> str:
        """
        Submit a new recipe for Clinical Governance Board approval.
        Returns approval request ID.
        """
        # This will be implemented in the resolver
        pass


@strawberry.type
class Subscription:
    """
    GraphQL Subscription root for Clinical Context Service.
    Provides real-time updates for context changes.
    """
    
    @strawberry.subscription
    async def context_updates(
        self,
        info,
        patient_id: str,
        recipe_ids: Optional[List[str]] = None
    ) -> strawberry.scalars.JSON:
        """
        Subscribe to context updates for a patient.
        Used for real-time context synchronization.
        """
        # This will be implemented in the resolver
        pass
    
    @strawberry.subscription
    async def cache_invalidation_events(self, info) -> strawberry.scalars.JSON:
        """
        Subscribe to cache invalidation events.
        Used for distributed cache synchronization.
        """
        # This will be implemented in the resolver
        pass


# Create the GraphQL schema
schema = strawberry.Schema(
    query=Query,
    mutation=Mutation,
    subscription=Subscription
)
