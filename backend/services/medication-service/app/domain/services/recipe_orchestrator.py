"""
Recipe Orchestrator - Flow 2 Implementation

This module implements the Recipe Orchestrator that coordinates between the Context Service
and Clinical Recipe Engine, providing the core Flow 2 integration logic as specified in
FLOW2_CONTEXT_INTEGRATION_PLAN.md.

Key Features:
- Context recipe determination based on medication type
- Context Service integration for real data
- Clinical recipe execution with real context
- Error handling and graceful degradation
- Performance monitoring and optimization
"""

import logging
from typing import Dict, List, Any, Optional, Union
from dataclasses import dataclass
from datetime import datetime
import asyncio
import time

from app.infrastructure.context_service_client import ContextServiceClient, ClinicalContext
from app.infrastructure.safety_gateway_client import SafetyGatewayClient
from app.domain.services.clinical_recipe_engine import ClinicalRecipeEngine, RecipeContext, RecipeResult
from app.domain.services.context_data_adapter import ContextDataAdapter

# Enhanced Orchestrator Components
from app.domain.services.request_analyzer import RequestAnalyzer
from app.domain.services.context_selection_engine import ContextSelectionEngine
from app.domain.services.priority_resolver import PriorityResolver

logger = logging.getLogger(__name__)


@dataclass
class MedicationSafetyRequest:
    """Request for medication safety validation using Flow 2"""
    patient_id: str
    medication: Dict[str, Any]
    provider_id: Optional[str] = None
    encounter_id: Optional[str] = None
    action_type: str = "prescribe"
    urgency: str = "routine"  # routine, urgent, emergency
    workflow_id: Optional[str] = None


@dataclass
class Flow2Result:
    """Complete result of Flow 2 execution"""
    request_id: str
    patient_id: str
    overall_safety_status: str  # SAFE, WARNING, UNSAFE, ERROR
    context_recipe_used: str
    clinical_recipes_executed: List[str]
    context_completeness_score: float
    execution_time_ms: float
    clinical_results: List[RecipeResult]
    context_data: Optional[ClinicalContext] = None
    safety_summary: Dict[str, Any] = None
    performance_metrics: Dict[str, Any] = None
    errors: List[str] = None


