"""
SLA Monitoring Service - Main FastAPI Application
Provides REST API for SLA monitoring, configuration, and alerting
"""

import asyncio
from datetime import datetime, timedelta
from typing import List, Optional, Dict, Any
import structlog
from fastapi import FastAPI, HTTPException, Depends, Query, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from prometheus_client import Counter, Histogram, Gauge, generate_latest
from prometheus_client.core import CONTENT_TYPE_LATEST
import uvicorn

from .models.sla_models import (
    SLATarget, SLAMeasurement, SLAViolation, SLAReport, SLAAlert,
    SLADashboardData, SLAConfiguration, SLAMetricType, SLASeverity,
    ServiceHealthStatus, DEFAULT_SLA_TARGETS
)
from .services.sla_monitor import SLAMonitor
from .collectors.metrics_collector import MetricsCollector, DEFAULT_SERVICE_ENDPOINTS
from .alerting.alert_manager import AlertManager
from .config.sla_config import SLAConfigManager
from .middleware.auth_middleware import AuthMiddleware

logger = structlog.get_logger()

# Prometheus metrics
SLA_EVALUATIONS_TOTAL = Counter(
    "sla_evaluations_total",
    "Total SLA evaluations performed",
    ["service_name", "metric_type", "status"]
)
SLA_VIOLATIONS_ACTIVE = Gauge(
    "sla_violations_active",
    "Active SLA violations",
    ["service_name", "severity"]
)
SLA_COMPLIANCE_PERCENTAGE = Gauge(
    "sla_compliance_percentage",
    "Service compliance percentage",
    ["service_name", "metric_type"]
)
SLA_RESPONSE_TIME = Histogram(
    "sla_monitoring_response_time_seconds",
    "SLA monitoring API response times"
)

