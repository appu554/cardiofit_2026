"""
Performance Monitor for Production Clinical Intelligence System

Enterprise-grade performance monitoring with real-time metrics,
alerting, and optimization recommendations for clinical systems.
"""

import logging
import asyncio
import time
from datetime import datetime, timezone, timedelta
from typing import Dict, List, Optional, Any, Callable
from dataclasses import dataclass, field
from enum import Enum
import statistics
import json

logger = logging.getLogger(__name__)


class MetricType(Enum):
    """Types of performance metrics"""
    RESPONSE_TIME = "response_time"
    THROUGHPUT = "throughput"
    ERROR_RATE = "error_rate"
    CPU_USAGE = "cpu_usage"
    MEMORY_USAGE = "memory_usage"
    CACHE_HIT_RATE = "cache_hit_rate"
    CLINICAL_ACCURACY = "clinical_accuracy"
    SAFETY_SCORE = "safety_score"
    AVAILABILITY = "availability"


class AlertSeverity(Enum):
    """Alert severity levels"""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFO = "info"


@dataclass
class PerformanceMetric:
    """Performance metric data point"""
    metric_id: str
    metric_type: MetricType
    value: float
    unit: str
    timestamp: datetime
    source: str
    tags: Dict[str, str] = field(default_factory=dict)
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class AlertThreshold:
    """Alert threshold configuration"""
    metric_type: MetricType
    warning_threshold: float
    critical_threshold: float
    comparison_operator: str  # "gt", "lt", "eq"
    duration_seconds: int = 60  # Alert if threshold exceeded for this duration
    enabled: bool = True


@dataclass
class PerformanceAlert:
    """Performance alert"""
    alert_id: str
    metric_type: MetricType
    severity: AlertSeverity
    current_value: float
    threshold_value: float
    message: str
    triggered_at: datetime
    resolved_at: Optional[datetime] = None
    acknowledged: bool = False
    metadata: Dict[str, Any] = field(default_factory=dict)


