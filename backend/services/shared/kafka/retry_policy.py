"""
Retry policies and circuit breaker for Kafka operations
"""

import time
import logging
import random
from typing import Callable, Any, Optional, Dict
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass
import threading

logger = logging.getLogger(__name__)

class RetryStrategy(str, Enum):
    """Retry strategy types"""
    FIXED_DELAY = "fixed_delay"
    EXPONENTIAL_BACKOFF = "exponential_backoff"
    LINEAR_BACKOFF = "linear_backoff"
    RANDOM_JITTER = "random_jitter"

@dataclass
class RetryConfig:
    """Configuration for retry behavior"""
    max_attempts: int = 3
    base_delay: float = 1.0  # seconds
    max_delay: float = 60.0  # seconds
    strategy: RetryStrategy = RetryStrategy.EXPONENTIAL_BACKOFF
    jitter: bool = True
    backoff_multiplier: float = 2.0
    
    # Exceptions that should trigger retries
    retryable_exceptions: tuple = (
        ConnectionError,
        TimeoutError,
        Exception  # Catch-all for now
    )
    
    # Exceptions that should NOT trigger retries
    non_retryable_exceptions: tuple = (
        ValueError,
        TypeError,
        KeyError
    )

class CircuitBreakerState(str, Enum):
    """Circuit breaker states"""
    CLOSED = "closed"      # Normal operation
    OPEN = "open"          # Failing, reject requests
    HALF_OPEN = "half_open"  # Testing if service recovered

@dataclass
class CircuitBreakerConfig:
    """Configuration for circuit breaker"""
    failure_threshold: int = 5  # Number of failures to open circuit
    recovery_timeout: float = 60.0  # Seconds before trying half-open
    success_threshold: int = 3  # Successes needed to close circuit from half-open
    timeout: float = 30.0  # Operation timeout

class CircuitBreaker:
    """Circuit breaker implementation"""
    
    def __init__(self, config: CircuitBreakerConfig):
        self.config = config
        self.state = CircuitBreakerState.CLOSED
        self.failure_count = 0
        self.success_count = 0
        self.last_failure_time: Optional[datetime] = None
        self.lock = threading.Lock()
    
    def call(self, func: Callable, *args, **kwargs) -> Any:
        """Execute function with circuit breaker protection"""
        with self.lock:
            if self.state == CircuitBreakerState.OPEN:
                if self._should_attempt_reset():
                    self.state = CircuitBreakerState.HALF_OPEN
                    self.success_count = 0
                    logger.info("Circuit breaker transitioning to HALF_OPEN")
                else:
                    raise Exception("Circuit breaker is OPEN")
        
        try:
            result = func(*args, **kwargs)
            self._on_success()
            return result
        except Exception as e:
            self._on_failure()
            raise
    
    def _should_attempt_reset(self) -> bool:
        """Check if enough time has passed to attempt reset"""
        if self.last_failure_time is None:
            return True
        
        time_since_failure = datetime.now() - self.last_failure_time
        return time_since_failure.total_seconds() >= self.config.recovery_timeout
    
    def _on_success(self):
        """Handle successful operation"""
        with self.lock:
            if self.state == CircuitBreakerState.HALF_OPEN:
                self.success_count += 1
                if self.success_count >= self.config.success_threshold:
                    self.state = CircuitBreakerState.CLOSED
                    self.failure_count = 0
                    logger.info("Circuit breaker CLOSED after successful recovery")
            elif self.state == CircuitBreakerState.CLOSED:
                self.failure_count = 0
    
    def _on_failure(self):
        """Handle failed operation"""
        with self.lock:
            self.failure_count += 1
            self.last_failure_time = datetime.now()
            
            if self.state == CircuitBreakerState.CLOSED:
                if self.failure_count >= self.config.failure_threshold:
                    self.state = CircuitBreakerState.OPEN
                    logger.warning(f"Circuit breaker OPENED after {self.failure_count} failures")
            elif self.state == CircuitBreakerState.HALF_OPEN:
                self.state = CircuitBreakerState.OPEN
                logger.warning("Circuit breaker returned to OPEN from HALF_OPEN")
    
    def get_state(self) -> Dict[str, Any]:
        """Get current circuit breaker state"""
        return {
            'state': self.state.value,
            'failure_count': self.failure_count,
            'success_count': self.success_count,
            'last_failure_time': self.last_failure_time.isoformat() if self.last_failure_time else None
        }

