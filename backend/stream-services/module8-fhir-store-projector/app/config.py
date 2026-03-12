"""Configuration for FHIR Store Projector"""

import os
from typing import Dict, Any
from dotenv import load_dotenv

load_dotenv()


class Config:
    """FHIR Store Projector configuration"""

    # Kafka Configuration
    KAFKA_BOOTSTRAP_SERVERS = os.getenv(
        'KAFKA_BOOTSTRAP_SERVERS',
        'pkc-your-cluster.us-east-1.aws.confluent.cloud:9092'
    )
    KAFKA_SECURITY_PROTOCOL = os.getenv('KAFKA_SECURITY_PROTOCOL', 'SASL_SSL')
    KAFKA_SASL_MECHANISM = os.getenv('KAFKA_SASL_MECHANISM', 'PLAIN')
    KAFKA_SASL_USERNAME = os.getenv('KAFKA_SASL_USERNAME', '')
    KAFKA_SASL_PASSWORD = os.getenv('KAFKA_SASL_PASSWORD', '')

    # Consumer Settings
    KAFKA_GROUP_ID = os.getenv('KAFKA_GROUP_ID', 'module8-fhir-store-projector')
    KAFKA_AUTO_OFFSET_RESET = os.getenv('KAFKA_AUTO_OFFSET_RESET', 'earliest')
    KAFKA_ENABLE_AUTO_COMMIT = os.getenv('KAFKA_ENABLE_AUTO_COMMIT', 'false').lower() == 'true'

    # Topics
    KAFKA_TOPIC_FHIR_UPSERT = os.getenv('KAFKA_TOPIC_FHIR_UPSERT', 'prod.ehr.fhir.upsert')
    KAFKA_TOPIC_DLQ = os.getenv('KAFKA_TOPIC_DLQ', 'prod.ehr.dlq.fhir-store-projector')

    # Batch Settings
    BATCH_SIZE = int(os.getenv('BATCH_SIZE', '20'))  # Small batch for API
    BATCH_TIMEOUT_SECONDS = float(os.getenv('BATCH_TIMEOUT_SECONDS', '10'))

    # Google Cloud Healthcare API
    GOOGLE_CLOUD_PROJECT_ID = os.getenv('GOOGLE_CLOUD_PROJECT_ID', 'cardiofit-905a8')
    GOOGLE_CLOUD_LOCATION = os.getenv('GOOGLE_CLOUD_LOCATION', 'us-central1')
    GOOGLE_CLOUD_DATASET_ID = os.getenv('GOOGLE_CLOUD_DATASET_ID', 'cardiofit_fhir_dataset')
    GOOGLE_CLOUD_FHIR_STORE_ID = os.getenv('GOOGLE_CLOUD_FHIR_STORE_ID', 'cardiofit_fhir_store')
    GOOGLE_APPLICATION_CREDENTIALS = os.getenv(
        'GOOGLE_APPLICATION_CREDENTIALS',
        'credentials/google-credentials.json'
    )

    # API Rate Limiting
    MAX_REQUESTS_PER_SECOND = int(os.getenv('MAX_REQUESTS_PER_SECOND', '200'))
    RETRY_MAX_ATTEMPTS = int(os.getenv('RETRY_MAX_ATTEMPTS', '3'))
    RETRY_BACKOFF_FACTOR = float(os.getenv('RETRY_BACKOFF_FACTOR', '2'))

    # Service Configuration
    SERVICE_PORT = int(os.getenv('SERVICE_PORT', '8056'))
    LOG_LEVEL = os.getenv('LOG_LEVEL', 'INFO')

    # Health Check
    HEALTH_CHECK_ENABLED = os.getenv('HEALTH_CHECK_ENABLED', 'true').lower() == 'true'
    HEALTH_CHECK_INTERVAL_SECONDS = int(os.getenv('HEALTH_CHECK_INTERVAL_SECONDS', '30'))

    @classmethod
    def get_kafka_config(cls) -> Dict[str, Any]:
        """Get Kafka consumer configuration"""
        return {
            'bootstrap_servers': cls.KAFKA_BOOTSTRAP_SERVERS.split(','),
            'group_id': cls.KAFKA_GROUP_ID,
            'auto_offset_reset': cls.KAFKA_AUTO_OFFSET_RESET,
            'enable_auto_commit': cls.KAFKA_ENABLE_AUTO_COMMIT,
            'security_protocol': cls.KAFKA_SECURITY_PROTOCOL,
            'sasl_mechanism': cls.KAFKA_SASL_MECHANISM,
            'sasl_plain_username': cls.KAFKA_SASL_USERNAME,
            'sasl_plain_password': cls.KAFKA_SASL_PASSWORD,
            'max_poll_records': cls.BATCH_SIZE,
            'session_timeout_ms': 30000,
            'heartbeat_interval_ms': 10000,
        }

    @classmethod
    def get_fhir_store_path(cls) -> str:
        """Get full FHIR store path"""
        return (
            f"projects/{cls.GOOGLE_CLOUD_PROJECT_ID}/"
            f"locations/{cls.GOOGLE_CLOUD_LOCATION}/"
            f"datasets/{cls.GOOGLE_CLOUD_DATASET_ID}/"
            f"fhirStores/{cls.GOOGLE_CLOUD_FHIR_STORE_ID}"
        )
