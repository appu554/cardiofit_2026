#!/usr/bin/env python3
"""
PostgreSQL Projector Startup Script
Sets up environment and starts the projector service
"""
import sys
import os
from pathlib import Path

# Change to projector directory first
projector_dir = Path(__file__).parent / "module8-postgresql-projector"
os.chdir(projector_dir)

# Add module8-shared and projector directory to Python path
shared_path = Path(__file__).parent / "module8-shared"
sys.path.insert(0, str(shared_path))
sys.path.insert(0, str(projector_dir))  # Add projector directory for app module imports

# Set environment variables
os.environ.setdefault("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
os.environ.setdefault("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")
os.environ.setdefault("POSTGRES_HOST", "localhost")
os.environ.setdefault("POSTGRES_PORT", "5433")
os.environ.setdefault("POSTGRES_DB", "cardiofit")
os.environ.setdefault("POSTGRES_USER", "cardiofit")
os.environ.setdefault("POSTGRES_PASSWORD", "cardiofit_analytics_pass")
os.environ.setdefault("POSTGRES_SCHEMA", "module8_projections")
os.environ.setdefault("BATCH_SIZE", "100")
os.environ.setdefault("BATCH_TIMEOUT_SECONDS", "5.0")
os.environ.setdefault("SERVICE_PORT", "8050")
os.environ.setdefault("LOG_LEVEL", "INFO")

print(f"✅ Python path: {sys.path[0]}")
print(f"✅ Working directory: {os.getcwd()}")
print(f"✅ Kafka: {os.environ['KAFKA_BOOTSTRAP_SERVERS']}")
print(f"✅ PostgreSQL: {os.environ['POSTGRES_HOST']}:{os.environ['POSTGRES_PORT']}/{os.environ['POSTGRES_DB']}")
print(f"✅ Starting PostgreSQL Projector on port {os.environ['SERVICE_PORT']}...")

# Import and run the main module
import app.main
