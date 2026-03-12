"""
Enhanced Circuit Breaker Manager for Device Data Ingestion Service

Provides service-specific circuit breaker instances with health check integration,
fallback strategy configuration, and comprehensive monitoring.
"""

import asyncio
import logging
import time
from datetime import datetime, timedelta
from enum import Enum
from typing import Dict, Any, Optional, Callable, List
from dataclasses import dataclass, field
import httpx
import json

logger = logging.getLogger(__name__)


class CircuitBreakerState(str, Enum):
    """Circuit breaker states"""
    CLOSED = "closed"      # Normal operation
    OPEN = "open"          # Failing, reject requests
    HALF_OPEN = "half_open"  # Testing if service recovered


class ServiceType(str, Enum):
    """Types of services that can be protected by circuit breakers"""
    AUTH_SERVICE = "auth_service"
    KAFKA_PRODUCER = "kafka_producer"
    GOOGLE_HEALTHCARE_API = "google_healthcare_api"
    PATIENT_SERVICE = "patient_service"
    FHIR_SERVICE = "fhir_service"


@dataclass
class CircuitBreakerConfig:
    """Configuration for circuit breaker behavior"""
    # Failure thresholds
    failure_threshold: int = 5  # Failures to open circuit
    success_threshold: int = 3  # Successes to close from half-open
    
    # Timing configuration
    recovery_timeout: float = 60.0  # Seconds before trying half-open
    operation_timeout: float = 30.0  # Individual operation timeout
    
    # Health check configuration
    health_check_enabled: bool = True
    health_check_interval: float = 30.0  # Seconds between health checks
    health_check_endpoint: Optional[str] = None
    
    # Fallback configuration
    fallback_enabled: bool = True
    fallback_strategy: str = "cached_response"  # cached_response, default_value, skip
    fallback_cache_ttl: float = 300.0  # 5 minutes
    
    # Monitoring
    metrics_enabled: bool = True
    alert_on_open: bool = True
    alert_on_half_open: bool = False


@dataclass
class CircuitBreakerMetrics:
    """Metrics for circuit breaker monitoring"""
    total_requests: int = 0
    successful_requests: int = 0
    failed_requests: int = 0
    circuit_opens: int = 0
    circuit_closes: int = 0
    fallback_executions: int = 0
    last_failure_time: Optional[datetime] = None
    last_success_time: Optional[datetime] = None
    current_state: CircuitBreakerState = CircuitBreakerState.CLOSED
    state_change_history: List[Dict[str, Any]] = field(default_factory=list)


