"""
Workflow Proposal Models for database persistence.

This module defines the database models for medication proposals used in the
Calculate > Validate > Commit workflow, replacing in-memory storage.
"""
import uuid
from datetime import datetime, timezone
from typing import Dict, Any, Optional, List
from enum import Enum
from sqlalchemy import Column, String, DateTime, Text, JSON, Float, Integer, Boolean
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.dialects.postgresql import UUID
from pydantic import BaseModel, Field

Base = declarative_base()


class ProposalStatus(str, Enum):
    """Proposal status enumeration"""
    PROPOSED = "proposed"
    VALIDATED = "validated"  
    COMMITTED = "committed"
    CANCELLED = "cancelled"
    EXPIRED = "expired"
    FAILED = "failed"


class WorkflowProposal(Base):
    """
    Database model for medication proposals in workflow.
    
    This model stores proposals throughout the Calculate > Validate > Commit lifecycle
    with full audit trail and metadata support.
    """
    __tablename__ = "workflow_proposals"
    
    # Primary identification
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    proposal_id = Column(String(50), unique=True, nullable=False, index=True)
    proposal_type = Column(String(50), nullable=False, default="medication_prescription")
    
    # Workflow tracking
    status = Column(String(20), nullable=False, default=ProposalStatus.PROPOSED.value, index=True)
    workflow_phase = Column(String(20), nullable=True, index=True)  # CALCULATE, VALIDATE, COMMIT
    correlation_id = Column(String(100), nullable=True, index=True)
    
    # Patient and provider context
    patient_id = Column(String(100), nullable=False, index=True)
    provider_id = Column(String(100), nullable=False, index=True)
    encounter_id = Column(String(100), nullable=True)
    
    # Medication details (stored as JSON for flexibility)
    medication_data = Column(JSON, nullable=False)
    clinical_context = Column(JSON, nullable=True)
    patient_context = Column(JSON, nullable=True)
    
    # Snapshot and consistency tracking
    snapshot_id = Column(String(100), nullable=True, index=True)
    snapshot_checksum = Column(String(64), nullable=True)
    kb_versions = Column(JSON, nullable=True)
    
    # Validation tracking
    validation_id = Column(String(100), nullable=True, index=True)
    validation_verdict = Column(String(20), nullable=True)
    validation_risk_score = Column(Float, nullable=True)
    validation_findings = Column(JSON, nullable=True)
    override_token = Column(String(100), nullable=True)
    
    # Commit tracking
    medication_order_id = Column(String(100), nullable=True, index=True)
    fhir_resource_id = Column(String(100), nullable=True)
    audit_trail_id = Column(String(100), nullable=True)
    
    # Priority and routing
    priority = Column(String(20), nullable=False, default="routine")
    urgency_level = Column(Integer, nullable=False, default=1)  # 1=routine, 5=urgent
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), nullable=False, default=datetime.utcnow, index=True)
    updated_at = Column(DateTime(timezone=True), nullable=False, default=datetime.utcnow, onupdate=datetime.utcnow)
    validated_at = Column(DateTime(timezone=True), nullable=True)
    committed_at = Column(DateTime(timezone=True), nullable=True)
    expires_at = Column(DateTime(timezone=True), nullable=True)
    
    # User tracking
    created_by = Column(String(100), nullable=False)
    updated_by = Column(String(100), nullable=True)
    committed_by = Column(String(100), nullable=True)
    
    # Processing metadata
    processing_time_ms = Column(Integer, nullable=True)
    error_message = Column(Text, nullable=True)
    retry_count = Column(Integer, nullable=False, default=0)
    
    # Additional metadata
    metadata = Column(JSON, nullable=True)
    notes = Column(Text, nullable=True)
    
    def __repr__(self):
        return f"<WorkflowProposal(id={self.proposal_id}, status={self.status}, patient={self.patient_id})>"
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert model to dictionary format."""
        return {
            "id": str(self.id),
            "proposal_id": self.proposal_id,
            "proposal_type": self.proposal_type,
            "status": self.status,
            "workflow_phase": self.workflow_phase,
            "correlation_id": self.correlation_id,
            "patient_id": self.patient_id,
            "provider_id": self.provider_id,
            "encounter_id": self.encounter_id,
            "medication_data": self.medication_data,
            "clinical_context": self.clinical_context,
            "patient_context": self.patient_context,
            "snapshot_id": self.snapshot_id,
            "snapshot_checksum": self.snapshot_checksum,
            "kb_versions": self.kb_versions,
            "validation_id": self.validation_id,
            "validation_verdict": self.validation_verdict,
            "validation_risk_score": self.validation_risk_score,
            "validation_findings": self.validation_findings,
            "override_token": self.override_token,
            "medication_order_id": self.medication_order_id,
            "fhir_resource_id": self.fhir_resource_id,
            "audit_trail_id": self.audit_trail_id,
            "priority": self.priority,
            "urgency_level": self.urgency_level,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
            "validated_at": self.validated_at.isoformat() if self.validated_at else None,
            "committed_at": self.committed_at.isoformat() if self.committed_at else None,
            "expires_at": self.expires_at.isoformat() if self.expires_at else None,
            "created_by": self.created_by,
            "updated_by": self.updated_by,
            "committed_by": self.committed_by,
            "processing_time_ms": self.processing_time_ms,
            "error_message": self.error_message,
            "retry_count": self.retry_count,
            "metadata": self.metadata,
            "notes": self.notes
        }


class ProposalAuditLog(Base):
    """
    Audit log for proposal operations.
    
    Tracks all operations performed on proposals for compliance and debugging.
    """
    __tablename__ = "proposal_audit_log"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    audit_trail_id = Column(String(100), nullable=False, index=True)
    proposal_id = Column(String(50), nullable=False, index=True)
    
    # Operation details
    operation = Column(String(50), nullable=False)  # CREATE, VALIDATE, COMMIT, etc.
    operation_status = Column(String(20), nullable=False)  # SUCCESS, FAILURE, IN_PROGRESS
    workflow_phase = Column(String(20), nullable=True)
    
    # Context
    user_id = Column(String(100), nullable=False)
    correlation_id = Column(String(100), nullable=True)
    session_id = Column(String(100), nullable=True)
    
    # Timing
    timestamp = Column(DateTime(timezone=True), nullable=False, default=datetime.utcnow, index=True)
    duration_ms = Column(Integer, nullable=True)
    
    # Data changes
    old_values = Column(JSON, nullable=True)
    new_values = Column(JSON, nullable=True)
    operation_context = Column(JSON, nullable=True)
    
    # Error tracking
    error_message = Column(Text, nullable=True)
    stack_trace = Column(Text, nullable=True)
    
    # Additional metadata
    metadata = Column(JSON, nullable=True)
    
    def __repr__(self):
        return f"<ProposalAuditLog(proposal={self.proposal_id}, operation={self.operation}, status={self.operation_status})>"


# Pydantic models for API responses

class ProposalSummary(BaseModel):
    """Summary view of a proposal."""
    proposal_id: str
    status: str
    patient_id: str
    medication_name: str
    provider_id: str
    created_at: datetime
    priority: str


class ProposalDetails(BaseModel):
    """Detailed view of a proposal."""
    id: str
    proposal_id: str
    proposal_type: str
    status: str
    workflow_phase: Optional[str]
    correlation_id: Optional[str]
    patient_id: str
    provider_id: str
    encounter_id: Optional[str]
    medication_data: Dict[str, Any]
    clinical_context: Optional[Dict[str, Any]]
    patient_context: Optional[Dict[str, Any]]
    snapshot_id: Optional[str]
    validation_id: Optional[str]
    validation_verdict: Optional[str]
    validation_risk_score: Optional[float]
    medication_order_id: Optional[str]
    fhir_resource_id: Optional[str]
    audit_trail_id: Optional[str]
    priority: str
    created_at: datetime
    updated_at: datetime
    validated_at: Optional[datetime]
    committed_at: Optional[datetime]
    expires_at: Optional[datetime]
    created_by: str
    committed_by: Optional[str]
    processing_time_ms: Optional[int]
    error_message: Optional[str]
    retry_count: int
    metadata: Optional[Dict[str, Any]]
    notes: Optional[str]


class ProposalCreateRequest(BaseModel):
    """Request model for creating proposals."""
    proposal_type: str = Field(default="medication_prescription")
    patient_id: str = Field(..., min_length=1)
    provider_id: str = Field(..., min_length=1)
    encounter_id: Optional[str] = None
    medication_data: Dict[str, Any] = Field(...)
    clinical_context: Optional[Dict[str, Any]] = None
    patient_context: Optional[Dict[str, Any]] = None
    correlation_id: Optional[str] = None
    priority: str = Field(default="routine")
    urgency_level: int = Field(default=1, ge=1, le=5)
    expires_in_hours: Optional[int] = Field(default=24, ge=1, le=168)  # Max 1 week
    metadata: Optional[Dict[str, Any]] = None
    notes: Optional[str] = None


class ProposalUpdateRequest(BaseModel):
    """Request model for updating proposals."""
    status: Optional[str] = None
    workflow_phase: Optional[str] = None
    snapshot_id: Optional[str] = None
    validation_id: Optional[str] = None
    validation_verdict: Optional[str] = None
    validation_risk_score: Optional[float] = None
    validation_findings: Optional[List[Dict[str, Any]]] = None
    override_token: Optional[str] = None
    medication_order_id: Optional[str] = None
    fhir_resource_id: Optional[str] = None
    audit_trail_id: Optional[str] = None
    processing_time_ms: Optional[int] = None
    error_message: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None
    notes: Optional[str] = None


class AuditLogEntry(BaseModel):
    """Audit log entry model."""
    id: str
    audit_trail_id: str
    proposal_id: str
    operation: str
    operation_status: str
    workflow_phase: Optional[str]
    user_id: str
    correlation_id: Optional[str]
    timestamp: datetime
    duration_ms: Optional[int]
    old_values: Optional[Dict[str, Any]]
    new_values: Optional[Dict[str, Any]]
    operation_context: Optional[Dict[str, Any]]
    error_message: Optional[str]
    metadata: Optional[Dict[str, Any]]