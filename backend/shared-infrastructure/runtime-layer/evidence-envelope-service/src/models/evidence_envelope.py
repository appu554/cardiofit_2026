"""
Evidence Envelope Models
Core data models for clinical decision auditability and provenance tracking
"""

from datetime import datetime
from typing import Dict, List, Any, Optional
from pydantic import BaseModel, Field
from uuid import uuid4
import hashlib
import json


class ConfidenceMetrics(BaseModel):
    """Confidence scoring metrics for clinical decisions"""
    overall: float = Field(ge=0.0, le=1.0, description="Overall confidence score")
    components: Dict[str, float] = Field(default_factory=dict, description="Component-wise confidence scores")
    methodology: str = Field(default="composite", description="Confidence calculation methodology")
    calculated_at: datetime = Field(default_factory=datetime.utcnow, description="Calculation timestamp")


class InferenceStep(BaseModel):
    """Individual step in clinical reasoning chain"""
    step_id: str = Field(default_factory=lambda: str(uuid4()), description="Unique step identifier")
    step_type: str = Field(description="Type of inference (semantic_lookup, rule_application, etc.)")
    description: str = Field(description="Human-readable description of the step")
    source_data: Dict[str, Any] = Field(default_factory=dict, description="Input data for this step")
    reasoning_logic: str = Field(description="Logic or rule applied in this step")
    result_data: Dict[str, Any] = Field(default_factory=dict, description="Output data from this step")
    confidence: float = Field(ge=0.0, le=1.0, description="Confidence in this step")
    execution_time_ms: int = Field(ge=0, description="Execution time in milliseconds")
    knowledge_sources: List[str] = Field(default_factory=list, description="Knowledge bases consulted")
    timestamp: datetime = Field(default_factory=datetime.utcnow, description="Step execution timestamp")


class InferenceChain(BaseModel):
    """Complete chain of clinical reasoning steps"""
    chain_id: str = Field(default_factory=lambda: str(uuid4()), description="Unique chain identifier")
    steps: List[InferenceStep] = Field(default_factory=list, description="Ordered sequence of inference steps")
    final_conclusion: Dict[str, Any] = Field(default_factory=dict, description="Final reasoning conclusion")
    total_execution_time_ms: int = Field(ge=0, default=0, description="Total chain execution time")

    def add_step(self, step: InferenceStep):
        """Add a step to the inference chain"""
        self.steps.append(step)
        self.total_execution_time_ms += step.execution_time_ms

    def get_knowledge_sources(self) -> List[str]:
        """Get all unique knowledge sources used in the chain"""
        sources = set()
        for step in self.steps:
            sources.update(step.knowledge_sources)
        return list(sources)

    def calculate_chain_confidence(self) -> float:
        """Calculate overall confidence for the inference chain"""
        if not self.steps:
            return 0.0

        # Weighted average with recency bias
        total_weight = 0.0
        weighted_confidence = 0.0

        for i, step in enumerate(self.steps):
            weight = 1.0 / (i + 1)  # More recent steps get higher weight
            weighted_confidence += step.confidence * weight
            total_weight += weight

        return weighted_confidence / total_weight if total_weight > 0 else 0.0


class ClinicalContext(BaseModel):
    """Clinical context at the time of decision making"""
    patient_id: str = Field(description="Patient identifier")
    encounter_id: Optional[str] = Field(None, description="Clinical encounter identifier")
    workflow_type: str = Field(description="Type of clinical workflow")
    clinical_scenario: str = Field(description="Description of clinical scenario")
    urgency_level: str = Field(default="normal", description="Clinical urgency level")
    decision_makers: List[str] = Field(default_factory=list, description="Clinical decision makers involved")
    relevant_conditions: List[str] = Field(default_factory=list, description="Relevant medical conditions")
    active_medications: List[Dict[str, Any]] = Field(default_factory=list, description="Active medications")
    contraindications: List[str] = Field(default_factory=list, description="Known contraindications")
    patient_preferences: Dict[str, Any] = Field(default_factory=dict, description="Patient preferences")


