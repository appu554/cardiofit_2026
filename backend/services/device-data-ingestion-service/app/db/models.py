"""
Database models for Transactional Outbox Pattern
Per-vendor outbox tables for true fault isolation
"""
from datetime import datetime
from typing import Optional, Dict, Any
from sqlalchemy import Column, String, Integer, Text, Boolean, DateTime, JSON
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.sql import func
import uuid

from .database import Base


class VendorOutboxRegistry(Base):
    """Vendor registry for dynamic outbox table management"""
    __tablename__ = "vendor_outbox_registry"
    
    vendor_id = Column(String(100), primary_key=True)
    vendor_name = Column(String(255), nullable=False)
    outbox_table_name = Column(String(255), nullable=False, unique=True)
    dead_letter_table_name = Column(String(255), nullable=False, unique=True)
    kafka_topic = Column(String(255), nullable=False, default="raw-device-data.v1")
    max_retries = Column(Integer, default=3)
    retry_backoff_seconds = Column(Integer, default=60)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


# Note: The actual outbox tables (fitbit_outbox, garmin_outbox, apple_health_outbox)
# are created via SQL migration for better performance and to avoid SQLAlchemy overhead
# for high-throughput operations. We use raw SQL queries for outbox operations.

class OutboxMessage:
    """
    Pydantic-like model for outbox messages (not a SQLAlchemy model)
    Used for type safety and validation in the outbox service
    """
    
    def __init__(
        self,
        id: Optional[str] = None,
        device_id: str = None,
        event_type: str = "device_reading",
        event_payload: Dict[str, Any] = None,
        kafka_topic: str = "raw-device-data.v1",
        kafka_key: Optional[str] = None,
        created_at: Optional[datetime] = None,
        processed_at: Optional[datetime] = None,
        retry_count: int = 0,
        max_retries: int = 3,
        last_error: Optional[str] = None,
        status: str = "pending",
        correlation_id: Optional[str] = None,
        trace_id: Optional[str] = None,
        vendor_id: Optional[str] = None,
        outbox_table: Optional[str] = None
    ):
        self.id = id or str(uuid.uuid4())
        self.device_id = device_id
        self.event_type = event_type
        self.event_payload = event_payload or {}
        self.kafka_topic = kafka_topic
        self.kafka_key = kafka_key or device_id
        self.created_at = created_at
        self.processed_at = processed_at
        self.retry_count = retry_count
        self.max_retries = max_retries
        self.last_error = last_error
        self.status = status
        self.correlation_id = correlation_id
        self.trace_id = trace_id
        self.vendor_id = vendor_id
        self.outbox_table = outbox_table
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization"""
        return {
            "id": self.id,
            "device_id": self.device_id,
            "event_type": self.event_type,
            "event_payload": self.event_payload,
            "kafka_topic": self.kafka_topic,
            "kafka_key": self.kafka_key,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "processed_at": self.processed_at.isoformat() if self.processed_at else None,
            "retry_count": self.retry_count,
            "max_retries": self.max_retries,
            "last_error": self.last_error,
            "status": self.status,
            "correlation_id": self.correlation_id,
            "trace_id": self.trace_id,
            "vendor_id": self.vendor_id,
            "outbox_table": self.outbox_table
        }
    
    @classmethod
    def from_db_row(cls, row, vendor_id: str = None, outbox_table: str = None):
        """Create OutboxMessage from database row"""
        return cls(
            id=str(row.id) if hasattr(row, 'id') else None,
            device_id=row.device_id if hasattr(row, 'device_id') else None,
            event_type=row.event_type if hasattr(row, 'event_type') else "device_reading",
            event_payload=row.event_payload if hasattr(row, 'event_payload') else {},
            kafka_topic=row.kafka_topic if hasattr(row, 'kafka_topic') else "raw-device-data.v1",
            kafka_key=row.kafka_key if hasattr(row, 'kafka_key') else None,
            created_at=row.created_at if hasattr(row, 'created_at') else None,
            processed_at=row.processed_at if hasattr(row, 'processed_at') else None,
            retry_count=row.retry_count if hasattr(row, 'retry_count') else 0,
            max_retries=row.max_retries if hasattr(row, 'max_retries') else 3,
            last_error=row.last_error if hasattr(row, 'last_error') else None,
            status=row.status if hasattr(row, 'status') else "pending",
            correlation_id=str(row.correlation_id) if hasattr(row, 'correlation_id') and row.correlation_id else None,
            trace_id=row.trace_id if hasattr(row, 'trace_id') else None,
            vendor_id=vendor_id,
            outbox_table=outbox_table
        )
    
    def can_retry(self) -> bool:
        """Check if message can be retried"""
        return self.retry_count < self.max_retries
    
    def increment_retry(self):
        """Increment retry count"""
        self.retry_count += 1
    
    def is_failed(self) -> bool:
        """Check if message has failed permanently"""
        return self.status == "failed" or self.retry_count >= self.max_retries


class DeadLetterMessage:
    """
    Model for dead letter messages (not a SQLAlchemy model)
    Used for type safety and validation in dead letter handling
    """
    
    def __init__(
        self,
        id: str,
        device_id: str,
        event_type: str,
        event_payload: Dict[str, Any],
        kafka_topic: str,
        kafka_key: Optional[str],
        original_created_at: datetime,
        failed_at: Optional[datetime] = None,
        final_error: str = None,
        retry_count: int = 0,
        correlation_id: Optional[str] = None,
        trace_id: Optional[str] = None,
        failure_reason: str = "max_retries_exceeded"
    ):
        self.id = id
        self.device_id = device_id
        self.event_type = event_type
        self.event_payload = event_payload
        self.kafka_topic = kafka_topic
        self.kafka_key = kafka_key
        self.original_created_at = original_created_at
        self.failed_at = failed_at or datetime.utcnow()
        self.final_error = final_error
        self.retry_count = retry_count
        self.correlation_id = correlation_id
        self.trace_id = trace_id
        self.failure_reason = failure_reason
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization"""
        return {
            "id": self.id,
            "device_id": self.device_id,
            "event_type": self.event_type,
            "event_payload": self.event_payload,
            "kafka_topic": self.kafka_topic,
            "kafka_key": self.kafka_key,
            "original_created_at": self.original_created_at.isoformat() if self.original_created_at else None,
            "failed_at": self.failed_at.isoformat() if self.failed_at else None,
            "final_error": self.final_error,
            "retry_count": self.retry_count,
            "correlation_id": self.correlation_id,
            "trace_id": self.trace_id,
            "failure_reason": self.failure_reason
        }
    
    @classmethod
    def from_outbox_message(cls, outbox_msg: OutboxMessage, final_error: str):
        """Create DeadLetterMessage from failed OutboxMessage"""
        return cls(
            id=outbox_msg.id,
            device_id=outbox_msg.device_id,
            event_type=outbox_msg.event_type,
            event_payload=outbox_msg.event_payload,
            kafka_topic=outbox_msg.kafka_topic,
            kafka_key=outbox_msg.kafka_key,
            original_created_at=outbox_msg.created_at,
            final_error=final_error,
            retry_count=outbox_msg.retry_count,
            correlation_id=outbox_msg.correlation_id,
            trace_id=outbox_msg.trace_id
        )


