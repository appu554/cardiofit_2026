"""
Test Security Framework Implementation
Tests PHI encryption, audit service, and break-glass access functionality.
"""
import asyncio
import pytest
import json
from datetime import datetime, timedelta
from typing import Dict, Any

# Import security services
from app.security.phi_encryption import PHIEncryptionService, phi_encryption_service
from app.security.audit_service import (
    AuditService, audit_service, AuditEventType, AuditLevel
)
from app.security.break_glass_access import (
    BreakGlassAccessService, break_glass_access_service,
    EmergencyAccessType, EmergencyJustification
)


class TestPHIEncryptionService:
    """Test PHI encryption and decryption functionality."""
    
    @pytest.mark.asyncio
    async def test_phi_field_identification(self):
        """Test automatic PHI field identification."""
        phi_service = PHIEncryptionService()
        
        # Test data with PHI fields
        test_data = {
            "patient_id": "12345",
            "patient_name": "John Doe",
            "workflow_id": "workflow_123",
            "clinical_data": {
                "diagnosis": "Diabetes Type 2",
                "medication_history": ["Metformin", "Insulin"],
                "lab_results": {"glucose": 180}
            },
            "non_phi_field": "some_value"
        }
        
        phi_fields = phi_service._identify_phi_fields(test_data)
        
        # Should identify PHI fields
        assert "patient_id" in phi_fields
        assert "patient_name" in phi_fields
        assert "clinical_data.diagnosis" in phi_fields
        assert "clinical_data.medication_history" in phi_fields
        assert "clinical_data.lab_results" in phi_fields
        
        # Should not identify non-PHI fields
        assert "workflow_id" not in phi_fields
        assert "non_phi_field" not in phi_fields
        
        print(f"✅ Identified PHI fields: {phi_fields}")
    
    @pytest.mark.asyncio
    async def test_workflow_state_encryption_decryption(self):
        """Test complete workflow state encryption and decryption."""
        phi_service = PHIEncryptionService()
        
        # Test workflow state with PHI
        workflow_state = {
            "patient_id": "patient_12345",
            "patient_name": "Jane Smith",
            "workflow_type": "medication_prescribing",
            "clinical_context": {
                "diagnosis": "Hypertension",
                "current_medications": ["Lisinopril", "Hydrochlorothiazide"],
                "allergies": ["Penicillin"],
                "lab_results": {
                    "creatinine": 1.2,
                    "potassium": 4.1
                }
            },
            "workflow_metadata": {
                "created_at": "2025-01-25T10:00:00Z",
                "provider_id": "provider_123"
            }
        }
        
        user_id = "test_user_123"
        workflow_instance_id = "workflow_instance_456"
        
        # Encrypt workflow state
        encrypted_state_json = await phi_service.encrypt_workflow_state(
            workflow_state, user_id, workflow_instance_id
        )
        
        # Verify encryption metadata is added
        encrypted_state = json.loads(encrypted_state_json)
        assert "_phi_encryption" in encrypted_state
        assert encrypted_state["_phi_encryption"]["encrypted_by"] == user_id
        assert encrypted_state["_phi_encryption"]["workflow_instance_id"] == workflow_instance_id
        
        # Verify PHI fields are encrypted
        assert encrypted_state["patient_name"]["_encrypted"] is True
        assert "_value" in encrypted_state["patient_name"]
        
        # Decrypt workflow state
        decrypted_state = await phi_service.decrypt_workflow_state(
            encrypted_state_json, user_id, workflow_instance_id
        )
        
        # Verify decryption restores original data
        assert decrypted_state["patient_id"] == workflow_state["patient_id"]
        assert decrypted_state["patient_name"] == workflow_state["patient_name"]
        assert decrypted_state["clinical_context"]["diagnosis"] == workflow_state["clinical_context"]["diagnosis"]
        
        # Verify non-PHI fields are unchanged
        assert decrypted_state["workflow_type"] == workflow_state["workflow_type"]
        assert decrypted_state["workflow_metadata"] == workflow_state["workflow_metadata"]
        
        print("✅ PHI encryption/decryption successful")
    
    @pytest.mark.asyncio
    async def test_phi_access_audit(self):
        """Test PHI access audit logging."""
        phi_service = PHIEncryptionService()
        
        # Test PHI access audit
        await phi_service.audit_phi_access(
            user_id="test_user_123",
            patient_id="patient_456",
            action="decrypt_workflow_state",
            workflow_instance_id="workflow_789",
            phi_fields=["patient_name", "diagnosis", "medication_history"]
        )
        
        # Verify audit log entry
        assert len(phi_service.phi_access_log) > 0
        latest_entry = phi_service.phi_access_log[-1]
        
        assert latest_entry["user_id"] == "test_user_123"
        assert latest_entry["patient_id"] == "patient_456"
        assert latest_entry["action"] == "decrypt_workflow_state"
        assert latest_entry["phi_fields_count"] == 3
        
        print("✅ PHI access audit logging successful")


