"""
Simple Performance SLA Framework Test
Tests the performance SLA implementation without external dependencies.
"""
import asyncio
import time
import logging
from datetime import datetime
from typing import Dict, Any, Callable
from dataclasses import dataclass
from enum import Enum

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


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


class SimplePerformanceSLAService:
    """
    Simplified Performance SLA Service for testing.
    Implements core latency budget and performance tracking functionality.
    """
    
    def __init__(self):
        self.latency_budgets = self._initialize_latency_budgets()
        self.performance_metrics = []
        self.sla_violations = []
        self.performance_stats = {
            "total_operations": 0,
            "successful_operations": 0,
            "sla_violations": 0,
            "average_latency_ms": 0.0
        }
        
        logger.info("✅ Simple Performance SLA Service initialized")
    
    def _initialize_latency_budgets(self) -> Dict[str, LatencyBudget]:
        """Initialize latency budgets as specified in the implementation plan."""
        return {
            # PRODUCTION SLA BUDGETS - Sub-second performance requirement (250ms total)
            "workflow_initialization": LatencyBudget(
                phase_name="workflow_initialization",
                budget_ms=10,
                tier=PerformanceTier.CRITICAL,
                timeout_ms=20,
                retry_count=2
            ),
            "context_fetching": LatencyBudget(
                phase_name="context_fetching",
                budget_ms=40,
                tier=PerformanceTier.HIGH,
                timeout_ms=80,
                retry_count=2
            ),
            "proposal_generation": LatencyBudget(
                phase_name="proposal_generation",
                budget_ms=50,
                tier=PerformanceTier.HIGH,
                timeout_ms=100,
                retry_count=3
            ),
            "safety_validation": LatencyBudget(
                phase_name="safety_validation",
                budget_ms=100,
                tier=PerformanceTier.STANDARD,
                timeout_ms=200,
                retry_count=2
            ),
            "commit_operation": LatencyBudget(
                phase_name="commit_operation",
                budget_ms=30,
                tier=PerformanceTier.HIGH,
                timeout_ms=60,
                retry_count=3
            ),
            "post_processing": LatencyBudget(
                phase_name="post_processing",
                budget_ms=20,
                tier=PerformanceTier.STANDARD,
                timeout_ms=40,
                retry_count=1
            )
        }
    
    async def track_phase_performance(
        self,
        phase_name: str,
        operation: Callable,
        context: Dict[str, Any] = None
    ) -> Any:
        """Track performance against SLA budget."""
        budget = self.latency_budgets.get(phase_name)
        if not budget:
            logger.warning(f"No budget defined for phase: {phase_name}")
            return await operation()
        
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
                        raise Exception(f"Timeout in {phase_name} after {budget.timeout_ms}ms")
                    await asyncio.sleep(0.01 * (2 ** attempt))  # Exponential backoff
                    
                except Exception as e:
                    if attempt == budget.retry_count:
                        raise
                    await asyncio.sleep(0.01 * (2 ** attempt))
            
            elapsed_ms = (time.time() - start_time) * 1000
            
            # Record performance metric
            metric = {
                "operation": phase_name,
                "execution_time_ms": elapsed_ms,
                "budget_ms": budget.budget_ms,
                "success": success,
                "timestamp": datetime.utcnow(),
                "violation": elapsed_ms > budget.budget_ms,
                "retry_count": retry_count,
                "tier": budget.tier.value
            }
            
            self._record_performance_metric(metric)
            
            # Check for SLA violation
            if elapsed_ms > budget.budget_ms:
                self._handle_sla_violation(phase_name, elapsed_ms, budget, context)
            
            return result
            
        except Exception as e:
            elapsed_ms = (time.time() - start_time) * 1000
            
            # Record failure metric
            metric = {
                "operation": phase_name,
                "execution_time_ms": elapsed_ms,
                "budget_ms": budget.budget_ms,
                "success": False,
                "timestamp": datetime.utcnow(),
                "violation": True,
                "retry_count": retry_count,
                "tier": budget.tier.value
            }
            
            self._record_performance_metric(metric)
            
            logger.error(f"❌ {phase_name} failed after {elapsed_ms:.1f}ms: {e}")
            raise
    
    def _record_performance_metric(self, metric: Dict[str, Any]):
        """Record performance metric and update statistics."""
        self.performance_metrics.append(metric)
        
        # Update statistics
        self.performance_stats["total_operations"] += 1
        if metric["success"]:
            self.performance_stats["successful_operations"] += 1
        if metric["violation"]:
            self.performance_stats["sla_violations"] += 1
        
        # Calculate rolling average latency
        recent_metrics = self.performance_metrics[-100:]  # Last 100 operations
        if recent_metrics:
            avg_latency = sum(m["execution_time_ms"] for m in recent_metrics) / len(recent_metrics)
            self.performance_stats["average_latency_ms"] = avg_latency
    
    def _handle_sla_violation(
        self,
        phase_name: str,
        elapsed_ms: float,
        budget: LatencyBudget,
        context: Dict[str, Any]
    ):
        """Handle SLA violation with logging."""
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
    
    def get_performance_summary(self) -> Dict[str, Any]:
        """Get performance summary."""
        return {
            "total_budget_ms": 250,
            "allocated_budget_ms": sum(b.budget_ms for b in self.latency_budgets.values()),
            "performance_stats": self.performance_stats,
            "recent_violations": len(self.sla_violations),
            "phase_count": len(self.latency_budgets)
        }