class RecipeOrchestrator:
    """
    Recipe Orchestrator - Core Flow 2 Implementation
    
    Coordinates between Context Service and Clinical Recipe Engine to provide
    comprehensive medication safety validation with real clinical context.
    
    Flow:
    1. Determine appropriate context recipe based on medication
    2. Get optimized context from Context Service
    3. Transform context for clinical recipes
    4. Execute applicable clinical recipes with real context
    5. Aggregate results and generate safety summary
    """
    
    def __init__(
        self,
        context_service_url: str = "http://localhost:8016",
        enable_safety_gateway: bool = False,
        safety_gateway_url: str = "localhost:8030",
        enable_enhanced_orchestration: bool = True
    ):
        # Existing components (maintained for backward compatibility)
        self.context_service_client = ContextServiceClient(context_service_url)
        self.clinical_recipe_engine = ClinicalRecipeEngine()
        self.context_data_adapter = ContextDataAdapter()

        # Enhanced Orchestrator Components (NEW)
        self.enable_enhanced_orchestration = enable_enhanced_orchestration
        if enable_enhanced_orchestration:
            self.request_analyzer = RequestAnalyzer()
            self.context_selection_engine = ContextSelectionEngine()
            self.priority_resolver = PriorityResolver()
            logger.info("🧠 Enhanced Orchestration enabled with clinical intelligence")
        else:
            logger.info("📋 Using legacy orchestration (simple boolean logic)")

        # Safety Gateway Platform integration (optional - for workflow engine use)
        self.enable_safety_gateway = enable_safety_gateway
        self.safety_gateway_client = None
        if enable_safety_gateway:
            self.safety_gateway_client = SafetyGatewayClient(safety_gateway_url)
        
        # Context recipe mapping based on medication characteristics
        self.context_recipe_mapping = {
            'anticoagulant': 'medication_safety_base_context_v2',
            'chemotherapy': 'medication_safety_base_context_v2', 
            'renal_adjustment': 'medication_renal_context_v2',
            'controlled_substance': 'medication_safety_base_context_v2',
            'high_risk': 'medication_safety_base_context_v2',
            'cae_integration': 'cae_integration_context_v1',
            'safety_gateway': 'safety_gateway_context_v1',
            'default': 'medication_safety_base_context_v2'
        }
    
    async def execute_medication_safety(self, request: MedicationSafetyRequest) -> Flow2Result:
        """
        Main orchestration method - implements complete Flow 2 logic
        
        This is the primary entry point for Flow 2 medication safety validation.
        """
        start_time = time.time()
        request_id = f"flow2_{request.patient_id}_{int(start_time * 1000)}"
        
        logger.info(f"🚀 Starting Flow 2 execution for request {request_id}")
        logger.info(f"   Patient: {request.patient_id}")
        logger.info(f"   Medication: {request.medication.get('name', 'Unknown')}")
        logger.info(f"   Action: {request.action_type}")
        logger.info(f"   Urgency: {request.urgency}")
        
        try:
            # Step 1: Enhanced Context Recipe Selection
            if self.enable_enhanced_orchestration:
                context_recipe_id, selection_details = await self._intelligent_context_selection(request)
                logger.info(f"🧠 Intelligent context recipe selected: {context_recipe_id}")
                logger.info(f"   Selection confidence: {selection_details.get('confidence', 0.0):.2f}")
                logger.info(f"   Selection strategy: {selection_details.get('strategy', 'unknown')}")
            else:
                context_recipe_id = self._determine_context_recipe(request)
                logger.info(f"📋 Legacy context recipe determined: {context_recipe_id}")
                selection_details = {"strategy": "legacy", "confidence": 0.5}

            # Step 2: Get context from Context Service
            context_data = await self._get_context_from_service(request, context_recipe_id)
            logger.info(f"📊 Context retrieved - Completeness: {context_data.completeness_score:.2%}")
            
            # Step 3: Transform context for clinical recipes
            recipe_context = self._transform_context_for_recipes(context_data, request)
            logger.info(f"🔄 Context transformed for clinical recipes")
            
            # Step 4: Execute clinical recipes with real context
            clinical_results = await self._execute_clinical_recipes(recipe_context)
            logger.info(f"⚡ Executed {len(clinical_results)} clinical recipes")
            
            # Step 5: Aggregate results and generate safety summary
            flow2_result = self._aggregate_results(
                request_id, request, context_recipe_id, context_data,
                clinical_results, start_time, selection_details
            )
            
            execution_time = (time.time() - start_time) * 1000
            logger.info(f"✅ Flow 2 completed in {execution_time:.1f}ms")
            logger.info(f"   Overall Status: {flow2_result.overall_safety_status}")
            logger.info(f"   Context Completeness: {flow2_result.context_completeness_score:.2%}")
            
            return flow2_result
            
        except Exception as e:
            execution_time = (time.time() - start_time) * 1000
            logger.error(f"❌ Flow 2 execution failed: {str(e)}")
            
            # Re-raise the exception instead of graceful degradation
            logger.error(f"❌ Medication safety orchestration failed: {str(e)}")
            raise

    async def _intelligent_context_selection(self, request: MedicationSafetyRequest) -> tuple[str, dict]:
        """
        Enhanced context recipe selection using clinical intelligence

        This method implements the sophisticated Enhanced Orchestrator pipeline:
        1. Multi-dimensional request analysis
        2. YAML-based rule matching and scoring
        3. Multi-match resolution with clinical priorities
        4. Comprehensive audit trails

        Returns:
            tuple: (context_recipe_id, selection_details)
        """

        try:
            logger.info(f"🧠 Starting intelligent context selection")

            # Step 1: Multi-dimensional request analysis
            analyzed_request = await self.request_analyzer.analyze_request(request)
            logger.info(f"📊 Request analyzed - Risk: {analyzed_request.enriched_context.overall_risk_level.value}, "
                       f"Complexity: {analyzed_request.enriched_context.complexity_score:.2f}")

            # Step 2: Context recipe selection using YAML rules
            selection_result = await self.context_selection_engine.select_context_recipe(analyzed_request)

            # Step 3: Handle multi-match scenarios
            if selection_result.multiple_matches and len(selection_result.matched_rules) > 1:
                logger.info(f"🔧 Multiple matches found - resolving with priority resolver")
                resolution_result = await self.priority_resolver.resolve_multiple_matches(
                    selection_result.matched_rules, analyzed_request
                )

                final_recipe = resolution_result.primary_recipe
                selection_details = {
                    "strategy": f"multi_match_{resolution_result.resolution_strategy.value}",
                    "confidence": resolution_result.confidence,
                    "primary_recipe": resolution_result.primary_recipe,
                    "secondary_recipes": resolution_result.secondary_recipes,
                    "selected_rules": [r.rule.name for r in resolution_result.selected_rules],
                    "resolution_time_ms": resolution_result.resolution_time_ms,
                    "rationale": resolution_result.combination_rationale
                }

                logger.info(f"✅ Multi-match resolved using {resolution_result.resolution_strategy.value}")

            else:
                # Single match or default
                final_recipe = selection_result.context_recipe_id
                selection_details = {
                    "strategy": "single_match" if selection_result.selected_rule else "default",
                    "confidence": selection_result.confidence_score,
                    "selected_rule": selection_result.selected_rule.rule.name if selection_result.selected_rule else None,
                    "selection_time_ms": selection_result.selection_time_ms,
                    "rationale": selection_result.clinical_rationale
                }

            # Step 4: Add analysis insights to selection details
            selection_details.update({
                "analysis_id": analyzed_request.analysis_id,
                "requires_clinical_rules": analyzed_request.requires_clinical_rules,
                "clinical_flags": list(analyzed_request.enriched_context.clinical_flags),
                "monitoring_requirements": analyzed_request.enriched_context.monitoring_requirements
            })

            logger.info(f"🎯 Intelligent selection completed: {final_recipe}")
            return final_recipe, selection_details

        except Exception as e:
            logger.error(f"❌ Intelligent context selection failed: {str(e)}")
            # Re-raise the exception instead of falling back
            raise RuntimeError(f"Enhanced orchestration failed: {str(e)}") from e

    def _determine_context_recipe(self, request: MedicationSafetyRequest) -> str:
        """
        Determine which context recipe is needed based on medication characteristics

        This implements the context recipe selection logic from Flow 2 plan.
        """
        medication = request.medication

        # Check for specific medication types that require specialized context
        if medication.get('is_anticoagulant', False):
            return self.context_recipe_mapping['anticoagulant']
        elif medication.get('is_chemotherapy', False):
            return self.context_recipe_mapping['chemotherapy']
        elif medication.get('requires_renal_adjustment', False):
            return self.context_recipe_mapping['renal_adjustment']
        elif medication.get('is_controlled_substance', False):
            return self.context_recipe_mapping['controlled_substance']
        elif medication.get('is_high_risk', False):
            return self.context_recipe_mapping['high_risk']
        elif request.urgency == 'emergency':
            return self.context_recipe_mapping['cae_integration']
        else:
            return self.context_recipe_mapping['default']

    async def _get_context_from_service(
        self,
        request: MedicationSafetyRequest,
        context_recipe_id: str
    ) -> ClinicalContext:
        """
        Get context from Context Service using the determined recipe

        Implements error handling and fallback strategies.
        """
        try:
            # Use the appropriate context service method based on recipe type
            if context_recipe_id == 'medication_safety_base_context_v2':
                return await self.context_service_client.get_medication_safety_context(
                    patient_id=request.patient_id,
                    medication_id=request.medication.get('id', 'unknown'),
                    provider_id=request.provider_id
                )
            elif context_recipe_id == 'medication_renal_context_v2':
                return await self.context_service_client.get_renal_adjustment_context(
                    patient_id=request.patient_id,
                    provider_id=request.provider_id
                )
            elif context_recipe_id == 'cae_integration_context_v1':
                return await self.context_service_client.get_cae_integration_context(
                    patient_id=request.patient_id,
                    provider_id=request.provider_id,
                    encounter_id=request.encounter_id
                )
            elif context_recipe_id == 'safety_gateway_context_v1':
                return await self.context_service_client.get_safety_gateway_context(
                    patient_id=request.patient_id,
                    provider_id=request.provider_id
                )
            else:
                # Default to medication prescribing context
                return await self.context_service_client.get_medication_prescribing_context(
                    patient_id=request.patient_id,
                    provider_id=request.provider_id,
                    encounter_id=request.encounter_id
                )

        except Exception as e:
            logger.error(f"❌ Context Service call failed: {str(e)}")
            # Re-raise the exception instead of using fallback
            raise RuntimeError(f"Context Service unavailable: {str(e)}") from e

    # Fallback methods removed - Enhanced Orchestrator requires all services to be available

    def _transform_context_for_recipes(
        self,
        context_data: ClinicalContext,
        request: MedicationSafetyRequest
    ) -> RecipeContext:
        """
        Transform Context Service data into format expected by Clinical Recipe Engine

        This uses the ContextDataAdapter to properly transform real clinical data.
        """
        logger.info(f"🔄 Transforming context data for clinical recipes using adapter")

        # Use the Context Data Adapter for proper transformation
        recipe_context = self.context_data_adapter.transform_context_for_recipes(
            context_data=context_data,
            medication_data=request.medication,
            action_type=request.action_type,
            provider_id=request.provider_id,
            encounter_id=request.encounter_id
        )

        # Validate the transformed data
        validation_results = self.context_data_adapter.validate_transformed_data(recipe_context)

        logger.info(f"✅ Context transformed successfully using adapter")
        logger.info(f"   Data quality score: {validation_results['data_quality_score']:.2%}")
        logger.info(f"   Missing critical data: {validation_results['missing_critical_data']}")

        return recipe_context

    async def _execute_clinical_recipes(self, recipe_context: RecipeContext) -> List[RecipeResult]:
        """
        Execute applicable clinical recipes with real context data

        Flow 2 Step 4: Pharmaceutical Intelligence Focus
        1. Execute pharmaceutical intelligence (clinical recipes)
        2. Optionally integrate with Safety Gateway Platform (if enabled by workflow engine)
        3. Return comprehensive pharmaceutical assessment

        Note: Safety orchestration is primarily handled by Workflow Engine
        """
        logger.info(f"⚡ Executing clinical recipes with real context")

        try:
            # Step 4a: Execute pharmaceutical intelligence (our core domain)
            clinical_results = await self.clinical_recipe_engine.execute_applicable_recipes(recipe_context)

            logger.info(f"✅ Clinical recipes executed successfully")
            logger.info(f"   Recipes executed: {len(clinical_results)}")

            for result in clinical_results:
                logger.info(f"   - {result.recipe_id}: {result.overall_status} ({result.execution_time_ms:.1f}ms)")

            # Step 4b: Optional Safety Gateway Platform integration (for workflow engine)
            if self.enable_safety_gateway and self.safety_gateway_client:
                logger.info(f"🛡️ Safety Gateway Platform integration enabled")
                safety_validation_results = await self._validate_safety_via_gateway(recipe_context, clinical_results)
                enhanced_results = self._combine_clinical_and_safety_results(clinical_results, safety_validation_results)
                logger.info(f"   Combined results: {len(enhanced_results)} total validations")
                return enhanced_results
            else:
                logger.info(f"ℹ️ Safety Gateway Platform integration disabled - pharmaceutical intelligence only")
                return clinical_results

        except Exception as e:
            logger.error(f"❌ Clinical recipe execution failed: {str(e)}")
            # Return empty results to allow graceful degradation
            return []

    async def _validate_safety_via_gateway(
        self,
        recipe_context: RecipeContext,
        clinical_results: List[RecipeResult]
    ) -> Dict[str, Any]:
        """
        Validate safety using Safety Gateway Platform

        The Safety Gateway Platform orchestrates CAE and other safety engines internally.
        This provides comprehensive safety validation beyond our pharmaceutical intelligence.
        """
        try:
            # Determine if safety validation is needed
            if not self._requires_safety_validation(recipe_context, clinical_results):
                logger.info("🔍 Safety validation not required for this medication")
                return self._create_no_safety_validation_result()

            logger.info("🛡️ Initiating Safety Gateway Platform validation")

            # Extract medication identifiers
            medication_ids = self._extract_medication_identifiers(recipe_context)

            # Call Safety Gateway Platform
            safety_response = await self.safety_gateway_client.validate_safety(
                patient_id=recipe_context.patient_id,
                medication_ids=medication_ids,
                clinical_context={
                    'patient_data': recipe_context.patient_data,
                    'clinical_data': recipe_context.clinical_data
                },
                action_type=recipe_context.action_type,
                priority=self._determine_safety_priority(recipe_context),
                request_id=f"flow2_safety_{recipe_context.patient_id}_{int(time.time() * 1000)}"
            )

            logger.info(f"✅ Safety Gateway Platform validation completed")
            logger.info(f"   Safety Status: {safety_response.get('status', 'UNKNOWN')}")
            logger.info(f"   Risk Score: {safety_response.get('risk_score', 0.0):.3f}")

            return safety_response

        except Exception as e:
            logger.error(f"❌ Safety Gateway Platform validation failed: {e}")
            # Return fail-closed safety response
            return {
                "status": "UNSAFE",
                "risk_score": 1.0,
                "confidence": 0.0,
                "violations": [f"Safety Gateway Platform validation failed: {str(e)}"],
                "warnings": ["Safety validation could not be completed"],
                "explanations": ["Manual safety review required"],
                "processing_time_ms": 0,
                "metadata": {"error": str(e), "fail_closed": True}
            }

    def _requires_safety_validation(
        self,
        recipe_context: RecipeContext,
        clinical_results: List[RecipeResult]
    ) -> bool:
        """
        Determine if Safety Gateway Platform validation is required

        Safety validation is required for:
        - High-risk medications
        - Emergency situations
        - Medications with warnings from clinical recipes
        - Controlled substances
        """
        medication_data = recipe_context.medication_data

        # Always validate high-risk medications
        if medication_data.get('is_high_risk', False):
            return True

        # Always validate controlled substances
        if medication_data.get('is_controlled_substance', False):
            return True

        # Validate if clinical recipes raised warnings
        has_warnings = any(result.overall_status in ["WARNING", "UNSAFE"] for result in clinical_results)
        if has_warnings:
            return True

        # Validate emergency situations
        if recipe_context.action_type == "emergency":
            return True

        # For routine medications, use basic validation
        return True  # For now, always validate for comprehensive safety

    def _extract_medication_identifiers(self, recipe_context: RecipeContext) -> List[str]:
        """Extract medication identifiers for Safety Gateway Platform"""
        medication_data = recipe_context.medication_data

        identifiers = []

        # Primary medication name
        if medication_data.get('name'):
            identifiers.append(medication_data['name'])

        # RxNorm code if available
        if medication_data.get('rxnorm_code'):
            identifiers.append(medication_data['rxnorm_code'])

        # Generic name if different
        if medication_data.get('generic_name') and medication_data['generic_name'] != medication_data.get('name'):
            identifiers.append(medication_data['generic_name'])

        return identifiers if identifiers else ['unknown_medication']

    def _determine_safety_priority(self, recipe_context: RecipeContext) -> str:
        """Determine safety validation priority"""
        if recipe_context.action_type == "emergency":
            return "emergency"
        elif recipe_context.medication_data.get('is_high_risk', False):
            return "urgent"
        else:
            return "normal"

    def _create_no_safety_validation_result(self) -> Dict[str, Any]:
        """Create result when safety validation is not required"""
        return {
            "status": "SAFE",
            "risk_score": 0.1,
            "confidence": 1.0,
            "violations": [],
            "warnings": [],
            "explanations": ["Safety validation not required for this medication"],
            "processing_time_ms": 0,
            "metadata": {"validation_required": False}
        }

    def _combine_clinical_and_safety_results(
        self,
        clinical_results: List[RecipeResult],
        safety_validation: Dict[str, Any]
    ) -> List[RecipeResult]:
        """
        Combine pharmaceutical intelligence results with Safety Gateway Platform validation

        This creates a comprehensive safety assessment by combining:
        1. Our pharmaceutical intelligence (dose calculations, formulary, etc.)
        2. Safety Gateway Platform validation (CAE + other safety engines)
        """
        combined_results = clinical_results.copy()

        # Create Safety Gateway Platform result as a RecipeResult
        safety_result = self._create_safety_gateway_recipe_result(safety_validation)
        combined_results.append(safety_result)

        # Enhance existing clinical results with safety insights
        for clinical_result in combined_results[:-1]:  # Exclude the safety result we just added
            self._enhance_clinical_result_with_safety(clinical_result, safety_validation)

        return combined_results

    def _create_safety_gateway_recipe_result(self, safety_validation: Dict[str, Any]) -> RecipeResult:
        """Convert Safety Gateway Platform response to RecipeResult format"""

        # Map Safety Gateway Platform status to our status
        status_mapping = {
            "SAFE": "SAFE",
            "WARNING": "WARNING",
            "UNSAFE": "UNSAFE"
        }

        safety_status = safety_validation.get('status', 'UNKNOWN')
        mapped_status = status_mapping.get(safety_status, 'WARNING')

        # Create validation results from Safety Gateway Platform response
        validations = []

        # Add violations as high-severity validations
        for violation in safety_validation.get('violations', []):
            validations.append({
                'passed': False,
                'severity': 'HIGH',
                'message': violation,
                'explanation': 'Safety Gateway Platform violation',
                'source': 'safety_gateway_platform'
            })

        # Add warnings as medium-severity validations
        for warning in safety_validation.get('warnings', []):
            validations.append({
                'passed': False,
                'severity': 'MEDIUM',
                'message': warning,
                'explanation': 'Safety Gateway Platform warning',
                'source': 'safety_gateway_platform'
            })

        # If no violations or warnings, add a positive validation
        if not validations:
            validations.append({
                'passed': True,
                'severity': 'INFO',
                'message': 'Safety Gateway Platform validation passed',
                'explanation': 'Comprehensive safety validation completed successfully',
                'source': 'safety_gateway_platform'
            })

        # Create clinical decision support
        cds = {
            'provider_summary': self._generate_safety_provider_summary(safety_validation),
            'patient_explanation': self._generate_safety_patient_explanation(safety_validation),
            'monitoring_requirements': safety_validation.get('monitoring_requirements', [])
        }

        return RecipeResult(
            recipe_id="safety_gateway_platform_validation",
            recipe_name="Safety Gateway Platform Comprehensive Validation",
            execution_time_ms=safety_validation.get('processing_time_ms', 0),
            validations=validations,
            overall_status=mapped_status,
            clinical_decision_support=cds,
            cost_considerations={},
            ml_insights={},
            performance_metrics={
                'risk_score': safety_validation.get('risk_score', 0.0),
                'confidence': safety_validation.get('confidence', 0.0),
                'engines_executed': safety_validation.get('metadata', {}).get('engines_executed', [])
            }
        )

    def _enhance_clinical_result_with_safety(
        self,
        clinical_result: RecipeResult,
        safety_validation: Dict[str, Any]
    ):
        """Enhance clinical recipe results with safety validation insights"""

        # Add safety metadata to performance metrics
        if not hasattr(clinical_result, 'performance_metrics') or not clinical_result.performance_metrics:
            clinical_result.performance_metrics = {}

        clinical_result.performance_metrics.update({
            'safety_gateway_status': safety_validation.get('status'),
            'safety_risk_score': safety_validation.get('risk_score', 0.0),
            'safety_confidence': safety_validation.get('confidence', 0.0)
        })

        # Enhance clinical decision support with safety insights
        if clinical_result.clinical_decision_support:
            # Add safety context to provider summary
            provider_summary = clinical_result.clinical_decision_support.get('provider_summary', '')
            safety_context = f" Safety Gateway Platform: {safety_validation.get('status', 'UNKNOWN')}"
            clinical_result.clinical_decision_support['provider_summary'] = provider_summary + safety_context

    def _generate_safety_provider_summary(self, safety_validation: Dict[str, Any]) -> str:
        """Generate provider summary for Safety Gateway Platform validation"""
        status = safety_validation.get('status', 'UNKNOWN')
        risk_score = safety_validation.get('risk_score', 0.0)
        violations = safety_validation.get('violations', [])
        warnings = safety_validation.get('warnings', [])

        if status == "UNSAFE":
            return f"UNSAFE: Safety Gateway Platform identified {len(violations)} critical safety issues. Review required before prescribing."
        elif status == "WARNING":
            return f"WARNING: Safety Gateway Platform identified {len(warnings)} safety concerns. Consider alternatives or additional monitoring."
        else:
            return f"SAFE: Safety Gateway Platform validation passed (Risk Score: {risk_score:.3f}). Medication appears safe to prescribe."

    def _generate_safety_patient_explanation(self, safety_validation: Dict[str, Any]) -> str:
        """Generate patient explanation for Safety Gateway Platform validation"""
        status = safety_validation.get('status', 'UNKNOWN')

        if status == "UNSAFE":
            return "Our safety systems have identified potential concerns with this medication for you. Your doctor will review these before prescribing."
        elif status == "WARNING":
            return "Our safety systems have identified some considerations for this medication. Your doctor will discuss these with you."
        else:
            return "Our comprehensive safety systems have reviewed this medication and it appears to be safe and appropriate for you."

    def _aggregate_results(
        self,
        request_id: str,
        request: MedicationSafetyRequest,
        context_recipe_id: str,
        context_data: ClinicalContext,
        clinical_results: List[RecipeResult],
        start_time: float,
        selection_details: dict = None
    ) -> Flow2Result:
        """
        Aggregate all results into a comprehensive Flow 2 result

        This provides the final safety assessment and recommendations.
        """
        execution_time = (time.time() - start_time) * 1000

        # Determine overall safety status
        overall_status = self._determine_overall_safety_status(clinical_results, context_data)

        # Extract executed recipe IDs
        clinical_recipes_executed = [result.recipe_id for result in clinical_results]

        # Generate safety summary
        safety_summary = self._generate_safety_summary(clinical_results, context_data)

        # Generate performance metrics
        performance_metrics = self._generate_performance_metrics(
            clinical_results, context_data, execution_time
        )

        # Collect any errors
        errors = []
        if context_data.connection_errors:
            errors.extend([error.get('error', 'Unknown error') for error in context_data.connection_errors])

        result = Flow2Result(
            request_id=request_id,
            patient_id=request.patient_id,
            overall_safety_status=overall_status,
            context_recipe_used=context_recipe_id,
            clinical_recipes_executed=clinical_recipes_executed,
            context_completeness_score=context_data.completeness_score,
            execution_time_ms=execution_time,
            clinical_results=clinical_results,
            context_data=context_data,
            safety_summary=safety_summary,
            performance_metrics=performance_metrics,
            errors=errors if errors else None
        )

        # Add enhanced orchestration details if available
        if selection_details:
            result.orchestration_details = {
                "enhanced_orchestration_enabled": True,
                "selection_strategy": selection_details.get("strategy", "unknown"),
                "selection_confidence": selection_details.get("confidence", 0.0),
                "clinical_intelligence": {
                    "analysis_id": selection_details.get("analysis_id"),
                    "requires_clinical_rules": selection_details.get("requires_clinical_rules", False),
                    "clinical_flags": selection_details.get("clinical_flags", []),
                    "monitoring_requirements": selection_details.get("monitoring_requirements", [])
                },
                "selection_rationale": selection_details.get("rationale", ""),
                "performance": {
                    "selection_time_ms": selection_details.get("selection_time_ms", 0.0),
                    "resolution_time_ms": selection_details.get("resolution_time_ms", 0.0)
                }
            }
        else:
            result.orchestration_details = {
                "enhanced_orchestration_enabled": False,
                "selection_strategy": "legacy",
                "selection_confidence": 0.5
            }

        logger.info(f"📊 Results aggregated successfully")
        logger.info(f"   Overall Status: {overall_status}")
        logger.info(f"   Clinical Recipes: {len(clinical_recipes_executed)}")
        logger.info(f"   Context Completeness: {context_data.completeness_score:.2%}")
        logger.info(f"   Execution Time: {execution_time:.1f}ms")

        # Log enhanced orchestration details
        if selection_details:
            logger.info(f"🧠 Enhanced Orchestration Details:")
            logger.info(f"   Strategy: {selection_details.get('strategy', 'unknown')}")
            logger.info(f"   Confidence: {selection_details.get('confidence', 0.0):.2f}")
            if selection_details.get('clinical_flags'):
                logger.info(f"   Clinical Flags: {selection_details['clinical_flags']}")
            if selection_details.get('monitoring_requirements'):
                logger.info(f"   Monitoring Required: {len(selection_details['monitoring_requirements'])} items")

        return result

    def _determine_overall_safety_status(
        self,
        clinical_results: List[RecipeResult],
        context_data: ClinicalContext
    ) -> str:
        """
        Determine overall safety status based on clinical results and context quality
        """
        if not clinical_results:
            return "ERROR"

        # Check for any UNSAFE results
        for result in clinical_results:
            if result.overall_status == "UNSAFE":
                return "UNSAFE"

        # Check for WARNING results
        has_warnings = any(result.overall_status == "WARNING" for result in clinical_results)

        # Consider context completeness
        low_completeness = context_data.completeness_score < 0.7

        # Check for critical safety flags
        critical_flags = [
            flag for flag in context_data.safety_flags
            if flag.get('severity') == 'CRITICAL'
        ]

        if critical_flags or low_completeness:
            return "WARNING"
        elif has_warnings:
            return "WARNING"
        else:
            return "SAFE"

    def _generate_safety_summary(
        self,
        clinical_results: List[RecipeResult],
        context_data: ClinicalContext
    ) -> Dict[str, Any]:
        """
        Generate comprehensive safety summary for clinical decision support
        """
        # Collect all validations
        all_validations = []
        for result in clinical_results:
            all_validations.extend(result.validations)

        # Categorize issues
        critical_issues = [v for v in all_validations if not v.passed and v.severity == "CRITICAL"]
        high_issues = [v for v in all_validations if not v.passed and v.severity == "HIGH"]
        medium_issues = [v for v in all_validations if not v.passed and v.severity == "MEDIUM"]

        # Generate recommendations
        recommendations = []
        for validation in critical_issues + high_issues:
            recommendations.extend(validation.alternatives)

        return {
            "total_validations": len(all_validations),
            "critical_issues": len(critical_issues),
            "high_issues": len(high_issues),
            "medium_issues": len(medium_issues),
            "context_completeness": context_data.completeness_score,
            "context_safety_flags": len(context_data.safety_flags),
            "recommendations": list(set(recommendations)),  # Remove duplicates
            "clinical_decision_support": {
                "provider_summary": self._generate_provider_summary(clinical_results),
                "patient_explanation": self._generate_patient_explanation(clinical_results),
                "monitoring_requirements": self._extract_monitoring_requirements(clinical_results)
            }
        }

    def _generate_performance_metrics(
        self,
        clinical_results: List[RecipeResult],
        context_data: ClinicalContext,
        total_execution_time: float
    ) -> Dict[str, Any]:
        """
        Generate performance metrics for monitoring and optimization
        """
        recipe_times = [result.execution_time_ms for result in clinical_results]

        return {
            "total_execution_time_ms": total_execution_time,
            "context_assembly_time_ms": context_data.assembly_duration_ms,
            "clinical_recipes_time_ms": sum(recipe_times),
            "recipes_executed": len(clinical_results),
            "average_recipe_time_ms": sum(recipe_times) / len(recipe_times) if recipe_times else 0,
            "context_completeness": context_data.completeness_score,
            "data_sources_used": len(context_data.source_metadata),
            "cache_performance": {
                "context_cached": "cache_hit" in context_data.source_metadata,
                "cache_strategy": context_data.source_metadata.get("cache_strategy", "unknown")
            }
        }

    def _generate_provider_summary(self, clinical_results: List[RecipeResult]) -> str:
        """Generate summary for healthcare providers"""
        if not clinical_results:
            return "No clinical validation performed due to system error."

        total_recipes = len(clinical_results)
        unsafe_count = sum(1 for r in clinical_results if r.overall_status == "UNSAFE")
        warning_count = sum(1 for r in clinical_results if r.overall_status == "WARNING")

        if unsafe_count > 0:
            return f"UNSAFE: {unsafe_count}/{total_recipes} safety checks failed. Review required before prescribing."
        elif warning_count > 0:
            return f"WARNING: {warning_count}/{total_recipes} safety checks raised concerns. Consider alternatives."
        else:
            return f"SAFE: All {total_recipes} safety checks passed. Medication appears safe to prescribe."

    def _generate_patient_explanation(self, clinical_results: List[RecipeResult]) -> str:
        """Generate patient-friendly explanation"""
        if not clinical_results:
            return "We're checking if this medication is safe for you."

        unsafe_count = sum(1 for r in clinical_results if r.overall_status == "UNSAFE")
        warning_count = sum(1 for r in clinical_results if r.overall_status == "WARNING")

        if unsafe_count > 0:
            return "We found some safety concerns with this medication for you. Your doctor will discuss alternatives."
        elif warning_count > 0:
            return "This medication may be suitable for you, but we want to monitor you carefully."
        else:
            return "This medication appears to be safe and appropriate for your condition."

    def _extract_monitoring_requirements(self, clinical_results: List[RecipeResult]) -> List[str]:
        """Extract monitoring requirements from clinical decision support"""
        monitoring = []

        for result in clinical_results:
            cds = result.clinical_decision_support
            if isinstance(cds, dict) and 'monitoring_recommendations' in cds:
                monitoring.extend(cds['monitoring_recommendations'])

        return list(set(monitoring))  # Remove duplicates

    async def health_check(self) -> Dict[str, Any]:
        """
        Health check for Recipe Orchestrator and dependencies
        """
        health_status = {
            "recipe_orchestrator": "healthy",
            "context_service": "unknown",
            "safety_gateway_platform": "unknown",
            "clinical_recipe_engine": "unknown",
            "registered_recipes": len(self.clinical_recipe_engine.recipes)
        }

        try:
            # Check Context Service health
            context_healthy = await self.context_service_client.health_check()
            health_status["context_service"] = "healthy" if context_healthy else "unhealthy"
        except Exception as e:
            health_status["context_service"] = f"error: {str(e)}"

        # Check Safety Gateway Platform health (if enabled)
        if self.enable_safety_gateway and self.safety_gateway_client:
            try:
                safety_health = await self.safety_gateway_client.health_check()
                health_status["safety_gateway_platform"] = safety_health.get("status", "unknown")
            except Exception as e:
                health_status["safety_gateway_platform"] = f"error: {str(e)}"
        else:
            health_status["safety_gateway_platform"] = "disabled (workflow engine handles safety orchestration)"

        try:
            # Check Clinical Recipe Engine
            recipe_catalog = self.clinical_recipe_engine.get_recipe_catalog()
            health_status["clinical_recipe_engine"] = "healthy"
            health_status["available_recipes"] = list(recipe_catalog.keys())
        except Exception as e:
            health_status["clinical_recipe_engine"] = f"error: {str(e)}"

        return health_status
