"""
Enhanced Message Envelope Implementation

Extends the existing EventEnvelope with enterprise-grade features including
security metadata, quality assessment, patient context, and HIPAA compliance.
"""

import hashlib
import json
import logging
import time
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Union
from dataclasses import dataclass, field
from enum import Enum
import uuid

logger = logging.getLogger(__name__)


class EnvelopeVersion(str, Enum):
    """Envelope schema versions"""
    V1_0 = "1.0"
    V2_0 = "2.0"  # Enhanced version
    V2_1 = "2.1"  # With security metadata
    V2_2 = "2.2"  # With quality assessment


class SecurityLevel(str, Enum):
    """Security classification levels"""
    PUBLIC = "public"
    INTERNAL = "internal"
    CONFIDENTIAL = "confidential"
    RESTRICTED = "restricted"  # HIPAA protected


class QualityLevel(str, Enum):
    """Data quality levels"""
    EXCELLENT = "excellent"  # 90-100%
    GOOD = "good"           # 70-89%
    FAIR = "fair"           # 50-69%
    POOR = "poor"           # <50%
    UNKNOWN = "unknown"


class ComplianceStatus(str, Enum):
    """Compliance status indicators"""
    COMPLIANT = "compliant"
    NON_COMPLIANT = "non_compliant"
    PENDING_REVIEW = "pending_review"
    EXEMPT = "exempt"


@dataclass
class SecurityMetadata:
    """Security and compliance metadata"""
    # Authentication context
    auth_method: str                              # JWT, API_KEY, etc.
    user_id: Optional[str] = None                # Authenticated user ID
    user_role: Optional[str] = None              # User role (doctor, patient, etc.)
    user_permissions: List[str] = field(default_factory=list)  # Granted permissions
    
    # Request integrity
    request_timestamp: Optional[int] = None       # Original request timestamp
    payload_hash: Optional[str] = None           # SHA-256 hash of payload
    source_ip: Optional[str] = None              # Request source IP
    user_agent: Optional[str] = None             # Request user agent
    
    # Compliance flags
    hipaa_eligible: bool = False                 # HIPAA protected data
    gdpr_compliant: bool = False                 # GDPR compliance status
    data_retention_days: Optional[int] = None    # Data retention policy
    
    # Security events
    security_events: List[str] = field(default_factory=list)  # Security warnings/events
    risk_score: float = 0.0                      # Security risk score (0-1)


@dataclass
class QualityMetadata:
    """Data quality assessment metadata"""
    # Quality dimensions (0-1 scores)
    completeness_score: float = 0.0             # Percentage of expected fields present
    validity_score: float = 0.0                 # Values within expected ranges
    consistency_score: float = 0.0              # Consistency with historical data
    accuracy_score: float = 0.0                 # Confidence based on device calibration
    timeliness_score: float = 0.0               # Freshness score
    
    # Overall quality
    overall_quality_score: float = 0.0          # Weighted average of dimensions
    quality_level: QualityLevel = QualityLevel.UNKNOWN
    
    # Quality flags
    anomaly_detected: bool = False               # ML anomaly detection flag
    manual_review_required: bool = False         # Requires human review
    quality_warnings: List[str] = field(default_factory=list)  # Quality issues
    
    # Validation results
    validation_passed: bool = True               # All validations passed
    validation_errors: List[str] = field(default_factory=list)  # Validation failures
    validation_warnings: List[str] = field(default_factory=list)  # Validation warnings


@dataclass
class PatientContext:
    """Patient context and privacy metadata"""
    patient_id: Optional[str] = None             # Patient identifier
    patient_consent_status: str = "unknown"     # Consent status
    consent_version: Optional[str] = None       # Consent version
    data_sharing_permissions: List[str] = field(default_factory=list)  # Sharing permissions
    
    # Privacy settings
    anonymization_level: str = "identified"     # identified, pseudonymized, anonymous
    geographic_restrictions: List[str] = field(default_factory=list)  # Geographic limits
    purpose_limitations: List[str] = field(default_factory=list)  # Data usage purposes
    
    # Clinical context
    clinical_conditions: List[str] = field(default_factory=list)  # Known conditions
    medication_list: List[str] = field(default_factory=list)  # Current medications
    care_team: List[str] = field(default_factory=list)  # Care team members


