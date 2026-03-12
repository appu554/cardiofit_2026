"""
Universal Device Handler - Main Controller

Single intelligent handler for all device types with processor pattern,
dynamic routing, and medical-grade processing capabilities.
"""

import logging
import time
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional
from dataclasses import dataclass

from .device_processor import (
    DeviceType, ProcessingContext, ProcessedDeviceData, 
    ProcessingResult, ParameterCriticality
)
from .device_registry import get_device_registry
from .routing_engine import get_routing_engine, RoutingStrategy
from ..envelope.envelope_factory import EnhancedEnvelopeFactory, AuthContext, RequestContext

logger = logging.getLogger(__name__)


@dataclass
class UniversalProcessingResult:
    """Result of universal device processing"""
    success: bool
    processed_data: Optional[ProcessedDeviceData]
    enhanced_envelope: Optional[Any]  # EnhancedEnvelope
    routing_info: Dict[str, Any]
    processing_time_ms: float
    emergency_detected: bool
    medical_alerts: List[str]
    error_message: Optional[str]
    fallback_used: bool


class UniversalDeviceHandler:
    """
    Universal device handler for intelligent processing of all device types
    
    Provides single entry point for device data processing with automatic
    device detection, processor selection, and medical-grade validation.
    """
    
    def __init__(self, envelope_factory: Optional[EnhancedEnvelopeFactory] = None):
        self.envelope_factory = envelope_factory
        self.processing_stats = {
            "total_processed": 0,
            "successful_processed": 0,
            "emergency_detected": 0,
            "fallback_used": 0,
            "processing_times": [],
            "device_type_counts": {},
            "error_counts": {}
        }
        self.created_at = datetime.now(timezone.utc)
    
    async def process_device_data(self, 
                                device_data: Dict[str, Any],
                                auth_context: AuthContext,
                                request_context: RequestContext,
                                patient_context_data: Optional[Dict[str, Any]] = None) -> UniversalProcessingResult:
        """
        Universal device data processing
        
        Args:
            device_data: Raw device data
            auth_context: Authentication context
            request_context: Request context
            patient_context_data: Optional patient context
            
        Returns:
            UniversalProcessingResult with processing outcome
        """
        start_time = time.time()
        
        try:
            self.processing_stats["total_processed"] += 1
            
            # Step 1: Create processing context
            processing_context = self._create_processing_context(device_data, auth_context, request_context)
            
            # Step 2: Route to appropriate processor
            routing_engine = get_routing_engine()
            routing_result = await routing_engine.route_device_data(device_data, processing_context)
            
            if not routing_result.processor:
                return self._create_error_result(
                    "No suitable processor found for device data",
                    routing_result, start_time, True
                )
            
            # Step 3: Process device data
            processor = routing_result.processor
            processed_data = processor.process_data(device_data, processing_context)
            
            # Step 4: Check for emergencies
            emergency_detected, emergency_message = processor.detect_emergency(device_data, processing_context)
            medical_alerts = []
            if emergency_detected:
                medical_alerts.append(emergency_message)
                self.processing_stats["emergency_detected"] += 1
            
            # Step 5: Create enhanced envelope if factory available
            enhanced_envelope = None
            if self.envelope_factory:
                try:
                    enhanced_envelope = await self.envelope_factory.create_device_data_envelope(
                        device_data=processed_data.processed_data,
                        auth_context=auth_context,
                        request_context=request_context,
                        patient_context_data=patient_context_data
                    )
                except Exception as e:
                    logger.warning(f"Failed to create enhanced envelope: {e}")
            
            # Step 6: Update statistics
            self._update_processing_stats(routing_result, processed_data, start_time, True)
            
            return UniversalProcessingResult(
                success=True,
                processed_data=processed_data,
                enhanced_envelope=enhanced_envelope,
                routing_info={
                    "device_type": routing_result.device_type.value,
                    "processor_id": processor.processor_id,
                    "confidence": routing_result.confidence,
                    "strategy": routing_result.strategy_used.value,
                    "routing_time_ms": routing_result.routing_time_ms
                },
                processing_time_ms=(time.time() - start_time) * 1000,
                emergency_detected=emergency_detected,
                medical_alerts=medical_alerts,
                error_message=None,
                fallback_used=routing_result.fallback_used
            )
            
        except Exception as e:
            logger.error(f"Universal processing failed: {e}")
            self._update_error_stats(str(e))
            
            return self._create_error_result(
                str(e), None, start_time, False
            )
    
    async def detect_device_type(self, device_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Detect device type without full processing
        
        Args:
            device_data: Device data to analyze
            
        Returns:
            Device type detection result
        """
        try:
            # Create minimal processing context
            processing_context = ProcessingContext(
                device_id=device_data.get("device_id", "unknown"),
                device_type=DeviceType.UNKNOWN,
                timestamp=int(time.time()),
                patient_id=device_data.get("patient_id"),
                user_context={},
                request_metadata={},
                processing_hints={}
            )
            
            # Route to get device type
            routing_engine = get_routing_engine()
            routing_result = await routing_engine.route_device_data(device_data, processing_context)
            
            return {
                "device_type": routing_result.device_type.value,
                "confidence": routing_result.confidence,
                "strategy_used": routing_result.strategy_used.value,
                "routing_time_ms": routing_result.routing_time_ms,
                "processor_available": routing_result.processor is not None,
                "fallback_used": routing_result.fallback_used,
                "detection_metadata": routing_result.detection_metadata
            }
            
        except Exception as e:
            logger.error(f"Device type detection failed: {e}")
            return {
                "device_type": DeviceType.UNKNOWN.value,
                "confidence": 0.0,
                "error": str(e)
            }
    
    async def get_supported_device_types(self) -> List[Dict[str, Any]]:
        """Get all supported device types with capabilities"""
        try:
            registry = await get_device_registry()
            supported_types = []
            
            for device_type in registry.get_all_supported_device_types():
                processors = registry.get_processors_for_device_type(device_type)
                if processors:
                    processor = processors[0]  # Get first available processor
                    capabilities = processor.get_device_capabilities(device_type)
                    
                    supported_types.append({
                        "device_type": device_type.value,
                        "processor_count": len(processors),
                        "medical_grade": capabilities.medical_grade,
                        "real_time_required": capabilities.real_time_required,
                        "emergency_detection": capabilities.emergency_detection,
                        "parameter_types": capabilities.parameter_types,
                        "compliance_metadata": capabilities.compliance_metadata
                    })
            
            return supported_types
            
        except Exception as e:
            logger.error(f"Failed to get supported device types: {e}")
            return []
    
    def _create_processing_context(self, device_data: Dict[str, Any], 
                                 auth_context: AuthContext, 
                                 request_context: RequestContext) -> ProcessingContext:
        """Create processing context from request data"""
        # Try to determine device type from data
        device_type = DeviceType.UNKNOWN
        reading_type = device_data.get("reading_type", "").lower()
        if reading_type:
            try:
                device_type = DeviceType(reading_type)
            except ValueError:
                pass
        
        return ProcessingContext(
            device_id=device_data.get("device_id", "unknown"),
            device_type=device_type,
            timestamp=device_data.get("timestamp", int(time.time())),
            patient_id=device_data.get("patient_id"),
            user_context={
                "user_id": auth_context.id,
                "user_role": auth_context.role,
                "permissions": auth_context.permissions
            },
            request_metadata={
                "source_ip": request_context.source_ip,
                "user_agent": request_context.user_agent,
                "request_id": request_context.request_id
            },
            processing_hints={
                "real_time": True,
                "medical_grade": True
            }
        )
    
    def _create_error_result(self, error_message: str, routing_result, start_time: float, fallback_used: bool) -> UniversalProcessingResult:
        """Create error result"""
        routing_info = {}
        if routing_result:
            routing_info = {
                "device_type": routing_result.device_type.value,
                "confidence": routing_result.confidence,
                "strategy": routing_result.strategy_used.value,
                "routing_time_ms": routing_result.routing_time_ms
            }
        
        return UniversalProcessingResult(
            success=False,
            processed_data=None,
            enhanced_envelope=None,
            routing_info=routing_info,
            processing_time_ms=(time.time() - start_time) * 1000,
            emergency_detected=False,
            medical_alerts=[],
            error_message=error_message,
            fallback_used=fallback_used
        )
    
    def _update_processing_stats(self, routing_result, processed_data, start_time: float, success: bool):
        """Update processing statistics"""
        processing_time = (time.time() - start_time) * 1000
        self.processing_stats["processing_times"].append(processing_time)
        
        if success:
            self.processing_stats["successful_processed"] += 1
        
        if routing_result.fallback_used:
            self.processing_stats["fallback_used"] += 1
        
        # Update device type counts
        device_type = routing_result.device_type.value
        self.processing_stats["device_type_counts"][device_type] = (
            self.processing_stats["device_type_counts"].get(device_type, 0) + 1
        )
    
    def _update_error_stats(self, error_message: str):
        """Update error statistics"""
        self.processing_stats["error_counts"][error_message] = (
            self.processing_stats["error_counts"].get(error_message, 0) + 1
        )
    
    def get_processing_stats(self) -> Dict[str, Any]:
        """Get comprehensive processing statistics"""
        processing_times = self.processing_stats["processing_times"]
        
        stats = {
            "total_processed": self.processing_stats["total_processed"],
            "successful_processed": self.processing_stats["successful_processed"],
            "success_rate": (
                self.processing_stats["successful_processed"] / 
                max(self.processing_stats["total_processed"], 1)
            ) * 100,
            "emergency_detected": self.processing_stats["emergency_detected"],
            "fallback_used": self.processing_stats["fallback_used"],
            "fallback_rate": (
                self.processing_stats["fallback_used"] / 
                max(self.processing_stats["total_processed"], 1)
            ) * 100,
            "device_type_distribution": self.processing_stats["device_type_counts"],
            "error_distribution": self.processing_stats["error_counts"],
            "uptime_seconds": (datetime.now(timezone.utc) - self.created_at).total_seconds()
        }
        
        if processing_times:
            stats["performance"] = {
                "avg_processing_time_ms": sum(processing_times) / len(processing_times),
                "min_processing_time_ms": min(processing_times),
                "max_processing_time_ms": max(processing_times),
                "p95_processing_time_ms": (
                    sorted(processing_times)[int(len(processing_times) * 0.95)] 
                    if len(processing_times) > 20 else max(processing_times)
                ),
                "sub_100ms_rate": (
                    sum(1 for t in processing_times if t < 100) / len(processing_times)
                ) * 100
            }
        
        return stats


# Global universal handler instance
_universal_handler: Optional[UniversalDeviceHandler] = None


async def get_universal_handler(envelope_factory: Optional[EnhancedEnvelopeFactory] = None) -> UniversalDeviceHandler:
    """Get or create the global universal device handler"""
    global _universal_handler
    
    if _universal_handler is None:
        _universal_handler = UniversalDeviceHandler(envelope_factory)
        logger.info("Global universal device handler initialized")
    
    return _universal_handler
