"""
Enhanced Prefetch Manager
Orchestrates intelligent data prefetching with ML predictions
"""

import asyncio
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Set
from collections import defaultdict
import structlog
import httpx
from concurrent.futures import ThreadPoolExecutor

from ..models.cache_models import (
    PrefetchRequest,
    PrefetchResponse,
    PrefetchPrediction,
    CacheRequest,
    CacheKeyType,
    SessionContext
)
from ..ml.prefetch_predictor import PrefetchPredictor
from .l1_cache_manager import L1CacheManager

logger = structlog.get_logger()


class DataSource:
    """Configuration for external data sources"""
    def __init__(self, name: str, base_url: str, timeout_ms: int = 500):
        self.name = name
        self.base_url = base_url
        self.timeout_ms = timeout_ms


class PrefetchManager:
    """
    Enhanced prefetch manager with ML-based prediction and intelligent orchestration

    Features:
    - ML-powered access prediction
    - Multi-source data fetching
    - Adaptive prefetch scheduling
    - Resource-aware execution
    - Performance monitoring
    """

    def __init__(
        self,
        l1_cache: L1CacheManager,
        predictor: PrefetchPredictor,
        max_concurrent_fetches: int = 20,
        prefetch_budget_mb: int = 100,
        min_confidence_threshold: float = 0.6
    ):
        self.l1_cache = l1_cache
        self.predictor = predictor
        self.max_concurrent_fetches = max_concurrent_fetches
        self.prefetch_budget_bytes = prefetch_budget_mb * 1024 * 1024
        self.min_confidence_threshold = min_confidence_threshold

        # Data sources configuration
        self._data_sources: Dict[str, DataSource] = {}

        # Prefetch scheduling
        self._prefetch_queue: asyncio.Queue = asyncio.Queue()
        self._active_prefetches: Dict[str, asyncio.Task] = {}

        # Performance tracking
        self._prefetch_metrics = {
            'total_requests': 0,
            'successful_prefetches': 0,
            'failed_prefetches': 0,
            'cache_hits_from_prefetch': 0,
            'total_bytes_prefetched': 0,
            'average_fetch_time_ms': 0.0
        }

        # Resource management
        self._current_prefetch_bytes = 0
        self._fetch_times: List[float] = []

        # HTTP client for external data fetching
        self._http_client: Optional[httpx.AsyncClient] = None

        # Background tasks
        self._scheduler_task: Optional[asyncio.Task] = None
        self._worker_tasks: List[asyncio.Task] = []

        # Thread pool for CPU-intensive operations
        self._executor = ThreadPoolExecutor(max_workers=4)

        logger.info(
            "prefetch_manager_initialized",
            max_concurrent=max_concurrent_fetches,
            budget_mb=prefetch_budget_mb,
            min_confidence=min_confidence_threshold
        )

    def add_data_source(self, source: DataSource):
        """Add a data source for prefetching"""
        self._data_sources[source.name] = source
        logger.info("data_source_added", name=source.name, base_url=source.base_url)

    async def prefetch_predictions(
        self,
        session_id: Optional[str] = None,
        user_id: Optional[str] = None,
        max_items: int = 50,
        confidence_threshold: Optional[float] = None
    ) -> PrefetchResponse:
        """
        Execute prefetch based on ML predictions
        """
        start_time = time.perf_counter()

        try:
            # Get current access patterns and session contexts
            access_patterns = self.l1_cache.get_access_patterns()
            session_contexts = self.l1_cache.get_session_contexts()

            # Generate predictions
            predictions = await self.predictor.predict_prefetch_candidates(
                access_patterns=access_patterns,
                session_contexts=session_contexts,
                current_session_id=session_id,
                max_candidates=max_items
            )

            # Filter by confidence threshold
            threshold = confidence_threshold or self.min_confidence_threshold
            filtered_predictions = [
                pred for pred in predictions
                if pred.confidence >= threshold
            ]

            # Execute prefetch operations
            prefetched_keys = []
            skipped_keys = []
            total_size_bytes = 0

            for prediction in filtered_predictions:
                # Check resource limits
                if (self._current_prefetch_bytes + 1024 * 1024) > self.prefetch_budget_bytes:  # Conservative estimate
                    skipped_keys.append(prediction.key)
                    continue

                # Attempt prefetch
                success, size_bytes = await self._prefetch_key(
                    prediction.key,
                    prediction.key_type,
                    session_id,
                    user_id
                )

                if success:
                    prefetched_keys.append(prediction.key)
                    total_size_bytes += size_bytes
                    self._current_prefetch_bytes += size_bytes
                else:
                    skipped_keys.append(prediction.key)

            processing_time_ms = (time.perf_counter() - start_time) * 1000

            # Update metrics
            self._prefetch_metrics['total_requests'] += 1
            self._prefetch_metrics['successful_prefetches'] += len(prefetched_keys)
            self._prefetch_metrics['failed_prefetches'] += len(skipped_keys)
            self._prefetch_metrics['total_bytes_prefetched'] += total_size_bytes

            response = PrefetchResponse(
                requested_keys=[p.key for p in filtered_predictions],
                prefetched_keys=prefetched_keys,
                skipped_keys=skipped_keys,
                total_prefetched=len(prefetched_keys),
                total_size_mb=total_size_bytes / (1024 * 1024),
                processing_time_ms=processing_time_ms,
                predictions_used=len(filtered_predictions)
            )

            logger.info(
                "prefetch_predictions_executed",
                total_predictions=len(predictions),
                filtered_predictions=len(filtered_predictions),
                successful_prefetches=len(prefetched_keys),
                processing_time_ms=processing_time_ms,
                session_id=session_id
            )

            return response

        except Exception as e:
            processing_time_ms = (time.perf_counter() - start_time) * 1000
            logger.error(
                "prefetch_predictions_error",
                error=str(e),
                processing_time_ms=processing_time_ms,
                session_id=session_id
            )

            return PrefetchResponse(
                requested_keys=[],
                prefetched_keys=[],
                skipped_keys=[],
                total_prefetched=0,
                total_size_mb=0.0,
                processing_time_ms=processing_time_ms,
                predictions_used=0
            )

    async def prefetch_explicit(self, request: PrefetchRequest) -> PrefetchResponse:
        """
        Execute explicit prefetch request for specific keys
        """
        start_time = time.perf_counter()

        try:
            prefetched_keys = []
            skipped_keys = []
            total_size_bytes = 0

            # Process each requested key
            for key in request.keys:
                # Check resource limits
                if (self._current_prefetch_bytes + 1024 * 1024) > request.prefetch_budget_mb * 1024 * 1024:
                    skipped_keys.extend(request.keys[len(prefetched_keys):])
                    break

                # Determine key type (heuristic-based)
                key_type = self._infer_key_type(key)

                # Attempt prefetch
                success, size_bytes = await self._prefetch_key(
                    key,
                    key_type,
                    request.session_context.session_id if request.session_context else None,
                    request.session_context.user_id if request.session_context else None
                )

                if success:
                    prefetched_keys.append(key)
                    total_size_bytes += size_bytes
                    self._current_prefetch_bytes += size_bytes
                else:
                    skipped_keys.append(key)

            processing_time_ms = (time.perf_counter() - start_time) * 1000

            response = PrefetchResponse(
                requested_keys=request.keys,
                prefetched_keys=prefetched_keys,
                skipped_keys=skipped_keys,
                total_prefetched=len(prefetched_keys),
                total_size_mb=total_size_bytes / (1024 * 1024),
                processing_time_ms=processing_time_ms,
                predictions_used=0
            )

            logger.info(
                "prefetch_explicit_executed",
                requested_keys=len(request.keys),
                successful_prefetches=len(prefetched_keys),
                processing_time_ms=processing_time_ms
            )

            return response

        except Exception as e:
            processing_time_ms = (time.perf_counter() - start_time) * 1000
            logger.error(
                "prefetch_explicit_error",
                error=str(e),
                processing_time_ms=processing_time_ms
            )

            return PrefetchResponse(
                requested_keys=request.keys,
                prefetched_keys=[],
                skipped_keys=request.keys,
                total_prefetched=0,
                total_size_mb=0.0,
                processing_time_ms=processing_time_ms,
                predictions_used=0
            )

    async def _prefetch_key(
        self,
        key: str,
        key_type: CacheKeyType,
        session_id: Optional[str] = None,
        user_id: Optional[str] = None
    ) -> tuple[bool, int]:
        """
        Prefetch data for a specific key
        """
        try:
            # Check if already cached
            cached_response = await self.l1_cache.get(key, session_id, user_id)
            if cached_response.hit:
                logger.debug("prefetch_already_cached", key=key)
                return True, 0

            # Determine data source and fetch data
            data_source = self._determine_data_source(key, key_type)
            if not data_source:
                logger.warning("prefetch_no_data_source", key=key, key_type=key_type)
                return False, 0

            # Fetch data from external source
            data = await self._fetch_external_data(data_source, key)
            if not data:
                return False, 0

            # Store in L1 cache
            cache_request = CacheRequest(
                key=key,
                key_type=key_type,
                data=data,
                ttl_seconds=self._get_ttl_for_key_type(key_type),
                session_id=session_id,
                user_id=user_id,
                source_system=data_source.name
            )

            success = await self.l1_cache.put(cache_request)
            size_bytes = len(str(data).encode('utf-8'))

            if success:
                logger.debug(
                    "prefetch_successful",
                    key=key,
                    size_bytes=size_bytes,
                    data_source=data_source.name
                )
                return True, size_bytes
            else:
                return False, 0

        except Exception as e:
            logger.error("prefetch_key_error", key=key, error=str(e))
            return False, 0

    def _determine_data_source(self, key: str, key_type: CacheKeyType) -> Optional[DataSource]:
        """
        Determine which data source to use for a key
        """
        # Heuristic-based data source mapping
        if key_type == CacheKeyType.PATIENT_CONTEXT:
            return self._data_sources.get("patient_service")
        elif key_type == CacheKeyType.CLINICAL_DATA:
            return self._data_sources.get("clinical_service")
        elif key_type == CacheKeyType.MEDICATION_DATA:
            return self._data_sources.get("medication_service")
        elif key_type == CacheKeyType.GUIDELINE_DATA:
            return self._data_sources.get("guideline_service")
        elif key_type == CacheKeyType.SEMANTIC_MESH:
            return self._data_sources.get("semantic_service")
        else:
            # Default to first available source
            return next(iter(self._data_sources.values())) if self._data_sources else None

    async def _fetch_external_data(self, data_source: DataSource, key: str) -> Optional[Dict[str, Any]]:
        """
        Fetch data from external data source
        """
        try:
            if not self._http_client:
                self._http_client = httpx.AsyncClient(
                    timeout=httpx.Timeout(timeout=data_source.timeout_ms / 1000.0)
                )

            # Construct URL based on key pattern
            url = f"{data_source.base_url}/api/data/{key}"

            fetch_start = time.perf_counter()
            response = await self._http_client.get(url)
            fetch_time_ms = (time.perf_counter() - fetch_start) * 1000

            # Track fetch times for metrics
            self._fetch_times.append(fetch_time_ms)
            if len(self._fetch_times) > 1000:
                self._fetch_times = self._fetch_times[-1000:]

            self._prefetch_metrics['average_fetch_time_ms'] = sum(self._fetch_times) / len(self._fetch_times)

            if response.status_code == 200:
                data = response.json()
                logger.debug(
                    "external_data_fetched",
                    key=key,
                    data_source=data_source.name,
                    fetch_time_ms=fetch_time_ms
                )
                return data
            else:
                logger.warning(
                    "external_data_fetch_failed",
                    key=key,
                    status_code=response.status_code,
                    data_source=data_source.name
                )
                return None

        except Exception as e:
            logger.error(
                "external_data_fetch_error",
                key=key,
                data_source=data_source.name,
                error=str(e)
            )
            return None

    def _infer_key_type(self, key: str) -> CacheKeyType:
        """
        Infer cache key type from key pattern
        """
        key_lower = key.lower()

        if "patient" in key_lower:
            return CacheKeyType.PATIENT_CONTEXT
        elif "medication" in key_lower or "drug" in key_lower:
            return CacheKeyType.MEDICATION_DATA
        elif "guideline" in key_lower or "rule" in key_lower:
            return CacheKeyType.GUIDELINE_DATA
        elif "semantic" in key_lower or "mesh" in key_lower:
            return CacheKeyType.SEMANTIC_MESH
        elif "workflow" in key_lower or "state" in key_lower:
            return CacheKeyType.WORKFLOW_STATE
        elif "session" in key_lower or "user" in key_lower:
            return CacheKeyType.USER_SESSION
        else:
            return CacheKeyType.CLINICAL_DATA

    def _get_ttl_for_key_type(self, key_type: CacheKeyType) -> int:
        """
        Get appropriate TTL based on key type
        """
        ttl_map = {
            CacheKeyType.PATIENT_CONTEXT: 30,      # 30 seconds - frequently changing
            CacheKeyType.CLINICAL_DATA: 20,        # 20 seconds - real-time data
            CacheKeyType.MEDICATION_DATA: 60,      # 1 minute - relatively stable
            CacheKeyType.GUIDELINE_DATA: 300,      # 5 minutes - stable reference data
            CacheKeyType.SEMANTIC_MESH: 120,       # 2 minutes - moderately stable
            CacheKeyType.WORKFLOW_STATE: 10,       # 10 seconds - highly dynamic
            CacheKeyType.USER_SESSION: 15,         # 15 seconds - session-specific
            CacheKeyType.EVIDENCE_ENVELOPE: 180    # 3 minutes - audit data
        }

        return ttl_map.get(key_type, 10)  # Default 10 seconds

    async def start_background_tasks(self):
        """Start background prefetch scheduling and workers"""
        # Start scheduler
        if not self._scheduler_task:
            self._scheduler_task = asyncio.create_task(self._background_scheduler())

        # Start worker tasks
        for i in range(min(4, self.max_concurrent_fetches)):
            worker_task = asyncio.create_task(self._prefetch_worker(f"worker-{i}"))
            self._worker_tasks.append(worker_task)

        logger.info("prefetch_background_tasks_started", workers=len(self._worker_tasks))

    async def stop_background_tasks(self):
        """Stop all background tasks"""
        # Stop scheduler
        if self._scheduler_task:
            self._scheduler_task.cancel()
            try:
                await self._scheduler_task
            except asyncio.CancelledError:
                pass

        # Stop workers
        for task in self._worker_tasks:
            task.cancel()

        if self._worker_tasks:
            await asyncio.gather(*self._worker_tasks, return_exceptions=True)

        self._worker_tasks.clear()

        # Close HTTP client
        if self._http_client:
            await self._http_client.aclose()

        logger.info("prefetch_background_tasks_stopped")

    async def _background_scheduler(self):
        """Background task for intelligent prefetch scheduling"""
        while True:
            try:
                # Run predictive prefetching every 30 seconds
                await asyncio.sleep(30)

                # Get active sessions
                session_contexts = self.l1_cache.get_session_contexts()

                for session_id, context in session_contexts.items():
                    if context.is_active():
                        # Schedule predictive prefetch for active session
                        try:
                            await self.prefetch_predictions(
                                session_id=session_id,
                                max_items=20,  # Conservative for background
                                confidence_threshold=0.75  # Higher threshold for background
                            )
                        except Exception as e:
                            logger.error(
                                "background_prefetch_error",
                                session_id=session_id,
                                error=str(e)
                            )

                # Clean up resource tracking
                self._current_prefetch_bytes *= 0.9  # Gradual decay for memory pressure

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("prefetch_scheduler_error", error=str(e))
                await asyncio.sleep(60)

    async def _prefetch_worker(self, worker_id: str):
        """Background worker for processing prefetch queue"""
        logger.debug("prefetch_worker_started", worker_id=worker_id)

        while True:
            try:
                # In a full implementation, this would process items from a queue
                # For now, just sleep to prevent busy waiting
                await asyncio.sleep(10)

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("prefetch_worker_error", worker_id=worker_id, error=str(e))
                await asyncio.sleep(30)

        logger.debug("prefetch_worker_stopped", worker_id=worker_id)

    def get_metrics(self) -> Dict[str, Any]:
        """Get prefetch performance metrics"""
        return {
            **self._prefetch_metrics,
            'current_prefetch_mb': self._current_prefetch_bytes / (1024 * 1024),
            'budget_utilization': self._current_prefetch_bytes / self.prefetch_budget_bytes,
            'active_data_sources': len(self._data_sources),
            'timestamp': datetime.utcnow().isoformat()
        }