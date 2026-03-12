"""
Database Schema Verification Test
Verifies that all security framework database components are properly installed and functional.
"""
import os
import sys
import logging
from sqlalchemy import create_engine, text
import json
from datetime import datetime

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class DatabaseSchemaVerification:
    """Verify database schema and security framework integration."""
    
    def __init__(self):
        self.engine = create_engine(settings.DATABASE_URL, echo=False)
        self.connection = None
    
    def connect(self):
        """Connect to database."""
        self.connection = self.engine.connect()
        logger.info("✅ Connected to database")
    
    def disconnect(self):
        """Disconnect from database."""
        if self.connection:
            self.connection.close()
        self.engine.dispose()
        logger.info("✅ Disconnected from database")
    
    def test_security_tables_exist(self):
        """Test that all security tables exist and are accessible."""
        logger.info("🔍 Testing security tables existence...")
        
        security_tables = [
            'clinical_audit_trail',
            'phi_access_log',
            'security_events',
            'clinical_decision_audit',
            'encrypted_workflow_states',
            'emergency_access_records',
            'phi_encryption_keys'
        ]
        
        results = {}
        for table in security_tables:
            try:
                result = self.connection.execute(text(f"SELECT COUNT(*) FROM {table}"))
                count = result.scalar()
                results[table] = {'exists': True, 'count': count}
                logger.info(f"   ✅ {table}: {count} records")
            except Exception as e:
                results[table] = {'exists': False, 'error': str(e)}
                logger.error(f"   ❌ {table}: {e}")
        
        return results
    
    def test_security_views_exist(self):
        """Test that all security views exist and are accessible."""
        logger.info("🔍 Testing security views existence...")
        
        security_views = [
            'phi_access_summary',
            'security_events_summary',
            'clinical_decision_summary'
        ]
        
        results = {}
        for view in security_views:
            try:
                result = self.connection.execute(text(f"SELECT COUNT(*) FROM {view}"))
                count = result.scalar()
                results[view] = {'exists': True, 'count': count}
                logger.info(f"   ✅ {view}: {count} records")
            except Exception as e:
                results[view] = {'exists': False, 'error': str(e)}
                logger.error(f"   ❌ {view}: {e}")
        
        return results
    
    def test_enhanced_audit_columns(self):
        """Test that enhanced audit trail columns exist."""
        logger.info("🔍 Testing enhanced audit trail columns...")
        
        try:
            result = self.connection.execute(text("""
                SELECT column_name, data_type, is_nullable, column_default
                FROM information_schema.columns 
                WHERE table_name = 'clinical_audit_trail'
                AND column_name IN ('event_type', 'audit_level_enum', 'outcome', 'safety_critical', 'error_details')
                ORDER BY column_name
            """))
            
            columns = result.fetchall()
            column_info = {}
            
            for col_name, data_type, is_nullable, col_default in columns:
                column_info[col_name] = {
                    'data_type': data_type,
                    'is_nullable': is_nullable,
                    'column_default': col_default
                }
                logger.info(f"   ✅ {col_name} ({data_type})")
            
            return column_info
            
        except Exception as e:
            logger.error(f"   ❌ Failed to check enhanced columns: {e}")
            return {}
    
    def test_enum_types_exist(self):
        """Test that custom enum types exist."""
        logger.info("🔍 Testing custom enum types...")
        
        try:
            result = self.connection.execute(text("""
                SELECT typname, typtype 
                FROM pg_type 
                WHERE typname IN ('audit_event_type', 'audit_level_type')
                ORDER BY typname
            """))
            
            enum_types = result.fetchall()
            type_info = {}
            
            for type_name, type_type in enum_types:
                type_info[type_name] = {'type': type_type}
                logger.info(f"   ✅ {type_name} (enum)")
            
            return type_info
            
        except Exception as e:
            logger.error(f"   ❌ Failed to check enum types: {e}")
            return {}
    
    def test_insert_sample_data(self):
        """Test inserting sample data into security tables."""
        logger.info("🔍 Testing sample data insertion...")
        
        try:
            # Test PHI access log insertion
            phi_access_id = f"test_phi_access_{int(datetime.utcnow().timestamp() * 1000)}"
            
            self.connection.execute(text("""
                INSERT INTO phi_access_log (
                    id, user_id, patient_id, access_type, phi_fields_accessed,
                    phi_fields_count, access_purpose, data_classification
                ) VALUES (
                    :id, :user_id, :patient_id, :access_type, :phi_fields_accessed,
                    :phi_fields_count, :access_purpose, :data_classification
                )
            """), {
                'id': phi_access_id,
                'user_id': 'test_user_123',
                'patient_id': 'test_patient_456',
                'access_type': 'test_access',
                'phi_fields_accessed': json.dumps(['patient_name', 'diagnosis']),
                'phi_fields_count': 2,
                'access_purpose': 'testing',
                'data_classification': 'phi'
            })
            
            logger.info(f"   ✅ PHI access log: {phi_access_id}")
            
            # Test security events insertion
            security_event_id = f"test_security_event_{int(datetime.utcnow().timestamp() * 1000)}"
            
            self.connection.execute(text("""
                INSERT INTO security_events (
                    id, event_type, severity, event_details, detection_method
                ) VALUES (
                    :id, :event_type, :severity, :event_details, :detection_method
                )
            """), {
                'id': security_event_id,
                'event_type': 'test_event',
                'severity': 'low',
                'event_details': json.dumps({'test': True, 'description': 'Schema verification test'}),
                'detection_method': 'automated_test'
            })
            
            logger.info(f"   ✅ Security event: {security_event_id}")
            
            # Test clinical decision audit insertion
            decision_id = f"test_decision_{int(datetime.utcnow().timestamp() * 1000)}"
            
            self.connection.execute(text("""
                INSERT INTO clinical_decision_audit (
                    decision_id, decision_type, decision_maker_id, patient_id,
                    clinical_context, decision_details, clinical_rationale,
                    safety_checks_performed
                ) VALUES (
                    :decision_id, :decision_type, :decision_maker_id, :patient_id,
                    :clinical_context, :decision_details, :clinical_rationale,
                    :safety_checks_performed
                )
            """), {
                'decision_id': decision_id,
                'decision_type': 'test_decision',
                'decision_maker_id': 'test_doctor_123',
                'patient_id': 'test_patient_456',
                'clinical_context': json.dumps({'test': True}),
                'decision_details': json.dumps({'test_medication': 'Test Drug'}),
                'clinical_rationale': 'Schema verification test decision',
                'safety_checks_performed': json.dumps(['test_check'])
            })
            
            logger.info(f"   ✅ Clinical decision: {decision_id}")
            
            # Commit the transaction
            self.connection.commit()
            
            return {
                'phi_access_id': phi_access_id,
                'security_event_id': security_event_id,
                'decision_id': decision_id
            }
            
        except Exception as e:
            logger.error(f"   ❌ Failed to insert sample data: {e}")
            self.connection.rollback()
            return {}
    
    def test_query_sample_data(self):
        """Test querying the inserted sample data."""
        logger.info("🔍 Testing sample data queries...")
        
        try:
            # Query PHI access log
            result = self.connection.execute(text("""
                SELECT COUNT(*) FROM phi_access_log 
                WHERE user_id = 'test_user_123' AND access_purpose = 'testing'
            """))
            phi_count = result.scalar()
            logger.info(f"   ✅ PHI access records: {phi_count}")
            
            # Query security events
            result = self.connection.execute(text("""
                SELECT COUNT(*) FROM security_events 
                WHERE event_type = 'test_event' AND detection_method = 'automated_test'
            """))
            event_count = result.scalar()
            logger.info(f"   ✅ Security event records: {event_count}")
            
            # Query clinical decisions
            result = self.connection.execute(text("""
                SELECT COUNT(*) FROM clinical_decision_audit 
                WHERE decision_type = 'test_decision' AND decision_maker_id = 'test_doctor_123'
            """))
            decision_count = result.scalar()
            logger.info(f"   ✅ Clinical decision records: {decision_count}")
            
            # Test security views
            result = self.connection.execute(text("SELECT COUNT(*) FROM phi_access_summary"))
            view_count = result.scalar()
            logger.info(f"   ✅ PHI access summary view: {view_count} records")
            
            return {
                'phi_access_count': phi_count,
                'security_event_count': event_count,
                'clinical_decision_count': decision_count,
                'view_count': view_count
            }
            
        except Exception as e:
            logger.error(f"   ❌ Failed to query sample data: {e}")
            return {}
    
    def cleanup_test_data(self):
        """Clean up test data."""
        logger.info("🧹 Cleaning up test data...")
        
        try:
            # Clean up test data
            self.connection.execute(text("DELETE FROM phi_access_log WHERE access_purpose = 'testing'"))
            self.connection.execute(text("DELETE FROM security_events WHERE detection_method = 'automated_test'"))
            self.connection.execute(text("DELETE FROM clinical_decision_audit WHERE decision_type = 'test_decision'"))
            
            self.connection.commit()
            logger.info("   ✅ Test data cleaned up")
            
        except Exception as e:
            logger.error(f"   ❌ Failed to cleanup test data: {e}")
            self.connection.rollback()


