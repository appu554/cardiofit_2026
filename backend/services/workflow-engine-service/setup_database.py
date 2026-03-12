#!/usr/bin/env python3
"""
Database setup script for Workflow Engine Service.
This script creates the necessary tables in Supabase PostgreSQL.
"""
import asyncio
import os
import sys
import logging
from pathlib import Path

# Add the current directory to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

# Add the backend services directory to Python path for shared imports
backend_dir = os.path.dirname(os.path.dirname(current_dir))
services_dir = os.path.join(backend_dir, "services")
sys.path.insert(0, services_dir)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def setup_database():
    """Set up the database tables."""
    try:
        from app.core.config import settings
        from app.db.database import engine, Base
        
        logger.info("Setting up Workflow Engine Service database...")
        logger.info(f"Database URL: {settings.DATABASE_URL[:50]}...")
        
        # Import all models to ensure they are registered
        from app.models import workflow_models, task_models
        
        # Create all tables
        logger.info("Creating database tables...")
        Base.metadata.create_all(bind=engine)
        logger.info("Database tables created successfully")
        
        return True
        
    except Exception as e:
        logger.error(f"Error setting up database: {e}")
        return False


async def run_sql_migration():
    """Run the SQL migration script."""
    try:
        import psycopg2
        from urllib.parse import urlparse
        from app.core.config import settings
        
        # Parse the database URL
        parsed_url = urlparse(settings.DATABASE_URL)
        
        # Connect to the database
        conn = psycopg2.connect(
            host=parsed_url.hostname,
            port=parsed_url.port,
            database=parsed_url.path[1:],  # Remove leading slash
            user=parsed_url.username,
            password=parsed_url.password
        )
        
        # Read the migration SQL file
        migration_file = Path(current_dir) / "migrations" / "001_create_workflow_tables.sql"
        
        if not migration_file.exists():
            logger.warning(f"Migration file not found: {migration_file}")
            return False
        
        with open(migration_file, 'r') as f:
            sql_content = f.read()
        
        # Execute the migration
        logger.info("Running SQL migration...")
        cursor = conn.cursor()
        cursor.execute(sql_content)
        conn.commit()
        cursor.close()
        conn.close()
        
        logger.info("SQL migration completed successfully")
        return True
        
    except ImportError:
        logger.warning("psycopg2 not available, skipping SQL migration")
        return False
    except Exception as e:
        logger.error(f"Error running SQL migration: {e}")
        return False


async def test_database_connection():
    """Test the database connection."""
    try:
        from app.core.config import settings
        from sqlalchemy import create_engine, text
        
        logger.info("Testing database connection...")
        
        # Create a test engine
        test_engine = create_engine(settings.DATABASE_URL)
        
        # Test the connection
        with test_engine.connect() as conn:
            result = conn.execute(text("SELECT 1"))
            row = result.fetchone()
            if row and row[0] == 1:
                logger.info("Database connection test successful")
                return True
            else:
                logger.error("Database connection test failed")
                return False
                
    except Exception as e:
        logger.error(f"Database connection test failed: {e}")
        return False


async def verify_tables():
    """Verify that all tables were created."""
    try:
        from app.core.config import settings
        from sqlalchemy import create_engine, text
        
        logger.info("Verifying database tables...")
        
        # Create a test engine
        test_engine = create_engine(settings.DATABASE_URL)
        
        # List of expected tables
        expected_tables = [
            'workflow_definitions',
            'workflow_instances', 
            'workflow_tasks',
            'workflow_events',
            'workflow_timers',
            'task_assignments',
            'task_comments',
            'task_attachments',
            'task_escalations',
            'workflow_events_log'
        ]
        
        # Check if tables exist
        with test_engine.connect() as conn:
            for table_name in expected_tables:
                result = conn.execute(text(f"""
                    SELECT EXISTS (
                        SELECT FROM information_schema.tables 
                        WHERE table_schema = 'public' 
                        AND table_name = '{table_name}'
                    );
                """))
                exists = result.fetchone()[0]
                
                if exists:
                    logger.info(f"✅ Table '{table_name}' exists")
                else:
                    logger.warning(f"❌ Table '{table_name}' does not exist")
        
        logger.info("Table verification completed")
        return True
        
    except Exception as e:
        logger.error(f"Error verifying tables: {e}")
        return False


async def main():
    """Main setup function."""
    logger.info("=" * 60)
    logger.info("WORKFLOW ENGINE SERVICE - DATABASE SETUP")
    logger.info("=" * 60)
    
    # Test database connection
    connection_ok = await test_database_connection()
    if not connection_ok:
        logger.error("Database connection failed. Please check your configuration.")
        return False
    
    # Run SQL migration (if available)
    migration_ok = await run_sql_migration()
    if migration_ok:
        logger.info("SQL migration completed")
    else:
        logger.info("Falling back to SQLAlchemy table creation")
        
        # Set up database using SQLAlchemy
        setup_ok = await setup_database()
        if not setup_ok:
            logger.error("Database setup failed")
            return False
    
    # Verify tables
    verify_ok = await verify_tables()
    if verify_ok:
        logger.info("Database setup completed successfully")
    else:
        logger.warning("Some issues found during table verification")
    
    logger.info("=" * 60)
    logger.info("Database setup process completed")
    logger.info("You can now run the service: python run_service.py")
    logger.info("=" * 60)
    
    return True


if __name__ == "__main__":
    asyncio.run(main())
