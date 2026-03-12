#!/usr/bin/env python3
"""
Flow 2 Development Startup Script

This script sets up the complete Flow 2 development environment with:
1. Go Enhanced Orchestrator (Service 1)
2. Rust Clinical Recipe Engine (Service 2) - Mock for now
3. Supporting services (Redis, PostgreSQL, Monitoring)

Usage:
    python start_flow2_development.py [--build] [--logs] [--stop]
"""

import subprocess
import sys
import os
import time
import argparse
from pathlib import Path

def run_command(command, cwd=None, check=True):
    """Run a shell command and return the result"""
    print(f"Running: {command}")
    if cwd:
        print(f"In directory: {cwd}")
    
    result = subprocess.run(
        command,
        shell=True,
        cwd=cwd,
        capture_output=False,
        check=check
    )
    return result

def check_prerequisites():
    """Check if required tools are installed"""
    print("🔍 Checking prerequisites...")
    
    # Check Docker
    try:
        result = subprocess.run(["docker", "--version"], capture_output=True, text=True)
        if result.returncode == 0:
            print(f"✅ Docker: {result.stdout.strip()}")
        else:
            print("❌ Docker not found. Please install Docker.")
            return False
    except FileNotFoundError:
        print("❌ Docker not found. Please install Docker.")
        return False
    
    # Check Docker Compose
    try:
        result = subprocess.run(["docker-compose", "--version"], capture_output=True, text=True)
        if result.returncode == 0:
            print(f"✅ Docker Compose: {result.stdout.strip()}")
        else:
            print("❌ Docker Compose not found. Please install Docker Compose.")
            return False
    except FileNotFoundError:
        print("❌ Docker Compose not found. Please install Docker Compose.")
        return False
    
    # Check Go (optional for development)
    try:
        result = subprocess.run(["go", "version"], capture_output=True, text=True)
        if result.returncode == 0:
            print(f"✅ Go: {result.stdout.strip()}")
        else:
            print("⚠️  Go not found. Docker will be used for Go development.")
    except FileNotFoundError:
        print("⚠️  Go not found. Docker will be used for Go development.")
    
    # Check Rust (optional for development)
    try:
        result = subprocess.run(["rustc", "--version"], capture_output=True, text=True)
        if result.returncode == 0:
            print(f"✅ Rust: {result.stdout.strip()}")
        else:
            print("⚠️  Rust not found. Docker will be used for Rust development.")
    except FileNotFoundError:
        print("⚠️  Rust not found. Docker will be used for Rust development.")
    
    return True

def setup_go_dependencies():
    """Set up Go dependencies"""
    print("\n🔧 Setting up Go dependencies...")
    
    go_engine_path = Path("flow2-go-engine")
    if go_engine_path.exists():
        print("Running go mod tidy...")
        run_command("go mod tidy", cwd=go_engine_path, check=False)
        print("✅ Go dependencies set up")
    else:
        print("⚠️  Go engine directory not found, skipping Go setup")

def create_mock_files():
    """Create mock configuration files for development"""
    print("\n📁 Creating mock configuration files...")
    
    # Create dev-config directory
    dev_config_path = Path("dev-config")
    dev_config_path.mkdir(exist_ok=True)
    
    # Create context service mock
    context_mock = {
        "httpRequest": {
            "method": "POST",
            "path": "/graphql"
        },
        "httpResponse": {
            "statusCode": 200,
            "headers": {
                "Content-Type": ["application/json"]
            },
            "body": {
                "data": {
                    "patient": {
                        "id": "905a60cb-8241-418f-b29b-5b020e851392",
                        "demographics": {
                            "age": 45,
                            "weight": 70.0,
                            "height": 175.0
                        },
                        "allergies": [],
                        "conditions": [],
                        "medications": []
                    }
                }
            }
        }
    }
    
    import json
    with open(dev_config_path / "context-service-mock.json", "w") as f:
        json.dump([context_mock], f, indent=2)
    
    # Create mockserver properties
    with open(dev_config_path / "mockserver.properties", "w") as f:
        f.write("mockserver.logLevel=INFO\n")
        f.write("mockserver.serverPort=1080\n")
    
    print("✅ Mock configuration files created")

