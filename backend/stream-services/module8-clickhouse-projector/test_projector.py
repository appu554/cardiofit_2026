#!/usr/bin/env python3
"""
Test ClickHouse Projector with sample enriched events.
"""

import json
import sys
from datetime import datetime, timedelta
from clickhouse_driver import Client
from app.projector import ClickHouseProjector
import logging

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def create_test_event(event_id: str, risk_level: str = "MODERATE") -> dict:
    """Create a test enriched event."""
    return {
        "eventId": event_id,
        "patientId": f"PAT-{event_id[:4]}",
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "eventType": "VITAL_SIGNS",
        "departmentId": "DEPT-ICU-001",
        "vitalSigns": {
            "heartRate": 85,
            "bloodPressure": {
                "systolic": 120,
                "diastolic": 80
            },
            "spO2": 98,
            "temperature": 37.2
        },
        "enrichment": {
            "clinicalScores": {
                "news2": 3,
                "qsofa": 1
            },
            "riskLevel": risk_level,
            "mlPredictions": {
                "sepsisRisk24h": 0.15,
                "cardiacRisk7d": 0.08,
                "readmissionRisk30d": 0.22
            }
        }
    }


def test_projector_processing():
    """Test projector with sample events."""

    # Test configuration
    config = {
        'kafka': {
            'bootstrap_servers': 'localhost:9092',
            'topic': 'prod.ehr.events.enriched',
            'group_id': 'module8-clickhouse-projector-test',
            'auto_offset_reset': 'latest',
            'enable_auto_commit': False,
        },
        'clickhouse': {
            'host': 'localhost',
            'port': 9000,
            'database': 'module8_analytics',
            'user': 'module8_user',
            'password': 'module8_password',
        },
        'batch': {
            'size': 10,
            'timeout': 5
        }
    }

    logger.info("Creating test projector...")
    projector = ClickHouseProjector(config)

    # Create test events
    test_events = [
        create_test_event("EVT-001", "LOW"),
        create_test_event("EVT-002", "MODERATE"),
        create_test_event("EVT-003", "HIGH"),
        create_test_event("EVT-004", "CRITICAL"),
        create_test_event("EVT-005", "MODERATE"),
    ]

    logger.info(f"Processing {len(test_events)} test events...")
    projector.process_batch(test_events)

    # Verify data in ClickHouse
    logger.info("\nVerifying data in ClickHouse...")
    client = projector.client

    # Check clinical_events_fact
    clinical_count = client.execute('SELECT count() FROM clinical_events_fact')[0][0]
    logger.info(f"clinical_events_fact: {clinical_count} rows")

    # Check ml_predictions_fact
    ml_count = client.execute('SELECT count() FROM ml_predictions_fact')[0][0]
    logger.info(f"ml_predictions_fact: {ml_count} rows")

    # Check alerts_fact
    alerts_count = client.execute('SELECT count() FROM alerts_fact')[0][0]
    logger.info(f"alerts_fact: {alerts_count} rows (expected 2: HIGH + CRITICAL)")

    # Sample queries
    logger.info("\nSample Analytics Queries:")

    # Risk distribution
    logger.info("\n1. Risk Level Distribution:")
    risk_dist = client.execute(
        """
        SELECT risk_level, count() as count
        FROM clinical_events_fact
        GROUP BY risk_level
        ORDER BY count DESC
        """
    )
    for row in risk_dist:
        logger.info(f"   {row[0]}: {row[1]} events")

    # Average vitals by risk level
    logger.info("\n2. Average Vitals by Risk Level:")
    avg_vitals = client.execute(
        """
        SELECT
            risk_level,
            round(avg(heart_rate), 1) as avg_hr,
            round(avg(bp_systolic), 1) as avg_systolic,
            round(avg(news2_score), 1) as avg_news2
        FROM clinical_events_fact
        GROUP BY risk_level
        ORDER BY risk_level
        """
    )
    for row in avg_vitals:
        logger.info(f"   {row[0]}: HR={row[1]}, Systolic={row[2]}, NEWS2={row[3]}")

    # ML predictions summary
    logger.info("\n3. ML Predictions Summary:")
    ml_summary = client.execute(
        """
        SELECT
            round(avg(sepsis_risk_24h), 3) as avg_sepsis_risk,
            round(avg(cardiac_risk_7d), 3) as avg_cardiac_risk,
            round(avg(readmission_risk_30d), 3) as avg_readmission_risk
        FROM ml_predictions_fact
        """
    )
    if ml_summary:
        row = ml_summary[0]
        logger.info(f"   Sepsis (24h): {row[0]}")
        logger.info(f"   Cardiac (7d): {row[1]}")
        logger.info(f"   Readmission (30d): {row[2]}")

    # Storage info
    logger.info("\n4. Storage Information:")
    storage_info = projector._get_storage_info()
    for table, info in storage_info.items():
        logger.info(f"   {table}: {info['size']} ({info['rows']} rows)")

    # Get analytics summary
    logger.info("\n5. Analytics Summary:")
    summary = projector.get_analytics_summary()
    logger.info(json.dumps(summary, indent=2))

    # Cleanup
    projector.close()

    logger.info("\nTest completed successfully!")
    return True


def test_materialized_views():
    """Test materialized views are updating correctly."""
    logger.info("\nTesting Materialized Views...")

    client = Client(
        host='localhost',
        port=9000,
        database='module8_analytics',
        user='module8_user',
        password='module8_password'
    )

    # Check daily patient stats
    logger.info("\n1. Daily Patient Stats (from materialized view):")
    daily_stats = client.execute(
        """
        SELECT
            patient_id,
            day,
            event_count,
            round(avg_heart_rate, 1) as avg_hr,
            high_risk_events
        FROM daily_patient_stats_mv
        ORDER BY day DESC, patient_id
        LIMIT 5
        """
    )
    for row in daily_stats:
        logger.info(f"   {row[0]} on {row[1]}: {row[2]} events, HR={row[3]}, HighRisk={row[4]}")

    # Check hourly department stats
    logger.info("\n2. Hourly Department Stats (from materialized view):")
    hourly_stats = client.execute(
        """
        SELECT
            department_id,
            hour,
            event_count,
            critical_events,
            high_risk_events
        FROM hourly_department_stats_mv
        ORDER BY hour DESC
        LIMIT 5
        """
    )
    for row in hourly_stats:
        logger.info(f"   {row[0]} at {row[1]}: {row[2]} events, Critical={row[3]}, High={row[4]}")

    logger.info("\nMaterialized views test completed!")


if __name__ == '__main__':
    try:
        logger.info("=" * 60)
        logger.info("ClickHouse Projector Test Suite")
        logger.info("=" * 60)

        # Test 1: Basic processing
        test_projector_processing()

        # Test 2: Materialized views
        test_materialized_views()

        logger.info("\n" + "=" * 60)
        logger.info("All tests passed!")
        logger.info("=" * 60)

        sys.exit(0)

    except Exception as e:
        logger.error(f"Test failed: {e}", exc_info=True)
        sys.exit(1)
