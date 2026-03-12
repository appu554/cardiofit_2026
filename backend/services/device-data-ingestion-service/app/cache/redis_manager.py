"""
Redis Cache Manager for Device Data Ingestion Service

Provides high-performance caching for device configurations, validation rules,
patient context, and authentication results with TTL management and cache invalidation.
"""

import asyncio
import json
import logging
import pickle
import time
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional, Union
from dataclasses import dataclass

# Try to import redis, fallback to None if not available
try:
    import redis.asyncio as redis
    from redis.asyncio import ConnectionPool
    REDIS_AVAILABLE = True
except ImportError:
    redis = None
    ConnectionPool = None
    REDIS_AVAILABLE = False

logger = logging.getLogger(__name__)


@dataclass
class CacheConfig:
    """Configuration for Redis cache"""
    redis_url: str = "redis://localhost:6379"
    max_connections: int = 20
    socket_timeout: float = 5.0
    socket_connect_timeout: float = 5.0
    retry_on_timeout: bool = True
    health_check_interval: float = 30.0
    
    # TTL configurations (in seconds)
    device_config_ttl: int = 3600  # 1 hour
    validation_rules_ttl: int = 1800  # 30 minutes
    patient_context_ttl: int = 900  # 15 minutes
    auth_result_ttl: int = 300  # 5 minutes
    circuit_breaker_ttl: int = 60  # 1 minute
    
    # Cache key prefixes
    device_config_prefix: str = "device_config"
    validation_rules_prefix: str = "validation_rules"
    patient_context_prefix: str = "patient_context"
    auth_result_prefix: str = "auth_result"
    circuit_breaker_prefix: str = "circuit_breaker"
    performance_metrics_prefix: str = "perf_metrics"


