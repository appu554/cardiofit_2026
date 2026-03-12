#!/usr/bin/env python3
"""
Python Script to Setup Kafka Topics for Stage 1 & Stage 2
Windows-compatible Kafka topic creation using confluent-kafka-python
"""

import sys
import time
from confluent_kafka.admin import AdminClient, NewTopic, ConfigResource, ResourceType
from confluent_kafka import KafkaException

def print_status(message):
    print(f"✅ {message}")

def print_error(message):
    print(f"❌ {message}")

def print_info(message):
    print(f"ℹ️  {message}")

def print_warning(message):
    print(f"⚠️  {message}")

# Your Kafka configuration
KAFKA_CONFIG = {
    'bootstrap.servers': 'pkc-619z3.us-east1.gcp.confluent.cloud:9092',
    'security.protocol': 'SASL_SSL',
    'sasl.mechanism': 'PLAIN',
    'sasl.username': 'LGJ3AQ2L6VRPW4S2',
    'sasl.password': '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl'
}

# Topic configurations
TOPICS_CONFIG = [
    # Input topic for raw device data
    {
        'name': 'raw-device-data.v1',
        'partitions': 12,
        'replication_factor': 3,
        'config': {
            'retention.ms': '259200000',  # 3 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Raw device data input topic'
    },
    
    # Stage 1 output topic
    {
        'name': 'validated-device-data.v1',
        'partitions': 12,
        'replication_factor': 3,
        'config': {
            'retention.ms': '259200000',  # 3 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Validated and enriched device data (Stage 1 → Stage 2)'
    },
    
    # Stage 1 DLQ topics
    {
        'name': 'failed-validation.v1',
        'partitions': 4,
        'replication_factor': 3,
        'config': {
            'retention.ms': '2592000000',  # 30 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Validation failures and parsing errors (Stage 1 DLQ)'
    },
    
    {
        'name': 'critical-data-dlq.v1',
        'partitions': 4,
        'replication_factor': 3,
        'config': {
            'retention.ms': '7776000000',  # 90 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Critical medical data that failed processing (Stage 1 DLQ)'
    },
    
    {
        'name': 'poison-messages.v1',
        'partitions': 2,
        'replication_factor': 3,
        'config': {
            'retention.ms': '31536000000',  # 365 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Messages that repeatedly fail processing (Stage 1 DLQ)'
    },
    
    # Stage 2 DLQ topics
    {
        'name': 'sink-write-failures.v1',
        'partitions': 6,
        'replication_factor': 3,
        'config': {
            'retention.ms': '1209600000',  # 14 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Failed writes to FHIR Store, Elasticsearch, MongoDB (Stage 2 DLQ)'
    },
    
    {
        'name': 'critical-sink-failures.v1',
        'partitions': 4,
        'replication_factor': 3,
        'config': {
            'retention.ms': '7776000000',  # 90 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Critical medical data sink write failures (Stage 2 DLQ)'
    },
    
    {
        'name': 'poison-messages-stage2.v1',
        'partitions': 2,
        'replication_factor': 3,
        'config': {
            'retention.ms': '31536000000',  # 365 days
            'cleanup.policy': 'delete',
            'compression.type': 'snappy',
            'min.insync.replicas': '2'
        },
        'description': 'Messages that repeatedly fail sink writes (Stage 2 DLQ)'
    }
]

def check_kafka_connection():
    """Test Kafka connection"""
    print_info("Testing Kafka connection...")

    try:
        admin_client = AdminClient(KAFKA_CONFIG)

        # Get cluster metadata to test connection
        metadata = admin_client.list_topics(timeout=10)

        print_status(f"Connected to Kafka cluster with {len(metadata.topics)} existing topics")
        return admin_client

    except KafkaException as e:
        print_error(f"Failed to connect to Kafka: {e}")
        return None
    except Exception as e:
        print_error(f"Unexpected error connecting to Kafka: {e}")
        return None

def create_topics(admin_client):
    """Create Kafka topics"""
    print_info("Creating Kafka topics...")
    
    # Prepare topics for creation
    new_topics = []
    for topic_config in TOPICS_CONFIG:
        new_topic = NewTopic(
            topic=topic_config['name'],
            num_partitions=topic_config['partitions'],
            replication_factor=topic_config['replication_factor'],
            config=topic_config['config']
        )
        new_topics.append(new_topic)
    
    # Create topics
    try:
        futures = admin_client.create_topics(new_topics, request_timeout=30)
        
        # Wait for topic creation
        for topic_name, future in futures.items():
            try:
                future.result()  # The result itself is None
                print_status(f"Topic '{topic_name}' created successfully")
            except KafkaException as e:
                if e.args[0].code() == 36:  # TopicExistsException
                    print_warning(f"Topic '{topic_name}' already exists")
                else:
                    print_error(f"Failed to create topic '{topic_name}': {e}")
            except Exception as e:
                print_error(f"Unexpected error creating topic '{topic_name}': {e}")
    
    except Exception as e:
        print_error(f"Failed to create topics: {e}")

def list_topics(admin_client):
    """List existing topics"""
    print_info("Listing existing topics...")
    
    try:
        metadata = admin_client.list_topics(timeout=10)
        
        print_info("Existing topics:")
        for topic_name in sorted(metadata.topics.keys()):
            if any(topic_name.startswith(prefix) for prefix in ['raw-device-data', 'validated-device-data', 'failed-validation', 'critical-data-dlq', 'poison-messages', 'sink-write-failures', 'critical-sink-failures']):
                topic_metadata = metadata.topics[topic_name]
                print_status(f"  {topic_name} (partitions: {len(topic_metadata.partitions)})")
        
    except Exception as e:
        print_error(f"Failed to list topics: {e}")

def describe_topics(admin_client):
    """Describe topic configurations"""
    print_info("Describing topic configurations...")
    
    topic_names = [config['name'] for config in TOPICS_CONFIG]
    
    try:
        # Get topic configurations
        resources = [ConfigResource(ResourceType.TOPIC, topic_name) for topic_name in topic_names]
        configs = admin_client.describe_configs(resources, request_timeout=10)
        
        for resource, config_future in configs.items():
            try:
                config = config_future.result()
                print_status(f"Topic: {resource.name}")
                
                # Show key configurations
                key_configs = ['retention.ms', 'cleanup.policy', 'compression.type', 'min.insync.replicas']
                for key in key_configs:
                    if key in config:
                        print(f"    {key}: {config[key].value}")
                print()
                
            except Exception as e:
                print_warning(f"Could not describe topic {resource.name}: {e}")
    
    except Exception as e:
        print_error(f"Failed to describe topics: {e}")

def install_dependencies():
    """Install required Python packages"""
    print_info("Installing required dependencies...")
    
    try:
        import subprocess
        result = subprocess.run([sys.executable, '-m', 'pip', 'install', 'confluent-kafka'], 
                              capture_output=True, text=True)
        
        if result.returncode == 0:
            print_status("confluent-kafka package installed successfully")
            return True
        else:
            print_error("Failed to install confluent-kafka package:")
            print(result.stderr)
            return False
    except Exception as e:
        print_error(f"Failed to install dependencies: {e}")
        return False

def main():
    """Main function"""
    print("🚀 Kafka Topics Setup for Stage 1 & Stage 2")
    print("=" * 50)
    print("🎯 Your Kafka Configuration:")
    print(f"   Bootstrap Server: {KAFKA_CONFIG['bootstrap.servers']}")
    print(f"   API Key: {KAFKA_CONFIG['sasl.username']}")
    print("=" * 50)
    
    # Check if confluent-kafka is installed
    try:
        import confluent_kafka
        print_status("confluent-kafka package is available")
    except ImportError:
        print_warning("confluent-kafka package not found")
        if input("Install confluent-kafka package? (y/n): ").lower().startswith('y'):
            if not install_dependencies():
                print_error("Failed to install dependencies. Exiting.")
                sys.exit(1)
        else:
            print_error("confluent-kafka package is required. Exiting.")
            sys.exit(1)
    
    # Menu
    while True:
        print("\nSelect an option:")
        print("1. Test Kafka connection")
        print("2. Create all topics")
        print("3. List existing topics")
        print("4. Describe topic configurations")
        print("5. Full setup (1-2)")
        print("0. Exit")
        
        choice = input("\nEnter your choice: ").strip()
        
        if choice == '1':
            admin_client = check_kafka_connection()
            if admin_client:
                print_status("Kafka connection successful!")

        elif choice == '2':
            admin_client = check_kafka_connection()
            if admin_client:
                create_topics(admin_client)

        elif choice == '3':
            admin_client = check_kafka_connection()
            if admin_client:
                list_topics(admin_client)

        elif choice == '4':
            admin_client = check_kafka_connection()
            if admin_client:
                describe_topics(admin_client)

        elif choice == '5':
            print_info("Running full Kafka setup...")
            admin_client = check_kafka_connection()
            if admin_client:
                print_status("Connection successful! Creating topics...")
                create_topics(admin_client)
                print_info("Waiting for topics to be ready...")
                time.sleep(5)
                list_topics(admin_client)
                print_status("Kafka setup completed!")
            else:
                print_error("Kafka setup failed - connection issue!")
                
        elif choice == '0':
            print_info("Goodbye!")
            break
            
        else:
            print_error("Invalid choice. Please try again.")

if __name__ == "__main__":
    main()
