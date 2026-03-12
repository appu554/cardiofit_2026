"""
Data models for Neo4j Graph Projector Service
"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel


class HealthResponse(BaseModel):
    """Health check response model"""
    status: str
    timestamp: datetime


class MetricsResponse(BaseModel):
    """Metrics response model"""
    messages_consumed: int
    messages_processed: int
    messages_failed: int
    batches_processed: int
    consumer_lag: int
    uptime_seconds: float


class StatusResponse(BaseModel):
    """Status response model"""
    status: str
    kafka_connected: bool
    neo4j_connected: bool
    consumer_group: str
    topics: list[str]
    batch_size: int
    batch_timeout_seconds: float
    metrics: MetricsResponse
    last_processed: Optional[datetime]


class ErrorResponse(BaseModel):
    """Error response model"""
    error: str
    detail: Optional[str] = None


__all__ = [
    "HealthResponse",
    "MetricsResponse",
    "StatusResponse",
    "ErrorResponse",
]
