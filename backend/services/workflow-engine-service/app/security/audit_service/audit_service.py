"""
Audit Service for Clinical Workflow Engine.
Implements comprehensive audit trails for medical-legal compliance.
"""
import logging
import json
from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
from enum import Enum
import asyncio
from dataclasses import dataclass, asdict

logger = logging.getLogger(__name__)


class AuditLevel(Enum):
    """Audit levels for different types of clinical activities."""
    STANDARD = "standard"      # Basic workflow operations
    DETAILED = "detailed"      # Clinical decision points
    COMPREHENSIVE = "comprehensive"  # PHI access, safety overrides


class AuditEventType(Enum):
    """Types of audit events in clinical workflows."""
    WORKFLOW_STARTED = "workflow_started"
    WORKFLOW_COMPLETED = "workflow_completed"
    WORKFLOW_FAILED = "workflow_failed"
    ACTIVITY_EXECUTED = "activity_executed"
    SAFETY_CHECK_PERFORMED = "safety_check_performed"
    SAFETY_OVERRIDE = "safety_override"
    PHI_ACCESSED = "phi_accessed"
    CLINICAL_DECISION = "clinical_decision"
    COMPENSATION_EXECUTED = "compensation_executed"
    BREAK_GLASS_ACCESS = "break_glass_access"
    USER_LOGIN = "user_login"
    USER_LOGOUT = "user_logout"
    DATA_EXPORT = "data_export"


@dataclass
class AuditEntry:
    """Structured audit entry for clinical compliance."""
    audit_id: str
    timestamp: datetime
    event_type: AuditEventType
    audit_level: AuditLevel
    user_id: str
    patient_id: Optional[str]
    workflow_instance_id: Optional[str]
    session_id: Optional[str]
    ip_address: Optional[str]
    user_agent: Optional[str]
    action_details: Dict[str, Any]
    clinical_context: Optional[Dict[str, Any]]
    phi_accessed: bool
    safety_critical: bool
    outcome: str  # success, failure, warning
    error_details: Optional[Dict[str, Any]]
    retention_years: int = 7  # Medical-legal requirement


