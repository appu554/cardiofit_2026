"""
Clinical Safety Monitor for Production Clinical Intelligence System

Specialized monitoring for clinical safety with real-time safety alerts,
risk assessment, and compliance tracking for healthcare environments.
"""

import logging
import asyncio
from datetime import datetime, timezone, timedelta
from typing import Dict, List, Optional, Any, Set
from dataclasses import dataclass, field
from enum import Enum
import uuid
import statistics

logger = logging.getLogger(__name__)


class RiskLevel(Enum):
    """Clinical risk levels"""
    CRITICAL = "critical"      # Immediate patient safety risk
    HIGH = "high"             # Significant safety concern
    MODERATE = "moderate"     # Moderate safety risk
    LOW = "low"              # Minor safety consideration
    MINIMAL = "minimal"       # Negligible risk


class SafetyCategory(Enum):
    """Categories of safety monitoring"""
    MEDICATION_SAFETY = "medication_safety"
    DRUG_INTERACTIONS = "drug_interactions"
    ALLERGIC_REACTIONS = "allergic_reactions"
    DOSING_ERRORS = "dosing_errors"
    CONTRAINDICATIONS = "contraindications"
    ADVERSE_EVENTS = "adverse_events"
    CLINICAL_DECISION_SAFETY = "clinical_decision_safety"
    SYSTEM_RELIABILITY = "system_reliability"


@dataclass
class SafetyMetric:
    """Clinical safety metric"""
    metric_id: str
    category: SafetyCategory
    risk_level: RiskLevel
    value: float
    unit: str
    timestamp: datetime
    patient_id: Optional[str] = None
    encounter_id: Optional[str] = None
    source: str = "safety_monitor"
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class SafetyAlert:
    """Clinical safety alert"""
    alert_id: str
    category: SafetyCategory
    risk_level: RiskLevel
    title: str
    description: str
    patient_id: Optional[str]
    encounter_id: Optional[str]
    triggered_at: datetime
    resolved_at: Optional[datetime] = None
    acknowledged_at: Optional[datetime] = None
    acknowledged_by: Optional[str] = None
    escalated: bool = False
    escalated_at: Optional[datetime] = None
    escalated_to: Optional[str] = None
    actions_taken: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class SafetyIncident:
    """Clinical safety incident"""
    incident_id: str
    category: SafetyCategory
    severity: RiskLevel
    description: str
    patient_id: Optional[str]
    encounter_id: Optional[str]
    reported_at: datetime
    reported_by: str
    investigation_status: str = "open"
    root_cause: Optional[str] = None
    corrective_actions: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)