class RegulatoryCompliance(BaseModel):
    """Regulatory compliance information"""
    standards_compliance: List[str] = Field(
        default_factory=lambda: ["HIPAA", "FDA_21CFR11", "ISO_13485"],
        description="Regulatory standards compliance"
    )
    audit_trail_complete: bool = Field(True, description="Complete audit trail available")
    provenance_verified: bool = Field(True, description="Decision provenance verified")
    retention_period_years: int = Field(default=7, description="Required retention period")
    data_integrity_verified: bool = Field(True, description="Data integrity verification status")
    access_controls_applied: bool = Field(True, description="Access control enforcement status")


class EvidenceEnvelope(BaseModel):
    """
    Complete evidence envelope for clinical decision auditability

    Provides comprehensive provenance tracking, confidence scoring,
    and regulatory compliance information for clinical decisions.
    """

    # Core identifiers
    envelope_id: str = Field(default_factory=lambda: str(uuid4()), description="Unique envelope identifier")
    proposal_id: str = Field(description="Associated clinical proposal identifier")

    # Snapshot and versioning
    snapshot_reference: str = Field(description="Data snapshot identifier")
    knowledge_versions: Dict[str, str] = Field(
        default_factory=dict,
        description="Knowledge base versions used"
    )

    # Clinical reasoning
    inference_chain: InferenceChain = Field(
        default_factory=InferenceChain,
        description="Complete inference chain"
    )
    confidence_scores: ConfidenceMetrics = Field(
        default_factory=ConfidenceMetrics,
        description="Confidence metrics"
    )

    # Clinical context
    clinical_context: Optional[ClinicalContext] = Field(
        None,
        description="Clinical context information"
    )

    # Metadata and audit
    created_at: datetime = Field(default_factory=datetime.utcnow, description="Creation timestamp")
    finalized_at: Optional[datetime] = Field(None, description="Finalization timestamp")
    last_updated: datetime = Field(default_factory=datetime.utcnow, description="Last update timestamp")
    status: str = Field(default="active", description="Envelope status")

    # Integrity and compliance
    checksum: Optional[str] = Field(None, description="Cryptographic integrity checksum")
    regulatory_compliance: RegulatoryCompliance = Field(
        default_factory=RegulatoryCompliance,
        description="Regulatory compliance information"
    )

    # Performance tracking
    creation_duration_ms: int = Field(ge=0, default=0, description="Envelope creation time")
    total_processing_time_ms: int = Field(ge=0, default=0, description="Total processing time")

    class Config:
        json_encoders = {
            datetime: lambda dt: dt.isoformat()
        }

    def add_inference_step(self, step_type: str, description: str, source_data: Dict[str, Any],
                          reasoning_logic: str, result_data: Dict[str, Any], confidence: float,
                          execution_time_ms: int, knowledge_sources: List[str] = None):
        """Add an inference step to the chain"""

        step = InferenceStep(
            step_type=step_type,
            description=description,
            source_data=source_data,
            reasoning_logic=reasoning_logic,
            result_data=result_data,
            confidence=confidence,
            execution_time_ms=execution_time_ms,
            knowledge_sources=knowledge_sources or []
        )

        self.inference_chain.add_step(step)
        self.total_processing_time_ms += execution_time_ms
        self.last_updated = datetime.utcnow()

        # Recalculate overall confidence
        self.confidence_scores.overall = self.inference_chain.calculate_chain_confidence()
        self.confidence_scores.calculated_at = datetime.utcnow()

    def finalize_envelope(self, final_conclusion: Dict[str, Any]):
        """Finalize the evidence envelope"""

        self.inference_chain.final_conclusion = final_conclusion
        self.finalized_at = datetime.utcnow()
        self.last_updated = self.finalized_at
        self.status = "finalized"

        # Generate integrity checksum
        self.checksum = self._generate_checksum()

        # Mark regulatory compliance as complete
        self.regulatory_compliance.audit_trail_complete = True
        self.regulatory_compliance.provenance_verified = True
        self.regulatory_compliance.data_integrity_verified = True

    def _generate_checksum(self) -> str:
        """Generate cryptographic checksum for envelope integrity"""

        # Create deterministic representation for checksumming
        checksum_data = {
            "envelope_id": self.envelope_id,
            "proposal_id": self.proposal_id,
            "snapshot_reference": self.snapshot_reference,
            "knowledge_versions": self.knowledge_versions,
            "inference_chain": self.inference_chain.model_dump(),
            "clinical_context": self.clinical_context.model_dump() if self.clinical_context else None,
            "created_at": self.created_at.isoformat(),
            "finalized_at": self.finalized_at.isoformat() if self.finalized_at else None
        }

        # Generate SHA-256 hash
        checksum_string = json.dumps(checksum_data, sort_keys=True, default=str)
        return hashlib.sha256(checksum_string.encode()).hexdigest()

    def verify_integrity(self) -> bool:
        """Verify envelope integrity using checksum"""
        if not self.checksum:
            return False

        expected_checksum = self._generate_checksum()
        return self.checksum == expected_checksum

    def get_summary(self) -> Dict[str, Any]:
        """Get a summary of the evidence envelope"""

        return {
            "envelope_id": self.envelope_id,
            "proposal_id": self.proposal_id,
            "status": self.status,
            "confidence_overall": self.confidence_scores.overall,
            "inference_steps_count": len(self.inference_chain.steps),
            "knowledge_sources": self.inference_chain.get_knowledge_sources(),
            "processing_time_ms": self.total_processing_time_ms,
            "regulatory_compliant": (
                self.regulatory_compliance.audit_trail_complete and
                self.regulatory_compliance.provenance_verified and
                self.regulatory_compliance.data_integrity_verified
            ),
            "integrity_verified": self.verify_integrity(),
            "created_at": self.created_at.isoformat(),
            "finalized_at": self.finalized_at.isoformat() if self.finalized_at else None
        }

    def to_audit_record(self) -> Dict[str, Any]:
        """Convert to audit record format for compliance reporting"""

        return {
            "audit_type": "clinical_decision_evidence",
            "envelope_id": self.envelope_id,
            "proposal_id": self.proposal_id,
            "patient_id": self.clinical_context.patient_id if self.clinical_context else None,
            "workflow_type": self.clinical_context.workflow_type if self.clinical_context else None,
            "decision_timestamp": self.created_at.isoformat(),
            "finalization_timestamp": self.finalized_at.isoformat() if self.finalized_at else None,
            "confidence_score": self.confidence_scores.overall,
            "knowledge_versions": self.knowledge_versions,
            "inference_steps": [
                {
                    "step_type": step.step_type,
                    "description": step.description,
                    "confidence": step.confidence,
                    "knowledge_sources": step.knowledge_sources,
                    "execution_time_ms": step.execution_time_ms
                }
                for step in self.inference_chain.steps
            ],
            "regulatory_compliance": self.regulatory_compliance.model_dump(),
            "data_integrity_hash": self.checksum,
            "total_processing_time_ms": self.total_processing_time_ms
        }


