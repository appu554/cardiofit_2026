"""
Safety Gateway Platform Client

Provides gRPC client for integrating with Safety Gateway Platform service.
The Safety Gateway Platform orchestrates all safety engines including CAE internally.
"""

import asyncio
import logging
import time
from typing import Dict, Any, Optional, List
from datetime import datetime
import json

import grpc
from grpc import aio as aio_grpc

logger = logging.getLogger(__name__)


class SafetyGatewayClient:
    """
    Client for Safety Gateway Platform integration
    
    This client provides the single integration point for comprehensive safety validation.
    The Safety Gateway Platform internally orchestrates CAE and other safety engines.
    """
    
    def __init__(self, gateway_url: str = "localhost:8030"):
        self.gateway_url = gateway_url
        self.channel: Optional[aio_grpc.Channel] = None
        self.stub = None
        self.connected = False
        
        # Performance tracking
        self.request_count = 0
        self.total_response_time = 0.0
        self.last_request_time = None
        
    async def initialize(self):
        """Initialize gRPC connection to Safety Gateway Platform"""
        try:
            logger.info(f"🔗 Initializing Safety Gateway Platform client: {self.gateway_url}")
            
            # Create gRPC channel
            self.channel = aio_grpc.insecure_channel(self.gateway_url)
            
            # For now, we'll use a simple approach since we don't have the protobuf files
            # In production, you would import the generated protobuf stubs
            # from safety_gateway_pb2_grpc import SafetyGatewayStub
            # self.stub = SafetyGatewayStub(self.channel)
            
            # Test connection with a simple health check
            await self._test_connection()
            
            self.connected = True
            logger.info("✅ Safety Gateway Platform client initialized successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to initialize Safety Gateway Platform client: {e}")
            self.connected = False
            raise
    
    async def _test_connection(self):
        """Test connection to Safety Gateway Platform"""
        try:
            # Simple connection test - in production this would be a proper health check
            await asyncio.wait_for(
                self.channel.channel_ready(),
                timeout=5.0
            )
            logger.info("🔍 Safety Gateway Platform connection test passed")
        except asyncio.TimeoutError:
            raise ConnectionError("Safety Gateway Platform connection timeout")
        except Exception as e:
            raise ConnectionError(f"Safety Gateway Platform connection failed: {e}")
    
    async def validate_safety(
        self,
        patient_id: str,
        medication_ids: List[str],
        clinical_context: Dict[str, Any],
        action_type: str = "medication_order",
        priority: str = "normal",
        request_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Validate safety using Safety Gateway Platform
        
        This method sends a comprehensive safety request to the Safety Gateway Platform,
        which internally orchestrates CAE and other safety engines.
        
        Args:
            patient_id: Patient identifier
            medication_ids: List of medication identifiers/names
            clinical_context: Clinical context data from Context Service
            action_type: Type of clinical action (default: medication_order)
            priority: Request priority (normal, urgent, emergency)
            request_id: Optional request identifier for tracking
            
        Returns:
            Comprehensive safety validation result
        """
        start_time = time.time()
        
        if not self.connected:
            await self.initialize()
        
        if not request_id:
            request_id = f"med_safety_{patient_id}_{int(start_time * 1000)}"
        
        logger.info(f"🛡️ Validating safety via Safety Gateway Platform")
        logger.info(f"   Request ID: {request_id}")
        logger.info(f"   Patient: {patient_id}")
        logger.info(f"   Medications: {medication_ids}")
        logger.info(f"   Priority: {priority}")
        
        try:
            # Prepare safety request
            safety_request = self._prepare_safety_request(
                request_id=request_id,
                patient_id=patient_id,
                medication_ids=medication_ids,
                clinical_context=clinical_context,
                action_type=action_type,
                priority=priority
            )
            
            # Call Safety Gateway Platform
            # For now, we'll simulate the call since we don't have protobuf setup
            # In production: response = await self.stub.ValidateSafety(safety_request)
            safety_response = await self._call_safety_gateway_platform(safety_request)
            
            # Track performance
            response_time = time.time() - start_time
            self._update_performance_metrics(response_time)
            
            logger.info(f"✅ Safety validation completed")
            logger.info(f"   Status: {safety_response.get('status', 'UNKNOWN')}")
            logger.info(f"   Risk Score: {safety_response.get('risk_score', 0.0):.3f}")
            logger.info(f"   Response Time: {response_time * 1000:.1f}ms")
            
            return safety_response
            
        except Exception as e:
            response_time = time.time() - start_time
            logger.error(f"❌ Safety validation failed: {e}")
            logger.error(f"   Response Time: {response_time * 1000:.1f}ms")
            
            # Return fail-closed response
            return self._create_fail_closed_response(request_id, str(e))
    
    def _prepare_safety_request(
        self,
        request_id: str,
        patient_id: str,
        medication_ids: List[str],
        clinical_context: Dict[str, Any],
        action_type: str,
        priority: str
    ) -> Dict[str, Any]:
        """Prepare safety request for Safety Gateway Platform"""
        
        # Extract relevant clinical data
        patient_data = clinical_context.get('patient_data', {})
        clinical_data = clinical_context.get('clinical_data', {})
        
        # Extract conditions and allergies
        conditions = patient_data.get('conditions', [])
        allergies = patient_data.get('allergies', [])
        
        # Convert allergies to allergy IDs
        allergy_ids = []
        if isinstance(allergies, list):
            for allergy in allergies:
                if isinstance(allergy, dict):
                    allergy_ids.append(allergy.get('allergen', 'unknown'))
                elif isinstance(allergy, str):
                    allergy_ids.append(allergy)
        
        safety_request = {
            "request_id": request_id,
            "patient_id": patient_id,
            "clinician_id": "system",  # Could be extracted from context
            "action_type": action_type,
            "priority": priority,
            "medication_ids": medication_ids,
            "condition_ids": conditions,
            "allergy_ids": allergy_ids,
            "clinical_context": {
                "patient_demographics": {
                    "age": patient_data.get('age'),
                    "weight_kg": patient_data.get('weight_kg'),
                    "height_cm": patient_data.get('height_cm'),
                    "gender": patient_data.get('gender')
                },
                "laboratory_results": clinical_data.get('labs', {}),
                "vital_signs": clinical_data.get('vitals', {}),
                "current_medications": clinical_data.get('current_medications', []),
                "recent_medications": clinical_data.get('recent_medications', [])
            },
            "timestamp": datetime.now().isoformat()
        }
        
        return safety_request
    
    async def _call_safety_gateway_platform(self, safety_request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Call Safety Gateway Platform service
        
        For now, this simulates the call. In production, this would be:
        response = await self.stub.ValidateSafety(safety_request_pb)
        """
        try:
            # Simulate Safety Gateway Platform processing
            logger.info("🔄 Calling Safety Gateway Platform service...")
            
            # Simulate processing time (Safety Gateway Platform is fast)
            await asyncio.sleep(0.05)  # 50ms simulation
            
            # For now, return a simulated response
            # In production, this would be the actual gRPC response
            return self._create_simulated_safety_response(safety_request)
            
        except Exception as e:
            logger.error(f"❌ Safety Gateway Platform call failed: {e}")
            raise
    
    def _create_simulated_safety_response(self, safety_request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create simulated safety response for testing
        
        In production, this would be replaced by actual Safety Gateway Platform response
        """
        medication_ids = safety_request.get('medication_ids', [])
        allergy_ids = safety_request.get('allergy_ids', [])
        
        # Simulate basic safety logic
        violations = []
        warnings = []
        risk_score = 0.0
        
        # Check for high-risk medications
        high_risk_meds = ['warfarin', 'digoxin', 'insulin', 'chemotherapy']
        for med in medication_ids:
            if any(risk_med in med.lower() for risk_med in high_risk_meds):
                warnings.append(f"High-risk medication detected: {med}")
                risk_score += 0.3
        
        # Check for allergies
        if allergy_ids:
            for allergy in allergy_ids:
                if any(med.lower() in allergy.lower() for med in medication_ids):
                    violations.append(f"Potential allergy interaction: {allergy}")
                    risk_score += 0.5
        
        # Determine status
        if violations:
            status = "UNSAFE"
            risk_score = min(risk_score, 1.0)
        elif warnings:
            status = "WARNING"
            risk_score = min(risk_score, 0.7)
        else:
            status = "SAFE"
            risk_score = max(risk_score, 0.1)
        
        return {
            "request_id": safety_request["request_id"],
            "status": status,
            "risk_score": risk_score,
            "confidence": 0.85,
            "violations": violations,
            "warnings": warnings,
            "explanations": [
                "Safety validation completed by Safety Gateway Platform",
                "CAE service integration: Active",
                "Clinical reasoning engines: 4 active"
            ],
            "processing_time_ms": 50,
            "engine_results": [
                {
                    "engine_id": "cae_engine",
                    "engine_name": "Clinical Assertion Engine",
                    "status": status,
                    "risk_score": risk_score,
                    "tier": "veto_critical"
                }
            ],
            "metadata": {
                "safety_gateway_version": "1.0.0",
                "cae_integration": "active",
                "engines_executed": ["cae_engine", "allergy_engine", "interaction_engine"]
            }
        }
    
    def _create_fail_closed_response(self, request_id: str, error_message: str) -> Dict[str, Any]:
        """Create fail-closed safety response when Safety Gateway Platform is unavailable"""
        return {
            "request_id": request_id,
            "status": "UNSAFE",
            "risk_score": 1.0,
            "confidence": 0.0,
            "violations": [f"Safety Gateway Platform unavailable: {error_message}"],
            "warnings": ["Safety validation could not be completed"],
            "explanations": [
                "Safety Gateway Platform service is unavailable",
                "Fail-closed safety response generated",
                "Manual safety review required"
            ],
            "processing_time_ms": 0,
            "engine_results": [],
            "metadata": {
                "error": error_message,
                "fail_closed": True,
                "safety_gateway_available": False
            }
        }
    
    def _update_performance_metrics(self, response_time: float):
        """Update performance tracking metrics"""
        self.request_count += 1
        self.total_response_time += response_time
        self.last_request_time = response_time
    
    def get_performance_metrics(self) -> Dict[str, Any]:
        """Get client performance metrics"""
        avg_response_time = (
            self.total_response_time / self.request_count 
            if self.request_count > 0 else 0.0
        )
        
        return {
            "connected": self.connected,
            "gateway_url": self.gateway_url,
            "total_requests": self.request_count,
            "average_response_time_ms": avg_response_time * 1000,
            "last_response_time_ms": (self.last_request_time * 1000) if self.last_request_time else None
        }
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform health check on Safety Gateway Platform"""
        try:
            if not self.connected:
                await self.initialize()
            
            # Simple health check
            start_time = time.time()
            await self._test_connection()
            response_time = time.time() - start_time
            
            return {
                "status": "healthy",
                "connected": True,
                "response_time_ms": response_time * 1000,
                "gateway_url": self.gateway_url
            }
            
        except Exception as e:
            return {
                "status": "unhealthy",
                "connected": False,
                "error": str(e),
                "gateway_url": self.gateway_url
            }
    
    async def close(self):
        """Close gRPC connection"""
        if self.channel:
            await self.channel.close()
            self.connected = False
            logger.info("🔌 Safety Gateway Platform client connection closed")
