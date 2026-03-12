"""
Clinical Monitoring and Metrics Service.
Implements clinical performance metrics, real-time monitoring, safety metrics collection, and SLA violation alerting.
"""
import logging
import asyncio
import time
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
from collections import defaultdict, deque
import json

logger = logging.getLogger(__name__)


@dataclass
class ClinicalMetric:
    """Represents a clinical performance metric."""
    metric_id: str
    metric_name: str
    metric_type: str  # counter, gauge, histogram, timer
    value: float
    unit: str
    timestamp: datetime
    tags: Dict[str, str] = field(default_factory=dict)
    safety_critical: bool = False
    threshold_warning: Optional[float] = None
    threshold_critical: Optional[float] = None


@dataclass
class SLAViolation:
    """Represents an SLA violation event."""
    violation_id: str
    workflow_id: str
    workflow_type: str
    sla_type: str  # response_time, completion_time, safety_check_time
    expected_value: float
    actual_value: float
    severity: str  # warning, critical
    timestamp: datetime
    patient_id: Optional[str] = None
    provider_id: Optional[str] = None
    details: Dict[str, Any] = field(default_factory=dict)


@dataclass
class SafetyAlert:
    """Represents a safety-related alert."""
    alert_id: str
    alert_type: str
    severity: str  # low, medium, high, critical
    message: str
    workflow_id: Optional[str] = None
    patient_id: Optional[str] = None
    timestamp: datetime
    acknowledged: bool = False
    resolved: bool = False
    details: Dict[str, Any] = field(default_factory=dict)


