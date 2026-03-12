"""
Enhanced Clinical Event Envelope

Comprehensive clinical event envelope with rich clinical context,
temporal sophistication, and complete provenance tracking for
healthcare enterprise systems.
"""

import logging
import uuid
from datetime import datetime, timezone
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass, field, asdict
from enum import Enum
import json

logger = logging.getLogger(__name__)


class EventType(Enum):
    """Types of clinical events"""
    CLINICAL_ASSERTION = "clinical_assertion"
    MEDICATION_ORDER = "medication_order"
    LABORATORY_RESULT = "laboratory_result"
    CLINICAL_DECISION = "clinical_decision"
    PATIENT_ENCOUNTER = "patient_encounter"
    ADVERSE_EVENT = "adverse_event"
    THERAPEUTIC_RESPONSE = "therapeutic_response"
    WORKFLOW_TRANSITION = "workflow_transition"
    SYSTEM_EVENT = "system_event"
    CLINICAL_QUERY = "clinical_query"


class EventSeverity(Enum):
    """Severity levels for clinical events"""
    CRITICAL = "critical"
    HIGH = "high"
    MODERATE = "moderate"
    LOW = "low"
    INFORMATIONAL = "informational"


class EventStatus(Enum):
    """Status of clinical events"""
    CREATED = "created"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"
    SUPERSEDED = "superseded"


@dataclass
class ClinicalContext:
    """Rich clinical context for events"""
    # Patient Context
    patient_id: str
    patient_mrn: Optional[str] = None
    patient_demographics: Dict[str, Any] = field(default_factory=dict)
    
    # Encounter Context
    encounter_id: Optional[str] = None
    encounter_type: Optional[str] = None  # inpatient, outpatient, emergency, etc.
    encounter_status: Optional[str] = None
    admission_date: Optional[datetime] = None
    discharge_date: Optional[datetime] = None
    
    # Care Team Context
    primary_provider_id: Optional[str] = None
    attending_physician_id: Optional[str] = None
    care_team_members: List[Dict[str, Any]] = field(default_factory=list)
    
    # Facility Context
    facility_id: Optional[str] = None
    facility_name: Optional[str] = None
    department_id: Optional[str] = None
    department_name: Optional[str] = None
    unit_id: Optional[str] = None
    unit_name: Optional[str] = None
    
    # Clinical State Context
    active_diagnoses: List[Dict[str, Any]] = field(default_factory=list)
    active_medications: List[Dict[str, Any]] = field(default_factory=list)
    active_allergies: List[Dict[str, Any]] = field(default_factory=list)
    vital_signs: Dict[str, Any] = field(default_factory=dict)
    laboratory_values: Dict[str, Any] = field(default_factory=dict)
    
    # Risk Factors
    risk_factors: List[str] = field(default_factory=list)
    clinical_warnings: List[Dict[str, Any]] = field(default_factory=list)


@dataclass
class TemporalContext:
    """Sophisticated temporal context for clinical events"""
    # Required Event Timestamps
    event_time: datetime  # When the clinical event actually occurred
    system_time: datetime  # When the system recorded the event

    # Optional Event Timestamps
    processing_time: Optional[datetime] = None  # When processing started
    completion_time: Optional[datetime] = None  # When processing completed

    # Clinical Time Context
    clinical_day: Optional[int] = None  # Day of admission/treatment
    shift_context: Optional[str] = None  # day, evening, night
    clinical_phase: Optional[str] = None  # admission, treatment, discharge

    # Temporal Relationships
    related_event_times: List[datetime] = field(default_factory=list)
    sequence_number: Optional[int] = None
    temporal_window_start: Optional[datetime] = None
    temporal_window_end: Optional[datetime] = None

    # Timing Constraints
    urgency_level: Optional[str] = None  # stat, urgent, routine
    response_deadline: Optional[datetime] = None
    escalation_time: Optional[datetime] = None


