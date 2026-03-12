"""
Monitoring and Observability API Endpoints for Calculate > Validate > Commit Workflow

Exposes:
- Health check endpoints (liveness, readiness, detailed health)
- Prometheus metrics endpoint
- Performance metrics and alerts
- System status dashboard data
"""

import logging
from typing import Dict, Any, Optional, List
from datetime import datetime, timedelta
from fastapi import APIRouter, Response, HTTPException, Query, Depends
from fastapi.responses import PlainTextResponse

from app.monitoring.health_checks import get_health_checker
from app.monitoring.workflow_metrics import get_metrics_collector
from app.config import settings

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/monitoring", tags=["monitoring"])

@router.get("/health", summary="Comprehensive Health Check")
async def health_check() -> Dict[str, Any]:
    """
    Comprehensive health check for all workflow components
    
    Returns detailed health status including:
    - Overall system status
    - Individual service health
    - Performance metrics
    - Response times
    """
    try:
        health_checker = get_health_checker()
        health_status = await health_checker.check_overall_health()
        return health_status
    except Exception as e:
        logger.error("Health check failed: %s", e)
        raise HTTPException(status_code=500, detail=f"Health check failed: {str(e)}")

@router.get("/health/live", summary="Liveness Probe")
async def liveness_probe() -> Dict[str, Any]:
    """
    Kubernetes liveness probe endpoint
    
    Simple check that the service is running and responsive.
    Used by Kubernetes to determine if the pod should be restarted.
    """
    try:
        # Basic service responsiveness check
        return {
            "status": "alive",
            "timestamp": datetime.now().isoformat(),
            "service": "workflow-engine-service"
        }
    except Exception as e:
        logger.error("Liveness probe failed: %s", e)
        raise HTTPException(status_code=500, detail="Service not alive")

@router.get("/health/ready", summary="Readiness Probe")
async def readiness_probe() -> Dict[str, Any]:
    """
    Kubernetes readiness probe endpoint
    
    Checks if service is ready to handle requests.
    Used by Kubernetes to determine if the pod should receive traffic.
    """
    try:
        health_checker = get_health_checker()
        readiness_status = await health_checker.get_readiness_status()
        
        if not readiness_status.get('ready', False):
            raise HTTPException(status_code=503, detail="Service not ready")
        
        return readiness_status
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Readiness probe failed: %s", e)
        raise HTTPException(status_code=500, detail=f"Readiness check failed: {str(e)}")

@router.get("/metrics", response_class=PlainTextResponse, summary="Prometheus Metrics")
async def prometheus_metrics() -> str:
    """
    Prometheus metrics endpoint
    
    Returns metrics in Prometheus text format for scraping:
    - Workflow performance counters
    - Phase timing histograms
    - Success/failure rates
    - Safety Gateway validation metrics
    """
    try:
        metrics_collector = get_metrics_collector()
        metrics_text = metrics_collector.export_prometheus_metrics()
        return metrics_text
    except Exception as e:
        logger.error("Failed to export Prometheus metrics: %s", e)
        return f"# Error exporting metrics: {str(e)}\n"

@router.get("/metrics/current", summary="Current Metrics Snapshot")
async def current_metrics() -> Dict[str, Any]:
    """
    Get current metrics snapshot in JSON format
    
    Returns:
    - Current workflow performance metrics
    - Phase-level statistics
    - Validation metrics
    - Performance target adherence
    """
    try:
        metrics_collector = get_metrics_collector()
        current_metrics = metrics_collector.get_current_metrics()
        return current_metrics
    except Exception as e:
        logger.error("Failed to get current metrics: %s", e)
        raise HTTPException(status_code=500, detail=f"Failed to get metrics: {str(e)}")

@router.get("/metrics/timeseries", summary="Time Series Metrics")
async def timeseries_metrics(
    metric_name: str = Query(..., description="Name of the metric to retrieve"),
    hours: int = Query(1, description="Number of hours of historical data", ge=1, le=24)
) -> Dict[str, Any]:
    """
    Get time series data for a specific metric
    
    Parameters:
    - metric_name: Name of the metric (e.g., 'calculate_duration_ms', 'validate_duration_ms')
    - hours: Number of hours of historical data to retrieve (1-24)
    """
    try:
        metrics_collector = get_metrics_collector()
        timeseries_data = metrics_collector.get_time_series_data(metric_name, hours)
        
        return {
            "metric_name": metric_name,
            "time_range_hours": hours,
            "data_points": len(timeseries_data),
            "data": timeseries_data
        }
    except Exception as e:
        logger.error("Failed to get timeseries data for %s: %s", metric_name, e)
        raise HTTPException(status_code=500, detail=f"Failed to get timeseries data: {str(e)}")