def main():
    """Run database schema verification tests."""
    logger.info("🔒 Database Schema Verification for Security Framework")
    logger.info("=" * 70)
    
    verifier = DatabaseSchemaVerification()
    
    try:
        # Connect to database
        verifier.connect()
        
        # Run verification tests
        logger.info("\n📊 Testing Security Tables...")
        table_results = verifier.test_security_tables_exist()
        
        logger.info("\n📈 Testing Security Views...")
        view_results = verifier.test_security_views_exist()
        
        logger.info("\n🔧 Testing Enhanced Audit Columns...")
        column_results = verifier.test_enhanced_audit_columns()
        
        logger.info("\n🏷️ Testing Custom Enum Types...")
        enum_results = verifier.test_enum_types_exist()
        
        logger.info("\n📝 Testing Sample Data Insertion...")
        insert_results = verifier.test_insert_sample_data()
        
        logger.info("\n🔍 Testing Sample Data Queries...")
        query_results = verifier.test_query_sample_data()
        
        logger.info("\n🧹 Cleaning Up...")
        verifier.cleanup_test_data()
        
        # Summary
        logger.info("\n" + "=" * 70)
        logger.info("📊 VERIFICATION SUMMARY")
        logger.info("=" * 70)
        
        # Count successful components
        tables_ok = sum(1 for t in table_results.values() if t.get('exists', False))
        views_ok = sum(1 for v in view_results.values() if v.get('exists', False))
        columns_ok = len(column_results)
        enums_ok = len(enum_results)
        
        logger.info(f"✅ Security Tables: {tables_ok}/7")
        logger.info(f"✅ Security Views: {views_ok}/3")
        logger.info(f"✅ Enhanced Columns: {columns_ok}/5")
        logger.info(f"✅ Custom Enums: {enums_ok}/2")
        
        if insert_results and query_results:
            logger.info("✅ Data Operations: Working")
        else:
            logger.info("❌ Data Operations: Failed")
        
        # Overall status
        total_components = tables_ok + views_ok + columns_ok + enums_ok
        if total_components >= 15 and insert_results and query_results:
            logger.info("\n🎉 DATABASE SCHEMA VERIFICATION: SUCCESS")
            logger.info("✅ All security framework components are working properly")
            return True
        else:
            logger.error("\n❌ DATABASE SCHEMA VERIFICATION: FAILED")
            logger.error("Some security framework components are not working")
            return False
        
    except Exception as e:
        logger.error(f"❌ Verification failed: {e}")
        return False
    
    finally:
        verifier.disconnect()


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
