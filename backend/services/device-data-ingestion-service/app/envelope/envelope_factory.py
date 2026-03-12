"""
Enhanced Envelope Factory

Centralized factory for creating enterprise-grade message envelopes with
device-specific logic, security context integration, and quality assessment.
"""

import hashlib
import json
import logging
import time
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple
import uuid
import asyncio

from .enhanced_envelope import (
    EnhancedEnvelope, SecurityMetadata, QualityMetadata, PatientContext,
    DeviceContext, ProcessingHints, LineageMetadata, SecurityLevel,
    QualityLevel, ComplianceStatus
)
from .performance_optimizations import PerformanceOptimizer, LazyMetadata

logger = logging.getLogger(__name__)


class AuthContext:
    """Authentication context from JWT validation"""
    def __init__(self, auth_result: Dict[str, Any]):
        self.user_id = auth_result.get('id') or auth_result.get('user_id')
        self.email = auth_result.get('email')
        self.role = auth_result.get('role')
        self.roles = auth_result.get('roles', [])
        self.permissions = auth_result.get('permissions', [])
        self.token_id = auth_result.get('token_id')
        self.issued_at = auth_result.get('iat') or auth_result.get('created_at')
        self.expires_at = auth_result.get('exp')
        self.is_active = auth_result.get('is_active', True)


class RequestContext:
    """Request context information"""
    def __init__(self, 
                 timestamp: Optional[int] = None,
                 source_ip: Optional[str] = None,
                 user_agent: Optional[str] = None,
                 request_id: Optional[str] = None):
        self.timestamp = timestamp or int(time.time())
        self.source_ip = source_ip
        self.user_agent = user_agent
        self.request_id = request_id or str(uuid.uuid4())


class DeviceCapabilities:
    """Device capabilities and specifications"""
    def __init__(self, device_type: str):
        self.device_type = device_type
        self.capabilities = self._get_device_capabilities(device_type)
    
    def _get_device_capabilities(self, device_type: str) -> Dict[str, Any]:
        """Get device capabilities based on device type"""
        capabilities_map = {
            "heart_rate": {
                "measurement_types": ["heart_rate", "rr_interval"],
                "accuracy_specifications": {
                    "heart_rate": {"range": [30, 220], "accuracy": "±2 bpm"},
                    "rr_interval": {"range": [200, 2000], "accuracy": "±5 ms"}
                },
                "typical_battery_life_hours": 168,  # 1 week
                "calibration_required": False
            },
            "blood_pressure": {
                "measurement_types": ["systolic", "diastolic", "mean_arterial_pressure"],
                "accuracy_specifications": {
                    "systolic": {"range": [70, 250], "accuracy": "±3 mmHg"},
                    "diastolic": {"range": [40, 150], "accuracy": "±3 mmHg"}
                },
                "typical_battery_life_hours": 720,  # 30 days
                "calibration_required": True,
                "calibration_interval_days": 365
            },
            "blood_glucose": {
                "measurement_types": ["glucose_level"],
                "accuracy_specifications": {
                    "glucose_level": {"range": [20, 600], "accuracy": "±15% or ±15 mg/dL"}
                },
                "typical_battery_life_hours": 8760,  # 1 year
                "calibration_required": True,
                "calibration_interval_days": 14
            },
            "temperature": {
                "measurement_types": ["body_temperature"],
                "accuracy_specifications": {
                    "body_temperature": {"range": [32, 42], "accuracy": "±0.1°C"}
                },
                "typical_battery_life_hours": 8760,  # 1 year
                "calibration_required": False
            },
            "oxygen_saturation": {
                "measurement_types": ["spo2", "pulse_rate"],
                "accuracy_specifications": {
                    "spo2": {"range": [70, 100], "accuracy": "±2%"},
                    "pulse_rate": {"range": [30, 250], "accuracy": "±3 bpm"}
                },
                "typical_battery_life_hours": 168,  # 1 week
                "calibration_required": False
            }
        }
        
        return capabilities_map.get(device_type, {
            "measurement_types": ["unknown"],
            "accuracy_specifications": {},
            "typical_battery_life_hours": 168,
            "calibration_required": False
        })


