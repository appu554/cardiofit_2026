"""
Prescription Entity - Two-Phase Operations Support
Implements the Propose/Commit pattern for medication prescriptions
"""

from dataclasses import dataclass, field
from typing import Optional, Dict, Any, List
from uuid import UUID, uuid4
from datetime import datetime
from enum import Enum

from ..value_objects.dose_specification import (
    DoseProposal, DoseSpecification, Frequency, Duration, Quantity
)


class PrescriptionStatus(Enum):
    """Two-phase prescription statuses"""
    PROPOSED = "PROPOSED"      # Phase 1: Proposal created, awaiting validation
    COMMITTED = "COMMITTED"    # Phase 2: Validated and committed by Workflow Engine
    CANCELLED = "CANCELLED"    # Cancelled before commitment


@dataclass
class PrescriptionProposal:
    """
    Phase 1: Medication Proposal (Stateless)
    
    Represents the "Calculate" phase of Calculate → Validate → Commit
    Contains all pharmaceutical intelligence without side effects
    """
    
    # Identity
    proposal_id: UUID = field(default_factory=uuid4)
    patient_id: str = field(default="")
    medication_id: UUID = field(default_factory=uuid4)
    
    # Proposed Therapy
    dose_proposal: DoseProposal = field(default=None)
    prescriber_id: str = field(default="")
    indication: str = field(default="")
    special_instructions: Optional[str] = None
    
    # Business Context Used
    business_context: Dict[str, Any] = field(default_factory=dict)
    calculation_metadata: Dict[str, Any] = field(default_factory=dict)
    
    # Proposal Metadata
    proposed_at: datetime = field(default_factory=datetime.utcnow)
    expires_at: Optional[datetime] = None
    recipe_id: str = field(default="")
    
    def __post_init__(self):
        """Validate prescription proposal"""
        if not self.patient_id:
            raise ValueError("Patient ID is required")
        if not self.prescriber_id:
            raise ValueError("Prescriber ID is required")
        if not self.dose_proposal:
            raise ValueError("Dose proposal is required")
        if not self.indication:
            raise ValueError("Indication is required")
    
    def is_expired(self) -> bool:
        """Check if proposal has expired"""
        if not self.expires_at:
            return False
        return datetime.utcnow() > self.expires_at
    
    def get_proposed_dose_summary(self) -> str:
        """Get summary of proposed dose"""
        return self.dose_proposal.to_summary_string()
    
    def has_warnings(self) -> bool:
        """Check if proposal has clinical warnings"""
        return self.dose_proposal.has_warnings()
    
    def is_high_confidence(self) -> bool:
        """Check if proposal has high confidence"""
        return self.dose_proposal.is_high_confidence()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'proposal_id': str(self.proposal_id),
            'patient_id': self.patient_id,
            'medication_id': str(self.medication_id),
            'dose_summary': self.get_proposed_dose_summary(),
            'prescriber_id': self.prescriber_id,
            'indication': self.indication,
            'special_instructions': self.special_instructions,
            'proposed_at': self.proposed_at.isoformat(),
            'expires_at': self.expires_at.isoformat() if self.expires_at else None,
            'confidence_score': float(self.dose_proposal.confidence_score),
            'warnings': self.dose_proposal.warnings,
            'clinical_notes': self.dose_proposal.clinical_notes,
            'recipe_id': self.recipe_id
        }


