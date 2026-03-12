"""
Intelligent Circuit Breaker with Learning Capabilities.
Implements adaptive circuit breaker patterns with performance prediction and optimization.
"""
import logging
import asyncio
import time
from typing import Dict, Any, List, Optional, Callable
from datetime import datetime, timedelta
from dataclasses import dataclass, field
from enum import Enum
import json
import statistics

logger = logging.getLogger(__name__)


class CircuitState(Enum):
    """Enhanced circuit breaker states."""
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"
    LEARNING = "learning"


class FailurePattern(Enum):
    """Types of failure patterns detected."""
    TIMEOUT = "timeout"
    ERROR_RATE = "error_rate"
    LATENCY_SPIKE = "latency_spike"
    RESOURCE_EXHAUSTION = "resource_exhaustion"
    CASCADING_FAILURE = "cascading_failure"


@dataclass
class CircuitBreakerConfig:
    """Configuration for intelligent circuit breaker."""
    service_name: str
    failure_threshold: int = 5
    recovery_timeout_ms: int = 30000
    half_open_max_calls: int = 3
    learning_window_size: int = 100
    latency_threshold_ms: float = 1000
    error_rate_threshold: float = 0.1
    learning_enabled: bool = True
    adaptive_thresholds: bool = True


@dataclass
class OperationResult:
    """Result of a circuit breaker protected operation."""
    success: bool
    execution_time_ms: float
    error_type: Optional[str] = None
    error_message: Optional[str] = None
    timestamp: datetime = field(default_factory=datetime.utcnow)
    context: Dict[str, Any] = field(default_factory=dict)


@dataclass
class FailureAnalysis:
    """Analysis of failure patterns."""
    pattern_type: FailurePattern
    confidence: float
    contributing_factors: List[str]
    recommended_action: str
    threshold_adjustment: Optional[Dict[str, float]] = None