class TestPerformanceSLAFramework:
    """Test Performance SLA Framework functionality."""
    
    def __init__(self):
        self.sla_service = SimplePerformanceSLAService()
    
    async def test_latency_budget_allocation(self):
        """Test that latency budgets are properly allocated."""
        print("🔍 Testing latency budget allocation...")
        
        # Verify total budget allocation (250ms total)
        total_allocated = sum(budget.budget_ms for budget in self.sla_service.latency_budgets.values())
        assert total_allocated == 250, f"Total budget should be 250ms, got {total_allocated}ms"
        
        # Verify individual phase budgets
        expected_budgets = {
            "workflow_initialization": 10,
            "context_fetching": 40,
            "proposal_generation": 50,
            "safety_validation": 100,
            "commit_operation": 30,
            "post_processing": 20
        }
        
        for phase, expected_budget in expected_budgets.items():
            actual_budget = self.sla_service.latency_budgets[phase].budget_ms
            assert actual_budget == expected_budget, f"{phase} budget should be {expected_budget}ms, got {actual_budget}ms"
        
        print(f"✅ Latency budget allocation verified: {total_allocated}ms total")
    
    async def test_successful_operation_tracking(self):
        """Test tracking of successful operations within budget."""
        print("🔍 Testing successful operation tracking...")
        
        # Test fast operation (within budget)
        async def fast_operation():
            await asyncio.sleep(0.005)  # 5ms - within 10ms budget
            return "success"
        
        result = await self.sla_service.track_phase_performance(
            "workflow_initialization",
            fast_operation,
            {"test": "fast_operation"}
        )
        
        assert result == "success"
        
        # Verify metrics were recorded
        assert len(self.sla_service.performance_metrics) > 0
        latest_metric = self.sla_service.performance_metrics[-1]
        
        assert latest_metric["operation"] == "workflow_initialization"
        assert latest_metric["success"] is True
        assert latest_metric["violation"] is False
        assert latest_metric["execution_time_ms"] < 10  # Should be under budget
        
        print(f"✅ Successful operation tracked: {latest_metric['execution_time_ms']:.1f}ms")
    
    async def test_sla_violation_detection(self):
        """Test SLA violation detection and handling."""
        print("🔍 Testing SLA violation detection...")
        
        # Test slow operation (exceeds budget)
        async def slow_operation():
            await asyncio.sleep(0.015)  # 15ms - exceeds 10ms budget
            return "slow_success"
        
        result = await self.sla_service.track_phase_performance(
            "workflow_initialization",
            slow_operation,
            {"test": "slow_operation"}
        )
        
        assert result == "slow_success"
        
        # Verify SLA violation was detected
        latest_metric = self.sla_service.performance_metrics[-1]
        assert latest_metric["violation"] is True
        assert latest_metric["execution_time_ms"] > 10  # Should exceed budget
        
        # Verify violation was recorded
        assert len(self.sla_service.sla_violations) > 0
        latest_violation = self.sla_service.sla_violations[-1]
        assert latest_violation["phase_name"] == "workflow_initialization"
        assert latest_violation["elapsed_ms"] > 10
        
        print(f"✅ SLA violation detected: {latest_metric['execution_time_ms']:.1f}ms > 10ms budget")
    
    async def test_timeout_handling(self):
        """Test timeout handling with retry logic."""
        print("🔍 Testing timeout handling...")
        
        # Test operation that times out
        async def timeout_operation():
            await asyncio.sleep(0.1)  # 100ms - will timeout at 20ms
            return "should_not_reach"
        
        try:
            await self.sla_service.track_phase_performance(
                "workflow_initialization",
                timeout_operation,
                {"test": "timeout_operation"}
            )
            assert False, "Should have raised exception"
        except Exception as e:
            assert "Timeout" in str(e)
            print(f"✅ Timeout handled correctly: {e}")
        
        # Verify failure was recorded
        latest_metric = self.sla_service.performance_metrics[-1]
        assert latest_metric["success"] is False
        assert latest_metric["violation"] is True
    
    async def test_performance_tier_classification(self):
        """Test performance tier classification."""
        print("🔍 Testing performance tier classification...")
        
        # Verify tier classifications
        tier_expectations = {
            "workflow_initialization": PerformanceTier.CRITICAL,
            "context_fetching": PerformanceTier.HIGH,
            "proposal_generation": PerformanceTier.HIGH,
            "safety_validation": PerformanceTier.STANDARD,
            "commit_operation": PerformanceTier.HIGH,
            "post_processing": PerformanceTier.STANDARD
        }
        
        for phase, expected_tier in tier_expectations.items():
            actual_tier = self.sla_service.latency_budgets[phase].tier
            assert actual_tier == expected_tier, f"{phase} should be {expected_tier.value}, got {actual_tier.value}"
        
        print("✅ Performance tier classification verified")
    
    async def test_retry_mechanism(self):
        """Test retry mechanism for failed operations."""
        print("🔍 Testing retry mechanism...")
        
        attempt_count = 0
        
        async def flaky_operation():
            nonlocal attempt_count
            attempt_count += 1
            if attempt_count < 3:  # Fail first 2 attempts
                raise Exception(f"Attempt {attempt_count} failed")
            return f"success_on_attempt_{attempt_count}"
        
        result = await self.sla_service.track_phase_performance(
            "proposal_generation",  # Has 3 retries
            flaky_operation,
            {"test": "retry_test"}
        )
        
        assert result == "success_on_attempt_3"
        assert attempt_count == 3
        
        # Verify retry count was recorded
        latest_metric = self.sla_service.performance_metrics[-1]
        assert latest_metric["retry_count"] == 2  # 2 retries (3rd attempt succeeded)
        
        print(f"✅ Retry mechanism worked: succeeded on attempt {attempt_count}")
    
    async def test_performance_summary(self):
        """Test performance summary generation."""
        print("🔍 Testing performance summary...")
        
        # Generate some test operations
        async def test_op():
            await asyncio.sleep(0.008)
            return "test"
        
        for i in range(5):
            await self.sla_service.track_phase_performance("safety_validation", test_op)
        
        summary = self.sla_service.get_performance_summary()
        
        # Verify summary structure
        assert "total_budget_ms" in summary
        assert "allocated_budget_ms" in summary
        assert "performance_stats" in summary
        assert "recent_violations" in summary
        assert "phase_count" in summary
        
        assert summary["total_budget_ms"] == 250
        assert summary["allocated_budget_ms"] == 250
        assert summary["phase_count"] == 6
        
        print(f"✅ Performance summary generated: {summary['performance_stats']['total_operations']} operations tracked")


