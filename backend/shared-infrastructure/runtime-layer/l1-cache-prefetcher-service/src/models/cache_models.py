"""
L1 Cache and Prefetcher Data Models
Defines the data structures for ultra-fast clinical data caching
"""

from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List, Union
from enum import Enum
from pydantic import BaseModel, Field, validator
import hashlib
import json


class CacheKeyType(str, Enum):
    """Types of cache keys for different data categories"""
    PATIENT_CONTEXT = "patient_context"
    CLINICAL_DATA = "clinical_data"
    MEDICATION_DATA = "medication_data"
    GUIDELINE_DATA = "guideline_data"
    SEMANTIC_MESH = "semantic_mesh"
    WORKFLOW_STATE = "workflow_state"
    USER_SESSION = "user_session"
    EVIDENCE_ENVELOPE = "evidence_envelope"


class AccessPattern(BaseModel):
    """Tracks data access patterns for ML prediction"""
    key: str
    key_type: CacheKeyType
    access_count: int = 0
    last_accessed: datetime
    access_frequency: float = 0.0  # accesses per hour
    session_correlation: Dict[str, int] = Field(default_factory=dict)
    temporal_pattern: List[int] = Field(default_factory=list)  # hour-of-day access pattern
    user_correlation: Dict[str, int] = Field(default_factory=dict)

    def update_access(self, session_id: Optional[str] = None, user_id: Optional[str] = None):
        """Update access patterns with new access"""
        self.access_count += 1
        current_time = datetime.utcnow()

        # Update frequency (exponential decay)
        if hasattr(self, 'last_accessed'):
            time_diff = (current_time - self.last_accessed).total_seconds() / 3600  # hours
            decay_factor = 0.95  # Decay old frequency data
            self.access_frequency = (self.access_frequency * decay_factor) + (1.0 / max(time_diff, 0.1))

        self.last_accessed = current_time

        # Update temporal pattern (hour of day)
        hour = current_time.hour
        if len(self.temporal_pattern) < 24:
            self.temporal_pattern = [0] * 24
        self.temporal_pattern[hour] += 1

        # Update session correlation
        if session_id:
            self.session_correlation[session_id] = self.session_correlation.get(session_id, 0) + 1

        # Update user correlation
        if user_id:
            self.user_correlation[user_id] = self.user_correlation.get(user_id, 0) + 1


class CacheEntry(BaseModel):
    """Individual cache entry with metadata"""
    key: str
    key_type: CacheKeyType
    data: Dict[str, Any]
    created_at: datetime = Field(default_factory=datetime.utcnow)
    last_accessed: datetime = Field(default_factory=datetime.utcnow)
    access_count: int = 1
    ttl_seconds: int = 10  # L1 cache default TTL
    expires_at: datetime
    size_bytes: int
    checksum: str
    source_system: Optional[str] = None
    session_id: Optional[str] = None
    user_id: Optional[str] = None

    def __init__(self, **data):
        super().__init__(**data)
        if not hasattr(self, 'expires_at') or self.expires_at is None:
            self.expires_at = self.created_at + timedelta(seconds=self.ttl_seconds)
        if not hasattr(self, 'size_bytes'):
            self.size_bytes = len(json.dumps(self.data, default=str).encode('utf-8'))
        if not hasattr(self, 'checksum'):
            self.checksum = self._generate_checksum()

    def _generate_checksum(self) -> str:
        """Generate checksum for data integrity"""
        data_str = json.dumps(self.data, sort_keys=True, default=str)
        return hashlib.sha256(data_str.encode()).hexdigest()[:16]  # Short checksum for speed

    def is_expired(self) -> bool:
        """Check if cache entry has expired"""
        return datetime.utcnow() > self.expires_at

    def is_valid(self) -> bool:
        """Verify data integrity"""
        return self._generate_checksum() == self.checksum

    def access(self, session_id: Optional[str] = None, user_id: Optional[str] = None):
        """Record access to this cache entry"""
        self.last_accessed = datetime.utcnow()
        self.access_count += 1
        if session_id:
            self.session_id = session_id
        if user_id:
            self.user_id = user_id

    def extend_ttl(self, additional_seconds: int):
        """Extend the TTL of this cache entry"""
        self.expires_at = datetime.utcnow() + timedelta(seconds=additional_seconds)


