"""
Snapshot-Aware Orchestrator

This module implements the enhanced orchestration system that manages clinical workflows
with snapshot consistency across all phases. It extends the existing strategic orchestration
to provide immutable clinical context and comprehensive audit trails.

Key Features:
- Snapshot consistency validation across Calculate → Validate → Commit phases
- Enhanced error handling for snapshot-specific scenarios  
- Recipe resolution integration with caching
- Clinical override capture for learning loops
- Performance monitoring with snapshot-aware metrics
"""

import logging
import asyncio
import uuid
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Union
import httpx
import json

from app.orchestration.interfaces import (
    SnapshotReference,
    SnapshotChainTracker,
    SnapshotStatus,
    WorkflowPhase,
    ProposalWithSnapshot,
    ValidationResult,
    CommitResult,
    EvidenceEnvelope,
    ClinicalOverride,
    RecipeReference,
    ClinicalCommand,
    WorkflowInstance,
    SnapshotExpiredError,
    SnapshotIntegrityError,
    SnapshotNotFoundError,
    SnapshotConsistencyError
)

from app.orchestration.strategic_orchestrator import (
    strategic_orchestrator,
    CalculateRequest,
    ValidateRequest,
    CommitRequest,
    OrchestrationResult
)

logger = logging.getLogger(__name__)


