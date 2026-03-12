"""
Quick health check script for all services
"""
import requests
import asyncio
import aiohttp
from typing import Dict

async def check_services_health():
    """Check health of all services."""
    
    services = {
        "Federation Gateway": "http://localhost:4000/health",
        "API Gateway": "http://localhost:8005/health", 
        "Auth Service": "http://localhost:8001/health",
        "Workflow Service": "http://localhost:8015/health",
        "Patient Service": "http://localhost:8003/health",
        "Observation Service": "http://localhost:8007/health",
        "Encounter Service": "http://localhost:8020/health"
    }
    
    print("🔍 Checking Services Health...")
    print("=" * 50)
    
    results = {}
    async with aiohttp.ClientSession() as session:
        for service_name, health_url in services.items():
            try:
                async with session.get(health_url, timeout=5) as response:
                    if response.status == 200:
                        health_data = await response.json()
                        print(f"✅ {service_name}: Healthy")
                        if 'version' in health_data:
                            print(f"   Version: {health_data['version']}")
                        results[service_name] = True
                    else:
                        print(f"❌ {service_name}: Unhealthy (Status: {response.status})")
                        results[service_name] = False
            except Exception as e:
                print(f"❌ {service_name}: Not accessible")
                print(f"   Error: {str(e)}")
                results[service_name] = False
    
    # Summary
    healthy_count = sum(1 for status in results.values() if status)
    total_count = len(results)
    
    print("\n" + "=" * 50)
    print(f"📊 Health Summary: {healthy_count}/{total_count} services healthy")
    
    if healthy_count == total_count:
        print("🎉 All services are healthy! Ready for integration testing.")
        print("\nRun the integration test:")
        print("python test_real_services_integration.py")
    elif healthy_count >= 5:
        print("⚠️  Most services are healthy. Integration testing may work.")
        print("Missing services:")
        for service, status in results.items():
            if not status:
                print(f"  - {service}")
    else:
        print("❌ Too many services are down. Please start more services.")
        print("\nTo start all services:")
        print("powershell -ExecutionPolicy Bypass -File start_all_services.ps1")
    
    return results

def check_services_sync():
    """Synchronous version using requests."""
    services = {
        "Federation Gateway": "http://localhost:4000/health",
        "API Gateway": "http://localhost:8005/health", 
        "Auth Service": "http://localhost:8001/health",
        "Workflow Service": "http://localhost:8015/health",
        "Patient Service": "http://localhost:8003/health",
        "Observation Service": "http://localhost:8007/health",
        "Encounter Service": "http://localhost:8020/health"
    }
    
    print("🔍 Quick Health Check...")
    print("=" * 30)
    
    results = {}
    for service_name, health_url in services.items():
        try:
            response = requests.get(health_url, timeout=3)
            if response.status_code == 200:
                print(f"✅ {service_name}")
                results[service_name] = True
            else:
                print(f"❌ {service_name} (Status: {response.status_code})")
                results[service_name] = False
        except Exception:
            print(f"❌ {service_name} (Not accessible)")
            results[service_name] = False
    
    healthy_count = sum(1 for status in results.values() if status)
    total_count = len(results)
    print(f"\n📊 {healthy_count}/{total_count} services healthy")
    
    return results

if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1 and sys.argv[1] == "--sync":
        check_services_sync()
    else:
        asyncio.run(check_services_health())
