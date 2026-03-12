"""
Intelligent Circuit Breaker with Learning for Clinical Assertion Engine

Implements intelligent circuit breakers that store failure patterns in graph
for learning and adaptive failure prevention.

Key Features:
- Adaptive circuit breaker with learning capabilities
- Failure pattern analysis and storage in graph
- Intelligent failure prediction
- Context-aware circuit breaker states
- Performance degradation detection
- Automatic recovery strategies
"""

import asyncio
import logging
import time
from typing import Dict, List, Optional, Any, Set, Callable
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta, timezone
from collections import defaultdict, deque
from enum import Enum
import json
import statistics
import hashlib

logger = logging.getLogger(__name__)


class CircuitState(Enum):
    """Circuit breaker states"""
    CLOSED = "closed"        # Normal operation
    OPEN = "open"           # Failing, blocking requests
    HALF_OPEN = "half_open" # Testing recovery


class FailureType(Enum):
    """Types of failures"""
    TIMEOUT = "timeout"
    ERROR = "error"
    PERFORMANCE_DEGRADATION = "performance_degradation"
    RESOURCE_EXHAUSTION = "resource_exhaustion"
    DEPENDENCY_FAILURE = "dependency_failure"


@dataclass
class FailureEvent:
    """Failure event record"""
    event_id: str
    service_name: str
    failure_type: FailureType
    timestamp: datetime
    context: Dict[str, Any]
    error_message: Optional[str]
    response_time_ms: Optional[float]
    request_context: Dict[str, Any]


@dataclass
class FailurePattern:
    """Discovered failure pattern"""
    pattern_id: str
    pattern_type: str
    frequency: int
    confidence_score: float
    context_conditions: List[str]
    temporal_conditions: List[str]
    predictive_indicators: List[str]
    mitigation_strategies: List[str]
    discovered_at: datetime
    last_seen: datetime


@dataclass
class CircuitBreakerConfig:
    """Circuit breaker configuration"""
    failure_threshold: int = 5
    recovery_timeout_ms: int = 60000
    success_threshold: int = 3
    timeout_ms: int = 5000
    performance_threshold_ms: float = 1000.0
    learning_enabled: bool = True
    adaptive_thresholds: bool = True


