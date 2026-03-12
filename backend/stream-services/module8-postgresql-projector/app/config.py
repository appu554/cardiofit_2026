"""
Configuration for PostgreSQL Projector Service
"""
import os
from typing import Dict, Any

# =====================================================
# Kafka Configuration
# =====================================================
KAFKA_CONFIG: Dict[str, Any] = {
    "bootstrap.servers": os.getenv(
        "KAFKA_BOOTSTRAP_SERVERS",
        "pkc-p11w6.us-east-1.aws.confluent.cloud:9092"
    ),
    "group.id": "module8-postgresql-projector-v3",
    "auto.offset.reset": "earliest",
    "enable.auto.commit": False,
    "max.poll.records": 500,
    "max.poll.interval.ms": 300000,
    "session.timeout.ms": 45000,
}

# Add security configuration only if using Confluent Cloud
security_protocol = os.getenv("KAFKA_SECURITY_PROTOCOL", "SASL_SSL")
if security_protocol == "SASL_SSL":
    KAFKA_CONFIG.update({
        "security.protocol": "SASL_SSL",
        "sasl.mechanism": "PLAIN",
        "sasl.username": os.getenv("KAFKA_API_KEY"),
        "sasl.password": os.getenv("KAFKA_API_SECRET"),
    })
elif security_protocol == "PLAINTEXT":
    KAFKA_CONFIG["security.protocol"] = "PLAINTEXT"

# Topics
TOPICS = ["prod.ehr.events.enriched"]

# =====================================================
# Batch Configuration
# =====================================================
BATCH_SIZE = int(os.getenv("BATCH_SIZE", "100"))
BATCH_TIMEOUT_SECONDS = float(os.getenv("BATCH_TIMEOUT_SECONDS", "5.0"))

# =====================================================
# DLQ Configuration
# =====================================================
DLQ_TOPIC = "prod.ehr.dlq.postgresql"

# =====================================================
# PostgreSQL Configuration
# =====================================================
POSTGRES_CONFIG = {
    "host": os.getenv("POSTGRES_HOST", "172.21.0.4"),  # Docker container IP
    "port": int(os.getenv("POSTGRES_PORT", "5432")),
    "database": os.getenv("POSTGRES_DB", "cardiofit_analytics"),
    "user": os.getenv("POSTGRES_USER", "cardiofit"),
    "password": os.getenv("POSTGRES_PASSWORD", "cardiofit_analytics_pass"),
}

# PostgreSQL schema
POSTGRES_SCHEMA = os.getenv("POSTGRES_SCHEMA", "module8_projections")

# Connection pool settings
POSTGRES_POOL_MIN_CONN = int(os.getenv("POSTGRES_POOL_MIN_CONN", "2"))
POSTGRES_POOL_MAX_CONN = int(os.getenv("POSTGRES_POOL_MAX_CONN", "10"))

# =====================================================
# Service Configuration
# =====================================================
SERVICE_PORT = int(os.getenv("SERVICE_PORT", "8050"))
SERVICE_HOST = os.getenv("SERVICE_HOST", "0.0.0.0")

# Metrics
METRICS_PORT = int(os.getenv("METRICS_PORT", "9090"))

# Logging
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")
