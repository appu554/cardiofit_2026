"""
Observability and metrics for Kafka operations
"""

import time
import logging
import threading
from typing import Dict, Any, Optional, List, Callable
from datetime import datetime, timedelta
from collections import defaultdict, deque
from dataclasses import dataclass, field
from contextlib import contextmanager
import json

try:
    from prometheus_client import Counter, Histogram, Gauge, Info
    PROMETHEUS_AVAILABLE = True
except ImportError:
    PROMETHEUS_AVAILABLE = False

logger = logging.getLogger(__name__)

@dataclass
class OperationMetrics:
    """Metrics for a specific operation"""
    operation_name: str
    total_count: int = 0
    success_count: int = 0
    failure_count: int = 0
    total_duration: float = 0.0
    min_duration: float = float('inf')
    max_duration: float = 0.0
    last_success: Optional[datetime] = None
    last_failure: Optional[datetime] = None
    recent_durations: deque = field(default_factory=lambda: deque(maxlen=100))
    
    def add_success(self, duration: float):
        """Record a successful operation"""
        self.total_count += 1
        self.success_count += 1
        self.total_duration += duration
        self.min_duration = min(self.min_duration, duration)
        self.max_duration = max(self.max_duration, duration)
        self.last_success = datetime.now()
        self.recent_durations.append(duration)
    
    def add_failure(self, duration: float):
        """Record a failed operation"""
        self.total_count += 1
        self.failure_count += 1
        self.total_duration += duration
        self.min_duration = min(self.min_duration, duration)
        self.max_duration = max(self.max_duration, duration)
        self.last_failure = datetime.now()
        self.recent_durations.append(duration)
    
    @property
    def success_rate(self) -> float:
        """Calculate success rate"""
        if self.total_count == 0:
            return 0.0
        return self.success_count / self.total_count
    
    @property
    def failure_rate(self) -> float:
        """Calculate failure rate"""
        if self.total_count == 0:
            return 0.0
        return self.failure_count / self.total_count
    
    @property
    def average_duration(self) -> float:
        """Calculate average duration"""
        if self.total_count == 0:
            return 0.0
        return self.total_duration / self.total_count
    
    @property
    def recent_average_duration(self) -> float:
        """Calculate recent average duration"""
        if not self.recent_durations:
            return 0.0
        return sum(self.recent_durations) / len(self.recent_durations)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            'operation_name': self.operation_name,
            'total_count': self.total_count,
            'success_count': self.success_count,
            'failure_count': self.failure_count,
            'success_rate': self.success_rate,
            'failure_rate': self.failure_rate,
            'average_duration': self.average_duration,
            'recent_average_duration': self.recent_average_duration,
            'min_duration': self.min_duration if self.min_duration != float('inf') else 0.0,
            'max_duration': self.max_duration,
            'last_success': self.last_success.isoformat() if self.last_success else None,
            'last_failure': self.last_failure.isoformat() if self.last_failure else None
        }

