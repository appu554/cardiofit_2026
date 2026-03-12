"""
Comprehensive Health Check System for Calculate > Validate > Commit Workflow

Provides deep health monitoring for all workflow components:
- Workflow Engine Service health
- Safety Gateway connectivity
- Database connections (PostgreSQL for proposals)
- External service dependencies
- Performance indicators
"""

import asyncio
import logging
import time
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass
import httpx
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy import text

from app.config import settings
from app.repositories.workflow_proposal_repository import get_workflow_proposal_repository
from .workflow_metrics import get_metrics_collector

logger = logging.getLogger(__name__)

@dataclass
class ServiceHealthStatus:
    """Health status for an individual service"""
    service_name: str
    status: str  # 'healthy', 'degraded', 'unhealthy', 'unknown'
    response_time_ms: Optional[float] = None
    error_message: Optional[str] = None
    last_check: Optional[datetime] = None
    version: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            'service_name': self.service_name,
            'status': self.status,
            'response_time_ms': self.response_time_ms,
            'error_message': self.error_message,
            'last_check': self.last_check.isoformat() if self.last_check else None,
            'version': self.version,
            'metadata': self.metadata or {}
        }

class WorkflowHealthChecker:
    """
    Comprehensive health checker for the Calculate > Validate > Commit workflow
    
    Monitors:
    - Core workflow engine functionality
    - Safety Gateway HTTP endpoints
    - Database connectivity and performance
    - External service dependencies
    - System resource utilization
    """
    
    def __init__(self):
        """Initialize health checker"""
        self.http_client = httpx.AsyncClient(timeout=10.0)
        self.health_cache = {}
        self.cache_ttl = 30  # seconds
        
        # Service endpoints
        self.safety_gateway_url = getattr(settings, 'SAFETY_GATEWAY_URL', 'http://localhost:8080')
        self.medication_service_url = getattr(settings, 'MEDICATION_SERVICE_URL', 'http://localhost:8004')
        
        logger.info("Workflow Health Checker initialized")
    
    async def check_overall_health(self) -> Dict[str, Any]:
        """
        Perform comprehensive health check of all workflow components
        
        Returns overall system health with detailed component status
        """
        start_time = time.time()
        
        # Run all health checks concurrently
        checks = await asyncio.gather(
            self.check_workflow_engine_health(),
            self.check_safety_gateway_health(),
            self.check_database_health(),
            self.check_medication_service_health(),
            self.check_metrics_health(),
            return_exceptions=True
        )
        
        # Process results
        workflow_health, safety_gateway_health, database_health, medication_health, metrics_health = checks
        
        # Handle any exceptions
        service_statuses = {}
        for name, result in [
            ('workflow_engine', workflow_health),
            ('safety_gateway', safety_gateway_health),
            ('database', database_health),
            ('medication_service', medication_health),
            ('metrics', metrics_health)
        ]:
            if isinstance(result, Exception):
                service_statuses[name] = ServiceHealthStatus(
                    service_name=name,
                    status='unhealthy',
                    error_message=str(result),
                    last_check=datetime.now(timezone.utc)
                )
            else:
                service_statuses[name] = result
        
        # Determine overall status
        overall_status = self._calculate_overall_status(service_statuses)
        
        total_time_ms = (time.time() - start_time) * 1000
        
        return {
            'status': overall_status,
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'check_duration_ms': total_time_ms,
            'services': {name: status.to_dict() for name, status in service_statuses.items()},
            'summary': self._generate_health_summary(service_statuses),
            'version': getattr(settings, 'SERVICE_VERSION', 'unknown')
        }
    
    async def check_workflow_engine_health(self) -> ServiceHealthStatus:
        """Check core workflow engine health"""
        start_time = time.time()
        
        try:
            # Test workflow engine internal components
            metrics_collector = get_metrics_collector()
            metrics_health = metrics_collector.get_health_status()
            
            # Check for critical issues
            error_rate = metrics_health.get('error_rate_percent', 0)
            avg_processing_time = metrics_health.get('avg_processing_time_ms', 0)
            
            status = 'healthy'
            if error_rate > 10:
                status = 'unhealthy'
            elif error_rate > 5 or avg_processing_time > 500:  # 500ms threshold
                status = 'degraded'
            
            response_time_ms = (time.time() - start_time) * 1000
            
            return ServiceHealthStatus(
                service_name='workflow_engine',
                status=status,
                response_time_ms=response_time_ms,
                last_check=datetime.now(timezone.utc),
                version=getattr(settings, 'SERVICE_VERSION', 'unknown'),
                metadata={
                    'error_rate_percent': error_rate,
                    'avg_processing_time_ms': avg_processing_time,
                    'total_requests': metrics_health.get('total_requests', 0),
                    'prometheus_enabled': metrics_health.get('prometheus_enabled', False)
                }
            )
            
        except Exception as e:
            logger.error("Workflow engine health check failed: %s", e)
            return ServiceHealthStatus(
                service_name='workflow_engine',
                status='unhealthy',
                error_message=str(e),
                last_check=datetime.now(timezone.utc)
            )
    
    async def check_safety_gateway_health(self) -> ServiceHealthStatus:
        """Check Safety Gateway HTTP endpoint health"""
        start_time = time.time()
        
        try:
            # Test Safety Gateway health endpoint
            health_url = f"{self.safety_gateway_url}/health"
            response = await self.http_client.get(health_url)
            
            response_time_ms = (time.time() - start_time) * 1000
            
            if response.status_code == 200:
                health_data = response.json() if response.headers.get('content-type', '').startswith('application/json') else {}
                
                return ServiceHealthStatus(
                    service_name='safety_gateway',
                    status='healthy',
                    response_time_ms=response_time_ms,
                    last_check=datetime.now(timezone.utc),
                    version=health_data.get('version'),
                    metadata={
                        'validation_engines': health_data.get('validation_engines', []),
                        'gRPC_available': health_data.get('grpc_available', False),
                        'http_available': True
                    }
                )
            else:
                return ServiceHealthStatus(
                    service_name='safety_gateway',
                    status='degraded',
                    response_time_ms=response_time_ms,
                    error_message=f"HTTP {response.status_code}",
                    last_check=datetime.now(timezone.utc)
                )
                
        except Exception as e:
            logger.warning("Safety Gateway health check failed: %s", e)
            return ServiceHealthStatus(
                service_name='safety_gateway',
                status='unhealthy',
                error_message=str(e),
                last_check=datetime.now(timezone.utc)
            )
    
    async def check_database_health(self) -> ServiceHealthStatus:
        """Check PostgreSQL database health for proposal persistence"""
        start_time = time.time()
        
        try:
            # Get repository and test database connectivity
            repo = get_workflow_proposal_repository()
            session = repo.get_session()
            
            # Test query
            result = session.execute(text("SELECT 1 as health_check"))
            result.fetchone()
            session.close()
            
            response_time_ms = (time.time() - start_time) * 1000
            
            # Test proposal repository functionality
            stats = await repo.get_proposal_statistics()
            
            return ServiceHealthStatus(
                service_name='database',
                status='healthy',
                response_time_ms=response_time_ms,
                last_check=datetime.now(timezone.utc),
                metadata={
                    'total_proposals': stats.get('total_proposals', 0),
                    'recent_24h': stats.get('recent_24h', 0),
                    'avg_processing_time_ms': stats.get('avg_processing_time_ms', 0),
                    'status_distribution': stats.get('status_distribution', {})
                }
            )
            
        except SQLAlchemyError as e:
            logger.error("Database health check failed: %s", e)
            return ServiceHealthStatus(
                service_name='database',
                status='unhealthy',
                error_message=f"Database error: {str(e)}",
                last_check=datetime.now(timezone.utc)
            )
        except Exception as e:
            logger.error("Database health check failed: %s", e)
            return ServiceHealthStatus(
                service_name='database',
                status='unhealthy',
                error_message=str(e),
                last_check=datetime.now(timezone.utc)
            )
    
    async def check_medication_service_health(self) -> ServiceHealthStatus:
        """Check Medication Service health"""
        start_time = time.time()
        
        try:
            # Test medication service health endpoint
            health_url = f"{self.medication_service_url}/health"
            response = await self.http_client.get(health_url)
            
            response_time_ms = (time.time() - start_time) * 1000
            
            if response.status_code == 200:
                health_data = response.json() if response.headers.get('content-type', '').startswith('application/json') else {}
                
                return ServiceHealthStatus(
                    service_name='medication_service',
                    status='healthy',
                    response_time_ms=response_time_ms,
                    last_check=datetime.now(timezone.utc),
                    version=health_data.get('version'),
                    metadata={
                        'database_connected': health_data.get('database', {}).get('connected', False),
                        'fhir_service_available': health_data.get('fhir_service', {}).get('available', False)
                    }
                )
            else:
                return ServiceHealthStatus(
                    service_name='medication_service',
                    status='degraded',
                    response_time_ms=response_time_ms,
                    error_message=f"HTTP {response.status_code}",
                    last_check=datetime.now(timezone.utc)
                )
                
        except Exception as e:
            logger.warning("Medication Service health check failed: %s", e)
            return ServiceHealthStatus(
                service_name='medication_service',
                status='degraded',  # Non-critical for core workflow
                error_message=str(e),
                last_check=datetime.now(timezone.utc)
            )
    
    async def check_metrics_health(self) -> ServiceHealthStatus:
        """Check metrics collection system health"""
        start_time = time.time()
        
        try:
            metrics_collector = get_metrics_collector()
            metrics_status = metrics_collector.get_health_status()
            
            response_time_ms = (time.time() - start_time) * 1000
            
            # Determine status based on metrics health
            status = 'healthy'
            if not metrics_status.get('prometheus_enabled', False):
                status = 'degraded'
            
            return ServiceHealthStatus(
                service_name='metrics',
                status=status,
                response_time_ms=response_time_ms,
                last_check=datetime.now(timezone.utc),
                metadata=metrics_status
            )
            
        except Exception as e:
            logger.error("Metrics health check failed: %s", e)
            return ServiceHealthStatus(
                service_name='metrics',
                status='degraded',  # Non-critical for core functionality
                error_message=str(e),
                last_check=datetime.now(timezone.utc)
            )
    
    def _calculate_overall_status(self, service_statuses: Dict[str, ServiceHealthStatus]) -> str:
        """Calculate overall system status from individual service statuses"""
        critical_services = ['workflow_engine', 'safety_gateway', 'database']
        non_critical_services = ['medication_service', 'metrics']
        
        # Check critical services
        critical_unhealthy = []
        critical_degraded = []
        
        for service_name in critical_services:
            status = service_statuses.get(service_name)
            if status:
                if status.status == 'unhealthy':
                    critical_unhealthy.append(service_name)
                elif status.status == 'degraded':
                    critical_degraded.append(service_name)
        
        # Determine overall status
        if critical_unhealthy:
            return 'unhealthy'
        elif critical_degraded:
            return 'degraded'
        
        # Check if any non-critical services are unhealthy
        non_critical_issues = sum(
            1 for service_name in non_critical_services
            if service_statuses.get(service_name) and service_statuses[service_name].status in ['unhealthy', 'degraded']
        )
        
        if non_critical_issues > 0:
            return 'degraded'
        
        return 'healthy'
    
    def _generate_health_summary(self, service_statuses: Dict[str, ServiceHealthStatus]) -> Dict[str, Any]:
        """Generate health summary statistics"""
        status_counts = {'healthy': 0, 'degraded': 0, 'unhealthy': 0, 'unknown': 0}
        total_services = len(service_statuses)
        avg_response_time = 0
        response_times = []
        
        for status in service_statuses.values():
            status_counts[status.status] = status_counts.get(status.status, 0) + 1
            if status.response_time_ms:
                response_times.append(status.response_time_ms)
        
        if response_times:
            avg_response_time = sum(response_times) / len(response_times)
        
        return {
            'total_services': total_services,
            'healthy_services': status_counts['healthy'],
            'degraded_services': status_counts['degraded'],
            'unhealthy_services': status_counts['unhealthy'],
            'avg_response_time_ms': avg_response_time,
            'health_percentage': (status_counts['healthy'] / total_services) * 100 if total_services > 0 else 0
        }
    
    async def get_readiness_status(self) -> Dict[str, Any]:
        """
        Get readiness status - indicates if service is ready to handle requests
        
        More focused than health check - only checks critical dependencies
        """
        start_time = time.time()
        
        try:
            # Check only critical components for readiness
            database_ready = await self._check_database_readiness()
            safety_gateway_ready = await self._check_safety_gateway_readiness()
            
            ready = database_ready and safety_gateway_ready
            check_duration_ms = (time.time() - start_time) * 1000
            
            return {
                'ready': ready,
                'timestamp': datetime.now(timezone.utc).isoformat(),
                'check_duration_ms': check_duration_ms,
                'components': {
                    'database': database_ready,
                    'safety_gateway': safety_gateway_ready
                }
            }
            
        except Exception as e:
            logger.error("Readiness check failed: %s", e)
            return {
                'ready': False,
                'timestamp': datetime.now(timezone.utc).isoformat(),
                'error': str(e)
            }
    
    async def _check_database_readiness(self) -> bool:
        """Quick database readiness check"""
        try:
            repo = get_workflow_proposal_repository()
            session = repo.get_session()
            session.execute(text("SELECT 1"))
            session.close()
            return True
        except Exception as e:
            logger.error("Database readiness check failed: %s", e)
            return False
    
    async def _check_safety_gateway_readiness(self) -> bool:
        """Quick Safety Gateway readiness check"""
        try:
            health_url = f"{self.safety_gateway_url}/health"
            response = await self.http_client.get(health_url, timeout=5.0)
            return response.status_code == 200
        except Exception as e:
            logger.error("Safety Gateway readiness check failed: %s", e)
            return False
    
    async def close(self):
        """Clean up resources"""
        await self.http_client.aclose()


# Global health checker instance
_health_checker: Optional[WorkflowHealthChecker] = None

def get_health_checker() -> WorkflowHealthChecker:
    """Get global health checker instance"""
    global _health_checker
    if _health_checker is None:
        _health_checker = WorkflowHealthChecker()
    return _health_checker

async def cleanup_health_checker():
    """Clean up global health checker"""
    global _health_checker
    if _health_checker:
        await _health_checker.close()
        _health_checker = None