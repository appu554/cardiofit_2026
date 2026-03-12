"""
SLA Monitor Service
Core SLA monitoring engine that evaluates compliance and manages violations
"""

import asyncio
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from collections import defaultdict, deque
import structlog
import statistics

from ..models.sla_models import (
    SLATarget,
    SLAMeasurement,
    SLAViolation,
    SLAAlert,
    SLAReport,
    SLAMetricSummary,
    SLAConfiguration,
    SLAStatus,
    SLASeverity,
    ServiceHealthStatus,
    SLADashboardData
)
from ..collectors.metrics_collector import MetricsCollector
from ..alerting.alert_manager import AlertManager

logger = structlog.get_logger()


class SLAMonitor:
    """
    Core SLA monitoring service that evaluates compliance and manages violations

    Features:
    - Real-time SLA compliance evaluation
    - Violation detection and tracking
    - Alert generation and management
    - Performance trend analysis
    - Comprehensive reporting
    """

    def __init__(
        self,
        metrics_collector: MetricsCollector,
        alert_manager: AlertManager,
        config: SLAConfiguration
    ):
        self.metrics_collector = metrics_collector
        self.alert_manager = alert_manager
        self.config = config

        # In-memory storage (in production, would use database)
        self._measurements: List[SLAMeasurement] = []
        self._violations: Dict[str, SLAViolation] = {}  # violation_id -> violation
        self._active_violations: Dict[str, SLAViolation] = {}  # target_id -> active violation
        self._alerts: List[SLAAlert] = []

        # Performance tracking
        self._measurement_history: Dict[str, deque] = defaultdict(lambda: deque(maxlen=1000))
        self._compliance_trends: Dict[str, deque] = defaultdict(lambda: deque(maxlen=100))

        # Monitoring tasks
        self._monitoring_task: Optional[asyncio.Task] = None
        self._cleanup_task: Optional[asyncio.Task] = None

        # Statistics
        self._evaluation_count = 0
        self._total_measurements = 0
        self._total_violations = 0

        logger.info("sla_monitor_initialized", config_id=config.config_id)

    async def start_monitoring(self):
        """
        Start SLA monitoring loops
        """
        if self._monitoring_task is None:
            self._monitoring_task = asyncio.create_task(self._monitoring_loop())

        if self._cleanup_task is None:
            self._cleanup_task = asyncio.create_task(self._cleanup_loop())

        logger.info("sla_monitoring_started")

    async def stop_monitoring(self):
        """
        Stop SLA monitoring loops
        """
        if self._monitoring_task:
            self._monitoring_task.cancel()
            try:
                await self._monitoring_task
            except asyncio.CancelledError:
                pass
            self._monitoring_task = None

        if self._cleanup_task:
            self._cleanup_task.cancel()
            try:
                await self._cleanup_task
            except asyncio.CancelledError:
                pass
            self._cleanup_task = None

        logger.info("sla_monitoring_stopped")

    async def _monitoring_loop(self):
        """
        Main monitoring loop that evaluates SLA compliance
        """
        while True:
            try:
                evaluation_start = time.perf_counter()

                # Get enabled targets
                enabled_targets = [t for t in self.config.targets if t.enabled]

                if enabled_targets:
                    # Collect metrics
                    measurements = await self.metrics_collector.collect_all_metrics(enabled_targets)
                    self._total_measurements += len(measurements)

                    # Evaluate SLA compliance
                    await self._evaluate_compliance(measurements)

                    # Store measurements
                    self._measurements.extend(measurements)

                    # Update trends
                    self._update_compliance_trends(measurements)

                evaluation_duration = (time.perf_counter() - evaluation_start) * 1000
                self._evaluation_count += 1

                logger.debug(
                    "sla_evaluation_completed",
                    measurements=len(measurements) if 'measurements' in locals() else 0,
                    evaluation_duration_ms=evaluation_duration,
                    total_evaluations=self._evaluation_count
                )

                # Wait for next evaluation
                await asyncio.sleep(self.config.default_evaluation_frequency_seconds)

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("sla_monitoring_loop_error", error=str(e))
                await asyncio.sleep(60)  # Wait longer on error

    async def _evaluate_compliance(self, measurements: List[SLAMeasurement]):
        """
        Evaluate SLA compliance for measurements
        """
        for measurement in measurements:
            try:
                # Get target for this measurement
                target = self._get_target_by_id(measurement.target_id)
                if not target:
                    continue

                # Check for violation
                if not measurement.is_compliant:
                    await self._handle_violation(measurement, target)
                else:
                    await self._handle_compliance_recovery(measurement, target)

                # Update measurement history
                self._measurement_history[measurement.target_id].append({
                    'timestamp': measurement.timestamp,
                    'value': measurement.measured_value,
                    'is_compliant': measurement.is_compliant,
                    'compliance_percentage': measurement.compliance_percentage
                })

            except Exception as e:
                logger.error(
                    "compliance_evaluation_error",
                    measurement_id=measurement.measurement_id,
                    error=str(e)
                )

    async def _handle_violation(self, measurement: SLAMeasurement, target: SLATarget):
        """
        Handle SLA violation
        """
        target_id = measurement.target_id

        if target_id in self._active_violations:
            # Extend existing violation
            violation = self._active_violations[target_id]
            violation.extend_violation(measurement)

            logger.debug(
                "sla_violation_extended",
                violation_id=violation.violation_id,
                consecutive_violations=violation.consecutive_violations,
                service_name=measurement.service_name,
                metric_type=measurement.metric_type
            )
        else:
            # Start new violation
            violation = SLAViolation(
                target_id=target_id,
                service_name=measurement.service_name,
                metric_type=measurement.metric_type,
                severity=target.severity,
                measured_value=measurement.measured_value,
                target_value=measurement.target_value,
                compliance_percentage=measurement.compliance_percentage,
                started_at=measurement.timestamp,
                metadata=measurement.metadata
            )

            self._active_violations[target_id] = violation
            self._violations[violation.violation_id] = violation
            self._total_violations += 1

            # Generate alert
            alert = await self._create_violation_alert(violation, "violation_start")
            if alert:
                self._alerts.append(alert)
                await self.alert_manager.send_alert(alert)

            logger.warning(
                "sla_violation_started",
                violation_id=violation.violation_id,
                service_name=measurement.service_name,
                metric_type=measurement.metric_type,
                measured_value=measurement.measured_value,
                target_value=measurement.target_value,
                severity=target.severity
            )

    async def _handle_compliance_recovery(self, measurement: SLAMeasurement, target: SLATarget):
        """
        Handle recovery from SLA violation
        """
        target_id = measurement.target_id

        if target_id in self._active_violations:
            violation = self._active_violations[target_id]

            # Check if we should resolve the violation (grace period)
            grace_period = timedelta(minutes=target.grace_period_minutes)
            if measurement.timestamp - violation.started_at >= grace_period:
                # Resolve violation
                violation.resolve()
                del self._active_violations[target_id]

                # Generate resolution alert
                alert = await self._create_violation_alert(violation, "violation_end")
                if alert:
                    self._alerts.append(alert)
                    await self.alert_manager.send_alert(alert)

                logger.info(
                    "sla_violation_resolved",
                    violation_id=violation.violation_id,
                    service_name=measurement.service_name,
                    metric_type=measurement.metric_type,
                    duration_minutes=violation.duration_minutes,
                    consecutive_violations=violation.consecutive_violations
                )

    async def _create_violation_alert(
        self,
        violation: SLAViolation,
        alert_type: str
    ) -> Optional[SLAAlert]:
        """
        Create alert for SLA violation
        """
        try:
            if alert_type == "violation_start":
                title = f"SLA Violation: {violation.service_name} - {violation.metric_type.value}"
                message = (
                    f"Service '{violation.service_name}' is violating SLA for {violation.metric_type.value}. "
                    f"Measured: {violation.measured_value:.2f}, "
                    f"Target: {violation.target_value:.2f}, "
                    f"Compliance: {violation.compliance_percentage:.1f}%"
                )
            else:  # violation_end
                title = f"SLA Violation Resolved: {violation.service_name} - {violation.metric_type.value}"
                message = (
                    f"SLA violation for '{violation.service_name}' has been resolved. "
                    f"Duration: {violation.duration_minutes:.1f} minutes, "
                    f"Consecutive violations: {violation.consecutive_violations}"
                )

            alert = SLAAlert(
                service_name=violation.service_name,
                metric_type=violation.metric_type,
                severity=violation.severity,
                alert_type=alert_type,
                title=title,
                message=message,
                violation_id=violation.violation_id,
                target_id=violation.target_id,
                metadata={
                    "measured_value": violation.measured_value,
                    "target_value": violation.target_value,
                    "compliance_percentage": violation.compliance_percentage
                }
            )

            if alert_type == "violation_end":
                alert.resolve()

            return alert

        except Exception as e:
            logger.error("alert_creation_error", error=str(e))
            return None

    def _update_compliance_trends(self, measurements: List[SLAMeasurement]):
        """
        Update compliance trend data
        """
        for measurement in measurements:
            key = f"{measurement.service_name}_{measurement.metric_type.value}"
            self._compliance_trends[key].append(measurement.compliance_percentage)

    async def _cleanup_loop(self):
        """
        Background cleanup loop for old data
        """
        while True:
            try:
                await asyncio.sleep(3600)  # Run every hour

                current_time = datetime.utcnow()

                # Clean up old measurements
                measurement_cutoff = current_time - timedelta(days=self.config.measurement_retention_days)
                self._measurements = [
                    m for m in self._measurements
                    if m.timestamp > measurement_cutoff
                ]

                # Clean up old violations
                violation_cutoff = current_time - timedelta(days=self.config.violation_retention_days)
                old_violation_ids = []
                for violation_id, violation in self._violations.items():
                    if violation.started_at < violation_cutoff:
                        old_violation_ids.append(violation_id)

                for violation_id in old_violation_ids:
                    del self._violations[violation_id]

                # Clean up old alerts
                alert_cutoff = current_time - timedelta(days=self.config.alert_retention_days)
                self._alerts = [
                    a for a in self._alerts
                    if a.triggered_at > alert_cutoff
                ]

                logger.debug(
                    "sla_data_cleanup_completed",
                    measurements_retained=len(self._measurements),
                    violations_retained=len(self._violations),
                    alerts_retained=len(self._alerts)
                )

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("cleanup_loop_error", error=str(e))

    def _get_target_by_id(self, target_id: str) -> Optional[SLATarget]:
        """
        Get SLA target by ID
        """
        for target in self.config.targets:
            if target.target_id == target_id:
                return target
        return None

    async def get_service_health_status(self, service_name: str) -> ServiceHealthStatus:
        """
        Get comprehensive health status for a service
        """
        current_time = datetime.utcnow()
        lookback_time = current_time - timedelta(hours=1)

        # Get recent measurements for service
        recent_measurements = [
            m for m in self._measurements
            if m.service_name == service_name and m.timestamp > lookback_time
        ]

        if not recent_measurements:
            return ServiceHealthStatus(
                service_name=service_name,
                overall_status=SLAStatus.NO_DATA,
                uptime_percentage=0.0
            )

        # Calculate overall compliance
        compliant_count = sum(1 for m in recent_measurements if m.is_compliant)
        total_count = len(recent_measurements)
        uptime_percentage = (compliant_count / total_count) * 100.0 if total_count > 0 else 0.0

        # Get performance metrics
        response_time_measurements = [
            m for m in recent_measurements
            if m.metric_type.value == "response_time"
        ]

        response_time_p95 = None
        if response_time_measurements:
            response_times = [m.measured_value for m in response_time_measurements]
            response_time_p95 = statistics.quantiles(response_times, n=20)[18]  # 95th percentile

        error_rate_measurements = [
            m for m in recent_measurements
            if m.metric_type.value == "error_rate"
        ]

        error_rate = None
        if error_rate_measurements:
            error_rates = [m.measured_value for m in error_rate_measurements]
            error_rate = statistics.mean(error_rates)

        # Get active violations
        active_violations = [
            v for v in self._active_violations.values()
            if v.service_name == service_name and v.is_active
        ]

        critical_violations = [
            v for v in active_violations
            if v.severity == SLASeverity.CRITICAL
        ]

        # Determine overall status
        if len(critical_violations) > 0:
            overall_status = SLAStatus.CRITICAL_VIOLATION
        elif len(active_violations) > 0:
            overall_status = SLAStatus.VIOLATION
        elif uptime_percentage >= 99.5:
            overall_status = SLAStatus.COMPLIANT
        elif uptime_percentage >= 95.0:
            overall_status = SLAStatus.WARNING
        else:
            overall_status = SLAStatus.VIOLATION

        # Last violation time
        last_violation_time = None
        if active_violations:
            last_violation_time = max(v.started_at for v in active_violations)

        return ServiceHealthStatus(
            service_name=service_name,
            overall_status=overall_status,
            uptime_percentage=uptime_percentage,
            response_time_p95_ms=response_time_p95,
            error_rate_percentage=error_rate,
            active_violations=len(active_violations),
            critical_violations=len(critical_violations),
            last_violation_time=last_violation_time
        )

    async def generate_sla_report(
        self,
        service_name: Optional[str] = None,
        period_hours: int = 24
    ) -> SLAReport:
        """
        Generate comprehensive SLA report
        """
        end_time = datetime.utcnow()
        start_time = end_time - timedelta(hours=period_hours)

        # Filter measurements
        if service_name:
            measurements = [
                m for m in self._measurements
                if m.service_name == service_name and start_time <= m.timestamp <= end_time
            ]
        else:
            measurements = [
                m for m in self._measurements
                if start_time <= m.timestamp <= end_time
            ]

        # Calculate overall metrics
        total_measurements = len(measurements)
        compliant_measurements = sum(1 for m in measurements if m.is_compliant)
        overall_compliance = (compliant_measurements / total_measurements * 100.0) if total_measurements > 0 else 0.0

        # Calculate uptime (availability measurements only)
        availability_measurements = [m for m in measurements if m.metric_type.value == "availability"]
        uptime_percentage = 0.0
        if availability_measurements:
            uptime_measurements = sum(m.measured_value for m in availability_measurements)
            uptime_percentage = uptime_measurements / len(availability_measurements)

        # Group by metric type
        metrics_by_type = defaultdict(list)
        for measurement in measurements:
            metrics_by_type[measurement.metric_type].append(measurement)

        # Generate metric summaries
        metric_summaries = []
        for metric_type, metric_measurements in metrics_by_type.items():
            if not metric_measurements:
                continue

            compliant = sum(1 for m in metric_measurements if m.is_compliant)
            compliance_pct = (compliant / len(metric_measurements)) * 100.0

            values = [m.measured_value for m in metric_measurements]
            avg_value = statistics.mean(values)
            p95_value = statistics.quantiles(values, n=20)[18] if len(values) > 1 else values[0]
            p99_value = statistics.quantiles(values, n=100)[98] if len(values) > 1 else values[0]

            # Get target info (assume same target for metric type)
            target_value = metric_measurements[0].target_value
            unit = "unknown"  # Would get from target configuration

            # Count violations
            violations = [
                v for v in self._violations.values()
                if v.metric_type == metric_type and
                start_time <= v.started_at <= end_time
            ]

            longest_violation = 0.0
            if violations:
                longest_violation = max(
                    v.duration_minutes or 0.0 for v in violations
                )

            summary = SLAMetricSummary(
                metric_type=metric_type,
                target_value=target_value,
                unit=unit,
                measurements_count=len(metric_measurements),
                compliant_measurements=compliant,
                compliance_percentage=compliance_pct,
                average_measured_value=avg_value,
                p95_measured_value=p95_value,
                p99_measured_value=p99_value,
                violations_count=len(violations),
                longest_violation_minutes=longest_violation
            )

            metric_summaries.append(summary)

        # Count violations
        period_violations = [
            v for v in self._violations.values()
            if start_time <= v.started_at <= end_time
        ]

        critical_violations = [
            v for v in period_violations
            if v.severity == SLASeverity.CRITICAL
        ]

        avg_violation_duration = 0.0
        if period_violations:
            durations = [v.duration_minutes or 0.0 for v in period_violations]
            avg_violation_duration = statistics.mean(durations)

        # Determine trend
        compliance_trend = "stable"
        trend_confidence = 0.0

        if service_name:
            service_name_to_use = service_name
        else:
            service_name_to_use = "all_services"

        return SLAReport(
            service_name=service_name_to_use,
            report_period_start=start_time,
            report_period_end=end_time,
            total_measurements=total_measurements,
            compliant_measurements=compliant_measurements,
            overall_compliance_percentage=overall_compliance,
            uptime_percentage=uptime_percentage,
            metric_summaries=metric_summaries,
            total_violations=len(period_violations),
            critical_violations=len(critical_violations),
            average_violation_duration_minutes=avg_violation_duration,
            compliance_trend=compliance_trend,
            trend_confidence=trend_confidence
        )

    async def get_dashboard_data(self, period_hours: int = 24) -> SLADashboardData:
        """
        Get data for SLA monitoring dashboard
        """
        end_time = datetime.utcnow()
        start_time = end_time - timedelta(hours=period_hours)

        # Get unique services
        services = list(set(t.service_name for t in self.config.targets))

        # Get service health statuses
        service_statuses = []
        healthy_count = 0

        for service_name in services:
            status = await self.get_service_health_status(service_name)
            service_statuses.append(status)
            if status.is_healthy:
                healthy_count += 1

        # Get recent violations
        recent_violations = [
            v for v in self._violations.values()
            if start_time <= v.started_at <= end_time
        ]

        # Get active alerts
        active_alerts = [a for a in self._alerts if a.is_active]

        # Calculate system uptime
        all_measurements = [
            m for m in self._measurements
            if start_time <= m.timestamp <= end_time
        ]

        system_uptime = 0.0
        if all_measurements:
            compliant_count = sum(1 for m in all_measurements if m.is_compliant)
            system_uptime = (compliant_count / len(all_measurements)) * 100.0

        # Generate compliance trends (hourly buckets)
        compliance_trends = {}
        for service_name in services:
            trends = []
            for i in range(24):  # Last 24 hours
                hour_start = end_time - timedelta(hours=24-i)
                hour_end = hour_start + timedelta(hours=1)

                hour_measurements = [
                    m for m in self._measurements
                    if (m.service_name == service_name and
                        hour_start <= m.timestamp < hour_end)
                ]

                if hour_measurements:
                    compliant = sum(1 for m in hour_measurements if m.is_compliant)
                    compliance_pct = (compliant / len(hour_measurements)) * 100.0
                    trends.append(compliance_pct)
                else:
                    trends.append(100.0)  # No data assumes compliance

            compliance_trends[service_name] = trends

        return SLADashboardData(
            period_start=start_time,
            period_end=end_time,
            total_services=len(services),
            healthy_services=healthy_count,
            services_with_violations=len(services) - healthy_count,
            system_uptime_percentage=system_uptime,
            service_statuses=service_statuses,
            recent_violations=recent_violations,
            active_alerts=active_alerts,
            compliance_trends=compliance_trends
        )

    def get_monitoring_statistics(self) -> Dict[str, Any]:
        """
        Get monitoring system statistics
        """
        return {
            "total_evaluations": self._evaluation_count,
            "total_measurements": self._total_measurements,
            "total_violations": self._total_violations,
            "active_violations": len(self._active_violations),
            "total_alerts": len(self._alerts),
            "active_alerts": len([a for a in self._alerts if a.is_active]),
            "configured_targets": len(self.config.targets),
            "enabled_targets": len([t for t in self.config.targets if t.enabled]),
            "monitoring_uptime": datetime.utcnow().isoformat()
        }