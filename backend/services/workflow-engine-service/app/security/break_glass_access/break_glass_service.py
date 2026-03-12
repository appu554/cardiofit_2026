"""
Break-Glass Access Service for Clinical Workflow Engine.
Implements emergency access patterns and audit trail for emergency procedures.
"""
import logging
import json
from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
from enum import Enum
import asyncio
from dataclasses import dataclass

from .audit_service import audit_service, AuditEventType, AuditLevel

logger = logging.getLogger(__name__)


class EmergencyAccessType(Enum):
    """Types of emergency access scenarios."""
    PATIENT_EMERGENCY = "patient_emergency"      # Life-threatening situation
    SYSTEM_FAILURE = "system_failure"           # Technical system unavailable
    WORKFLOW_OVERRIDE = "workflow_override"     # Clinical workflow interruption
    DATA_ACCESS = "data_access"                 # Emergency PHI access
    SAFETY_OVERRIDE = "safety_override"         # Override safety checks


class EmergencyJustification(Enum):
    """Predefined emergency justifications for compliance."""
    CARDIAC_ARREST = "cardiac_arrest"
    RESPIRATORY_FAILURE = "respiratory_failure"
    SEVERE_BLEEDING = "severe_bleeding"
    ANAPHYLAXIS = "anaphylaxis"
    STROKE = "stroke"
    SEPSIS = "sepsis"
    SYSTEM_OUTAGE = "system_outage"
    NETWORK_FAILURE = "network_failure"
    OTHER_EMERGENCY = "other_emergency"


@dataclass
class BreakGlassSession:
    """Break-glass access session with time limits and audit trail."""
    session_id: str
    user_id: str
    patient_id: Optional[str]
    access_type: EmergencyAccessType
    justification: EmergencyJustification
    clinical_details: str
    supervisor_approval: Optional[str]
    started_at: datetime
    expires_at: datetime
    is_active: bool
    actions_performed: List[Dict[str, Any]]
    audit_trail: List[str]