class ClinicalMonitoringService:
    """
    Service for clinical performance monitoring and metrics collection.
    
    Features:
    - Real-time clinical performance metrics
    - SLA violation detection and alerting
    - Safety metrics collection
    - Clinical dashboard data aggregation
    - Automated alerting for critical issues
    - Historical trend analysis
    """
    
    def __init__(self):
        self.metrics: Dict[str, ClinicalMetric] = {}
        self.metric_history: Dict[str, deque] = defaultdict(lambda: deque(maxlen=1000))
        self.sla_violations: List[SLAViolation] = []
        self.safety_alerts: List[SafetyAlert] = []
        self.active_workflows: Dict[str, Dict[str, Any]] = {}
        
        # Performance thresholds
        self.sla_thresholds = {
            "pessimistic_workflow_ms": 250,
            "optimistic_workflow_ms": 150,
            "digital_reflex_arc_ms": 100,
            "safety_validation_ms": 200,
            "medication_proposal_ms": 500,
            "emergency_response_ms": 60
        }
        
        # Safety thresholds
        self.safety_thresholds = {
            "unsafe_decision_rate": 0.01,  # 1% max
            "timeout_rate": 0.05,  # 5% max
            "error_rate": 0.02,  # 2% max
            "compensation_rate": 0.10  # 10% max
        }
        
        # Start monitoring
        asyncio.create_task(self._start_monitoring())
        
        logger.info("✅ Clinical Monitoring Service initialized")
    
    async def record_workflow_start(
        self,
        workflow_id: str,
        workflow_type: str,
        execution_pattern: str,
        patient_id: Optional[str] = None,
        provider_id: Optional[str] = None
    ):
        """Record the start of a clinical workflow."""
        self.active_workflows[workflow_id] = {
            "workflow_type": workflow_type,
            "execution_pattern": execution_pattern,
            "patient_id": patient_id,
            "provider_id": provider_id,
            "started_at": datetime.utcnow(),
            "status": "running"
        }
        
        # Record workflow start metric
        await self.record_metric(
            metric_id=f"workflow_started_{workflow_type}",
            metric_name=f"Workflow Started - {workflow_type}",
            metric_type="counter",
            value=1,
            unit="count",
            tags={
                "workflow_type": workflow_type,
                "execution_pattern": execution_pattern,
                "patient_id": patient_id or "unknown",
                "provider_id": provider_id or "unknown"
            }
        )
    
    async def record_workflow_completion(
        self,
        workflow_id: str,
        status: str,
        execution_time_ms: float,
        sla_compliance: bool,
        safety_validation_result: Optional[Dict[str, Any]] = None
    ):
        """Record the completion of a clinical workflow."""
        if workflow_id not in self.active_workflows:
            logger.warning(f"⚠️ Workflow completion recorded for unknown workflow: {workflow_id}")
            return
        
        workflow_info = self.active_workflows[workflow_id]
        workflow_info["status"] = status
        workflow_info["completed_at"] = datetime.utcnow()
        workflow_info["execution_time_ms"] = execution_time_ms
        workflow_info["sla_compliance"] = sla_compliance
        
        # Record completion metrics
        await self.record_metric(
            metric_id=f"workflow_completed_{workflow_info['workflow_type']}",
            metric_name=f"Workflow Completed - {workflow_info['workflow_type']}",
            metric_type="counter",
            value=1,
            unit="count",
            tags={
                "workflow_type": workflow_info["workflow_type"],
                "execution_pattern": workflow_info["execution_pattern"],
                "status": status,
                "sla_compliance": str(sla_compliance)
            }
        )
        
        # Record execution time
        await self.record_metric(
            metric_id=f"workflow_execution_time_{workflow_info['workflow_type']}",
            metric_name=f"Workflow Execution Time - {workflow_info['workflow_type']}",
            metric_type="histogram",
            value=execution_time_ms,
            unit="milliseconds",
            tags={
                "workflow_type": workflow_info["workflow_type"],
                "execution_pattern": workflow_info["execution_pattern"]
            }
        )
        
        # Check for SLA violations
        if not sla_compliance:
            await self._record_sla_violation(workflow_id, workflow_info, execution_time_ms)
        
        # Record safety metrics if available
        if safety_validation_result:
            await self._record_safety_metrics(workflow_id, safety_validation_result)
        
        # Clean up completed workflow
        del self.active_workflows[workflow_id]
    
    async def record_metric(
        self,
        metric_id: str,
        metric_name: str,
        metric_type: str,
        value: float,
        unit: str,
        tags: Optional[Dict[str, str]] = None,
        safety_critical: bool = False,
        threshold_warning: Optional[float] = None,
        threshold_critical: Optional[float] = None
    ):
        """Record a clinical performance metric."""
        metric = ClinicalMetric(
            metric_id=metric_id,
            metric_name=metric_name,
            metric_type=metric_type,
            value=value,
            unit=unit,
            timestamp=datetime.utcnow(),
            tags=tags or {},
            safety_critical=safety_critical,
            threshold_warning=threshold_warning,
            threshold_critical=threshold_critical
        )
        
        # Store current metric
        self.metrics[metric_id] = metric
        
        # Store in history
        self.metric_history[metric_id].append(metric)
        
        # Check thresholds
        await self._check_metric_thresholds(metric)
        
        logger.debug(f"📊 Recorded metric: {metric_name} = {value} {unit}")
    
    async def record_safety_alert(
        self,
        alert_type: str,
        severity: str,
        message: str,
        workflow_id: Optional[str] = None,
        patient_id: Optional[str] = None,
        details: Optional[Dict[str, Any]] = None
    ) -> str:
        """Record a safety-related alert."""
        alert_id = f"safety_alert_{int(time.time() * 1000)}"
        
        alert = SafetyAlert(
            alert_id=alert_id,
            alert_type=alert_type,
            severity=severity,
            message=message,
            workflow_id=workflow_id,
            patient_id=patient_id,
            timestamp=datetime.utcnow(),
            details=details or {}
        )
        
        self.safety_alerts.append(alert)
        
        # Record safety alert metric
        await self.record_metric(
            metric_id=f"safety_alert_{alert_type}",
            metric_name=f"Safety Alert - {alert_type}",
            metric_type="counter",
            value=1,
            unit="count",
            tags={
                "alert_type": alert_type,
                "severity": severity,
                "workflow_id": workflow_id or "unknown",
                "patient_id": patient_id or "unknown"
            },
            safety_critical=True
        )
        
        logger.warning(f"🚨 Safety alert recorded: {alert_type} - {message}")
        
        # Send immediate notification for critical alerts
        if severity == "critical":
            await self._send_critical_alert_notification(alert)
        
        return alert_id
    
    async def get_dashboard_data(self) -> Dict[str, Any]:
        """Get real-time dashboard data."""
        current_time = datetime.utcnow()
        
        # Active workflows
        active_count = len(self.active_workflows)
        
        # Recent metrics (last hour)
        recent_metrics = {}
        for metric_id, history in self.metric_history.items():
            recent_values = [
                m for m in history 
                if (current_time - m.timestamp).total_seconds() < 3600
            ]
            if recent_values:
                recent_metrics[metric_id] = {
                    "current_value": recent_values[-1].value,
                    "average": sum(m.value for m in recent_values) / len(recent_values),
                    "count": len(recent_values)
                }
        
        # SLA compliance
        recent_violations = [
            v for v in self.sla_violations
            if (current_time - v.timestamp).total_seconds() < 3600
        ]
        
        # Safety alerts
        unresolved_alerts = [a for a in self.safety_alerts if not a.resolved]
        critical_alerts = [a for a in unresolved_alerts if a.severity == "critical"]
        
        return {
            "timestamp": current_time.isoformat(),
            "active_workflows": active_count,
            "recent_metrics": recent_metrics,
            "sla_violations": {
                "recent_count": len(recent_violations),
                "total_count": len(self.sla_violations)
            },
            "safety_alerts": {
                "unresolved_count": len(unresolved_alerts),
                "critical_count": len(critical_alerts),
                "recent_alerts": [
                    {
                        "alert_id": a.alert_id,
                        "alert_type": a.alert_type,
                        "severity": a.severity,
                        "message": a.message,
                        "timestamp": a.timestamp.isoformat()
                    }
                    for a in unresolved_alerts[-10:]  # Last 10 alerts
                ]
            },
            "performance_summary": await self._get_performance_summary()
        }
    
    async def get_workflow_metrics(self, workflow_type: Optional[str] = None) -> Dict[str, Any]:
        """Get workflow-specific metrics."""
        workflow_metrics = {}
        
        for metric_id, history in self.metric_history.items():
            if workflow_type and workflow_type not in metric_id:
                continue
            
            if history:
                latest = history[-1]
                workflow_metrics[metric_id] = {
                    "metric_name": latest.metric_name,
                    "current_value": latest.value,
                    "unit": latest.unit,
                    "timestamp": latest.timestamp.isoformat(),
                    "tags": latest.tags,
                    "safety_critical": latest.safety_critical
                }
        
        return workflow_metrics
    
    async def _record_sla_violation(
        self,
        workflow_id: str,
        workflow_info: Dict[str, Any],
        execution_time_ms: float
    ):
        """Record an SLA violation."""
        workflow_type = workflow_info["workflow_type"]
        execution_pattern = workflow_info["execution_pattern"]
        
        # Determine expected SLA based on execution pattern
        expected_sla = self.sla_thresholds.get(f"{execution_pattern}_workflow_ms", 1000)
        
        severity = "critical" if execution_time_ms > expected_sla * 2 else "warning"
        
        violation = SLAViolation(
            violation_id=f"sla_violation_{int(time.time() * 1000)}",
            workflow_id=workflow_id,
            workflow_type=workflow_type,
            sla_type="execution_time",
            expected_value=expected_sla,
            actual_value=execution_time_ms,
            severity=severity,
            timestamp=datetime.utcnow(),
            patient_id=workflow_info.get("patient_id"),
            provider_id=workflow_info.get("provider_id"),
            details={
                "execution_pattern": execution_pattern,
                "violation_percentage": ((execution_time_ms - expected_sla) / expected_sla) * 100
            }
        )
        
        self.sla_violations.append(violation)
        
        logger.warning(f"⚠️ SLA violation: {workflow_type} took {execution_time_ms:.1f}ms (expected {expected_sla}ms)")
    
    async def _record_safety_metrics(
        self,
        workflow_id: str,
        safety_validation_result: Dict[str, Any]
    ):
        """Record safety validation metrics."""
        verdict = safety_validation_result.get("verdict", "UNKNOWN")
        processing_time = safety_validation_result.get("processing_time_ms", 0)
        
        # Record safety verdict
        await self.record_metric(
            metric_id="safety_validation_verdict",
            metric_name="Safety Validation Verdict",
            metric_type="counter",
            value=1,
            unit="count",
            tags={"verdict": verdict},
            safety_critical=True
        )
        
        # Record safety processing time
        await self.record_metric(
            metric_id="safety_validation_time",
            metric_name="Safety Validation Processing Time",
            metric_type="histogram",
            value=processing_time,
            unit="milliseconds",
            safety_critical=True,
            threshold_warning=150.0,
            threshold_critical=200.0
        )
        
        # Record unsafe decisions
        if verdict == "UNSAFE":
            await self.record_safety_alert(
                alert_type="unsafe_decision",
                severity="high",
                message=f"Unsafe decision detected in workflow {workflow_id}",
                workflow_id=workflow_id,
                details=safety_validation_result
            )
    
    async def _check_metric_thresholds(self, metric: ClinicalMetric):
        """Check if metric exceeds thresholds."""
        if metric.threshold_critical and metric.value >= metric.threshold_critical:
            await self.record_safety_alert(
                alert_type="metric_threshold_critical",
                severity="critical",
                message=f"Critical threshold exceeded: {metric.metric_name} = {metric.value} {metric.unit}",
                details={"metric": metric.__dict__}
            )
        elif metric.threshold_warning and metric.value >= metric.threshold_warning:
            await self.record_safety_alert(
                alert_type="metric_threshold_warning",
                severity="medium",
                message=f"Warning threshold exceeded: {metric.metric_name} = {metric.value} {metric.unit}",
                details={"metric": metric.__dict__}
            )
    
    async def _get_performance_summary(self) -> Dict[str, Any]:
        """Get performance summary statistics."""
        current_time = datetime.utcnow()
        
        # Calculate success rates, average times, etc.
        recent_workflows = [
            w for w in self.active_workflows.values()
            if w.get("completed_at") and (current_time - w["completed_at"]).total_seconds() < 3600
        ]
        
        if not recent_workflows:
            return {"message": "No recent workflow data available"}
        
        successful_workflows = [w for w in recent_workflows if w.get("status") == "completed"]
        success_rate = (len(successful_workflows) / len(recent_workflows)) * 100 if recent_workflows else 0
        
        return {
            "recent_workflows_count": len(recent_workflows),
            "success_rate_percent": round(success_rate, 2),
            "average_execution_time_ms": sum(
                w.get("execution_time_ms", 0) for w in recent_workflows
            ) / len(recent_workflows) if recent_workflows else 0,
            "sla_compliance_rate_percent": sum(
                1 for w in recent_workflows if w.get("sla_compliance", False)
            ) / len(recent_workflows) * 100 if recent_workflows else 0
        }
    
    async def _send_critical_alert_notification(self, alert: SafetyAlert):
        """Send notification for critical safety alerts."""
        logger.critical(f"🚨 CRITICAL SAFETY ALERT: {alert.message}")
        # In real implementation, send to monitoring systems, pages, etc.
    
    async def _start_monitoring(self):
        """Start background monitoring tasks."""
        logger.info("🔄 Starting clinical monitoring background tasks")
        
        # Start periodic cleanup
        asyncio.create_task(self._periodic_cleanup())
        
        # Start threshold monitoring
        asyncio.create_task(self._monitor_thresholds())
    
    async def _periodic_cleanup(self):
        """Periodic cleanup of old data."""
        while True:
            try:
                current_time = datetime.utcnow()
                
                # Clean up old SLA violations (keep 7 days)
                cutoff_time = current_time - timedelta(days=7)
                self.sla_violations = [
                    v for v in self.sla_violations
                    if v.timestamp > cutoff_time
                ]
                
                # Clean up resolved safety alerts (keep 7 days)
                self.safety_alerts = [
                    a for a in self.safety_alerts
                    if not a.resolved or a.timestamp > cutoff_time
                ]
                
                await asyncio.sleep(3600)  # Run every hour
                
            except Exception as e:
                logger.error(f"❌ Monitoring cleanup error: {e}")
                await asyncio.sleep(3600)
    
    async def _monitor_thresholds(self):
        """Monitor safety thresholds continuously."""
        while True:
            try:
                # Check safety threshold violations
                current_time = datetime.utcnow()
                hour_ago = current_time - timedelta(hours=1)
                
                # Calculate recent rates
                recent_workflows = len([
                    w for w in self.active_workflows.values()
                    if w.get("completed_at") and w["completed_at"] > hour_ago
                ])
                
                if recent_workflows > 0:
                    # Check error rate
                    error_count = len([
                        w for w in self.active_workflows.values()
                        if w.get("status") == "failed" and w.get("completed_at", datetime.min) > hour_ago
                    ])
                    error_rate = error_count / recent_workflows
                    
                    if error_rate > self.safety_thresholds["error_rate"]:
                        await self.record_safety_alert(
                            alert_type="high_error_rate",
                            severity="critical",
                            message=f"High error rate detected: {error_rate:.2%} (threshold: {self.safety_thresholds['error_rate']:.2%})",
                            details={"error_rate": error_rate, "error_count": error_count, "total_workflows": recent_workflows}
                        )
                
                await asyncio.sleep(300)  # Check every 5 minutes
                
            except Exception as e:
                logger.error(f"❌ Threshold monitoring error: {e}")
                await asyncio.sleep(300)


# Create singleton instance
clinical_monitoring_service = ClinicalMonitoringService()
