"""
Response schemas for FastAPI endpoints
"""
from pydantic import BaseModel, Field
from typing import Optional, Dict, Any
from datetime import datetime


class HealthResponse(BaseModel):
    """Health check response"""
    status: str = Field(..., description="Service status (healthy/unhealthy)")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    service: str = "postgresql-projector"
    version: str = "1.0.0"


class MetricsResponse(BaseModel):
    """Metrics response"""
    messages_consumed: int = Field(..., description="Total messages consumed from Kafka")
    messages_processed: int = Field(..., description="Total messages successfully processed")
    messages_failed: int = Field(..., description="Total messages failed")
    batches_processed: int = Field(..., description="Total batches processed")
    consumer_lag: int = Field(..., description="Current consumer lag")
    uptime_seconds: float = Field(..., description="Service uptime in seconds")


class StatusResponse(BaseModel):
    """Service status response"""
    service: str = "postgresql-projector"
    status: str = Field(..., description="running/stopped/error")
    kafka_connected: bool = Field(..., description="Kafka connection status")
    postgres_connected: bool = Field(..., description="PostgreSQL connection status")
    consumer_group: str = Field(..., description="Kafka consumer group ID")
    topics: list[str] = Field(..., description="Subscribed topics")
    batch_size: int = Field(..., description="Current batch size")
    batch_timeout_seconds: float = Field(..., description="Batch timeout in seconds")
    metrics: MetricsResponse = Field(..., description="Current metrics")
    last_processed: Optional[datetime] = Field(None, description="Timestamp of last processed batch")


class ErrorResponse(BaseModel):
    """Error response"""
    error: str = Field(..., description="Error message")
    detail: Optional[str] = Field(None, description="Detailed error information")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
