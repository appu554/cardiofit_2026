"""
Google FHIR Service Layer for KB7 Terminology Hybrid Architecture

This module provides a service layer that integrates the Google FHIR Healthcare API
client with the KB7 hybrid query router, enabling seamless terminology operations
across PostgreSQL, GraphDB, and Google FHIR stores.
"""

import json
import asyncio
import logging
from typing import Dict, List, Any, Optional, Union, Tuple
from datetime import datetime, timedelta
from enum import Enum
import aioredis
from dataclasses import dataclass

from .google_config import GoogleFHIRConfig, load_google_fhir_config
from .google_fhir_terminology_client import (
    GoogleFHIRTerminologyClient,
    GoogleFHIRTerminologyError,
    create_google_fhir_client
)
from .client import FHIRTerminologyClient, QueryRouterUnavailableError
from .models import (
    CodeSystemLookupRequest, CodeSystemLookupResponse,
    ValueSetExpandRequest, ConceptMapTranslateRequest,
    ValidateCodeRequest, ValidateCodeResponse,
    OperationOutcome
)

logger = logging.getLogger(__name__)


class SyncDirection(str, Enum):
    """Data synchronization direction"""
    LOCAL_TO_GOOGLE = "local_to_google"
    GOOGLE_TO_LOCAL = "google_to_local"
    BIDIRECTIONAL = "bidirectional"


class FallbackStrategy(str, Enum):
    """Fallback strategy when primary service fails"""
    NONE = "none"
    LOCAL_ONLY = "local_only"
    GOOGLE_ONLY = "google_only"
    BEST_EFFORT = "best_effort"


@dataclass
class OperationResult:
    """Result of a terminology operation with metadata"""
    success: bool
    data: Optional[Dict[str, Any]] = None
    source: Optional[str] = None  # 'google', 'local', 'hybrid'
    latency_ms: Optional[int] = None
    cached: bool = False
    error: Optional[str] = None
    fallback_used: bool = False


@dataclass
class SyncResult:
    """Result of a synchronization operation"""
    success: bool
    direction: SyncDirection
    resources_synced: int = 0
    errors: List[str] = None
    duration_ms: int = 0

    def __post_init__(self):
        if self.errors is None:
            self.errors = []


