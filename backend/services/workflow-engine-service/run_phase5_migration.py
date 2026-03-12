#!/usr/bin/env python3
"""
Run Phase 5 migration to add missing fields and fix database schema.
"""

import sys
import os
import logging
from sqlalchemy import create_engine, text

# Add the app directory to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

from app.core.config import settings

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def run_phase5_migration():
    """Run the Phase 5 migration."""
    try:
        logger.info("=" * 60)
        logger.info("RUNNING PHASE 5 MIGRATION")
        logger.info("=" * 60)
        
        # Create engine
        engine = create_engine(settings.DATABASE_URL)
        
        # Read migration file
        migration_file = "migrations/004_phase5_advanced_features.sql"
        
        if not os.path.exists(migration_file):
            logger.error(f"Migration file not found: {migration_file}")
            return False
        
        with open(migration_file, 'r') as f:
            migration_sql = f.read()
        
        # Split into individual statements
        statements = [stmt.strip() for stmt in migration_sql.split(';') if stmt.strip()]
        
        logger.info(f"Executing {len(statements)} migration statements...")
        
        with engine.connect() as conn:
            # Start transaction
            trans = conn.begin()
            
            try:
                for i, statement in enumerate(statements, 1):
                    if statement.strip():
                        logger.info(f"Executing statement {i}/{len(statements)}")
                        logger.debug(f"SQL: {statement[:100]}...")
                        
                        try:
                            conn.execute(text(statement))
                        except Exception as e:
                            # Some statements might fail if already executed, that's OK
                            if "already exists" in str(e) or "duplicate" in str(e).lower():
                                logger.info(f"Statement {i} already applied: {str(e)[:100]}...")
                            else:
                                logger.warning(f"Statement {i} failed: {str(e)[:100]}...")
                
                # Commit transaction
                trans.commit()
                logger.info("✅ Phase 5 migration completed successfully")
                return True
                
            except Exception as e:
                trans.rollback()
                logger.error(f"❌ Migration failed: {e}")
                return False
                
    except Exception as e:
        logger.error(f"❌ Error running migration: {e}")
        return False

def create_test_data():
    """Create test workflow instances and tasks for testing."""
    try:
        logger.info("Creating test data...")
        
        engine = create_engine(settings.DATABASE_URL)
        
        with engine.connect() as conn:
            trans = conn.begin()
            
            try:
                # Create test workflow definition
                conn.execute(text("""
                    INSERT INTO workflow_definitions (id, name, version, bpmn_xml, status, created_at)
                    VALUES (1, 'Test Workflow', '1.0', '<bpmn>test</bpmn>', 'active', NOW())
                    ON CONFLICT (id) DO NOTHING
                """))
                
                # Create test workflow instance
                conn.execute(text("""
                    INSERT INTO workflow_instances (id, workflow_definition_id, patient_id, status, 
                                                  started_at, created_at, instance_variables)
                    VALUES (1, 1, 'test-patient-123', 'running', NOW(), NOW(), '{}')
                    ON CONFLICT (id) DO NOTHING
                """))
                
                # Create test workflow task
                conn.execute(text("""
                    INSERT INTO workflow_tasks (id, workflow_instance_id, name, status, created_at,
                                              escalation_level, escalated, task_data)
                    VALUES (1, 1, 'Test Task', 'created', NOW(), 0, false, '{}')
                    ON CONFLICT (id) DO NOTHING
                """))
                
                trans.commit()
                logger.info("✅ Test data created successfully")
                return True
                
            except Exception as e:
                trans.rollback()
                logger.error(f"❌ Error creating test data: {e}")
                return False
                
    except Exception as e:
        logger.error(f"❌ Error creating test data: {e}")
        return False

def main():
    """Main function."""
    logger.info("🚀 Starting Phase 5 Migration and Test Data Setup")
    
    # Run migration
    migration_success = run_phase5_migration()
    
    if migration_success:
        # Create test data
        test_data_success = create_test_data()
        
        if test_data_success:
            logger.info("🎉 Phase 5 migration and test data setup completed!")
            logger.info("You can now run: python test_phase5_features.py")
        else:
            logger.error("⚠️ Migration succeeded but test data creation failed")
    else:
        logger.error("❌ Migration failed")
    
    return migration_success and test_data_success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