@router.get("/alerts", summary="Active Alerts")
async def get_alerts() -> Dict[str, Any]:
    """
    Get active alerts based on current metrics
    
    Returns alerts for:
    - Performance target violations
    - High error rates
    - Validation issues
    - System degradation
    """
    try:
        metrics_collector = get_metrics_collector()
        alerts = metrics_collector.get_alerts()
        
        # Categorize alerts by severity
        critical_alerts = [alert for alert in alerts if alert.get('severity') == 'critical']
        warning_alerts = [alert for alert in alerts if alert.get('severity') == 'warning']
        
        return {
            "timestamp": datetime.now().isoformat(),
            "total_alerts": len(alerts),
            "critical_alerts": len(critical_alerts),
            "warning_alerts": len(warning_alerts),
            "alerts": {
                "critical": critical_alerts,
                "warning": warning_alerts
            }
        }
    except Exception as e:
        logger.error("Failed to get alerts: %s", e)
        raise HTTPException(status_code=500, detail=f"Failed to get alerts: {str(e)}")

@router.get("/status", summary="System Status Dashboard")
async def system_status() -> Dict[str, Any]:
    """
    System status dashboard data
    
    Combines health, metrics, and alert information for a comprehensive
    system status view suitable for dashboards and monitoring UIs.
    """
    try:
        # Get health status
        health_checker = get_health_checker()
        health_status = await health_checker.check_overall_health()
        
        # Get metrics
        metrics_collector = get_metrics_collector()
        current_metrics = metrics_collector.get_current_metrics()
        alerts = metrics_collector.get_alerts()
        
        # Calculate key performance indicators
        kpis = _calculate_kpis(current_metrics, health_status)
        
        return {
            "timestamp": datetime.now().isoformat(),
            "overall_status": health_status.get('status', 'unknown'),
            "health_summary": health_status.get('summary', {}),
            "kpis": kpis,
            "alerts_summary": {
                "total": len(alerts),
                "critical": len([a for a in alerts if a.get('severity') == 'critical']),
                "warning": len([a for a in alerts if a.get('severity') == 'warning'])
            },
            "services": {
                name: {
                    "status": service_data.get('status'),
                    "response_time_ms": service_data.get('response_time_ms')
                }
                for name, service_data in health_status.get('services', {}).items()
            },
            "performance": {
                "avg_processing_time_ms": current_metrics.get('workflow_metrics', {}).get('avg_processing_time_ms', 0),
                "performance_targets_met_percent": current_metrics.get('workflow_metrics', {}).get('performance_targets_met', 0),
                "total_requests": current_metrics.get('workflow_metrics', {}).get('total_requests', 0),
                "success_rate_percent": _calculate_success_rate(current_metrics.get('workflow_metrics', {}))
            }
        }
    except Exception as e:
        logger.error("Failed to get system status: %s", e)
        raise HTTPException(status_code=500, detail=f"Failed to get system status: {str(e)}")

@router.get("/performance", summary="Performance Dashboard")
async def performance_dashboard() -> Dict[str, Any]:
    """
    Performance-focused dashboard data
    
    Detailed performance metrics for monitoring workflow efficiency:
    - Phase timing breakdown
    - Performance target adherence
    - Throughput metrics
    - Trend analysis
    """
    try:
        metrics_collector = get_metrics_collector()
        current_metrics = metrics_collector.get_current_metrics()
        
        # Get recent performance trends
        phase_trends = {}
        for phase in ['calculate', 'validate', 'commit']:
            trend_data = metrics_collector.get_time_series_data(f"{phase}_duration_ms", hours=1)
            phase_trends[phase] = {
                "recent_data_points": len(trend_data),
                "avg_duration_ms": sum(point['value'] for point in trend_data) / len(trend_data) if trend_data else 0,
                "trend": _calculate_trend(trend_data)
            }
        
        workflow_metrics = current_metrics.get('workflow_metrics', {})
        phase_metrics = current_metrics.get('phase_metrics', {})
        
        return {
            "timestamp": datetime.now().isoformat(),
            "overview": {
                "total_requests": workflow_metrics.get('total_requests', 0),
                "avg_processing_time_ms": workflow_metrics.get('avg_processing_time_ms', 0),
                "performance_targets_met_percent": workflow_metrics.get('performance_targets_met', 0),
                "success_rate_percent": _calculate_success_rate(workflow_metrics)
            },
            "phase_performance": {
                phase_name: {
                    "avg_time_ms": phase_data.get('average_time_ms', 0),
                    "target_ms": phase_data.get('performance_target_ms', 0),
                    "meets_target": phase_data.get('meets_target', True),
                    "success_rate": phase_data.get('success_rate', 1.0),
                    "total_executions": phase_data.get('total_executions', 0)
                }
                for phase_name, phase_data in phase_metrics.items()
            },
            "trends": phase_trends,
            "performance_targets": current_metrics.get('performance_targets', {})
        }
    except Exception as e:
        logger.error("Failed to get performance dashboard: %s", e)
        raise HTTPException(status_code=500, detail=f"Failed to get performance data: {str(e)}")

