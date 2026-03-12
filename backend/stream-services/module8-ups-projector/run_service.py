#!/usr/bin/env python3
"""
Run UPS Projector Service

Configures Python path and starts the service.
"""

import sys
from pathlib import Path

# Add src directory to Python path
src_dir = Path(__file__).parent / "src"
sys.path.insert(0, str(src_dir))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8055,
        reload=True,
        log_level="info"
    )