async def main():
    """Run all Performance SLA Framework tests."""
    print("⚡ Testing Performance SLA Framework (Simplified)")
    print("=" * 60)
    
    test_framework = TestPerformanceSLAFramework()
    
    try:
        await test_framework.test_latency_budget_allocation()
        await test_framework.test_performance_tier_classification()
        await test_framework.test_successful_operation_tracking()
        await test_framework.test_sla_violation_detection()
        await test_framework.test_timeout_handling()
        await test_framework.test_retry_mechanism()
        await test_framework.test_performance_summary()
        
        print("\n" + "=" * 60)
        print("✅ All Performance SLA Framework Tests Completed Successfully!")
        print("⚡ Latency Budget Allocation: ✅ Working (250ms total)")
        print("🎯 Performance Tier Classification: ✅ Working") 
        print("📊 SLA Violation Detection: ✅ Working")
        print("⏱️ Timeout & Retry Handling: ✅ Working")
        print("🔄 Retry Mechanisms: ✅ Working")
        print("📈 Performance Summary: ✅ Working")
        print("\n🎉 Performance SLA Framework Core Implementation Complete!")
        
        # Show final performance summary
        summary = test_framework.sla_service.get_performance_summary()
        print(f"\n📊 Final Performance Summary:")
        print(f"   Total Operations: {summary['performance_stats']['total_operations']}")
        print(f"   Successful Operations: {summary['performance_stats']['successful_operations']}")
        print(f"   SLA Violations: {summary['performance_stats']['sla_violations']}")
        print(f"   Average Latency: {summary['performance_stats']['average_latency_ms']:.1f}ms")
        print(f"   Budget Utilization: {summary['allocated_budget_ms']}/{summary['total_budget_ms']}ms")
        
    except Exception as e:
        print(f"\n❌ Performance SLA Framework test failed: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    return True


if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
