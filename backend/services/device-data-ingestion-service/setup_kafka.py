#!/usr/bin/env python3
"""
Kafka setup script for Device Data Ingestion Service
Creates the required Kafka topic and registers Avro schema
"""
import os
import sys
import json
import logging
from typing import Dict, Any

# Add shared modules to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'shared'))

from confluent_kafka.admin import AdminClient, NewTopic
from confluent_kafka import KafkaError

from app.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Device Data Avro Schema
DEVICE_DATA_AVRO_SCHEMA = {
    "type": "record",
    "name": "DeviceDataEvent",
    "namespace": "com.clinicalsynthesishub.devicedata",
    "doc": "Schema for device data events from medical devices and wearables",
    "fields": [
        {
            "name": "data",
            "type": {
                "type": "record",
                "name": "DeviceReading",
                "fields": [
                    {
                        "name": "device_id",
                        "type": "string",
                        "doc": "Unique device identifier"
                    },
                    {
                        "name": "timestamp",
                        "type": "long",
                        "doc": "Unix timestamp of the reading"
                    },
                    {
                        "name": "reading_type",
                        "type": {
                            "type": "enum",
                            "name": "ReadingType",
                            "symbols": [
                                "heart_rate",
                                "blood_pressure_systolic", 
                                "blood_pressure_diastolic",
                                "blood_glucose",
                                "temperature",
                                "oxygen_saturation",
                                "weight",
                                "steps",
                                "sleep_duration",
                                "respiratory_rate"
                            ]
                        },
                        "doc": "Type of device reading"
                    },
                    {
                        "name": "value",
                        "type": "double",
                        "doc": "Numeric value of the reading"
                    },
                    {
                        "name": "unit",
                        "type": "string",
                        "doc": "Unit of measurement"
                    },
                    {
                        "name": "patient_id",
                        "type": ["null", "string"],
                        "default": None,
                        "doc": "Associated patient ID if known"
                    },
                    {
                        "name": "metadata",
                        "type": ["null", "string"],
                        "default": None,
                        "doc": "Additional metadata as JSON string"
                    },
                    {
                        "name": "vendor_info",
                        "type": {
                            "type": "record",
                            "name": "VendorInfo",
                            "fields": [
                                {
                                    "name": "vendor_id",
                                    "type": "string",
                                    "doc": "Vendor identifier"
                                },
                                {
                                    "name": "vendor_name",
                                    "type": "string",
                                    "doc": "Vendor display name"
                                }
                            ]
                        },
                        "doc": "Information about the device vendor"
                    }
                ]
            },
            "doc": "Device reading data"
        },
        {
            "name": "metadata",
            "type": {
                "type": "record",
                "name": "IngestionMetadata",
                "fields": [
                    {
                        "name": "ingestion_timestamp",
                        "type": "string",
                        "doc": "ISO timestamp when data was ingested"
                    },
                    {
                        "name": "service",
                        "type": "string",
                        "doc": "Name of the ingestion service"
                    },
                    {
                        "name": "version",
                        "type": "string",
                        "doc": "Version of the ingestion service"
                    }
                ]
            },
            "doc": "Ingestion metadata"
        }
    ]
}