def _calculate_kpis(metrics: Dict[str, Any], health_status: Dict[str, Any]) -> Dict[str, Any]:
    """Calculate key performance indicators"""
    workflow_metrics = metrics.get('workflow_metrics', {})
    
    total_requests = workflow_metrics.get('total_requests', 0)
    successful_workflows = workflow_metrics.get('successful_workflows', 0)
    failed_workflows = workflow_metrics.get('failed_workflows', 0)
    
    success_rate = (successful_workflows / total_requests * 100) if total_requests > 0 else 100
    error_rate = (failed_workflows / total_requests * 100) if total_requests > 0 else 0
    
    return {
        "success_rate_percent": round(success_rate, 2),
        "error_rate_percent": round(error_rate, 2),
        "avg_processing_time_ms": workflow_metrics.get('avg_processing_time_ms', 0),
        "performance_targets_met_percent": workflow_metrics.get('performance_targets_met', 0),
        "system_health_score": _calculate_health_score(health_status),
        "total_workflows_24h": total_requests  # In a real implementation, this would be filtered by time
    }

def _calculate_success_rate(workflow_metrics: Dict[str, Any]) -> float:
    """Calculate workflow success rate percentage"""
    total = workflow_metrics.get('total_requests', 0)
    successful = workflow_metrics.get('successful_workflows', 0)
    return (successful / total * 100) if total > 0 else 100.0

def _calculate_health_score(health_status: Dict[str, Any]) -> float:
    """Calculate overall health score (0-100)"""
    services = health_status.get('services', {})
    if not services:
        return 100.0
    
    scores = []
    for service_data in services.values():
        status = service_data.get('status', 'unknown')
        if status == 'healthy':
            scores.append(100)
        elif status == 'degraded':
            scores.append(70)
        elif status == 'unhealthy':
            scores.append(0)
        else:
            scores.append(50)  # unknown
    
    return sum(scores) / len(scores)

def _calculate_trend(data_points: List[Dict[str, Any]]) -> str:
    """Calculate trend direction from time series data"""
    if len(data_points) < 2:
        return "stable"
    
    # Simple trend calculation based on first and last values
    first_value = data_points[0]['value']
    last_value = data_points[-1]['value']
    
    change_percent = ((last_value - first_value) / first_value * 100) if first_value > 0 else 0
    
    if change_percent > 10:
        return "increasing"
    elif change_percent < -10:
        return "decreasing"
    else:
        return "stable"

@router.get("/config", summary="Monitoring Configuration")
async def monitoring_config() -> Dict[str, Any]:
    """
    Get monitoring system configuration
    
    Returns current monitoring settings and capabilities
    """
    try:
        metrics_collector = get_metrics_collector()
        
        return {
            "monitoring_enabled": True,
            "prometheus_enabled": metrics_collector.enable_prometheus,
            "health_checks_enabled": True,
            "performance_targets": metrics_collector.performance_targets,
            "service_version": getattr(settings, 'SERVICE_VERSION', 'unknown'),
            "environment": getattr(settings, 'ENVIRONMENT', 'development'),
            "endpoints": {
                "health": "/monitoring/health",
                "readiness": "/monitoring/health/ready",
                "liveness": "/monitoring/health/live",
                "metrics": "/monitoring/metrics",
                "alerts": "/monitoring/alerts",
                "status": "/monitoring/status"
            }
        }
    except Exception as e:
        logger.error("Failed to get monitoring config: %s", e)
        raise HTTPException(status_code=500, detail=f"Failed to get config: {str(e)}")

# Health check for the monitoring system itself
@router.get("/self-check", summary="Monitoring System Self-Check")
async def monitoring_self_check() -> Dict[str, Any]:
    """
    Self-check for the monitoring system itself
    
    Verifies that monitoring components are functioning correctly
    """
    try:
        start_time = datetime.now()
        
        # Test metrics collector
        metrics_collector = get_metrics_collector()
        metrics_health = metrics_collector.get_health_status()
        
        # Test health checker
        health_checker = get_health_checker()
        
        end_time = datetime.now()
        check_duration_ms = (end_time - start_time).total_seconds() * 1000
        
        return {
            "monitoring_status": "healthy",
            "timestamp": end_time.isoformat(),
            "check_duration_ms": check_duration_ms,
            "components": {
                "metrics_collector": {
                    "status": "healthy",
                    "prometheus_enabled": metrics_health.get('prometheus_enabled', False)
                },
                "health_checker": {
                    "status": "healthy",
                    "initialized": health_checker is not None
                }
            }
        }
    except Exception as e:
        logger.error("Monitoring self-check failed: %s", e)
        raise HTTPException(status_code=500, detail=f"Monitoring system unhealthy: {str(e)}")