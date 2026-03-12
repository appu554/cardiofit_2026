"""
Workflow Engine Monitoring and Metrics for Calculate > Validate > Commit Flow

Provides comprehensive monitoring for the medication workflow including:
- Phase performance tracking (Calculate/Validate/Commit timing)
- Success/failure rates
- Safety Gateway validation metrics
- Proposal persistence metrics
- GraphQL orchestration performance
"""

import logging
import time
import threading
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, Optional, List, Callable
from dataclasses import dataclass, asdict
from contextlib import asynccontextmanager
from collections import defaultdict, deque
import json

try:
    # Prometheus metrics support
    from prometheus_client import (
        Counter, Histogram, Gauge, Summary, CollectorRegistry,
        generate_latest, multiprocess, CONTENT_TYPE_LATEST
    )
    PROMETHEUS_AVAILABLE = True
except ImportError:
    PROMETHEUS_AVAILABLE = False

logger = logging.getLogger(__name__)

@dataclass
class WorkflowMetrics:
    """Workflow performance metrics"""
    total_requests: int = 0
    successful_workflows: int = 0
    warning_workflows: int = 0
    failed_workflows: int = 0
    avg_processing_time_ms: float = 0.0
    performance_targets_met: float = 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

@dataclass
class PhaseMetrics:
    """Individual phase performance metrics"""
    phase_name: str
    average_time_ms: float = 0.0
    success_rate: float = 1.0
    error_rate: float = 0.0
    performance_target_ms: float = 0.0
    meets_target: bool = True
    total_executions: int = 0
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

@dataclass
class ValidationMetrics:
    """Safety Gateway validation metrics"""
    total_validations: int = 0
    safe_validations: int = 0
    warning_validations: int = 0
    unsafe_validations: int = 0
    validation_errors: int = 0
    avg_validation_time_ms: float = 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