class KafkaMetricsCollector:
    """Collects and manages Kafka operation metrics"""
    
    def __init__(self, service_name: str):
        self.service_name = service_name
        self.metrics: Dict[str, OperationMetrics] = {}
        self.lock = threading.Lock()
        
        # Prometheus metrics (if available)
        if PROMETHEUS_AVAILABLE:
            self._setup_prometheus_metrics()
        
        # Custom metrics storage
        self.custom_metrics: Dict[str, Any] = defaultdict(dict)
        
        logger.info(f"KafkaMetricsCollector initialized for service: {service_name}")
    
    def _setup_prometheus_metrics(self):
        """Setup Prometheus metrics"""
        try:
            # Use service-specific metric names to avoid conflicts
            service_prefix = self.service_name.replace('-', '_').replace('.', '_')

            self.kafka_operations_total = Counter(
                f'{service_prefix}_kafka_operations_total',
                'Total number of Kafka operations',
                ['service', 'operation', 'status']
            )

            self.kafka_operation_duration = Histogram(
                f'{service_prefix}_kafka_operation_duration_seconds',
                'Duration of Kafka operations',
                ['service', 'operation']
            )

            self.kafka_producer_messages = Counter(
                f'{service_prefix}_kafka_producer_messages_total',
                'Total number of messages produced',
                ['service', 'topic', 'status']
            )

            self.kafka_consumer_messages = Counter(
                f'{service_prefix}_kafka_consumer_messages_total',
                'Total number of messages consumed',
                ['service', 'topic', 'status']
            )

            self.kafka_connection_status = Gauge(
                f'{service_prefix}_kafka_connection_status',
                'Kafka connection status (1=connected, 0=disconnected)',
                ['service']
            )

            self.kafka_lag = Gauge(
                f'{service_prefix}_kafka_consumer_lag',
                'Consumer lag in messages',
                ['service', 'topic', 'partition']
            )

            self.prometheus_enabled = True
            logger.info("Prometheus metrics initialized")

        except Exception as e:
            self.prometheus_enabled = False
            logger.error(f"Failed to setup Prometheus metrics: {e}")
            # Set None values for all metrics
            self.kafka_operations_total = None
            self.kafka_operation_duration = None
            self.kafka_producer_messages = None
            self.kafka_consumer_messages = None
            self.kafka_connection_status = None
            self.kafka_lag = None
    
    @contextmanager
    def measure_operation(self, operation_name: str):
        """Context manager to measure operation duration"""
        start_time = time.time()
        success = False
        
        try:
            yield
            success = True
        except Exception:
            success = False
            raise
        finally:
            duration = time.time() - start_time
            self.record_operation(operation_name, duration, success)
    
    def record_operation(self, operation_name: str, duration: float, success: bool):
        """Record an operation result"""
        with self.lock:
            if operation_name not in self.metrics:
                self.metrics[operation_name] = OperationMetrics(operation_name)
            
            if success:
                self.metrics[operation_name].add_success(duration)
            else:
                self.metrics[operation_name].add_failure(duration)
        
        # Update Prometheus metrics
        if PROMETHEUS_AVAILABLE and hasattr(self, 'prometheus_enabled') and self.prometheus_enabled:
            status = 'success' if success else 'failure'
            if self.kafka_operations_total:
                self.kafka_operations_total.labels(
                    service=self.service_name,
                    operation=operation_name,
                    status=status
                ).inc()

            if self.kafka_operation_duration:
                self.kafka_operation_duration.labels(
                    service=self.service_name,
                    operation=operation_name
                ).observe(duration)
    
    def record_message_produced(self, topic: str, success: bool):
        """Record a message production event"""
        if PROMETHEUS_AVAILABLE and hasattr(self, 'prometheus_enabled') and self.prometheus_enabled:
            status = 'success' if success else 'failure'
            if self.kafka_producer_messages:
                self.kafka_producer_messages.labels(
                    service=self.service_name,
                    topic=topic,
                    status=status
                ).inc()

    def record_message_consumed(self, topic: str, success: bool):
        """Record a message consumption event"""
        if PROMETHEUS_AVAILABLE and hasattr(self, 'prometheus_enabled') and self.prometheus_enabled:
            status = 'success' if success else 'failure'
            if self.kafka_consumer_messages:
                self.kafka_consumer_messages.labels(
                    service=self.service_name,
                    topic=topic,
                    status=status
                ).inc()

    def update_connection_status(self, connected: bool):
        """Update connection status"""
        if PROMETHEUS_AVAILABLE and hasattr(self, 'prometheus_enabled') and self.prometheus_enabled:
            if self.kafka_connection_status:
                self.kafka_connection_status.labels(service=self.service_name).set(1 if connected else 0)

    def update_consumer_lag(self, topic: str, partition: int, lag: int):
        """Update consumer lag"""
        if PROMETHEUS_AVAILABLE and hasattr(self, 'prometheus_enabled') and self.prometheus_enabled:
            if self.kafka_lag:
                self.kafka_lag.labels(
                    service=self.service_name,
                    topic=topic,
                    partition=str(partition)
                ).set(lag)
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get all collected metrics"""
        with self.lock:
            return {
                'service_name': self.service_name,
                'timestamp': datetime.now().isoformat(),
                'operations': {name: metrics.to_dict() for name, metrics in self.metrics.items()},
                'custom_metrics': dict(self.custom_metrics)
            }
    
    def get_operation_metrics(self, operation_name: str) -> Optional[Dict[str, Any]]:
        """Get metrics for a specific operation"""
        with self.lock:
            if operation_name in self.metrics:
                return self.metrics[operation_name].to_dict()
            return None
    
    def reset_metrics(self):
        """Reset all metrics"""
        with self.lock:
            self.metrics.clear()
            self.custom_metrics.clear()
        logger.info("Metrics reset")
    
    def add_custom_metric(self, name: str, value: Any, labels: Optional[Dict[str, str]] = None):
        """Add a custom metric"""
        with self.lock:
            if labels:
                if name not in self.custom_metrics:
                    self.custom_metrics[name] = {}
                label_key = json.dumps(labels, sort_keys=True)
                self.custom_metrics[name][label_key] = {
                    'value': value,
                    'labels': labels,
                    'timestamp': datetime.now().isoformat()
                }
            else:
                self.custom_metrics[name] = {
                    'value': value,
                    'timestamp': datetime.now().isoformat()
                }

class HealthChecker:
    """Health checker for Kafka components"""
    
    def __init__(self, service_name: str):
        self.service_name = service_name
        self.health_checks: Dict[str, Callable] = {}
        self.last_check_results: Dict[str, Dict[str, Any]] = {}
        self.lock = threading.Lock()
    
    def register_health_check(self, name: str, check_func: Callable) -> None:
        """Register a health check function"""
        with self.lock:
            self.health_checks[name] = check_func
        logger.info(f"Registered health check: {name}")
    
    def run_health_checks(self) -> Dict[str, Any]:
        """Run all registered health checks"""
        results = {
            'service_name': self.service_name,
            'timestamp': datetime.now().isoformat(),
            'overall_status': 'healthy',
            'checks': {}
        }
        
        with self.lock:
            for name, check_func in self.health_checks.items():
                try:
                    start_time = time.time()
                    check_result = check_func()
                    duration = time.time() - start_time
                    
                    if isinstance(check_result, bool):
                        status = 'healthy' if check_result else 'unhealthy'
                        details = {}
                    elif isinstance(check_result, dict):
                        status = check_result.get('status', 'unknown')
                        details = check_result.get('details', {})
                    else:
                        status = 'unknown'
                        details = {'result': str(check_result)}
                    
                    results['checks'][name] = {
                        'status': status,
                        'duration': duration,
                        'details': details
                    }
                    
                    if status != 'healthy':
                        results['overall_status'] = 'unhealthy'
                        
                except Exception as e:
                    results['checks'][name] = {
                        'status': 'error',
                        'error': str(e),
                        'duration': 0
                    }
                    results['overall_status'] = 'unhealthy'
            
            self.last_check_results = results
        
        return results
    
    def get_last_results(self) -> Dict[str, Any]:
        """Get last health check results"""
        with self.lock:
            return self.last_check_results.copy()

class PerformanceTracker:
    """Tracks performance metrics over time"""
    
    def __init__(self, window_size: int = 1000):
        self.window_size = window_size
        self.data_points: deque = deque(maxlen=window_size)
        self.lock = threading.Lock()
    
    def add_data_point(self, value: float, timestamp: Optional[datetime] = None):
        """Add a performance data point"""
        if timestamp is None:
            timestamp = datetime.now()
        
        with self.lock:
            self.data_points.append({
                'value': value,
                'timestamp': timestamp
            })
    
    def get_statistics(self, time_window: Optional[timedelta] = None) -> Dict[str, Any]:
        """Get performance statistics"""
        with self.lock:
            if not self.data_points:
                return {'count': 0}
            
            # Filter by time window if specified
            if time_window:
                cutoff_time = datetime.now() - time_window
                filtered_points = [
                    dp for dp in self.data_points 
                    if dp['timestamp'] > cutoff_time
                ]
            else:
                filtered_points = list(self.data_points)
            
            if not filtered_points:
                return {'count': 0}
            
            values = [dp['value'] for dp in filtered_points]
            
            return {
                'count': len(values),
                'min': min(values),
                'max': max(values),
                'mean': sum(values) / len(values),
                'median': sorted(values)[len(values) // 2],
                'p95': sorted(values)[int(len(values) * 0.95)] if len(values) > 20 else max(values),
                'p99': sorted(values)[int(len(values) * 0.99)] if len(values) > 100 else max(values)
            }

# Global metrics collector instance
_metrics_collector: Optional[KafkaMetricsCollector] = None

def get_metrics_collector(service_name: str = "kafka-service") -> KafkaMetricsCollector:
    """Get global metrics collector instance"""
    global _metrics_collector
    if _metrics_collector is None:
        _metrics_collector = KafkaMetricsCollector(service_name)
    return _metrics_collector

def set_metrics_collector(collector: KafkaMetricsCollector):
    """Set global metrics collector instance"""
    global _metrics_collector
    _metrics_collector = collector
