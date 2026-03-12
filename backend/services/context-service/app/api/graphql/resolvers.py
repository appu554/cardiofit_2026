"""
GraphQL Resolvers for Clinical Context Service
Implements the business logic for GraphQL queries, mutations, and subscriptions
"""
import logging
from typing import List, Optional, Dict, Any
from datetime import datetime
import asyncio

from app.services.context_assembly_service import ContextAssemblyService
from app.services.recipe_management_service import RecipeManagementService
from app.services.cache_service import CacheService
from app.services.recipe_governance import RecipeGovernance
from app.models.context_models import (
    ClinicalContext, ContextRecipe, ClinicalDataError, RecipeValidationError
)
from app.api.graphql.schema import (
    ClinicalContext as GQLClinicalContext,
    ContextFieldsResponse, ContextAvailabilityResponse, RecipeInfo,
    CacheStats, ServiceHealth, SafetyFlag as GQLSafetyFlag,
    SourceMetadata as GQLSourceMetadata, ConnectionError as GQLConnectionError,
    ContextStatusEnum, SafetyFlagTypeEnum, SafetySeverityEnum, DataSourceTypeEnum
)

logger = logging.getLogger(__name__)


class ContextResolver:
    """
    Resolver class for clinical context GraphQL operations.
    Orchestrates context assembly, recipe management, and caching.
    """
    
    def __init__(self):
        self.context_assembly_service = ContextAssemblyService()
        self.recipe_management_service = RecipeManagementService()
        self.cache_service = CacheService()
        self.governance = RecipeGovernance()
    
    async def get_context_by_recipe(
        self,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None,
        force_refresh: bool = False,
        workflow_id: Optional[str] = None
    ) -> GQLClinicalContext:
        """
        Primary GraphQL resolver for recipe-based context assembly.
        Implements the core context retrieval workflow.
        """
        try:
            logger.info(f"📊 GraphQL getContextByRecipe request:")
            logger.info(f"   Patient: {patient_id}")
            logger.info(f"   Recipe: {recipe_id}")
            logger.info(f"   Provider: {provider_id}")
            logger.info(f"   Workflow: {workflow_id}")
            
            # Load recipe
            recipe = await self.recipe_management_service.load_recipe(recipe_id)
            
            # Validate recipe governance
            if not recipe.validate_governance():
                raise RecipeValidationError(f"Recipe {recipe_id} not approved by Clinical Governance Board")
            
            # Assemble clinical context
            clinical_context = await self.context_assembly_service.assemble_context(
                patient_id=patient_id,
                recipe=recipe,
                provider_id=provider_id,
                encounter_id=encounter_id,
                force_refresh=force_refresh
            )
            
            # Convert to GraphQL response
            gql_context = await self._convert_to_graphql_context(clinical_context)
            
            logger.info(f"✅ Context assembled successfully")
            logger.info(f"   Context ID: {gql_context.context_id}")
            logger.info(f"   Completeness: {gql_context.completeness_score:.2%}")
            logger.info(f"   Safety Flags: {len(gql_context.safety_flags)}")
            
            return gql_context
            
        except Exception as e:
            logger.error(f"❌ Context assembly failed: {e}")
            # Return error context
            return await self._create_error_context(patient_id, recipe_id, str(e))
    
    async def get_context_fields(
        self,
        patient_id: str,
        fields: List[str],
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None,
        max_age_hours: int = 24
    ) -> ContextFieldsResponse:
        """
        Resolver for field-specific context queries.
        Optimized for targeted data retrieval.
        """
        try:
            logger.info(f"📊 Field-specific context request for patient {patient_id}: {fields}")
            
            # Create a dynamic recipe for the requested fields
            dynamic_recipe = await self._create_dynamic_recipe_for_fields(fields, max_age_hours)
            
            # Assemble context using dynamic recipe
            clinical_context = await self.context_assembly_service.assemble_context(
                patient_id=patient_id,
                recipe=dynamic_recipe,
                provider_id=provider_id,
                encounter_id=encounter_id,
                force_refresh=False
            )
            
            # Extract only requested fields
            field_data = {}
            for field in fields:
                if field in clinical_context.assembled_data:
                    field_data[field] = clinical_context.assembled_data[field]
            
            return ContextFieldsResponse(
                data=field_data,
                completeness=clinical_context.completeness_score,
                metadata=clinical_context.source_metadata,
                status=self._convert_context_status(clinical_context.status)
            )
            
        except Exception as e:
            logger.error(f"❌ Field-specific context assembly failed: {e}")
            return ContextFieldsResponse(
                data={},
                completeness=0.0,
                metadata={},
                status=ContextStatusEnum.FAILED
            )
    
    async def validate_context_availability(
        self,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None
    ) -> ContextAvailabilityResponse:
        """
        Resolver for context availability validation.
        Provides pre-flight checks for workflow orchestration.
        """
        try:
            logger.info(f"🔍 Validating context availability for patient {patient_id}, recipe {recipe_id}")

            # Try to load recipe (may fail due to governance)
            try:
                recipe = await self.recipe_management_service.load_recipe(recipe_id)
            except Exception as recipe_error:
                logger.warning(f"⚠️ Recipe {recipe_id} failed to load: {recipe_error}")
                # Return availability response indicating recipe is not available
                return ContextAvailabilityResponse(
                    available=False,
                    recipe_id=recipe_id,
                    patient_id=patient_id,
                    estimated_completeness=0.0,
                    unavailable_sources=[f"recipe_governance_error: {str(recipe_error)}"],
                    estimated_assembly_time_ms=0,
                    cache_available=False
                )

            # Check cache availability
            cache_key = recipe.get_cache_key(patient_id, provider_id)
            cached_context = await self.cache_service.get(cache_key)
            cache_available = cached_context is not None

            # Estimate completeness based on data source health
            service_health = await self._check_data_source_health(recipe)
            available_sources = [ds for ds, healthy in service_health.items() if healthy]
            unavailable_sources = [ds for ds, healthy in service_health.items() if not healthy]

            estimated_completeness = len(available_sources) / len(service_health) if service_health else 0.0

            # Estimate assembly time based on recipe SLA and source health
            estimated_assembly_time = recipe.sla_ms
            if len(unavailable_sources) > 0:
                estimated_assembly_time *= 1.5  # Increase estimate for unhealthy sources

            return ContextAvailabilityResponse(
                available=len(unavailable_sources) == 0,
                recipe_id=recipe_id,
                patient_id=patient_id,
                estimated_completeness=estimated_completeness,
                unavailable_sources=unavailable_sources,
                estimated_assembly_time_ms=int(estimated_assembly_time),
                cache_available=cache_available
            )
            
        except Exception as e:
            logger.error(f"❌ Context availability validation failed: {e}")
            return ContextAvailabilityResponse(
                available=False,
                recipe_id=recipe_id,
                patient_id=patient_id,
                estimated_completeness=0.0,
                unavailable_sources=["unknown"],
                estimated_assembly_time_ms=0,
                cache_available=False
            )
    
    async def get_available_recipes(
        self,
        clinical_scenario: Optional[str] = None,
        workflow_category: Optional[str] = None
    ) -> List[RecipeInfo]:
        """
        Resolver for getting available clinical context recipes.
        """
        try:
            if clinical_scenario:
                recipes = await self.recipe_management_service.get_applicable_recipes(
                    clinical_scenario, workflow_category
                )
            else:
                # Get all recipes (including non-approved ones for testing)
                recipes = []
                for recipe in self.recipe_management_service.loaded_recipes.values():
                    if not recipe.is_expired():
                        if workflow_category is None or recipe.workflow_category == workflow_category:
                            recipes.append(recipe)
            
            # Convert to GraphQL response
            recipe_infos = []
            for recipe in recipes:
                recipe_info = RecipeInfo(
                    recipe_id=recipe.recipe_id,
                    recipe_name=recipe.recipe_name,
                    version=recipe.version,
                    clinical_scenario=recipe.clinical_scenario,
                    workflow_category=recipe.workflow_category,
                    execution_pattern=recipe.execution_pattern,
                    sla_ms=recipe.sla_ms,
                    governance_approved=recipe.validate_governance(),
                    effective_date=recipe.governance_metadata.effective_date if recipe.governance_metadata else None,
                    expiry_date=recipe.governance_metadata.expiry_date if recipe.governance_metadata else None
                )
                recipe_infos.append(recipe_info)
            
            logger.info(f"📋 Found {len(recipe_infos)} available recipes")
            return recipe_infos
            
        except Exception as e:
            logger.error(f"❌ Failed to get available recipes: {e}")
            return []
    
    async def get_recipe_info(
        self,
        recipe_id: str,
        version: str = "latest"
    ) -> Optional[RecipeInfo]:
        """
        Resolver for getting detailed recipe information.
        """
        try:
            recipe = await self.recipe_management_service.load_recipe(recipe_id, version)
            
            return RecipeInfo(
                recipe_id=recipe.recipe_id,
                recipe_name=recipe.recipe_name,
                version=recipe.version,
                clinical_scenario=recipe.clinical_scenario,
                workflow_category=recipe.workflow_category,
                execution_pattern=recipe.execution_pattern,
                sla_ms=recipe.sla_ms,
                governance_approved=recipe.validate_governance(),
                effective_date=recipe.governance_metadata.effective_date if recipe.governance_metadata else None,
                expiry_date=recipe.governance_metadata.expiry_date if recipe.governance_metadata else None
            )
            
        except Exception as e:
            logger.error(f"❌ Failed to get recipe info for {recipe_id}: {e}")
            return None
    
    async def get_cache_stats(self) -> CacheStats:
        """
        Resolver for getting cache performance statistics.
        """
        try:
            stats = await self.cache_service.get_cache_stats()
            
            return CacheStats(
                total_entries=stats["l1_cache"]["entries"] + stats.get("l2_entries", 0),
                hit_ratio=stats["overall_hit_ratio"],
                l1_entries=stats["l1_cache"]["entries"],
                l2_entries=stats.get("l2_entries", 0),
                last_updated=datetime.utcnow(),
                performance_metrics=stats["performance"]
            )
            
        except Exception as e:
            logger.error(f"❌ Failed to get cache stats: {e}")
            return CacheStats(
                total_entries=0,
                hit_ratio=0.0,
                l1_entries=0,
                l2_entries=0,
                last_updated=datetime.utcnow(),
                performance_metrics={}
            )
    
    async def get_service_health(self) -> List[ServiceHealth]:
        """
        Resolver for getting health status of data source services.
        """
        try:
            # This would check health of all configured data sources
            # For now, return mock health data
            health_statuses = []
            
            from app.models.data_source_types import get_all_data_source_types, get_data_source_config
            
            for source_type in get_all_data_source_types():
                config = get_data_source_config(source_type)
                if config:
                    # Mock health check - in production would do actual health checks
                    health_status = ServiceHealth(
                        service_type=self._convert_data_source_type(source_type),
                        endpoint=config.endpoint,
                        healthy=True,  # Mock as healthy
                        response_time_ms=50,  # Mock response time
                        last_check=datetime.utcnow(),
                        error_message=None
                    )
                    health_statuses.append(health_status)
            
            return health_statuses
            
        except Exception as e:
            logger.error(f"❌ Failed to get service health: {e}")
            return []
    
    async def invalidate_patient_context(
        self,
        patient_id: str,
        recipe_ids: Optional[List[str]] = None
    ) -> bool:
        """
        Mutation resolver for invalidating patient context cache.
        """
        try:
            if recipe_ids:
                # Invalidate specific recipe contexts
                for recipe_id in recipe_ids:
                    recipe = await self.recipe_management_service.load_recipe(recipe_id)
                    cache_key = recipe.get_cache_key(patient_id)
                    await self.cache_service.invalidate(cache_key)
            else:
                # Invalidate all contexts for patient
                await self.cache_service.invalidate_patient_contexts(patient_id)
            
            logger.info(f"✅ Patient context invalidated: {patient_id}")
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to invalidate patient context: {e}")
            return False
    
    async def _convert_to_graphql_context(self, context: ClinicalContext) -> GQLClinicalContext:
        """Convert internal ClinicalContext to GraphQL ClinicalContext"""

        # Convert safety flags
        gql_safety_flags = []
        for flag in context.safety_flags:
            gql_flag = GQLSafetyFlag(
                flag_type=SafetyFlagTypeEnum(flag.flag_type.value),
                severity=SafetySeverityEnum(flag.severity.value),
                message=flag.message,
                data_point=flag.data_point,
                details=flag.details,
                timestamp=flag.timestamp
            )
            gql_safety_flags.append(gql_flag)

        # Convert connection errors
        gql_connection_errors = []
        for error in context.connection_errors:
            gql_error = GQLConnectionError(
                data_point=error["data_point"],
                source=error["source"],
                error=error["error"],
                timestamp=datetime.utcnow()  # Would use actual timestamp from error
            )
            gql_connection_errors.append(gql_error)

        # Serialize data_freshness to ensure datetime objects are converted to strings
        serialized_data_freshness = self._serialize_datetime_dict(context.data_freshness)

        # Serialize source_metadata to ensure datetime objects are converted to strings
        serialized_source_metadata = self._serialize_datetime_dict(context.source_metadata)

        # Serialize assembled_data to ensure datetime objects are converted to strings
        serialized_assembled_data = self._serialize_datetime_dict(context.assembled_data)

        return GQLClinicalContext(
            context_id=context.context_id,
            patient_id=context.patient_id,
            recipe_used=context.recipe_used,
            assembled_data=serialized_assembled_data,
            completeness_score=context.completeness_score,
            data_freshness=serialized_data_freshness,
            source_metadata=serialized_source_metadata,
            safety_flags=gql_safety_flags,
            governance_tags=context.governance_tags,
            provider_id=context.provider_id,
            encounter_id=context.encounter_id,
            assembled_at=context.assembled_at,
            assembly_duration_ms=context.assembly_duration_ms,
            status=self._convert_context_status(context.status),
            connection_errors=gql_connection_errors,
            cache_hit=context.cache_hit,
            cache_key=context.cache_key,
            ttl_seconds=context.ttl_seconds
        )
    
    async def _create_error_context(
        self,
        patient_id: str,
        recipe_id: str,
        error_message: str
    ) -> GQLClinicalContext:
        """Create an error context response"""
        return GQLClinicalContext(
            context_id=f"error_{patient_id}_{recipe_id}",
            patient_id=patient_id,
            recipe_used=recipe_id,
            assembled_data={},
            completeness_score=0.0,
            data_freshness={},
            source_metadata={},
            safety_flags=[],
            governance_tags=[],
            assembled_at=datetime.utcnow(),
            assembly_duration_ms=0.0,
            status=ContextStatusEnum.FAILED,
            connection_errors=[
                GQLConnectionError(
                    data_point="system",
                    source="context_service",
                    error=error_message,
                    timestamp=datetime.utcnow()
                )
            ],
            cache_hit=False,
            cache_key="",
            ttl_seconds=0
        )
    
    def _convert_context_status(self, status) -> ContextStatusEnum:
        """Convert internal ContextStatus to GraphQL enum"""
        status_mapping = {
            "success": ContextStatusEnum.SUCCESS,
            "partial": ContextStatusEnum.PARTIAL,
            "failed": ContextStatusEnum.FAILED,
            "unavailable": ContextStatusEnum.UNAVAILABLE
        }
        return status_mapping.get(status.value if hasattr(status, 'value') else status, ContextStatusEnum.FAILED)

    def _serialize_datetime_dict(self, data: Any) -> Any:
        """
        Recursively serialize datetime objects and complex objects in dictionaries to JSON-serializable format.
        This ensures JSON serialization works properly.
        """
        from app.models.context_models import SourceMetadata

        if isinstance(data, dict):
            return {key: self._serialize_datetime_dict(value) for key, value in data.items()}
        elif isinstance(data, list):
            return [self._serialize_datetime_dict(item) for item in data]
        elif isinstance(data, SourceMetadata):
            # Convert SourceMetadata to dictionary
            return {
                "source_type": data.source_type.value,
                "source_endpoint": data.source_endpoint,
                "retrieved_at": data.retrieved_at.isoformat(),
                "data_version": data.data_version,
                "completeness": data.completeness,
                "response_time_ms": data.response_time_ms,
                "cache_hit": data.cache_hit,
                "error_message": data.error_message
            }
        elif isinstance(data, datetime):
            return data.isoformat()
        elif hasattr(data, 'isoformat'):  # Handle other datetime-like objects
            return data.isoformat()
        elif hasattr(data, 'value'):  # Handle Enum objects
            return data.value
        else:
            return data
    
    def _convert_data_source_type(self, source_type) -> DataSourceTypeEnum:
        """Convert internal DataSourceType to GraphQL enum"""
        return DataSourceTypeEnum(source_type.value)
    
    async def _create_dynamic_recipe_for_fields(self, fields: List[str], max_age_hours: int) -> ContextRecipe:
        """Create a dynamic recipe for field-specific queries"""
        # This would create a minimal recipe for the requested fields
        # Simplified implementation for now
        from app.models.context_models import DataPoint, DataSourceType
        
        data_points = []
        for field in fields:
            # Map fields to appropriate data sources (simplified)
            if field in ["age", "gender", "weight"]:
                source_type = DataSourceType.PATIENT_SERVICE
            elif field in ["medications", "prescriptions"]:
                source_type = DataSourceType.MEDICATION_SERVICE
            elif field in ["allergies"]:
                source_type = DataSourceType.ALLERGY_SERVICE
            else:
                source_type = DataSourceType.PATIENT_SERVICE
            
            data_point = DataPoint(
                name=field,
                source_type=source_type,
                fields=[field],
                required=False,
                max_age_hours=max_age_hours
            )
            data_points.append(data_point)
        
        # Create minimal recipe
        recipe = ContextRecipe(
            recipe_id="dynamic_fields",
            recipe_name="Dynamic Fields Recipe",
            version="1.0",
            clinical_scenario="field_query",
            workflow_category="query",
            execution_pattern="optimistic",
            required_data_points=data_points
        )
        
        return recipe
    
    async def _check_data_source_health(self, recipe: ContextRecipe) -> Dict[str, bool]:
        """Check health of data sources required by recipe"""
        health_status = {}
        
        for data_point in recipe.required_data_points:
            source_type = data_point.source_type.value
            # Mock health check - in production would do actual health checks
            health_status[source_type] = True
        
        return health_status


# Create global resolver instance
context_resolver = ContextResolver()
