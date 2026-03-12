"""
Setup script for Clinical Workflow Engine database.
Creates all necessary tables and initial data for clinical workflows.
"""
import os
import sys
from pathlib import Path
import logging
from sqlalchemy import create_engine, text
from sqlalchemy.exc import SQLAlchemyError

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def connect_to_database():
    """
    Connect to the Supabase PostgreSQL database using SQLAlchemy.
    """
    try:
        engine = create_engine(
            settings.DATABASE_URL,
            pool_pre_ping=True,
            echo=False
        )
        connection = engine.connect()
        logger.info("✅ Connected to Supabase PostgreSQL database")
        return connection, engine
    except Exception as e:
        logger.error(f"❌ Failed to connect to database: {e}")
        return None, None


def run_migration_file(connection, migration_file):
    """
    Run a single migration file.
    """
    try:
        migration_path = Path(__file__).parent / "migrations" / migration_file

        if not migration_path.exists():
            logger.error(f"❌ Migration file not found: {migration_file}")
            return False

        logger.info(f"🔄 Running migration: {migration_file}")

        with open(migration_path, 'r', encoding='utf-8') as f:
            sql_content = f.read()

        # Execute the migration
        connection.execute(text(sql_content))
        connection.commit()
        logger.info(f"✅ Migration completed: {migration_file}")
        return True

    except Exception as e:
        logger.error(f"❌ Migration failed {migration_file}: {e}")
        return False


def create_clinical_workflow_data(connection):
    """
    Create initial clinical workflow definitions and sample data.
    """
    try:
        logger.info("🔄 Creating initial clinical workflow data...")

        # Create medication ordering workflow definition
        medication_workflow_sql = """
        INSERT INTO workflow_definitions (
            fhir_id, name, version, status, category, description, created_by
        ) VALUES (
            'medication-ordering-workflow-v1',
            'Medication Ordering Workflow',
            '1.0.0',
            'active',
            'clinical-protocol',
            'Clinical workflow for safe medication ordering with safety checks',
            'system'
        ) ON CONFLICT (fhir_id) DO NOTHING;
        """

        connection.execute(text(medication_workflow_sql))
        
        # Create admission workflow definition
        admission_workflow_sql = """
        INSERT INTO workflow_definitions (
            fhir_id, name, version, status, category, description, created_by
        ) VALUES (
            'patient-admission-workflow-v1',
            'Patient Admission Workflow',
            '1.0.0',
            'active',
            'clinical-protocol',
            'Clinical workflow for patient admission with parallel processing',
            'system'
        ) ON CONFLICT (fhir_id) DO NOTHING;
        """

        connection.execute(text(admission_workflow_sql))

        # Create discharge workflow definition
        discharge_workflow_sql = """
        INSERT INTO workflow_definitions (
            fhir_id, name, version, status, category, description, created_by
        ) VALUES (
            'patient-discharge-workflow-v1',
            'Patient Discharge Workflow',
            '1.0.0',
            'active',
            'clinical-protocol',
            'Clinical workflow for patient discharge with medication reconciliation',
            'system'
        ) ON CONFLICT (fhir_id) DO NOTHING;
        """

        connection.execute(text(discharge_workflow_sql))

        # Create initial PHI encryption key
        encryption_key_sql = """
        INSERT INTO phi_encryption_keys (
            key_id, key_version, encrypted_key, algorithm, created_by, status
        ) VALUES (
            'clinical-phi-key-v1',
            1,
            'encrypted_master_key_placeholder', -- This should be properly encrypted in production
            'AES-256-GCM',
            'system',
            'active'
        ) ON CONFLICT (key_id) DO NOTHING;
        """

        connection.execute(text(encryption_key_sql))
        connection.commit()
        
        logger.info("✅ Initial clinical workflow data created")
        return True
        
    except Exception as e:
        logger.error(f"❌ Failed to create initial data: {e}")
        return False