class IntelligentCircuitBreaker:
    """
    Intelligent circuit breaker with learning capabilities
    
    This circuit breaker learns from failure patterns and adapts its behavior
    to prevent failures before they occur, using graph intelligence to store
    and analyze failure patterns.
    """
    
    def __init__(self, service_name: str, config: Optional[CircuitBreakerConfig] = None):
        self.service_name = service_name
        self.config = config or CircuitBreakerConfig()
        
        # Circuit breaker state
        self.state = CircuitState.CLOSED
        self.failure_count = 0
        self.success_count = 0
        self.last_failure_time: Optional[datetime] = None
        self.state_changed_at = datetime.now(timezone.utc)
        
        # Failure tracking
        self.failure_events: deque = deque(maxlen=1000)
        self.failure_patterns: Dict[str, FailurePattern] = {}
        self.response_times: deque = deque(maxlen=100)
        
        # Learning components
        self.pattern_detector = FailurePatternDetector()
        self.predictor = FailurePredictor()
        
        # Performance metrics
        self.total_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        self.blocked_requests = 0
        
        # Adaptive thresholds
        self.adaptive_failure_threshold = self.config.failure_threshold
        self.adaptive_timeout_ms = self.config.timeout_ms
        
        logger.info(f"Intelligent Circuit Breaker initialized for {service_name}")
    
    async def call(self, func: Callable, *args, **kwargs) -> Any:
        """
        Execute function with circuit breaker protection
        
        Args:
            func: Function to execute
            *args: Function arguments
            **kwargs: Function keyword arguments
            
        Returns:
            Function result or raises CircuitBreakerOpenError
        """
        try:
            self.total_requests += 1
            
            # Check circuit state
            if self.state == CircuitState.OPEN:
                if await self._should_attempt_reset():
                    await self._transition_to_half_open()
                else:
                    self.blocked_requests += 1
                    raise CircuitBreakerOpenError(f"Circuit breaker is OPEN for {self.service_name}")
            
            # Predict failure before execution
            if self.config.learning_enabled:
                failure_risk = await self._predict_failure_risk(kwargs.get('context', {}))
                if failure_risk > 0.8:
                    logger.warning(f"High failure risk detected ({failure_risk:.3f}) for {self.service_name}")
                    # Could implement preemptive action here
            
            # Execute function with timeout
            start_time = time.time()
            try:
                result = await asyncio.wait_for(
                    func(*args, **kwargs),
                    timeout=self.adaptive_timeout_ms / 1000.0
                )
                
                # Record success
                execution_time = (time.time() - start_time) * 1000
                await self._record_success(execution_time, kwargs.get('context', {}))
                
                return result
                
            except asyncio.TimeoutError:
                execution_time = (time.time() - start_time) * 1000
                await self._record_failure(
                    FailureType.TIMEOUT,
                    f"Function timeout after {execution_time:.2f}ms",
                    execution_time,
                    kwargs.get('context', {})
                )
                raise
                
            except Exception as e:
                execution_time = (time.time() - start_time) * 1000
                await self._record_failure(
                    FailureType.ERROR,
                    str(e),
                    execution_time,
                    kwargs.get('context', {})
                )
                raise
                
        except CircuitBreakerOpenError:
            raise
        except Exception as e:
            logger.error(f"Circuit breaker error for {self.service_name}: {e}")
            raise
    
    async def _record_success(self, response_time_ms: float, context: Dict[str, Any]):
        """Record successful execution"""
        try:
            self.successful_requests += 1
            self.response_times.append(response_time_ms)
            
            # Check for performance degradation
            if response_time_ms > self.config.performance_threshold_ms:
                await self._record_failure(
                    FailureType.PERFORMANCE_DEGRADATION,
                    f"Slow response: {response_time_ms:.2f}ms",
                    response_time_ms,
                    context
                )
                return
            
            # Reset failure count on success
            if self.state == CircuitState.HALF_OPEN:
                self.success_count += 1
                if self.success_count >= self.config.success_threshold:
                    await self._transition_to_closed()
            else:
                self.failure_count = max(0, self.failure_count - 1)
            
            # Learn from success patterns
            if self.config.learning_enabled:
                await self._learn_from_success(response_time_ms, context)
                
        except Exception as e:
            logger.warning(f"Error recording success: {e}")
    
    async def _record_failure(
        self, 
        failure_type: FailureType, 
        error_message: str,
        response_time_ms: float,
        context: Dict[str, Any]
    ):
        """Record failure event"""
        try:
            self.failed_requests += 1
            self.failure_count += 1
            self.last_failure_time = datetime.now(timezone.utc)
            
            # Create failure event
            failure_event = FailureEvent(
                event_id=f"{self.service_name}_{int(time.time() * 1000)}",
                service_name=self.service_name,
                failure_type=failure_type,
                timestamp=self.last_failure_time,
                context=context,
                error_message=error_message,
                response_time_ms=response_time_ms,
                request_context=context
            )
            
            self.failure_events.append(failure_event)
            
            # Check if circuit should open
            if self.failure_count >= self.adaptive_failure_threshold:
                await self._transition_to_open()
            
            # Learn from failure patterns
            if self.config.learning_enabled:
                await self._learn_from_failure(failure_event)
                
        except Exception as e:
            logger.warning(f"Error recording failure: {e}")
    
    async def _transition_to_open(self):
        """Transition circuit breaker to OPEN state"""
        try:
            self.state = CircuitState.OPEN
            self.state_changed_at = datetime.now(timezone.utc)
            
            logger.warning(f"Circuit breaker OPENED for {self.service_name} "
                          f"(failures: {self.failure_count})")
            
            # Analyze failure patterns for learning
            if self.config.learning_enabled:
                await self._analyze_failure_patterns()
                
        except Exception as e:
            logger.warning(f"Error transitioning to open: {e}")
    
    async def _transition_to_half_open(self):
        """Transition circuit breaker to HALF_OPEN state"""
        try:
            self.state = CircuitState.HALF_OPEN
            self.state_changed_at = datetime.now(timezone.utc)
            self.success_count = 0
            
            logger.info(f"Circuit breaker HALF_OPEN for {self.service_name}")
            
        except Exception as e:
            logger.warning(f"Error transitioning to half-open: {e}")
    
    async def _transition_to_closed(self):
        """Transition circuit breaker to CLOSED state"""
        try:
            self.state = CircuitState.CLOSED
            self.state_changed_at = datetime.now(timezone.utc)
            self.failure_count = 0
            self.success_count = 0
            
            logger.info(f"Circuit breaker CLOSED for {self.service_name}")
            
            # Adapt thresholds based on learning
            if self.config.adaptive_thresholds:
                await self._adapt_thresholds()
                
        except Exception as e:
            logger.warning(f"Error transitioning to closed: {e}")
    
    async def _should_attempt_reset(self) -> bool:
        """Check if circuit should attempt reset"""
        try:
            if not self.last_failure_time:
                return True
            
            time_since_failure = (datetime.now(timezone.utc) - self.last_failure_time).total_seconds() * 1000
            return time_since_failure >= self.config.recovery_timeout_ms
            
        except Exception:
            return False
    
    async def _predict_failure_risk(self, context: Dict[str, Any]) -> float:
        """Predict failure risk based on context and patterns"""
        try:
            return await self.predictor.predict_failure_risk(
                self.service_name,
                context,
                self.failure_patterns,
                self.failure_events
            )
        except Exception as e:
            logger.warning(f"Error predicting failure risk: {e}")
            return 0.0
    
    async def _learn_from_success(self, response_time_ms: float, context: Dict[str, Any]):
        """Learn from successful executions"""
        try:
            # Update performance baselines
            if len(self.response_times) >= 10:
                avg_response_time = statistics.mean(self.response_times)
                if avg_response_time < self.config.performance_threshold_ms * 0.8:
                    # Performance is good, can be more aggressive with timeouts
                    self.adaptive_timeout_ms = max(
                        self.config.timeout_ms * 0.8,
                        avg_response_time * 2
                    )
                    
        except Exception as e:
            logger.warning(f"Error learning from success: {e}")
    
    async def _learn_from_failure(self, failure_event: FailureEvent):
        """Learn from failure events"""
        try:
            # Detect patterns in failures
            patterns = await self.pattern_detector.detect_patterns(
                self.failure_events,
                failure_event
            )
            
            # Update failure patterns
            for pattern in patterns:
                self.failure_patterns[pattern.pattern_id] = pattern
                
        except Exception as e:
            logger.warning(f"Error learning from failure: {e}")
    
    async def _analyze_failure_patterns(self):
        """Analyze failure patterns when circuit opens"""
        try:
            recent_failures = [
                event for event in self.failure_events
                if (datetime.now(timezone.utc) - event.timestamp).total_seconds() < 300  # Last 5 minutes
            ]
            
            if len(recent_failures) >= 3:
                # Look for common patterns
                failure_types = [event.failure_type for event in recent_failures]
                most_common_type = max(set(failure_types), key=failure_types.count)
                
                logger.info(f"Most common failure type for {self.service_name}: {most_common_type.value}")
                
                # Adjust adaptive thresholds based on failure patterns
                if most_common_type == FailureType.TIMEOUT:
                    self.adaptive_timeout_ms = min(
                        self.adaptive_timeout_ms * 1.5,
                        self.config.timeout_ms * 2
                    )
                elif most_common_type == FailureType.PERFORMANCE_DEGRADATION:
                    self.adaptive_failure_threshold = max(
                        self.adaptive_failure_threshold - 1,
                        2
                    )
                    
        except Exception as e:
            logger.warning(f"Error analyzing failure patterns: {e}")
    
    async def _adapt_thresholds(self):
        """Adapt circuit breaker thresholds based on learning"""
        try:
            if len(self.failure_events) < 10:
                return
            
            # Calculate success rate
            recent_events = list(self.failure_events)[-50:]  # Last 50 events
            success_rate = self.successful_requests / max(self.total_requests, 1)
            
            # Adapt failure threshold based on success rate
            if success_rate > 0.95:
                # High success rate, can be more tolerant
                self.adaptive_failure_threshold = min(
                    self.adaptive_failure_threshold + 1,
                    self.config.failure_threshold * 2
                )
            elif success_rate < 0.8:
                # Low success rate, be more aggressive
                self.adaptive_failure_threshold = max(
                    self.adaptive_failure_threshold - 1,
                    2
                )
                
            logger.debug(f"Adapted thresholds for {self.service_name}: "
                        f"failure_threshold={self.adaptive_failure_threshold}, "
                        f"timeout_ms={self.adaptive_timeout_ms}")
                        
        except Exception as e:
            logger.warning(f"Error adapting thresholds: {e}")
    
    def get_stats(self) -> Dict[str, Any]:
        """Get circuit breaker statistics"""
        try:
            success_rate = self.successful_requests / max(self.total_requests, 1)
            avg_response_time = statistics.mean(self.response_times) if self.response_times else 0.0
            
            return {
                'service_name': self.service_name,
                'state': self.state.value,
                'performance': {
                    'total_requests': self.total_requests,
                    'successful_requests': self.successful_requests,
                    'failed_requests': self.failed_requests,
                    'blocked_requests': self.blocked_requests,
                    'success_rate': success_rate,
                    'average_response_time_ms': avg_response_time
                },
                'thresholds': {
                    'failure_threshold': self.adaptive_failure_threshold,
                    'timeout_ms': self.adaptive_timeout_ms,
                    'current_failure_count': self.failure_count,
                    'current_success_count': self.success_count
                },
                'patterns': {
                    'discovered_patterns': len(self.failure_patterns),
                    'recent_failures': len([
                        e for e in self.failure_events
                        if (datetime.now(timezone.utc) - e.timestamp).total_seconds() < 3600
                    ])
                },
                'state_info': {
                    'state_changed_at': self.state_changed_at.isoformat(),
                    'last_failure_time': self.last_failure_time.isoformat() if self.last_failure_time else None
                }
            }
            
        except Exception as e:
            logger.warning(f"Error getting circuit breaker stats: {e}")
            return {'error': str(e)}


