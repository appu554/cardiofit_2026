"""
Application Commands for Medication Microservice
Implements the two-phase Propose/Commit operations
"""

from dataclasses import dataclass
from typing import Optional, Dict, Any
from uuid import UUID
from datetime import datetime

from ...domain.value_objects.dose_specification import Frequency, Duration


@dataclass
class ProposeMedicationCommand:
    """
    Command for Phase 1: Propose Medication (Stateless)
    
    This command triggers the "Calculate" phase of Calculate → Validate → Commit
    Contains all information needed for pharmaceutical intelligence calculations
    """
    
    # Patient and Medication
    patient_id: str
    medication_id: UUID
    prescriber_id: str
    
    # Clinical Context
    indication: str
    requested_frequency: Frequency
    requested_duration: Duration
    special_instructions: Optional[str] = None
    
    # Request Metadata
    command_type: str = "PROPOSE_MEDICATION"
    requested_at: datetime = None
    request_id: Optional[str] = None
    
    def __post_init__(self):
        """Validate command"""
        if not self.patient_id:
            raise ValueError("Patient ID is required")
        if not self.medication_id:
            raise ValueError("Medication ID is required")
        if not self.prescriber_id:
            raise ValueError("Prescriber ID is required")
        if not self.indication:
            raise ValueError("Indication is required")
        if not self.requested_frequency:
            raise ValueError("Requested frequency is required")
        if not self.requested_duration:
            raise ValueError("Requested duration is required")
        
        if self.requested_at is None:
            self.requested_at = datetime.utcnow()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for logging and serialization"""
        return {
            'command_type': self.command_type,
            'patient_id': self.patient_id,
            'medication_id': str(self.medication_id),
            'prescriber_id': self.prescriber_id,
            'indication': self.indication,
            'requested_frequency': self.requested_frequency.to_display_string(),
            'requested_duration': self.requested_duration.to_display_string(),
            'special_instructions': self.special_instructions,
            'requested_at': self.requested_at.isoformat(),
            'request_id': self.request_id
        }


@dataclass
class CommitPrescriptionCommand:
    """
    Command for Phase 2: Commit Prescription (Stateful)
    
    This command is called by the Workflow Engine after Safety Gateway validation
    Represents the "Commit" phase of Calculate → Validate → Commit
    """
    
    # Proposal Reference
    proposal_id: UUID
    
    # Commitment Context
    committed_by: str  # Workflow Engine identifier
    safety_validation_passed: bool = True
    
    # Optional Modifications from Safety Gateway
    dose_modifications: Optional[Dict[str, Any]] = None
    frequency_modifications: Optional[Dict[str, Any]] = None
    duration_modifications: Optional[Dict[str, Any]] = None
    quantity_modifications: Optional[Dict[str, Any]] = None
    
    # Commitment Metadata
    command_type: str = "COMMIT_PRESCRIPTION"
    committed_at: datetime = None
    validation_metadata: Optional[Dict[str, Any]] = None
    
    def __post_init__(self):
        """Validate command"""
        if not self.proposal_id:
            raise ValueError("Proposal ID is required")
        if not self.committed_by:
            raise ValueError("Committed by is required")
        if not self.safety_validation_passed:
            raise ValueError("Cannot commit prescription that failed safety validation")
        
        if self.committed_at is None:
            self.committed_at = datetime.utcnow()
    
    def has_modifications(self) -> bool:
        """Check if Safety Gateway applied any modifications"""
        return any([
            self.dose_modifications,
            self.frequency_modifications,
            self.duration_modifications,
            self.quantity_modifications
        ])
    
    def get_all_modifications(self) -> Dict[str, Any]:
        """Get all modifications as a single dictionary"""
        modifications = {}
        
        if self.dose_modifications:
            modifications['dose'] = self.dose_modifications
        if self.frequency_modifications:
            modifications['frequency'] = self.frequency_modifications
        if self.duration_modifications:
            modifications['duration'] = self.duration_modifications
        if self.quantity_modifications:
            modifications['quantity'] = self.quantity_modifications
        
        return modifications
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for logging and serialization"""
        return {
            'command_type': self.command_type,
            'proposal_id': str(self.proposal_id),
            'committed_by': self.committed_by,
            'safety_validation_passed': self.safety_validation_passed,
            'has_modifications': self.has_modifications(),
            'modifications': self.get_all_modifications(),
            'committed_at': self.committed_at.isoformat(),
            'validation_metadata': self.validation_metadata
        }


