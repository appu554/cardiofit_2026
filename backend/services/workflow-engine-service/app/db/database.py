"""
Database configuration and session management for Supabase PostgreSQL.
"""
import logging
from sqlalchemy import create_engine, MetaData
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import QueuePool
from app.core.config import settings

logger = logging.getLogger(__name__)

# Create SQLAlchemy engine optimized for Supabase PostgreSQL
engine = create_engine(
    settings.DATABASE_URL,
    poolclass=QueuePool,
    pool_size=5,
    max_overflow=10,
    pool_pre_ping=True,
    pool_recycle=300,
    echo=settings.DEBUG
)

# Create SessionLocal class
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

# Create Base class for models
Base = declarative_base()

# Metadata for migrations
metadata = MetaData()


def get_db():
    """
    Dependency to get database session.
    """
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


async def init_db():
    """
    Initialize database tables.
    Since we're using Supabase and tables are already created,
    this just validates the models are properly imported.
    """
    try:
        # Import all models to ensure they are registered
        from app.models import workflow_models, task_models

        # Skip actual table creation since we're using Supabase
        # Tables are created manually via SQL editor
        logger.info("Database models loaded successfully (using Supabase)")
        return True
    except Exception as e:
        logger.error(f"Error loading database models: {e}")
        return False


async def close_db():
    """
    Close database connections.
    """
    try:
        engine.dispose()
        logger.info("Database connections closed")
    except Exception as e:
        logger.error(f"Error closing database: {e}")
