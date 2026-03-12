"""
Check existing database schema and tables.
"""
import os
import sys
from sqlalchemy import create_engine, text, inspect
import logging

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def check_database_schema():
    """
    Check what tables and columns already exist in the database.
    """
    try:
        # Connect to database
        engine = create_engine(settings.DATABASE_URL, echo=False)
        connection = engine.connect()
        inspector = inspect(engine)
        
        logger.info("🔍 Checking existing database schema...")
        logger.info("=" * 60)
        
        # Get all table names
        table_names = inspector.get_table_names()
        logger.info(f"📊 Found {len(table_names)} tables in database")
        
        if not table_names:
            logger.info("✅ Database is empty - ready for fresh setup")
            return True
        
        # Check each table
        workflow_related_tables = []
        for table_name in table_names:
            if any(keyword in table_name.lower() for keyword in ['workflow', 'task', 'clinical']):
                workflow_related_tables.append(table_name)
        
        if workflow_related_tables:
            logger.info(f"🔧 Found {len(workflow_related_tables)} workflow-related tables:")
            for table in workflow_related_tables:
                columns = inspector.get_columns(table)
                column_names = [col['name'] for col in columns]
                logger.info(f"   📋 {table}: {len(columns)} columns")
                logger.info(f"      Columns: {', '.join(column_names[:10])}")  # Show first 10 columns
                if len(column_names) > 10:
                    logger.info(f"      ... and {len(column_names) - 10} more")
        
        # Check specific tables we need
        required_tables = [
            'workflow_definitions',
            'workflow_instances', 
            'workflow_tasks',
            'workflow_events',
            'clinical_activity_executions',
            'clinical_audit_trail',
            'clinical_errors',
            'clinical_workflow_metrics',
            'emergency_access_records',
            'phi_encryption_keys',
            'clinical_timers'
        ]
        
        logger.info("\n🎯 Checking required tables:")
        existing_tables = []
        missing_tables = []
        
        for table in required_tables:
            if table in table_names:
                existing_tables.append(table)
                logger.info(f"   ✅ {table} - EXISTS")
            else:
                missing_tables.append(table)
                logger.info(f"   ❌ {table} - MISSING")
        
        logger.info(f"\n📈 Summary:")
        logger.info(f"   Existing tables: {len(existing_tables)}")
        logger.info(f"   Missing tables: {len(missing_tables)}")
        
        if missing_tables:
            logger.info(f"\n🔧 Tables to create: {', '.join(missing_tables)}")
        
        # Check for any conflicting columns in existing tables
        if 'workflow_instances' in table_names:
            logger.info(f"\n🔍 Checking workflow_instances table structure:")
            columns = inspector.get_columns('workflow_instances')
            column_names = [col['name'] for col in columns]
            
            required_columns = ['id', 'external_id', 'definition_id', 'patient_id', 'status']
            for col in required_columns:
                if col in column_names:
                    logger.info(f"   ✅ Column '{col}' exists")
                else:
                    logger.info(f"   ❌ Column '{col}' missing")
        
        connection.close()
        engine.dispose()
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Error checking database schema: {e}")
        return False


def main():
    """
    Main function to check database schema.
    """
    logger.info("🏥 Database Schema Check")
    logger.info("=" * 40)
    
    success = check_database_schema()
    
    if success:
        logger.info("\n✅ Database schema check completed!")
    else:
        logger.error("\n❌ Database schema check failed!")
    
    return success


if __name__ == "__main__":
    main()
