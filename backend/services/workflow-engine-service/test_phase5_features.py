#!/usr/bin/env python3
"""
Test script for Phase 5 Advanced Features implementation.
Tests timer management, escalation mechanisms, gateway handling, and error recovery.
"""

import asyncio
import sys
import os
import json
from datetime import datetime, timedelta
from typing import Dict, Any

# Add the app directory to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

from app.services.timer_service import timer_service
from app.services.escalation_service import escalation_service
from app.services.gateway_service import gateway_service
from app.services.error_recovery_service import error_recovery_service, ErrorType, RecoveryStrategy
from app.db.database import get_db


async def test_timer_service():
    """Test timer service functionality."""
    print("\n=== Testing Timer Service ===")
    
    try:
        # Initialize timer service
        success = await timer_service.initialize()
        print(f"Timer service initialization: {'✓' if success else '✗'}")
        
        if not success:
            return False
        
        # Test creating a timer
        due_date = datetime.utcnow() + timedelta(seconds=30)
        timer = await timer_service.create_timer(
            workflow_instance_id=1,
            timer_name="test_timer",
            due_date=due_date,
            timer_type="deadline",
            timer_data={"test": "data"}
        )
        
        print(f"Timer creation: {'✓' if timer else '✗'}")
        if timer:
            print(f"  Timer ID: {timer.id}")
            print(f"  Timer name: {timer.timer_name}")
            print(f"  Due date: {timer.due_date}")
            print(f"  Status: {timer.status}")
        
        # Test cancelling a timer
        if timer:
            cancelled = await timer_service.cancel_timer(timer.id, "test_cancellation")
            print(f"Timer cancellation: {'✓' if cancelled else '✗'}")
        
        return True
        
    except Exception as e:
        print(f"Timer service test failed: {e}")
        return False


async def test_escalation_service():
    """Test escalation service functionality."""
    print("\n=== Testing Escalation Service ===")
    
    try:
        # Initialize escalation service
        success = await escalation_service.initialize()
        print(f"Escalation service initialization: {'✓' if success else '✗'}")
        
        if not success:
            return False
        
        # Test creating an escalation chain
        escalation_created = await escalation_service.create_escalation_chain(
            task_id=1,
            escalation_type="human_task"
        )
        
        print(f"Escalation chain creation: {'✓' if escalation_created else '✗'}")
        
        # Test cancelling an escalation chain
        if escalation_created:
            cancelled = await escalation_service.cancel_escalation_chain(
                task_id=1,
                reason="test_cancellation"
            )
            print(f"Escalation chain cancellation: {'✓' if cancelled else '✗'}")
        
        return True
        
    except Exception as e:
        print(f"Escalation service test failed: {e}")
        return False


async def test_gateway_service():
    """Test gateway service functionality."""
    print("\n=== Testing Gateway Service ===")
    
    try:
        # Initialize gateway service
        success = await gateway_service.initialize()
        print(f"Gateway service initialization: {'✓' if success else '✗'}")
        
        if not success:
            return False
        
        # Test creating a parallel gateway
        parallel_created = await gateway_service.create_parallel_gateway(
            gateway_id="test_parallel_gateway",
            workflow_instance_id=1,
            required_tokens=["token1", "token2", "token3"],
            timeout_minutes=5
        )
        
        print(f"Parallel gateway creation: {'✓' if parallel_created else '✗'}")
        
        # Test signaling the gateway
        if parallel_created:
            signal1 = await gateway_service.signal_gateway(
                gateway_id="test_parallel_gateway",
                token_name="token1",
                token_data={"source": "test"}
            )
            print(f"Gateway signal 1: {'✓' if signal1 else '✗'}")
            
            signal2 = await gateway_service.signal_gateway(
                gateway_id="test_parallel_gateway",
                token_name="token2",
                token_data={"source": "test"}
            )
            print(f"Gateway signal 2: {'✓' if signal2 else '✗'}")
        
        # Test creating an inclusive gateway
        inclusive_created = await gateway_service.create_inclusive_gateway(
            gateway_id="test_inclusive_gateway",
            workflow_instance_id=1,
            possible_tokens=["option1", "option2", "option3"],
            minimum_tokens=2,
            timeout_minutes=3
        )
        
        print(f"Inclusive gateway creation: {'✓' if inclusive_created else '✗'}")
        
        # Test creating an event gateway
        event_created = await gateway_service.create_event_gateway(
            gateway_id="test_event_gateway",
            workflow_instance_id=1,
            event_conditions={
                "patient_admitted": {"type": "any"},
                "lab_results_ready": {"type": "any"}
            },
            timeout_minutes=10
        )
        
        print(f"Event gateway creation: {'✓' if event_created else '✗'}")
        
        # Test gateway status
        if parallel_created:
            status = await gateway_service.get_gateway_status("test_parallel_gateway")
            print(f"Gateway status retrieval: {'✓' if status else '✗'}")
            if status:
                print(f"  Gateway type: {status.get('gateway_type')}")
                print(f"  Required tokens: {status.get('required_tokens')}")
                print(f"  Received tokens: {status.get('received_tokens')}")
                print(f"  Completed: {status.get('completed')}")
        
        return True
        
    except Exception as e:
        print(f"Gateway service test failed: {e}")
        return False