class QualityAssessment:
    """Quality assessment engine for device data"""
    
    @staticmethod
    def assess_data_quality(device_data: Dict[str, Any], 
                          device_capabilities: DeviceCapabilities,
                          historical_context: Optional[Dict[str, Any]] = None) -> QualityMetadata:
        """Assess data quality across multiple dimensions"""
        
        quality = QualityMetadata()
        
        # Completeness assessment
        quality.completeness_score = QualityAssessment._assess_completeness(device_data)
        
        # Validity assessment
        quality.validity_score = QualityAssessment._assess_validity(
            device_data, device_capabilities
        )
        
        # Consistency assessment
        quality.consistency_score = QualityAssessment._assess_consistency(
            device_data, historical_context
        )
        
        # Accuracy assessment
        quality.accuracy_score = QualityAssessment._assess_accuracy(
            device_data, device_capabilities
        )
        
        # Timeliness assessment
        quality.timeliness_score = QualityAssessment._assess_timeliness(device_data)
        
        # Calculate overall quality
        quality._recalculate_overall_quality = lambda: None  # Placeholder
        QualityAssessment._calculate_overall_quality(quality)
        
        # Set validation status
        quality.validation_passed = (
            quality.completeness_score >= 0.8 and
            quality.validity_score >= 0.8 and
            quality.overall_quality_score >= 0.7
        )
        
        return quality
    
    @staticmethod
    def _assess_completeness(device_data: Dict[str, Any]) -> float:
        """Assess data completeness"""
        required_fields = ['device_id', 'timestamp', 'reading_type', 'value', 'unit']
        optional_fields = ['patient_id', 'metadata']
        
        present_required = sum(1 for field in required_fields if device_data.get(field))
        present_optional = sum(1 for field in optional_fields if device_data.get(field))
        
        # Weight required fields more heavily
        completeness = (present_required / len(required_fields)) * 0.8 + \
                      (present_optional / len(optional_fields)) * 0.2
        
        return min(1.0, completeness)
    
    @staticmethod
    def _assess_validity(device_data: Dict[str, Any], 
                        device_capabilities: DeviceCapabilities) -> float:
        """Assess data validity against device specifications"""
        reading_type = device_data.get('reading_type')
        value = device_data.get('value')
        
        if not reading_type or value is None:
            return 0.0
        
        # Check if reading type is supported
        supported_types = device_capabilities.capabilities.get('measurement_types', [])
        if reading_type not in supported_types:
            return 0.5  # Partial validity
        
        # Check value range
        accuracy_specs = device_capabilities.capabilities.get('accuracy_specifications', {})
        if reading_type in accuracy_specs:
            spec_range = accuracy_specs[reading_type].get('range', [])
            if len(spec_range) == 2:
                min_val, max_val = spec_range
                if min_val <= value <= max_val:
                    return 1.0
                else:
                    return 0.3  # Out of range
        
        return 0.8  # Default validity for unknown ranges
    
    @staticmethod
    def _assess_consistency(device_data: Dict[str, Any], 
                          historical_context: Optional[Dict[str, Any]]) -> float:
        """Assess consistency with historical data"""
        if not historical_context:
            return 0.8  # Default when no history available
        
        # Simple consistency check - can be enhanced with ML
        current_value = device_data.get('value')
        if current_value is None:
            return 0.0
        
        # Check against recent values (simplified)
        recent_values = historical_context.get('recent_values', [])
        if recent_values:
            avg_recent = sum(recent_values) / len(recent_values)
            deviation = abs(current_value - avg_recent) / avg_recent if avg_recent > 0 else 0
            
            # High consistency if within 20% of recent average
            if deviation <= 0.2:
                return 1.0
            elif deviation <= 0.5:
                return 0.7
            else:
                return 0.4
        
        return 0.8  # Default consistency
    
    @staticmethod
    def _assess_accuracy(device_data: Dict[str, Any], 
                        device_capabilities: DeviceCapabilities) -> float:
        """Assess accuracy based on device capabilities"""
        metadata = device_data.get('metadata', {})
        
        # Check device health indicators
        battery_level = metadata.get('battery_level')
        signal_quality = metadata.get('signal_quality')
        
        accuracy_score = 1.0
        
        # Reduce accuracy for low battery
        if battery_level is not None:
            if battery_level < 10:
                accuracy_score *= 0.6
            elif battery_level < 25:
                accuracy_score *= 0.8
        
        # Reduce accuracy for poor signal
        if signal_quality:
            quality_map = {
                'excellent': 1.0,
                'good': 0.9,
                'fair': 0.7,
                'poor': 0.4
            }
            accuracy_score *= quality_map.get(signal_quality.lower(), 0.8)
        
        return min(1.0, accuracy_score)
    
    @staticmethod
    def _assess_timeliness(device_data: Dict[str, Any]) -> float:
        """Assess data timeliness"""
        timestamp = device_data.get('timestamp')
        if not timestamp:
            return 0.0
        
        current_time = int(time.time())
        age_seconds = current_time - timestamp
        
        # Fresh data (< 5 minutes) gets full score
        if age_seconds < 300:
            return 1.0
        # Recent data (< 1 hour) gets good score
        elif age_seconds < 3600:
            return 0.8
        # Older data (< 24 hours) gets fair score
        elif age_seconds < 86400:
            return 0.6
        # Very old data gets poor score
        else:
            return 0.3
    
    @staticmethod
    def _calculate_overall_quality(quality: QualityMetadata):
        """Calculate overall quality score and level"""
        scores = [
            quality.completeness_score,
            quality.validity_score,
            quality.consistency_score,
            quality.accuracy_score,
            quality.timeliness_score
        ]
        
        # Weighted average
        weights = [0.25, 0.25, 0.2, 0.2, 0.1]
        quality.overall_quality_score = sum(s * w for s, w in zip(scores, weights))
        
        # Determine quality level
        if quality.overall_quality_score >= 0.9:
            quality.quality_level = QualityLevel.EXCELLENT
        elif quality.overall_quality_score >= 0.7:
            quality.quality_level = QualityLevel.GOOD
        elif quality.overall_quality_score >= 0.5:
            quality.quality_level = QualityLevel.FAIR
        else:
            quality.quality_level = QualityLevel.POOR


