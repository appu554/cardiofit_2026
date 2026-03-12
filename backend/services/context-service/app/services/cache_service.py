"""
Multi-Layer Cache Service for Clinical Context
Implements Pillar 3: Multi-Layer Intelligent Cache (The "Performance Accelerator")
"""
import logging
import json
import asyncio
from typing import Optional, Dict, Any, List
from datetime import datetime, timedelta
import redis.asyncio as redis
import time

from app.models.context_models import ClinicalContext

logger = logging.getLogger(__name__)


class CacheService:
    """
    Multi-layer intelligent cache system for clinical context.
    Implements L1 (in-process) + L2 (distributed Redis) + L3 (service-level) caching.
    """
    
    def __init__(self):
        # L1: In-Process Workflow Cache (fastest, workflow-scoped)
        self.l1_workflow_cache: Dict[str, Any] = {}
        self.l1_cache_stats = {
            "hits": 0,
            "misses": 0,
            "entries": 0,
            "max_size": 1000
        }
        
        # L2: Distributed Redis Cache (shared across instances)
        self.l2_redis_client = None
        self.l2_cache_stats = {
            "hits": 0,
            "misses": 0,
            "connection_errors": 0
        }
        
        # Cache configuration
        self.default_ttl_seconds = 300  # 5 minutes
        self.max_l1_entries = 1000
        self.redis_connection_timeout = 5
        
        # Performance tracking
        self.performance_metrics = {
            "l1_avg_response_time_ms": 0.0,
            "l2_avg_response_time_ms": 0.0,
            "overall_hit_ratio": 0.0,
            "cache_warming_active": False
        }
        
        # Initialize Redis connection
        asyncio.create_task(self._initialize_redis())
    
    async def _initialize_redis(self):
        """Initialize Redis connection for L2 cache"""
        try:
            self.l2_redis_client = redis.Redis(
                host="localhost",
                port=6379,
                decode_responses=True,
                socket_connect_timeout=self.redis_connection_timeout,
                socket_timeout=self.redis_connection_timeout,
                health_check_interval=30,
                retry_on_timeout=True,
                max_connections=20
            )
            
            # Test connection
            await self.l2_redis_client.ping()
            logger.info("✅ Redis L2 cache connection established")
            
        except Exception as e:
            logger.warning(f"⚠️ Redis L2 cache unavailable: {e}")
            self.l2_redis_client = None
    
    async def get(self, cache_key: str, workflow_id: Optional[str] = None) -> Optional[ClinicalContext]:
        """
        Multi-layer cache retrieval with performance optimization.
        Implements the L1 -> L2 -> L3 cache hierarchy.
        """
        start_time = time.time()
        
        try:
            # L1: Check workflow-scoped cache first (fastest)
            if workflow_id:
                workflow_cache_key = f"{workflow_id}:{cache_key}"
                if workflow_cache_key in self.l1_workflow_cache:
                    self.l1_cache_stats["hits"] += 1
                    response_time = (time.time() - start_time) * 1000
                    self._update_l1_performance_metrics(response_time)
                    
                    logger.debug(f"🚀 L1 cache hit: {cache_key} ({response_time:.2f}ms)")
                    return self.l1_workflow_cache[workflow_cache_key]
            
            # Check regular L1 cache
            if cache_key in self.l1_workflow_cache:
                self.l1_cache_stats["hits"] += 1
                response_time = (time.time() - start_time) * 1000
                self._update_l1_performance_metrics(response_time)
                
                logger.debug(f"🚀 L1 cache hit: {cache_key} ({response_time:.2f}ms)")
                return self.l1_workflow_cache[cache_key]
            
            self.l1_cache_stats["misses"] += 1
            
            # L2: Check distributed Redis cache
            if self.l2_redis_client:
                try:
                    l2_start_time = time.time()
                    cached_data = await self.l2_redis_client.get(cache_key)
                    
                    if cached_data:
                        self.l2_cache_stats["hits"] += 1
                        
                        # Deserialize clinical context
                        context = ClinicalContext.from_json(cached_data)
                        
                        # Promote to L1 for workflow
                        await self._promote_to_l1(cache_key, context, workflow_id)
                        
                        response_time = (time.time() - l2_start_time) * 1000
                        self._update_l2_performance_metrics(response_time)
                        
                        logger.debug(f"⚡ L2 cache hit: {cache_key} ({response_time:.2f}ms)")
                        return context
                    else:
                        self.l2_cache_stats["misses"] += 1
                        
                except Exception as e:
                    self.l2_cache_stats["connection_errors"] += 1
                    logger.warning(f"L2 cache error for {cache_key}: {e}")
            
            # L3: Cache miss - would fetch fresh data (handled by caller)
            logger.debug(f"❌ Cache miss: {cache_key}")
            return None
            
        except Exception as e:
            logger.error(f"Cache retrieval error for {cache_key}: {e}")
            return None
        
        finally:
            self._update_overall_performance_metrics()
    
    async def set(
        self,
        cache_key: str,
        context: ClinicalContext,
        ttl_seconds: Optional[int] = None,
        workflow_id: Optional[str] = None
    ):
        """
        Multi-layer cache storage with intelligent promotion.
        Stores in both L1 and L2 caches for optimal performance.
        """
        if ttl_seconds is None:
            ttl_seconds = self.default_ttl_seconds
        
        try:
            # Store in L1 cache (in-process)
            await self._store_in_l1(cache_key, context, workflow_id)
            
            # Store in L2 cache (distributed Redis)
            if self.l2_redis_client:
                try:
                    context_json = context.to_json()
                    await self.l2_redis_client.setex(
                        cache_key,
                        ttl_seconds,
                        context_json
                    )
                    logger.debug(f"💾 Stored in L2 cache: {cache_key} (TTL: {ttl_seconds}s)")
                    
                except Exception as e:
                    logger.warning(f"L2 cache storage error for {cache_key}: {e}")
            
            logger.debug(f"✅ Context cached: {cache_key}")
            
        except Exception as e:
            logger.error(f"Cache storage error for {cache_key}: {e}")
    
    async def invalidate(self, cache_key: str, workflow_id: Optional[str] = None):
        """
        Invalidate cache entry across all layers.
        Used for cache invalidation events.
        """
        try:
            # Invalidate L1 cache
            if workflow_id:
                workflow_cache_key = f"{workflow_id}:{cache_key}"
                if workflow_cache_key in self.l1_workflow_cache:
                    del self.l1_workflow_cache[workflow_cache_key]
                    self.l1_cache_stats["entries"] -= 1
            
            if cache_key in self.l1_workflow_cache:
                del self.l1_workflow_cache[cache_key]
                self.l1_cache_stats["entries"] -= 1
            
            # Invalidate L2 cache
            if self.l2_redis_client:
                try:
                    await self.l2_redis_client.delete(cache_key)
                except Exception as e:
                    logger.warning(f"L2 cache invalidation error for {cache_key}: {e}")
            
            logger.debug(f"🗑️ Cache invalidated: {cache_key}")
            
        except Exception as e:
            logger.error(f"Cache invalidation error for {cache_key}: {e}")
    
    async def invalidate_patient_contexts(self, patient_id: str):
        """
        Invalidate all cache entries for a patient.
        Used when patient data changes.
        """
        try:
            # L1: Clear workflow caches
            keys_to_remove = [
                key for key in self.l1_workflow_cache.keys()
                if patient_id in key
            ]
            
            for key in keys_to_remove:
                del self.l1_workflow_cache[key]
                self.l1_cache_stats["entries"] -= 1
            
            # L2: Clear distributed cache
            if self.l2_redis_client:
                try:
                    pattern = f"*patient:{patient_id}*"
                    keys = await self.l2_redis_client.keys(pattern)
                    if keys:
                        await self.l2_redis_client.delete(*keys)
                        logger.info(f"🗑️ Invalidated {len(keys)} L2 cache entries for patient {patient_id}")
                except Exception as e:
                    logger.warning(f"L2 cache pattern invalidation error: {e}")
            
            logger.info(f"🗑️ Cache invalidated for patient {patient_id}")
            
        except Exception as e:
            logger.error(f"Patient cache invalidation error: {e}")
    
    async def warm_cache(self, cache_keys: List[str], contexts: List[ClinicalContext]):
        """
        Predictive cache warming for frequently accessed contexts.
        """
        self.performance_metrics["cache_warming_active"] = True
        
        try:
            warming_tasks = []
            for cache_key, context in zip(cache_keys, contexts):
                task = asyncio.create_task(
                    self.set(cache_key, context, ttl_seconds=600)  # 10 minutes for warmed cache
                )
                warming_tasks.append(task)
            
            await asyncio.gather(*warming_tasks, return_exceptions=True)
            logger.info(f"🔥 Cache warmed with {len(cache_keys)} entries")
            
        except Exception as e:
            logger.error(f"Cache warming error: {e}")
        
        finally:
            self.performance_metrics["cache_warming_active"] = False
    
    async def get_cache_stats(self) -> Dict[str, Any]:
        """Get comprehensive cache statistics"""
        l2_info = {}
        if self.l2_redis_client:
            try:
                l2_info = await self.l2_redis_client.info("memory")
            except Exception:
                l2_info = {"error": "Redis unavailable"}
        
        return {
            "l1_cache": {
                "hits": self.l1_cache_stats["hits"],
                "misses": self.l1_cache_stats["misses"],
                "entries": self.l1_cache_stats["entries"],
                "max_size": self.l1_cache_stats["max_size"],
                "hit_ratio": self._calculate_l1_hit_ratio()
            },
            "l2_cache": {
                "hits": self.l2_cache_stats["hits"],
                "misses": self.l2_cache_stats["misses"],
                "connection_errors": self.l2_cache_stats["connection_errors"],
                "hit_ratio": self._calculate_l2_hit_ratio(),
                "redis_info": l2_info
            },
            "performance": self.performance_metrics,
            "overall_hit_ratio": self.performance_metrics["overall_hit_ratio"]
        }
    
    async def _promote_to_l1(self, cache_key: str, context: ClinicalContext, workflow_id: Optional[str]):
        """Promote L2 cache hit to L1 cache"""
        await self._store_in_l1(cache_key, context, workflow_id)
    
    async def _store_in_l1(self, cache_key: str, context: ClinicalContext, workflow_id: Optional[str]):
        """Store context in L1 cache with size management"""
        # Check L1 cache size limit
        if len(self.l1_workflow_cache) >= self.max_l1_entries:
            await self._evict_l1_entries()
        
        # Store in workflow-specific cache if workflow_id provided
        if workflow_id:
            workflow_cache_key = f"{workflow_id}:{cache_key}"
            self.l1_workflow_cache[workflow_cache_key] = context
        
        # Always store in regular cache
        self.l1_workflow_cache[cache_key] = context
        self.l1_cache_stats["entries"] = len(self.l1_workflow_cache)
    
    async def _evict_l1_entries(self):
        """Evict oldest L1 cache entries when size limit reached"""
        # Simple LRU eviction - remove 10% of entries
        entries_to_remove = max(1, len(self.l1_workflow_cache) // 10)
        
        # Remove oldest entries (this is simplified - in production would use proper LRU)
        keys_to_remove = list(self.l1_workflow_cache.keys())[:entries_to_remove]
        
        for key in keys_to_remove:
            del self.l1_workflow_cache[key]
        
        self.l1_cache_stats["entries"] = len(self.l1_workflow_cache)
        logger.debug(f"🧹 Evicted {entries_to_remove} L1 cache entries")
    
    def _update_l1_performance_metrics(self, response_time_ms: float):
        """Update L1 cache performance metrics"""
        current_avg = self.performance_metrics["l1_avg_response_time_ms"]
        self.performance_metrics["l1_avg_response_time_ms"] = (current_avg + response_time_ms) / 2
    
    def _update_l2_performance_metrics(self, response_time_ms: float):
        """Update L2 cache performance metrics"""
        current_avg = self.performance_metrics["l2_avg_response_time_ms"]
        self.performance_metrics["l2_avg_response_time_ms"] = (current_avg + response_time_ms) / 2
    
    def _update_overall_performance_metrics(self):
        """Update overall cache performance metrics"""
        l1_hit_ratio = self._calculate_l1_hit_ratio()
        l2_hit_ratio = self._calculate_l2_hit_ratio()
        
        # Weighted average based on cache layer usage
        self.performance_metrics["overall_hit_ratio"] = (l1_hit_ratio * 0.7) + (l2_hit_ratio * 0.3)
    
    def _calculate_l1_hit_ratio(self) -> float:
        """Calculate L1 cache hit ratio"""
        total_requests = self.l1_cache_stats["hits"] + self.l1_cache_stats["misses"]
        if total_requests == 0:
            return 0.0
        return self.l1_cache_stats["hits"] / total_requests
    
    def _calculate_l2_hit_ratio(self) -> float:
        """Calculate L2 cache hit ratio"""
        total_requests = self.l2_cache_stats["hits"] + self.l2_cache_stats["misses"]
        if total_requests == 0:
            return 0.0
        return self.l2_cache_stats["hits"] / total_requests
