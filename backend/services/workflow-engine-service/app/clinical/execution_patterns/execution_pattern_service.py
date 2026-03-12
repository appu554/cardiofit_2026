"""
Clinical Workflow Execution Pattern Service.
Implements the three clinical workflow execution patterns: Pessimistic, Optimistic, and Digital Reflex Arc.
"""
import logging
import asyncio
import time
from enum import Enum
from dataclasses import dataclass
from typing import Dict, List, Optional, Any, Callable
from datetime import datetime, timedelta

from app.models.clinical_activity_models import ClinicalContext, ClinicalError, ClinicalErrorType

logger = logging.getLogger(__name__)


class ExecutionPattern(Enum):
    """Clinical workflow execution patterns."""
    PESSIMISTIC = "pessimistic"          # High-risk workflows
    OPTIMISTIC = "optimistic"            # Low-risk workflows  
    DIGITAL_REFLEX_ARC = "digital_reflex_arc"  # Autonomous workflows


@dataclass
class PatternConfiguration:
    """Configuration for a clinical workflow execution pattern."""
    pattern: ExecutionPattern
    execution_flow: str  # synchronous, asynchronous, autonomous
    safety_validation: str  # mandatory_before_commit, parallel_with_commit, real_time_continuous
    user_feedback: str  # wait_for_completion, immediate_optimistic, notification_only
    sla_budget_ms: int  # milliseconds
    compensation: Optional[str] = None  # automatic_if_unsafe, manual_intervention
    human_intervention: Optional[str] = None  # exception_based, always_required
    example_workflows: List[str] = None