class GoogleFHIRHybridService:
    """
    Hybrid service combining Google FHIR Healthcare API with KB7 query router.

    This service provides intelligent routing between local terminology stores
    (PostgreSQL/GraphDB) and Google FHIR, with automatic fallback and synchronization.
    """

    def __init__(self,
                 google_config: Optional[GoogleFHIRConfig] = None,
                 query_router_url: str = "http://localhost:8087",
                 redis_url: str = "redis://localhost:6379",
                 fallback_strategy: FallbackStrategy = FallbackStrategy.BEST_EFFORT,
                 enable_sync: bool = True,
                 sync_interval: int = 300):
        """
        Initialize hybrid Google FHIR service.

        Args:
            google_config: Google FHIR configuration
            query_router_url: KB7 query router endpoint
            redis_url: Redis URL for caching
            fallback_strategy: Strategy when primary service fails
            enable_sync: Whether to enable automatic synchronization
            sync_interval: Sync interval in seconds
        """
        self.google_config = google_config or load_google_fhir_config()
        self.query_router_url = query_router_url
        self.redis_url = redis_url
        self.fallback_strategy = fallback_strategy
        self.enable_sync = enable_sync
        self.sync_interval = sync_interval

        # Clients
        self._google_client: Optional[GoogleFHIRTerminologyClient] = None
        self._local_client: Optional[FHIRTerminologyClient] = None
        self._redis_client: Optional[aioredis.Redis] = None

        # Background tasks
        self._sync_task: Optional[asyncio.Task] = None
        self._running = False

        # Statistics
        self._stats = {
            "google_requests": 0,
            "local_requests": 0,
            "google_errors": 0,
            "local_errors": 0,
            "fallback_count": 0,
            "sync_count": 0,
            "cache_hits": 0
        }

        logger.info(
            "Initialized Google FHIR Hybrid Service",
            extra={
                "query_router_url": query_router_url,
                "fallback_strategy": fallback_strategy.value,
                "enable_sync": enable_sync
            }
        )

    async def __aenter__(self):
        """Async context manager entry."""
        await self._initialize()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self._cleanup()

    async def _initialize(self):
        """Initialize all clients and background tasks."""
        try:
            # Initialize Redis client
            self._redis_client = await aioredis.from_url(self.redis_url)

            # Initialize Google FHIR client
            self._google_client = await create_google_fhir_client(
                config=self.google_config,
                redis_url=self.redis_url
            )

            # Initialize local terminology client
            self._local_client = FHIRTerminologyClient(
                query_router_url=self.query_router_url,
                redis_url=self.redis_url
            )

            # Start background sync if enabled
            if self.enable_sync:
                self._running = True
                self._sync_task = asyncio.create_task(self._background_sync())

            logger.info("All clients initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize hybrid service: {e}")
            raise

    async def _cleanup(self):
        """Cleanup resources and stop background tasks."""
        self._running = False

        if self._sync_task:
            self._sync_task.cancel()
            try:
                await self._sync_task
            except asyncio.CancelledError:
                pass

        if self._google_client:
            await self._google_client._cleanup()

        if self._redis_client:
            await self._redis_client.close()

        logger.info("Hybrid service cleanup completed")

    async def _background_sync(self):
        """Background task for periodic synchronization."""
        while self._running:
            try:
                await asyncio.sleep(self.sync_interval)
                if self._running:
                    await self._sync_terminology_data()
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Background sync error: {e}")

    async def health_check(self) -> Dict[str, Any]:
        """
        Perform comprehensive health check.

        Returns:
            Dict: Health status of all components
        """
        results = {
            "status": "healthy",
            "timestamp": datetime.now().isoformat(),
            "components": {}
        }

        # Check Google FHIR
        try:
            google_health = await self._google_client.health_check()
            results["components"]["google_fhir"] = google_health
        except Exception as e:
            results["components"]["google_fhir"] = {
                "status": "unhealthy",
                "error": str(e)
            }
            results["status"] = "degraded"

        # Check local query router
        try:
            local_health = await self._local_client.health_check()
            results["components"]["local_router"] = local_health
        except Exception as e:
            results["components"]["local_router"] = {
                "status": "unhealthy",
                "error": str(e)
            }
            results["status"] = "degraded"

        # Check Redis
        try:
            await self._redis_client.ping()
            results["components"]["redis"] = {"status": "healthy"}
        except Exception as e:
            results["components"]["redis"] = {
                "status": "unhealthy",
                "error": str(e)
            }

        # Overall status
        unhealthy_count = sum(1 for comp in results["components"].values()
                            if comp.get("status") != "healthy")
        if unhealthy_count >= len(results["components"]):
            results["status"] = "unhealthy"

        # Add statistics
        results["statistics"] = self._stats.copy()

        return results

    async def lookup_code(self, request: CodeSystemLookupRequest,
                         prefer_source: Optional[str] = None) -> OperationResult:
        """
        Perform CodeSystem $lookup with intelligent routing.

        Args:
            request: CodeSystem lookup request
            prefer_source: Preferred source ('google' or 'local')

        Returns:
            OperationResult: Lookup results with metadata
        """
        start_time = datetime.now()

        # Determine optimal routing strategy
        use_google_first = self._should_use_google_first(request, prefer_source)

        if use_google_first:
            # Try Google FHIR first
            result = await self._try_google_lookup(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.GOOGLE_ONLY:
                result = await self._try_local_lookup(request)
                if result.success:
                    result.fallback_used = True
        else:
            # Try local first
            result = await self._try_local_lookup(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.LOCAL_ONLY:
                result = await self._try_google_lookup(request)
                if result.success:
                    result.fallback_used = True

        result.latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        return result

    async def expand_valueset(self, request: ValueSetExpandRequest,
                            prefer_source: Optional[str] = None) -> OperationResult:
        """
        Perform ValueSet $expand with intelligent routing.

        Args:
            request: ValueSet expansion request
            prefer_source: Preferred source ('google' or 'local')

        Returns:
            OperationResult: Expansion results with metadata
        """
        start_time = datetime.now()

        use_google_first = self._should_use_google_first(request, prefer_source)

        if use_google_first:
            result = await self._try_google_expand(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.GOOGLE_ONLY:
                result = await self._try_local_expand(request)
                if result.success:
                    result.fallback_used = True
        else:
            result = await self._try_local_expand(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.LOCAL_ONLY:
                result = await self._try_google_expand(request)
                if result.success:
                    result.fallback_used = True

        result.latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        return result

    async def translate_concept(self, request: ConceptMapTranslateRequest,
                              prefer_source: Optional[str] = None) -> OperationResult:
        """
        Perform ConceptMap $translate with intelligent routing.

        Args:
            request: ConceptMap translation request
            prefer_source: Preferred source ('google' or 'local')

        Returns:
            OperationResult: Translation results with metadata
        """
        start_time = datetime.now()

        use_google_first = self._should_use_google_first(request, prefer_source)

        if use_google_first:
            result = await self._try_google_translate(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.GOOGLE_ONLY:
                result = await self._try_local_translate(request)
                if result.success:
                    result.fallback_used = True
        else:
            result = await self._try_local_translate(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.LOCAL_ONLY:
                result = await self._try_google_translate(request)
                if result.success:
                    result.fallback_used = True

        result.latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        return result

    async def validate_code(self, request: ValidateCodeRequest,
                          prefer_source: Optional[str] = None) -> OperationResult:
        """
        Perform $validate-code with intelligent routing.

        Args:
            request: Code validation request
            prefer_source: Preferred source ('google' or 'local')

        Returns:
            OperationResult: Validation results with metadata
        """
        start_time = datetime.now()

        use_google_first = self._should_use_google_first(request, prefer_source)

        if use_google_first:
            result = await self._try_google_validate(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.GOOGLE_ONLY:
                result = await self._try_local_validate(request)
                if result.success:
                    result.fallback_used = True
        else:
            result = await self._try_local_validate(request)
            if not result.success and self.fallback_strategy != FallbackStrategy.LOCAL_ONLY:
                result = await self._try_google_validate(request)
                if result.success:
                    result.fallback_used = True

        result.latency_ms = int((datetime.now() - start_time).total_seconds() * 1000)
        return result

    def _should_use_google_first(self, request: Any, prefer_source: Optional[str]) -> bool:
        """
        Determine whether to try Google FHIR first based on request characteristics.

        Args:
            request: FHIR operation request
            prefer_source: User preference

        Returns:
            bool: True if Google should be tried first
        """
        if prefer_source == "google":
            return True
        elif prefer_source == "local":
            return False

        # Intelligent routing based on request characteristics
        # Use Google for official terminology (like SNOMED, LOINC)
        if hasattr(request, 'system_url') and request.system_url:
            official_systems = [
                "http://snomed.info/sct",
                "http://loinc.org",
                "http://unitsofmeasure.org",
                "http://hl7.org/fhir/sid/icd-10"
            ]
            if any(sys in request.system_url for sys in official_systems):
                return True

        # Use local for custom/local terminology
        return False

    async def _try_google_lookup(self, request: CodeSystemLookupRequest) -> OperationResult:
        """Try CodeSystem lookup via Google FHIR."""
        try:
            self._stats["google_requests"] += 1
            response = await self._google_client.lookup_code(request)
            return OperationResult(
                success=True,
                data=response.dict(),
                source="google"
            )
        except Exception as e:
            self._stats["google_errors"] += 1
            logger.warning(f"Google lookup failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="google"
            )

    async def _try_local_lookup(self, request: CodeSystemLookupRequest) -> OperationResult:
        """Try CodeSystem lookup via local query router."""
        try:
            self._stats["local_requests"] += 1
            response = await self._local_client.lookup_code(request)
            return OperationResult(
                success=True,
                data=response.dict(),
                source="local"
            )
        except Exception as e:
            self._stats["local_errors"] += 1
            logger.warning(f"Local lookup failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="local"
            )

    async def _try_google_expand(self, request: ValueSetExpandRequest) -> OperationResult:
        """Try ValueSet expansion via Google FHIR."""
        try:
            self._stats["google_requests"] += 1
            response = await self._google_client.expand_valueset(request)
            return OperationResult(
                success=True,
                data=response,
                source="google"
            )
        except Exception as e:
            self._stats["google_errors"] += 1
            logger.warning(f"Google expansion failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="google"
            )

    async def _try_local_expand(self, request: ValueSetExpandRequest) -> OperationResult:
        """Try ValueSet expansion via local query router."""
        try:
            self._stats["local_requests"] += 1
            response = await self._local_client.expand_valueset(request)
            return OperationResult(
                success=True,
                data=response.dict(),
                source="local"
            )
        except Exception as e:
            self._stats["local_errors"] += 1
            logger.warning(f"Local expansion failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="local"
            )

    async def _try_google_translate(self, request: ConceptMapTranslateRequest) -> OperationResult:
        """Try ConceptMap translation via Google FHIR."""
        try:
            self._stats["google_requests"] += 1
            response = await self._google_client.translate_concept(request)
            return OperationResult(
                success=True,
                data=response,
                source="google"
            )
        except Exception as e:
            self._stats["google_errors"] += 1
            logger.warning(f"Google translation failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="google"
            )

    async def _try_local_translate(self, request: ConceptMapTranslateRequest) -> OperationResult:
        """Try ConceptMap translation via local query router."""
        try:
            self._stats["local_requests"] += 1
            response = await self._local_client.translate_concept(request)
            return OperationResult(
                success=True,
                data=response.dict(),
                source="local"
            )
        except Exception as e:
            self._stats["local_errors"] += 1
            logger.warning(f"Local translation failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="local"
            )

    async def _try_google_validate(self, request: ValidateCodeRequest) -> OperationResult:
        """Try code validation via Google FHIR."""
        try:
            self._stats["google_requests"] += 1
            response = await self._google_client.validate_code(request)
            return OperationResult(
                success=True,
                data=response.dict(),
                source="google"
            )
        except Exception as e:
            self._stats["google_errors"] += 1
            logger.warning(f"Google validation failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="google"
            )

    async def _try_local_validate(self, request: ValidateCodeRequest) -> OperationResult:
        """Try code validation via local query router."""
        try:
            self._stats["local_requests"] += 1
            response = await self._local_client.validate_code(request)
            return OperationResult(
                success=True,
                data=response.dict(),
                source="local"
            )
        except Exception as e:
            self._stats["local_errors"] += 1
            logger.warning(f"Local validation failed: {e}")
            return OperationResult(
                success=False,
                error=str(e),
                source="local"
            )

    async def _sync_terminology_data(self) -> SyncResult:
        """
        Synchronize terminology data between local and Google FHIR.

        Returns:
            SyncResult: Synchronization results
        """
        start_time = datetime.now()

        try:
            # For now, implement basic sync logic
            # In a full implementation, this would:
            # 1. Compare last modified timestamps
            # 2. Identify differences
            # 3. Sync changed resources

            self._stats["sync_count"] += 1

            # Placeholder implementation
            await asyncio.sleep(0.1)  # Simulate sync work

            duration = int((datetime.now() - start_time).total_seconds() * 1000)

            return SyncResult(
                success=True,
                direction=SyncDirection.BIDIRECTIONAL,
                resources_synced=0,  # Would be actual count
                duration_ms=duration
            )

        except Exception as e:
            logger.error(f"Synchronization failed: {e}")
            duration = int((datetime.now() - start_time).total_seconds() * 1000)

            return SyncResult(
                success=False,
                direction=SyncDirection.BIDIRECTIONAL,
                errors=[str(e)],
                duration_ms=duration
            )

    async def get_statistics(self) -> Dict[str, Any]:
        """
        Get comprehensive service statistics.

        Returns:
            Dict: Service statistics and metrics
        """
        # Get Google client stats
        google_stats = {}
        if self._google_client:
            google_stats = await self._google_client.get_statistics()

        # Get local client stats
        local_stats = {}
        if self._local_client:
            local_stats = await self._local_client.get_statistics()

        return {
            "hybrid_service": self._stats.copy(),
            "google_fhir": google_stats,
            "local_router": local_stats,
            "configuration": {
                "fallback_strategy": self.fallback_strategy.value,
                "sync_enabled": self.enable_sync,
                "sync_interval": self.sync_interval
            },
            "timestamp": datetime.now().isoformat()
        }


async def create_hybrid_service(
    google_config: Optional[GoogleFHIRConfig] = None,
    query_router_url: str = "http://localhost:8087",
    redis_url: str = "redis://localhost:6379",
    **kwargs
) -> GoogleFHIRHybridService:
    """
    Factory function to create and initialize hybrid service.

    Args:
        google_config: Google FHIR configuration
        query_router_url: Local query router URL
        redis_url: Redis URL for caching
        **kwargs: Additional service configuration

    Returns:
        GoogleFHIRHybridService: Initialized hybrid service
    """
    service = GoogleFHIRHybridService(
        google_config=google_config,
        query_router_url=query_router_url,
        redis_url=redis_url,
        **kwargs
    )

    await service._initialize()
    return service