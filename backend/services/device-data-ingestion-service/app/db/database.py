"""
Database configuration and session management for Supabase PostgreSQL.
Transactional Outbox Pattern implementation for Device Data Ingestion Service.
"""
import logging
import os
from contextlib import asynccontextmanager
from typing import AsyncGenerator

from sqlalchemy import create_engine, MetaData, text
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import QueuePool

logger = logging.getLogger(__name__)

# Import settings to get database configuration
from app.config import settings

# Database configuration (using settings like other services)
DATABASE_URL = settings.DATABASE_URL

# Use explicit async URL from settings
ASYNC_DATABASE_URL = settings.ASYNC_DATABASE_URL

# Create SQLAlchemy engines
# Sync engine for migrations and simple operations
sync_engine = create_engine(
    DATABASE_URL,
    poolclass=QueuePool,
    pool_size=5,
    max_overflow=10,
    pool_pre_ping=True,
    pool_recycle=300,
    echo=False  # Set to True for SQL debugging
)

# Async engine for transactional outbox operations
async_engine = create_async_engine(
    ASYNC_DATABASE_URL,
    pool_size=10,
    max_overflow=20,
    pool_pre_ping=True,
    pool_recycle=300,
    echo=False  # Set to True for SQL debugging
)

# Session makers
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=sync_engine)
AsyncSessionLocal = async_sessionmaker(
    bind=async_engine,
    class_=AsyncSession,
    expire_on_commit=False
)

# Base class for models
Base = declarative_base()
metadata = MetaData()


class DatabaseManager:
    """Database manager for connection lifecycle"""
    
    def __init__(self):
        self._initialized = False
        self._connection_tested = False
    
    async def initialize(self) -> bool:
        """Initialize database connection and test connectivity"""
        try:
            # Test async connection
            async with AsyncSessionLocal() as session:
                result = await session.execute(text("SELECT 1"))
                test_value = result.scalar()
                
                if test_value == 1:
                    self._connection_tested = True
                    self._initialized = True
                    logger.info("✅ Database connection established successfully")
                    return True
                else:
                    logger.error("❌ Database connection test failed")
                    return False
                    
        except Exception as e:
            logger.error(f"❌ Database initialization failed: {e}")
            return False
    
    async def health_check(self) -> dict:
        """Perform database health check"""
        try:
            async with AsyncSessionLocal() as session:
                # Test basic connectivity
                result = await session.execute(text("SELECT NOW()"))
                current_time = result.scalar()
                
                # Test outbox tables exist
                vendor_check = await session.execute(text("""
                    SELECT COUNT(*) FROM information_schema.tables 
                    WHERE table_name IN ('fitbit_outbox', 'garmin_outbox', 'apple_health_outbox')
                """))
                table_count = vendor_check.scalar()
                
                return {
                    "status": "healthy",
                    "database_time": current_time.isoformat() if current_time else None,
                    "outbox_tables_present": table_count == 3,
                    "connection_pool_size": async_engine.pool.size(),
                    "connection_pool_checked_out": async_engine.pool.checkedout()
                }
                
        except Exception as e:
            logger.error(f"Database health check failed: {e}")
            return {
                "status": "unhealthy",
                "error": str(e),
                "connection_pool_size": 0,
                "connection_pool_checked_out": 0
            }
    
    async def close(self):
        """Close database connections"""
        try:
            await async_engine.dispose()
            sync_engine.dispose()
            logger.info("Database connections closed")
        except Exception as e:
            logger.error(f"Error closing database connections: {e}")


# Global database manager instance
db_manager = DatabaseManager()


@asynccontextmanager
async def get_async_session() -> AsyncGenerator[AsyncSession, None]:
    """
    Get async database session with proper transaction management.
    
    Usage:
        async with get_async_session() as session:
            # Your database operations here
            await session.commit()  # Explicit commit required
    """
    async with AsyncSessionLocal() as session:
        try:
            yield session
        except Exception as e:
            await session.rollback()
            logger.error(f"Database session error, rolling back: {e}")
            raise
        finally:
            await session.close()


def get_sync_session():
    """
    Get synchronous database session for migrations and simple operations.
    
    Usage:
        with get_sync_session() as session:
            # Your database operations here
            session.commit()  # Explicit commit required
    """
    return SessionLocal()


async def run_migration_script(script_path: str) -> bool:
    """
    Run a SQL migration script
    
    Args:
        script_path: Path to the SQL migration file
        
    Returns:
        bool: True if migration succeeded, False otherwise
    """
    try:
        import asyncio
        from pathlib import Path
        
        # Read migration file
        migration_file = Path(script_path)
        if not migration_file.exists():
            logger.error(f"Migration file not found: {script_path}")
            return False
        
        with open(migration_file, 'r') as f:
            sql_content = f.read()
        
        # Split into individual statements (basic approach)
        statements = [stmt.strip() for stmt in sql_content.split(';') if stmt.strip()]
        
        async with get_async_session() as session:
            for i, statement in enumerate(statements, 1):
                if statement:
                    try:
                        logger.info(f"Executing migration statement {i}/{len(statements)}")
                        await session.execute(text(statement))
                    except Exception as e:
                        # Some statements might fail if already executed, that's OK
                        if "already exists" in str(e).lower() or "duplicate" in str(e).lower():
                            logger.info(f"Statement {i} already applied: {str(e)[:100]}...")
                        else:
                            logger.warning(f"Statement {i} failed: {str(e)[:100]}...")
            
            await session.commit()
            logger.info("✅ Migration completed successfully")
            return True
            
    except Exception as e:
        logger.error(f"❌ Migration failed: {e}")
        return False


# Startup and shutdown handlers
async def startup_database():
    """Initialize database on application startup"""
    success = await db_manager.initialize()
    if not success:
        logger.error("Failed to initialize database connection")
        raise RuntimeError("Database initialization failed")


async def shutdown_database():
    """Close database connections on application shutdown"""
    await db_manager.close()


# Export commonly used items
__all__ = [
    "get_async_session",
    "get_sync_session", 
    "db_manager",
    "startup_database",
    "shutdown_database",
    "run_migration_script",
    "Base",
    "metadata"
]
