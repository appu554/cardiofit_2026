"""
Metrics Collector
Collects performance metrics from all runtime layer services
"""

import asyncio
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
import structlog
import httpx
import aiohttp
from prometheus_client.parser import text_string_to_metric_families

from ..models.sla_models import SLAMetricType, SLAMeasurement, SLATarget

logger = structlog.get_logger()


class ServiceEndpoint:
    """Configuration for a service endpoint"""
    def __init__(
        self,
        service_name: str,
        health_url: str,
        metrics_url: str,
        timeout_seconds: int = 5
    ):
        self.service_name = service_name
        self.health_url = health_url
        self.metrics_url = metrics_url
        self.timeout_seconds = timeout_seconds


class MetricsCollector:
    """
    Collects metrics from all runtime layer services for SLA monitoring

    Features:
    - Prometheus metrics scraping
    - Health endpoint monitoring
    - Custom metric calculation
    - Availability tracking
    - Performance measurement
    """

    def __init__(self):
        self.service_endpoints: List[ServiceEndpoint] = []
        self._http_client: Optional[httpx.AsyncClient] = None
        self._collection_history: Dict[str, List[Dict[str, Any]]] = {}
        self._last_collection_times: Dict[str, datetime] = {}

        # Metrics cache for calculation
        self._metrics_cache: Dict[str, Dict[str, float]] = {}

        logger.info("metrics_collector_initialized")

    def add_service_endpoint(self, endpoint: ServiceEndpoint):
        """Add a service endpoint for monitoring"""
        self.service_endpoints.append(endpoint)
        self._collection_history[endpoint.service_name] = []
        logger.info(
            "service_endpoint_added",
            service_name=endpoint.service_name,
            health_url=endpoint.health_url
        )

    async def collect_all_metrics(
        self,
        targets: List[SLATarget]
    ) -> List[SLAMeasurement]:
        """
        Collect all metrics for SLA evaluation
        """
        measurements = []
        collection_start = time.perf_counter()

        if not self._http_client:
            self._http_client = httpx.AsyncClient(
                timeout=httpx.Timeout(timeout=10.0),
                limits=httpx.Limits(max_connections=20)
            )

        # Group targets by service
        targets_by_service = {}
        for target in targets:
            if target.service_name not in targets_by_service:
                targets_by_service[target.service_name] = []
            targets_by_service[target.service_name].append(target)

        # Collect metrics for each service
        collection_tasks = []
        for service_name, service_targets in targets_by_service.items():
            task = self._collect_service_metrics(service_name, service_targets)
            collection_tasks.append(task)

        # Execute collections concurrently
        results = await asyncio.gather(*collection_tasks, return_exceptions=True)

        # Process results
        for result in results:
            if isinstance(result, list):
                measurements.extend(result)
            elif isinstance(result, Exception):
                logger.error("metrics_collection_error", error=str(result))

        collection_duration = (time.perf_counter() - collection_start) * 1000

        logger.info(
            "metrics_collection_completed",
            total_measurements=len(measurements),
            collection_duration_ms=collection_duration,
            services_monitored=len(targets_by_service)
        )

        return measurements

    async def _collect_service_metrics(
        self,
        service_name: str,
        targets: List[SLATarget]
    ) -> List[SLAMeasurement]:
        """
        Collect metrics for a specific service
        """
        measurements = []
        endpoint = self._get_endpoint_for_service(service_name)

        if not endpoint:
            logger.warning("service_endpoint_not_found", service_name=service_name)
            return measurements

        try:
            # Collect different metric types
            for target in targets:
                measurement = await self._collect_metric(endpoint, target)
                if measurement:
                    measurements.append(measurement)

            # Update collection history
            self._last_collection_times[service_name] = datetime.utcnow()

        except Exception as e:
            logger.error(
                "service_metrics_collection_error",
                service_name=service_name,
                error=str(e)
            )

        return measurements

    async def _collect_metric(
        self,
        endpoint: ServiceEndpoint,
        target: SLATarget
    ) -> Optional[SLAMeasurement]:
        """
        Collect a specific metric for a service
        """
        window_end = datetime.utcnow()
        window_start = window_end - timedelta(minutes=target.measurement_window_minutes)

        try:
            measured_value = None
            metadata = {}

            if target.metric_type == SLAMetricType.AVAILABILITY:
                measured_value, metadata = await self._measure_availability(endpoint)

            elif target.metric_type == SLAMetricType.RESPONSE_TIME:
                measured_value, metadata = await self._measure_response_time(endpoint)

            elif target.metric_type == SLAMetricType.ERROR_RATE:
                measured_value, metadata = await self._measure_error_rate(endpoint)

            elif target.metric_type == SLAMetricType.THROUGHPUT:
                measured_value, metadata = await self._measure_throughput(endpoint)

            elif target.metric_type == SLAMetricType.CACHE_HIT_RATE:
                measured_value, metadata = await self._measure_cache_hit_rate(endpoint)

            elif target.metric_type == SLAMetricType.ML_PREDICTION_ACCURACY:
                measured_value, metadata = await self._measure_ml_accuracy(endpoint)

            if measured_value is not None:
                return SLAMeasurement.from_target_and_value(
                    target=target,
                    measured_value=measured_value,
                    window_start=window_start,
                    window_end=window_end,
                    metadata=metadata
                )

        except Exception as e:
            logger.error(
                "metric_collection_error",
                service_name=endpoint.service_name,
                metric_type=target.metric_type,
                error=str(e)
            )

        return None

    async def _measure_availability(
        self,
        endpoint: ServiceEndpoint
    ) -> Tuple[float, Dict[str, Any]]:
        """
        Measure service availability by checking health endpoint
        """
        try:
            start_time = time.perf_counter()
            response = await self._http_client.get(
                endpoint.health_url,
                timeout=endpoint.timeout_seconds
            )
            response_time_ms = (time.perf_counter() - start_time) * 1000

            is_available = response.status_code == 200
            availability = 100.0 if is_available else 0.0

            metadata = {
                "status_code": response.status_code,
                "response_time_ms": response_time_ms,
                "endpoint": endpoint.health_url
            }

            if is_available:
                try:
                    health_data = response.json()
                    metadata["health_data"] = health_data
                except:
                    pass

            return availability, metadata

        except Exception as e:
            logger.warning(
                "availability_check_failed",
                service_name=endpoint.service_name,
                error=str(e)
            )
            return 0.0, {"error": str(e), "endpoint": endpoint.health_url}

    async def _measure_response_time(
        self,
        endpoint: ServiceEndpoint
    ) -> Tuple[float, Dict[str, Any]]:
        """
        Measure average response time from Prometheus metrics
        """
        try:
            prometheus_metrics = await self._fetch_prometheus_metrics(endpoint)

            # Look for response time histogram or summary
            response_time_metrics = [
                "http_request_duration_seconds",
                "request_duration_seconds",
                "response_time_seconds",
                "l1_cache_response_time_seconds"
            ]

            for metric_name in response_time_metrics:
                if metric_name in prometheus_metrics:
                    # Get P95 or average response time
                    p95_key = f"{metric_name}_quantile_0_95"
                    avg_key = f"{metric_name}_mean"

                    if p95_key in prometheus_metrics:
                        response_time_s = prometheus_metrics[p95_key]
                        response_time_ms = response_time_s * 1000.0
                        return response_time_ms, {
                            "metric_used": p95_key,
                            "percentile": "p95"
                        }
                    elif avg_key in prometheus_metrics:
                        response_time_s = prometheus_metrics[avg_key]
                        response_time_ms = response_time_s * 1000.0
                        return response_time_ms, {
                            "metric_used": avg_key,
                            "calculation": "average"
                        }

            # Fallback: measure direct health endpoint response time
            start_time = time.perf_counter()
            response = await self._http_client.get(
                endpoint.health_url,
                timeout=endpoint.timeout_seconds
            )
            response_time_ms = (time.perf_counter() - start_time) * 1000

            return response_time_ms, {
                "method": "direct_measurement",
                "endpoint": endpoint.health_url,
                "status_code": response.status_code
            }

        except Exception as e:
            logger.warning(
                "response_time_measurement_failed",
                service_name=endpoint.service_name,
                error=str(e)
            )
            # Return a high response time to indicate failure
            return 30000.0, {"error": str(e), "method": "error_fallback"}

    async def _measure_error_rate(
        self,
        endpoint: ServiceEndpoint
    ) -> Tuple[float, Dict[str, Any]]:
        """
        Measure error rate from Prometheus metrics
        """
        try:
            prometheus_metrics = await self._fetch_prometheus_metrics(endpoint)

            # Look for request counters
            total_requests = prometheus_metrics.get("http_requests_total", 0)
            error_requests = (
                prometheus_metrics.get("http_requests_total_5xx", 0) +
                prometheus_metrics.get("http_requests_total_4xx", 0)
            )

            if total_requests > 0:
                error_rate = (error_requests / total_requests) * 100.0
                return error_rate, {
                    "total_requests": total_requests,
                    "error_requests": error_requests,
                    "calculation_method": "prometheus_counters"
                }
            else:
                # No requests in measurement window
                return 0.0, {
                    "total_requests": 0,
                    "note": "no_requests_in_window"
                }

        except Exception as e:
            logger.warning(
                "error_rate_measurement_failed",
                service_name=endpoint.service_name,
                error=str(e)
            )
            return 0.0, {"error": str(e)}

    async def _measure_throughput(
        self,
        endpoint: ServiceEndpoint
    ) -> Tuple[float, Dict[str, Any]]:
        """
        Measure requests per second throughput
        """
        try:
            prometheus_metrics = await self._fetch_prometheus_metrics(endpoint)

            # Look for request rate metrics
            request_rate = prometheus_metrics.get("http_requests_per_second", 0)
            if request_rate > 0:
                return request_rate, {"metric_used": "http_requests_per_second"}

            # Calculate from total requests over time window
            total_requests = prometheus_metrics.get("http_requests_total", 0)

            # Get previous measurement to calculate rate
            service_name = endpoint.service_name
            if service_name in self._metrics_cache:
                prev_requests = self._metrics_cache[service_name].get("http_requests_total", 0)
                prev_time = self._last_collection_times.get(service_name)

                if prev_time:
                    time_diff_seconds = (datetime.utcnow() - prev_time).total_seconds()
                    if time_diff_seconds > 0:
                        request_rate = (total_requests - prev_requests) / time_diff_seconds
                        return request_rate, {
                            "calculation_method": "rate_from_counter",
                            "time_window_seconds": time_diff_seconds
                        }

            # Update cache for next calculation
            if service_name not in self._metrics_cache:
                self._metrics_cache[service_name] = {}
            self._metrics_cache[service_name]["http_requests_total"] = total_requests

            return 0.0, {"note": "insufficient_data_for_rate_calculation"}

        except Exception as e:
            logger.warning(
                "throughput_measurement_failed",
                service_name=endpoint.service_name,
                error=str(e)
            )
            return 0.0, {"error": str(e)}

    async def _measure_cache_hit_rate(
        self,
        endpoint: ServiceEndpoint
    ) -> Tuple[float, Dict[str, Any]]:
        """
        Measure cache hit rate (specific to cache services)
        """
        try:
            prometheus_metrics = await self._fetch_prometheus_metrics(endpoint)

            # Look for cache-specific metrics
            cache_hits = prometheus_metrics.get("l1_cache_requests_total_hit", 0)
            cache_misses = prometheus_metrics.get("l1_cache_requests_total_miss", 0)
            total_cache_requests = cache_hits + cache_misses

            if total_cache_requests > 0:
                hit_rate = (cache_hits / total_cache_requests) * 100.0
                return hit_rate, {
                    "cache_hits": cache_hits,
                    "cache_misses": cache_misses,
                    "total_requests": total_cache_requests
                }
            else:
                return 0.0, {"note": "no_cache_requests_in_window"}

        except Exception as e:
            logger.warning(
                "cache_hit_rate_measurement_failed",
                service_name=endpoint.service_name,
                error=str(e)
            )
            return 0.0, {"error": str(e)}

    async def _measure_ml_accuracy(
        self,
        endpoint: ServiceEndpoint
    ) -> Tuple[float, Dict[str, Any]]:
        """
        Measure ML prediction accuracy (specific to ML services)
        """
        try:
            prometheus_metrics = await self._fetch_prometheus_metrics(endpoint)

            # Look for ML-specific accuracy metrics
            accuracy = prometheus_metrics.get("prefetch_accuracy_ratio", 0) * 100.0

            if accuracy > 0:
                return accuracy, {
                    "metric_used": "prefetch_accuracy_ratio",
                    "measurement_type": "ml_prediction_accuracy"
                }
            else:
                return 0.0, {"note": "no_ml_accuracy_data"}

        except Exception as e:
            logger.warning(
                "ml_accuracy_measurement_failed",
                service_name=endpoint.service_name,
                error=str(e)
            )
            return 0.0, {"error": str(e)}

    async def _fetch_prometheus_metrics(
        self,
        endpoint: ServiceEndpoint
    ) -> Dict[str, float]:
        """
        Fetch and parse Prometheus metrics from service
        """
        try:
            response = await self._http_client.get(
                endpoint.metrics_url,
                timeout=endpoint.timeout_seconds
            )

            if response.status_code != 200:
                logger.warning(
                    "prometheus_metrics_fetch_failed",
                    service_name=endpoint.service_name,
                    status_code=response.status_code
                )
                return {}

            # Parse Prometheus text format
            metrics_data = {}
            for family in text_string_to_metric_families(response.text):
                for sample in family.samples:
                    metric_name = sample.name
                    metric_value = sample.value

                    # Include labels in metric name for specificity
                    if sample.labels:
                        label_str = "_".join(f"{k}_{v}" for k, v in sample.labels.items())
                        metric_name = f"{metric_name}_{label_str}"

                    metrics_data[metric_name] = metric_value

            return metrics_data

        except Exception as e:
            logger.warning(
                "prometheus_metrics_parse_error",
                service_name=endpoint.service_name,
                error=str(e)
            )
            return {}

    def _get_endpoint_for_service(self, service_name: str) -> Optional[ServiceEndpoint]:
        """
        Get endpoint configuration for a service
        """
        for endpoint in self.service_endpoints:
            if endpoint.service_name == service_name:
                return endpoint
        return None

    async def close(self):
        """
        Close HTTP client and cleanup resources
        """
        if self._http_client:
            await self._http_client.aclose()
            self._http_client = None

        logger.info("metrics_collector_closed")


# Default service endpoints for runtime layer
DEFAULT_SERVICE_ENDPOINTS = [
    ServiceEndpoint(
        service_name="flink-stream-processor",
        health_url="http://localhost:8081/jobs",  # Flink JobManager REST API
        metrics_url="http://localhost:8081/metrics",
        timeout_seconds=10
    ),
    ServiceEndpoint(
        service_name="evidence-envelope-service",
        health_url="http://localhost:8020/health",
        metrics_url="http://localhost:8020/metrics",
        timeout_seconds=5
    ),
    ServiceEndpoint(
        service_name="l1-cache-prefetcher-service",
        health_url="http://localhost:8030/health",
        metrics_url="http://localhost:8030/metrics",
        timeout_seconds=5
    )
]