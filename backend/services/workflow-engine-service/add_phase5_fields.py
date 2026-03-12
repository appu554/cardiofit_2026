#!/usr/bin/env python3
"""
Add missing Phase 5 fields to existing tables.
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

def add_phase5_fields():
    """Add missing Phase 5 fields to existing tables."""
    try:
        logger.info("=" * 60)
        logger.info("ADDING PHASE 5 FIELDS TO EXISTING TABLES")
        logger.info("=" * 60)
        
        # Create engine
        engine = create_engine(settings.DATABASE_URL)
        
        # SQL statements to add missing fields
        statements = [
            # Add escalation fields to workflow_tasks table
            """
            ALTER TABLE workflow_tasks 
            ADD COLUMN IF NOT EXISTS escalation_level INTEGER DEFAULT 0,
            ADD COLUMN IF NOT EXISTS escalated BOOLEAN DEFAULT FALSE,
            ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMP WITHOUT TIME ZONE,
            ADD COLUMN IF NOT EXISTS escalation_data JSON DEFAULT '{}'
            """,
            
            # Add gateway tracking fields to workflow_instances table
            """
            ALTER TABLE workflow_instances 
            ADD COLUMN IF NOT EXISTS active_gateways JSON DEFAULT '[]',
            ADD COLUMN IF NOT EXISTS completed_gateways JSON DEFAULT '[]'
            """,
            
            # Add error tracking fields to workflow_instances table
            """
            ALTER TABLE workflow_instances 
            ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0,
            ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMP WITHOUT TIME ZONE,
            ADD COLUMN IF NOT EXISTS recovery_attempts INTEGER DEFAULT 0
            """,
            
            # Create indexes for new escalation fields
            "CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalated ON workflow_tasks(escalated)",
            "CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalation_level ON workflow_tasks(escalation_level)",
            
            # Create indexes for new workflow_instances fields
            "CREATE INDEX IF NOT EXISTS idx_workflow_instances_error_count ON workflow_instances(error_count)",
            "CREATE INDEX IF NOT EXISTS idx_workflow_instances_last_error_at ON workflow_instances(last_error_at)"
        ]
        
        logger.info(f"Executing {len(statements)} field addition statements...")
        
        with engine.connect() as conn:
            trans = conn.begin()
            
            try:
                for i, statement in enumerate(statements, 1):
                    logger.info(f"Executing statement {i}/{len(statements)}")
                    logger.debug(f"SQL: {statement.strip()[:100]}...")
                    
                    try:
                        conn.execute(text(statement.strip()))
                        logger.info(f"✅ Statement {i} executed successfully")
                    except Exception as e:
                        if "already exists" in str(e) or "duplicate" in str(e).lower():
                            logger.info(f"✅ Statement {i} already applied: {str(e)[:100]}...")
                        else:
                            logger.warning(f"⚠️ Statement {i} failed: {str(e)[:100]}...")
                
                # Commit transaction
                trans.commit()
                logger.info("✅ Phase 5 fields added successfully")
                return True
                
            except Exception as e:
                trans.rollback()
                logger.error(f"❌ Field addition failed: {e}")
                return False
                
    except Exception as e:
        logger.error(f"❌ Error adding fields: {e}")
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
                                                  started_at, created_at, instance_variables,
                                                  active_gateways, completed_gateways, error_count,
                                                  recovery_attempts)
                    VALUES (1, 1, 'test-patient-123', 'running', NOW(), NOW(), '{}',
                            '[]', '[]', 0, 0)
                    ON CONFLICT (id) DO NOTHING
                """))
                
                # Create test workflow task
                conn.execute(text("""
                    INSERT INTO workflow_tasks (id, workflow_instance_id, name, status, created_at,
                                              escalation_level, escalated, escalation_data,
                                              input_variables, output_variables)
                    VALUES (1, 1, 'Test Task', 'created', NOW(), 0, false, '{}', '{}', '{}')
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
    logger.info("🚀 Starting Phase 5 Field Addition and Test Data Setup")
    
    # Add fields
    fields_success = add_phase5_fields()
    
    if fields_success:
        # Create test data
        test_data_success = create_test_data()
        
        if test_data_success:
            logger.info("🎉 Phase 5 fields and test data setup completed!")
            logger.info("You can now run: python test_phase5_features.py")
        else:
            logger.error("⚠️ Field addition succeeded but test data creation failed")
    else:
        logger.error("❌ Field addition failed")
    
    return fields_success and test_data_success

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
