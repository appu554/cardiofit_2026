#!/usr/bin/env python3
"""Test if FHIR Store Projector can start"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'module8-shared'))

print("=" * 60)
print("FHIR Store Projector - Startup Test")
print("=" * 60)

# Test 1: Import config
print("\n1. Testing config import...")
try:
    from app.config import Config
    print(f"   ✓ Config loaded")
    print(f"   - Kafka: {Config.KAFKA_BOOTSTRAP_SERVERS}")
    print(f"   - Topic: {Config.KAFKA_TOPIC_FHIR_UPSERT}")
    print(f"   - Port: {Config.SERVICE_PORT}")
except Exception as e:
    print(f"   ✗ Config import failed: {e}")
    sys.exit(1)

# Test 2: Import FHIR handler
print("\n2. Testing FHIR Store Handler import...")
try:
    from app.services.fhir_store_handler import FHIRStoreHandler
    print(f"   ✓ FHIR Handler loaded")
except Exception as e:
    print(f"   ✗ FHIR Handler import failed: {e}")
    sys.exit(1)

# Test 3: Import projector
print("\n3. Testing Projector import...")
try:
    from app.services.projector import FHIRStoreProjector
    print(f"   ✓ Projector loaded")
except Exception as e:
    print(f"   ✗ Projector import failed: {e}")
    sys.exit(1)

# Test 4: Import FastAPI app
print("\n4. Testing FastAPI app import...")
try:
    from app.main import app
    print(f"   ✓ FastAPI app loaded")
except Exception as e:
    print(f"   ✗ FastAPI app import failed: {e}")
    sys.exit(1)

# Test 5: Build configuration
print("\n5. Testing configuration build...")
try:
    config = {
        'kafka': Config.get_kafka_config(),
        'topics': {
            'fhir_upsert': Config.KAFKA_TOPIC_FHIR_UPSERT,
            'dlq': Config.KAFKA_TOPIC_DLQ,
        },
        'batch_size': Config.BATCH_SIZE,
        'batch_timeout_seconds': Config.BATCH_TIMEOUT_SECONDS,
        'fhir_store': {
            'project_id': Config.GOOGLE_CLOUD_PROJECT_ID,
            'location': Config.GOOGLE_CLOUD_LOCATION,
            'dataset_id': Config.GOOGLE_CLOUD_DATASET_ID,
            'store_id': Config.GOOGLE_CLOUD_FHIR_STORE_ID,
            'credentials_path': Config.GOOGLE_APPLICATION_CREDENTIALS,
            'max_retries': Config.RETRY_MAX_ATTEMPTS,
            'retry_backoff_factor': Config.RETRY_BACKOFF_FACTOR,
        },
    }
    print(f"   ✓ Configuration built")
    print(f"   - Batch size: {config['batch_size']}")
    print(f"   - FHIR store: {Config.get_fhir_store_path()}")
except Exception as e:
    print(f"   ✗ Configuration build failed: {e}")
    sys.exit(1)

# Test 6: Initialize projector (without starting Kafka consumer)
print("\n6. Testing Projector initialization...")
try:
    projector = FHIRStoreProjector(config)
    print(f"   ✓ Projector initialized")
    print(f"   - Handler stats: {projector.handler.get_stats()}")
except Exception as e:
    print(f"   ✗ Projector initialization failed: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

print("\n" + "=" * 60)
print("✓ All startup tests passed!")
print("=" * 60)
print("\nService is ready to start. Run:")
print("  ./start-fhir-store-projector.sh")
print("\nor:")
print("  python3 run.py")