class CircuitBreakerOpenError(Exception):
    """Exception raised when circuit breaker is open"""
    pass


class FailurePatternDetector:
    """Detects patterns in failure events"""

    def __init__(self):
        self.pattern_cache: Dict[str, FailurePattern] = {}

    async def detect_patterns(
        self,
        failure_events: deque,
        new_failure: FailureEvent
    ) -> List[FailurePattern]:
        """Detect patterns in failure events"""
        try:
            patterns = []

            # Convert deque to list for analysis
            events = list(failure_events)
            if len(events) < 3:
                return patterns

            # Temporal pattern detection
            temporal_pattern = await self._detect_temporal_pattern(events)
            if temporal_pattern:
                patterns.append(temporal_pattern)

            # Context pattern detection
            context_pattern = await self._detect_context_pattern(events, new_failure)
            if context_pattern:
                patterns.append(context_pattern)

            # Failure type pattern detection
            type_pattern = await self._detect_failure_type_pattern(events)
            if type_pattern:
                patterns.append(type_pattern)

            return patterns

        except Exception as e:
            logger.warning(f"Error detecting patterns: {e}")
            return []

    async def _detect_temporal_pattern(self, events: List[FailureEvent]) -> Optional[FailurePattern]:
        """Detect temporal patterns in failures"""
        try:
            if len(events) < 5:
                return None

            # Analyze time intervals between failures
            intervals = []
            for i in range(1, len(events)):
                interval = (events[i].timestamp - events[i-1].timestamp).total_seconds()
                intervals.append(interval)

            # Check for regular intervals (indicating periodic failures)
            if len(intervals) >= 3:
                avg_interval = statistics.mean(intervals)
                std_interval = statistics.stdev(intervals) if len(intervals) > 1 else 0

                # If intervals are relatively consistent, it's a pattern
                if std_interval < avg_interval * 0.3:  # Low variance
                    pattern_id = f"temporal_{int(avg_interval)}"

                    return FailurePattern(
                        pattern_id=pattern_id,
                        pattern_type="temporal",
                        frequency=len(intervals),
                        confidence_score=0.8,
                        context_conditions=[],
                        temporal_conditions=[f"interval_avg_{avg_interval:.1f}s"],
                        predictive_indicators=[f"next_failure_in_{avg_interval:.1f}s"],
                        mitigation_strategies=["increase_timeout", "add_retry_delay"],
                        discovered_at=datetime.now(timezone.utc),
                        last_seen=events[-1].timestamp
                    )

            return None

        except Exception as e:
            logger.warning(f"Error detecting temporal pattern: {e}")
            return None

    async def _detect_context_pattern(
        self,
        events: List[FailureEvent],
        new_failure: FailureEvent
    ) -> Optional[FailurePattern]:
        """Detect context-based patterns"""
        try:
            # Look for common context elements
            context_keys = set()
            for event in events[-10:]:  # Last 10 events
                if event.context:
                    context_keys.update(event.context.keys())

            if not context_keys:
                return None

            # Find context values that appear frequently in failures
            common_contexts = {}
            for key in context_keys:
                values = []
                for event in events[-10:]:
                    if event.context and key in event.context:
                        values.append(str(event.context[key]))

                if values:
                    most_common = max(set(values), key=values.count)
                    frequency = values.count(most_common)

                    if frequency >= 3:  # Appears in at least 3 failures
                        common_contexts[key] = most_common

            if common_contexts:
                pattern_id = f"context_{hashlib.md5(str(common_contexts).encode()).hexdigest()[:8]}"

                return FailurePattern(
                    pattern_id=pattern_id,
                    pattern_type="context",
                    frequency=len(events),
                    confidence_score=0.7,
                    context_conditions=[f"{k}={v}" for k, v in common_contexts.items()],
                    temporal_conditions=[],
                    predictive_indicators=list(common_contexts.keys()),
                    mitigation_strategies=["context_specific_handling", "parameter_validation"],
                    discovered_at=datetime.now(timezone.utc),
                    last_seen=new_failure.timestamp
                )

            return None

        except Exception as e:
            logger.warning(f"Error detecting context pattern: {e}")
            return None

    async def _detect_failure_type_pattern(self, events: List[FailureEvent]) -> Optional[FailurePattern]:
        """Detect failure type patterns"""
        try:
            if len(events) < 5:
                return None

            # Analyze failure type distribution
            failure_types = [event.failure_type for event in events[-20:]]  # Last 20 events
            type_counts = {}
            for failure_type in failure_types:
                type_counts[failure_type] = type_counts.get(failure_type, 0) + 1

            # Find dominant failure type
            if type_counts:
                dominant_type = max(type_counts.keys(), key=lambda k: type_counts[k])
                frequency = type_counts[dominant_type]

                if frequency >= len(failure_types) * 0.6:  # 60% or more of same type
                    pattern_id = f"type_{dominant_type.value}"

                    mitigation_map = {
                        FailureType.TIMEOUT: ["increase_timeout", "optimize_performance"],
                        FailureType.ERROR: ["error_handling", "input_validation"],
                        FailureType.PERFORMANCE_DEGRADATION: ["performance_optimization", "resource_scaling"],
                        FailureType.RESOURCE_EXHAUSTION: ["resource_scaling", "load_balancing"],
                        FailureType.DEPENDENCY_FAILURE: ["dependency_circuit_breaker", "fallback_strategy"]
                    }

                    return FailurePattern(
                        pattern_id=pattern_id,
                        pattern_type="failure_type",
                        frequency=frequency,
                        confidence_score=0.9,
                        context_conditions=[],
                        temporal_conditions=[],
                        predictive_indicators=[f"failure_type_{dominant_type.value}"],
                        mitigation_strategies=mitigation_map.get(dominant_type, ["general_resilience"]),
                        discovered_at=datetime.now(timezone.utc),
                        last_seen=events[-1].timestamp
                    )

            return None

        except Exception as e:
            logger.warning(f"Error detecting failure type pattern: {e}")
            return None