class TestAuditService:
    """Test comprehensive audit service functionality."""
    
    @pytest.mark.asyncio
    async def test_workflow_event_logging(self):
        """Test workflow event audit logging."""
        audit_svc = AuditService()
        
        # Test workflow event logging
        audit_id = await audit_svc.log_workflow_event(
            event_type=AuditEventType.WORKFLOW_STARTED,
            user_id="provider_123",
            workflow_instance_id="workflow_456",
            patient_id="patient_789",
            action_details={
                "workflow_type": "medication_prescribing",
                "medication": "Metformin",
                "dosage": "500mg"
            },
            audit_level=AuditLevel.STANDARD,
            session_id="session_123"
        )
        
        # Verify audit entry created
        assert audit_id.startswith("audit_")
        assert len(audit_svc.audit_entries) > 0
        
        latest_entry = audit_svc.audit_entries[-1]
        assert latest_entry.event_type == AuditEventType.WORKFLOW_STARTED
        assert latest_entry.user_id == "provider_123"
        assert latest_entry.patient_id == "patient_789"
        assert latest_entry.workflow_instance_id == "workflow_456"
        
        print(f"✅ Workflow event logged with ID: {audit_id}")
    
    @pytest.mark.asyncio
    async def test_clinical_decision_logging(self):
        """Test clinical decision audit logging."""
        audit_svc = AuditService()
        
        # Test clinical decision logging
        audit_id = await audit_svc.log_clinical_decision(
            user_id="doctor_123",
            patient_id="patient_456",
            workflow_instance_id="workflow_789",
            decision_type="medication_prescription",
            decision_details={
                "medication": "Lisinopril",
                "dosage": "10mg",
                "frequency": "daily"
            },
            clinical_rationale="Patient has hypertension with BP 150/90",
            safety_checks_performed=["drug_interaction", "allergy_check", "dosage_validation"],
            overrides_applied=["formulary_override"]
        )
        
        # Verify clinical decision audit
        assert audit_id.startswith("audit_")
        latest_entry = audit_svc.audit_entries[-1]
        
        assert latest_entry.event_type == AuditEventType.CLINICAL_DECISION
        assert latest_entry.audit_level == AuditLevel.COMPREHENSIVE
        assert latest_entry.safety_critical is True  # Due to overrides
        assert "clinical_rationale" in latest_entry.action_details
        
        print(f"✅ Clinical decision logged with ID: {audit_id}")
    
    @pytest.mark.asyncio
    async def test_safety_override_logging(self):
        """Test safety override audit logging."""
        audit_svc = AuditService()
        
        # Test safety override logging
        audit_id = await audit_svc.log_safety_override(
            user_id="doctor_456",
            patient_id="patient_789",
            workflow_instance_id="workflow_123",
            override_type="drug_interaction_override",
            safety_warning="Potential interaction between Warfarin and Aspirin",
            clinical_justification="Patient requires anticoagulation for atrial fibrillation, benefits outweigh risks",
            supervisor_approval="supervisor_789"
        )
        
        # Verify safety override audit
        assert audit_id.startswith("audit_")
        latest_entry = audit_svc.audit_entries[-1]
        
        assert latest_entry.event_type == AuditEventType.SAFETY_OVERRIDE
        assert latest_entry.audit_level == AuditLevel.COMPREHENSIVE
        assert latest_entry.safety_critical is True
        assert "supervisor_approval" in latest_entry.action_details
        
        print(f"✅ Safety override logged with ID: {audit_id}")
    
    @pytest.mark.asyncio
    async def test_audit_trail_search(self):
        """Test audit trail search functionality."""
        audit_svc = AuditService()
        
        # Add multiple audit entries
        patient_id = "search_test_patient"
        
        await audit_svc.log_workflow_event(
            event_type=AuditEventType.WORKFLOW_STARTED,
            user_id="user_1",
            workflow_instance_id="workflow_1",
            patient_id=patient_id,
            action_details={"test": "entry_1"}
        )
        
        await audit_svc.log_workflow_event(
            event_type=AuditEventType.WORKFLOW_COMPLETED,
            user_id="user_1",
            workflow_instance_id="workflow_1",
            patient_id=patient_id,
            action_details={"test": "entry_2"}
        )
        
        # Search audit trail
        search_results = await audit_svc.search_audit_trail(
            patient_id=patient_id,
            limit=10
        )
        
        # Verify search results
        assert len(search_results) >= 2
        assert all(entry["patient_id"] == patient_id for entry in search_results)
        
        print(f"✅ Audit trail search returned {len(search_results)} entries")


