"""
Repository for Workflow Proposals database operations.

This repository provides data access layer for medication proposals,
replacing in-memory storage with persistent database operations.
"""
import logging
import uuid
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, Optional, List, Tuple
from sqlalchemy import and_, or_, desc, asc, func
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.orm import Session, sessionmaker
from sqlalchemy.engine import Engine

from app.models.workflow_proposal import (
    WorkflowProposal, 
    ProposalAuditLog,
    ProposalStatus,
    ProposalCreateRequest,
    ProposalUpdateRequest,
    ProposalSummary,
    ProposalDetails
)

logger = logging.getLogger(__name__)


class WorkflowProposalRepository:
    """
    Repository class for workflow proposal database operations.
    
    Provides CRUD operations, search capabilities, and audit logging
    for medication proposals in the Calculate > Validate > Commit workflow.
    """
    
    def __init__(self, db_engine: Engine):
        """Initialize repository with database engine."""
        self.engine = db_engine
        self.SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=db_engine)
        logger.info("Workflow Proposal Repository initialized")
    
    def get_session(self) -> Session:
        """Get database session."""
        return self.SessionLocal()
    
    async def create_proposal(
        self, 
        request: ProposalCreateRequest, 
        created_by: str,
        correlation_id: Optional[str] = None
    ) -> WorkflowProposal:
        """
        Create a new medication proposal.
        
        Args:
            request: Proposal creation request
            created_by: User ID creating the proposal
            correlation_id: Optional correlation ID for tracking
            
        Returns:
            Created WorkflowProposal instance
        """
        session = self.get_session()
        try:
            # Generate unique proposal ID
            proposal_id = f"prop_{uuid.uuid4().hex[:12]}"
            
            # Calculate expiration time
            expires_at = None
            if request.expires_in_hours:
                expires_at = datetime.now(timezone.utc) + timedelta(hours=request.expires_in_hours)
            
            # Create proposal instance
            proposal = WorkflowProposal(
                proposal_id=proposal_id,
                proposal_type=request.proposal_type,
                status=ProposalStatus.PROPOSED.value,
                workflow_phase="CALCULATE",
                correlation_id=correlation_id or f"corr_{uuid.uuid4().hex[:8]}",
                patient_id=request.patient_id,
                provider_id=request.provider_id,
                encounter_id=request.encounter_id,
                medication_data=request.medication_data,
                clinical_context=request.clinical_context,
                patient_context=request.patient_context,
                priority=request.priority,
                urgency_level=request.urgency_level,
                expires_at=expires_at,
                created_by=created_by,
                metadata=request.metadata,
                notes=request.notes
            )
            
            session.add(proposal)
            session.commit()
            session.refresh(proposal)
            
            # Create audit log entry
            await self._create_audit_entry(
                session=session,
                proposal_id=proposal.proposal_id,
                operation="CREATE_PROPOSAL",
                operation_status="SUCCESS",
                user_id=created_by,
                correlation_id=proposal.correlation_id,
                new_values={
                    "status": proposal.status,
                    "patient_id": proposal.patient_id,
                    "medication": request.medication_data.get("name", "unknown")
                }
            )
            
            logger.info(f"Created proposal {proposal_id} for patient {request.patient_id}")
            return proposal
            
        except Exception as e:
            session.rollback()
            logger.error(f"Failed to create proposal: {e}")
            raise
        finally:
            session.close()
    
    async def get_proposal_by_id(self, proposal_id: str) -> Optional[WorkflowProposal]:
        """Get proposal by ID."""
        session = self.get_session()
        try:
            proposal = session.query(WorkflowProposal).filter(
                WorkflowProposal.proposal_id == proposal_id
            ).first()
            return proposal
        except Exception as e:
            logger.error(f"Failed to get proposal {proposal_id}: {e}")
            raise
        finally:
            session.close()
    
    async def update_proposal(
        self, 
        proposal_id: str, 
        updates: ProposalUpdateRequest,
        updated_by: str
    ) -> Optional[WorkflowProposal]:
        """
        Update proposal with new data.
        
        Args:
            proposal_id: Proposal ID to update
            updates: Update request with new values
            updated_by: User ID performing update
            
        Returns:
            Updated WorkflowProposal or None if not found
        """
        session = self.get_session()
        try:
            proposal = session.query(WorkflowProposal).filter(
                WorkflowProposal.proposal_id == proposal_id
            ).first()
            
            if not proposal:
                return None
            
            # Store old values for audit
            old_values = {
                "status": proposal.status,
                "workflow_phase": proposal.workflow_phase,
                "validation_verdict": proposal.validation_verdict
            }
            
            # Apply updates
            update_fields = updates.dict(exclude_unset=True)
            for field, value in update_fields.items():
                if hasattr(proposal, field):
                    setattr(proposal, field, value)
            
            # Set updated metadata
            proposal.updated_by = updated_by
            proposal.updated_at = datetime.now(timezone.utc)
            
            # Special handling for status changes
            if updates.status == ProposalStatus.VALIDATED.value and updates.validation_id:
                proposal.validated_at = datetime.now(timezone.utc)
            elif updates.status == ProposalStatus.COMMITTED.value and updates.medication_order_id:
                proposal.committed_at = datetime.now(timezone.utc)
                proposal.committed_by = updated_by
            
            session.commit()
            session.refresh(proposal)
            
            # Create audit log entry
            await self._create_audit_entry(
                session=session,
                proposal_id=proposal_id,
                operation="UPDATE_PROPOSAL",
                operation_status="SUCCESS",
                user_id=updated_by,
                correlation_id=proposal.correlation_id,
                old_values=old_values,
                new_values=update_fields
            )
            
            logger.info(f"Updated proposal {proposal_id}")
            return proposal
            
        except Exception as e:
            session.rollback()
            logger.error(f"Failed to update proposal {proposal_id}: {e}")
            raise
        finally:
            session.close()
    
    async def search_proposals(
        self,
        patient_id: Optional[str] = None,
        provider_id: Optional[str] = None,
        status: Optional[str] = None,
        workflow_phase: Optional[str] = None,
        correlation_id: Optional[str] = None,
        created_after: Optional[datetime] = None,
        created_before: Optional[datetime] = None,
        limit: int = 100,
        offset: int = 0,
        sort_by: str = "created_at",
        sort_desc: bool = True
    ) -> Tuple[List[WorkflowProposal], int]:
        """
        Search proposals with filters.
        
        Returns:
            Tuple of (proposals_list, total_count)
        """
        session = self.get_session()
        try:
            query = session.query(WorkflowProposal)
            
            # Apply filters
            if patient_id:
                query = query.filter(WorkflowProposal.patient_id == patient_id)
            if provider_id:
                query = query.filter(WorkflowProposal.provider_id == provider_id)
            if status:
                query = query.filter(WorkflowProposal.status == status)
            if workflow_phase:
                query = query.filter(WorkflowProposal.workflow_phase == workflow_phase)
            if correlation_id:
                query = query.filter(WorkflowProposal.correlation_id == correlation_id)
            if created_after:
                query = query.filter(WorkflowProposal.created_at >= created_after)
            if created_before:
                query = query.filter(WorkflowProposal.created_at <= created_before)
            
            # Get total count before pagination
            total_count = query.count()
            
            # Apply sorting
            if hasattr(WorkflowProposal, sort_by):
                sort_column = getattr(WorkflowProposal, sort_by)
                if sort_desc:
                    query = query.order_by(desc(sort_column))
                else:
                    query = query.order_by(asc(sort_column))
            
            # Apply pagination
            proposals = query.offset(offset).limit(limit).all()
            
            return proposals, total_count
            
        except Exception as e:
            logger.error(f"Failed to search proposals: {e}")
            raise
        finally:
            session.close()
    
    async def get_proposals_by_status(
        self, 
        status: str, 
        limit: int = 100
    ) -> List[WorkflowProposal]:
        """Get proposals by status."""
        session = self.get_session()
        try:
            proposals = session.query(WorkflowProposal).filter(
                WorkflowProposal.status == status
            ).order_by(desc(WorkflowProposal.created_at)).limit(limit).all()
            
            return proposals
        except Exception as e:
            logger.error(f"Failed to get proposals by status {status}: {e}")
            raise
        finally:
            session.close()
    
    async def get_expired_proposals(self) -> List[WorkflowProposal]:
        """Get proposals that have expired."""
        session = self.get_session()
        try:
            now = datetime.now(timezone.utc)
            proposals = session.query(WorkflowProposal).filter(
                and_(
                    WorkflowProposal.expires_at <= now,
                    WorkflowProposal.status.in_([
                        ProposalStatus.PROPOSED.value,
                        ProposalStatus.VALIDATED.value
                    ])
                )
            ).all()
            
            return proposals
        except Exception as e:
            logger.error(f"Failed to get expired proposals: {e}")
            raise
        finally:
            session.close()
    
    async def mark_proposals_expired(self, proposal_ids: List[str], updated_by: str) -> int:
        """Mark multiple proposals as expired."""
        session = self.get_session()
        try:
            count = session.query(WorkflowProposal).filter(
                WorkflowProposal.proposal_id.in_(proposal_ids)
            ).update(
                {
                    "status": ProposalStatus.EXPIRED.value,
                    "updated_by": updated_by,
                    "updated_at": datetime.now(timezone.utc)
                },
                synchronize_session=False
            )
            
            session.commit()
            
            # Create audit entries
            for proposal_id in proposal_ids:
                await self._create_audit_entry(
                    session=session,
                    proposal_id=proposal_id,
                    operation="EXPIRE_PROPOSAL",
                    operation_status="SUCCESS",
                    user_id=updated_by
                )
            
            logger.info(f"Marked {count} proposals as expired")
            return count
            
        except Exception as e:
            session.rollback()
            logger.error(f"Failed to mark proposals expired: {e}")
            raise
        finally:
            session.close()
    
    async def get_proposal_statistics(self) -> Dict[str, Any]:
        """Get proposal statistics."""
        session = self.get_session()
        try:
            # Count by status
            status_counts = session.query(
                WorkflowProposal.status,
                func.count(WorkflowProposal.id).label('count')
            ).group_by(WorkflowProposal.status).all()
            
            # Count by workflow phase
            phase_counts = session.query(
                WorkflowProposal.workflow_phase,
                func.count(WorkflowProposal.id).label('count')
            ).group_by(WorkflowProposal.workflow_phase).all()
            
            # Recent activity (last 24 hours)
            last_24h = datetime.now(timezone.utc) - timedelta(hours=24)
            recent_count = session.query(WorkflowProposal).filter(
                WorkflowProposal.created_at >= last_24h
            ).count()
            
            # Average processing time for completed proposals
            avg_processing_time = session.query(
                func.avg(WorkflowProposal.processing_time_ms)
            ).filter(
                WorkflowProposal.status == ProposalStatus.COMMITTED.value
            ).scalar()
            
            return {
                "status_distribution": {status: count for status, count in status_counts},
                "phase_distribution": {phase or "unknown": count for phase, count in phase_counts},
                "total_proposals": sum(count for _, count in status_counts),
                "recent_24h": recent_count,
                "avg_processing_time_ms": avg_processing_time or 0,
                "timestamp": datetime.now(timezone.utc).isoformat()
            }
            
        except Exception as e:
            logger.error(f"Failed to get statistics: {e}")
            raise
        finally:
            session.close()
    
    async def get_audit_log(
        self, 
        proposal_id: Optional[str] = None,
        audit_trail_id: Optional[str] = None,
        limit: int = 100
    ) -> List[ProposalAuditLog]:
        """Get audit log entries."""
        session = self.get_session()
        try:
            query = session.query(ProposalAuditLog)
            
            if proposal_id:
                query = query.filter(ProposalAuditLog.proposal_id == proposal_id)
            if audit_trail_id:
                query = query.filter(ProposalAuditLog.audit_trail_id == audit_trail_id)
            
            logs = query.order_by(desc(ProposalAuditLog.timestamp)).limit(limit).all()
            return logs
            
        except Exception as e:
            logger.error(f"Failed to get audit log: {e}")
            raise
        finally:
            session.close()
    
    async def _create_audit_entry(
        self,
        session: Session,
        proposal_id: str,
        operation: str,
        operation_status: str,
        user_id: str,
        correlation_id: Optional[str] = None,
        workflow_phase: Optional[str] = None,
        old_values: Optional[Dict[str, Any]] = None,
        new_values: Optional[Dict[str, Any]] = None,
        operation_context: Optional[Dict[str, Any]] = None,
        error_message: Optional[str] = None,
        duration_ms: Optional[int] = None
    ) -> ProposalAuditLog:
        """Create audit log entry."""
        try:
            audit_entry = ProposalAuditLog(
                audit_trail_id=f"audit_{uuid.uuid4().hex[:16]}",
                proposal_id=proposal_id,
                operation=operation,
                operation_status=operation_status,
                workflow_phase=workflow_phase,
                user_id=user_id,
                correlation_id=correlation_id,
                duration_ms=duration_ms,
                old_values=old_values,
                new_values=new_values,
                operation_context=operation_context,
                error_message=error_message
            )
            
            session.add(audit_entry)
            session.commit()
            return audit_entry
            
        except Exception as e:
            logger.error(f"Failed to create audit entry: {e}")
            raise
    
    async def cleanup_old_proposals(self, retention_days: int = 30) -> int:
        """Clean up old proposals based on retention policy."""
        session = self.get_session()
        try:
            cutoff_date = datetime.now(timezone.utc) - timedelta(days=retention_days)
            
            # Only clean up completed or expired proposals
            count = session.query(WorkflowProposal).filter(
                and_(
                    WorkflowProposal.updated_at <= cutoff_date,
                    WorkflowProposal.status.in_([
                        ProposalStatus.COMMITTED.value,
                        ProposalStatus.EXPIRED.value,
                        ProposalStatus.CANCELLED.value,
                        ProposalStatus.FAILED.value
                    ])
                )
            ).delete(synchronize_session=False)
            
            session.commit()
            
            logger.info(f"Cleaned up {count} old proposals older than {retention_days} days")
            return count
            
        except Exception as e:
            session.rollback()
            logger.error(f"Failed to cleanup old proposals: {e}")
            raise
        finally:
            session.close()


# Global repository instance (to be initialized with database engine)
workflow_proposal_repository: Optional[WorkflowProposalRepository] = None


def init_workflow_proposal_repository(db_engine: Engine) -> WorkflowProposalRepository:
    """Initialize global workflow proposal repository."""
    global workflow_proposal_repository
    workflow_proposal_repository = WorkflowProposalRepository(db_engine)
    return workflow_proposal_repository


def get_workflow_proposal_repository() -> WorkflowProposalRepository:
    """Get the global workflow proposal repository."""
    if workflow_proposal_repository is None:
        raise RuntimeError("Workflow proposal repository not initialized")
    return workflow_proposal_repository