class FailurePredictor:
    """Predicts failure risk based on patterns and context"""

    def __init__(self):
        self.prediction_cache: Dict[str, Tuple[float, datetime]] = {}
        self.cache_ttl = 60  # Cache predictions for 60 seconds

    async def predict_failure_risk(
        self,
        service_name: str,
        context: Dict[str, Any],
        patterns: Dict[str, FailurePattern],
        recent_events: deque
    ) -> float:
        """Predict failure risk based on current context and patterns"""
        try:
            # Create cache key
            context_key = hashlib.md5(
                f"{service_name}_{json.dumps(context, sort_keys=True)}".encode()
            ).hexdigest()

            # Check cache
            if context_key in self.prediction_cache:
                risk, timestamp = self.prediction_cache[context_key]
                if (datetime.now(timezone.utc) - timestamp).total_seconds() < self.cache_ttl:
                    return risk

            risk_score = 0.0

            # Analyze temporal risk
            temporal_risk = await self._calculate_temporal_risk(recent_events)
            risk_score += temporal_risk * 0.4

            # Analyze context risk
            context_risk = await self._calculate_context_risk(context, patterns)
            risk_score += context_risk * 0.4

            # Analyze pattern risk
            pattern_risk = await self._calculate_pattern_risk(patterns, recent_events)
            risk_score += pattern_risk * 0.2

            # Normalize risk score
            final_risk = min(risk_score, 1.0)

            # Cache the prediction
            self.prediction_cache[context_key] = (final_risk, datetime.now(timezone.utc))

            return final_risk

        except Exception as e:
            logger.warning(f"Error predicting failure risk: {e}")
            return 0.0

    async def _calculate_temporal_risk(self, recent_events: deque) -> float:
        """Calculate risk based on temporal patterns"""
        try:
            events = list(recent_events)
            if len(events) < 2:
                return 0.0

            # Check recent failure frequency
            now = datetime.now(timezone.utc)
            recent_failures = [
                event for event in events
                if (now - event.timestamp).total_seconds() < 300  # Last 5 minutes
            ]

            if len(recent_failures) >= 3:
                return 0.8  # High risk if 3+ failures in 5 minutes
            elif len(recent_failures) >= 2:
                return 0.5  # Medium risk if 2 failures in 5 minutes

            return 0.1  # Low baseline risk

        except Exception:
            return 0.0

    async def _calculate_context_risk(
        self,
        context: Dict[str, Any],
        patterns: Dict[str, FailurePattern]
    ) -> float:
        """Calculate risk based on context matching patterns"""
        try:
            if not context or not patterns:
                return 0.0

            max_risk = 0.0

            for pattern in patterns.values():
                if pattern.pattern_type == "context":
                    # Check if current context matches pattern conditions
                    matches = 0
                    total_conditions = len(pattern.context_conditions)

                    for condition in pattern.context_conditions:
                        if "=" in condition:
                            key, value = condition.split("=", 1)
                            if key in context and str(context[key]) == value:
                                matches += 1

                    if total_conditions > 0:
                        match_ratio = matches / total_conditions
                        pattern_risk = match_ratio * pattern.confidence_score
                        max_risk = max(max_risk, pattern_risk)

            return max_risk

        except Exception:
            return 0.0

    async def _calculate_pattern_risk(
        self,
        patterns: Dict[str, FailurePattern],
        recent_events: deque
    ) -> float:
        """Calculate risk based on pattern analysis"""
        try:
            if not patterns:
                return 0.0

            # Check if we're in a pattern-predicted failure window
            now = datetime.now(timezone.utc)
            max_risk = 0.0

            for pattern in patterns.values():
                if pattern.pattern_type == "temporal":
                    # Check if we're due for a failure based on temporal pattern
                    time_since_last = (now - pattern.last_seen).total_seconds()

                    # Extract expected interval from temporal conditions
                    for condition in pattern.temporal_conditions:
                        if "interval_avg_" in condition:
                            try:
                                interval_str = condition.replace("interval_avg_", "").replace("s", "")
                                expected_interval = float(interval_str)

                                # Calculate risk based on how close we are to expected failure time
                                if time_since_last >= expected_interval * 0.8:
                                    proximity_risk = min(time_since_last / expected_interval, 1.0)
                                    pattern_risk = proximity_risk * pattern.confidence_score
                                    max_risk = max(max_risk, pattern_risk)

                            except ValueError:
                                continue

            return max_risk

        except Exception:
            return 0.0
