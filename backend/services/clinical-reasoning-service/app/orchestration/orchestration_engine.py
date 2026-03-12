"""
Orchestration Engine for Clinical Assertion Engine

Main orchestration engine that coordinates request routing, parallel execution,
and decision aggregation for optimal clinical reasoning performance.
"""

import asyncio
import logging
from datetime import datetime
from typing import Dict, List, Optional, Any

from .request_router import RequestRouter, ClinicalRequest
from .parallel_executor import ParallelExecutor, ReasonerResult, ReasonerStatus
from .decision_aggregator import DecisionAggregator, ClinicalAssertion
from .priority_queue import PriorityQueue

logger = logging.getLogger(__name__)


class OrchestrationEngine:
    """
    Main orchestration engine for Clinical Assertion Engine
    
    Coordinates the entire clinical reasoning pipeline:
    1. Request routing and priority classification
    2. Parallel reasoner execution
    3. Decision aggregation and conflict resolution
    4. Performance optimization and monitoring
    """
    
    def __init__(self, max_queue_size: int = 1000, max_concurrent: int = 100):
        # Initialize components
        self.request_router = RequestRouter()
        self.parallel_executor = ParallelExecutor()
        self.decision_aggregator = DecisionAggregator()
        self.priority_queue = PriorityQueue(max_queue_size, max_concurrent)

        # Track reasoner errors for reporting to Safety Gateway Platform
        self.last_reasoner_errors = []

        # Initialize intelligence components
        try:
            from intelligence.rule_engine import SelfImprovingRuleEngine
            from intelligence.performance_optimizer import PerformanceOptimizer
            from intelligence.confidence_evolver import ConfidenceEvolver
            from intelligence.pattern_learner import PatternLearner

            self.rule_engine = SelfImprovingRuleEngine()
            self.performance_optimizer = PerformanceOptimizer()
            self.confidence_evolver = ConfidenceEvolver()
            self.pattern_learner = PatternLearner()

            logger.info("✓ Intelligence components initialized")
        except ImportError as e:
            logger.warning(f"Intelligence components not available: {e}")
            self.rule_engine = None
            self.performance_optimizer = None
            self.confidence_evolver = None
            self.pattern_learner = None

        # Initialize event envelope system
        try:
            from events.event_processors import EventProcessorRegistry
            from events.event_sourcing import EventStore, IdempotencyManager
            from events.clinical_context_assembler import ContextEnrichmentEngine

            self.event_processor_registry = EventProcessorRegistry()
            self.event_store = EventStore()
            self.idempotency_manager = IdempotencyManager()
            self.context_enrichment_engine = ContextEnrichmentEngine()

            logger.info("✓ Event envelope system initialized")
        except ImportError as e:
            logger.warning(f"Event envelope system not available: {e}")
            self.event_processor_registry = None
            self.event_store = None
            self.idempotency_manager = None
            self.context_enrichment_engine = None

        # Reasoner instances (will be injected)
        self.reasoner_instances = {}

        # Performance tracking
        self.performance_stats = {
            'total_requests': 0,
            'successful_requests': 0,
            'failed_requests': 0,
            'average_response_time': 0.0,
            'p99_response_time': 0.0,
            'response_times': []
        }

        # Start queue processor
        self.queue_processor_task = None

        logger.info("Orchestration Engine initialized")
    
    def register_reasoner(self, reasoner_type: str, reasoner_instance: Any):
        """Register a reasoner instance with the orchestration engine"""
        self.reasoner_instances[reasoner_type] = reasoner_instance
        logger.info(f"Registered reasoner: {reasoner_type}")
    
    async def start(self):
        """Start the orchestration engine"""
        logger.info("Starting Orchestration Engine")
        
        # Start queue processor
        self.queue_processor_task = asyncio.create_task(
            self.priority_queue.process_queue(self._process_clinical_request)
        )
        
        logger.info("Orchestration Engine started")
    
    async def stop(self):
        """Stop the orchestration engine"""
        logger.info("Stopping Orchestration Engine")
        
        if self.queue_processor_task:
            self.queue_processor_task.cancel()
            try:
                await self.queue_processor_task
            except asyncio.CancelledError:
                pass
        
        logger.info("Orchestration Engine stopped")
    
    async def generate_clinical_assertions(self, raw_request: Dict[str, Any]) -> List[ClinicalAssertion]:
        logger.info(f"OrchestrationEngine: Starting generation for request: {raw_request.get('request_id', 'N/A')}")
        """
        Main entry point for clinical assertion generation with intelligence enhancement

        Args:
            raw_request: Raw request data from gRPC

        Returns:
            List of clinical assertions
        """
        start_time = datetime.utcnow()

        try:
            # Route and classify request
            clinical_request = await self.request_router.route_request(raw_request)

            # Use performance optimizer for sub-100ms guarantee if available
            if self.performance_optimizer:
                cache_key = f"assertions_{clinical_request.patient_id}_{hash(str(sorted(clinical_request.medication_ids)))}"

                async def compute_assertions():
                    # Perform computation
                    assertions = await self._compute_assertions_with_intelligence(clinical_request)
                    logger.info(f"OrchestrationEngine: Finished generation for request: {raw_request.get('request_id', 'N/A')}. Found {len(assertions)} assertions.")
                    return assertions

                assertions, response_time = await self.performance_optimizer.ensure_sub_100ms_response(
                    compute_assertions, cache_key
                )
            else:
                # Fallback to standard processing
                future = await self.priority_queue.enqueue_request(clinical_request)
                assertions = await future
                response_time = (datetime.utcnow() - start_time).total_seconds() * 1000

            # Update performance statistics
            self._update_performance_stats(response_time, success=True)

            logger.info(f"Generated {len(assertions)} clinical assertions "
                       f"for patient {clinical_request.patient_id} "
                       f"in {response_time:.2f}ms")

            return assertions

        except Exception as e:
            response_time = (datetime.utcnow() - start_time).total_seconds() * 1000
            self._update_performance_stats(response_time, success=False)
            logger.error(f"Error generating clinical assertions: {e}")
            raise
    
    async def _process_clinical_request(self, request: ClinicalRequest) -> List[ClinicalAssertion]:
        """
        Process a clinical request through the full pipeline

        Args:
            request: Classified clinical request

        Returns:
            List of clinical assertions
        """
        try:
            return await self._compute_assertions_with_intelligence(request)

        except Exception as e:
            logger.error(f"Error processing clinical request: {e}")
            # Return empty list rather than failing completely
            return []

    async def _compute_assertions_with_intelligence(self, request: ClinicalRequest) -> List[ClinicalAssertion]:
        """
        Compute assertions with intelligence enhancement

        Args:
            request: Clinical request

        Returns:
            List of enhanced clinical assertions
        """
        try:
            # Clear previous errors
            self.last_reasoner_errors = []

            # 1. Enrich request with patient context if available
            if self.context_enrichment_engine:
                try:
                    request = await self.context_enrichment_engine.enrich_request_with_context(request)
                    # Handle both object and dictionary types for patient_id logging
                    patient_id = request.get('patient_id', 'unknown') if isinstance(request, dict) else request.patient_id
                    logger.info(f"Enriched patient context for patient {patient_id}")
                except Exception as e:
                    logger.warning(f"Failed to enrich patient context: {e}")

            # 2. Execute reasoners in parallel
            logger.info("Executing reasoners in parallel...")
            reasoner_results = await self.parallel_executor.execute_reasoners(
                request, self.reasoner_instances
            )
            logger.info(f"Finished executing reasoners. Results: {len(reasoner_results)}")

            # Capture reasoner errors for Safety Gateway Platform
            from .parallel_executor import ReasonerStatus
            for reasoner_type, result in reasoner_results.items():
                if hasattr(result, 'status') and result.status == ReasonerStatus.FAILED:
                    if hasattr(result, 'error_message') and result.error_message:
                        self.last_reasoner_errors.append(f"{result.reasoner_type}: {result.error_message}")
                    else:
                        self.last_reasoner_errors.append(f"{result.reasoner_type}: Reasoner failed without providing a specific error message.")

            # 3. Aggregate decisions
            logger.info("Aggregating decisions...")
            assertions = await self.decision_aggregator.aggregate_decisions(
                request, reasoner_results
            )
            logger.info(f"Finished aggregating decisions. Assertions: {len(assertions)}")

            # 4. Enhance confidence scores if confidence evolver available
            if self.confidence_evolver and assertions:
                for assertion in assertions:
                    try:
                        # Create evidence for confidence evolution
                        from intelligence.confidence_evolver import ConfidenceEvidence
                        evidence = [
                            ConfidenceEvidence(
                                evidence_type="reasoner_output",
                                strength=assertion.confidence_score,
                                weight=1.0,
                                source=f"reasoner_{assertion.assertion_type}",
                                timestamp=datetime.utcnow()
                            )
                        ]

                        # Evolve confidence
                        evolved_confidence = await self.confidence_evolver.evolve_confidence(
                            assertion.assertion_id, evidence
                        )

                        # Update assertion confidence
                        assertion.confidence_score = evolved_confidence.overall_confidence

                    except Exception as e:
                        logger.warning(f"Error evolving confidence for assertion {assertion.assertion_id}: {e}")

            # 5. Generate pattern-based predictions if pattern learner available
            if self.pattern_learner and request.medication_ids:
                try:
                    pattern_prediction = await self.pattern_learner.predict_clinical_outcome(
                        request.clinical_context or {},
                        request.medication_ids
                    )

                    # Add pattern prediction as an assertion if confidence is high enough
                    if pattern_prediction.confidence > 0.7:
                        from .decision_aggregator import ClinicalAssertion, AssertionSeverity

                        # Map prediction outcome to protobuf enum severity
                        severity_map = {
                            "adverse_event": clinical_reasoning_pb2.AssertionSeverity.SEVERITY_HIGH,
                            "therapeutic_success": clinical_reasoning_pb2.AssertionSeverity.SEVERITY_LOW,
                            "neutral": clinical_reasoning_pb2.AssertionSeverity.SEVERITY_MODERATE
                        }

                        pattern_assertion = ClinicalAssertion(
                            assertion_id=f"pattern_{pattern_prediction.prediction_id}",
                            assertion_type="pattern_prediction",
                            severity=severity_map.get(pattern_prediction.predicted_outcome, clinical_reasoning_pb2.AssertionSeverity.SEVERITY_MODERATE),
                            title=f"Pattern-based Prediction: {pattern_prediction.predicted_outcome}",
                            description=pattern_prediction.explanation,
                            explanation=f"Based on {len(pattern_prediction.supporting_patterns)} learned patterns",
                            confidence_score=pattern_prediction.confidence,
                            evidence_sources=["Pattern Learning Engine"],
                            recommendations=pattern_prediction.recommendations,
                            created_at=datetime.utcnow()
                        )

                        assertions.append(pattern_assertion)

                except Exception as e:
                    logger.warning(f"Error generating pattern prediction: {e}")

            return assertions

        except Exception as e:
            logger.error(f"Error in intelligence-enhanced computation: {e}")
            # Fallback to basic reasoner execution
            reasoner_results = await self.parallel_executor.execute_reasoners(
                request, self.reasoner_instances
            )
            return await self.decision_aggregator.aggregate_decisions(request, reasoner_results)
    
    async def check_medication_interactions(self, raw_request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Specialized method for medication interaction checking
        
        Args:
            raw_request: Raw request data
            
        Returns:
            Interaction check results
        """
        # Force reasoner types to interaction only
        raw_request['reasoner_types'] = ['interaction']
        
        # Generate assertions
        assertions = await self.generate_clinical_assertions(raw_request)
        
        # Convert to interaction-specific format
        interactions = []
        for assertion in assertions:
            if assertion.assertion_type == 'interaction':
                interactions.append({
                    'interaction_id': assertion.assertion_id,
                    'severity': assertion.severity.value,
                    'description': assertion.description,
                    'explanation': assertion.explanation,
                    'confidence': assertion.confidence_score,
                    'evidence_sources': assertion.evidence_sources,
                    'recommendations': assertion.recommendations
                })
        
        return {
            'patient_id': raw_request.get('patient_id'),
            'interactions': interactions,
            'total_interactions': len(interactions),
            'has_critical_interactions': any(i['severity'] == 'critical' for i in interactions)
        }
    
    async def check_contraindications(self, raw_request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Specialized method for contraindication checking
        
        Args:
            raw_request: Raw request data
            
        Returns:
            Contraindication check results
        """
        # Force reasoner types to contraindication only
        raw_request['reasoner_types'] = ['contraindication']
        
        # Generate assertions
        assertions = await self.generate_clinical_assertions(raw_request)
        
        # Convert to contraindication-specific format
        contraindications = []
        for assertion in assertions:
            if assertion.assertion_type == 'contraindication':
                contraindications.append({
                    'contraindication_id': assertion.assertion_id,
                    'severity': assertion.severity.value,
                    'description': assertion.description,
                    'explanation': assertion.explanation,
                    'confidence': assertion.confidence_score,
                    'evidence_sources': assertion.evidence_sources,
                    'recommendations': assertion.recommendations
                })
        
        return {
            'patient_id': raw_request.get('patient_id'),
            'contraindications': contraindications,
            'total_contraindications': len(contraindications),
            'has_absolute_contraindications': any(c['severity'] == 'critical' for c in contraindications)
        }
    
    def _update_performance_stats(self, response_time_ms: float, success: bool):
        """Update performance statistics"""
        self.performance_stats['total_requests'] += 1
        
        if success:
            self.performance_stats['successful_requests'] += 1
        else:
            self.performance_stats['failed_requests'] += 1
        
        # Update response times
        self.performance_stats['response_times'].append(response_time_ms)
        
        # Keep only last 1000 response times for memory efficiency
        if len(self.performance_stats['response_times']) > 1000:
            self.performance_stats['response_times'] = self.performance_stats['response_times'][-1000:]
        
        # Update average response time
        response_times = self.performance_stats['response_times']
        self.performance_stats['average_response_time'] = sum(response_times) / len(response_times)
        
        # Update p99 response time
        if len(response_times) >= 10:
            sorted_times = sorted(response_times)
            p99_index = int(len(sorted_times) * 0.99)
            self.performance_stats['p99_response_time'] = sorted_times[p99_index]

    async def learn_from_outcome(self, assertion_id: str, outcome_positive: bool,
                               outcome_strength: float = 1.0,
                               clinical_context: Dict[str, Any] = None):
        """
        Learn from clinical outcome to improve future predictions

        Args:
            assertion_id: ID of assertion that led to outcome
            outcome_positive: Whether outcome was positive
            outcome_strength: Strength of the outcome evidence
            clinical_context: Clinical context of the outcome
        """
        try:
            # Update rule engine if available
            if self.rule_engine:
                await self.rule_engine.learn_from_outcome(
                    assertion_id, outcome_positive, clinical_context or {}, outcome_strength
                )

            # Update confidence evolver if available
            if self.confidence_evolver:
                await self.confidence_evolver.update_from_outcome(
                    assertion_id, outcome_positive, outcome_strength, clinical_context
                )

            logger.info(f"Applied learning from outcome for assertion {assertion_id}: "
                       f"positive={outcome_positive}, strength={outcome_strength}")

        except Exception as e:
            logger.error(f"Error learning from outcome: {e}")

    async def learn_from_override(self, assertion_id: str, override_reason: str,
                                clinical_context: Dict[str, Any] = None,
                                clinician_expertise: float = 1.0):
        """
        Learn from clinician override to improve future predictions

        Args:
            assertion_id: ID of assertion that was overridden
            override_reason: Reason for override
            clinical_context: Clinical context of override
            clinician_expertise: Expertise level of clinician
        """
        try:
            # Update rule engine if available
            if self.rule_engine:
                await self.rule_engine.learn_from_override(
                    assertion_id, override_reason, clinical_context or {}, clinician_expertise
                )

            logger.info(f"Applied learning from override for assertion {assertion_id}: "
                       f"reason={override_reason}")

        except Exception as e:
            logger.error(f"Error learning from override: {e}")

    async def process_clinical_event_envelope(self, envelope_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process clinical event using enhanced event envelope system

        Args:
            envelope_data: Clinical event envelope data

        Returns:
            Processing result with enhanced context
        """
        try:
            # Import event envelope classes
            from events.clinical_event_envelope import ClinicalEventEnvelope
            from events.event_processors import ProcessingResult

            # Create event envelope from data
            envelope = ClinicalEventEnvelope.from_dict(envelope_data)

            # Check idempotency
            if self.idempotency_manager:
                is_duplicate, original_event_id = await self.idempotency_manager.check_idempotency(envelope)

                if is_duplicate:
                    logger.info(f"Duplicate event detected: {envelope.metadata.event_id} "
                               f"(original: {original_event_id})")
                    return {
                        "status": "duplicate",
                        "original_event_id": original_event_id,
                        "message": "Event already processed"
                    }

            # Enrich clinical context
            if self.context_enrichment_engine:
                envelope = await self.context_enrichment_engine.enrich_envelope(envelope)

            # Process event using workflow-specific processor
            if self.event_processor_registry:
                processing_outcome = await self.event_processor_registry.process_event(envelope)

                # Store event in event store
                if self.event_store and processing_outcome.result != ProcessingResult.FAILED:
                    stream_id = await self.event_store.append_event(envelope)
                    processing_outcome.metadata["stream_id"] = stream_id

                # Mark as processed for idempotency
                if (self.idempotency_manager and
                    processing_outcome.result in [ProcessingResult.SUCCESS, ProcessingResult.COMPLETED]):
                    await self.idempotency_manager.mark_event_processed(envelope)

                # Convert to response format
                return {
                    "status": "processed",
                    "event_id": envelope.metadata.event_id,
                    "processing_result": processing_outcome.result.value,
                    "warnings": processing_outcome.warnings,
                    "errors": processing_outcome.errors,
                    "recommendations": processing_outcome.recommendations,
                    "next_actions": processing_outcome.next_actions,
                    "processing_duration_ms": processing_outcome.processing_duration_ms,
                    "metadata": processing_outcome.metadata
                }
            else:
                # Fallback to standard processing
                return await self._process_envelope_fallback(envelope)

        except Exception as e:
            logger.error(f"Error processing clinical event envelope: {e}")
            return {
                "status": "error",
                "error": str(e),
                "message": "Failed to process clinical event envelope"
            }

    async def _process_envelope_fallback(self, envelope) -> Dict[str, Any]:
        """Fallback processing when event envelope system is not available"""
        try:
            # Convert envelope to standard request format
            raw_request = {
                "patient_id": envelope.clinical_context.patient_id,
                "medication_ids": [med.get("name", "") for med in envelope.clinical_context.active_medications],
                "condition_ids": [diag.get("code", "") for diag in envelope.clinical_context.active_diagnoses],
                "allergy_ids": [allergy.get("allergen", "") for allergy in envelope.clinical_context.active_allergies],
                "reasoner_types": ["interaction", "contraindication", "dosing"],
                "clinical_context": {
                    "encounter_id": envelope.clinical_context.encounter_id,
                    "encounter_type": envelope.clinical_context.encounter_type,
                    "facility_id": envelope.clinical_context.facility_id
                }
            }

            # Process using standard orchestration
            assertions = await self.generate_clinical_assertions(raw_request)

            return {
                "status": "processed_fallback",
                "event_id": envelope.metadata.event_id,
                "assertions": [asdict(assertion) for assertion in assertions],
                "message": "Processed using fallback method"
            }

        except Exception as e:
            logger.error(f"Error in envelope fallback processing: {e}")
            return {
                "status": "error",
                "error": str(e),
                "message": "Fallback processing failed"
            }

    async def get_event_history(self, patient_id: str,
                              event_types: Optional[List[str]] = None,
                              days_back: int = 30) -> List[Dict[str, Any]]:
        """
        Get clinical event history for patient

        Args:
            patient_id: Patient identifier
            event_types: Optional filter for event types
            days_back: Number of days to look back

        Returns:
            List of clinical events
        """
        try:
            if not self.event_store:
                return []

            # Calculate time range
            end_time = datetime.utcnow()
            start_time = end_time - timedelta(days=days_back)

            # Get events from event store
            events = await self.event_store.get_events_by_patient(
                patient_id, event_types, start_time, end_time
            )

            # Convert to response format
            event_history = []
            for event in events:
                event_history.append({
                    "event_id": event.metadata.event_id,
                    "event_type": event.metadata.event_type.value,
                    "event_time": event.temporal_context.event_time.isoformat(),
                    "event_status": event.metadata.event_status.value,
                    "event_severity": event.metadata.event_severity.value,
                    "event_data": event.event_data,
                    "clinical_context": {
                        "encounter_id": event.clinical_context.encounter_id,
                        "facility_id": event.clinical_context.facility_id,
                        "primary_provider_id": event.clinical_context.primary_provider_id
                    }
                })

            logger.info(f"Retrieved {len(event_history)} events for patient {patient_id}")
            return event_history

        except Exception as e:
            logger.error(f"Error getting event history: {e}")
            return []

    async def get_system_status(self) -> Dict[str, Any]:
        """Get comprehensive system status"""
        queue_status = await self.priority_queue.get_queue_status()

        status = {
            'orchestration_engine': {
                'status': 'running' if self.queue_processor_task and not self.queue_processor_task.done() else 'stopped',
                'registered_reasoners': list(self.reasoner_instances.keys()),
                'performance': self.performance_stats.copy()
            },
            'request_router': {
                'stats': self.request_router.get_stats()
            },
            'parallel_executor': {
                'stats': self.parallel_executor.get_stats()
            },
            'decision_aggregator': {
                'stats': self.decision_aggregator.get_stats()
            },
            'priority_queue': queue_status
        }

        # Add intelligence component status
        if self.rule_engine:
            status['intelligence'] = {
                'rule_engine': self.rule_engine.get_performance_metrics(),
                'performance_optimizer': self.performance_optimizer.get_performance_metrics() if self.performance_optimizer else None,
                'confidence_evolver': self.confidence_evolver.get_confidence_statistics() if self.confidence_evolver else None,
                'pattern_learner': self.pattern_learner.get_learning_statistics() if self.pattern_learner else None
            }

        # Add event envelope system status
        if self.event_processor_registry:
            status['event_envelope_system'] = {
                'event_processor_registry': self.event_processor_registry.get_registry_stats(),
                'event_store': self.event_store.get_store_stats() if self.event_store else None,
                'idempotency_manager': self.idempotency_manager.get_idempotency_stats() if self.idempotency_manager else None,
                'context_enrichment_engine': self.context_enrichment_engine.get_engine_stats() if self.context_enrichment_engine else None
            }

        return status
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform health check"""
        health_status = {
            'status': 'healthy',
            'timestamp': datetime.utcnow().isoformat(),
            'components': {}
        }
        
        # Check queue processor
        if not self.queue_processor_task or self.queue_processor_task.done():
            health_status['status'] = 'unhealthy'
            health_status['components']['queue_processor'] = 'stopped'
        else:
            health_status['components']['queue_processor'] = 'running'
        
        # Check reasoner availability
        if not self.reasoner_instances:
            health_status['status'] = 'degraded'
            health_status['components']['reasoners'] = 'none_registered'
        else:
            health_status['components']['reasoners'] = f"{len(self.reasoner_instances)}_registered"
        
        # Check queue status
        queue_status = await self.priority_queue.get_queue_status()
        if queue_status['queue_utilization'] > 0.9:
            health_status['status'] = 'degraded'
            health_status['components']['queue'] = 'high_utilization'
        else:
            health_status['components']['queue'] = 'normal'
        
        return health_status