class IntelligentCircuitBreaker:
    """
    Intelligent Circuit Breaker with learning and adaptation capabilities.
    
    Features:
    - Adaptive failure thresholds based on historical data
    - Failure pattern recognition and prediction
    - Context-aware circuit breaking decisions
    - Performance optimization through learning
    - Cascading failure prevention
    """
    
    def __init__(self, config: CircuitBreakerConfig):
        self.config = config
        self.state = CircuitState.CLOSED
        self.failure_count = 0
        self.success_count = 0
        self.half_open_calls = 0
        self.last_failure_time: Optional[datetime] = None
        self.last_success_time: Optional[datetime] = None
        
        # Learning and adaptation
        self.operation_history: List[OperationResult] = []
        self.failure_patterns: List[FailureAnalysis] = []
        self.adaptive_thresholds = {
            'failure_threshold': config.failure_threshold,
            'latency_threshold_ms': config.latency_threshold_ms,
            'error_rate_threshold': config.error_rate_threshold
        }
        
        # Performance metrics
        self.total_requests = 0
        self.blocked_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        
        # Context tracking
        self.context_patterns: Dict[str, Dict[str, Any]] = {}
        
        # Start learning process
        if config.learning_enabled:
            asyncio.create_task(self._start_learning_process())
        
        logger.info(f"✅ Intelligent Circuit Breaker initialized for {config.service_name}")
    
    async def execute(
        self,
        operation: Callable,
        context: Optional[Dict[str, Any]] = None,
        timeout_ms: Optional[float] = None
    ) -> Any:
        """
        Execute operation with intelligent circuit breaker protection.
        """
        self.total_requests += 1
        context = context or {}
        
        # Check circuit state and make intelligent decision
        if await self._should_block_request(context):
            self.blocked_requests += 1
            raise CircuitBreakerOpenError(f"Circuit breaker is OPEN for {self.config.service_name}")
        
        # Predict failure risk if learning is enabled
        if self.config.learning_enabled:
            failure_risk = await self._predict_failure_risk(context)
            if failure_risk > 0.8:
                logger.warning(f"High failure risk detected ({failure_risk:.3f}) for {self.config.service_name}")
                # Could implement preemptive action here
        
        # Execute operation with monitoring
        start_time = time.time()
        result = None
        success = False
        error_type = None
        error_message = None
        
        try:
            # Apply timeout if specified
            if timeout_ms:
                result = await asyncio.wait_for(operation(), timeout=timeout_ms / 1000)
            else:
                result = await operation()
            
            success = True
            execution_time_ms = (time.time() - start_time) * 1000
            
            # Record successful operation
            await self._record_success(execution_time_ms, context)
            
            return result
            
        except asyncio.TimeoutError:
            execution_time_ms = (time.time() - start_time) * 1000
            error_type = "timeout"
            error_message = f"Operation timed out after {timeout_ms}ms"
            await self._record_failure(execution_time_ms, error_type, error_message, context)
            raise
            
        except Exception as e:
            execution_time_ms = (time.time() - start_time) * 1000
            error_type = type(e).__name__
            error_message = str(e)
            await self._record_failure(execution_time_ms, error_type, error_message, context)
            raise
        
        finally:
            # Record operation result for learning
            if self.config.learning_enabled:
                operation_result = OperationResult(
                    success=success,
                    execution_time_ms=(time.time() - start_time) * 1000,
                    error_type=error_type,
                    error_message=error_message,
                    context=context
                )
                await self._record_operation_result(operation_result)
    
    async def _should_block_request(self, context: Dict[str, Any]) -> bool:
        """Intelligent decision on whether to block request."""
        if self.state == CircuitState.CLOSED:
            return False
        
        elif self.state == CircuitState.OPEN:
            # Check if recovery timeout has passed
            if (self.last_failure_time and 
                (datetime.utcnow() - self.last_failure_time).total_seconds() * 1000 > self.config.recovery_timeout_ms):
                await self._transition_to_half_open()
                return False
            return True
        
        elif self.state == CircuitState.HALF_OPEN:
            # Allow limited requests in half-open state
            if self.half_open_calls >= self.config.half_open_max_calls:
                return True
            self.half_open_calls += 1
            return False
        
        elif self.state == CircuitState.LEARNING:
            # In learning state, allow requests but monitor closely
            return False
        
        return False
    
    async def _predict_failure_risk(self, context: Dict[str, Any]) -> float:
        """Predict failure risk based on context and historical data."""
        if len(self.operation_history) < 10:
            return 0.0
        
        risk_factors = []
        
        # Analyze recent failure rate
        recent_operations = self.operation_history[-20:]
        recent_failure_rate = sum(1 for op in recent_operations if not op.success) / len(recent_operations)
        risk_factors.append(recent_failure_rate)
        
        # Analyze latency trends
        recent_latencies = [op.execution_time_ms for op in recent_operations if op.success]
        if recent_latencies:
            avg_latency = statistics.mean(recent_latencies)
            latency_risk = min(avg_latency / self.adaptive_thresholds['latency_threshold_ms'], 1.0)
            risk_factors.append(latency_risk)
        
        # Context-based risk assessment
        context_risk = await self._assess_context_risk(context)
        risk_factors.append(context_risk)
        
        # Time-based risk (higher risk during known problematic periods)
        time_risk = await self._assess_time_based_risk()
        risk_factors.append(time_risk)
        
        # Calculate weighted average risk
        weights = [0.4, 0.3, 0.2, 0.1]  # Prioritize recent failure rate
        weighted_risk = sum(risk * weight for risk, weight in zip(risk_factors, weights))
        
        return min(weighted_risk, 1.0)
    
    async def _assess_context_risk(self, context: Dict[str, Any]) -> float:
        """Assess risk based on request context."""
        if not context:
            return 0.0
        
        risk = 0.0
        
        # Check for known problematic context patterns
        for pattern_key, pattern_data in self.context_patterns.items():
            if pattern_key in context:
                context_value = str(context[pattern_key])
                if context_value in pattern_data:
                    failure_rate = pattern_data[context_value].get('failure_rate', 0.0)
                    risk = max(risk, failure_rate)
        
        return min(risk, 1.0)
    
    async def _assess_time_based_risk(self) -> float:
        """Assess risk based on time patterns."""
        current_hour = datetime.utcnow().hour
        
        # Analyze historical failure patterns by hour
        hourly_failures = {}
        for op in self.operation_history:
            hour = op.timestamp.hour
            if hour not in hourly_failures:
                hourly_failures[hour] = {'total': 0, 'failures': 0}
            
            hourly_failures[hour]['total'] += 1
            if not op.success:
                hourly_failures[hour]['failures'] += 1
        
        if current_hour in hourly_failures and hourly_failures[current_hour]['total'] > 5:
            failure_rate = hourly_failures[current_hour]['failures'] / hourly_failures[current_hour]['total']
            return failure_rate
        
        return 0.0
    
    async def _record_success(self, execution_time_ms: float, context: Dict[str, Any]):
        """Record successful operation and update circuit state."""
        self.successful_requests += 1
        self.success_count += 1
        self.last_success_time = datetime.utcnow()
        
        # Update context patterns
        await self._update_context_patterns(context, success=True)
        
        # Handle state transitions
        if self.state == CircuitState.HALF_OPEN:
            # Successful call in half-open state
            if self.success_count >= 2:  # Require multiple successes
                await self._transition_to_closed()
        
        elif self.state == CircuitState.LEARNING:
            # Continue learning but consider transitioning to closed
            if self.success_count > self.failure_count * 2:
                await self._transition_to_closed()
    
    async def _record_failure(
        self,
        execution_time_ms: float,
        error_type: str,
        error_message: str,
        context: Dict[str, Any]
    ):
        """Record failed operation and update circuit state."""
        self.failed_requests += 1
        self.failure_count += 1
        self.last_failure_time = datetime.utcnow()
        
        # Update context patterns
        await self._update_context_patterns(context, success=False)
        
        # Analyze failure pattern
        if self.config.learning_enabled:
            await self._analyze_failure_pattern(error_type, error_message, context)
        
        # Handle state transitions
        if self.state == CircuitState.CLOSED:
            if self.failure_count >= self.adaptive_thresholds['failure_threshold']:
                await self._transition_to_open()
        
        elif self.state == CircuitState.HALF_OPEN:
            # Any failure in half-open state transitions back to open
            await self._transition_to_open()
        
        elif self.state == CircuitState.LEARNING:
            # In learning state, be more tolerant but still protect
            if self.failure_count >= self.adaptive_thresholds['failure_threshold'] * 2:
                await self._transition_to_open()
    
    async def _update_context_patterns(self, context: Dict[str, Any], success: bool):
        """Update context-based failure patterns."""
        for key, value in context.items():
            if key not in self.context_patterns:
                self.context_patterns[key] = {}
            
            value_str = str(value)
            if value_str not in self.context_patterns[key]:
                self.context_patterns[key][value_str] = {'total': 0, 'failures': 0}
            
            self.context_patterns[key][value_str]['total'] += 1
            if not success:
                self.context_patterns[key][value_str]['failures'] += 1
            
            # Calculate failure rate
            pattern_data = self.context_patterns[key][value_str]
            pattern_data['failure_rate'] = pattern_data['failures'] / pattern_data['total']
    
    async def _analyze_failure_pattern(
        self,
        error_type: str,
        error_message: str,
        context: Dict[str, Any]
    ):
        """Analyze failure patterns and recommend adaptations."""
        # Determine failure pattern type
        pattern_type = FailurePattern.ERROR_RATE  # Default
        
        if error_type == "TimeoutError":
            pattern_type = FailurePattern.TIMEOUT
        elif "resource" in error_message.lower():
            pattern_type = FailurePattern.RESOURCE_EXHAUSTION
        elif len(self.operation_history) > 10:
            recent_failures = [op for op in self.operation_history[-10:] if not op.success]
            if len(recent_failures) > 7:  # High recent failure rate
                pattern_type = FailurePattern.CASCADING_FAILURE
        
        # Create failure analysis
        analysis = FailureAnalysis(
            pattern_type=pattern_type,
            confidence=0.8,  # Could be more sophisticated
            contributing_factors=[error_type, f"context: {context}"],
            recommended_action=self._get_recommended_action(pattern_type)
        )
        
        self.failure_patterns.append(analysis)
        
        # Apply adaptive threshold adjustments
        if self.config.adaptive_thresholds:
            await self._apply_threshold_adjustments(analysis)
    
    def _get_recommended_action(self, pattern_type: FailurePattern) -> str:
        """Get recommended action for failure pattern."""
        recommendations = {
            FailurePattern.TIMEOUT: "Increase timeout threshold or reduce operation complexity",
            FailurePattern.ERROR_RATE: "Investigate error causes and improve error handling",
            FailurePattern.LATENCY_SPIKE: "Optimize performance or increase latency threshold",
            FailurePattern.RESOURCE_EXHAUSTION: "Scale resources or implement backpressure",
            FailurePattern.CASCADING_FAILURE: "Implement bulkhead pattern and reduce failure threshold"
        }
        return recommendations.get(pattern_type, "Monitor and investigate")
    
    async def _apply_threshold_adjustments(self, analysis: FailureAnalysis):
        """Apply adaptive threshold adjustments based on failure analysis."""
        if analysis.pattern_type == FailurePattern.CASCADING_FAILURE:
            # Be more aggressive with failure threshold
            self.adaptive_thresholds['failure_threshold'] = max(2, self.adaptive_thresholds['failure_threshold'] - 1)
            logger.info(f"Reduced failure threshold to {self.adaptive_thresholds['failure_threshold']} due to cascading failures")
        
        elif analysis.pattern_type == FailurePattern.TIMEOUT:
            # Increase timeout threshold
            self.adaptive_thresholds['latency_threshold_ms'] *= 1.2
            logger.info(f"Increased latency threshold to {self.adaptive_thresholds['latency_threshold_ms']:.1f}ms due to timeouts")
    
    async def _transition_to_open(self):
        """Transition circuit breaker to OPEN state."""
        old_state = self.state
        self.state = CircuitState.OPEN
        self.half_open_calls = 0
        logger.error(f"🚨 Circuit breaker OPEN: {self.config.service_name} (failures: {self.failure_count})")
        
        # In production: Send alert
        await self._send_state_change_alert(old_state, self.state)
    
    async def _transition_to_half_open(self):
        """Transition circuit breaker to HALF_OPEN state."""
        old_state = self.state
        self.state = CircuitState.HALF_OPEN
        self.half_open_calls = 0
        self.success_count = 0
        logger.info(f"🔄 Circuit breaker HALF_OPEN: {self.config.service_name}")
        
        await self._send_state_change_alert(old_state, self.state)
    
    async def _transition_to_closed(self):
        """Transition circuit breaker to CLOSED state."""
        old_state = self.state
        self.state = CircuitState.CLOSED
        self.failure_count = 0
        self.success_count = 0
        logger.info(f"✅ Circuit breaker CLOSED: {self.config.service_name}")
        
        await self._send_state_change_alert(old_state, self.state)
    
    async def _record_operation_result(self, result: OperationResult):
        """Record operation result for learning."""
        self.operation_history.append(result)
        
        # Keep only recent history
        if len(self.operation_history) > self.config.learning_window_size:
            self.operation_history = self.operation_history[-self.config.learning_window_size:]
    
    async def _start_learning_process(self):
        """Start background learning and optimization process."""
        while True:
            try:
                await asyncio.sleep(300)  # Learn every 5 minutes
                
                if len(self.operation_history) > 20:
                    await self._optimize_thresholds()
                    await self._detect_patterns()
                
            except Exception as e:
                logger.error(f"Learning process error for {self.config.service_name}: {e}")
                await asyncio.sleep(300)
    
    async def _optimize_thresholds(self):
        """Optimize circuit breaker thresholds based on learning."""
        if not self.config.adaptive_thresholds:
            return
        
        recent_ops = self.operation_history[-50:]
        if len(recent_ops) < 20:
            return
        
        # Calculate optimal failure threshold
        failure_rate = sum(1 for op in recent_ops if not op.success) / len(recent_ops)
        
        if failure_rate < 0.05:  # Very low failure rate
            # Can be more tolerant
            self.adaptive_thresholds['failure_threshold'] = min(10, self.adaptive_thresholds['failure_threshold'] + 1)
        elif failure_rate > 0.2:  # High failure rate
            # Be more aggressive
            self.adaptive_thresholds['failure_threshold'] = max(2, self.adaptive_thresholds['failure_threshold'] - 1)
    
    async def _detect_patterns(self):
        """Detect patterns in operation history."""
        # This could be enhanced with more sophisticated pattern detection
        # For now, just log insights
        recent_ops = self.operation_history[-30:]
        if len(recent_ops) < 10:
            return
        
        success_rate = sum(1 for op in recent_ops if op.success) / len(recent_ops)
        avg_latency = statistics.mean([op.execution_time_ms for op in recent_ops if op.success])
        
        logger.info(f"📊 {self.config.service_name} patterns: {success_rate:.1%} success, {avg_latency:.1f}ms avg latency")
    
    async def _send_state_change_alert(self, old_state: CircuitState, new_state: CircuitState):
        """Send alert for circuit breaker state changes."""
        try:
            # In production: Send to monitoring system
            logger.info(f"🔔 Circuit breaker state change: {self.config.service_name} {old_state.value} → {new_state.value}")
        except Exception as e:
            logger.error(f"Failed to send state change alert: {e}")
    
    def get_status(self) -> Dict[str, Any]:
        """Get current circuit breaker status."""
        return {
            'service_name': self.config.service_name,
            'state': self.state.value,
            'failure_count': self.failure_count,
            'success_count': self.success_count,
            'total_requests': self.total_requests,
            'blocked_requests': self.blocked_requests,
            'success_rate': round(self.successful_requests / max(self.total_requests, 1), 3),
            'adaptive_thresholds': self.adaptive_thresholds,
            'last_failure': self.last_failure_time.isoformat() if self.last_failure_time else None,
            'last_success': self.last_success_time.isoformat() if self.last_success_time else None,
            'failure_patterns_detected': len(self.failure_patterns)
        }


class CircuitBreakerOpenError(Exception):
    """Exception raised when circuit breaker is open."""
    pass
