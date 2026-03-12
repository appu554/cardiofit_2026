"""
SLA Monitoring Data Models
Defines SLA targets, measurements, and compliance tracking structures
"""

from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List, Union
from enum import Enum
from pydantic import BaseModel, Field, validator
import uuid


class SLAMetricType(str, Enum):
    """Types of SLA metrics to monitor"""
    AVAILABILITY = "availability"
    RESPONSE_TIME = "response_time"
    ERROR_RATE = "error_rate"
    THROUGHPUT = "throughput"
    DATA_ACCURACY = "data_accuracy"
    CACHE_HIT_RATE = "cache_hit_rate"
    ML_PREDICTION_ACCURACY = "ml_prediction_accuracy"


class SLASeverity(str, Enum):
    """Severity levels for SLA violations"""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFO = "info"


class SLAStatus(str, Enum):
    """SLA compliance status"""
    COMPLIANT = "compliant"
    WARNING = "warning"
    VIOLATION = "violation"
    CRITICAL_VIOLATION = "critical_violation"
    NO_DATA = "no_data"


class SLATarget(BaseModel):
    """SLA target definition"""
    target_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    service_name: str
    metric_type: SLAMetricType
    target_value: float
    operator: str = Field(..., regex="^(gt|gte|lt|lte|eq)$")  # greater than, less than, etc.
    unit: str  # "ms", "percent", "rps", etc.
    measurement_window_minutes: int = 5
    evaluation_frequency_seconds: int = 30
    grace_period_minutes: int = 2
    severity: SLASeverity = SLASeverity.HIGH
    enabled: bool = True
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    def is_compliant(self, measured_value: float) -> bool:
        """Check if measured value meets SLA target"""
        if self.operator == "gt":
            return measured_value > self.target_value
        elif self.operator == "gte":
            return measured_value >= self.target_value
        elif self.operator == "lt":
            return measured_value < self.target_value
        elif self.operator == "lte":
            return measured_value <= self.target_value
        elif self.operator == "eq":
            return abs(measured_value - self.target_value) < 0.001
        return False

    def get_compliance_percentage(self, measured_value: float) -> float:
        """Calculate compliance as percentage"""
        if self.operator in ["gt", "gte"]:
            if measured_value >= self.target_value:
                return 100.0
            return (measured_value / self.target_value) * 100.0
        elif self.operator in ["lt", "lte"]:
            if measured_value <= self.target_value:
                return 100.0
            return max(0.0, (self.target_value / measured_value) * 100.0)
        return 100.0 if self.is_compliant(measured_value) else 0.0


class SLAMeasurement(BaseModel):
    """Individual SLA measurement"""
    measurement_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    target_id: str
    service_name: str
    metric_type: SLAMetricType
    measured_value: float
    target_value: float
    is_compliant: bool
    compliance_percentage: float
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    measurement_window_start: datetime
    measurement_window_end: datetime
    metadata: Dict[str, Any] = Field(default_factory=dict)

    @classmethod
    def from_target_and_value(
        cls,
        target: SLATarget,
        measured_value: float,
        window_start: datetime,
        window_end: datetime,
        metadata: Optional[Dict[str, Any]] = None
    ) -> "SLAMeasurement":
        """Create measurement from SLA target and measured value"""
        is_compliant = target.is_compliant(measured_value)
        compliance_percentage = target.get_compliance_percentage(measured_value)

        return cls(
            target_id=target.target_id,
            service_name=target.service_name,
            metric_type=target.metric_type,
            measured_value=measured_value,
            target_value=target.target_value,
            is_compliant=is_compliant,
            compliance_percentage=compliance_percentage,
            measurement_window_start=window_start,
            measurement_window_end=window_end,
            metadata=metadata or {}
        )


