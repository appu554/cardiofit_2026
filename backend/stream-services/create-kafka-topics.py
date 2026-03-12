#!/usr/bin/env python3
"""
Simple Kafka Topics Creator
Creates all required topics for Stage 1 & Stage 2 testing
"""

import sys
import time
from confluent_kafka.admin import AdminClient, NewTopic
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

# Topics to create
TOPICS = [
    {'name': 'raw-device-data.v1', 'partitions': 12, 'description': 'Raw device data input'},
    {'name': 'validated-device-data.v1', 'partitions': 12, 'description': 'Stage 1 → Stage 2'},
    {'name': 'failed-validation.v1', 'partitions': 4, 'description': 'Stage 1 DLQ'},
    {'name': 'critical-data-dlq.v1', 'partitions': 4, 'description': 'Critical data DLQ'},
    {'name': 'poison-messages.v1', 'partitions': 2, 'description': 'Stage 1 poison messages'},
    {'name': 'sink-write-failures.v1', 'partitions': 6, 'description': 'Stage 2 DLQ'},
    {'name': 'critical-sink-failures.v1', 'partitions': 4, 'description': 'Critical sink failures'},
    {'name': 'poison-messages-stage2.v1', 'partitions': 2, 'description': 'Stage 2 poison messages'}
]

def create_topics():
    """Create all required Kafka topics"""
    print_info("Creating Kafka topics for Stage 1 & Stage 2...")
    
    try:
        # Create admin client
        admin_client = AdminClient(KAFKA_CONFIG)
        
        # Test connection first
        print_info("Testing connection...")
        metadata = admin_client.list_topics(timeout=10)
        print_status(f"Connected to Kafka cluster with {len(metadata.topics)} existing topics")
        
        # Prepare topics for creation
        new_topics = []
        for topic in TOPICS:
            new_topic = NewTopic(
                topic=topic['name'],
                num_partitions=topic['partitions'],
                replication_factor=3,
                config={
                    'cleanup.policy': 'delete',
                    'compression.type': 'snappy',
                    'min.insync.replicas': '2'
                }
            )
            new_topics.append(new_topic)
        
        # Create topics
        print_info("Creating topics...")
        futures = admin_client.create_topics(new_topics, request_timeout=30)
        
        # Check results
        success_count = 0
        for topic_name, future in futures.items():
            try:
                future.result()  # Wait for completion
                print_status(f"Created topic: {topic_name}")
                success_count += 1
            except KafkaException as e:
                if e.args[0].code() == 36:  # TopicExistsException
                    print_warning(f"Topic already exists: {topic_name}")
                    success_count += 1
                else:
                    print_error(f"Failed to create {topic_name}: {e}")
            except Exception as e:
                print_error(f"Error creating {topic_name}: {e}")
        
        print_info(f"Topic creation completed: {success_count}/{len(TOPICS)} topics ready")
        
        # List created topics
        print_info("Verifying created topics...")
        time.sleep(2)
        metadata = admin_client.list_topics(timeout=10)
        
        print_status("Available topics for testing:")
        for topic in TOPICS:
            if topic['name'] in metadata.topics:
                topic_meta = metadata.topics[topic['name']]
                print_status(f"  ✅ {topic['name']} ({len(topic_meta.partitions)} partitions) - {topic['description']}")
            else:
                print_error(f"  ❌ {topic['name']} - Not found")
        
        return True
        
    except Exception as e:
        print_error(f"Failed to create topics: {e}")
        return False

def main():
    """Main function"""
    print("🚀 Kafka Topics Creator for Stage 1 & Stage 2")
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
        print_error("confluent-kafka package not found")
        print_info("Installing confluent-kafka...")
        try:
            import subprocess
            result = subprocess.run([sys.executable, '-m', 'pip', 'install', 'confluent-kafka'], 
                                  capture_output=True, text=True)
            if result.returncode == 0:
                print_status("confluent-kafka installed successfully")
            else:
                print_error("Failed to install confluent-kafka")
                print("Please run: pip install confluent-kafka")
                sys.exit(1)
        except Exception as e:
            print_error(f"Installation failed: {e}")
            sys.exit(1)
    
    print()
    print("📋 Topics to create:")
    for topic in TOPICS:
        print(f"   • {topic['name']} ({topic['partitions']} partitions) - {topic['description']}")
    
    print()
    choice = input("Create all topics? (y/n): ").strip().lower()
    
    if choice in ['y', 'yes']:
        if create_topics():
            print()
            print_status("🎉 Kafka setup completed successfully!")
            print()
            print("Next steps:")
            print("1. Terminal 1: python run-stage1.py")
            print("2. Terminal 2: python run-stage2.py")
            print("3. Terminal 3: python run-tests.py")
        else:
            print_error("Kafka setup failed!")
    else:
        print_info("Kafka setup cancelled")

if __name__ == "__main__":
    main()
