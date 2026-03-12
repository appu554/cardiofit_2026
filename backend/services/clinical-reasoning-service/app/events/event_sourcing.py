"""
Advanced Idempotency & Event Sourcing

Sophisticated idempotency strategies and event sourcing patterns
for clinical audit trails, ensuring data consistency and
complete clinical event history tracking.
"""

import logging
import asyncio
import hashlib
import json
from datetime import datetime, timezone, timedelta
from typing import Dict, List, Optional, Any, Tuple, Set
from dataclasses import dataclass, field, asdict
from enum import Enum
import uuid

from .clinical_event_envelope import ClinicalEventEnvelope, EventStatus

logger = logging.getLogger(__name__)


class IdempotencyStrategy(Enum):
    """Idempotency strategies for different event types"""
    CONTENT_HASH = "content_hash"
    TEMPORAL_WINDOW = "temporal_window"
    SEMANTIC_DEDUPLICATION = "semantic_deduplication"
    BUSINESS_KEY = "business_key"
    COMPOSITE_KEY = "composite_key"


class EventStreamStatus(Enum):
    """Status of event streams"""
    ACTIVE = "active"
    ARCHIVED = "archived"
    CORRUPTED = "corrupted"
    REBUILDING = "rebuilding"


@dataclass
class IdempotencyKey:
    """Idempotency key for event deduplication"""
    key_value: str
    strategy: IdempotencyStrategy
    created_at: datetime
    expires_at: Optional[datetime] = None
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class EventSnapshot:
    """Point-in-time snapshot of event stream state"""
    snapshot_id: str
    stream_id: str
    snapshot_version: int
    snapshot_timestamp: datetime
    aggregate_state: Dict[str, Any]
    last_event_id: str
    last_event_timestamp: datetime
    event_count: int
    checksum: str
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class EventStream:
    """Event stream for maintaining event history"""
    stream_id: str
    stream_type: str
    created_at: datetime
    last_updated: datetime
    version: int
    status: EventStreamStatus
    events: List[ClinicalEventEnvelope] = field(default_factory=list)
    snapshots: List[EventSnapshot] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)


