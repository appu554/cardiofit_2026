"""
L1 Cache Manager
Ultra-fast in-memory cache with <10ms response times
"""

import asyncio
import time
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List, Tuple
from collections import OrderedDict
import structlog
from cachetools import LFUCache, TTLCache
import psutil
import threading

from ..models.cache_models import (
    CacheEntry,
    CacheKeyType,
    CacheRequest,
    CacheResponse,
    CacheMetrics,
    SessionContext,
    AccessPattern
)

logger = structlog.get_logger()


class L1CacheManager:
    """
    Ultra-fast L1 cache with intelligent eviction and session awareness

    Features:
    - <10ms response times for cached data
    - LFU + TTL hybrid eviction strategy
    - Session-aware caching with per-user quotas
    - Memory pressure management
    - Real-time metrics and monitoring
    """

    def __init__(
        self,
        max_size_mb: int = 512,
        default_ttl_seconds: int = 10,
        max_entries: int = 50000,
        memory_pressure_threshold: float = 0.85
    ):
        self.max_size_bytes = max_size_mb * 1024 * 1024
        self.default_ttl_seconds = default_ttl_seconds
        self.max_entries = max_entries
        self.memory_pressure_threshold = memory_pressure_threshold

        # Primary cache storage (key -> CacheEntry)
        self._cache: OrderedDict[str, CacheEntry] = OrderedDict()

        # Session-specific caches for locality
        self._session_caches: Dict[str, OrderedDict[str, CacheEntry]] = {}

        # Access patterns for ML training
        self._access_patterns: Dict[str, AccessPattern] = {}

        # Metrics tracking
        self._metrics = CacheMetrics()
        self._response_times: List[float] = []

        # Memory management
        self._current_size_bytes = 0
        self._lock = threading.RLock()

        # Session contexts
        self._session_contexts: Dict[str, SessionContext] = {}

        # Background cleanup task
        self._cleanup_task: Optional[asyncio.Task] = None

        logger.info(
            "l1_cache_initialized",
            max_size_mb=max_size_mb,
            default_ttl=default_ttl_seconds,
            max_entries=max_entries
        )

    async def get(
        self,
        key: str,
        session_id: Optional[str] = None,
        user_id: Optional[str] = None
    ) -> CacheResponse:
        """
        Get data from L1 cache with <10ms response time
        """
        start_time = time.perf_counter()

        try:
            with self._lock:
                # Check session cache first for better locality
                entry = None
                from_prefetch = False

                if session_id and session_id in self._session_caches:
                    entry = self._session_caches[session_id].get(key)

                # Fallback to main cache
                if not entry:
                    entry = self._cache.get(key)

                if entry:
                    # Verify entry is still valid
                    if entry.is_expired():
                        self._remove_entry(key, session_id)
                        entry = None
                        self._metrics.expired_count += 1
                    elif not entry.is_valid():
                        self._remove_entry(key, session_id)
                        entry = None
                        logger.warning("cache_integrity_violation", key=key)

                # Calculate response time
                response_time_ms = (time.perf_counter() - start_time) * 1000
                self._response_times.append(response_time_ms)

                # Keep only last 1000 response times for metrics
                if len(self._response_times) > 1000:
                    self._response_times = self._response_times[-1000:]

                if entry:
                    # Cache hit
                    entry.access(session_id, user_id)
                    self._update_access_pattern(key, entry.key_type, session_id, user_id)
                    self._metrics.hit_count += 1

                    # Move to end (LRU aspect)
                    if key in self._cache:
                        self._cache.move_to_end(key)

                    logger.debug(
                        "l1_cache_hit",
                        key=key,
                        response_time_ms=response_time_ms,
                        session_id=session_id
                    )

                    return CacheResponse(
                        key=key,
                        data=entry.data,
                        hit=True,
                        response_time_ms=response_time_ms,
                        from_prefetch=from_prefetch,
                        expires_at=entry.expires_at,
                        cache_level="L1"
                    )
                else:
                    # Cache miss
                    self._metrics.miss_count += 1

                    logger.debug(
                        "l1_cache_miss",
                        key=key,
                        response_time_ms=response_time_ms,
                        session_id=session_id
                    )

                    return CacheResponse(
                        key=key,
                        data=None,
                        hit=False,
                        response_time_ms=response_time_ms,
                        cache_level="miss"
                    )

        except Exception as e:
            response_time_ms = (time.perf_counter() - start_time) * 1000
            logger.error(
                "l1_cache_get_error",
                key=key,
                error=str(e),
                response_time_ms=response_time_ms
            )

            return CacheResponse(
                key=key,
                data=None,
                hit=False,
                response_time_ms=response_time_ms,
                cache_level="error"
            )

    async def put(
        self,
        request: CacheRequest
    ) -> bool:
        """
        Store data in L1 cache with intelligent eviction
        """
        start_time = time.perf_counter()

        try:
            with self._lock:
                # Create cache entry
                entry = CacheEntry(
                    key=request.key,
                    key_type=request.key_type,
                    data=request.data,
                    ttl_seconds=request.ttl_seconds or self.default_ttl_seconds,
                    session_id=request.session_id,
                    user_id=request.user_id,
                    source_system=request.source_system
                )

                # Check if we need to make space
                if not self._has_capacity(entry.size_bytes):
                    evicted = self._make_space(entry.size_bytes)
                    if not evicted:
                        logger.warning(
                            "l1_cache_full",
                            key=request.key,
                            entry_size=entry.size_bytes
                        )
                        return False

                # Remove existing entry if present
                if request.key in self._cache:
                    old_entry = self._cache[request.key]
                    self._current_size_bytes -= old_entry.size_bytes

                # Add to main cache
                self._cache[request.key] = entry
                self._current_size_bytes += entry.size_bytes

                # Add to session cache if applicable
                if request.session_id:
                    if request.session_id not in self._session_caches:
                        self._session_caches[request.session_id] = OrderedDict()

                    session_cache = self._session_caches[request.session_id]
                    session_cache[request.key] = entry

                    # Limit session cache size
                    while len(session_cache) > 1000:  # Max 1000 entries per session
                        oldest_key = next(iter(session_cache))
                        del session_cache[oldest_key]

                # Update access pattern
                self._update_access_pattern(
                    request.key,
                    request.key_type,
                    request.session_id,
                    request.user_id
                )

                # Update metrics
                self._metrics.total_entries = len(self._cache)
                self._metrics.total_size_bytes = self._current_size_bytes

                processing_time_ms = (time.perf_counter() - start_time) * 1000

                logger.debug(
                    "l1_cache_stored",
                    key=request.key,
                    size_bytes=entry.size_bytes,
                    ttl_seconds=entry.ttl_seconds,
                    processing_time_ms=processing_time_ms
                )

                return True

        except Exception as e:
            logger.error(
                "l1_cache_put_error",
                key=request.key,
                error=str(e)
            )
            return False

    async def invalidate(
        self,
        key: str,
        session_id: Optional[str] = None
    ) -> bool:
        """
        Remove entry from cache
        """
        try:
            with self._lock:
                removed = self._remove_entry(key, session_id)

                if removed:
                    logger.debug("l1_cache_invalidated", key=key, session_id=session_id)

                return removed

        except Exception as e:
            logger.error(
                "l1_cache_invalidate_error",
                key=key,
                error=str(e)
            )
            return False

    async def invalidate_session(self, session_id: str) -> int:
        """
        Invalidate all entries for a session
        """
        try:
            with self._lock:
                count = 0

                # Remove from session cache
                if session_id in self._session_caches:
                    session_cache = self._session_caches[session_id]
                    for key in list(session_cache.keys()):
                        if self._remove_entry(key, session_id):
                            count += 1
                    del self._session_caches[session_id]

                # Remove session context
                if session_id in self._session_contexts:
                    del self._session_contexts[session_id]

                logger.info(
                    "l1_cache_session_invalidated",
                    session_id=session_id,
                    entries_removed=count
                )

                return count

        except Exception as e:
            logger.error(
                "l1_cache_session_invalidate_error",
                session_id=session_id,
                error=str(e)
            )
            return 0

    async def get_metrics(self) -> CacheMetrics:
        """
        Get current cache performance metrics
        """
        with self._lock:
            # Update response time metrics
            if self._response_times:
                self._response_times.sort()
                self._metrics.average_response_time_ms = sum(self._response_times) / len(self._response_times)

                p95_idx = int(len(self._response_times) * 0.95)
                p99_idx = int(len(self._response_times) * 0.99)
                self._metrics.p95_response_time_ms = self._response_times[p95_idx]
                self._metrics.p99_response_time_ms = self._response_times[p99_idx]

            # Update memory utilization
            self._metrics.memory_utilization = self._current_size_bytes / self.max_size_bytes

            # Update current totals
            self._metrics.total_entries = len(self._cache)
            self._metrics.total_size_bytes = self._current_size_bytes
            self._metrics.timestamp = datetime.utcnow()

            return self._metrics.copy(deep=True)

    def _has_capacity(self, additional_bytes: int) -> bool:
        """Check if cache has capacity for additional data"""
        return (
            len(self._cache) < self.max_entries and
            (self._current_size_bytes + additional_bytes) <= self.max_size_bytes and
            psutil.virtual_memory().percent < (self.memory_pressure_threshold * 100)
        )

    def _make_space(self, required_bytes: int) -> bool:
        """
        Make space by evicting entries using LFU + TTL strategy
        """
        initial_size = self._current_size_bytes
        target_size = self.max_size_bytes * 0.8  # Target 80% utilization after cleanup

        # First, remove expired entries
        self._remove_expired_entries()

        # If still need space, use LFU eviction
        if self._current_size_bytes + required_bytes > self.max_size_bytes:
            # Sort by access count (LFU) and last access time
            entries_by_priority = sorted(
                self._cache.items(),
                key=lambda x: (x[1].access_count, x[1].last_accessed)
            )

            for key, entry in entries_by_priority:
                if self._current_size_bytes + required_bytes <= target_size:
                    break

                self._remove_entry(key)
                self._metrics.eviction_count += 1

        freed_bytes = initial_size - self._current_size_bytes

        logger.debug(
            "l1_cache_space_made",
            freed_bytes=freed_bytes,
            required_bytes=required_bytes,
            current_utilization=self._current_size_bytes / self.max_size_bytes
        )

        return self._current_size_bytes + required_bytes <= self.max_size_bytes

    def _remove_expired_entries(self):
        """Remove all expired entries"""
        current_time = datetime.utcnow()
        expired_keys = []

        for key, entry in self._cache.items():
            if entry.is_expired():
                expired_keys.append(key)

        for key in expired_keys:
            self._remove_entry(key)
            self._metrics.expired_count += 1

    def _remove_entry(self, key: str, session_id: Optional[str] = None) -> bool:
        """Remove entry from cache and update size tracking"""
        removed = False

        # Remove from main cache
        if key in self._cache:
            entry = self._cache[key]
            self._current_size_bytes -= entry.size_bytes
            del self._cache[key]
            removed = True

        # Remove from session cache
        if session_id and session_id in self._session_caches:
            session_cache = self._session_caches[session_id]
            if key in session_cache:
                del session_cache[key]

        return removed

    def _update_access_pattern(
        self,
        key: str,
        key_type: CacheKeyType,
        session_id: Optional[str],
        user_id: Optional[str]
    ):
        """Update access patterns for ML prediction"""
        if key not in self._access_patterns:
            self._access_patterns[key] = AccessPattern(
                key=key,
                key_type=key_type,
                last_accessed=datetime.utcnow()
            )

        pattern = self._access_patterns[key]
        pattern.update_access(session_id, user_id)

        # Update session context
        if session_id:
            if session_id not in self._session_contexts:
                self._session_contexts[session_id] = SessionContext(
                    session_id=session_id,
                    user_id=user_id or "unknown",
                    workflow_type="unknown"
                )

            context = self._session_contexts[session_id]
            context.update_activity(key)

    async def start_background_tasks(self):
        """Start background maintenance tasks"""
        if self._cleanup_task is None:
            self._cleanup_task = asyncio.create_task(self._background_cleanup())

    async def stop_background_tasks(self):
        """Stop background maintenance tasks"""
        if self._cleanup_task:
            self._cleanup_task.cancel()
            try:
                await self._cleanup_task
            except asyncio.CancelledError:
                pass
            self._cleanup_task = None

    async def _background_cleanup(self):
        """Background task for cache maintenance"""
        while True:
            try:
                # Clean up expired entries every 30 seconds
                await asyncio.sleep(30)

                with self._lock:
                    self._remove_expired_entries()

                    # Clean up inactive sessions (older than 1 hour)
                    inactive_sessions = []
                    for session_id, context in self._session_contexts.items():
                        if not context.is_active(inactive_threshold_minutes=60):
                            inactive_sessions.append(session_id)

                    for session_id in inactive_sessions:
                        await self.invalidate_session(session_id)

                logger.debug(
                    "l1_cache_background_cleanup",
                    total_entries=len(self._cache),
                    size_mb=self._current_size_bytes / (1024 * 1024),
                    active_sessions=len(self._session_contexts)
                )

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("l1_cache_cleanup_error", error=str(e))
                await asyncio.sleep(60)  # Wait longer on error

    def get_access_patterns(self) -> Dict[str, AccessPattern]:
        """Get access patterns for ML training"""
        return self._access_patterns.copy()

    def get_session_contexts(self) -> Dict[str, SessionContext]:
        """Get session contexts for prediction"""
        return self._session_contexts.copy()