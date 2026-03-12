"""
Test UPS Projector UPSERT Logic

Tests table creation and first UPSERT operation.
"""

import json
import time
import psycopg2
from psycopg2.extras import execute_batch


def test_table_creation():
    """Test that UPS read model table exists with correct schema."""
    conn = psycopg2.connect(
        host="localhost",
        port=5433,
        database="cardiofit_analytics",
        user="cardiofit",
        password="cardiofit_analytics_pass"
    )

    try:
        with conn.cursor() as cursor:
            # Check table exists
            cursor.execute("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables
                    WHERE table_schema = 'module8_projections'
                      AND table_name = 'ups_read_model'
                );
            """)
            exists = cursor.fetchone()[0]
            assert exists, "Table ups_read_model does not exist"

            # Check column count
            cursor.execute("""
                SELECT COUNT(*)
                FROM information_schema.columns
                WHERE table_schema = 'module8_projections'
                  AND table_name = 'ups_read_model';
            """)
            column_count = cursor.fetchone()[0]
            print(f"Table has {column_count} columns")

            # Check key columns
            cursor.execute("""
                SELECT column_name, data_type
                FROM information_schema.columns
                WHERE table_schema = 'module8_projections'
                  AND table_name = 'ups_read_model'
                ORDER BY ordinal_position;
            """)
            columns = cursor.fetchall()
            print("\nTable schema:")
            for col_name, col_type in columns:
                print(f"  {col_name}: {col_type}")

            # Check indexes
            cursor.execute("""
                SELECT indexname, indexdef
                FROM pg_indexes
                WHERE schemaname = 'module8_projections'
                  AND tablename = 'ups_read_model';
            """)
            indexes = cursor.fetchall()
            print(f"\nTable has {len(indexes)} indexes:")
            for idx_name, idx_def in indexes:
                print(f"  {idx_name}")

            print("\n✅ Table creation verified")

    finally:
        conn.close()


def test_first_upsert():
    """Test first UPSERT operation with sample patient data."""
    conn = psycopg2.connect(
        host="localhost",
        port=5433,
        database="cardiofit_analytics",
        user="cardiofit",
        password="cardiofit_analytics_pass"
    )

    try:
        with conn.cursor() as cursor:
            # Sample patient data
            patient_id = "P12345"
            demographics = json.dumps({
                "first_name": "John",
                "last_name": "Doe",
                "age": 45,
                "gender": "M"
            })
            current_department = "ICU_01"
            current_location = "ICU-Room-101"

            latest_vitals = json.dumps({
                "heart_rate": 95,
                "respiratory_rate": 18,
                "blood_pressure_systolic": 135,
                "blood_pressure_diastolic": 85,
                "temperature": 37.2,
                "spo2": 96
            })
            latest_vitals_timestamp = int(time.time() * 1000)

            news2_score = 3
            news2_category = "MEDIUM"
            qsofa_score = 0
            sofa_score = 2
            risk_level = "MODERATE"

            ml_predictions = json.dumps({
                "sepsis_probability": 0.15,
                "deterioration_risk": 0.25,
                "model_version": "1.0.0"
            })
            ml_predictions_timestamp = latest_vitals_timestamp

            active_alerts = json.dumps([
                {
                    "alert_id": "ALERT-001",
                    "type": "VITAL_THRESHOLD",
                    "priority": "MEDIUM",
                    "message": "Heart rate elevated",
                    "timestamp": latest_vitals_timestamp
                }
            ])
            active_alerts_count = 1

            protocol_compliance = json.dumps({
                "status": "COMPLIANT",
                "protocols_checked": ["SEPSIS_BUNDLE", "EARLY_WARNING"],
                "last_checked": latest_vitals_timestamp
            })
            protocol_status = "COMPLIANT"

            last_event_id = "evt_123456"
            last_event_type = "VITAL_SIGNS"
            last_updated = latest_vitals_timestamp
            event_count = 1

            # UPSERT query
            start_time = time.time()

            upsert_query = """
                INSERT INTO module8_projections.ups_read_model (
                    patient_id,
                    demographics,
                    current_department,
                    current_location,
                    admission_timestamp,
                    latest_vitals,
                    latest_vitals_timestamp,
                    news2_score,
                    news2_category,
                    qsofa_score,
                    sofa_score,
                    risk_level,
                    ml_predictions,
                    ml_predictions_timestamp,
                    active_alerts,
                    active_alerts_count,
                    protocol_compliance,
                    protocol_status,
                    vitals_trend,
                    trend_confidence,
                    last_event_id,
                    last_event_type,
                    last_updated,
                    event_count
                ) VALUES (
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                )
                ON CONFLICT (patient_id) DO UPDATE SET
                    demographics = COALESCE(EXCLUDED.demographics, ups_read_model.demographics),
                    current_department = COALESCE(EXCLUDED.current_department, ups_read_model.current_department),
                    current_location = COALESCE(EXCLUDED.current_location, ups_read_model.current_location),
                    latest_vitals = CASE
                        WHEN EXCLUDED.latest_vitals IS NOT NULL AND
                             EXCLUDED.latest_vitals_timestamp > COALESCE(ups_read_model.latest_vitals_timestamp, 0)
                        THEN EXCLUDED.latest_vitals
                        ELSE ups_read_model.latest_vitals
                    END,
                    latest_vitals_timestamp = CASE
                        WHEN EXCLUDED.latest_vitals_timestamp > COALESCE(ups_read_model.latest_vitals_timestamp, 0)
                        THEN EXCLUDED.latest_vitals_timestamp
                        ELSE ups_read_model.latest_vitals_timestamp
                    END,
                    news2_score = COALESCE(EXCLUDED.news2_score, ups_read_model.news2_score),
                    news2_category = COALESCE(EXCLUDED.news2_category, ups_read_model.news2_category),
                    qsofa_score = COALESCE(EXCLUDED.qsofa_score, ups_read_model.qsofa_score),
                    sofa_score = COALESCE(EXCLUDED.sofa_score, ups_read_model.sofa_score),
                    risk_level = COALESCE(EXCLUDED.risk_level, ups_read_model.risk_level),
                    ml_predictions = CASE
                        WHEN EXCLUDED.ml_predictions IS NOT NULL AND
                             EXCLUDED.ml_predictions_timestamp > COALESCE(ups_read_model.ml_predictions_timestamp, 0)
                        THEN EXCLUDED.ml_predictions
                        ELSE ups_read_model.ml_predictions
                    END,
                    ml_predictions_timestamp = CASE
                        WHEN EXCLUDED.ml_predictions_timestamp > COALESCE(ups_read_model.ml_predictions_timestamp, 0)
                        THEN EXCLUDED.ml_predictions_timestamp
                        ELSE ups_read_model.ml_predictions_timestamp
                    END,
                    active_alerts = EXCLUDED.active_alerts,
                    active_alerts_count = EXCLUDED.active_alerts_count,
                    protocol_compliance = COALESCE(EXCLUDED.protocol_compliance, ups_read_model.protocol_compliance),
                    protocol_status = COALESCE(EXCLUDED.protocol_status, ups_read_model.protocol_status),
                    last_event_id = EXCLUDED.last_event_id,
                    last_event_type = EXCLUDED.last_event_type,
                    last_updated = EXCLUDED.last_updated,
                    event_count = ups_read_model.event_count + EXCLUDED.event_count,
                    updated_at = NOW()
                RETURNING *;
            """

            cursor.execute(upsert_query, (
                patient_id,
                demographics,
                current_department,
                current_location,
                None,  # admission_timestamp
                latest_vitals,
                latest_vitals_timestamp,
                news2_score,
                news2_category,
                qsofa_score,
                sofa_score,
                risk_level,
                ml_predictions,
                ml_predictions_timestamp,
                active_alerts,
                active_alerts_count,
                protocol_compliance,
                protocol_status,
                None,  # vitals_trend
                None,  # trend_confidence
                last_event_id,
                last_event_type,
                last_updated,
                event_count
            ))

            upsert_time_ms = (time.time() - start_time) * 1000
            result = cursor.fetchone()

            conn.commit()

            print(f"\n✅ UPSERT completed in {upsert_time_ms:.2f}ms")
            print(f"Patient ID: {result[0]}")
            print(f"Department: {result[2]}")
            print(f"Risk Level: {result[11]}")
            print(f"Event Count: {result[23]}")

            # Verify with SELECT
            start_time = time.time()
            cursor.execute("""
                SELECT * FROM module8_projections.ups_read_model
                WHERE patient_id = %s;
            """, (patient_id,))
            select_time_ms = (time.time() - start_time) * 1000

            row = cursor.fetchone()
            print(f"\n✅ SELECT completed in {select_time_ms:.2f}ms (target: <10ms)")

            # Parse JSONB fields (psycopg2 returns JSONB as dict, not string)
            vitals = row[5] if row[5] else {}
            predictions = row[12] if row[12] else {}
            alerts = row[14] if row[14] else []

            print(f"\nPatient Summary:")
            print(f"  ID: {row[0]}")
            print(f"  Department: {row[2]}")
            print(f"  Location: {row[3]}")
            print(f"  Latest Vitals: HR={vitals.get('heart_rate')}, SpO2={vitals.get('spo2')}")
            print(f"  Risk Level: {row[11]}")
            print(f"  NEWS2 Score: {row[7]} ({row[8]})")
            print(f"  ML Predictions: Sepsis={predictions.get('sepsis_probability')}")
            print(f"  Active Alerts: {len(alerts)}")
            print(f"  Last Updated: {row[22]}")

            # Test second UPSERT (should increment event_count)
            print("\n--- Testing UPDATE scenario ---")

            # Simulate new event with updated vitals
            new_vitals = json.dumps({
                "heart_rate": 88,
                "respiratory_rate": 16,
                "blood_pressure_systolic": 128,
                "blood_pressure_diastolic": 82,
                "temperature": 37.1,
                "spo2": 98
            })
            new_vitals_timestamp = int(time.time() * 1000)
            new_event_id = "evt_123457"

            start_time = time.time()
            cursor.execute(upsert_query, (
                patient_id,
                None,  # demographics unchanged
                current_department,
                current_location,
                None,
                new_vitals,
                new_vitals_timestamp,
                2,  # improved NEWS2 score
                "LOW",
                0,
                1,  # improved SOFA
                "LOW",  # improved risk
                None,  # no new predictions
                None,
                json.dumps([]),  # no active alerts
                0,
                protocol_compliance,
                "COMPLIANT",
                None,
                None,
                new_event_id,
                "VITAL_SIGNS",
                new_vitals_timestamp,
                1  # will be added to existing count
            ))

            update_time_ms = (time.time() - start_time) * 1000
            updated_row = cursor.fetchone()

            conn.commit()

            print(f"✅ UPDATE completed in {update_time_ms:.2f}ms")
            print(f"New Event Count: {updated_row[23]} (should be >= 2)")
            print(f"New Risk Level: {updated_row[11]}")
            print(f"New NEWS2 Score: {updated_row[7]}")

            assert updated_row[23] >= 2, "Event count should increment"
            assert updated_row[11] == "LOW", "Risk level should update"

            print("\n✅ All UPSERT tests passed!")

    finally:
        conn.close()


def test_query_performance():
    """Test query performance for common patterns."""
    conn = psycopg2.connect(
        host="localhost",
        port=5433,
        database="cardiofit_analytics",
        user="cardiofit",
        password="cardiofit_analytics_pass"
    )

    try:
        with conn.cursor() as cursor:
            # Test 1: Single patient lookup
            start_time = time.time()
            cursor.execute("""
                SELECT * FROM module8_projections.ups_read_model
                WHERE patient_id = 'P12345';
            """)
            cursor.fetchone()
            lookup_time_ms = (time.time() - start_time) * 1000

            print(f"\n📊 Query Performance:")
            print(f"  Single patient lookup: {lookup_time_ms:.2f}ms (target: <10ms)")

            # Test 2: JSONB query on vitals
            start_time = time.time()
            cursor.execute("""
                SELECT patient_id, latest_vitals->>'heart_rate' as hr
                FROM module8_projections.ups_read_model
                WHERE (latest_vitals->>'heart_rate')::int > 80;
            """)
            cursor.fetchall()
            jsonb_time_ms = (time.time() - start_time) * 1000

            print(f"  JSONB vitals query: {jsonb_time_ms:.2f}ms")

            # Test 3: Risk level filter
            start_time = time.time()
            cursor.execute("""
                SELECT patient_id, risk_level, news2_score
                FROM module8_projections.ups_read_model
                WHERE risk_level IN ('HIGH', 'CRITICAL');
            """)
            cursor.fetchall()
            risk_time_ms = (time.time() - start_time) * 1000

            print(f"  Risk level filter: {risk_time_ms:.2f}ms")

            # Test 4: Department summary
            start_time = time.time()
            cursor.execute("""
                SELECT
                    current_department,
                    COUNT(*) as patient_count,
                    AVG(news2_score) as avg_news2
                FROM module8_projections.ups_read_model
                GROUP BY current_department;
            """)
            results = cursor.fetchall()
            dept_time_ms = (time.time() - start_time) * 1000

            print(f"  Department summary: {dept_time_ms:.2f}ms")
            for dept, count, avg_news2 in results:
                avg_display = f"{avg_news2:.1f}" if avg_news2 else "0.0"
                print(f"    {dept}: {count} patients, avg NEWS2: {avg_display}")

            print("\n✅ Query performance tests completed")

    finally:
        conn.close()


if __name__ == "__main__":
    print("=== UPS Projector UPSERT Tests ===\n")

    print("Test 1: Table Creation")
    test_table_creation()

    print("\n" + "="*50 + "\n")
    print("Test 2: First UPSERT and UPDATE")
    test_first_upsert()

    print("\n" + "="*50 + "\n")
    print("Test 3: Query Performance")
    test_query_performance()

    print("\n" + "="*50)
    print("✅ All tests completed successfully!")
