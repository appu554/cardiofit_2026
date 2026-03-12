"""
Clinical Assertion Engine gRPC Client Library

This client library provides a simple interface for microservices to request
clinical assertions from the Clinical Reasoning Service. It follows the
established patterns from the Global Outbox Service client.
"""

import grpc
import asyncio
import logging
from typing import List, Dict, Any, Optional, AsyncIterator
import uuid

logger = logging.getLogger(__name__)

# Try to import gRPC protocol buffers
GRPC_AVAILABLE = True
clinical_reasoning_pb2 = None
clinical_reasoning_pb2_grpc = None

try:
    # Try to import from shared directory first
    import clinical_reasoning_pb2
    import clinical_reasoning_pb2_grpc
    logger.info("✓ Clinical Reasoning gRPC protocol buffers loaded successfully")

except ImportError as e:
    logger.warning(f"gRPC protocol buffers not available: {e}")
    logger.warning("Protocol buffers not found in shared directory")
    GRPC_AVAILABLE = False
except Exception as e:
    logger.warning(f"Unexpected error loading gRPC protocol buffers: {e}")
    GRPC_AVAILABLE = False


class CAEgRPCClient:
    """
    gRPC client for Clinical Assertion Engine
    
    Provides high-level interface for microservices to request clinical assertions.
    Follows the established patterns from GlobalOutboxClient.
    """
    
    def __init__(self, cae_service_url: str = "localhost:8027", service_name: str = "unknown-service"):
        self.cae_service_url = cae_service_url
        self.service_name = service_name
        self._channel = None
        self._stub = None
        self.connected = False
        
        logger.info(f"Initialized CAE gRPC client for {service_name} -> {cae_service_url}")
    
    async def __aenter__(self):
        """Async context manager entry"""
        await self._connect()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self._disconnect()
    
    async def _connect(self):
        """Establish gRPC connection to CAE service"""
        if not GRPC_AVAILABLE:
            logger.warning("gRPC not available, skipping connection")
            return
            
        try:
            self._channel = grpc.aio.insecure_channel(self.cae_service_url)
            self._stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(self._channel)
            
            # Test connection with health check
            await self.health_check()
            self.connected = True
            logger.info(f"✓ Connected to CAE service at {self.cae_service_url}")
            
        except Exception as e:
            logger.error(f"Failed to connect to CAE service: {e}")
            if self._channel:
                await self._channel.close()
                self._channel = None
                self._stub = None
            raise
    
    async def _disconnect(self):
        """Close gRPC connection"""
        if self._channel:
            await self._channel.close()
            self._channel = None
            self._stub = None
            self.connected = False
            logger.info("Disconnected from CAE service")
    
    async def health_check(self) -> bool:
        """Check if CAE service is healthy"""
        if not self._stub:
            return False
            
        try:
            request = clinical_reasoning_pb2.HealthCheckRequest(service=self.service_name)
            response = await self._stub.HealthCheck(request)
            
            is_serving = response.status == clinical_reasoning_pb2.HealthCheckResponse.ServingStatus.SERVING
            logger.debug(f"CAE health check: {'✓' if is_serving else '✗'}")
            return is_serving
            
        except Exception as e:
            logger.error(f"CAE health check failed: {e}")
            return False
    
    async def generate_clinical_assertions(
        self,
        patient_id: str,
        medication_ids: List[str] = None,
        condition_ids: List[str] = None,
        reasoner_types: List[str] = None,
        correlation_id: str = None,
        priority: str = "standard"
    ) -> Dict[str, Any]:
        """
        Generate comprehensive clinical assertions
        
        Args:
            patient_id: Patient identifier
            medication_ids: List of medication identifiers
            condition_ids: List of condition identifiers
            reasoner_types: Types of reasoners to use ["interaction", "dosing", "contraindication"]
            correlation_id: Correlation ID for tracing
            priority: Request priority ("critical", "urgent", "standard", "batch")
            
        Returns:
            Dictionary containing clinical assertions and metadata
        """
        if not self._stub:
            raise RuntimeError("Not connected to CAE service")
        
        # Build request
        request = clinical_reasoning_pb2.ClinicalAssertionRequest(
            patient_id=patient_id,
            correlation_id=correlation_id or f"corr_{uuid.uuid4().hex[:8]}",
            medication_ids=medication_ids or [],
            condition_ids=condition_ids or [],
            reasoner_types=reasoner_types or ["interaction", "dosing", "contraindication"],
            priority=self._convert_priority(priority)
        )
        
        try:
            response = await self._stub.GenerateAssertions(request)
            return self._convert_response_to_dict(response)
            
        except grpc.RpcError as e:
            logger.error(f"gRPC error generating assertions: {e}")
            raise RuntimeError(f"CAE service error: {e.details()}")
        except Exception as e:
            logger.error(f"Error generating assertions: {e}")
            raise
    
    async def check_medication_interactions(
        self,
        patient_id: str,
        medication_ids: List[str],
        new_medication_id: str = None,
        patient_context: Dict[str, Any] = None
    ) -> Dict[str, Any]:
        """
        Check for medication interactions
        
        Args:
            patient_id: Patient identifier
            medication_ids: List of current medication identifiers
            new_medication_id: New medication to check against current medications
            patient_context: Additional patient context
            
        Returns:
            Dictionary containing interaction results
        """
        if not self._stub:
            raise RuntimeError("Not connected to CAE service")
        
        request = clinical_reasoning_pb2.MedicationInteractionRequest(
            patient_id=patient_id,
            medication_ids=medication_ids,
            new_medication_id=new_medication_id or ""
        )
        
        try:
            response = await self._stub.CheckMedicationInteractions(request)
            return self._convert_interaction_response_to_dict(response)
            
        except grpc.RpcError as e:
            logger.error(f"gRPC error checking interactions: {e}")
            raise RuntimeError(f"CAE service error: {e.details()}")
        except Exception as e:
            logger.error(f"Error checking interactions: {e}")
            raise

    async def calculate_dosing(
        self,
        patient_id: str,
        medication_id: str,
        patient_parameters: Dict[str, Any] = None,
        indication: str = None
    ) -> Dict[str, Any]:
        """
        Calculate medication dosing recommendations

        Args:
            patient_id: Patient identifier
            medication_id: Medication identifier
            patient_parameters: Patient parameters (weight, age, renal function, etc.)
            indication: Clinical indication for the medication

        Returns:
            Dictionary containing dosing recommendations
        """
        if not self._stub:
            raise RuntimeError("Not connected to CAE service")

        request = clinical_reasoning_pb2.DosingCalculationRequest(
            patient_id=patient_id,
            medication_id=medication_id,
            indication=indication or ""
        )

        try:
            response = await self._stub.CalculateDosing(request)
            return self._convert_dosing_response_to_dict(response)

        except grpc.RpcError as e:
            logger.error(f"gRPC error calculating dosing: {e}")
            raise RuntimeError(f"CAE service error: {e.details()}")
        except Exception as e:
            logger.error(f"Error calculating dosing: {e}")
            raise

    async def check_contraindications(
        self,
        patient_id: str,
        medication_ids: List[str],
        condition_ids: List[str] = None,
        allergy_ids: List[str] = None,
        patient_context: Dict[str, Any] = None
    ) -> Dict[str, Any]:
        """
        Check for contraindications

        Args:
            patient_id: Patient identifier
            medication_ids: List of medication identifiers
            condition_ids: List of condition identifiers
            allergy_ids: List of allergy identifiers
            patient_context: Additional patient context

        Returns:
            Dictionary containing contraindication results
        """
        if not self._stub:
            raise RuntimeError("Not connected to CAE service")

        request = clinical_reasoning_pb2.ContraindicationRequest(
            patient_id=patient_id,
            medication_ids=medication_ids,
            condition_ids=condition_ids or [],
            allergy_ids=allergy_ids or []
        )

        try:
            response = await self._stub.CheckContraindications(request)
            return self._convert_contraindication_response_to_dict(response)

        except grpc.RpcError as e:
            logger.error(f"gRPC error checking contraindications: {e}")
            raise RuntimeError(f"CAE service error: {e.details()}")
        except Exception as e:
            logger.error(f"Error checking contraindications: {e}")
            raise
    
    def _convert_priority(self, priority: str):
        """Convert string priority to protobuf enum"""
        priority_map = {
            "critical": clinical_reasoning_pb2.AssertionPriority.PRIORITY_CRITICAL,
            "urgent": clinical_reasoning_pb2.AssertionPriority.PRIORITY_URGENT,
            "standard": clinical_reasoning_pb2.AssertionPriority.PRIORITY_STANDARD,
            "batch": clinical_reasoning_pb2.AssertionPriority.PRIORITY_BATCH
        }
        return priority_map.get(priority.lower(), clinical_reasoning_pb2.AssertionPriority.PRIORITY_STANDARD)
    
    def _convert_response_to_dict(self, response) -> Dict[str, Any]:
        """Convert protobuf response to dictionary"""
        return {
            "request_id": response.request_id,
            "correlation_id": response.correlation_id,
            "assertions": [self._convert_assertion_to_dict(a) for a in response.assertions],
            "metadata": {
                "reasoner_version": response.metadata.reasoner_version,
                "knowledge_version": response.metadata.knowledge_version,
                "processing_time_ms": response.metadata.processing_time_ms,
                "warnings": list(response.metadata.warnings)
            },
            "generated_at": response.generated_at.ToDatetime().isoformat()
        }
    
    def _convert_assertion_to_dict(self, assertion) -> Dict[str, Any]:
        """Convert protobuf assertion to dictionary"""
        return {
            "id": assertion.id,
            "type": assertion.type,
            "severity": self._severity_to_string(assertion.severity),
            "title": assertion.title,
            "description": assertion.description,
            "explanation": assertion.explanation,
            "evidence_sources": list(assertion.evidence_sources),
            "confidence_score": assertion.confidence_score,
            "recommendations": [self._convert_recommendation_to_dict(r) for r in assertion.recommendations]
        }
    
    def _convert_recommendation_to_dict(self, recommendation) -> Dict[str, Any]:
        """Convert protobuf recommendation to dictionary"""
        return {
            "id": recommendation.id,
            "type": recommendation.type,
            "description": recommendation.description,
            "rationale": recommendation.rationale,
            "priority": self._recommendation_priority_to_string(recommendation.priority)
        }
    
    def _convert_interaction_response_to_dict(self, response) -> Dict[str, Any]:
        """Convert interaction response to dictionary"""
        return {
            "interactions": [self._convert_interaction_to_dict(i) for i in response.interactions],
            "metadata": {
                "reasoner_version": response.metadata.reasoner_version,
                "knowledge_version": response.metadata.knowledge_version,
                "processing_time_ms": response.metadata.processing_time_ms,
                "warnings": list(response.metadata.warnings)
            }
        }
    
    def _convert_interaction_to_dict(self, interaction) -> Dict[str, Any]:
        """Convert protobuf interaction to dictionary"""
        return {
            "interaction_id": interaction.interaction_id,
            "medication_a": interaction.medication_a,
            "medication_b": interaction.medication_b,
            "severity": self._severity_to_string(interaction.severity),
            "description": interaction.description,
            "mechanism": interaction.mechanism,
            "clinical_effect": interaction.clinical_effect,
            "evidence_sources": list(interaction.evidence_sources),
            "confidence_score": interaction.confidence_score
        }

    def _convert_dosing_response_to_dict(self, response) -> Dict[str, Any]:
        """Convert dosing response to dictionary"""
        return {
            "dosing": {
                "medication_id": response.dosing.medication_id,
                "dose": response.dosing.dose,
                "frequency": response.dosing.frequency,
                "route": response.dosing.route,
                "duration": response.dosing.duration,
                "rationale": response.dosing.rationale,
                "warnings": list(response.dosing.warnings)
            },
            "adjustments": [self._convert_adjustment_to_dict(a) for a in response.adjustments],
            "metadata": {
                "reasoner_version": response.metadata.reasoner_version,
                "knowledge_version": response.metadata.knowledge_version,
                "processing_time_ms": response.metadata.processing_time_ms,
                "warnings": list(response.metadata.warnings)
            }
        }

    def _convert_adjustment_to_dict(self, adjustment) -> Dict[str, Any]:
        """Convert dosing adjustment to dictionary"""
        return {
            "type": adjustment.type,
            "adjustment": adjustment.adjustment,
            "rationale": adjustment.rationale,
            "required": adjustment.required
        }

    def _convert_contraindication_response_to_dict(self, response) -> Dict[str, Any]:
        """Convert contraindication response to dictionary"""
        return {
            "contraindications": [self._convert_contraindication_to_dict(c) for c in response.contraindications],
            "metadata": {
                "reasoner_version": response.metadata.reasoner_version,
                "knowledge_version": response.metadata.knowledge_version,
                "processing_time_ms": response.metadata.processing_time_ms,
                "warnings": list(response.metadata.warnings)
            }
        }

    def _convert_contraindication_to_dict(self, contraindication) -> Dict[str, Any]:
        """Convert protobuf contraindication to dictionary"""
        return {
            "contraindication_id": contraindication.contraindication_id,
            "medication_id": contraindication.medication_id,
            "condition_id": contraindication.condition_id,
            "severity": self._severity_to_string(contraindication.severity),
            "type": contraindication.type,
            "description": contraindication.description,
            "rationale": contraindication.rationale,
            "evidence_sources": list(contraindication.evidence_sources),
            "override_possible": contraindication.override_possible,
            "override_rationale": contraindication.override_rationale
        }
    
    def _severity_to_string(self, severity) -> str:
        """Convert severity enum to string"""
        severity_map = {
            clinical_reasoning_pb2.AssertionSeverity.SEVERITY_INFO: "info",
            clinical_reasoning_pb2.AssertionSeverity.SEVERITY_LOW: "low",
            clinical_reasoning_pb2.AssertionSeverity.SEVERITY_MODERATE: "moderate",
            clinical_reasoning_pb2.AssertionSeverity.SEVERITY_HIGH: "high",
            clinical_reasoning_pb2.AssertionSeverity.SEVERITY_CRITICAL: "critical"
        }
        return severity_map.get(severity, "unknown")
    
    def _recommendation_priority_to_string(self, priority) -> str:
        """Convert recommendation priority enum to string"""
        priority_map = {
            clinical_reasoning_pb2.RecommendationPriority.RECOMMENDATION_PRIORITY_OPTIONAL: "optional",
            clinical_reasoning_pb2.RecommendationPriority.RECOMMENDATION_PRIORITY_RECOMMENDED: "recommended",
            clinical_reasoning_pb2.RecommendationPriority.RECOMMENDATION_PRIORITY_REQUIRED: "required"
        }
        return priority_map.get(priority, "unknown")


# Convenience functions for easy integration
async def get_clinical_assertions(patient_id: str, **kwargs) -> Dict[str, Any]:
    """Convenience function for getting clinical assertions"""
    async with CAEgRPCClient() as client:
        return await client.generate_clinical_assertions(patient_id, **kwargs)

async def check_drug_interactions(patient_id: str, medication_ids: List[str], **kwargs) -> Dict[str, Any]:
    """Convenience function for checking drug interactions"""
    async with CAEgRPCClient() as client:
        return await client.check_medication_interactions(patient_id, medication_ids, **kwargs)
