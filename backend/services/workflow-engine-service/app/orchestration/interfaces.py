"""
Snapshot-Aware Orchestration Interfaces

This module defines the core interfaces, data models, and type definitions for the enhanced
snapshot-aware orchestration system. These interfaces extend the existing strategic orchestration
to support snapshot consistency across workflow phases.
"""

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any, Union
from datetime import datetime
from enum import Enum
from pydantic import BaseModel, Field, validator
import hashlib
import json


class SnapshotStatus(Enum):
    """Status enumeration for snapshot lifecycle"""
    CREATED = "created"
    ACTIVE = "active"
    EXPIRED = "expired"
    ARCHIVED = "archived"
    CORRUPTED = "corrupted"


class WorkflowPhase(Enum):
    """Workflow phases that use snapshots"""
    CALCULATE = "calculate"
    VALIDATE = "validate"
    COMMIT = "commit"
    OVERRIDE = "override"


@dataclass
class SnapshotReference:
    """
    Immutable reference to a clinical snapshot used across workflow phases.
    
    This class ensures snapshot consistency and provides integrity validation
    for clinical data used throughout the workflow execution.
    """
    snapshot_id: str
    checksum: str
    created_at: datetime
    expires_at: datetime
    status: SnapshotStatus
    phase_created: WorkflowPhase
    patient_id: str
    context_version: str
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def is_valid(self) -> bool:
        """Check if snapshot is still valid and not expired"""
        now = datetime.utcnow()
        return (
            self.status == SnapshotStatus.ACTIVE and
            self.expires_at > now
        )
    
    def validate_integrity(self, data: Dict[str, Any]) -> bool:
        """Validate snapshot data integrity using checksum"""
        try:
            # Create deterministic checksum from data
            data_str = json.dumps(data, sort_keys=True, separators=(',', ':'))
            calculated_checksum = hashlib.sha256(data_str.encode()).hexdigest()
            return calculated_checksum == self.checksum
        except Exception:
            return False
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "snapshot_id": self.snapshot_id,
            "checksum": self.checksum,
            "created_at": self.created_at.isoformat(),
            "expires_at": self.expires_at.isoformat(),
            "status": self.status.value,
            "phase_created": self.phase_created.value,
            "patient_id": self.patient_id,
            "context_version": self.context_version,
            "metadata": self.metadata,
            "is_valid": self.is_valid()
        }


