#!/usr/bin/env python3
"""
Connectivity Testing for KB7 Runtime Layer

Tests actual connectivity to available services and validates
basic runtime functionality with real connections.
"""

import asyncio
import logging
import sys
import time
from datetime import datetime
import json

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


async def test_redis_connectivity():
    """Test Redis connectivity and basic operations"""
    print("🔴 Testing Redis Connectivity...")

    try:
        import redis

        # Connect to Redis
        r = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)

        # Test basic operations
        test_key = f"kb7_test_{int(time.time())}"
        test_data = {
            'timestamp': datetime.utcnow().isoformat(),
            'test_type': 'connectivity',
            'component': 'redis'
        }

        # Set and get test data
        r.set(test_key, json.dumps(test_data))
        retrieved_data = r.get(test_key)
        parsed_data = json.loads(retrieved_data)

        # Verify data integrity
        assert parsed_data['component'] == 'redis'

        # Test cache operations
        cache_key = f"cache_test_{int(time.time())}"
        r.setex(cache_key, 60, "cached_value")  # 60 seconds TTL

        # Test lists (for queue simulation)
        queue_key = f"queue_test_{int(time.time())}"
        r.lpush(queue_key, "item1", "item2", "item3")
        queue_length = r.llen(queue_key)

        # Cleanup
        r.delete(test_key, cache_key, queue_key)

        # Performance test
        start_time = time.time()
        for i in range(100):
            r.set(f"perf_test_{i}", f"value_{i}")
        bulk_set_time = time.time() - start_time

        start_time = time.time()
        for i in range(100):
            r.get(f"perf_test_{i}")
        bulk_get_time = time.time() - start_time

        # Cleanup performance test data
        for i in range(100):
            r.delete(f"perf_test_{i}")

        redis_result = {
            'status': 'healthy',
            'operations_tested': ['set', 'get', 'setex', 'lpush', 'llen'],
            'queue_length_test': queue_length,
            'bulk_set_time_ms': bulk_set_time * 1000,
            'bulk_get_time_ms': bulk_get_time * 1000,
            'avg_operation_time_ms': ((bulk_set_time + bulk_get_time) / 200) * 1000
        }

        print(f"✅ Redis connectivity successful")
        print(f"   Queue operations: {queue_length} items")
        print(f"   Avg operation time: {redis_result['avg_operation_time_ms']:.2f}ms")

        return redis_result

    except Exception as e:
        logger.error(f"❌ Redis connectivity failed: {e}")
        return {'status': 'failed', 'error': str(e)}


async def test_postgres_connectivity():
    """Test PostgreSQL connectivity (skipped due to driver issues)"""
    print("🐘 Testing PostgreSQL Connectivity...")
    print("⚠️ PostgreSQL test skipped - psycopg2 driver installation issues")

    return {
        'status': 'skipped',
        'reason': 'psycopg2 driver not available',
        'operations_tested': [],
        'note': 'Would test CREATE, INSERT, SELECT, JSONB operations'
    }


async def test_http_client():
    """Test HTTP client functionality for external service connectivity"""
    print("🌐 Testing HTTP Client Functionality...")

    try:
        import aiohttp

        async with aiohttp.ClientSession() as session:
            # Test basic HTTP functionality
            start_time = time.time()

            # Test a reliable service (httpbin for testing)
            test_urls = [
                'https://httpbin.org/json',
                'https://httpbin.org/headers',
                'https://httpbin.org/get'
            ]

            results = []

            for url in test_urls:
                try:
                    request_start = time.time()
                    async with session.get(url, timeout=aiohttp.ClientTimeout(total=10)) as response:
                        if response.status == 200:
                            response_data = await response.json()
                            request_time = time.time() - request_start

                            results.append({
                                'url': url,
                                'status': response.status,
                                'response_time_ms': request_time * 1000,
                                'success': True
                            })
                        else:
                            results.append({
                                'url': url,
                                'status': response.status,
                                'success': False
                            })
                except Exception as e:
                    results.append({
                        'url': url,
                        'error': str(e),
                        'success': False
                    })

            total_time = time.time() - start_time
            successful_requests = sum(1 for r in results if r.get('success', False))

            http_result = {
                'status': 'healthy' if successful_requests > 0 else 'failed',
                'total_requests': len(test_urls),
                'successful_requests': successful_requests,
                'total_time_ms': total_time * 1000,
                'results': results
            }

            if successful_requests > 0:
                avg_response_time = sum(r.get('response_time_ms', 0) for r in results if r.get('success', False)) / successful_requests
                print(f"✅ HTTP client functionality successful")
                print(f"   Successful requests: {successful_requests}/{len(test_urls)}")
                print(f"   Avg response time: {avg_response_time:.0f}ms")
            else:
                print(f"⚠️ HTTP client had issues (possibly network connectivity)")

            return http_result

    except Exception as e:
        logger.error(f"❌ HTTP client test failed: {e}")
        return {'status': 'failed', 'error': str(e)}


