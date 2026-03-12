"""
Device Cache Manager - High-level caching operations for device data ingestion

Provides intelligent caching strategies for device configurations, validation rules,
patient context, and authentication results with automatic cache warming and invalidation.
"""

import asyncio
import hashlib
import logging
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional, Tuple
import json

from app.cache.redis_manager import get_cache_manager, RedisCacheManager

logger = logging.getLogger(__name__)


class DeviceCacheManager:
    """High-level cache manager for device data ingestion operations"""
    
    def __init__(self):
        self.redis_manager: Optional[RedisCacheManager] = None
        self.cache_warming_enabled = True
        self.cache_warming_task: Optional[asyncio.Task] = None
        
        # Cache warming configuration
        self.warming_config = {
            "validation_rules_interval": 1800,  # 30 minutes
            "popular_devices_interval": 3600,   # 1 hour
            "patient_context_interval": 900     # 15 minutes
        }
        
        # Cache statistics
        self.stats = {
            "cache_warming_runs": 0,
            "cache_invalidations": 0,
            "cache_preloads": 0
        }
    
    async def initialize(self):
        """Initialize the device cache manager"""
        try:
            self.redis_manager = await get_cache_manager()
            
            if self.cache_warming_enabled:
                self.cache_warming_task = asyncio.create_task(self._cache_warming_loop())
            
            logger.info("Device cache manager initialized successfully")
            
        except Exception as e:
            logger.error(f"Failed to initialize device cache manager: {e}")
            raise
    
    # Authentication Result Caching
    async def get_cached_auth_result(self, token: str) -> Optional[Dict[str, Any]]:
        """Get cached authentication result"""
        if not self.redis_manager:
            return None
        
        # Create hash of token for cache key (don't store actual token)
        token_hash = hashlib.sha256(token.encode()).hexdigest()[:16]
        
        cached_result = await self.redis_manager.get_auth_result(token_hash)
        if cached_result:
            # Check if cached result is still valid
            cached_time = cached_result.get('cached_at')
            if cached_time:
                cache_age = datetime.now().timestamp() - cached_time
                if cache_age < self.redis_manager.config.auth_result_ttl:
                    logger.debug(f"Using cached auth result (age: {cache_age:.1f}s)")
                    return cached_result
        
        return None
    
    async def cache_auth_result(self, token: str, auth_result: Dict[str, Any]) -> bool:
        """Cache authentication result"""
        if not self.redis_manager:
            return False
        
        # Create hash of token for cache key
        token_hash = hashlib.sha256(token.encode()).hexdigest()[:16]
        
        # Add caching metadata
        cached_result = {
            **auth_result,
            'cached_at': datetime.now().timestamp(),
            'cache_version': '1.0'
        }
        
        success = await self.redis_manager.set_auth_result(token_hash, cached_result)
        if success:
            logger.debug("Auth result cached successfully")
        
        return success
    
    # Device Configuration Caching
    async def get_device_configuration(self, device_id: str) -> Optional[Dict[str, Any]]:
        """Get device configuration with intelligent caching"""
        if not self.redis_manager:
            return None
        
        # Try cache first
        cached_config = await self.redis_manager.get_device_config(device_id)
        if cached_config:
            logger.debug(f"Device config cache HIT for {device_id}")
            return cached_config
        
        # Cache miss - would typically fetch from database here
        logger.debug(f"Device config cache MISS for {device_id}")
        
        # For now, return default configuration
        default_config = self._get_default_device_config(device_id)
        
        # Cache the default config
        await self.redis_manager.set_device_config(device_id, default_config)
        
        return default_config
    
    def _get_default_device_config(self, device_id: str) -> Dict[str, Any]:
        """Get default device configuration"""
        return {
            "device_id": device_id,
            "sampling_rate": 60,  # seconds
            "data_retention_days": 30,
            "compression_enabled": True,
            "encryption_required": True,
            "max_batch_size": 100,
            "timeout_seconds": 30,
            "retry_attempts": 3,
            "created_at": datetime.now().isoformat(),
            "version": "1.0"
        }
    
    # Validation Rules Caching
    async def get_validation_rules_cached(self, device_type: str) -> Optional[Dict[str, Any]]:
        """Get validation rules with caching"""
        if not self.redis_manager:
            return None
        
        cached_rules = await self.redis_manager.get_validation_rules(device_type)
        if cached_rules:
            logger.debug(f"Validation rules cache HIT for {device_type}")
            return cached_rules
        
        logger.debug(f"Validation rules cache MISS for {device_type}")
        
        # Would typically load from YAML files here
        # For now, return None to trigger file loading
        return None
    
    async def cache_validation_rules(self, device_type: str, rules: Dict[str, Any]) -> bool:
        """Cache validation rules"""
        if not self.redis_manager:
            return False
        
        # Add caching metadata
        cached_rules = {
            **rules,
            'cached_at': datetime.now().timestamp(),
            'cache_version': '1.0'
        }
        
        success = await self.redis_manager.set_validation_rules(device_type, cached_rules)
        if success:
            logger.debug(f"Validation rules cached for {device_type}")
        
        return success
    
    # Patient Context Caching
    async def get_patient_context_cached(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get patient context with caching"""
        if not self.redis_manager:
            return None
        
        cached_context = await self.redis_manager.get_patient_context(patient_id)
        if cached_context:
            logger.debug(f"Patient context cache HIT for {patient_id}")
            return cached_context
        
        logger.debug(f"Patient context cache MISS for {patient_id}")
        return None
    
    async def cache_patient_context(self, patient_id: str, context: Dict[str, Any]) -> bool:
        """Cache patient context"""
        if not self.redis_manager:
            return False
        
        # Add caching metadata
        cached_context = {
            **context,
            'cached_at': datetime.now().timestamp(),
            'cache_version': '1.0'
        }
        
        success = await self.redis_manager.set_patient_context(patient_id, cached_context)
        if success:
            logger.debug(f"Patient context cached for {patient_id}")
        
        return success
    
    # Cache Warming and Preloading
    async def _cache_warming_loop(self):
        """Background task for cache warming"""
        while True:
            try:
                await asyncio.sleep(self.warming_config["validation_rules_interval"])
                await self._warm_validation_rules_cache()
                self.stats["cache_warming_runs"] += 1
                
            except Exception as e:
                logger.error(f"Cache warming error: {e}")
    
    async def _warm_validation_rules_cache(self):
        """Warm up validation rules cache"""
        try:
            # Common device types to pre-warm
            device_types = [
                "heart_rate", "blood_pressure", "blood_glucose", 
                "temperature", "oxygen_saturation", "weight"
            ]
            
            for device_type in device_types:
                # Check if already cached
                cached = await self.redis_manager.get_validation_rules(device_type)
                if not cached:
                    # Would load from YAML and cache here
                    logger.debug(f"Would warm cache for {device_type} validation rules")
            
            logger.debug("Validation rules cache warming completed")
            
        except Exception as e:
            logger.error(f"Validation rules cache warming failed: {e}")
    
    async def preload_device_configurations(self, device_ids: List[str]) -> int:
        """Preload device configurations into cache"""
        if not self.redis_manager:
            return 0
        
        preloaded_count = 0
        
        for device_id in device_ids:
            try:
                # Check if already cached
                cached = await self.redis_manager.get_device_config(device_id)
                if not cached:
                    # Load and cache configuration
                    config = self._get_default_device_config(device_id)
                    success = await self.redis_manager.set_device_config(device_id, config)
                    if success:
                        preloaded_count += 1
                        
            except Exception as e:
                logger.warning(f"Failed to preload config for device {device_id}: {e}")
        
        self.stats["cache_preloads"] += preloaded_count
        logger.info(f"Preloaded {preloaded_count} device configurations")
        
        return preloaded_count
    
    # Cache Invalidation
    async def invalidate_device_cache(self, device_id: str) -> bool:
        """Invalidate all cache entries for a device"""
        if not self.redis_manager:
            return False
        
        try:
            # Delete device configuration
            await self.redis_manager.delete(
                self.redis_manager._generate_key(
                    self.redis_manager.config.device_config_prefix, 
                    device_id
                )
            )
            
            self.stats["cache_invalidations"] += 1
            logger.info(f"Invalidated cache for device {device_id}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to invalidate cache for device {device_id}: {e}")
            return False
    
    async def invalidate_patient_cache(self, patient_id: str) -> bool:
        """Invalidate patient context cache"""
        if not self.redis_manager:
            return False
        
        try:
            await self.redis_manager.delete(
                self.redis_manager._generate_key(
                    self.redis_manager.config.patient_context_prefix, 
                    patient_id
                )
            )
            
            self.stats["cache_invalidations"] += 1
            logger.info(f"Invalidated patient cache for {patient_id}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to invalidate patient cache for {patient_id}: {e}")
            return False
    
    async def invalidate_validation_rules_cache(self, device_type: str) -> bool:
        """Invalidate validation rules cache"""
        if not self.redis_manager:
            return False
        
        try:
            await self.redis_manager.delete(
                self.redis_manager._generate_key(
                    self.redis_manager.config.validation_rules_prefix, 
                    device_type
                )
            )
            
            self.stats["cache_invalidations"] += 1
            logger.info(f"Invalidated validation rules cache for {device_type}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to invalidate validation rules cache for {device_type}: {e}")
            return False
    
    # Performance and Statistics
    async def get_cache_statistics(self) -> Dict[str, Any]:
        """Get comprehensive cache statistics"""
        if not self.redis_manager:
            return {"error": "Cache manager not initialized"}
        
        redis_metrics = await self.redis_manager.get_performance_metrics()
        
        return {
            "redis_metrics": redis_metrics,
            "device_cache_stats": self.stats,
            "cache_warming_enabled": self.cache_warming_enabled,
            "warming_config": self.warming_config,
            "timestamp": datetime.now().isoformat()
        }
    
    async def get_cache_health(self) -> Dict[str, Any]:
        """Get cache health status"""
        if not self.redis_manager:
            return {
                "status": "unhealthy",
                "reason": "Cache manager not initialized"
            }
        
        redis_healthy = self.redis_manager.is_healthy
        
        return {
            "status": "healthy" if redis_healthy else "unhealthy",
            "redis_healthy": redis_healthy,
            "cache_warming_active": self.cache_warming_task is not None and not self.cache_warming_task.done(),
            "timestamp": datetime.now().isoformat()
        }
    
    async def cleanup(self):
        """Cleanup cache manager resources"""
        if self.cache_warming_task:
            self.cache_warming_task.cancel()
            try:
                await self.cache_warming_task
            except asyncio.CancelledError:
                pass
        
        logger.info("Device cache manager cleaned up")


# Global device cache manager instance
device_cache_manager: Optional[DeviceCacheManager] = None


async def get_device_cache_manager() -> DeviceCacheManager:
    """Get or create global device cache manager instance"""
    global device_cache_manager
    
    if device_cache_manager is None:
        device_cache_manager = DeviceCacheManager()
        await device_cache_manager.initialize()
    
    return device_cache_manager


async def cleanup_device_cache_manager():
    """Cleanup global device cache manager"""
    global device_cache_manager
    
    if device_cache_manager:
        await device_cache_manager.cleanup()
        device_cache_manager = None