class SLAViolation(BaseModel):
    """SLA violation event"""
    violation_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    target_id: str
    service_name: str
    metric_type: SLAMetricType
    severity: SLASeverity
    measured_value: float
    target_value: float
    compliance_percentage: float
    started_at: datetime
    ended_at: Optional[datetime] = None
    duration_minutes: Optional[float] = None
    consecutive_violations: int = 1
    is_active: bool = True
    resolution_notes: Optional[str] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)

    def resolve(self, resolution_notes: Optional[str] = None):
        """Mark violation as resolved"""
        self.ended_at = datetime.utcnow()
        self.is_active = False
        self.duration_minutes = (self.ended_at - self.started_at).total_seconds() / 60.0
        if resolution_notes:
            self.resolution_notes = resolution_notes

    def extend_violation(self, new_measurement: SLAMeasurement):
        """Extend violation with new non-compliant measurement"""
        self.consecutive_violations += 1
        self.measured_value = new_measurement.measured_value
        self.compliance_percentage = new_measurement.compliance_percentage
        self.metadata.update(new_measurement.metadata)


class SLAReport(BaseModel):
    """Comprehensive SLA report"""
    report_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    service_name: str
    report_period_start: datetime
    report_period_end: datetime
    generated_at: datetime = Field(default_factory=datetime.utcnow)

    # Overall metrics
    total_measurements: int = 0
    compliant_measurements: int = 0
    overall_compliance_percentage: float = 0.0
    uptime_percentage: float = 0.0

    # Per-metric breakdown
    metric_summaries: List["SLAMetricSummary"] = Field(default_factory=list)

    # Violations
    total_violations: int = 0
    critical_violations: int = 0
    average_violation_duration_minutes: float = 0.0

    # Trends
    compliance_trend: str = "stable"  # "improving", "degrading", "stable"
    trend_confidence: float = 0.0

    @property
    def sla_status(self) -> SLAStatus:
        """Determine overall SLA status"""
        if self.overall_compliance_percentage >= 99.9:
            return SLAStatus.COMPLIANT
        elif self.overall_compliance_percentage >= 95.0:
            return SLAStatus.WARNING
        elif self.critical_violations > 0:
            return SLAStatus.CRITICAL_VIOLATION
        else:
            return SLAStatus.VIOLATION


class SLAMetricSummary(BaseModel):
    """Summary for a specific metric type"""
    metric_type: SLAMetricType
    target_value: float
    unit: str
    measurements_count: int
    compliant_measurements: int
    compliance_percentage: float
    average_measured_value: float
    p95_measured_value: float
    p99_measured_value: float
    violations_count: int
    longest_violation_minutes: float = 0.0


class SLAAlert(BaseModel):
    """SLA alert/notification"""
    alert_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    service_name: str
    metric_type: SLAMetricType
    severity: SLASeverity
    alert_type: str  # "violation_start", "violation_end", "degradation_warning"
    title: str
    message: str
    triggered_at: datetime = Field(default_factory=datetime.utcnow)
    resolved_at: Optional[datetime] = None
    is_active: bool = True
    violation_id: Optional[str] = None
    target_id: str
    metadata: Dict[str, Any] = Field(default_factory=dict)

    def resolve(self):
        """Mark alert as resolved"""
        self.resolved_at = datetime.utcnow()
        self.is_active = False


class ServiceHealthStatus(BaseModel):
    """Overall health status for a service"""
    service_name: str
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    overall_status: SLAStatus
    uptime_percentage: float
    response_time_p95_ms: Optional[float] = None
    error_rate_percentage: Optional[float] = None
    active_violations: int = 0
    critical_violations: int = 0
    last_violation_time: Optional[datetime] = None

    # Service-specific metrics
    custom_metrics: Dict[str, float] = Field(default_factory=dict)

    @property
    def is_healthy(self) -> bool:
        """Check if service is healthy"""
        return self.overall_status in [SLAStatus.COMPLIANT, SLAStatus.WARNING]


