"""
Cloud-Native Metrics Collector for Transactional Outbox Pattern

CRITICAL DESIGN DECISION: Metrics are emitted directly to monitoring systems 
(Google Cloud Monitoring, Prometheus, Datadog) rather than written to the 
transactional database.

RATIONALE:
- Reduces write load on primary PostgreSQL database
- Follows cloud-native observability patterns
- Prevents metrics collection from impacting transactional performance
- Enables real-time monitoring without database queries
"""
import logging
import time
from datetime import datetime
from typing import Dict, Any, Optional, List
from contextlib import asynccontextmanager

try:
    from google.cloud import monitoring_v3
    from google.cloud.monitoring_v3 import TimeSeries, Point, TimeInterval
    from google.api_core.exceptions import GoogleAPIError
    GOOGLE_CLOUD_AVAILABLE = True
except ImportError:
    GOOGLE_CLOUD_AVAILABLE = False
    monitoring_v3 = None

from app.config import settings

logger = logging.getLogger(__name__)


class CloudNativeMetricsCollector:
    """
    Cloud-native metrics emission without database writes
    
    Supports multiple backends:
    - Google Cloud Monitoring (primary)
    - Prometheus (future)
    - Datadog (future)
    - Local logging (fallback)
    """
    
    def __init__(self):
        self.project_name = f"projects/{settings.GCP_PROJECT_ID}"
        self.monitoring_client = None
        self.metrics_enabled = settings.ENABLE_CLOUD_METRICS
        self._initialize_client()
    
    def _initialize_client(self):
        """Initialize Google Cloud Monitoring client"""
        if not self.metrics_enabled:
            logger.info("Cloud metrics disabled")
            return
            
        if not GOOGLE_CLOUD_AVAILABLE:
            logger.warning("Google Cloud Monitoring not available, falling back to logging")
            return
            
        try:
            self.monitoring_client = monitoring_v3.MetricServiceClient()
            logger.info("✅ Google Cloud Monitoring client initialized")
        except Exception as e:
            logger.error(f"❌ Failed to initialize Google Cloud Monitoring: {e}")
            self.monitoring_client = None
    
    async def emit_outbox_queue_depth(self, vendor_id: str, queue_depth: int):
        """Emit outbox queue depth metrics"""
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/queue_depth",
            "value": queue_depth,
            "labels": {
                "vendor_id": vendor_id,
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    async def emit_processing_latency(self, vendor_id: str, latency_ms: float):
        """Emit processing latency metrics"""
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/processing_latency_ms",
            "value": latency_ms,
            "labels": {
                "vendor_id": vendor_id,
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    async def emit_message_success(self, vendor_id: str, count: int = 1):
        """Emit successful message processing count"""
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/messages_processed",
            "value": count,
            "labels": {
                "vendor_id": vendor_id,
                "status": "success",
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    async def emit_message_failure(self, vendor_id: str, error_type: str, count: int = 1):
        """Emit failed message processing count"""
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/messages_processed",
            "value": count,
            "labels": {
                "vendor_id": vendor_id,
                "status": "failed",
                "error_type": error_type,
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    async def emit_dead_letter_alert(self, vendor_id: str, message_id: str, error: str):
        """Emit critical alert for dead letter messages"""
        # This is a critical alert - always log even if cloud metrics fail
        logger.error(f"🚨 DEAD LETTER ALERT: Message {message_id} from {vendor_id} moved to dead letter", extra={
            "vendor_id": vendor_id,
            "message_id": message_id,
            "error": error,
            "alert_type": "dead_letter",
            "severity": "critical"
        })
        
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/dead_letter_messages",
            "value": 1,
            "labels": {
                "vendor_id": vendor_id,
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    async def emit_skip_locked_contention(self, vendor_id: str, contention_ratio: float):
        """Emit SELECT FOR UPDATE SKIP LOCKED contention metrics"""
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/skip_locked_contention_ratio",
            "value": contention_ratio,
            "labels": {
                "vendor_id": vendor_id,
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    async def emit_publisher_health(self, vendor_id: str, is_healthy: bool):
        """Emit publisher service health status"""
        metric_data = {
            "metric_type": "custom.googleapis.com/outbox/publisher_health",
            "value": 1 if is_healthy else 0,
            "labels": {
                "vendor_id": vendor_id,
                "service": "device-data-ingestion"
            }
        }
        await self._emit_metric(metric_data)
    
    @asynccontextmanager
    async def measure_processing_time(self, vendor_id: str):
        """Context manager to measure and emit processing time"""
        start_time = time.time()
        try:
            yield
        finally:
            end_time = time.time()
            latency_ms = (end_time - start_time) * 1000
            await self.emit_processing_latency(vendor_id, latency_ms)
    
    async def _emit_metric(self, metric_data: Dict[str, Any]):
        """Internal method to emit metrics to configured backend"""
        if not self.metrics_enabled:
            return
            
        # Always log metrics locally for debugging
        logger.info(f"📊 Metric: {metric_data['metric_type']} = {metric_data['value']}", extra={
            "metric_type": metric_data["metric_type"],
            "metric_value": metric_data["value"],
            "metric_labels": metric_data.get("labels", {})
        })
        
        # Emit to Google Cloud Monitoring if available
        if self.monitoring_client:
            try:
                await self._emit_to_google_cloud(metric_data)
            except Exception as e:
                logger.error(f"Failed to emit metric to Google Cloud: {e}")
    
    async def _emit_to_google_cloud(self, metric_data: Dict[str, Any]):
        """Emit metric to Google Cloud Monitoring"""
        try:
            # Create time series
            now = time.time()
            seconds = int(now)
            nanos = int((now - seconds) * 10 ** 9)
            
            interval = TimeInterval({
                "end_time": {"seconds": seconds, "nanos": nanos}
            })
            
            point = Point({
                "interval": interval,
                "value": {"double_value": float(metric_data["value"])}
            })
            
            # Build metric descriptor
            metric = {
                "type": metric_data["metric_type"],
                "labels": metric_data.get("labels", {})
            }
            
            # Build resource
            resource = {
                "type": "generic_node",
                "labels": {
                    "location": "global",
                    "namespace": "device-data-ingestion",
                    "node_id": "outbox-service"
                }
            }
            
            series = TimeSeries({
                "metric": metric,
                "resource": resource,
                "points": [point]
            })
            
            # Send to Google Cloud Monitoring
            self.monitoring_client.create_time_series(
                name=self.project_name,
                time_series=[series]
            )
            
        except GoogleAPIError as e:
            logger.error(f"Google Cloud Monitoring API error: {e}")
        except Exception as e:
            logger.error(f"Unexpected error emitting to Google Cloud: {e}")
    
    async def emit_batch_metrics(self, metrics_batch: List[Dict[str, Any]]):
        """Emit multiple metrics in a single batch for efficiency"""
        if not self.metrics_enabled or not metrics_batch:
            return
            
        # Log all metrics
        for metric_data in metrics_batch:
            logger.info(f"📊 Batch Metric: {metric_data['metric_type']} = {metric_data['value']}")
        
        # Emit to Google Cloud in batch if available
        if self.monitoring_client:
            try:
                await self._emit_batch_to_google_cloud(metrics_batch)
            except Exception as e:
                logger.error(f"Failed to emit batch metrics to Google Cloud: {e}")
    
    async def _emit_batch_to_google_cloud(self, metrics_batch: List[Dict[str, Any]]):
        """Emit multiple metrics to Google Cloud Monitoring in a single API call"""
        try:
            time_series_list = []
            now = time.time()
            seconds = int(now)
            nanos = int((now - seconds) * 10 ** 9)
            
            interval = TimeInterval({
                "end_time": {"seconds": seconds, "nanos": nanos}
            })
            
            for metric_data in metrics_batch:
                point = Point({
                    "interval": interval,
                    "value": {"double_value": float(metric_data["value"])}
                })
                
                metric = {
                    "type": metric_data["metric_type"],
                    "labels": metric_data.get("labels", {})
                }
                
                resource = {
                    "type": "generic_node",
                    "labels": {
                        "location": "global",
                        "namespace": "device-data-ingestion",
                        "node_id": "outbox-service"
                    }
                }
                
                series = TimeSeries({
                    "metric": metric,
                    "resource": resource,
                    "points": [point]
                })
                
                time_series_list.append(series)
            
            # Send batch to Google Cloud Monitoring
            self.monitoring_client.create_time_series(
                name=self.project_name,
                time_series=time_series_list
            )
            
            logger.info(f"✅ Emitted {len(metrics_batch)} metrics to Google Cloud Monitoring")
            
        except GoogleAPIError as e:
            logger.error(f"Google Cloud Monitoring batch API error: {e}")
        except Exception as e:
            logger.error(f"Unexpected error emitting batch to Google Cloud: {e}")
    
    def get_health_status(self) -> Dict[str, Any]:
        """Get health status of metrics collector"""
        return {
            "metrics_enabled": self.metrics_enabled,
            "google_cloud_available": GOOGLE_CLOUD_AVAILABLE,
            "monitoring_client_initialized": self.monitoring_client is not None,
            "project_name": self.project_name
        }


# Global metrics collector instance
metrics_collector = CloudNativeMetricsCollector()