@dataclass
class DeviceContext:
    """Device context and capabilities metadata"""
    device_id: str                               # Device identifier
    device_type: str                             # Device type
    manufacturer: Optional[str] = None           # Device manufacturer
    model: Optional[str] = None                  # Device model
    firmware_version: Optional[str] = None      # Firmware version
    
    # Device capabilities
    measurement_types: List[str] = field(default_factory=list)  # Supported measurements
    accuracy_specifications: Dict[str, Any] = field(default_factory=dict)  # Accuracy specs
    calibration_status: str = "unknown"         # Calibration status
    last_calibration_date: Optional[str] = None # Last calibration
    
    # Device health
    battery_level: Optional[float] = None       # Battery percentage
    signal_quality: Optional[str] = None        # Signal quality indicator
    connection_status: str = "unknown"          # Connection status
    error_codes: List[str] = field(default_factory=list)  # Device error codes


@dataclass
class ProcessingHints:
    """Processing optimization hints"""
    priority_level: str = "normal"              # low, normal, high, critical
    processing_complexity: str = "simple"       # simple, moderate, complex
    expected_processing_time_ms: Optional[int] = None  # Expected processing time
    
    # Routing hints
    preferred_processing_region: Optional[str] = None  # Geographic preference
    required_capabilities: List[str] = field(default_factory=list)  # Required processing capabilities
    
    # Performance hints
    cache_eligible: bool = True                  # Can be cached
    batch_eligible: bool = True                  # Can be batched
    parallel_processing: bool = True             # Can be processed in parallel
    
    # Clinical hints
    medical_emergency: bool = False              # Medical emergency flag
    requires_immediate_attention: bool = False   # Immediate attention required
    clinical_decision_support: bool = False      # Requires CDS integration


@dataclass
class LineageMetadata:
    """Message lineage and tracing metadata"""
    # Required fields first (no defaults)
    trace_id: str                                # Distributed trace ID
    span_id: str                                 # Current span ID
    message_id: str                              # Unique message ID
    created_at: str                              # Envelope creation time

    # Optional fields with defaults
    parent_span_id: Optional[str] = None        # Parent span ID
    correlation_id: Optional[str] = None        # Correlation ID
    causation_id: Optional[str] = None          # Causation ID
    ingestion_time: Optional[str] = None        # Data ingestion time
    processing_start_time: Optional[str] = None # Processing start time

    # Fields with factory defaults
    processing_chain: List[str] = field(default_factory=list)  # Services that processed this
    transformation_history: List[str] = field(default_factory=list)  # Applied transformations


