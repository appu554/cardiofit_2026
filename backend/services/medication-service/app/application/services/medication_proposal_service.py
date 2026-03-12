"""
Medication Proposal Service - Core Application Service
Implements the two-phase Propose/Commit operations with Business Context Recipe pattern
"""

import logging
from typing import Optional, Dict, Any
from uuid import UUID
from datetime import datetime, timedelta

from ...domain.entities.medication import Medication
from ...domain.entities.prescription import PrescriptionProposal, Prescription
from ...domain.value_objects.dose_specification import DoseCalculationContext
from ...domain.services.business_context_recipe_book import BusinessContextRecipeBook
from ..commands.medication_commands import (
    ProposeMedicationCommand, CommitPrescriptionCommand, CancelPrescriptionCommand,
    ProposeMedicationResult, CommitPrescriptionResult
)

logger = logging.getLogger(__name__)


class MedicationProposalService:
    """
    Core Application Service implementing the Four Pillars of Excellence
    
    Pillar 1: Pure Domain Expert - Focuses only on pharmaceutical intelligence
    Pillar 2: Two-Phase Operations - Implements Propose/Commit pattern
    Pillar 3: Deep Clinical Intelligence - Advanced dose calculations and protocols
    Pillar 4: Clear Integration Contracts - Business Context Recipe pattern
    """
    
    def __init__(
        self,
        medication_repository,
        prescription_repository,
        context_service_client,
        recipe_book: BusinessContextRecipeBook,
        cache_manager,
        event_publisher
    ):
        self.medication_repository = medication_repository
        self.prescription_repository = prescription_repository
        self.context_service_client = context_service_client
        self.recipe_book = recipe_book
        self.cache_manager = cache_manager
        self.event_publisher = event_publisher
    
    async def propose_medication(
        self, 
        command: ProposeMedicationCommand
    ) -> ProposeMedicationResult:
        """
        Phase 1: Propose Medication (Stateless Operation)
        
        Implements the "Calculate" phase of Calculate → Validate → Commit
        This is a pure function with zero side effects
        """
        try:
            logger.info(f"Starting medication proposal for patient {command.patient_id}")
            
            # 1. LOAD MEDICATION (Domain Expert Knowledge)
            medication = await self.medication_repository.get_by_id(command.medication_id)
            if not medication:
                return ProposeMedicationResult(
                    success=False,
                    error_message=f"Medication not found: {command.medication_id}"
                )
            
            # 2. SELECT RECIPE (Business Context Recipe Pattern)
            recipe = self.recipe_book.select_recipe_for(command)
            logger.info(f"Selected recipe: {recipe.id} for medication proposal")
            
            # 3. FETCH BUSINESS CONTEXT (Single, Targeted Call)
            business_context = await self._fetch_business_context(
                command.patient_id, 
                recipe,
                command.medication_id
            )
            
            # 4. VALIDATE CONTEXT QUALITY
            validation_result = self.recipe_book.validate_context(recipe, business_context)
            if not validation_result['valid']:
                return ProposeMedicationResult(
                    success=False,
                    error_message=f"Invalid business context: {validation_result['errors']}",
                    warnings=validation_result['warnings']
                )
            
            # 5. CREATE DOSE CALCULATION CONTEXT
            dose_context = self._create_dose_calculation_context(
                command.patient_id,
                business_context
            )
            
            # 6. EXECUTE PHARMACEUTICAL INTELLIGENCE (Pure Domain Logic)
            dose_proposal = medication.calculate_dose(
                context=dose_context,
                indication=command.indication,
                frequency=command.requested_frequency,
                duration=command.requested_duration
            )
            
            # 7. CREATE PRESCRIPTION PROPOSAL (No Persistence - Stateless)
            proposal = PrescriptionProposal(
                patient_id=command.patient_id,
                medication_id=command.medication_id,
                dose_proposal=dose_proposal,
                prescriber_id=command.prescriber_id,
                indication=command.indication,
                special_instructions=command.special_instructions,
                business_context=business_context,
                calculation_metadata={
                    'recipe_id': recipe.id,
                    'recipe_version': recipe.version,
                    'calculation_timestamp': datetime.utcnow().isoformat(),
                    'context_validation': validation_result
                },
                expires_at=datetime.utcnow() + timedelta(hours=1),  # 1 hour expiry
                recipe_id=recipe.id
            )
            
            # 8. CACHE PROPOSAL (For later commitment)
            await self._cache_proposal(proposal, recipe)
            
            logger.info(f"Medication proposal created successfully: {proposal.proposal_id}")
            
            return ProposeMedicationResult(
                success=True,
                proposal_id=proposal.proposal_id,
                warnings=dose_proposal.warnings,
                proposal_summary=proposal.get_proposed_dose_summary(),
                confidence_score=float(dose_proposal.confidence_score),
                expires_at=proposal.expires_at
            )
            
        except Exception as e:
            logger.error(f"Error in medication proposal: {str(e)}", exc_info=True)
            return ProposeMedicationResult(
                success=False,
                error_message=f"Proposal failed: {str(e)}"
            )
    
    async def commit_prescription(
        self, 
        command: CommitPrescriptionCommand
    ) -> CommitPrescriptionResult:
        """
        Phase 2: Commit Prescription (Stateful Operation)
        
        Implements the "Commit" phase after Safety Gateway validation
        This is an idempotent operation with side effects (persistence + events)
        """
        try:
            logger.info(f"Starting prescription commitment for proposal {command.proposal_id}")
            
            # 1. RETRIEVE CACHED PROPOSAL
            proposal = await self._get_cached_proposal(command.proposal_id)
            if not proposal:
                return CommitPrescriptionResult(
                    success=False,
                    error_message=f"Proposal not found or expired: {command.proposal_id}"
                )
            
            # 2. CHECK PROPOSAL EXPIRY
            if proposal.is_expired():
                return CommitPrescriptionResult(
                    success=False,
                    error_message=f"Proposal has expired: {command.proposal_id}"
                )
            
            # 3. CREATE COMMITTED PRESCRIPTION
            prescription = Prescription.from_proposal(
                proposal=proposal,
                committed_by=command.committed_by,
                modifications=command.get_all_modifications()
            )
            
            # 4. PERSIST PRESCRIPTION (Stateful Operation)
            await self.prescription_repository.save(prescription)
            
            # 5. PUBLISH EVENTS (Outbox Pattern)
            await self._publish_prescription_events(prescription, proposal)
            
            # 6. CLEANUP CACHED PROPOSAL
            await self._cleanup_proposal_cache(command.proposal_id)
            
            logger.info(f"Prescription committed successfully: {prescription.prescription_id}")
            
            return CommitPrescriptionResult(
                success=True,
                prescription_id=prescription.prescription_id,
                modifications_applied=command.get_all_modifications(),
                prescription_summary=prescription.get_dose_summary()
            )
            
        except Exception as e:
            logger.error(f"Error in prescription commitment: {str(e)}", exc_info=True)
            return CommitPrescriptionResult(
                success=False,
                error_message=f"Commitment failed: {str(e)}"
            )
    
    async def cancel_prescription(
        self, 
        command: CancelPrescriptionCommand
    ) -> CommitPrescriptionResult:
        """Cancel a prescription proposal or committed prescription"""
        try:
            if command.proposal_id:
                # Cancel cached proposal
                await self._cleanup_proposal_cache(command.proposal_id)
                logger.info(f"Proposal cancelled: {command.proposal_id}")
            
            if command.prescription_id:
                # Cancel committed prescription
                prescription = await self.prescription_repository.get_by_id(command.prescription_id)
                if prescription:
                    prescription.cancel(command.cancellation_reason, command.cancelled_by)
                    await self.prescription_repository.save(prescription)
                    await self._publish_cancellation_event(prescription)
                    logger.info(f"Prescription cancelled: {command.prescription_id}")
            
            return CommitPrescriptionResult(success=True)
            
        except Exception as e:
            logger.error(f"Error in prescription cancellation: {str(e)}", exc_info=True)
            return CommitPrescriptionResult(
                success=False,
                error_message=f"Cancellation failed: {str(e)}"
            )
    
    # === PRIVATE HELPER METHODS ===
    
    async def _fetch_business_context(
        self, 
        patient_id: str, 
        recipe, 
        medication_id: UUID
    ) -> Dict[str, Any]:
        """
        Fetch business context using the Business Context Recipe pattern
        Makes a single, targeted GraphQL call to Context Service
        """
        # Check cache first
        cache_key = self.recipe_book.get_cache_key(
            recipe, 
            patientId=patient_id, 
            medicationId=str(medication_id)
        )
        
        cached_context = await self.cache_manager.get(cache_key)
        if cached_context:
            logger.debug(f"Using cached business context for {cache_key}")
            return cached_context
        
        # Fetch from Context Service
        logger.debug(f"Fetching business context using recipe {recipe.id}")
        context = await self.context_service_client.get_context(
            patient_id=patient_id,
            query=recipe.context_requirements.query
        )
        
        # Cache the result
        await self.cache_manager.set(
            cache_key, 
            context, 
            ttl=recipe.cache_settings.ttl
        )
        
        return context
    
    def _create_dose_calculation_context(
        self, 
        patient_id: str, 
        business_context: Dict[str, Any]
    ) -> DoseCalculationContext:
        """Create dose calculation context from business context"""
        patient_data = business_context.get('patient', {})
        demographics = patient_data.get('demographics', {})
        vitals = patient_data.get('vitals', {})
        labs = patient_data.get('labs', {})
        
        # Extract latest vitals if available
        latest_vitals = vitals.get('latest', {}) if vitals else {}
        
        # Extract lab values
        latest_labs = {}
        if labs and isinstance(labs, dict) and 'latest' in labs:
            for lab in labs.get('latest', []):
                if lab.get('code') == 'LOINC:33914-3':  # eGFR
                    latest_labs['egfr'] = lab.get('value')
                elif lab.get('code') == 'LOINC:2160-0':  # Creatinine
                    latest_labs['creatinine'] = lab.get('value')
        
        return DoseCalculationContext(
            patient_id=patient_id,
            weight_kg=demographics.get('weightKg') or latest_vitals.get('weightKg'),
            height_cm=demographics.get('heightCm') or latest_vitals.get('heightCm'),
            age_years=demographics.get('ageYears'),
            age_months=demographics.get('ageMonths'),
            creatinine_clearance=latest_labs.get('creatinine_clearance'),
            egfr=latest_labs.get('egfr'),
            liver_function=patient_data.get('conditions', {}).get('liver_function'),
            pregnancy_status=demographics.get('pregnancyStatus'),
            breastfeeding_status=demographics.get('breastfeedingStatus')
        )
    
    async def _cache_proposal(self, proposal: PrescriptionProposal, recipe):
        """Cache proposal for later commitment"""
        cache_key = f"proposal:{proposal.proposal_id}"
        await self.cache_manager.set(
            cache_key, 
            proposal.to_dict(), 
            ttl=3600  # 1 hour
        )
    
    async def _get_cached_proposal(self, proposal_id: UUID) -> Optional[PrescriptionProposal]:
        """Retrieve cached proposal"""
        cache_key = f"proposal:{proposal_id}"
        cached_data = await self.cache_manager.get(cache_key)
        
        if not cached_data:
            return None
        
        # Reconstruct proposal from cached data
        # This is simplified - in production, you'd want proper serialization
        return cached_data  # Placeholder for proper deserialization
    
    async def _cleanup_proposal_cache(self, proposal_id: UUID):
        """Clean up cached proposal"""
        cache_key = f"proposal:{proposal_id}"
        await self.cache_manager.delete(cache_key)
    
    async def _publish_prescription_events(
        self, 
        prescription: Prescription, 
        proposal: PrescriptionProposal
    ):
        """Publish prescription events using outbox pattern"""
        # Publish MedicationCommitted event
        await self.event_publisher.publish_event(
            aggregate_id=prescription.prescription_id,
            aggregate_type="Prescription",
            event_type="MedicationCommitted",
            event_data={
                'prescription_id': str(prescription.prescription_id),
                'proposal_id': str(prescription.proposal_id),
                'patient_id': prescription.patient_id,
                'medication_id': str(prescription.medication_id),
                'prescriber_id': prescription.prescriber_id,
                'dose_summary': prescription.get_dose_summary(),
                'committed_by': prescription.committed_by,
                'committed_at': prescription.commit_timestamp.isoformat()
            }
        )
    
    async def _publish_cancellation_event(self, prescription: Prescription):
        """Publish prescription cancellation event"""
        await self.event_publisher.publish_event(
            aggregate_id=prescription.prescription_id,
            aggregate_type="Prescription",
            event_type="MedicationCancelled",
            event_data={
                'prescription_id': str(prescription.prescription_id),
                'patient_id': prescription.patient_id,
                'cancellation_reason': prescription.commit_metadata.get('cancellation_reason'),
                'cancelled_by': prescription.commit_metadata.get('cancelled_by'),
                'cancelled_at': prescription.commit_metadata.get('cancelled_at')
            }
        )