async def test_error_recovery_service():
    """Test error recovery service functionality."""
    print("\n=== Testing Error Recovery Service ===")
    
    try:
        # Initialize error recovery service
        success = await error_recovery_service.initialize()
        print(f"Error recovery service initialization: {'✓' if success else '✗'}")
        
        if not success:
            return False
        
        # Test handling different types of errors
        error_types = [
            (ErrorType.TASK_FAILURE, "Task execution failed"),
            (ErrorType.SERVICE_UNAVAILABLE, "External service is unavailable"),
            (ErrorType.TIMEOUT, "Operation timed out"),
            (ErrorType.VALIDATION_ERROR, "Data validation failed"),
            (ErrorType.NETWORK_ERROR, "Network connection error")
        ]
        
        error_ids = []
        for error_type, error_message in error_types:
            error_id = await error_recovery_service.handle_error(
                workflow_instance_id=1,
                error_type=error_type,
                error_message=error_message,
                error_data={"test": "data", "timestamp": datetime.utcnow().isoformat()}
            )
            
            if error_id:
                error_ids.append(error_id)
                print(f"Error handling ({error_type.value}): ✓ (ID: {error_id})")
            else:
                print(f"Error handling ({error_type.value}): ✗")
        
        # Test error status retrieval
        for error_id in error_ids[:2]:  # Test first 2 errors
            status = await error_recovery_service.get_error_status(error_id)
            print(f"Error status retrieval ({error_id}): {'✓' if status else '✗'}")
            if status:
                print(f"  Error type: {status.get('error_type')}")
                print(f"  Recovery strategy: {status.get('recovery_strategy')}")
                print(f"  Retry count: {status.get('retry_count')}")
        
        # Test manual retry
        if error_ids:
            retry_success = await error_recovery_service.handle_retry_timer(
                error_id=error_ids[0],
                retry_count=1,
                retry_handler="default"
            )
            print(f"Manual retry: {'✓' if retry_success else '✗'}")
        
        return True
        
    except Exception as e:
        print(f"Error recovery service test failed: {e}")
        return False


async def test_integration():
    """Test integration between Phase 5 services."""
    print("\n=== Testing Service Integration ===")
    
    try:
        # Test timer-based escalation
        print("Testing timer-based escalation...")
        
        # Create a timer that should trigger an escalation
        escalation_timer = await timer_service.create_timer(
            workflow_instance_id=1,
            timer_name="escalation_test_timer",
            due_date=datetime.utcnow() + timedelta(seconds=5),
            timer_type="escalation",
            timer_data={
                "task_id": 1,
                "escalation_level": 1,
                "escalation_rule": {
                    "target_type": "supervisor",
                    "target_value": "test_supervisor",
                    "action": "notify"
                }
            },
            callback_name="task_escalation"
        )
        
        print(f"Escalation timer creation: {'✓' if escalation_timer else '✗'}")
        
        # Test error-triggered gateway timeout
        print("Testing error-triggered gateway timeout...")
        
        # Create a gateway with short timeout
        timeout_gateway = await gateway_service.create_parallel_gateway(
            gateway_id="timeout_test_gateway",
            workflow_instance_id=1,
            required_tokens=["never_coming_token"],
            timeout_minutes=1  # Very short timeout for testing
        )
        
        print(f"Timeout gateway creation: {'✓' if timeout_gateway else '✗'}")
        
        # Test error recovery with gateway
        print("Testing error recovery with gateway...")
        
        error_id = await error_recovery_service.handle_error(
            workflow_instance_id=1,
            error_type=ErrorType.TIMEOUT,
            error_message="Gateway timeout occurred",
            error_data={
                "gateway_id": "timeout_test_gateway",
                "gateway_type": "parallel"
            },
            custom_strategy=RecoveryStrategy.ALTERNATIVE_PATH
        )
        
        print(f"Gateway error handling: {'✓' if error_id else '✗'}")
        
        return True
        
    except Exception as e:
        print(f"Integration test failed: {e}")
        return False