class IdempotencyManager:
    """
    Advanced idempotency manager for clinical events
    
    Provides sophisticated deduplication strategies to ensure
    clinical events are processed exactly once, even in the
    presence of retries, network failures, or system restarts.
    """
    
    def __init__(self, default_ttl_hours: int = 24):
        self.default_ttl_hours = default_ttl_hours
        
        # Idempotency key storage
        self.idempotency_keys: Dict[str, IdempotencyKey] = {}
        self.processed_events: Dict[str, ClinicalEventEnvelope] = {}
        
        # Strategy-specific configurations
        self.temporal_window_seconds = {
            "medication_order": 300,    # 5 minutes
            "laboratory_result": 60,    # 1 minute
            "clinical_decision": 600,   # 10 minutes
            "adverse_event": 1800       # 30 minutes
        }
        
        # Semantic similarity thresholds
        self.similarity_thresholds = {
            "medication_order": 0.95,
            "laboratory_result": 0.98,
            "clinical_decision": 0.90,
            "adverse_event": 0.85
        }
        
        logger.info("Idempotency Manager initialized")
    
    async def check_idempotency(self, envelope: ClinicalEventEnvelope) -> Tuple[bool, Optional[str]]:
        """
        Check if event is idempotent (already processed)
        
        Args:
            envelope: Clinical event envelope to check
            
        Returns:
            Tuple of (is_duplicate, original_event_id)
        """
        try:
            # Clean expired keys first
            await self._clean_expired_keys()
            
            # Determine idempotency strategy
            strategy = self._determine_strategy(envelope)
            
            # Generate idempotency key
            idempotency_key = await self._generate_idempotency_key(envelope, strategy)
            
            # Check if key exists
            if idempotency_key in self.idempotency_keys:
                existing_key = self.idempotency_keys[idempotency_key]
                
                # Check if key is still valid
                if not self._is_key_expired(existing_key):
                    # Find original event
                    original_event_id = existing_key.metadata.get("original_event_id")
                    logger.info(f"Duplicate event detected: {envelope.metadata.event_id} "
                               f"(original: {original_event_id})")
                    return True, original_event_id
                else:
                    # Remove expired key
                    del self.idempotency_keys[idempotency_key]
            
            # Store new idempotency key
            await self._store_idempotency_key(envelope, idempotency_key, strategy)
            
            return False, None
            
        except Exception as e:
            logger.error(f"Error checking idempotency: {e}")
            # In case of error, allow processing to continue
            return False, None
    
    async def mark_event_processed(self, envelope: ClinicalEventEnvelope):
        """Mark event as successfully processed"""
        try:
            self.processed_events[envelope.metadata.event_id] = envelope
            
            # Update idempotency key metadata
            strategy = self._determine_strategy(envelope)
            idempotency_key = await self._generate_idempotency_key(envelope, strategy)
            
            if idempotency_key in self.idempotency_keys:
                self.idempotency_keys[idempotency_key].metadata["processed_at"] = datetime.now(timezone.utc)
                self.idempotency_keys[idempotency_key].metadata["processing_status"] = "completed"
            
            logger.debug(f"Marked event as processed: {envelope.metadata.event_id}")
            
        except Exception as e:
            logger.error(f"Error marking event as processed: {e}")
    
    def _determine_strategy(self, envelope: ClinicalEventEnvelope) -> IdempotencyStrategy:
        """Determine appropriate idempotency strategy for event"""
        event_type = envelope.metadata.event_type.value
        
        # Strategy mapping based on event type
        strategy_map = {
            "medication_order": IdempotencyStrategy.COMPOSITE_KEY,
            "laboratory_result": IdempotencyStrategy.BUSINESS_KEY,
            "clinical_decision": IdempotencyStrategy.SEMANTIC_DEDUPLICATION,
            "adverse_event": IdempotencyStrategy.TEMPORAL_WINDOW,
            "patient_encounter": IdempotencyStrategy.BUSINESS_KEY
        }
        
        return strategy_map.get(event_type, IdempotencyStrategy.CONTENT_HASH)
    
    async def _generate_idempotency_key(self, envelope: ClinicalEventEnvelope, 
                                      strategy: IdempotencyStrategy) -> str:
        """Generate idempotency key based on strategy"""
        if strategy == IdempotencyStrategy.CONTENT_HASH:
            return await self._generate_content_hash_key(envelope)
        elif strategy == IdempotencyStrategy.TEMPORAL_WINDOW:
            return await self._generate_temporal_window_key(envelope)
        elif strategy == IdempotencyStrategy.SEMANTIC_DEDUPLICATION:
            return await self._generate_semantic_key(envelope)
        elif strategy == IdempotencyStrategy.BUSINESS_KEY:
            return await self._generate_business_key(envelope)
        elif strategy == IdempotencyStrategy.COMPOSITE_KEY:
            return await self._generate_composite_key(envelope)
        else:
            return await self._generate_content_hash_key(envelope)
    
    async def _generate_content_hash_key(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate content-based hash key"""
        # Create hash from event data and clinical context
        content = {
            "event_data": envelope.event_data,
            "patient_id": envelope.clinical_context.patient_id,
            "event_type": envelope.metadata.event_type.value
        }
        
        content_str = json.dumps(content, sort_keys=True, default=str)
        return hashlib.sha256(content_str.encode()).hexdigest()
    
    async def _generate_temporal_window_key(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate temporal window-based key"""
        event_type = envelope.metadata.event_type.value
        window_seconds = self.temporal_window_seconds.get(event_type, 300)
        
        # Round timestamp to window boundary
        event_time = envelope.temporal_context.event_time
        window_start = event_time.replace(second=0, microsecond=0)
        window_minutes = (window_start.minute // (window_seconds // 60)) * (window_seconds // 60)
        window_start = window_start.replace(minute=window_minutes)
        
        # Create key from patient, event type, and time window
        key_data = {
            "patient_id": envelope.clinical_context.patient_id,
            "event_type": event_type,
            "window_start": window_start.isoformat()
        }
        
        key_str = json.dumps(key_data, sort_keys=True)
        return hashlib.sha256(key_str.encode()).hexdigest()
    
    async def _generate_semantic_key(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate semantic similarity-based key"""
        # Extract semantic features for comparison
        semantic_features = {
            "patient_id": envelope.clinical_context.patient_id,
            "event_type": envelope.metadata.event_type.value,
            "primary_content": self._extract_primary_content(envelope),
            "clinical_context_hash": self._hash_clinical_context(envelope)
        }
        
        key_str = json.dumps(semantic_features, sort_keys=True, default=str)
        return hashlib.sha256(key_str.encode()).hexdigest()
    
    async def _generate_business_key(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate business logic-based key"""
        event_type = envelope.metadata.event_type.value
        
        if event_type == "laboratory_result":
            # Use test name, patient, and collection time
            key_data = {
                "patient_id": envelope.clinical_context.patient_id,
                "test_name": envelope.event_data.get("test_name", ""),
                "collection_time": envelope.event_data.get("collection_time", ""),
                "specimen_id": envelope.event_data.get("specimen_id", "")
            }
        elif event_type == "patient_encounter":
            # Use patient, encounter type, and admission time
            key_data = {
                "patient_id": envelope.clinical_context.patient_id,
                "encounter_type": envelope.clinical_context.encounter_type,
                "admission_date": envelope.clinical_context.admission_date
            }
        else:
            # Fallback to content hash
            return await self._generate_content_hash_key(envelope)
        
        key_str = json.dumps(key_data, sort_keys=True, default=str)
        return hashlib.sha256(key_str.encode()).hexdigest()
    
    async def _generate_composite_key(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate composite key combining multiple strategies"""
        # Combine content hash and temporal window
        content_key = await self._generate_content_hash_key(envelope)
        temporal_key = await self._generate_temporal_window_key(envelope)
        
        composite_data = {
            "content_hash": content_key[:16],  # First 16 chars
            "temporal_hash": temporal_key[:16],  # First 16 chars
            "patient_id": envelope.clinical_context.patient_id
        }
        
        key_str = json.dumps(composite_data, sort_keys=True)
        return hashlib.sha256(key_str.encode()).hexdigest()
    
    def _extract_primary_content(self, envelope: ClinicalEventEnvelope) -> str:
        """Extract primary content for semantic comparison"""
        event_data = envelope.event_data
        
        # Extract key fields based on event type
        event_type = envelope.metadata.event_type.value
        
        if event_type == "medication_order":
            return f"{event_data.get('medication_name', '')}_{event_data.get('dosage', '')}"
        elif event_type == "laboratory_result":
            return f"{event_data.get('test_name', '')}_{event_data.get('result_value', '')}"
        elif event_type == "clinical_decision":
            return f"{event_data.get('decision_type', '')}_{event_data.get('recommendation', '')}"
        else:
            return str(event_data)
    
    def _hash_clinical_context(self, envelope: ClinicalEventEnvelope) -> str:
        """Create hash of relevant clinical context"""
        context_data = {
            "encounter_id": envelope.clinical_context.encounter_id,
            "primary_provider_id": envelope.clinical_context.primary_provider_id,
            "facility_id": envelope.clinical_context.facility_id
        }
        
        context_str = json.dumps(context_data, sort_keys=True, default=str)
        return hashlib.sha256(context_str.encode()).hexdigest()[:16]
    
    async def _store_idempotency_key(self, envelope: ClinicalEventEnvelope, 
                                   key: str, strategy: IdempotencyStrategy):
        """Store idempotency key with metadata"""
        expires_at = datetime.now(timezone.utc) + timedelta(hours=self.default_ttl_hours)
        
        idempotency_key = IdempotencyKey(
            key_value=key,
            strategy=strategy,
            created_at=datetime.now(timezone.utc),
            expires_at=expires_at,
            metadata={
                "original_event_id": envelope.metadata.event_id,
                "event_type": envelope.metadata.event_type.value,
                "patient_id": envelope.clinical_context.patient_id,
                "processing_status": "pending"
            }
        )
        
        self.idempotency_keys[key] = idempotency_key
    
    def _is_key_expired(self, key: IdempotencyKey) -> bool:
        """Check if idempotency key has expired"""
        if key.expires_at is None:
            return False
        
        return datetime.now(timezone.utc) > key.expires_at
    
    async def _clean_expired_keys(self):
        """Clean up expired idempotency keys"""
        current_time = datetime.now(timezone.utc)
        expired_keys = []
        
        for key_value, key_obj in self.idempotency_keys.items():
            if self._is_key_expired(key_obj):
                expired_keys.append(key_value)
        
        for key_value in expired_keys:
            del self.idempotency_keys[key_value]
        
        if expired_keys:
            logger.debug(f"Cleaned up {len(expired_keys)} expired idempotency keys")
    
    def get_idempotency_stats(self) -> Dict[str, Any]:
        """Get idempotency manager statistics"""
        strategy_counts = {}
        for key in self.idempotency_keys.values():
            strategy = key.strategy.value
            strategy_counts[strategy] = strategy_counts.get(strategy, 0) + 1
        
        return {
            "total_keys": len(self.idempotency_keys),
            "processed_events": len(self.processed_events),
            "strategy_distribution": strategy_counts,
            "default_ttl_hours": self.default_ttl_hours
        }


class EventStore:
    """
    Event store for clinical event sourcing

    Provides complete audit trail and event history for clinical
    events with snapshot capabilities for performance optimization.
    """

    def __init__(self, snapshot_frequency: int = 100):
        self.snapshot_frequency = snapshot_frequency

        # Event storage
        self.event_streams: Dict[str, EventStream] = {}
        self.event_index: Dict[str, str] = {}  # event_id -> stream_id

        # Performance optimization
        self.stream_cache: Dict[str, EventStream] = {}
        self.cache_size_limit = 1000

        logger.info("Event Store initialized")

    async def append_event(self, envelope: ClinicalEventEnvelope,
                          stream_id: Optional[str] = None) -> str:
        """
        Append event to event stream

        Args:
            envelope: Clinical event envelope to append
            stream_id: Optional stream ID (auto-generated if not provided)

        Returns:
            Stream ID where event was appended
        """
        try:
            # Generate stream ID if not provided
            if not stream_id:
                stream_id = self._generate_stream_id(envelope)

            # Get or create event stream
            stream = await self._get_or_create_stream(stream_id, envelope)

            # Append event to stream
            stream.events.append(envelope)
            stream.last_updated = datetime.now(timezone.utc)
            stream.version += 1

            # Update event index
            self.event_index[envelope.metadata.event_id] = stream_id

            # Check if snapshot is needed
            if len(stream.events) % self.snapshot_frequency == 0:
                await self._create_snapshot(stream)

            # Update cache
            self.stream_cache[stream_id] = stream
            await self._manage_cache_size()

            logger.debug(f"Appended event {envelope.metadata.event_id} to stream {stream_id}")
            return stream_id

        except Exception as e:
            logger.error(f"Error appending event to store: {e}")
            raise

    async def get_event_stream(self, stream_id: str) -> Optional[EventStream]:
        """Get event stream by ID"""
        try:
            # Check cache first
            if stream_id in self.stream_cache:
                return self.stream_cache[stream_id]

            # Get from storage
            if stream_id in self.event_streams:
                stream = self.event_streams[stream_id]
                self.stream_cache[stream_id] = stream
                return stream

            return None

        except Exception as e:
            logger.error(f"Error getting event stream {stream_id}: {e}")
            return None

    async def get_events_by_patient(self, patient_id: str,
                                  event_types: Optional[List[str]] = None,
                                  start_time: Optional[datetime] = None,
                                  end_time: Optional[datetime] = None) -> List[ClinicalEventEnvelope]:
        """Get events for a specific patient with optional filtering"""
        try:
            events = []

            for stream in self.event_streams.values():
                for event in stream.events:
                    # Filter by patient ID
                    if event.clinical_context.patient_id != patient_id:
                        continue

                    # Filter by event types
                    if event_types and event.metadata.event_type.value not in event_types:
                        continue

                    # Filter by time range
                    if start_time and event.temporal_context.event_time < start_time:
                        continue

                    if end_time and event.temporal_context.event_time > end_time:
                        continue

                    events.append(event)

            # Sort by event time
            events.sort(key=lambda e: e.temporal_context.event_time)

            logger.debug(f"Retrieved {len(events)} events for patient {patient_id}")
            return events

        except Exception as e:
            logger.error(f"Error getting events for patient {patient_id}: {e}")
            return []

    async def get_event_by_id(self, event_id: str) -> Optional[ClinicalEventEnvelope]:
        """Get specific event by ID"""
        try:
            # Find stream containing the event
            stream_id = self.event_index.get(event_id)
            if not stream_id:
                return None

            # Get stream and find event
            stream = await self.get_event_stream(stream_id)
            if not stream:
                return None

            for event in stream.events:
                if event.metadata.event_id == event_id:
                    return event

            return None

        except Exception as e:
            logger.error(f"Error getting event {event_id}: {e}")
            return None

    async def create_snapshot(self, stream_id: str) -> Optional[EventSnapshot]:
        """Create snapshot for event stream"""
        try:
            stream = await self.get_event_stream(stream_id)
            if not stream:
                return None

            return await self._create_snapshot(stream)

        except Exception as e:
            logger.error(f"Error creating snapshot for stream {stream_id}: {e}")
            return None

    async def restore_from_snapshot(self, stream_id: str,
                                  snapshot_id: Optional[str] = None) -> Optional[EventStream]:
        """Restore event stream from snapshot"""
        try:
            stream = await self.get_event_stream(stream_id)
            if not stream:
                return None

            # Find snapshot (latest if not specified)
            target_snapshot = None
            if snapshot_id:
                target_snapshot = next((s for s in stream.snapshots if s.snapshot_id == snapshot_id), None)
            else:
                # Get latest snapshot
                if stream.snapshots:
                    target_snapshot = max(stream.snapshots, key=lambda s: s.snapshot_timestamp)

            if not target_snapshot:
                return stream  # No snapshot available, return full stream

            # Create restored stream
            restored_stream = EventStream(
                stream_id=stream.stream_id,
                stream_type=stream.stream_type,
                created_at=stream.created_at,
                last_updated=target_snapshot.snapshot_timestamp,
                version=target_snapshot.snapshot_version,
                status=stream.status,
                events=[],  # Events will be loaded from snapshot
                snapshots=[target_snapshot],
                metadata=stream.metadata.copy()
            )

            # Load events after snapshot
            snapshot_event_id = target_snapshot.last_event_id
            found_snapshot_event = False

            for event in stream.events:
                if event.metadata.event_id == snapshot_event_id:
                    found_snapshot_event = True
                    continue

                if found_snapshot_event:
                    restored_stream.events.append(event)

            logger.info(f"Restored stream {stream_id} from snapshot {target_snapshot.snapshot_id}")
            return restored_stream

        except Exception as e:
            logger.error(f"Error restoring stream {stream_id} from snapshot: {e}")
            return None

    def _generate_stream_id(self, envelope: ClinicalEventEnvelope) -> str:
        """Generate stream ID for event"""
        # Use patient ID and event type as basis for stream ID
        patient_id = envelope.clinical_context.patient_id
        event_type = envelope.metadata.event_type.value

        # For some event types, use encounter-based streams
        if envelope.clinical_context.encounter_id and event_type in ["medication_order", "laboratory_result"]:
            return f"encounter_{envelope.clinical_context.encounter_id}_{event_type}"
        else:
            return f"patient_{patient_id}_{event_type}"

    async def _get_or_create_stream(self, stream_id: str,
                                  envelope: ClinicalEventEnvelope) -> EventStream:
        """Get existing stream or create new one"""
        stream = await self.get_event_stream(stream_id)

        if not stream:
            # Create new stream
            stream = EventStream(
                stream_id=stream_id,
                stream_type=envelope.metadata.event_type.value,
                created_at=datetime.now(timezone.utc),
                last_updated=datetime.now(timezone.utc),
                version=0,
                status=EventStreamStatus.ACTIVE,
                events=[],
                snapshots=[],
                metadata={
                    "patient_id": envelope.clinical_context.patient_id,
                    "encounter_id": envelope.clinical_context.encounter_id
                }
            )

            self.event_streams[stream_id] = stream
            logger.info(f"Created new event stream: {stream_id}")

        return stream

    async def _create_snapshot(self, stream: EventStream) -> EventSnapshot:
        """Create snapshot of current stream state"""
        try:
            # Calculate aggregate state (simplified)
            aggregate_state = {
                "event_count": len(stream.events),
                "last_event_timestamp": stream.events[-1].temporal_context.event_time.isoformat() if stream.events else None,
                "patient_id": stream.metadata.get("patient_id"),
                "encounter_id": stream.metadata.get("encounter_id")
            }

            # Calculate checksum
            checksum_data = {
                "stream_id": stream.stream_id,
                "version": stream.version,
                "event_count": len(stream.events),
                "last_event_id": stream.events[-1].metadata.event_id if stream.events else None
            }
            checksum_str = json.dumps(checksum_data, sort_keys=True)
            checksum = hashlib.sha256(checksum_str.encode()).hexdigest()

            # Create snapshot
            snapshot = EventSnapshot(
                snapshot_id=str(uuid.uuid4()),
                stream_id=stream.stream_id,
                snapshot_version=stream.version,
                snapshot_timestamp=datetime.now(timezone.utc),
                aggregate_state=aggregate_state,
                last_event_id=stream.events[-1].metadata.event_id if stream.events else "",
                last_event_timestamp=stream.events[-1].temporal_context.event_time if stream.events else datetime.now(timezone.utc),
                event_count=len(stream.events),
                checksum=checksum,
                metadata={"created_by": "event_store"}
            )

            # Add to stream
            stream.snapshots.append(snapshot)

            # Keep only recent snapshots (last 10)
            if len(stream.snapshots) > 10:
                stream.snapshots = stream.snapshots[-10:]

            logger.info(f"Created snapshot {snapshot.snapshot_id} for stream {stream.stream_id}")
            return snapshot

        except Exception as e:
            logger.error(f"Error creating snapshot: {e}")
            raise

    async def _manage_cache_size(self):
        """Manage cache size to prevent memory issues"""
        if len(self.stream_cache) > self.cache_size_limit:
            # Remove oldest entries (simple LRU)
            # In production, use proper LRU implementation
            oldest_streams = list(self.stream_cache.keys())[:len(self.stream_cache) - self.cache_size_limit + 100]

            for stream_id in oldest_streams:
                del self.stream_cache[stream_id]

            logger.debug(f"Cleaned cache, removed {len(oldest_streams)} streams")

    def get_store_stats(self) -> Dict[str, Any]:
        """Get event store statistics"""
        total_events = sum(len(stream.events) for stream in self.event_streams.values())
        total_snapshots = sum(len(stream.snapshots) for stream in self.event_streams.values())

        stream_types = {}
        for stream in self.event_streams.values():
            stream_type = stream.stream_type
            stream_types[stream_type] = stream_types.get(stream_type, 0) + 1

        return {
            "total_streams": len(self.event_streams),
            "total_events": total_events,
            "total_snapshots": total_snapshots,
            "cached_streams": len(self.stream_cache),
            "stream_types": stream_types,
            "snapshot_frequency": self.snapshot_frequency
        }
