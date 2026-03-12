#!/usr/bin/env python3
"""
Unified Kafka Topic Management Script for CardioFit Platform
Manages all Kafka topics based on centralized YAML configuration
Supports creation, validation, and monitoring of topics
"""

import sys
import time
import yaml
import json
import argparse
from typing import Dict, List, Any, Optional
from pathlib import Path
from datetime import datetime
from confluent_kafka.admin import AdminClient, NewTopic, ConfigResource, ResourceType
from confluent_kafka import KafkaException
import logging

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class KafkaTopicManager:
    """Manages Kafka topics based on YAML configuration"""

    def __init__(self, config_file: str, environment: str = "development"):
        self.config_file = Path(config_file)
        self.environment = environment
        self.config = self._load_configuration()
        self.admin_client = None
        self.kafka_config = self._get_kafka_config()

    def _load_configuration(self) -> Dict[str, Any]:
        """Load topic configuration from YAML file"""
        try:
            with open(self.config_file, 'r') as f:
                config = yaml.safe_load(f)
                logger.info(f"✅ Loaded configuration from {self.config_file}")
                return config
        except Exception as e:
            logger.error(f"❌ Failed to load configuration: {e}")
            sys.exit(1)

    def _get_kafka_config(self) -> Dict[str, Any]:
        """Get Kafka connection configuration based on environment"""
        # Check if using Confluent Cloud or local
        if self.environment == "production" or self.environment == "staging":
            cluster_config = self.config['clusters']['confluent_cloud']
            return {
                'bootstrap.servers': cluster_config['bootstrap_servers'],
                'security.protocol': cluster_config['security_protocol'],
                'sasl.mechanism': cluster_config['sasl_mechanism'],
                'sasl.username': 'LGJ3AQ2L6VRPW4S2',  # Should be from environment variable
                'sasl.password': '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl'  # Should be from environment variable
            }
        else:
            cluster_config = self.config['clusters']['local_development']
            return {
                'bootstrap.servers': cluster_config['bootstrap_servers'],
                'security.protocol': cluster_config['security_protocol']
            }

    def connect(self) -> bool:
        """Establish connection to Kafka cluster"""
        try:
            self.admin_client = AdminClient(self.kafka_config)
            # Test connection by listing topics
            metadata = self.admin_client.list_topics(timeout=10)
            logger.info(f"✅ Connected to Kafka cluster with {len(metadata.topics)} existing topics")
            logger.info(f"🎯 Environment: {self.environment}")
            return True
        except Exception as e:
            logger.error(f"❌ Failed to connect to Kafka: {e}")
            return False

    def get_all_topic_configs(self) -> List[Dict[str, Any]]:
        """Get all topic configurations with environment overrides applied"""
        all_topics = []
        env_config = self.config['environments'].get(self.environment, {})
        defaults = self.config.get('defaults', {})

        for category, topics in self.config['topics'].items():
            for topic in topics:
                # Apply defaults
                topic_config = {**defaults, **topic}

                # Apply environment overrides
                if 'partitions' not in topic and 'default_partitions' in env_config:
                    topic_config['partitions'] = env_config['default_partitions']
                if 'replication_factor' not in topic and 'default_replication_factor' in env_config:
                    topic_config['replication_factor'] = env_config['default_replication_factor']
                if 'min_insync_replicas' not in topic and 'min_insync_replicas' in env_config:
                    topic_config['min_insync_replicas'] = env_config['min_insync_replicas']

                topic_config['category'] = category
                all_topics.append(topic_config)

        return all_topics

    def create_topics(self, category_filter: Optional[str] = None, dry_run: bool = False) -> Dict[str, Any]:
        """Create Kafka topics based on configuration"""
        results = {
            'created': [],
            'already_exists': [],
            'failed': []
        }

        all_topics = self.get_all_topic_configs()

        # Filter by category if specified
        if category_filter:
            all_topics = [t for t in all_topics if t['category'] == category_filter]
            logger.info(f"📁 Filtered to category: {category_filter}")

        logger.info(f"📋 Processing {len(all_topics)} topics...")

        if dry_run:
            logger.info("🔍 DRY RUN MODE - No topics will be created")
            for topic_config in all_topics:
                print(f"Would create: {topic_config['name']} with {topic_config.get('partitions', 1)} partitions")
            return results

        # Prepare NewTopic objects
        new_topics = []
        for topic_config in all_topics:
            # Extract Kafka configs
            kafka_configs = {}
            if 'retention_ms' in topic_config:
                kafka_configs['retention.ms'] = str(topic_config['retention_ms'])
            if 'cleanup_policy' in topic_config:
                kafka_configs['cleanup.policy'] = topic_config['cleanup_policy']
            if 'compression_type' in topic_config:
                kafka_configs['compression.type'] = topic_config['compression_type']
            if 'min_insync_replicas' in topic_config:
                kafka_configs['min.insync.replicas'] = str(topic_config['min_insync_replicas'])
            if 'segment_bytes' in topic_config:
                kafka_configs['segment.bytes'] = str(topic_config['segment_bytes'])
            if 'segment_ms' in topic_config:
                kafka_configs['segment.ms'] = str(topic_config['segment_ms'])
            if 'max_message_bytes' in topic_config:
                kafka_configs['max.message.bytes'] = str(topic_config['max_message_bytes'])

            new_topic = NewTopic(
                topic=topic_config['name'],
                num_partitions=topic_config.get('partitions', 1),
                replication_factor=topic_config.get('replication_factor', 1),
                config=kafka_configs
            )
            new_topics.append((new_topic, topic_config))

        # Create topics
        for new_topic, config in new_topics:
            try:
                futures = self.admin_client.create_topics([new_topic], request_timeout=30)
                for topic_name, future in futures.items():
                    try:
                        future.result()
                        logger.info(f"✅ Created topic: {topic_name}")
                        results['created'].append(topic_name)
                    except KafkaException as e:
                        if e.args[0].code() == 36:  # TopicExistsException
                            logger.debug(f"⚠️ Topic already exists: {topic_name}")
                            results['already_exists'].append(topic_name)
                        else:
                            logger.error(f"❌ Failed to create topic {topic_name}: {e}")
                            results['failed'].append(topic_name)
            except Exception as e:
                logger.error(f"❌ Unexpected error creating topic {new_topic.topic}: {e}")
                results['failed'].append(new_topic.topic)

        return results

    def validate_topics(self) -> Dict[str, Any]:
        """Validate existing topics against configuration"""
        validation_results = {
            'compliant': [],
            'missing': [],
            'misconfigured': []
        }

        all_topics = self.get_all_topic_configs()
        metadata = self.admin_client.list_topics(timeout=10)
        existing_topics = metadata.topics

        for topic_config in all_topics:
            topic_name = topic_config['name']

            if topic_name not in existing_topics:
                validation_results['missing'].append(topic_name)
                logger.warning(f"⚠️ Missing topic: {topic_name}")
            else:
                # Check partition count
                topic_metadata = existing_topics[topic_name]
                actual_partitions = len(topic_metadata.partitions)
                expected_partitions = topic_config.get('partitions', 1)

                if actual_partitions != expected_partitions:
                    validation_results['misconfigured'].append({
                        'topic': topic_name,
                        'issue': f'Partition count mismatch: expected {expected_partitions}, got {actual_partitions}'
                    })
                    logger.warning(f"⚠️ Misconfigured topic {topic_name}: partition count mismatch")
                else:
                    validation_results['compliant'].append(topic_name)

        return validation_results

    def list_topics_by_category(self) -> Dict[str, List[str]]:
        """List all topics organized by category"""
        topics_by_category = {}

        for category in self.config['topics'].keys():
            topics = self.config['topics'][category]
            topics_by_category[category] = [t['name'] for t in topics]

        return topics_by_category

    def describe_topic(self, topic_name: str) -> Optional[Dict[str, Any]]:
        """Get detailed information about a specific topic"""
        # Find topic in configuration
        for category, topics in self.config['topics'].items():
            for topic in topics:
                if topic['name'] == topic_name:
                    # Get actual topic metadata
                    metadata = self.admin_client.list_topics(timeout=10)
                    if topic_name in metadata.topics:
                        topic_metadata = metadata.topics[topic_name]
                        return {
                            'name': topic_name,
                            'category': category,
                            'description': topic.get('description', ''),
                            'configured_partitions': topic.get('partitions', 1),
                            'actual_partitions': len(topic_metadata.partitions),
                            'replication_factor': topic.get('replication_factor', 1),
                            'retention_ms': topic.get('retention_ms'),
                            'producers': topic.get('producers', []),
                            'consumers': topic.get('consumers', [])
                        }
                    else:
                        return {
                            'name': topic_name,
                            'category': category,
                            'description': topic.get('description', ''),
                            'status': 'NOT CREATED',
                            'configured_partitions': topic.get('partitions', 1),
                            'producers': topic.get('producers', []),
                            'consumers': topic.get('consumers', [])
                        }
        return None

    def generate_documentation(self, output_file: str = "kafka-topics-documentation.md"):
        """Generate markdown documentation for all topics"""
        with open(output_file, 'w') as f:
            f.write("# Kafka Topics Documentation\n\n")
            f.write(f"Generated: {datetime.now().isoformat()}\n")
            f.write(f"Environment: {self.environment}\n\n")

            for category, topics in self.config['topics'].items():
                f.write(f"## {category.replace('_', ' ').title()}\n\n")

                for topic in topics:
                    f.write(f"### `{topic['name']}`\n\n")
                    f.write(f"**Description:** {topic.get('description', 'N/A')}\n\n")
                    f.write(f"**Configuration:**\n")
                    f.write(f"- Partitions: {topic.get('partitions', 'default')}\n")
                    f.write(f"- Retention: {topic.get('retention_ms', 'default')}ms\n")
                    f.write(f"- Compression: {topic.get('compression_type', 'default')}\n\n")

                    if 'producers' in topic:
                        f.write(f"**Producers:** {', '.join(topic['producers'])}\n\n")
                    if 'consumers' in topic:
                        f.write(f"**Consumers:** {', '.join(topic['consumers'])}\n\n")

                    f.write("---\n\n")

        logger.info(f"📝 Documentation generated: {output_file}")

    def delete_topics(self, topic_names: List[str], confirm: bool = False):
        """Delete specified topics (use with caution!)"""
        if not confirm:
            logger.error("❌ Topic deletion requires explicit confirmation (--confirm flag)")
            return

        try:
            futures = self.admin_client.delete_topics(topic_names, request_timeout=30)
            for topic, future in futures.items():
                try:
                    future.result()
                    logger.info(f"✅ Deleted topic: {topic}")
                except Exception as e:
                    logger.error(f"❌ Failed to delete topic {topic}: {e}")
        except Exception as e:
            logger.error(f"❌ Failed to delete topics: {e}")


