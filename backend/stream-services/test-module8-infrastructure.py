#!/usr/bin/env python3
"""
Module 8 Infrastructure Test Script

Tests connectivity and basic operations for all storage services:
- MongoDB
- Elasticsearch
- ClickHouse
- Redis

Run after starting infrastructure with:
    ./manage-module8-infrastructure.sh start
"""

import sys
import logging
from datetime import datetime
from typing import Dict
from module8_storage_clients import Module8Storage, StorageConfig

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class InfrastructureTester:
    """Test all Module 8 infrastructure services"""

    def __init__(self):
        self.storage = Module8Storage()
        self.results: Dict[str, Dict] = {}

    def run_all_tests(self) -> bool:
        """Run all infrastructure tests"""
        logger.info("=" * 60)
        logger.info("Module 8 Infrastructure Tests")
        logger.info("=" * 60)

        # Connect to all services
        try:
            logger.info("\n1. Connecting to all services...")
            self.storage.connect_all()
            logger.info("✓ All services connected")
        except Exception as e:
            logger.error(f"✗ Connection failed: {e}")
            return False

        # Run individual tests
        tests = [
            ("MongoDB", self.test_mongodb),
            ("Elasticsearch", self.test_elasticsearch),
            ("ClickHouse", self.test_clickhouse),
            ("Redis", self.test_redis)
        ]

        all_passed = True
        for name, test_func in tests:
            logger.info(f"\n{'=' * 60}")
            logger.info(f"Testing {name}")
            logger.info("=" * 60)

            try:
                result = test_func()
                self.results[name] = result
                if result['success']:
                    logger.info(f"✓ {name} tests passed")
                else:
                    logger.error(f"✗ {name} tests failed: {result.get('error')}")
                    all_passed = False
            except Exception as e:
                logger.error(f"✗ {name} test error: {e}")
                self.results[name] = {'success': False, 'error': str(e)}
                all_passed = False

        # Health check
        logger.info(f"\n{'=' * 60}")
        logger.info("Final Health Check")
        logger.info("=" * 60)
        health = self.storage.health_check_all()
        for service, status in health.items():
            symbol = "✓" if status else "✗"
            logger.info(f"{symbol} {service}: {'Healthy' if status else 'Unhealthy'}")

        # Close connections
        logger.info(f"\n{'=' * 60}")
        self.storage.close_all()
        logger.info("✓ All connections closed")

        # Summary
        logger.info(f"\n{'=' * 60}")
        logger.info("Test Summary")
        logger.info("=" * 60)
        for service, result in self.results.items():
            symbol = "✓" if result['success'] else "✗"
            logger.info(f"{symbol} {service}: {result.get('message', 'No message')}")

        logger.info(f"\n{'=' * 60}")
        if all_passed:
            logger.info("✓ ALL TESTS PASSED")
        else:
            logger.error("✗ SOME TESTS FAILED")
        logger.info("=" * 60)

        return all_passed

    def test_mongodb(self) -> Dict:
        """Test MongoDB operations"""
        try:
            # Test connection
            if not self.storage.mongo.health_check():
                return {'success': False, 'error': 'Health check failed'}

            # Test insert
            test_doc = {
                'test_id': 'test_mongo_1',
                'message': 'MongoDB test document',
                'timestamp': datetime.utcnow(),
                'data': {'value': 123, 'nested': {'field': 'test'}}
            }

            result = self.storage.mongo.clinical_events.insert_one(test_doc)
            logger.info(f"  Inserted document: {result.inserted_id}")

            # Test find
            found = self.storage.mongo.clinical_events.find_one({'test_id': 'test_mongo_1'})
            if not found:
                return {'success': False, 'error': 'Document not found after insert'}
            logger.info(f"  Found document: {found['test_id']}")

            # Test update
            self.storage.mongo.clinical_events.update_one(
                {'test_id': 'test_mongo_1'},
                {'$set': {'updated': True}}
            )
            logger.info("  Updated document")

            # Test count
            count = self.storage.mongo.clinical_events.count_documents({'test_id': 'test_mongo_1'})
            logger.info(f"  Document count: {count}")

            # Test delete
            delete_result = self.storage.mongo.clinical_events.delete_one({'test_id': 'test_mongo_1'})
            logger.info(f"  Deleted {delete_result.deleted_count} document(s)")

            return {
                'success': True,
                'message': f'CRUD operations successful (inserted ID: {result.inserted_id})'
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def test_elasticsearch(self) -> Dict:
        """Test Elasticsearch operations"""
        try:
            # Test connection
            if not self.storage.es.health_check():
                return {'success': False, 'error': 'Health check failed'}

            # Test index creation
            index_name = 'test_clinical_events'
            test_doc = {
                'event_id': 'test_es_1',
                'event_type': 'test',
                'message': 'Elasticsearch test event',
                'timestamp': datetime.utcnow().isoformat(),
                'data': {'value': 456}
            }

            # Index document
            result = self.storage.es.index(index=index_name, document=test_doc, doc_id='test_es_1')
            logger.info(f"  Indexed document: {result['_id']}")

            # Wait for indexing
            import time
            time.sleep(1)

            # Test get
            doc = self.storage.es.get(index=index_name, doc_id='test_es_1')
            logger.info(f"  Retrieved document: {doc['_source']['event_id']}")

            # Test search
            search_result = self.storage.es.search(
                index=index_name,
                query={'match': {'event_type': 'test'}}
            )
            hits = search_result['hits']['total']['value']
            logger.info(f"  Search found {hits} document(s)")

            # Test delete
            delete_result = self.storage.es.delete(index=index_name, doc_id='test_es_1')
            logger.info(f"  Deleted document: {delete_result['result']}")

            # Clean up index
            self.storage.es.client.indices.delete(index=index_name, ignore=[404])
            logger.info(f"  Cleaned up test index")

            return {
                'success': True,
                'message': f'CRUD operations successful (indexed: {result["_id"]})'
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def test_clickhouse(self) -> Dict:
        """Test ClickHouse operations"""
        try:
            # Test connection
            if not self.storage.clickhouse.health_check():
                return {'success': False, 'error': 'Health check failed'}

            # Test simple query
            result = self.storage.clickhouse.execute("SELECT 1 as test")
            logger.info(f"  Simple query result: {result}")

            # Test database query
            databases = self.storage.clickhouse.execute("SHOW DATABASES")
            logger.info(f"  Found {len(databases)} database(s)")

            # Test tables query
            tables = self.storage.clickhouse.execute(
                f"SHOW TABLES FROM {self.storage.config.clickhouse_database}"
            )
            logger.info(f"  Found {len(tables)} table(s)")

            # Test patient_events table
            count_query = "SELECT count() FROM patient_events"
            count_result = self.storage.clickhouse.execute(count_query)
            event_count = count_result[0][0] if count_result else 0
            logger.info(f"  Patient events count: {event_count}")

            # Test vital_signs table
            vital_count_query = "SELECT count() FROM vital_signs"
            vital_count_result = self.storage.clickhouse.execute(vital_count_query)
            vital_count = vital_count_result[0][0] if vital_count_result else 0
            logger.info(f"  Vital signs count: {vital_count}")

            # Test insert
            test_event_id = f"test_ch_{datetime.utcnow().timestamp()}"
            self.storage.clickhouse.insert_patient_event(
                event_id=test_event_id,
                patient_id='TEST_PATIENT',
                event_type='test',
                event_time=datetime.utcnow(),
                event_data={'test': True, 'value': 789}
            )
            logger.info(f"  Inserted test event: {test_event_id}")

            # Verify insert
            verify_query = f"SELECT count() FROM patient_events WHERE event_id = '{test_event_id}'"
            verify_result = self.storage.clickhouse.execute(verify_query)
            inserted_count = verify_result[0][0] if verify_result else 0
            logger.info(f"  Verified insert: {inserted_count} record(s)")

            # Clean up
            delete_query = f"ALTER TABLE patient_events DELETE WHERE event_id = '{test_event_id}'"
            self.storage.clickhouse.execute(delete_query)
            logger.info(f"  Cleaned up test data")

            return {
                'success': True,
                'message': f'Queries successful (tables: {len(tables)}, events: {event_count})'
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def test_redis(self) -> Dict:
        """Test Redis operations"""
        try:
            # Test connection
            if not self.storage.redis.health_check():
                return {'success': False, 'error': 'Health check failed'}

            # Test string operations
            test_key = 'test_redis_string'
            self.storage.redis.set(test_key, 'test_value', ex=60)
            logger.info(f"  Set key: {test_key}")

            value = self.storage.redis.get(test_key)
            logger.info(f"  Got value: {value}")

            exists = self.storage.redis.exists(test_key)
            logger.info(f"  Key exists: {exists}")

            # Test JSON operations
            test_json_key = 'test_redis_json'
            test_data = {'patient_id': 'P123', 'data': [1, 2, 3]}
            self.storage.redis.set(test_json_key, test_data, ex=60)
            logger.info(f"  Set JSON key: {test_json_key}")

            json_value = self.storage.redis.get(test_json_key, as_json=True)
            logger.info(f"  Got JSON value: {json_value}")

            # Test counter
            counter_key = 'test_counter'
            count = self.storage.redis.incr(counter_key)
            logger.info(f"  Incremented counter: {count}")

            # Test hash operations
            hash_name = 'test_hash'
            self.storage.redis.hset(hash_name, 'field1', 'value1')
            self.storage.redis.hset(hash_name, 'field2', {'nested': 'data'})
            logger.info(f"  Set hash fields: {hash_name}")

            hash_value = self.storage.redis.hget(hash_name, 'field1')
            logger.info(f"  Got hash field: {hash_value}")

            all_fields = self.storage.redis.hgetall(hash_name)
            logger.info(f"  Got all hash fields: {len(all_fields)} field(s)")

            # Test patient caching
            patient_data = {
                'patient_id': 'TEST_P123',
                'name': 'Test Patient',
                'vitals': {'hr': 75, 'bp': '120/80'}
            }
            self.storage.redis.cache_patient('TEST_P123', patient_data, ttl=60)
            logger.info(f"  Cached patient: TEST_P123")

            cached = self.storage.redis.get_cached_patient('TEST_P123')
            logger.info(f"  Retrieved cached patient: {cached['patient_id']}")

            # Clean up
            self.storage.redis.delete(
                test_key,
                test_json_key,
                counter_key,
                hash_name,
                'patient:TEST_P123'
            )
            logger.info(f"  Cleaned up test data")

            return {
                'success': True,
                'message': f'CRUD operations successful (string, JSON, hash, counter)'
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}


def main():
    """Main test execution"""
    tester = InfrastructureTester()

    try:
        success = tester.run_all_tests()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        logger.info("\nTest interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Test execution failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
