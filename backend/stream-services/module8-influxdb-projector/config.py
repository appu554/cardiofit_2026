"""Configuration for InfluxDB Projector Service."""
import os
from typing import List
from dotenv import load_dotenv

load_dotenv()


class Config:
    """InfluxDB Projector Configuration."""

    # Service Configuration
    SERVICE_NAME: str = os.getenv("SERVICE_NAME", "influxdb-projector")
    SERVICE_PORT: int = int(os.getenv("SERVICE_PORT", "8054"))
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")

    # InfluxDB Configuration
    INFLUXDB_URL: str = os.getenv("INFLUXDB_URL", "http://localhost:8086")
    INFLUXDB_ORG: str = os.getenv("INFLUXDB_ORG", "cardiofit")
    INFLUXDB_TOKEN: str = os.getenv("INFLUXDB_TOKEN", "")

    # Bucket Names
    INFLUXDB_BUCKET_REALTIME: str = os.getenv("INFLUXDB_BUCKET_REALTIME", "vitals_realtime")
    INFLUXDB_BUCKET_1MIN: str = os.getenv("INFLUXDB_BUCKET_1MIN", "vitals_1min")
    INFLUXDB_BUCKET_1HOUR: str = os.getenv("INFLUXDB_BUCKET_1HOUR", "vitals_1hour")

    # Retention Policies
    RETENTION_REALTIME: int = 7 * 24 * 3600  # 7 days in seconds
    RETENTION_1MIN: int = 90 * 24 * 3600  # 90 days in seconds
    RETENTION_1HOUR: int = 2 * 365 * 24 * 3600  # 2 years in seconds

    # Batch Settings
    INFLUXDB_BATCH_SIZE: int = int(os.getenv("INFLUXDB_BATCH_SIZE", "200"))
    INFLUXDB_FLUSH_INTERVAL: int = int(os.getenv("INFLUXDB_FLUSH_INTERVAL", "5000"))

    # Kafka Configuration
    KAFKA_BOOTSTRAP_SERVERS: str = os.getenv(
        "KAFKA_BOOTSTRAP_SERVERS",
        "pkc-9q8rv.ap-south-2.aws.confluent.cloud:9092"
    )
    KAFKA_SASL_USERNAME: str = os.getenv("KAFKA_SASL_USERNAME", "")
    KAFKA_SASL_PASSWORD: str = os.getenv("KAFKA_SASL_PASSWORD", "")
    KAFKA_CONSUMER_GROUP: str = os.getenv("KAFKA_CONSUMER_GROUP", "influxdb-projector-group")
    KAFKA_TOPIC: str = os.getenv("KAFKA_TOPIC", "prod.ehr.events.enriched")

    # Kafka Consumer Settings
    KAFKA_AUTO_OFFSET_RESET: str = "latest"
    KAFKA_MAX_POLL_RECORDS: int = 100
    KAFKA_SESSION_TIMEOUT_MS: int = 30000
    KAFKA_HEARTBEAT_INTERVAL_MS: int = 10000

    @classmethod
    def validate(cls) -> None:
        """Validate required configuration."""
        required_vars = [
            ("INFLUXDB_TOKEN", cls.INFLUXDB_TOKEN),
            ("KAFKA_SASL_USERNAME", cls.KAFKA_SASL_USERNAME),
            ("KAFKA_SASL_PASSWORD", cls.KAFKA_SASL_PASSWORD),
        ]

        missing = [name for name, value in required_vars if not value]
        if missing:
            raise ValueError(f"Missing required environment variables: {', '.join(missing)}")


config = Config()
