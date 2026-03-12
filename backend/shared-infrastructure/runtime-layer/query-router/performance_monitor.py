"""
Performance Monitor for Multi-KB Query Router
Comprehensive metrics collection and analysis for query routing optimization
Tracks latency, throughput, error rates, and data source performance
"""

import asyncio
import time
from typing import Dict, Optional, Any, List
from datetime import datetime, timedelta
from collections import defaultdict, deque
from dataclasses import dataclass, field
from loguru import logger

from .multi_kb_query_router import MultiKBQueryRequest, MultiKBQueryResponse, QueryPattern, DataSource


@dataclass
class QueryMetrics:
    """Metrics for individual query execution"""
    request_id: str
    pattern: str
    kb_id: Optional[str]
    sources_used: List[str]
    latency_ms: float
    cache_status: str
    timestamp: datetime
    success: bool
    error_type: Optional[str] = None


@dataclass
class PerformanceStats:
    """Aggregated performance statistics"""
    total_queries: int = 0
    successful_queries: int = 0
    failed_queries: int = 0
    average_latency_ms: float = 0.0
    p50_latency_ms: float = 0.0
    p95_latency_ms: float = 0.0
    p99_latency_ms: float = 0.0
    cache_hit_rate: float = 0.0
    queries_per_second: float = 0.0
    error_rate: float = 0.0

    # Per-KB stats
    kb_query_counts: Dict[str, int] = field(default_factory=dict)
    kb_avg_latencies: Dict[str, float] = field(default_factory=dict)
    kb_error_rates: Dict[str, float] = field(default_factory=dict)

    # Per-pattern stats
    pattern_query_counts: Dict[str, int] = field(default_factory=dict)
    pattern_avg_latencies: Dict[str, float] = field(default_factory=dict)
    pattern_success_rates: Dict[str, float] = field(default_factory=dict)

    # Per-source stats
    source_usage_counts: Dict[str, int] = field(default_factory=dict)
    source_avg_latencies: Dict[str, float] = field(default_factory=dict)
    source_health_scores: Dict[str, float] = field(default_factory=dict)


