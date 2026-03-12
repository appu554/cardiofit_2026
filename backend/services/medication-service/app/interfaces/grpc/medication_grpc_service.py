"""
gRPC Service Implementation for Medication Microservice
Implements the two-phase Propose/Commit operations
"""

import logging
from typing import Dict, Any
import grpc
from uuid import UUID
from datetime import datetime

# Import generated gRPC classes (would be generated from .proto file)
# from . import medication_service_pb2
# from . import medication_service_pb2_grpc

from ...application.services.medication_proposal_service import MedicationProposalService
from ...application.commands.medication_commands import (
    ProposeMedicationCommand, CommitPrescriptionCommand, CancelPrescriptionCommand
)
from ...domain.value_objects.dose_specification import Frequency, Duration, FrequencyType

logger = logging.getLogger(__name__)


class MedicationGrpcService:
    """
    gRPC Service implementing the Four Pillars of Excellence
    
    Provides the two-phase interface: ProposeMedication and CommitPrescription
    This is the primary interface for the Workflow Engine
    """
    
    def __init__(self, medication_proposal_service: MedicationProposalService):
        self.medication_proposal_service = medication_proposal_service
    
    async def ProposeMedication(self, request, context):
        """
        Phase 1: Propose Medication (Stateless)
        
        This is the "Calculate" phase of Calculate → Validate → Commit
        Pure function with zero side effects - only pharmaceutical intelligence
        """
        try:
            logger.info(f"gRPC ProposeMedication called for patient {request.patient_id}")
            
            # Convert gRPC request to domain command
            command = self._convert_to_propose_command(request)
            
            # Execute pharmaceutical intelligence
            result = await self.medication_proposal_service.propose_medication(command)
            
            # Convert result to gRPC response
            response = self._convert_to_propose_response(result)
            
            logger.info(f"ProposeMedication completed: success={result.success}")
            return response
            
        except Exception as e:
            logger.error(f"Error in ProposeMedication: {str(e)}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return self._create_error_propose_response(str(e))
    
    async def CommitPrescription(self, request, context):
        """
        Phase 2: Commit Prescription (Stateful)
        
        This is the "Commit" phase called by Workflow Engine after Safety Gateway validation
        Idempotent operation with side effects (persistence + events)
        """
        try:
            logger.info(f"gRPC CommitPrescription called for proposal {request.proposal_id}")
            
            # Convert gRPC request to domain command
            command = self._convert_to_commit_command(request)
            
            # Execute commitment
            result = await self.medication_proposal_service.commit_prescription(command)
            
            # Convert result to gRPC response
            response = self._convert_to_commit_response(result)
            
            logger.info(f"CommitPrescription completed: success={result.success}")
            return response
            
        except Exception as e:
            logger.error(f"Error in CommitPrescription: {str(e)}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return self._create_error_commit_response(str(e))
    
    async def CancelPrescription(self, request, context):
        """Cancel a prescription proposal or committed prescription"""
        try:
            logger.info(f"gRPC CancelPrescription called")
            
            # Convert gRPC request to domain command
            command = self._convert_to_cancel_command(request)
            
            # Execute cancellation
            result = await self.medication_proposal_service.cancel_prescription(command)
            
            # Convert result to gRPC response
            response = self._convert_to_cancel_response(result)
            
            logger.info(f"CancelPrescription completed: success={result.success}")
            return response
            
        except Exception as e:
            logger.error(f"Error in CancelPrescription: {str(e)}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return self._create_error_cancel_response(str(e))
    
    # === CONVERSION METHODS ===
    
    def _convert_to_propose_command(self, request) -> ProposeMedicationCommand:
        """Convert gRPC request to domain command"""
        
        # Convert frequency
        frequency = self._convert_grpc_frequency(request.requested_frequency)
        
        # Convert duration
        duration = self._convert_grpc_duration(request.requested_duration)
        
        return ProposeMedicationCommand(
            patient_id=request.patient_id,
            medication_id=UUID(request.medication_id),
            prescriber_id=request.prescriber_id,
            indication=request.indication,
            requested_frequency=frequency,
            requested_duration=duration,
            special_instructions=request.special_instructions or None,
            request_id=request.request_id or None,
            requested_at=self._convert_grpc_timestamp(request.requested_at) if request.requested_at else None
        )
    
    def _convert_to_commit_command(self, request) -> CommitPrescriptionCommand:
        """Convert gRPC commit request to domain command"""
        
        # Extract modifications
        modifications = {}
        if request.dose_modifications:
            modifications['dose'] = {
                'new_value': request.dose_modifications.new_value,
                'new_unit': request.dose_modifications.new_unit,
                'reason': request.dose_modifications.modification_reason,
                'modified_by': request.dose_modifications.modified_by
            }
        
        if request.frequency_modifications:
            modifications['frequency'] = {
                'new_frequency': self._convert_grpc_frequency(request.frequency_modifications.new_frequency),
                'reason': request.frequency_modifications.modification_reason,
                'modified_by': request.frequency_modifications.modified_by
            }
        
        return CommitPrescriptionCommand(
            proposal_id=UUID(request.proposal_id),
            committed_by=request.committed_by,
            safety_validation_passed=request.safety_validation_passed,
            dose_modifications=modifications.get('dose'),
            frequency_modifications=modifications.get('frequency'),
            duration_modifications=modifications.get('duration'),
            quantity_modifications=modifications.get('quantity'),
            committed_at=self._convert_grpc_timestamp(request.committed_at) if request.committed_at else None,
            validation_metadata=self._convert_validation_metadata(request.validation_metadata) if request.validation_metadata else None
        )
    
    def _convert_to_cancel_command(self, request) -> CancelPrescriptionCommand:
        """Convert gRPC cancel request to domain command"""
        
        prescription_id = None
        proposal_id = None
        
        if request.WhichOneof('identifier') == 'prescription_id':
            prescription_id = UUID(request.prescription_id)
        elif request.WhichOneof('identifier') == 'proposal_id':
            proposal_id = UUID(request.proposal_id)
        
        return CancelPrescriptionCommand(
            prescription_id=prescription_id,
            proposal_id=proposal_id,
            cancelled_by=request.cancelled_by,
            cancellation_reason=request.cancellation_reason,
            cancelled_at=self._convert_grpc_timestamp(request.cancelled_at) if request.cancelled_at else None
        )
    
    def _convert_grpc_frequency(self, grpc_freq) -> Frequency:
        """Convert gRPC frequency to domain frequency"""
        if not grpc_freq:
            return None
        
        freq_type = FrequencyType(grpc_freq.type.lower())
        
        return Frequency(
            type=freq_type,
            interval_hours=grpc_freq.interval_hours if grpc_freq.interval_hours > 0 else None,
            times_per_day=grpc_freq.times_per_day if grpc_freq.times_per_day > 0 else None,
            specific_times=list(grpc_freq.specific_times) if grpc_freq.specific_times else None,
            max_doses_per_day=grpc_freq.max_doses_per_day if grpc_freq.max_doses_per_day > 0 else None,
            conditions=grpc_freq.conditions if grpc_freq.conditions else None
        )
    
    def _convert_grpc_duration(self, grpc_dur) -> Duration:
        """Convert gRPC duration to domain duration"""
        if not grpc_dur:
            return None
        
        return Duration(
            days=grpc_dur.days if grpc_dur.days > 0 else None,
            weeks=grpc_dur.weeks if grpc_dur.weeks > 0 else None,
            months=grpc_dur.months if grpc_dur.months > 0 else None,
            indefinite=grpc_dur.indefinite,
            until_condition=grpc_dur.until_condition if grpc_dur.until_condition else None
        )
    
    def _convert_grpc_timestamp(self, grpc_timestamp) -> datetime:
        """Convert gRPC timestamp to Python datetime"""
        if not grpc_timestamp:
            return None
        return datetime.fromtimestamp(grpc_timestamp.seconds + grpc_timestamp.nanos / 1e9)
    
    def _convert_validation_metadata(self, grpc_metadata) -> Dict[str, Any]:
        """Convert gRPC validation metadata to dictionary"""
        if not grpc_metadata:
            return {}
        
        return {
            'drug_interaction_checked': grpc_metadata.drug_interaction_checked,
            'allergy_checked': grpc_metadata.allergy_checked,
            'contraindication_checked': grpc_metadata.contraindication_checked,
            'dose_limit_checked': grpc_metadata.dose_limit_checked,
            'safety_warnings': list(grpc_metadata.safety_warnings),
            'safety_overrides': list(grpc_metadata.safety_overrides),
            'validation_engine_version': grpc_metadata.validation_engine_version,
            'validated_at': self._convert_grpc_timestamp(grpc_metadata.validated_at)
        }
    
    # === RESPONSE CONVERSION METHODS ===
    
    def _convert_to_propose_response(self, result):
        """Convert domain result to gRPC response"""
        # This would use the generated protobuf classes
        # Simplified implementation for demonstration
        
        response_data = {
            'success': result.success,
            'proposal_id': str(result.proposal_id) if result.proposal_id else '',
            'error_message': result.error_message or '',
            'warnings': result.warnings or [],
            'proposal_summary': result.proposal_summary or '',
            'confidence_score': result.confidence_score or 0.0,
            'expires_at': result.expires_at.isoformat() if result.expires_at else ''
        }
        
        # In real implementation, this would create the protobuf response object
        logger.debug(f"Propose response: {response_data}")
        return response_data  # Placeholder
    
    def _convert_to_commit_response(self, result):
        """Convert domain result to gRPC response"""
        response_data = {
            'success': result.success,
            'prescription_id': str(result.prescription_id) if result.prescription_id else '',
            'error_message': result.error_message or '',
            'prescription_summary': result.prescription_summary or '',
            'modifications_applied': bool(result.modifications_applied),
            'modification_summary': result.modifications_applied or {}
        }
        
        logger.debug(f"Commit response: {response_data}")
        return response_data  # Placeholder
    
    def _convert_to_cancel_response(self, result):
        """Convert domain result to gRPC response"""
        response_data = {
            'success': result.success,
            'error_message': result.error_message or '',
            'cancelled_at': datetime.utcnow().isoformat()
        }
        
        logger.debug(f"Cancel response: {response_data}")
        return response_data  # Placeholder
    
    # === ERROR RESPONSE METHODS ===
    
    def _create_error_propose_response(self, error_message: str):
        """Create error response for propose operation"""
        return {
            'success': False,
            'proposal_id': '',
            'error_message': error_message,
            'warnings': [],
            'proposal_summary': '',
            'confidence_score': 0.0,
            'expires_at': ''
        }
    
    def _create_error_commit_response(self, error_message: str):
        """Create error response for commit operation"""
        return {
            'success': False,
            'prescription_id': '',
            'error_message': error_message,
            'prescription_summary': '',
            'modifications_applied': False,
            'modification_summary': {}
        }
    
    def _create_error_cancel_response(self, error_message: str):
        """Create error response for cancel operation"""
        return {
            'success': False,
            'error_message': error_message,
            'cancelled_at': datetime.utcnow().isoformat()
        }
