"""Pydantic schemas for MongoDB documents."""

from typing import Dict, List, Any, Optional
from datetime import datetime
from pydantic import BaseModel, Field


class PredictionDetail(BaseModel):
    """ML prediction detail."""

    model_name: str
    prediction: float
    confidence: float
    threshold: float
    alert_triggered: bool
    shap_values: Optional[Dict[str, float]] = None
    lime_explanation: Optional[Dict[str, Any]] = None


class MLExplanation(BaseModel):
    """ML explanation document for MongoDB."""

    patient_id: str = Field(..., alias="patientId")
    event_id: str = Field(..., alias="eventId")
    timestamp: datetime
    predictions: Dict[str, PredictionDetail]
    feature_importance: Optional[Dict[str, float]] = None
    created_at: datetime = Field(default_factory=datetime.utcnow)

    class Config:
        populate_by_name = True


class TimelineEvent(BaseModel):
    """Individual event in patient timeline."""

    event_id: str = Field(..., alias="eventId")
    timestamp: datetime
    event_type: str = Field(..., alias="eventType")
    summary: str
    risk_level: Optional[str] = Field(None, alias="riskLevel")
    vital_signs: Optional[Dict[str, Any]] = Field(None, alias="vitalSigns")
    predictions: Optional[Dict[str, float]] = None

    class Config:
        populate_by_name = True


class PatientTimeline(BaseModel):
    """Patient timeline aggregation document."""

    patient_id: str = Field(..., alias="_id")
    events: List[TimelineEvent] = Field(default_factory=list)
    last_updated: datetime = Field(default_factory=datetime.utcnow, alias="lastUpdated")
    event_count: int = Field(default=0, alias="eventCount")
    first_event_time: Optional[datetime] = Field(None, alias="firstEventTime")
    latest_event_time: Optional[datetime] = Field(None, alias="latestEventTime")

    class Config:
        populate_by_name = True


class ClinicalDocument(BaseModel):
    """Full clinical event document for MongoDB."""

    event_id: str = Field(..., alias="_id")
    patient_id: str = Field(..., alias="patientId")
    timestamp: datetime
    event_type: str = Field(..., alias="eventType")
    device_type: Optional[str] = Field(None, alias="deviceType")

    # Original data
    vital_signs: Optional[Dict[str, Any]] = Field(None, alias="vitalSigns")
    lab_results: Optional[Dict[str, Any]] = Field(None, alias="labResults")

    # Enrichments
    enrichments: Optional[Dict[str, Any]] = None
    ml_predictions: Optional[Dict[str, Any]] = Field(None, alias="mlPredictions")

    # Metadata
    ingestion_time: Optional[datetime] = Field(None, alias="ingestionTime")
    processing_time: Optional[datetime] = Field(None, alias="processingTime")
    created_at: datetime = Field(default_factory=datetime.utcnow, alias="createdAt")

    # Human-readable summary
    summary: Optional[str] = None

    class Config:
        populate_by_name = True