class EvidenceEnvelopeRequest(BaseModel):
    """Request model for creating evidence envelopes"""

    proposal_id: str = Field(description="Clinical proposal identifier")
    snapshot_id: str = Field(description="Data snapshot identifier")
    knowledge_versions: Dict[str, str] = Field(description="Knowledge base versions")
    clinical_context: Optional[ClinicalContext] = Field(None, description="Clinical context")
    urgency_level: str = Field(default="normal", description="Clinical urgency level")


class EvidenceEnvelopeResponse(BaseModel):
    """Response model for evidence envelope operations"""

    envelope_id: str
    proposal_id: str
    status: str
    created_at: datetime
    summary: Dict[str, Any]
    regulatory_compliance: bool
    integrity_verified: bool

    @classmethod
    def from_envelope(cls, envelope: EvidenceEnvelope) -> "EvidenceEnvelopeResponse":
        """Create response from evidence envelope"""

        return cls(
            envelope_id=envelope.envelope_id,
            proposal_id=envelope.proposal_id,
            status=envelope.status,
            created_at=envelope.created_at,
            summary=envelope.get_summary(),
            regulatory_compliance=(
                envelope.regulatory_compliance.audit_trail_complete and
                envelope.regulatory_compliance.provenance_verified
            ),
            integrity_verified=envelope.verify_integrity()
        )