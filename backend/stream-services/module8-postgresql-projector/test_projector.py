"""
Quick test script for PostgreSQL Projector Service
Tests database connection and schema
"""
import psycopg2
from psycopg2.extras import RealDictCursor
import json
from datetime import datetime

# Configuration
POSTGRES_CONFIG = {
    "host": "172.21.0.4",
    "port": 5432,
    "database": "cardiofit_analytics",
    "user": "cardiofit",
    "password": "cardiofit_analytics_pass",
}

SCHEMA = "module8_projections"


def test_connection():
    """Test PostgreSQL connection"""
    print("Testing PostgreSQL connection...")
    try:
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        with conn.cursor() as cur:
            cur.execute("SELECT version();")
            version = cur.fetchone()[0]
            print(f"✓ Connected to PostgreSQL: {version}")
        conn.close()
        return True
    except Exception as e:
        print(f"✗ Connection failed: {e}")
        return False


def test_schema_exists():
    """Test schema and tables exist"""
    print("\nTesting schema...")
    try:
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        with conn.cursor() as cur:
            # Check schema
            cur.execute(
                "SELECT schema_name FROM information_schema.schemata WHERE schema_name = %s",
                (SCHEMA,)
            )
            if not cur.fetchone():
                print(f"✗ Schema {SCHEMA} does not exist")
                return False
            print(f"✓ Schema {SCHEMA} exists")

            # Check tables
            cur.execute(f"""
                SELECT table_name
                FROM information_schema.tables
                WHERE table_schema = '{SCHEMA}'
                ORDER BY table_name
            """)
            tables = [row[0] for row in cur.fetchall()]
            print(f"✓ Found {len(tables)} tables: {', '.join(tables)}")

        conn.close()
        return True
    except Exception as e:
        print(f"✗ Schema test failed: {e}")
        return False


def test_insert_sample_data():
    """Insert sample event to test tables"""
    print("\nTesting sample data insertion...")
    try:
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        with conn.cursor() as cur:
            cur.execute(f"SET search_path TO {SCHEMA}, public")

            # Sample event data
            event_id = "TEST-EVENT-001"
            patient_id = "PAT-TEST-001"
            timestamp = datetime.utcnow()
            event_type = "VITAL_SIGNS"
            event_data = {
                "id": event_id,
                "patient_id": patient_id,
                "event_type": event_type,
                "raw_data": {
                    "heart_rate": 85,
                    "blood_pressure_systolic": 120,
                    "blood_pressure_diastolic": 80,
                    "spo2": 98.5,
                    "temperature_celsius": 37.2
                },
                "enrichments": {
                    "NEWS2Score": 3,
                    "qSOFAScore": 0,
                    "riskLevel": "LOW"
                }
            }

            # Insert to enriched_events
            cur.execute("""
                INSERT INTO enriched_events
                (event_id, patient_id, timestamp, event_type, event_data)
                VALUES (%s, %s, %s, %s, %s)
            """, (event_id, patient_id, timestamp, event_type, json.dumps(event_data)))

            # Insert to patient_vitals
            cur.execute("""
                INSERT INTO patient_vitals
                (event_id, patient_id, timestamp, heart_rate, bp_systolic,
                 bp_diastolic, spo2, temperature_celsius)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            """, (event_id, patient_id, timestamp, 85, 120, 80, 98.5, 37.2))

            # Insert to clinical_scores
            cur.execute("""
                INSERT INTO clinical_scores
                (event_id, patient_id, timestamp, news2_score, qsofa_score, risk_level)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (event_id, patient_id, timestamp, 3, 0, "LOW"))

            # Insert to event_metadata
            cur.execute("""
                INSERT INTO event_metadata
                (event_id, patient_id, encounter_id, department_id,
                 device_id, timestamp, event_type)
                VALUES (%s, %s, %s, %s, %s, %s, %s)
            """, (event_id, patient_id, "ENC-001", "ICU", "DEV-001", timestamp, event_type))

        conn.commit()
        print(f"✓ Inserted sample event: {event_id}")
        conn.close()
        return True
    except Exception as e:
        print(f"✗ Insert failed: {e}")
        return False


def test_queries():
    """Test sample queries"""
    print("\nTesting queries...")
    try:
        conn = psycopg2.connect(**POSTGRES_CONFIG, cursor_factory=RealDictCursor)
        with conn.cursor() as cur:
            cur.execute(f"SET search_path TO {SCHEMA}, public")

            # Test query 1: Count events
            cur.execute("SELECT COUNT(*) as count FROM enriched_events")
            count = cur.fetchone()["count"]
            print(f"✓ Total events: {count}")

            # Test query 2: Latest vitals
            cur.execute("""
                SELECT * FROM latest_patient_vitals
                WHERE patient_id = 'PAT-TEST-001'
            """)
            vitals = cur.fetchone()
            if vitals:
                print(f"✓ Latest vitals for PAT-TEST-001: HR={vitals['heart_rate']}, BP={vitals['bp_systolic']}/{vitals['bp_diastolic']}")

            # Test query 3: Complete event detail
            cur.execute("""
                SELECT event_id, patient_id, event_type, heart_rate, news2_score, risk_level
                FROM complete_event_detail
                WHERE patient_id = 'PAT-TEST-001'
            """)
            event = cur.fetchone()
            if event:
                print(f"✓ Complete event detail: {dict(event)}")

        conn.close()
        return True
    except Exception as e:
        print(f"✗ Query test failed: {e}")
        return False


def cleanup_test_data():
    """Clean up test data"""
    print("\nCleaning up test data...")
    try:
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        with conn.cursor() as cur:
            cur.execute(f"SET search_path TO {SCHEMA}, public")
            cur.execute("DELETE FROM enriched_events WHERE event_id LIKE 'TEST-%'")
        conn.commit()
        conn.close()
        print("✓ Test data cleaned up")
        return True
    except Exception as e:
        print(f"✗ Cleanup failed: {e}")
        return False


def main():
    """Run all tests"""
    print("=" * 60)
    print("PostgreSQL Projector Service - Database Test")
    print("=" * 60)

    tests = [
        ("Connection", test_connection),
        ("Schema", test_schema_exists),
        ("Insert", test_insert_sample_data),
        ("Queries", test_queries),
        ("Cleanup", cleanup_test_data),
    ]

    results = []
    for name, test_func in tests:
        try:
            result = test_func()
            results.append((name, result))
        except Exception as e:
            print(f"\n✗ Test {name} crashed: {e}")
            results.append((name, False))

    # Summary
    print("\n" + "=" * 60)
    print("Test Summary")
    print("=" * 60)
    for name, passed in results:
        status = "✓ PASS" if passed else "✗ FAIL"
        print(f"{name:20} {status}")

    total_passed = sum(1 for _, passed in results if passed)
    print(f"\nTotal: {total_passed}/{len(results)} tests passed")

    if total_passed == len(results):
        print("\n🎉 All tests passed! PostgreSQL projector is ready.")
    else:
        print("\n⚠️  Some tests failed. Check configuration and schema.")


if __name__ == "__main__":
    main()
