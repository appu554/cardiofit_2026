#!/usr/bin/env python3
"""
Elasticsearch Projector Service Runner
Standalone execution with proper Python path configuration
"""
import sys
import os
from pathlib import Path

# Add src directory to Python path
src_dir = Path(__file__).parent / "src"
sys.path.insert(0, str(src_dir))

# Add shared module to path
shared_dir = Path(__file__).parent.parent / "module8-shared"
sys.path.insert(0, str(shared_dir))

if __name__ == "__main__":
    import uvicorn
    from main import app

    print("Starting Elasticsearch Projector Service on port 8052...")
    print(f"Python path: {sys.path[:3]}")

    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8052,
        log_level="info"
    )
