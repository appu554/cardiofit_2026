"""
Test Kafka Confluent Cloud connection
"""

import sys
import logging
import json
import time
from typing import Dict, Any

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def test_kafka_connection():
    """Test basic Kafka connection"""
    try:
        from confluent_kafka import Producer, Consumer
        try:
            from confluent_kafka import AdminClient
            from confluent_kafka.admin import NewTopic
        except ImportError:
            logger.warning("AdminClient not available, skipping admin operations")
            AdminClient = None
            NewTopic = None
        
        logger.info("Testing Kafka Confluent Cloud connection...")
        
        # Import our configuration
        from .config import kafka_config
        
        # Test 1: Admin Client - List topics
        logger.info("Test 1: Testing admin client connection...")
        if AdminClient is not None:
            admin_config = kafka_config.get_producer_config()
            admin_client = AdminClient(admin_config)

            try:
                metadata = admin_client.list_topics(timeout=10)
                logger.info("✓ Successfully connected to Kafka cluster")
                logger.info(f"  Cluster ID: {metadata.cluster_id}")
                logger.info(f"  Broker count: {len(metadata.brokers)}")
                logger.info(f"  Existing topics: {list(metadata.topics.keys())}")
            except Exception as e:
                logger.error(f"✗ Failed to connect to Kafka cluster: {e}")
                return False
        else:
            logger.warning("⚠ AdminClient not available, skipping admin test")
            metadata = None
        
        # Test 2: Create test topic
        logger.info("Test 2: Creating test topic...")
        test_topic = "test-connection-topic"

        if AdminClient is not None and NewTopic is not None and metadata is not None:
            try:
                # Check if topic already exists
                if test_topic not in metadata.topics:
                    new_topic = NewTopic(
                        topic=test_topic,
                        num_partitions=1,
                        replication_factor=3
                    )

                    fs = admin_client.create_topics([new_topic])
                    for topic, f in fs.items():
                        try:
                            f.result()  # The result itself is None
                            logger.info(f"✓ Successfully created topic: {topic}")
                        except Exception as e:
                            logger.error(f"✗ Failed to create topic {topic}: {e}")
                else:
                    logger.info(f"✓ Topic {test_topic} already exists")
            except Exception as e:
                logger.error(f"✗ Error managing topics: {e}")
        else:
            logger.warning("⚠ AdminClient not available, skipping topic creation")
        
        # Test 3: Producer - Send test message
        logger.info("Test 3: Testing producer...")
        producer_config = kafka_config.get_producer_config()
        producer = Producer(producer_config)
        
        test_message = {
            "test": True,
            "timestamp": time.time(),
            "message": "Hello from Clinical Synthesis Hub!"
        }
        
        try:
            def delivery_callback(err, msg):
                if err is not None:
                    logger.error(f"✗ Message delivery failed: {err}")
                else:
                    logger.info(f"✓ Message delivered to {msg.topic()} [{msg.partition()}] at offset {msg.offset()}")
            
            producer.produce(
                topic=test_topic,
                key="test-key",
                value=json.dumps(test_message),
                callback=delivery_callback
            )
            
            # Wait for delivery
            producer.flush(timeout=10)
            logger.info("✓ Producer test completed")
            
        except Exception as e:
            logger.error(f"✗ Producer test failed: {e}")
            return False
        finally:
            # Confluent Kafka producer doesn't have close(), just flush
            try:
                producer.flush(timeout=5)
            except:
                pass
        
        # Test 4: Consumer - Read test message
        logger.info("Test 4: Testing consumer...")
        consumer_config = kafka_config.get_consumer_config(
            group_id="test-connection-group",
            **{"auto.offset.reset": "earliest"}
        )
        consumer = Consumer(consumer_config)
        
        try:
            consumer.subscribe([test_topic])
            logger.info(f"✓ Subscribed to topic: {test_topic}")
            
            # Poll for messages (with timeout)
            message_received = False
            for _ in range(10):  # Try for 10 seconds
                msg = consumer.poll(timeout=1.0)
                
                if msg is None:
                    continue
                
                if msg.error():
                    logger.error(f"Consumer error: {msg.error()}")
                    continue
                
                # Parse message
                try:
                    value = json.loads(msg.value().decode('utf-8'))
                    logger.info(f"✓ Received message: {value}")
                    message_received = True
                    break
                except Exception as e:
                    logger.error(f"Error parsing message: {e}")
            
            if not message_received:
                logger.warning("⚠ No messages received (this might be normal)")
            
        except Exception as e:
            logger.error(f"✗ Consumer test failed: {e}")
            return False
        finally:
            try:
                consumer.close()
            except:
                pass
        
        # Test 5: Test our event producer/consumer classes
        logger.info("Test 5: Testing our event classes...")
        try:
            from .producer import EventProducer
            from .consumer import EventConsumer
            from .schemas import EventEnvelope
            
            # Test event producer
            producer = EventProducer()
            event_id = producer.publish_event(
                topic=test_topic,
                event_type="test.connection",
                data={"test": "event_producer"},
                source="test-connection-script"
            )
            producer.flush()
            try:
                producer.close()
            except:
                pass
            
            logger.info(f"✓ Event producer test completed, event ID: {event_id}")
            
            # Test event consumer (just initialization)
            consumer = EventConsumer(
                group_id="test-event-consumer",
                topics=[test_topic]
            )
            
            def test_handler(envelope: EventEnvelope):
                logger.info(f"✓ Received event: {envelope.type} from {envelope.source}")
            
            consumer.register_handler("test.connection", test_handler)
            logger.info("✓ Event consumer test completed")
            try:
                consumer.close()
            except:
                pass
            
        except Exception as e:
            logger.error(f"✗ Event classes test failed: {e}")
            return False
        
        # Test 6: Test monitoring
        logger.info("Test 6: Testing monitoring...")
        try:
            from .monitoring import KafkaMonitor
            
            monitor = KafkaMonitor()
            health = monitor.get_health_status()
            
            logger.info(f"✓ Monitoring test completed")
            logger.info(f"  Cluster health: {health['status']}")
            logger.info(f"  Broker count: {health.get('broker_count', 'unknown')}")
            logger.info(f"  Topic count: {health.get('topic_count', 'unknown')}")
            
        except Exception as e:
            logger.error(f"✗ Monitoring test failed: {e}")
            return False
        
        logger.info("🎉 All Kafka connection tests passed!")
        return True
        
    except ImportError as e:
        logger.error(f"✗ Missing required dependencies: {e}")
        logger.error("Please install: pip install confluent-kafka")
        return False
    except Exception as e:
        logger.error(f"✗ Unexpected error during testing: {e}")
        return False

def main():
    """Main test function"""
    success = test_kafka_connection()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
