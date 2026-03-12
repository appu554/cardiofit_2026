#!/usr/bin/env python3
"""Standalone runner for InfluxDB Projector Service."""
import sys
import os

# Add parent directory to path for module8-shared
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

if __name__ == "__main__":
    from main import app
    import uvicorn
    from config import config

    uvicorn.run(
        app,
        host="0.0.0.0",
        port=config.SERVICE_PORT,
        log_level=config.LOG_LEVEL.lower()
    )
