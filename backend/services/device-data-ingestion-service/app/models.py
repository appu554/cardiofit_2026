"""
Data models for Device Data Ingestion Service
"""
from datetime import datetime
from typing import Optional, Dict, Any
from pydantic import BaseModel, Field, validator
import uuid


class DeviceReading(BaseModel):
    """Model for incoming device data"""
    
    device_id: str = Field(..., description="Unique device identifier")
    timestamp: int = Field(..., description="Unix timestamp of the reading")
    reading_type: str = Field(..., description="Type of reading (e.g., heart_rate, blood_pressure)")
    value: float = Field(..., description="Numeric value of the reading")
    unit: str = Field(..., description="Unit of measurement")
    patient_id: Optional[str] = Field(None, description="Associated patient ID if known")
    metadata: Optional[Dict[str, Any]] = Field(default_factory=dict, description="Additional metadata")
    
    @validator('device_id')
    def validate_device_id(cls, v):
        if not v or len(v.strip()) == 0:
            raise ValueError('Device ID cannot be empty')
        return v.strip()
    
    @validator('reading_type')
    def validate_reading_type(cls, v):
        allowed_types = [
            'heart_rate', 'blood_pressure', 'blood_pressure_systolic', 'blood_pressure_diastolic',
            'blood_glucose', 'temperature', 'oxygen_saturation', 'weight',
            'steps', 'sleep_duration', 'respiratory_rate'
        ]
        if v not in allowed_types:
            raise ValueError(f'Reading type must be one of: {", ".join(allowed_types)}')
        return v
    
    @validator('value')
    def validate_value(cls, v):
        if v is None:
            raise ValueError('Value cannot be null')
        return v
    
    @validator('timestamp')
    def validate_timestamp(cls, v):
        if v <= 0:
            raise ValueError('Timestamp must be positive')
        # Check if timestamp is reasonable (not too far in past or future)
        current_time = datetime.now().timestamp()
        if v > current_time + 3600:  # 1 hour in future
            raise ValueError('Timestamp cannot be more than 1 hour in the future')
        if v < current_time - (365 * 24 * 3600):  # 1 year in past
            raise ValueError('Timestamp cannot be more than 1 year in the past')
        return v


class IngestionResponse(BaseModel):
    """Response model for successful ingestion"""
    
    status: str = "accepted"
    message: str = "Data queued for processing"
    ingestion_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    timestamp: datetime = Field(default_factory=datetime.utcnow)


class ErrorResponse(BaseModel):
    """Response model for errors"""
    
    status: str = "error"
    message: str
    error_code: Optional[str] = None
    timestamp: str = Field(default_factory=lambda: datetime.utcnow().isoformat())


class HealthResponse(BaseModel):
    """Health check response"""
    
    status: str = "healthy"
    service: str = "Device Data Ingestion Service"
    version: str = "1.0.0"
    timestamp: str = Field(default_factory=lambda: datetime.utcnow().isoformat())
    kafka_connected: bool = False
    dependencies: Dict[str, str] = Field(default_factory=dict)