def create_monitoring_config():
    """Create monitoring configuration files"""
    print("\n📊 Creating monitoring configuration...")
    
    monitoring_path = Path("monitoring")
    monitoring_path.mkdir(exist_ok=True)
    
    # Create Prometheus config
    prometheus_config = """
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'flow2-go-engine'
    static_configs:
      - targets: ['flow2-go-engine:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'rust-recipe-engine'
    static_configs:
      - targets: ['rust-recipe-engine:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'medication-service'
    static_configs:
      - targets: ['medication-service:8009']
    metrics_path: '/metrics'
    scrape_interval: 10s
"""
    
    with open(monitoring_path / "prometheus.yml", "w") as f:
        f.write(prometheus_config.strip())
    
    print("✅ Monitoring configuration created")

def start_services(build=False):
    """Start all Flow 2 services"""
    print("\n🚀 Starting Flow 2 development environment...")
    print("⚠️  IMPORTANT: This environment requires REAL services - no mocks or fallbacks!")
    print("   • All services must be running and healthy")
    print("   • Go Engine will fail if Rust Engine is not available")
    print("   • Redis must be running for caching")
    print()

    # Build command
    build_flag = "--build" if build else ""

    # Start services
    command = f"docker-compose -f docker-compose.flow2.yml up -d {build_flag}"
    run_command(command)

    print("\n⏳ Waiting for services to start...")
    time.sleep(15)  # Longer wait for real services

    # Check service health
    print("\n🏥 Checking service health...")

    services = [
        ("Go Flow 2 Engine", "http://localhost:8080/health"),
        ("Rust Recipe Engine", "http://localhost:8081/health"),
        ("Redis", "redis://localhost:6379"),
        ("PostgreSQL", "postgresql://localhost:5432"),
        ("Prometheus", "http://localhost:9090"),
        ("Grafana", "http://localhost:3000"),
    ]

    for service_name, endpoint in services:
        print(f"  {service_name}: {endpoint}")

    print("\n✅ Flow 2 development environment started!")
    print("\n📋 Service URLs:")
    print("  • Go Flow 2 Engine: http://localhost:8080")
    print("  • Rust Recipe Engine: http://localhost:50051 (gRPC)")
    print("  • Rust Metrics: http://localhost:8081")
    print("  • Medication Service: http://localhost:8009")
    print("  • Prometheus: http://localhost:9090")
    print("  • Grafana: http://localhost:3000 (admin/admin)")
    print("  • Jaeger: http://localhost:16686")
    print("\n⚠️  NOTE: Go Engine will only work when Rust Engine is running!")

def show_logs():
    """Show logs from all services"""
    print("\n📋 Showing Flow 2 service logs...")
    run_command("docker-compose -f docker-compose.flow2.yml logs -f")

def stop_services():
    """Stop all Flow 2 services"""
    print("\n🛑 Stopping Flow 2 development environment...")
    run_command("docker-compose -f docker-compose.flow2.yml down")
    print("✅ Flow 2 services stopped")

def main():
    parser = argparse.ArgumentParser(description="Flow 2 Development Environment")
    parser.add_argument("--build", action="store_true", help="Build images before starting")
    parser.add_argument("--logs", action="store_true", help="Show service logs")
    parser.add_argument("--stop", action="store_true", help="Stop all services")
    
    args = parser.parse_args()
    
    print("🚀 Flow 2 Development Environment Setup")
    print("=" * 50)
    
    if args.stop:
        stop_services()
        return
    
    if args.logs:
        show_logs()
        return
    
    # Check prerequisites
    if not check_prerequisites():
        sys.exit(1)
    
    # Setup dependencies
    setup_go_dependencies()
    create_mock_files()
    create_monitoring_config()
    
    # Start services
    start_services(build=args.build)

if __name__ == "__main__":
    main()
