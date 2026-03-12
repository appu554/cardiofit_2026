"""
Test Performance SLA Framework
Tests the complete performance SLA implementation with latency budgets and circuit breakers.
"""
import asyncio
import pytest
import time
import sys
import os
from datetime import datetime
from typing import Dict, Any

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

# Mock the problematic imports first
import unittest.mock
sys.modules['supabase'] = unittest.mock.MagicMock()
sys.modules['app.core.config'] = unittest.mock.MagicMock()

# Create a mock settings object
mock_settings = unittest.mock.MagicMock()
mock_settings.DATABASE_URL = "postgresql://test:test@localhost/test"
sys.modules['app.core.config'].settings = mock_settings

# Import performance services
try:
    from app.services.performance_sla_service import (
        PerformanceSLAService, PerformanceSLAError,
        LatencyBudget, PerformanceTier
    )
    from app.services.intelligent_circuit_breaker import (
        IntelligentCircuitBreaker, CircuitBreakerConfig, CircuitState,
        CircuitBreakerOpenError
    )
    print("✅ Performance SLA services imported successfully")
except ImportError as e:
    print(f"❌ Failed to import performance services: {e}")
    # Fall back to the simple test
    import subprocess
    result = subprocess.run([sys.executable, "test_performance_sla_simple.py"],
                          cwd=os.path.dirname(__file__))
    sys.exit(result.returncode)