class PerformanceMonitor:
    """
    Enterprise-grade performance monitoring system
    
    Provides real-time performance metrics collection, alerting,
    and optimization recommendations for clinical intelligence systems.
    """
    
    def __init__(self, collection_interval_seconds: int = 30):
        self.collection_interval = collection_interval_seconds
        
        # Metrics storage
        self.metrics_history: Dict[MetricType, List[PerformanceMetric]] = {
            metric_type: [] for metric_type in MetricType
        }
        
        # Alert configuration
        self.alert_thresholds: Dict[MetricType, AlertThreshold] = {}
        self.active_alerts: Dict[str, PerformanceAlert] = {}
        self.alert_history: List[PerformanceAlert] = []
        
        # Performance targets for clinical systems
        self._initialize_clinical_thresholds()
        
        # Monitoring state
        self.monitoring_active = False
        self.monitoring_task = None
        
        # Performance statistics
        self.performance_stats = {
            "total_metrics_collected": 0,
            "alerts_triggered": 0,
            "alerts_resolved": 0,
            "average_response_time": 0.0,
            "system_availability": 100.0,
            "last_collection": None
        }
        
        logger.info("Performance Monitor initialized")
    
    def _initialize_clinical_thresholds(self):
        """Initialize clinical system performance thresholds"""
        # Clinical systems require strict performance standards
        self.alert_thresholds = {
            MetricType.RESPONSE_TIME: AlertThreshold(
                metric_type=MetricType.RESPONSE_TIME,
                warning_threshold=100.0,  # 100ms warning
                critical_threshold=500.0,  # 500ms critical
                comparison_operator="gt",
                duration_seconds=30
            ),
            MetricType.ERROR_RATE: AlertThreshold(
                metric_type=MetricType.ERROR_RATE,
                warning_threshold=1.0,  # 1% error rate warning
                critical_threshold=5.0,  # 5% error rate critical
                comparison_operator="gt",
                duration_seconds=60
            ),
            MetricType.CLINICAL_ACCURACY: AlertThreshold(
                metric_type=MetricType.CLINICAL_ACCURACY,
                warning_threshold=85.0,  # 85% accuracy warning
                critical_threshold=80.0,  # 80% accuracy critical
                comparison_operator="lt",
                duration_seconds=300
            ),
            MetricType.SAFETY_SCORE: AlertThreshold(
                metric_type=MetricType.SAFETY_SCORE,
                warning_threshold=90.0,  # 90% safety warning
                critical_threshold=85.0,  # 85% safety critical
                comparison_operator="lt",
                duration_seconds=60
            ),
            MetricType.AVAILABILITY: AlertThreshold(
                metric_type=MetricType.AVAILABILITY,
                warning_threshold=99.0,  # 99% availability warning
                critical_threshold=95.0,  # 95% availability critical
                comparison_operator="lt",
                duration_seconds=120
            ),
            MetricType.CPU_USAGE: AlertThreshold(
                metric_type=MetricType.CPU_USAGE,
                warning_threshold=70.0,  # 70% CPU warning
                critical_threshold=90.0,  # 90% CPU critical
                comparison_operator="gt",
                duration_seconds=180
            ),
            MetricType.MEMORY_USAGE: AlertThreshold(
                metric_type=MetricType.MEMORY_USAGE,
                warning_threshold=80.0,  # 80% memory warning
                critical_threshold=95.0,  # 95% memory critical
                comparison_operator="gt",
                duration_seconds=180
            ),
            MetricType.CACHE_HIT_RATE: AlertThreshold(
                metric_type=MetricType.CACHE_HIT_RATE,
                warning_threshold=80.0,  # 80% cache hit rate warning
                critical_threshold=70.0,  # 70% cache hit rate critical
                comparison_operator="lt",
                duration_seconds=300
            )
        }
    
    async def start_monitoring(self):
        """Start performance monitoring"""
        if self.monitoring_active:
            logger.warning("Performance monitoring already active")
            return
        
        self.monitoring_active = True
        self.monitoring_task = asyncio.create_task(self._monitoring_loop())
        logger.info("Performance monitoring started")
    
    async def stop_monitoring(self):
        """Stop performance monitoring"""
        if not self.monitoring_active:
            return
        
        self.monitoring_active = False
        if self.monitoring_task:
            self.monitoring_task.cancel()
            try:
                await self.monitoring_task
            except asyncio.CancelledError:
                pass
        
        logger.info("Performance monitoring stopped")
    
    async def _monitoring_loop(self):
        """Main monitoring loop"""
        try:
            while self.monitoring_active:
                await self._collect_metrics()
                await self._check_alerts()
                await asyncio.sleep(self.collection_interval)
        except asyncio.CancelledError:
            logger.info("Monitoring loop cancelled")
        except Exception as e:
            logger.error(f"Error in monitoring loop: {e}")
    
    async def _collect_metrics(self):
        """Collect performance metrics"""
        try:
            timestamp = datetime.now(timezone.utc)
            
            # Collect system metrics
            await self._collect_system_metrics(timestamp)
            
            # Collect application metrics
            await self._collect_application_metrics(timestamp)
            
            # Collect clinical metrics
            await self._collect_clinical_metrics(timestamp)
            
            # Update statistics
            self.performance_stats["total_metrics_collected"] += len(MetricType)
            self.performance_stats["last_collection"] = timestamp
            
        except Exception as e:
            logger.error(f"Error collecting metrics: {e}")
    
    async def _collect_system_metrics(self, timestamp: datetime):
        """Collect system-level metrics"""
        try:
            import psutil
            
            # CPU usage
            cpu_percent = psutil.cpu_percent(interval=1)
            await self.record_metric(PerformanceMetric(
                metric_id=f"cpu_usage_{timestamp.timestamp()}",
                metric_type=MetricType.CPU_USAGE,
                value=cpu_percent,
                unit="percent",
                timestamp=timestamp,
                source="system",
                tags={"component": "system"}
            ))
            
            # Memory usage
            memory = psutil.virtual_memory()
            memory_percent = memory.percent
            await self.record_metric(PerformanceMetric(
                metric_id=f"memory_usage_{timestamp.timestamp()}",
                metric_type=MetricType.MEMORY_USAGE,
                value=memory_percent,
                unit="percent",
                timestamp=timestamp,
                source="system",
                tags={"component": "system"}
            ))
            
        except ImportError:
            # psutil not available, use mock data
            await self._collect_mock_system_metrics(timestamp)
        except Exception as e:
            logger.error(f"Error collecting system metrics: {e}")
    
    async def _collect_mock_system_metrics(self, timestamp: datetime):
        """Collect mock system metrics when psutil not available"""
        import random
        
        # Mock CPU usage (simulate normal operation)
        cpu_percent = random.uniform(20, 60)
        await self.record_metric(PerformanceMetric(
            metric_id=f"cpu_usage_{timestamp.timestamp()}",
            metric_type=MetricType.CPU_USAGE,
            value=cpu_percent,
            unit="percent",
            timestamp=timestamp,
            source="system_mock",
            tags={"component": "system"}
        ))
        
        # Mock memory usage
        memory_percent = random.uniform(40, 70)
        await self.record_metric(PerformanceMetric(
            metric_id=f"memory_usage_{timestamp.timestamp()}",
            metric_type=MetricType.MEMORY_USAGE,
            value=memory_percent,
            unit="percent",
            timestamp=timestamp,
            source="system_mock",
            tags={"component": "system"}
        ))
    
    async def _collect_application_metrics(self, timestamp: datetime):
        """Collect application-level metrics"""
        try:
            # Response time (would be collected from actual requests)
            response_time = await self._get_average_response_time()
            await self.record_metric(PerformanceMetric(
                metric_id=f"response_time_{timestamp.timestamp()}",
                metric_type=MetricType.RESPONSE_TIME,
                value=response_time,
                unit="milliseconds",
                timestamp=timestamp,
                source="application",
                tags={"component": "cae"}
            ))
            
            # Error rate
            error_rate = await self._get_error_rate()
            await self.record_metric(PerformanceMetric(
                metric_id=f"error_rate_{timestamp.timestamp()}",
                metric_type=MetricType.ERROR_RATE,
                value=error_rate,
                unit="percent",
                timestamp=timestamp,
                source="application",
                tags={"component": "cae"}
            ))
            
            # Cache hit rate
            cache_hit_rate = await self._get_cache_hit_rate()
            await self.record_metric(PerformanceMetric(
                metric_id=f"cache_hit_rate_{timestamp.timestamp()}",
                metric_type=MetricType.CACHE_HIT_RATE,
                value=cache_hit_rate,
                unit="percent",
                timestamp=timestamp,
                source="application",
                tags={"component": "cache"}
            ))
            
            # Availability
            availability = await self._get_system_availability()
            await self.record_metric(PerformanceMetric(
                metric_id=f"availability_{timestamp.timestamp()}",
                metric_type=MetricType.AVAILABILITY,
                value=availability,
                unit="percent",
                timestamp=timestamp,
                source="application",
                tags={"component": "system"}
            ))
            
        except Exception as e:
            logger.error(f"Error collecting application metrics: {e}")
    
    async def _collect_clinical_metrics(self, timestamp: datetime):
        """Collect clinical-specific metrics"""
        try:
            # Clinical accuracy
            clinical_accuracy = await self._get_clinical_accuracy()
            await self.record_metric(PerformanceMetric(
                metric_id=f"clinical_accuracy_{timestamp.timestamp()}",
                metric_type=MetricType.CLINICAL_ACCURACY,
                value=clinical_accuracy,
                unit="percent",
                timestamp=timestamp,
                source="clinical",
                tags={"component": "validation"}
            ))
            
            # Safety score
            safety_score = await self._get_safety_score()
            await self.record_metric(PerformanceMetric(
                metric_id=f"safety_score_{timestamp.timestamp()}",
                metric_type=MetricType.SAFETY_SCORE,
                value=safety_score,
                unit="percent",
                timestamp=timestamp,
                source="clinical",
                tags={"component": "safety"}
            ))
            
        except Exception as e:
            logger.error(f"Error collecting clinical metrics: {e}")
    
    async def record_metric(self, metric: PerformanceMetric):
        """Record a performance metric"""
        try:
            # Store metric
            self.metrics_history[metric.metric_type].append(metric)
            
            # Keep only recent metrics (last 24 hours)
            cutoff_time = datetime.now(timezone.utc) - timedelta(hours=24)
            self.metrics_history[metric.metric_type] = [
                m for m in self.metrics_history[metric.metric_type]
                if m.timestamp > cutoff_time
            ]
            
            logger.debug(f"Recorded metric: {metric.metric_type.value} = {metric.value} {metric.unit}")
            
        except Exception as e:
            logger.error(f"Error recording metric: {e}")
    
    async def _check_alerts(self):
        """Check for alert conditions"""
        try:
            for metric_type, threshold in self.alert_thresholds.items():
                if not threshold.enabled:
                    continue
                
                # Get recent metrics for this type
                recent_metrics = self._get_recent_metrics(metric_type, threshold.duration_seconds)
                
                if not recent_metrics:
                    continue
                
                # Calculate average value over duration
                avg_value = statistics.mean([m.value for m in recent_metrics])
                
                # Check thresholds
                alert_severity = self._check_threshold(avg_value, threshold)
                
                if alert_severity:
                    await self._trigger_alert(metric_type, alert_severity, avg_value, threshold)
                else:
                    # Check if we should resolve any existing alerts
                    await self._check_alert_resolution(metric_type, avg_value, threshold)
            
        except Exception as e:
            logger.error(f"Error checking alerts: {e}")
    
    def _get_recent_metrics(self, metric_type: MetricType, duration_seconds: int) -> List[PerformanceMetric]:
        """Get metrics from the specified duration"""
        cutoff_time = datetime.now(timezone.utc) - timedelta(seconds=duration_seconds)
        return [
            m for m in self.metrics_history[metric_type]
            if m.timestamp > cutoff_time
        ]
    
    def _check_threshold(self, value: float, threshold: AlertThreshold) -> Optional[AlertSeverity]:
        """Check if value exceeds threshold"""
        if threshold.comparison_operator == "gt":
            if value > threshold.critical_threshold:
                return AlertSeverity.CRITICAL
            elif value > threshold.warning_threshold:
                return AlertSeverity.HIGH
        elif threshold.comparison_operator == "lt":
            if value < threshold.critical_threshold:
                return AlertSeverity.CRITICAL
            elif value < threshold.warning_threshold:
                return AlertSeverity.HIGH
        elif threshold.comparison_operator == "eq":
            if abs(value - threshold.critical_threshold) < 0.01:
                return AlertSeverity.CRITICAL
            elif abs(value - threshold.warning_threshold) < 0.01:
                return AlertSeverity.HIGH
        
        return None

    async def _trigger_alert(self, metric_type: MetricType, severity: AlertSeverity,
                           current_value: float, threshold: AlertThreshold):
        """Trigger a performance alert"""
        try:
            # Check if alert already exists
            alert_key = f"{metric_type.value}_{severity.value}"

            if alert_key in self.active_alerts:
                # Alert already active
                return

            # Create new alert
            alert = PerformanceAlert(
                alert_id=f"alert_{metric_type.value}_{datetime.now().timestamp()}",
                metric_type=metric_type,
                severity=severity,
                current_value=current_value,
                threshold_value=(threshold.critical_threshold if severity == AlertSeverity.CRITICAL
                               else threshold.warning_threshold),
                message=self._generate_alert_message(metric_type, severity, current_value, threshold),
                triggered_at=datetime.now(timezone.utc),
                metadata={
                    "threshold_config": {
                        "warning": threshold.warning_threshold,
                        "critical": threshold.critical_threshold,
                        "operator": threshold.comparison_operator,
                        "duration": threshold.duration_seconds
                    }
                }
            )

            # Store alert
            self.active_alerts[alert_key] = alert
            self.alert_history.append(alert)

            # Update statistics
            self.performance_stats["alerts_triggered"] += 1

            # Log alert
            logger.warning(f"Performance alert triggered: {alert.message}")

            # Send alert notification (would integrate with alerting system)
            await self._send_alert_notification(alert)

        except Exception as e:
            logger.error(f"Error triggering alert: {e}")

    async def _check_alert_resolution(self, metric_type: MetricType, current_value: float,
                                    threshold: AlertThreshold):
        """Check if alerts should be resolved"""
        try:
            alerts_to_resolve = []

            for alert_key, alert in self.active_alerts.items():
                if alert.metric_type != metric_type:
                    continue

                # Check if value is back within acceptable range
                should_resolve = False

                if threshold.comparison_operator == "gt":
                    should_resolve = current_value <= threshold.warning_threshold
                elif threshold.comparison_operator == "lt":
                    should_resolve = current_value >= threshold.warning_threshold
                elif threshold.comparison_operator == "eq":
                    should_resolve = abs(current_value - threshold.warning_threshold) > 0.01

                if should_resolve:
                    alerts_to_resolve.append(alert_key)

            # Resolve alerts
            for alert_key in alerts_to_resolve:
                await self._resolve_alert(alert_key)

        except Exception as e:
            logger.error(f"Error checking alert resolution: {e}")

    async def _resolve_alert(self, alert_key: str):
        """Resolve an active alert"""
        try:
            if alert_key not in self.active_alerts:
                return

            alert = self.active_alerts[alert_key]
            alert.resolved_at = datetime.now(timezone.utc)

            # Remove from active alerts
            del self.active_alerts[alert_key]

            # Update statistics
            self.performance_stats["alerts_resolved"] += 1

            logger.info(f"Performance alert resolved: {alert.message}")

            # Send resolution notification
            await self._send_alert_resolution_notification(alert)

        except Exception as e:
            logger.error(f"Error resolving alert: {e}")

    def _generate_alert_message(self, metric_type: MetricType, severity: AlertSeverity,
                              current_value: float, threshold: AlertThreshold) -> str:
        """Generate alert message"""
        threshold_value = (threshold.critical_threshold if severity == AlertSeverity.CRITICAL
                          else threshold.warning_threshold)

        return (f"{severity.value.upper()} Alert: {metric_type.value} is {current_value:.2f}, "
                f"threshold is {threshold_value:.2f}")

    async def _send_alert_notification(self, alert: PerformanceAlert):
        """Send alert notification (placeholder for integration)"""
        # In production, this would integrate with:
        # - Email/SMS notifications
        # - Slack/Teams alerts
        # - PagerDuty/OpsGenie
        # - Monitoring dashboards
        logger.info(f"Alert notification sent: {alert.alert_id}")

    async def _send_alert_resolution_notification(self, alert: PerformanceAlert):
        """Send alert resolution notification"""
        logger.info(f"Alert resolution notification sent: {alert.alert_id}")

    # Metric collection helper methods (would integrate with actual systems)

    async def _get_average_response_time(self) -> float:
        """Get average response time"""
        # In production, this would query actual response time metrics
        import random
        return random.uniform(50, 150)  # Mock response time in ms

    async def _get_error_rate(self) -> float:
        """Get current error rate"""
        # In production, this would query actual error metrics
        import random
        return random.uniform(0, 2)  # Mock error rate percentage

    async def _get_cache_hit_rate(self) -> float:
        """Get cache hit rate"""
        # In production, this would query cache metrics
        import random
        return random.uniform(85, 95)  # Mock cache hit rate percentage

    async def _get_system_availability(self) -> float:
        """Get system availability"""
        # In production, this would calculate from uptime metrics
        import random
        return random.uniform(99, 100)  # Mock availability percentage

    async def _get_clinical_accuracy(self) -> float:
        """Get clinical accuracy score"""
        # In production, this would query validation metrics
        import random
        return random.uniform(88, 95)  # Mock clinical accuracy percentage

    async def _get_safety_score(self) -> float:
        """Get safety score"""
        # In production, this would query safety metrics
        import random
        return random.uniform(92, 98)  # Mock safety score percentage

    # Public API methods

    def get_current_metrics(self) -> Dict[MetricType, Optional[PerformanceMetric]]:
        """Get current (latest) metrics for each type"""
        current_metrics = {}

        for metric_type in MetricType:
            metrics = self.metrics_history[metric_type]
            if metrics:
                current_metrics[metric_type] = max(metrics, key=lambda m: m.timestamp)
            else:
                current_metrics[metric_type] = None

        return current_metrics

    def get_metrics_history(self, metric_type: MetricType,
                          hours_back: int = 1) -> List[PerformanceMetric]:
        """Get metrics history for specified type and time range"""
        cutoff_time = datetime.now(timezone.utc) - timedelta(hours=hours_back)
        return [
            m for m in self.metrics_history[metric_type]
            if m.timestamp > cutoff_time
        ]

    def get_active_alerts(self) -> List[PerformanceAlert]:
        """Get all active alerts"""
        return list(self.active_alerts.values())

    def get_alert_history(self, hours_back: int = 24) -> List[PerformanceAlert]:
        """Get alert history for specified time range"""
        cutoff_time = datetime.now(timezone.utc) - timedelta(hours=hours_back)
        return [
            alert for alert in self.alert_history
            if alert.triggered_at > cutoff_time
        ]

    def acknowledge_alert(self, alert_id: str) -> bool:
        """Acknowledge an alert"""
        for alert in self.active_alerts.values():
            if alert.alert_id == alert_id:
                alert.acknowledged = True
                logger.info(f"Alert acknowledged: {alert_id}")
                return True
        return False

    def get_performance_summary(self) -> Dict[str, Any]:
        """Get comprehensive performance summary"""
        current_metrics = self.get_current_metrics()
        active_alerts = self.get_active_alerts()

        # Calculate performance scores
        response_time_metric = current_metrics.get(MetricType.RESPONSE_TIME)
        error_rate_metric = current_metrics.get(MetricType.ERROR_RATE)
        availability_metric = current_metrics.get(MetricType.AVAILABILITY)
        clinical_accuracy_metric = current_metrics.get(MetricType.CLINICAL_ACCURACY)
        safety_score_metric = current_metrics.get(MetricType.SAFETY_SCORE)

        return {
            "performance_stats": self.performance_stats,
            "current_metrics": {
                "response_time_ms": response_time_metric.value if response_time_metric else None,
                "error_rate_percent": error_rate_metric.value if error_rate_metric else None,
                "availability_percent": availability_metric.value if availability_metric else None,
                "clinical_accuracy_percent": clinical_accuracy_metric.value if clinical_accuracy_metric else None,
                "safety_score_percent": safety_score_metric.value if safety_score_metric else None
            },
            "alerts": {
                "active_count": len(active_alerts),
                "critical_count": len([a for a in active_alerts if a.severity == AlertSeverity.CRITICAL]),
                "high_count": len([a for a in active_alerts if a.severity == AlertSeverity.HIGH]),
                "active_alerts": [
                    {
                        "alert_id": alert.alert_id,
                        "metric_type": alert.metric_type.value,
                        "severity": alert.severity.value,
                        "message": alert.message,
                        "triggered_at": alert.triggered_at.isoformat(),
                        "acknowledged": alert.acknowledged
                    }
                    for alert in active_alerts
                ]
            },
            "health_status": self._calculate_overall_health_status(),
            "recommendations": self._generate_performance_recommendations()
        }

    def _calculate_overall_health_status(self) -> str:
        """Calculate overall system health status"""
        active_alerts = self.get_active_alerts()

        if any(alert.severity == AlertSeverity.CRITICAL for alert in active_alerts):
            return "critical"
        elif any(alert.severity == AlertSeverity.HIGH for alert in active_alerts):
            return "degraded"
        elif any(alert.severity in [AlertSeverity.MEDIUM, AlertSeverity.LOW] for alert in active_alerts):
            return "warning"
        else:
            return "healthy"

    def _generate_performance_recommendations(self) -> List[str]:
        """Generate performance optimization recommendations"""
        recommendations = []
        current_metrics = self.get_current_metrics()

        # Response time recommendations
        response_time_metric = current_metrics.get(MetricType.RESPONSE_TIME)
        if response_time_metric and response_time_metric.value > 100:
            recommendations.append("Consider optimizing response times - current average exceeds 100ms")

        # Cache recommendations
        cache_hit_metric = current_metrics.get(MetricType.CACHE_HIT_RATE)
        if cache_hit_metric and cache_hit_metric.value < 85:
            recommendations.append("Improve cache hit rate - consider cache warming or optimization")

        # Error rate recommendations
        error_rate_metric = current_metrics.get(MetricType.ERROR_RATE)
        if error_rate_metric and error_rate_metric.value > 1:
            recommendations.append("Investigate and reduce error rate")

        # Clinical accuracy recommendations
        accuracy_metric = current_metrics.get(MetricType.CLINICAL_ACCURACY)
        if accuracy_metric and accuracy_metric.value < 90:
            recommendations.append("Review clinical validation processes to improve accuracy")

        return recommendations