app = FastAPI(
    title="SLA Monitoring Service",
    description="Real-time SLA monitoring and alerting for CardioFit Runtime Layer",
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Authentication middleware
auth_middleware = AuthMiddleware()

# Global service instances
sla_monitor: Optional[SLAMonitor] = None
metrics_collector: Optional[MetricsCollector] = None
alert_manager: Optional[AlertManager] = None
config_manager: Optional[SLAConfigManager] = None


@app.on_event("startup")
async def startup_event():
    """Initialize SLA monitoring components"""
    global sla_monitor, metrics_collector, alert_manager, config_manager

    try:
        logger.info("sla_monitoring_service_starting")

        # Initialize configuration manager
        config_manager = SLAConfigManager()
        config = await config_manager.load_configuration()

        # Initialize metrics collector with default endpoints
        metrics_collector = MetricsCollector()
        for endpoint in DEFAULT_SERVICE_ENDPOINTS:
            metrics_collector.add_service_endpoint(endpoint)

        # Initialize alert manager
        alert_manager = AlertManager(
            slack_webhook_url=config.slack_webhook_url,
            email_recipients=config.email_recipients
        )

        # Initialize SLA monitor
        sla_monitor = SLAMonitor(
            metrics_collector=metrics_collector,
            alert_manager=alert_manager,
            config=config
        )

        # Start background monitoring
        asyncio.create_task(background_monitoring())

        logger.info("sla_monitoring_service_started", version="1.0.0")

    except Exception as e:
        logger.error("sla_monitoring_service_startup_failed", error=str(e))
        raise


@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup SLA monitoring components"""
    global sla_monitor, metrics_collector, alert_manager

    try:
        logger.info("sla_monitoring_service_shutting_down")

        if metrics_collector:
            await metrics_collector.close()

        logger.info("sla_monitoring_service_shutdown_complete")

    except Exception as e:
        logger.error("sla_monitoring_service_shutdown_error", error=str(e))


async def background_monitoring():
    """Background task for continuous SLA monitoring"""
    while True:
        try:
            if sla_monitor and config_manager:
                config = await config_manager.load_configuration()
                await sla_monitor.run_evaluation_cycle(config.targets)

            # Wait for next evaluation cycle (default 30 seconds)
            await asyncio.sleep(30)

        except Exception as e:
            logger.error("background_monitoring_error", error=str(e))
            await asyncio.sleep(60)  # Wait longer on error


# Health and monitoring endpoints
@app.get("/health")
async def health_check():
    """Basic health check"""
    return {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}


@app.get("/health/ready")
async def readiness_check():
    """Detailed readiness check"""
    checks = {
        "sla_monitor": sla_monitor is not None,
        "metrics_collector": metrics_collector is not None,
        "alert_manager": alert_manager is not None,
        "config_manager": config_manager is not None
    }

    all_ready = all(checks.values())

    return {
        "ready": all_ready,
        "checks": checks,
        "timestamp": datetime.utcnow().isoformat()
    }


@app.get("/metrics")
async def prometheus_metrics():
    """Prometheus metrics endpoint"""
    return JSONResponse(
        content=generate_latest().decode(),
        media_type=CONTENT_TYPE_LATEST
    )


# SLA Configuration endpoints
@app.get("/api/v1/configuration", response_model=SLAConfiguration)
@SLA_RESPONSE_TIME.time()
async def get_configuration(user=Depends(auth_middleware.get_current_user)):
    """Get current SLA configuration"""
    if not config_manager:
        raise HTTPException(status_code=503, detail="Configuration manager not available")

    try:
        config = await config_manager.load_configuration()
        return config
    except Exception as e:
        logger.error("get_configuration_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to load configuration")


@app.put("/api/v1/configuration", response_model=SLAConfiguration)
@SLA_RESPONSE_TIME.time()
async def update_configuration(
    config: SLAConfiguration,
    user=Depends(auth_middleware.get_current_user)
):
    """Update SLA configuration"""
    if not config_manager:
        raise HTTPException(status_code=503, detail="Configuration manager not available")

    try:
        updated_config = await config_manager.save_configuration(config)
        logger.info("sla_configuration_updated", config_id=updated_config.config_id)
        return updated_config
    except Exception as e:
        logger.error("update_configuration_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to update configuration")


# SLA Target management
@app.get("/api/v1/targets", response_model=List[SLATarget])
@SLA_RESPONSE_TIME.time()
async def get_sla_targets(
    service_name: Optional[str] = Query(None),
    metric_type: Optional[SLAMetricType] = Query(None),
    user=Depends(auth_middleware.get_current_user)
):
    """Get SLA targets with optional filtering"""
    if not config_manager:
        raise HTTPException(status_code=503, detail="Configuration manager not available")

    try:
        config = await config_manager.load_configuration()
        targets = config.targets

        if service_name:
            targets = [t for t in targets if t.service_name == service_name]

        if metric_type:
            targets = [t for t in targets if t.metric_type == metric_type]

        return targets
    except Exception as e:
        logger.error("get_sla_targets_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to get SLA targets")


@app.post("/api/v1/targets", response_model=SLATarget)
@SLA_RESPONSE_TIME.time()
async def create_sla_target(
    target: SLATarget,
    user=Depends(auth_middleware.get_current_user)
):
    """Create new SLA target"""
    if not config_manager:
        raise HTTPException(status_code=503, detail="Configuration manager not available")

    try:
        config = await config_manager.load_configuration()
        config.add_target(target)
        await config_manager.save_configuration(config)

        logger.info(
            "sla_target_created",
            target_id=target.target_id,
            service_name=target.service_name,
            metric_type=target.metric_type
        )

        return target
    except Exception as e:
        logger.error("create_sla_target_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to create SLA target")


@app.delete("/api/v1/targets/{target_id}")
@SLA_RESPONSE_TIME.time()
async def delete_sla_target(
    target_id: str,
    user=Depends(auth_middleware.get_current_user)
):
    """Delete SLA target"""
    if not config_manager:
        raise HTTPException(status_code=503, detail="Configuration manager not available")

    try:
        config = await config_manager.load_configuration()
        config.remove_target(target_id)
        await config_manager.save_configuration(config)

        logger.info("sla_target_deleted", target_id=target_id)

        return {"message": "SLA target deleted successfully"}
    except Exception as e:
        logger.error("delete_sla_target_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to delete SLA target")


# SLA Monitoring and reporting
@app.get("/api/v1/dashboard", response_model=SLADashboardData)
@SLA_RESPONSE_TIME.time()
async def get_dashboard_data(
    hours: int = Query(24, ge=1, le=168),
    user=Depends(auth_middleware.get_current_user)
):
    """Get SLA dashboard data for specified time period"""
    if not sla_monitor:
        raise HTTPException(status_code=503, detail="SLA monitor not available")

    try:
        period_end = datetime.utcnow()
        period_start = period_end - timedelta(hours=hours)

        dashboard_data = await sla_monitor.generate_dashboard_data(period_start, period_end)
        return dashboard_data

    except Exception as e:
        logger.error("get_dashboard_data_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to generate dashboard data")


@app.get("/api/v1/violations", response_model=List[SLAViolation])
@SLA_RESPONSE_TIME.time()
async def get_violations(
    service_name: Optional[str] = Query(None),
    active_only: bool = Query(True),
    hours: int = Query(24, ge=1, le=168),
    user=Depends(auth_middleware.get_current_user)
):
    """Get SLA violations with filtering"""
    if not sla_monitor:
        raise HTTPException(status_code=503, detail="SLA monitor not available")

    try:
        violations = await sla_monitor.get_violations(
            service_name=service_name,
            active_only=active_only,
            hours_back=hours
        )
        return violations

    except Exception as e:
        logger.error("get_violations_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to get violations")


@app.get("/api/v1/reports/{service_name}", response_model=SLAReport)
@SLA_RESPONSE_TIME.time()
async def get_service_report(
    service_name: str,
    hours: int = Query(24, ge=1, le=168),
    user=Depends(auth_middleware.get_current_user)
):
    """Generate SLA report for specific service"""
    if not sla_monitor:
        raise HTTPException(status_code=503, detail="SLA monitor not available")

    try:
        period_end = datetime.utcnow()
        period_start = period_end - timedelta(hours=hours)

        report = await sla_monitor.generate_service_report(
            service_name, period_start, period_end
        )
        return report

    except Exception as e:
        logger.error("get_service_report_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to generate service report")


# Manual SLA evaluation
@app.post("/api/v1/evaluate")
@SLA_RESPONSE_TIME.time()
async def trigger_manual_evaluation(
    background_tasks: BackgroundTasks,
    service_name: Optional[str] = Query(None),
    user=Depends(auth_middleware.get_current_user)
):
    """Trigger manual SLA evaluation"""
    if not sla_monitor or not config_manager:
        raise HTTPException(status_code=503, detail="SLA monitoring services not available")

    try:
        config = await config_manager.load_configuration()
        targets = config.targets

        if service_name:
            targets = [t for t in targets if t.service_name == service_name]

        background_tasks.add_task(sla_monitor.run_evaluation_cycle, targets)

        return {
            "message": "Manual SLA evaluation triggered",
            "targets_count": len(targets),
            "service_filter": service_name
        }

    except Exception as e:
        logger.error("trigger_manual_evaluation_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to trigger evaluation")


# Alert management
@app.get("/api/v1/alerts", response_model=List[SLAAlert])
@SLA_RESPONSE_TIME.time()
async def get_alerts(
    active_only: bool = Query(True),
    hours: int = Query(24, ge=1, le=168),
    user=Depends(auth_middleware.get_current_user)
):
    """Get SLA alerts"""
    if not alert_manager:
        raise HTTPException(status_code=503, detail="Alert manager not available")

    try:
        alerts = await alert_manager.get_alerts(
            active_only=active_only,
            hours_back=hours
        )
        return alerts

    except Exception as e:
        logger.error("get_alerts_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to get alerts")


@app.post("/api/v1/alerts/{alert_id}/resolve")
@SLA_RESPONSE_TIME.time()
async def resolve_alert(
    alert_id: str,
    user=Depends(auth_middleware.get_current_user)
):
    """Mark alert as resolved"""
    if not alert_manager:
        raise HTTPException(status_code=503, detail="Alert manager not available")

    try:
        success = await alert_manager.resolve_alert(alert_id)
        if success:
            logger.info("alert_resolved", alert_id=alert_id, resolved_by=user.get("sub"))
            return {"message": "Alert resolved successfully"}
        else:
            raise HTTPException(status_code=404, detail="Alert not found")

    except HTTPException:
        raise
    except Exception as e:
        logger.error("resolve_alert_error", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to resolve alert")


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8050,
        reload=True,
        log_config={
            "version": 1,
            "disable_existing_loggers": False,
            "formatters": {
                "default": {
                    "format": "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
                },
            },
            "handlers": {
                "default": {
                    "formatter": "default",
                    "class": "logging.StreamHandler",
                    "stream": "ext://sys.stdout",
                },
            },
            "root": {
                "level": "INFO",
                "handlers": ["default"],
            },
        }
    )