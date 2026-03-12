"""
Database Audit Service for Clinical Workflow Engine.
Integrates the security framework with the database for persistent audit trails.
"""
import logging
import json
from typing import Dict, Any, List, Optional
from datetime import datetime
import asyncio
import asyncpg
from sqlalchemy import create_engine, text
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker

from app.core.config import settings
from app.security.audit_service import AuditEntry, AuditEventType, AuditLevel

logger = logging.getLogger(__name__)


class DatabaseAuditService:
    """
    Database integration service for persistent audit trails.
    Stores audit entries in the clinical_audit_trail table for 7-year retention.
    """
    
    def __init__(self):
        self.engine = None
        self.async_session = None
        self._initialize_database()
    
    def _initialize_database(self):
        """Initialize database connection for audit storage."""
        try:
            # Create async engine for database operations
            self.engine = create_async_engine(
                settings.DATABASE_URL.replace('postgresql://', 'postgresql+asyncpg://'),
                echo=False,
                pool_size=5,
                max_overflow=10
            )
            
            # Create async session factory
            async_session_factory = sessionmaker(
                self.engine, class_=AsyncSession, expire_on_commit=False
            )
            self.async_session = async_session_factory
            
            logger.info("✅ Database audit service initialized")
            
        except Exception as e:
            logger.error(f"❌ Failed to initialize database audit service: {e}")
            raise
    
    async def store_audit_entry(self, audit_entry: AuditEntry) -> str:
        """
        Store audit entry in the clinical_audit_trail table.
        """
        try:
            async with self.async_session() as session:
                # Convert audit entry to database format
                audit_data = {
                    'id': audit_entry.audit_id,
                    'workflow_instance_id': self._get_workflow_instance_db_id(audit_entry.workflow_instance_id),
                    'patient_id': audit_entry.patient_id,
                    'provider_id': audit_entry.user_id,
                    'action_type': audit_entry.event_type.value,
                    'action_details': audit_entry.action_details,
                    'clinical_context': audit_entry.clinical_context or {},
                    'phi_accessed': audit_entry.phi_accessed,
                    'phi_fields_accessed': self._extract_phi_fields(audit_entry.action_details),
                    'timestamp': audit_entry.timestamp,
                    'session_id': audit_entry.session_id,
                    'ip_address': audit_entry.ip_address,
                    'user_agent': audit_entry.user_agent,
                    'audit_level': audit_entry.audit_level.value,
                    'event_type': audit_entry.event_type.value,
                    'audit_level_enum': audit_entry.audit_level.value,
                    'outcome': audit_entry.outcome,
                    'error_details': audit_entry.error_details or {},
                    'safety_critical': audit_entry.safety_critical,
                    'retention_years': audit_entry.retention_years
                }
                
                # Insert audit entry
                query = text("""
                    INSERT INTO clinical_audit_trail (
                        id, workflow_instance_id, patient_id, provider_id, action_type,
                        action_details, clinical_context, phi_accessed, phi_fields_accessed,
                        timestamp, session_id, ip_address, user_agent, audit_level,
                        event_type, audit_level_enum, outcome, error_details, 
                        safety_critical, retention_years
                    ) VALUES (
                        :id, :workflow_instance_id, :patient_id, :provider_id, :action_type,
                        :action_details, :clinical_context, :phi_accessed, :phi_fields_accessed,
                        :timestamp, :session_id, :ip_address, :user_agent, :audit_level,
                        :event_type, :audit_level_enum, :outcome, :error_details,
                        :safety_critical, :retention_years
                    )
                """)
                
                await session.execute(query, audit_data)
                await session.commit()
                
                logger.info(f"✅ Stored audit entry in database: {audit_entry.audit_id}")
                return audit_entry.audit_id
                
        except Exception as e:
            logger.error(f"❌ Failed to store audit entry: {e}")
            raise
    
    async def store_phi_access_log(
        self,
        user_id: str,
        patient_id: str,
        access_type: str,
        phi_fields: List[str],
        workflow_instance_id: Optional[str] = None,
        session_id: Optional[str] = None,
        ip_address: Optional[str] = None,
        user_agent: Optional[str] = None
    ) -> str:
        """
        Store PHI access log entry for detailed HIPAA compliance tracking.
        """
        try:
            async with self.async_session() as session:
                phi_access_id = f"phi_access_{int(datetime.utcnow().timestamp() * 1000000)}"
                
                phi_data = {
                    'id': phi_access_id,
                    'user_id': user_id,
                    'patient_id': patient_id,
                    'workflow_instance_id': self._get_workflow_instance_db_id(workflow_instance_id),
                    'access_type': access_type,
                    'phi_fields_accessed': phi_fields,
                    'phi_fields_count': len(phi_fields),
                    'access_timestamp': datetime.utcnow(),
                    'session_id': session_id,
                    'ip_address': ip_address,
                    'user_agent': user_agent,
                    'access_purpose': 'clinical_care',
                    'data_classification': 'phi'
                }
                
                query = text("""
                    INSERT INTO phi_access_log (
                        id, user_id, patient_id, workflow_instance_id, access_type,
                        phi_fields_accessed, phi_fields_count, access_timestamp,
                        session_id, ip_address, user_agent, access_purpose, data_classification
                    ) VALUES (
                        :id, :user_id, :patient_id, :workflow_instance_id, :access_type,
                        :phi_fields_accessed, :phi_fields_count, :access_timestamp,
                        :session_id, :ip_address, :user_agent, :access_purpose, :data_classification
                    )
                """)
                
                await session.execute(query, phi_data)
                await session.commit()
                
                logger.info(f"✅ Stored PHI access log: {phi_access_id}")
                return phi_access_id
                
        except Exception as e:
            logger.error(f"❌ Failed to store PHI access log: {e}")
            raise
    
    async def store_clinical_decision(
        self,
        decision_id: str,
        decision_type: str,
        decision_maker_id: str,
        patient_id: str,
        clinical_context: Dict[str, Any],
        decision_details: Dict[str, Any],
        clinical_rationale: str,
        safety_checks_performed: List[str],
        overrides_applied: Optional[List[str]] = None,
        supervisor_approval: Optional[str] = None,
        workflow_instance_id: Optional[str] = None
    ) -> str:
        """
        Store clinical decision audit entry for medical-legal compliance.
        """
        try:
            async with self.async_session() as session:
                decision_data = {
                    'decision_id': decision_id,
                    'decision_type': decision_type,
                    'decision_maker_id': decision_maker_id,
                    'patient_id': patient_id,
                    'workflow_instance_id': self._get_workflow_instance_db_id(workflow_instance_id),
                    'clinical_context': clinical_context,
                    'decision_details': decision_details,
                    'clinical_rationale': clinical_rationale,
                    'safety_checks_performed': safety_checks_performed,
                    'safety_warnings': [],
                    'overrides_applied': overrides_applied or [],
                    'supervisor_approval': supervisor_approval,
                    'decision_timestamp': datetime.utcnow()
                }
                
                query = text("""
                    INSERT INTO clinical_decision_audit (
                        decision_id, decision_type, decision_maker_id, patient_id,
                        workflow_instance_id, clinical_context, decision_details,
                        clinical_rationale, safety_checks_performed, safety_warnings,
                        overrides_applied, supervisor_approval, decision_timestamp
                    ) VALUES (
                        :decision_id, :decision_type, :decision_maker_id, :patient_id,
                        :workflow_instance_id, :clinical_context, :decision_details,
                        :clinical_rationale, :safety_checks_performed, :safety_warnings,
                        :overrides_applied, :supervisor_approval, :decision_timestamp
                    )
                """)
                
                await session.execute(query, decision_data)
                await session.commit()
                
                logger.info(f"✅ Stored clinical decision audit: {decision_id}")
                return decision_id
                
        except Exception as e:
            logger.error(f"❌ Failed to store clinical decision: {e}")
            raise
    
    async def store_encrypted_workflow_state(
        self,
        workflow_instance_id: str,
        encrypted_state: str,
        encryption_key_id: str,
        phi_fields_encrypted: List[str],
        encrypted_by: str
    ) -> str:
        """
        Store encrypted workflow state for PHI protection.
        """
        try:
            async with self.async_session() as session:
                state_data = {
                    'workflow_instance_id': self._get_workflow_instance_db_id(workflow_instance_id),
                    'encrypted_state': encrypted_state,
                    'encryption_key_id': encryption_key_id,
                    'phi_fields_encrypted': phi_fields_encrypted,
                    'encryption_metadata': {
                        'encryption_timestamp': datetime.utcnow().isoformat(),
                        'phi_fields_count': len(phi_fields_encrypted)
                    },
                    'encrypted_by': encrypted_by,
                    'encrypted_at': datetime.utcnow()
                }
                
                # Use UPSERT to handle updates
                query = text("""
                    INSERT INTO encrypted_workflow_states (
                        workflow_instance_id, encrypted_state, encryption_key_id,
                        phi_fields_encrypted, encryption_metadata, encrypted_by, encrypted_at
                    ) VALUES (
                        :workflow_instance_id, :encrypted_state, :encryption_key_id,
                        :phi_fields_encrypted, :encryption_metadata, :encrypted_by, :encrypted_at
                    )
                    ON CONFLICT (workflow_instance_id) 
                    DO UPDATE SET
                        encrypted_state = EXCLUDED.encrypted_state,
                        encryption_key_id = EXCLUDED.encryption_key_id,
                        phi_fields_encrypted = EXCLUDED.phi_fields_encrypted,
                        encryption_metadata = EXCLUDED.encryption_metadata,
                        encrypted_by = EXCLUDED.encrypted_by,
                        encrypted_at = EXCLUDED.encrypted_at,
                        updated_at = NOW()
                """)
                
                await session.execute(query, state_data)
                await session.commit()
                
                logger.info(f"✅ Stored encrypted workflow state: {workflow_instance_id}")
                return workflow_instance_id
                
        except Exception as e:
            logger.error(f"❌ Failed to store encrypted workflow state: {e}")
            raise
    
    async def search_audit_trail(
        self,
        patient_id: Optional[str] = None,
        user_id: Optional[str] = None,
        workflow_instance_id: Optional[str] = None,
        event_type: Optional[str] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
        limit: int = 100
    ) -> List[Dict[str, Any]]:
        """
        Search audit trail in database with filtering.
        """
        try:
            async with self.async_session() as session:
                # Build dynamic query
                where_conditions = []
                params = {'limit': limit}
                
                if patient_id:
                    where_conditions.append("patient_id = :patient_id")
                    params['patient_id'] = patient_id
                
                if user_id:
                    where_conditions.append("provider_id = :user_id")
                    params['user_id'] = user_id
                
                if workflow_instance_id:
                    where_conditions.append("workflow_instance_id = :workflow_instance_id")
                    params['workflow_instance_id'] = self._get_workflow_instance_db_id(workflow_instance_id)
                
                if event_type:
                    where_conditions.append("event_type = :event_type")
                    params['event_type'] = event_type
                
                if start_date:
                    where_conditions.append("timestamp >= :start_date")
                    params['start_date'] = start_date
                
                if end_date:
                    where_conditions.append("timestamp <= :end_date")
                    params['end_date'] = end_date
                
                where_clause = " AND ".join(where_conditions) if where_conditions else "1=1"
                
                query = text(f"""
                    SELECT * FROM clinical_audit_trail
                    WHERE {where_clause}
                    ORDER BY timestamp DESC
                    LIMIT :limit
                """)
                
                result = await session.execute(query, params)
                rows = result.fetchall()
                
                # Convert to dictionaries
                audit_entries = []
                for row in rows:
                    audit_entries.append(dict(row._mapping))
                
                logger.info(f"✅ Retrieved {len(audit_entries)} audit entries from database")
                return audit_entries
                
        except Exception as e:
            logger.error(f"❌ Failed to search audit trail: {e}")
            return []
    
    def _get_workflow_instance_db_id(self, workflow_instance_id: Optional[str]) -> Optional[int]:
        """Convert workflow instance ID to database ID (placeholder implementation)."""
        if not workflow_instance_id:
            return None
        
        # In a real implementation, this would query the workflow_instances table
        # For now, return None to avoid foreign key constraints
        return None
    
    def _extract_phi_fields(self, action_details: Dict[str, Any]) -> List[str]:
        """Extract PHI fields from action details."""
        phi_fields = []
        
        # Look for common PHI field indicators
        if isinstance(action_details, dict):
            for key, value in action_details.items():
                if any(phi_pattern in key.lower() for phi_pattern in [
                    'patient', 'phi', 'medical_record', 'ssn', 'diagnosis', 
                    'medication', 'allergy', 'lab_result'
                ]):
                    phi_fields.append(key)
        
        return phi_fields
    
    async def close(self):
        """Close database connections."""
        if self.engine:
            await self.engine.dispose()
            logger.info("✅ Database audit service connections closed")


# Global instance
database_audit_service = DatabaseAuditService()