class EnhancedEnvelopeFactory:
    """
    Enhanced envelope factory with enterprise-grade features
    
    Creates comprehensive message envelopes with security metadata,
    quality assessment, patient context, and processing hints.
    """
    
    def __init__(self, service_name: str, cache_manager=None):
        self.service_name = service_name
        self.creation_metrics = {
            "total_created": 0,
            "creation_times": [],
            "quality_scores": [],
            "security_events": 0
        }

        # Initialize performance optimizer
        self.performance_optimizer = PerformanceOptimizer(cache_manager)
        self.performance_enabled = True

    async def initialize(self):
        """Initialize the envelope factory with performance optimizations"""
        if self.performance_enabled:
            await self.performance_optimizer.initialize()
            logger.info("Enhanced envelope factory initialized with performance optimizations")

    async def cleanup(self):
        """Cleanup envelope factory resources"""
        if self.performance_enabled:
            await self.performance_optimizer.cleanup()
            logger.info("Enhanced envelope factory cleaned up")
    
    async def create_device_data_envelope(self,
                                        device_data: Dict[str, Any],
                                        auth_context: AuthContext,
                                        request_context: RequestContext,
                                        patient_context_data: Optional[Dict[str, Any]] = None,
                                        historical_context: Optional[Dict[str, Any]] = None) -> EnhancedEnvelope:
        """
        Create enhanced envelope for device data with full metadata enrichment
        
        Args:
            device_data: Raw device data payload
            auth_context: Authentication context from JWT validation
            request_context: Request context information
            patient_context_data: Optional patient context data
            historical_context: Optional historical data for quality assessment
            
        Returns:
            EnhancedEnvelope with complete metadata
        """
        start_time = time.time()

        try:
            # Get optimized envelope template if performance optimization enabled
            envelope_template = None
            if self.performance_enabled:
                envelope_template = await self.performance_optimizer.get_envelope_template()

            # Generate unique identifiers
            envelope_id = str(uuid.uuid4())
            trace_id = str(uuid.uuid4())
            span_id = str(uuid.uuid4())
            
            # Extract device information
            device_id = device_data.get('device_id', 'unknown')
            device_type = device_data.get('reading_type', 'unknown')
            
            # Create device capabilities
            device_capabilities = DeviceCapabilities(device_type)

            # Create security metadata (critical - always synchronous)
            security = self._create_security_metadata(
                device_data, auth_context, request_context
            )

            # Assess data quality (critical - always synchronous)
            quality = QualityAssessment.assess_data_quality(
                device_data, device_capabilities, historical_context
            )

            # Create patient context with lazy loading if performance optimization enabled
            patient_context = None
            if patient_context_data:
                patient_context = self._create_patient_context(device_data, patient_context_data)
            elif self.performance_enabled:
                # Create lazy loader for patient context
                patient_id = device_data.get('patient_id')
                if patient_id:
                    async def load_patient_context():
                        cached_context = await self.performance_optimizer.get_cached_metadata(
                            "patient_context", patient_id
                        )
                        return self._create_patient_context(device_data, cached_context)

                    # For now, load synchronously but could be made lazy
                    patient_context = await load_patient_context()

            # Create device context with caching
            device_context = await self._create_device_context_optimized(
                device_data, device_capabilities
            )
            
            # Create processing hints
            processing_hints = self._create_processing_hints(
                device_data, quality, security
            )
            
            # Create lineage metadata
            lineage = LineageMetadata(
                trace_id=trace_id,
                span_id=span_id,
                message_id=envelope_id,
                correlation_id=request_context.request_id,
                created_at=datetime.now(timezone.utc).isoformat(),
                ingestion_time=datetime.now(timezone.utc).isoformat()
            )
            
            # Create enhanced envelope
            envelope = EnhancedEnvelope(
                id=envelope_id,
                source=self.service_name,
                type=f"device.data.{device_type}",
                subject=f"Device/{device_id}",
                time=datetime.now(timezone.utc).isoformat(),
                data=device_data,
                security=security,
                quality=quality,
                patient_context=patient_context,
                device_context=device_context,
                processing_hints=processing_hints,
                lineage=lineage,
                correlation_id=request_context.request_id
            )
            
            # Update metrics
            creation_time = time.time() - start_time
            self._update_metrics(creation_time, quality.overall_quality_score, security)

            # Record performance metrics
            if self.performance_enabled:
                self.performance_optimizer.record_creation_time(creation_time)

            # Return envelope template to pool if used
            if envelope_template and self.performance_enabled:
                await self.performance_optimizer.return_envelope_template(envelope_template)

            logger.debug(f"Created enhanced envelope {envelope_id} in {creation_time:.3f}s")

            # Queue async enrichments for non-critical metadata
            if self.performance_enabled:
                await self._queue_async_enrichments(envelope_id, device_data, auth_context)

            return envelope
            
        except Exception as e:
            logger.error(f"Failed to create enhanced envelope: {e}")
            # Create minimal envelope for fallback
            return await self._create_fallback_envelope(device_data, auth_context, request_context)
    
    def _create_security_metadata(self, 
                                device_data: Dict[str, Any],
                                auth_context: AuthContext,
                                request_context: RequestContext) -> SecurityMetadata:
        """Create security metadata from authentication and request context"""
        
        # Calculate payload hash for integrity
        payload_hash = hashlib.sha256(
            json.dumps(device_data, sort_keys=True).encode()
        ).hexdigest()
        
        # Determine HIPAA eligibility
        hipaa_eligible = (
            auth_context.role in ['doctor', 'nurse', 'patient'] and
            device_data.get('patient_id') is not None
        )
        
        # Calculate risk score (simplified)
        risk_score = 0.0
        if not hipaa_eligible:
            risk_score += 0.1
        if request_context.source_ip and request_context.source_ip.startswith('10.'):
            risk_score += 0.1  # Internal network
        
        return SecurityMetadata(
            auth_method="JWT",
            user_id=auth_context.user_id,
            user_role=auth_context.role,
            user_permissions=auth_context.permissions,
            request_timestamp=request_context.timestamp,
            payload_hash=payload_hash,
            source_ip=request_context.source_ip,
            user_agent=request_context.user_agent,
            hipaa_eligible=hipaa_eligible,
            gdpr_compliant=True,  # Assume compliant for now
            data_retention_days=2555 if hipaa_eligible else 365,  # 7 years for HIPAA
            risk_score=risk_score
        )
    
    def _create_patient_context(self, 
                              device_data: Dict[str, Any],
                              patient_context_data: Optional[Dict[str, Any]]) -> Optional[PatientContext]:
        """Create patient context metadata"""
        patient_id = device_data.get('patient_id')
        if not patient_id:
            return None
        
        context_data = patient_context_data or {}
        
        return PatientContext(
            patient_id=patient_id,
            patient_consent_status=context_data.get('consent_status', 'unknown'),
            consent_version=context_data.get('consent_version'),
            data_sharing_permissions=context_data.get('sharing_permissions', []),
            anonymization_level="identified",  # Default for authenticated users
            clinical_conditions=context_data.get('conditions', []),
            medication_list=context_data.get('medications', []),
            care_team=context_data.get('care_team', [])
        )
    
    def _create_device_context(self, 
                             device_data: Dict[str, Any],
                             device_capabilities: DeviceCapabilities) -> DeviceContext:
        """Create device context metadata"""
        metadata = device_data.get('metadata', {})
        
        return DeviceContext(
            device_id=device_data.get('device_id', 'unknown'),
            device_type=device_data.get('reading_type', 'unknown'),
            manufacturer=metadata.get('manufacturer'),
            model=metadata.get('model'),
            firmware_version=metadata.get('firmware_version'),
            measurement_types=device_capabilities.capabilities.get('measurement_types', []),
            accuracy_specifications=device_capabilities.capabilities.get('accuracy_specifications', {}),
            calibration_status=metadata.get('calibration_status', 'unknown'),
            battery_level=metadata.get('battery_level'),
            signal_quality=metadata.get('signal_quality'),
            connection_status=metadata.get('connection_status', 'unknown')
        )

    async def _create_device_context_optimized(self,
                                             device_data: Dict[str, Any],
                                             device_capabilities: DeviceCapabilities) -> DeviceContext:
        """Create device context with caching optimization"""
        device_id = device_data.get('device_id', 'unknown')
        device_type = device_data.get('reading_type', 'unknown')

        # Try to get cached device configuration
        cached_config = None
        if self.performance_enabled:
            cached_config = await self.performance_optimizer.get_cached_metadata(
                "device_config", f"{device_id}:{device_type}"
            )

        metadata = device_data.get('metadata', {})

        # Use cached data if available, otherwise use provided metadata
        if cached_config:
            return DeviceContext(
                device_id=device_id,
                device_type=device_type,
                manufacturer=cached_config.get('manufacturer') or metadata.get('manufacturer'),
                model=cached_config.get('model') or metadata.get('model'),
                firmware_version=metadata.get('firmware_version'),
                measurement_types=cached_config.get('capabilities', device_capabilities.capabilities.get('measurement_types', [])),
                accuracy_specifications=cached_config.get('accuracy_specs', device_capabilities.capabilities.get('accuracy_specifications', {})),
                calibration_status=metadata.get('calibration_status', 'unknown'),
                battery_level=metadata.get('battery_level'),
                signal_quality=metadata.get('signal_quality'),
                connection_status=metadata.get('connection_status', 'unknown')
            )
        else:
            # Fallback to original method
            return self._create_device_context(device_data, device_capabilities)

    async def _queue_async_enrichments(self,
                                     envelope_id: str,
                                     device_data: Dict[str, Any],
                                     auth_context: AuthContext):
        """Queue async enrichments for non-critical metadata"""
        if not self.performance_enabled:
            return

        # Example: Queue device registry lookup for detailed specifications
        async def enrich_device_specs():
            device_id = device_data.get('device_id')
            if device_id:
                # Simulate device registry lookup
                await asyncio.sleep(0.1)  # Simulate external API call
                return {
                    "detailed_specs": True,
                    "warranty_info": "Active",
                    "last_maintenance": "2024-01-15"
                }

        # Example: Queue patient medical history lookup
        async def enrich_patient_history():
            patient_id = device_data.get('patient_id')
            if patient_id:
                # Simulate medical history lookup
                await asyncio.sleep(0.05)  # Simulate database query
                return {
                    "recent_vitals": [],
                    "medical_alerts": [],
                    "care_plan_updates": []
                }

        # Queue enrichments
        await self.performance_optimizer.enqueue_async_enrichment(
            envelope_id, enrich_device_specs
        )

        await self.performance_optimizer.enqueue_async_enrichment(
            envelope_id, enrich_patient_history
        )
    
    def _create_processing_hints(self, 
                               device_data: Dict[str, Any],
                               quality: QualityMetadata,
                               security: SecurityMetadata) -> ProcessingHints:
        """Create processing hints for optimization"""
        
        # Determine priority based on data type and quality
        priority = "normal"
        if device_data.get('reading_type') in ['heart_rate', 'blood_pressure']:
            if quality.overall_quality_score < 0.5:
                priority = "high"  # Poor quality vital signs need attention
        
        # Determine if medical emergency
        medical_emergency = False
        if device_data.get('reading_type') == 'heart_rate':
            value = device_data.get('value', 0)
            if value > 150 or value < 40:
                medical_emergency = True
                priority = "critical"
        
        return ProcessingHints(
            priority_level=priority,
            processing_complexity="moderate" if security.hipaa_eligible else "simple",
            cache_eligible=quality.overall_quality_score > 0.7,
            batch_eligible=not medical_emergency,
            parallel_processing=True,
            medical_emergency=medical_emergency,
            requires_immediate_attention=medical_emergency,
            clinical_decision_support=security.hipaa_eligible
        )
    
    async def _create_fallback_envelope(self,
                                      device_data: Dict[str, Any],
                                      auth_context: AuthContext,
                                      request_context: RequestContext) -> EnhancedEnvelope:
        """Create minimal envelope when full creation fails"""
        envelope_id = str(uuid.uuid4())
        
        return EnhancedEnvelope(
            id=envelope_id,
            source=self.service_name,
            type="device.data.unknown",
            subject=f"Device/{device_data.get('device_id', 'unknown')}",
            time=datetime.now(timezone.utc).isoformat(),
            data=device_data,
            correlation_id=request_context.request_id
        )
    
    def _update_metrics(self, creation_time: float, quality_score: float, security: SecurityMetadata):
        """Update factory performance metrics"""
        self.creation_metrics["total_created"] += 1
        self.creation_metrics["creation_times"].append(creation_time)
        self.creation_metrics["quality_scores"].append(quality_score)
        
        if security.security_events:
            self.creation_metrics["security_events"] += len(security.security_events)
        
        # Keep only recent metrics (last 1000)
        if len(self.creation_metrics["creation_times"]) > 1000:
            self.creation_metrics["creation_times"] = self.creation_metrics["creation_times"][-1000:]
            self.creation_metrics["quality_scores"] = self.creation_metrics["quality_scores"][-1000:]
    
    def get_performance_metrics(self) -> Dict[str, Any]:
        """Get comprehensive factory performance metrics including optimizations"""
        creation_times = self.creation_metrics["creation_times"]
        quality_scores = self.creation_metrics["quality_scores"]

        base_metrics = {
            "total_envelopes_created": self.creation_metrics["total_created"],
            "security_events_detected": self.creation_metrics["security_events"],
            "performance_optimizations_enabled": self.performance_enabled
        }

        if creation_times:
            avg_creation_time = sum(creation_times) / len(creation_times)
            avg_quality_score = sum(quality_scores) / len(quality_scores) if quality_scores else 0

            base_metrics.update({
                "avg_creation_time_ms": round(avg_creation_time * 1000, 2),
                "avg_quality_score": round(avg_quality_score, 3),
                "p95_creation_time_ms": round(sorted(creation_times)[int(len(creation_times) * 0.95)] * 1000, 2) if len(creation_times) > 20 else 0,
                "quality_distribution": {
                    "excellent": sum(1 for s in quality_scores if s >= 0.9),
                    "good": sum(1 for s in quality_scores if 0.7 <= s < 0.9),
                    "fair": sum(1 for s in quality_scores if 0.5 <= s < 0.7),
                    "poor": sum(1 for s in quality_scores if s < 0.5)
                }
            })
        else:
            base_metrics["status"] = "no_data"

        # Add performance optimization metrics if enabled
        if self.performance_enabled:
            optimization_metrics = self.performance_optimizer.get_performance_metrics()
            base_metrics["performance_optimizations"] = optimization_metrics

        return base_metrics
