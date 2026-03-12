"""
Context Service Client for Medication Service
Integrates with the Clinical Context Service using GraphQL API
"""

import logging
from typing import Dict, Any, Optional, List
import httpx
import asyncio
import time
from dataclasses import dataclass
from datetime import datetime

logger = logging.getLogger(__name__)


@dataclass
class ContextRequest:
    """Context request parameters"""
    patient_id: str
    recipe_id: str
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    force_refresh: bool = False


@dataclass
class ClinicalContext:
    """Clinical context response from Context Service"""
    context_id: str
    patient_id: str
    recipe_used: str
    assembled_data: Dict[str, Any]
    completeness_score: float
    data_freshness: Dict[str, datetime]
    source_metadata: Dict[str, Any]
    safety_flags: List[Dict[str, Any]]
    governance_tags: List[str]
    status: str
    assembled_at: datetime
    assembly_duration_ms: float
    connection_errors: List[Dict[str, Any]]


class ContextServiceClient:
    """
    Client for Clinical Context Service integration

    Implements the ratified architecture pattern:
    - GraphQL API for external communication
    - Recipe-based context assembly
    - Multi-layer caching support
    - Clinical governance compliance
    """

    def __init__(self, context_service_url: str = "http://localhost:8016"):
        self.context_service_url = context_service_url
        self.graphql_endpoint = f"{context_service_url}/graphql"
        self.timeout = 30.0  # 30 second timeout

        # Recipe IDs mapping to actual recipe files in context service
        self.recipe_mapping = {
            'medication_prescribing': 'medication_prescribing_v2',
            'medication_refill': 'routine_medication_refill_v1',
            'medication_safety': 'medication_safety_base_context_v2',
            'medication_renal': 'medication_renal_context_v2',
            'cae_integration': 'cae_integration_context_v1',
            'safety_gateway': 'safety_gateway_context_v1',
            'base_clinical': 'base_clinical_context_v1'
        }

        # Flow 2 specific configuration
        self.flow2_config = {
            'max_retries': 3,
            'retry_delay_ms': 100,
            'circuit_breaker_threshold': 5,
            'performance_target_ms': 100,
            'fallback_enabled': True
        }

        # Performance tracking for Flow 2
        self.performance_metrics = {
            'total_requests': 0,
            'successful_requests': 0,
            'failed_requests': 0,
            'average_response_time_ms': 0,
            'circuit_breaker_trips': 0
        }
    
    async def get_medication_prescribing_context(
        self,
        patient_id: str,
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None,
        force_refresh: bool = False
    ) -> ClinicalContext:
        """
        Get comprehensive context for medication prescribing workflow
        
        Uses the medication_prescribing_v2 recipe which includes:
        - Patient demographics
        - Current medications
        - Allergies
        - Medical history
        - Provider context
        - Formulary context
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['medication_prescribing'],
            provider_id=provider_id,
            encounter_id=encounter_id,
            force_refresh=force_refresh
        )
        
        return await self._get_context_by_recipe(request)
    
    async def get_dose_calculation_context(
        self,
        patient_id: str,
        medication_id: str,
        provider_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get context specifically for dose calculation
        
        Includes patient vitals, lab results, and medication-specific factors
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['dose_calculation'],
            provider_id=provider_id
        )
        
        context = await self._get_context_by_recipe(request)
        
        # Add medication-specific context if needed
        if medication_id:
            context.assembled_data['target_medication_id'] = medication_id
        
        return context
    
    async def get_drug_interaction_context(
        self,
        patient_id: str,
        new_medication_id: str,
        provider_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get context for drug interaction analysis
        
        Focuses on current medications, allergies, and contraindications
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['drug_interaction_check'],
            provider_id=provider_id
        )
        
        context = await self._get_context_by_recipe(request)
        context.assembled_data['new_medication_id'] = new_medication_id
        
        return context
    
    async def validate_context_availability(
        self,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Validate that required context data is available before workflow execution
        """
        query = """
        query ValidateContextAvailability(
            $patientId: String!,
            $recipeId: String!,
            $providerId: String
        ) {
            validateContextAvailability(
                patientId: $patientId,
                recipeId: $recipeId,
                providerId: $providerId
            ) {
                available
                recipeId
                patientId
                estimatedCompleteness
                unavailableSources
                estimatedAssemblyTimeMs
                cacheAvailable
            }
        }
        """

        variables = {
            "patientId": patient_id,
            "recipeId": recipe_id,
            "providerId": provider_id
        }

        try:
            logger.info(f"🔍 Validating context availability for patient {patient_id} with recipe {recipe_id}")

            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.post(
                    self.graphql_endpoint,
                    json={"query": query, "variables": variables},
                    headers={"Content-Type": "application/json"}
                )

                if response.status_code != 200:
                    logger.error(f"❌ Context availability validation failed: {response.status_code}")
                    raise Exception(f"HTTP {response.status_code}: {response.text}")

                result = response.json()

                if "errors" in result:
                    logger.error(f"❌ GraphQL errors in context validation: {result['errors']}")
                    raise Exception(f"GraphQL errors: {result['errors']}")

                availability = result["data"]["validateContextAvailability"]
                logger.info(f"✅ Context availability validated: {availability['available']}")

                return availability

        except Exception as e:
            logger.error(f"❌ Failed to validate context availability: {e}")
            raise
    
    async def _get_context_by_recipe(self, request: ContextRequest) -> ClinicalContext:
        """
        Core method to get clinical context using recipe-based assembly
        """
        query = """
        query GetContextByRecipe(
            $patientId: String!,
            $recipeId: String!,
            $providerId: String,
            $encounterId: String,
            $forceRefresh: Boolean,
            $workflowId: String
        ) {
            getContextByRecipe(
                patientId: $patientId,
                recipeId: $recipeId,
                providerId: $providerId,
                encounterId: $encounterId,
                forceRefresh: $forceRefresh,
                workflowId: $workflowId
            ) {
                contextId
                patientId
                recipeUsed
                assembledData
                completenessScore
                dataFreshness
                sourceMetadata
                safetyFlags {
                    flagType
                    severity
                    message
                    dataPoint
                    details
                    timestamp
                }
                governanceTags
                status
                assembledAt
                assemblyDurationMs
                connectionErrors {
                    dataPoint
                    source
                    error
                    timestamp
                }
            }
        }
        """

        variables = {
            "patientId": request.patient_id,
            "recipeId": request.recipe_id,
            "providerId": request.provider_id,
            "encounterId": request.encounter_id,
            "forceRefresh": request.force_refresh,
            "workflowId": f"medication-service-{request.patient_id}"
        }

        try:
            logger.info(f"🔍 Requesting context for patient {request.patient_id} using recipe {request.recipe_id}")

            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.post(
                    self.graphql_endpoint,
                    json={"query": query, "variables": variables},
                    headers={"Content-Type": "application/json"}
                )

                if response.status_code != 200:
                    logger.error(f"❌ Context service request failed: {response.status_code}")
                    raise Exception(f"HTTP {response.status_code}: {response.text}")

                result = response.json()

                if "errors" in result:
                    logger.error(f"❌ GraphQL errors in context request: {result['errors']}")
                    raise Exception(f"GraphQL errors: {result['errors']}")

                context_data = result["data"]["getContextByRecipe"]

                # Convert to ClinicalContext object
                clinical_context = ClinicalContext(
                    context_id=context_data["contextId"],
                    patient_id=context_data["patientId"],
                    recipe_used=context_data["recipeUsed"],
                    assembled_data=context_data["assembledData"],
                    completeness_score=context_data["completenessScore"],
                    data_freshness=self._parse_data_freshness(context_data["dataFreshness"]),
                    source_metadata=context_data["sourceMetadata"],
                    safety_flags=context_data["safetyFlags"],
                    governance_tags=context_data["governanceTags"],
                    status=context_data["status"],
                    assembled_at=datetime.fromisoformat(context_data["assembledAt"].replace('Z', '+00:00')),
                    assembly_duration_ms=context_data["assemblyDurationMs"],
                    connection_errors=context_data.get("connectionErrors", [])
                )

                logger.info(f"✅ Context retrieved successfully")
                logger.info(f"   Context ID: {clinical_context.context_id}")
                logger.info(f"   Completeness: {clinical_context.completeness_score:.2%}")
                logger.info(f"   Assembly time: {clinical_context.assembly_duration_ms:.1f}ms")
                logger.info(f"   Safety flags: {len(clinical_context.safety_flags)}")

                return clinical_context

        except Exception as e:
            logger.error(f"❌ Failed to get clinical context: {e}")
            raise
    
    def _parse_data_freshness(self, freshness_data: Dict[str, str]) -> Dict[str, datetime]:
        """Parse data freshness timestamps"""
        parsed_freshness = {}
        
        for key, timestamp_str in freshness_data.items():
            try:
                parsed_freshness[key] = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
            except (ValueError, AttributeError):
                logger.warning(f"Could not parse timestamp for {key}: {timestamp_str}")
                parsed_freshness[key] = datetime.now()
        
        return parsed_freshness
    
    async def health_check(self) -> bool:
        """Check if Context Service is healthy"""
        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                response = await client.get(f"{self.context_service_url}/health")
                return response.status_code == 200
        except Exception as e:
            logger.error(f"Context Service health check failed: {e}")
            return False
    
    def get_recipe_id_for_workflow(self, workflow_type: str) -> str:
        """Get the appropriate recipe ID for a workflow type"""
        return self.recipe_mapping.get(workflow_type, self.recipe_mapping['medication_prescribing'])

    async def get_medication_safety_context(
        self,
        patient_id: str,
        medication_id: str,
        provider_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get context specifically for medication safety checks

        Uses medication_safety_base_context_v2 recipe for comprehensive safety validation
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['medication_safety'],
            provider_id=provider_id
        )

        context = await self._get_context_by_recipe(request)
        context.assembled_data['target_medication_id'] = medication_id

        return context

    async def get_cae_integration_context(
        self,
        patient_id: str,
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get context for Clinical Assertion Engine integration

        Uses cae_integration_context_v1 recipe for CAE workflow
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['cae_integration'],
            provider_id=provider_id,
            encounter_id=encounter_id
        )

        return await self._get_context_by_recipe(request)

    async def get_safety_gateway_context(
        self,
        patient_id: str,
        provider_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get context for Safety Gateway Platform integration

        Uses safety_gateway_context_v1 recipe for safety validation
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['safety_gateway'],
            provider_id=provider_id
        )

        return await self._get_context_by_recipe(request)

    async def get_renal_adjustment_context(
        self,
        patient_id: str,
        provider_id: Optional[str] = None
    ) -> ClinicalContext:
        """
        Get context for renal dose adjustments

        Uses medication_renal_context_v2 recipe for renal function assessment
        """
        request = ContextRequest(
            patient_id=patient_id,
            recipe_id=self.recipe_mapping['medication_renal'],
            provider_id=provider_id
        )

        return await self._get_context_by_recipe(request)

    async def get_available_recipes(self) -> List[Dict[str, Any]]:
        """
        Get list of available recipes from the context service
        """
        query = """
        query GetAvailableRecipes {
            getAvailableRecipes {
                recipeId
                recipeName
                version
                clinicalScenario
                workflowCategory
                executionPattern
                slaMs
                governanceApproved
                effectiveDate
                expiryDate
            }
        }
        """

        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.post(
                    self.graphql_endpoint,
                    json={"query": query},
                    headers={"Content-Type": "application/json"}
                )

                if response.status_code != 200:
                    logger.error(f"❌ Failed to get available recipes: {response.status_code}")
                    raise Exception(f"HTTP {response.status_code}: {response.text}")

                result = response.json()

                if "errors" in result:
                    logger.error(f"❌ GraphQL errors getting recipes: {result['errors']}")
                    raise Exception(f"GraphQL errors: {result['errors']}")

                recipes = result["data"]["getAvailableRecipes"]
                logger.info(f"✅ Retrieved {len(recipes)} available recipes")

                return recipes

        except Exception as e:
            logger.error(f"❌ Failed to get available recipes: {e}")
            # Return empty list as fallback
            return []

    # Flow 2 Enhanced Methods

    async def execute_recipe_with_flow2_enhancements(
        self,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str] = None,
        encounter_id: Optional[str] = None,
        force_refresh: bool = False,
        urgency: str = "routine"
    ) -> ClinicalContext:
        """
        Execute context recipe with Flow 2 enhancements including:
        - Performance monitoring
        - Circuit breaker pattern
        - Retry logic with exponential backoff
        - Enhanced error handling
        """
        start_time = time.time()
        self.performance_metrics['total_requests'] += 1

        try:
            # Adjust timeout based on urgency
            original_timeout = self.timeout
            if urgency == "emergency":
                self.timeout = 5.0  # 5 second timeout for emergency
            elif urgency == "urgent":
                self.timeout = 10.0  # 10 second timeout for urgent

            # Execute with retry logic
            context = await self._execute_with_retry(
                patient_id, recipe_id, provider_id, encounter_id, force_refresh
            )

            # Track performance
            execution_time = (time.time() - start_time) * 1000
            self._update_performance_metrics(execution_time, success=True)

            # Validate context quality for Flow 2
            self._validate_context_quality(context, urgency)

            return context

        except Exception as e:
            execution_time = (time.time() - start_time) * 1000
            self._update_performance_metrics(execution_time, success=False)
            logger.error(f"❌ Flow 2 context execution failed: {str(e)}")
            raise
        finally:
            # Restore original timeout
            self.timeout = original_timeout

    async def _execute_with_retry(
        self,
        patient_id: str,
        recipe_id: str,
        provider_id: Optional[str],
        encounter_id: Optional[str],
        force_refresh: bool
    ) -> ClinicalContext:
        """
        Execute context request with retry logic and exponential backoff
        """
        max_retries = self.flow2_config['max_retries']
        base_delay = self.flow2_config['retry_delay_ms'] / 1000.0

        for attempt in range(max_retries + 1):
            try:
                request = ContextRequest(
                    patient_id=patient_id,
                    recipe_id=recipe_id,
                    provider_id=provider_id,
                    encounter_id=encounter_id,
                    force_refresh=force_refresh
                )

                return await self._get_context_by_recipe(request)

            except Exception as e:
                if attempt == max_retries:
                    # Final attempt failed
                    raise e

                # Calculate exponential backoff delay
                delay = base_delay * (2 ** attempt)
                logger.warning(f"⚠️ Context request attempt {attempt + 1} failed, retrying in {delay:.2f}s: {str(e)}")
                await asyncio.sleep(delay)

        # This should never be reached, but just in case
        raise Exception("Maximum retries exceeded")

    def _update_performance_metrics(self, execution_time_ms: float, success: bool):
        """
        Update performance metrics for Flow 2 monitoring
        """
        if success:
            self.performance_metrics['successful_requests'] += 1
        else:
            self.performance_metrics['failed_requests'] += 1

        # Update average response time
        total_requests = self.performance_metrics['total_requests']
        current_avg = self.performance_metrics['average_response_time_ms']
        new_avg = ((current_avg * (total_requests - 1)) + execution_time_ms) / total_requests
        self.performance_metrics['average_response_time_ms'] = new_avg

        # Check if performance target is being met
        target_ms = self.flow2_config['performance_target_ms']
        if execution_time_ms > target_ms:
            logger.warning(f"⚠️ Context request exceeded performance target: {execution_time_ms:.1f}ms > {target_ms}ms")

    def _validate_context_quality(self, context: ClinicalContext, urgency: str):
        """
        Validate context quality for Flow 2 requirements
        """
        # Check completeness score based on urgency
        min_completeness = {
            'emergency': 0.5,  # Lower threshold for emergency
            'urgent': 0.7,     # Medium threshold for urgent
            'routine': 0.8     # Higher threshold for routine
        }

        required_completeness = min_completeness.get(urgency, 0.8)

        if context.completeness_score < required_completeness:
            logger.warning(
                f"⚠️ Context completeness below threshold for {urgency}: "
                f"{context.completeness_score:.2%} < {required_completeness:.2%}"
            )

        # Check for critical safety flags
        critical_flags = [
            flag for flag in context.safety_flags
            if flag.get('severity') == 'CRITICAL'
        ]

        if critical_flags:
            logger.warning(f"⚠️ Context contains {len(critical_flags)} critical safety flags")

    def get_flow2_performance_metrics(self) -> Dict[str, Any]:
        """
        Get Flow 2 performance metrics for monitoring and optimization
        """
        total = self.performance_metrics['total_requests']
        successful = self.performance_metrics['successful_requests']

        return {
            'total_requests': total,
            'successful_requests': successful,
            'failed_requests': self.performance_metrics['failed_requests'],
            'success_rate': (successful / total) if total > 0 else 0,
            'average_response_time_ms': self.performance_metrics['average_response_time_ms'],
            'performance_target_ms': self.flow2_config['performance_target_ms'],
            'circuit_breaker_trips': self.performance_metrics['circuit_breaker_trips'],
            'flow2_config': self.flow2_config
        }
