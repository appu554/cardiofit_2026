"""
Performance Optimizations for Enhanced Envelope Factory

Implements lazy loading, async processing, caching strategies, and memory pooling
for sub-50ms envelope creation performance.
"""

import asyncio
import logging
import time
from datetime import datetime
from typing import Any, Dict, List, Optional, Callable
from dataclasses import dataclass
from collections import deque

logger = logging.getLogger(__name__)


@dataclass
class LazyMetadata:
    """Lazy-loaded metadata container"""
    loader_func: Callable
    cache_key: Optional[str] = None
    ttl_seconds: int = 300  # 5 minutes default
    loaded_at: Optional[float] = None
    _value: Optional[Any] = None
    _loading: bool = False
    
    async def get_value(self) -> Any:
        """Get value with lazy loading"""
        # Check if already loaded and not expired
        if self._value is not None and self.loaded_at:
            if time.time() - self.loaded_at < self.ttl_seconds:
                return self._value
        
        # Prevent concurrent loading
        if self._loading:
            while self._loading:
                await asyncio.sleep(0.01)
            return self._value
        
        try:
            self._loading = True
            self._value = await self.loader_func()
            self.loaded_at = time.time()
            return self._value
        finally:
            self._loading = False


class EnvelopeObjectPool:
    """Object pool for envelope instances to reduce GC pressure"""
    
    def __init__(self, max_size: int = 1000):
        self.max_size = max_size
        self.pool: deque = deque()
        self.created_count = 0
        self.reused_count = 0
        self._lock = asyncio.Lock()
    
    async def get_envelope_template(self) -> Dict[str, Any]:
        """Get a reusable envelope template"""
        async with self._lock:
            if self.pool:
                self.reused_count += 1
                return self.pool.popleft()
            else:
                self.created_count += 1
                return self._create_template()
    
    async def return_envelope_template(self, template: Dict[str, Any]):
        """Return envelope template to pool"""
        async with self._lock:
            if len(self.pool) < self.max_size:
                self._reset_template(template)
                self.pool.append(template)
    
    def _create_template(self) -> Dict[str, Any]:
        """Create new envelope template"""
        return {
            "id": None,
            "source": None,
            "type": None,
            "subject": None,
            "time": None,
            "data": None,
            "version": "2.2",
            "security": None,
            "quality": None,
            "patient_context": None,
            "device_context": None,
            "processing_hints": None,
            "lineage": None,
            "correlation_id": None,
            "metadata": {}
        }
    
    def _reset_template(self, template: Dict[str, Any]):
        """Reset template for reuse"""
        for key in template:
            if key == "version":
                template[key] = "2.2"
            elif key == "metadata":
                template[key] = {}
            else:
                template[key] = None
    
    def get_stats(self) -> Dict[str, Any]:
        """Get pool statistics"""
        return {
            "pool_size": len(self.pool),
            "max_size": self.max_size,
            "created_count": self.created_count,
            "reused_count": self.reused_count,
            "reuse_rate": self.reused_count / max(self.created_count + self.reused_count, 1)
        }


class AsyncEnrichmentProcessor:
    """Async processor for non-critical envelope enrichments"""
    
    def __init__(self, max_workers: int = 4):
        self.enrichment_queue: asyncio.Queue = asyncio.Queue()
        self.processing_task: Optional[asyncio.Task] = None
        self.is_running = False
        
        # Enrichment metrics
        self.metrics = {
            "total_enrichments": 0,
            "successful_enrichments": 0,
            "failed_enrichments": 0,
            "enrichment_times": deque(maxlen=1000)
        }
    
    async def start(self):
        """Start async enrichment processing"""
        if not self.is_running:
            self.is_running = True
            self.processing_task = asyncio.create_task(self._process_enrichments())
            logger.info("Async enrichment processor started")
    
    async def stop(self):
        """Stop async enrichment processing"""
        self.is_running = False
        if self.processing_task:
            self.processing_task.cancel()
            try:
                await self.processing_task
            except asyncio.CancelledError:
                pass
        logger.info("Async enrichment processor stopped")
    
    async def enqueue_enrichment(self, 
                                envelope_id: str,
                                enrichment_func: Callable,
                                callback: Optional[Callable] = None):
        """Enqueue enrichment for async processing"""
        enrichment_item = {
            "envelope_id": envelope_id,
            "enrichment_func": enrichment_func,
            "callback": callback,
            "queued_at": time.time()
        }
        
        await self.enrichment_queue.put(enrichment_item)
    
    async def _process_enrichments(self):
        """Process enrichments asynchronously"""
        while self.is_running:
            try:
                enrichment_item = await asyncio.wait_for(
                    self.enrichment_queue.get(), timeout=1.0
                )
                await self._process_single_enrichment(enrichment_item)
            except asyncio.TimeoutError:
                continue
            except Exception as e:
                logger.error(f"Error in enrichment processing: {e}")
    
    async def _process_single_enrichment(self, enrichment_item: Dict[str, Any]):
        """Process a single enrichment item"""
        start_time = time.time()
        
        try:
            self.metrics["total_enrichments"] += 1
            
            enrichment_func = enrichment_item["enrichment_func"]
            
            if asyncio.iscoroutinefunction(enrichment_func):
                result = await enrichment_func()
            else:
                result = enrichment_func()
            
            callback = enrichment_item.get("callback")
            if callback:
                if asyncio.iscoroutinefunction(callback):
                    await callback(result)
                else:
                    callback(result)
            
            self.metrics["successful_enrichments"] += 1
            
        except Exception as e:
            self.metrics["failed_enrichments"] += 1
            logger.warning(f"Enrichment failed for envelope {enrichment_item['envelope_id']}: {e}")
        
        finally:
            enrichment_time = time.time() - start_time
            self.metrics["enrichment_times"].append(enrichment_time)
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get enrichment processing metrics"""
        return {
            "queue_size": self.enrichment_queue.qsize(),
            "total_enrichments": self.metrics["total_enrichments"],
            "successful_enrichments": self.metrics["successful_enrichments"],
            "failed_enrichments": self.metrics["failed_enrichments"],
            "success_rate": (
                self.metrics["successful_enrichments"] / 
                max(self.metrics["total_enrichments"], 1)
            ),
            "is_running": self.is_running
        }


class CacheIntegration:
    """Integration layer for caching in envelope creation"""
    
    def __init__(self, cache_manager):
        self.cache_manager = cache_manager
        self.cache_hits = 0
        self.cache_misses = 0
        self.cache_sets = 0
    
    async def get_device_configuration(self, device_id: str, device_type: str) -> Optional[Dict[str, Any]]:
        """Get device configuration with caching"""
        if not self.cache_manager:
            return None
        
        # Try cache first
        cached_config = await self.cache_manager.get_device_configuration(device_id)
        if cached_config:
            self.cache_hits += 1
            return cached_config
        
        self.cache_misses += 1
        
        # Load from source (simulate external service)
        config = await self._load_device_configuration(device_id, device_type)
        
        # Cache the result
        if config:
            await self.cache_manager.set_device_config(device_id, config)
            self.cache_sets += 1
        
        return config
    
    async def get_patient_context(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get patient context with caching"""
        if not self.cache_manager:
            return None
        
        # Try cache first
        cached_context = await self.cache_manager.get_patient_context_cached(patient_id)
        if cached_context:
            self.cache_hits += 1
            return cached_context
        
        self.cache_misses += 1
        
        # Load from source (simulate Patient Service)
        context = await self._load_patient_context(patient_id)
        
        # Cache the result
        if context:
            await self.cache_manager.cache_patient_context(patient_id, context)
            self.cache_sets += 1
        
        return context
    
    async def _load_device_configuration(self, device_id: str, device_type: str) -> Optional[Dict[str, Any]]:
        """Load device configuration from source"""
        # Simulate external service call
        await asyncio.sleep(0.01)
        
        return {
            "device_id": device_id,
            "device_type": device_type,
            "manufacturer": "Generic",
            "model": "Unknown",
            "capabilities": ["basic_monitoring"],
            "loaded_at": time.time()
        }
    
    async def _load_patient_context(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Load patient context from source"""
        # Simulate Patient Service call
        await asyncio.sleep(0.02)
        
        return {
            "patient_id": patient_id,
            "consent_status": "active",
            "data_sharing_permissions": ["research", "care_coordination"],
            "clinical_conditions": [],
            "medications": [],
            "loaded_at": time.time()
        }
    
    def get_cache_stats(self) -> Dict[str, Any]:
        """Get cache integration statistics"""
        total_operations = self.cache_hits + self.cache_misses
        hit_rate = (self.cache_hits / total_operations * 100) if total_operations > 0 else 0
        
        return {
            "cache_hits": self.cache_hits,
            "cache_misses": self.cache_misses,
            "cache_sets": self.cache_sets,
            "hit_rate_percent": round(hit_rate, 2),
            "total_operations": total_operations
        }


class PerformanceOptimizer:
    """Main performance optimization coordinator"""

    def __init__(self, cache_manager=None):
        self.object_pool = EnvelopeObjectPool()
        self.async_processor = AsyncEnrichmentProcessor()
        self.cache_integration = CacheIntegration(cache_manager) if cache_manager else None

        # Performance metrics
        self.optimization_metrics = {
            "envelope_creation_times": deque(maxlen=1000),
            "cache_enabled": cache_manager is not None,
            "async_processing_enabled": True,
            "object_pooling_enabled": True
        }

    async def initialize(self):
        """Initialize performance optimizations"""
        await self.async_processor.start()
        logger.info("Performance optimizations initialized")

    async def cleanup(self):
        """Cleanup performance optimizations"""
        await self.async_processor.stop()
        logger.info("Performance optimizations cleaned up")

    async def get_envelope_template(self) -> Dict[str, Any]:
        """Get optimized envelope template"""
        return await self.object_pool.get_envelope_template()

    async def return_envelope_template(self, template: Dict[str, Any]):
        """Return envelope template to pool"""
        await self.object_pool.return_envelope_template(template)

    async def get_cached_metadata(self, metadata_type: str, key: str) -> Optional[Dict[str, Any]]:
        """Get cached metadata with optimization"""
        if not self.cache_integration:
            return None

        if metadata_type == "device_config":
            device_id, device_type = key.split(":", 1) if ":" in key else (key, "unknown")
            return await self.cache_integration.get_device_configuration(device_id, device_type)
        elif metadata_type == "patient_context":
            return await self.cache_integration.get_patient_context(key)

        return None

    async def enqueue_async_enrichment(self,
                                     envelope_id: str,
                                     enrichment_func: Callable,
                                     callback: Optional[Callable] = None):
        """Enqueue async enrichment"""
        await self.async_processor.enqueue_enrichment(
            envelope_id, enrichment_func, callback
        )

    def record_creation_time(self, creation_time: float):
        """Record envelope creation time"""
        self.optimization_metrics["envelope_creation_times"].append(creation_time)

    def get_performance_metrics(self) -> Dict[str, Any]:
        """Get comprehensive performance metrics"""
        creation_times = list(self.optimization_metrics["envelope_creation_times"])

        metrics = {
            "object_pool": self.object_pool.get_stats(),
            "async_processor": self.async_processor.get_metrics(),
            "optimization_status": {
                "cache_enabled": self.optimization_metrics["cache_enabled"],
                "async_processing_enabled": self.optimization_metrics["async_processing_enabled"],
                "object_pooling_enabled": self.optimization_metrics["object_pooling_enabled"]
            }
        }

        if self.cache_integration:
            metrics["cache_integration"] = self.cache_integration.get_cache_stats()

        if creation_times:
            avg_time = sum(creation_times) / len(creation_times)
            p95_time = sorted(creation_times)[int(len(creation_times) * 0.95)] if len(creation_times) > 20 else avg_time

            metrics["envelope_creation"] = {
                "avg_time_ms": round(avg_time * 1000, 2),
                "p95_time_ms": round(p95_time * 1000, 2),
                "total_created": len(creation_times),
                "sub_50ms_rate": sum(1 for t in creation_times if t < 0.05) / len(creation_times)
            }

        return metrics
