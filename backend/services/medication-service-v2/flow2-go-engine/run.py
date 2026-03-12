#!/usr/bin/env python3
"""
Flow 2 Go Engine Runner

Simple script to run the Go Enhanced Orchestrator with proper setup.

Usage:
    python run.py [--dev] [--build] [--test]
"""

import subprocess
import sys
import os
import argparse
from pathlib import Path

def run_command(command, cwd=None, check=True):
    """Run a shell command"""
    print(f"Running: {command}")
    if cwd:
        print(f"In directory: {cwd}")
    
    result = subprocess.run(
        command,
        shell=True,
        cwd=cwd,
        check=check
    )
    return result

def setup_go_environment():
    """Set up Go environment"""
    print("🔧 Setting up Go environment...")
    
    # Check if Go is installed
    try:
        result = subprocess.run(["go", "version"], capture_output=True, text=True)
        if result.returncode == 0:
            print(f"✅ {result.stdout.strip()}")
        else:
            print("❌ Go not found. Please install Go 1.21+")
            return False
    except FileNotFoundError:
        print("❌ Go not found. Please install Go 1.21+")
        return False
    
    # Initialize Go module
    if not Path("go.mod").exists():
        print("Initializing Go module...")
        run_command("go mod init flow2-go-engine")
    
    # Download dependencies
    print("Downloading Go dependencies...")
    run_command("go mod tidy")
    
    return True

def build_service():
    """Build the Go service"""
    print("🏗️  Building Go service...")
    run_command("go build -o bin/flow2-go-engine ./cmd/server")
    print("✅ Build completed")

def run_service():
    """Run the Go service"""
    print("🚀 Starting Flow 2 Go Engine...")
    print("⚠️  IMPORTANT: This service requires real dependencies:")
    print("   • Rust Recipe Engine at localhost:50051")
    print("   • Redis at localhost:6379")
    print("   • Context Service (when implemented)")
    print("   • Medication API (when implemented)")
    print()

    # Set environment variables for development
    env = os.environ.copy()
    env.update({
        "RUST_ENGINE_ADDRESS": "localhost:50051",  # Real Rust engine required
        "REDIS_URL": "redis://localhost:6379",     # Real Redis required
        "CONTEXT_SERVICE_URL": "http://localhost:8080",  # Will fail until implemented
        "MEDICATION_API_URL": "http://localhost:8009",   # Will fail until implemented
        "SERVER_PORT": "8080",
        "SERVER_ENVIRONMENT": "development",
        "LOG_LEVEL": "info",
    })
    
    try:
        subprocess.run(
            ["go", "run", "./cmd/server"],
            env=env,
            check=True
        )
    except KeyboardInterrupt:
        print("\n🛑 Service stopped by user")
    except subprocess.CalledProcessError as e:
        print(f"❌ Service failed: {e}")
        return False
    
    return True

def run_tests():
    """Run Go tests"""
    print("🧪 Running Go tests...")
    
    # Run unit tests
    result = run_command("go test ./...", check=False)
    
    if result.returncode == 0:
        print("✅ All tests passed")
        return True
    else:
        print("❌ Some tests failed")
        return False

def main():
    parser = argparse.ArgumentParser(description="Flow 2 Go Engine Runner")
    parser.add_argument("--dev", action="store_true", help="Development mode with auto-reload")
    parser.add_argument("--build", action="store_true", help="Build the service")
    parser.add_argument("--test", action="store_true", help="Run tests")
    
    args = parser.parse_args()
    
    print("🚀 Flow 2 Go Enhanced Orchestrator")
    print("=" * 50)
    
    # Setup Go environment
    if not setup_go_environment():
        sys.exit(1)
    
    # Run tests if requested
    if args.test:
        if not run_tests():
            sys.exit(1)
        return
    
    # Build if requested
    if args.build:
        build_service()
        return
    
    # Run the service
    if args.dev:
        print("🔄 Running in development mode...")
        # In development mode, we'll use 'go run' for auto-reload
        run_service()
    else:
        # Build and run
        build_service()
        print("🚀 Starting built service...")
        try:
            subprocess.run(["./bin/flow2-go-engine"], check=True)
        except KeyboardInterrupt:
            print("\n🛑 Service stopped by user")
        except FileNotFoundError:
            print("❌ Built binary not found. Run with --build first.")
            sys.exit(1)

if __name__ == "__main__":
    main()
