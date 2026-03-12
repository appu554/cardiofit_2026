"""
Test Database Security Integration
Tests the integration between security framework and database schema.
"""
import asyncio
import pytest
import json
import sys
import os
from datetime import datetime
from typing import Dict, Any

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

# Import security services directly
try:
    from app.security.phi_encryption import phi_encryption_service
    from app.security.audit_service import audit_service, AuditEventType, AuditLevel
    from app.security.break_glass_access import break_glass_access_service, EmergencyAccessType, EmergencyJustification
    print("✅ Security services imported successfully")
except ImportError as e:
    print(f"❌ Failed to import security services: {e}")
    sys.exit(1)


class TestDatabaseSecurityIntegration:
    """Test database integration with security framework."""
    
    @pytest.mark.asyncio
    async def test_audit_trail_database_storage(self):
        """Test that audit entries are stored in database."""
        print("🔍 Testing audit trail database storage...")
        
        # Log a workflow event
        audit_id = await audit_service.log_workflow_event(
            event_type=AuditEventType.WORKFLOW_STARTED,
            user_id="test_provider_123",
            workflow_instance_id="test_workflow_456",
            patient_id="test_patient_789",
            action_details={
                "workflow_type": "medication_prescribing",
                "medication": "Test Medication",
                "test_run": True
            },
            audit_level=AuditLevel.STANDARD,
            phi_accessed=True
        )
        
        # Wait a moment for database storage
        await asyncio.sleep(0.1)
        
        # Search for the audit entry in database
        search_results = await database_audit_service.search_audit_trail(
            patient_id="test_patient_789",
            user_id="test_provider_123",
            limit=10
        )
        
        # Verify audit entry was stored
        assert len(search_results) > 0, "Audit entry should be stored in database"
        
        # Find our specific audit entry
        our_entry = None
        for entry in search_results:
            if entry.get('action_details', {}).get('test_run') is True:
                our_entry = entry
                break
        
        assert our_entry is not None, "Our test audit entry should be found"
        assert our_entry['patient_id'] == "test_patient_789"
        assert our_entry['provider_id'] == "test_provider_123"
        assert our_entry['phi_accessed'] is True
        
        print(f"✅ Audit entry stored and retrieved: {audit_id}")
    
    @pytest.mark.asyncio
    async def test_phi_access_logging(self):
        """Test PHI access logging in database."""
        print("🔍 Testing PHI access database logging...")
        
        # Log PHI access
        phi_access_id = await database_audit_service.store_phi_access_log(
            user_id="test_doctor_456",
            patient_id="test_patient_123",
            access_type="decrypt_workflow_state",
            phi_fields=["patient_name", "diagnosis", "medication_history"],
            workflow_instance_id="test_workflow_789",
            session_id="test_session_123"
        )
        
        assert phi_access_id.startswith("phi_access_"), "PHI access ID should be generated"
        print(f"✅ PHI access logged: {phi_access_id}")
    
    @pytest.mark.asyncio
    async def test_clinical_decision_audit(self):
        """Test clinical decision audit storage."""
        print("🔍 Testing clinical decision audit storage...")
        
        # Store clinical decision
        decision_id = f"decision_{int(datetime.utcnow().timestamp() * 1000)}"
        
        stored_id = await database_audit_service.store_clinical_decision(
            decision_id=decision_id,
            decision_type="medication_prescription",
            decision_maker_id="test_doctor_789",
            patient_id="test_patient_456",
            clinical_context={
                "diagnosis": "Hypertension",
                "current_bp": "150/90",
                "weight": "75kg"
            },
            decision_details={
                "medication": "Lisinopril",
                "dosage": "10mg",
                "frequency": "daily"
            },
            clinical_rationale="Patient has elevated BP requiring ACE inhibitor therapy",
            safety_checks_performed=["drug_interaction", "allergy_check", "renal_function"],
            overrides_applied=["formulary_override"],
            supervisor_approval="supervisor_123"
        )
        
        assert stored_id == decision_id, "Decision ID should match"
        print(f"✅ Clinical decision stored: {decision_id}")
    
    @pytest.mark.asyncio
    async def test_encrypted_workflow_state_storage(self):
        """Test encrypted workflow state storage."""
        print("🔍 Testing encrypted workflow state storage...")
        
        # Create test workflow state
        workflow_state = {
            "patient_id": "test_patient_encrypted_123",
            "patient_name": "Test Patient",
            "workflow_type": "medication_prescribing",
            "clinical_data": {
                "diagnosis": "Diabetes Type 2",
                "medications": ["Metformin", "Insulin"]
            }
        }
        
        # Encrypt workflow state
        encrypted_state = await phi_encryption_service.encrypt_workflow_state(
            workflow_state, 
            "test_provider_456", 
            "test_workflow_encrypted_789"
        )
        
        # The encryption service should have automatically stored it in database
        # Let's verify by trying to store it again (should update)
        stored_id = await database_audit_service.store_encrypted_workflow_state(
            workflow_instance_id="test_workflow_encrypted_789",
            encrypted_state=encrypted_state,
            encryption_key_id="test-key-v1",
            phi_fields_encrypted=["patient_name", "diagnosis", "medications"],
            encrypted_by="test_provider_456"
        )
        
        assert stored_id == "test_workflow_encrypted_789", "Workflow instance ID should match"
        print(f"✅ Encrypted workflow state stored: {stored_id}")
    
    @pytest.mark.asyncio
    async def test_break_glass_database_integration(self):
        """Test break-glass access database integration."""
        print("🔍 Testing break-glass database integration...")
        
        # Initiate break-glass access
        result = await break_glass_access_service.initiate_break_glass_access(
            user_id="emergency_doctor_123",
            access_type=EmergencyAccessType.PATIENT_EMERGENCY,
            justification=EmergencyJustification.CARDIAC_ARREST,
            clinical_details="Patient in cardiac arrest, need immediate medication access",
            patient_id="emergency_patient_456",
            supervisor_approval="supervisor_emergency_789"
        )
        
        session_id = result["session_id"]
        
        # Log emergency action
        audit_id = await break_glass_access_service.log_break_glass_action(
            session_id=session_id,
            user_id="emergency_doctor_123",
            action_type="access_medication_history",
            action_details={
                "urgency": "critical",
                "data_accessed": "allergy_list",
                "test_emergency": True
            }
        )
        
        # Terminate session
        summary = await break_glass_access_service.terminate_break_glass_session(
            session_id=session_id,
            user_id="emergency_doctor_123",
            termination_reason="emergency_resolved"
        )
        
        # Verify audit entries were created in database
        search_results = await database_audit_service.search_audit_trail(
            patient_id="emergency_patient_456",
            user_id="emergency_doctor_123",
            limit=10
        )
        
        # Should find multiple audit entries for break-glass session
        break_glass_entries = [
            entry for entry in search_results 
            if entry.get('action_details', {}).get('test_emergency') is True
        ]
        
        assert len(break_glass_entries) > 0, "Break-glass audit entries should be in database"
        print(f"✅ Break-glass session completed with {len(break_glass_entries)} audit entries")
    
    @pytest.mark.asyncio
    async def test_audit_trail_search_functionality(self):
        """Test comprehensive audit trail search."""
        print("🔍 Testing audit trail search functionality...")
        
        # Create multiple audit entries for testing
        test_patient_id = "search_test_patient_123"
        test_user_id = "search_test_user_456"
        
        # Create different types of audit entries
        audit_entries = []
        
        # Workflow started
        audit_id_1 = await audit_service.log_workflow_event(
            event_type=AuditEventType.WORKFLOW_STARTED,
            user_id=test_user_id,
            workflow_instance_id="search_workflow_1",
            patient_id=test_patient_id,
            action_details={"workflow_type": "medication_prescribing", "search_test": True}
        )
        audit_entries.append(audit_id_1)
        
        # Clinical decision
        audit_id_2 = await audit_service.log_clinical_decision(
            user_id=test_user_id,
            patient_id=test_patient_id,
            workflow_instance_id="search_workflow_1",
            decision_type="medication_prescription",
            decision_details={"medication": "Search Test Med"},
            clinical_rationale="Search test rationale",
            safety_checks_performed=["search_test_check"]
        )
        audit_entries.append(audit_id_2)
        
        # Workflow completed
        audit_id_3 = await audit_service.log_workflow_event(
            event_type=AuditEventType.WORKFLOW_COMPLETED,
            user_id=test_user_id,
            workflow_instance_id="search_workflow_1",
            patient_id=test_patient_id,
            action_details={"status": "success", "search_test": True}
        )
        audit_entries.append(audit_id_3)
        
        # Wait for database storage
        await asyncio.sleep(0.2)
        
        # Search by patient ID
        patient_results = await database_audit_service.search_audit_trail(
            patient_id=test_patient_id,
            limit=20
        )
        
        search_test_entries = [
            entry for entry in patient_results 
            if entry.get('action_details', {}).get('search_test') is True
        ]
        
        assert len(search_test_entries) >= 2, f"Should find at least 2 search test entries, found {len(search_test_entries)}"
        
        # Search by user ID
        user_results = await database_audit_service.search_audit_trail(
            user_id=test_user_id,
            limit=20
        )
        
        assert len(user_results) >= 3, f"Should find at least 3 entries for user, found {len(user_results)}"
        
        # Search by event type
        workflow_results = await database_audit_service.search_audit_trail(
            event_type="workflow_started",
            limit=20
        )
        
        assert len(workflow_results) >= 1, "Should find workflow started events"
        
        print(f"✅ Audit trail search: {len(patient_results)} patient entries, {len(user_results)} user entries")


async def main():
    """Run all database security integration tests."""
    print("🔒 Testing Database Security Integration")
    print("=" * 60)
    
    # Test Database Security Integration
    print("\n📊 Testing Database Integration...")
    db_test = TestDatabaseSecurityIntegration()
    
    try:
        await db_test.test_audit_trail_database_storage()
        await db_test.test_phi_access_logging()
        await db_test.test_clinical_decision_audit()
        await db_test.test_encrypted_workflow_state_storage()
        await db_test.test_break_glass_database_integration()
        await db_test.test_audit_trail_search_functionality()
        
        print("\n" + "=" * 60)
        print("✅ All Database Security Integration Tests Completed Successfully!")
        print("🔒 PHI Encryption Database Integration: ✅ Working")
        print("📋 Audit Service Database Integration: ✅ Working") 
        print("🚨 Break-Glass Database Integration: ✅ Working")
        print("🔍 Audit Trail Search: ✅ Working")
        print("\n🎉 Database Security Integration Complete!")
        
    except Exception as e:
        print(f"\n❌ Database integration test failed: {e}")
        raise


if __name__ == "__main__":
    asyncio.run(main())
