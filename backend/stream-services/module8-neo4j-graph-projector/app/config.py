"""
Configuration for Neo4j Graph Projector Service
Loads environment variables and Kafka/Neo4j configuration
"""
import os
from typing import List

# Service Configuration
SERVICE_HOST = os.getenv("SERVICE_HOST", "0.0.0.0")
SERVICE_PORT = int(os.getenv("SERVICE_PORT", "8057"))

# Kafka Topics
TOPICS: List[str] = [
    os.getenv("TOPIC_GRAPH_MUTATIONS", "prod.ehr.graph.mutations")
]

# Batch Configuration
BATCH_SIZE = int(os.getenv("BATCH_SIZE", "50"))
BATCH_TIMEOUT_SECONDS = float(os.getenv("BATCH_TIMEOUT_SECONDS", "5.0"))

# Kafka Configuration
# Support both local PLAINTEXT and Confluent Cloud SASL_SSL
bootstrap_servers = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
security_protocol = os.getenv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")

KAFKA_CONFIG = {
    "bootstrap.servers": bootstrap_servers,
    "group.id": os.getenv("KAFKA_CONSUMER_GROUP", "neo4j-graph-projector-group"),
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
}

# Add SASL authentication if using SASL_SSL
if security_protocol == "SASL_SSL":
    KAFKA_CONFIG.update({
        "security.protocol": "SASL_SSL",
        "sasl.mechanism": "PLAIN",
        "sasl.username": os.getenv("KAFKA_API_KEY", ""),
        "sasl.password": os.getenv("KAFKA_API_SECRET", ""),
    })
else:
    KAFKA_CONFIG["security.protocol"] = "PLAINTEXT"

# Neo4j Configuration (using localhost port mapping)
NEO4J_CONFIG = {
    "uri": os.getenv("NEO4J_URI", "bolt://localhost:7687"),
    "username": os.getenv("NEO4J_USERNAME", "neo4j"),
    "password": os.getenv("NEO4J_PASSWORD", "CardioFit2024!"),
    "database": os.getenv("NEO4J_DATABASE", "neo4j"),
}

# Dead Letter Queue Topic
DLQ_TOPIC = os.getenv("DLQ_TOPIC", "prod.ehr.dlq.neo4j")
