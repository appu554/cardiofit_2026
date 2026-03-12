"""
Database Connection Manager for Global Outbox Service

Handles PostgreSQL connections using asyncpg with connection pooling,
health checks, and migration execution capabilities.
"""

import asyncio
import asyncpg
import logging
from typing import Optional, Dict, Any, List
from contextlib import asynccontextmanager
from pathlib import Path

from app.core.config import settings

logger = logging.getLogger(__name__)

class DatabaseManager:
    """
    Database connection manager with connection pooling and health monitoring
    
    Features:
    - Connection pooling with configurable size
    - Health check capabilities
    - Migration execution
    - Transaction management
    - Query execution with proper error handling
    """
    
    def __init__(self):
        self.pool: Optional[asyncpg.Pool] = None
        self._is_connected = False
        self._health_status = False
    
    async def connect(self) -> bool:
        """
        Create database connection pool
        
        Returns:
            bool: True if connection successful, False otherwise
        """
        try:
            logger.info("Creating database connection pool...")
            
            # Create connection pool
            self.pool = await asyncpg.create_pool(
                settings.get_database_url(),
                min_size=5,
                max_size=settings.DATABASE_POOL_SIZE,
                max_queries=50000,
                max_inactive_connection_lifetime=300,
                command_timeout=settings.DATABASE_POOL_TIMEOUT,
                server_settings={
                    'application_name': settings.PROJECT_NAME,
                    'timezone': 'UTC'
                }
            )
            
            # Test the connection
            async with self.pool.acquire() as conn:
                await conn.fetchval("SELECT 1")
            
            self._is_connected = True
            self._health_status = True
            
            logger.info(f"✅ Database connection pool created successfully")
            logger.info(f"   Pool size: {settings.DATABASE_POOL_SIZE}")
            logger.info(f"   Database: {settings.get_database_url().split('@')[1].split('/')[0]}")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to create database connection pool: {e}")
            self._is_connected = False
            self._health_status = False
            return False
    
    async def disconnect(self):
        """Close database connection pool"""
        if self.pool:
            try:
                await self.pool.close()
                logger.info("Database connection pool closed")
            except Exception as e:
                logger.error(f"Error closing database pool: {e}")
            finally:
                self.pool = None
                self._is_connected = False
                self._health_status = False
    
    async def health_check(self) -> Dict[str, Any]:
        """
        Comprehensive database health check
        
        Returns:
            Dict with health status and metrics
        """
        if not self.pool:
            return {
                "status": "unhealthy",
                "error": "No database connection pool",
                "connected": False
            }
        
        try:
            start_time = asyncio.get_event_loop().time()
            
            async with self.pool.acquire() as conn:
                # Test basic connectivity
                result = await conn.fetchval("SELECT 1")
                
                # Get connection info
                server_version = await conn.fetchval("SELECT version()")
                current_database = await conn.fetchval("SELECT current_database()")
                current_user = await conn.fetchval("SELECT current_user")
                
                # Get pool statistics
                pool_size = self.pool.get_size()
                pool_min_size = self.pool.get_min_size()
                pool_max_size = self.pool.get_max_size()
                
            end_time = asyncio.get_event_loop().time()
            response_time_ms = (end_time - start_time) * 1000
            
            self._health_status = True
            
            return {
                "status": "healthy",
                "connected": True,
                "response_time_ms": round(response_time_ms, 2),
                "database": current_database,
                "user": current_user,
                "server_version": server_version.split(' ')[0] if server_version else "unknown",
                "pool": {
                    "size": pool_size,
                    "min_size": pool_min_size,
                    "max_size": pool_max_size
                }
            }
            
        except Exception as e:
            logger.error(f"Database health check failed: {e}")
            self._health_status = False
            
            return {
                "status": "unhealthy",
                "connected": False,
                "error": str(e)
            }
    
    async def execute_migration(self, migration_file: Optional[str] = None) -> bool:
        """
        Execute database migration
        
        Args:
            migration_file: Optional specific migration file path
            
        Returns:
            bool: True if migration successful
        """
        try:
            # Default migration file
            if not migration_file:
                migration_file = Path(__file__).parent.parent.parent / "migrations" / "001_create_outbox_tables.sql"
            else:
                migration_file = Path(migration_file)
            
            if not migration_file.exists():
                logger.error(f"Migration file not found: {migration_file}")
                return False
            
            logger.info(f"Executing database migration: {migration_file.name}")
            
            # Read migration SQL
            with open(migration_file, "r", encoding="utf-8") as f:
                migration_sql = f.read()
            
            # Execute migration
            async with self.pool.acquire() as conn:
                async with conn.transaction():
                    await conn.execute(migration_sql)
            
            logger.info("✅ Database migration completed successfully")
            return True
            
        except Exception as e:
            logger.error(f"❌ Database migration failed: {e}")
            return False
    
    @asynccontextmanager
    async def get_connection(self):
        """
        Get a database connection from the pool
        
        Usage:
            async with db_manager.get_connection() as conn:
                result = await conn.fetchval("SELECT 1")
        """
        if not self.pool:
            raise RuntimeError("Database pool not initialized")
        
        async with self.pool.acquire() as conn:
            yield conn
    
    @asynccontextmanager
    async def get_transaction(self):
        """
        Get a database transaction
        
        Usage:
            async with db_manager.get_transaction() as conn:
                await conn.execute("INSERT ...")
                # Transaction automatically committed or rolled back
        """
        if not self.pool:
            raise RuntimeError("Database pool not initialized")
        
        async with self.pool.acquire() as conn:
            async with conn.transaction():
                yield conn
    
    async def execute_query(self, query: str, *args) -> Any:
        """Execute a query and return the result"""
        async with self.get_connection() as conn:
            return await conn.fetchval(query, *args)
    
    async def execute_many(self, query: str, args_list: List[tuple]) -> None:
        """Execute a query multiple times with different parameters"""
        async with self.get_connection() as conn:
            await conn.executemany(query, args_list)
    
    async def fetch_all(self, query: str, *args) -> List[Dict[str, Any]]:
        """Fetch all rows from a query"""
        async with self.get_connection() as conn:
            rows = await conn.fetch(query, *args)
            return [dict(row) for row in rows]
    
    async def fetch_one(self, query: str, *args) -> Optional[Dict[str, Any]]:
        """Fetch one row from a query"""
        async with self.get_connection() as conn:
            row = await conn.fetchrow(query, *args)
            return dict(row) if row else None
    
    @property
    def is_connected(self) -> bool:
        """Check if database is connected"""
        return self._is_connected and self.pool is not None
    
    @property
    def is_healthy(self) -> bool:
        """Check if database is healthy"""
        return self._health_status
    
    async def get_outbox_stats(self) -> Dict[str, Any]:
        """Get outbox-specific database statistics"""
        try:
            async with self.get_connection() as conn:
                # Get queue depths by service
                queue_depths = await conn.fetch("""
                    SELECT origin_service, COUNT(*) as queue_depth
                    FROM global_event_outbox 
                    WHERE status = 'pending'
                    GROUP BY origin_service
                """)
                
                # Get total events processed in last 24 hours
                total_processed = await conn.fetchval("""
                    SELECT COUNT(*) 
                    FROM global_event_outbox 
                    WHERE processed_at > NOW() - INTERVAL '24 hours'
                """)
                
                # Get dead letter count
                dead_letter_count = await conn.fetchval("""
                    SELECT COUNT(*) FROM global_dead_letter_queue
                """)
                
                return {
                    "queue_depths": {row["origin_service"]: row["queue_depth"] for row in queue_depths},
                    "total_processed_24h": total_processed or 0,
                    "dead_letter_count": dead_letter_count or 0,
                    "timestamp": asyncio.get_event_loop().time()
                }
                
        except Exception as e:
            logger.error(f"Failed to get outbox stats: {e}")
            return {
                "queue_depths": {},
                "total_processed_24h": 0,
                "dead_letter_count": 0,
                "error": str(e)
            }

# Global database manager instance
db_manager = DatabaseManager()