class EnhancedCircuitBreaker:
    """Enhanced circuit breaker with health checks and fallback strategies"""
    
    def __init__(self, service_name: str, config: CircuitBreakerConfig):
        self.service_name = service_name
        self.config = config
        self.state = CircuitBreakerState.CLOSED
        self.failure_count = 0
        self.success_count = 0
        self.last_failure_time: Optional[datetime] = None
        self.last_success_time: Optional[datetime] = None
        self.metrics = CircuitBreakerMetrics()
        self.fallback_cache: Dict[str, Any] = {}
        self.health_check_task: Optional[asyncio.Task] = None
        self._lock = asyncio.Lock()
        
        # Start health check if enabled
        if self.config.health_check_enabled and self.config.health_check_endpoint:
            self.health_check_task = asyncio.create_task(self._health_check_loop())
    
    async def call(self, func: Callable, *args, **kwargs) -> Any:
        """Execute function with circuit breaker protection"""
        async with self._lock:
            self.metrics.total_requests += 1
            
            # Check if circuit is open
            if self.state == CircuitBreakerState.OPEN:
                if self._should_attempt_reset():
                    await self._transition_to_half_open()
                else:
                    # Circuit is open, try fallback
                    if self.config.fallback_enabled:
                        return await self._execute_fallback(func, *args, **kwargs)
                    else:
                        raise CircuitBreakerOpenError(
                            f"Circuit breaker for {self.service_name} is OPEN"
                        )
        
        # Execute the function
        try:
            # Apply timeout
            if asyncio.iscoroutinefunction(func):
                result = await asyncio.wait_for(
                    func(*args, **kwargs),
                    timeout=self.config.operation_timeout
                )
            else:
                result = func(*args, **kwargs)
            
            await self._on_success(result)
            return result
            
        except Exception as e:
            await self._on_failure(e)
            
            # Try fallback if available
            if self.config.fallback_enabled:
                return await self._execute_fallback(func, *args, **kwargs)
            else:
                raise
    
    async def _on_success(self, result: Any):
        """Handle successful operation"""
        async with self._lock:
            self.metrics.successful_requests += 1
            self.metrics.last_success_time = datetime.now()
            self.last_success_time = datetime.now()
            
            if self.state == CircuitBreakerState.HALF_OPEN:
                self.success_count += 1
                if self.success_count >= self.config.success_threshold:
                    await self._transition_to_closed()
            elif self.state == CircuitBreakerState.CLOSED:
                # Reset failure count on success
                self.failure_count = 0
            
            # Cache successful result for fallback
            if self.config.fallback_strategy == "cached_response":
                cache_key = self._generate_cache_key()
                self.fallback_cache[cache_key] = {
                    "result": result,
                    "timestamp": datetime.now(),
                    "ttl": self.config.fallback_cache_ttl
                }
    
    async def _on_failure(self, exception: Exception):
        """Handle failed operation"""
        async with self._lock:
            self.metrics.failed_requests += 1
            self.metrics.last_failure_time = datetime.now()
            self.last_failure_time = datetime.now()
            self.failure_count += 1
            
            logger.warning(
                f"Circuit breaker {self.service_name} recorded failure: {exception}"
            )
            
            if self.state == CircuitBreakerState.CLOSED:
                if self.failure_count >= self.config.failure_threshold:
                    await self._transition_to_open()
            elif self.state == CircuitBreakerState.HALF_OPEN:
                await self._transition_to_open()
    
    async def _transition_to_open(self):
        """Transition circuit breaker to OPEN state"""
        old_state = self.state
        self.state = CircuitBreakerState.OPEN
        self.metrics.circuit_opens += 1
        self.metrics.current_state = self.state
        
        await self._record_state_change(old_state, self.state, "Failure threshold exceeded")
        
        if self.config.alert_on_open:
            await self._send_alert("CIRCUIT_BREAKER_OPEN", {
                "service": self.service_name,
                "failure_count": self.failure_count,
                "threshold": self.config.failure_threshold
            })
        
        logger.error(
            f"Circuit breaker {self.service_name} OPENED after {self.failure_count} failures"
        )
    
    async def _transition_to_half_open(self):
        """Transition circuit breaker to HALF_OPEN state"""
        old_state = self.state
        self.state = CircuitBreakerState.HALF_OPEN
        self.success_count = 0
        self.metrics.current_state = self.state
        
        await self._record_state_change(old_state, self.state, "Recovery timeout elapsed")
        
        if self.config.alert_on_half_open:
            await self._send_alert("CIRCUIT_BREAKER_HALF_OPEN", {
                "service": self.service_name,
                "recovery_attempt": True
            })
        
        logger.info(f"Circuit breaker {self.service_name} transitioning to HALF_OPEN")
    
    async def _transition_to_closed(self):
        """Transition circuit breaker to CLOSED state"""
        old_state = self.state
        self.state = CircuitBreakerState.CLOSED
        self.failure_count = 0
        self.success_count = 0
        self.metrics.circuit_closes += 1
        self.metrics.current_state = self.state
        
        await self._record_state_change(old_state, self.state, "Recovery successful")
        
        await self._send_alert("CIRCUIT_BREAKER_CLOSED", {
            "service": self.service_name,
            "recovery_successful": True
        })
        
        logger.info(f"Circuit breaker {self.service_name} CLOSED - service recovered")
    
    def _should_attempt_reset(self) -> bool:
        """Check if circuit breaker should attempt reset"""
        if not self.last_failure_time:
            return True
        
        time_since_failure = datetime.now() - self.last_failure_time
        return time_since_failure.total_seconds() >= self.config.recovery_timeout
    
    async def _execute_fallback(self, func: Callable, *args, **kwargs) -> Any:
        """Execute fallback strategy"""
        self.metrics.fallback_executions += 1
        
        if self.config.fallback_strategy == "cached_response":
            return await self._get_cached_response()
        elif self.config.fallback_strategy == "default_value":
            return await self._get_default_value(func)
        elif self.config.fallback_strategy == "skip":
            logger.warning(f"Skipping operation for {self.service_name} due to circuit breaker")
            return None
        else:
            raise CircuitBreakerOpenError(
                f"No fallback available for {self.service_name}"
            )
    
    async def _get_cached_response(self) -> Any:
        """Get cached response for fallback"""
        cache_key = self._generate_cache_key()
        cached_item = self.fallback_cache.get(cache_key)
        
        if cached_item:
            # Check if cache is still valid
            age = datetime.now() - cached_item["timestamp"]
            if age.total_seconds() <= cached_item["ttl"]:
                logger.info(f"Using cached response for {self.service_name}")
                return cached_item["result"]
        
        # No valid cache available
        raise CircuitBreakerOpenError(
            f"No cached response available for {self.service_name}"
        )
    
    async def _get_default_value(self, func: Callable) -> Any:
        """Get default value based on function type"""
        # This could be enhanced with function-specific defaults
        return {}
    
    def _generate_cache_key(self) -> str:
        """Generate cache key for fallback storage"""
        return f"{self.service_name}_default"
    
    async def _record_state_change(self, old_state: CircuitBreakerState, 
                                 new_state: CircuitBreakerState, reason: str):
        """Record state change for monitoring"""
        change_record = {
            "timestamp": datetime.now().isoformat(),
            "old_state": old_state.value,
            "new_state": new_state.value,
            "reason": reason,
            "failure_count": self.failure_count,
            "success_count": self.success_count
        }
        
        self.metrics.state_change_history.append(change_record)
        
        # Keep only last 100 state changes
        if len(self.metrics.state_change_history) > 100:
            self.metrics.state_change_history = self.metrics.state_change_history[-100:]
    
    async def _send_alert(self, alert_type: str, data: Dict[str, Any]):
        """Send alert for circuit breaker events"""
        # This would integrate with your alerting system
        logger.warning(f"ALERT: {alert_type} - {json.dumps(data)}")
    
    async def _health_check_loop(self):
        """Continuous health check loop"""
        while True:
            try:
                await asyncio.sleep(self.config.health_check_interval)
                
                if self.state == CircuitBreakerState.OPEN:
                    # Perform health check
                    is_healthy = await self._perform_health_check()
                    if is_healthy:
                        logger.info(f"Health check passed for {self.service_name}, attempting recovery")
                        async with self._lock:
                            await self._transition_to_half_open()
                
            except Exception as e:
                logger.error(f"Health check error for {self.service_name}: {e}")
    
    async def _perform_health_check(self) -> bool:
        """Perform health check on the service"""
        if not self.config.health_check_endpoint:
            return False
        
        try:
            async with httpx.AsyncClient(timeout=10.0) as client:
                response = await client.get(self.config.health_check_endpoint)
                return response.status_code == 200
        except Exception as e:
            logger.debug(f"Health check failed for {self.service_name}: {e}")
            return False
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get current metrics"""
        return {
            "service_name": self.service_name,
            "state": self.state.value,
            "total_requests": self.metrics.total_requests,
            "successful_requests": self.metrics.successful_requests,
            "failed_requests": self.metrics.failed_requests,
            "success_rate": (
                self.metrics.successful_requests / max(self.metrics.total_requests, 1)
            ) * 100,
            "circuit_opens": self.metrics.circuit_opens,
            "circuit_closes": self.metrics.circuit_closes,
            "fallback_executions": self.metrics.fallback_executions,
            "last_failure_time": self.metrics.last_failure_time.isoformat() if self.metrics.last_failure_time else None,
            "last_success_time": self.metrics.last_success_time.isoformat() if self.metrics.last_success_time else None,
            "failure_count": self.failure_count,
            "success_count": self.success_count
        }
    
    async def reset(self):
        """Manually reset circuit breaker to CLOSED state"""
        async with self._lock:
            old_state = self.state
            self.state = CircuitBreakerState.CLOSED
            self.failure_count = 0
            self.success_count = 0
            await self._record_state_change(old_state, self.state, "Manual reset")
            logger.info(f"Circuit breaker {self.service_name} manually reset to CLOSED")
    
    async def cleanup(self):
        """Cleanup resources"""
        if self.health_check_task:
            self.health_check_task.cancel()
            try:
                await self.health_check_task
            except asyncio.CancelledError:
                pass


class CircuitBreakerOpenError(Exception):
    """Exception raised when circuit breaker is open"""
    pass


class CircuitBreakerManager:
    """Manages multiple circuit breakers for different services"""

    def __init__(self):
        self.circuit_breakers: Dict[str, EnhancedCircuitBreaker] = {}
        self.default_configs: Dict[ServiceType, CircuitBreakerConfig] = {}
        self._setup_default_configs()

    def _setup_default_configs(self):
        """Setup default configurations for different service types"""
        self.default_configs = {
            ServiceType.AUTH_SERVICE: CircuitBreakerConfig(
                failure_threshold=3,
                recovery_timeout=30.0,
                operation_timeout=10.0,
                health_check_enabled=True,
                health_check_endpoint="http://localhost:8001/api/health",
                health_check_interval=30.0,
                fallback_enabled=True,
                fallback_strategy="default_value"
            ),
            ServiceType.KAFKA_PRODUCER: CircuitBreakerConfig(
                failure_threshold=5,
                recovery_timeout=60.0,
                operation_timeout=30.0,
                health_check_enabled=False,  # Kafka doesn't have HTTP health endpoint
                fallback_enabled=True,
                fallback_strategy="skip"  # Skip publishing on failure
            ),
            ServiceType.GOOGLE_HEALTHCARE_API: CircuitBreakerConfig(
                failure_threshold=3,
                recovery_timeout=120.0,  # Longer recovery for external API
                operation_timeout=60.0,
                health_check_enabled=False,  # No direct health endpoint
                fallback_enabled=True,
                fallback_strategy="cached_response"
            ),
            ServiceType.PATIENT_SERVICE: CircuitBreakerConfig(
                failure_threshold=3,
                recovery_timeout=45.0,
                operation_timeout=15.0,
                health_check_enabled=True,
                health_check_endpoint="http://localhost:8003/api/health",
                fallback_enabled=True,
                fallback_strategy="cached_response"
            ),
            ServiceType.FHIR_SERVICE: CircuitBreakerConfig(
                failure_threshold=3,
                recovery_timeout=60.0,
                operation_timeout=30.0,
                health_check_enabled=True,
                health_check_endpoint="http://localhost:8014/api/health",
                fallback_enabled=True,
                fallback_strategy="cached_response"
            )
        }

    def get_circuit_breaker(self, service_name: str,
                          service_type: ServiceType,
                          config: Optional[CircuitBreakerConfig] = None) -> EnhancedCircuitBreaker:
        """Get or create circuit breaker for a service"""
        if service_name not in self.circuit_breakers:
            # Use provided config or default for service type
            cb_config = config or self.default_configs.get(service_type, CircuitBreakerConfig())

            self.circuit_breakers[service_name] = EnhancedCircuitBreaker(
                service_name=service_name,
                config=cb_config
            )

            logger.info(f"Created circuit breaker for {service_name}")

        return self.circuit_breakers[service_name]

    async def call_with_circuit_breaker(self, service_name: str,
                                      service_type: ServiceType,
                                      func: Callable,
                                      *args, **kwargs) -> Any:
        """Execute function with circuit breaker protection"""
        circuit_breaker = self.get_circuit_breaker(service_name, service_type)
        return await circuit_breaker.call(func, *args, **kwargs)

    def get_all_metrics(self) -> Dict[str, Dict[str, Any]]:
        """Get metrics for all circuit breakers"""
        return {
            name: cb.get_metrics()
            for name, cb in self.circuit_breakers.items()
        }

    def get_service_metrics(self, service_name: str) -> Optional[Dict[str, Any]]:
        """Get metrics for a specific service"""
        if service_name in self.circuit_breakers:
            return self.circuit_breakers[service_name].get_metrics()
        return None

    async def reset_circuit_breaker(self, service_name: str):
        """Reset a specific circuit breaker"""
        if service_name in self.circuit_breakers:
            await self.circuit_breakers[service_name].reset()
            logger.info(f"Reset circuit breaker for {service_name}")
        else:
            logger.warning(f"Circuit breaker {service_name} not found")

    async def reset_all_circuit_breakers(self):
        """Reset all circuit breakers"""
        for name, cb in self.circuit_breakers.items():
            await cb.reset()
        logger.info("Reset all circuit breakers")

    def get_circuit_breaker_states(self) -> Dict[str, str]:
        """Get current states of all circuit breakers"""
        return {
            name: cb.state.value
            for name, cb in self.circuit_breakers.items()
        }

    def get_open_circuit_breakers(self) -> List[str]:
        """Get list of circuit breakers that are currently open"""
        return [
            name for name, cb in self.circuit_breakers.items()
            if cb.state == CircuitBreakerState.OPEN
        ]

    async def cleanup(self):
        """Cleanup all circuit breakers"""
        for cb in self.circuit_breakers.values():
            await cb.cleanup()
        self.circuit_breakers.clear()
        logger.info("Cleaned up all circuit breakers")


# Global circuit breaker manager instance
circuit_breaker_manager = CircuitBreakerManager()
