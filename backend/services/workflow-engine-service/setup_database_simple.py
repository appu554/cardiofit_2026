#!/usr/bin/env python3
"""
Simple database setup script for Workflow Engine Service.
This script creates the necessary tables in Supabase PostgreSQL.
"""
import os
import sys
import logging
from pathlib import Path

# Load environment variables from .env file
try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    print("⚠️  python-dotenv not available, .env file won't be loaded")

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


def test_imports():
    """Test if required modules are available."""
    try:
        import sqlalchemy
        logger.info("✅ SQLAlchemy available")
        return True
    except ImportError as e:
        logger.error(f"❌ SQLAlchemy not available: {e}")
        logger.error("Please install dependencies: pip install sqlalchemy psycopg2-binary python-dotenv")
        return False


def test_database_connection():
    """Test the database connection."""
    try:
        from app.core.config import settings
        from sqlalchemy import create_engine, text
        
        logger.info("Testing database connection...")
        logger.info(f"Database URL: {settings.DATABASE_URL[:50]}...")
        
        # Create a test engine
        test_engine = create_engine(settings.DATABASE_URL)
        
        # Test the connection
        with test_engine.connect() as conn:
            result = conn.execute(text("SELECT 1"))
            row = result.fetchone()
            if row and row[0] == 1:
                logger.info("✅ Database connection test successful")
                return True
            else:
                logger.error("❌ Database connection test failed")
                return False
                
    except Exception as e:
        logger.error(f"❌ Database connection test failed: {e}")
        return False


def run_sql_migration():
    """Run the SQL migration script directly."""
    try:
        import psycopg2
        from urllib.parse import urlparse
        from app.core.config import settings
        
        logger.info("Running SQL migration...")
        
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
            logger.error(f"❌ Migration file not found: {migration_file}")
            return False
        
        with open(migration_file, 'r') as f:
            sql_content = f.read()
        
        # Execute the migration
        cursor = conn.cursor()
        cursor.execute(sql_content)
        conn.commit()
        cursor.close()
        conn.close()
        
        logger.info("✅ SQL migration completed successfully")
        return True
        
    except ImportError:
        logger.error("❌ psycopg2 not available, cannot run SQL migration")
        logger.error("Please install: pip install psycopg2-binary")
        return False
    except Exception as e:
        logger.error(f"❌ Error running SQL migration: {e}")
        return False


def create_tables_with_sqlalchemy():
    """Create tables using SQLAlchemy."""
    try:
        from app.core.config import settings
        from app.db.database import engine, Base
        
        logger.info("Creating tables with SQLAlchemy...")
        
        # Import all models to ensure they are registered
        from app.models import workflow_models, task_models
        
        # Create all tables
        Base.metadata.create_all(bind=engine)
        logger.info("✅ Database tables created successfully with SQLAlchemy")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Error creating tables with SQLAlchemy: {e}")
        return False


def verify_tables():
    """Verify that tables were created."""
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
            'workflow_timers'
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
        
        logger.info("✅ Table verification completed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Error verifying tables: {e}")
        return False


def main():
    """Main setup function."""
    logger.info("=" * 60)
    logger.info("WORKFLOW ENGINE SERVICE - SIMPLE DATABASE SETUP")
    logger.info("=" * 60)
    
    # Test imports
    if not test_imports():
        logger.error("Required dependencies not available. Exiting.")
        return False
    
    # Test database connection
    if not test_database_connection():
        logger.error("Database connection failed. Please check your configuration.")
        return False
    
    # Try SQL migration first
    migration_ok = run_sql_migration()
    
    if not migration_ok:
        logger.info("SQL migration failed, trying SQLAlchemy...")
        sqlalchemy_ok = create_tables_with_sqlalchemy()
        if not sqlalchemy_ok:
            logger.error("Both SQL migration and SQLAlchemy failed")
            return False
    
    # Verify tables
    verify_tables()
    
    logger.info("=" * 60)
    logger.info("✅ Database setup completed!")
    logger.info("You can now run: python test_service.py")
    logger.info("Then run: python run_service.py")
    logger.info("=" * 60)
    
    return True


if __name__ == "__main__":
    main()