def main():
    parser = argparse.ArgumentParser(description='Kafka Topic Management for CardioFit Platform')
    parser.add_argument('--config', '-c',
                       default='../config/topics-config.yaml',
                       help='Path to topics configuration YAML file')
    parser.add_argument('--environment', '-e',
                       choices=['development', 'staging', 'production'],
                       default='development',
                       help='Environment to use')
    parser.add_argument('--action', '-a',
                       choices=['create', 'validate', 'list', 'describe', 'document', 'delete'],
                       required=True,
                       help='Action to perform')
    parser.add_argument('--category',
                       help='Filter by topic category')
    parser.add_argument('--topic',
                       help='Specific topic name (for describe action)')
    parser.add_argument('--dry-run',
                       action='store_true',
                       help='Perform dry run without making changes')
    parser.add_argument('--confirm',
                       action='store_true',
                       help='Confirm destructive operations')

    args = parser.parse_args()

    # Resolve config path
    config_path = Path(__file__).parent / args.config
    if not config_path.exists():
        logger.error(f"❌ Configuration file not found: {config_path}")
        sys.exit(1)

    # Initialize manager
    manager = KafkaTopicManager(str(config_path), args.environment)

    # Connect to Kafka
    if not manager.connect():
        logger.error("❌ Failed to connect to Kafka cluster")
        sys.exit(1)

    # Perform requested action
    if args.action == 'create':
        results = manager.create_topics(args.category, args.dry_run)
        print("\n📊 Summary:")
        print(f"  Created: {len(results['created'])}")
        print(f"  Already exists: {len(results['already_exists'])}")
        print(f"  Failed: {len(results['failed'])}")

    elif args.action == 'validate':
        results = manager.validate_topics()
        print("\n📊 Validation Summary:")
        print(f"  Compliant: {len(results['compliant'])}")
        print(f"  Missing: {len(results['missing'])}")
        print(f"  Misconfigured: {len(results['misconfigured'])}")

        if results['missing']:
            print("\n⚠️ Missing topics:")
            for topic in results['missing']:
                print(f"  - {topic}")

        if results['misconfigured']:
            print("\n⚠️ Misconfigured topics:")
            for item in results['misconfigured']:
                print(f"  - {item['topic']}: {item['issue']}")

    elif args.action == 'list':
        topics_by_category = manager.list_topics_by_category()
        print("\n📋 Topics by Category:")
        for category, topics in topics_by_category.items():
            print(f"\n{category.replace('_', ' ').title()}:")
            for topic in topics:
                print(f"  - {topic}")

    elif args.action == 'describe':
        if not args.topic:
            logger.error("❌ Topic name required for describe action")
            sys.exit(1)

        info = manager.describe_topic(args.topic)
        if info:
            print(f"\n📋 Topic: {info['name']}")
            print(f"  Category: {info['category']}")
            print(f"  Description: {info.get('description', 'N/A')}")
            print(f"  Status: {info.get('status', 'CREATED')}")
            print(f"  Partitions: {info.get('actual_partitions', info.get('configured_partitions'))}")
            print(f"  Producers: {', '.join(info.get('producers', []))}")
            print(f"  Consumers: {', '.join(info.get('consumers', []))}")
        else:
            logger.error(f"❌ Topic not found in configuration: {args.topic}")

    elif args.action == 'document':
        manager.generate_documentation()

    elif args.action == 'delete':
        if not args.topic:
            logger.error("❌ Topic name required for delete action")
            sys.exit(1)
        manager.delete_topics([args.topic], args.confirm)


if __name__ == "__main__":
    main()