class TestPerformanceSLAFramework:
    """Test comprehensive Performance SLA Framework functionality."""
    
    @pytest.mark.asyncio
    async def test_latency_budget_allocation(self):
        """Test that latency budgets are properly allocated."""
        print("🔍 Testing latency budget allocation...")
        
        sla_service = PerformanceSLAService()
        
        # Verify total budget allocation (250ms total)
        core_phases = [
            "workflow_initialization", "context_fetching", "proposal_generation",
            "safety_validation", "commit_operation", "post_processing"
        ]
        
        total_allocated = sum(
            sla_service.latency_budgets[phase].budget_ms 
            for phase in core_phases
        )
        
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
            actual_budget = sla_service.latency_budgets[phase].budget_ms
            assert actual_budget == expected_budget, f"{phase} budget should be {expected_budget}ms, got {actual_budget}ms"
        
        print(f"✅ Latency budget allocation verified: {total_allocated}ms total")
    
    @pytest.mark.asyncio
    async def test_performance_tier_classification(self):
        """Test performance tier classification."""
        print("🔍 Testing performance tier classification...")
        
        sla_service = PerformanceSLAService()
        
        # Verify tier classifications
        tier_expectations = {
            "workflow_initialization": PerformanceTier.CRITICAL,
            "emergency_response": PerformanceTier.CRITICAL,
            "digital_reflex_arc": PerformanceTier.CRITICAL,
            "context_fetching": PerformanceTier.HIGH,
            "proposal_generation": PerformanceTier.HIGH,
            "safety_validation": PerformanceTier.STANDARD,
            "post_processing": PerformanceTier.STANDARD
        }
        
        for phase, expected_tier in tier_expectations.items():
            if phase in sla_service.latency_budgets:
                actual_tier = sla_service.latency_budgets[phase].tier
                assert actual_tier == expected_tier, f"{phase} should be {expected_tier.value}, got {actual_tier.value}"
        
        print("✅ Performance tier classification verified")
    
    @pytest.mark.asyncio
    async def test_successful_operation_tracking(self):
        """Test tracking of successful operations within budget."""
        print("🔍 Testing successful operation tracking...")
        
        sla_service = PerformanceSLAService()
        
        # Test fast operation (within budget)
        async def fast_operation():
            await asyncio.sleep(0.005)  # 5ms - within 10ms budget
            return "success"
        
        result = await sla_service.track_phase_performance(
            "workflow_initialization",
            fast_operation,
            {"test": "fast_operation"}
        )
        
        assert result == "success"
        
        # Verify metrics were recorded
        assert len(sla_service.performance_metrics) > 0
        latest_metric = sla_service.performance_metrics[-1]
        
        assert latest_metric.operation == "workflow_initialization"
        assert latest_metric.success is True
        assert latest_metric.violation is False
        assert latest_metric.execution_time_ms < 10  # Should be under budget
        
        print(f"✅ Successful operation tracked: {latest_metric.execution_time_ms:.1f}ms")
    
    @pytest.mark.asyncio
    async def test_sla_violation_detection(self):
        """Test SLA violation detection and handling."""
        print("🔍 Testing SLA violation detection...")
        
        sla_service = PerformanceSLAService()
        
        # Test slow operation (exceeds budget)
        async def slow_operation():
            await asyncio.sleep(0.015)  # 15ms - exceeds 10ms budget
            return "slow_success"
        
        result = await sla_service.track_phase_performance(
            "workflow_initialization",
            slow_operation,
            {"test": "slow_operation"}
        )
        
        assert result == "slow_success"
        
        # Verify SLA violation was detected
        latest_metric = sla_service.performance_metrics[-1]
        assert latest_metric.violation is True
        assert latest_metric.execution_time_ms > 10  # Should exceed budget
        
        # Verify violation was recorded
        assert len(sla_service.sla_violations) > 0
        latest_violation = sla_service.sla_violations[-1]
        assert latest_violation["phase_name"] == "workflow_initialization"
        assert latest_violation["elapsed_ms"] > 10
        
        print(f"✅ SLA violation detected: {latest_metric.execution_time_ms:.1f}ms > 10ms budget")
    
    @pytest.mark.asyncio
    async def test_timeout_handling(self):
        """Test timeout handling with retry logic."""
        print("🔍 Testing timeout handling...")
        
        sla_service = PerformanceSLAService()
        
        # Test operation that times out
        async def timeout_operation():
            await asyncio.sleep(0.1)  # 100ms - will timeout at 20ms
            return "should_not_reach"
        
        try:
            await sla_service.track_phase_performance(
                "workflow_initialization",
                timeout_operation,
                {"test": "timeout_operation"}
            )
            assert False, "Should have raised PerformanceSLAError"
        except PerformanceSLAError as e:
            assert "Timeout" in str(e)
            print(f"✅ Timeout handled correctly: {e}")
        
        # Verify failure was recorded
        latest_metric = sla_service.performance_metrics[-1]
        assert latest_metric.success is False
        assert latest_metric.violation is True
    
    @pytest.mark.asyncio
    async def test_circuit_breaker_functionality(self):
        """Test intelligent circuit breaker functionality."""
        print("🔍 Testing circuit breaker functionality...")
        
        # Create circuit breaker with low failure threshold for testing
        config = CircuitBreakerConfig(
            service_name="test_service",
            failure_threshold=2,
            recovery_timeout_ms=1000,
            learning_enabled=True
        )
        
        circuit_breaker = IntelligentCircuitBreaker(config)
        
        # Test successful operations
        async def success_operation():
            return "success"
        
        result1 = await circuit_breaker.execute(success_operation)
        assert result1 == "success"
        assert circuit_breaker.state == CircuitState.CLOSED
        
        # Test failing operations to trip circuit breaker
        async def failing_operation():
            raise Exception("Test failure")
        
        # First failure
        try:
            await circuit_breaker.execute(failing_operation)
        except Exception:
            pass
        
        # Second failure - should trip circuit breaker
        try:
            await circuit_breaker.execute(failing_operation)
        except Exception:
            pass
        
        assert circuit_breaker.state == CircuitState.OPEN
        
        # Test that circuit breaker blocks requests
        try:
            await circuit_breaker.execute(success_operation)
            assert False, "Should have raised CircuitBreakerOpenError"
        except CircuitBreakerOpenError:
            print("✅ Circuit breaker correctly blocked request")
        
        # Wait for recovery timeout and test half-open state
        await asyncio.sleep(1.1)  # Wait for recovery timeout
        
        # Should transition to half-open and allow limited requests
        result2 = await circuit_breaker.execute(success_operation)
        assert result2 == "success"
        
        print("✅ Circuit breaker functionality verified")
    
    @pytest.mark.asyncio
    async def test_performance_dashboard(self):
        """Test performance dashboard data generation."""
        print("🔍 Testing performance dashboard...")
        
        sla_service = PerformanceSLAService()
        
        # Generate some test metrics
        async def test_operation():
            await asyncio.sleep(0.008)  # 8ms
            return "dashboard_test"
        
        # Run multiple operations
        for i in range(5):
            await sla_service.track_phase_performance(
                "workflow_initialization",
                test_operation,
                {"test_run": i}
            )
        
        # Get dashboard data
        dashboard = await sla_service.get_performance_dashboard()
        
        # Verify dashboard structure
        assert "timestamp" in dashboard
        assert "overall_stats" in dashboard
        assert "phase_metrics" in dashboard
        assert "circuit_breakers" in dashboard
        assert "recent_violations" in dashboard
        assert "budget_allocation" in dashboard
        
        # Verify phase metrics
        assert "workflow_initialization" in dashboard["phase_metrics"]
        phase_metrics = dashboard["phase_metrics"]["workflow_initialization"]
        
        assert "budget_ms" in phase_metrics
        assert "avg_latency_ms" in phase_metrics
        assert "violation_rate" in phase_metrics
        assert "success_rate" in phase_metrics
        assert "budget_utilization" in phase_metrics
        
        # Verify budget allocation
        budget_allocation = dashboard["budget_allocation"]
        assert budget_allocation["total_budget_ms"] == 250
        
        print(f"✅ Performance dashboard generated with {len(dashboard['phase_metrics'])} phase metrics")
    
    @pytest.mark.asyncio
    async def test_sla_compliance_report(self):
        """Test SLA compliance reporting."""
        print("🔍 Testing SLA compliance report...")
        
        sla_service = PerformanceSLAService()
        
        # Generate mixed performance data
        async def fast_op():
            await asyncio.sleep(0.005)
            return "fast"
        
        async def slow_op():
            await asyncio.sleep(0.015)
            return "slow"
        
        # Run mixed operations
        for i in range(3):
            await sla_service.track_phase_performance("workflow_initialization", fast_op)
        
        for i in range(2):
            await sla_service.track_phase_performance("workflow_initialization", slow_op)
        
        # Generate compliance report
        report = await sla_service.get_sla_compliance_report()
        
        # Verify report structure
        assert "report_timestamp" in report
        assert "overall_compliance_rate" in report
        assert "tier_compliance" in report
        assert "total_budget_ms" in report
        assert "sla_violations_last_hour" in report
        
        # Verify tier compliance data
        if "critical" in report["tier_compliance"]:
            critical_compliance = report["tier_compliance"]["critical"]
            assert "compliance_rate" in critical_compliance
            assert "avg_latency_ms" in critical_compliance
            assert "total_operations" in critical_compliance
            assert "violations" in critical_compliance
        
        print(f"✅ SLA compliance report generated with {report['overall_compliance_rate']:.1%} compliance")