@dataclass
class Prescription:
    """
    Phase 2: Committed Prescription (Stateful)
    
    Represents the "Commit" phase after Safety Gateway validation
    Contains the final, committed prescription with audit trail
    """
    
    # Identity
    prescription_id: UUID = field(default_factory=uuid4)
    proposal_id: UUID = field(default_factory=uuid4)  # Links back to original proposal
    patient_id: str = field(default="")
    medication_id: UUID = field(default_factory=uuid4)
    
    # Committed Therapy (may differ from proposal after validation)
    dose: DoseSpecification = field(default=None)
    frequency: Frequency = field(default=None)
    duration: Duration = field(default=None)
    quantity: Quantity = field(default=None)
    
    # Prescription Details
    prescriber_id: str = field(default="")
    indication: str = field(default="")
    special_instructions: Optional[str] = None
    refills: int = 0
    
    # Two-Phase Lifecycle
    status: PrescriptionStatus = PrescriptionStatus.PROPOSED
    
    # Phase 1: Proposal
    proposal_timestamp: datetime = field(default_factory=datetime.utcnow)
    proposal_context: Dict[str, Any] = field(default_factory=dict)
    
    # Phase 2: Commitment
    commit_timestamp: Optional[datetime] = None
    commit_metadata: Dict[str, Any] = field(default_factory=dict)
    committed_by: Optional[str] = None  # Workflow Engine identifier
    
    # Audit Trail
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
    version: int = 1
    
    def __post_init__(self):
        """Validate prescription"""
        if not self.patient_id:
            raise ValueError("Patient ID is required")
        if not self.prescriber_id:
            raise ValueError("Prescriber ID is required")
        if not self.indication:
            raise ValueError("Indication is required")
    
    @classmethod
    def from_proposal(
        cls, 
        proposal: PrescriptionProposal,
        committed_by: str,
        modifications: Optional[Dict[str, Any]] = None
    ) -> 'Prescription':
        """
        Create committed prescription from proposal
        
        This is called by the Workflow Engine after Safety Gateway validation
        """
        # Use proposal values or modifications from Safety Gateway
        final_dose = proposal.dose_proposal.calculated_dose
        final_frequency = proposal.dose_proposal.frequency
        final_duration = proposal.dose_proposal.duration
        final_quantity = proposal.dose_proposal.quantity
        
        # Apply any modifications from Safety Gateway validation
        if modifications:
            if 'dose' in modifications:
                final_dose = modifications['dose']
            if 'frequency' in modifications:
                final_frequency = modifications['frequency']
            if 'duration' in modifications:
                final_duration = modifications['duration']
            if 'quantity' in modifications:
                final_quantity = modifications['quantity']
        
        return cls(
            proposal_id=proposal.proposal_id,
            patient_id=proposal.patient_id,
            medication_id=proposal.medication_id,
            dose=final_dose,
            frequency=final_frequency,
            duration=final_duration,
            quantity=final_quantity,
            prescriber_id=proposal.prescriber_id,
            indication=proposal.indication,
            special_instructions=proposal.special_instructions,
            status=PrescriptionStatus.COMMITTED,
            proposal_timestamp=proposal.proposed_at,
            proposal_context={
                'original_proposal': proposal.to_dict(),
                'business_context': proposal.business_context,
                'calculation_metadata': proposal.calculation_metadata
            },
            commit_timestamp=datetime.utcnow(),
            commit_metadata={
                'committed_by': committed_by,
                'modifications_applied': modifications or {},
                'safety_validation_passed': True
            },
            committed_by=committed_by
        )
    
    def commit(
        self, 
        committed_by: str,
        modifications: Optional[Dict[str, Any]] = None
    ):
        """
        Commit a proposed prescription
        
        This method is called by the Workflow Engine after validation
        """
        if self.status != PrescriptionStatus.PROPOSED:
            raise ValueError(f"Cannot commit prescription in status: {self.status}")
        
        # Apply modifications if provided
        if modifications:
            if 'dose' in modifications:
                self.dose = modifications['dose']
            if 'frequency' in modifications:
                self.frequency = modifications['frequency']
            if 'duration' in modifications:
                self.duration = modifications['duration']
            if 'quantity' in modifications:
                self.quantity = modifications['quantity']
        
        # Update status and metadata
        self.status = PrescriptionStatus.COMMITTED
        self.commit_timestamp = datetime.utcnow()
        self.committed_by = committed_by
        self.commit_metadata = {
            'committed_by': committed_by,
            'modifications_applied': modifications or {},
            'safety_validation_passed': True
        }
        self.updated_at = datetime.utcnow()
        self.version += 1
    
    def cancel(self, reason: str, cancelled_by: str):
        """Cancel a prescription"""
        if self.status == PrescriptionStatus.COMMITTED:
            raise ValueError("Cannot cancel committed prescription")
        
        self.status = PrescriptionStatus.CANCELLED
        self.commit_metadata = {
            'cancelled_by': cancelled_by,
            'cancellation_reason': reason,
            'cancelled_at': datetime.utcnow().isoformat()
        }
        self.updated_at = datetime.utcnow()
        self.version += 1
    
    def is_proposed(self) -> bool:
        """Check if prescription is in proposed state"""
        return self.status == PrescriptionStatus.PROPOSED
    
    def is_committed(self) -> bool:
        """Check if prescription is committed"""
        return self.status == PrescriptionStatus.COMMITTED
    
    def is_cancelled(self) -> bool:
        """Check if prescription is cancelled"""
        return self.status == PrescriptionStatus.CANCELLED
    
    def get_dose_summary(self) -> str:
        """Get summary of prescription dose"""
        if not self.dose or not self.frequency or not self.duration:
            return "Incomplete prescription"
        
        return f"{self.dose.to_display_string()} {self.frequency.to_display_string()} for {self.duration.to_display_string()}"
    
    def get_audit_trail(self) -> Dict[str, Any]:
        """Get complete audit trail"""
        return {
            'prescription_id': str(self.prescription_id),
            'proposal_id': str(self.proposal_id),
            'status': self.status.value,
            'lifecycle': {
                'proposed_at': self.proposal_timestamp.isoformat(),
                'committed_at': self.commit_timestamp.isoformat() if self.commit_timestamp else None,
                'committed_by': self.committed_by
            },
            'proposal_context': self.proposal_context,
            'commit_metadata': self.commit_metadata,
            'version': self.version,
            'created_at': self.created_at.isoformat(),
            'updated_at': self.updated_at.isoformat()
        }
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'prescription_id': str(self.prescription_id),
            'proposal_id': str(self.proposal_id),
            'patient_id': self.patient_id,
            'medication_id': str(self.medication_id),
            'dose_summary': self.get_dose_summary(),
            'prescriber_id': self.prescriber_id,
            'indication': self.indication,
            'special_instructions': self.special_instructions,
            'refills': self.refills,
            'status': self.status.value,
            'proposed_at': self.proposal_timestamp.isoformat(),
            'committed_at': self.commit_timestamp.isoformat() if self.commit_timestamp else None,
            'committed_by': self.committed_by
        }
    
    def __str__(self) -> str:
        return f"Prescription({self.status.value}: {self.get_dose_summary()})"
    
    def __repr__(self) -> str:
        return f"Prescription(id={self.prescription_id}, status={self.status.value})"