@dataclass
class ProvenanceContext:
    """Complete provenance and audit trail"""
    # Required fields first
    source_system: str
    created_by: str
    created_at: datetime

    # Source Information
    source_system_version: Optional[str] = None
    source_user_id: Optional[str] = None
    source_user_role: Optional[str] = None
    source_session_id: Optional[str] = None

    # Data Lineage
    data_sources: List[Dict[str, Any]] = field(default_factory=list)
    transformation_history: List[Dict[str, Any]] = field(default_factory=list)
    validation_results: List[Dict[str, Any]] = field(default_factory=list)

    # Processing Context
    processing_engine_version: Optional[str] = None
    reasoning_algorithms: List[str] = field(default_factory=list)
    confidence_scores: Dict[str, float] = field(default_factory=dict)
    evidence_sources: List[str] = field(default_factory=list)

    # Audit Trail
    modified_by: Optional[str] = None
    modified_at: Optional[datetime] = None
    access_log: List[Dict[str, Any]] = field(default_factory=list)

    # Compliance Context
    regulatory_context: Dict[str, Any] = field(default_factory=dict)
    privacy_classification: Optional[str] = None
    retention_policy: Optional[str] = None


@dataclass
class EventMetadata:
    """Comprehensive event metadata"""
    # Event Identity
    event_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    event_version: str = "1.0"
    event_schema_version: str = "2.0"
    
    # Event Classification
    event_type: EventType = EventType.SYSTEM_EVENT
    event_subtype: Optional[str] = None
    event_category: Optional[str] = None
    event_severity: EventSeverity = EventSeverity.INFORMATIONAL
    event_status: EventStatus = EventStatus.CREATED
    
    # Event Relationships
    parent_event_id: Optional[str] = None
    root_event_id: Optional[str] = None
    correlation_id: Optional[str] = None
    causation_id: Optional[str] = None
    related_event_ids: List[str] = field(default_factory=list)
    
    # Processing Metadata
    processing_priority: int = 5  # 1-10 scale
    retry_count: int = 0
    max_retries: int = 3
    timeout_seconds: int = 300
    
    # Quality Metadata
    data_quality_score: Optional[float] = None
    completeness_score: Optional[float] = None
    consistency_score: Optional[float] = None
    
    # Custom Extensions
    custom_attributes: Dict[str, Any] = field(default_factory=dict)
    tags: List[str] = field(default_factory=list)