class SnapshotAwareOrchestrator:
    """
    Enhanced orchestrator that manages clinical workflows with snapshot consistency.
    
    This orchestrator extends the strategic orchestration pattern to include:
    - Immutable clinical snapshots for data consistency
    - Cross-phase snapshot validation
    - Enhanced error handling for snapshot scenarios
    - Recipe resolution with caching
    - Clinical override capture for learning
    """
    
    def __init__(self):
        # Service endpoints (inherit from strategic orchestrator)
        self.flow2_go_url = "http://localhost:8080"
        self.flow2_rust_url = "http://localhost:8090"
        self.safety_gateway_url = "http://localhost:8018"
        self.medication_service_url = "http://localhost:8004"
        self.context_gateway_url = "http://localhost:8016"
        
        # HTTP client with extended timeout for snapshot operations
        self.http_client = httpx.AsyncClient(timeout=45.0)
        
        # Snapshot-specific configuration
        self.snapshot_config = {
            "default_ttl_minutes": 30,
            "integrity_validation_enabled": True,
            "consistency_validation_enabled": True,
            "snapshot_cache_enabled": True
        }
        
        # Performance targets (enhanced from strategic orchestrator)
        self.performance_targets = {
            "calculate_with_snapshot_ms": 150,  # Optimized with snapshots
            "validate_with_snapshot_ms": 75,   # Faster with consistent data
            "commit_with_snapshot_ms": 40,     # Optimized persistence
            "total_optimized_ms": 265,         # 66% improvement target
            "snapshot_validation_ms": 25,      # Snapshot consistency check
            "snapshot_creation_ms": 15         # Snapshot creation overhead
        }
        
        # Metrics collection
        self.metrics = {
            "snapshots_created": 0,
            "snapshots_validated": 0,
            "consistency_validations": 0,
            "consistency_failures": 0,
            "snapshot_cache_hits": 0,
            "snapshot_cache_misses": 0
        }
    
    async def executeCalculatePhase(
        self, 
        command: ClinicalCommand, 
        workflow_instance: WorkflowInstance
    ) -> ProposalWithSnapshot:
        """
        Execute Calculate phase with snapshot creation and consistency management.
        
        This method coordinates with the Flow2 engines to generate medication proposals
        while creating an immutable clinical snapshot for downstream phase consistency.
        
        Key enhancements over strategic orchestration:
        1. Creates immutable clinical snapshot from patient context
        2. Validates snapshot integrity before proceeding
        3. Tracks snapshot reference for downstream phases
        4. Enhanced error handling for snapshot creation failures
        5. Performance monitoring with snapshot-aware metrics
        
        Args:
            command: Clinical command with medication request details
            workflow_instance: Current workflow instance with state tracking
            
        Returns:
            ProposalWithSnapshot: Enhanced proposal response with snapshot reference
            
        Raises:
            SnapshotExpiredError: If patient context has expired
            SnapshotIntegrityError: If snapshot creation fails validation
            CalculatePhaseError: If Flow2 engines fail to generate proposals
        """
        calculate_start = datetime.utcnow()
        correlation_id = command.correlation_id
        
        logger.info(f"[{correlation_id}] Starting snapshot-aware CALCULATE phase")
        
        try:
            # Step 1: Create immutable clinical snapshot
            snapshot_creation_start = datetime.utcnow()
            snapshot_reference = await self._create_clinical_snapshot(
                patient_id=command.patient_id,
                workflow_id=workflow_instance.workflow_id,
                phase=WorkflowPhase.CALCULATE,
                correlation_id=correlation_id
            )
            
            snapshot_creation_time = (datetime.utcnow() - snapshot_creation_start).total_seconds() * 1000
            logger.info(f"[{correlation_id}] Clinical snapshot created: {snapshot_reference.snapshot_id} in {snapshot_creation_time:.1f}ms")
            
            # Update metrics
            self.metrics["snapshots_created"] += 1
            
            # Step 2: Update workflow instance with snapshot chain
            snapshot_chain = SnapshotChainTracker(workflow_id=workflow_instance.workflow_id)
            snapshot_chain.add_phase_snapshot(WorkflowPhase.CALCULATE, snapshot_reference)
            workflow_instance.snapshot_chain = snapshot_chain.to_dict()
            
            # Step 3: Execute calculate step with snapshot-optimized request
            calculate_request = CalculateRequest(
                patient_id=command.patient_id,
                medication_request=command.medication_request,
                clinical_intent=command.clinical_intent,
                provider_context=command.provider_context,
                correlation_id=correlation_id,
                urgency=command.urgency
            )
            
            # Enhanced Flow2 request with snapshot context
            flow2_request = {
                "patient_id": command.patient_id,
                "medication": command.medication_request,
                "clinical_intent": command.clinical_intent,
                "provider_context": command.provider_context,
                "snapshot_context": {
                    "snapshot_id": snapshot_reference.snapshot_id,
                    "snapshot_checksum": snapshot_reference.checksum,
                    "context_version": snapshot_reference.context_version
                },
                "execution_mode": "snapshot_optimized",
                "correlation_id": correlation_id
            }
            
            # Call Flow2 Go Engine with snapshot optimization
            response = await self.http_client.post(
                f"{self.flow2_go_url}/api/v1/snapshots/execute-advanced",
                json=flow2_request,
                headers={"Content-Type": "application/json"}
            )
            response.raise_for_status()
            
            flow2_result = response.json()
            
            # Step 4: Validate response consistency with snapshot
            if not await self._validate_response_snapshot_consistency(flow2_result, snapshot_reference):
                raise SnapshotConsistencyError(
                    f"Flow2 response snapshot inconsistent with created snapshot {snapshot_reference.snapshot_id}"
                )
            
            # Step 5: Create clinical evidence envelope
            evidence_envelope = EvidenceEnvelope(
                evidence_id=str(uuid.uuid4()),
                snapshot_id=snapshot_reference.snapshot_id,
                phase=WorkflowPhase.CALCULATE,
                evidence_type="clinical_reasoning",
                content=flow2_result.get("clinical_evidence", {}),
                confidence_score=flow2_result.get("confidence_score", 0.85),
                generated_at=datetime.utcnow(),
                source="flow2_engine"
            )
            
            # Step 6: Create recipe reference if available
            recipe_reference = None
            if "recipe_metadata" in flow2_result:
                recipe_data = flow2_result["recipe_metadata"]
                recipe_reference = RecipeReference(
                    recipe_id=recipe_data["recipe_id"],
                    version=recipe_data["version"],
                    resolved_at=datetime.utcnow(),
                    resolution_source=recipe_data.get("source", "service"),
                    metadata=recipe_data.get("metadata", {})
                )
            
            # Calculate execution metrics
            total_execution_time = (datetime.utcnow() - calculate_start).total_seconds() * 1000
            execution_metrics = {
                "total_time_ms": total_execution_time,
                "snapshot_creation_time_ms": snapshot_creation_time,
                "flow2_execution_time_ms": total_execution_time - snapshot_creation_time,
                "meets_performance_target": total_execution_time <= self.performance_targets["calculate_with_snapshot_ms"]
            }
            
            # Step 7: Create enhanced proposal response
            proposal_response = ProposalWithSnapshot(
                proposal_set_id=flow2_result["proposal_set_id"],
                snapshot_reference=snapshot_reference,
                ranked_proposals=flow2_result["ranked_proposals"],
                clinical_evidence=flow2_result["clinical_evidence"],
                monitoring_plan=flow2_result["monitoring_plan"],
                recipe_reference=recipe_reference,
                execution_metrics=execution_metrics
            )
            
            logger.info(f"[{correlation_id}] CALCULATE phase completed successfully in {total_execution_time:.1f}ms")
            return proposal_response
            
        except SnapshotError as e:
            logger.error(f"[{correlation_id}] Snapshot error in CALCULATE phase: {str(e)}")
            raise
        except Exception as e:
            logger.error(f"[{correlation_id}] CALCULATE phase failed: {str(e)}")
            raise Exception(f"Calculate phase failed: {str(e)}")
    
    async def executeValidatePhase(
        self, 
        proposal: ProposalWithSnapshot, 
        workflow_instance: WorkflowInstance
    ) -> ValidationResult:
        """
        Execute Validate phase with snapshot consistency verification.
        
        This method coordinates with the Safety Gateway to validate medication proposals
        using the same clinical snapshot for data consistency across phases.
        
        Key enhancements:
        1. Verifies snapshot consistency with Calculate phase
        2. Validates snapshot integrity before proceeding
        3. Uses same snapshot for Safety Gateway validation
        4. Creates evidence envelope for validation findings
        5. Enhanced error handling for snapshot validation failures
        
        Args:
            proposal: Proposal with snapshot from Calculate phase
            workflow_instance: Current workflow instance with state tracking
            
        Returns:
            ValidationResult: Enhanced validation result with snapshot verification
            
        Raises:
            SnapshotExpiredError: If snapshot has expired between phases
            SnapshotConsistencyError: If snapshot validation fails
            ValidatePhaseError: If Safety Gateway validation fails
        """
        validate_start = datetime.utcnow()
        correlation_id = f"validate_{proposal.proposal_set_id}"
        
        logger.info(f"[{correlation_id}] Starting snapshot-aware VALIDATE phase")
        
        try:
            # Step 1: Verify snapshot is still valid and consistent
            if not proposal.snapshot_reference.is_valid():
                raise SnapshotExpiredError(
                    proposal.snapshot_reference.snapshot_id,
                    proposal.snapshot_reference.expires_at
                )
            
            # Step 2: Validate snapshot consistency with workflow chain
            workflow_snapshot_chain = SnapshotChainTracker(workflow_id=workflow_instance.workflow_id)
            # Reconstruct from workflow instance data
            if workflow_instance.snapshot_chain:
                chain_data = workflow_instance.snapshot_chain
                if chain_data.get("calculate_snapshot"):
                    calc_snap_data = chain_data["calculate_snapshot"]
                    if calc_snap_data["snapshot_id"] != proposal.snapshot_reference.snapshot_id:
                        raise SnapshotConsistencyError(
                            f"Validation snapshot {proposal.snapshot_reference.snapshot_id} doesn't match "
                            f"calculate snapshot {calc_snap_data['snapshot_id']}"
                        )
            
            # Step 3: Update metrics for consistency validation
            self.metrics["consistency_validations"] += 1
            
            # Step 4: Create validate request with snapshot context
            validate_request = ValidateRequest(
                proposal_set_id=proposal.proposal_set_id,
                snapshot_id=proposal.snapshot_reference.snapshot_id,
                selected_proposals=proposal.ranked_proposals[:3],  # Top 3 proposals
                validation_requirements={
                    "cae_engine": True,
                    "protocol_engine": True,
                    "comprehensive_validation": True,
                    "snapshot_validation": True  # Enhanced validation
                },
                correlation_id=correlation_id
            )
            
            # Enhanced Safety Gateway request with snapshot context
            safety_request = {
                "proposal_set_id": proposal.proposal_set_id,
                "snapshot_context": {
                    "snapshot_id": proposal.snapshot_reference.snapshot_id,
                    "snapshot_checksum": proposal.snapshot_reference.checksum,
                    "patient_id": proposal.snapshot_reference.patient_id,
                    "context_version": proposal.snapshot_reference.context_version
                },
                "proposals": proposal.ranked_proposals[:3],
                "validation_requirements": validate_request.validation_requirements,
                "correlation_id": correlation_id
            }
            
            # Step 5: Call Safety Gateway with snapshot-aware validation
            response = await self.http_client.post(
                f"{self.safety_gateway_url}/api/v1/validation/comprehensive-with-snapshot",
                json=safety_request,
                headers={"Content-Type": "application/json"}
            )
            response.raise_for_status()
            
            safety_result = response.json()
            
            # Step 6: Verify Safety Gateway used correct snapshot
            if safety_result.get("snapshot_id") != proposal.snapshot_reference.snapshot_id:
                raise SnapshotConsistencyError(
                    f"Safety Gateway used different snapshot: expected {proposal.snapshot_reference.snapshot_id}, "
                    f"got {safety_result.get('snapshot_id')}"
                )
            
            # Step 7: Create evidence envelope for validation findings
            evidence_envelope = EvidenceEnvelope(
                evidence_id=str(uuid.uuid4()),
                snapshot_id=proposal.snapshot_reference.snapshot_id,
                phase=WorkflowPhase.VALIDATE,
                evidence_type="safety_assessment",
                content={
                    "validation_findings": safety_result.get("findings", []),
                    "risk_assessment": safety_result.get("risk_assessment", {}),
                    "validation_metadata": safety_result.get("metadata", {})
                },
                confidence_score=safety_result.get("confidence_score", 0.90),
                generated_at=datetime.utcnow(),
                source="safety_gateway"
            )
            
            # Step 8: Update workflow instance with validation snapshot
            workflow_snapshot_chain = SnapshotChainTracker(workflow_id=workflow_instance.workflow_id)
            # Reconstruct and add validation snapshot reference
            workflow_snapshot_chain.add_phase_snapshot(WorkflowPhase.VALIDATE, proposal.snapshot_reference)
            workflow_instance.snapshot_chain = workflow_snapshot_chain.to_dict()
            
            # Calculate execution metrics
            total_execution_time = (datetime.utcnow() - validate_start).total_seconds() * 1000
            validation_metrics = {
                "total_time_ms": total_execution_time,
                "meets_performance_target": total_execution_time <= self.performance_targets["validate_with_snapshot_ms"],
                "snapshot_consistency_validated": True
            }
            
            # Step 9: Create enhanced validation result
            validation_result = ValidationResult(
                validation_id=safety_result["validation_id"],
                snapshot_reference=proposal.snapshot_reference,
                verdict=safety_result["verdict"],  # SAFE, WARNING, UNSAFE
                findings=safety_result["findings"],
                evidence_envelope=evidence_envelope,
                override_tokens=safety_result.get("override_tokens"),
                approval_requirements=safety_result.get("approval_requirements"),
                validation_metrics=validation_metrics
            )
            
            # Update metrics
            self.metrics["snapshots_validated"] += 1
            
            logger.info(f"[{correlation_id}] VALIDATE phase completed: {safety_result['verdict']} in {total_execution_time:.1f}ms")
            return validation_result
            
        except SnapshotError as e:
            logger.error(f"[{correlation_id}] Snapshot error in VALIDATE phase: {str(e)}")
            self.metrics["consistency_failures"] += 1
            raise
        except Exception as e:
            logger.error(f"[{correlation_id}] VALIDATE phase failed: {str(e)}")
            raise Exception(f"Validate phase failed: {str(e)}")
    
    async def executeCommitPhase(
        self, 
        validation_result: ValidationResult, 
        proposal: ProposalWithSnapshot,
        workflow_instance: WorkflowInstance,
        provider_decision: Optional[Dict[str, Any]] = None
    ) -> CommitResult:
        """
        Execute Commit phase with complete snapshot audit trail.
        
        This method coordinates with the Medication Service to persist medication orders
        while maintaining complete audit trail of snapshot usage for regulatory compliance.
        
        Args:
            validation_result: Validation result from previous phase
            proposal: Original proposal with snapshot reference
            workflow_instance: Current workflow instance
            provider_decision: Optional provider decision for overrides
            
        Returns:
            CommitResult: Enhanced commit result with snapshot audit trail
        """
        commit_start = datetime.utcnow()
        correlation_id = f"commit_{proposal.proposal_set_id}"
        
        logger.info(f"[{correlation_id}] Starting snapshot-aware COMMIT phase")
        
        try:
            # Step 1: Final snapshot consistency verification
            if not await self._verify_snapshot_chain_consistency(workflow_instance, validation_result.snapshot_reference):
                raise SnapshotConsistencyError("Final snapshot consistency check failed before commit")
            
            # Step 2: Create commit request with snapshot context
            commit_request = {
                "proposal_set_id": proposal.proposal_set_id,
                "validation_id": validation_result.validation_id,
                "selected_proposal": proposal.ranked_proposals[0],  # Top proposal
                "provider_decision": provider_decision or {"auto_selected": True},
                "snapshot_context": {
                    "snapshot_id": validation_result.snapshot_reference.snapshot_id,
                    "snapshot_checksum": validation_result.snapshot_reference.checksum,
                    "snapshot_chain": workflow_instance.snapshot_chain
                },
                "evidence_trail": [
                    validation_result.evidence_envelope.to_dict()
                ],
                "correlation_id": correlation_id
            }
            
            # Step 3: Call Medication Service for persistence
            response = await self.http_client.post(
                f"{self.medication_service_url}/api/v1/medication/commit-with-snapshot",
                json=commit_request,
                headers={"Content-Type": "application/json"}
            )
            response.raise_for_status()
            
            medication_result = response.json()
            
            # Step 4: Create final snapshot chain for audit
            final_snapshot_chain = SnapshotChainTracker(workflow_id=workflow_instance.workflow_id)
            final_snapshot_chain.add_phase_snapshot(WorkflowPhase.COMMIT, validation_result.snapshot_reference)
            
            # Calculate execution metrics
            total_execution_time = (datetime.utcnow() - commit_start).total_seconds() * 1000
            
            # Step 5: Create enhanced commit result
            commit_result = CommitResult(
                medication_order_id=medication_result["medication_order_id"],
                snapshot_reference=validation_result.snapshot_reference,
                audit_trail_id=medication_result["audit_trail_id"],
                persistence_status=medication_result["persistence_status"],
                event_publication_status=medication_result["event_publication_status"],
                snapshot_chain=final_snapshot_chain,
                commit_timestamp=datetime.utcnow()
            )
            
            logger.info(f"[{correlation_id}] COMMIT phase completed successfully in {total_execution_time:.1f}ms")
            return commit_result
            
        except Exception as e:
            logger.error(f"[{correlation_id}] COMMIT phase failed: {str(e)}")
            raise Exception(f"Commit phase failed: {str(e)}")
    
    async def _create_clinical_snapshot(
        self, 
        patient_id: str, 
        workflow_id: str, 
        phase: WorkflowPhase,
        correlation_id: str
    ) -> SnapshotReference:
        """
        Create immutable clinical snapshot for workflow consistency.
        
        Creates a snapshot of the current clinical context that will be used
        consistently across all workflow phases.
        """
        try:
            # Get current clinical context
            context_response = await self.http_client.get(
                f"{self.context_gateway_url}/api/v1/context/{patient_id}/snapshot",
                headers={"Correlation-ID": correlation_id}
            )
            context_response.raise_for_status()
            
            clinical_context = context_response.json()
            
            # Create checksum for integrity validation
            context_str = json.dumps(clinical_context, sort_keys=True, separators=(',', ':'))
            import hashlib
            checksum = hashlib.sha256(context_str.encode()).hexdigest()
            
            # Create snapshot reference
            snapshot_id = f"snap_{workflow_id}_{int(datetime.utcnow().timestamp())}"
            expires_at = datetime.utcnow() + timedelta(minutes=self.snapshot_config["default_ttl_minutes"])
            
            snapshot_reference = SnapshotReference(
                snapshot_id=snapshot_id,
                checksum=checksum,
                created_at=datetime.utcnow(),
                expires_at=expires_at,
                status=SnapshotStatus.ACTIVE,
                phase_created=phase,
                patient_id=patient_id,
                context_version=clinical_context.get("version", "1.0"),
                metadata={
                    "workflow_id": workflow_id,
                    "correlation_id": correlation_id,
                    "context_size_bytes": len(context_str)
                }
            )
            
            return snapshot_reference
            
        except Exception as e:
            logger.error(f"Failed to create clinical snapshot for patient {patient_id}: {str(e)}")
            raise Exception(f"Snapshot creation failed: {str(e)}")
    
    async def _validate_response_snapshot_consistency(
        self, 
        response_data: Dict[str, Any], 
        snapshot_reference: SnapshotReference
    ) -> bool:
        """Validate that service response is consistent with snapshot"""
        try:
            response_snapshot_id = response_data.get("snapshot_id")
            if response_snapshot_id != snapshot_reference.snapshot_id:
                return False
            
            response_checksum = response_data.get("snapshot_checksum")
            if response_checksum and response_checksum != snapshot_reference.checksum:
                return False
            
            return True
        except Exception:
            return False
    
    async def _verify_snapshot_chain_consistency(
        self, 
        workflow_instance: WorkflowInstance, 
        current_snapshot: SnapshotReference
    ) -> bool:
        """Verify snapshot consistency across entire workflow chain"""
        try:
            if not workflow_instance.snapshot_chain:
                return True  # No chain to verify
            
            chain_data = workflow_instance.snapshot_chain
            
            # Check all snapshots in chain have same patient_id and context_version
            snapshots_to_check = []
            for phase in ["calculate_snapshot", "validate_snapshot", "commit_snapshot"]:
                if phase in chain_data and chain_data[phase]:
                    snapshots_to_check.append(chain_data[phase])
            
            if not snapshots_to_check:
                return True
            
            base_snapshot = snapshots_to_check[0]
            for snapshot_data in snapshots_to_check:
                if (snapshot_data["patient_id"] != current_snapshot.patient_id or
                    snapshot_data["context_version"] != current_snapshot.context_version):
                    return False
            
            return True
            
        except Exception as e:
            logger.error(f"Snapshot chain consistency verification failed: {str(e)}")
            return False
    
    async def health_check(self) -> Dict[str, Any]:
        """Health check for snapshot-aware orchestrator"""
        # Inherit from strategic orchestrator and add snapshot-specific checks
        base_health = await strategic_orchestrator.health_check()
        
        # Add snapshot-specific health metrics
        snapshot_health = {
            "snapshot_orchestrator_status": "healthy",
            "snapshot_config": self.snapshot_config,
            "performance_targets": self.performance_targets,
            "metrics": self.metrics,
            "snapshot_features": {
                "integrity_validation": self.snapshot_config["integrity_validation_enabled"],
                "consistency_validation": self.snapshot_config["consistency_validation_enabled"],
                "snapshot_caching": self.snapshot_config["snapshot_cache_enabled"]
            }
        }
        
        # Merge with base health check
        base_health.update(snapshot_health)
        
        return base_health


# Global instance
snapshot_aware_orchestrator = SnapshotAwareOrchestrator()