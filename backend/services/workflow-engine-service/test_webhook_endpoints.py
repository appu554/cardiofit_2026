#!/usr/bin/env python3
"""
Test webhook endpoints across all microservices.
"""

import asyncio
import httpx
import json


async def test_webhook_health_endpoints():
    """Test webhook health endpoints for all microservices."""
    print("🔍 Testing Webhook Health Endpoints...")
    
    services = {
        "patient-service": "http://localhost:8003/api/webhooks/workflow-events/health",
        "encounter-service": "http://localhost:8020/api/webhooks/workflow-events/health", 
        "order-service": "http://localhost:8013/api/webhooks/workflow-events/health",
        "scheduling-service": "http://localhost:8014/api/webhooks/workflow-events/health",
        "organization-service": "http://localhost:8012/api/webhooks/workflow-events/health",
        "medication-service": "http://localhost:8009/api/webhooks/workflow-events/health"
    }
    
    async with httpx.AsyncClient(timeout=10.0) as client:
        for service_name, health_url in services.items():
            try:
                response = await client.get(health_url)
                if response.status_code == 200:
                    data = response.json()
                    print(f"   ✅ {service_name}: {data.get('status', 'unknown')} (v{data.get('webhook_version', '?')})")
                else:
                    print(f"   ❌ {service_name}: HTTP {response.status_code}")
            except Exception as e:
                print(f"   ❌ {service_name}: {str(e)}")


async def test_webhook_event_endpoints():
    """Test sending actual events to webhook endpoints."""
    print("\n📡 Testing Webhook Event Endpoints...")
    
    services = {
        "patient-service": "http://localhost:8003/api/webhooks/workflow-events",
        "encounter-service": "http://localhost:8020/api/webhooks/workflow-events", 
        "order-service": "http://localhost:8013/api/webhooks/workflow-events",
        "scheduling-service": "http://localhost:8014/api/webhooks/workflow-events",
        "organization-service": "http://localhost:8012/api/webhooks/workflow-events",
        "medication-service": "http://localhost:8009/api/webhooks/workflow-events"
    }
    
    test_event = {
        "event_type": "workflow.test.event",
        "event_data": {
            "test": "data",
            "workflow_instance_id": "test-workflow-123",
            "patient_id": "test-patient-123"
        },
        "source": "workflow-engine-service",
        "created_at": "2024-01-01T00:00:00Z"
    }
    
    async with httpx.AsyncClient(timeout=10.0) as client:
        for service_name, webhook_url in services.items():
            try:
                response = await client.post(
                    webhook_url,
                    json=test_event,
                    headers={"Content-Type": "application/json"}
                )
                if response.status_code in [200, 201, 202]:
                    print(f"   ✅ {service_name}: Event received successfully")
                else:
                    print(f"   ❌ {service_name}: HTTP {response.status_code}")
            except Exception as e:
                print(f"   ❌ {service_name}: {str(e)}")


async def main():
    """Run webhook endpoint tests."""
    print("=" * 60)
    print("WORKFLOW ENGINE SERVICE - WEBHOOK ENDPOINT TESTS")
    print("=" * 60)
    
    await test_webhook_health_endpoints()
    await test_webhook_event_endpoints()
    
    print("\n" + "=" * 60)
    print("NEXT STEPS:")
    print("1. If all endpoints are healthy, run the Phase 4 integration test:")
    print("   python test_phase4_integration.py")
    print("2. The event publisher should now send real webhooks instead of mock messages")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