async def test_database_models():
    """Test database models for Phase 5 features."""
    print("\n=== Testing Database Models ===")
    
    try:
        from app.models.workflow_models import (
            WorkflowTimer, WorkflowEscalation, WorkflowGateway, WorkflowError
        )
        
        db = next(get_db())
        
        # Test timer model
        timer = WorkflowTimer(
            workflow_instance_id=1,
            timer_name="test_db_timer",
            due_date=datetime.utcnow() + timedelta(hours=1),
            status="active",
            timer_data={"test": "database"}
        )
        
        db.add(timer)
        db.commit()
        db.refresh(timer)
        
        print(f"Timer model creation: ✓ (ID: {timer.id})")
        
        # Test escalation model
        escalation = WorkflowEscalation(
            workflow_instance_id=1,
            task_id=1,
            escalation_level=1,
            escalation_type="test_escalation",
            escalation_target="test_user",
            escalation_reason="Testing database model",
            status="active",
            escalation_data={"test": "database"}
        )
        
        db.add(escalation)
        db.commit()
        db.refresh(escalation)
        
        print(f"Escalation model creation: ✓ (ID: {escalation.id})")
        
        # Test gateway model
        gateway = WorkflowGateway(
            workflow_instance_id=1,
            gateway_id="test_db_gateway",
            gateway_type="parallel",
            required_tokens=["token1", "token2"],
            received_tokens=["token1"],
            status="waiting",
            gateway_data={"test": "database"}
        )
        
        db.add(gateway)
        db.commit()
        db.refresh(gateway)
        
        print(f"Gateway model creation: ✓ (ID: {gateway.id})")
        
        # Test error model
        error = WorkflowError(
            workflow_instance_id=1,
            error_id="test_db_error",
            error_type="system_error",
            error_message="Testing database model",
            recovery_strategy="retry",
            status="active",
            error_data={"test": "database"}
        )
        
        db.add(error)
        db.commit()
        db.refresh(error)
        
        print(f"Error model creation: ✓ (ID: {error.id})")
        
        # Clean up test records
        db.delete(timer)
        db.delete(escalation)
        db.delete(gateway)
        db.delete(error)
        db.commit()
        
        print("Database cleanup: ✓")
        
        return True
        
    except Exception as e:
        print(f"Database model test failed: {e}")
        return False


async def main():
    """Run all Phase 5 tests."""
    print("🚀 Starting Phase 5 Advanced Features Tests")
    print("=" * 50)
    
    tests = [
        ("Timer Service", test_timer_service),
        ("Escalation Service", test_escalation_service),
        ("Gateway Service", test_gateway_service),
        ("Error Recovery Service", test_error_recovery_service),
        ("Database Models", test_database_models),
        ("Service Integration", test_integration)
    ]
    
    results = {}
    
    for test_name, test_func in tests:
        print(f"\n🧪 Running {test_name} tests...")
        try:
            result = await test_func()
            results[test_name] = result
            status = "✅ PASSED" if result else "❌ FAILED"
            print(f"{test_name}: {status}")
        except Exception as e:
            results[test_name] = False
            print(f"{test_name}: ❌ FAILED - {e}")
    
    # Summary
    print("\n" + "=" * 50)
    print("📊 Test Summary")
    print("=" * 50)
    
    passed = sum(1 for result in results.values() if result)
    total = len(results)
    
    for test_name, result in results.items():
        status = "✅ PASSED" if result else "❌ FAILED"
        print(f"{test_name:25} {status}")
    
    print(f"\nOverall: {passed}/{total} tests passed")
    
    if passed == total:
        print("🎉 All Phase 5 Advanced Features tests passed!")
        return 0
    else:
        print("⚠️  Some tests failed. Please check the implementation.")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
