#!/usr/bin/env python3
"""
Run database migration to add missing fields
"""

import asyncio
import sys
import os
from pathlib import Path

# Add the app directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '.'))

async def run_migration():
    """Run the database migration"""
    try:
        print("🔧 Running database migration to fix missing fields...")
        
        # Import database components
        from app.db.database import get_db, engine
        from sqlalchemy import text
        
        # Read migration SQL
        migration_file = Path(__file__).parent / "migrations" / "002_add_missing_fields.sql"
        
        if not migration_file.exists():
            print(f"❌ Migration file not found: {migration_file}")
            return False
        
        migration_sql = migration_file.read_text()
        print(f"✅ Migration SQL loaded from {migration_file}")
        
        # Execute migration
        with engine.connect() as connection:
            # Split SQL into individual statements
            statements = [stmt.strip() for stmt in migration_sql.split(';') if stmt.strip()]
            
            for i, statement in enumerate(statements, 1):
                try:
                    print(f"   Executing statement {i}/{len(statements)}...")
                    connection.execute(text(statement))
                    connection.commit()
                    print(f"   ✅ Statement {i} executed successfully")
                except Exception as e:
                    print(f"   ⚠️  Statement {i} failed (might already exist): {e}")
                    continue
        
        print("✅ Database migration completed successfully!")
        
        # Test the database connection
        print("\n🧪 Testing database connection...")
        db = next(get_db())
        
        # Test querying workflow instances
        from app.models.workflow_models import WorkflowInstance, WorkflowTask
        
        instances = db.query(WorkflowInstance).limit(1).all()
        print(f"✅ Successfully queried workflow_instances table ({len(instances)} records)")
        
        tasks = db.query(WorkflowTask).limit(1).all()
        print(f"✅ Successfully queried workflow_tasks table ({len(tasks)} records)")
        
        # Test the new fields
        try:
            # Test updated_at field
            instance_with_updated_at = db.query(WorkflowInstance.updated_at).limit(1).first()
            print("✅ updated_at field is accessible on WorkflowInstance")
            
            # Test escalated field  
            task_with_escalated = db.query(WorkflowTask.escalated).limit(1).first()
            print("✅ escalated field is accessible on WorkflowTask")
            
        except Exception as e:
            print(f"❌ Error testing new fields: {e}")
            return False
        
        print("\n🎉 Migration completed successfully! The monitoring errors should be resolved.")
        return True
        
    except Exception as e:
        print(f"❌ Migration failed: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    """Main function"""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - DATABASE MIGRATION")
    print("=" * 60)
    
    # Run the migration
    result = asyncio.run(run_migration())
    
    if result:
        print("\n✅ Migration completed successfully!")
        print("\nNext steps:")
        print("1. Restart the WorkflowEngine service: python run_service.py")
        print("2. The monitoring errors should be resolved")
        print("3. Test the federation integration")
        return 0
    else:
        print("\n❌ Migration failed!")
        return 1

if __name__ == "__main__":
    exit(main())