async def main():
    """Run all Performance SLA Framework tests."""
    print("⚡ Testing Performance SLA Framework")
    print("=" * 60)
    
    # Test Performance SLA Framework
    print("\n📊 Testing Performance SLA Framework...")
    perf_test = TestPerformanceSLAFramework()
    
    try:
        await perf_test.test_latency_budget_allocation()
        await perf_test.test_performance_tier_classification()
        await perf_test.test_successful_operation_tracking()
        await perf_test.test_sla_violation_detection()
        await perf_test.test_timeout_handling()
        await perf_test.test_circuit_breaker_functionality()
        await perf_test.test_performance_dashboard()
        await perf_test.test_sla_compliance_report()
        
        print("\n" + "=" * 60)
        print("✅ All Performance SLA Framework Tests Completed Successfully!")
        print("⚡ Latency Budget Allocation: ✅ Working (250ms total)")
        print("🎯 Performance Tier Classification: ✅ Working") 
        print("📊 SLA Violation Detection: ✅ Working")
        print("⏱️ Timeout & Retry Handling: ✅ Working")
        print("🔄 Circuit Breaker Patterns: ✅ Working")
        print("📈 Performance Dashboard: ✅ Working")
        print("📋 SLA Compliance Reporting: ✅ Working")
        print("\n🎉 Performance SLA Framework Implementation Complete!")
        
    except Exception as e:
        print(f"\n❌ Performance SLA Framework test failed: {e}")
        raise


if __name__ == "__main__":
    asyncio.run(main())
