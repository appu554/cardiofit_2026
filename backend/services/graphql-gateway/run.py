"""
Run script for the GraphQL Gateway with reload support.
This script is used to start the server with proper reload support.
"""

import uvicorn
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Get environment
ENVIRONMENT = os.getenv("ENVIRONMENT", "development")

if __name__ == "__main__":
    # Configure uvicorn logging
    log_config = uvicorn.config.LOGGING_CONFIG
    log_config["formatters"]["access"]["fmt"] = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    log_config["formatters"]["default"]["fmt"] = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"

    # Run the server with reload support
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8006,
        log_config=log_config,
        log_level="info",
        reload=ENVIRONMENT != "production",
        reload_dirs=["./"]
    )
