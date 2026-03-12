"""
Performance SLA Service for Clinical Workflow Engine.
Implements detailed latency budgets, circuit breaker patterns, and performance optimization.
"""
import logging
import asyncio
import time
from typing import Dict, Any, List, Optional, Callable
from datetime import datetime, timedelta
from dataclasses import dataclass, field
from enum import Enum
import json

logger = logging.getLogger(__name__)


class CircuitState(Enum):
    """Circuit breaker states."""
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"


class PerformanceTier(Enum):
    """Performance tiers for different operations."""
    CRITICAL = "critical"      # < 50ms
    HIGH = "high"             # < 100ms
    STANDARD = "standard"     # < 250ms
    BACKGROUND = "background" # < 1000ms


@dataclass
class LatencyBudget:
    """Latency budget configuration for workflow phases."""
    phase_name: str
    budget_ms: int
    tier: PerformanceTier
    timeout_ms: int
    retry_count: int = 3
    circuit_breaker_enabled: bool = True
    failure_threshold: int = 5
    recovery_timeout_ms: int = 30000


@dataclass
class PerformanceMetric:
    """Performance metric data."""
    operation: str
    phase: str
    execution_time_ms: float
    budget_ms: int
    success: bool
    timestamp: datetime
    violation: bool = False
    circuit_state: Optional[str] = None
    retry_count: int = 0


@dataclass
class CircuitBreakerState:
    """Circuit breaker state tracking."""
    operation: str
    state: CircuitState = CircuitState.CLOSED
    failure_count: int = 0
    last_failure_time: Optional[datetime] = None
    last_success_time: Optional[datetime] = None
    half_open_attempts: int = 0
    total_requests: int = 0
    successful_requests: int = 0