def create_kafka_topic() -> bool:
    """Create the raw device data Kafka topic"""
    logger.info("Creating Kafka topic for device data...")
    
    try:
        # Configure admin client
        admin_config = {
            'bootstrap.servers': settings.KAFKA_BOOTSTRAP_SERVERS,
            'security.protocol': 'SASL_SSL',
            'sasl.mechanism': 'PLAIN',
            'sasl.username': settings.KAFKA_API_KEY,
            'sasl.password': settings.KAFKA_API_SECRET,
        }
        
        admin_client = AdminClient(admin_config)
        
        # Check if topic already exists
        metadata = admin_client.list_topics(timeout=10)
        if settings.KAFKA_TOPIC_DEVICE_DATA in metadata.topics:
            logger.info(f"✓ Topic {settings.KAFKA_TOPIC_DEVICE_DATA} already exists")
            return True
        
        # Create new topic
        new_topic = NewTopic(
            topic=settings.KAFKA_TOPIC_DEVICE_DATA,
            num_partitions=6,  # More partitions for device data volume
            replication_factor=3,  # Confluent Cloud default
            config={
                'cleanup.policy': 'delete',
                'retention.ms': '2592000000',  # 30 days retention
                'compression.type': 'snappy',
                'max.message.bytes': '1048576',  # 1MB max message size
                'min.insync.replicas': '2'  # Ensure durability
            }
        )
        
        # Create topic
        fs = admin_client.create_topics([new_topic])
        
        # Wait for creation
        for topic, f in fs.items():
            try:
                f.result()  # The result itself is None
                logger.info(f"✓ Successfully created topic: {topic}")
                return True
            except Exception as e:
                logger.error(f"✗ Failed to create topic {topic}: {e}")
                return False
                
    except Exception as e:
        logger.error(f"✗ Failed to create Kafka topic: {e}")
        return False


def save_avro_schema() -> bool:
    """Save the Avro schema to file for reference"""
    logger.info("Saving Avro schema...")
    
    try:
        schema_dir = os.path.join(os.path.dirname(__file__), 'schemas')
        os.makedirs(schema_dir, exist_ok=True)
        
        schema_file = os.path.join(schema_dir, 'device_data_event.avsc')
        
        with open(schema_file, 'w') as f:
            json.dump(DEVICE_DATA_AVRO_SCHEMA, f, indent=2)
        
        logger.info(f"✓ Avro schema saved to: {schema_file}")
        return True
        
    except Exception as e:
        logger.error(f"✗ Failed to save Avro schema: {e}")
        return False


def test_kafka_connection() -> bool:
    """Test Kafka connection"""
    logger.info("Testing Kafka connection...")
    
    try:
        from confluent_kafka import Producer
        
        # Configure producer for testing
        producer_config = {
            'bootstrap.servers': settings.KAFKA_BOOTSTRAP_SERVERS,
            'security.protocol': 'SASL_SSL',
            'sasl.mechanism': 'PLAIN',
            'sasl.username': settings.KAFKA_API_KEY,
            'sasl.password': settings.KAFKA_API_SECRET,
            'client.id': 'device-data-ingestion-test'
        }
        
        producer = Producer(producer_config)
        
        # Get cluster metadata
        metadata = producer.list_topics(timeout=10)
        
        logger.info(f"✓ Connected to Kafka cluster with {len(metadata.topics)} topics")
        return True
        
    except Exception as e:
        logger.error(f"✗ Kafka connection test failed: {e}")
        return False


def main():
    """Main setup function"""
    logger.info("🚀 Setting up Kafka for Device Data Ingestion Service...")
    
    # Step 1: Test connection
    if not test_kafka_connection():
        logger.error("❌ Setup failed: Cannot connect to Kafka")
        return False
    
    # Step 2: Create topic
    if not create_kafka_topic():
        logger.error("❌ Setup failed: Cannot create Kafka topic")
        return False
    
    # Step 3: Save schema
    if not save_avro_schema():
        logger.error("❌ Setup failed: Cannot save Avro schema")
        return False
    
    logger.info("🎉 Kafka setup completed successfully!")
    logger.info("")
    logger.info("Configuration:")
    logger.info(f"  Topic: {settings.KAFKA_TOPIC_DEVICE_DATA}")
    logger.info(f"  Bootstrap Servers: {settings.KAFKA_BOOTSTRAP_SERVERS}")
    logger.info(f"  API Key: {settings.KAFKA_API_KEY}")
    logger.info("")
    logger.info("Next steps:")
    logger.info("1. Start the Device Data Ingestion Service")
    logger.info("2. Test the ingestion endpoints")
    logger.info("3. Monitor the Kafka topic for messages")
    
    return True


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