class PrefetchPrediction(BaseModel):
    """ML-based prefetch prediction"""
    key: str
    key_type: CacheKeyType
    confidence: float = Field(..., ge=0.0, le=1.0)
    predicted_access_time: datetime
    time_to_access_seconds: int
    session_context: Dict[str, Any] = Field(default_factory=dict)
    user_context: Dict[str, Any] = Field(default_factory=dict)
    trigger_factors: List[str] = Field(default_factory=list)
    priority_score: float = 0.0

    def __init__(self, **data):
        super().__init__(**data)
        if not hasattr(self, 'time_to_access_seconds'):
            self.time_to_access_seconds = int((self.predicted_access_time - datetime.utcnow()).total_seconds())
        # Calculate priority score based on confidence and urgency
        urgency_factor = max(0.1, 1.0 / max(self.time_to_access_seconds, 1))
        self.priority_score = self.confidence * urgency_factor


class CacheMetrics(BaseModel):
    """Cache performance metrics"""
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    total_entries: int = 0
    total_size_bytes: int = 0
    hit_count: int = 0
    miss_count: int = 0
    prefetch_hit_count: int = 0
    prefetch_miss_count: int = 0
    average_response_time_ms: float = 0.0
    p95_response_time_ms: float = 0.0
    p99_response_time_ms: float = 0.0
    eviction_count: int = 0
    expired_count: int = 0
    memory_utilization: float = 0.0

    @property
    def hit_rate(self) -> float:
        """Calculate cache hit rate"""
        total_requests = self.hit_count + self.miss_count
        return self.hit_count / total_requests if total_requests > 0 else 0.0

    @property
    def prefetch_accuracy(self) -> float:
        """Calculate prefetch prediction accuracy"""
        total_prefetch = self.prefetch_hit_count + self.prefetch_miss_count
        return self.prefetch_hit_count / total_prefetch if total_prefetch > 0 else 0.0


class SessionContext(BaseModel):
    """Session-specific context for intelligent caching"""
    session_id: str
    user_id: str
    workflow_type: str
    patient_context: Optional[Dict[str, Any]] = None
    clinical_domain: Optional[str] = None
    started_at: datetime = Field(default_factory=datetime.utcnow)
    last_activity: datetime = Field(default_factory=datetime.utcnow)
    access_pattern: List[str] = Field(default_factory=list)  # Recently accessed keys
    predicted_next: List[str] = Field(default_factory=list)  # Predicted next accesses
    cache_budget_mb: int = 10  # Per-session cache budget
    priority_boost: float = 1.0  # Session-specific priority multiplier

    def update_activity(self, accessed_key: Optional[str] = None):
        """Update session activity"""
        self.last_activity = datetime.utcnow()
        if accessed_key:
            self.access_pattern.append(accessed_key)
            # Keep only last 50 accesses for pattern analysis
            if len(self.access_pattern) > 50:
                self.access_pattern = self.access_pattern[-50:]

    def is_active(self, inactive_threshold_minutes: int = 30) -> bool:
        """Check if session is still active"""
        threshold = datetime.utcnow() - timedelta(minutes=inactive_threshold_minutes)
        return self.last_activity > threshold


class CacheRequest(BaseModel):
    """Request to cache data"""
    key: str
    key_type: CacheKeyType
    data: Dict[str, Any]
    ttl_seconds: int = 10
    session_id: Optional[str] = None
    user_id: Optional[str] = None
    source_system: Optional[str] = None
    priority: int = Field(default=1, ge=1, le=10)


class CacheResponse(BaseModel):
    """Response from cache operations"""
    key: str
    data: Optional[Dict[str, Any]] = None
    hit: bool
    response_time_ms: float
    from_prefetch: bool = False
    expires_at: Optional[datetime] = None
    cache_level: str = "L1"  # L1, L2, or miss


class PrefetchRequest(BaseModel):
    """Request to prefetch data"""
    keys: List[str]
    session_context: Optional[SessionContext] = None
    prediction_confidence_threshold: float = Field(default=0.7, ge=0.0, le=1.0)
    max_prefetch_items: int = Field(default=50, ge=1, le=200)
    prefetch_budget_mb: int = Field(default=100, ge=1, le=500)


class PrefetchResponse(BaseModel):
    """Response from prefetch operations"""
    requested_keys: List[str]
    prefetched_keys: List[str]
    skipped_keys: List[str]
    total_prefetched: int
    total_size_mb: float
    processing_time_ms: float
    predictions_used: int