class SLADashboardData(BaseModel):
    """Data structure for SLA monitoring dashboard"""
    generated_at: datetime = Field(default_factory=datetime.utcnow)
    period_start: datetime
    period_end: datetime

    # Overall system health
    total_services: int
    healthy_services: int
    services_with_violations: int
    system_uptime_percentage: float

    # Service statuses
    service_statuses: List[ServiceHealthStatus] = Field(default_factory=list)

    # Recent violations
    recent_violations: List[SLAViolation] = Field(default_factory=list)

    # Active alerts
    active_alerts: List[SLAAlert] = Field(default_factory=list)

    # Compliance trends (last 24 hours)
    compliance_trends: Dict[str, List[float]] = Field(default_factory=dict)

    @property
    def system_health_score(self) -> float:
        """Overall system health score (0-100)"""
        if self.total_services == 0:
            return 100.0

        base_score = (self.healthy_services / self.total_services) * 100.0

        # Penalty for critical violations
        critical_penalty = min(20.0, self.services_with_violations * 5.0)

        # Uptime bonus/penalty
        uptime_modifier = (self.system_uptime_percentage - 95.0) * 0.5

        return max(0.0, min(100.0, base_score - critical_penalty + uptime_modifier))


class SLAConfiguration(BaseModel):
    """SLA monitoring system configuration"""
    config_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # Global settings
    enabled: bool = True
    default_measurement_window_minutes: int = 5
    default_evaluation_frequency_seconds: int = 30
    alert_cooldown_minutes: int = 10
    violation_grace_period_minutes: int = 2

    # Service targets
    targets: List[SLATarget] = Field(default_factory=list)

    # Alert channels
    alert_channels: List[str] = Field(default_factory=lambda: ["email", "slack"])
    email_recipients: List[str] = Field(default_factory=list)
    slack_webhook_url: Optional[str] = None

    # Retention settings
    measurement_retention_days: int = 30
    violation_retention_days: int = 90
    alert_retention_days: int = 30

    def get_targets_for_service(self, service_name: str) -> List[SLATarget]:
        """Get all SLA targets for a specific service"""
        return [target for target in self.targets if target.service_name == service_name and target.enabled]

    def add_target(self, target: SLATarget):
        """Add new SLA target"""
        self.targets.append(target)
        self.updated_at = datetime.utcnow()

    def remove_target(self, target_id: str):
        """Remove SLA target by ID"""
        self.targets = [t for t in self.targets if t.target_id != target_id]
        self.updated_at = datetime.utcnow()


# Default SLA targets for runtime layer services
DEFAULT_SLA_TARGETS = [
    # Flink Stream Processing
    SLATarget(
        service_name="flink-stream-processor",
        metric_type=SLAMetricType.AVAILABILITY,
        target_value=99.9,
        operator="gte",
        unit="percent",
        severity=SLASeverity.CRITICAL
    ),
    SLATarget(
        service_name="flink-stream-processor",
        metric_type=SLAMetricType.RESPONSE_TIME,
        target_value=500,  # 500ms for stream processing
        operator="lte",
        unit="ms",
        severity=SLASeverity.HIGH
    ),

    # Evidence Envelope Service
    SLATarget(
        service_name="evidence-envelope-service",
        metric_type=SLAMetricType.AVAILABILITY,
        target_value=99.95,
        operator="gte",
        unit="percent",
        severity=SLASeverity.CRITICAL
    ),
    SLATarget(
        service_name="evidence-envelope-service",
        metric_type=SLAMetricType.RESPONSE_TIME,
        target_value=200,  # 200ms for evidence operations
        operator="lte",
        unit="ms",
        severity=SLASeverity.HIGH
    ),

    # L1 Cache Service
    SLATarget(
        service_name="l1-cache-prefetcher-service",
        metric_type=SLAMetricType.AVAILABILITY,
        target_value=99.99,
        operator="gte",
        unit="percent",
        severity=SLASeverity.CRITICAL
    ),
    SLATarget(
        service_name="l1-cache-prefetcher-service",
        metric_type=SLAMetricType.RESPONSE_TIME,
        target_value=10,  # 10ms for L1 cache hits
        operator="lte",
        unit="ms",
        severity=SLASeverity.HIGH
    ),
    SLATarget(
        service_name="l1-cache-prefetcher-service",
        metric_type=SLAMetricType.CACHE_HIT_RATE,
        target_value=85.0,
        operator="gte",
        unit="percent",
        severity=SLASeverity.MEDIUM
    ),
    SLATarget(
        service_name="l1-cache-prefetcher-service",
        metric_type=SLAMetricType.ML_PREDICTION_ACCURACY,
        target_value=70.0,
        operator="gte",
        unit="percent",
        severity=SLASeverity.MEDIUM
    )
]