class RedisCacheManager:
    """High-performance Redis cache manager with connection pooling and health monitoring"""

    def __init__(self, config: CacheConfig):
        self.config = config
        self.pool: Optional[Any] = None  # ConnectionPool when available
        self.redis_client: Optional[Any] = None  # redis.Redis when available
        self.is_healthy = False
        self.health_check_task: Optional[asyncio.Task] = None
        self._lock = asyncio.Lock()
        self.fallback_mode = not REDIS_AVAILABLE
        
        # Performance metrics
        self.metrics = {
            "cache_hits": 0,
            "cache_misses": 0,
            "cache_sets": 0,
            "cache_deletes": 0,
            "connection_errors": 0,
            "operation_times": []
        }
    
    async def initialize(self):
        """Initialize Redis connection pool and client with fallback support"""
        if not REDIS_AVAILABLE:
            logger.warning("Redis module not installed, running in fallback mode")
            logger.info("Install redis with: pip install redis")
            logger.info("Cache operations will be skipped - performance optimizations disabled")
            self.is_healthy = False
            self.fallback_mode = True
            return

        try:
            # Create connection pool
            self.pool = ConnectionPool.from_url(
                self.config.redis_url,
                max_connections=self.config.max_connections,
                socket_timeout=self.config.socket_timeout,
                socket_connect_timeout=self.config.socket_connect_timeout,
                retry_on_timeout=self.config.retry_on_timeout,
                decode_responses=False  # We'll handle encoding ourselves
            )

            # Create Redis client
            self.redis_client = redis.Redis(connection_pool=self.pool)

            # Test connection
            await self.redis_client.ping()
            self.is_healthy = True
            self.fallback_mode = False

            # Start health check task
            self.health_check_task = asyncio.create_task(self._health_check_loop())

            logger.info("Redis cache manager initialized successfully")

        except Exception as e:
            logger.warning(f"Redis not available, running in fallback mode: {e}")
            logger.info("Cache operations will be skipped - performance optimizations disabled")
            self.is_healthy = False
            self.fallback_mode = True
            # Don't raise exception - allow service to continue without caching
    
    async def _health_check_loop(self):
        """Continuous health check for Redis connection"""
        while True:
            try:
                await asyncio.sleep(self.config.health_check_interval)
                
                if self.redis_client:
                    start_time = time.time()
                    await self.redis_client.ping()
                    ping_time = time.time() - start_time
                    
                    self.is_healthy = True
                    logger.debug(f"Redis health check passed (ping: {ping_time:.3f}s)")
                else:
                    self.is_healthy = False
                    
            except Exception as e:
                self.is_healthy = False
                self.metrics["connection_errors"] += 1
                logger.warning(f"Redis health check failed: {e}")
    
    async def get(self, key: str, default: Any = None) -> Any:
        """Get value from cache with performance tracking"""
        if self.fallback_mode or not self.is_healthy or not self.redis_client:
            logger.debug(f"Cache unavailable (fallback mode: {self.fallback_mode}), returning default for key: {key}")
            self.metrics["cache_misses"] += 1
            return default
        
        start_time = time.time()
        try:
            async with self._lock:
                value = await self.redis_client.get(key)
                
            if value is not None:
                self.metrics["cache_hits"] += 1
                # Try JSON first, then pickle for complex objects
                try:
                    result = json.loads(value.decode('utf-8'))
                except (json.JSONDecodeError, UnicodeDecodeError):
                    result = pickle.loads(value)
                
                operation_time = time.time() - start_time
                self.metrics["operation_times"].append(operation_time)
                
                logger.debug(f"Cache HIT for key: {key} (time: {operation_time:.3f}s)")
                return result
            else:
                self.metrics["cache_misses"] += 1
                logger.debug(f"Cache MISS for key: {key}")
                return default
                
        except Exception as e:
            self.metrics["connection_errors"] += 1
            logger.warning(f"Cache get error for key {key}: {e}")
            return default
    
    async def set(self, key: str, value: Any, ttl: Optional[int] = None) -> bool:
        """Set value in cache with TTL"""
        if self.fallback_mode or not self.is_healthy or not self.redis_client:
            logger.debug(f"Cache unavailable (fallback mode: {self.fallback_mode}), skipping set for key: {key}")
            return False
        
        start_time = time.time()
        try:
            # Serialize value
            if isinstance(value, (dict, list, tuple)):
                try:
                    serialized_value = json.dumps(value).encode('utf-8')
                except (TypeError, ValueError):
                    serialized_value = pickle.dumps(value)
            elif isinstance(value, str):
                serialized_value = value.encode('utf-8')
            else:
                serialized_value = pickle.dumps(value)
            
            async with self._lock:
                if ttl:
                    await self.redis_client.setex(key, ttl, serialized_value)
                else:
                    await self.redis_client.set(key, serialized_value)
            
            self.metrics["cache_sets"] += 1
            operation_time = time.time() - start_time
            self.metrics["operation_times"].append(operation_time)
            
            logger.debug(f"Cache SET for key: {key} (TTL: {ttl}s, time: {operation_time:.3f}s)")
            return True
            
        except Exception as e:
            self.metrics["connection_errors"] += 1
            logger.warning(f"Cache set error for key {key}: {e}")
            return False
    
    async def delete(self, key: str) -> bool:
        """Delete key from cache"""
        if self.fallback_mode or not self.is_healthy or not self.redis_client:
            return False
        
        try:
            async with self._lock:
                result = await self.redis_client.delete(key)
            
            self.metrics["cache_deletes"] += 1
            logger.debug(f"Cache DELETE for key: {key}")
            return bool(result)
            
        except Exception as e:
            self.metrics["connection_errors"] += 1
            logger.warning(f"Cache delete error for key {key}: {e}")
            return False
    
    async def delete_pattern(self, pattern: str) -> int:
        """Delete all keys matching pattern"""
        if self.fallback_mode or not self.is_healthy or not self.redis_client:
            return 0
        
        try:
            async with self._lock:
                keys = await self.redis_client.keys(pattern)
                if keys:
                    deleted_count = await self.redis_client.delete(*keys)
                    self.metrics["cache_deletes"] += deleted_count
                    logger.info(f"Cache DELETE pattern: {pattern} (deleted: {deleted_count})")
                    return deleted_count
            return 0
            
        except Exception as e:
            self.metrics["connection_errors"] += 1
            logger.warning(f"Cache delete pattern error for {pattern}: {e}")
            return 0
    
    async def exists(self, key: str) -> bool:
        """Check if key exists in cache"""
        if self.fallback_mode or not self.is_healthy or not self.redis_client:
            return False
        
        try:
            async with self._lock:
                result = await self.redis_client.exists(key)
            return bool(result)
            
        except Exception as e:
            logger.warning(f"Cache exists error for key {key}: {e}")
            return False
    
    async def get_ttl(self, key: str) -> int:
        """Get TTL for a key (-1 if no TTL, -2 if key doesn't exist)"""
        if self.fallback_mode or not self.is_healthy or not self.redis_client:
            return -2
        
        try:
            async with self._lock:
                ttl = await self.redis_client.ttl(key)
            return ttl
            
        except Exception as e:
            logger.warning(f"Cache TTL error for key {key}: {e}")
            return -2
    
    def _generate_key(self, prefix: str, identifier: str) -> str:
        """Generate cache key with prefix"""
        return f"{prefix}:{identifier}"
    
    # Device Configuration Cache Methods
    async def get_device_config(self, device_id: str) -> Optional[Dict[str, Any]]:
        """Get device configuration from cache"""
        key = self._generate_key(self.config.device_config_prefix, device_id)
        return await self.get(key)
    
    async def set_device_config(self, device_id: str, config: Dict[str, Any]) -> bool:
        """Set device configuration in cache"""
        key = self._generate_key(self.config.device_config_prefix, device_id)
        return await self.set(key, config, self.config.device_config_ttl)
    
    # Validation Rules Cache Methods
    async def get_validation_rules(self, device_type: str) -> Optional[Dict[str, Any]]:
        """Get validation rules from cache"""
        key = self._generate_key(self.config.validation_rules_prefix, device_type)
        return await self.get(key)
    
    async def set_validation_rules(self, device_type: str, rules: Dict[str, Any]) -> bool:
        """Set validation rules in cache"""
        key = self._generate_key(self.config.validation_rules_prefix, device_type)
        return await self.set(key, rules, self.config.validation_rules_ttl)
    
    # Patient Context Cache Methods
    async def get_patient_context(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get patient context from cache"""
        key = self._generate_key(self.config.patient_context_prefix, patient_id)
        return await self.get(key)
    
    async def set_patient_context(self, patient_id: str, context: Dict[str, Any]) -> bool:
        """Set patient context in cache"""
        key = self._generate_key(self.config.patient_context_prefix, patient_id)
        return await self.set(key, context, self.config.patient_context_ttl)
    
    # Auth Result Cache Methods
    async def get_auth_result(self, token_hash: str) -> Optional[Dict[str, Any]]:
        """Get auth result from cache"""
        key = self._generate_key(self.config.auth_result_prefix, token_hash)
        return await self.get(key)
    
    async def set_auth_result(self, token_hash: str, auth_result: Dict[str, Any]) -> bool:
        """Set auth result in cache"""
        key = self._generate_key(self.config.auth_result_prefix, token_hash)
        return await self.set(key, auth_result, self.config.auth_result_ttl)
    
    # Performance Metrics
    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get current performance metrics"""
        if self.fallback_mode:
            return {
                "status": "fallback_mode",
                "redis_available": False,
                "fallback_mode": True,
                "message": "Redis module not installed or Redis server not available",
                "cache_hit_rate": 0,
                "cache_hits": 0,
                "cache_misses": self.metrics["cache_misses"],
                "cache_sets": 0,
                "cache_deletes": 0,
                "connection_errors": 0,
                "avg_operation_time_ms": 0,
                "total_operations": self.metrics["cache_misses"]
            }

        cache_hit_rate = 0
        if self.metrics["cache_hits"] + self.metrics["cache_misses"] > 0:
            cache_hit_rate = (
                self.metrics["cache_hits"] /
                (self.metrics["cache_hits"] + self.metrics["cache_misses"])
            ) * 100

        avg_operation_time = 0
        if self.metrics["operation_times"]:
            avg_operation_time = sum(self.metrics["operation_times"]) / len(self.metrics["operation_times"])

        return {
            "status": "active",
            "redis_available": True,
            "fallback_mode": False,
            "is_healthy": self.is_healthy,
            "cache_hit_rate": round(cache_hit_rate, 2),
            "cache_hits": self.metrics["cache_hits"],
            "cache_misses": self.metrics["cache_misses"],
            "cache_sets": self.metrics["cache_sets"],
            "cache_deletes": self.metrics["cache_deletes"],
            "connection_errors": self.metrics["connection_errors"],
            "avg_operation_time_ms": round(avg_operation_time * 1000, 2),
            "total_operations": (
                self.metrics["cache_hits"] +
                self.metrics["cache_misses"] +
                self.metrics["cache_sets"] +
                self.metrics["cache_deletes"]
            )
        }
    
    async def cleanup(self):
        """Cleanup Redis connections and tasks"""
        if self.health_check_task:
            self.health_check_task.cancel()
            try:
                await self.health_check_task
            except asyncio.CancelledError:
                pass
        
        if self.redis_client:
            await self.redis_client.close()
        
        if self.pool:
            await self.pool.disconnect()
        
        logger.info("Redis cache manager cleaned up")


# Global cache manager instance
cache_manager: Optional[RedisCacheManager] = None


async def get_cache_manager() -> RedisCacheManager:
    """Get or create global cache manager instance"""
    global cache_manager
    
    if cache_manager is None:
        config = CacheConfig()
        cache_manager = RedisCacheManager(config)
        await cache_manager.initialize()
    
    return cache_manager


async def cleanup_cache_manager():
    """Cleanup global cache manager"""
    global cache_manager
    
    if cache_manager:
        await cache_manager.cleanup()
        cache_manager = None