class PerformanceSLAService:
    """
    Performance SLA Service implementing detailed latency budgets and circuit breakers.
    
    Features:
    - 250ms total workflow budget allocation
    - Phase-specific latency budgets
    - Circuit breaker patterns for resilience
    - Performance violation detection and alerting
    - Automatic performance optimization
    - Real-time SLA monitoring
    """
    
    def __init__(self):
        self.latency_budgets = self._initialize_latency_budgets()
        self.circuit_breakers: Dict[str, CircuitBreakerState] = {}
        self.performance_metrics: List[PerformanceMetric] = []
        self.sla_violations: List[Dict[str, Any]] = []
        self.performance_stats = {
            "total_operations": 0,
            "successful_operations": 0,
            "sla_violations": 0,
            "circuit_breaker_trips": 0,
            "average_latency_ms": 0.0
        }
        
        # Start background monitoring
        asyncio.create_task(self._start_performance_monitoring())
        
        logger.info("✅ Performance SLA Service initialized with 250ms budget allocation")
    
    def _initialize_latency_budgets(self) -> Dict[str, LatencyBudget]:
        """Initialize latency budgets as specified in the implementation plan."""
        return {
            # PRODUCTION SLA BUDGETS - Sub-second performance requirement (250ms total)
            "workflow_initialization": LatencyBudget(
                phase_name="workflow_initialization",
                budget_ms=10,
                tier=PerformanceTier.CRITICAL,
                timeout_ms=20,
                retry_count=2,
                failure_threshold=3
            ),
            "context_fetching": LatencyBudget(
                phase_name="context_fetching",
                budget_ms=40,
                tier=PerformanceTier.HIGH,
                timeout_ms=80,
                retry_count=2,
                failure_threshold=5
            ),
            "proposal_generation": LatencyBudget(
                phase_name="proposal_generation",
                budget_ms=50,
                tier=PerformanceTier.HIGH,
                timeout_ms=100,
                retry_count=3,
                failure_threshold=5
            ),
            "safety_validation": LatencyBudget(
                phase_name="safety_validation",
                budget_ms=100,
                tier=PerformanceTier.STANDARD,
                timeout_ms=200,
                retry_count=2,
                failure_threshold=3
            ),
            "commit_operation": LatencyBudget(
                phase_name="commit_operation",
                budget_ms=30,
                tier=PerformanceTier.HIGH,
                timeout_ms=60,
                retry_count=3,
                failure_threshold=5
            ),
            "post_processing": LatencyBudget(
                phase_name="post_processing",
                budget_ms=20,
                tier=PerformanceTier.STANDARD,
                timeout_ms=40,
                retry_count=1,
                failure_threshold=10
            ),
            
            # Execution pattern specific budgets
            "pessimistic_workflow": LatencyBudget(
                phase_name="pessimistic_workflow",
                budget_ms=250,
                tier=PerformanceTier.STANDARD,
                timeout_ms=500,
                retry_count=2,
                failure_threshold=5
            ),
            "optimistic_workflow": LatencyBudget(
                phase_name="optimistic_workflow",
                budget_ms=150,
                tier=PerformanceTier.HIGH,
                timeout_ms=300,
                retry_count=2,
                failure_threshold=5
            ),
            "digital_reflex_arc": LatencyBudget(
                phase_name="digital_reflex_arc",
                budget_ms=100,
                tier=PerformanceTier.CRITICAL,
                timeout_ms=200,
                retry_count=1,
                failure_threshold=3
            ),
            
            # Emergency and safety operations
            "emergency_response": LatencyBudget(
                phase_name="emergency_response",
                budget_ms=60,
                tier=PerformanceTier.CRITICAL,
                timeout_ms=120,
                retry_count=1,
                failure_threshold=2
            ),
            "break_glass_access": LatencyBudget(
                phase_name="break_glass_access",
                budget_ms=30,
                tier=PerformanceTier.CRITICAL,
                timeout_ms=60,
                retry_count=1,
                failure_threshold=2
            )
        }
    
    async def track_phase_performance(
        self,
        phase_name: str,
        operation: Callable,
        context: Optional[Dict[str, Any]] = None
    ) -> Any:
        """
        Track performance against SLA budget with circuit breaker protection.
        """
        budget = self.latency_budgets.get(phase_name)
        if not budget:
            logger.warning(f"No budget defined for phase: {phase_name}")
            return await operation()
        
        # Check circuit breaker
        circuit_key = f"{phase_name}_circuit"
        if budget.circuit_breaker_enabled:
            circuit_state = await self._check_circuit_breaker(circuit_key, budget)
            if circuit_state == CircuitState.OPEN:
                raise PerformanceSLAError(f"Circuit breaker OPEN for {phase_name}")
        
        start_time = time.time()
        success = False
        result = None
        retry_count = 0
        
        try:
            # Execute with timeout and retry
            for attempt in range(budget.retry_count + 1):
                try:
                    retry_count = attempt
                    result = await asyncio.wait_for(
                        operation(),
                        timeout=budget.timeout_ms / 1000
                    )
                    success = True
                    break
                    
                except asyncio.TimeoutError:
                    if attempt == budget.retry_count:
                        raise PerformanceSLAError(f"Timeout in {phase_name} after {budget.timeout_ms}ms")
                    await asyncio.sleep(0.01 * (2 ** attempt))  # Exponential backoff
                    
                except Exception as e:
                    if attempt == budget.retry_count:
                        raise
                    await asyncio.sleep(0.01 * (2 ** attempt))
            
            elapsed_ms = (time.time() - start_time) * 1000
            
            # Record performance metric
            metric = PerformanceMetric(
                operation=phase_name,
                phase=phase_name,
                execution_time_ms=elapsed_ms,
                budget_ms=budget.budget_ms,
                success=success,
                timestamp=datetime.utcnow(),
                violation=elapsed_ms > budget.budget_ms,
                circuit_state=self.circuit_breakers.get(circuit_key, CircuitBreakerState(phase_name)).state.value,
                retry_count=retry_count
            )
            
            await self._record_performance_metric(metric)
            
            # Update circuit breaker on success
            if budget.circuit_breaker_enabled:
                await self._record_circuit_success(circuit_key)
            
            # Check for SLA violation
            if elapsed_ms > budget.budget_ms:
                await self._handle_sla_violation(phase_name, elapsed_ms, budget, context)
            
            return result
            
        except Exception as e:
            elapsed_ms = (time.time() - start_time) * 1000
            
            # Record failure metric
            metric = PerformanceMetric(
                operation=phase_name,
                phase=phase_name,
                execution_time_ms=elapsed_ms,
                budget_ms=budget.budget_ms,
                success=False,
                timestamp=datetime.utcnow(),
                violation=True,
                circuit_state=self.circuit_breakers.get(circuit_key, CircuitBreakerState(phase_name)).state.value,
                retry_count=retry_count
            )
            
            await self._record_performance_metric(metric)
            
            # Update circuit breaker on failure
            if budget.circuit_breaker_enabled:
                await self._record_circuit_failure(circuit_key, budget)
            
            logger.error(f"❌ {phase_name} failed after {elapsed_ms:.1f}ms: {e}")
            raise
    
    async def _check_circuit_breaker(
        self,
        circuit_key: str,
        budget: LatencyBudget
    ) -> CircuitState:
        """Check circuit breaker state and handle transitions."""
        if circuit_key not in self.circuit_breakers:
            self.circuit_breakers[circuit_key] = CircuitBreakerState(operation=circuit_key)
        
        circuit = self.circuit_breakers[circuit_key]
        current_time = datetime.utcnow()
        
        if circuit.state == CircuitState.OPEN:
            # Check if recovery timeout has passed
            if (circuit.last_failure_time and 
                (current_time - circuit.last_failure_time).total_seconds() * 1000 > budget.recovery_timeout_ms):
                circuit.state = CircuitState.HALF_OPEN
                circuit.half_open_attempts = 0
                logger.info(f"🔄 Circuit breaker transitioning to HALF_OPEN: {circuit_key}")
        
        elif circuit.state == CircuitState.HALF_OPEN:
            # Limit half-open attempts
            if circuit.half_open_attempts >= 3:
                circuit.state = CircuitState.OPEN
                circuit.last_failure_time = current_time
                logger.warning(f"🚨 Circuit breaker back to OPEN: {circuit_key}")
        
        return circuit.state
    
    async def _record_circuit_success(self, circuit_key: str):
        """Record successful circuit breaker operation."""
        if circuit_key in self.circuit_breakers:
            circuit = self.circuit_breakers[circuit_key]
            circuit.successful_requests += 1
            circuit.last_success_time = datetime.utcnow()
            
            if circuit.state == CircuitState.HALF_OPEN:
                circuit.state = CircuitState.CLOSED
                circuit.failure_count = 0
                logger.info(f"✅ Circuit breaker CLOSED: {circuit_key}")
    
    async def _record_circuit_failure(self, circuit_key: str, budget: LatencyBudget):
        """Record failed circuit breaker operation."""
        if circuit_key not in self.circuit_breakers:
            self.circuit_breakers[circuit_key] = CircuitBreakerState(operation=circuit_key)
        
        circuit = self.circuit_breakers[circuit_key]
        circuit.failure_count += 1
        circuit.last_failure_time = datetime.utcnow()
        
        if circuit.state == CircuitState.HALF_OPEN:
            circuit.half_open_attempts += 1
        
        # Trip circuit breaker if failure threshold exceeded
        if circuit.failure_count >= budget.failure_threshold:
            circuit.state = CircuitState.OPEN
            self.performance_stats["circuit_breaker_trips"] += 1
            logger.error(f"🚨 Circuit breaker OPENED: {circuit_key} (failures: {circuit.failure_count})")
    
    async def _record_performance_metric(self, metric: PerformanceMetric):
        """Record performance metric and update statistics."""
        self.performance_metrics.append(metric)
        
        # Update statistics
        self.performance_stats["total_operations"] += 1
        if metric.success:
            self.performance_stats["successful_operations"] += 1
        if metric.violation:
            self.performance_stats["sla_violations"] += 1
        
        # Calculate rolling average latency
        recent_metrics = self.performance_metrics[-100:]  # Last 100 operations
        if recent_metrics:
            avg_latency = sum(m.execution_time_ms for m in recent_metrics) / len(recent_metrics)
            self.performance_stats["average_latency_ms"] = avg_latency
        
        # Keep only recent metrics in memory
        if len(self.performance_metrics) > 1000:
            self.performance_metrics = self.performance_metrics[-500:]
    
    async def _handle_sla_violation(
        self,
        phase_name: str,
        elapsed_ms: float,
        budget: LatencyBudget,
        context: Optional[Dict[str, Any]]
    ):
        """Handle SLA violation with alerting and logging."""
        violation_percentage = ((elapsed_ms - budget.budget_ms) / budget.budget_ms) * 100
        
        violation = {
            "violation_id": f"sla_violation_{int(time.time() * 1000000)}",
            "phase_name": phase_name,
            "elapsed_ms": elapsed_ms,
            "budget_ms": budget.budget_ms,
            "violation_percentage": violation_percentage,
            "tier": budget.tier.value,
            "timestamp": datetime.utcnow().isoformat(),
            "context": context or {}
        }
        
        self.sla_violations.append(violation)
        
        # Log violation with appropriate severity
        if budget.tier == PerformanceTier.CRITICAL:
            logger.error(f"🚨 CRITICAL SLA VIOLATION: {phase_name} took {elapsed_ms:.1f}ms > {budget.budget_ms}ms ({violation_percentage:.1f}% over)")
        else:
            logger.warning(f"⚠️ SLA VIOLATION: {phase_name} took {elapsed_ms:.1f}ms > {budget.budget_ms}ms ({violation_percentage:.1f}% over)")
        
        # In production: Send to monitoring/alerting system
        await self._send_sla_violation_alert(violation)
    
    async def _send_sla_violation_alert(self, violation: Dict[str, Any]):
        """Send SLA violation alert to monitoring system."""
        try:
            # In production: Send to monitoring system (Prometheus, DataDog, etc.)
            logger.info(f"📊 SLA violation alert sent: {violation['violation_id']}")
        except Exception as e:
            logger.error(f"Failed to send SLA violation alert: {e}")

    async def _start_performance_monitoring(self):
        """Start background performance monitoring and optimization."""
        while True:
            try:
                await asyncio.sleep(60)  # Monitor every minute

                # Analyze performance trends
                await self._analyze_performance_trends()

                # Optimize circuit breaker thresholds
                await self._optimize_circuit_breakers()

                # Clean up old metrics
                await self._cleanup_old_metrics()

            except Exception as e:
                logger.error(f"Performance monitoring error: {e}")
                await asyncio.sleep(60)

    async def _analyze_performance_trends(self):
        """Analyze performance trends and identify optimization opportunities."""
        if len(self.performance_metrics) < 10:
            return

        # Analyze recent performance by phase
        recent_metrics = self.performance_metrics[-100:]
        phase_performance = {}

        for metric in recent_metrics:
            if metric.phase not in phase_performance:
                phase_performance[metric.phase] = {
                    'total_time': 0,
                    'count': 0,
                    'violations': 0,
                    'successes': 0
                }

            phase_performance[metric.phase]['total_time'] += metric.execution_time_ms
            phase_performance[metric.phase]['count'] += 1
            if metric.violation:
                phase_performance[metric.phase]['violations'] += 1
            if metric.success:
                phase_performance[metric.phase]['successes'] += 1

        # Log performance insights
        for phase, stats in phase_performance.items():
            if stats['count'] > 0:
                avg_time = stats['total_time'] / stats['count']
                violation_rate = stats['violations'] / stats['count']
                success_rate = stats['successes'] / stats['count']

                if violation_rate > 0.1:  # More than 10% violations
                    logger.warning(f"📈 Performance concern: {phase} has {violation_rate:.1%} SLA violations (avg: {avg_time:.1f}ms)")
                elif success_rate > 0.95 and avg_time < self.latency_budgets.get(phase, LatencyBudget(phase, 100, PerformanceTier.STANDARD, 200)).budget_ms * 0.8:
                    logger.info(f"✅ Performance excellent: {phase} running at {avg_time:.1f}ms with {success_rate:.1%} success rate")

    async def _optimize_circuit_breakers(self):
        """Optimize circuit breaker thresholds based on performance data."""
        for circuit_key, circuit in self.circuit_breakers.items():
            if circuit.total_requests > 50:  # Enough data for optimization
                success_rate = circuit.successful_requests / circuit.total_requests

                # Adjust failure threshold based on success rate
                if success_rate > 0.95:
                    # High success rate - can be more tolerant
                    budget = self.latency_budgets.get(circuit.operation.replace('_circuit', ''))
                    if budget and budget.failure_threshold < 10:
                        budget.failure_threshold += 1
                        logger.info(f"🔧 Increased failure threshold for {circuit.operation}: {budget.failure_threshold}")

                elif success_rate < 0.8:
                    # Low success rate - be more aggressive
                    budget = self.latency_budgets.get(circuit.operation.replace('_circuit', ''))
                    if budget and budget.failure_threshold > 2:
                        budget.failure_threshold -= 1
                        logger.info(f"🔧 Decreased failure threshold for {circuit.operation}: {budget.failure_threshold}")

    async def _cleanup_old_metrics(self):
        """Clean up old performance metrics to prevent memory growth."""
        cutoff_time = datetime.utcnow() - timedelta(hours=1)

        # Keep only recent metrics
        self.performance_metrics = [
            m for m in self.performance_metrics
            if m.timestamp > cutoff_time
        ]

        # Keep only recent SLA violations
        self.sla_violations = [
            v for v in self.sla_violations
            if datetime.fromisoformat(v['timestamp'].replace('Z', '+00:00')) > cutoff_time
        ]

    async def get_performance_dashboard(self) -> Dict[str, Any]:
        """Get comprehensive performance dashboard data."""
        current_time = datetime.utcnow()

        # Calculate phase-specific metrics
        phase_metrics = {}
        for phase_name, budget in self.latency_budgets.items():
            recent_metrics = [
                m for m in self.performance_metrics
                if m.phase == phase_name and (current_time - m.timestamp).total_seconds() < 3600
            ]

            if recent_metrics:
                avg_latency = sum(m.execution_time_ms for m in recent_metrics) / len(recent_metrics)
                violation_rate = sum(1 for m in recent_metrics if m.violation) / len(recent_metrics)
                success_rate = sum(1 for m in recent_metrics if m.success) / len(recent_metrics)

                phase_metrics[phase_name] = {
                    'budget_ms': budget.budget_ms,
                    'avg_latency_ms': round(avg_latency, 2),
                    'violation_rate': round(violation_rate, 3),
                    'success_rate': round(success_rate, 3),
                    'tier': budget.tier.value,
                    'total_requests': len(recent_metrics),
                    'budget_utilization': round((avg_latency / budget.budget_ms) * 100, 1)
                }

        # Circuit breaker status
        circuit_status = {}
        for circuit_key, circuit in self.circuit_breakers.items():
            circuit_status[circuit_key] = {
                'state': circuit.state.value,
                'failure_count': circuit.failure_count,
                'success_rate': round(circuit.successful_requests / max(circuit.total_requests, 1), 3),
                'last_failure': circuit.last_failure_time.isoformat() if circuit.last_failure_time else None
            }

        # Recent SLA violations
        recent_violations = [
            v for v in self.sla_violations
            if (current_time - datetime.fromisoformat(v['timestamp'].replace('Z', '+00:00'))).total_seconds() < 3600
        ]

        return {
            'timestamp': current_time.isoformat(),
            'overall_stats': self.performance_stats,
            'phase_metrics': phase_metrics,
            'circuit_breakers': circuit_status,
            'recent_violations': {
                'count': len(recent_violations),
                'critical_count': len([v for v in recent_violations if v.get('tier') == 'critical']),
                'violations': recent_violations[-10:]  # Last 10 violations
            },
            'budget_allocation': {
                'total_budget_ms': 250,
                'allocated_budget_ms': sum(b.budget_ms for b in self.latency_budgets.values() if 'workflow' not in b.phase_name),
                'remaining_budget_ms': 250 - sum(b.budget_ms for b in self.latency_budgets.values() if 'workflow' not in b.phase_name)
            }
        }

    async def get_sla_compliance_report(self) -> Dict[str, Any]:
        """Generate SLA compliance report."""
        current_time = datetime.utcnow()

        # Calculate compliance by tier
        tier_compliance = {}
        for tier in PerformanceTier:
            tier_budgets = [b for b in self.latency_budgets.values() if b.tier == tier]
            tier_metrics = []

            for budget in tier_budgets:
                phase_metrics = [
                    m for m in self.performance_metrics
                    if m.phase == budget.phase_name and (current_time - m.timestamp).total_seconds() < 3600
                ]
                tier_metrics.extend(phase_metrics)

            if tier_metrics:
                compliance_rate = sum(1 for m in tier_metrics if not m.violation) / len(tier_metrics)
                avg_latency = sum(m.execution_time_ms for m in tier_metrics) / len(tier_metrics)

                tier_compliance[tier.value] = {
                    'compliance_rate': round(compliance_rate, 3),
                    'avg_latency_ms': round(avg_latency, 2),
                    'total_operations': len(tier_metrics),
                    'violations': len([m for m in tier_metrics if m.violation])
                }

        return {
            'report_timestamp': current_time.isoformat(),
            'overall_compliance_rate': round(
                self.performance_stats['successful_operations'] / max(self.performance_stats['total_operations'], 1), 3
            ),
            'tier_compliance': tier_compliance,
            'total_budget_ms': 250,
            'sla_violations_last_hour': len([
                v for v in self.sla_violations
                if (current_time - datetime.fromisoformat(v['timestamp'].replace('Z', '+00:00'))).total_seconds() < 3600
            ])
        }


class PerformanceSLAError(Exception):
    """Exception raised for performance SLA violations."""
    pass


# Global instance
performance_sla_service = PerformanceSLAService()
