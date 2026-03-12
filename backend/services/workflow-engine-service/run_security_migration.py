"""
Run Security Framework Database Migration
Applies the security framework enhancements to the database schema.
"""
import os
import sys
import logging
from sqlalchemy import create_engine, text
import asyncio

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def run_security_migration():
    """
    Run the security framework database migration.
    """
    try:
        logger.info("🔒 Starting Security Framework Database Migration")
        logger.info("=" * 60)
        
        # Connect to database
        engine = create_engine(settings.DATABASE_URL, echo=False)
        connection = engine.connect()
        
        # Read migration file
        migration_file = os.path.join(
            os.path.dirname(__file__),
            'migrations',
            '006_security_framework_enhancements_simple.sql'
        )
        
        if not os.path.exists(migration_file):
            logger.error(f"❌ Migration file not found: {migration_file}")
            return False
        
        logger.info(f"📄 Reading migration file: {migration_file}")
        
        with open(migration_file, 'r', encoding='utf-8') as f:
            migration_sql = f.read()
        
        # Execute migration in transaction
        logger.info("🔄 Executing security framework migration...")
        
        trans = connection.begin()
        try:
            # Split SQL into individual statements
            statements = [stmt.strip() for stmt in migration_sql.split(';') if stmt.strip()]
            
            for i, statement in enumerate(statements, 1):
                if statement.strip():
                    logger.info(f"   Executing statement {i}/{len(statements)}")
                    connection.execute(text(statement))
            
            trans.commit()
            logger.info("✅ Security framework migration completed successfully!")
            
        except Exception as e:
            trans.rollback()
            logger.error(f"❌ Migration failed, rolled back: {e}")
            raise
        
        # Verify new tables exist
        logger.info("\n🔍 Verifying new security tables...")
        
        security_tables = [
            'phi_access_log',
            'security_events', 
            'clinical_decision_audit',
            'encrypted_workflow_states'
        ]
        
        for table in security_tables:
            try:
                result = connection.execute(text(f"SELECT COUNT(*) FROM {table}"))
                count = result.scalar()
                logger.info(f"   ✅ {table}: {count} records")
            except Exception as e:
                logger.error(f"   ❌ {table}: Error - {e}")
        
        # Verify enhanced columns
        logger.info("\n🔍 Verifying enhanced audit trail columns...")
        
        try:
            result = connection.execute(text("""
                SELECT column_name 
                FROM information_schema.columns 
                WHERE table_name = 'clinical_audit_trail' 
                AND column_name IN ('event_type', 'audit_level_enum', 'outcome', 'safety_critical')
                ORDER BY column_name
            """))
            
            enhanced_columns = [row[0] for row in result]
            logger.info(f"   ✅ Enhanced columns: {', '.join(enhanced_columns)}")
            
        except Exception as e:
            logger.error(f"   ❌ Failed to verify enhanced columns: {e}")
        
        # Verify security views
        logger.info("\n🔍 Verifying security views...")
        
        security_views = [
            'phi_access_summary',
            'security_events_summary',
            'clinical_decision_summary'
        ]
        
        for view in security_views:
            try:
                result = connection.execute(text(f"SELECT COUNT(*) FROM {view}"))
                count = result.scalar()
                logger.info(f"   ✅ {view}: {count} records")
            except Exception as e:
                logger.error(f"   ❌ {view}: Error - {e}")
        
        connection.close()
        engine.dispose()
        
        logger.info("\n" + "=" * 60)
        logger.info("🎉 Security Framework Database Migration Complete!")
        logger.info("✅ All security tables and enhancements applied")
        logger.info("🔒 Database ready for security framework integration")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Security migration failed: {e}")
        return False


def verify_security_schema():
    """
    Verify that all security framework components are properly installed.
    """
    try:
        logger.info("\n🔍 Security Framework Schema Verification")
        logger.info("=" * 50)
        
        engine = create_engine(settings.DATABASE_URL, echo=False)
        connection = engine.connect()
        
        # Check required tables
        required_tables = [
            'clinical_audit_trail',
            'phi_access_log',
            'security_events',
            'clinical_decision_audit', 
            'encrypted_workflow_states',
            'emergency_access_records',
            'phi_encryption_keys'
        ]
        
        logger.info("📋 Checking required security tables:")
        missing_tables = []
        
        for table in required_tables:
            try:
                result = connection.execute(text(f"SELECT 1 FROM {table} LIMIT 1"))
                logger.info(f"   ✅ {table}")
            except Exception:
                logger.error(f"   ❌ {table} - MISSING")
                missing_tables.append(table)
        
        # Check required views
        required_views = [
            'phi_access_summary',
            'security_events_summary', 
            'clinical_decision_summary'
        ]
        
        logger.info("\n📊 Checking security views:")
        missing_views = []
        
        for view in required_views:
            try:
                result = connection.execute(text(f"SELECT 1 FROM {view} LIMIT 1"))
                logger.info(f"   ✅ {view}")
            except Exception:
                logger.error(f"   ❌ {view} - MISSING")
                missing_views.append(view)
        
        # Check enhanced audit trail columns
        logger.info("\n🔍 Checking enhanced audit trail:")
        
        try:
            result = connection.execute(text("""
                SELECT column_name, data_type 
                FROM information_schema.columns 
                WHERE table_name = 'clinical_audit_trail'
                AND column_name IN ('event_type', 'audit_level_enum', 'outcome', 'safety_critical', 'error_details')
                ORDER BY column_name
            """))
            
            enhanced_columns = result.fetchall()
            if enhanced_columns:
                logger.info("   ✅ Enhanced audit columns:")
                for col_name, col_type in enhanced_columns:
                    logger.info(f"      - {col_name} ({col_type})")
            else:
                logger.error("   ❌ Enhanced audit columns missing")
        
        except Exception as e:
            logger.error(f"   ❌ Failed to check enhanced columns: {e}")
        
        connection.close()
        engine.dispose()
        
        # Summary
        total_missing = len(missing_tables) + len(missing_views)
        
        if total_missing == 0:
            logger.info("\n✅ Security Framework Schema: COMPLETE")
            logger.info("🔒 All security components properly installed")
            return True
        else:
            logger.error(f"\n❌ Security Framework Schema: INCOMPLETE")
            logger.error(f"Missing {total_missing} components")
            if missing_tables:
                logger.error(f"Missing tables: {', '.join(missing_tables)}")
            if missing_views:
                logger.error(f"Missing views: {', '.join(missing_views)}")
            return False
        
    except Exception as e:
        logger.error(f"❌ Schema verification failed: {e}")
        return False


def main():
    """
    Main function to run security migration and verification.
    """
    logger.info("🏥 Clinical Workflow Engine - Security Framework Migration")
    logger.info("=" * 70)
    
    # Run migration
    migration_success = run_security_migration()
    
    if not migration_success:
        logger.error("❌ Migration failed - aborting")
        return False
    
    # Verify schema
    verification_success = verify_security_schema()
    
    if verification_success:
        logger.info("\n🎉 Security Framework Migration: SUCCESS")
        logger.info("✅ Database ready for production security features")
    else:
        logger.error("\n❌ Security Framework Migration: FAILED")
        logger.error("Database schema incomplete")
    
    return verification_success


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