def verify_database_setup(connection):
    """
    Verify that all tables and data are properly set up.
    """
    try:
        logger.info("🔍 Verifying database setup...")

        # Check core workflow tables
        core_tables = [
            'workflow_definitions',
            'workflow_instances',
            'workflow_tasks',
            'workflow_events'
        ]

        # Check clinical workflow tables
        clinical_tables = [
            'clinical_activity_executions',
            'clinical_audit_trail',
            'clinical_errors',
            'clinical_workflow_metrics',
            'emergency_access_records',
            'phi_encryption_keys',
            'clinical_timers'
        ]

        all_tables = core_tables + clinical_tables

        for table in all_tables:
            result = connection.execute(text(
                "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = :table_name"
            ), {"table_name": table}).fetchone()

            if result and result[0] > 0:
                logger.info(f"✅ Table exists: {table}")
            else:
                logger.error(f"❌ Table missing: {table}")
                return False

        # Check workflow definitions
        workflow_result = connection.execute(text(
            "SELECT COUNT(*) FROM workflow_definitions"
        )).fetchone()
        workflow_count = workflow_result[0] if workflow_result else 0
        logger.info(f"📊 Workflow definitions: {workflow_count}")

        # Check PHI encryption keys
        key_result = connection.execute(text(
            "SELECT COUNT(*) FROM phi_encryption_keys WHERE status = 'active'"
        )).fetchone()
        key_count = key_result[0] if key_result else 0
        logger.info(f"🔐 Active encryption keys: {key_count}")

        logger.info("✅ Database setup verification completed")
        return True

    except Exception as e:
        logger.error(f"❌ Database verification failed: {e}")
        return False


def main():
    """
    Main setup function for Clinical Workflow Engine database.
    """
    logger.info("🏥 Clinical Workflow Engine Database Setup")
    logger.info("=" * 60)

    # Connect to database
    connection, engine = connect_to_database()
    if not connection:
        logger.error("❌ Cannot proceed without database connection")
        return False
    
    try:
        # Run migrations in order
        migrations = [
            "001_create_workflow_tables.sql",
            "002_add_missing_fields.sql", 
            "003_phase4_integration_tables.sql",
            "004_phase5_advanced_features.sql",
            "005_clinical_workflow_engine_tables.sql"
        ]
        
        logger.info("🔄 Running database migrations...")
        
        for migration in migrations:
            success = run_migration_file(connection, migration)
            if not success:
                logger.error(f"❌ Migration failed: {migration}")
                return False

        logger.info("✅ All migrations completed successfully")

        # Create initial clinical workflow data
        success = create_clinical_workflow_data(connection)
        if not success:
            logger.error("❌ Failed to create initial data")
            return False

        # Verify setup
        success = verify_database_setup(connection)
        if not success:
            logger.error("❌ Database verification failed")
            return False
        
        logger.info("=" * 60)
        logger.info("🎉 Clinical Workflow Engine Database Setup Complete!")
        logger.info("✅ All tables created successfully")
        logger.info("✅ Initial workflow definitions added")
        logger.info("✅ PHI encryption keys configured")
        logger.info("✅ Clinical audit trail ready")
        logger.info("✅ Performance metrics tables ready")
        logger.info("✅ Emergency access system ready")
        logger.info("")
        logger.info("📋 Database Summary:")
        logger.info("   - Core workflow tables: ✅")
        logger.info("   - Clinical activity tracking: ✅") 
        logger.info("   - Enhanced audit trail: ✅")
        logger.info("   - PHI encryption support: ✅")
        logger.info("   - Performance metrics: ✅")
        logger.info("   - Break-glass access: ✅")
        logger.info("")
        logger.info("🚀 Ready to start Clinical Workflow Engine!")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Setup failed: {e}")
        return False
        
    finally:
        if connection:
            connection.close()
        if engine:
            engine.dispose()
        logger.info("🔌 Database connection closed")


if __name__ == "__main__":
    success = main()
    if success:
        print("\n✅ Database setup completed successfully!")
        sys.exit(0)
    else:
        print("\n❌ Database setup failed!")
        sys.exit(1)