@dataclass
class EnhancedEnvelope:
    """
    Enhanced message envelope with enterprise-grade metadata
    
    Extends the basic EventEnvelope with security, quality, patient context,
    device context, processing hints, and lineage tracking.
    """
    
    # Core envelope fields (compatible with existing EventEnvelope)
    id: str                                      # Unique envelope identifier
    source: str                                  # Event source (service name)
    type: str                                    # Event type
    subject: str                                 # Event subject
    time: str                                    # Event timestamp (ISO 8601)
    data: Dict[str, Any]                         # Event payload
    version: str = EnvelopeVersion.V2_2          # Schema version
    
    # Enhanced metadata
    security: Optional[SecurityMetadata] = None
    quality: Optional[QualityMetadata] = None
    patient_context: Optional[PatientContext] = None
    device_context: Optional[DeviceContext] = None
    processing_hints: Optional[ProcessingHints] = None
    lineage: Optional[LineageMetadata] = None
    
    # Legacy compatibility
    correlation_id: Optional[str] = None         # For backward compatibility
    causation_id: Optional[str] = None          # For backward compatibility
    metadata: Optional[Dict[str, Any]] = None   # Additional metadata
    
    def __post_init__(self):
        """Initialize default values and ensure consistency"""
        if self.metadata is None:
            self.metadata = {}
        
        # Ensure lineage metadata exists
        if self.lineage is None:
            self.lineage = LineageMetadata(
                trace_id=str(uuid.uuid4()),
                span_id=str(uuid.uuid4()),
                message_id=self.id,
                correlation_id=self.correlation_id,
                causation_id=self.causation_id,
                created_at=self.time
            )
        
        # Sync correlation/causation IDs for backward compatibility
        if self.correlation_id and self.lineage:
            self.lineage.correlation_id = self.correlation_id
        if self.causation_id and self.lineage:
            self.lineage.causation_id = self.causation_id
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary with proper serialization"""
        result = {
            "id": self.id,
            "source": self.source,
            "type": self.type,
            "subject": self.subject,
            "time": self.time,
            "data": self.data,
            "version": self.version,
            "correlation_id": self.correlation_id,
            "causation_id": self.causation_id,
            "metadata": self.metadata
        }
        
        # Add enhanced metadata if present
        if self.security:
            result["security"] = self._serialize_dataclass(self.security)
        if self.quality:
            result["quality"] = self._serialize_dataclass(self.quality)
        if self.patient_context:
            result["patient_context"] = self._serialize_dataclass(self.patient_context)
        if self.device_context:
            result["device_context"] = self._serialize_dataclass(self.device_context)
        if self.processing_hints:
            result["processing_hints"] = self._serialize_dataclass(self.processing_hints)
        if self.lineage:
            result["lineage"] = self._serialize_dataclass(self.lineage)
        
        return result
    
    def _serialize_dataclass(self, obj: Any) -> Dict[str, Any]:
        """Serialize dataclass to dictionary"""
        if hasattr(obj, '__dataclass_fields__'):
            result = {}
            for field_name, field_def in obj.__dataclass_fields__.items():
                value = getattr(obj, field_name)
                if isinstance(value, Enum):
                    result[field_name] = value.value
                elif hasattr(value, '__dataclass_fields__'):
                    result[field_name] = self._serialize_dataclass(value)
                else:
                    result[field_name] = value
            return result
        return obj
    
    def to_json(self) -> str:
        """Convert to JSON string"""
        return json.dumps(self.to_dict(), default=str, indent=2)
    
    def get_legacy_envelope(self) -> Dict[str, Any]:
        """Get legacy EventEnvelope format for backward compatibility"""
        return {
            "id": self.id,
            "source": self.source,
            "type": self.type,
            "subject": self.subject,
            "time": self.time,
            "data": self.data,
            "correlation_id": self.correlation_id,
            "causation_id": self.causation_id,
            "metadata": self.metadata,
            "version": "1.0"  # Legacy version
        }
    
    def calculate_payload_hash(self) -> str:
        """Calculate SHA-256 hash of the payload for integrity verification"""
        payload_str = json.dumps(self.data, sort_keys=True, default=str)
        return hashlib.sha256(payload_str.encode()).hexdigest()
    
    def add_processing_step(self, service_name: str, transformation: Optional[str] = None):
        """Add a processing step to the lineage"""
        if self.lineage:
            self.lineage.processing_chain.append(service_name)
            if transformation:
                self.lineage.transformation_history.append(transformation)
    
    def update_quality_score(self, dimension: str, score: float):
        """Update a specific quality dimension score"""
        if not self.quality:
            self.quality = QualityMetadata()
        
        if hasattr(self.quality, f"{dimension}_score"):
            setattr(self.quality, f"{dimension}_score", score)
            self._recalculate_overall_quality()
    
    def _recalculate_overall_quality(self):
        """Recalculate overall quality score"""
        if not self.quality:
            return
        
        scores = [
            self.quality.completeness_score,
            self.quality.validity_score,
            self.quality.consistency_score,
            self.quality.accuracy_score,
            self.quality.timeliness_score
        ]
        
        # Weighted average (can be customized)
        weights = [0.25, 0.25, 0.2, 0.2, 0.1]
        self.quality.overall_quality_score = sum(s * w for s, w in zip(scores, weights))
        
        # Determine quality level
        if self.quality.overall_quality_score >= 0.9:
            self.quality.quality_level = QualityLevel.EXCELLENT
        elif self.quality.overall_quality_score >= 0.7:
            self.quality.quality_level = QualityLevel.GOOD
        elif self.quality.overall_quality_score >= 0.5:
            self.quality.quality_level = QualityLevel.FAIR
        else:
            self.quality.quality_level = QualityLevel.POOR
