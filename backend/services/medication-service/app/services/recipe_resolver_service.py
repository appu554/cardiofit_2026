"""
Recipe Resolver Service for Medication Service V2

This service implements the internal Recipe Resolver system that moved from external 
to internal ownership. It handles recipe resolution, field merging, conditional logic,
and caching with performance targets of <10ms resolution time.
"""

import asyncio
import hashlib
import json
import logging
import time
from typing import Dict, List, Optional, Set, Any, Tuple
from datetime import datetime, timedelta

from ..models.recipe import (
    RecipeTemplate, FieldRequirement, ResolutionContext, ResolvedField,
    RecipeResolutionResult, FieldConflict, RecipeCache, ConditionalRule,
    FieldType, RecipePhase, ResolutionStrategy, FreshnessRequirement,
    RecipeNotFoundError, FieldResolutionError, FreshnessViolationError
)
from ..repositories.recipe_repository import RecipeRepository
from ..infrastructure.caching import CacheManager
from ..infrastructure.context_client import ContextClient
from ..utils.performance_monitor import PerformanceMonitor


logger = logging.getLogger(__name__)


class RecipeResolverService:
    """
    Internal Recipe Resolver Service for Medication Service V2
    
    Key Features:
    - Recipe template management with protocol-specific implementations
    - Conditional rule engine for patient-specific field resolution
    - Field merging and deduplication with conflict resolution
    - Caching integration with configurable TTL and freshness validation
    - Performance optimization targeting <10ms resolution time
    """
    
    def __init__(
        self,
        recipe_repository: RecipeRepository,
        cache_manager: CacheManager,
        context_client: ContextClient,
        performance_monitor: PerformanceMonitor
    ):
        self.recipe_repository = recipe_repository
        self.cache_manager = cache_manager
        self.context_client = context_client
        self.performance_monitor = performance_monitor
        self.logger = logger
        
        # Performance configuration
        self.target_resolution_time_ms = 10.0
        self.cache_hit_target = 0.85
        
        # Resolution statistics
        self.resolution_stats = {
            'total_resolutions': 0,
            'cache_hits': 0,
            'average_resolution_time_ms': 0.0,
            'field_conflicts': 0,
            'freshness_violations': 0
        }
    
    async def resolve_recipe(
        self, 
        context: ResolutionContext,
        phases: Optional[List[RecipePhase]] = None
    ) -> RecipeResolutionResult:
        """
        Resolve recipe fields based on patient context and clinical scenario.
        
        Args:
            context: Resolution context with patient and medication information
            phases: Specific phases to resolve (default: all phases)
            
        Returns:
            RecipeResolutionResult with resolved fields and metadata
            
        Raises:
            RecipeNotFoundError: No matching recipe template found
            FieldResolutionError: Required fields could not be resolved
        """
        start_time = time.time()
        
        try:
            # Check cache first
            cache_key = self._generate_cache_key(context, phases)
            cached_result = await self._get_cached_result(cache_key)
            
            if cached_result and not context.force_refresh:
                self._update_resolution_stats(
                    time.time() - start_time, cache_hit=True
                )
                return cached_result
            
            # Find appropriate recipe template
            recipe_template = await self._select_recipe_template(context)
            if not recipe_template:
                raise RecipeNotFoundError(
                    f"No recipe template found for context: {context.dict()}"
                )
            
            # Resolve fields for requested phases
            phases = phases or list(RecipePhase)
            resolution_result = await self._resolve_fields(
                recipe_template, context, phases
            )
            
            # Cache the result
            await self._cache_result(cache_key, resolution_result, recipe_template)
            
            # Update performance statistics
            resolution_time_ms = (time.time() - start_time) * 1000
            self._update_resolution_stats(resolution_time_ms, cache_hit=False)
            
            # Log performance warning if target exceeded
            if resolution_time_ms > self.target_resolution_time_ms:
                self.logger.warning(
                    f"Recipe resolution time {resolution_time_ms:.2f}ms exceeds "
                    f"target {self.target_resolution_time_ms}ms"
                )
            
            return resolution_result
            
        except Exception as e:
            self.logger.error(f"Recipe resolution failed: {str(e)}")
            raise
    
    async def resolve_protocol_specific_recipe(
        self,
        protocol_id: str,
        context: ResolutionContext,
        phases: Optional[List[RecipePhase]] = None
    ) -> RecipeResolutionResult:
        """
        Resolve recipe for a specific clinical protocol.
        
        Args:
            protocol_id: Clinical protocol identifier (e.g., "hypertension-standard")
            context: Resolution context
            phases: Specific phases to resolve
            
        Returns:
            RecipeResolutionResult for the protocol
        """
        # Add protocol context to resolution context
        enhanced_context = context.copy()
        enhanced_context.patient_conditions.append(protocol_id)
        
        # Find protocol-specific recipe
        recipe_template = await self.recipe_repository.get_protocol_recipe(
            protocol_id
        )
        
        if not recipe_template:
            # Fall back to standard recipe with protocol context
            return await self.resolve_recipe(enhanced_context, phases)
        
        # Resolve using protocol-specific template
        return await self._resolve_fields(recipe_template, enhanced_context, phases)
    
    async def validate_recipe_completeness(
        self,
        result: RecipeResolutionResult,
        required_completeness: float = 0.9
    ) -> Tuple[bool, List[str]]:
        """
        Validate that recipe resolution meets completeness requirements.
        
        Args:
            result: Recipe resolution result to validate
            required_completeness: Minimum completeness score (0.0-1.0)
            
        Returns:
            Tuple of (is_valid, validation_errors)
        """
        errors = []
        
        # Check completeness score
        if result.completeness_score < required_completeness:
            errors.append(
                f"Completeness score {result.completeness_score:.2f} below "
                f"required {required_completeness:.2f}"
            )
        
        # Check missing required fields
        if result.missing_required_fields:
            errors.append(
                f"Missing required fields: {', '.join(result.missing_required_fields)}"
            )
        
        # Check freshness violations
        if result.freshness_violations:
            errors.append(
                f"Freshness violations: {', '.join(result.freshness_violations)}"
            )
        
        # Check validation errors
        if result.validation_errors:
            errors.extend(result.validation_errors)
        
        return len(errors) == 0, errors
    
    async def get_recipe_templates(
        self,
        clinical_scenario: Optional[str] = None
    ) -> List[RecipeTemplate]:
        """Get available recipe templates, optionally filtered by scenario."""
        return await self.recipe_repository.get_templates(clinical_scenario)
    
    async def register_recipe_template(
        self,
        template: RecipeTemplate
    ) -> RecipeTemplate:
        """Register a new recipe template."""
        return await self.recipe_repository.create_template(template)
    
    def get_resolution_statistics(self) -> Dict[str, Any]:
        """Get resolution performance statistics."""
        cache_hit_rate = (
            self.resolution_stats['cache_hits'] / 
            max(self.resolution_stats['total_resolutions'], 1)
        )
        
        return {
            **self.resolution_stats,
            'cache_hit_rate': cache_hit_rate,
            'target_resolution_time_ms': self.target_resolution_time_ms,
            'cache_hit_target': self.cache_hit_target,
            'performance_target_met': (
                self.resolution_stats['average_resolution_time_ms'] <= 
                self.target_resolution_time_ms
            ),
            'cache_target_met': cache_hit_rate >= self.cache_hit_target
        }
    
    async def _select_recipe_template(
        self, 
        context: ResolutionContext
    ) -> Optional[RecipeTemplate]:
        """
        Select the most appropriate recipe template based on context.
        
        Uses priority-based selection with conditional trigger evaluation.
        """
        # Get all available templates
        templates = await self.recipe_repository.get_templates()
        
        # Build context data for trigger evaluation
        context_data = self._build_context_data(context)
        
        # Find matching templates and sort by priority
        matching_templates = []
        for template in templates:
            if template.evaluate_triggers(context_data):
                matching_templates.append(template)
        
        if not matching_templates:
            # Try to find a default template
            default_template = await self.recipe_repository.get_default_template()
            return default_template
        
        # Sort by priority (higher priority first)
        matching_templates.sort(key=lambda t: t.priority, reverse=True)
        return matching_templates[0]
    
    async def _resolve_fields(
        self,
        recipe_template: RecipeTemplate,
        context: ResolutionContext,
        phases: List[RecipePhase]
    ) -> RecipeResolutionResult:
        """
        Resolve fields for the specified phases using the recipe template.
        """
        start_time = time.time()
        
        # Initialize result
        result = RecipeResolutionResult(
            recipe_id=recipe_template.recipe_id,
            recipe_version=recipe_template.version,
            resolution_context=context,
            resolution_time_ms=0.0,
            total_fields_requested=0,
            total_fields_resolved=0,
            completeness_score=0.0
        )
        
        # Build context data for conditional evaluation
        context_data = self._build_context_data(context)
        
        # Resolve fields for each phase
        for phase in phases:
            field_requirements = recipe_template.get_fields_for_phase(phase)
            
            # Filter fields based on conditional rules
            active_requirements = []
            for req in field_requirements:
                if req.is_conditionally_required(context_data):
                    active_requirements.append(req)
            
            result.total_fields_requested += len(active_requirements)
            
            # Resolve fields for this phase
            resolved_fields = await self._resolve_phase_fields(
                active_requirements, context, context_data
            )
            
            # Store resolved fields in appropriate phase
            phase_fields = result.get_fields_for_phase(phase)
            phase_fields.update(resolved_fields)
            result.total_fields_resolved += len(resolved_fields)
        
        # Calculate completeness score
        if result.total_fields_requested > 0:
            result.completeness_score = (
                result.total_fields_resolved / result.total_fields_requested
            )
        else:
            result.completeness_score = 1.0
        
        # Set resolution time
        result.resolution_time_ms = (time.time() - start_time) * 1000
        
        # Validate result
        await self._validate_resolution_result(result, recipe_template)
        
        return result
    
    async def _resolve_phase_fields(
        self,
        field_requirements: List[FieldRequirement],
        context: ResolutionContext,
        context_data: Dict[str, Any]
    ) -> Dict[str, ResolvedField]:
        """
        Resolve fields for a specific phase with conflict resolution.
        """
        resolved_fields = {}
        conflicts = []
        
        # Group requirements by field path for conflict detection
        field_groups = {}
        for req in field_requirements:
            if req.field_path not in field_groups:
                field_groups[req.field_path] = []
            field_groups[req.field_path].append(req)
        
        # Resolve each field group
        for field_path, requirements in field_groups.items():
            try:
                field_candidates = []
                
                # Get candidate values from different sources
                for req in requirements:
                    candidates = await self._get_field_candidates(
                        req, context, context_data
                    )
                    field_candidates.extend(candidates)
                
                if not field_candidates:
                    # Field could not be resolved
                    if any(req.required for req in requirements):
                        # Add to missing required fields if any requirement is required
                        # (will be populated in the result after this method returns)
                        pass
                    continue
                
                # Resolve conflicts if multiple candidates exist
                if len(field_candidates) > 1:
                    resolved_field, conflict = await self._resolve_field_conflict(
                        field_path, field_candidates, requirements[0].resolution_strategy
                        if requirements else ResolutionStrategy.MOST_RECENT
                    )
                    if conflict:
                        conflicts.append(conflict)
                else:
                    resolved_field = field_candidates[0]
                
                # Validate freshness
                freshness_valid = True
                for req in requirements:
                    if not req.validate_freshness(resolved_field.timestamp):
                        freshness_valid = False
                        break
                
                resolved_field.freshness_valid = freshness_valid
                resolved_fields[field_path] = resolved_field
                
            except Exception as e:
                self.logger.error(
                    f"Error resolving field {field_path}: {str(e)}"
                )
                continue
        
        return resolved_fields
    
    async def _get_field_candidates(
        self,
        requirement: FieldRequirement,
        context: ResolutionContext,
        context_data: Dict[str, Any]
    ) -> List[ResolvedField]:
        """
        Get candidate field values from various data sources.
        """
        candidates = []
        
        try:
            # Get data from context client based on field type
            if requirement.field_type == FieldType.DEMOGRAPHIC:
                data = await self.context_client.get_patient_demographics(
                    context.patient_id
                )
            elif requirement.field_type == FieldType.LABORATORY:
                data = await self.context_client.get_patient_labs(
                    context.patient_id
                )
            elif requirement.field_type == FieldType.VITAL:
                data = await self.context_client.get_patient_vitals(
                    context.patient_id
                )
            elif requirement.field_type == FieldType.MEDICATION:
                data = await self.context_client.get_patient_medications(
                    context.patient_id
                )
            elif requirement.field_type == FieldType.ALLERGY:
                data = await self.context_client.get_patient_allergies(
                    context.patient_id
                )
            elif requirement.field_type == FieldType.CONDITION:
                data = await self.context_client.get_patient_conditions(
                    context.patient_id
                )
            else:
                # Try generic context data
                data = context_data
            
            # Extract field value using path
            field_value = self._extract_field_value(data, requirement.field_path)
            
            if field_value is not None:
                candidate = ResolvedField(
                    field_path=requirement.field_path,
                    value=field_value,
                    field_type=requirement.field_type,
                    source=f"{requirement.field_type.value}_service",
                    timestamp=datetime.utcnow(),  # This should come from data source
                    freshness_valid=True,  # Will be validated later
                    priority=requirement.priority
                )
                candidates.append(candidate)
                
        except Exception as e:
            self.logger.error(
                f"Error getting candidates for {requirement.field_path}: {str(e)}"
            )
        
        return candidates
    
    async def _resolve_field_conflict(
        self,
        field_path: str,
        candidates: List[ResolvedField],
        strategy: ResolutionStrategy
    ) -> Tuple[ResolvedField, Optional[FieldConflict]]:
        """
        Resolve conflicts between multiple field candidates.
        """
        if len(candidates) == 1:
            return candidates[0], None
        
        # Create conflict record
        conflict = FieldConflict(
            field_path=field_path,
            conflicting_values=candidates,
            resolution_strategy=strategy
        )
        
        # Apply resolution strategy
        if strategy == ResolutionStrategy.MOST_RECENT:
            resolved = max(candidates, key=lambda c: c.timestamp)
        elif strategy == ResolutionStrategy.HIGHEST_PRIORITY:
            resolved = max(candidates, key=lambda c: c.priority)
        elif strategy == ResolutionStrategy.FAIL_ON_CONFLICT:
            raise FieldResolutionError(
                f"Conflict resolution failed for {field_path}: "
                f"multiple values found and FAIL_ON_CONFLICT strategy specified"
            )
        else:
            # Default to most recent
            resolved = max(candidates, key=lambda c: c.timestamp)
        
        conflict.resolved_value = resolved
        return resolved, conflict
    
    def _build_context_data(self, context: ResolutionContext) -> Dict[str, Any]:
        """
        Build context data dictionary for conditional rule evaluation.
        """
        return {
            'patient': {
                'id': context.patient_id,
                'age': context.patient_age,
                'weight_kg': context.patient_weight_kg,
                'pregnancy_status': context.patient_pregnancy_status,
                'renal_function': context.patient_renal_function,
                'conditions': context.patient_conditions,
                'allergies': context.patient_allergies
            },
            'medication': {
                'code': context.medication_code,
                'name': context.medication_name,
                'indication': context.indication
            },
            'request': {
                'priority': context.priority,
                'provider_id': context.provider_id,
                'encounter_id': context.encounter_id
            }
        }
    
    def _extract_field_value(self, data: Dict[str, Any], field_path: str) -> Any:
        """
        Extract field value from nested data using dot notation.
        """
        try:
            keys = field_path.split('.')
            value = data
            for key in keys:
                if isinstance(value, dict):
                    value = value.get(key)
                elif isinstance(value, list) and key.isdigit():
                    value = value[int(key)]
                else:
                    return None
            return value
        except (KeyError, IndexError, TypeError, ValueError):
            return None
    
    async def _validate_resolution_result(
        self,
        result: RecipeResolutionResult,
        recipe_template: RecipeTemplate
    ):
        """
        Validate the resolution result and populate validation errors.
        """
        # Check for missing required fields
        all_requirements = recipe_template.get_all_fields()
        context_data = self._build_context_data(result.resolution_context)
        
        for req in all_requirements:
            if req.is_conditionally_required(context_data):
                if req.field_path not in result.get_all_resolved_fields():
                    result.missing_required_fields.append(req.field_path)
        
        # Check freshness violations
        for field_path, resolved_field in result.get_all_resolved_fields().items():
            if not resolved_field.freshness_valid:
                result.freshness_violations.append(field_path)
        
        # Set overall validation status
        result.validation_passed = (
            len(result.missing_required_fields) == 0 and
            len(result.freshness_violations) == 0 and
            len(result.validation_errors) == 0
        )
    
    def _generate_cache_key(
        self,
        context: ResolutionContext,
        phases: Optional[List[RecipePhase]]
    ) -> str:
        """
        Generate cache key for resolution context.
        """
        # Create deterministic key from context and phases
        cache_data = {
            'patient_id': context.patient_id,
            'medication_code': context.medication_code,
            'medication_name': context.medication_name,
            'indication': context.indication,
            'patient_age': context.patient_age,
            'patient_weight_kg': context.patient_weight_kg,
            'patient_pregnancy_status': context.patient_pregnancy_status,
            'patient_renal_function': context.patient_renal_function,
            'patient_conditions': sorted(context.patient_conditions),
            'patient_allergies': sorted(context.patient_allergies),
            'phases': sorted([p.value for p in (phases or list(RecipePhase))])
        }
        
        # Generate SHA-256 hash
        cache_json = json.dumps(cache_data, sort_keys=True)
        return hashlib.sha256(cache_json.encode()).hexdigest()
    
    async def _get_cached_result(self, cache_key: str) -> Optional[RecipeResolutionResult]:
        """Get cached resolution result if available and valid."""
        try:
            cached_data = await self.cache_manager.get(f"recipe_resolution:{cache_key}")
            if cached_data:
                cached_result = RecipeResolutionResult.parse_obj(cached_data)
                return cached_result
        except Exception as e:
            self.logger.warning(f"Cache retrieval failed: {str(e)}")
        
        return None
    
    async def _cache_result(
        self,
        cache_key: str,
        result: RecipeResolutionResult,
        recipe_template: RecipeTemplate
    ):
        """Cache resolution result with appropriate TTL."""
        try:
            await self.cache_manager.set(
                f"recipe_resolution:{cache_key}",
                result.dict(),
                ttl_seconds=recipe_template.cache_ttl_seconds
            )
        except Exception as e:
            self.logger.warning(f"Cache storage failed: {str(e)}")
    
    def _update_resolution_stats(self, resolution_time_ms: float, cache_hit: bool):
        """Update resolution performance statistics."""
        self.resolution_stats['total_resolutions'] += 1
        
        if cache_hit:
            self.resolution_stats['cache_hits'] += 1
        
        # Update rolling average resolution time
        total = self.resolution_stats['total_resolutions']
        current_avg = self.resolution_stats['average_resolution_time_ms']
        self.resolution_stats['average_resolution_time_ms'] = (
            (current_avg * (total - 1) + resolution_time_ms) / total
        )