class AuditService:
    """
    Comprehensive audit service for clinical workflow compliance.
    Implements 7-year retention and medical-legal audit requirements.
    """
    
    def __init__(self):
        self.audit_entries: List[AuditEntry] = []
        self.retention_policy_years = 7
        self.max_memory_entries = 10000  # Rotate to persistent storage
        
    async def log_workflow_event(
        self,
        event_type: AuditEventType,
        user_id: str,
        workflow_instance_id: str,
        action_details: Dict[str, Any],
        patient_id: Optional[str] = None,
        clinical_context: Optional[Dict[str, Any]] = None,
        session_id: Optional[str] = None,
        ip_address: Optional[str] = None,
        user_agent: Optional[str] = None,
        audit_level: AuditLevel = AuditLevel.STANDARD,
        phi_accessed: bool = False,
        safety_critical: bool = False,
        outcome: str = "success",
        error_details: Optional[Dict[str, Any]] = None
    ) -> str:
        """
        Log a clinical workflow event with comprehensive audit details.
        """
        try:
            audit_id = f"audit_{int(datetime.utcnow().timestamp() * 1000000)}"
            
            audit_entry = AuditEntry(
                audit_id=audit_id,
                timestamp=datetime.utcnow(),
                event_type=event_type,
                audit_level=audit_level,
                user_id=user_id,
                patient_id=patient_id,
                workflow_instance_id=workflow_instance_id,
                session_id=session_id,
                ip_address=ip_address,
                user_agent=user_agent,
                action_details=action_details,
                clinical_context=clinical_context,
                phi_accessed=phi_accessed,
                safety_critical=safety_critical,
                outcome=outcome,
                error_details=error_details
            )
            
            # Store audit entry
            await self._store_audit_entry(audit_entry)
            
            # Log for monitoring
            logger.info(f"Audit: {event_type.value} by {user_id} - {outcome}")
            
            # Alert on critical events
            if safety_critical or phi_accessed or event_type == AuditEventType.SAFETY_OVERRIDE:
                await self._alert_critical_event(audit_entry)
            
            return audit_id
            
        except Exception as e:
            logger.error(f"Failed to log audit event: {e}")
            # Audit failures should not break workflow execution
            return f"audit_failed_{int(datetime.utcnow().timestamp())}"
    
    async def log_clinical_decision(
        self,
        user_id: str,
        patient_id: str,
        workflow_instance_id: str,
        decision_type: str,
        decision_details: Dict[str, Any],
        clinical_rationale: str,
        safety_checks_performed: List[str],
        overrides_applied: Optional[List[str]] = None,
        session_id: Optional[str] = None
    ) -> str:
        """
        Log clinical decision points for medical-legal compliance.
        """
        action_details = {
            "decision_type": decision_type,
            "decision_details": decision_details,
            "clinical_rationale": clinical_rationale,
            "safety_checks_performed": safety_checks_performed,
            "overrides_applied": overrides_applied or [],
            "decision_timestamp": datetime.utcnow().isoformat()
        }
        
        return await self.log_workflow_event(
            event_type=AuditEventType.CLINICAL_DECISION,
            user_id=user_id,
            patient_id=patient_id,
            workflow_instance_id=workflow_instance_id,
            action_details=action_details,
            audit_level=AuditLevel.COMPREHENSIVE,
            safety_critical=bool(overrides_applied),
            session_id=session_id
        )
    
    async def log_safety_override(
        self,
        user_id: str,
        patient_id: str,
        workflow_instance_id: str,
        override_type: str,
        safety_warning: str,
        clinical_justification: str,
        supervisor_approval: Optional[str] = None,
        session_id: Optional[str] = None
    ) -> str:
        """
        Log safety overrides with enhanced audit requirements.
        """
        action_details = {
            "override_type": override_type,
            "safety_warning": safety_warning,
            "clinical_justification": clinical_justification,
            "supervisor_approval": supervisor_approval,
            "override_timestamp": datetime.utcnow().isoformat(),
            "requires_follow_up": True
        }
        
        return await self.log_workflow_event(
            event_type=AuditEventType.SAFETY_OVERRIDE,
            user_id=user_id,
            patient_id=patient_id,
            workflow_instance_id=workflow_instance_id,
            action_details=action_details,
            audit_level=AuditLevel.COMPREHENSIVE,
            safety_critical=True,
            session_id=session_id
        )
    
    async def log_phi_access(
        self,
        user_id: str,
        patient_id: str,
        access_type: str,
        phi_fields: List[str],
        workflow_instance_id: Optional[str] = None,
        session_id: Optional[str] = None,
        ip_address: Optional[str] = None
    ) -> str:
        """
        Log PHI access for HIPAA compliance.
        """
        action_details = {
            "access_type": access_type,
            "phi_fields_accessed": phi_fields,
            "phi_fields_count": len(phi_fields),
            "access_timestamp": datetime.utcnow().isoformat()
        }
        
        return await self.log_workflow_event(
            event_type=AuditEventType.PHI_ACCESSED,
            user_id=user_id,
            patient_id=patient_id,
            workflow_instance_id=workflow_instance_id,
            action_details=action_details,
            audit_level=AuditLevel.COMPREHENSIVE,
            phi_accessed=True,
            session_id=session_id,
            ip_address=ip_address
        )
    
    async def search_audit_trail(
        self,
        patient_id: Optional[str] = None,
        user_id: Optional[str] = None,
        workflow_instance_id: Optional[str] = None,
        event_type: Optional[AuditEventType] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
        audit_level: Optional[AuditLevel] = None,
        limit: int = 100
    ) -> List[Dict[str, Any]]:
        """
        Search audit trail with filtering for compliance reporting.
        """
        try:
            filtered_entries = []
            
            for entry in self.audit_entries:
                # Apply filters
                if patient_id and entry.patient_id != patient_id:
                    continue
                if user_id and entry.user_id != user_id:
                    continue
                if workflow_instance_id and entry.workflow_instance_id != workflow_instance_id:
                    continue
                if event_type and entry.event_type != event_type:
                    continue
                if audit_level and entry.audit_level != audit_level:
                    continue
                if start_date and entry.timestamp < start_date:
                    continue
                if end_date and entry.timestamp > end_date:
                    continue
                
                filtered_entries.append(asdict(entry))
                
                if len(filtered_entries) >= limit:
                    break
            
            logger.info(f"Audit search returned {len(filtered_entries)} entries")
            return filtered_entries
            
        except Exception as e:
            logger.error(f"Failed to search audit trail: {e}")
            return []
    
    async def export_audit_trail(
        self,
        patient_id: str,
        requesting_user_id: str,
        export_format: str = "json"
    ) -> Dict[str, Any]:
        """
        Export complete audit trail for a patient (medical-legal requirement).
        """
        try:
            # Log the export request
            await self.log_workflow_event(
                event_type=AuditEventType.DATA_EXPORT,
                user_id=requesting_user_id,
                patient_id=patient_id,
                workflow_instance_id=None,
                action_details={
                    "export_type": "audit_trail",
                    "export_format": export_format,
                    "export_timestamp": datetime.utcnow().isoformat()
                },
                audit_level=AuditLevel.COMPREHENSIVE
            )
            
            # Get all audit entries for patient
            patient_entries = await self.search_audit_trail(
                patient_id=patient_id,
                limit=10000  # No limit for exports
            )
            
            export_data = {
                "patient_id": patient_id,
                "export_timestamp": datetime.utcnow().isoformat(),
                "exported_by": requesting_user_id,
                "total_entries": len(patient_entries),
                "retention_period_years": self.retention_policy_years,
                "audit_entries": patient_entries
            }
            
            logger.info(f"Exported {len(patient_entries)} audit entries for patient {patient_id}")
            return export_data
            
        except Exception as e:
            logger.error(f"Failed to export audit trail: {e}")
            raise
    
    async def _store_audit_entry(self, audit_entry: AuditEntry) -> None:
        """
        Store audit entry with rotation to persistent storage.
        """
        self.audit_entries.append(audit_entry)

        # Store in database for persistent audit trail
        try:
            from app.database_audit_service import database_audit_service
            await database_audit_service.store_audit_entry(audit_entry)
        except Exception as e:
            logger.error(f"Failed to store audit entry in database: {e}")
            # Continue execution - audit failures should not break workflows

        # Rotate to persistent storage if memory limit reached
        if len(self.audit_entries) > self.max_memory_entries:
            await self._rotate_to_persistent_storage()
    
    async def _rotate_to_persistent_storage(self) -> None:
        """
        Rotate audit entries to persistent storage (database).
        In production, this would write to a secure audit database.
        """
        try:
            # In production: Write to secure audit database
            # await audit_database.bulk_insert(self.audit_entries[:-1000])
            
            # Keep recent entries in memory
            self.audit_entries = self.audit_entries[-1000:]
            
            logger.info("Rotated audit entries to persistent storage")
            
        except Exception as e:
            logger.error(f"Failed to rotate audit entries: {e}")
    
    async def _alert_critical_event(self, audit_entry: AuditEntry) -> None:
        """
        Alert on critical audit events (safety overrides, PHI access, etc.).
        """
        try:
            alert_data = {
                "alert_type": "critical_audit_event",
                "event_type": audit_entry.event_type.value,
                "user_id": audit_entry.user_id,
                "patient_id": audit_entry.patient_id,
                "timestamp": audit_entry.timestamp.isoformat(),
                "safety_critical": audit_entry.safety_critical,
                "phi_accessed": audit_entry.phi_accessed
            }
            
            # In production: Send to monitoring/alerting system
            logger.warning(f"CRITICAL AUDIT EVENT: {audit_entry.event_type.value} by {audit_entry.user_id}")
            
        except Exception as e:
            logger.error(f"Failed to send critical event alert: {e}")


# Global instance
audit_service = AuditService()
