"""
Setup script for Kafka integration
"""

import subprocess
import sys
import os
import logging
from pathlib import Path

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def install_dependencies():
    """Install required Kafka dependencies"""
    logger.info("Installing Kafka dependencies...")
    
    # Get the directory of this script
    script_dir = Path(__file__).parent
    requirements_file = script_dir / "requirements.txt"
    
    try:
        # Install dependencies
        subprocess.check_call([
            sys.executable, "-m", "pip", "install", "-r", str(requirements_file)
        ])
        logger.info("✓ Successfully installed Kafka dependencies")
        return True
    except subprocess.CalledProcessError as e:
        logger.error(f"✗ Failed to install dependencies: {e}")
        return False

def create_topics():
    """Create required Kafka topics"""
    logger.info("Creating Kafka topics...")

    try:
        try:
            from confluent_kafka.admin import AdminClient, NewTopic
        except ImportError:
            logger.warning("AdminClient not available, skipping topic creation")
            return True

        from .config import kafka_config, TopicNames
        
        # Get admin client
        admin_config = kafka_config.get_producer_config()
        admin_client = AdminClient(admin_config)
        
        # Define topics to create
        topics_to_create = [
            # Raw data ingestion
            TopicNames.RAW_DEVICE_DATA,
            TopicNames.RAW_IMAGING_DATA,
            TopicNames.RAW_DOCUMENT_DATA,
            
            # FHIR events
            TopicNames.FHIR_PATIENT_EVENTS,
            TopicNames.FHIR_ENCOUNTER_EVENTS,
            TopicNames.FHIR_OBSERVATION_EVENTS,
            TopicNames.FHIR_MEDICATION_EVENTS,
            TopicNames.FHIR_ORDER_EVENTS,
            TopicNames.FHIR_CONDITION_EVENTS,
            
            # Processed events
            TopicNames.CLINICAL_ALERTS,
            TopicNames.WORKFLOW_EVENTS,
            TopicNames.NOTIFICATION_EVENTS,
            
            # Read model updates
            TopicNames.READ_MODEL_UPDATES,
            TopicNames.SEARCH_INDEX_UPDATES,
            
            # Dead letter queues
            TopicNames.DLQ_PROCESSING_ERRORS,
            TopicNames.DLQ_VALIDATION_ERRORS,
        ]
        
        # Get existing topics
        metadata = admin_client.list_topics(timeout=10)
        existing_topics = set(metadata.topics.keys())
        
        # Create new topics
        new_topics = []
        for topic_name in topics_to_create:
            if topic_name not in existing_topics:
                new_topic = NewTopic(
                    topic=topic_name,
                    num_partitions=3,  # Default 3 partitions
                    replication_factor=3,  # Confluent Cloud default
                    config={
                        'cleanup.policy': 'delete',
                        'retention.ms': '604800000',  # 7 days
                        'compression.type': 'snappy'
                    }
                )
                new_topics.append(new_topic)
        
        if new_topics:
            # Create topics
            fs = admin_client.create_topics(new_topics)
            
            # Wait for creation
            for topic, f in fs.items():
                try:
                    f.result()  # The result itself is None
                    logger.info(f"✓ Created topic: {topic}")
                except Exception as e:
                    logger.error(f"✗ Failed to create topic {topic}: {e}")
        else:
            logger.info("✓ All required topics already exist")
        
        return True
        
    except Exception as e:
        logger.error(f"✗ Failed to create topics: {e}")
        return False

def test_connection():
    """Test Kafka connection"""
    logger.info("Testing Kafka connection...")
    
    try:
        # Import and run connection test
        from .test_connection import test_kafka_connection
        return test_kafka_connection()
    except Exception as e:
        logger.error(f"✗ Connection test failed: {e}")
        return False

def setup_monitoring():
    """Setup monitoring configuration"""
    logger.info("Setting up monitoring...")
    
    try:
        from .monitoring import get_kafka_monitor
        
        # Initialize monitor
        monitor = get_kafka_monitor()
        
        # Start monitoring (will run in background)
        monitor.start_monitoring(interval=60)  # Monitor every minute
        
        # Get initial health status
        health = monitor.get_health_status()
        logger.info(f"✓ Monitoring setup complete")
        logger.info(f"  Cluster status: {health['status']}")
        logger.info(f"  Broker count: {health.get('broker_count', 'unknown')}")
        
        return True
        
    except Exception as e:
        logger.error(f"✗ Failed to setup monitoring: {e}")
        return False

def main():
    """Main setup function"""
    logger.info("🚀 Starting Kafka setup for Clinical Synthesis Hub...")
    
    # Step 1: Install dependencies
    if not install_dependencies():
        logger.error("❌ Setup failed at dependency installation")
        return False
    
    # Step 2: Test connection
    if not test_connection():
        logger.error("❌ Setup failed at connection test")
        return False
    
    # Step 3: Create topics
    if not create_topics():
        logger.error("❌ Setup failed at topic creation")
        return False
    
    # Step 4: Setup monitoring
    if not setup_monitoring():
        logger.error("❌ Setup failed at monitoring setup")
        return False
    
    logger.info("🎉 Kafka setup completed successfully!")
    logger.info("")
    logger.info("Next steps:")
    logger.info("1. Start implementing event producers in your services")
    logger.info("2. Create event consumers/workers for data processing")
    logger.info("3. Monitor the system using the monitoring endpoints")
    logger.info("")
    logger.info("Available topics:")
    
    try:
        from .config import TopicNames
        for attr_name in dir(TopicNames):
            if not attr_name.startswith('_'):
                topic_name = getattr(TopicNames, attr_name)
                logger.info(f"  - {topic_name}")
    except:
        pass
    
    return True

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