class WorkflowMetricsCollector:
    """
    Comprehensive metrics collection for Calculate > Validate > Commit workflow
    
    Features:
    - Prometheus metrics integration
    - Time-series data collection
    - Performance target tracking
    - Alert generation
    - Health status monitoring
    """
    
    def __init__(self, enable_prometheus: bool = True):
        """Initialize metrics collector"""
        self.enable_prometheus = enable_prometheus and PROMETHEUS_AVAILABLE
        self.metrics_registry = None
        
        # Performance targets (from Phase 4 spec)
        self.performance_targets = {
            'calculate': 175,  # ms
            'validate': 100,   # ms 
            'commit': 50,      # ms
            'total': 325       # ms
        }
        
        # Metrics storage
        self.workflow_metrics = WorkflowMetrics()
        self.phase_metrics = {
            'calculate': PhaseMetrics('calculate', performance_target_ms=175),
            'validate': PhaseMetrics('validate', performance_target_ms=100),
            'commit': PhaseMetrics('commit', performance_target_ms=50)
        }
        self.validation_metrics = ValidationMetrics()
        
        # Time series data (last 1000 data points)
        self.time_series = defaultdict(lambda: deque(maxlen=1000))
        
        # Active measurements
        self.active_measurements = {}
        
        # Thread safety
        self._lock = threading.Lock()
        
        # Initialize Prometheus metrics if available
        if self.enable_prometheus:
            self._init_prometheus_metrics()
        
        logger.info("Workflow Metrics Collector initialized (Prometheus: %s)", self.enable_prometheus)
    
    def _init_prometheus_metrics(self):
        """Initialize Prometheus metrics"""
        try:
            self.metrics_registry = CollectorRegistry()
            
            # Workflow counters
            self.workflow_requests_total = Counter(
                'workflow_requests_total',
                'Total workflow requests',
                ['status', 'workflow_type'],
                registry=self.metrics_registry
            )
            
            # Phase timing histograms
            self.phase_duration_seconds = Histogram(
                'workflow_phase_duration_seconds',
                'Phase execution duration',
                ['phase', 'status'],
                buckets=[0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0],
                registry=self.metrics_registry
            )
            
            # Validation metrics
            self.validation_requests_total = Counter(
                'safety_gateway_validations_total',
                'Total Safety Gateway validation requests',
                ['verdict', 'engine'],
                registry=self.metrics_registry
            )
            
            self.validation_duration_seconds = Histogram(
                'safety_gateway_validation_duration_seconds',
                'Safety Gateway validation duration',
                ['engine'],
                buckets=[0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5],
                registry=self.metrics_registry
            )
            
            # Proposal metrics
            self.proposal_operations_total = Counter(
                'proposal_operations_total',
                'Total proposal database operations',
                ['operation', 'status'],
                registry=self.metrics_registry
            )
            
            # Performance target gauge
            self.performance_target_adherence = Gauge(
                'workflow_performance_target_adherence_ratio',
                'Ratio of workflows meeting performance targets',
                ['phase'],
                registry=self.metrics_registry
            )
            
            # Active workflow gauge
            self.active_workflows = Gauge(
                'workflow_active_total',
                'Currently active workflows',
                ['phase'],
                registry=self.metrics_registry
            )
            
            logger.info("✅ Prometheus metrics initialized")
            
        except Exception as e:
            logger.error("Failed to initialize Prometheus metrics: %s", e)
            self.enable_prometheus = False
    
    @asynccontextmanager
    async def measure_workflow_phase(
        self, 
        phase: str, 
        correlation_id: str,
        metadata: Optional[Dict[str, Any]] = None
    ):
        """Context manager to measure workflow phase execution"""
        start_time = time.time()
        measurement_id = f"{correlation_id}_{phase}_{start_time}"
        
        # Store measurement start
        with self._lock:
            self.active_measurements[measurement_id] = {
                'phase': phase,
                'correlation_id': correlation_id,
                'start_time': start_time,
                'metadata': metadata or {}
            }
            
            # Update active workflows gauge
            if self.enable_prometheus:
                self.active_workflows.labels(phase=phase).inc()
        
        try:
            yield measurement_id
            status = 'success'
        except Exception as e:
            status = 'error'
            logger.error("Phase %s failed for correlation %s: %s", phase, correlation_id, e)
            raise
        finally:
            end_time = time.time()
            duration_ms = (end_time - start_time) * 1000
            
            # Record phase completion
            await self._record_phase_completion(
                phase, correlation_id, duration_ms, status, metadata
            )
            
            # Clean up measurement
            with self._lock:
                self.active_measurements.pop(measurement_id, None)
                if self.enable_prometheus:
                    self.active_workflows.labels(phase=phase).dec()
    
    async def _record_phase_completion(
        self,
        phase: str,
        correlation_id: str,
        duration_ms: float,
        status: str,
        metadata: Optional[Dict[str, Any]]
    ):
        """Record phase completion metrics"""
        timestamp = datetime.now(timezone.utc)
        
        with self._lock:
            # Update phase metrics
            if phase in self.phase_metrics:
                phase_metric = self.phase_metrics[phase]
                phase_metric.total_executions += 1
                
                # Update running average
                total_time = (phase_metric.average_time_ms * (phase_metric.total_executions - 1)) + duration_ms
                phase_metric.average_time_ms = total_time / phase_metric.total_executions
                
                # Update success/error rates
                if status == 'success':
                    phase_metric.success_rate = (
                        (phase_metric.success_rate * (phase_metric.total_executions - 1)) + 1.0
                    ) / phase_metric.total_executions
                else:
                    phase_metric.error_rate = (
                        (phase_metric.error_rate * (phase_metric.total_executions - 1)) + 1.0
                    ) / phase_metric.total_executions
                
                # Check performance target
                phase_metric.meets_target = phase_metric.average_time_ms <= phase_metric.performance_target_ms
            
            # Store time series data
            self.time_series[f"{phase}_duration_ms"].append({
                'timestamp': timestamp.isoformat(),
                'value': duration_ms,
                'status': status,
                'correlation_id': correlation_id
            })
        
        # Update Prometheus metrics
        if self.enable_prometheus:
            self.phase_duration_seconds.labels(
                phase=phase, 
                status=status
            ).observe(duration_ms / 1000.0)
            
            # Update performance adherence
            if phase in self.phase_metrics:
                adherence_ratio = 1.0 if duration_ms <= self.performance_targets[phase] else 0.0
                self.performance_target_adherence.labels(phase=phase).set(adherence_ratio)
        
        logger.info(
            "Phase %s completed: %s (%.2fms) - Target: %sms",
            phase, status, duration_ms, self.performance_targets.get(phase, 'N/A')
        )
    
    async def record_workflow_completion(
        self,
        correlation_id: str,
        status: str,  # 'success', 'warning', 'error'
        total_duration_ms: float,
        workflow_type: str = 'medication_request'
    ):
        """Record complete workflow execution"""
        with self._lock:
            self.workflow_metrics.total_requests += 1
            
            if status == 'success':
                self.workflow_metrics.successful_workflows += 1
            elif status == 'warning':
                self.workflow_metrics.warning_workflows += 1
            else:
                self.workflow_metrics.failed_workflows += 1
            
            # Update average processing time
            total_time = (
                self.workflow_metrics.avg_processing_time_ms * (self.workflow_metrics.total_requests - 1)
            ) + total_duration_ms
            self.workflow_metrics.avg_processing_time_ms = total_time / self.workflow_metrics.total_requests
            
            # Update performance target adherence
            meets_target = total_duration_ms <= self.performance_targets['total']
            successful_and_on_time = self.workflow_metrics.successful_workflows
            if meets_target and status == 'success':
                successful_and_on_time += 1
            
            self.workflow_metrics.performance_targets_met = (
                successful_and_on_time / self.workflow_metrics.total_requests
            ) * 100.0
        
        # Update Prometheus metrics
        if self.enable_prometheus:
            self.workflow_requests_total.labels(
                status=status,
                workflow_type=workflow_type
            ).inc()
        
        logger.info(
            "Workflow %s completed: %s (%.2fms total) - Performance targets met: %.1f%%",
            correlation_id, status, total_duration_ms, self.workflow_metrics.performance_targets_met
        )
    
    async def record_validation_result(
        self,
        correlation_id: str,
        verdict: str,  # 'SAFE', 'WARNING', 'UNSAFE', 'ERROR'
        duration_ms: float,
        engine: str = 'comprehensive'
    ):
        """Record Safety Gateway validation result"""
        with self._lock:
            self.validation_metrics.total_validations += 1
            
            if verdict == 'SAFE':
                self.validation_metrics.safe_validations += 1
            elif verdict == 'WARNING':
                self.validation_metrics.warning_validations += 1
            elif verdict == 'UNSAFE':
                self.validation_metrics.unsafe_validations += 1
            else:
                self.validation_metrics.validation_errors += 1
            
            # Update average validation time
            total_time = (
                self.validation_metrics.avg_validation_time_ms * (self.validation_metrics.total_validations - 1)
            ) + duration_ms
            self.validation_metrics.avg_validation_time_ms = total_time / self.validation_metrics.total_validations
        
        # Update Prometheus metrics
        if self.enable_prometheus:
            self.validation_requests_total.labels(
                verdict=verdict.lower(),
                engine=engine
            ).inc()
            
            self.validation_duration_seconds.labels(engine=engine).observe(duration_ms / 1000.0)
        
        logger.info("Validation completed: %s verdict in %.2fms", verdict, duration_ms)
    
    async def record_proposal_operation(
        self,
        operation: str,  # 'create', 'update', 'commit'
        status: str,     # 'success', 'error'
        duration_ms: Optional[float] = None
    ):
        """Record proposal database operation"""
        if self.enable_prometheus:
            self.proposal_operations_total.labels(
                operation=operation,
                status=status
            ).inc()
        
        if duration_ms:
            logger.debug("Proposal %s operation: %s (%.2fms)", operation, status, duration_ms)
    
    def get_current_metrics(self) -> Dict[str, Any]:
        """Get current metrics snapshot"""
        with self._lock:
            return {
                'timestamp': datetime.now(timezone.utc).isoformat(),
                'workflow_metrics': self.workflow_metrics.to_dict(),
                'phase_metrics': {
                    name: metrics.to_dict()
                    for name, metrics in self.phase_metrics.items()
                },
                'validation_metrics': self.validation_metrics.to_dict(),
                'performance_targets': self.performance_targets,
                'active_measurements': len(self.active_measurements)
            }
    
    def get_time_series_data(
        self,
        metric_name: str,
        hours: int = 1
    ) -> List[Dict[str, Any]]:
        """Get time series data for a specific metric"""
        cutoff_time = datetime.now(timezone.utc) - timedelta(hours=hours)
        
        with self._lock:
            series = self.time_series.get(metric_name, [])
            return [
                point for point in series
                if datetime.fromisoformat(point['timestamp']) > cutoff_time
            ]
    
    def get_health_status(self) -> Dict[str, Any]:
        """Get monitoring system health status"""
        with self._lock:
            # Calculate health indicators
            total_requests = self.workflow_metrics.total_requests
            error_rate = 0.0
            if total_requests > 0:
                error_rate = (self.workflow_metrics.failed_workflows / total_requests) * 100.0
            
            # Determine overall health
            if error_rate > 10.0:
                status = 'unhealthy'
            elif error_rate > 5.0:
                status = 'degraded'
            else:
                status = 'healthy'
            
            return {
                'status': status,
                'error_rate_percent': error_rate,
                'performance_targets_met_percent': self.workflow_metrics.performance_targets_met,
                'avg_processing_time_ms': self.workflow_metrics.avg_processing_time_ms,
                'total_requests': total_requests,
                'prometheus_enabled': self.enable_prometheus,
                'active_measurements': len(self.active_measurements),
                'last_updated': datetime.now(timezone.utc).isoformat()
            }
    
    def get_alerts(self) -> List[Dict[str, Any]]:
        """Generate alerts based on current metrics"""
        alerts = []
        
        with self._lock:
            # Check performance targets
            for phase_name, metrics in self.phase_metrics.items():
                if not metrics.meets_target and metrics.total_executions > 10:
                    alerts.append({
                        'type': 'performance_target_miss',
                        'severity': 'warning',
                        'phase': phase_name,
                        'message': f"Phase {phase_name} missing performance target: {metrics.average_time_ms:.1f}ms > {metrics.performance_target_ms}ms",
                        'current_avg': metrics.average_time_ms,
                        'target': metrics.performance_target_ms
                    })
            
            # Check error rates
            total_requests = self.workflow_metrics.total_requests
            if total_requests > 0:
                error_rate = (self.workflow_metrics.failed_workflows / total_requests) * 100.0
                if error_rate > 5.0:
                    severity = 'critical' if error_rate > 10.0 else 'warning'
                    alerts.append({
                        'type': 'high_error_rate',
                        'severity': severity,
                        'message': f"High workflow error rate: {error_rate:.1f}%",
                        'error_rate': error_rate,
                        'failed_workflows': self.workflow_metrics.failed_workflows,
                        'total_workflows': total_requests
                    })
            
            # Check validation errors
            if self.validation_metrics.total_validations > 0:
                validation_error_rate = (
                    self.validation_metrics.validation_errors / self.validation_metrics.total_validations
                ) * 100.0
                if validation_error_rate > 2.0:
                    alerts.append({
                        'type': 'validation_error_rate',
                        'severity': 'warning',
                        'message': f"High validation error rate: {validation_error_rate:.1f}%",
                        'validation_error_rate': validation_error_rate
                    })
        
        return alerts
    
    def export_prometheus_metrics(self) -> str:
        """Export Prometheus metrics in text format"""
        if not self.enable_prometheus or not self.metrics_registry:
            return "# Prometheus metrics not available\n"
        
        try:
            return generate_latest(self.metrics_registry)
        except Exception as e:
            logger.error("Failed to export Prometheus metrics: %s", e)
            return f"# Error exporting metrics: {e}\n"


# Global metrics collector instance
_metrics_collector: Optional[WorkflowMetricsCollector] = None

def get_metrics_collector() -> WorkflowMetricsCollector:
    """Get global metrics collector instance"""
    global _metrics_collector
    if _metrics_collector is None:
        _metrics_collector = WorkflowMetricsCollector()
    return _metrics_collector

def init_metrics_collector(enable_prometheus: bool = True) -> WorkflowMetricsCollector:
    """Initialize global metrics collector"""
    global _metrics_collector
    _metrics_collector = WorkflowMetricsCollector(enable_prometheus=enable_prometheus)
    return _metrics_collector