"""
ClickHouse Projector Service - Main Entry Point
FastAPI service for OLAP analytics projection
"""

from fastapi import FastAPI
from contextlib import asynccontextmanager
import logging
import os
import yaml
import threading
from typing import Dict, Any

from projector import ClickHouseProjector

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global projector instance
projector: ClickHouseProjector = None
projector_thread: threading.Thread = None


def load_config() -> Dict[str, Any]:
    """Load configuration from YAML file or environment variables."""
    config_path = os.getenv('CONFIG_PATH', 'config.yaml')

    # Default configuration
    config = {
        'kafka': {
            'bootstrap_servers': os.getenv('KAFKA_BOOTSTRAP_SERVERS', 'localhost:9092'),
            'topic': 'prod.ehr.events.enriched',
            'group_id': 'module8-clickhouse-projector-v2',
            'auto_offset_reset': 'earliest',
            'enable_auto_commit': False,
            'max_poll_records': 500,
            'max_poll_interval_ms': 300000,
        },
        'clickhouse': {
            'host': os.getenv('CLICKHOUSE_HOST', 'clickhouse'),
            'port': int(os.getenv('CLICKHOUSE_PORT', '9000')),
            'database': os.getenv('CLICKHOUSE_DATABASE', 'module8_analytics'),
            'user': os.getenv('CLICKHOUSE_USER', 'module8_user'),
            'password': os.getenv('CLICKHOUSE_PASSWORD', 'module8_password'),
        },
        'batch': {
            'size': 500,  # Larger batches for analytics workloads
            'timeout': 30  # 30 seconds for analytics batching
        },
        'service': {
            'name': 'module8-clickhouse-projector',
            'version': '1.0.0',
            'port': 8053
        }
    }

    # Override with YAML config if exists
    if os.path.exists(config_path):
        try:
            with open(config_path, 'r') as f:
                yaml_config = yaml.safe_load(f)
                config.update(yaml_config)
            logger.info(f"Loaded configuration from {config_path}")
        except Exception as e:
            logger.warning(f"Failed to load YAML config: {e}, using defaults")

    return config


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage projector lifecycle."""
    global projector, projector_thread

    # Startup
    logger.info("Starting ClickHouse Projector Service...")
    config = load_config()

    projector = ClickHouseProjector(config)

    # Start consuming in background thread (synchronous method from base class)
    def run_projector():
        try:
            projector.start()
        except Exception as e:
            logger.error(f"Projector thread error: {e}")

    projector_thread = threading.Thread(target=run_projector, daemon=True)
    projector_thread.start()

    logger.info("ClickHouse Projector Service started successfully")

    yield

    # Shutdown
    logger.info("Shutting down ClickHouse Projector Service...")
    if projector:
        projector.close()


# Create FastAPI app
app = FastAPI(
    title="ClickHouse Projector Service",
    description="OLAP analytics projection for enriched clinical events",
    version="1.0.0",
    lifespan=lifespan
)


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "module8-clickhouse-projector",
        "version": "1.0.0"
    }


@app.get("/metrics")
async def get_metrics():
    """Get projector metrics."""
    if not projector:
        return {"error": "Projector not initialized"}

    metrics = projector.get_metrics()
    analytics_summary = projector.get_analytics_summary()

    return {
        "projector_metrics": metrics,
        "analytics_summary": analytics_summary
    }


@app.get("/analytics/summary")
async def get_analytics_summary():
    """Get detailed analytics summary from ClickHouse."""
    if not projector:
        return {"error": "Projector not initialized"}

    return projector.get_analytics_summary()


@app.post("/analytics/query")
async def execute_analytics_query(query: Dict[str, Any]):
    """
    Execute ad-hoc analytics query against ClickHouse.

    Example queries:
    - Daily patient event counts
    - Department-level risk distributions
    - ML prediction accuracy analysis
    """
    if not projector or not projector.client:
        return {"error": "Projector not initialized"}

    try:
        sql = query.get('sql')
        if not sql:
            return {"error": "SQL query required"}

        # Safety check: only allow SELECT queries
        if not sql.strip().upper().startswith('SELECT'):
            return {"error": "Only SELECT queries allowed"}

        result = projector.client.execute(sql)

        return {
            "query": sql,
            "rows": len(result),
            "data": result[:100]  # Limit to 100 rows for API response
        }
    except Exception as e:
        logger.error(f"Query execution error: {e}")
        return {"error": str(e)}


if __name__ == "__main__":
    import uvicorn

    config = load_config()
    port = config['service']['port']

    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=port,
        reload=False,
        log_level="info"
    )