class PerformanceMonitor:
    """
    Comprehensive performance monitoring for Multi-KB Query Router

    Features:
    - Real-time latency tracking with percentiles
    - Per-KB, per-pattern, and per-source metrics
    - Cache effectiveness analysis
    - Error rate monitoring with categorization
    - Resource utilization tracking
    - Performance alerting and anomaly detection
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.enabled = config.get('enabled', True)

        if not self.enabled:
            return

        # Metrics retention
        self.retention_hours = config.get('retention_hours', 24)
        self.max_metrics_count = config.get('max_metrics_count', 10000)

        # Performance thresholds
        self.slow_query_threshold_ms = config.get('slow_query_threshold_ms', 1000)
        self.error_rate_threshold = config.get('error_rate_threshold', 0.05)  # 5%
        self.cache_hit_rate_threshold = config.get('cache_hit_rate_threshold', 0.70)  # 70%

        # Metrics storage
        self.query_metrics: deque[QueryMetrics] = deque(maxlen=self.max_metrics_count)
        self.latency_samples: deque[float] = deque(maxlen=1000)  # For percentile calculations

        # Real-time counters
        self.counters = {
            'total_queries': 0,
            'successful_queries': 0,
            'failed_queries': 0,
            'cache_hits': 0,
            'slow_queries': 0
        }

        # Per-dimension tracking
        self.kb_metrics = defaultdict(lambda: {'count': 0, 'latency_sum': 0, 'errors': 0})
        self.pattern_metrics = defaultdict(lambda: {'count': 0, 'latency_sum': 0, 'errors': 0})
        self.source_metrics = defaultdict(lambda: {'count': 0, 'latency_sum': 0, 'errors': 0})

        # Time-based tracking
        self.hourly_stats = defaultdict(lambda: {'queries': 0, 'latency_sum': 0, 'errors': 0})

        # Alerting
        self.alert_conditions = {}
        self.last_alert_times = {}

        # Background tasks
        self._cleanup_task = None
        self._stats_calculation_task = None

        logger.info("Performance Monitor initialized")

    async def initialize(self):
        """Initialize background monitoring tasks"""
        if not self.enabled:
            return

        # Start background tasks
        self._cleanup_task = asyncio.create_task(self._cleanup_old_metrics())
        self._stats_calculation_task = asyncio.create_task(self._calculate_stats_periodically())

    async def record_query_start(self, request: MultiKBQueryRequest):
        """Record query start for timing"""
        if not self.enabled:
            return

        # Store start time for latency calculation
        request._start_time = time.time()

    async def record_query_complete(self, request: MultiKBQueryRequest, response: MultiKBQueryResponse):
        """Record completed query metrics"""
        if not self.enabled:
            return

        try:
            # Calculate latency
            latency_ms = response.latency_ms

            # Create metrics record
            metrics = QueryMetrics(
                request_id=request.request_id,
                pattern=request.pattern.value,
                kb_id=request.kb_id,
                sources_used=response.sources_used,
                latency_ms=latency_ms,
                cache_status=response.cache_status,
                timestamp=datetime.utcnow(),
                success=not ('error' in response.data and response.data['error']),
                error_type=response.data.get('error_type') if 'error' in response.data else None
            )

            # Store metrics
            self.query_metrics.append(metrics)
            self.latency_samples.append(latency_ms)

            # Update counters
            self.counters['total_queries'] += 1
            if metrics.success:
                self.counters['successful_queries'] += 1
            else:
                self.counters['failed_queries'] += 1

            if response.cache_status in ['hit', 'partial']:
                self.counters['cache_hits'] += 1

            if latency_ms > self.slow_query_threshold_ms:
                self.counters['slow_queries'] += 1

            # Update dimensional metrics
            await self._update_dimensional_metrics(metrics)

            # Check for performance alerts
            await self._check_alert_conditions(metrics)

        except Exception as e:
            logger.error(f"Failed to record query metrics: {e}")

    async def _update_dimensional_metrics(self, metrics: QueryMetrics):
        """Update per-KB, per-pattern, and per-source metrics"""

        # KB metrics
        if metrics.kb_id:
            kb_stat = self.kb_metrics[metrics.kb_id]
            kb_stat['count'] += 1
            kb_stat['latency_sum'] += metrics.latency_ms
            if not metrics.success:
                kb_stat['errors'] += 1

        # Pattern metrics
        pattern_stat = self.pattern_metrics[metrics.pattern]
        pattern_stat['count'] += 1
        pattern_stat['latency_sum'] += metrics.latency_ms
        if not metrics.success:
            pattern_stat['errors'] += 1

        # Source metrics
        for source in metrics.sources_used:
            source_stat = self.source_metrics[source]
            source_stat['count'] += 1
            source_stat['latency_sum'] += metrics.latency_ms
            if not metrics.success:
                source_stat['errors'] += 1

        # Hourly metrics
        hour_key = datetime.utcnow().strftime("%Y-%m-%d-%H")
        hourly_stat = self.hourly_stats[hour_key]
        hourly_stat['queries'] += 1
        hourly_stat['latency_sum'] += metrics.latency_ms
        if not metrics.success:
            hourly_stat['errors'] += 1

    async def _check_alert_conditions(self, metrics: QueryMetrics):
        """Check for performance alert conditions"""

        # High latency alert
        if metrics.latency_ms > self.slow_query_threshold_ms * 2:  # 2x threshold
            await self._trigger_alert(
                'high_latency',
                f"Query {metrics.request_id} took {metrics.latency_ms:.2f}ms (pattern: {metrics.pattern})"
            )

        # Check error rate (every 100 queries)
        if self.counters['total_queries'] % 100 == 0:
            error_rate = self.counters['failed_queries'] / self.counters['total_queries']
            if error_rate > self.error_rate_threshold:
                await self._trigger_alert(
                    'high_error_rate',
                    f"Error rate is {error_rate:.3f} (threshold: {self.error_rate_threshold})"
                )

        # Check cache hit rate (every 100 queries)
        if self.counters['total_queries'] % 100 == 0 and self.counters['total_queries'] > 0:
            cache_hit_rate = self.counters['cache_hits'] / self.counters['total_queries']
            if cache_hit_rate < self.cache_hit_rate_threshold:
                await self._trigger_alert(
                    'low_cache_hit_rate',
                    f"Cache hit rate is {cache_hit_rate:.3f} (threshold: {self.cache_hit_rate_threshold})"
                )

    async def _trigger_alert(self, alert_type: str, message: str):
        """Trigger performance alert with rate limiting"""
        current_time = time.time()
        last_alert = self.last_alert_times.get(alert_type, 0)

        # Rate limit alerts (minimum 5 minutes between same alert type)
        if current_time - last_alert > 300:  # 5 minutes
            logger.warning(f"PERFORMANCE ALERT [{alert_type}]: {message}")
            self.last_alert_times[alert_type] = current_time

            # Store alert condition
            self.alert_conditions[alert_type] = {
                'message': message,
                'timestamp': datetime.utcnow(),
                'count': self.alert_conditions.get(alert_type, {}).get('count', 0) + 1
            }

    async def get_metrics(self) -> PerformanceStats:
        """Get comprehensive performance statistics"""
        if not self.enabled:
            return PerformanceStats()

        try:
            stats = PerformanceStats()

            # Basic counters
            stats.total_queries = self.counters['total_queries']
            stats.successful_queries = self.counters['successful_queries']
            stats.failed_queries = self.counters['failed_queries']

            if stats.total_queries > 0:
                stats.error_rate = stats.failed_queries / stats.total_queries
                stats.cache_hit_rate = self.counters['cache_hits'] / stats.total_queries

                # Calculate latency percentiles
                if self.latency_samples:
                    sorted_latencies = sorted(self.latency_samples)
                    count = len(sorted_latencies)

                    stats.average_latency_ms = sum(sorted_latencies) / count
                    stats.p50_latency_ms = sorted_latencies[int(count * 0.5)]
                    stats.p95_latency_ms = sorted_latencies[int(count * 0.95)]
                    stats.p99_latency_ms = sorted_latencies[int(count * 0.99)]

                # Calculate queries per second (last hour)
                one_hour_ago = datetime.utcnow() - timedelta(hours=1)
                recent_queries = [
                    m for m in self.query_metrics
                    if m.timestamp >= one_hour_ago
                ]
                stats.queries_per_second = len(recent_queries) / 3600.0

            # KB-specific metrics
            for kb_id, kb_stat in self.kb_metrics.items():
                if kb_stat['count'] > 0:
                    stats.kb_query_counts[kb_id] = kb_stat['count']
                    stats.kb_avg_latencies[kb_id] = kb_stat['latency_sum'] / kb_stat['count']
                    stats.kb_error_rates[kb_id] = kb_stat['errors'] / kb_stat['count']

            # Pattern-specific metrics
            for pattern, pattern_stat in self.pattern_metrics.items():
                if pattern_stat['count'] > 0:
                    stats.pattern_query_counts[pattern] = pattern_stat['count']
                    stats.pattern_avg_latencies[pattern] = pattern_stat['latency_sum'] / pattern_stat['count']
                    stats.pattern_success_rates[pattern] = 1.0 - (pattern_stat['errors'] / pattern_stat['count'])

            # Source-specific metrics
            for source, source_stat in self.source_metrics.items():
                if source_stat['count'] > 0:
                    stats.source_usage_counts[source] = source_stat['count']
                    stats.source_avg_latencies[source] = source_stat['latency_sum'] / source_stat['count']

                    # Calculate health score (1.0 = perfect, 0.0 = all failures)
                    success_rate = 1.0 - (source_stat['errors'] / source_stat['count'])
                    avg_latency = source_stat['latency_sum'] / source_stat['count']

                    # Health score considers both success rate and latency
                    latency_factor = max(0, 1.0 - (avg_latency / 2000))  # Penalty for >2s latency
                    stats.source_health_scores[source] = success_rate * latency_factor

            return stats

        except Exception as e:
            logger.error(f"Failed to calculate metrics: {e}")
            return PerformanceStats()

    async def get_real_time_stats(self) -> Dict[str, Any]:
        """Get real-time performance statistics"""
        if not self.enabled:
            return {}

        # Last 5 minutes of metrics
        five_min_ago = datetime.utcnow() - timedelta(minutes=5)
        recent_metrics = [m for m in self.query_metrics if m.timestamp >= five_min_ago]

        if not recent_metrics:
            return {
                'queries_last_5min': 0,
                'avg_latency_5min': 0,
                'error_rate_5min': 0
            }

        total = len(recent_metrics)
        errors = sum(1 for m in recent_metrics if not m.success)
        avg_latency = sum(m.latency_ms for m in recent_metrics) / total

        return {
            'queries_last_5min': total,
            'avg_latency_5min': avg_latency,
            'error_rate_5min': errors / total if total > 0 else 0,
            'queries_per_minute': total / 5.0,
            'slow_queries_5min': sum(1 for m in recent_metrics if m.latency_ms > self.slow_query_threshold_ms)
        }

    async def get_kb_performance_comparison(self) -> Dict[str, Any]:
        """Get performance comparison across Knowledge Bases"""
        if not self.enabled:
            return {}

        kb_comparison = {}

        for kb_id, kb_stat in self.kb_metrics.items():
            if kb_stat['count'] > 0:
                avg_latency = kb_stat['latency_sum'] / kb_stat['count']
                error_rate = kb_stat['errors'] / kb_stat['count']

                kb_comparison[kb_id] = {
                    'query_count': kb_stat['count'],
                    'avg_latency_ms': avg_latency,
                    'error_rate': error_rate,
                    'performance_score': self._calculate_performance_score(avg_latency, error_rate)
                }

        return kb_comparison

    def _calculate_performance_score(self, avg_latency: float, error_rate: float) -> float:
        """Calculate performance score (0-100) based on latency and error rate"""
        # Latency score (100 for <100ms, 0 for >2000ms)
        latency_score = max(0, 100 - (avg_latency / 20))

        # Error score (100 for 0% errors, 0 for >10% errors)
        error_score = max(0, 100 - (error_rate * 1000))

        # Combined score with weights
        return (latency_score * 0.7) + (error_score * 0.3)

    async def get_slow_queries(self, limit: int = 10) -> List[Dict[str, Any]]:
        """Get slowest queries from recent history"""
        if not self.enabled:
            return []

        # Get queries from last hour, sorted by latency
        one_hour_ago = datetime.utcnow() - timedelta(hours=1)
        recent_queries = [
            m for m in self.query_metrics
            if m.timestamp >= one_hour_ago
        ]

        slow_queries = sorted(recent_queries, key=lambda x: x.latency_ms, reverse=True)[:limit]

        return [
            {
                'request_id': q.request_id,
                'pattern': q.pattern,
                'kb_id': q.kb_id,
                'latency_ms': q.latency_ms,
                'sources_used': q.sources_used,
                'timestamp': q.timestamp.isoformat(),
                'cache_status': q.cache_status
            }
            for q in slow_queries
        ]

    async def get_error_analysis(self) -> Dict[str, Any]:
        """Get detailed error analysis"""
        if not self.enabled:
            return {}

        # Get errors from last hour
        one_hour_ago = datetime.utcnow() - timedelta(hours=1)
        recent_errors = [
            m for m in self.query_metrics
            if m.timestamp >= one_hour_ago and not m.success
        ]

        error_analysis = {
            'total_errors': len(recent_errors),
            'error_by_type': defaultdict(int),
            'error_by_pattern': defaultdict(int),
            'error_by_kb': defaultdict(int),
            'error_by_source': defaultdict(int)
        }

        for error in recent_errors:
            if error.error_type:
                error_analysis['error_by_type'][error.error_type] += 1
            error_analysis['error_by_pattern'][error.pattern] += 1
            if error.kb_id:
                error_analysis['error_by_kb'][error.kb_id] += 1
            for source in error.sources_used:
                error_analysis['error_by_source'][source] += 1

        return dict(error_analysis)

    async def get_alert_status(self) -> Dict[str, Any]:
        """Get current alert status"""
        return {
            'active_alerts': len(self.alert_conditions),
            'alert_conditions': dict(self.alert_conditions),
            'last_alert_times': dict(self.last_alert_times)
        }

    async def _cleanup_old_metrics(self):
        """Background task to clean up old metrics"""
        while True:
            try:
                cutoff_time = datetime.utcnow() - timedelta(hours=self.retention_hours)

                # Remove old metrics
                old_count = len(self.query_metrics)
                self.query_metrics = deque(
                    (m for m in self.query_metrics if m.timestamp >= cutoff_time),
                    maxlen=self.max_metrics_count
                )

                removed_count = old_count - len(self.query_metrics)
                if removed_count > 0:
                    logger.debug(f"Cleaned up {removed_count} old metrics")

                # Clean old hourly stats
                current_hour = datetime.utcnow().strftime("%Y-%m-%d-%H")
                cutoff_hours = [
                    (datetime.utcnow() - timedelta(hours=i)).strftime("%Y-%m-%d-%H")
                    for i in range(self.retention_hours + 1, 48)  # Keep extra hours for safety
                ]

                for hour_key in cutoff_hours:
                    self.hourly_stats.pop(hour_key, None)

                await asyncio.sleep(3600)  # Run every hour

            except Exception as e:
                logger.error(f"Metrics cleanup failed: {e}")
                await asyncio.sleep(3600)

    async def _calculate_stats_periodically(self):
        """Background task to calculate and log periodic statistics"""
        while True:
            try:
                await asyncio.sleep(300)  # Every 5 minutes

                stats = await self.get_metrics()
                real_time = await self.get_real_time_stats()

                logger.info(
                    f"Performance Stats: "
                    f"Queries/5min: {real_time.get('queries_last_5min', 0)}, "
                    f"Avg Latency: {real_time.get('avg_latency_5min', 0):.1f}ms, "
                    f"Error Rate: {real_time.get('error_rate_5min', 0):.3f}, "
                    f"Cache Hit Rate: {stats.cache_hit_rate:.3f}"
                )

            except Exception as e:
                logger.error(f"Periodic stats calculation failed: {e}")

    async def shutdown(self):
        """Shutdown performance monitor"""
        if self._cleanup_task:
            self._cleanup_task.cancel()
        if self._stats_calculation_task:
            self._stats_calculation_task.cancel()

        logger.info("Performance Monitor shutdown completed")