@dataclass
class SnapshotChainTracker:
    """
    Tracks snapshot usage across all workflow phases.
    
    Ensures snapshot consistency and provides audit trail for clinical decisions
    made using specific data snapshots.
    """
    workflow_id: str
    calculate_snapshot: Optional[SnapshotReference] = None
    validate_snapshot: Optional[SnapshotReference] = None
    commit_snapshot: Optional[SnapshotReference] = None
    override_snapshot: Optional[SnapshotReference] = None
    chain_created_at: datetime = field(default_factory=datetime.utcnow)
    
    def add_phase_snapshot(self, phase: WorkflowPhase, snapshot: SnapshotReference) -> None:
        """Add snapshot reference for a specific workflow phase"""
        if phase == WorkflowPhase.CALCULATE:
            self.calculate_snapshot = snapshot
        elif phase == WorkflowPhase.VALIDATE:
            self.validate_snapshot = snapshot
        elif phase == WorkflowPhase.COMMIT:
            self.commit_snapshot = snapshot
        elif phase == WorkflowPhase.OVERRIDE:
            self.override_snapshot = snapshot
    
    def validate_chain_consistency(self) -> bool:
        """Validate that all snapshots in the chain are consistent"""
        snapshots = [s for s in [
            self.calculate_snapshot,
            self.validate_snapshot,
            self.commit_snapshot
        ] if s is not None]
        
        if len(snapshots) < 2:
            return True  # Single or no snapshots are consistent by definition
        
        # Check that all snapshots have the same patient_id and context_version
        base_snapshot = snapshots[0]
        return all(
            s.patient_id == base_snapshot.patient_id and
            s.context_version == base_snapshot.context_version
            for s in snapshots[1:]
        )
    
    def get_primary_snapshot(self) -> Optional[SnapshotReference]:
        """Get the primary snapshot used for this workflow (usually from calculate phase)"""
        return (self.calculate_snapshot or 
                self.validate_snapshot or 
                self.commit_snapshot or 
                self.override_snapshot)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses and persistence"""
        return {
            "workflow_id": self.workflow_id,
            "calculate_snapshot": self.calculate_snapshot.to_dict() if self.calculate_snapshot else None,
            "validate_snapshot": self.validate_snapshot.to_dict() if self.validate_snapshot else None,
            "commit_snapshot": self.commit_snapshot.to_dict() if self.commit_snapshot else None,
            "override_snapshot": self.override_snapshot.to_dict() if self.override_snapshot else None,
            "chain_created_at": self.chain_created_at.isoformat(),
            "is_consistent": self.validate_chain_consistency()
        }


@dataclass
class RecipeReference:
    """
    Reference to a clinical recipe used in workflow execution.
    
    Tracks recipe versions and resolution metadata for audit and learning purposes.
    """
    recipe_id: str
    version: str
    resolved_at: datetime
    resolution_source: str  # "cache", "service", "fallback"
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "recipe_id": self.recipe_id,
            "version": self.version,
            "resolved_at": self.resolved_at.isoformat(),
            "resolution_source": self.resolution_source,
            "metadata": self.metadata
        }


@dataclass
class EvidenceEnvelope:
    """
    Container for clinical evidence generated during workflow execution.
    
    Provides structured container for clinical reasoning, safety assessments,
    and decision support evidence linked to specific snapshots.
    """
    evidence_id: str
    snapshot_id: str
    phase: WorkflowPhase
    evidence_type: str  # "clinical_reasoning", "safety_assessment", "decision_support"
    content: Dict[str, Any]
    confidence_score: float
    generated_at: datetime
    source: str  # "flow2_engine", "safety_gateway", "clinical_rules"
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "evidence_id": self.evidence_id,
            "snapshot_id": self.snapshot_id,
            "phase": self.phase.value,
            "evidence_type": self.evidence_type,
            "content": self.content,
            "confidence_score": self.confidence_score,
            "generated_at": self.generated_at.isoformat(),
            "source": self.source
        }


@dataclass
class ClinicalOverride:
    """
    Record of clinical provider override decisions.
    
    Captures provider justifications and context for learning loop analysis.
    """
    override_id: str
    workflow_id: str
    snapshot_id: str
    override_type: str  # "warning_override", "safety_override", "protocol_override"
    original_verdict: str
    overridden_to: str
    clinician_id: str
    justification: str
    override_tokens: List[str]
    override_timestamp: datetime
    patient_context: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses and learning analysis"""
        return {
            "override_id": self.override_id,
            "workflow_id": self.workflow_id,
            "snapshot_id": self.snapshot_id,
            "override_type": self.override_type,
            "original_verdict": self.original_verdict,
            "overridden_to": self.overridden_to,
            "clinician_id": self.clinician_id,
            "justification": self.justification,
            "override_tokens": self.override_tokens,
            "override_timestamp": self.override_timestamp.isoformat(),
            "patient_context": self.patient_context
        }


@dataclass
class ProposalWithSnapshot:
    """
    Enhanced proposal response that includes snapshot metadata.
    
    Extends the standard proposal response to include snapshot references
    for consistency validation across workflow phases.
    """
    proposal_set_id: str
    snapshot_reference: SnapshotReference
    ranked_proposals: List[Dict[str, Any]]
    clinical_evidence: Dict[str, Any]
    monitoring_plan: Dict[str, Any]
    recipe_reference: Optional[RecipeReference] = None
    execution_metrics: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "proposal_set_id": self.proposal_set_id,
            "snapshot_reference": self.snapshot_reference.to_dict(),
            "ranked_proposals": self.ranked_proposals,
            "clinical_evidence": self.clinical_evidence,
            "monitoring_plan": self.monitoring_plan,
            "recipe_reference": self.recipe_reference.to_dict() if self.recipe_reference else None,
            "execution_metrics": self.execution_metrics
        }


@dataclass
class ValidationResult:
    """
    Enhanced validation result with snapshot consistency validation.
    
    Includes snapshot verification to ensure validation was performed
    on the same clinical data as the calculation phase.
    """
    validation_id: str
    snapshot_reference: SnapshotReference
    verdict: str  # "SAFE", "WARNING", "UNSAFE"
    findings: List[Dict[str, Any]]
    evidence_envelope: EvidenceEnvelope
    override_tokens: Optional[List[str]] = None
    approval_requirements: Optional[Dict[str, Any]] = None
    validation_metrics: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "validation_id": self.validation_id,
            "snapshot_reference": self.snapshot_reference.to_dict(),
            "verdict": self.verdict,
            "findings": self.findings,
            "evidence_envelope": self.evidence_envelope.to_dict(),
            "override_tokens": self.override_tokens,
            "approval_requirements": self.approval_requirements,
            "validation_metrics": self.validation_metrics
        }


