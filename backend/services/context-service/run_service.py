#!/usr/bin/env python3
"""
Run script for Clinical Context Service.
This script adds the backend directory to the Python path and starts the service.
"""

import sys
import os
import subprocess
import logging

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, backend_dir)

# Set environment variables
os.environ["PYTHONPATH"] = backend_dir
os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"

# Context Service specific environment variables
os.environ["REDIS_URL"] = os.environ.get("REDIS_URL", "redis://localhost:6379")
os.environ["KAFKA_BOOTSTRAP_SERVERS"] = os.environ.get("KAFKA_BOOTSTRAP_SERVERS", "pkc-619z3.us-east1.gcp.confluent.cloud:9092")
os.environ["KAFKA_API_KEY"] = os.environ.get("KAFKA_API_KEY", "LGJ3AQ2L6VRPW4S2")
os.environ["KAFKA_API_SECRET"] = os.environ.get("KAFKA_API_SECRET", "2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl")

# Data source service URLs
os.environ["PATIENT_SERVICE_URL"] = os.environ.get("PATIENT_SERVICE_URL", "http://localhost:8003")
os.environ["MEDICATION_SERVICE_URL"] = os.environ.get("MEDICATION_SERVICE_URL", "http://localhost:8009")
os.environ["LAB_SERVICE_URL"] = os.environ.get("LAB_SERVICE_URL", "http://localhost:8000")
os.environ["CONDITION_SERVICE_URL"] = os.environ.get("CONDITION_SERVICE_URL", "http://localhost:8010")
os.environ["ENCOUNTER_SERVICE_URL"] = os.environ.get("ENCOUNTER_SERVICE_URL", "http://localhost:8020")
os.environ["OBSERVATION_SERVICE_URL"] = os.environ.get("OBSERVATION_SERVICE_URL", "http://localhost:8007")
os.environ["CAE_SERVICE_URL"] = os.environ.get("CAE_SERVICE_URL", "http://localhost:8027")

# FHIR Store configuration
os.environ["FHIR_STORE_PATH"] = os.environ.get("FHIR_STORE_PATH", "projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store")

# Print configuration
print("=" * 80)
print("🚀 Starting Clinical Context Service")
print("   Implementing the Three Pillars of Excellence:")
print("   1. Federated GraphQL API (The 'Unified Data Graph')")
print("   2. Clinical Context Recipe System (The 'Governance Engine')")
print("   3. Multi-Layer Intelligent Cache (The 'Performance Accelerator')")
print("=" * 80)
print("Configuration:")
print(f"  Python Path: {sys.path[0]}")
print(f"  PYTHONPATH: {os.environ['PYTHONPATH']}")
print(f"  AUTH_SERVICE_URL: {os.environ['AUTH_SERVICE_URL']}")
print(f"  REDIS_URL: {os.environ['REDIS_URL']}")
print(f"  KAFKA_BOOTSTRAP_SERVERS: {os.environ['KAFKA_BOOTSTRAP_SERVERS']}")
print("")
print("Data Source Connections:")
print(f"  Patient Service: {os.environ['PATIENT_SERVICE_URL']}")
print(f"  Medication Service: {os.environ['MEDICATION_SERVICE_URL']}")
print(f"  Lab Service: {os.environ['LAB_SERVICE_URL']}")
print(f"  Condition Service: {os.environ['CONDITION_SERVICE_URL']}")
print(f"  Encounter Service: {os.environ['ENCOUNTER_SERVICE_URL']}")
print(f"  Observation Service: {os.environ['OBSERVATION_SERVICE_URL']}")
print(f"  CAE Service: {os.environ['CAE_SERVICE_URL']}")
print(f"  FHIR Store: {os.environ['FHIR_STORE_PATH']}")
print("")
print("Service Endpoints:")
print(f"  HTTP API: http://localhost:8016")
print(f"  GraphQL API: http://localhost:8016/graphql")
print(f"  Health Check: http://localhost:8016/health")
print(f"  Service Status: http://localhost:8016/status")
print(f"  Metrics: http://localhost:8016/metrics")
print("=" * 80)

# Run the service using uvicorn with specific Python path
python_path = r"C:\Users\apoor\AppData\Local\Microsoft\WindowsApps\PythonSoftwareFoundation.Python.3.12_qbz5n2kfra8p0\python.exe"
cmd = [python_path, "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8016", "--reload"]
subprocess.run(cmd, env=os.environ)  # Pass the environment variables to the subprocess