@dataclass
class ClinicalEventEnvelope:
    """
    Enhanced Clinical Event Envelope
    
    Comprehensive wrapper for all clinical events with rich context,
    temporal sophistication, and complete provenance tracking.
    """
    # Core Event Data
    event_data: Dict[str, Any]
    
    # Context Information
    clinical_context: ClinicalContext
    temporal_context: TemporalContext
    provenance_context: ProvenanceContext
    
    # Event Metadata
    metadata: EventMetadata
    
    # Event Envelope Metadata
    envelope_version: str = "2.0"
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    
    def __post_init__(self):
        """Post-initialization validation and setup"""
        # Ensure temporal context consistency
        if not self.temporal_context.system_time:
            self.temporal_context.system_time = self.created_at
        
        # Set provenance created_at if not provided
        if not self.provenance_context.created_at:
            self.provenance_context.created_at = self.created_at
        
        # Validate required fields
        self._validate_envelope()
    
    def _validate_envelope(self):
        """Validate envelope completeness and consistency"""
        errors = []
        
        # Required field validation
        # Handle both dictionary and object types for clinical_context
        if isinstance(self.clinical_context, dict):
            # Dictionary access for patient_id
            if not self.clinical_context.get('patient_id'):
                errors.append("Patient ID is required in clinical context")
        else:
            # Object attribute access for patient_id
            if not self.clinical_context.patient_id:
                errors.append("Patient ID is required in clinical context")
        
        if not self.provenance_context.source_system:
            errors.append("Source system is required in provenance context")
        
        if not self.provenance_context.created_by:
            errors.append("Created by is required in provenance context")
        
        # Temporal consistency validation
        if (self.temporal_context.completion_time and 
            self.temporal_context.processing_time and
            self.temporal_context.completion_time < self.temporal_context.processing_time):
            errors.append("Completion time cannot be before processing time")
        
        if errors:
            raise ValueError(f"Event envelope validation failed: {'; '.join(errors)}")
    
    def add_related_event(self, event_id: str, relationship_type: str = "related"):
        """Add related event reference"""
        if event_id not in self.metadata.related_event_ids:
            self.metadata.related_event_ids.append(event_id)
        
        # Add to custom attributes for relationship type tracking
        if "event_relationships" not in self.metadata.custom_attributes:
            self.metadata.custom_attributes["event_relationships"] = {}
        
        self.metadata.custom_attributes["event_relationships"][event_id] = relationship_type
    
    def add_provenance_entry(self, source: str, transformation: str, 
                           confidence: Optional[float] = None):
        """Add provenance tracking entry"""
        entry = {
            "source": source,
            "transformation": transformation,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "confidence": confidence
        }
        
        self.provenance_context.transformation_history.append(entry)
        
        if source not in self.provenance_context.evidence_sources:
            self.provenance_context.evidence_sources.append(source)
        
        if confidence is not None:
            self.provenance_context.confidence_scores[source] = confidence
    
    def update_status(self, new_status: EventStatus, updated_by: str):
        """Update event status with audit trail"""
        old_status = self.metadata.event_status
        self.metadata.event_status = new_status
        
        # Add to access log
        access_entry = {
            "action": "status_update",
            "old_status": old_status.value,
            "new_status": new_status.value,
            "updated_by": updated_by,
            "timestamp": datetime.now(timezone.utc).isoformat()
        }
        
        self.provenance_context.access_log.append(access_entry)
        
        # Update modification tracking
        self.provenance_context.modified_by = updated_by
        self.provenance_context.modified_at = datetime.now(timezone.utc)
    
    def add_clinical_warning(self, warning_type: str, severity: str, 
                           description: str, source: str):
        """Add clinical warning to context"""
        warning = {
            "type": warning_type,
            "severity": severity,
            "description": description,
            "source": source,
            "timestamp": datetime.now(timezone.utc).isoformat()
        }
        
        self.clinical_context.clinical_warnings.append(warning)
        
        # Update event severity if warning is more severe
        warning_severity_map = {
            "critical": EventSeverity.CRITICAL,
            "high": EventSeverity.HIGH,
            "moderate": EventSeverity.MODERATE,
            "low": EventSeverity.LOW
        }
        
        # Convert severity to string if it's not already a string
        if not isinstance(severity, str):
            severity = str(severity)
        warning_severity = warning_severity_map.get(severity.lower(), EventSeverity.LOW)
        
        # Escalate event severity if needed
        severity_order = [
            EventSeverity.INFORMATIONAL,
            EventSeverity.LOW,
            EventSeverity.MODERATE,
            EventSeverity.HIGH,
            EventSeverity.CRITICAL
        ]
        
        current_index = severity_order.index(self.metadata.event_severity)
        warning_index = severity_order.index(warning_severity)
        
        if warning_index > current_index:
            self.metadata.event_severity = warning_severity
    
    def calculate_processing_duration(self) -> Optional[float]:
        """Calculate processing duration in seconds"""
        if (self.temporal_context.processing_time and 
            self.temporal_context.completion_time):
            delta = (self.temporal_context.completion_time - 
                    self.temporal_context.processing_time)
            return delta.total_seconds()
        return None
    
    def get_event_age_seconds(self) -> float:
        """Get event age in seconds"""
        now = datetime.now(timezone.utc)
        return (now - self.temporal_context.event_time).total_seconds()
    
    def is_expired(self, max_age_seconds: int) -> bool:
        """Check if event has expired based on age"""
        return self.get_event_age_seconds() > max_age_seconds
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert envelope to dictionary for serialization"""
        return {
            "event_data": self.event_data,
            "clinical_context": asdict(self.clinical_context),
            "temporal_context": asdict(self.temporal_context),
            "provenance_context": asdict(self.provenance_context),
            "metadata": asdict(self.metadata),
            "envelope_version": self.envelope_version,
            "created_at": self.created_at.isoformat()
        }
    
    def to_json(self) -> str:
        """Convert envelope to JSON string"""
        return json.dumps(self.to_dict(), default=str, indent=2)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ClinicalEventEnvelope':
        """Create envelope from dictionary"""
        # Convert datetime strings back to datetime objects
        def parse_datetime(dt_str):
            if isinstance(dt_str, str):
                return datetime.fromisoformat(dt_str.replace('Z', '+00:00'))
            return dt_str
        
        # Parse temporal context dates
        temporal_data = data.get("temporal_context", {})
        for field in ["event_time", "system_time", "processing_time", "completion_time"]:
            if field in temporal_data and temporal_data[field]:
                temporal_data[field] = parse_datetime(temporal_data[field])
        
        # Parse provenance context dates
        provenance_data = data.get("provenance_context", {})
        for field in ["created_at", "modified_at"]:
            if field in provenance_data and provenance_data[field]:
                provenance_data[field] = parse_datetime(provenance_data[field])
        
        # Parse envelope created_at
        if "created_at" in data:
            data["created_at"] = parse_datetime(data["created_at"])
        
        return cls(
            event_data=data["event_data"],
            clinical_context=ClinicalContext(**data["clinical_context"]),
            temporal_context=TemporalContext(**temporal_data),
            provenance_context=ProvenanceContext(**provenance_data),
            metadata=EventMetadata(**data["metadata"]),
            envelope_version=data.get("envelope_version", "2.0"),
            created_at=data.get("created_at", datetime.now(timezone.utc))
        )
    
    @classmethod
    def from_json(cls, json_str: str) -> 'ClinicalEventEnvelope':
        """Create envelope from JSON string"""
        data = json.loads(json_str)
        return cls.from_dict(data)
    
    def add_clinical_warning(self, warning_type: str, severity: str,
                           description: str, source: str):
        """Add clinical warning to context"""
        warning = {
            "type": warning_type,
            "severity": severity,
            "description": description,
            "source": source,
            "timestamp": datetime.now(timezone.utc).isoformat()
        }

        self.clinical_context.clinical_warnings.append(warning)

        # Update event severity if warning is more severe
        warning_severity_map = {
            "critical": EventSeverity.CRITICAL,
            "high": EventSeverity.HIGH,
            "moderate": EventSeverity.MODERATE,
            "low": EventSeverity.LOW
        }

        # Convert severity to string if it's not already a string
        if not isinstance(severity, str):
            severity = str(severity)
        warning_severity = warning_severity_map.get(severity.lower(), EventSeverity.LOW)

        # Escalate event severity if needed
        severity_order = [
            EventSeverity.INFORMATIONAL,
            EventSeverity.LOW,
            EventSeverity.MODERATE,
            EventSeverity.HIGH,
            EventSeverity.CRITICAL
        ]

        current_index = severity_order.index(self.metadata.event_severity)
        warning_index = severity_order.index(warning_severity)

        if warning_index > current_index:
            self.metadata.event_severity = warning_severity

    def calculate_processing_duration(self) -> Optional[float]:
        """Calculate processing duration in seconds"""
        if (self.temporal_context.processing_time and
            self.temporal_context.completion_time):
            delta = (self.temporal_context.completion_time -
                    self.temporal_context.processing_time)
            return delta.total_seconds()
        return None

    def get_event_age_seconds(self) -> float:
        """Get event age in seconds"""
        now = datetime.now(timezone.utc)
        return (now - self.temporal_context.event_time).total_seconds()

    def is_expired(self, max_age_seconds: int) -> bool:
        """Check if event has expired based on age"""
        return self.get_event_age_seconds() > max_age_seconds

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ClinicalEventEnvelope':
        """Create envelope from dictionary"""
        # Convert datetime strings back to datetime objects
        def parse_datetime(dt_str):
            if isinstance(dt_str, str):
                return datetime.fromisoformat(dt_str.replace('Z', '+00:00'))
            return dt_str

        # Parse temporal context dates
        temporal_data = data.get("temporal_context", {})
        for field in ["event_time", "system_time", "processing_time", "completion_time"]:
            if field in temporal_data and temporal_data[field]:
                temporal_data[field] = parse_datetime(temporal_data[field])

        # Parse provenance context dates
        provenance_data = data.get("provenance_context", {})
        for field in ["created_at", "modified_at"]:
            if field in provenance_data and provenance_data[field]:
                provenance_data[field] = parse_datetime(provenance_data[field])

        # Parse envelope created_at
        if "created_at" in data:
            data["created_at"] = parse_datetime(data["created_at"])

        return cls(
            event_data=data["event_data"],
            clinical_context=ClinicalContext(**data["clinical_context"]),
            temporal_context=TemporalContext(**temporal_data),
            provenance_context=ProvenanceContext(**provenance_data),
            metadata=EventMetadata(**data["metadata"]),
            envelope_version=data.get("envelope_version", "2.0"),
            created_at=data.get("created_at", datetime.now(timezone.utc))
        )

    @classmethod
    def from_json(cls, json_str: str) -> 'ClinicalEventEnvelope':
        """Create envelope from JSON string"""
        data = json.loads(json_str)
        return cls.from_dict(data)

    def __str__(self) -> str:
        """String representation of envelope"""
        return (f"ClinicalEventEnvelope("
                f"event_id={self.metadata.event_id}, "
                f"type={self.metadata.event_type.value}, "
                f"patient_id={self.clinical_context.patient_id}, "
                f"status={self.metadata.event_status.value})")

    def __repr__(self) -> str:
        """Detailed representation of envelope"""
        return self.__str__()