class ClinicalExecutionPatternService:
    """
    Service for managing clinical workflow execution patterns.
    Implements risk-appropriate execution strategies.
    """
    
    def __init__(self):
        self.patterns = self._initialize_patterns()
        self.active_executions = {}
        
    def _initialize_patterns(self) -> Dict[ExecutionPattern, PatternConfiguration]:
        """Initialize the three clinical execution patterns."""
        return {
            ExecutionPattern.PESSIMISTIC: PatternConfiguration(
                pattern=ExecutionPattern.PESSIMISTIC,
                execution_flow="synchronous",
                safety_validation="mandatory_before_commit",
                user_feedback="wait_for_completion",
                sla_budget_ms=250,
                example_workflows=[
                    "medication_prescribing",
                    "high_alert_medication_orders",
                    "patient_discharge_decisions",
                    "controlled_substance_prescribing",
                    "chemotherapy_orders"
                ]
            ),
            ExecutionPattern.OPTIMISTIC: PatternConfiguration(
                pattern=ExecutionPattern.OPTIMISTIC,
                execution_flow="asynchronous",
                safety_validation="parallel_with_commit",
                user_feedback="immediate_optimistic",
                sla_budget_ms=150,
                compensation="automatic_if_unsafe",
                example_workflows=[
                    "routine_medication_refill",
                    "standard_lab_orders",
                    "clinical_documentation",
                    "appointment_scheduling",
                    "routine_vitals_entry"
                ]
            ),
            ExecutionPattern.DIGITAL_REFLEX_ARC: PatternConfiguration(
                pattern=ExecutionPattern.DIGITAL_REFLEX_ARC,
                execution_flow="autonomous",
                safety_validation="real_time_continuous",
                user_feedback="notification_only",
                sla_budget_ms=100,
                human_intervention="exception_based",
                example_workflows=[
                    "clinical_deterioration_response",
                    "critical_value_alerts",
                    "sepsis_protocol_activation",
                    "cardiac_arrest_response",
                    "anaphylaxis_protocol"
                ]
            )
        }
    
    async def execute_workflow_with_pattern(
        self,
        workflow_type: str,
        pattern: ExecutionPattern,
        workflow_data: Dict[str, Any],
        execution_context: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute a workflow using the specified execution pattern.
        
        Args:
            workflow_type: Type of workflow to execute
            pattern: Execution pattern to use
            workflow_data: Workflow input data
            execution_context: Execution context (patient_id, provider_id, etc.)
            
        Returns:
            Workflow execution result
        """
        try:
            execution_id = f"exec_{int(time.time() * 1000)}"
            start_time = time.time()
            
            logger.info(f"🔄 Executing workflow {workflow_type} with {pattern.value} pattern")
            
            # Get pattern configuration
            config = self.patterns[pattern]
            
            # Track execution
            self.active_executions[execution_id] = {
                "workflow_type": workflow_type,
                "pattern": pattern,
                "start_time": start_time,
                "config": config,
                "status": "running"
            }
            
            # Execute based on pattern
            if pattern == ExecutionPattern.PESSIMISTIC:
                result = await self._execute_pessimistic_pattern(
                    execution_id, workflow_type, workflow_data, execution_context, config
                )
            elif pattern == ExecutionPattern.OPTIMISTIC:
                result = await self._execute_optimistic_pattern(
                    execution_id, workflow_type, workflow_data, execution_context, config
                )
            elif pattern == ExecutionPattern.DIGITAL_REFLEX_ARC:
                result = await self._execute_digital_reflex_arc_pattern(
                    execution_id, workflow_type, workflow_data, execution_context, config
                )
            else:
                raise ValueError(f"Unknown execution pattern: {pattern}")
            
            # Calculate execution time
            execution_time_ms = (time.time() - start_time) * 1000
            
            # Check SLA compliance
            if execution_time_ms > config.sla_budget_ms:
                logger.warning(f"⚠️ SLA violation: {execution_time_ms:.1f}ms > {config.sla_budget_ms}ms")
                result["sla_violation"] = True
            else:
                logger.info(f"✅ SLA met: {execution_time_ms:.1f}ms <= {config.sla_budget_ms}ms")
                result["sla_violation"] = False
            
            # Update execution tracking
            self.active_executions[execution_id].update({
                "status": "completed",
                "execution_time_ms": execution_time_ms,
                "result": result
            })
            
            result.update({
                "execution_id": execution_id,
                "execution_pattern": pattern.value,
                "execution_time_ms": execution_time_ms,
                "sla_budget_ms": config.sla_budget_ms
            })
            
            return result
            
        except Exception as e:
            logger.error(f"❌ Workflow execution failed: {e}")
            
            # Update execution tracking
            if execution_id in self.active_executions:
                self.active_executions[execution_id].update({
                    "status": "failed",
                    "error": str(e)
                })
            
            raise
    
    async def _execute_pessimistic_pattern(
        self,
        execution_id: str,
        workflow_type: str,
        workflow_data: Dict[str, Any],
        execution_context: Dict[str, Any],
        config: PatternConfiguration
    ) -> Dict[str, Any]:
        """
        Execute workflow using pessimistic pattern.
        Flow: Generate Proposal → WAIT for Safety Validation → Commit if Safe
        """
        logger.info(f"🔒 Executing pessimistic pattern for {workflow_type}")
        
        # Phase 1: Generate Proposal (synchronous)
        logger.info("📊 Phase 1: Generate proposal (synchronous)")
        proposal = await self._generate_proposal(workflow_type, workflow_data, execution_context)
        
        # Phase 2: Mandatory Safety Validation (synchronous - WAIT for completion)
        logger.info("🛡️ Phase 2: Mandatory safety validation (synchronous)")
        safety_result = await self._validate_safety_synchronous(proposal, execution_context)
        
        # Phase 3: Commit only if safe (synchronous)
        if safety_result["verdict"] == "SAFE":
            logger.info("✅ Phase 3: Committing safe proposal")
            commit_result = await self._commit_proposal(proposal, safety_result)
            
            return {
                "status": "completed",
                "pattern": "pessimistic",
                "proposal": proposal,
                "safety_validation": safety_result,
                "commit_result": commit_result,
                "user_feedback": "wait_for_completion"
            }
        else:
            logger.warning("🚨 Proposal blocked by safety validation")
            return {
                "status": "blocked",
                "pattern": "pessimistic",
                "proposal": proposal,
                "safety_validation": safety_result,
                "reason": "safety_validation_failed",
                "user_feedback": "wait_for_completion"
            }
    
    async def _execute_optimistic_pattern(
        self,
        execution_id: str,
        workflow_type: str,
        workflow_data: Dict[str, Any],
        execution_context: Dict[str, Any],
        config: PatternConfiguration
    ) -> Dict[str, Any]:
        """
        Execute workflow using optimistic pattern.
        Flow: Generate Proposal → Immediate UI Feedback → Validate Async → Compensate if Unsafe
        """
        logger.info(f"⚡ Executing optimistic pattern for {workflow_type}")
        
        # Phase 1: Generate Proposal (fast)
        logger.info("📊 Phase 1: Generate proposal (fast)")
        proposal = await self._generate_proposal(workflow_type, workflow_data, execution_context)
        
        # Phase 2: Immediate optimistic commit
        logger.info("⚡ Phase 2: Immediate optimistic commit")
        commit_result = await self._commit_proposal_optimistic(proposal)
        
        # Phase 3: Parallel safety validation (async)
        logger.info("🛡️ Phase 3: Parallel safety validation (async)")
        safety_task = asyncio.create_task(
            self._validate_safety_asynchronous(proposal, execution_context, execution_id)
        )
        
        return {
            "status": "committed_optimistically",
            "pattern": "optimistic",
            "proposal": proposal,
            "commit_result": commit_result,
            "safety_validation_task": "running_async",
            "user_feedback": "immediate_optimistic",
            "compensation_available": True
        }
    
    async def _execute_digital_reflex_arc_pattern(
        self,
        execution_id: str,
        workflow_type: str,
        workflow_data: Dict[str, Any],
        execution_context: Dict[str, Any],
        config: PatternConfiguration
    ) -> Dict[str, Any]:
        """
        Execute workflow using digital reflex arc pattern.
        Flow: Autonomous execution with real-time continuous safety monitoring
        """
        logger.info(f"🤖 Executing digital reflex arc pattern for {workflow_type}")
        
        # Autonomous execution with continuous monitoring
        logger.info("🤖 Autonomous execution with continuous safety monitoring")
        
        # Start continuous safety monitoring
        monitoring_task = asyncio.create_task(
            self._continuous_safety_monitoring(execution_id, execution_context)
        )
        
        try:
            # Execute autonomously (very fast)
            proposal = await self._generate_proposal_autonomous(workflow_type, workflow_data, execution_context)
            commit_result = await self._commit_proposal_autonomous(proposal)
            
            # Notify humans (no waiting)
            await self._send_notification_only(proposal, commit_result, execution_context)
            
            return {
                "status": "executed_autonomously",
                "pattern": "digital_reflex_arc",
                "proposal": proposal,
                "commit_result": commit_result,
                "user_feedback": "notification_only",
                "human_intervention": "exception_based",
                "continuous_monitoring": "active"
            }
            
        finally:
            # Stop continuous monitoring
            monitoring_task.cancel()
    
    # Helper methods for proposal generation and validation
    async def _generate_proposal(self, workflow_type: str, workflow_data: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Generate workflow proposal."""
        return {
            "proposal_id": f"prop_{int(time.time() * 1000)}",
            "workflow_type": workflow_type,
            "data": workflow_data,
            "context": context,
            "generated_at": datetime.utcnow().isoformat()
        }
    
    async def _generate_proposal_autonomous(self, workflow_type: str, workflow_data: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Generate proposal for autonomous execution (optimized for speed)."""
        proposal = await self._generate_proposal(workflow_type, workflow_data, context)
        proposal["autonomous"] = True
        return proposal
    
    async def _validate_safety_synchronous(self, proposal: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Synchronous safety validation (wait for result)."""
        # Simulate safety validation
        await asyncio.sleep(0.05)  # 50ms safety check
        return {
            "verdict": "SAFE",
            "validation_time_ms": 50,
            "synchronous": True
        }
    
    async def _validate_safety_asynchronous(self, proposal: Dict[str, Any], context: Dict[str, Any], execution_id: str) -> Dict[str, Any]:
        """Asynchronous safety validation with compensation if unsafe."""
        # Simulate async safety validation
        await asyncio.sleep(0.1)  # 100ms async safety check
        
        safety_result = {
            "verdict": "SAFE",
            "validation_time_ms": 100,
            "asynchronous": True
        }
        
        # If unsafe, trigger compensation
        if safety_result["verdict"] != "SAFE":
            await self._trigger_compensation(execution_id, safety_result)
        
        return safety_result
    
    async def _continuous_safety_monitoring(self, execution_id: str, context: Dict[str, Any]):
        """Continuous safety monitoring for digital reflex arc."""
        try:
            while True:
                # Monitor safety conditions
                await asyncio.sleep(0.01)  # 10ms monitoring interval
                # Real implementation would check actual safety conditions
        except asyncio.CancelledError:
            logger.info(f"Stopped continuous monitoring for {execution_id}")
    
    async def _commit_proposal(self, proposal: Dict[str, Any], safety_result: Dict[str, Any]) -> Dict[str, Any]:
        """Commit proposal after safety validation."""
        return {
            "committed": True,
            "commit_time": datetime.utcnow().isoformat(),
            "safety_approved": True
        }
    
    async def _commit_proposal_optimistic(self, proposal: Dict[str, Any]) -> Dict[str, Any]:
        """Optimistic commit (before safety validation)."""
        return {
            "committed": True,
            "commit_time": datetime.utcnow().isoformat(),
            "optimistic": True
        }
    
    async def _commit_proposal_autonomous(self, proposal: Dict[str, Any]) -> Dict[str, Any]:
        """Autonomous commit (very fast)."""
        return {
            "committed": True,
            "commit_time": datetime.utcnow().isoformat(),
            "autonomous": True
        }
    
    async def _send_notification_only(self, proposal: Dict[str, Any], commit_result: Dict[str, Any], context: Dict[str, Any]):
        """Send notification without waiting for human response."""
        logger.info(f"📱 Notification sent for autonomous execution: {proposal['proposal_id']}")
    
    async def _trigger_compensation(self, execution_id: str, safety_result: Dict[str, Any]):
        """Trigger compensation workflow for unsafe optimistic execution."""
        logger.warning(f"🔄 Triggering compensation for {execution_id}: {safety_result['verdict']}")
    
    def get_pattern_for_workflow(self, workflow_type: str) -> ExecutionPattern:
        """Determine the appropriate execution pattern for a workflow type."""
        # High-risk workflows use pessimistic pattern
        high_risk_workflows = [
            "medication_prescribing", "high_alert_medication_orders", 
            "patient_discharge_decisions", "controlled_substance_prescribing",
            "chemotherapy_orders"
        ]
        
        # Autonomous workflows use digital reflex arc
        autonomous_workflows = [
            "clinical_deterioration_response", "critical_value_alerts",
            "sepsis_protocol_activation", "cardiac_arrest_response",
            "anaphylaxis_protocol"
        ]
        
        if workflow_type in high_risk_workflows:
            return ExecutionPattern.PESSIMISTIC
        elif workflow_type in autonomous_workflows:
            return ExecutionPattern.DIGITAL_REFLEX_ARC
        else:
            return ExecutionPattern.OPTIMISTIC
    
    def get_active_executions(self) -> Dict[str, Any]:
        """Get currently active workflow executions."""
        return self.active_executions.copy()


# Create singleton instance
clinical_execution_pattern_service = ClinicalExecutionPatternService()