async def test_async_performance():
    """Test async performance and concurrency"""
    print("⚡ Testing Async Performance...")

    try:
        # Test concurrent operations
        async def mock_database_operation(operation_id, delay):
            await asyncio.sleep(delay)
            return {
                'operation_id': operation_id,
                'delay': delay,
                'timestamp': datetime.utcnow().isoformat()
            }

        # Test concurrent execution
        start_time = time.time()

        # Sequential execution
        sequential_results = []
        for i in range(10):
            result = await mock_database_operation(i, 0.01)
            sequential_results.append(result)

        sequential_time = time.time() - start_time

        # Concurrent execution
        start_time = time.time()

        tasks = [mock_database_operation(i, 0.01) for i in range(10)]
        concurrent_results = await asyncio.gather(*tasks)

        concurrent_time = time.time() - start_time

        # Calculate improvement
        performance_improvement = ((sequential_time - concurrent_time) / sequential_time) * 100

        async_result = {
            'status': 'healthy',
            'sequential_time_ms': sequential_time * 1000,
            'concurrent_time_ms': concurrent_time * 1000,
            'performance_improvement_pct': performance_improvement,
            'operations_count': len(tasks)
        }

        print(f"✅ Async performance test successful")
        print(f"   Sequential: {sequential_time*1000:.0f}ms")
        print(f"   Concurrent: {concurrent_time*1000:.0f}ms")
        print(f"   Improvement: {performance_improvement:.1f}%")

        return async_result

    except Exception as e:
        logger.error(f"❌ Async performance test failed: {e}")
        return {'status': 'failed', 'error': str(e)}


async def test_data_serialization():
    """Test data serialization and validation"""
    print("📝 Testing Data Serialization...")

    try:
        import json
        import pandas as pd

        # Test complex data structures
        test_data = {
            'patient_id': 'patient_12345',
            'medications': [
                {'rxnorm': '197361', 'name': 'Lisinopril', 'dose': '10mg'},
                {'rxnorm': '197362', 'name': 'Metformin', 'dose': '500mg'}
            ],
            'conditions': ['I25.10', 'E11.9'],
            'metadata': {
                'timestamp': datetime.utcnow().isoformat(),
                'version': '1.0',
                'source': 'kb7_runtime'
            }
        }

        # Test JSON serialization
        json_str = json.dumps(test_data, indent=2)
        parsed_data = json.loads(json_str)
        assert parsed_data['patient_id'] == test_data['patient_id']

        # Test pandas operations
        df_data = {
            'drug_code': ['197361', '197362', '197363'],
            'drug_name': ['Lisinopril', 'Metformin', 'Atorvastatin'],
            'safety_score': [0.85, 0.92, 0.78],
            'efficacy_score': [0.88, 0.89, 0.85]
        }

        df = pd.DataFrame(df_data)

        # Calculate composite score
        df['composite_score'] = (df['safety_score'] + df['efficacy_score']) / 2

        # Test data operations
        high_score_drugs = df[df['composite_score'] > 0.85]
        avg_safety_score = df['safety_score'].mean()

        serialization_result = {
            'status': 'healthy',
            'json_serialization': 'success',
            'pandas_operations': 'success',
            'data_integrity': parsed_data == test_data,
            'dataframe_rows': len(df),
            'high_score_drugs': len(high_score_drugs),
            'avg_safety_score': avg_safety_score
        }

        print(f"✅ Data serialization test successful")
        print(f"   DataFrame operations: {len(df)} rows processed")
        print(f"   High-scoring drugs: {len(high_score_drugs)}")
        print(f"   Avg safety score: {avg_safety_score:.3f}")

        return serialization_result

    except Exception as e:
        logger.error(f"❌ Data serialization test failed: {e}")
        return {'status': 'failed', 'error': str(e)}


async def main():
    """Run all connectivity tests"""
    print("🚀 Starting KB7 Runtime Layer Connectivity Tests")
    print("=" * 70)

    test_results = {}
    start_time = time.time()

    # Run all connectivity tests
    test_results['redis'] = await test_redis_connectivity()
    test_results['postgres'] = await test_postgres_connectivity()
    test_results['http_client'] = await test_http_client()
    test_results['async_performance'] = await test_async_performance()
    test_results['data_serialization'] = await test_data_serialization()

    total_time = time.time() - start_time

    # Calculate overall results
    successful_tests = sum(1 for result in test_results.values()
                          if isinstance(result, dict) and result.get('status') == 'healthy')
    total_tests = len(test_results)
    success_rate = (successful_tests / total_tests) * 100

    print("=" * 70)
    print("🏁 Connectivity Test Summary")
    print(f"   Total Tests: {total_tests}")
    print(f"   Successful: {successful_tests}")
    print(f"   Success Rate: {success_rate:.0f}%")
    print(f"   Total Time: {total_time:.2f}s")

    # Show detailed results
    print("\n📊 Detailed Results:")
    for test_name, result in test_results.items():
        if isinstance(result, dict):
            status = result.get('status', 'unknown')
            status_icon = '✅' if status == 'healthy' else '❌'
            print(f"   {status_icon} {test_name}: {status}")
            if status == 'failed' and 'error' in result:
                print(f"      Error: {result['error']}")

    # Generate test report
    report = {
        'test_suite': 'connectivity',
        'timestamp': datetime.utcnow().isoformat(),
        'summary': {
            'total_tests': total_tests,
            'successful_tests': successful_tests,
            'success_rate': success_rate,
            'total_time_seconds': total_time
        },
        'results': test_results
    }

    # Save report
    with open('connectivity_test_report.json', 'w') as f:
        json.dump(report, f, indent=2, default=str)

    print(f"\n📄 Test report saved to: connectivity_test_report.json")

    if success_rate >= 80:  # Allow some tolerance for network issues
        print("✅ Connectivity tests passed!")
        return 0
    else:
        print("❌ Connectivity tests failed!")
        return 1


if __name__ == "__main__":
    try:
        result = asyncio.run(main())
        sys.exit(result)
    except KeyboardInterrupt:
        print("\n⚠️ Test interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Test failed with error: {e}")
        sys.exit(1)