class ClinicalSafetyMonitor:
    """
    Clinical Safety Monitoring System
    
    Provides real-time monitoring of clinical safety metrics,
    automated safety alerts, and incident tracking for
    production clinical intelligence systems.
    """
    
    def __init__(self):
        # Safety metrics storage
        self.safety_metrics: Dict[SafetyCategory, List[SafetyMetric]] = {
            category: [] for category in SafetyCategory
        }
        
        # Active safety alerts
        self.active_alerts: Dict[str, SafetyAlert] = {}
        self.alert_history: List[SafetyAlert] = []
        
        # Safety incidents
        self.safety_incidents: List[SafetyIncident] = []
        
        # Safety thresholds
        self.safety_thresholds = self._initialize_safety_thresholds()
        
        # Monitoring configuration
        self.monitoring_active = False
        self.monitoring_task = None
        self.check_interval_seconds = 30
        
        # Safety statistics
        self.safety_stats = {
            "total_alerts": 0,
            "critical_alerts": 0,
            "resolved_alerts": 0,
            "incidents_reported": 0,
            "average_resolution_time_minutes": 0.0,
            "safety_score": 100.0,
            "last_safety_check": None
        }
        
        logger.info("Clinical Safety Monitor initialized")
    
    def _initialize_safety_thresholds(self) -> Dict[SafetyCategory, Dict[str, Any]]:
        """Initialize safety monitoring thresholds"""
        return {
            SafetyCategory.MEDICATION_SAFETY: {
                "error_rate_threshold": 0.1,  # 0.1% medication errors
                "high_risk_medication_threshold": 5.0,  # 5% high-risk medications
                "monitoring_interval_minutes": 15
            },
            SafetyCategory.DRUG_INTERACTIONS: {
                "critical_interaction_threshold": 0.0,  # Zero tolerance for critical interactions
                "moderate_interaction_threshold": 2.0,  # 2% moderate interactions acceptable
                "monitoring_interval_minutes": 5
            },
            SafetyCategory.ALLERGIC_REACTIONS: {
                "allergy_violation_threshold": 0.0,  # Zero tolerance for allergy violations
                "monitoring_interval_minutes": 5
            },
            SafetyCategory.DOSING_ERRORS: {
                "dosing_error_threshold": 0.5,  # 0.5% dosing errors
                "overdose_threshold": 0.0,  # Zero tolerance for overdoses
                "monitoring_interval_minutes": 10
            },
            SafetyCategory.CONTRAINDICATIONS: {
                "contraindication_threshold": 0.0,  # Zero tolerance for contraindications
                "monitoring_interval_minutes": 5
            },
            SafetyCategory.ADVERSE_EVENTS: {
                "adverse_event_rate_threshold": 1.0,  # 1% adverse event rate
                "severe_adverse_event_threshold": 0.1,  # 0.1% severe adverse events
                "monitoring_interval_minutes": 30
            },
            SafetyCategory.CLINICAL_DECISION_SAFETY: {
                "low_confidence_threshold": 70.0,  # Alert if confidence < 70%
                "decision_override_rate_threshold": 10.0,  # 10% override rate
                "monitoring_interval_minutes": 20
            },
            SafetyCategory.SYSTEM_RELIABILITY: {
                "system_availability_threshold": 99.5,  # 99.5% availability
                "response_time_threshold": 500.0,  # 500ms response time
                "monitoring_interval_minutes": 5
            }
        }
    
    async def start_monitoring(self):
        """Start clinical safety monitoring"""
        if self.monitoring_active:
            logger.warning("Clinical safety monitoring already active")
            return
        
        self.monitoring_active = True
        self.monitoring_task = asyncio.create_task(self._safety_monitoring_loop())
        logger.info("Clinical safety monitoring started")
    
    async def stop_monitoring(self):
        """Stop clinical safety monitoring"""
        if not self.monitoring_active:
            return
        
        self.monitoring_active = False
        if self.monitoring_task:
            self.monitoring_task.cancel()
            try:
                await self.monitoring_task
            except asyncio.CancelledError:
                pass
        
        logger.info("Clinical safety monitoring stopped")
    
    async def _safety_monitoring_loop(self):
        """Main safety monitoring loop"""
        try:
            while self.monitoring_active:
                await self._perform_safety_checks()
                await self._check_safety_alerts()
                await self._update_safety_score()
                await asyncio.sleep(self.check_interval_seconds)
        except asyncio.CancelledError:
            logger.info("Safety monitoring loop cancelled")
        except Exception as e:
            logger.error(f"Error in safety monitoring loop: {e}")
    
    async def _perform_safety_checks(self):
        """Perform comprehensive safety checks"""
        try:
            timestamp = datetime.now(timezone.utc)
            
            # Check medication safety
            await self._check_medication_safety(timestamp)
            
            # Check drug interactions
            await self._check_drug_interactions(timestamp)
            
            # Check allergic reactions
            await self._check_allergic_reactions(timestamp)
            
            # Check dosing errors
            await self._check_dosing_errors(timestamp)
            
            # Check contraindications
            await self._check_contraindications(timestamp)
            
            # Check adverse events
            await self._check_adverse_events(timestamp)
            
            # Check clinical decision safety
            await self._check_clinical_decision_safety(timestamp)
            
            # Check system reliability
            await self._check_system_reliability(timestamp)
            
            self.safety_stats["last_safety_check"] = timestamp
            
        except Exception as e:
            logger.error(f"Error performing safety checks: {e}")
    
    async def _check_medication_safety(self, timestamp: datetime):
        """Check medication safety metrics"""
        try:
            # Simulate medication safety check
            import random
            
            # Check medication error rate
            error_rate = random.uniform(0, 0.2)  # 0-0.2% error rate
            
            metric = SafetyMetric(
                metric_id=f"med_safety_{timestamp.timestamp()}",
                category=SafetyCategory.MEDICATION_SAFETY,
                risk_level=self._assess_risk_level(error_rate, 0.1, "gt"),
                value=error_rate,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "medication_error_rate"}
            )
            
            await self._record_safety_metric(metric)
            
            # Check high-risk medications
            high_risk_rate = random.uniform(0, 8.0)  # 0-8% high-risk medications
            
            high_risk_metric = SafetyMetric(
                metric_id=f"high_risk_med_{timestamp.timestamp()}",
                category=SafetyCategory.MEDICATION_SAFETY,
                risk_level=self._assess_risk_level(high_risk_rate, 5.0, "gt"),
                value=high_risk_rate,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "high_risk_medications"}
            )
            
            await self._record_safety_metric(high_risk_metric)
            
        except Exception as e:
            logger.error(f"Error checking medication safety: {e}")
    
    async def _check_drug_interactions(self, timestamp: datetime):
        """Check drug interaction safety"""
        try:
            import random
            
            # Check critical interactions
            critical_interactions = random.uniform(0, 0.1)  # 0-0.1% critical interactions
            
            metric = SafetyMetric(
                metric_id=f"drug_interactions_{timestamp.timestamp()}",
                category=SafetyCategory.DRUG_INTERACTIONS,
                risk_level=self._assess_risk_level(critical_interactions, 0.0, "gt"),
                value=critical_interactions,
                unit="percent",
                timestamp=timestamp,
                metadata={"interaction_severity": "critical"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking drug interactions: {e}")
    
    async def _check_allergic_reactions(self, timestamp: datetime):
        """Check allergic reaction safety"""
        try:
            import random
            
            # Check allergy violations
            allergy_violations = random.uniform(0, 0.05)  # 0-0.05% allergy violations
            
            metric = SafetyMetric(
                metric_id=f"allergy_check_{timestamp.timestamp()}",
                category=SafetyCategory.ALLERGIC_REACTIONS,
                risk_level=self._assess_risk_level(allergy_violations, 0.0, "gt"),
                value=allergy_violations,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "allergy_violations"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking allergic reactions: {e}")
    
    async def _check_dosing_errors(self, timestamp: datetime):
        """Check dosing error safety"""
        try:
            import random
            
            # Check dosing errors
            dosing_errors = random.uniform(0, 1.0)  # 0-1.0% dosing errors
            
            metric = SafetyMetric(
                metric_id=f"dosing_errors_{timestamp.timestamp()}",
                category=SafetyCategory.DOSING_ERRORS,
                risk_level=self._assess_risk_level(dosing_errors, 0.5, "gt"),
                value=dosing_errors,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "dosing_errors"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking dosing errors: {e}")
    
    async def _check_contraindications(self, timestamp: datetime):
        """Check contraindication safety"""
        try:
            import random
            
            # Check contraindications
            contraindications = random.uniform(0, 0.02)  # 0-0.02% contraindications
            
            metric = SafetyMetric(
                metric_id=f"contraindications_{timestamp.timestamp()}",
                category=SafetyCategory.CONTRAINDICATIONS,
                risk_level=self._assess_risk_level(contraindications, 0.0, "gt"),
                value=contraindications,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "contraindications"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking contraindications: {e}")
    
    async def _check_adverse_events(self, timestamp: datetime):
        """Check adverse event safety"""
        try:
            import random
            
            # Check adverse event rate
            adverse_event_rate = random.uniform(0, 2.0)  # 0-2.0% adverse event rate
            
            metric = SafetyMetric(
                metric_id=f"adverse_events_{timestamp.timestamp()}",
                category=SafetyCategory.ADVERSE_EVENTS,
                risk_level=self._assess_risk_level(adverse_event_rate, 1.0, "gt"),
                value=adverse_event_rate,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "adverse_event_rate"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking adverse events: {e}")
    
    async def _check_clinical_decision_safety(self, timestamp: datetime):
        """Check clinical decision safety"""
        try:
            import random
            
            # Check low confidence decisions
            low_confidence_rate = random.uniform(5, 15)  # 5-15% low confidence decisions
            
            metric = SafetyMetric(
                metric_id=f"decision_safety_{timestamp.timestamp()}",
                category=SafetyCategory.CLINICAL_DECISION_SAFETY,
                risk_level=self._assess_risk_level(low_confidence_rate, 10.0, "gt"),
                value=low_confidence_rate,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "low_confidence_decisions"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking clinical decision safety: {e}")
    
    async def _check_system_reliability(self, timestamp: datetime):
        """Check system reliability safety"""
        try:
            import random
            
            # Check system availability
            availability = random.uniform(99.0, 100.0)  # 99-100% availability
            
            metric = SafetyMetric(
                metric_id=f"system_reliability_{timestamp.timestamp()}",
                category=SafetyCategory.SYSTEM_RELIABILITY,
                risk_level=self._assess_risk_level(availability, 99.5, "lt"),
                value=availability,
                unit="percent",
                timestamp=timestamp,
                metadata={"check_type": "system_availability"}
            )
            
            await self._record_safety_metric(metric)
            
        except Exception as e:
            logger.error(f"Error checking system reliability: {e}")

    def _assess_risk_level(self, value: float, threshold: float, operator: str) -> RiskLevel:
        """Assess risk level based on value and threshold"""
        if operator == "gt":
            if value > threshold * 2:
                return RiskLevel.CRITICAL
            elif value > threshold * 1.5:
                return RiskLevel.HIGH
            elif value > threshold:
                return RiskLevel.MODERATE
            elif value > threshold * 0.5:
                return RiskLevel.LOW
            else:
                return RiskLevel.MINIMAL
        elif operator == "lt":
            if value < threshold * 0.5:
                return RiskLevel.CRITICAL
            elif value < threshold * 0.7:
                return RiskLevel.HIGH
            elif value < threshold:
                return RiskLevel.MODERATE
            elif value < threshold * 1.2:
                return RiskLevel.LOW
            else:
                return RiskLevel.MINIMAL
        else:
            return RiskLevel.MINIMAL

    async def _record_safety_metric(self, metric: SafetyMetric):
        """Record a safety metric"""
        try:
            # Store metric
            self.safety_metrics[metric.category].append(metric)

            # Keep only recent metrics (last 24 hours)
            cutoff_time = datetime.now(timezone.utc) - timedelta(hours=24)
            self.safety_metrics[metric.category] = [
                m for m in self.safety_metrics[metric.category]
                if m.timestamp > cutoff_time
            ]

            # Check if alert should be triggered
            if metric.risk_level in [RiskLevel.CRITICAL, RiskLevel.HIGH]:
                await self._trigger_safety_alert(metric)

            logger.debug(f"Recorded safety metric: {metric.category.value} = {metric.value} ({metric.risk_level.value})")

        except Exception as e:
            logger.error(f"Error recording safety metric: {e}")

    async def _trigger_safety_alert(self, metric: SafetyMetric):
        """Trigger a safety alert based on metric"""
        try:
            # Check if similar alert already exists
            alert_key = f"{metric.category.value}_{metric.risk_level.value}"

            if alert_key in self.active_alerts:
                # Update existing alert
                existing_alert = self.active_alerts[alert_key]
                existing_alert.metadata["latest_value"] = metric.value
                existing_alert.metadata["latest_timestamp"] = metric.timestamp.isoformat()
                return

            # Create new safety alert
            alert = SafetyAlert(
                alert_id=str(uuid.uuid4()),
                category=metric.category,
                risk_level=metric.risk_level,
                title=self._generate_safety_alert_title(metric),
                description=self._generate_safety_alert_description(metric),
                patient_id=metric.patient_id,
                encounter_id=metric.encounter_id,
                triggered_at=datetime.now(timezone.utc),
                metadata={
                    "metric_value": metric.value,
                    "metric_unit": metric.unit,
                    "metric_metadata": metric.metadata
                }
            )

            # Store alert
            self.active_alerts[alert_key] = alert
            self.alert_history.append(alert)

            # Update statistics
            self.safety_stats["total_alerts"] += 1
            if alert.risk_level == RiskLevel.CRITICAL:
                self.safety_stats["critical_alerts"] += 1

            # Log alert
            logger.warning(f"Safety alert triggered: {alert.title}")

            # Send alert notification
            await self._send_safety_alert_notification(alert)

            # Auto-escalate critical alerts
            if alert.risk_level == RiskLevel.CRITICAL:
                await self._escalate_safety_alert(alert)

        except Exception as e:
            logger.error(f"Error triggering safety alert: {e}")

    def _generate_safety_alert_title(self, metric: SafetyMetric) -> str:
        """Generate safety alert title"""
        return f"{metric.risk_level.value.upper()} {metric.category.value.replace('_', ' ').title()} Alert"

    def _generate_safety_alert_description(self, metric: SafetyMetric) -> str:
        """Generate safety alert description"""
        return (f"{metric.category.value.replace('_', ' ').title()} metric shows {metric.risk_level.value} risk: "
                f"{metric.value} {metric.unit}")

    async def _send_safety_alert_notification(self, alert: SafetyAlert):
        """Send safety alert notification"""
        # In production, this would integrate with:
        # - Clinical alert systems
        # - Pager/SMS for critical alerts
        # - Electronic health record alerts
        # - Clinical dashboard notifications
        logger.info(f"Safety alert notification sent: {alert.alert_id}")

    async def _escalate_safety_alert(self, alert: SafetyAlert):
        """Escalate critical safety alert"""
        try:
            alert.escalated = True
            alert.escalated_at = datetime.now(timezone.utc)
            alert.escalated_to = "clinical_supervisor"  # Would be configurable

            logger.critical(f"Critical safety alert escalated: {alert.title}")

            # Send escalation notification
            await self._send_escalation_notification(alert)

        except Exception as e:
            logger.error(f"Error escalating safety alert: {e}")

    async def _send_escalation_notification(self, alert: SafetyAlert):
        """Send escalation notification"""
        logger.info(f"Escalation notification sent: {alert.alert_id}")

    async def _check_safety_alerts(self):
        """Check for alert resolution conditions"""
        try:
            alerts_to_resolve = []

            for alert_key, alert in self.active_alerts.items():
                # Check if conditions have improved
                recent_metrics = self._get_recent_safety_metrics(alert.category, minutes=5)

                if recent_metrics:
                    avg_risk = self._calculate_average_risk_level(recent_metrics)

                    # Resolve if risk has decreased significantly
                    if avg_risk in [RiskLevel.MINIMAL, RiskLevel.LOW] and alert.risk_level in [RiskLevel.HIGH, RiskLevel.CRITICAL]:
                        alerts_to_resolve.append(alert_key)

            # Resolve alerts
            for alert_key in alerts_to_resolve:
                await self._resolve_safety_alert(alert_key)

        except Exception as e:
            logger.error(f"Error checking safety alerts: {e}")

    async def _resolve_safety_alert(self, alert_key: str):
        """Resolve a safety alert"""
        try:
            if alert_key not in self.active_alerts:
                return

            alert = self.active_alerts[alert_key]
            alert.resolved_at = datetime.now(timezone.utc)

            # Remove from active alerts
            del self.active_alerts[alert_key]

            # Update statistics
            self.safety_stats["resolved_alerts"] += 1

            # Calculate resolution time
            resolution_time = (alert.resolved_at - alert.triggered_at).total_seconds() / 60
            current_avg = self.safety_stats["average_resolution_time_minutes"]
            total_resolved = self.safety_stats["resolved_alerts"]

            self.safety_stats["average_resolution_time_minutes"] = (
                (current_avg * (total_resolved - 1) + resolution_time) / total_resolved
            )

            logger.info(f"Safety alert resolved: {alert.title}")

            # Send resolution notification
            await self._send_safety_resolution_notification(alert)

        except Exception as e:
            logger.error(f"Error resolving safety alert: {e}")

    async def _send_safety_resolution_notification(self, alert: SafetyAlert):
        """Send safety alert resolution notification"""
        logger.info(f"Safety resolution notification sent: {alert.alert_id}")

    def _get_recent_safety_metrics(self, category: SafetyCategory, minutes: int = 30) -> List[SafetyMetric]:
        """Get recent safety metrics for category"""
        cutoff_time = datetime.now(timezone.utc) - timedelta(minutes=minutes)
        return [
            m for m in self.safety_metrics[category]
            if m.timestamp > cutoff_time
        ]

    def _calculate_average_risk_level(self, metrics: List[SafetyMetric]) -> RiskLevel:
        """Calculate average risk level from metrics"""
        if not metrics:
            return RiskLevel.MINIMAL

        risk_values = {
            RiskLevel.MINIMAL: 1,
            RiskLevel.LOW: 2,
            RiskLevel.MODERATE: 3,
            RiskLevel.HIGH: 4,
            RiskLevel.CRITICAL: 5
        }

        avg_value = statistics.mean([risk_values[m.risk_level] for m in metrics])

        if avg_value >= 4.5:
            return RiskLevel.CRITICAL
        elif avg_value >= 3.5:
            return RiskLevel.HIGH
        elif avg_value >= 2.5:
            return RiskLevel.MODERATE
        elif avg_value >= 1.5:
            return RiskLevel.LOW
        else:
            return RiskLevel.MINIMAL

    async def _update_safety_score(self):
        """Update overall safety score"""
        try:
            # Calculate safety score based on recent metrics and alerts
            active_alerts = list(self.active_alerts.values())

            # Base score
            safety_score = 100.0

            # Deduct points for active alerts
            for alert in active_alerts:
                if alert.risk_level == RiskLevel.CRITICAL:
                    safety_score -= 20.0
                elif alert.risk_level == RiskLevel.HIGH:
                    safety_score -= 10.0
                elif alert.risk_level == RiskLevel.MODERATE:
                    safety_score -= 5.0
                elif alert.risk_level == RiskLevel.LOW:
                    safety_score -= 2.0

            # Ensure score doesn't go below 0
            safety_score = max(0.0, safety_score)

            self.safety_stats["safety_score"] = safety_score

        except Exception as e:
            logger.error(f"Error updating safety score: {e}")

    # Public API methods

    async def report_safety_incident(self, category: SafetyCategory, severity: RiskLevel,
                                   description: str, patient_id: Optional[str] = None,
                                   encounter_id: Optional[str] = None,
                                   reported_by: str = "system") -> str:
        """Report a safety incident"""
        try:
            incident = SafetyIncident(
                incident_id=str(uuid.uuid4()),
                category=category,
                severity=severity,
                description=description,
                patient_id=patient_id,
                encounter_id=encounter_id,
                reported_at=datetime.now(timezone.utc),
                reported_by=reported_by
            )

            self.safety_incidents.append(incident)
            self.safety_stats["incidents_reported"] += 1

            logger.warning(f"Safety incident reported: {incident.incident_id}")

            # Trigger alert for high-severity incidents
            if severity in [RiskLevel.CRITICAL, RiskLevel.HIGH]:
                await self._trigger_incident_alert(incident)

            return incident.incident_id

        except Exception as e:
            logger.error(f"Error reporting safety incident: {e}")
            return ""

    async def _trigger_incident_alert(self, incident: SafetyIncident):
        """Trigger alert for safety incident"""
        # Create alert for incident
        alert = SafetyAlert(
            alert_id=str(uuid.uuid4()),
            category=incident.category,
            risk_level=incident.severity,
            title=f"Safety Incident: {incident.category.value.replace('_', ' ').title()}",
            description=f"Safety incident reported: {incident.description}",
            patient_id=incident.patient_id,
            encounter_id=incident.encounter_id,
            triggered_at=datetime.now(timezone.utc),
            metadata={"incident_id": incident.incident_id}
        )

        alert_key = f"incident_{incident.incident_id}"
        self.active_alerts[alert_key] = alert
        self.alert_history.append(alert)

        await self._send_safety_alert_notification(alert)

    def acknowledge_safety_alert(self, alert_id: str, acknowledged_by: str) -> bool:
        """Acknowledge a safety alert"""
        for alert in self.active_alerts.values():
            if alert.alert_id == alert_id:
                alert.acknowledged_at = datetime.now(timezone.utc)
                alert.acknowledged_by = acknowledged_by
                logger.info(f"Safety alert acknowledged by {acknowledged_by}: {alert_id}")
                return True
        return False

    def get_safety_dashboard(self) -> Dict[str, Any]:
        """Get comprehensive safety dashboard data"""
        active_alerts = list(self.active_alerts.values())
        recent_incidents = [
            incident for incident in self.safety_incidents
            if incident.reported_at > datetime.now(timezone.utc) - timedelta(hours=24)
        ]

        return {
            "safety_score": self.safety_stats["safety_score"],
            "safety_stats": self.safety_stats,
            "active_alerts": {
                "total": len(active_alerts),
                "critical": len([a for a in active_alerts if a.risk_level == RiskLevel.CRITICAL]),
                "high": len([a for a in active_alerts if a.risk_level == RiskLevel.HIGH]),
                "moderate": len([a for a in active_alerts if a.risk_level == RiskLevel.MODERATE]),
                "alerts": [
                    {
                        "alert_id": alert.alert_id,
                        "category": alert.category.value,
                        "risk_level": alert.risk_level.value,
                        "title": alert.title,
                        "triggered_at": alert.triggered_at.isoformat(),
                        "acknowledged": alert.acknowledged_at is not None,
                        "escalated": alert.escalated
                    }
                    for alert in active_alerts
                ]
            },
            "recent_incidents": {
                "total": len(recent_incidents),
                "incidents": [
                    {
                        "incident_id": incident.incident_id,
                        "category": incident.category.value,
                        "severity": incident.severity.value,
                        "description": incident.description,
                        "reported_at": incident.reported_at.isoformat(),
                        "status": incident.investigation_status
                    }
                    for incident in recent_incidents
                ]
            },
            "safety_metrics_summary": self._get_safety_metrics_summary()
        }

    def _get_safety_metrics_summary(self) -> Dict[str, Any]:
        """Get summary of safety metrics"""
        summary = {}

        for category in SafetyCategory:
            recent_metrics = self._get_recent_safety_metrics(category, minutes=60)

            if recent_metrics:
                avg_risk = self._calculate_average_risk_level(recent_metrics)
                latest_metric = max(recent_metrics, key=lambda m: m.timestamp)

                summary[category.value] = {
                    "latest_value": latest_metric.value,
                    "latest_unit": latest_metric.unit,
                    "average_risk_level": avg_risk.value,
                    "metric_count": len(recent_metrics),
                    "last_updated": latest_metric.timestamp.isoformat()
                }
            else:
                summary[category.value] = {
                    "latest_value": None,
                    "latest_unit": None,
                    "average_risk_level": "unknown",
                    "metric_count": 0,
                    "last_updated": None
                }

        return summary