class TestBreakGlassAccessService:
    """Test break-glass emergency access functionality."""
    
    @pytest.mark.asyncio
    async def test_initiate_break_glass_access(self):
        """Test break-glass access initiation."""
        bg_service = BreakGlassAccessService()
        
        # Test break-glass access initiation
        result = await bg_service.initiate_break_glass_access(
            user_id="emergency_user_123",
            access_type=EmergencyAccessType.PATIENT_EMERGENCY,
            justification=EmergencyJustification.CARDIAC_ARREST,
            clinical_details="Patient in cardiac arrest, need immediate access to medication history",
            patient_id="emergency_patient_456",
            supervisor_approval="supervisor_789"
        )
        
        # Verify break-glass session created
        assert result["access_granted"] is True
        assert "session_id" in result
        assert "expires_at" in result
        assert result["timeout_minutes"] == 30
        
        session_id = result["session_id"]
        assert session_id in bg_service.active_sessions
        
        session = bg_service.active_sessions[session_id]
        assert session.user_id == "emergency_user_123"
        assert session.patient_id == "emergency_patient_456"
        assert session.is_active is True
        
        print(f"✅ Break-glass access initiated: {session_id}")
        return session_id
    
    @pytest.mark.asyncio
    async def test_validate_break_glass_session(self):
        """Test break-glass session validation."""
        bg_service = BreakGlassAccessService()
        
        # First initiate a session
        result = await bg_service.initiate_break_glass_access(
            user_id="test_user_456",
            access_type=EmergencyAccessType.SYSTEM_FAILURE,
            justification=EmergencyJustification.SYSTEM_OUTAGE,
            clinical_details="System outage, need emergency access",
            patient_id="patient_123"
        )
        
        session_id = result["session_id"]
        
        # Test valid session validation
        is_valid = await bg_service.validate_break_glass_session(session_id, "test_user_456")
        assert is_valid is True
        
        # Test invalid user validation
        is_valid = await bg_service.validate_break_glass_session(session_id, "wrong_user")
        assert is_valid is False
        
        # Test non-existent session validation
        is_valid = await bg_service.validate_break_glass_session("fake_session", "test_user_456")
        assert is_valid is False
        
        print("✅ Break-glass session validation successful")
    
    @pytest.mark.asyncio
    async def test_log_break_glass_action(self):
        """Test logging actions during break-glass session."""
        bg_service = BreakGlassAccessService()
        
        # Initiate break-glass session
        result = await bg_service.initiate_break_glass_access(
            user_id="action_user_789",
            access_type=EmergencyAccessType.DATA_ACCESS,
            justification=EmergencyJustification.OTHER_EMERGENCY,
            clinical_details="Emergency data access needed",
            patient_id="patient_456"
        )
        
        session_id = result["session_id"]
        
        # Log break-glass action
        audit_id = await bg_service.log_break_glass_action(
            session_id=session_id,
            user_id="action_user_789",
            action_type="access_patient_data",
            action_details={
                "data_accessed": "medication_history",
                "reason": "emergency_treatment"
            },
            workflow_instance_id="emergency_workflow_123"
        )
        
        # Verify action logged
        assert audit_id.startswith("audit_")
        
        session = bg_service.active_sessions[session_id]
        assert len(session.actions_performed) == 1
        assert session.actions_performed[0]["action_type"] == "access_patient_data"
        
        print(f"✅ Break-glass action logged: {audit_id}")
    
    @pytest.mark.asyncio
    async def test_terminate_break_glass_session(self):
        """Test break-glass session termination."""
        bg_service = BreakGlassAccessService()
        
        # Initiate break-glass session
        result = await bg_service.initiate_break_glass_access(
            user_id="terminate_user_123",
            access_type=EmergencyAccessType.WORKFLOW_OVERRIDE,
            justification=EmergencyJustification.SEPSIS,
            clinical_details="Sepsis protocol override needed",
            patient_id="patient_789"
        )
        
        session_id = result["session_id"]
        
        # Log some actions
        await bg_service.log_break_glass_action(
            session_id=session_id,
            user_id="terminate_user_123",
            action_type="override_safety_check",
            action_details={"check_type": "drug_interaction"}
        )
        
        # Terminate session
        summary = await bg_service.terminate_break_glass_session(
            session_id=session_id,
            user_id="terminate_user_123",
            termination_reason="emergency_resolved"
        )
        
        # Verify session terminated
        assert summary["session_id"] == session_id
        assert summary["termination_reason"] == "emergency_resolved"
        assert len(summary["actions_performed"]) == 1
        assert session_id not in bg_service.active_sessions
        
        print(f"✅ Break-glass session terminated: {session_id}")