# Enhanced vendor configuration constants for all medical devices
SUPPORTED_VENDORS = {
    # Consumer Fitness Devices
    "fitbit": {
        "outbox_table": "fitbit_outbox",
        "dead_letter_table": "fitbit_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "sleep_duration", "weight"]
    },
    "garmin": {
        "outbox_table": "garmin_outbox",
        "dead_letter_table": "garmin_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "sleep_duration", "weight", "oxygen_saturation"]
    },
    "apple_health": {
        "outbox_table": "apple_health_outbox",
        "dead_letter_table": "apple_health_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "sleep_duration", "weight", "ecg"]
    },
    "samsung_health": {
        "outbox_table": "samsung_health_outbox",
        "dead_letter_table": "samsung_health_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "sleep_duration", "weight", "oxygen_saturation"]
    },
    "polar": {
        "outbox_table": "polar_outbox",
        "dead_letter_table": "polar_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "sleep_duration"]
    },
    "suunto": {
        "outbox_table": "suunto_outbox",
        "dead_letter_table": "suunto_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "sleep_duration"]
    },

    # Medical Grade Devices
    "withings": {
        "outbox_table": "withings_outbox",
        "dead_letter_table": "withings_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["weight", "blood_pressure", "temperature", "heart_rate"]
    },
    "omron": {
        "outbox_table": "omron_outbox",
        "dead_letter_table": "omron_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["blood_pressure", "heart_rate", "weight"]
    },

    # Clinical/Hospital Devices
    "medical_device": {
        "outbox_table": "medical_device_outbox",
        "dead_letter_table": "medical_device_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["ecg", "blood_pressure", "blood_glucose", "temperature", "oxygen_saturation", "heart_rate"]
    },

    # Fallback for Unknown Devices
    "generic_device": {
        "outbox_table": "generic_device_outbox",
        "dead_letter_table": "generic_device_dead_letter",
        "kafka_topic": "raw-device-data.v1",
        "device_types": ["heart_rate", "steps", "weight", "temperature", "blood_pressure", "blood_glucose", "ecg", "oxygen_saturation", "sleep_duration"]
    }
}


def get_vendor_config(vendor_id: str) -> Optional[Dict[str, str]]:
    """Get vendor configuration by vendor ID"""
    return SUPPORTED_VENDORS.get(vendor_id.lower())


def is_supported_vendor(vendor_id: str) -> bool:
    """Check if vendor is supported"""
    return vendor_id.lower() in SUPPORTED_VENDORS
