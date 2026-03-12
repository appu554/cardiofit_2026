"""
Clinical Reasoning Service gRPC Server

This module implements the gRPC server for the Clinical Assertion Engine,
following the established patterns from the Global Outbox Service.
"""

import grpc
import asyncio
import logging
import sys
import os
from concurrent import futures
from typing import AsyncIterator, Dict, Any, List, Optional
from datetime import datetime

# Import the generated protobuf files. They are in the same 'app' directory.
# The 'start_cae_server.py' script adds the 'app' directory to the path,
# so we can import them directly.
try:
    import clinical_reasoning_pb2
    import clinical_reasoning_pb2_grpc
    from clinical_reasoning_pb2 import (
        ClinicalAssertion,
        ClinicalAssertionRequest,
        ClinicalAssertionResponse,
        ClinicalRecommendation,
        AssertionSeverity,
        AssertionPriority,
        RecommendationPriority,
        RecommendationPriority
    )
except ImportError as e:
    logging.basicConfig(level=logging.INFO)
    logging.warning("⚠️  Protocol buffer files not found or import error. Run 'python compile_proto.py' first.")
    logging.error(f"Import error: {e}")
    clinical_reasoning_pb2 = None
    clinical_reasoning_pb2_grpc = None

# Import Timestamp
from google.protobuf.timestamp_pb2 import Timestamp

# Configuration and other imports
try:
    from core.config import settings
    from learning.learning_manager import learning_manager
    from graph.graphdb_client import graphdb_client
except ImportError:
    # Mock objects for fallback or testing environments
    class MockLearningManager:
        async def initialize(self): pass
        async def track_clinical_outcome(self, **kwargs): return True
        async def track_clinician_override(self, **kwargs): return True

    learning_manager = MockLearningManager()
    graphdb_client = None

    class Settings:
        PROJECT_NAME = "Clinical Reasoning Service"
        VERSION = "1.0.0"
        GRPC_PORT = 8027
        GRPC_MAX_WORKERS = 10
        LOG_LEVEL = "INFO"

    settings = Settings()

logger = logging.getLogger(__name__)