class BreakGlassAccessService:
    """
    Break-Glass Access Service for emergency clinical situations.
    Implements time-limited emergency access with comprehensive audit trails.
    """
    
    def __init__(self):
        self.active_sessions: Dict[str, BreakGlassSession] = {}
        self.session_timeout_minutes = 30  # Emergency sessions expire after 30 minutes
        self.max_concurrent_sessions = 10
        
    async def initiate_break_glass_access(
        self,
        user_id: str,
        access_type: EmergencyAccessType,
        justification: EmergencyJustification,
        clinical_details: str,
        patient_id: Optional[str] = None,
        supervisor_approval: Optional[str] = None,
        session_id: Optional[str] = None,
        ip_address: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Initiate break-glass emergency access with audit trail.
        """
        try:
            # Generate session ID
            if not session_id:
                session_id = f"breakglass_{int(datetime.utcnow().timestamp() * 1000)}"
            
            # Check concurrent session limits
            if len(self.active_sessions) >= self.max_concurrent_sessions:
                await self._cleanup_expired_sessions()
                
                if len(self.active_sessions) >= self.max_concurrent_sessions:
                    raise Exception("Maximum concurrent break-glass sessions exceeded")
            
            # Create break-glass session
            session = BreakGlassSession(
                session_id=session_id,
                user_id=user_id,
                patient_id=patient_id,
                access_type=access_type,
                justification=justification,
                clinical_details=clinical_details,
                supervisor_approval=supervisor_approval,
                started_at=datetime.utcnow(),
                expires_at=datetime.utcnow() + timedelta(minutes=self.session_timeout_minutes),
                is_active=True,
                actions_performed=[],
                audit_trail=[]
            )
            
            # Store active session
            self.active_sessions[session_id] = session
            
            # Log break-glass initiation
            audit_id = await audit_service.log_workflow_event(
                event_type=AuditEventType.BREAK_GLASS_ACCESS,
                user_id=user_id,
                patient_id=patient_id,
                workflow_instance_id=None,
                action_details={
                    "action": "initiate_break_glass",
                    "access_type": access_type.value,
                    "justification": justification.value,
                    "clinical_details": clinical_details,
                    "supervisor_approval": supervisor_approval,
                    "session_timeout_minutes": self.session_timeout_minutes
                },
                audit_level=AuditLevel.COMPREHENSIVE,
                safety_critical=True,
                session_id=session_id,
                ip_address=ip_address
            )
            
            session.audit_trail.append(audit_id)
            
            # Alert security team
            await self._alert_break_glass_access(session)
            
            logger.warning(f"Break-glass access initiated: {session_id} by {user_id}")
            
            return {
                "session_id": session_id,
                "access_granted": True,
                "expires_at": session.expires_at.isoformat(),
                "timeout_minutes": self.session_timeout_minutes,
                "audit_id": audit_id,
                "message": "Emergency access granted - all actions will be audited"
            }
            
        except Exception as e:
            logger.error(f"Failed to initiate break-glass access: {e}")
            
            # Log failed attempt
            await audit_service.log_workflow_event(
                event_type=AuditEventType.BREAK_GLASS_ACCESS,
                user_id=user_id,
                patient_id=patient_id,
                workflow_instance_id=None,
                action_details={
                    "action": "initiate_break_glass_failed",
                    "error": str(e),
                    "access_type": access_type.value,
                    "justification": justification.value
                },
                audit_level=AuditLevel.COMPREHENSIVE,
                safety_critical=True,
                outcome="failure",
                error_details={"error": str(e)},
                session_id=session_id,
                ip_address=ip_address
            )
            
            raise
    
    async def validate_break_glass_session(
        self,
        session_id: str,
        user_id: str
    ) -> bool:
        """
        Validate that break-glass session is active and belongs to user.
        """
        try:
            session = self.active_sessions.get(session_id)
            
            if not session:
                logger.warning(f"Break-glass session not found: {session_id}")
                return False
            
            if not session.is_active:
                logger.warning(f"Break-glass session inactive: {session_id}")
                return False
            
            if session.user_id != user_id:
                logger.warning(f"Break-glass session user mismatch: {session_id}")
                return False
            
            if datetime.utcnow() > session.expires_at:
                logger.warning(f"Break-glass session expired: {session_id}")
                await self._expire_session(session_id)
                return False
            
            return True
            
        except Exception as e:
            logger.error(f"Failed to validate break-glass session: {e}")
            return False
    
    async def log_break_glass_action(
        self,
        session_id: str,
        user_id: str,
        action_type: str,
        action_details: Dict[str, Any],
        workflow_instance_id: Optional[str] = None
    ) -> str:
        """
        Log action performed during break-glass session.
        """
        try:
            session = self.active_sessions.get(session_id)
            
            if not session:
                raise Exception(f"Break-glass session not found: {session_id}")
            
            if not await self.validate_break_glass_session(session_id, user_id):
                raise Exception(f"Invalid break-glass session: {session_id}")
            
            # Log the action
            audit_id = await audit_service.log_workflow_event(
                event_type=AuditEventType.BREAK_GLASS_ACCESS,
                user_id=user_id,
                patient_id=session.patient_id,
                workflow_instance_id=workflow_instance_id,
                action_details={
                    "action": "break_glass_action",
                    "action_type": action_type,
                    "action_details": action_details,
                    "break_glass_session_id": session_id,
                    "access_type": session.access_type.value,
                    "justification": session.justification.value
                },
                audit_level=AuditLevel.COMPREHENSIVE,
                safety_critical=True,
                session_id=session_id
            )
            
            # Record action in session
            action_record = {
                "timestamp": datetime.utcnow().isoformat(),
                "action_type": action_type,
                "action_details": action_details,
                "audit_id": audit_id
            }
            
            session.actions_performed.append(action_record)
            session.audit_trail.append(audit_id)
            
            logger.info(f"Break-glass action logged: {action_type} in session {session_id}")
            return audit_id
            
        except Exception as e:
            logger.error(f"Failed to log break-glass action: {e}")
            raise
    
    async def terminate_break_glass_session(
        self,
        session_id: str,
        user_id: str,
        termination_reason: str = "manual_termination"
    ) -> Dict[str, Any]:
        """
        Terminate break-glass session and generate summary report.
        """
        try:
            session = self.active_sessions.get(session_id)
            
            if not session:
                raise Exception(f"Break-glass session not found: {session_id}")
            
            if session.user_id != user_id:
                raise Exception(f"Unauthorized termination attempt for session: {session_id}")
            
            # Mark session as inactive
            session.is_active = False
            
            # Log termination
            audit_id = await audit_service.log_workflow_event(
                event_type=AuditEventType.BREAK_GLASS_ACCESS,
                user_id=user_id,
                patient_id=session.patient_id,
                workflow_instance_id=None,
                action_details={
                    "action": "terminate_break_glass",
                    "termination_reason": termination_reason,
                    "session_duration_minutes": (datetime.utcnow() - session.started_at).total_seconds() / 60,
                    "actions_performed_count": len(session.actions_performed),
                    "access_type": session.access_type.value,
                    "justification": session.justification.value
                },
                audit_level=AuditLevel.COMPREHENSIVE,
                safety_critical=True,
                session_id=session_id
            )
            
            session.audit_trail.append(audit_id)
            
            # Generate session summary
            session_summary = {
                "session_id": session_id,
                "user_id": user_id,
                "patient_id": session.patient_id,
                "access_type": session.access_type.value,
                "justification": session.justification.value,
                "clinical_details": session.clinical_details,
                "supervisor_approval": session.supervisor_approval,
                "started_at": session.started_at.isoformat(),
                "terminated_at": datetime.utcnow().isoformat(),
                "duration_minutes": (datetime.utcnow() - session.started_at).total_seconds() / 60,
                "actions_performed": session.actions_performed,
                "audit_trail": session.audit_trail,
                "termination_reason": termination_reason
            }
            
            # Remove from active sessions
            del self.active_sessions[session_id]
            
            logger.info(f"Break-glass session terminated: {session_id}")
            return session_summary
            
        except Exception as e:
            logger.error(f"Failed to terminate break-glass session: {e}")
            raise
    
    async def get_active_sessions(self, user_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Get list of active break-glass sessions (for monitoring).
        """
        try:
            await self._cleanup_expired_sessions()
            
            active_sessions = []
            for session in self.active_sessions.values():
                if user_id and session.user_id != user_id:
                    continue
                
                session_info = {
                    "session_id": session.session_id,
                    "user_id": session.user_id,
                    "patient_id": session.patient_id,
                    "access_type": session.access_type.value,
                    "justification": session.justification.value,
                    "started_at": session.started_at.isoformat(),
                    "expires_at": session.expires_at.isoformat(),
                    "actions_count": len(session.actions_performed),
                    "time_remaining_minutes": max(0, (session.expires_at - datetime.utcnow()).total_seconds() / 60)
                }
                active_sessions.append(session_info)
            
            return active_sessions
            
        except Exception as e:
            logger.error(f"Failed to get active sessions: {e}")
            return []
    
    async def _cleanup_expired_sessions(self) -> None:
        """Clean up expired break-glass sessions."""
        try:
            current_time = datetime.utcnow()
            expired_sessions = []
            
            for session_id, session in self.active_sessions.items():
                if current_time > session.expires_at:
                    expired_sessions.append(session_id)
            
            for session_id in expired_sessions:
                await self._expire_session(session_id)
            
            if expired_sessions:
                logger.info(f"Cleaned up {len(expired_sessions)} expired break-glass sessions")
                
        except Exception as e:
            logger.error(f"Failed to cleanup expired sessions: {e}")
    
    async def _expire_session(self, session_id: str) -> None:
        """Expire a break-glass session due to timeout."""
        try:
            session = self.active_sessions.get(session_id)
            if session:
                session.is_active = False
                
                # Log expiration
                await audit_service.log_workflow_event(
                    event_type=AuditEventType.BREAK_GLASS_ACCESS,
                    user_id=session.user_id,
                    patient_id=session.patient_id,
                    workflow_instance_id=None,
                    action_details={
                        "action": "session_expired",
                        "session_duration_minutes": (datetime.utcnow() - session.started_at).total_seconds() / 60,
                        "actions_performed_count": len(session.actions_performed)
                    },
                    audit_level=AuditLevel.COMPREHENSIVE,
                    safety_critical=True,
                    session_id=session_id
                )
                
                del self.active_sessions[session_id]
                logger.warning(f"Break-glass session expired: {session_id}")
                
        except Exception as e:
            logger.error(f"Failed to expire session {session_id}: {e}")
    
    async def _alert_break_glass_access(self, session: BreakGlassSession) -> None:
        """Alert security team about break-glass access."""
        try:
            alert_data = {
                "alert_type": "break_glass_access_initiated",
                "session_id": session.session_id,
                "user_id": session.user_id,
                "patient_id": session.patient_id,
                "access_type": session.access_type.value,
                "justification": session.justification.value,
                "clinical_details": session.clinical_details,
                "supervisor_approval": session.supervisor_approval,
                "timestamp": session.started_at.isoformat()
            }
            
            # In production: Send to security monitoring system
            logger.warning(f"BREAK-GLASS ACCESS ALERT: {session.session_id} by {session.user_id}")
            
        except Exception as e:
            logger.error(f"Failed to send break-glass alert: {e}")


# Global instance
break_glass_access_service = BreakGlassAccessService()
