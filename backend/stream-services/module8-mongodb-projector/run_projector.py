#!/usr/bin/env python3
"""Run script for MongoDB Projector service."""

import sys
import os
from pathlib import Path

# Add parent directory to path for module8-shared import
parent_dir = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(parent_dir))

# Add current directory to path
current_dir = Path(__file__).resolve().parent
sys.path.insert(0, str(current_dir))


def main():
    """Run the MongoDB Projector service."""
    import uvicorn
    from app.config import get_settings

    settings = get_settings()

    print("=" * 60)
    print("Starting MongoDB Projector Service")
    print("=" * 60)
    print(f"Service: {settings.service_name}")
    print(f"Port: {settings.service_port}")
    print(f"Kafka Topic: {settings.kafka_topic}")
    print(f"Kafka Group: {settings.kafka_group_id}")
    print(f"MongoDB URI: {settings.mongodb_uri}")
    print(f"MongoDB Database: {settings.mongodb_database}")
    print(f"Batch Size: {settings.batch_size}")
    print(f"Batch Timeout: {settings.batch_timeout_seconds}s")
    print("=" * 60)

    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=settings.service_port,
        reload=False,
        log_level="info",
    )


if __name__ == "__main__":
    main()