class RetryHandler:
    """Handles retry logic with various strategies"""
    
    def __init__(self, config: RetryConfig):
        self.config = config
    
    def execute_with_retry(self, func: Callable, *args, **kwargs) -> Any:
        """Execute function with retry logic"""
        last_exception = None
        
        for attempt in range(self.config.max_attempts):
            try:
                return func(*args, **kwargs)
            except self.config.non_retryable_exceptions as e:
                logger.error(f"Non-retryable exception: {e}")
                raise
            except self.config.retryable_exceptions as e:
                last_exception = e
                
                if attempt == self.config.max_attempts - 1:
                    logger.error(f"Max retry attempts ({self.config.max_attempts}) exceeded")
                    break
                
                delay = self._calculate_delay(attempt)
                logger.warning(f"Attempt {attempt + 1} failed: {e}. Retrying in {delay:.2f}s")
                time.sleep(delay)
        
        # If we get here, all retries failed
        raise last_exception
    
    def _calculate_delay(self, attempt: int) -> float:
        """Calculate delay for next retry attempt"""
        if self.config.strategy == RetryStrategy.FIXED_DELAY:
            delay = self.config.base_delay
        elif self.config.strategy == RetryStrategy.EXPONENTIAL_BACKOFF:
            delay = self.config.base_delay * (self.config.backoff_multiplier ** attempt)
        elif self.config.strategy == RetryStrategy.LINEAR_BACKOFF:
            delay = self.config.base_delay * (attempt + 1)
        elif self.config.strategy == RetryStrategy.RANDOM_JITTER:
            delay = self.config.base_delay + random.uniform(0, self.config.base_delay)
        else:
            delay = self.config.base_delay
        
        # Apply jitter if enabled
        if self.config.jitter and self.config.strategy != RetryStrategy.RANDOM_JITTER:
            jitter = random.uniform(0.1, 0.3) * delay
            delay += jitter
        
        # Ensure delay doesn't exceed max_delay
        return min(delay, self.config.max_delay)

class ResilientKafkaOperation:
    """Combines retry logic and circuit breaker for Kafka operations"""
    
    def __init__(
        self,
        retry_config: Optional[RetryConfig] = None,
        circuit_breaker_config: Optional[CircuitBreakerConfig] = None
    ):
        self.retry_config = retry_config or RetryConfig()
        self.circuit_breaker_config = circuit_breaker_config or CircuitBreakerConfig()
        
        self.retry_handler = RetryHandler(self.retry_config)
        self.circuit_breaker = CircuitBreaker(self.circuit_breaker_config)
    
    def execute(self, func: Callable, *args, **kwargs) -> Any:
        """Execute function with both retry and circuit breaker protection"""
        def wrapped_func():
            return self.circuit_breaker.call(func, *args, **kwargs)
        
        return self.retry_handler.execute_with_retry(wrapped_func)
    
    def get_status(self) -> Dict[str, Any]:
        """Get status of retry handler and circuit breaker"""
        return {
            'retry_config': {
                'max_attempts': self.retry_config.max_attempts,
                'strategy': self.retry_config.strategy.value,
                'base_delay': self.retry_config.base_delay,
                'max_delay': self.retry_config.max_delay
            },
            'circuit_breaker': self.circuit_breaker.get_state()
        }

# Default configurations for different scenarios
DEFAULT_PRODUCER_RETRY_CONFIG = RetryConfig(
    max_attempts=3,
    base_delay=1.0,
    max_delay=30.0,
    strategy=RetryStrategy.EXPONENTIAL_BACKOFF,
    jitter=True
)

DEFAULT_CONSUMER_RETRY_CONFIG = RetryConfig(
    max_attempts=5,
    base_delay=0.5,
    max_delay=60.0,
    strategy=RetryStrategy.EXPONENTIAL_BACKOFF,
    jitter=True
)

DEFAULT_CIRCUIT_BREAKER_CONFIG = CircuitBreakerConfig(
    failure_threshold=5,
    recovery_timeout=60.0,
    success_threshold=3,
    timeout=30.0
)

# Factory functions for common configurations
def create_producer_resilient_operation() -> ResilientKafkaOperation:
    """Create resilient operation for producers"""
    return ResilientKafkaOperation(
        retry_config=DEFAULT_PRODUCER_RETRY_CONFIG,
        circuit_breaker_config=DEFAULT_CIRCUIT_BREAKER_CONFIG
    )

def create_consumer_resilient_operation() -> ResilientKafkaOperation:
    """Create resilient operation for consumers"""
    return ResilientKafkaOperation(
        retry_config=DEFAULT_CONSUMER_RETRY_CONFIG,
        circuit_breaker_config=DEFAULT_CIRCUIT_BREAKER_CONFIG
    )

def create_admin_resilient_operation() -> ResilientKafkaOperation:
    """Create resilient operation for admin operations"""
    admin_retry_config = RetryConfig(
        max_attempts=3,
        base_delay=2.0,
        max_delay=60.0,
        strategy=RetryStrategy.EXPONENTIAL_BACKOFF
    )
    
    admin_circuit_breaker_config = CircuitBreakerConfig(
        failure_threshold=3,
        recovery_timeout=120.0,
        success_threshold=2
    )
    
    return ResilientKafkaOperation(
        retry_config=admin_retry_config,
        circuit_breaker_config=admin_circuit_breaker_config
    )
