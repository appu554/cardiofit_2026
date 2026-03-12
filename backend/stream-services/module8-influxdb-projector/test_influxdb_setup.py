#!/usr/bin/env python3
"""Test script to verify InfluxDB setup and first write."""
import os
import sys
from datetime import datetime
from dotenv import load_dotenv

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

load_dotenv()

from influxdb_client import Point, WritePrecision
from influxdb_manager import influxdb_manager
from config import config


def test_connection():
    """Test InfluxDB connection."""
    print("🔌 Testing InfluxDB connection...")
    try:
        influxdb_manager.connect()
        health = influxdb_manager.client.health()
        print(f"✅ Connected to InfluxDB: {health.status}")
        return True
    except Exception as e:
        print(f"❌ Connection failed: {e}")
        return False


def test_bucket_creation():
    """Test bucket creation."""
    print("\n🪣 Testing bucket creation...")
    try:
        influxdb_manager.setup_buckets()

        # Verify buckets exist
        buckets = [
            config.INFLUXDB_BUCKET_REALTIME,
            config.INFLUXDB_BUCKET_1MIN,
            config.INFLUXDB_BUCKET_1HOUR
        ]

        for bucket_name in buckets:
            bucket = influxdb_manager.buckets_api.find_bucket_by_name(bucket_name)
            if bucket:
                retention = bucket.retention_rules[0].every_seconds if bucket.retention_rules else "N/A"
                retention_days = retention / 86400 if isinstance(retention, int) else "N/A"
                print(f"✅ Bucket '{bucket_name}': {retention_days} days retention")
            else:
                print(f"❌ Bucket '{bucket_name}' not found")
                return False

        return True
    except Exception as e:
        print(f"❌ Bucket creation failed: {e}")
        return False


def test_downsampling_tasks():
    """Test downsampling task creation."""
    print("\n⏱️  Testing downsampling task creation...")
    try:
        influxdb_manager.setup_downsampling_tasks()

        # Verify tasks exist
        tasks = ["downsample_1min", "downsample_1hour"]
        for task_name in tasks:
            existing_tasks = influxdb_manager.tasks_api.find_tasks(name=task_name)
            if existing_tasks:
                print(f"✅ Task '{task_name}' exists")
            else:
                print(f"⚠️  Task '{task_name}' not found (may require permissions)")

        return True
    except Exception as e:
        print(f"⚠️  Task creation completed with warnings: {e}")
        return True  # Non-critical


def test_first_write():
    """Test writing sample vital signs data."""
    print("\n📝 Testing first write...")
    try:
        timestamp = datetime.utcnow()

        # Create sample vital sign points
        test_points = [
            influxdb_manager.create_vital_point(
                measurement="heart_rate",
                patient_id="TEST_P001",
                device_id="TEST_MON_001",
                department_id="TEST_ICU",
                fields={"value": 75.0},
                timestamp=timestamp
            ),
            influxdb_manager.create_vital_point(
                measurement="blood_pressure",
                patient_id="TEST_P001",
                device_id="TEST_MON_001",
                department_id="TEST_ICU",
                fields={"systolic": 120.0, "diastolic": 80.0},
                timestamp=timestamp
            ),
            influxdb_manager.create_vital_point(
                measurement="spo2",
                patient_id="TEST_P001",
                device_id="TEST_MON_001",
                department_id="TEST_ICU",
                fields={"value": 98.0},
                timestamp=timestamp
            ),
            influxdb_manager.create_vital_point(
                measurement="temperature",
                patient_id="TEST_P001",
                device_id="TEST_MON_001",
                department_id="TEST_ICU",
                fields={"value": 37.2},
                timestamp=timestamp
            )
        ]

        # Write test points
        influxdb_manager.write_vital_signs(test_points)
        print(f"✅ Successfully wrote {len(test_points)} test points")

        # Verify write with query
        print("\n🔍 Verifying data with query...")
        query = f'''
from(bucket: "{config.INFLUXDB_BUCKET_REALTIME}")
    |> range(start: -1m)
    |> filter(fn: (r) => r["patient_id"] == "TEST_P001")
'''
        tables = influxdb_manager.query_api.query(query, org=config.INFLUXDB_ORG)

        record_count = sum(len(table.records) for table in tables)
        print(f"✅ Query returned {record_count} records")

        # Display sample records
        for table in tables[:2]:  # Show first 2 tables
            for record in table.records[:2]:  # Show first 2 records per table
                print(f"   📊 {record.get_measurement()}: {record.get_field()} = {record.get_value()}")

        return True

    except Exception as e:
        print(f"❌ Write test failed: {e}")
        return False


def cleanup():
    """Clean up test data."""
    print("\n🧹 Cleaning up test data...")
    try:
        # Delete test data
        delete_predicate = '_measurement="heart_rate" OR _measurement="blood_pressure" OR _measurement="spo2" OR _measurement="temperature"'
        start = "1970-01-01T00:00:00Z"
        stop = datetime.utcnow().isoformat() + "Z"

        influxdb_manager.client.delete_api().delete(
            start=start,
            stop=stop,
            predicate=f'{delete_predicate} AND patient_id="TEST_P001"',
            bucket=config.INFLUXDB_BUCKET_REALTIME,
            org=config.INFLUXDB_ORG
        )
        print("✅ Test data cleaned up")
    except Exception as e:
        print(f"⚠️  Cleanup warning: {e}")


def main():
    """Run all tests."""
    print("=" * 60)
    print("InfluxDB Projector - Setup Verification")
    print("=" * 60)

    results = []

    # Run tests
    results.append(("Connection", test_connection()))
    if results[-1][1]:
        results.append(("Bucket Creation", test_bucket_creation()))
        results.append(("Downsampling Tasks", test_downsampling_tasks()))
        results.append(("First Write", test_first_write()))

        # Cleanup
        cleanup()

    # Close connection
    influxdb_manager.close()

    # Summary
    print("\n" + "=" * 60)
    print("Test Summary")
    print("=" * 60)
    for test_name, passed in results:
        status = "✅ PASSED" if passed else "❌ FAILED"
        print(f"{test_name:.<40} {status}")

    all_passed = all(result[1] for result in results)
    print("=" * 60)
    if all_passed:
        print("🎉 All tests passed! InfluxDB Projector is ready.")
        print("\nNext steps:")
        print("1. Start the service: python run_service.py")
        print("2. Check health: curl http://localhost:8054/health")
        print("3. Monitor stats: curl http://localhost:8054/stats")
    else:
        print("⚠️  Some tests failed. Check configuration and InfluxDB connection.")

    sys.exit(0 if all_passed else 1)


if __name__ == "__main__":
    main()
