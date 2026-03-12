"""
Elasticsearch Projector Service - Main Entry Point
FastAPI service providing health checks and search endpoints
"""
import asyncio
import logging
import os
import threading
from contextlib import asynccontextmanager
from typing import Dict, Any, Optional

from fastapi import FastAPI, HTTPException, Query
from pydantic import BaseModel

from projector.elasticsearch_projector import ElasticsearchProjector

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global projector instance
projector: Optional[ElasticsearchProjector] = None
projector_thread: Optional[threading.Thread] = None


# Request/Response Models
class SearchRequest(BaseModel):
    query: str
    index: str = "clinical_events-*"
    size: int = 10


class SearchResponse(BaseModel):
    total: int
    hits: list
    took: int


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan context manager for startup and shutdown"""
    global projector, projector_thread

    # Startup
    logger.info("Starting Elasticsearch Projector Service")

    # Kafka configuration
    kafka_config = {
        "bootstrap.servers": os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
        "security.protocol": os.getenv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
        "group.id": "module8-elasticsearch-projector-v3",
        "auto.offset.reset": "earliest",
        "enable.auto.commit": True,
    }

    # Add SASL config if using authentication
    if os.getenv("KAFKA_SASL_MECHANISM"):
        kafka_config["sasl.mechanism"] = os.getenv("KAFKA_SASL_MECHANISM")
        kafka_config["sasl.username"] = os.getenv("KAFKA_SASL_USERNAME")
        kafka_config["sasl.password"] = os.getenv("KAFKA_SASL_PASSWORD")

    elasticsearch_url = os.getenv("ELASTICSEARCH_URL", "http://localhost:9200")

    # Initialize projector
    projector = ElasticsearchProjector(
        kafka_config=kafka_config,
        elasticsearch_url=elasticsearch_url,
        batch_size=int(os.getenv("BATCH_SIZE", "100")),
        flush_timeout=int(os.getenv("FLUSH_TIMEOUT", "5"))
    )

    # Initialize Elasticsearch indices synchronously
    logger.info("Initializing Elasticsearch indices...")
    try:
        # The initialize method is async but we need to run the ES setup
        # Since ES client is sync, we can call the sync parts directly
        from projector.index_templates import get_all_templates
        for template_name, template_body in get_all_templates().items():
            try:
                projector.es.indices.put_index_template(
                    name=template_name,
                    body=template_body
                )
                logger.info(f"Created index template: {template_name}")
            except Exception as e:
                logger.warning(f"Template {template_name} may already exist: {e}")

        # Create initial indices
        for index in ["patients", "clinical_events-2024", "clinical_documents-2024", "alerts-2024"]:
            if not projector.es.indices.exists(index=index):
                projector.es.indices.create(index=index)
                logger.info(f"Created index: {index}")
    except Exception as e:
        logger.error(f"Failed to initialize Elasticsearch: {e}")
        raise

    # Start consuming in background thread (synchronous method from base class)
    def run_projector():
        try:
            projector.start()
        except Exception as e:
            logger.error(f"Projector thread error: {e}")

    projector_thread = threading.Thread(target=run_projector, daemon=True)
    projector_thread.start()

    logger.info("Elasticsearch Projector Service started successfully")

    yield

    # Shutdown
    logger.info("Shutting down Elasticsearch Projector Service")
    if projector:
        projector.shutdown()


# Create FastAPI app
app = FastAPI(
    title="Elasticsearch Projector Service",
    description="Clinical event search and analytics with Elasticsearch",
    version="1.0.0",
    lifespan=lifespan
)


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "elasticsearch-projector",
        "version": "1.0.0",
        "status": "running"
    }


@app.get("/health")
async def health():
    """Health check endpoint"""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    # Get basic health info synchronously
    try:
        es_health = projector.es.cluster.health()
        index_stats = {}
        for index in ["patients", "clinical_events-2024", "clinical_documents-2024", "alerts-2024"]:
            if projector.es.indices.exists(index=index):
                count = projector.es.count(index=index)
                index_stats[index] = count.get("count", 0)

        return {
            "status": "healthy",
            "projector": projector.get_projector_name(),
            "elasticsearch": {
                "connected": True,
                "cluster_status": es_health.get("status"),
                "number_of_nodes": es_health.get("number_of_nodes")
            },
            "index_statistics": index_stats,
            "processing_statistics": projector.stats
        }
    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Health check failed: {str(e)}")


@app.get("/stats")
async def stats():
    """Get processing statistics"""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    return {
        "statistics": projector.stats,
        "elasticsearch_url": projector.es_url
    }


@app.post("/search", response_model=SearchResponse)
async def search(request: SearchRequest):
    """
    Full-text search across clinical events

    Example queries:
    - "high blood pressure"
    - "heart rate > 100"
    - "patient diabetes"
    """
    if not projector or not projector.es:
        raise HTTPException(status_code=503, detail="Elasticsearch not available")

    try:
        # Build Elasticsearch query
        es_query = {
            "query": {
                "query_string": {
                    "query": request.query,
                    "default_operator": "AND"
                }
            },
            "size": request.size,
            "sort": [{"timestamp": {"order": "desc"}}]
        }

        # Execute search
        result = projector.es.search(index=request.index, body=es_query)

        # Format response
        hits = []
        for hit in result.get("hits", {}).get("hits", []):
            hits.append({
                "id": hit.get("_id"),
                "score": hit.get("_score"),
                "source": hit.get("_source")
            })

        return SearchResponse(
            total=result.get("hits", {}).get("total", {}).get("value", 0),
            hits=hits,
            took=result.get("took", 0)
        )

    except Exception as e:
        logger.error(f"Search error: {e}")
        raise HTTPException(status_code=500, detail=f"Search failed: {str(e)}")


@app.get("/search/patient/{patient_id}")
async def search_patient_events(
    patient_id: str,
    limit: int = Query(default=50, le=1000)
):
    """Get all events for a specific patient"""
    if not projector or not projector.es:
        raise HTTPException(status_code=503, detail="Elasticsearch not available")

    try:
        query = {
            "query": {"term": {"patientId": patient_id}},
            "size": limit,
            "sort": [{"timestamp": {"order": "desc"}}]
        }

        result = projector.es.search(index="clinical_events-*", body=query)

        return {
            "patientId": patient_id,
            "totalEvents": result.get("hits", {}).get("total", {}).get("value", 0),
            "events": [hit["_source"] for hit in result.get("hits", {}).get("hits", [])]
        }

    except Exception as e:
        logger.error(f"Patient search error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/alerts/active")
async def get_active_alerts(
    severity: Optional[str] = Query(None, regex="^(LOW|MEDIUM|HIGH|CRITICAL)$"),
    limit: int = Query(default=100, le=1000)
):
    """Get active (unacknowledged) alerts"""
    if not projector or not projector.es:
        raise HTTPException(status_code=503, detail="Elasticsearch not available")

    try:
        # Build query for unacknowledged alerts
        must_filters = [{"term": {"acknowledged": False}}]

        if severity:
            must_filters.append({"term": {"severity": severity}})

        query = {
            "query": {"bool": {"must": must_filters}},
            "size": limit,
            "sort": [{"createdAt": {"order": "desc"}}]
        }

        result = projector.es.search(index="alerts-*", body=query)

        return {
            "totalActiveAlerts": result.get("hits", {}).get("total", {}).get("value", 0),
            "alerts": [hit["_source"] for hit in result.get("hits", {}).get("hits", [])]
        }

    except Exception as e:
        logger.error(f"Alerts search error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/aggregations/risk-distribution")
async def risk_distribution():
    """Get distribution of patients by risk level"""
    if not projector or not projector.es:
        raise HTTPException(status_code=503, detail="Elasticsearch not available")

    try:
        query = {
            "size": 0,
            "aggs": {
                "risk_levels": {
                    "terms": {"field": "currentState.currentRiskLevel"}
                }
            }
        }

        result = projector.es.search(index="patients", body=query)

        buckets = result.get("aggregations", {}).get("risk_levels", {}).get("buckets", [])

        return {
            "distribution": [
                {"riskLevel": bucket["key"], "count": bucket["doc_count"]}
                for bucket in buckets
            ]
        }

    except Exception as e:
        logger.error(f"Aggregation error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8052)