@dataclass
class CancelPrescriptionCommand:
    """
    Command to cancel a prescription proposal before commitment
    """
    
    # Prescription Reference
    prescription_id: Optional[UUID] = None
    proposal_id: Optional[UUID] = None
    
    # Cancellation Context
    cancelled_by: str = ""
    cancellation_reason: str = ""
    
    # Command Metadata
    command_type: str = "CANCEL_PRESCRIPTION"
    cancelled_at: datetime = None
    
    def __post_init__(self):
        """Validate command"""
        if not self.prescription_id and not self.proposal_id:
            raise ValueError("Either prescription_id or proposal_id is required")
        if not self.cancelled_by:
            raise ValueError("Cancelled by is required")
        if not self.cancellation_reason:
            raise ValueError("Cancellation reason is required")
        
        if self.cancelled_at is None:
            self.cancelled_at = datetime.utcnow()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for logging and serialization"""
        return {
            'command_type': self.command_type,
            'prescription_id': str(self.prescription_id) if self.prescription_id else None,
            'proposal_id': str(self.proposal_id) if self.proposal_id else None,
            'cancelled_by': self.cancelled_by,
            'cancellation_reason': self.cancellation_reason,
            'cancelled_at': self.cancelled_at.isoformat()
        }


@dataclass
class ProposeProtocolMedicationCommand:
    """
    Command for proposing medications as part of a clinical protocol
    Extends the basic propose medication for protocol-based therapy
    """
    
    # Basic Medication Proposal
    patient_id: str
    protocol_id: UUID
    protocol_day: int
    protocol_cycle: int
    prescriber_id: str
    
    # Protocol Context
    protocol_medications: list  # List of medications in the protocol
    dose_modifications: Optional[Dict[str, Any]] = None
    
    # Command Metadata
    command_type: str = "PROPOSE_PROTOCOL_MEDICATION"
    requested_at: datetime = None
    
    def __post_init__(self):
        """Validate command"""
        if not self.patient_id:
            raise ValueError("Patient ID is required")
        if not self.protocol_id:
            raise ValueError("Protocol ID is required")
        if not self.prescriber_id:
            raise ValueError("Prescriber ID is required")
        if not self.protocol_medications:
            raise ValueError("Protocol medications are required")
        if self.protocol_day < 1:
            raise ValueError("Protocol day must be positive")
        if self.protocol_cycle < 1:
            raise ValueError("Protocol cycle must be positive")
        
        if self.requested_at is None:
            self.requested_at = datetime.utcnow()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for logging and serialization"""
        return {
            'command_type': self.command_type,
            'patient_id': self.patient_id,
            'protocol_id': str(self.protocol_id),
            'protocol_day': self.protocol_day,
            'protocol_cycle': self.protocol_cycle,
            'prescriber_id': self.prescriber_id,
            'medication_count': len(self.protocol_medications),
            'has_dose_modifications': bool(self.dose_modifications),
            'requested_at': self.requested_at.isoformat()
        }


# Command Result Classes

@dataclass
class ProposeMedicationResult:
    """Result of medication proposal command"""
    
    success: bool
    proposal_id: Optional[UUID] = None
    error_message: Optional[str] = None
    warnings: list = None
    proposal_summary: Optional[str] = None
    confidence_score: Optional[float] = None
    expires_at: Optional[datetime] = None
    
    def __post_init__(self):
        if self.warnings is None:
            self.warnings = []
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'success': self.success,
            'proposal_id': str(self.proposal_id) if self.proposal_id else None,
            'error_message': self.error_message,
            'warnings': self.warnings,
            'proposal_summary': self.proposal_summary,
            'confidence_score': self.confidence_score,
            'expires_at': self.expires_at.isoformat() if self.expires_at else None
        }


@dataclass
class CommitPrescriptionResult:
    """Result of prescription commitment command"""
    
    success: bool
    prescription_id: Optional[UUID] = None
    error_message: Optional[str] = None
    modifications_applied: Optional[Dict[str, Any]] = None
    prescription_summary: Optional[str] = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization"""
        return {
            'success': self.success,
            'prescription_id': str(self.prescription_id) if self.prescription_id else None,
            'error_message': self.error_message,
            'modifications_applied': self.modifications_applied,
            'prescription_summary': self.prescription_summary
        }