@dataclass
class CommitResult:
    """
    Enhanced commit result with snapshot audit trail.
    
    Provides complete audit trail of snapshot usage and clinical decisions
    for regulatory compliance and learning analysis.
    """
    medication_order_id: str
    snapshot_reference: SnapshotReference
    audit_trail_id: str
    persistence_status: str
    event_publication_status: str
    snapshot_chain: SnapshotChainTracker
    commit_timestamp: datetime = field(default_factory=datetime.utcnow)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "medication_order_id": self.medication_order_id,
            "snapshot_reference": self.snapshot_reference.to_dict(),
            "audit_trail_id": self.audit_trail_id,
            "persistence_status": self.persistence_status,
            "event_publication_status": self.event_publication_status,
            "snapshot_chain": self.snapshot_chain.to_dict(),
            "commit_timestamp": self.commit_timestamp.isoformat()
        }


# Pydantic models for API validation

class ClinicalCommand(BaseModel):
    """Enhanced clinical command with snapshot awareness"""
    patient_id: str = Field(..., description="Patient identifier")
    medication_request: Dict[str, Any] = Field(..., description="Medication details")
    clinical_intent: Dict[str, Any] = Field(..., description="Clinical indication and goals")
    provider_context: Dict[str, Any] = Field(..., description="Provider and context information")
    correlation_id: str = Field(..., description="Workflow correlation ID")
    urgency: str = Field(default="ROUTINE", description="Request urgency level")
    snapshot_requirements: Dict[str, Any] = Field(default_factory=dict, description="Snapshot-specific requirements")
    
    @validator('patient_id')
    def validate_patient_id(cls, v):
        if not v or len(v.strip()) == 0:
            raise ValueError('Patient ID cannot be empty')
        return v.strip()
    
    @validator('correlation_id')
    def validate_correlation_id(cls, v):
        if not v or len(v.strip()) == 0:
            raise ValueError('Correlation ID cannot be empty')
        return v.strip()


class WorkflowInstance(BaseModel):
    """Enhanced workflow instance with snapshot state tracking"""
    workflow_id: str = Field(..., description="Unique workflow identifier")
    patient_id: str = Field(..., description="Patient identifier")
    status: str = Field(..., description="Current workflow status")
    snapshot_chain: Dict[str, Any] = Field(default_factory=dict, description="Snapshot chain tracker")
    recipe_reference: Optional[Dict[str, Any]] = Field(None, description="Recipe reference if applicable")
    evidence_envelopes: List[Dict[str, Any]] = Field(default_factory=list, description="Clinical evidence")
    override_history: List[Dict[str, Any]] = Field(default_factory=list, description="Provider overrides")
    created_at: datetime = Field(default_factory=datetime.utcnow, description="Workflow creation timestamp")
    updated_at: datetime = Field(default_factory=datetime.utcnow, description="Last update timestamp")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }


# Exception classes for snapshot-specific errors

class SnapshotError(Exception):
    """Base exception for snapshot-related errors"""
    pass


class SnapshotExpiredError(SnapshotError):
    """Raised when snapshot has expired between workflow phases"""
    def __init__(self, snapshot_id: str, expired_at: datetime):
        self.snapshot_id = snapshot_id
        self.expired_at = expired_at
        super().__init__(f"Snapshot {snapshot_id} expired at {expired_at}")


class SnapshotIntegrityError(SnapshotError):
    """Raised when snapshot checksum validation fails"""
    def __init__(self, snapshot_id: str, expected_checksum: str, actual_checksum: str):
        self.snapshot_id = snapshot_id
        self.expected_checksum = expected_checksum
        self.actual_checksum = actual_checksum
        super().__init__(f"Snapshot {snapshot_id} integrity check failed: expected {expected_checksum}, got {actual_checksum}")


class SnapshotNotFoundError(SnapshotError):
    """Raised when referenced snapshot cannot be found"""
    def __init__(self, snapshot_id: str):
        self.snapshot_id = snapshot_id
        super().__init__(f"Snapshot {snapshot_id} not found")


class SnapshotConsistencyError(SnapshotError):
    """Raised when snapshot consistency validation fails between phases"""
    def __init__(self, message: str, snapshot_chain: Optional[SnapshotChainTracker] = None):
        self.snapshot_chain = snapshot_chain
        super().__init__(message)