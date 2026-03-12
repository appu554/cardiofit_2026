#!/usr/bin/env python3
"""
Startup script for KB-Clinical-Pathways service
Handles database setup and service startup
"""

import subprocess
import time
import sys
import requests
import json

def run_command(command, cwd=None):
    """Run a shell command and return the result"""
    try:
        result = subprocess.run(
            command, 
            shell=True, 
            cwd=cwd,
            capture_output=True, 
            text=True, 
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        print(f"❌ Command failed: {command}")
        print(f"Error: {e.stderr}")
        return None

def check_service_health(url, max_retries=30, delay=2):
    """Check if service is healthy"""
    for i in range(max_retries):
        try:
            response = requests.get(f"{url}/health", timeout=5)
            if response.status_code == 200:
                return True
        except:
            pass
        
        print(f"⏳ Waiting for service... ({i+1}/{max_retries})")
        time.sleep(delay)
    
    return False

def main():
    print("🚀 Starting KB-Clinical-Pathways Service Setup")
    print("=" * 60)
    
    # Step 1: Stop any existing containers
    print("🛑 Stopping existing containers...")
    run_command("docker-compose down")
    
    # Step 2: Start database and dependencies
    print("🗄️  Starting database and dependencies...")
    result = run_command("docker-compose up -d db redis kafka")
    if result is None:
        print("❌ Failed to start dependencies")
        sys.exit(1)
    
    # Step 3: Wait for database to be ready
    print("⏳ Waiting for database to be ready...")
    time.sleep(10)
    
    # Step 4: Build and start the KB-Clinical-Pathways service
    print("🔨 Building KB-Clinical-Pathways service...")
    result = run_command("docker-compose build kb-clinical-pathways")
    if result is None:
        print("❌ Failed to build service")
        sys.exit(1)
    
    print("🚀 Starting KB-Clinical-Pathways service...")
    result = run_command("docker-compose up -d kb-clinical-pathways")
    if result is None:
        print("❌ Failed to start service")
        sys.exit(1)
    
    # Step 5: Wait for service to be healthy
    print("🔍 Checking service health...")
    if not check_service_health("http://localhost:8084"):
        print("❌ Service failed to start properly")
        print("\n📋 Service logs:")
        run_command("docker-compose logs kb-clinical-pathways")
        sys.exit(1)
    
    print("✅ Service is healthy!")
    
    # Step 6: Run basic tests
    print("🧪 Running basic functionality tests...")
    
    # Test health endpoint
    try:
        response = requests.get("http://localhost:8084/health")
        if response.status_code == 200:
            print("✅ Health check passed")
        else:
            print("❌ Health check failed")
    except Exception as e:
        print(f"❌ Health check error: {e}")
    
    # Test readiness endpoint
    try:
        response = requests.get("http://localhost:8084/ready")
        if response.status_code == 200:
            print("✅ Readiness check passed")
        else:
            print("❌ Readiness check failed")
    except Exception as e:
        print(f"❌ Readiness check error: {e}")
    
    # Test list pathways endpoint
    try:
        response = requests.get("http://localhost:8084/v1/pathways")
        if response.status_code == 200:
            data = response.json()
            print(f"✅ List pathways passed - Found {len(data.get('pathways', []))} pathways")
        else:
            print("❌ List pathways failed")
    except Exception as e:
        print(f"❌ List pathways error: {e}")
    
    # Test service stats
    try:
        response = requests.get("http://localhost:8084/v1/stats")
        if response.status_code == 200:
            data = response.json()
            print("✅ Service stats passed")
            print(f"   📊 Pathways: {data.get('pathways', {})}")
            print(f"   📈 Executions: {data.get('executions', {})}")
        else:
            print("❌ Service stats failed")
    except Exception as e:
        print(f"❌ Service stats error: {e}")
    
    print("\n" + "=" * 60)
    print("🎉 KB-Clinical-Pathways Service is running!")
    print("📍 Service URL: http://localhost:8084")
    print("📋 Health Check: http://localhost:8084/health")
    print("📊 Metrics: http://localhost:8084/metrics")
    print("📖 API Docs: See README.md for endpoint documentation")
    
    print("\n🔧 Useful commands:")
    print("  View logs: docker-compose logs -f kb-clinical-pathways")
    print("  Stop service: docker-compose stop kb-clinical-pathways")
    print("  Restart service: docker-compose restart kb-clinical-pathways")
    print("  Run tests: python kb-clinical-pathways/test_service.py")
    
    print("\n📝 Next steps:")
    print("  1. Create your first clinical pathway")
    print("  2. Test pathway execution")
    print("  3. Monitor service metrics")
    
    # Show service status
    print("\n📊 Current service status:")
    run_command("docker-compose ps")

if __name__ == "__main__":
    main()