class ClinicalReasoningServicer(clinical_reasoning_pb2_grpc.ClinicalReasoningServiceServicer):
    """
    gRPC service implementation for Clinical Assertion Engine
    
    Provides high-performance clinical reasoning capabilities for other microservices.
    Follows the established patterns from Global Outbox Service.
    """
    
    # Severity mapping from string to protobuf enum
    SEVERITY_MAP = {
        'info': AssertionSeverity.SEVERITY_INFO,
        'low': AssertionSeverity.SEVERITY_LOW,
        'moderate': AssertionSeverity.SEVERITY_MODERATE,
        'high': AssertionSeverity.SEVERITY_HIGH,
        'critical': AssertionSeverity.SEVERITY_CRITICAL
    }

    def __init__(self):
        self.service_name = settings.PROJECT_NAME
        self.version = settings.VERSION

        # Initialize CAE Engine with Neo4j
        try:
            from app.cae_engine_neo4j import CAEEngine
            self.cae_engine = CAEEngine()
            logger.info("✓ CAE Engine with Neo4j initialized")
        except ImportError as e:
            logger.error(f"Failed to load CAE engine: {e}")
            self.cae_engine = None

        # Initialize real clinical reasoners
        try:
            from app.reasoners.medication_interaction import MedicationInteractionReasoner
            from app.reasoners.dosing_calculator import DosingCalculator
            from app.reasoners.contraindication import ContraindicationReasoner
            from app.reasoners.duplicate_therapy import DuplicateTherapyReasoner
            from app.reasoners.clinical_context import ClinicalContextReasoner
            from app.context.patient_context_assembler import PatientContextAssembler
            from app.events.clinical_context_assembler import ContextEnrichmentEngine
            from app.cache.redis_client import CAERedisClient

            # Initialize CAE Redis client
            self.redis_client = CAERedisClient()

            # Initialize and configure the Context Enrichment Engine
            self.context_enrichment_engine = ContextEnrichmentEngine()
            self.context_enrichment_engine.default_assembler.graphdb_client = graphdb_client
            # Context enrichment is now handled by CAE Engine
            logger.info("✓ Context enrichment handled by CAE Engine")

            # Initialize clinical reasoners
            self.medication_interaction_reasoner = MedicationInteractionReasoner()
            self.dosing_calculator = DosingCalculator()
            self.contraindication_reasoner = ContraindicationReasoner()
            self.duplicate_therapy_reasoner = DuplicateTherapyReasoner()
            self.clinical_context_reasoner = ClinicalContextReasoner()
            
            # Initialize patient context assembler
            self.context_assembler = PatientContextAssembler(redis_client=self.redis_client)
            
            # The Context Enrichment Engine is already initialized above with GraphDB client
            # so we don't need to reinitialize it here
            logger.info("✓ Context Enrichment Engine already initialized with GraphDB client")

            # Clinical reasoning is now handled by CAE Engine with Neo4j
            logger.info("✓ Clinical reasoning handled by CAE Engine with Neo4j integration")

            logger.info("✓ Real clinical reasoners, context assembler, and Redis cache initialized")
        except ImportError as e:
            logger.error(f"CRITICAL: Failed to load real reasoners, server cannot start. Error: {e}", exc_info=True)
            raise

        logger.info(f"Initialized {self.__class__.__name__} v{self.version}")
    
    async def GenerateAssertions(self, request, context):
        """
        Generate comprehensive clinical assertions for a patient context

        This is the main entry point for clinical reasoning requests.
        """
        start_time = datetime.utcnow()

        try:
            logger.info(f"Generating assertions for patient {request.patient_id}")

            # Convert gRPC request to orchestration format
            # Extract allergy_ids from patient_context instead of direct field
            patient_context_dict = {}
            if request.patient_context:
                patient_context_dict = dict(request.patient_context)

            allergy_ids = patient_context_dict.get('allergy_ids', [])
            if isinstance(allergy_ids, str):
                allergy_ids = [allergy_ids]
            elif not isinstance(allergy_ids, list):
                allergy_ids = []

            raw_request = {
                'patient_id': request.patient_id,
                'correlation_id': request.correlation_id or f"corr_{request.patient_id}",
                'reasoner_types': list(request.reasoner_types) or ["interaction", "dosing", "contraindication"],
                'medication_ids': list(request.medication_ids) if request.medication_ids else [],
                'condition_ids': list(request.condition_ids) if request.condition_ids else [],
                'allergy_ids': allergy_ids,
                'clinical_context': self._extract_clinical_context(request),
                'temporal_context': self._extract_temporal_context(request)
            }

            # Use CAE Engine if available
            reasoner_errors = []
            if self.cae_engine:
                try:
                    # Convert to CAE Engine format
                    # Handle both Safety Gateway format (medication_ids) and direct format (medications)
                    if hasattr(request, 'medication_ids') and request.medication_ids:
                        # Safety Gateway format - convert medication_ids to medication objects
                        medications = []
                        for med_id in request.medication_ids:
                            # Parse medication ID (e.g., "warfarin_5mg" -> name="warfarin", dose="5mg")
                            if '_' in med_id:
                                name, dose = med_id.rsplit('_', 1)
                                medications.append({
                                    'name': name,
                                    'dose': dose,
                                    'frequency': 'unknown'
                                })
                            else:
                                medications.append({
                                    'name': med_id,
                                    'dose': 'unknown',
                                    'frequency': 'unknown'
                                })
                    elif hasattr(request, 'medications') and request.medications:
                        # Direct format - use medication objects directly
                        medications = [
                            {
                                'name': med.name,
                                'dose': med.dose,
                                'frequency': getattr(med, 'frequency', 'unknown')
                            }
                            for med in request.medications
                        ]
                    else:
                        medications = []

                    # Handle conditions
                    if hasattr(request, 'condition_ids') and request.condition_ids:
                        conditions = [{'name': cond_id} for cond_id in request.condition_ids]
                    elif hasattr(request, 'conditions') and request.conditions:
                        conditions = [{'name': cond.name} for cond in request.conditions]
                    else:
                        conditions = []

                    # Handle allergies
                    if hasattr(request, 'allergy_ids') and request.allergy_ids:
                        allergies = [{'substance': allergy_id, 'reaction': 'unknown', 'severity': 'unknown'}
                                   for allergy_id in request.allergy_ids]
                    elif hasattr(request, 'allergies') and request.allergies:
                        allergies = [
                            {
                                'substance': allergy.substance,
                                'reaction': getattr(allergy, 'reaction', 'unknown'),
                                'severity': getattr(allergy, 'severity', 'unknown')
                            }
                            for allergy in request.allergies
                        ]
                    else:
                        allergies = []

                    # Get patient context
                    patient_context = getattr(request, 'context', {})

                    clinical_context = {
                        'patient': {
                            'id': request.patient_id,
                            'age': int(patient_context.get('patient_age', 0)) if patient_context.get('patient_age') else 0,
                            'weight': float(patient_context.get('patient_weight', 0)) if patient_context.get('patient_weight') else 0,
                            'gender': patient_context.get('patient_gender', 'unknown')
                        },
                        'medications': medications,
                        'conditions': conditions,
                        'allergies': allergies
                    }

                    # Get clinical safety validation from CAE Engine
                    cae_result = await self.cae_engine.validate_safety(clinical_context)

                    # Convert CAE Engine findings to protobuf format
                    pb_assertions = []
                    for finding in cae_result.get('findings', []):
                        try:
                            # Convert CAE finding to protobuf format
                            severity_str = finding.get('severity', 'MEDIUM')
                            severity_value = self.SEVERITY_MAP.get(severity_str.lower(), AssertionSeverity.SEVERITY_UNSPECIFIED)

                            # Create ClinicalRecommendation object instead of string
                            recommendation_text = finding.get('recommendation', 'No recommendation')
                            clinical_recommendation = ClinicalRecommendation(
                                id=f"rec_{len(pb_assertions)}",
                                type="clinical_guidance",
                                description=recommendation_text,
                                rationale=finding.get('evidence', {}).get('source', 'Clinical reasoning'),
                                priority=RecommendationPriority.RECOMMENDATION_PRIORITY_RECOMMENDED
                            )

                            pb_assertion = ClinicalAssertion(
                                id=finding.get('finding_id', f"finding_{len(pb_assertions)}"),
                                type=finding.get('finding_type', 'CLINICAL_FINDING'),
                                severity=severity_value,
                                title=finding.get('message', 'Clinical Finding'),
                                description=finding.get('message', 'No description provided'),
                                explanation=finding.get('recommendation', 'No explanation provided'),
                                confidence_score=finding.get('evidence', {}).get('confidence', 0.5),
                                evidence_sources=[finding.get('evidence', {}).get('source', 'CAE Engine')],
                                recommendations=[clinical_recommendation]
                            )
                            pb_assertions.append(pb_assertion)
                        except Exception as e:
                            logger.error(f"Error converting finding to protobuf: {e}")
                            reasoner_errors.append(f"Failed to convert finding to protobuf: {str(e)}")

                    processing_time = cae_result.get('performance', {}).get('total_execution_time_ms', 0)

                    # If we got 0 assertions, add a warning
                    if len(pb_assertions) == 0:
                        reasoner_errors.append("No clinical findings generated by CAE Engine")
                        logger.warning(f"Generated 0 findings for patient {request.patient_id}")

                    # Add overall status information
                    overall_status = cae_result.get('overall_status', 'UNKNOWN')
                    logger.info(f"CAE Engine result for patient {request.patient_id}: {overall_status} with {len(pb_assertions)} findings")

                except Exception as e:
                    logger.error(f"CAE Engine error: {e}")
                    reasoner_errors.append(f"CAE Engine error: {str(e)}")
                    pb_assertions = []
                    processing_time = (datetime.utcnow() - start_time).total_seconds() * 1000

            else:
                # Fallback to mock response
                logger.warning("CAE Engine not available, using mock response")
                pb_assertions = self._create_mock_assertions(request.patient_id, raw_request.get('reasoner_types', []))
                processing_time = 50
                reasoner_errors.append("CAE Engine not available - using mock response")

            # Create metadata with reasoner errors as warnings
            metadata = clinical_reasoning_pb2.AssertionMetadata(
                reasoner_version=self.version,
                knowledge_version="1.0.0",
                processing_time_ms=int(processing_time),
                warnings=reasoner_errors
            )

            # Create timestamp
            generated_at = Timestamp()
            generated_at.FromDatetime(datetime.utcnow())

            # Build response
            response = clinical_reasoning_pb2.ClinicalAssertionResponse(
                request_id=f"req_{raw_request['correlation_id']}",
                correlation_id=raw_request['correlation_id'],
                assertions=pb_assertions,
                metadata=metadata,
                generated_at=generated_at
            )

            logger.info(f"Generated {len(pb_assertions)} assertions for patient {request.patient_id} "
                       f"in {processing_time:.2f}ms")
            return response

        except Exception as e:
            logger.error(f"Error generating assertions: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return clinical_reasoning_pb2.ClinicalAssertionResponse()
    
    async def CheckMedicationInteractions(self, request, context):
        """
        Check for medication interactions using real clinical logic
        """
        start_time = datetime.now()
        try:
            logger.info(f"Checking medication interactions for patient {request.patient_id}")

            # Use real medication interaction reasoner if available
            if self.medication_interaction_reasoner:
                # Get complete patient context from GraphDB
                patient_context = {}
                if hasattr(self, 'context_assembler'):
                    try:
                        full_context = await self.context_assembler.get_patient_context(request.patient_id)
                        patient_context = {
                            "age": full_context.demographics.get("age", 0),
                            "weight": full_context.demographics.get("weight", 70),
                            "gender": full_context.demographics.get("gender", "unknown"),
                            "kidney_function": full_context.demographics.get("kidney_function", "normal"),
                            "liver_function": full_context.demographics.get("liver_function", "normal"),
                            "pregnancy_status": full_context.demographics.get("pregnancy_status", "unknown"),
                            "active_conditions": [c.get("code", "") for c in full_context.active_conditions],
                            "current_medications": [m.get("code", "") for m in full_context.current_medications],
                            "allergies": [a.get("substance", "") for a in full_context.allergies],
                            "risk_factors": full_context.risk_factors
                        }
                        logger.info(f"Using complete patient context for {request.patient_id}")
                    except Exception as e:
                        logger.warning(f"Failed to get patient context: {e}")
                        # Fallback to request context
                        if request.patient_context:
                            for key, value in request.patient_context.fields.items():
                                patient_context[key] = self._extract_struct_value(value)

                # Check interactions using real clinical logic with full context
                interactions_data = await self.medication_interaction_reasoner.check_interactions(
                    patient_id=request.patient_id,
                    medication_ids=list(request.medication_ids),
                    new_medication_id=request.new_medication_id if request.new_medication_id else None,
                    patient_context=patient_context
                )

                # Convert to protobuf format
                interactions = []
                # Handle new reasoner response format
                if isinstance(interactions_data, dict) and 'assertions' in interactions_data:
                    interaction_list = interactions_data['assertions']
                else:
                    interaction_list = interactions_data

                for interaction_data in interaction_list:
                    interaction = clinical_reasoning_pb2.DrugInteraction(
                        interaction_id=interaction_data["interaction_id"],
                        medication_a=interaction_data["medication_a"],
                        medication_b=interaction_data["medication_b"],
                        severity=self._severity_to_protobuf(interaction_data["severity"]),
                        description=interaction_data["description"],
                        mechanism=interaction_data["mechanism"],
                        clinical_effect=interaction_data["clinical_effect"],
                        evidence_sources=interaction_data["evidence_sources"],
                        confidence_score=interaction_data["confidence_score"]
                    )
                    interactions.append(interaction)
            else:
                # Fallback to mock response
                interactions = self._create_mock_interactions(request.medication_ids)

            # Calculate processing time
            processing_time = int((datetime.now() - start_time).total_seconds() * 1000)

            # Create metadata
            metadata = clinical_reasoning_pb2.AssertionMetadata(
                reasoner_version=self.version,
                knowledge_version="clinical_db_v1.0.0",
                processing_time_ms=processing_time,
                warnings=[]
            )

            response = clinical_reasoning_pb2.MedicationInteractionResponse(
                interactions=interactions,
                metadata=metadata
            )

            logger.info(f"Found {len(interactions)} interactions for patient {request.patient_id}")
            return response

        except Exception as e:
            logger.error(f"Error checking interactions: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return clinical_reasoning_pb2.MedicationInteractionResponse()
    
    async def CalculateDosing(self, request, context):
        """
        Calculate medication dosing recommendations using real clinical logic
        """
        start_time = datetime.now()
        try:
            logger.info(f"Calculating dosing for patient {request.patient_id}, medication {request.medication_id}")

            # Use real dosing calculator if available
            if self.dosing_calculator:
                # Convert patient parameters from protobuf Struct to dict
                patient_parameters = {}
                if request.patient_parameters:
                    for key, value in request.patient_parameters.fields.items():
                        patient_parameters[key] = self._extract_struct_value(value)

                # Calculate dosing using real clinical logic
                dosing_recommendation = await self.dosing_calculator.calculate_dosing(
                    patient_id=request.patient_id,
                    medication_id=request.medication_id,
                    patient_parameters=patient_parameters,
                    indication=request.indication if request.indication else None
                )

                # Convert to protobuf format
                dosing = clinical_reasoning_pb2.DosingRecommendation(
                    medication_id=dosing_recommendation.medication_id,
                    dose=dosing_recommendation.dose,
                    frequency=dosing_recommendation.frequency,
                    route=dosing_recommendation.route,
                    duration=dosing_recommendation.duration,
                    rationale=dosing_recommendation.rationale,
                    warnings=dosing_recommendation.warnings
                )

                # Convert adjustments
                adjustments = []
                for adj in dosing_recommendation.adjustments:
                    adjustment = clinical_reasoning_pb2.DosingAdjustment(
                        type=adj.type,
                        adjustment=adj.adjustment,
                        rationale=adj.rationale,
                        required=adj.required
                    )
                    adjustments.append(adjustment)
            else:
                # Fallback to mock response
                dosing = self._create_mock_dosing(request.medication_id)
                adjustments = []

            # Calculate processing time
            processing_time = int((datetime.now() - start_time).total_seconds() * 1000)

            metadata = clinical_reasoning_pb2.AssertionMetadata(
                reasoner_version=self.version,
                knowledge_version="clinical_db_v1.0.0",
                processing_time_ms=processing_time,
                warnings=[]
            )

            response = clinical_reasoning_pb2.DosingCalculationResponse(
                dosing=dosing,
                adjustments=adjustments,
                metadata=metadata
            )

            return response

        except Exception as e:
            logger.error(f"Error calculating dosing: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return clinical_reasoning_pb2.DosingCalculationResponse()
    
    async def CheckContraindications(self, request, context):
        """
        Check for contraindications using real clinical logic
        """
        start_time = datetime.now()
        try:
            logger.info(f"Checking contraindications for patient {request.patient_id}")

            # Use real contraindication reasoner if available
            if self.contraindication_reasoner:
                # Convert patient context from protobuf Struct to dict
                patient_context = {}
                if request.patient_context:
                    for key, value in request.patient_context.fields.items():
                        patient_context[key] = self._extract_struct_value(value)

                # Extract allergy_ids from patient_context
                allergy_ids = patient_context.get('allergy_ids', [])
                if isinstance(allergy_ids, str):
                    allergy_ids = [allergy_ids]
                elif not isinstance(allergy_ids, list):
                    allergy_ids = []

                # Check contraindications using real clinical logic
                contraindications_data = await self.contraindication_reasoner.check_contraindications(
                    patient_id=request.patient_id,
                    medication_ids=list(request.medication_ids),
                    condition_ids=list(request.condition_ids) if request.condition_ids else None,
                    allergy_ids=allergy_ids,
                    patient_context=patient_context
                )

                # Convert to protobuf format
                contraindications = []
                # Handle new reasoner response format
                if isinstance(contraindications_data, dict) and 'assertions' in contraindications_data:
                    contraindication_list = contraindications_data['assertions']
                else:
                    contraindication_list = contraindications_data

                for contraindication_data in contraindication_list:
                    contraindication = clinical_reasoning_pb2.Contraindication(
                        contraindication_id=contraindication_data["contraindication_id"],
                        medication_id=contraindication_data["medication_id"],
                        condition_id=contraindication_data["condition_id"],
                        severity=self._severity_to_protobuf(contraindication_data["severity"]),
                        type=contraindication_data["type"],
                        description=contraindication_data["description"],
                        rationale=contraindication_data["rationale"],
                        evidence_sources=contraindication_data["evidence_sources"],
                        override_possible=contraindication_data["override_possible"],
                        override_rationale=contraindication_data["override_rationale"]
                    )
                    contraindications.append(contraindication)
            else:
                # Fallback to mock response
                contraindications = self._create_mock_contraindications(request.medication_ids)

            # Calculate processing time
            processing_time = int((datetime.now() - start_time).total_seconds() * 1000)

            metadata = clinical_reasoning_pb2.AssertionMetadata(
                reasoner_version=self.version,
                knowledge_version="clinical_db_v1.0.0",
                processing_time_ms=processing_time,
                warnings=[]
            )

            response = clinical_reasoning_pb2.ContraindicationResponse(
                contraindications=contraindications,
                metadata=metadata
            )

            return response

        except Exception as e:
            logger.error(f"Error checking contraindications: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return clinical_reasoning_pb2.ContraindicationResponse()
    
    async def HealthCheck(self, request, context):
        """
        Health check for service monitoring
        """
        try:
            # TODO: Add actual health checks (database, external services, etc.)
            response = clinical_reasoning_pb2.HealthCheckResponse(
                status=clinical_reasoning_pb2.HealthCheckResponse.ServingStatus.SERVING
            )
            return response
            
        except Exception as e:
            logger.error(f"Health check failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Health check failed: {str(e)}")
            return clinical_reasoning_pb2.HealthCheckResponse(
                status=clinical_reasoning_pb2.HealthCheckResponse.ServingStatus.NOT_SERVING
            )
    
    async def StreamAssertions(self, request, context) -> AsyncIterator[clinical_reasoning_pb2.ClinicalAssertionUpdate]:
        """
        Stream real-time assertion updates for complex reasoning
        """
        try:
            request_id = f"stream_{request.correlation_id}"
            logger.info(f"Starting assertion stream for request {request_id}")
            
            # TODO: Implement actual streaming logic
            # For now, send a few mock updates
            for i in range(3):
                await asyncio.sleep(1)  # Simulate processing time
                
                mock_assertion = self._create_mock_assertion(f"stream_assertion_{i}")
                
                yield clinical_reasoning_pb2.ClinicalAssertionUpdate(
                    request_id=request_id,
                    type=clinical_reasoning_pb2.ClinicalAssertionUpdate.UpdateType.UPDATE_TYPE_PARTIAL,
                    assertion=mock_assertion,
                    status=f"Processing step {i+1}/3"
                )
            
            # Send final update
            final_assertion = self._create_mock_assertion("final_assertion")
            yield clinical_reasoning_pb2.ClinicalAssertionUpdate(
                request_id=request_id,
                type=clinical_reasoning_pb2.ClinicalAssertionUpdate.UpdateType.UPDATE_TYPE_COMPLETE,
                assertion=final_assertion,
                status="Complete"
            )
                
        except Exception as e:
            logger.error(f"Error in streaming assertions: {e}")
            yield clinical_reasoning_pb2.ClinicalAssertionUpdate(
                request_id=request_id,
                type=clinical_reasoning_pb2.ClinicalAssertionUpdate.UpdateType.UPDATE_TYPE_ERROR,
                status=f"Error: {str(e)}"
            )
    
    def _create_mock_assertions(self, patient_id: str, reasoner_types: list) -> list:
        """Create mock assertions for testing"""
        assertions = []
        
        for reasoner_type in reasoner_types:
            assertion = self._create_mock_assertion(f"{reasoner_type}_assertion")
            assertions.append(assertion)
        
        return assertions
    
    def _create_mock_assertion(self, assertion_id: str) -> clinical_reasoning_pb2.ClinicalAssertion:
        """Create a mock clinical assertion"""
        return clinical_reasoning_pb2.ClinicalAssertion(
            id=assertion_id,
            type="interaction",
            severity=clinical_reasoning_pb2.AssertionSeverity.SEVERITY_MODERATE,
            title="Mock Clinical Assertion",
            description="This is a mock assertion for testing purposes",
            explanation="Mock explanation of the clinical reasoning",
            evidence_sources=["Mock Evidence Source 1", "Mock Evidence Source 2"],
            confidence_score=0.85,
            recommendations=[
                clinical_reasoning_pb2.ClinicalRecommendation(
                    id="rec_1",
                    type="monitor",
                    description="Monitor patient closely",
                    rationale="Mock rationale for monitoring",
                    priority=clinical_reasoning_pb2.RecommendationPriority.RECOMMENDATION_PRIORITY_RECOMMENDED
                )
            ]
        )
    
    def _create_mock_interactions(self, medication_ids: list) -> list:
        """Create mock drug interactions"""
        if len(medication_ids) < 2:
            return []
        
        return [
            clinical_reasoning_pb2.DrugInteraction(
                interaction_id="mock_interaction_1",
                medication_a=medication_ids[0],
                medication_b=medication_ids[1] if len(medication_ids) > 1 else "mock_med",
                severity=clinical_reasoning_pb2.AssertionSeverity.SEVERITY_MODERATE,
                description="Mock drug interaction",
                mechanism="Mock interaction mechanism",
                clinical_effect="Mock clinical effect",
                evidence_sources=["Mock Drug Database"],
                confidence_score=0.8
            )
        ]

    def _extract_struct_value(self, value):
        """Extract value from protobuf Struct Value"""
        if value.HasField('string_value'):
            return value.string_value
        elif value.HasField('number_value'):
            return value.number_value
        elif value.HasField('bool_value'):
            return value.bool_value
        elif value.HasField('list_value'):
            return [self._extract_struct_value(v) for v in value.list_value.values]
        elif value.HasField('struct_value'):
            return {k: self._extract_struct_value(v) for k, v in value.struct_value.fields.items()}
        else:
            return None

    def _extract_clinical_context(self, request) -> Dict[str, Any]:
        """Extract clinical context from gRPC request"""
        clinical_context = {}

        if hasattr(request, 'patient_context') and request.patient_context:
            for key, value in request.patient_context.fields.items():
                clinical_context[key] = self._extract_struct_value(value)

        # Add encounter type if available
        if hasattr(request, 'encounter_type') and request.encounter_type:
            clinical_context['encounter_type'] = request.encounter_type

        return clinical_context

    def _extract_temporal_context(self, request) -> Dict[str, Any]:
        """Extract temporal context from gRPC request"""
        temporal_context = {}

        # Add current timestamp
        temporal_context['request_time'] = datetime.utcnow().isoformat()

        # Add any temporal fields from request
        if hasattr(request, 'temporal_context') and request.temporal_context:
            for key, value in request.temporal_context.fields.items():
                temporal_context[key] = self._extract_struct_value(value)

        return temporal_context

    def _severity_to_protobuf(self, severity_str: str):
        """Convert severity string to protobuf enum"""
        severity_map = {
            "critical": AssertionSeverity.SEVERITY_CRITICAL,
            "high": AssertionSeverity.SEVERITY_HIGH,
            "moderate": AssertionSeverity.SEVERITY_MODERATE,
            "low": AssertionSeverity.SEVERITY_LOW,
            "info": AssertionSeverity.SEVERITY_INFO
        }
        return severity_map.get(severity_str, AssertionSeverity.SEVERITY_MODERATE)

    def _create_mock_dosing(self, medication_id: str):
        """Create mock dosing recommendation"""
        return clinical_reasoning_pb2.DosingRecommendation(
            medication_id=medication_id,
            dose="10mg",
            frequency="twice daily",
            route="oral",
            duration="7 days",
            rationale="Mock dosing rationale",
            warnings=["Mock dosing warning"]
        )
    
    def _create_mock_contraindications(self, medication_ids: list) -> list:
        """Create mock contraindications"""
        return [
            clinical_reasoning_pb2.Contraindication(
                contraindication_id="mock_contraindication_1",
                medication_id=medication_ids[0] if medication_ids else "mock_med",
                condition_id="mock_condition",
                severity=AssertionSeverity.SEVERITY_HIGH,
                type="relative",
                description="Mock contraindication",
                rationale="Mock contraindication rationale",
                evidence_sources=["Mock Clinical Guidelines"],
                override_possible=True,
                override_rationale="Mock override rationale"
            )
        ]


async def serve_grpc():
    """
    Start the Clinical Reasoning gRPC server

    Follows the established pattern from Global Outbox Service.
    """
    if not clinical_reasoning_pb2 or not clinical_reasoning_pb2_grpc:
        raise RuntimeError("Protocol buffer files not available. Run 'python compile_proto.py' first.")

    # Create gRPC server
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=settings.GRPC_MAX_WORKERS)
    )

    # Create servicer instance
    servicer = ClinicalReasoningServicer()

    # Add servicer
    clinical_reasoning_pb2_grpc.add_ClinicalReasoningServiceServicer_to_server(
        servicer, server
    )

    # Initialize CAE Engine if available
    if servicer.cae_engine:
        initialized = await servicer.cae_engine.initialize()
        if initialized:
            logger.info("✓ CAE Engine initialized and ready")
        else:
            logger.error("❌ CAE Engine initialization failed")

    # Configure server address
    listen_addr = f'[::]:{settings.GRPC_PORT}'
    server.add_insecure_port(listen_addr)

    logger.info(f"🚀 Starting Clinical Reasoning gRPC server on {listen_addr}")
    await server.start()

    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down gRPC server...")

        # Close CAE Engine
        if servicer.cae_engine:
            await servicer.cae_engine.close()
            logger.info("✓ CAE Engine closed")

        await server.stop(grace=5)


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=getattr(logging, settings.LOG_LEVEL),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Run the gRPC server
    asyncio.run(serve_grpc())
