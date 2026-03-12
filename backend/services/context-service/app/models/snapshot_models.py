"""
Clinical Snapshot Data Models

This module defines the data models for the Recipe Snapshot architecture,
providing immutable clinical snapshots with cryptographic integrity.
"""

from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List
from pydantic import BaseModel, Field
from enum import Enum
import uuid


class SnapshotStatus(str, Enum):
    """Snapshot status enumeration"""
    ACTIVE = "active"
    EXPIRED = "expired"
    INVALIDATED = "invalidated"


class SignatureMethod(str, Enum):
    """Digital signature method enumeration"""
    RSA_2048 = "rsa-2048"
    ECDSA_P256 = "ecdsa-p256"
    MOCK = "mock"  # For development/testing


class SnapshotRequest(BaseModel):
    """Request model for creating a clinical snapshot"""
    patient_id: str = Field(..., description="Patient ID for the snapshot")
    recipe_id: str = Field(..., description="Recipe ID to use for data assembly")
    provider_id: Optional[str] = Field(None, description="Provider ID requesting the snapshot")
    encounter_id: Optional[str] = Field(None, description="Encounter ID associated with the snapshot")
    ttl_hours: int = Field(1, ge=1, le=24, description="Time-to-live in hours (1-24)")
    force_refresh: bool = Field(False, description="Force refresh from data sources")
    signature_method: SignatureMethod = Field(SignatureMethod.MOCK, description="Digital signature method")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }


class ClinicalSnapshot(BaseModel):
    """Immutable clinical snapshot with cryptographic integrity"""
    
    # Core identification
    id: str = Field(default_factory=lambda: str(uuid.uuid4()), description="Unique snapshot ID")
    patient_id: str = Field(..., description="Patient ID")
    recipe_id: str = Field(..., description="Recipe ID used for assembly")
    context_id: str = Field(..., description="Original context assembly ID")
    
    # Clinical data
    data: Dict[str, Any] = Field(..., description="Assembled clinical data")
    completeness_score: float = Field(..., ge=0.0, le=1.0, description="Data completeness score")
    
    # Integrity verification
    checksum: str = Field(..., description="SHA-256 checksum of the clinical data")
    signature: str = Field(..., description="Digital signature for authenticity")
    signature_method: SignatureMethod = Field(..., description="Signature method used")
    
    # Lifecycle management
    status: SnapshotStatus = Field(SnapshotStatus.ACTIVE, description="Snapshot status")
    created_at: datetime = Field(default_factory=datetime.utcnow, description="Creation timestamp")
    expires_at: datetime = Field(..., description="Expiration timestamp")
    accessed_count: int = Field(0, description="Number of times accessed")
    last_accessed_at: Optional[datetime] = Field(None, description="Last access timestamp")
    
    # Audit and traceability
    provider_id: Optional[str] = Field(None, description="Provider ID who requested the snapshot")
    encounter_id: Optional[str] = Field(None, description="Associated encounter ID")
    assembly_metadata: Dict[str, Any] = Field(..., description="Original assembly metadata")
    
    # Evidence envelope for clinical safety
    evidence_envelope: Dict[str, Any] = Field(default_factory=dict, description="Clinical evidence and decision trail")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
    
    def is_expired(self) -> bool:
        """Check if the snapshot has expired"""
        return datetime.utcnow() > self.expires_at
    
    def is_valid(self) -> bool:
        """Check if the snapshot is valid (not expired or invalidated)"""
        return self.status == SnapshotStatus.ACTIVE and not self.is_expired()
    
    def mark_accessed(self) -> None:
        """Mark the snapshot as accessed"""
        self.accessed_count += 1
        self.last_accessed_at = datetime.utcnow()


class SnapshotValidationResult(BaseModel):
    """Result of snapshot integrity validation"""
    
    snapshot_id: str = Field(..., description="Snapshot ID being validated")
    valid: bool = Field(..., description="Whether the snapshot is valid")
    checksum_valid: bool = Field(..., description="Whether the checksum is valid")
    signature_valid: bool = Field(..., description="Whether the signature is valid")
    not_expired: bool = Field(..., description="Whether the snapshot has not expired")
    
    errors: List[str] = Field(default_factory=list, description="List of validation errors")
    warnings: List[str] = Field(default_factory=list, description="List of validation warnings")
    
    validated_at: datetime = Field(default_factory=datetime.utcnow, description="Validation timestamp")
    validation_duration_ms: float = Field(..., description="Validation duration in milliseconds")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }


class SnapshotSummary(BaseModel):
    """Summary information about a clinical snapshot"""
    
    id: str = Field(..., description="Snapshot ID")
    patient_id: str = Field(..., description="Patient ID")
    recipe_id: str = Field(..., description="Recipe ID")
    status: SnapshotStatus = Field(..., description="Snapshot status")
    created_at: datetime = Field(..., description="Creation timestamp")
    expires_at: datetime = Field(..., description="Expiration timestamp")
    completeness_score: float = Field(..., description="Data completeness score")
    accessed_count: int = Field(..., description="Access count")
    provider_id: Optional[str] = Field(None, description="Provider ID")
    encounter_id: Optional[str] = Field(None, description="Encounter ID")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }


class SnapshotMetrics(BaseModel):
    """Metrics for snapshot operations"""
    
    total_snapshots: int = Field(..., description="Total number of snapshots")
    active_snapshots: int = Field(..., description="Number of active snapshots")
    expired_snapshots: int = Field(..., description="Number of expired snapshots")
    
    average_completeness: float = Field(..., description="Average completeness score")
    average_ttl_hours: float = Field(..., description="Average TTL in hours")
    
    creation_rate_per_hour: float = Field(..., description="Snapshot creation rate per hour")
    access_rate_per_hour: float = Field(..., description="Snapshot access rate per hour")
    
    top_recipes: List[Dict[str, Any]] = Field(..., description="Most used recipes")
    top_providers: List[Dict[str, Any]] = Field(..., description="Most active providers")
    
    calculated_at: datetime = Field(default_factory=datetime.utcnow, description="Metrics calculation timestamp")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }


# Database collection models for MongoDB
class SnapshotDocument(BaseModel):
    """MongoDB document model for clinical snapshots"""
    
    _id: str = Field(..., alias="id")
    patient_id: str
    recipe_id: str
    context_id: str
    data: Dict[str, Any]
    completeness_score: float
    checksum: str
    signature: str
    signature_method: str
    status: str
    created_at: datetime
    expires_at: datetime
    accessed_count: int
    last_accessed_at: Optional[datetime]
    provider_id: Optional[str]
    encounter_id: Optional[str]
    assembly_metadata: Dict[str, Any]
    evidence_envelope: Dict[str, Any]
    
    # TTL index will be created on expires_at field
    
    class Config:
        allow_population_by_field_name = True
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }