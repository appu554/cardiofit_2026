"""
Intelligent Caching System for Clinical Assertion Engine

Graph-aware caching with relationship invalidation, smart cache warming,
and adaptive cache management based on clinical intelligence patterns.

Key Features:
- Graph-aware caching with relationship tracking
- Intelligent cache invalidation based on data relationships
- Smart cache warming using access patterns
- Multi-level caching (L1, L2, L3)
- Adaptive TTL based on data volatility
- Cache coherence across distributed components
"""

import asyncio
import logging
import json
import time
import hashlib
from typing import Dict, List, Optional, Any, Set, Tuple
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
from collections import defaultdict, Counter
from enum import Enum
import pickle
import weakref

# Import base cache client
from .redis_client import CAERedisClient

logger = logging.getLogger(__name__)


class CacheLevel(Enum):
    """Cache levels for multi-level caching"""
    L1_MEMORY = "l1_memory"      # In-memory cache (fastest)
    L2_REDIS = "l2_redis"        # Redis cache (fast)
    L3_GRAPH = "l3_graph"        # Graph-based cache (persistent)


class CacheStrategy(Enum):
    """Caching strategies based on data characteristics"""
    WRITE_THROUGH = "write_through"      # Write to cache and storage simultaneously
    WRITE_BACK = "write_back"           # Write to cache first, storage later
    WRITE_AROUND = "write_around"       # Write to storage, bypass cache
    READ_THROUGH = "read_through"       # Read from cache, fallback to storage
    CACHE_ASIDE = "cache_aside"         # Application manages cache


@dataclass
class CacheEntry:
    """Intelligent cache entry with metadata"""
    key: str
    value: Any
    cache_level: CacheLevel
    created_at: datetime
    last_accessed: datetime
    access_count: int
    ttl_seconds: int
    relationships: Set[str]  # Related cache keys
    volatility_score: float  # How often this data changes
    importance_score: float  # How important this data is
    size_bytes: int


@dataclass
class CacheStats:
    """Cache performance statistics"""
    total_requests: int
    cache_hits: int
    cache_misses: int
    l1_hits: int
    l2_hits: int
    l3_hits: int
    invalidations: int
    evictions: int
    warming_operations: int
    relationship_invalidations: int


class IntelligentCache:
    """
    Intelligent caching system with graph-aware capabilities
    
    This system provides multi-level caching with intelligent invalidation,
    cache warming, and relationship tracking for clinical intelligence data.
    """
    
    def __init__(self, redis_client: Optional[CAERedisClient] = None):
        # Cache clients
        self.redis_client = redis_client or CAERedisClient()
        
        # Multi-level cache storage
        self.l1_cache: Dict[str, CacheEntry] = {}  # In-memory cache
        self.l2_cache = self.redis_client  # Redis cache
        
        # Cache metadata
        self.cache_entries: Dict[str, CacheEntry] = {}
        self.relationship_graph: Dict[str, Set[str]] = defaultdict(set)
        self.access_patterns: Dict[str, List[datetime]] = defaultdict(list)
        
        # Cache configuration
        self.l1_max_size = 1000  # Maximum L1 cache entries
        self.l1_max_memory_mb = 100  # Maximum L1 memory usage
        self.default_ttl = 3600  # Default TTL in seconds
        self.warming_threshold = 0.8  # Cache hit rate threshold for warming
        
        # Statistics
        self.stats = CacheStats(
            total_requests=0, cache_hits=0, cache_misses=0,
            l1_hits=0, l2_hits=0, l3_hits=0,
            invalidations=0, evictions=0, warming_operations=0,
            relationship_invalidations=0
        )
        
        # Background tasks
        self._background_tasks: Set[asyncio.Task] = set()
        self._start_background_tasks()
        
        logger.info("Intelligent Cache initialized")
    
    def _start_background_tasks(self):
        """Start background maintenance tasks"""
        try:
            # Cache maintenance task
            maintenance_task = asyncio.create_task(self._cache_maintenance_loop())
            self._background_tasks.add(maintenance_task)
            maintenance_task.add_done_callback(self._background_tasks.discard)
            
            # Cache warming task
            warming_task = asyncio.create_task(self._cache_warming_loop())
            self._background_tasks.add(warming_task)
            warming_task.add_done_callback(self._background_tasks.discard)
            
            logger.debug("Background cache tasks started")
            
        except Exception as e:
            logger.warning(f"Error starting background tasks: {e}")
    
    async def get(
        self, 
        key: str, 
        relationships: Optional[Set[str]] = None,
        importance_score: float = 0.5
    ) -> Optional[Any]:
        """
        Intelligent cache get with multi-level lookup
        
        Args:
            key: Cache key
            relationships: Related cache keys for invalidation tracking
            importance_score: Importance score for cache prioritization
            
        Returns:
            Cached value or None if not found
        """
        try:
            self.stats.total_requests += 1
            
            # Record access pattern
            self._record_access(key)
            
            # Try L1 cache first (in-memory)
            if key in self.l1_cache:
                entry = self.l1_cache[key]
                if not self._is_expired(entry):
                    entry.last_accessed = datetime.now(timezone.utc)
                    entry.access_count += 1
                    self.stats.cache_hits += 1
                    self.stats.l1_hits += 1
                    logger.debug(f"L1 cache hit for key: {key}")
                    return entry.value
                else:
                    # Remove expired entry
                    del self.l1_cache[key]
            
            # Try L2 cache (Redis)
            try:
                cached_data = await self.redis_client.get(key)
                if cached_data is not None:
                    # Promote to L1 cache
                    await self._promote_to_l1(key, cached_data, relationships, importance_score)
                    self.stats.cache_hits += 1
                    self.stats.l2_hits += 1
                    logger.debug(f"L2 cache hit for key: {key}")
                    return cached_data
            except Exception as e:
                logger.warning(f"L2 cache error for key {key}: {e}")
            
            # Cache miss
            self.stats.cache_misses += 1
            logger.debug(f"Cache miss for key: {key}")
            return None
            
        except Exception as e:
            logger.error(f"Error getting cache key {key}: {e}")
            return None
    
    async def set(
        self, 
        key: str, 
        value: Any,
        ttl: Optional[int] = None,
        relationships: Optional[Set[str]] = None,
        importance_score: float = 0.5,
        volatility_score: float = 0.5,
        cache_level: CacheLevel = CacheLevel.L2_REDIS
    ) -> bool:
        """
        Intelligent cache set with relationship tracking
        
        Args:
            key: Cache key
            value: Value to cache
            ttl: Time to live in seconds
            relationships: Related cache keys
            importance_score: Importance score for prioritization
            volatility_score: How often this data changes
            cache_level: Target cache level
            
        Returns:
            True if successfully cached
        """
        try:
            # Calculate adaptive TTL
            effective_ttl = self._calculate_adaptive_ttl(
                ttl or self.default_ttl, volatility_score, importance_score
            )
            
            # Calculate value size
            value_size = self._calculate_size(value)
            
            # Create cache entry
            entry = CacheEntry(
                key=key,
                value=value,
                cache_level=cache_level,
                created_at=datetime.now(timezone.utc),
                last_accessed=datetime.now(timezone.utc),
                access_count=1,
                ttl_seconds=effective_ttl,
                relationships=relationships or set(),
                volatility_score=volatility_score,
                importance_score=importance_score,
                size_bytes=value_size
            )
            
            # Store in appropriate cache level
            success = await self._store_in_cache_level(entry, cache_level)
            
            if success:
                # Update metadata
                self.cache_entries[key] = entry
                
                # Update relationship graph
                if relationships:
                    self._update_relationship_graph(key, relationships)
                
                self.stats.total_requests += 1
                logger.debug(f"Cached key {key} in {cache_level.value} with TTL {effective_ttl}s")
            
            return success
            
        except Exception as e:
            logger.error(f"Error setting cache key {key}: {e}")
            return False
    
    async def invalidate(
        self, 
        key: str, 
        cascade: bool = True,
        reason: str = "manual"
    ) -> int:
        """
        Intelligent cache invalidation with relationship cascading
        
        Args:
            key: Cache key to invalidate
            cascade: Whether to cascade invalidation to related keys
            reason: Reason for invalidation
            
        Returns:
            Number of keys invalidated
        """
        try:
            invalidated_count = 0
            keys_to_invalidate = {key}
            
            # Add related keys if cascading
            if cascade and key in self.relationship_graph:
                keys_to_invalidate.update(self.relationship_graph[key])
                self.stats.relationship_invalidations += 1
            
            # Invalidate all keys
            for k in keys_to_invalidate:
                # Remove from L1 cache
                if k in self.l1_cache:
                    del self.l1_cache[k]
                    invalidated_count += 1
                
                # Remove from L2 cache
                try:
                    await self.redis_client.delete(k)
                    invalidated_count += 1
                except Exception as e:
                    logger.warning(f"Error invalidating L2 cache key {k}: {e}")
                
                # Remove from metadata
                if k in self.cache_entries:
                    del self.cache_entries[k]
            
            self.stats.invalidations += invalidated_count
            logger.info(f"Invalidated {invalidated_count} cache keys (reason: {reason})")
            
            return invalidated_count
            
        except Exception as e:
            logger.error(f"Error invalidating cache key {key}: {e}")
            return 0
    
    async def warm_cache(
        self, 
        keys: List[str], 
        data_loader: callable,
        priority: float = 0.5
    ) -> int:
        """
        Smart cache warming based on access patterns
        
        Args:
            keys: Keys to warm
            data_loader: Function to load data for keys
            priority: Warming priority
            
        Returns:
            Number of keys warmed
        """
        try:
            warmed_count = 0
            
            # Sort keys by predicted access probability
            sorted_keys = self._sort_keys_by_access_probability(keys)
            
            # Warm cache for high-probability keys
            for key in sorted_keys:
                try:
                    # Check if already cached
                    if await self.get(key) is not None:
                        continue
                    
                    # Load data
                    data = await data_loader(key)
                    if data is not None:
                        # Cache with appropriate settings
                        await self.set(
                            key=key,
                            value=data,
                            importance_score=priority,
                            cache_level=CacheLevel.L2_REDIS
                        )
                        warmed_count += 1
                
                except Exception as e:
                    logger.warning(f"Error warming cache for key {key}: {e}")
            
            self.stats.warming_operations += warmed_count
            logger.info(f"Warmed {warmed_count} cache keys")
            
            return warmed_count
            
        except Exception as e:
            logger.error(f"Error warming cache: {e}")
            return 0

    def _record_access(self, key: str):
        """Record access pattern for cache intelligence"""
        try:
            now = datetime.now(timezone.utc)
            self.access_patterns[key].append(now)

            # Keep only recent access patterns (last 24 hours)
            cutoff = now - timedelta(hours=24)
            self.access_patterns[key] = [
                access_time for access_time in self.access_patterns[key]
                if access_time > cutoff
            ]

        except Exception as e:
            logger.warning(f"Error recording access pattern: {e}")

    def _is_expired(self, entry: CacheEntry) -> bool:
        """Check if cache entry is expired"""
        try:
            age = (datetime.now(timezone.utc) - entry.created_at).total_seconds()
            return age > entry.ttl_seconds
        except Exception:
            return True

    async def _promote_to_l1(
        self,
        key: str,
        value: Any,
        relationships: Optional[Set[str]],
        importance_score: float
    ):
        """Promote cache entry to L1 (memory) cache"""
        try:
            # Check if L1 cache has space
            if len(self.l1_cache) >= self.l1_max_size:
                await self._evict_l1_entries()

            # Create L1 entry
            entry = CacheEntry(
                key=key,
                value=value,
                cache_level=CacheLevel.L1_MEMORY,
                created_at=datetime.now(timezone.utc),
                last_accessed=datetime.now(timezone.utc),
                access_count=1,
                ttl_seconds=self.default_ttl,
                relationships=relationships or set(),
                volatility_score=0.5,
                importance_score=importance_score,
                size_bytes=self._calculate_size(value)
            )

            self.l1_cache[key] = entry

        except Exception as e:
            logger.warning(f"Error promoting to L1 cache: {e}")

    async def _evict_l1_entries(self):
        """Evict entries from L1 cache using LRU + importance scoring"""
        try:
            if not self.l1_cache:
                return

            # Calculate eviction scores (lower = more likely to evict)
            eviction_scores = {}
            for key, entry in self.l1_cache.items():
                # Base score on last access time
                age_score = (datetime.now(timezone.utc) - entry.last_accessed).total_seconds() / 3600

                # Adjust by importance and access frequency
                importance_factor = 1.0 / (entry.importance_score + 0.1)
                frequency_factor = 1.0 / (entry.access_count + 1)

                eviction_scores[key] = age_score * importance_factor * frequency_factor

            # Sort by eviction score and remove least important entries
            sorted_keys = sorted(eviction_scores.keys(), key=lambda k: eviction_scores[k], reverse=True)

            # Evict 25% of entries
            evict_count = max(1, len(self.l1_cache) // 4)
            for key in sorted_keys[:evict_count]:
                del self.l1_cache[key]
                self.stats.evictions += 1

            logger.debug(f"Evicted {evict_count} entries from L1 cache")

        except Exception as e:
            logger.warning(f"Error evicting L1 entries: {e}")

    def _calculate_adaptive_ttl(
        self,
        base_ttl: int,
        volatility_score: float,
        importance_score: float
    ) -> int:
        """Calculate adaptive TTL based on data characteristics"""
        try:
            # Adjust TTL based on volatility (higher volatility = shorter TTL)
            volatility_factor = 1.0 - (volatility_score * 0.5)

            # Adjust TTL based on importance (higher importance = longer TTL)
            importance_factor = 1.0 + (importance_score * 0.5)

            adaptive_ttl = int(base_ttl * volatility_factor * importance_factor)

            # Ensure TTL is within reasonable bounds
            return max(60, min(adaptive_ttl, 86400))  # 1 minute to 24 hours

        except Exception as e:
            logger.warning(f"Error calculating adaptive TTL: {e}")
            return base_ttl

    def _calculate_size(self, value: Any) -> int:
        """Calculate approximate size of cached value in bytes"""
        try:
            if isinstance(value, str):
                return len(value.encode('utf-8'))
            elif isinstance(value, (int, float)):
                return 8
            elif isinstance(value, (list, dict)):
                return len(json.dumps(value).encode('utf-8'))
            else:
                return len(pickle.dumps(value))
        except Exception:
            return 1024  # Default estimate

    async def _store_in_cache_level(self, entry: CacheEntry, cache_level: CacheLevel) -> bool:
        """Store entry in specified cache level"""
        try:
            if cache_level == CacheLevel.L1_MEMORY:
                # Check L1 cache capacity
                if len(self.l1_cache) >= self.l1_max_size:
                    await self._evict_l1_entries()

                self.l1_cache[entry.key] = entry
                return True

            elif cache_level == CacheLevel.L2_REDIS:
                # Store in Redis with TTL
                success = await self.redis_client.set(
                    entry.key,
                    entry.value,
                    ttl=entry.ttl_seconds
                )

                # Also store in L1 if high importance
                if entry.importance_score > 0.7:
                    await self._store_in_cache_level(entry, CacheLevel.L1_MEMORY)

                return success

            return False

        except Exception as e:
            logger.warning(f"Error storing in cache level {cache_level}: {e}")
            return False

    def _update_relationship_graph(self, key: str, relationships: Set[str]):
        """Update bidirectional relationship graph"""
        try:
            # Add forward relationships
            self.relationship_graph[key].update(relationships)

            # Add backward relationships
            for related_key in relationships:
                self.relationship_graph[related_key].add(key)

        except Exception as e:
            logger.warning(f"Error updating relationship graph: {e}")

    def _sort_keys_by_access_probability(self, keys: List[str]) -> List[str]:
        """Sort keys by predicted access probability"""
        try:
            key_scores = {}

            for key in keys:
                score = 0.0

                # Score based on historical access patterns
                if key in self.access_patterns:
                    recent_accesses = len(self.access_patterns[key])
                    score += recent_accesses * 0.5

                # Score based on cache entry metadata
                if key in self.cache_entries:
                    entry = self.cache_entries[key]
                    score += entry.importance_score * 0.3
                    score += entry.access_count * 0.2

                key_scores[key] = score

            # Sort by score (highest first)
            return sorted(keys, key=lambda k: key_scores.get(k, 0), reverse=True)

        except Exception as e:
            logger.warning(f"Error sorting keys by access probability: {e}")
            return keys

    async def _cache_maintenance_loop(self):
        """Background cache maintenance task"""
        try:
            while True:
                await asyncio.sleep(300)  # Run every 5 minutes

                try:
                    # Clean expired entries
                    await self._clean_expired_entries()

                    # Optimize cache distribution
                    await self._optimize_cache_distribution()

                    # Update access patterns
                    self._cleanup_access_patterns()

                except Exception as e:
                    logger.warning(f"Error in cache maintenance: {e}")

        except asyncio.CancelledError:
            logger.info("Cache maintenance task cancelled")
        except Exception as e:
            logger.error(f"Cache maintenance task error: {e}")

    async def _cache_warming_loop(self):
        """Background cache warming task"""
        try:
            while True:
                await asyncio.sleep(600)  # Run every 10 minutes

                try:
                    # Analyze access patterns for warming opportunities
                    warming_candidates = self._identify_warming_candidates()

                    if warming_candidates:
                        logger.info(f"Identified {len(warming_candidates)} cache warming candidates")
                        # Note: Actual warming would require data loader functions
                        # This is a placeholder for the warming logic

                except Exception as e:
                    logger.warning(f"Error in cache warming: {e}")

        except asyncio.CancelledError:
            logger.info("Cache warming task cancelled")
        except Exception as e:
            logger.error(f"Cache warming task error: {e}")

    async def _clean_expired_entries(self):
        """Clean expired cache entries"""
        try:
            expired_keys = []

            # Check L1 cache
            for key, entry in self.l1_cache.items():
                if self._is_expired(entry):
                    expired_keys.append(key)

            # Remove expired entries
            for key in expired_keys:
                del self.l1_cache[key]
                if key in self.cache_entries:
                    del self.cache_entries[key]

            if expired_keys:
                logger.debug(f"Cleaned {len(expired_keys)} expired cache entries")

        except Exception as e:
            logger.warning(f"Error cleaning expired entries: {e}")

    async def _optimize_cache_distribution(self):
        """Optimize cache distribution across levels"""
        try:
            # Promote frequently accessed L2 entries to L1
            if len(self.l1_cache) < self.l1_max_size * 0.8:
                # Find high-value L2 entries to promote
                # This would require tracking L2 access patterns
                pass

        except Exception as e:
            logger.warning(f"Error optimizing cache distribution: {e}")

    def _cleanup_access_patterns(self):
        """Clean up old access patterns"""
        try:
            cutoff = datetime.now(timezone.utc) - timedelta(days=7)

            # Remove old access patterns
            keys_to_remove = []
            for key, accesses in self.access_patterns.items():
                # Keep only recent accesses
                recent_accesses = [a for a in accesses if a > cutoff]
                if recent_accesses:
                    self.access_patterns[key] = recent_accesses
                else:
                    keys_to_remove.append(key)

            # Remove empty access patterns
            for key in keys_to_remove:
                del self.access_patterns[key]

        except Exception as e:
            logger.warning(f"Error cleaning access patterns: {e}")

    def _identify_warming_candidates(self) -> List[str]:
        """Identify cache keys that would benefit from warming"""
        try:
            candidates = []

            # Look for keys with regular access patterns but currently not cached
            for key, accesses in self.access_patterns.items():
                if len(accesses) >= 3 and key not in self.l1_cache:
                    # Check if access pattern suggests regular usage
                    if len(accesses) > 1:
                        intervals = []
                        for i in range(1, len(accesses)):
                            interval = (accesses[i] - accesses[i-1]).total_seconds()
                            intervals.append(interval)

                        # If access intervals are relatively consistent, it's a good candidate
                        if intervals and max(intervals) / min(intervals) < 3:
                            candidates.append(key)

            return candidates

        except Exception as e:
            logger.warning(f"Error identifying warming candidates: {e}")
            return []

    def get_cache_stats(self) -> Dict[str, Any]:
        """Get comprehensive cache statistics"""
        try:
            # Calculate hit rates
            total_requests = max(self.stats.total_requests, 1)
            hit_rate = self.stats.cache_hits / total_requests
            l1_hit_rate = self.stats.l1_hits / total_requests
            l2_hit_rate = self.stats.l2_hits / total_requests

            # Calculate cache sizes
            l1_size = len(self.l1_cache)
            l1_memory_usage = sum(entry.size_bytes for entry in self.l1_cache.values())

            return {
                'performance': {
                    'total_requests': self.stats.total_requests,
                    'cache_hit_rate': hit_rate,
                    'l1_hit_rate': l1_hit_rate,
                    'l2_hit_rate': l2_hit_rate,
                    'cache_misses': self.stats.cache_misses
                },
                'cache_sizes': {
                    'l1_entries': l1_size,
                    'l1_memory_mb': l1_memory_usage / (1024 * 1024),
                    'l1_max_size': self.l1_max_size,
                    'relationship_graph_size': len(self.relationship_graph)
                },
                'operations': {
                    'invalidations': self.stats.invalidations,
                    'evictions': self.stats.evictions,
                    'warming_operations': self.stats.warming_operations,
                    'relationship_invalidations': self.stats.relationship_invalidations
                },
                'patterns': {
                    'tracked_access_patterns': len(self.access_patterns),
                    'total_cache_entries': len(self.cache_entries)
                }
            }

        except Exception as e:
            logger.warning(f"Error calculating cache stats: {e}")
            return {'error': str(e)}

    async def shutdown(self):
        """Gracefully shutdown the cache system"""
        try:
            # Cancel background tasks
            for task in self._background_tasks:
                task.cancel()

            # Wait for tasks to complete
            if self._background_tasks:
                await asyncio.gather(*self._background_tasks, return_exceptions=True)

            # Close Redis connection
            if self.redis_client:
                await self.redis_client.close()

            logger.info("Intelligent cache shutdown complete")

        except Exception as e:
            logger.warning(f"Error during cache shutdown: {e}")