async def main():
    """Run all security framework tests."""
    print("🔒 Testing Clinical Workflow Engine Security Framework")
    print("=" * 60)
    
    # Test PHI Encryption Service
    print("\n📊 Testing PHI Encryption Service...")
    phi_test = TestPHIEncryptionService()
    await phi_test.test_phi_field_identification()
    await phi_test.test_workflow_state_encryption_decryption()
    await phi_test.test_phi_access_audit()
    
    # Test Audit Service
    print("\n📋 Testing Audit Service...")
    audit_test = TestAuditService()
    await audit_test.test_workflow_event_logging()
    await audit_test.test_clinical_decision_logging()
    await audit_test.test_safety_override_logging()
    await audit_test.test_audit_trail_search()
    
    # Test Break-Glass Access Service
    print("\n🚨 Testing Break-Glass Access Service...")
    bg_test = TestBreakGlassAccessService()
    await bg_test.test_initiate_break_glass_access()
    await bg_test.test_validate_break_glass_session()
    await bg_test.test_log_break_glass_action()
    await bg_test.test_terminate_break_glass_session()
    
    print("\n" + "=" * 60)
    print("✅ All Security Framework Tests Completed Successfully!")
    print("🔒 PHI Encryption: ✅ Working")
    print("📋 Audit Service: ✅ Working") 
    print("🚨 Break-Glass Access: ✅ Working")
    print("\n🎉 Security Framework Implementation Complete!")


if __name__ == "__main__":
    asyncio.run(main())
